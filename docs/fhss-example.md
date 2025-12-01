# FHSS Two-Radio Example

This document describes how to build a working Frequency Hopping Spread Spectrum (FHSS) example using two YardStick One devices.

## Overview

FHSS communication requires:
1. **Master** - Transmits timing beacons and defines the hop sequence
2. **Client** - Synchronizes to the master and follows the hop sequence

Both devices hop through the same channel sequence at the same rate, allowing them to communicate while spreading their RF energy across multiple frequencies.

## How FHSS Works in rfcat Firmware

### Channel Hopping Mechanism

1. **Timer T2** drives the hopping - fires at configurable intervals (default ~150ms)
2. When T2 fires, the firmware:
   - Increments `curChanIdx` (wraps at `NumChannelHops`)
   - Looks up `g_Channels[curChanIdx]` to get the physical channel number
   - Sets `CHANNR` register to that channel
   - Returns to RX mode
3. During each dwell period, the device can TX/RX on that channel

### Synchronization Process

1. **Master** enters `MAC_STATE_SYNCINGMASTER`
   - Transmits beacons on each channel as it hops
   - Beacon contains current channel index
2. **Client** enters `MAC_STATE_SYNCHING`
   - Listens on channel 0 for beacons
   - When beacon received, extracts timing offset
   - Adjusts T2 counter to align with master
   - Enters `MAC_STATE_SYNCHED`

---

## Example 1: FHSS Master (Beacon Transmitter)

### `cmd/fhss-master/main.go`

```go
// fhss-master is the master/beacon device for FHSS communication
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/gousb"
	"github.com/herlein/gocat/pkg/yardstick"
)

var (
	deviceSel   = flag.String("d", "", yardstick.DeviceFlagUsage())
	baseFreq    = flag.Float64("freq", 433.0, "Base frequency in MHz")
	numChannels = flag.Int("channels", 25, "Number of hop channels (1-255)")
	chanSpacing = flag.Float64("spacing", 0.2, "Channel spacing in MHz")
	dwellTime   = flag.Int("dwell", 150, "Dwell time per channel in ms")
	dataRate    = flag.Int("rate", 4800, "Data rate in baud")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "FHSS Master - Beacon Transmitter\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "This device becomes the timing master and transmits\n")
		fmt.Fprintf(os.Stderr, "synchronization beacons that clients can lock onto.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -freq 433.0 -channels 25 -spacing 0.2 -dwell 150\n", os.Args[0])
	}
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Open device
	fmt.Println("Opening YardStick One (Master)...")
	device, err := yardstick.SelectDevice(ctx, yardstick.DeviceSelector(*deviceSel))
	if err != nil {
		return fmt.Errorf("failed to open device: %w", err)
	}
	defer device.Close()

	fmt.Printf("Connected: %s\n", device)

	// Configure radio for FHSS
	if err := configureRadio(device); err != nil {
		return fmt.Errorf("configure radio: %w", err)
	}

	// Set up channel hop sequence
	if err := setupChannels(device); err != nil {
		return fmt.Errorf("setup channels: %w", err)
	}

	// Print configuration
	fmt.Printf("\nFHSS Master Configuration:\n")
	fmt.Printf("  Base Frequency: %.3f MHz\n", *baseFreq)
	fmt.Printf("  Channels:       %d\n", *numChannels)
	fmt.Printf("  Channel Spacing: %.3f MHz\n", *chanSpacing)
	fmt.Printf("  Frequency Range: %.3f - %.3f MHz\n",
		*baseFreq, *baseFreq+float64(*numChannels-1)**chanSpacing)
	fmt.Printf("  Dwell Time:     %d ms\n", *dwellTime)
	fmt.Printf("  Data Rate:      %d baud\n", *dataRate)
	fmt.Println()

	// Enter sync master mode and start beaconing
	if err := startMaster(device); err != nil {
		return fmt.Errorf("start master: %w", err)
	}

	fmt.Println("FHSS Master active - transmitting beacons...")
	fmt.Println("Press Ctrl+C to stop")

	// Wait for signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nStopping...")
	return stopMaster(device)
}

func configureRadio(device *yardstick.Device) error {
	// Set base frequency
	freqHz := uint32(*baseFreq * 1e6)
	if err := device.SetFrequency(freqHz); err != nil {
		return fmt.Errorf("set frequency: %w", err)
	}

	// Set channel spacing
	spacingHz := uint32(*chanSpacing * 1e6)
	if err := device.SetChannelSpacing(spacingHz); err != nil {
		return fmt.Errorf("set channel spacing: %w", err)
	}

	// Configure modulation (2-FSK is common for FHSS)
	// These would use existing register poke methods
	// For now, assume a sensible default configuration

	return nil
}

func setupChannels(device *yardstick.Device) error {
	// Build channel sequence
	// Simple linear sequence: 0, 1, 2, ... N-1
	// Real FHSS systems use pseudo-random sequences
	channels := make([]byte, *numChannels)
	for i := 0; i < *numChannels; i++ {
		channels[i] = byte(i)
	}

	// Send FHSS_SET_CHANNELS command
	// Format: [num_channels_lo][num_channels_hi][channel_list...]
	data := make([]byte, 2+len(channels))
	data[0] = byte(len(channels) & 0xFF)
	data[1] = byte(len(channels) >> 8)
	copy(data[2:], channels)

	_, err := device.Send(yardstick.AppNIC, yardstick.FHSSSetChannels, data, yardstick.USBDefaultTimeout)
	return err
}

func startMaster(device *yardstick.Device) error {
	// Set MAC state to SYNCINGMASTER
	// This starts the beacon transmission process
	_, err := device.Send(yardstick.AppNIC, yardstick.FHSSSetState,
		[]byte{yardstick.MACStateSyncingMaster}, yardstick.USBDefaultTimeout)
	if err != nil {
		return fmt.Errorf("set state: %w", err)
	}

	// Start hopping
	_, err = device.Send(yardstick.AppNIC, yardstick.FHSSStartHopping, nil, yardstick.USBDefaultTimeout)
	if err != nil {
		return fmt.Errorf("start hopping: %w", err)
	}

	return nil
}

func stopMaster(device *yardstick.Device) error {
	// Stop hopping
	device.Send(yardstick.AppNIC, yardstick.FHSSStopHopping, nil, yardstick.USBDefaultTimeout)

	// Return to non-hopping state
	device.Send(yardstick.AppNIC, yardstick.FHSSSetState,
		[]byte{yardstick.MACStateNonHopping}, yardstick.USBDefaultTimeout)

	return nil
}
```

