# Implementation Plan: Firmware-Based Spectrum Analyzer

This document details the step-by-step plan to replace the broken software-based RF scanner with a firmware-based spectrum analyzer approach, as used by rfcat's `specan()` function.

## Background

The current `pkg/scanner` implementation uses software-controlled scanning:
- For each frequency: IDLE → set freq → CAL → RX → read RSSI → IDLE
- This requires 5+ USB round-trips per frequency
- The radio spends minimal time actually receiving
- RSSI readings are often invalid (0x80) due to AGC not settling
- Result: Scanner shows -90.5 dBm (noise floor) and misses all signals

The firmware-based approach:
- Firmware handles the entire sweep internally
- Radio stays in RX mode with minimal state transitions
- Fixed 2ms dwell time per channel (handled by firmware)
- Bulk RSSI data returned efficiently
- This is proven to work in rfcat's spectrum analyzer

## Critical Constraint: Don't Break Existing Tests

**ALL changes must preserve the `make tests` functionality.** The tests use:
- `pkg/yardstick/device.go` - USB communication
- `pkg/yardstick/radio.go` - `RFXmit`, `RFRecv`, `SetModeRX`, `SetModeIDLE`
- `pkg/config/` - Configuration loading
- `pkg/profiles/` - Profile definitions
- `pkg/registers/` - Register read/write

We will **NOT** modify any of these core files except to **ADD** new methods.

## Implementation Phases

---

## Phase 1: Add Firmware SPECAN Protocol Support to yardstick Package

### Step 1.1: Add SPECAN Constants

**File:** `pkg/yardstick/constants.go`

Add these constants after the existing NIC Commands section (~line 80):

```go
// Spectrum Analyzer Commands (sent via APP_NIC)
const (
    SPECANStart = 0x40  // RFCAT_START_SPECAN - start spectrum analysis
    SPECANStop  = 0x41  // RFCAT_STOP_SPECAN - stop spectrum analysis
)

// Spectrum Analyzer Queue
const (
    SPECANQueue = 0x01  // Queue ID for receiving spectrum data
)
```

**Verification:** `go build ./...` should succeed. Run `make tests` - should pass unchanged.

### Step 1.2: Add SetFrequency Method

**File:** `pkg/yardstick/radio.go`

Add method to set the radio frequency using FREQ registers. This wraps the poke operations.

```go
// SetFrequency sets the radio frequency in Hz
// Uses the CC1111's 24 MHz crystal reference
func (d *Device) SetFrequency(freqHz uint32) error {
    // Calculate FREQ registers for 24 MHz crystal
    // FREQ = (freq_hz * 65536) / 24000000
    freq := uint32((uint64(freqHz) * 65536) / 24000000)

    freq2 := uint8((freq >> 16) & 0xFF)
    freq1 := uint8((freq >> 8) & 0xFF)
    freq0 := uint8(freq & 0xFF)

    // Write FREQ2, FREQ1, FREQ0 registers
    if err := d.PokeByte(0xDF09, freq2); err != nil {  // RegFREQ2
        return fmt.Errorf("failed to set FREQ2: %w", err)
    }
    if err := d.PokeByte(0xDF0A, freq1); err != nil {  // RegFREQ1
        return fmt.Errorf("failed to set FREQ1: %w", err)
    }
    if err := d.PokeByte(0xDF0B, freq0); err != nil {  // RegFREQ0
        return fmt.Errorf("failed to set FREQ0: %w", err)
    }

    return nil
}

// GetFrequency returns the current radio frequency in Hz
func (d *Device) GetFrequency() (uint32, error) {
    freq2, err := d.PeekByte(0xDF09)
    if err != nil {
        return 0, err
    }
    freq1, err := d.PeekByte(0xDF0A)
    if err != nil {
        return 0, err
    }
    freq0, err := d.PeekByte(0xDF0B)
    if err != nil {
        return 0, err
    }

    freq := uint32(freq2)<<16 | uint32(freq1)<<8 | uint32(freq0)
    // Convert back to Hz: freq_hz = (FREQ * 24000000) / 65536
    freqHz := (uint64(freq) * 24000000) / 65536
    return uint32(freqHz), nil
}
```

**Verification:** `go build ./...` should succeed. Run `make tests` - should pass unchanged.

