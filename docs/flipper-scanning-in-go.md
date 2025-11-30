# Frequency Scanner Design Document

This document specifies the design for implementing Flipper Zero-style frequency scanning functionality in Go for the YardStick One.

## Executive Summary

The frequency scanner enables detection and identification of RF signals across the sub-GHz spectrum. It employs a two-phase scanning approach: a fast coarse scan across predefined frequencies followed by a precision fine scan around detected signals. This design adapts the Flipper Zero's proven scanning algorithm for the YardStick One's CC1111 hardware.

## Hardware Differences

### Crystal Frequency
| Device | Crystal | Impact |
|--------|---------|--------|
| Flipper Zero (CC1101) | 26 MHz | All frequency calculations use 26 MHz base |
| YardStick One (CC1111) | 24 MHz | Must adjust all calculations accordingly |

### Frequency Calculation Formulas

**CC1111 (YS1) - 24 MHz Crystal:**
```
FREQ_REG = (freq_hz * 65536) / 24000000
Actual_Freq = (FREQ_REG * 24000000) / 65536
```

**Channel Bandwidth (CC1111):**
```
BW_Hz = 24000000 / (8 * (4 + CHANBW_M) * 2^CHANBW_E)
```

### RSSI Conversion

The CC1111 RSSI register uses a different offset than CC1101:

```go
// CC1111 RSSI to dBm conversion
func RSSIToDBm(rssi uint8) float32 {
    if rssi >= 128 {
        return float32(rssi-256)/2.0 - 74.0
    }
    return float32(rssi)/2.0 - 74.0
}
```

## Package Structure

```
pkg/
  scanner/
    scanner.go       // Main Scanner type and interface
    config.go        // ScanConfig, presets, frequency lists
    result.go        // ScanResult, SignalInfo types
    smoother.go      // Frequency smoothing algorithm
    constants.go     // Constants and defaults
```

## Core Types

### ScanResult

```go
// ScanResult holds the result of a single scan cycle
type ScanResult struct {
    // Coarse scan results
    CoarseFrequency uint32   // Hz - frequency with strongest signal in coarse scan
    CoarseRSSI      float32  // dBm - signal strength at coarse frequency

    // Fine scan results (only populated if signal detected)
    FineFrequency   uint32   // Hz - refined frequency from fine scan
    FineRSSI        float32  // dBm - signal strength at fine frequency

    // Metadata
    Timestamp       time.Time
    SignalDetected  bool     // True if RSSI exceeded threshold
}
```

### SignalInfo

```go
// SignalInfo represents a detected signal with history
type SignalInfo struct {
    Frequency       uint32    // Hz - smoothed frequency
    RawFrequency    uint32    // Hz - last measured frequency
    RSSI            float32   // dBm - current signal strength
    MaxRSSI         float32   // dBm - maximum observed RSSI
    FirstSeen       time.Time
    LastSeen        time.Time
    DetectionCount  uint32    // Number of times detected
}
```

### ScanConfig

```go
// ScanConfig defines scanning parameters
type ScanConfig struct {
    // Frequency lists
    CoarseFrequencies []uint32 // Hz - frequencies for coarse scan

    // Scan parameters
    RSSIThreshold     float32       // dBm - minimum signal detection threshold
    FineScanRange     uint32        // Hz - range around detected signal (± this value)
    FineScanStep      uint32        // Hz - step size for fine scan
    DwellTime         time.Duration // Time to wait for RSSI measurement

    // Callback (optional)
    OnSignalDetected  func(info *SignalInfo)
    OnSignalLost      func(info *SignalInfo)
}
```

### Scanner Interface

```go
// Scanner provides frequency scanning capabilities
type Scanner interface {
    // Lifecycle
    Start() error
    Stop() error
    IsRunning() bool

    // Configuration
    SetConfig(config *ScanConfig) error
    GetConfig() *ScanConfig

    // Scanning
    ScanOnce() (*ScanResult, error)
    ScanContinuous(ctx context.Context, results chan<- *ScanResult) error

    // Signal tracking
    GetActiveSignals() []*SignalInfo
    ClearSignalHistory()
}
```

## Implementation Details

### Scanner Structure