---

## Example 2: FHSS Client (Synchronized Receiver)

### `cmd/fhss-client/main.go`

```go
// fhss-client synchronizes to an FHSS master and receives data
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/gousb"
	"github.com/herlein/gocat/pkg/yardstick"
)

var (
	deviceSel   = flag.String("d", "", yardstick.DeviceFlagUsage())
	baseFreq    = flag.Float64("freq", 433.0, "Base frequency in MHz (must match master)")
	numChannels = flag.Int("channels", 25, "Number of hop channels (must match master)")
	chanSpacing = flag.Float64("spacing", 0.2, "Channel spacing in MHz (must match master)")
	syncTimeout = flag.Duration("sync-timeout", 30*time.Second, "Timeout for synchronization")
	verbose     = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "FHSS Client - Synchronized Receiver\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "This device synchronizes to an FHSS master and receives\n")
		fmt.Fprintf(os.Stderr, "data while frequency hopping.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -freq 433.0 -channels 25 -spacing 0.2\n", os.Args[0])
	}
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Open device
	fmt.Println("Opening YardStick One (Client)...")
	device, err := yardstick.SelectDevice(ctx, yardstick.DeviceSelector(*deviceSel))
	if err != nil {
		return fmt.Errorf("failed to open device: %w", err)
	}
	defer device.Close()

	fmt.Printf("Connected: %s\n", device)

	// Configure radio (must match master)
	if err := configureRadio(device); err != nil {
		return fmt.Errorf("configure radio: %w", err)
	}

	// Set up channel sequence (must match master)
	if err := setupChannels(device); err != nil {
		return fmt.Errorf("setup channels: %w", err)
	}

	fmt.Printf("\nFHSS Client Configuration:\n")
	fmt.Printf("  Base Frequency:  %.3f MHz\n", *baseFreq)
	fmt.Printf("  Channels:        %d\n", *numChannels)
	fmt.Printf("  Channel Spacing: %.3f MHz\n", *chanSpacing)
	fmt.Printf("  Sync Timeout:    %v\n", *syncTimeout)
	fmt.Println()

	// Set up signal handling
	sigCtx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Synchronize to master
	fmt.Println("Searching for FHSS master...")
	if err := synchronize(sigCtx, device); err != nil {
		return fmt.Errorf("synchronize: %w", err)
	}

	fmt.Println("Synchronized! Now receiving...")

	// Receive loop
	return receiveLoop(sigCtx, device)
}

func configureRadio(device *yardstick.Device) error {
	// Set base frequency
	freqHz := uint32(*baseFreq * 1e6)
	if err := device.SetFrequency(freqHz); err != nil {
		return fmt.Errorf("set frequency: %w", err)
	}

	// Set channel spacing
	spacingHz := uint32(*chanSpacing * 1e6)
	if err := device.SetChannelSpacing(spacingHz); err != nil {
		return fmt.Errorf("set channel spacing: %w", err)
	}

	return nil
}

func setupChannels(device *yardstick.Device) error {
	// Must use same channel sequence as master
	channels := make([]byte, *numChannels)
	for i := 0; i < *numChannels; i++ {
		channels[i] = byte(i)
	}

	data := make([]byte, 2+len(channels))
	data[0] = byte(len(channels) & 0xFF)
	data[1] = byte(len(channels) >> 8)
	copy(data[2:], channels)

	_, err := device.Send(yardstick.AppNIC, yardstick.FHSSSetChannels, data, yardstick.USBDefaultTimeout)
	return err
}

func synchronize(ctx context.Context, device *yardstick.Device) error {
	// Create timeout context
	syncCtx, cancel := context.WithTimeout(ctx, *syncTimeout)
	defer cancel()

	// Enter sync mode - device will search for master beacons
	_, err := device.Send(yardstick.AppNIC, yardstick.FHSSStartSync,
		[]byte{0x00, 0x00}, // Cell ID = 0
		yardstick.USBDefaultTimeout)
	if err != nil {
		return fmt.Errorf("start sync: %w", err)
	}

	// Poll state until synchronized or timeout
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	startTime := time.Now()
	dots := 0

	for {
		select {
		case <-syncCtx.Done():
			// Stop sync attempt
			device.Send(yardstick.AppNIC, yardstick.FHSSSetState,
				[]byte{yardstick.MACStateNonHopping}, yardstick.USBDefaultTimeout)
			return fmt.Errorf("sync timeout after %v", *syncTimeout)

		case <-ticker.C:
			// Check current state
			resp, err := device.Send(yardstick.AppNIC, yardstick.FHSSGetState, nil, yardstick.USBDefaultTimeout)
			if err != nil {
				continue
			}

			if len(resp) > 0 {
				state := resp[0]

				if *verbose {
					fmt.Printf("State: 0x%02x\n", state)
				}

				if state == yardstick.MACStateSynched {
					elapsed := time.Since(startTime)
					fmt.Printf("\nSynchronized in %v\n", elapsed.Round(time.Millisecond))
					return nil
				}
			}

			// Progress indicator
			dots++
			if dots%10 == 0 {
				fmt.Print(".")
			}
		}
	}
}

func receiveLoop(ctx context.Context, device *yardstick.Device) error {
	fmt.Println("\nReceiving (Ctrl+C to stop)...")
	fmt.Println("─────────────────────────────")

	packetCount := 0

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("\n\nReceived %d packets\n", packetCount)
			// Stop hopping and return to normal mode
			device.Send(yardstick.AppNIC, yardstick.FHSSStopHopping, nil, yardstick.USBDefaultTimeout)
			device.Send(yardstick.AppNIC, yardstick.FHSSSetState,
				[]byte{yardstick.MACStateNonHopping}, yardstick.USBDefaultTimeout)
			return nil

		default:
			// Try to receive a packet
			data, err := device.Recv(500 * time.Millisecond)
			if err != nil {
				// Timeout is normal, continue
				continue
			}

			if len(data) > 0 {
				packetCount++
				timestamp := time.Now().Format("15:04:05.000")

				// Get current channel for display
				chanResp, _ := device.Send(yardstick.AppNIC, yardstick.FHSSGetState, nil, yardstick.USBDefaultTimeout)
				chanInfo := ""
				if len(chanResp) > 1 {
					chanInfo = fmt.Sprintf(" [ch:%d]", chanResp[1])
				}

				fmt.Printf("%s%s RX %d bytes: %x\n", timestamp, chanInfo, len(data), data)

				if *verbose {
					// Try to print as ASCII if printable
					printable := true
					for _, b := range data {
						if b < 32 || b > 126 {
							printable = false
							break
						}
					}
					if printable {
						fmt.Printf("         ASCII: %s\n", string(data))
					}
				}
			}
		}
	}
}
```

