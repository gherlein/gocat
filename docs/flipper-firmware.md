# Flipper Zero CC1101 Firmware Analysis

This document analyzes the Flipper Zero CC1101 driver implementation to guide the development of similar functionality in Go for the YardStick One.

## Overview

The Flipper Zero firmware implements a layered CC1101 driver architecture:
1. **Low-level driver** (`lib/drivers/cc1101.c`) - Direct SPI communication with CC1101
2. **Configuration presets** (`lib/subghz/devices/cc1101_configs.c`) - Pre-defined radio configurations
3. **Device abstraction** (`lib/subghz/devices/types.h`) - Generic SubGhz device interface
4. **External CC1101 support** (`applications/drivers/subghz/cc1101_ext/`) - External module driver

## Key Constants

From `cc1101_regs.h`:

```
Crystal Frequency:   CC1101_QUARTZ = 26,000,000 Hz
Frequency Mask:      CC1101_FMASK  = 0xFFFFFF
Frequency Divisor:   CC1101_FDIV   = 0x10000 (65536)
IF Divisor:          CC1101_IFDIV  = 0x400 (1024)
SPI Timeout:         CC1101_TIMEOUT = 250ms

SPI Control Bits:
- CC1101_READ  = 0x80 (bit 7) - Read operation
- CC1101_BURST = 0x40 (bit 6) - Burst access mode
```

## Low-Level Driver Functions

### SPI Communication

| Function | Purpose | Go Implementation Notes |
|----------|---------|------------------------|
| `cc1101_strobe(handle, strobe)` | Execute command strobe | Send single byte, receive status |
| `cc1101_write_reg(handle, reg, data)` | Write single register | Send [reg, data], check status |
| `cc1101_read_reg(handle, reg, *data)` | Read single register | Send [reg\|0x80, 0], receive data in byte 2 |

### Device Control Functions

| Function | Strobe Command | Purpose |
|----------|---------------|---------|
| `cc1101_reset()` | `SRES` (0x30) | Software reset chip |
| `cc1101_get_status()` | `SNOP` (0x3D) | Read status byte only |
| `cc1101_shutdown()` | `SPWD` (0x39) | Enter power-down mode |
| `cc1101_calibrate()` | `SCAL` (0x33) | Calibrate frequency synthesizer |
| `cc1101_switch_to_idle()` | `SIDLE` (0x36) | Exit RX/TX, enter idle |
| `cc1101_switch_to_rx()` | `SRX` (0x34) | Enable receive mode |
| `cc1101_switch_to_tx()` | `STX` (0x35) | Enable transmit mode |
| `cc1101_flush_rx()` | `SFRX` (0x3A) | Flush RX FIFO |
| `cc1101_flush_tx()` | `SFTX` (0x3B) | Flush TX FIFO |

### Status Polling

```c
bool cc1101_wait_status_state(handle, state, timeout_us)
```
- Polls chip status using `SNOP` strobe
- Waits until `status.STATE` matches target state
- Returns true if state reached within timeout

### Status Byte Structure

```
Bit 7:   CHIP_RDYn (0 = ready, 1 = not ready)
Bits 6-4: STATE (current radio state)
Bits 3-0: FIFO_BYTES_AVAILABLE
```

Radio States (3 bits):
- `000` (0): IDLE
- `001` (1): RX
- `010` (2): TX
- `011` (3): FSTXON (Fast TX ready)
- `100` (4): CALIBRATE
- `101` (5): SETTLING
- `110` (6): RXFIFO_OVERFLOW
- `111` (7): TXFIFO_UNDERFLOW

### Device Identification

| Function | Register | Purpose |
|----------|----------|---------|
| `cc1101_get_partnumber()` | `STATUS_PARTNUM` (0x30) | Read chip part number |
| `cc1101_get_version()` | `STATUS_VERSION` (0x31) | Read chip version |
| `cc1101_get_rssi()` | `STATUS_RSSI` (0x34) | Read raw RSSI value |

### Frequency Configuration

```c
uint32_t cc1101_set_frequency(handle, value_hz)
```

