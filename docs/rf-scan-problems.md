# RF Scanner Analysis: Differences Between Python Implementations and Go Scanner

This document analyzes the differences between the Python RF scanning implementations (rfcat, Flipper Zero) and our Go scanner implementation, identifying potential causes for missed RF energy detection.

## Overview

Our Go scanner consistently shows -90.5 dBm across all frequencies (noise floor) and fails to detect RF transmissions that should be visible. This analysis compares our implementation against working Python implementations to identify the root causes.

## Critical Differences

### 1. CRITICAL: Missing MCSM0 Auto-Calibration Setting

**Flipper/rfcat**: Set `MCSM0 = 0x18`
```c
CC1101_MCSM0, 0x18, // Autocalibrate on idle-to-rx/tx, PO_TIMEOUT is 64 cycles(149-155us)
```

**Our Go code**: Does NOT set MCSM0

**Impact**: Without auto-calibration enabled, the frequency synthesizer may not lock properly when transitioning to RX mode. The manual SCAL strobe we issue might not be sufficient.

**Fix**: Add `MCSM0 = 0x18` to the coarse preset.

---

### 2. CRITICAL: Missing FIFOTHR ADC_RETENTION Bit

**Flipper**: Sets `FIFOTHR = 0x47` or `0x07` (ADC_RETENTION bit = bit 6)
```c
CC1101_FIFOTHR, 0x47, // The only important bit is ADC_RETENTION
```

**Our Go code**: Does NOT set FIFOTHR

**Impact**: The ADC_RETENTION bit keeps the ADC circuitry powered. Without it, **RSSI readings can be invalid** because the ADC needs time to power up after each state transition. This is likely why we see 0x80 (invalid) values so frequently.

**Fix**: Add `FIFOTHR = 0x47` to the coarse preset.

---

### 3. Missing State Verification After RX Strobe

**Flipper**: Waits for RX state confirmation with status polling
```c
cc1101_switch_to_rx(&furi_hal_spi_bus_handle_subghz);
// waiting for the chip to switch to Rx mode
furi_check(cc1101_wait_status_state(&furi_hal_spi_bus_handle_subghz, CC1101StateRX, 10000));
```

**Our Go code**: Just sleeps 1ms + dwell time
```go
// 5. Enter RX mode using strobe (faster than firmware command)
if err := s.device.StrobeModeRX(); err != nil {
    return 0, fmt.Errorf("failed to set RX: %w", err)
}

// 6. Wait for AGC to settle - minimum 1ms for state transition + dwell time
time.Sleep(time.Millisecond + dwellTime)
```

**Impact**: The radio might not actually be in RX mode when we read RSSI. State transitions can take variable time depending on PLL lock, calibration, etc.

**Fix**: Poll MARCSTATE until RX state (0x0D) is confirmed before reading RSSI.

---

### 4. RSSI Conversion Formula Analysis

**rfcat** (uses XOR approach):
```python
rssi_values = [ (ord23(x)^0x80)/2 - 88 for x in rssi_values ]
```

**Flipper** (uses signed conversion):
```c
float rssi = rssi_dec;
if(rssi_dec >= 128) {
    rssi = ((rssi - 256.0f) / 2.0f) - 74.0f;
} else {
    rssi = (rssi / 2.0f) - 74.0f;
}
```

**Our Go code**:
```go
func RSSIToDBm(rssi uint8) float32 {
    if rssi == 0x80 {
        return RSSIInvalidValue
    }
    if rssi >= 128 {
        return float32(int(rssi)-256)/2.0 - 74.0
    }
    return float32(rssi)/2.0 - 74.0
}
```

**Analysis**: Our formula matches Flipper (using -74 offset for CC1111). The rfcat -88 offset with XOR is mathematically equivalent to signed conversion but uses a different offset, likely due to different crystal frequency or front-end configuration.

**Status**: Formula is correct ✓

---

### 5. AGC Configuration Comparison

**Flipper 650kHz OOK preset** (closest to our wide scan):
```c
CC1101_AGCCTRL2, 0x07, // 00 - DVGA all; 000 - MAX LNA+LNA2; 111 - MAIN_TARGET 42 dB
CC1101_AGCCTRL1, 0x00, // LNA 2 gain decreased first, relative carrier sense disabled
CC1101_AGCCTRL0, 0x91, // Medium hysteresis, 16 samples AGC, 8dB boundary
```

**Our Go code**:
```go
CoarseAGCCTRL2 uint8 = 0x07
CoarseAGCCTRL1 uint8 = 0x00
CoarseAGCCTRL0 uint8 = 0x91
```

**Status**: AGC settings match! ✓

---