---

## Example 3: FHSS Transmitter (Sends Data While Hopping)

### `cmd/fhss-tx/main.go`

```go
// fhss-tx transmits data while synchronized to an FHSS network
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/gousb"
	"github.com/herlein/gocat/pkg/yardstick"
)

var (
	deviceSel   = flag.String("d", "", yardstick.DeviceFlagUsage())
	baseFreq    = flag.Float64("freq", 433.0, "Base frequency in MHz")
	numChannels = flag.Int("channels", 25, "Number of hop channels")
	chanSpacing = flag.Float64("spacing", 0.2, "Channel spacing in MHz")
	message     = flag.String("msg", "", "Message to send (interactive if empty)")
	repeat      = flag.Int("repeat", 1, "Number of times to repeat message")
	interval    = flag.Duration("interval", 500*time.Millisecond, "Interval between repeats")
	master      = flag.Bool("master", false, "Also act as sync master")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "FHSS Transmitter\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Transmits data while frequency hopping.\n")
		fmt.Fprintf(os.Stderr, "Can be combined with master mode or sync to existing master.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -master -msg \"Hello FHSS\"      # Master + transmit\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -msg \"Hello\" -repeat 10       # Client + repeat\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s                               # Interactive mode\n", os.Args[0])
	}
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := gousb.NewContext()
	defer ctx.Close()

	fmt.Println("Opening YardStick One (TX)...")
	device, err := yardstick.SelectDevice(ctx, yardstick.DeviceSelector(*deviceSel))
	if err != nil {
		return fmt.Errorf("failed to open device: %w", err)
	}
	defer device.Close()

	fmt.Printf("Connected: %s\n", device)

	// Configure radio
	if err := configureRadio(device); err != nil {
		return fmt.Errorf("configure: %w", err)
	}

	// Set up channels
	if err := setupChannels(device); err != nil {
		return fmt.Errorf("setup channels: %w", err)
	}

	// Start hopping (as master or sync to existing)
	if *master {
		fmt.Println("Starting as FHSS master...")
		if err := startAsMaster(device); err != nil {
			return err
		}
	} else {
		fmt.Println("Synchronizing to FHSS master...")
		if err := syncToMaster(device); err != nil {
			return err
		}
	}

	// Set up signal handling
	sigCtx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Transmit
	if *message != "" {
		// Single message mode
		return transmitMessage(sigCtx, device, *message, *repeat, *interval)
	}

	// Interactive mode
	return interactiveMode(sigCtx, device)
}

func configureRadio(device *yardstick.Device) error {
	freqHz := uint32(*baseFreq * 1e6)
	if err := device.SetFrequency(freqHz); err != nil {
		return err
	}

	spacingHz := uint32(*chanSpacing * 1e6)
	return device.SetChannelSpacing(spacingHz)
}

func setupChannels(device *yardstick.Device) error {
	channels := make([]byte, *numChannels)
	for i := 0; i < *numChannels; i++ {
		channels[i] = byte(i)
	}

	data := make([]byte, 2+len(channels))
	data[0] = byte(len(channels) & 0xFF)
	data[1] = byte(len(channels) >> 8)
	copy(data[2:], channels)

	_, err := device.Send(yardstick.AppNIC, yardstick.FHSSSetChannels, data, yardstick.USBDefaultTimeout)
	return err
}

func startAsMaster(device *yardstick.Device) error {
	// Enter sync master state
	_, err := device.Send(yardstick.AppNIC, yardstick.FHSSSetState,
		[]byte{yardstick.MACStateSyncMaster}, yardstick.USBDefaultTimeout)
	if err != nil {
		return err
	}

	// Start hopping
	_, err = device.Send(yardstick.AppNIC, yardstick.FHSSStartHopping, nil, yardstick.USBDefaultTimeout)
	return err
}

func syncToMaster(device *yardstick.Device) error {
	// Start sync process
	_, err := device.Send(yardstick.AppNIC, yardstick.FHSSStartSync,
		[]byte{0x00, 0x00}, yardstick.USBDefaultTimeout)
	if err != nil {
		return err
	}

	// Wait for sync (simplified - real code would poll state)
	fmt.Print("Syncing")
	for i := 0; i < 100; i++ {
		time.Sleep(100 * time.Millisecond)
		fmt.Print(".")

		resp, err := device.Send(yardstick.AppNIC, yardstick.FHSSGetState, nil, yardstick.USBDefaultTimeout)
		if err == nil && len(resp) > 0 && resp[0] == yardstick.MACStateSynched {
			fmt.Println(" OK")
			return nil
		}
	}

	return fmt.Errorf("sync timeout")
}

func transmitMessage(ctx context.Context, device *yardstick.Device, msg string, count int, interval time.Duration) error {
	data := []byte(msg)

	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Use FHSS_XMIT for transmission during hopping
		// Format: [length][data...]
		txData := make([]byte, 1+len(data))
		txData[0] = byte(len(data))
		copy(txData[1:], data)

		_, err := device.Send(yardstick.AppNIC, yardstick.FHSSXmit, txData, yardstick.USBDefaultTimeout)
		if err != nil {
			fmt.Printf("TX error: %v\n", err)
		} else {
			fmt.Printf("TX %d/%d: %s\n", i+1, count, msg)
		}

		if i < count-1 {
			time.Sleep(interval)
		}
	}

	return nil
}

func interactiveMode(ctx context.Context, device *yardstick.Device) error {
	fmt.Println("\nInteractive mode - type messages to transmit (Ctrl+C to exit)")
	fmt.Println("─────────────────────────────────────────────────────────────")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		msg := strings.TrimSpace(scanner.Text())
		if msg == "" {
			continue
		}

		data := []byte(msg)
		txData := make([]byte, 1+len(data))
		txData[0] = byte(len(data))
		copy(txData[1:], data)

		_, err := device.Send(yardstick.AppNIC, yardstick.FHSSXmit, txData, yardstick.USBDefaultTimeout)
		if err != nil {
			fmt.Printf("TX error: %v\n", err)
		} else {
			fmt.Printf("Sent: %s (%d bytes)\n", msg, len(data))
		}
	}

	return nil
}
```