**Formula:**
```
FREQ_REG = (freq_hz * 65536) / 26000000

Writes to:
- FREQ2 (0x0D): bits 23-16
- FREQ1 (0x0E): bits 15-8
- FREQ0 (0x0F): bits 7-0
```

**Returns:** Actual synthesized frequency (may differ slightly due to quantization)

### Intermediate Frequency

```c
uint32_t cc1101_set_intermediate_frequency(handle, value_hz)
```

**Formula:**
```
IF_REG = (freq_hz * 1024) / 26000000
Writes to: FSCTRL0 (0x0C)
```

### Power Amplifier Table

```c
void cc1101_set_pa_table(handle, value[8])
```
- Writes 8-byte PA table using burst mode
- Register: `PATABLE` (0x3E) with `CC1101_BURST`
- Used for ASK/OOK modulation ramping and power control

### FIFO Operations

```c
uint8_t cc1101_write_fifo(handle, data, size)
```
- Writes up to 64 bytes to TX FIFO
- Uses burst write to `FIFO` (0x3F) register
- Returns bytes written

```c
uint8_t cc1101_read_fifo(handle, data, *size)
```
- First reads byte count from FIFO
- Then reads data bytes (max 64)
- Returns bytes read

## Register Map Summary

### Configuration Registers (0x00-0x2E)

| Register | Address | Description |
|----------|---------|-------------|
| IOCFG2 | 0x00 | GDO2 output pin configuration |
| IOCFG1 | 0x01 | GDO1 output pin configuration |
| IOCFG0 | 0x02 | GDO0 output pin configuration |
| FIFOTHR | 0x03 | RX/TX FIFO thresholds |
| SYNC1 | 0x04 | Sync word high byte |
| SYNC0 | 0x05 | Sync word low byte |
| PKTLEN | 0x06 | Packet length |
| PKTCTRL1 | 0x07 | Packet automation control 1 |
| PKTCTRL0 | 0x08 | Packet automation control 0 |
| ADDR | 0x09 | Device address |
| CHANNR | 0x0A | Channel number |
| FSCTRL1 | 0x0B | Frequency synthesizer control 1 |
| FSCTRL0 | 0x0C | Frequency synthesizer control 0 |
| FREQ2 | 0x0D | Frequency control word high |
| FREQ1 | 0x0E | Frequency control word mid |
| FREQ0 | 0x0F | Frequency control word low |
| MDMCFG4 | 0x10 | Modem config (bandwidth, data rate exp) |
| MDMCFG3 | 0x11 | Modem config (data rate mantissa) |
| MDMCFG2 | 0x12 | Modem config (modulation, sync mode) |
| MDMCFG1 | 0x13 | Modem config (preamble, channel spacing) |
| MDMCFG0 | 0x14 | Modem config (channel spacing) |
| DEVIATN | 0x15 | Modem deviation setting |
| MCSM2 | 0x16 | Main radio state machine config 2 |
| MCSM1 | 0x17 | Main radio state machine config 1 |
| MCSM0 | 0x18 | Main radio state machine config 0 |
| FOCCFG | 0x19 | Frequency offset compensation |
| BSCFG | 0x1A | Bit synchronization config |
| AGCCTRL2 | 0x1B | AGC control 2 |
| AGCCTRL1 | 0x1C | AGC control 1 |
| AGCCTRL0 | 0x1D | AGC control 0 |
| WOREVT1 | 0x1E | Wake on radio event 0 timeout high |
| WOREVT0 | 0x1F | Wake on radio event 0 timeout low |
| WORCTRL | 0x20 | Wake on radio control |
| FREND1 | 0x21 | Front end RX config |
| FREND0 | 0x22 | Front end TX config |
| FSCAL3 | 0x23 | Frequency synthesizer cal 3 |
| FSCAL2 | 0x24 | Frequency synthesizer cal 2 |
| FSCAL1 | 0x25 | Frequency synthesizer cal 1 |
| FSCAL0 | 0x26 | Frequency synthesizer cal 0 |

### Strobe Commands (0x30-0x3D)