### Step 1.3: Add SetChannelSpacing Method

**File:** `pkg/yardstick/radio.go`

Add method to configure channel spacing for spectrum analysis.

```go
// SetChannelSpacing sets the channel spacing for spectrum analysis
// Uses MDMCFG0 and MDMCFG1 registers
// spacing = (Fxtal / 2^18) * (256 + CHANSPC_M) * 2^CHANSPC_E
// For 24 MHz crystal: spacing = 91.552734 * (256 + M) * 2^E
func (d *Device) SetChannelSpacing(spacingHz uint32) error {
    // Find E and M that give closest match
    fxtal := float64(24000000)
    target := float64(spacingHz)

    var bestE, bestM uint8
    var bestError float64 = 1e12

    for e := uint8(0); e < 4; e++ {
        // m = (spacing * 2^18) / (fxtal * 2^e) - 256
        divisor := fxtal * float64(uint32(1)<<e)
        m := (target * float64(uint32(1)<<18)) / divisor - 256

        if m >= 0 && m <= 255 {
            mRounded := uint8(m + 0.5) // Round to nearest
            actual := (fxtal / float64(uint32(1)<<18)) * (256 + float64(mRounded)) * float64(uint32(1)<<e)
            err := actual - target
            if err < 0 {
                err = -err
            }
            if err < bestError {
                bestError = err
                bestE = e
                bestM = mRounded
            }
        }
    }

    // Read current MDMCFG1 to preserve other bits
    mdmcfg1, err := d.PeekByte(0xDF10) // RegMDMCFG1
    if err != nil {
        return fmt.Errorf("failed to read MDMCFG1: %w", err)
    }

    // MDMCFG1[1:0] = CHANSPC_E, preserve bits 7:2
    mdmcfg1 = (mdmcfg1 & 0xFC) | (bestE & 0x03)

    if err := d.PokeByte(0xDF10, mdmcfg1); err != nil {
        return fmt.Errorf("failed to set MDMCFG1: %w", err)
    }

    // MDMCFG0 = CHANSPC_M
    if err := d.PokeByte(0xDF11, bestM); err != nil {
        return fmt.Errorf("failed to set MDMCFG0: %w", err)
    }

    return nil
}

// GetChannelSpacing returns the current channel spacing in Hz
func (d *Device) GetChannelSpacing() (uint32, error) {
    mdmcfg1, err := d.PeekByte(0xDF10)
    if err != nil {
        return 0, err
    }
    mdmcfg0, err := d.PeekByte(0xDF11)
    if err != nil {
        return 0, err
    }

    chanspcE := mdmcfg1 & 0x03
    chanspcM := mdmcfg0

    // spacing = (24e6 / 2^18) * (256 + M) * 2^E
    spacing := (24000000.0 / float64(uint32(1)<<18)) * (256 + float64(chanspcM)) * float64(uint32(1)<<chanspcE)
    return uint32(spacing), nil
}
```

**Verification:** `go build ./...` should succeed. Run `make tests` - should pass unchanged.

### Step 1.4: Add RecvFromApp Method

**File:** `pkg/yardstick/device.go`

Add a method to receive data from a specific application, needed for SPECAN data.

```go
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
        remaining_time := time.Until(deadline)
        if remaining_time <= 0 {
            return nil, fmt.Errorf("timeout waiting for app 0x%02X data", app)
        }

        readTimeout := 100 * time.Millisecond
        if remaining_time < readTimeout {
            readTimeout = remaining_time
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
```

**Verification:** `go build ./...` should succeed. Run `make tests` - should pass unchanged.

---

## Phase 2: Create New Spectrum Analyzer Package

### Step 2.1: Create pkg/specan/specan.go

Create a new, simpler package for firmware-based spectrum analysis.

**File:** `pkg/specan/specan.go`