---

## Running the Examples

### Hardware Setup

1. Connect two YardStick One devices to your computer
2. Ensure they are recognized (run `./bin/lsys1` to list)
3. Note their device identifiers (e.g., `#0` and `#1`, or by serial number)

### Terminal 1: Start Master

```bash
# Build the examples first
go build -o bin/fhss-master ./cmd/fhss-master
go build -o bin/fhss-client ./cmd/fhss-client
go build -o bin/fhss-tx ./cmd/fhss-tx

# Start the master on device #0
./bin/fhss-master -d "#0" -freq 433.0 -channels 25 -spacing 0.2

# Output:
# Opening YardStick One (Master)...
# Connected: Great Scott Gadgets YARD Stick One (Serial: xxxx)
#
# FHSS Master Configuration:
#   Base Frequency: 433.000 MHz
#   Channels:       25
#   Channel Spacing: 0.200 MHz
#   Frequency Range: 433.000 - 437.800 MHz
#   Dwell Time:     150 ms
#   Data Rate:      4800 baud
#
# FHSS Master active - transmitting beacons...
# Press Ctrl+C to stop
```

### Terminal 2: Start Client

```bash
# Start the client on device #1
./bin/fhss-client -d "#1" -freq 433.0 -channels 25 -spacing 0.2

# Output:
# Opening YardStick One (Client)...
# Connected: Great Scott Gadgets YARD Stick One (Serial: yyyy)
#
# FHSS Client Configuration:
#   Base Frequency:  433.000 MHz
#   Channels:        25
#   Channel Spacing: 0.200 MHz
#   Sync Timeout:    30s
#
# Searching for FHSS master...
# ..........
# Synchronized in 1.523s
#
# Receiving (Ctrl+C to stop)...
# ─────────────────────────────
# 12:34:56.789 [ch:5] RX 12 bytes: 48656c6c6f20464853530a
#          ASCII: Hello FHSS
```

