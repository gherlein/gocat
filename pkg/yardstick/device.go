package yardstick

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/gousb"
)

// Device represents a YardStick One USB device
type Device struct {
	usbDevice    *gousb.Device
	usbConfig    *gousb.Config
	usbInterface *gousb.Interface
	epIn         *gousb.InEndpoint
	epOut        *gousb.OutEndpoint
	Serial       string
	Manufacturer string
	Product      string
	Bus          int
	Address      int
	recvBuf      []byte
	recvMu       sync.Mutex
}

// FindAllDevices finds all connected YardStick One devices
func FindAllDevices(context *gousb.Context) ([]*Device, error) {
	devices := []*Device{}

	usbDevices, err := context.OpenDevices(func(descriptor *gousb.DeviceDesc) bool {
		return descriptor.Vendor == gousb.ID(VendorID) && descriptor.Product == gousb.ID(ProductID)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate devices: %w", err)
	}

	for _, usbDev := range usbDevices {
		device, err := wrapDevice(usbDev)
		if err != nil {
			usbDev.Close()
			continue
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// OpenDevice opens a specific YardStick One device by serial number
func OpenDevice(context *gousb.Context, serial string) (*Device, error) {
	usbDev, err := context.OpenDeviceWithVIDPID(gousb.ID(VendorID), gousb.ID(ProductID))
	if err != nil {
		return nil, fmt.Errorf("failed to open device: %w", err)
	}
	if usbDev == nil {
		return nil, fmt.Errorf("device not found")
	}

	device, err := wrapDevice(usbDev)
	if err != nil {
		usbDev.Close()
		return nil, err
	}

	if serial != "" && device.Serial != serial {
		device.Close()
		return nil, fmt.Errorf("device serial mismatch: wanted %s, got %s", serial, device.Serial)
	}

	return device, nil
}

func wrapDevice(usbDev *gousb.Device) (*Device, error) {
	manufacturer, _ := usbDev.Manufacturer()
	product, _ := usbDev.Product()
	serial, _ := usbDev.SerialNumber()

	usbDev.SetAutoDetach(true)

	config, err := usbDev.Config(1)
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration: %w", err)
	}

	iface, err := config.Interface(0, 0)
	if err != nil {
		config.Close()
		return nil, fmt.Errorf("failed to claim interface: %w", err)
	}

	// Get EP5 IN endpoint (0x85)
	epIn, err := iface.InEndpoint(5)
	if err != nil {
		iface.Close()
		config.Close()
		return nil, fmt.Errorf("failed to get IN endpoint: %w", err)
	}

	// Get EP5 OUT endpoint (0x05)
	epOut, err := iface.OutEndpoint(5)
	if err != nil {
		iface.Close()
		config.Close()
		return nil, fmt.Errorf("failed to get OUT endpoint: %w", err)
	}

	desc := usbDev.Desc
	device := &Device{
		usbDevice:    usbDev,
		usbConfig:    config,
		usbInterface: iface,
		epIn:         epIn,
		epOut:        epOut,
		Serial:       serial,
		Manufacturer: manufacturer,
		Product:      product,
		Bus:          desc.Bus,
		Address:      desc.Address,
		recvBuf:      make([]byte, 0, EP5OutBufferSize),
	}

	// Drain any stale data from the receive endpoint
	device.drainReceiveBuffer()

	return device, nil
}

// Control performs a USB control transfer (for EP0 vendor commands)
func (d *Device) Control(requestType uint8, request uint8, value uint16, index uint16, data []byte) (int, error) {
	return d.usbDevice.Control(requestType, request, value, index, data)
}

// Close closes the device and releases all resources
func (d *Device) Close() error {
	// Try to put radio back to IDLE state before closing
	// This ensures the device is in a known state for next use
	if d.epOut != nil {
		d.setRadioIDLE()
	}

	if d.usbInterface != nil {
		d.usbInterface.Close()
	}
	if d.usbConfig != nil {
		d.usbConfig.Close()
	}
	if d.usbDevice != nil {
		return d.usbDevice.Close()
	}
	return nil
}

// drainReceiveBuffer reads and discards any stale data from the receive endpoint
// This is called on device open to clear any data left from previous sessions
func (d *Device) drainReceiveBuffer() {
	buf := make([]byte, 512)
	// Do a few quick reads with very short timeout to clear any pending data
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		n, err := d.epIn.ReadContext(ctx, buf)
		cancel()
		if err != nil || n == 0 {
			break // No more data or error, we're done
		}
	}
	// Clear internal buffer as well
	d.recvBuf = d.recvBuf[:0]
}

// RecoverUSB attempts to recover USB communication after failures
// This drains buffers and performs a brief reset sequence
func (d *Device) RecoverUSB() error {
	d.recvMu.Lock()
	defer d.recvMu.Unlock()

	// Wait a bit to let any pending transfers complete/timeout
	time.Sleep(50 * time.Millisecond)

	// Drain any pending data
	buf := make([]byte, 512)
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		_, err := d.epIn.ReadContext(ctx, buf)
		cancel()
		if err != nil {
			break
		}
	}

	// Clear internal buffer
	d.recvBuf = d.recvBuf[:0]

	// Wait again
	time.Sleep(50 * time.Millisecond)

	// Try a simple ping to verify communication is working
	testData := []byte{0x55, 0xAA}
	_, err := d.Send(AppSystem, SysCmdPing, testData, 500*time.Millisecond)
	if err != nil {
		return fmt.Errorf("USB recovery failed: ping test failed: %w", err)
	}

	return nil
}

// setRadioIDLE puts the radio into IDLE state using direct register poke
// This is a simplified version that doesn't wait for response, used during cleanup
func (d *Device) setRadioIDLE() {
	// Build POKE command for RFST register (0xDFE1) with SIDLE value (0x04)
	payload := make([]byte, 3)
	binary.LittleEndian.PutUint16(payload[0:2], 0xDFE1) // RFST register address
	payload[2] = 0x04                                   // SIDLE strobe

	packet := make([]byte, 4+len(payload))
	packet[0] = AppSystem
	packet[1] = SysCmdPoke
	binary.LittleEndian.PutUint16(packet[2:4], uint16(len(payload)))
	copy(packet[4:], payload)

	// Send without waiting for response (best effort during cleanup)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	d.epOut.WriteContext(ctx, packet)
}

// String returns a human-readable description of the device
func (d *Device) String() string {
	return fmt.Sprintf("%s %s (Serial: %s)", d.Manufacturer, d.Product, d.Serial)
}

// Send sends a command to the device via EP5 and waits for response
// Protocol: app(1) + cmd(1) + length(2 LE) + payload
func (d *Device) Send(app uint8, cmd uint8, payload []byte, timeout time.Duration) ([]byte, error) {
	if timeout == 0 {
		timeout = USBDefaultTimeout
	}

	// Build the command packet
	packet := make([]byte, 4+len(payload))
	packet[0] = app
	packet[1] = cmd
	binary.LittleEndian.PutUint16(packet[2:4], uint16(len(payload)))
	if len(payload) > 0 {
		copy(packet[4:], payload)
	}

	// Send the packet with timeout
	writeCtx, writeCancel := context.WithTimeout(context.Background(), timeout)
	n, err := d.epOut.WriteContext(writeCtx, packet)
	writeCancel()
	if err != nil {
		// Check if it was a timeout/cancellation
		if writeCtx.Err() != nil {
			return nil, fmt.Errorf("write timeout: %w", err)
		}
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "cancel") || strings.Contains(errStr, "timeout") {
			return nil, fmt.Errorf("write timeout: %w", err)
		}
		return nil, fmt.Errorf("failed to write to EP5: %w", err)
	}
	if n != len(packet) {
		return nil, fmt.Errorf("short write: wrote %d of %d bytes", n, len(packet))
	}

	// Read the response
	return d.Recv(app, cmd, timeout)
}