```go
type scanner struct {
    device    *yardstick.Device
    config    *ScanConfig

    // State
    mu        sync.RWMutex
    running   bool
    stopChan  chan struct{}

    // Signal tracking
    signals       map[uint32]*SignalInfo  // Key: rounded frequency
    signalsMu     sync.RWMutex

    // Smoothing
    smoother      *FrequencySmoother

    // Saved radio config (to restore after scanning)
    savedConfig   *registers.RegisterMap
}
```

### Scanning Algorithm

#### Phase 1: Coarse Scan

```go
func (s *scanner) coarseScan() (*ScanResult, error) {
    result := &ScanResult{Timestamp: time.Now()}

    // Load wide bandwidth preset
    if err := s.loadCoarsePreset(); err != nil {
        return nil, fmt.Errorf("failed to load coarse preset: %w", err)
    }

    var maxRSSI float32 = -200.0  // Very low initial value
    var maxFreq uint32 = 0

    for _, freq := range s.config.CoarseFrequencies {
        rssi, err := s.measureRSSI(freq)
        if err != nil {
            continue  // Skip failed measurements
        }

        if rssi > maxRSSI {
            maxRSSI = rssi
            maxFreq = freq
        }
    }

    result.CoarseFrequency = maxFreq
    result.CoarseRSSI = maxRSSI
    result.SignalDetected = maxRSSI >= s.config.RSSIThreshold

    return result, nil
}
```

#### Phase 2: Fine Scan

```go
func (s *scanner) fineScan(coarseResult *ScanResult) (*ScanResult, error) {
    if !coarseResult.SignalDetected {
        return coarseResult, nil
    }

    // Load narrow bandwidth preset
    if err := s.loadFinePreset(); err != nil {
        return nil, fmt.Errorf("failed to load fine preset: %w", err)
    }

    center := coarseResult.CoarseFrequency
    startFreq := center - s.config.FineScanRange
    endFreq := center + s.config.FineScanRange

    var maxRSSI float32 = -200.0
    var maxFreq uint32 = 0

    for freq := startFreq; freq <= endFreq; freq += s.config.FineScanStep {
        rssi, err := s.measureRSSI(freq)
        if err != nil {
            continue
        }

        if rssi > maxRSSI {
            maxRSSI = rssi
            maxFreq = freq
        }
    }

    coarseResult.FineFrequency = maxFreq
    coarseResult.FineRSSI = maxRSSI

    return coarseResult, nil
}
```

#### RSSI Measurement

```go
func (s *scanner) measureRSSI(freqHz uint32) (float32, error) {
    // 1. Go to IDLE
    if err := registers.SetIDLE(s.device); err != nil {
        return 0, err
    }

    // 2. Set frequency
    if err := s.setFrequency(freqHz); err != nil {
        return 0, err
    }

    // 3. Calibrate (strobe SCAL)
    if err := registers.Strobe(s.device, registers.StrobeSCAL); err != nil {
        return 0, err
    }

    // 4. Wait for IDLE state (calibration complete)
    if err := s.device.WaitForState(yardstick.MarcStateIdle, 10*time.Millisecond); err != nil {
        return 0, err
    }

    // 5. Enter RX mode
    if err := registers.SetRX(s.device); err != nil {
        return 0, err
    }

    // 6. Wait for AGC to settle
    time.Sleep(s.config.DwellTime)

    // 7. Read RSSI
    rssiRaw, err := s.device.GetRSSI()
    if err != nil {
        return 0, err
    }

    // 8. Return to IDLE
    _ = registers.SetIDLE(s.device)

    return RSSIToDBm(rssiRaw), nil
}
```

### Radio Configuration Presets

#### Wide Bandwidth Preset (Coarse Scan)

Optimized for fast signal detection across wide bandwidth:

