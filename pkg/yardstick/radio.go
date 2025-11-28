package yardstick

import (
	"encoding/binary"
	"fmt"
	"time"
)

// RFST register address for direct strobe commands
const RegRFST = 0xDFE1

// MCSM1 register address for RX/TX behavior configuration
const RegMCSM1 = 0xDF13

// MARCSTATE register address for radio state
const RegMARCSTATE = 0xDF3B

// MARCSTATE values
const (
	MarcStateIdle = 0x01
	MarcStateRX   = 0x0D
	MarcStateTX   = 0x13
)

// SetModeRX puts the radio into receive mode
// This issues the SYS_CMD_RFMODE command which calls firmware RxMode()
func (d *Device) SetModeRX() error {
	// Issue RFMODE command to enter RX - firmware handles MCSM1 and strobe
	_, err := d.Send(AppSystem, SysCmdRFMode, []byte{RFSTSrx}, USBDefaultTimeout)
	if err != nil {
		return fmt.Errorf("failed to set RX mode: %w", err)
	}

	// Wait briefly for radio to transition to RX state
	time.Sleep(10 * time.Millisecond)

	// Verify we're in RX mode
	state, err := d.GetMARCSTATE()
	if err != nil {
		return fmt.Errorf("failed to verify RX mode: %w", err)
	}

	if state != MarcStateRX {
		return fmt.Errorf("radio not in RX mode: MARCSTATE=0x%02X (expected 0x%02X)", state, MarcStateRX)
	}

	return nil
}

// SetModeTX puts the radio into transmit mode
// Note: Normal transmit is done via RFXmit, not by setting TX mode directly
func (d *Device) SetModeTX() error {
	// Issue RFMODE command to enter TX - firmware handles MCSM1 and strobe
	_, err := d.Send(AppSystem, SysCmdRFMode, []byte{RFSTStx}, USBDefaultTimeout)
	if err != nil {
		return fmt.Errorf("failed to set TX mode: %w", err)
	}

	return nil
}

// SetModeIDLE puts the radio into idle mode
func (d *Device) SetModeIDLE() error {
	// Issue RFMODE command to enter IDLE - firmware handles the strobe
	_, err := d.Send(AppSystem, SysCmdRFMode, []byte{RFSTSidle}, USBDefaultTimeout)
	if err != nil {
		return fmt.Errorf("failed to set IDLE mode: %w", err)
	}

	return nil
}

// StrobeModeRX sends an SRX strobe without changing MCSM1 (transient state change)
func (d *Device) StrobeModeRX() error {
	return d.PokeByte(RegRFST, RFSTSrx)
}

// StrobeModeTX sends an STX strobe without changing MCSM1 (transient state change)
func (d *Device) StrobeModeTX() error {
	return d.PokeByte(RegRFST, RFSTStx)
}

// StrobeModeIDLE sends an SIDLE strobe without changing MCSM1 (transient state change)
func (d *Device) StrobeModeIDLE() error {
	return d.PokeByte(RegRFST, RFSTSidle)
}

// GetMARCSTATE returns the current radio state machine state
func (d *Device) GetMARCSTATE() (uint8, error) {
	return d.PeekByte(RegMARCSTATE)
}