```go
// Package specan provides firmware-based spectrum analysis for YardStick One
package specan

import (
    "fmt"
    "sync"
    "time"

    "github.com/herlein/gocat/pkg/yardstick"
)

// SpecAn represents a firmware-based spectrum analyzer
type SpecAn struct {
    device      *yardstick.Device
    baseFreq    uint32  // Base frequency in Hz
    chanSpacing uint32  // Channel spacing in Hz
    numChans    uint8   // Number of channels (max 255)

    mu          sync.Mutex
    running     bool
    stopChan    chan struct{}
    dataChan    chan *Frame
}

// Frame represents a single spectrum sweep result
type Frame struct {
    Timestamp   time.Time
    BaseFreq    uint32     // Hz
    ChanSpacing uint32     // Hz
    NumChans    int
    RSSI        []float32  // dBm values for each channel
}

// Config holds spectrum analyzer configuration
type Config struct {
    CenterFreq  uint32  // Hz - center frequency
    Bandwidth   uint32  // Hz - total bandwidth to scan
    NumChans    uint8   // Number of channels (1-255)
}

// New creates a new spectrum analyzer
func New(device *yardstick.Device) *SpecAn {
    return &SpecAn{
        device:   device,
        dataChan: make(chan *Frame, 10),
        stopChan: make(chan struct{}),
    }
}

// Configure sets up the spectrum analyzer parameters
func (s *SpecAn) Configure(cfg *Config) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.running {
        return fmt.Errorf("cannot configure while running")
    }

    if cfg.NumChans == 0 || cfg.NumChans > 255 {
        return fmt.Errorf("numChans must be 1-255, got %d", cfg.NumChans)
    }

    // Calculate base frequency and channel spacing
    halfBW := cfg.Bandwidth / 2
    s.baseFreq = cfg.CenterFreq - halfBW
    s.chanSpacing = cfg.Bandwidth / uint32(cfg.NumChans)
    s.numChans = cfg.NumChans

    // Set base frequency on device
    if err := s.device.SetFrequency(s.baseFreq); err != nil {
        return fmt.Errorf("failed to set frequency: %w", err)
    }

    // Set channel spacing
    if err := s.device.SetChannelSpacing(s.chanSpacing); err != nil {
        return fmt.Errorf("failed to set channel spacing: %w", err)
    }

    return nil
}

// Start begins the firmware spectrum analyzer
func (s *SpecAn) Start() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.running {
        return fmt.Errorf("already running")
    }

    // Send START_SPECAN command with channel count
    cmd := []byte{s.numChans}
    _, err := s.device.Send(yardstick.AppNIC, yardstick.SPECANStart, cmd, yardstick.USBDefaultTimeout)
    if err != nil {
        return fmt.Errorf("failed to start specan: %w", err)
    }

    s.running = true
    s.stopChan = make(chan struct{})
    s.dataChan = make(chan *Frame, 10)

    // Start receive goroutine
    go s.receiveLoop()

    return nil
}

// Stop halts the spectrum analyzer
func (s *SpecAn) Stop() error {
    s.mu.Lock()
    if !s.running {
        s.mu.Unlock()
        return nil
    }
    s.running = false
    close(s.stopChan)
    s.mu.Unlock()

    // Send STOP_SPECAN command
    _, err := s.device.Send(yardstick.AppNIC, yardstick.SPECANStop, nil, yardstick.USBDefaultTimeout)
    if err != nil {
        return fmt.Errorf("failed to stop specan: %w", err)
    }

    return nil
}

// IsRunning returns true if the analyzer is running
func (s *SpecAn) IsRunning() bool {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.running
}

// Frames returns a channel that receives spectrum frames
func (s *SpecAn) Frames() <-chan *Frame {
    return s.dataChan
}

// receiveLoop continuously receives RSSI data from firmware
func (s *SpecAn) receiveLoop() {
    defer close(s.dataChan)

    for {
        select {
        case <-s.stopChan:
            return
        default:
        }

        // Receive from APP_SPECAN, SPECAN_QUEUE
        data, err := s.device.RecvFromApp(yardstick.AppSPECAN, yardstick.SPECANQueue, 1*time.Second)
        if err != nil {
            // Timeout is normal, check if we should stop
            s.mu.Lock()
            running := s.running
            s.mu.Unlock()
            if !running {
                return
            }
            continue
        }

        if len(data) == 0 {
            continue
        }

        // Convert raw RSSI to dBm
        // rfcat formula: (raw ^ 0x80) / 2 - 88
        rssiDBm := make([]float32, len(data))
        for i, raw := range data {
            rssiDBm[i] = float32(int8(raw^0x80))/2.0 - 88.0
        }

        frame := &Frame{
            Timestamp:   time.Now(),
            BaseFreq:    s.baseFreq,
            ChanSpacing: s.chanSpacing,
            NumChans:    len(data),
            RSSI:        rssiDBm,
        }

        // Non-blocking send
        select {
        case s.dataChan <- frame:
        default:
            // Drop if channel full
        }
    }
}

// GetFrequencyForChannel returns the frequency for a given channel index
func (s *SpecAn) GetFrequencyForChannel(chanIdx int) uint32 {
    return s.baseFreq + uint32(chanIdx)*s.chanSpacing
}

// FrequencyForChannel is a helper to calculate frequency from frame parameters
func FrequencyForChannel(frame *Frame, chanIdx int) uint32 {
    return frame.BaseFreq + uint32(chanIdx)*frame.ChanSpacing
}
```