```go
var CoarseScanPreset = &registers.RegisterMap{
    // 650 kHz bandwidth for CC1111 (24 MHz crystal)
    // BW = 24000000 / (8 * (4 + 0) * 2^0) = 750 kHz (closest achievable)
    // Use CHANBW_E=0, CHANBW_M=1 for ~600 kHz
    MDMCFG4: 0x1F,  // CHANBW_E=0, CHANBW_M=1, DRATE_E=15
    MDMCFG3: 0x7F,  // DRATE_M (high value for fast response)

    // ASK/OOK, no sync (detect any energy)
    MDMCFG2: 0x30,  // MOD_FORMAT=ASK/OOK, SYNC_MODE=none

    // AGC settings optimized for signal detection
    AGCCTRL2: 0x07, // MAX_DVGA_GAIN=all, MAX_LNA_GAIN=max, MAGN_TARGET=42dB
    AGCCTRL1: 0x00, // AGC_LNA_PRIORITY=0, CS_REL_THR=disabled
    AGCCTRL0: 0x91, // HYST_LEVEL=medium, WAIT_TIME=32, AGC_FREEZE=normal

    // Frontend optimized for wideband
    FREND1: 0xB6,
    FREND0: 0x10,
}
```

#### Narrow Bandwidth Preset (Fine Scan)

Optimized for precise frequency measurement:

```go
var FineScanPreset = &registers.RegisterMap{
    // ~58 kHz bandwidth for CC1111 (24 MHz crystal)
    // BW = 24000000 / (8 * (4 + 3) * 2^3) = 53.6 kHz
    MDMCFG4: 0xF7,  // CHANBW_E=3, CHANBW_M=3, DRATE_E=7
    MDMCFG3: 0x7F,  // DRATE_M

    // Same modulation
    MDMCFG2: 0x30,  // MOD_FORMAT=ASK/OOK, SYNC_MODE=none

    // AGC settings
    AGCCTRL2: 0x07,
    AGCCTRL1: 0x00,
    AGCCTRL0: 0x91,

    // Frontend for narrowband
    FREND1: 0x56,
    FREND0: 0x10,
}
```

### Frequency Lists

#### Default Frequency List (17 frequencies)

Covers common sub-GHz ISM and license-free bands:

```go
var DefaultFrequencies = []uint32{
    // 300-348 MHz band
    300000000,
    303875000,  // Garage doors
    304250000,
    310000000,  // US keyless entry
    315000000,  // US keyless entry
    318000000,

    // 387-464 MHz band
    390000000,
    418000000,
    433075000,  // LPD433 first channel
    433420000,
    433920000,  // LPD433 center (most common)
    434420000,
    434775000,  // LPD433 last channel
    438900000,

    // 779-928 MHz band
    868350000,  // EU SRD
    915000000,  // US ISM
    925000000,
}
```

#### Hopper Frequency List (6 frequencies)

Subset for rapid scanning:

```go
var HopperFrequencies = []uint32{
    310000000,  // 300 MHz band
    315000000,
    390000000,  // 400 MHz band
    433920000,
    868350000,  // 800 MHz band
    915000000,
}
```

### Frequency Smoothing

Prevents display jitter while maintaining responsiveness:

```go
type FrequencySmoother struct {
    value     float64
    threshold float64  // Hz - above this, use fast adaptation
    kFast     float64  // Adaptation coefficient for large changes
    kSlow     float64  // Adaptation coefficient for small changes
}

func NewFrequencySmoother() *FrequencySmoother {
    return &FrequencySmoother{
        value:     0,
        threshold: 500000,  // 500 kHz
        kFast:     0.9,
        kSlow:     0.03,
    }
}

func (s *FrequencySmoother) Update(newValue float64) float64 {
    if s.value == 0 {
        s.value = newValue
        return newValue
    }

    var k float64
    if math.Abs(newValue - s.value) > s.threshold {
        k = s.kFast  // Fast adaptation for large changes
    } else {
        k = s.kSlow  // Slow adaptation for stability
    }

    s.value += (newValue - s.value) * k
    return s.value
}

func (s *FrequencySmoother) Reset() {
    s.value = 0
}
```

### Signal Tracking

Manages detection state with hysteresis:

