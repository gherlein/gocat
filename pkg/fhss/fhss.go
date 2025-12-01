// Package fhss provides Frequency Hopping Spread Spectrum (FHSS) functionality
// for the YardStick One RF transceiver.
package fhss

import (
	"fmt"
	"sync"

	"github.com/herlein/gocat/pkg/yardstick"
)

// FHSS provides frequency hopping spread spectrum functionality
type FHSS struct {
	device   *yardstick.Device
	channels []uint8
	mu       sync.Mutex
}

// MACState represents the current FHSS MAC layer state
type MACState uint8

// MACData contains MAC layer timing and state information
type MACData struct {
	State            MACState // Current MAC state
	TxMsgIdx         uint8    // Current TX message buffer index
	TxMsgIdxDone     uint8    // Last completed TX message index
	CurChanIdx       uint16   // Current channel index in hop sequence
	NumChannels      uint16   // Total channels in sequence
	NumChannelHops   uint16   // Number of hops completed
	TLastHop         uint16   // Timer value at last hop
	TLastStateChange uint32   // Timer value at last state change
	MACThreshold     uint32   // MAC timing threshold
	MACTimer         uint32   // Current MAC timer value
}

// New creates a new FHSS controller for the given device
func New(device *yardstick.Device) *FHSS {
	return &FHSS{
		device:   device,
		channels: make([]uint8, 0),
	}
}

// SetChannels configures the channel hop sequence.
// The channels are indices into the frequency table; the actual frequency
// for channel N is: base_freq + (N * channel_spacing)
func (f *FHSS) SetChannels(channels []uint8) error {
	if len(channels) > yardstick.FHSSMaxChannels {
		return fmt.Errorf("too many channels: %d > %d", len(channels), yardstick.FHSSMaxChannels)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Build command: [num_channels_lo][num_channels_hi][channel_list...]
	data := make([]byte, 2+len(channels))
	data[0] = byte(len(channels) & 0xFF)
	data[1] = byte(len(channels) >> 8)
	copy(data[2:], channels)

	_, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSSetChannels, data, yardstick.USBDefaultTimeout)
	if err != nil {
		return err
	}

	f.channels = make([]uint8, len(channels))
	copy(f.channels, channels)
	return nil
}

// GetChannels returns the current channel hop sequence from the device
func (f *FHSS) GetChannels() ([]uint8, error) {
	resp, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSGetChannels, nil, yardstick.USBDefaultTimeout)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// StartHopping begins automatic frequency hopping using the Timer T2 interrupt
func (f *FHSS) StartHopping() error {
	_, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSStartHopping, nil, yardstick.USBDefaultTimeout)
	return err
}

// StopHopping stops automatic frequency hopping
func (f *FHSS) StopHopping() error {
	_, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSStopHopping, nil, yardstick.USBDefaultTimeout)
	return err
}

// NextChannel manually advances to the next channel in the hop sequence
func (f *FHSS) NextChannel() (uint8, error) {
	resp, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSNextChannel, nil, yardstick.USBDefaultTimeout)
	if err != nil {
		return 0, err
	}
	if len(resp) < 1 {
		return 0, fmt.Errorf("no channel returned")
	}
	return resp[0], nil
}

// ChangeChannel sets the radio to a specific channel index
func (f *FHSS) ChangeChannel(channel uint8) error {
	_, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSChangeChannel, []byte{channel}, yardstick.USBDefaultTimeout)
	return err
}

// GetState returns the current MAC state
func (f *FHSS) GetState() (MACState, error) {
	resp, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSGetState, nil, yardstick.USBDefaultTimeout)
	if err != nil {
		return 0, err
	}
	if len(resp) < 1 {
		return 0, fmt.Errorf("no state returned")
	}
	return MACState(resp[0]), nil
}

// SetState sets the MAC state
func (f *FHSS) SetState(state MACState) error {
	_, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSSetState, []byte{byte(state)}, yardstick.USBDefaultTimeout)
	return err
}

// Transmit sends data during FHSS operation using the FHSS_XMIT command
func (f *FHSS) Transmit(data []byte) error {
	if len(data) > yardstick.FHSSMaxTXMsgLen {
		return fmt.Errorf("data too large: %d > %d", len(data), yardstick.FHSSMaxTXMsgLen)
	}

	msg := make([]byte, 1+len(data))
	msg[0] = byte(len(data))
	copy(msg[1:], data)

	_, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSXmit, msg, yardstick.USBDefaultTimeout)
	return err
}