| Strobe | Address | Description |
|--------|---------|-------------|
| SRES | 0x30 | Reset chip |
| SFSTXON | 0x31 | Enable frequency synthesizer |
| SXOFF | 0x32 | Turn off crystal oscillator |
| SCAL | 0x33 | Calibrate frequency synthesizer |
| SRX | 0x34 | Enable RX |
| STX | 0x35 | Enable TX |
| SIDLE | 0x36 | Exit RX/TX, enter idle |
| SWOR | 0x38 | Start Wake-on-Radio |
| SPWD | 0x39 | Enter power down |
| SFRX | 0x3A | Flush RX FIFO |
| SFTX | 0x3B | Flush TX FIFO |
| SWORRST | 0x3C | Reset WOR timer |
| SNOP | 0x3D | No operation (get status) |

### Status Registers (0x30-0x3D with BURST bit)

| Register | Address | Description |
|----------|---------|-------------|
| PARTNUM | 0x30 | Chip part number |
| VERSION | 0x31 | Chip version |
| FREQEST | 0x32 | Frequency offset estimate |
| LQI | 0x33 | Link quality indicator |
| RSSI | 0x34 | Received signal strength |
| MARCSTATE | 0x35 | Main radio state |
| WORTIME1 | 0x36 | WOR timer high byte |
| WORTIME0 | 0x37 | WOR timer low byte |
| PKTSTATUS | 0x38 | Packet status |
| VCO_VC_DAC | 0x39 | VCO calibration |
| TXBYTES | 0x3A | TX FIFO bytes |
| RXBYTES | 0x3B | RX FIFO bytes |

## Pre-defined Presets

The firmware includes several pre-configured radio presets:

### OOK 270kHz Async
- **Use case:** Key fobs, simple remotes
- **Bandwidth:** 270.833 kHz
- **Data rate:** 3.79 kBaud
- **Modulation:** ASK/OOK, no preamble/sync
- **Key settings:**
  - IOCFG0 = 0x0D (async serial data)
  - PKTCTRL0 = 0x32 (async, continuous)
  - MDMCFG2 = 0x30 (ASK/OOK)
  - PA Table: [0x00, 0xC0, ...] (12dBm)

### OOK 650kHz Async
- **Use case:** Wider bandwidth reception
- **Bandwidth:** 650 kHz
- **Data rate:** 3.79 kBaud
- **Key difference:** MDMCFG4 = 0x17

### 2FSK Dev 2.38kHz Async
- **Use case:** Narrow-band FSK
- **Bandwidth:** 270.833 kHz
- **Deviation:** 2.38 kHz
- **Key settings:**
  - MDMCFG2 = 0x04 (2-FSK)
  - DEVIATN = 0x04

### 2FSK Dev 47.6kHz Async
- **Use case:** Wide-band FSK
- **Deviation:** 47.6 kHz
- **Key settings:**
  - DEVIATN = 0x47

### MSK 99.97kb Async
- **Use case:** High-speed MSK
- **Data rate:** 99.97 kBaud
- **Sync word:** 0x464C
- **Key settings:**
  - MDMCFG2 = 0x72

### GFSK 9.99kb Async
- **Use case:** Standard GFSK
- **Data rate:** 9.99 kBaud
- **Deviation:** 19.04 kHz
- **Sync word:** 0x464C

## High-Level Device Interface

The `SubGhzDeviceInterconnect` structure defines the device abstraction:

### Lifecycle Management
- `begin()` - Initialize device
- `end()` - Shutdown device
- `is_connect()` - Check if device connected
- `reset()` - Reset device
- `sleep()` - Enter sleep mode
- `idle()` - Enter idle mode

### Configuration
- `load_preset(preset, preset_data)` - Load radio configuration
- `set_frequency(frequency)` - Set operating frequency
- `is_frequency_valid(frequency)` - Validate frequency

### Transmit Operations
- `set_tx()` - Switch to TX mode
- `flush_tx()` - Flush TX FIFO
- `start_async_tx(callback, context)` - Start async transmission
- `is_async_complete_tx()` - Check TX completion
- `stop_async_tx()` - Stop async transmission

### Receive Operations
- `set_rx()` - Switch to RX mode
- `flush_rx()` - Flush RX FIFO
- `start_async_rx(callback, context)` - Start async reception
- `stop_async_rx()` - Stop async reception