### 6. Missing FSCTRL1 (IF Frequency)

**Flipper**: Sets `FSCTRL1 = 0x06` (IF = ~152 kHz)
```c
CC1101_FSCTRL1, 0x06, // IF = (26*10^6) / (2^10) * 0x06 = 152343.75Hz
```

**Our Go code**: Does NOT set FSCTRL1 (uses chip default)

**Impact**: IF (Intermediate Frequency) affects receiver sensitivity and RSSI accuracy. The default may not be optimal for scanning.

**Fix**: Add `FSCTRL1 = 0x06` to the coarse preset.

---

### 7. Missing TEST Registers for Wide Bandwidth

**rfcat**: Adjusts TEST1/TEST2 based on bandwidth
```python
def setMdmChanBW(self, bw, ...):
    # ... calculate bandwidth ...

    # Adjust TEST1 and TEST2 registers based on bandwidth
    if bw > 325e3:
        self.setRFRegister(TEST2, 0x88)  # Wider bandwidth
        self.setRFRegister(TEST1, 0x31)
    else:
        self.setRFRegister(TEST2, 0x81)  # Narrower bandwidth
        self.setRFRegister(TEST1, 0x35)
```

**Our Go code**: Does NOT set TEST1/TEST2

**Impact**: TEST registers affect analog front-end performance. Without proper settings for wide bandwidth, sensitivity may be degraded.

**Fix**: Add `TEST2 = 0x88` and `TEST1 = 0x31` for wide bandwidth scanning.

---

### 8. Missing FOCCFG (Frequency Offset Compensation)

**Flipper**: Sets `FOCCFG = 0x18` or `0x16`
```c
CC1101_FOCCFG, 0x18, // no frequency offset compensation, POST_K same as PRE_K, PRE_K is 4K, GATE is off
```

**Our Go code**: Does NOT set FOCCFG (uses chip default)

**Impact**: Frequency offset compensation settings affect how the receiver tracks frequency variations. The default may cause issues with signal detection.

**Fix**: Add `FOCCFG = 0x18` to the coarse preset.

---

### 9. Architecture Difference: Firmware vs Software Scanning

**rfcat Spectrum Analyzer**:
- Sends `RFCAT_START_SPECAN` command to firmware
- Firmware handles multi-channel scanning internally
- Radio stays in RX mode continuously
- Firmware rapidly sweeps through channels and returns bulk RSSI data
- Minimal state transitions, maximum dwell time per frequency

**Our Go Scanner**:
- Software-controlled scanning
- For each frequency: IDLE → set freq → CAL → RX → read RSSI → IDLE
- Many state transitions (5 per frequency)
- Overhead from USB communication for each step
- Short effective dwell time due to transition overhead

**Impact**: Our approach has much more overhead and the radio spends less time actually receiving on each frequency. A signal could easily be missed during state transitions.

**Potential Fix**: Consider using the YardStick One's firmware spectrum analyzer mode if available, or minimize state transitions.

---

## Summary: Priority Fixes

| Priority | Issue | Register | Value | Impact |
|----------|-------|----------|-------|--------|
| **1** | Missing ADC_RETENTION bit | FIFOTHR | 0x47 | RSSI values invalid without powered ADC |
| **2** | Missing auto-calibration | MCSM0 | 0x18 | Frequency synth may not lock |
| **3** | No state verification | N/A | Poll MARCSTATE | May read RSSI before RX mode active |
| **4** | Missing IF frequency | FSCTRL1 | 0x06 | Receiver sensitivity affected |
| **5** | Missing wide BW TEST regs | TEST2/TEST1 | 0x88/0x31 | Analog performance degraded |
| **6** | Missing freq offset comp | FOCCFG | 0x18 | Signal tracking issues |
| **7** | Short dwell time | N/A | Increase to 5-10ms | More time to detect signals |

---

## Recommended Code Changes

### 1. Update `pkg/scanner/constants.go`

Add these registers to the coarse preset:

```go
const (
    // Existing coarse preset values...
    CoarseMDMCFG4  uint8 = 0x1F
    CoarseMDMCFG3  uint8 = 0x7F
    CoarseMDMCFG2  uint8 = 0x30
    CoarseAGCCTRL2 uint8 = 0x07
    CoarseAGCCTRL1 uint8 = 0x00
    CoarseAGCCTRL0 uint8 = 0x91
    CoarseFREND1   uint8 = 0xB6
    CoarseFREND0   uint8 = 0x10

    // NEW: Critical missing registers
    CoarseMCSM0    uint8 = 0x18  // Auto-calibrate on idle-to-rx/tx
    CoarseFIFOTHR  uint8 = 0x47  // ADC_RETENTION bit - CRITICAL for RSSI
    CoarseFSCTRL1  uint8 = 0x06  // IF frequency ~152 kHz
    CoarseTEST2    uint8 = 0x88  // Wide bandwidth analog setting
    CoarseTEST1    uint8 = 0x31  // Wide bandwidth analog setting
    CoarseFOCCFG   uint8 = 0x18  // Frequency offset compensation
)
```

