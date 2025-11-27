# GoCat Design Document

## Overview

GoCat is a Go SDK and toolkit for interacting with YardStick One (CC1111) USB radio devices. This design implements the initial configuration discovery and persistence system.

## Architecture

### Component Layers

```
┌─────────────────────────────────────┐
│     Example Programs (cmd/)         │
│  ys1-dump-config, ys1-load-config   │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│   Configuration Layer (pkg/config)  │
│  Serialization, File I/O            │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│  Register Access (pkg/registers)    │
│  Peek, Poke, Strobe, State Mgmt     │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│   Device Layer (pkg/yardstick)      │
│  USB Enumeration, Control Transfers │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│        gousb (USB Library)          │
└─────────────────────────────────────┘
```

## Directory Structure

```
/gocat/
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── PROJECT.md
├── docs/
│   ├── DESIGN.md
│   ├── usb-from-go.md
│   └── ys1-interfaces.md
├── etc/
│   └── yardsticks/           # Configuration files
│       └── <serial>.json
├── pkg/
│   ├── yardstick/            # Core device interface
│   │   ├── device.go         # Device enumeration and USB
│   │   └── constants.go      # VID/PID constants
│   ├── registers/            # Register access layer
│   │   ├── registers.go      # Register definitions
│   │   ├── access.go         # Peek/Poke implementation
│   │   └── state.go          # Radio state management
│   └── config/               # Configuration persistence
│       ├── config.go         # Configuration struct
│       └── storage.go        # Save/Load operations
└── cmd/
    ├── ys1-dump-config/      # Dump device config
    │   └── main.go
    └── ys1-load-config/      # Load config to device
        └── main.go
```

## Implementation Details

### 1. Device Layer (`pkg/yardstick`)

**Purpose**: USB device enumeration and low-level control transfer interface.

**Key Types**:
```go
type Device struct {
    usbDevice *gousb.Device
    usbConfig *gousb.Config
    usbInterface *gousb.Interface
    Serial string
    Manufacturer string
    Product string
}
```

**Key Functions**:
- `FindAllDevices(context) ([]*Device, error)` - Enumerate all YS1 devices (VID:PID 0x1d50:0x605b)
- `OpenDevice(context, serial) (*Device, error)` - Open specific device by serial
- `Control(requestType, request, value, index, data) (int, error)` - Raw control transfer wrapper
- `Close() error` - Release USB resources

**Constants**:
```go
const (
    VendorID  = 0x1d50
    ProductID = 0x605b
)
```

### 2. Register Access Layer (`pkg/registers`)

**Purpose**: CC1111-specific register operations with proper state management.

**Key Types**:
```go
type RegisterMap struct {
    // Radio configuration registers (0xDF00-0xDF3D)
    SYNC1       uint8
    SYNC0       uint8
    PKTLEN      uint8
    PKTCTRL1    uint8
    PKTCTRL0    uint8
    ADDR        uint8
    CHANNR      uint8
    // ... (all 62 registers)
}

type RadioState uint8

const (
    StateIDLE RadioState = 0x01
    StateRX   RadioState = 0x0D
    StateTX   RadioState = 0x13
)
```

**Key Functions**:
- `Peek(device, address) (uint8, error)` - Read single register
- `Poke(device, address, value) error` - Write single register
- `Strobe(device, command) error` - Send strobe command to RFST
- `GetRadioState(device) (RadioState, error)` - Read MARCSTATE
- `SetIDLE(device) error` - Strobe IDLE
- `ReadAllRegisters(device) (*RegisterMap, error)` - Dump all configuration
- `WriteAllRegisters(device, *RegisterMap) error` - Restore configuration

**Control Transfer Protocol**:
Based on CC1111 USB firmware protocol (reverse engineered from RfCat):
- Peek: `Control(0xC0, CMD_PEEK, 0, address, buffer[1])`
- Poke: `Control(0x40, CMD_POKE, value, address, nil)`

### 3. Configuration Layer (`pkg/config`)

**Purpose**: Serialize and persist device configurations.

**Key Types**:
```go
type DeviceConfig struct {
    Serial       string      `json:"serial"`
    Manufacturer string      `json:"manufacturer"`
    Product      string      `json:"product"`
    Timestamp    time.Time   `json:"timestamp"`
    Registers    RegisterMap `json:"registers"`
}
```

**Key Functions**:
- `DumpFromDevice(device) (*DeviceConfig, error)` - Read device state
- `ApplyToDevice(device, config) error` - Write device state
- `SaveToFile(config, path) error` - Persist as JSON
- `LoadFromFile(path) (*DeviceConfig, error)` - Load from JSON
- `GetConfigPath(serial) string` - Generate path `etc/yardsticks/<serial>.json`