**Verification:** `go build ./...` should succeed. Run `make tests` - should pass unchanged.

### Step 2.2: Create pkg/specan/analysis.go

Add helper functions for analyzing spectrum data.

**File:** `pkg/specan/analysis.go`

```go
package specan

// FindPeaks finds channels with RSSI above threshold
func FindPeaks(frame *Frame, thresholdDBm float32) []Peak {
    var peaks []Peak
    for i, rssi := range frame.RSSI {
        if rssi >= thresholdDBm {
            peaks = append(peaks, Peak{
                ChannelIndex: i,
                FrequencyHz:  FrequencyForChannel(frame, i),
                RSSI:         rssi,
            })
        }
    }
    return peaks
}

// Peak represents a detected signal peak
type Peak struct {
    ChannelIndex int
    FrequencyHz  uint32
    RSSI         float32
}

// MaxRSSI returns the channel with maximum RSSI
func MaxRSSI(frame *Frame) (channelIndex int, frequencyHz uint32, rssi float32) {
    if len(frame.RSSI) == 0 {
        return -1, 0, -200.0
    }

    maxIdx := 0
    maxVal := frame.RSSI[0]

    for i, v := range frame.RSSI {
        if v > maxVal {
            maxVal = v
            maxIdx = i
        }
    }

    return maxIdx, FrequencyForChannel(frame, maxIdx), maxVal
}

// AverageRSSI calculates the average RSSI across all channels
func AverageRSSI(frame *Frame) float32 {
    if len(frame.RSSI) == 0 {
        return -200.0
    }

    var sum float32
    for _, v := range frame.RSSI {
        sum += v
    }
    return sum / float32(len(frame.RSSI))
}
```

**Verification:** `go build ./...` should succeed. Run `make tests` - should pass unchanged.

---

## Phase 3: Update rf-scanner Command

### Step 3.1: Rewrite cmd/rf-scanner/main.go

Replace the existing software-based scanner with the firmware-based one.

**File:** `cmd/rf-scanner/main.go` (complete rewrite)