### 2. Update `measureRSSIOnce()` in `pkg/scanner/scanner.go`

Add state verification after RX strobe:

```go
// 5. Enter RX mode using strobe
if err := s.device.StrobeModeRX(); err != nil {
    return 0, fmt.Errorf("failed to set RX: %w", err)
}

// 5a. NEW: Wait for RX state to be confirmed
for i := 0; i < 100; i++ {
    state, err := s.device.GetMARCSTATE()
    if err != nil {
        break
    }
    if state == 0x0D { // RX state confirmed
        break
    }
    time.Sleep(100 * time.Microsecond)
}

// 6. Wait for AGC to settle
time.Sleep(dwellTime)
```

### 3. Update `loadPreset()` to include new registers

Add loading for the new registers (MCSM0, FIFOTHR, FSCTRL1, TEST2, TEST1, FOCCFG).

---

## Testing Plan

After implementing fixes:

1. Run `rf-scanner` with debug output and verify:
   - No more 0x80 invalid RSSI values
   - RSSI varies (not stuck at -90.5 dBm)

2. Test signal detection:
   - Start `rf-scanner` on one YardStick
   - Transmit on another YardStick at known frequency
   - Scanner should detect signal above threshold

3. Compare with rfcat:
   - Run rfcat's `specan()` on same frequency range
   - Compare detected signals with our scanner

---

## Alternative Approach: Firmware-Based Spectrum Analyzer

The YardStick One firmware includes a built-in spectrum analyzer mode that handles all channel sweeping internally. This is how rfcat's `specan()` function works and provides much better performance than software-controlled scanning.

### How rfcat Firmware Spectrum Analyzer Works

#### Protocol Overview

1. **Start Scanning**: Host sends `RFCAT_START_SPECAN` (0x40) command via APP_NIC (0x42)
2. **Firmware Loop**: Firmware sweeps through channels, reading RSSI at each
3. **Data Return**: Firmware sends RSSI data back via APP_SPECAN (0x43), SPECAN_QUEUE (0x01)
4. **Stop Scanning**: Host sends `RFCAT_STOP_SPECAN` (0x41) command

#### Firmware State Machine

From `appFHSSNIC.c`:

```c
case MAC_STATE_PREP_SPECAN:
    RFOFF;
    PKTCTRL1 =  0xE5;       // highest PQT, address check, append_status
    PKTCTRL0 =  0x04;       // crc enabled (we don't want packets)
    FSCTRL1 =   0x12;       // freq if
    FSCTRL0 =   0x00;
    MCSM0 =     0x10;       // autocal/no auto-cal
    AGCCTRL2 |= AGCCTRL2_MAX_DVGA_GAIN;  // disable 3 highest gain settings
    macdata.mac_state = MAC_STATE_SPECAN;
    chan_table = rfrxbuf[0];

case MAC_STATE_SPECAN:
    for (processbuffer = 0; processbuffer < macdata.synched_chans; processbuffer++)
    {
        /* tune radio and start RX */
        CHANNR = processbuffer;  // Uses channel number directly
        RFOFF;
        RFRX;
        sleepMillis(2);          // 2ms dwell time per channel

        /* read RSSI */
        chan_table[processbuffer] = (RSSI);
    }

    /* end RX */
    RFOFF;
    txdata( APP_SPECAN, SPECAN_QUEUE, (u8)macdata.synched_chans, (__xdata u8*)&chan_table[0] );
    break;
```

#### Python Client Side (rfcat)

From `rflib/__init__.py`:

```python
def _doSpecAn(self, centfreq, inc, count):
    '''
    store radio config and start sending spectrum analysis data
    centfreq = Center Frequency
    '''
    if count > 255:
        raise Exception("sorry, only 255 samples per pass... (count)")

    spectrum = (count * inc)
    halfspec = spectrum / 2.0
    basefreq = centfreq - halfspec

    self.getRadioConfig()
    self._specan_backup_radiocfg = self.radiocfg

    self.setFreq(basefreq)       # Set base frequency (FREQ0/1/2)
    self.setMdmChanSpc(inc)      # Set channel spacing (MDMCFG0/1)

    freq, fbytes = self.getFreq()
    delta = self.getMdmChanSpc()

    self.send(APP_NIC, RFCAT_START_SPECAN, b"%c" % (count))  # count = num channels
    return freq, delta

def _stopSpecAn(self):
    '''stop sending rfdata and return radio to original config'''
    self.send(APP_NIC, RFCAT_STOP_SPECAN, b'')
    self.radiocfg = self._specan_backup_radiocfg
    self.setRadioConfig()
```