### Terminal 3: Transmit Data (Optional)

```bash
# Use a third device, or repurpose the master to also transmit
./bin/fhss-tx -d "#0" -master -msg "Hello FHSS" -repeat 100 -interval 1s

# Or interactive mode:
./bin/fhss-tx -d "#0" -master
# > Hello World
# Sent: Hello World (11 bytes)
# > Test message
# Sent: Test message (12 bytes)
```

---

## Configuration Parameters

### Matching Parameters

**Critical:** These must be identical on master and client:
- Base frequency (`-freq`)
- Number of channels (`-channels`)
- Channel spacing (`-spacing`)
- Channel sequence (code must generate same sequence)

### Tunable Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `freq` | 433.0 MHz | Starting frequency |
| `channels` | 25 | Number of hop channels |
| `spacing` | 0.2 MHz | Gap between channels |
| `dwell` | 150 ms | Time on each channel |

### Frequency Calculations

```
Total Bandwidth = channels × spacing
               = 25 × 0.2 MHz = 5 MHz

Frequency Range = freq to (freq + (channels-1) × spacing)
               = 433.0 to 437.8 MHz

Hop Rate = 1000 / dwell
        = 1000 / 150 = 6.67 hops/second
```

---

## Pseudo-Random Channel Sequences

For better interference resistance, use a pseudo-random hop sequence instead of linear:

