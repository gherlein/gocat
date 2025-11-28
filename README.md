# gocat

> **⚠️ EXPERIMENTAL** - This project is in early development with minimal testing beyond the included examples. The API will change. Use at your own risk. Pull requests welcome!

A Go library for controlling the [YardStick One](https://greatscottgadgets.com/yardstickone/) (YS1) sub-GHz RF transceiver, designed for building production RF tools and applications.

## Overview

gocat provides a native Go module for YardStick One hardware, enabling developers to build robust, deployable RF applications. It communicates with the CC1111-based RFCat firmware, allowing Go applications to transmit and receive on frequencies from 300-928 MHz.

### Goals

1. **Production-ready Go module**: A clean, well-tested library for embedding YS1 control in Go applications—security tools, IoT gateways, RF monitoring systems, automation infrastructure
2. **Single-binary deployment**: No Python runtime, no pip dependencies, no virtualenvs—just compile and run
3. **Concurrent by design**: Leverage Go's goroutines for efficient async TX/RX, multi-device coordination, and real-time packet processing

### Why Go over Python?

- **Deployment**: Ship a single static binary vs. managing Python environments
- **Performance**: Native compilation, no interpreter overhead for time-sensitive RF operations
- **Concurrency**: First-class goroutines vs. Python's GIL limitations
- **Integration**: Easy embedding in existing Go infrastructure (Kubernetes operators, network tools, security platforms)

## Relationship to RFCat

gocat uses the same CC1111 firmware as [RFCat](https://github.com/atlas0fd00m/rfcat)—we're reimplementing the host-side USB protocol in Go, not the device firmware. RFCat compatibility is useful for validation and testing, but the goal is a standalone Go library, not a Python replacement.

| Component | RFCat | gocat |
|-----------|-------|-------|
| Language | Python | Go |
| Firmware | CC1111 RFCat firmware | Same firmware |
| USB Protocol | EP5 bulk transfers | Same protocol |
| Goal | Interactive RF research | Embedded production tools |

gocat was developed by analyzing RFCat's source code and the CC1111 firmware to understand:
- USB command/response protocol format
- Radio register configuration sequences
- TX/RX state machine transitions
- Amplifier and power control

## Development with Claude

This project was developed collaboratively with [Claude](https://claude.ai), Anthropic's AI assistant, using [Claude Code](https://claude.com/claude-code). The development process involved:

1. **Protocol Analysis**: Claude read the RFCat Python source (`rflib/chipcon_usb.py`, `rflib/chipcon_nic.py`) and CC1111 firmware (`firmware/cc1111rf.c`, `firmware/chipcon_usb.c`) to understand the USB protocol and radio control sequences.

2. **Documentation**: Key findings were documented in `docs/`:
   - `rfcat-packet-format.md` - USB EP5 protocol specification
   - `recv-and-xmit.md` - TX/RX operation sequences
   - `defaults-in-rfcat.md` - Default register values
   - `configuration.md` - JSON configuration format

3. **Implementation**: Go code was written to replicate RFCat behavior:
   - USB device enumeration and control
   - Register read/write (peek/poke)
   - Radio mode control (RX/TX/IDLE)
   - Packet transmission and reception
   - YS1 amplifier control

4. **Debugging**: When issues arose (timeouts, state corruption, missing packets), Claude analyzed firmware behavior and suggested fixes based on how RFCat handles similar situations.

## Installation

```bash
# Clone the repository
git clone https://github.com/herlein/gocat
cd gocat

# Build all tools
make build

# Tools are placed in bin/
ls bin/
```

### Dependencies

- Go 1.21+
- libusb-1.0 development headers
- Linux udev rules for YardStick One (or run as root)

## Tools

| Tool | Description |
|------|-------------|
| `lsys1` | List connected YardStick One devices |
| `ys1-dump-config` | Dump device configuration to JSON |
| `ys1-load-config` | Load configuration from JSON |
| `test-configs` | Load config and verify it was applied |
| `send-recv` | Send or receive RF packets |
| `test-10-repeat` | Reliability test between two devices |

## Quick Start

### List Devices

```bash
./bin/lsys1
```

### Send and Receive

Terminal 1 (receiver):
```bash
./bin/send-recv -m recv -c etc/defaults.json -d "1:20" -v
```

Terminal 2 (sender):
```bash
./bin/send-recv -m send -c etc/defaults.json -d "1:19" -data "Hello World!"
```

### Reliability Testing

With two YS1 devices connected:
```bash
./bin/test-10-repeat -c etc/defaults.json -v
```

## Configuration

Radio settings are stored in JSON files. See `etc/defaults.json` for an example:

```json
{
  "registers": {
    "sync1": 211,
    "sync0": 145,
    "pktlen": 16,
    "freq2": 37,
    "freq1": 149,
    "freq0": 85,
    "mdmcfg2": 3,
    ...
  }
}
```

Key settings:
- **sync1/sync0**: 16-bit sync word for packet detection
- **pktlen**: Fixed packet length (bytes)
- **freq2/freq1/freq0**: Carrier frequency (901.999 MHz default)
- **mdmcfg2**: Modulation and sync mode (2-FSK, 30/32 sync)
- **pa_table**: TX power levels

## Library Usage

The `pkg/yardstick` module is designed for embedding in larger Go applications:

```go
package main

import (
    "log"
    "time"

    "github.com/google/gousb"
    "github.com/herlein/gocat/pkg/yardstick"
    "github.com/herlein/gocat/pkg/config"
)

func main() {
    ctx := gousb.NewContext()
    defer ctx.Close()

    // Open device by serial, bus:address, or index
    device, err := yardstick.SelectDevice(ctx, "")
    if err != nil {
        log.Fatal(err)
    }
    defer device.Close()

    // Load and apply configuration
    cfg, _ := config.LoadFromFile("etc/defaults.json")
    config.ApplyToDevice(device, cfg)

    // Enable YS1 front-end amplifiers
    device.SetAmpMode(1)

    // Receive packets
    device.SetModeRX()
    data, err := device.RFRecv(time.Second, 0)
    if err == nil {
        log.Printf("Received %d bytes: %x", len(data), data)
    }

    // Transmit packets
    device.RFXmit([]byte("Hello RF!"), 0, 0)
}
```

For multi-device scenarios (e.g., relay, monitoring), open multiple devices by serial number or bus:address and coordinate with goroutines.

## Project Structure

```
gocat/
├── cmd/                    # Command-line tools
│   ├── lsys1/             # Device listing
│   ├── send-recv/         # TX/RX utility
│   ├── test-10-repeat/    # Reliability testing
│   └── ...
├── pkg/
│   ├── yardstick/         # Core YS1 library
│   │   ├── device.go      # USB device handling
│   │   ├── radio.go       # RF operations
│   │   ├── selector.go    # Device selection
│   │   └── constants.go   # Protocol constants
│   ├── config/            # Configuration management
│   └── registers/         # CC1111 register definitions
├── etc/                   # Configuration files
├── docs/                  # Protocol documentation
└── Makefile
```

## Current Status

**This is experimental software.** It has only been tested with the included example programs (`send-recv`, `test-10-repeat`) on a single Linux machine with two YS1 devices. There are no unit tests, no CI, and the API is not stable.

- **Early development**: Core TX/RX works in basic scenarios
- **Minimal testing**: Only self-test examples, no production validation
- **Linux only**: Untested on macOS/Windows
- **Fixed-length packets only**: Variable-length mode not working
- **No advanced features**: Frequency hopping, AES encryption not implemented

## Roadmap

Priorities for production readiness:

- [ ] Comprehensive test coverage
- [ ] API documentation and examples
- [ ] Variable-length packet mode
- [ ] Robust error handling and recovery
- [ ] macOS/Windows support
- [ ] Performance benchmarking
- [ ] Frequency hopping support
- [ ] AES encryption support

## References

- [RFCat](https://github.com/atlas0fd00m/rfcat) - Original Python implementation
- [YardStick One](https://greatscottgadgets.com/yardstickone/) - Hardware documentation
- [CC1111 Datasheet](https://www.ti.com/product/CC1111) - Radio transceiver IC
- [gousb](https://github.com/google/gousb) - Go USB library

## License

MIT License - See LICENSE file for details.

## Acknowledgments

- **atlas0fd00m** and RFCat contributors for the CC1111 firmware and protocol reference
- **Great Scott Gadgets** for the YardStick One hardware
- **Anthropic** for Claude, which assisted in reverse-engineering the protocol and developing this implementation