// Recv reads a response from the device via EP5
// Response format: '@'(1) + app(1) + cmd(1) + length(2 LE) + payload
func (d *Device) Recv(expectedApp uint8, expectedCmd uint8, timeout time.Duration) ([]byte, error) {
	d.recvMu.Lock()
	defer d.recvMu.Unlock()

	if timeout == 0 {
		timeout = USBDefaultTimeout
	}

	deadline := time.Now().Add(timeout)
	buf := make([]byte, 512) // Match Python's buffer size

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for response")
		}

		// First check if we already have a complete response buffered
		response, remaining, err := d.parseResponse(expectedApp, expectedCmd)
		if err == nil {
			d.recvBuf = remaining
			return response, nil
		}

		// Calculate remaining time for this read operation
		remaining_time := time.Until(deadline)
		if remaining_time <= 0 {
			return nil, fmt.Errorf("timeout waiting for response")
		}

		// Use a shorter read timeout (100ms) to allow periodic deadline checks
		readTimeout := 100 * time.Millisecond
		if remaining_time < readTimeout {
			readTimeout = remaining_time
		}

		// Read from EP5 with context timeout
		ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
		n, err := d.epIn.ReadContext(ctx, buf)
		cancel()

		if err != nil {
			// Check if it's a timeout/canceled error (normal, just retry)
			if ctx.Err() != nil {
				// Context was canceled or timed out, this is expected
				continue
			}
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "timeout") ||
				strings.Contains(errStr, "timed out") ||
				strings.Contains(errStr, "canceled") ||
				strings.Contains(errStr, "context") ||
				strings.Contains(errStr, "libusb") {
				continue
			}
			return nil, fmt.Errorf("failed to read from EP5: %w", err)
		}

		if n == 0 {
			continue
		}

		// Append to receive buffer
		d.recvBuf = append(d.recvBuf, buf[:n]...)
	}
}

