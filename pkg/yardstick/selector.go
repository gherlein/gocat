package yardstick

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/gousb"
)

// DeviceSelector specifies how to identify a YardStick One device
// Supported formats:
//   - ""           : Use first available device
//   - "serial"     : Match by serial number (e.g., "009a")
//   - "bus:addr"   : Match by USB bus and address (e.g., "1:10")
//   - "#N"         : Use Nth device, 0-indexed (e.g., "#0", "#1")
type DeviceSelector string

// SelectDevice opens a YardStick One device matching the selector
func SelectDevice(context *gousb.Context, selector DeviceSelector) (*Device, error) {
	sel := string(selector)

	// Empty selector - use first device
	if sel == "" {
		return openFirstDevice(context)
	}

	// Index selector: #0, #1, etc.
	if strings.HasPrefix(sel, "#") {
		indexStr := sel[1:]
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			return nil, fmt.Errorf("invalid device index: %s", sel)
		}
		return openDeviceByIndex(context, index)
	}

	// Bus:Address selector: 1:10, 2:5, etc.
	if strings.Contains(sel, ":") {
		parts := strings.SplitN(sel, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid bus:address format: %s", sel)
		}
		bus, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid bus number: %s", parts[0])
		}
		addr, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid address number: %s", parts[1])
		}
		return openDeviceByBusAddr(context, bus, addr)
	}

	// Serial number selector
	return openDeviceBySerial(context, sel)
}

// openFirstDevice opens the first available YardStick One
func openFirstDevice(context *gousb.Context) (*Device, error) {
	devices, err := FindAllDevices(context)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no YardStick One devices found")
	}

	// Close all except the first
	for i := 1; i < len(devices); i++ {
		devices[i].Close()
	}

	return devices[0], nil
}

// openDeviceByIndex opens the Nth YardStick One (0-indexed)
func openDeviceByIndex(context *gousb.Context, index int) (*Device, error) {
	devices, err := FindAllDevices(context)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no YardStick One devices found")
	}
	if index < 0 || index >= len(devices) {
		// Close all devices
		for _, d := range devices {
			d.Close()
		}
		return nil, fmt.Errorf("device index %d out of range (found %d devices)", index, len(devices))
	}

	// Close all except the selected one
	for i, d := range devices {
		if i != index {
			d.Close()
		}
	}

	return devices[index], nil
}

// openDeviceByBusAddr opens a YardStick One by USB bus and address
func openDeviceByBusAddr(context *gousb.Context, bus, addr int) (*Device, error) {
	devices, err := FindAllDevices(context)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no YardStick One devices found")
	}

	var selected *Device
	for _, d := range devices {
		if d.Bus == bus && d.Address == addr {
			selected = d
		} else {
			d.Close()
		}
	}

	if selected == nil {
		return nil, fmt.Errorf("no YardStick One found at bus %d address %d", bus, addr)
	}

	return selected, nil
}

// openDeviceBySerial opens a YardStick One by serial number
func openDeviceBySerial(context *gousb.Context, serial string) (*Device, error) {
	devices, err := FindAllDevices(context)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no YardStick One devices found")
	}

	var matches []*Device
	for _, d := range devices {
		if d.Serial == serial {
			matches = append(matches, d)
		} else {
			d.Close()
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no YardStick One found with serial %s", serial)
	}

	if len(matches) > 1 {
		// Multiple devices with same serial - close all and return error
		for _, d := range matches {
			d.Close()
		}
		return nil, fmt.Errorf("multiple devices (%d) found with serial %s; use bus:addr format (e.g., 1:10) or index format (e.g., #0)", len(matches), serial)
	}

	return matches[0], nil
}

// ParseDeviceFlag is a helper for command-line flag parsing
// Returns usage string for the -d flag
func DeviceFlagUsage() string {
	return `Device selector. Formats:
    ""        - Use first available device
    "serial"  - Match by serial number (e.g., "009a")
    "bus:addr"- Match by USB location (e.g., "1:10")
    "#N"      - Use Nth device, 0-indexed (e.g., "#0", "#1")`
}