#### RSSI Data Reception

From `rflib/ccspecan.py`:

```python
while not self._stopping:
    rssi_values, timestamp = self._data.recv(APP_SPECAN, SPECAN_QUEUE, 10000)
    # Convert raw RSSI to dBm using XOR method
    rssi_values = [ ((ord23(x)^0x80)/2) - 88 for x in rssi_values ]
    frequency_axis = numpy.linspace(self._low_frequency, self._high_frequency,
                                     num=len(rssi_values), endpoint=True)
    self._new_frame_callback(numpy.copy(frequency_axis), numpy.copy(rssi_values))
```

### Design: Go Firmware-Based Scanner

#### Constants (pkg/yardstick/constants.go additions)

```go
// Spectrum Analyzer Commands (sent via APP_NIC)
const (
    SPECANStart = 0x40  // RFCAT_START_SPECAN - start spectrum analysis
    SPECANStop  = 0x41  // RFCAT_STOP_SPECAN - stop spectrum analysis
)

// Spectrum Analyzer Application
const (
    AppSPECAN     = 0x43  // Already exists
    SPECANQueue   = 0x01  // Queue ID for spectrum data
)
```

#### New Scanner Interface

```go
// pkg/scanner/firmware_scanner.go

type FirmwareScanner struct {
    device     *yardstick.Device
    baseFreq   uint32  // Base frequency in Hz
    chanSpacing uint32 // Channel spacing in Hz
    numChans   uint8   // Number of channels (max 255)
    running    bool
    dataChan   chan []float32
}

// NewFirmwareScanner creates a scanner that uses the firmware spectrum analyzer
func NewFirmwareScanner(device *yardstick.Device) *FirmwareScanner {
    return &FirmwareScanner{
        device:   device,
        dataChan: make(chan []float32, 10),
    }
}

// Configure sets up the spectrum analyzer parameters
func (fs *FirmwareScanner) Configure(centerFreq, bandwidth uint32, numChans uint8) error {
    if numChans > 255 {
        return fmt.Errorf("max 255 channels supported")
    }

    // Calculate base frequency and channel spacing
    halfBW := bandwidth / 2
    fs.baseFreq = centerFreq - halfBW
    fs.chanSpacing = bandwidth / uint32(numChans)
    fs.numChans = numChans

    // Set base frequency on device
    if err := fs.device.SetFrequency(fs.baseFreq); err != nil {
        return fmt.Errorf("failed to set frequency: %w", err)
    }

    // Set channel spacing
    if err := fs.device.SetChannelSpacing(fs.chanSpacing); err != nil {
        return fmt.Errorf("failed to set channel spacing: %w", err)
    }

    return nil
}

// Start begins the firmware spectrum analyzer
func (fs *FirmwareScanner) Start() error {
    if fs.running {
        return fmt.Errorf("scanner already running")
    }

    // Send START_SPECAN command with channel count
    cmd := []byte{fs.numChans}
    if err := fs.device.SendCommand(yardstick.AppNIC, yardstick.SPECANStart, cmd); err != nil {
        return fmt.Errorf("failed to start specan: %w", err)
    }

    fs.running = true

    // Start receive goroutine
    go fs.receiveLoop()

    return nil
}

// Stop halts the firmware spectrum analyzer
func (fs *FirmwareScanner) Stop() error {
    if !fs.running {
        return nil
    }

    fs.running = false

    // Send STOP_SPECAN command
    if err := fs.device.SendCommand(yardstick.AppNIC, yardstick.SPECANStop, nil); err != nil {
        return fmt.Errorf("failed to stop specan: %w", err)
    }

    close(fs.dataChan)
    return nil
}

// Data returns a channel that receives RSSI data frames
func (fs *FirmwareScanner) Data() <-chan []float32 {
    return fs.dataChan
}

// receiveLoop continuously receives RSSI data from firmware
func (fs *FirmwareScanner) receiveLoop() {
    for fs.running {
        // Receive from APP_SPECAN, SPECAN_QUEUE
        data, err := fs.device.RecvFromApp(yardstick.AppSPECAN, yardstick.SPECANQueue, 10*time.Second)
        if err != nil {
            if fs.running {
                log.Printf("specan recv error: %v", err)
            }
            continue
        }

        if len(data) == 0 {
            continue
        }

        // Convert raw RSSI to dBm
        rssiDBm := make([]float32, len(data))
        for i, raw := range data {
            // rfcat formula: (raw ^ 0x80) / 2 - 88
            // This is equivalent to signed conversion with -88 offset
            rssiDBm[i] = float32(int8(raw^0x80))/2.0 - 88.0
        }

        select {
        case fs.dataChan <- rssiDBm:
        default:
            // Drop frame if channel full
        }
    }
}

// GetFrequencyForChannel returns the frequency for a given channel index
func (fs *FirmwareScanner) GetFrequencyForChannel(chanIdx int) uint32 {
    return fs.baseFreq + uint32(chanIdx)*fs.chanSpacing
}
```

