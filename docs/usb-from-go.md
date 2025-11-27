# USB Device Access from Go

## Library

Use `github.com/google/gousb` - Go bindings for libusb-1.0.

```go
import "github.com/google/gousb"
```

## Device Enumeration

```go
context := gousb.NewContext()
defer context.Close()

devices, err := context.OpenDevices(func(descriptor *gousb.DeviceDesc) bool {
    // Return true to open matching devices
    return descriptor.Vendor == 0x1234 && descriptor.Product == 0x5678
})
defer func() {
    for _, device := range devices {
        device.Close()
    }
}()
```

List all devices:

```go
devices, _ := context.OpenDevices(func(descriptor *gousb.DeviceDesc) bool {
    fmt.Printf("VID:PID %04x:%04x\n", descriptor.Vendor, descriptor.Product)
    return false // Don't open, just enumerate
})
```

## USB Terminology

- **Configuration**: Top-level device state. Devices have ≥1 configurations, only one active at a time.
- **Interface**: Logical device function within a configuration (e.g., control vs data interface).
- **Alternate Setting**: Different modes of an interface (e.g., different bandwidth allocations).
- **Endpoint**: Unidirectional data pipe. Address format: `0x8N` for IN (device→host), `0x0N` for OUT (host→device).
- **Transfer Types**:
  - **Control**: Bidirectional configuration/status (endpoint 0)
  - **Bulk**: Large, non-time-critical transfers
  - **Interrupt**: Small, periodic transfers
  - **Isochronous**: Streaming with guaranteed bandwidth, no error recovery

## Opening a Specific Device

```go
context := gousb.NewContext()
defer context.Close()

device, err := context.OpenDeviceWithVIDPID(0x1234, 0x5678)
if err != nil {
    panic(err)
}
defer device.Close()

// Set active configuration (usually 1)
configuration, err := device.Config(1)
if err != nil {
    panic(err)
}
defer configuration.Close()

// Claim interface (required before endpoint access)
interface, err := configuration.Interface(0, 0) // interface 0, alternate setting 0
if err != nil {
    panic(err)
}
defer interface.Close()
```

Auto-detach kernel driver if necessary:

```go
device.SetAutoDetach(true)
```

## Endpoint Discovery

```go
for _, descriptor := range configuration.Desc.Interfaces[0].AltSettings[0].Endpoints {
    fmt.Printf("Endpoint 0x%02x: %s, %s, MaxPacketSize=%d\n",
        descriptor.Address,
        descriptor.Direction,      // "IN" or "OUT"
        descriptor.TransferType,   // "Control", "Bulk", "Interrupt", "Isochronous"
        descriptor.MaxPacketSize)
}
```

## Control Transfers (Configuration Commands)

Control transfers use endpoint 0. Standard format:

```go
// bmRequestType bitmap:
// [7]: Direction (0=OUT, 1=IN)
// [6:5]: Type (0=Standard, 1=Class, 2=Vendor, 3=Reserved)
// [4:0]: Recipient (0=Device, 1=Interface, 2=Endpoint, 3=Other)

// Example: Vendor-specific IN request to device
requestType := 0xC0 // 1100_0000 = IN | Vendor | Device

bytesTransferred, err := device.Control(
    requestType,    // bmRequestType
    0x01,          // bRequest (vendor-defined)
    0x0000,        // wValue (request-specific parameter)
    0x0000,        // wIndex (typically interface/endpoint index)
    buffer,        // data buffer (IN: receives data, OUT: sends data)
)

// Example: Vendor-specific OUT request
requestType := 0x40 // 0100_0000 = OUT | Vendor | Device
bytesTransferred, err := device.Control(0x40, 0x02, 0x0001, 0x0000, []byte{0xAA, 0xBB})
```

Common standard requests (Type=0):

```go
const (
    GET_STATUS        = 0x00
    CLEAR_FEATURE     = 0x01
    SET_FEATURE       = 0x03
    SET_ADDRESS       = 0x05
    GET_DESCRIPTOR    = 0x06
    SET_DESCRIPTOR    = 0x07
    GET_CONFIGURATION = 0x08
    SET_CONFIGURATION = 0x09
)
```

## Register Access

No built-in register abstraction. Implement via control transfers or bulk/interrupt transfers depending on device protocol.

Typical pattern for register read:

```go
func readRegister(device *gousb.Device, address uint16) (byte, error) {
    buffer := make([]byte, 1)
    _, err := device.Control(0xC0, VENDOR_READ_REG, 0, address, buffer)
    return buffer[0], err
}
```