// parseResponse attempts to parse a complete response from the buffer
func (d *Device) parseResponse(expectedApp uint8, expectedCmd uint8) ([]byte, []byte, error) {
	// Find the response marker '@'
	markerIdx := -1
	for i, b := range d.recvBuf {
		if b == ResponseMarker {
			markerIdx = i
			break
		}
	}

	if markerIdx == -1 {
		return nil, d.recvBuf, fmt.Errorf("no response marker found")
	}

	// Discard any data before the marker
	data := d.recvBuf[markerIdx:]

	// Need at least 5 bytes for header: marker + app + cmd + length(2)
	if len(data) < 5 {
		return nil, d.recvBuf, fmt.Errorf("incomplete header")
	}

	// Parse header
	app := data[1]
	cmd := data[2]
	length := binary.LittleEndian.Uint16(data[3:5])

	// Check if we have the complete payload
	totalLen := 5 + int(length)
	if len(data) < totalLen {
		return nil, d.recvBuf, fmt.Errorf("incomplete payload: have %d, need %d", len(data), totalLen)
	}

	// Verify app and cmd match (optional, but useful for debugging)
	if app != expectedApp || cmd != expectedCmd {
		// This might be a different response, skip it and look for another
		return nil, d.recvBuf[markerIdx+1:], fmt.Errorf("response mismatch: got app=0x%02X cmd=0x%02X, expected app=0x%02X cmd=0x%02X",
			app, cmd, expectedApp, expectedCmd)
	}

	// Extract payload
	payload := make([]byte, length)
	copy(payload, data[5:totalLen])

	// Return remaining data
	remaining := data[totalLen:]
	return payload, remaining, nil
}