```go
// rf-scanner is a firmware-based frequency scanner for the YardStick One
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
    "github.com/herlein/gocat/pkg/specan"
    "github.com/herlein/gocat/pkg/yardstick"
)

var (
    centerFreq  = flag.Float64("center", 433.92, "Center frequency in MHz")
    bandwidth   = flag.Float64("bw", 2.0, "Bandwidth in MHz")
    numChans    = flag.Int("chans", 100, "Number of channels (1-255)")
    threshold   = flag.Float64("threshold", -70.0, "RSSI threshold in dBm for peak detection")
    duration    = flag.Duration("duration", 0, "Scan duration (0 = indefinite)")
    deviceSel   = flag.String("d", "", yardstick.DeviceFlagUsage())
    listOnly    = flag.Bool("l", false, "List devices only")
    verbose     = flag.Bool("v", false, "Verbose output - show all frames")
    quiet       = flag.Bool("q", false, "Quiet mode - only show detected signals")
)

func main() {
    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
        fmt.Fprintf(os.Stderr, "Firmware-based RF Spectrum Analyzer for YardStick One\n\n")
        fmt.Fprintf(os.Stderr, "Options:\n")
        flag.PrintDefaults()
        fmt.Fprintf(os.Stderr, "\nExamples:\n")
        fmt.Fprintf(os.Stderr, "  %s -center 433.92 -bw 2           # Scan 432.92-434.92 MHz\n", os.Args[0])
        fmt.Fprintf(os.Stderr, "  %s -center 915 -bw 10 -chans 200  # Wide scan at 915 MHz\n", os.Args[0])
        fmt.Fprintf(os.Stderr, "  %s -threshold -80 -q              # Only show signals above -80 dBm\n", os.Args[0])
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

    if *listOnly {
        return listDevices(ctx)
    }

    // Validate parameters
    if *numChans < 1 || *numChans > 255 {
        return fmt.Errorf("chans must be 1-255")
    }

    // Open device
    fmt.Println("Opening YardStick One...")
    device, err := yardstick.SelectDevice(ctx, yardstick.DeviceSelector(*deviceSel))
    if err != nil {
        return fmt.Errorf("failed to open device: %w", err)
    }
    defer device.Close()

    fmt.Printf("Connected to: %s\n", device)

    // Create spectrum analyzer
    sa := specan.New(device)

    // Configure
    cfg := &specan.Config{
        CenterFreq: uint32(*centerFreq * 1e6),
        Bandwidth:  uint32(*bandwidth * 1e6),
        NumChans:   uint8(*numChans),
    }

    fmt.Printf("\nConfiguration:\n")
    fmt.Printf("  Center:     %.3f MHz\n", *centerFreq)
    fmt.Printf("  Bandwidth:  %.3f MHz\n", *bandwidth)
    fmt.Printf("  Channels:   %d\n", *numChans)
    fmt.Printf("  Range:      %.3f - %.3f MHz\n",
        *centerFreq - *bandwidth/2, *centerFreq + *bandwidth/2)
    fmt.Printf("  Resolution: %.3f kHz per channel\n", *bandwidth * 1000 / float64(*numChans))
    fmt.Printf("  Threshold:  %.1f dBm\n", *threshold)
    fmt.Println()

    if err := sa.Configure(cfg); err != nil {
        return fmt.Errorf("configure failed: %w", err)
    }

    // Set up signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Start analyzer
    if err := sa.Start(); err != nil {
        return fmt.Errorf("start failed: %w", err)
    }
    defer sa.Stop()

    // Set up timeout if specified
    var timeoutCtx context.Context
    var cancel context.CancelFunc
    if *duration > 0 {
        timeoutCtx, cancel = context.WithTimeout(context.Background(), *duration)
        fmt.Printf("Scanning for %v...\n", *duration)
    } else {
        timeoutCtx, cancel = context.WithCancel(context.Background())
        fmt.Println("Scanning... (Press Ctrl+C to stop)")
    }
    defer cancel()

    // Display header
    if !*quiet {
        fmt.Println("\n Frame | Max Freq (MHz) | Max RSSI | Avg RSSI | Peaks")
        fmt.Println("-------+----------------+----------+----------+-------")
    }

    frameCount := 0
    peakCount := 0

    for {
        select {
        case <-sigChan:
            fmt.Println("\n\nStopping...")
            goto done

        case <-timeoutCtx.Done():
            goto done

        case frame, ok := <-sa.Frames():
            if !ok {
                goto done
            }

            frameCount++
            maxIdx, maxFreq, maxRSSI := specan.MaxRSSI(frame)
            avgRSSI := specan.AverageRSSI(frame)
            peaks := specan.FindPeaks(frame, float32(*threshold))

            if len(peaks) > 0 {
                peakCount += len(peaks)
                if *quiet {
                    // Quiet mode: only show peaks
                    for _, p := range peaks {
                        fmt.Printf("SIGNAL: %.3f MHz @ %.1f dBm\n",
                            float64(p.FrequencyHz)/1e6, p.RSSI)
                    }
                }
            }

            if !*quiet {
                if *verbose || len(peaks) > 0 {
                    fmt.Printf(" %5d | %14.3f | %8.1f | %8.1f | %d\n",
                        frameCount, float64(maxFreq)/1e6, maxRSSI, avgRSSI, len(peaks))
                } else if frameCount%50 == 0 {
                    // Periodic status update
                    fmt.Printf(" %5d | %14.3f | %8.1f | %8.1f | scanning...\n",
                        frameCount, float64(maxFreq)/1e6, maxRSSI, avgRSSI)
                }
            }

            // Debug: print full spectrum on verbose with signal
            if *verbose && len(peaks) > 0 && maxIdx >= 0 {
                fmt.Printf("        Channel %d: raw index in spectrum\n", maxIdx)
            }
        }
    }

done:
    fmt.Printf("\n--- Summary ---\n")
    fmt.Printf("Frames:  %d\n", frameCount)
    fmt.Printf("Signals: %d (above %.1f dBm)\n", peakCount, *threshold)
    return nil
}

func listDevices(ctx *gousb.Context) error {
    devices, err := yardstick.FindAllDevices(ctx)
    if err != nil {
        return fmt.Errorf("failed to list devices: %w", err)
    }

    if len(devices) == 0 {
        fmt.Println("No YardStick One devices found")
        return nil
    }

    fmt.Printf("Found %d YardStick One device(s):\n\n", len(devices))
    for i, d := range devices {
        defer d.Close()
        fmt.Printf("  #%d  %s  %d:%d\n", i, d.Serial, d.Bus, d.Address)
    }
    return nil
}
```