```go
type signalTracker struct {
    signals     map[uint32]*SignalInfo
    mu          sync.RWMutex
    holdCounter int           // Counts down when signal lost
    holdMax     int           // Maximum hold count (e.g., 20)
    lostAt      int           // Counter value when "lost" callback fires (e.g., 15)
}

func (t *signalTracker) update(result *ScanResult) {
    t.mu.Lock()
    defer t.mu.Unlock()

    if result.SignalDetected {
        // Reset hold counter
        t.holdCounter = t.holdMax

        // Round frequency for lookup (10 kHz resolution)
        key := (result.FineFrequency / 10000) * 10000

        info, exists := t.signals[key]
        if !exists {
            info = &SignalInfo{
                Frequency:  result.FineFrequency,
                FirstSeen:  result.Timestamp,
            }
            t.signals[key] = info
        }

        info.RawFrequency = result.FineFrequency
        info.RSSI = result.FineRSSI
        info.LastSeen = result.Timestamp
        info.DetectionCount++
        if result.FineRSSI > info.MaxRSSI {
            info.MaxRSSI = result.FineRSSI
        }
    } else {
        // Decrement hold counter
        if t.holdCounter > 0 {
            t.holdCounter--

            if t.holdCounter == t.lostAt {
                // Signal considered lost - trigger callback
            }
        }
    }
}
```

## API Usage Examples

### Basic Single Scan

```go
func ExampleSingleScan() {
    ctx := gousb.NewContext()
    defer ctx.Close()

    device, err := yardstick.OpenDevice(ctx, "")
    if err != nil {
        log.Fatal(err)
    }
    defer device.Close()

    // Create scanner with default config
    s := scanner.New(device, scanner.DefaultConfig())

    // Perform single scan
    result, err := s.ScanOnce()
    if err != nil {
        log.Fatal(err)
    }

    if result.SignalDetected {
        fmt.Printf("Signal detected at %.3f MHz (%.1f dBm)\n",
            float64(result.FineFrequency)/1e6,
            result.FineRSSI)
    }
}
```

### Continuous Scanning with Callbacks

```go
func ExampleContinuousScan() {
    ctx := gousb.NewContext()
    defer ctx.Close()

    device, err := yardstick.OpenDevice(ctx, "")
    if err != nil {
        log.Fatal(err)
    }
    defer device.Close()

    config := &scanner.ScanConfig{
        CoarseFrequencies: scanner.DefaultFrequencies,
        RSSIThreshold:     -93.0,
        FineScanRange:     300000,
        FineScanStep:      20000,
        DwellTime:         2 * time.Millisecond,
        OnSignalDetected: func(info *scanner.SignalInfo) {
            fmt.Printf("DETECTED: %.3f MHz @ %.1f dBm\n",
                float64(info.Frequency)/1e6, info.RSSI)
        },
        OnSignalLost: func(info *scanner.SignalInfo) {
            fmt.Printf("LOST: %.3f MHz (seen %d times)\n",
                float64(info.Frequency)/1e6, info.DetectionCount)
        },
    }

    s := scanner.New(device, config)

    // Start continuous scanning
    scanCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    results := make(chan *scanner.ScanResult, 10)

    go func() {
        if err := s.ScanContinuous(scanCtx, results); err != nil {
            log.Printf("Scan error: %v", err)
        }
    }()

    // Process results
    for result := range results {
        if result.SignalDetected {
            fmt.Printf("%.3f MHz: %.1f dBm\n",
                float64(result.FineFrequency)/1e6, result.FineRSSI)
        }
    }
}
```

### Custom Frequency Range

```go
func ExampleCustomRange() {
    // Scan only 433 MHz band
    config := &scanner.ScanConfig{
        CoarseFrequencies: []uint32{
            433000000, 433250000, 433500000, 433750000,
            434000000, 434250000, 434500000, 434750000,
        },
        RSSIThreshold: -90.0,
        FineScanRange: 100000,  // ±100 kHz
        FineScanStep:  10000,   // 10 kHz steps
        DwellTime:     3 * time.Millisecond,
    }

    s := scanner.New(device, config)
    // ...
}
```

## Constants

```go
package scanner

import "time"

const (
    // RSSI threshold for signal detection
    DefaultRSSIThreshold float32 = -93.0 // dBm

    // Fine scan parameters
    DefaultFineScanRange uint32 = 300000  // Hz (±300 kHz)
    DefaultFineScanStep  uint32 = 20000   // Hz (20 kHz steps)

    // Timing
    DefaultDwellTime     time.Duration = 2 * time.Millisecond
    DefaultScanInterval  time.Duration = 10 * time.Millisecond

    // Signal tracking
    DefaultHoldMax       int = 20  // Hold counter maximum
    DefaultLostThreshold int = 15  // Counter when "lost" fires

    // Frequency smoothing
    DefaultSmoothThreshold float64 = 500000  // Hz
    DefaultKFast           float64 = 0.9
    DefaultKSlow           float64 = 0.03

    // CC1111 crystal frequency
    CrystalHz uint32 = 24000000
)
```