// WaitForState polls MARCSTATE until the desired state is reached or timeout
func (d *Device) WaitForState(state uint8, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		current, err := d.GetMARCSTATE()
		if err != nil {
			return fmt.Errorf("failed to read MARCSTATE: %w", err)
		}
		if current == state {
			return nil
		}
		time.Sleep(1 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for radio state 0x%02X", state)
}

// RFXmit transmits RF data
// data: the RF payload to transmit (max 255 bytes for standard, use RFXmitLong for larger)
// repeat: number of times to repeat (0 = once, 65535 = forever)
// offset: start offset within data for repeat transmissions
func (d *Device) RFXmit(data []byte, repeat uint16, offset uint16) error {
	if len(data) > RFMaxTXBlock {
		if repeat > 0 || offset > 0 {
			return fmt.Errorf("repeat/offset not supported for long transmit")
		}
		return d.RFXmitLong(data)
	}

	// Build NIC_XMIT payload:
	// Bytes 0-1: data_len (little-endian)
	// Bytes 2-3: repeat count
	// Bytes 4-5: offset
	// Bytes 6+:  RF data
	payload := make([]byte, 6+len(data))
	binary.LittleEndian.PutUint16(payload[0:2], uint16(len(data)))
	binary.LittleEndian.PutUint16(payload[2:4], repeat)
	binary.LittleEndian.PutUint16(payload[4:6], offset)
	copy(payload[6:], data)

	// Calculate wait time based on data length and repeats
	waitLen := len(data)
	if repeat > 0 {
		waitLen += int(repeat) * (len(data) - int(offset))
	}
	waitTime := USBTXWaitTimeout * time.Duration((waitLen/RFMaxTXBlock)+1)

	response, err := d.Send(AppNIC, NICXmit, payload, waitTime)
	if err != nil {
		return fmt.Errorf("transmit failed: %w", err)
	}

	// Check response for errors
	// Note: transmit() returns 1 on success, 0 on failure
	// Some firmware versions return ASCII '0' (0x30) on success
	if len(response) > 0 {
		code := response[0]
		// Success codes: 1 (new firmware), '0'/0x30 (old firmware), 0 (some versions)
		if code != 1 && code != '0' && code != 0 {
			return fmt.Errorf("transmit error: device returned 0x%02X", code)
		}
	}

	return nil
}

// RFXmitLong transmits RF data larger than 255 bytes using chunked transfer
func (d *Device) RFXmitLong(data []byte) error {
	if len(data) > RFMaxTXLong {
		return fmt.Errorf("data too large: %d bytes exceeds maximum %d", len(data), RFMaxTXLong)
	}

	dataLen := len(data)

	// Split data into chunks
	var chunks [][]byte
	for i := 0; i < dataLen; i += RFMaxTXChunk {
		end := i + RFMaxTXChunk
		if end > dataLen {
			end = dataLen
		}
		chunks = append(chunks, data[i:end])
	}

	// Calculate preload count (chunks to send in initial packet)
	preload := RFMaxTXBlock / RFMaxTXChunk
	if preload > len(chunks) {
		preload = len(chunks)
	}

	// Build initial payload with preloaded chunks
	initialData := make([]byte, 0, 3+preload*RFMaxTXChunk)
	lenBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(lenBytes, uint16(dataLen))
	initialData = append(initialData, lenBytes...)
	initialData = append(initialData, byte(preload))
	for i := 0; i < preload; i++ {
		initialData = append(initialData, chunks[i]...)
	}

	// Send initial long transmit command
	waitTime := USBTXWaitTimeout * time.Duration(preload)
	response, err := d.Send(AppNIC, NICLongXmit, initialData, waitTime)
	if err != nil {
		return fmt.Errorf("long transmit init failed: %w", err)
	}

	if len(response) > 0 && response[0] != 0 {
		return fmt.Errorf("long transmit init error: 0x%02X", response[0])
	}

	// Send remaining chunks
	for chIdx := preload; chIdx < len(chunks); chIdx++ {
		chunk := chunks[chIdx]

		// Retry loop for buffer availability
		for retries := 0; retries < 100; retries++ {
			payload := make([]byte, 1+len(chunk))
			payload[0] = byte(len(chunk))
			copy(payload[1:], chunk)

			response, err = d.Send(AppNIC, NICLongXmitMore, payload, USBTXWaitTimeout)
			if err != nil {
				return fmt.Errorf("long transmit chunk %d failed: %w", chIdx, err)
			}

			if len(response) > 0 {
				if response[0] == RCTempErrBufferNotAvailable {
					time.Sleep(1 * time.Millisecond)
					continue
				}
				if response[0] != 0 {
					return fmt.Errorf("long transmit chunk %d error: 0x%02X", chIdx, response[0])
				}
			}
			break
		}
	}

	// Signal completion with zero-length chunk
	response, err = d.Send(AppNIC, NICLongXmitMore, []byte{0}, USBTXWaitTimeout)
	if err != nil {
		return fmt.Errorf("long transmit completion failed: %w", err)
	}

	if len(response) > 0 && response[0] != 0 {
		return fmt.Errorf("long transmit completion error: 0x%02X", response[0])
	}

	return nil
}

// RFRecv receives RF data with timeout
// Returns the received data and any error
// Set blocksize > 255 for large packet mode (max 512)
func (d *Device) RFRecv(timeout time.Duration, blocksize uint16) ([]byte, error) {
	// Configure large block receive if needed
	if blocksize > 255 {
		if blocksize > RFMaxRXBlock {
			return nil, fmt.Errorf("blocksize %d exceeds maximum %d", blocksize, RFMaxRXBlock)
		}
		payload := make([]byte, 2)
		binary.LittleEndian.PutUint16(payload, blocksize)
		_, err := d.Send(AppNIC, NICSetRecvLarge, payload, USBDefaultTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to set large receive mode: %w", err)
		}
	}

	// Receive packet from NIC
	data, err := d.Recv(AppNIC, NICRecv, timeout)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// RFRecvLoop continuously receives RF packets and sends them to a channel
// Stops when the stop channel is closed or receives a value
func (d *Device) RFRecvLoop(timeout time.Duration, packets chan<- []byte, stop <-chan struct{}) error {
	// Ensure we're in RX mode
	if err := d.SetModeRX(); err != nil {
		return fmt.Errorf("failed to enter RX mode: %w", err)
	}

	for {
		select {
		case <-stop:
			return nil
		default:
			data, err := d.RFRecv(timeout, 0)
			if err != nil {
				// Timeout is normal, continue waiting
				continue
			}
			// Non-blocking send to channel
			select {
			case packets <- data:
			default:
				// Channel full, drop packet
			}
		}
	}
}

// SetRecvLargeMode configures the device for large packet reception
// Set blocksize to 0 to disable large mode
func (d *Device) SetRecvLargeMode(blocksize uint16) error {
	payload := make([]byte, 2)
	binary.LittleEndian.PutUint16(payload, blocksize)
	_, err := d.Send(AppNIC, NICSetRecvLarge, payload, USBDefaultTimeout)
	if err != nil {
		return fmt.Errorf("failed to set receive blocksize: %w", err)
	}
	return nil
}

// GetRSSI returns the current RSSI (Received Signal Strength Indicator) value
// Returns raw register value; convert to dBm: rssi_dBm = (rssi - 74) for most cases
func (d *Device) GetRSSI() (uint8, error) {
	return d.PeekByte(0xDF3A) // RegRSSI
}

// GetLQI returns the Link Quality Indicator
// Lower values indicate better link quality
// Bit 7 (0x80) indicates CRC OK when set
func (d *Device) GetLQI() (uint8, error) {
	return d.PeekByte(0xDF39) // RegLQI
}

// GetPKTSTATUS returns the packet status register
// Bit 7: CRC_OK, Bit 6: CS (Carrier Sense), Bit 5: PQT_REACHED
// Bit 4: CCA, Bit 3: SFD, Bit 2: GDO2, Bit 1: reserved, Bit 0: GDO0
func (d *Device) GetPKTSTATUS() (uint8, error) {
	return d.PeekByte(0xDF3C) // RegPKTSTATUS
}

// RSSIToDBm converts raw RSSI register value to dBm
// The offset depends on data rate, but -74 is typical for many configurations
func RSSIToDBm(rssi uint8) int {
	// RSSI is a signed value in 0.5 dBm steps with offset
	if rssi >= 128 {
		return int(rssi) - 256 - 74
	}
	return int(rssi) - 74
}

// RadioStatus holds diagnostic information about received packets
type RadioStatus struct {
	RSSI      uint8
	RSSIdBm   int
	LQI       uint8
	CRCOk     bool
	MARCSTATE uint8
	PKTSTATUS uint8
}

// GetRadioStatus reads current radio status registers
func (d *Device) GetRadioStatus() (*RadioStatus, error) {
	rssi, err := d.GetRSSI()
	if err != nil {
		return nil, fmt.Errorf("failed to read RSSI: %w", err)
	}

	lqi, err := d.GetLQI()
	if err != nil {
		return nil, fmt.Errorf("failed to read LQI: %w", err)
	}

	marcstate, err := d.GetMARCSTATE()
	if err != nil {
		return nil, fmt.Errorf("failed to read MARCSTATE: %w", err)
	}

	pktstatus, err := d.GetPKTSTATUS()
	if err != nil {
		return nil, fmt.Errorf("failed to read PKTSTATUS: %w", err)
	}

	return &RadioStatus{
		RSSI:      rssi,
		RSSIdBm:   RSSIToDBm(rssi),
		LQI:       lqi & 0x7F, // Lower 7 bits are LQI
		CRCOk:     (lqi & 0x80) != 0,
		MARCSTATE: marcstate,
		PKTSTATUS: pktstatus,
	}, nil
}