**Verification:** `go build ./...` should succeed. Run `make tests` - should pass unchanged.

---

## Phase 4: Remove Old Scanner Package

### Step 4.1: Remove pkg/scanner/* Files

Once the new implementation is working, remove the old scanner package:

```bash
# First verify the build and tests still work
make build
make tests

# Then remove the old scanner package
rm -rf pkg/scanner/
```

**Note:** The old scanner package is NOT used by `make tests` - it's only used by `rf-scanner`. So removing it should not affect tests.

### Step 4.2: Update go.mod if needed

Run `go mod tidy` to clean up any unused dependencies.

---

## Phase 5: Testing and Verification

### Step 5.1: Run Make Tests After Each Step

After EVERY change:
```bash
go build ./...
make tests
```

The tests should pass unchanged throughout the entire implementation.

### Step 5.2: Manual rf-scanner Testing

Once Phase 3 is complete:

1. **Basic test:**
   ```bash
   ./bin/rf-scanner -center 433.92 -bw 1 -chans 50
   ```

2. **Test with transmission (requires 2 YS1 devices):**
   - Terminal 1: `./bin/rf-scanner -center 433.92 -bw 2 -threshold -80`
   - Terminal 2: `./bin/profile-test -profile 433-2fsk-std-4.8k -repeat 5 -tx '#0' -rx '#1'`
   - Scanner should detect transmissions

3. **Compare with rfcat:**
   ```python
   from rflib import RfCat
   d = RfCat()
   d.specan(433920000, 25000, 100)  # Should show similar results
   ```

---

## Summary: Files to Create/Modify

### New Files:
- `pkg/specan/specan.go` - New firmware-based spectrum analyzer
- `pkg/specan/analysis.go` - Helper functions for analyzing spectrum data

### Modified Files:
- `pkg/yardstick/constants.go` - Add SPECAN constants
- `pkg/yardstick/radio.go` - Add SetFrequency, GetFrequency, SetChannelSpacing, GetChannelSpacing
- `pkg/yardstick/device.go` - Add RecvFromApp, parseResponseFromApp
- `cmd/rf-scanner/main.go` - Complete rewrite to use firmware-based scanner

### Files to Remove (after verification):
- `pkg/scanner/scanner.go`
- `pkg/scanner/config.go`
- `pkg/scanner/constants.go`
- `pkg/scanner/errors.go`
- `pkg/scanner/result.go`
- `pkg/scanner/smoother.go`
- `pkg/scanner/tracker.go`

### Files NOT Modified (critical for test preservation):
- `pkg/yardstick/device.go` - Only ADD methods, don't modify existing
- `pkg/yardstick/radio.go` - Only ADD methods, don't modify existing
- `pkg/config/*` - Not touched
- `pkg/profiles/*` - Not touched
- `pkg/registers/*` - Not touched
- `cmd/profile-test/*` - Not touched
- `Makefile` - Not touched

---

## Risk Mitigation

1. **Don't modify existing working code** - Only add new methods
2. **Run `make tests` after every change** - Catch regressions immediately
3. **Keep the old scanner until new one is verified working**
4. **Test with actual RF transmissions** before declaring success

---

## Expected Outcome

After implementation:
- `make tests` passes (unchanged)
- `./bin/rf-scanner` detects signals that were previously invisible
- RSSI values vary and respond to actual RF energy
- No more 0x80 invalid values
- Spectrum display matches rfcat's `specan()` output
