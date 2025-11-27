package yardstick

import (
	"encoding/binary"
	"fmt"
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

	return &Device{
		usbDevice:    usbDev,
		usbConfig:    config,
		usbInterface: iface,
		epIn:         epIn,
		epOut:        epOut,
		Serial:       serial,
		Manufacturer: manufacturer,
		Product:      product,
		recvBuf:      make([]byte, 0, EP5OutBufferSize),
	}, nil
}

// Control performs a USB control transfer (for EP0 vendor commands)
func (d *Device) Control(requestType uint8, request uint8, value uint16, index uint16, data []byte) (int, error) {
	return d.usbDevice.Control(requestType, request, value, index, data)
}

// Close closes the device and releases all resources
func (d *Device) Close() error {
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

	// Send the packet
	n, err := d.epOut.Write(packet)
	if err != nil {
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
	buf := make([]byte, EP5MaxPacketSize)

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for response")
		}

		// Read from EP5
		n, err := d.epIn.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("failed to read from EP5: %w", err)
		}

		if n == 0 {
			continue
		}

		// Append to receive buffer
		d.recvBuf = append(d.recvBuf, buf[:n]...)

		// Try to parse a complete response
		response, remaining, err := d.parseResponse(expectedApp, expectedCmd)
		if err != nil {
			// Not enough data yet, continue reading
			continue
		}

		// Save remaining data for next read
		d.recvBuf = remaining
		return response, nil
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