### Signal Quality
- `get_rssi()` - Get RSSI in dBm
- `get_lqi()` - Get link quality indicator

### Packet Operations
- `rx_pipe_not_empty()` - Check for received data
- `is_rx_data_crc_valid()` - Validate received CRC
- `read_packet(data, size)` - Read packet from FIFO
- `write_packet(data, size)` - Write packet to FIFO

## RSSI Conversion

The Flipper firmware converts raw RSSI to dBm:

```c
float rssi_dbm;
if (rssi_raw >= 128) {
    rssi_dbm = ((rssi_raw - 256.0f) / 2.0f) - 74.0f;
} else {
    rssi_dbm = (rssi_raw / 2.0f) - 74.0f;
}
```

## Valid Frequency Ranges

The CC1101 supports these frequency bands:
- **Band 1:** 299,999,755 - 348,000,335 Hz (~300-348 MHz)
- **Band 2:** 386,999,938 - 464,000,000 Hz (~387-464 MHz)
- **Band 3:** 778,999,847 - 928,000,000 Hz (~779-928 MHz)

## Async RX/TX Implementation

### Async RX
1. Configure GDO0 for async serial data output (IOCFG0 = 0x0D)
2. Set up GPIO interrupt on rising/falling edges
3. Use timer to measure pulse durations
4. Callback receives (level, duration) for each edge

### Async TX
1. Configure GDO0 as output
2. Use DMA to toggle GPIO based on timer
3. Callback provides next level/duration
4. Double-buffering with half-transfer interrupts

## Frequency Scanning and Signal Detection

The Flipper Zero implements a two-stage frequency analyzer for detecting RF signals across the sub-GHz spectrum.

### Overview

The frequency analyzer uses a **coarse-then-fine** scanning approach:
1. **Coarse scan:** Wide bandwidth (650 kHz), scans predefined frequency list
2. **Fine scan:** Narrow bandwidth (58 kHz), scans ±300 kHz around detected signal

### Key Constants

```c
SUBGHZ_FREQUENCY_ANALYZER_THRESHOLD = -93.0f  // dBm, minimum signal strength
```

### Scanning Presets

Two bandwidth presets are used during scanning:

**Wide Bandwidth (Coarse Scan):**
```c
{CC1101_MDMCFG4, 0b00010111}  // Rx BW filter = 650 kHz
```

**Narrow Bandwidth (Fine Scan):**
```c
{CC1101_MDMCFG4, 0b11110111}  // Rx BW filter = 58.035714 kHz
```

### Radio Configuration for Scanning

Before scanning, the radio is configured with optimized AGC settings:

```c
// Symbol rate configuration
CC1101_MDMCFG3 = 0b01111111

// AGC settings optimized for signal detection
CC1101_AGCCTRL2 = 0b00000111  // DVGA all, MAX LNA+LNA2, MAGN_TARGET 42 dB
CC1101_AGCCTRL1 = 0b00001000  // LNA2 gain minimum first, carrier sense disabled
CC1101_AGCCTRL0 = 0b00110000  // No hysteresis, 64 samples AGC, 4dB boundary
```

### Frequency Lists

The firmware defines two types of frequency lists:

**Standard Frequency List** (for frequency analyzer coarse scan):
```c
// 300-348 MHz band
300000000, 303875000, 304250000, 310000000, 315000000, 318000000

// 387-464 MHz band
390000000, 418000000, 433075000, 433420000, 433920000 (default),
434420000, 434775000, 438900000

// 779-928 MHz band
868350000, 915000000, 925000000
```

**Hopper Frequency List** (subset for quick scanning):
```c
310000000, 315000000, 318000000, 390000000, 433920000, 868350000
```

### Scanning Algorithm

#### Phase 1: Coarse Scan

```
1. Set idle mode
2. Load wide bandwidth preset (650 kHz)
3. For each frequency in frequency list:
   a. Switch to idle
   b. Set frequency
   c. Calibrate synthesizer
   d. Wait for IDLE state
   e. Switch to RX mode
   f. Wait 2ms for AGC to settle
   g. Read RSSI
   h. Track frequency with highest RSSI
4. If max RSSI > threshold (-93 dBm):
   → Proceed to fine scan
```