#### Required Device Methods

Add these methods to `pkg/yardstick/device.go`:

```go
// SetChannelSpacing sets the channel spacing for spectrum analysis
// Uses MDMCFG0 and MDMCFG1 registers
func (d *Device) SetChannelSpacing(spacing uint32) error {
    // Calculate CHANSPC_E and CHANSPC_M from spacing
    // spacing = (Fxtal / 2^18) * (256 + CHANSPC_M) * 2^CHANSPC_E
    // For 26 MHz crystal: spacing = 99.182129 * (256 + M) * 2^E

    // Find E and M that give closest match
    fxtal := float64(26000000)
    target := float64(spacing)

    var bestE, bestM uint8
    bestError := math.MaxFloat64

    for e := uint8(0); e < 4; e++ {
        // m = (spacing * 2^18) / (fxtal * 2^e) - 256
        m := (target * float64(uint32(1)<<18)) / (fxtal * float64(uint32(1)<<e)) - 256
        if m >= 0 && m <= 255 {
            actual := (fxtal / float64(uint32(1)<<18)) * (256 + m) * float64(uint32(1)<<e)
            err := math.Abs(actual - target)
            if err < bestError {
                bestError = err
                bestE = e
                bestM = uint8(m)
            }
        }
    }

    // MDMCFG1[1:0] = CHANSPC_E
    mdmcfg1, err := d.GetRFRegister(MDMCFG1)
    if err != nil {
        return err
    }
    mdmcfg1 = (mdmcfg1 & 0xFC) | (bestE & 0x03)

    if err := d.SetRFRegister(MDMCFG1, mdmcfg1); err != nil {
        return err
    }

    // MDMCFG0 = CHANSPC_M
    return d.SetRFRegister(MDMCFG0, bestM)
}

// RecvFromApp receives data from a specific application and queue
func (d *Device) RecvFromApp(app, queue uint8, timeout time.Duration) ([]byte, error) {
    // Implementation depends on existing EP5 receive mechanism
    // Need to filter by app/queue in the response
    // ...
}
```

### Advantages of Firmware-Based Approach

| Aspect | Software Scanning | Firmware Scanning |
|--------|-------------------|-------------------|
| USB transactions per sweep | 5 × N channels | 2 (start + data) |
| Dwell time per channel | ~1-5ms effective | 2ms guaranteed |
| State transitions | IDLE→CAL→RX per channel | Single RX mode |
| Latency | High (USB round-trips) | Low (bulk transfer) |
| CPU usage | High | Low |
| Max sweep rate | ~10-50 Hz | ~100+ Hz |

### Implementation Steps

1. **Add Constants**: Add `SPECANStart`, `SPECANStop`, `SPECANQueue` to constants
2. **Add Device Methods**:
   - `SetChannelSpacing()`
   - `RecvFromApp()` or modify existing receive to filter by app
3. **Create FirmwareScanner**: New scanner implementation
4. **Update rf-scanner CLI**: Add `-firmware` flag to use firmware scanner
5. **Test**: Verify against rfcat's specan output

### Limitations

- Maximum 255 channels per sweep (firmware limitation)
- Fixed 2ms dwell time per channel in firmware
- No per-channel configuration (all channels use same settings)
- Requires specific firmware version (rfcat FHSSNIC)

---

## References

- CC1101 Datasheet: https://www.ti.com/lit/ds/symlink/cc1101.pdf
- CC1111 Datasheet: https://www.ti.com/lit/ds/symlink/cc1111.pdf
- Flipper Zero CC1101 configs: `/external/flipperzero-firmware/lib/subghz/devices/cc1101_configs.c`
- rfcat chipcon_nic: `/external/rfcat/rflib/chipcon_nic.py`
- rfcat firmware FHSS: `/external/rfcat/firmware/appFHSSNIC.c`
- rfcat spectrum analyzer: `/external/rfcat/rflib/ccspecan.py`