// StartSync begins synchronization to a hopping network with the given cell ID
func (f *FHSS) StartSync(cellID uint16) error {
	data := []byte{byte(cellID & 0xFF), byte(cellID >> 8)}
	_, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSStartSync, data, yardstick.USBDefaultTimeout)
	return err
}

// GetMACData returns detailed MAC layer information
func (f *FHSS) GetMACData() (*MACData, error) {
	resp, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSGetMACData, nil, yardstick.USBDefaultTimeout)
	if err != nil {
		return nil, err
	}

	// The MAC_DATA_t structure from firmware is roughly 24 bytes
	// Parse what we can based on the response size
	if len(resp) < 10 {
		return nil, fmt.Errorf("response too short: %d bytes", len(resp))
	}

	macData := &MACData{
		State:        MACState(resp[0]),
		TxMsgIdx:     resp[1],
		TxMsgIdxDone: resp[2],
		CurChanIdx:   uint16(resp[3]) | uint16(resp[4])<<8,
		NumChannels:  uint16(resp[5]) | uint16(resp[6])<<8,
	}

	if len(resp) >= 12 {
		macData.NumChannelHops = uint16(resp[7]) | uint16(resp[8])<<8
		macData.TLastHop = uint16(resp[9]) | uint16(resp[10])<<8
	}

	return macData, nil
}

// SetMACThreshold configures the MAC timing threshold (dwell time related)
func (f *FHSS) SetMACThreshold(threshold uint32) error {
	data := []byte{
		byte(threshold),
		byte(threshold >> 8),
		byte(threshold >> 16),
		byte(threshold >> 24),
	}
	_, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSSetMACThreshold, data, yardstick.USBDefaultTimeout)
	return err
}

// GetMACThreshold returns the current MAC timing threshold
func (f *FHSS) GetMACThreshold() (uint32, error) {
	resp, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSGetMACThreshold, nil, yardstick.USBDefaultTimeout)
	if err != nil {
		return 0, err
	}
	if len(resp) < 4 {
		return 0, fmt.Errorf("response too short: %d bytes", len(resp))
	}
	return uint32(resp[0]) | uint32(resp[1])<<8 | uint32(resp[2])<<16 | uint32(resp[3])<<24, nil
}

// SetMACPeriod configures the MAC period (dwell time)
func (f *FHSS) SetMACPeriod(period uint16) error {
	data := []byte{byte(period & 0xFF), byte(period >> 8)}
	_, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSSetMACPeriod, data, yardstick.USBDefaultTimeout)
	return err
}

// BecomeMaster sets this device as the sync master for an FHSS network
func (f *FHSS) BecomeMaster() error {
	return f.SetState(MACState(yardstick.MACStateSyncMaster))
}

// BecomeClient sets this device to sync mode to join an FHSS network
func (f *FHSS) BecomeClient() error {
	return f.SetState(MACState(yardstick.MACStateSynching))
}

// Stop returns to non-hopping mode
func (f *FHSS) Stop() error {
	if err := f.StopHopping(); err != nil {
		return err
	}
	return f.SetState(MACState(yardstick.MACStateNonHopping))
}

// String returns a string representation of the MAC state
func (s MACState) String() string {
	switch uint8(s) {
	case yardstick.MACStateNonHopping:
		return "NonHopping"
	case yardstick.MACStateDiscovery:
		return "Discovery"
	case yardstick.MACStateSynching:
		return "Synching"
	case yardstick.MACStateSynched:
		return "Synched"
	case yardstick.MACStateSyncMaster:
		return "SyncMaster"
	case yardstick.MACStateSyncingMaster:
		return "SyncingMaster"
	case yardstick.MACStateLongXmit:
		return "LongXmit"
	case yardstick.MACStateLongXmitFail:
		return "LongXmitFail"
	case yardstick.MACStatePrepSpecan:
		return "PrepSpecan"
	case yardstick.MACStateSpecan:
		return "Specan"
	default:
		return fmt.Sprintf("Unknown(0x%02X)", uint8(s))
	}
}