#### Phase 2: Fine Scan

```
1. Set idle mode
2. Load narrow bandwidth preset (58 kHz)
3. Scan range: coarse_frequency ± 300 kHz, step 20 kHz
   For each frequency in range:
   a. Switch to idle
   b. Set frequency
   c. Calibrate synthesizer
   d. Wait for IDLE state
   e. Switch to RX mode
   f. Wait 2ms for AGC to settle
   g. Read RSSI
   h. Track frequency with highest RSSI
4. Return fine frequency with highest RSSI
```

### Adaptive Frequency Smoothing

The detected frequency is smoothed using an adaptive exponential running average:

```c
float k;
if (fabsf(newVal - filVal) > 500000.0f)
    k = 0.9;   // Fast adaptation for large changes
else
    k = 0.03;  // Slow adaptation for small changes

filVal += (newVal - filVal) * k;
```

This prevents jitter in the displayed frequency while still responding quickly to new signals.

### Signal Hold Behavior

- **Sample hold counter:** 20 counts
- When signal detected, counter resets to 20
- Counter decrements each scan cycle when no signal
- At count 15, a "signal lost" callback is triggered
- At count 0, frequency display clears

### Scanning Timing

- Main loop delay: 10ms between scan cycles
- Per-frequency dwell time: 2ms for RSSI measurement
- Coarse scan: ~17 frequencies × 2ms = ~34ms
- Fine scan: ~30 frequencies × 2ms = ~60ms (±300kHz in 20kHz steps)
- Total cycle time: ~100-150ms depending on signal presence

### Data Structures

```c
typedef struct {
    uint32_t frequency_coarse;
    float rssi_coarse;
    uint32_t frequency_fine;
    float rssi_fine;
} FrequencyRSSI;
```

### Callback Interface

```c
typedef void (*SubGhzFrequencyAnalyzerWorkerPairCallback)(
    void* context,
    uint32_t frequency,  // Detected frequency in Hz
    float rssi,          // Signal strength in dBm
    bool signal          // true = signal present, false = signal lost
);
```

### Go Implementation Recommendations for Frequency Scanning

```go
// FrequencyScanResult holds the result of a frequency scan
type FrequencyScanResult struct {
    FrequencyCoarse uint32
    RSSICoarse      float32
    FrequencyFine   uint32
    RSSIFine        float32
}

// FrequencyScanner interface for signal detection
type FrequencyScanner interface {
    // ScanFrequencies performs a coarse scan across frequency list
    ScanFrequencies(frequencies []uint32) (*FrequencyScanResult, error)

    // ScanRange performs a fine scan around a center frequency
    ScanRange(center uint32, rangeHz uint32, stepHz uint32) (*FrequencyScanResult, error)

    // SetRSSIThreshold sets minimum signal detection threshold
    SetRSSIThreshold(threshold float32)
}

// Default frequency lists
var DefaultFrequencies = []uint32{
    300000000, 303875000, 304250000, 310000000, 315000000, 318000000,
    390000000, 418000000, 433075000, 433420000, 433920000,
    434420000, 434775000, 438900000,
    868350000, 915000000, 925000000,
}

var HopperFrequencies = []uint32{
    310000000, 315000000, 318000000, 390000000, 433920000, 868350000,
}

// Scanning presets
var ScanPresetWide = Preset{
    Name: "Scan Wide 650kHz",
    Registers: [][2]byte{
        {MDMCFG4, 0x17},  // 650 kHz bandwidth
    },
}

var ScanPresetNarrow = Preset{
    Name: "Scan Narrow 58kHz",
    Registers: [][2]byte{
        {MDMCFG4, 0xF7},  // 58 kHz bandwidth
    },
}

const (
    DefaultRSSIThreshold = -93.0  // dBm
    FineScanRange        = 300000 // ±300 kHz
    FineScanStep         = 20000  // 20 kHz steps
    DwellTimeMs          = 2      // ms per frequency
)
```

### Scanning Sequence (Pseudocode)