// RecvFromApp receives data from a specific application and queue
// This is used for spectrum analyzer data which comes from APP_SPECAN
func (d *Device) RecvFromApp(app uint8, queue uint8, timeout time.Duration) ([]byte, error) {
	d.recvMu.Lock()
	defer d.recvMu.Unlock()

	if timeout == 0 {
		timeout = USBDefaultTimeout
	}

	deadline := time.Now().Add(timeout)
	buf := make([]byte, 512)

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for app 0x%02X data", app)
		}

		// Check if we already have a matching response buffered
		response, remaining, err := d.parseResponseFromApp(app, queue)
		if err == nil {
			d.recvBuf = remaining
			return response, nil
		}

		// Calculate remaining time
		remainingTime := time.Until(deadline)
		if remainingTime <= 0 {
			return nil, fmt.Errorf("timeout waiting for app 0x%02X data", app)
		}

		readTimeout := 100 * time.Millisecond
		if remainingTime < readTimeout {
			readTimeout = remainingTime
		}

		ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
		n, err := d.epIn.ReadContext(ctx, buf)
		cancel()

		if err != nil {
			if ctx.Err() != nil {
				continue
			}
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "canceled") {
				continue
			}
			return nil, fmt.Errorf("failed to read from EP5: %w", err)
		}

		if n > 0 {
			d.recvBuf = append(d.recvBuf, buf[:n]...)
		}
	}
}

// parseResponseFromApp parses a response for a specific app/queue
func (d *Device) parseResponseFromApp(app uint8, queue uint8) ([]byte, []byte, error) {
	// Find the response marker '@'
	markerIdx := -1
	for i, b := range d.recvBuf {
		if b == ResponseMarker {
			markerIdx = i
			break
		}
	}

	if markerIdx == -1 {
		return nil, d.recvBuf, fmt.Errorf("no response marker found")
	}

	data := d.recvBuf[markerIdx:]

	// Need at least 5 bytes for header: marker + app + cmd + length(2)
	if len(data) < 5 {
		return nil, d.recvBuf, fmt.Errorf("incomplete header")
	}

	respApp := data[1]
	respQueue := data[2]
	length := binary.LittleEndian.Uint16(data[3:5])

	totalLen := 5 + int(length)
	if len(data) < totalLen {
		return nil, d.recvBuf, fmt.Errorf("incomplete payload")
	}

	// Check if this matches what we're looking for
	if respApp != app || respQueue != queue {
		// Skip this response and look for another
		return nil, d.recvBuf[markerIdx+1:], fmt.Errorf("app/queue mismatch")
	}

	payload := make([]byte, length)
	copy(payload, data[5:totalLen])

	remaining := data[totalLen:]
	return payload, remaining, nil
}

// Ping sends a ping command and verifies the response
func (d *Device) Ping(data []byte) error {
	response, err := d.Send(AppSystem, SysCmdPing, data, USBDefaultTimeout)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	if len(response) != len(data) {
		return fmt.Errorf("ping response length mismatch: sent %d bytes, got %d", len(data), len(response))
	}

	for i := range data {
		if response[i] != data[i] {
			return fmt.Errorf("ping response data mismatch at byte %d: sent 0x%02X, got 0x%02X", i, data[i], response[i])
		}
	}

	return nil
}

// Peek reads bytes from device memory using EP5 protocol
func (d *Device) Peek(address uint16, length uint16) ([]byte, error) {
	// Payload: bytecount(2 LE) + address(2 LE)
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint16(payload[0:2], length)
	binary.LittleEndian.PutUint16(payload[2:4], address)

	response, err := d.Send(AppSystem, SysCmdPeek, payload, USBDefaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("peek failed at 0x%04X: %w", address, err)
	}

	return response, nil
}

// PeekByte reads a single byte from device memory
func (d *Device) PeekByte(address uint16) (uint8, error) {
	data, err := d.Peek(address, 1)
	if err != nil {
		return 0, err
	}
	if len(data) < 1 {
		return 0, fmt.Errorf("peek returned no data")
	}
	return data[0], nil
}