```go
// Generate pseudo-random hop sequence using LFSR
func generateHopSequence(numChannels int, seed uint32) []byte {
	channels := make([]byte, numChannels)
	used := make(map[byte]bool)

	lfsr := seed
	idx := 0

	for idx < numChannels {
		// 16-bit LFSR with taps at 16, 14, 13, 11
		bit := ((lfsr >> 0) ^ (lfsr >> 2) ^ (lfsr >> 3) ^ (lfsr >> 5)) & 1
		lfsr = (lfsr >> 1) | (bit << 15)

		ch := byte(lfsr % uint32(numChannels))
		if !used[ch] {
			channels[idx] = ch
			used[ch] = true
			idx++
		}
	}

	return channels
}

// Both master and client use the same seed
channels := generateHopSequence(25, 0x1234)
```

---

## Troubleshooting

### Client Won't Synchronize

1. **Check parameters match** - freq, channels, spacing must be identical
2. **Increase sync timeout** - `--sync-timeout 60s`
3. **Verify master is running** - Check for beacons with `rf-scanner`
4. **Distance** - Ensure devices are within range

### Data Not Received

1. **Verify sync state** - Client should be in `MAC_STATE_SYNCHED` (0x03)
2. **Check data rate** - Must match between TX and RX
3. **Timing** - Ensure both devices have stable USB connection

### Debugging

```bash
# Monitor RF activity
./bin/rf-scanner -center 435.0 -bw 10 -chans 200 -threshold -60

# Check MAC state
./bin/fhss-client -v  # Verbose mode shows state changes

# Verify channel setup
# Add debug output to show channel list after setup
```

---

## Next Steps

1. Implement the constants in `pkg/yardstick/constants.go` (Phase 1)
2. Add FHSS command support to device
3. Build and test with two devices
4. Refine timing based on actual hardware behavior