```go
func (s *Scanner) DetectSignal() (*FrequencyScanResult, error) {
    result := &FrequencyScanResult{}

    // Phase 1: Coarse scan
    s.device.Idle()
    s.device.LoadPreset(ScanPresetWide)

    for _, freq := range s.frequencies {
        s.device.Idle()
        actualFreq, _ := s.device.SetFrequency(freq)
        s.device.Calibrate()
        s.device.WaitForState(StateIdle, 10*time.Millisecond)
        s.device.RX()
        time.Sleep(2 * time.Millisecond)

        rssi := s.device.GetRSSI()
        if rssi > result.RSSICoarse {
            result.RSSICoarse = rssi
            result.FrequencyCoarse = actualFreq
        }
    }

    // Check threshold
    if result.RSSICoarse < s.threshold {
        return result, nil  // No signal detected
    }

    // Phase 2: Fine scan
    s.device.Idle()
    s.device.LoadPreset(ScanPresetNarrow)

    startFreq := result.FrequencyCoarse - FineScanRange
    endFreq := result.FrequencyCoarse + FineScanRange

    for freq := startFreq; freq < endFreq; freq += FineScanStep {
        s.device.Idle()
        actualFreq, _ := s.device.SetFrequency(freq)
        s.device.Calibrate()
        s.device.WaitForState(StateIdle, 10*time.Millisecond)
        s.device.RX()
        time.Sleep(2 * time.Millisecond)

        rssi := s.device.GetRSSI()
        if rssi > result.RSSIFine {
            result.RSSIFine = rssi
            result.FrequencyFine = actualFreq
        }
    }

    return result, nil
}
```

## Go Implementation Recommendations

### Required Functions

1. **Low-level SPI:**
   - `Strobe(cmd byte) Status`
   - `WriteReg(reg, data byte) Status`
   - `ReadReg(reg byte) (byte, Status)`
   - `WriteBurst(reg byte, data []byte) Status`
   - `ReadBurst(reg byte, length int) ([]byte, Status)`

2. **Device Control:**
   - `Reset() error`
   - `GetStatus() Status`
   - `Idle() error`
   - `RX() error`
   - `TX() error`
   - `Sleep() error`
   - `FlushRX() error`
   - `FlushTX() error`
   - `Calibrate() error`

3. **Configuration:**
   - `SetFrequency(hz uint32) (uint32, error)`
   - `SetModulation(mod Modulation) error`
   - `SetDataRate(baud float64) error`
   - `SetDeviation(hz float64) error`
   - `SetBandwidth(hz float64) error`
   - `SetPATable(table [8]byte) error`
   - `LoadPreset(preset Preset) error`

4. **Data Transfer:**
   - `WriteFIFO(data []byte) error`
   - `ReadFIFO() ([]byte, error)`
   - `GetRSSI() float64`
   - `GetLQI() byte`

5. **Status:**
   - `GetPartNumber() byte`
   - `GetVersion() byte`
   - `WaitForState(state State, timeout time.Duration) error`

### Status Structure

```go
type Status struct {
    Ready           bool  // CHIP_RDYn (inverted)
    State           State // Current radio state
    FIFOBytesAvail  byte  // FIFO bytes available
}

type State byte
const (
    StateIdle State = iota
    StateRX
    StateTX
    StateFSTXON
    StateCalibrate
    StateSettling
    StateRXOverflow
    StateTXUnderflow
)
```

### Preset Configuration

Define presets as register/value pairs similar to Flipper:

```go
type Preset struct {
    Name      string
    Registers [][2]byte // {register, value} pairs
    PATable   [8]byte
}

var PresetOOK270kHz = Preset{
    Name: "OOK 270kHz",
    Registers: [][2]byte{
        {IOCFG0, 0x0D},
        {FIFOTHR, 0x47},
        {PKTCTRL0, 0x32},
        // ...
    },
    PATable: [8]byte{0x00, 0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
}
```

## References

- Flipper Zero firmware: `/external/flipperzero-firmware/`
- CC1101/CC1110 datasheet: `docs/cc1110-cc1111.md`
- RfCat functionality: `docs/rfcat-functionality.md`
- YS1 interfaces: `docs/ys1-interfaces.md`