// Poke writes bytes to device memory using EP5 protocol
func (d *Device) Poke(address uint16, data []byte) error {
	// Payload: address(2 LE) + data
	payload := make([]byte, 2+len(data))
	binary.LittleEndian.PutUint16(payload[0:2], address)
	copy(payload[2:], data)

	response, err := d.Send(AppSystem, SysCmdPoke, payload, USBDefaultTimeout)
	if err != nil {
		return fmt.Errorf("poke failed at 0x%04X: %w", address, err)
	}

	// Response contains bytes left (should be 0 on success)
	if len(response) >= 2 {
		bytesLeft := binary.LittleEndian.Uint16(response[0:2])
		if bytesLeft != 0 {
			return fmt.Errorf("poke incomplete: %d bytes left", bytesLeft)
		}
	}

	return nil
}

// PokeByte writes a single byte to device memory
func (d *Device) PokeByte(address uint16, value uint8) error {
	return d.Poke(address, []byte{value})
}

// GetBuildType returns the firmware build type string
func (d *Device) GetBuildType() (string, error) {
	response, err := d.Send(AppSystem, SysCmdBuildType, nil, USBDefaultTimeout)
	if err != nil {
		return "", fmt.Errorf("failed to get build type: %w", err)
	}

	// Trim null terminator if present
	for i, b := range response {
		if b == 0 {
			return string(response[:i]), nil
		}
	}
	return string(response), nil
}

// GetPartNum returns the chip part number
func (d *Device) GetPartNum() (uint8, error) {
	response, err := d.Send(AppSystem, SysCmdPartNum, nil, USBDefaultTimeout)
	if err != nil {
		return 0, fmt.Errorf("failed to get part number: %w", err)
	}
	if len(response) < 1 {
		return 0, fmt.Errorf("empty part number response")
	}
	return response[0], nil
}

// GetCompiler returns the compiler version string
func (d *Device) GetCompiler() (string, error) {
	response, err := d.Send(AppSystem, SysCmdCompiler, nil, USBDefaultTimeout)
	if err != nil {
		return "", fmt.Errorf("failed to get compiler: %w", err)
	}

	// Trim null terminator if present
	for i, b := range response {
		if b == 0 {
			return string(response[:i]), nil
		}
	}
	return string(response), nil
}

// SetRFMode sets the radio mode (RX, TX, IDLE)
func (d *Device) SetRFMode(mode uint8) error {
	_, err := d.Send(AppSystem, SysCmdRFMode, []byte{mode}, USBDefaultTimeout)
	if err != nil {
		return fmt.Errorf("failed to set RF mode: %w", err)
	}
	return nil
}

// SetLEDMode sets the LED mode
func (d *Device) SetLEDMode(mode uint8) error {
	_, err := d.Send(AppSystem, SysCmdLEDMode, []byte{mode}, USBDefaultTimeout)
	if err != nil {
		return fmt.Errorf("failed to set LED mode: %w", err)
	}
	return nil
}

// EP0PeekX reads from XDATA memory using EP0 control transfer (alternative method)
func (d *Device) EP0PeekX(address uint16, length uint16) ([]byte, error) {
	data := make([]byte, length)
	_, err := d.Control(RequestTypeVendorIn, EP0CmdPeekX, address, 0, data)
	if err != nil {
		return nil, fmt.Errorf("EP0 peek failed at 0x%04X: %w", address, err)
	}
	return data, nil
}

// EP0PokeX writes to XDATA memory using EP0 control transfer (alternative method)
func (d *Device) EP0PokeX(address uint16, data []byte) error {
	_, err := d.Control(RequestTypeVendorOut, EP0CmdPokeX, address, 0, data)
	if err != nil {
		return fmt.Errorf("EP0 poke failed at 0x%04X: %w", address, err)
	}
	return nil
}

// GetDebugCodes returns the last debug/error codes from the device
func (d *Device) GetDebugCodes() (uint8, uint8, error) {
	data := make([]byte, 2)
	_, err := d.Control(RequestTypeVendorIn, EP0CmdGetDebugCodes, 0, 0, data)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get debug codes: %w", err)
	}
	return data[0], data[1], nil
}