## Error Handling

```go
var (
    ErrScannerRunning    = errors.New("scanner is already running")
    ErrScannerNotRunning = errors.New("scanner is not running")
    ErrDeviceNotReady    = errors.New("device is not ready")
    ErrInvalidConfig     = errors.New("invalid scanner configuration")
    ErrFrequencyOutOfRange = errors.New("frequency out of valid range")
)
```

## Thread Safety

The scanner implementation is fully thread-safe:

1. **Configuration changes** - Protected by `sync.RWMutex`
2. **Signal tracking** - Protected by separate `sync.RWMutex`
3. **Device access** - Serialized through scanner (no concurrent register access)
4. **Callbacks** - Called synchronously from scan goroutine

## Performance Considerations

### Timing Analysis

| Operation | Duration | Notes |
|-----------|----------|-------|
| Frequency change | ~200 µs | Register write via USB |
| Calibration | ~500 µs | SCAL strobe + wait |
| RX settle | 2 ms | AGC stabilization |
| RSSI read | ~100 µs | Single register read |
| **Per-frequency total** | **~3 ms** | |

### Scan Cycle Timing

| Phase | Frequencies | Time |
|-------|-------------|------|
| Coarse scan (default) | 17 | ~51 ms |
| Fine scan | 31 (±300kHz/20kHz) | ~93 ms |
| Inter-cycle delay | - | 10 ms |
| **Full cycle (signal present)** | | **~154 ms** |
| **Full cycle (no signal)** | | **~61 ms** |

### Optimization Strategies

1. **Parallel preset loading** - Pre-compute register values
2. **Batch register writes** - Combine adjacent registers
3. **Adaptive dwell time** - Reduce when no signals expected
4. **Frequency list caching** - Avoid allocation in hot path

## Testing Strategy

### Unit Tests

```go
func TestRSSIConversion(t *testing.T) {
    tests := []struct {
        raw      uint8
        expected float32
    }{
        {0, -74.0},
        {128, -74.0},
        {200, -102.0},  // (200-256)/2 - 74 = -102
        {100, -24.0},   // 100/2 - 74 = -24
    }

    for _, tt := range tests {
        result := RSSIToDBm(tt.raw)
        if math.Abs(float64(result-tt.expected)) > 0.1 {
            t.Errorf("RSSIToDBm(%d) = %f, want %f", tt.raw, result, tt.expected)
        }
    }
}

func TestFrequencySmoother(t *testing.T) {
    s := NewFrequencySmoother()

    // First value should be returned as-is
    v := s.Update(433920000)
    assert.Equal(t, 433920000.0, v)

    // Small change - slow adaptation
    v = s.Update(433925000)
    assert.InDelta(t, 433920150.0, v, 100)  // Only ~0.03 of the way

    // Large change - fast adaptation
    v = s.Update(868350000)
    assert.InDelta(t, 829307850.0, v, 1000)  // ~0.9 of the way
}
```

### Integration Tests

```go
func TestScannerIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    ctx := gousb.NewContext()
    defer ctx.Close()

    device, err := yardstick.OpenDevice(ctx, "")
    if err != nil {
        t.Skipf("No device available: %v", err)
    }
    defer device.Close()

    s := New(device, DefaultConfig())

    // Test single scan completes without error
    result, err := s.ScanOnce()
    require.NoError(t, err)
    require.NotNil(t, result)
    require.False(t, result.Timestamp.IsZero())
}
```

## File Layout

```
pkg/scanner/
  scanner.go         // Scanner interface and implementation
  scanner_test.go    // Unit tests for scanner
  config.go          // ScanConfig, presets, validation
  config_test.go     // Config validation tests
  result.go          // ScanResult, SignalInfo types
  smoother.go        // FrequencySmoother implementation
  smoother_test.go   // Smoother algorithm tests
  tracker.go         // Signal tracking with hysteresis
  tracker_test.go    // Tracker tests
  frequencies.go     // Frequency lists and validation
  constants.go       // Package constants
  errors.go          // Error definitions
  doc.go             // Package documentation
```