### 4. Example Program: `ys1-dump-config`

**Purpose**: Enumerate YS1 devices and dump configurations.

**Behavior**:
1. Find all YardStick One devices on USB
2. For each device:
   - Read device info (serial, manufacturer, product)
   - Ensure radio in IDLE state
   - Read all registers (0xDF00-0xDF3D)
   - Restore original radio state
   - Save to `etc/yardsticks/<serial>.json`
3. Print summary

**Usage**:
```bash
$ ./bin/ys1-dump-config
Found 2 YardStick One devices
Dumping device: YARDSTICKONE-ABCD1234
  Configuration saved to etc/yardsticks/ABCD1234.json
Dumping device: YARDSTICKONE-EFGH5678
  Configuration saved to etc/yardsticks/EFGH5678.json
```

### 5. Example Program: `ys1-load-config` (Future)

**Purpose**: Restore device configurations from files.

**Behavior**:
1. Find device by serial or use first available
2. Load configuration from file
3. Set IDLE
4. Write all registers
5. Restore desired radio state

## Register Access Protocol

Based on RfCat analysis, CC1111 USB control transfers:

```go
// Vendor commands
const (
    CMD_PEEK   = 0x01  // Read register
    CMD_POKE   = 0x02  // Write register
    CMD_PING   = 0x00  // Verify connectivity
    CMD_STATUS = 0x04  // Get status
)

// Control transfer formats
// PEEK: bmRequestType=0xC0 (IN|Vendor|Device)
//       bRequest=0x01
//       wValue=0x0000
//       wIndex=register_address
//       data=buffer[1 byte]

// POKE: bmRequestType=0x40 (OUT|Vendor|Device)
//       bRequest=0x02
//       wValue=register_value
//       wIndex=register_address
//       data=nil
```

## Register Map Coverage

Full CC1111 radio configuration space (0xDF00-0xDF3D):

**Frequency Synthesis** (6 registers):
- FREQ2/1/0, FSCTRL1/0, FSCAL3/2/1/0

**Modulation** (6 registers):
- MDMCFG4/3/2/1/0, DEVIATN

**Packet Handling** (6 registers):
- SYNC1/0, PKTLEN, PKTCTRL1/0, ADDR, CHANNR

**State Machine** (3 registers):
- MCSM2/1/0

**AGC/Frontend** (9 registers):
- FOCCFG, BSCFG, AGCCTRL2/1/0, FREND1/0, TEST2/1/0

**Power** (8 registers):
- PA_TABLE[0-7]

**GPIO** (3 registers):
- IOCFG2/1/0

**Status** (7 read-only registers):
- PARTNUM, VERSION, FREQEST, LQI, RSSI, MARCSTATE, PKTSTATUS

## Configuration File Format

JSON format for human readability and easy editing:

```json
{
  "serial": "ABCD1234",
  "manufacturer": "Great Scott Gadgets",
  "product": "YARD Stick One",
  "timestamp": "2025-11-26T17:00:00Z",
  "registers": {
    "sync1": 211,
    "sync0": 145,
    "pktlen": 255,
    "pktctrl1": 4,
    "pktctrl0": 69,
    "addr": 0,
    "channr": 0,
    "freq2": 36,
    "freq1": 59,
    "freq0": 71,
    "mdmcfg4": 203,
    "mdmcfg3": 147,
    "mdmcfg2": 3,
    "mdmcfg1": 34,
    "mdmcfg0": 248
  }
}
```

## Error Handling Strategy

1. **Device Not Found**: Graceful exit with helpful message
2. **Permission Denied**: Suggest udev rule installation
3. **Register Access Failed**: Retry once, then fail with register address
4. **State Management**: Always attempt to restore original state on error
5. **File I/O**: Create directories if missing, fail on write errors

## Build System

**Makefile targets**:
```makefile
all: build
build: bin/ys1-dump-config bin/ys1-load-config
clean: # Remove bin/ and build artifacts
install: # Copy binaries to /usr/local/bin (optional)
test: # Run go test
fmt: # Run gofmt
```

**Binary output**: `bin/ys1-dump-config`, `bin/ys1-load-config`

## Dependencies

- `github.com/google/gousb` - USB device access
- Standard library: encoding/json, fmt, os, path/filepath, time

## Future Enhancements

1. Configuration diff tool
2. Live monitoring mode
3. Register change detection
4. Configuration templates
5. Device-to-device cloning
6. YAML format support
7. Validation of register values before write

## Testing Strategy

Integration test with two YS1 devices:
1. Dump config from device A
2. Modify specific registers
3. Load to device B
4. Verify device B matches expected config
5. Transmit from A, receive on B to verify functionality