Typical pattern for register write:

```go
func writeRegister(device *gousb.Device, address uint16, value byte) error {
    _, err := device.Control(0x40, VENDOR_WRITE_REG, uint16(value), address, nil)
    return err
}
```

## Bulk Transfers (Large Data)

```go
// Open endpoint
endpointIn, err := interface.InEndpoint(0x81)  // 0x81 = EP1 IN
if err != nil {
    panic(err)
}

// Read
buffer := make([]byte, endpointIn.Desc.MaxPacketSize)
bytesRead, err := endpointIn.Read(buffer)

// Write
endpointOut, err := interface.OutEndpoint(0x01)  // 0x01 = EP1 OUT
bytesWritten, err := endpointOut.Write([]byte{0x01, 0x02, 0x03})
```

## Interrupt Transfers (Periodic Data)

Same API as bulk transfers. USB controller handles scheduling.

```go
endpointIn, err := interface.InEndpoint(0x82)  // Interrupt IN endpoint
buffer := make([]byte, 64)
bytesRead, err := endpointIn.Read(buffer)
```

## Streaming Data

### Synchronous Streaming

```go
for {
    buffer := make([]byte, 16384)
    bytesRead, err := endpointIn.Read(buffer)
    if err != nil {
        break
    }
    processData(buffer[:bytesRead])
}
```

### Asynchronous Streaming (High Throughput)

Use separate goroutines:

```go
go func() {
    for {
        buffer := make([]byte, 16384)
        bytesRead, err := endpointIn.Read(buffer)
        if err != nil {
            return
        }
        dataChannel <- buffer[:bytesRead]
    }
}()

for data := range dataChannel {
    processData(data)
}
```

### Isochronous Transfers

Not directly supported in current gousb API. Requires libusb's async API via cgo:

```go
// #include <libusb-1.0/libusb.h>
import "C"

// Manual transfer allocation and submission required
```

## Timeouts

Set per-transfer:

```go
endpointIn.ReadTimeout = 5 * time.Second
endpointOut.WriteTimeout = 5 * time.Second
```

## Error Handling

Key errors from `gousb`:

- `gousb.ErrorNotFound`: Device/interface/endpoint not found
- `gousb.ErrorBusy`: Resource claimed by another process
- `gousb.ErrorTimeout`: Transfer timeout
- `gousb.ErrorPipe`: Endpoint halted (STALL condition)
- `gousb.ErrorOverflow`: More data received than buffer size
- `gousb.ErrorNoDevice`: Device disconnected

Clear STALL condition:

```go
err := endpointIn.ClearStall()
```

## Complete Example

```go
package main

import (
    "fmt"
    "github.com/google/gousb"
)

func main() {
    context := gousb.NewContext()
    defer context.Close()

    device, err := context.OpenDeviceWithVIDPID(0x1234, 0x5678)
    if err != nil {
        panic(err)
    }
    defer device.Close()

    device.SetAutoDetach(true)

    configuration, err := device.Config(1)
    if err != nil {
        panic(err)
    }
    defer configuration.Close()

    iface, err := configuration.Interface(0, 0)
    if err != nil {
        panic(err)
    }
    defer iface.Close()

    // Configuration via control transfer
    _, err = device.Control(0x40, 0x01, 0x0001, 0x0000, nil)
    if err != nil {
        panic(err)
    }

    // Stream data
    endpointIn, err := iface.InEndpoint(0x81)
    if err != nil {
        panic(err)
    }

    for {
        buffer := make([]byte, 512)
        bytesRead, err := endpointIn.Read(buffer)
        if err != nil {
            panic(err)
        }
        fmt.Printf("Read %d bytes: %x\n", bytesRead, buffer[:bytesRead])
    }
}
```

## Debugging

Enable libusb debug output:

```go
context.Debug(4) // 0=None, 1=Error, 2=Warning, 3=Info, 4=Debug
```

Query device strings:

```go
manufacturer, _ := device.Manufacturer()
product, _ := device.Product()
serial, _ := device.SerialNumber()
fmt.Printf("%s %s (%s)\n", manufacturer, product, serial)
```

## Permissions (Linux)

Create udev rule `/etc/udev/rules.d/99-custom-usb.rules`:

```
SUBSYSTEM=="usb", ATTR{idVendor}=="1234", ATTR{idProduct}=="5678", MODE="0666"
```

Reload: `sudo udevadm control --reload-rules && sudo udevadm trigger`