## JSON Configuration Format

Scanner configuration can be loaded from JSON files, enabling portable and shareable scan profiles.

### Schema Overview

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "YardStick One Scanner Configuration",
  "type": "object",
  "properties": {
    "name": "string - configuration name",
    "description": "string - human-readable description",
    "version": "string - schema version (1.0)",
    "created": "string - ISO 8601 timestamp",
    "frequencies": {
      "coarse": "array of uint32 - Hz, frequencies for coarse scan",
      "hopper": "array of uint32 - Hz, subset for rapid hopping (optional)",
      "bands": "array of band objects - define custom frequency ranges"
    },
    "scan_parameters": {
      "rssi_threshold_dbm": "float - minimum signal detection level",
      "fine_scan_range_hz": "uint32 - range around detected signal (±)",
      "fine_scan_step_hz": "uint32 - step size for fine scan",
      "dwell_time_ms": "uint32 - time to wait for RSSI measurement",
      "scan_interval_ms": "uint32 - delay between scan cycles"
    },
    "signal_tracking": {
      "hold_max": "int - max hold counter value",
      "lost_threshold": "int - counter value when signal considered lost",
      "frequency_resolution_hz": "uint32 - grouping resolution for signals"
    },
    "smoothing": {
      "enabled": "bool - enable frequency smoothing",
      "threshold_hz": "float - change threshold for fast/slow adaptation",
      "k_fast": "float - adaptation coefficient for large changes",
      "k_slow": "float - adaptation coefficient for small changes"
    },
    "radio_presets": {
      "coarse": "object - register overrides for coarse scan",
      "fine": "object - register overrides for fine scan"
    },
    "output": {
      "log_signals": "bool - log detected signals to file",
      "log_path": "string - path for signal log file",
      "log_format": "string - csv|json|text"
    }
  }
}
```

### Complete JSON Structure

```json
{
  "name": "default-scanner",
  "description": "Default frequency scanner configuration",
  "version": "1.0",
  "created": "2025-01-15T00:00:00Z",

  "frequencies": {
    "coarse": [
      300000000, 303875000, 304250000, 310000000, 315000000, 318000000,
      390000000, 418000000, 433075000, 433420000, 433920000,
      434420000, 434775000, 438900000,
      868350000, 915000000, 925000000
    ],
    "hopper": [
      310000000, 315000000, 390000000, 433920000, 868350000, 915000000
    ],
    "bands": [
      {
        "name": "300MHz",
        "start_hz": 300000000,
        "end_hz": 348000000,
        "step_hz": 500000,
        "enabled": true
      },
      {
        "name": "400MHz",
        "start_hz": 387000000,
        "end_hz": 464000000,
        "step_hz": 500000,
        "enabled": true
      },
      {
        "name": "800MHz",
        "start_hz": 779000000,
        "end_hz": 928000000,
        "step_hz": 1000000,
        "enabled": true
      }
    ]
  },

  "scan_parameters": {
    "rssi_threshold_dbm": -93.0,
    "fine_scan_range_hz": 300000,
    "fine_scan_step_hz": 20000,
    "dwell_time_ms": 2,
    "scan_interval_ms": 10
  },

  "signal_tracking": {
    "hold_max": 20,
    "lost_threshold": 15,
    "frequency_resolution_hz": 10000
  },

  "smoothing": {
    "enabled": true,
    "threshold_hz": 500000,
    "k_fast": 0.9,
    "k_slow": 0.03
  },

  "radio_presets": {
    "coarse": {
      "mdmcfg4": 31,
      "mdmcfg3": 127,
      "mdmcfg2": 48,
      "agcctrl2": 7,
      "agcctrl1": 0,
      "agcctrl0": 145,
      "frend1": 182,
      "frend0": 16
    },
    "fine": {
      "mdmcfg4": 247,
      "mdmcfg3": 127,
      "mdmcfg2": 48,
      "agcctrl2": 7,
      "agcctrl1": 0,
      "agcctrl0": 145,
      "frend1": 86,
      "frend0": 16
    }
  },

  "output": {
    "log_signals": false,
    "log_path": "",
    "log_format": "json"
  }
}
```

### Go Type Definitions for JSON

```go
// ScannerConfigFile represents the JSON configuration file structure
type ScannerConfigFile struct {
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Version     string    `json:"version"`
    Created     time.Time `json:"created"`

    Frequencies     FrequencyConfig     `json:"frequencies"`
    ScanParameters  ScanParametersJSON  `json:"scan_parameters"`
    SignalTracking  SignalTrackingJSON  `json:"signal_tracking"`
    Smoothing       SmoothingJSON       `json:"smoothing"`
    RadioPresets    RadioPresetsJSON    `json:"radio_presets"`
    Output          OutputConfig        `json:"output"`
}

// FrequencyConfig defines frequency lists and bands
type FrequencyConfig struct {
    Coarse []uint32    `json:"coarse"`
    Hopper []uint32    `json:"hopper,omitempty"`
    Bands  []BandConfig `json:"bands,omitempty"`
}

// BandConfig defines a frequency band for scanning
type BandConfig struct {
    Name    string `json:"name"`
    StartHz uint32 `json:"start_hz"`
    EndHz   uint32 `json:"end_hz"`
    StepHz  uint32 `json:"step_hz"`
    Enabled bool   `json:"enabled"`
}

// ScanParametersJSON holds scan timing and threshold settings
type ScanParametersJSON struct {
    RSSIThresholdDBm float32 `json:"rssi_threshold_dbm"`
    FineScanRangeHz  uint32  `json:"fine_scan_range_hz"`
    FineScanStepHz   uint32  `json:"fine_scan_step_hz"`
    DwellTimeMs      uint32  `json:"dwell_time_ms"`
    ScanIntervalMs   uint32  `json:"scan_interval_ms"`
}

// SignalTrackingJSON holds signal detection hysteresis settings
type SignalTrackingJSON struct {
    HoldMax             int    `json:"hold_max"`
    LostThreshold       int    `json:"lost_threshold"`
    FrequencyResolutionHz uint32 `json:"frequency_resolution_hz"`
}

// SmoothingJSON holds frequency smoothing algorithm settings
type SmoothingJSON struct {
    Enabled     bool    `json:"enabled"`
    ThresholdHz float64 `json:"threshold_hz"`
    KFast       float64 `json:"k_fast"`
    KSlow       float64 `json:"k_slow"`
}

// RadioPresetsJSON holds register values for scan presets
type RadioPresetsJSON struct {
    Coarse RegisterOverrides `json:"coarse"`
    Fine   RegisterOverrides `json:"fine"`
}

// RegisterOverrides allows partial register configuration
type RegisterOverrides struct {
    MDMCFG4  *uint8 `json:"mdmcfg4,omitempty"`
    MDMCFG3  *uint8 `json:"mdmcfg3,omitempty"`
    MDMCFG2  *uint8 `json:"mdmcfg2,omitempty"`
    MDMCFG1  *uint8 `json:"mdmcfg1,omitempty"`
    MDMCFG0  *uint8 `json:"mdmcfg0,omitempty"`
    AGCCTRL2 *uint8 `json:"agcctrl2,omitempty"`
    AGCCTRL1 *uint8 `json:"agcctrl1,omitempty"`
    AGCCTRL0 *uint8 `json:"agcctrl0,omitempty"`
    FREND1   *uint8 `json:"frend1,omitempty"`
    FREND0   *uint8 `json:"frend0,omitempty"`
    FOCCFG   *uint8 `json:"foccfg,omitempty"`
    BSCFG    *uint8 `json:"bscfg,omitempty"`
}

// OutputConfig defines signal logging options
type OutputConfig struct {
    LogSignals bool   `json:"log_signals"`
    LogPath    string `json:"log_path,omitempty"`
    LogFormat  string `json:"log_format,omitempty"` // csv, json, text
}
```

### Loading Configuration

```go
// LoadConfigFile loads scanner configuration from a JSON file
func LoadConfigFile(path string) (*ScannerConfigFile, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }

    var config ScannerConfigFile
    if err := json.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    if err := config.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    return &config, nil
}

// ToScanConfig converts JSON config to runtime ScanConfig
func (c *ScannerConfigFile) ToScanConfig() *ScanConfig {
    config := &ScanConfig{
        CoarseFrequencies: c.Frequencies.Coarse,
        RSSIThreshold:     c.ScanParameters.RSSIThresholdDBm,
        FineScanRange:     c.ScanParameters.FineScanRangeHz,
        FineScanStep:      c.ScanParameters.FineScanStepHz,
        DwellTime:         time.Duration(c.ScanParameters.DwellTimeMs) * time.Millisecond,
    }

    // Expand bands into frequency list if no coarse frequencies specified
    if len(config.CoarseFrequencies) == 0 {
        config.CoarseFrequencies = c.expandBands()
    }

    return config
}

// expandBands generates frequency list from band definitions
func (c *ScannerConfigFile) expandBands() []uint32 {
    var freqs []uint32
    for _, band := range c.Frequencies.Bands {
        if !band.Enabled {
            continue
        }
        for freq := band.StartHz; freq <= band.EndHz; freq += band.StepHz {
            freqs = append(freqs, freq)
        }
    }
    return freqs
}

// Validate checks configuration for errors
func (c *ScannerConfigFile) Validate() error {
    if c.Version != "1.0" {
        return fmt.Errorf("unsupported config version: %s", c.Version)
    }

    if len(c.Frequencies.Coarse) == 0 && len(c.Frequencies.Bands) == 0 {
        return errors.New("no frequencies or bands specified")
    }

    for _, freq := range c.Frequencies.Coarse {
        if !isValidFrequency(freq) {
            return fmt.Errorf("frequency %d Hz out of valid range", freq)
        }
    }

    if c.ScanParameters.RSSIThresholdDBm > 0 {
        return errors.New("RSSI threshold must be negative (dBm)")
    }

    if c.ScanParameters.DwellTimeMs < 1 || c.ScanParameters.DwellTimeMs > 100 {
        return errors.New("dwell time must be 1-100 ms")
    }

    return nil
}

// isValidFrequency checks if frequency is within CC1111 supported bands
func isValidFrequency(freq uint32) bool {
    // 300-348 MHz band
    if freq >= 300000000 && freq <= 348000000 {
        return true
    }
    // 387-464 MHz band
    if freq >= 387000000 && freq <= 464000000 {
        return true
    }
    // 779-928 MHz band
    if freq >= 779000000 && freq <= 928000000 {
        return true
    }
    return false
}
```

### Saving Configuration

```go
// SaveConfigFile saves scanner configuration to a JSON file
func SaveConfigFile(config *ScannerConfigFile, path string) error {
    config.Created = time.Now()

    data, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }

    if err := os.WriteFile(path, data, 0644); err != nil {
        return fmt.Errorf("failed to write config file: %w", err)
    }

    return nil
}
```

### Preset Configuration Templates

Located in `etc/scanner/`:

| File | Description |
|------|-------------|
| `default.json` | Full spectrum scan with default parameters |
| `433-only.json` | 433 MHz band focused scan |
| `keyfob.json` | Optimized for key fob detection (300-433 MHz) |
| `fast-hopper.json` | Minimal frequency list for rapid detection |
| `high-sensitivity.json` | Lower threshold, longer dwell time |

### CLI Usage Example

```bash
# Scan using default configuration
ys1-scan

# Scan using specific configuration file
ys1-scan --config etc/scanner/433-only.json

# Scan with overrides
ys1-scan --config etc/scanner/default.json --threshold -85

# Generate configuration from current settings
ys1-scan --save-config my-scan.json
```

## Future Enhancements

1. **Spectrum sweep mode** - Full band sweep with RSSI array output
2. **Waterfall display data** - Time-series RSSI for visualization
3. **Protocol detection** - Identify signal modulation and encoding
4. **Frequency database** - Known frequency/protocol lookup
5. **Multi-device support** - Parallel scanning with multiple YS1 units
6. **Squelch control** - Dynamic threshold adjustment
7. **Center frequency tracking** - Follow drifting signals

## References

- [Flipper Zero Firmware Analysis](flipper-firmware.md)
- [CC1110/CC1111 Datasheet](cc1110-cc1111.md)
- [YardStick One Interfaces](ys1-interfaces.md)
- [RfCat Functionality](rfcat-functionality.md)
