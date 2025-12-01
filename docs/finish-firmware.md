# Complete Firmware Feature Implementation Plan

This document provides a comprehensive plan for implementing all remaining rfcat firmware features in gocat. The plan is organized into phases with detailed implementation steps, testing strategies, and documentation.

## Current State

### Already Implemented
- USB device communication (EP0, EP5)
- Basic radio configuration (frequency, modulation, data rate, etc.)
- Packet TX/RX via `send-recv`
- Spectrum analyzer (`rf-scanner` using firmware SPECAN)
- Register peek/poke
- Device selection and enumeration
- Radio profiles and configuration management

### Constants Already Defined
The following NIC commands are already in `pkg/yardstick/constants.go`:
- `NICSetAESMode`, `NICGetAESMode`, `NICSetAESIV`, `NICSetAESKey`
- `NICSetAmpMode`, `NICGetAmpMode`
- `NICLongXmit`, `NICLongXmitMore`
- `NICSetRecvLarge`

### Missing Constants
FHSS commands need to be added.

---

## Phase 1: FHSS Constants and Infrastructure

**Goal:** Add all FHSS-related constants and prepare the codebase for FHSS implementation.

### 1.1 Add FHSS Constants to `pkg/yardstick/constants.go`

```go
// FHSS Commands (APP_NIC = 0x42)
const (
    FHSSSetChannels     = 0x10 // Set channel hop sequence
    FHSSNextChannel     = 0x11 // Hop to next channel
    FHSSChangeChannel   = 0x12 // Change to specific channel
    FHSSSetMACThreshold = 0x13 // Set MAC timing threshold
    FHSSGetMACThreshold = 0x14 // Get MAC timing threshold
    FHSSSetMACData      = 0x15 // Set raw MAC data
    FHSSGetMACData      = 0x16 // Get raw MAC data
    FHSSXmit            = 0x17 // Transmit during FHSS
    FHSSGetChannels     = 0x18 // Get channel hop sequence
    FHSSSetState        = 0x20 // Set MAC state
    FHSSGetState        = 0x21 // Get MAC state
    FHSSStartSync       = 0x22 // Start network synchronization
    FHSSStartHopping    = 0x23 // Begin automatic hopping
    FHSSStopHopping     = 0x24 // Stop automatic hopping
    FHSSSetMACPeriod    = 0x25 // Set MAC period/dwell time
)

// FHSS MAC States
const (
    MACStateNonHopping    = 0x00 // Standard non-hopping mode
    MACStateDiscovery     = 0x01 // Network discovery mode
    MACStateSynching      = 0x02 // Synchronizing to master
    MACStateSynched       = 0x03 // Synchronized and hopping
    MACStateSyncMaster    = 0x04 // Operating as sync master
    MACStateSyncingMaster = 0x05 // Actively beaconing as master
    MACStateLongXmit      = 0x06 // Long transmit mode
    MACStateLongXmitFail  = 0x07 // Long transmit failed
    MACStatePrepSpecan    = 0x40 // Preparing spectrum analyzer
    MACStateSpecan        = 0x41 // Spectrum analyzer active
)

// FHSS Limits
const (
    FHSSMaxChannels   = 880 // Maximum channels in hop sequence
    FHSSMaxTXMsgs     = 2   // Number of TX message buffers
    FHSSMaxTXMsgLen   = 240 // Maximum message length per buffer
)
```

### 1.2 Add AES Constants

```go
// AES Crypto Modes (matches CC1111 ENCCS register)
const (
    AESModeECB    = 0x00 // Electronic Codebook
    AESModeCBC    = 0x10 // Cipher Block Chaining
    AESModeCFB    = 0x20 // Cipher Feedback
    AESModeOFB    = 0x30 // Output Feedback
    AESModeCTR    = 0x40 // Counter
    AESModeCBCMAC = 0x50 // CBC Message Authentication Code
)

// AES Crypto Flags
const (
    AESCryptoNone       = 0x00
    AESCryptoOutEnable  = 0x08 // Enable outbound (TX) crypto
    AESCryptoOutEncrypt = 0x04 // Encrypt outbound (else decrypt)
    AESCryptoInEnable   = 0x02 // Enable inbound (RX) crypto
    AESCryptoInEncrypt  = 0x01 // Encrypt inbound (else decrypt)
)

// Common AES configurations
const (
    // Encrypt TX, Decrypt RX with CBC
    AESCryptoDefault = AESModeCBC | AESCryptoOutEnable | AESCryptoOutEncrypt | AESCryptoInEnable
)
```

### 1.3 Files to Modify
- `pkg/yardstick/constants.go` - Add all new constants

### 1.4 Testing
- Verify constants match firmware defines in `FHSS.h` and `global.h`
- Build test to ensure no syntax errors

---

## Phase 2: AES Encryption Support

**Goal:** Implement hardware AES encryption/decryption support.

### 2.1 Create `pkg/yardstick/aes.go`

```go
package yardstick

// AESConfig holds AES encryption configuration
type AESConfig struct {
    Mode      uint8    // AES mode (ECB, CBC, etc.)
    Key       [16]byte // 128-bit encryption key
    IV        [16]byte // 128-bit initialization vector
    EncryptTX bool     // Encrypt outgoing packets
    DecryptRX bool     // Decrypt incoming packets
}

// SetAESMode configures the AES crypto mode
func (d *Device) SetAESMode(mode uint8) error

// GetAESMode returns the current AES mode
func (d *Device) GetAESMode() (uint8, error)

// SetAESKey sets the 128-bit AES encryption key
func (d *Device) SetAESKey(key [16]byte) error

// SetAESIV sets the 128-bit initialization vector
func (d *Device) SetAESIV(iv [16]byte) error

// ConfigureAES is a convenience function to set up AES in one call
func (d *Device) ConfigureAES(cfg *AESConfig) error

// DisableAES turns off all AES encryption
func (d *Device) DisableAES() error
```

### 2.2 Implementation Details

```go
func (d *Device) SetAESMode(mode uint8) error {
    _, err := d.Send(AppNIC, NICSetAESMode, []byte{mode}, USBDefaultTimeout)
    return err
}

func (d *Device) GetAESMode() (uint8, error) {
    resp, err := d.Send(AppNIC, NICGetAESMode, nil, USBDefaultTimeout)
    if err != nil {
        return 0, err
    }
    if len(resp) < 1 {
        return 0, fmt.Errorf("empty response")
    }
    return resp[0], nil
}

func (d *Device) SetAESKey(key [16]byte) error {
    _, err := d.Send(AppNIC, NICSetAESKey, key[:], USBDefaultTimeout)
    return err
}

func (d *Device) SetAESIV(iv [16]byte) error {
    _, err := d.Send(AppNIC, NICSetAESIV, iv[:], USBDefaultTimeout)
    return err
}

func (d *Device) ConfigureAES(cfg *AESConfig) error {
    // Set key first
    if err := d.SetAESKey(cfg.Key); err != nil {
        return fmt.Errorf("set key: %w", err)
    }

    // Set IV
    if err := d.SetAESIV(cfg.IV); err != nil {
        return fmt.Errorf("set IV: %w", err)
    }

    // Build mode byte
    mode := cfg.Mode
    if cfg.EncryptTX {
        mode |= AESCryptoOutEnable | AESCryptoOutEncrypt
    }
    if cfg.DecryptRX {
        mode |= AESCryptoInEnable
    }

    return d.SetAESMode(mode)
}

func (d *Device) DisableAES() error {
    return d.SetAESMode(AESCryptoNone)
}
```

### 2.3 Create `cmd/aes-test/main.go`

Test tool for AES functionality:
- Set key and IV
- Encrypt data on one device
- Decrypt on another
- Verify round-trip

### 2.4 Testing Strategy
1. Unit tests for message construction
2. Integration test with two YardStick devices
3. Verify encrypted data cannot be read without correct key
4. Test all AES modes (ECB, CBC, CBC-MAC)

---

## Phase 3: Amplifier Control

**Goal:** Implement external amplifier control for YardStick One.

### 3.1 Add to `pkg/yardstick/device.go` or create `pkg/yardstick/amplifier.go`

```go
// AmpMode represents the amplifier state
type AmpMode uint8

const (
    AmpModeOff AmpMode = 0 // Amplifier disabled
    AmpModeOn  AmpMode = 1 // Amplifier enabled
)

// SetAmpMode enables or disables the external amplifier
func (d *Device) SetAmpMode(mode AmpMode) error {
    _, err := d.Send(AppNIC, NICSetAmpMode, []byte{byte(mode)}, USBDefaultTimeout)
    return err
}

// GetAmpMode returns the current amplifier state
func (d *Device) GetAmpMode() (AmpMode, error) {
    resp, err := d.Send(AppNIC, NICGetAmpMode, nil, USBDefaultTimeout)
    if err != nil {
        return AmpModeOff, err
    }
    if len(resp) < 1 {
        return AmpModeOff, fmt.Errorf("empty response")
    }
    return AmpMode(resp[0]), nil
}

// EnableAmplifier is a convenience function
func (d *Device) EnableAmplifier() error {
    return d.SetAmpMode(AmpModeOn)
}

// DisableAmplifier is a convenience function
func (d *Device) DisableAmplifier() error {
    return d.SetAmpMode(AmpModeOff)
}
```

### 3.2 Integration with Transmit
- Add `--amp` flag to `send-recv` command
- Auto-enable amp for TX, disable after

### 3.3 Testing
- Verify amplifier state changes
- Measure power output difference with SDR

---

## Phase 4: Long Packet Transmission

**Goal:** Implement transmission of packets larger than 255 bytes.

### 4.1 Create `pkg/yardstick/longxmit.go`

```go
package yardstick

import (
    "fmt"
)

// LongTransmit sends a packet larger than RFMaxTXBlock bytes
// Data is chunked and streamed to the device
func (d *Device) LongTransmit(data []byte) error {
    if len(data) == 0 {
        return fmt.Errorf("empty data")
    }

    if len(data) <= RFMaxTXBlock {
        // Use standard transmit for small packets
        return d.Transmit(data, 0, 0)
    }

    totalLen := uint16(len(data))

    // Calculate number of initial blocks to preload
    // Firmware expects at least 2 blocks preloaded
    numPreload := 2
    if len(data) < RFMaxTXChunk*2 {
        numPreload = 1
    }

    // Build initial command: [len_lo][len_hi][num_blocks][data...]
    initData := make([]byte, 3+numPreload*RFMaxTXChunk)
    initData[0] = byte(totalLen & 0xFF)
    initData[1] = byte(totalLen >> 8)
    initData[2] = byte(numPreload)

    // Copy initial blocks
    offset := 3
    for i := 0; i < numPreload && i*RFMaxTXChunk < len(data); i++ {
        end := (i + 1) * RFMaxTXChunk
        if end > len(data) {
            end = len(data)
        }
        copy(initData[offset:], data[i*RFMaxTXChunk:end])
        offset += RFMaxTXChunk
    }

    // Send NIC_LONG_XMIT to start
    resp, err := d.Send(AppNIC, NICLongXmit, initData[:offset], USBTXWaitTimeout)
    if err != nil {
        return fmt.Errorf("start long xmit: %w", err)
    }
    if len(resp) > 0 && resp[0] != RCNoError {
        return fmt.Errorf("start long xmit failed: 0x%02x", resp[0])
    }

    // Send remaining chunks
    dataOffset := numPreload * RFMaxTXChunk
    for dataOffset < len(data) {
        chunkEnd := dataOffset + RFMaxTXChunk
        if chunkEnd > len(data) {
            chunkEnd = len(data)
        }

        chunk := data[dataOffset:chunkEnd]
        chunkData := make([]byte, 1+len(chunk))
        chunkData[0] = byte(len(chunk))
        copy(chunkData[1:], chunk)

        resp, err := d.Send(AppNIC, NICLongXmitMore, chunkData, USBTXWaitTimeout)
        if err != nil {
            return fmt.Errorf("long xmit chunk at %d: %w", dataOffset, err)
        }

        // Check for buffer not available - need to retry
        if len(resp) > 0 && resp[0] == RCTempErrBufferNotAvailable {
            // Buffer full, wait and retry
            time.Sleep(10 * time.Millisecond)
            continue
        }

        if len(resp) > 0 && resp[0] != RCNoError {
            return fmt.Errorf("long xmit chunk failed: 0x%02x", resp[0])
        }

        dataOffset = chunkEnd
    }

    // Send final empty chunk to signal completion
    resp, err = d.Send(AppNIC, NICLongXmitMore, []byte{0}, USBTXWaitTimeout)
    if err != nil {
        return fmt.Errorf("finalize long xmit: %w", err)
    }
    if len(resp) > 0 && resp[0] != LCENoError {
        return fmt.Errorf("long xmit completion failed: 0x%02x", resp[0])
    }

    return nil
}
```

### 4.2 Large Packet Reception

```go
// SetLargeReceive configures the device for receiving large packets
// Set size to 0 to disable large receive mode
func (d *Device) SetLargeReceive(size uint16) error {
    data := []byte{byte(size & 0xFF), byte(size >> 8)}
    resp, err := d.Send(AppNIC, NICSetRecvLarge, data, USBDefaultTimeout)
    if err != nil {
        return err
    }
    // Response contains the configured size
    if len(resp) >= 2 {
        configured := uint16(resp[0]) | uint16(resp[1])<<8
        if configured != size {
            return fmt.Errorf("configured size %d != requested %d", configured, size)
        }
    }
    return nil
}
```

### 4.3 Update `send-recv` Command
- Add `--long` flag for long packet mode
- Support reading large files for transmission

### 4.4 Testing
- Transmit 1KB, 10KB, 64KB packets
- Verify data integrity
- Test buffer underrun recovery

---

## Phase 5: FHSS (Frequency Hopping Spread Spectrum)

**Goal:** Implement full FHSS support including channel hopping and synchronization.

### 5.1 Create `pkg/fhss/fhss.go`

```go
package fhss

import (
    "fmt"
    "sync"
    "time"

    "github.com/herlein/gocat/pkg/yardstick"
)

// FHSS provides frequency hopping spread spectrum functionality
type FHSS struct {
    device   *yardstick.Device
    channels []uint8
    state    MACState
    mu       sync.Mutex
}

// MACState represents the current FHSS MAC layer state
type MACState uint8

// MACData contains MAC layer timing and state information
type MACData struct {
    State           MACState
    TxMsgIdx        uint8
    TxMsgIdxDone    uint8
    CurChanIdx      uint16
    NumChannels     uint16
    NumChannelHops  uint16
    TLastHop        uint16
    TLastStateChange uint32
    MACThreshold    uint32
    MACTimer        uint32
    DesperatelySeeking uint16
    SynchedChans    uint16
}

// New creates a new FHSS controller
func New(device *yardstick.Device) *FHSS {
    return &FHSS{
        device:   device,
        channels: make([]uint8, 0),
        state:    yardstick.MACStateNonHopping,
    }
}

// SetChannels configures the channel hop sequence
func (f *FHSS) SetChannels(channels []uint8) error {
    if len(channels) > yardstick.FHSSMaxChannels {
        return fmt.Errorf("too many channels: %d > %d", len(channels), yardstick.FHSSMaxChannels)
    }

    f.mu.Lock()
    defer f.mu.Unlock()

    // Build command: [num_channels_lo][num_channels_hi][channel_list...]
    data := make([]byte, 2+len(channels))
    data[0] = byte(len(channels) & 0xFF)
    data[1] = byte(len(channels) >> 8)
    copy(data[2:], channels)

    _, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSSetChannels, data, yardstick.USBDefaultTimeout)
    if err != nil {
        return err
    }

    f.channels = make([]uint8, len(channels))
    copy(f.channels, channels)
    return nil
}

// GetChannels returns the current channel hop sequence
func (f *FHSS) GetChannels() ([]uint8, error) {
    resp, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSGetChannels, nil, yardstick.USBDefaultTimeout)
    if err != nil {
        return nil, err
    }
    return resp, nil
}

// StartHopping begins automatic frequency hopping
func (f *FHSS) StartHopping() error {
    _, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSStartHopping, nil, yardstick.USBDefaultTimeout)
    return err
}

// StopHopping stops automatic frequency hopping
func (f *FHSS) StopHopping() error {
    _, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSStopHopping, nil, yardstick.USBDefaultTimeout)
    return err
}

// NextChannel manually advances to the next channel
func (f *FHSS) NextChannel() (uint8, error) {
    resp, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSNextChannel, nil, yardstick.USBDefaultTimeout)
    if err != nil {
        return 0, err
    }
    if len(resp) < 1 {
        return 0, fmt.Errorf("no channel returned")
    }
    return resp[0], nil
}

// ChangeChannel sets a specific channel
func (f *FHSS) ChangeChannel(channel uint8) error {
    _, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSChangeChannel, []byte{channel}, yardstick.USBDefaultTimeout)
    return err
}

// GetState returns the current MAC state
func (f *FHSS) GetState() (MACState, error) {
    resp, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSGetState, nil, yardstick.USBDefaultTimeout)
    if err != nil {
        return 0, err
    }
    if len(resp) < 1 {
        return 0, fmt.Errorf("no state returned")
    }
    return MACState(resp[0]), nil
}

// SetState sets the MAC state
func (f *FHSS) SetState(state MACState) error {
    _, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSSetState, []byte{byte(state)}, yardstick.USBDefaultTimeout)
    return err
}

// Transmit sends data during FHSS operation
func (f *FHSS) Transmit(data []byte) error {
    if len(data) > yardstick.FHSSMaxTXMsgLen {
        return fmt.Errorf("data too large: %d > %d", len(data), yardstick.FHSSMaxTXMsgLen)
    }

    msg := make([]byte, 1+len(data))
    msg[0] = byte(len(data))
    copy(msg[1:], data)

    _, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSXmit, msg, yardstick.USBDefaultTimeout)
    return err
}

// StartSync begins synchronization to a hopping network
func (f *FHSS) StartSync(cellID uint16) error {
    data := []byte{byte(cellID & 0xFF), byte(cellID >> 8)}
    _, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSStartSync, data, yardstick.USBDefaultTimeout)
    return err
}

// GetMACData returns detailed MAC layer information
func (f *FHSS) GetMACData() (*MACData, error) {
    resp, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSGetMACData, nil, yardstick.USBDefaultTimeout)
    if err != nil {
        return nil, err
    }

    // Parse the MAC_DATA_t structure from firmware
    // This requires careful alignment with the C struct
    if len(resp) < 20 {
        return nil, fmt.Errorf("response too short: %d", len(resp))
    }

    return &MACData{
        State:            MACState(resp[0]),
        TxMsgIdx:         resp[1],
        TxMsgIdxDone:     resp[2],
        CurChanIdx:       uint16(resp[3]) | uint16(resp[4])<<8,
        NumChannels:      uint16(resp[5]) | uint16(resp[6])<<8,
        NumChannelHops:   uint16(resp[7]) | uint16(resp[8])<<8,
        // ... parse remaining fields
    }, nil
}

// SetMACThreshold configures the MAC timing threshold
func (f *FHSS) SetMACThreshold(threshold uint32) error {
    data := []byte{byte(threshold)}
    _, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSSetMACThreshold, data, yardstick.USBDefaultTimeout)
    return err
}

// GetMACThreshold returns the current MAC timing threshold
func (f *FHSS) GetMACThreshold() (uint32, error) {
    resp, err := f.device.Send(yardstick.AppNIC, yardstick.FHSSGetMACThreshold, nil, yardstick.USBDefaultTimeout)
    if err != nil {
        return 0, err
    }
    if len(resp) < 4 {
        return 0, fmt.Errorf("response too short")
    }
    return uint32(resp[0]) | uint32(resp[1])<<8 | uint32(resp[2])<<16 | uint32(resp[3])<<24, nil
}
```

### 5.2 Create `pkg/fhss/discovery.go`

```go
package fhss

import (
    "context"
    "time"
)

// DiscoveryResult contains information about a discovered hopping network
type DiscoveryResult struct {
    CellID      uint16
    Channels    []uint8
    DwellTime   time.Duration
    SyncPattern []byte
}

// Discover searches for FHSS networks
func (f *FHSS) Discover(ctx context.Context, timeout time.Duration) ([]*DiscoveryResult, error) {
    // Set state to discovery mode
    if err := f.SetState(yardstick.MACStateDiscovery); err != nil {
        return nil, err
    }
    defer f.SetState(yardstick.MACStateNonHopping)

    // Implementation depends on protocol being searched for
    // This is a framework for custom discovery implementations

    results := make([]*DiscoveryResult, 0)

    // ... discovery logic

    return results, nil
}
```

### 5.3 Create `cmd/fhss-test/main.go`

Command-line tool for FHSS testing:
- `fhss-test channels` - List/set channels
- `fhss-test hop` - Start/stop hopping
- `fhss-test sync` - Synchronize to network
- `fhss-test master` - Become sync master
- `fhss-test xmit` - Transmit while hopping

### 5.4 Testing Strategy
1. Two-device synchronization test
2. Channel sequence verification
3. Timing accuracy measurement
4. Data transmission during hopping

---

## Phase 6: System Commands

**Goal:** Implement remaining system-level commands.

### 6.1 Add to `pkg/yardstick/device.go`

```go
// Reset triggers a device reset via watchdog
func (d *Device) Reset() error {
    // Magic signature required: "RSTN"
    data := []byte{'R', 'S', 'T', 'N'}
    _, err := d.ControlTransfer(RequestTypeVendorIn, EP0CmdReset,
        uint16(data[0])|uint16(data[1])<<8,
        uint16(data[2])|uint16(data[3])<<8,
        nil, USBDefaultTimeout)
    return err
}

// EnterBootloader puts the device into firmware update mode
func (d *Device) EnterBootloader() error {
    _, err := d.Send(AppSystem, SysCmdBootloader, nil, USBDefaultTimeout)
    return err
}

// GetPartNumber returns the chip part number
func (d *Device) GetPartNumber() (uint8, error) {
    resp, err := d.Send(AppSystem, SysCmdPartNum, nil, USBDefaultTimeout)
    if err != nil {
        return 0, err
    }
    if len(resp) < 1 {
        return 0, fmt.Errorf("empty response")
    }
    return resp[0], nil
}

// GetBuildType returns firmware build information
func (d *Device) GetBuildType() (string, error) {
    resp, err := d.Send(AppSystem, SysCmdBuildType, nil, USBDefaultTimeout)
    if err != nil {
        return "", err
    }
    return string(resp), nil
}

// GetDebugCodes returns the last debug/error codes
func (d *Device) GetDebugCodes() (location uint8, errorCode uint8, error) {
    resp := make([]byte, 2)
    _, err := d.ControlTransfer(RequestTypeVendorIn, EP0CmdGetDebugCodes, 0, 0, resp, USBDefaultTimeout)
    if err != nil {
        return 0, 0, err
    }
    return resp[0], resp[1], nil
}

// SetLEDMode configures LED behavior
func (d *Device) SetLEDMode(mode uint8) error {
    _, err := d.Send(AppSystem, SysCmdLEDMode, []byte{mode}, USBDefaultTimeout)
    return err
}

// GetClock returns the firmware clock value
func (d *Device) GetClock() (uint32, error) {
    resp, err := d.Send(AppSystem, SysCmdGetClock, nil, USBDefaultTimeout)
    if err != nil {
        return 0, err
    }
    if len(resp) < 4 {
        return 0, fmt.Errorf("response too short")
    }
    return uint32(resp[0]) | uint32(resp[1])<<8 | uint32(resp[2])<<16 | uint32(resp[3])<<24, nil
}
```

### 6.2 Create `cmd/ys1-info/main.go`

Device information tool:
- Display part number, serial, firmware build
- Show debug codes
- Test connectivity (ping)

---

## Phase 7: Integration and Polish

**Goal:** Integrate all features and create comprehensive documentation.

### 7.1 Update `send-recv` Command

Add flags for new features:
```
--aes-key       AES encryption key (hex)
--aes-iv        AES initialization vector (hex)
--aes-mode      AES mode (ecb, cbc, cbc-mac)
--amp           Enable amplifier for TX
--long          Use long packet mode
--fhss          Enable FHSS mode
--channels      FHSS channel list (comma-separated)
```

### 7.2 Create Helper Package `pkg/crypto/aes.go`

Utility functions for AES operations:
- Key generation
- Hex encoding/decoding
- Configuration presets

### 7.3 Update README.md

Add sections for:
- AES encryption usage
- FHSS operation
- Long packet transmission
- Amplifier control

### 7.4 Create Examples

- `examples/encrypted-chat/` - Two-device encrypted messaging
- `examples/fhss-beacon/` - FHSS master beacon
- `examples/fhss-client/` - FHSS synchronized client
- `examples/large-file-tx/` - Large file transmission

---

## Phase 8: Testing and Verification

### 8.1 Unit Tests

Create test files for each package:
- `pkg/yardstick/aes_test.go`
- `pkg/yardstick/longxmit_test.go`
- `pkg/fhss/fhss_test.go`

### 8.2 Integration Tests

Require two YardStick One devices:
- `tests/integration/aes_roundtrip_test.go`
- `tests/integration/fhss_sync_test.go`
- `tests/integration/long_packet_test.go`

### 8.3 Hardware Verification Checklist

- [ ] AES encryption verified with known test vectors
- [ ] FHSS timing matches expected dwell times
- [ ] Long packets verified up to 64KB
- [ ] Amplifier power difference measurable
- [ ] All commands return expected responses

---

## README Documentation Sections

The following sections should be added to README.md after implementation:

### AES Encryption

```markdown
## AES Hardware Encryption

gocat supports the CC1111's built-in AES co-processor for encrypted RF communication.

### Quick Start

```bash
# Set up encrypted communication between two devices
# Device 1 (TX):
./bin/send-recv -m send -c configs/433-2fsk.json \
    --aes-key "0123456789ABCDEF0123456789ABCDEF" \
    --aes-mode cbc \
    -data "Secret message"

# Device 2 (RX):
./bin/send-recv -m recv -c configs/433-2fsk.json \
    --aes-key "0123456789ABCDEF0123456789ABCDEF" \
    --aes-mode cbc
```

### Programmatic Usage

```go
import "github.com/herlein/gocat/pkg/yardstick"

// Configure AES
device.ConfigureAES(&yardstick.AESConfig{
    Mode:      yardstick.AESModeCBC,
    Key:       [16]byte{...},
    IV:        [16]byte{...},
    EncryptTX: true,
    DecryptRX: true,
})

// Transmit (automatically encrypted)
device.Transmit([]byte("Hello, World!"), 0, 0)

// Disable encryption
device.DisableAES()
```

### Supported Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| ECB | Electronic Codebook | Simple, no IV needed |
| CBC | Cipher Block Chaining | Recommended for most uses |
| CBC-MAC | Message Authentication | Integrity verification |
```

### Frequency Hopping

```markdown
## Frequency Hopping Spread Spectrum (FHSS)

gocat supports frequency hopping for spread spectrum communication.

### Setting Up Channel Hopping

```go
import "github.com/herlein/gocat/pkg/fhss"

// Create FHSS controller
fh := fhss.New(device)

// Define channel sequence (83 channels is common)
channels := make([]uint8, 83)
for i := range channels {
    channels[i] = uint8(i)
}
fh.SetChannels(channels)

// Start hopping
fh.StartHopping()

// Transmit while hopping
fh.Transmit([]byte("Hopping message"))

// Stop hopping
fh.StopHopping()
```

### Synchronizing Two Devices

```go
// Master device
fh.SetState(yardstick.MACStateSyncMaster)
fh.StartHopping()

// Client device
fh.StartSync(0x0000) // Cell ID
// Will automatically synchronize and begin hopping
```
```

### Long Packet Transmission

```markdown
## Long Packet Transmission

For packets larger than 255 bytes, use the long transmit mode:

```go
// Read a large file
data, _ := os.ReadFile("firmware.bin")

// Transmit (automatically uses long mode if needed)
device.LongTransmit(data)
```

### Command Line

```bash
# Send a large file
./bin/send-recv -m send -c configs/433-2fsk.json --long --file firmware.bin
```
```

### External Amplifier

```markdown
## External Amplifier Control (YardStick One)

The YardStick One has external TX/RX amplifiers that can be controlled:

```go
// Enable amplifier for higher power output
device.EnableAmplifier()

// Transmit with amplifier
device.Transmit(data, 0, 0)

// Disable amplifier
device.DisableAmplifier()
```

**Note:** The amplifier significantly increases power consumption and RF output.
Use responsibly and in compliance with local regulations.
```

---

## Implementation Timeline Estimate

| Phase | Description | Complexity | Dependencies |
|-------|-------------|------------|--------------|
| 1 | Constants | Low | None |
| 2 | AES Encryption | Medium | Phase 1 |
| 3 | Amplifier Control | Low | Phase 1 |
| 4 | Long Packet TX/RX | Medium | Phase 1 |
| 5 | FHSS | High | Phase 1 |
| 6 | System Commands | Low | Phase 1 |
| 7 | Integration | Medium | Phases 2-6 |
| 8 | Testing | Medium | Phase 7 |

**Recommended Order:** 1 → 3 → 2 → 4 → 6 → 5 → 7 → 8

Start with simpler features (amplifier, constants) to build confidence before tackling FHSS.

---

## File Summary

### New Files to Create

| File | Description |
|------|-------------|
| `pkg/yardstick/aes.go` | AES encryption support |
| `pkg/yardstick/amplifier.go` | Amplifier control |
| `pkg/yardstick/longxmit.go` | Long packet transmission |
| `pkg/fhss/fhss.go` | FHSS controller |
| `pkg/fhss/discovery.go` | FHSS network discovery |
| `cmd/aes-test/main.go` | AES testing tool |
| `cmd/fhss-test/main.go` | FHSS testing tool |
| `cmd/ys1-info/main.go` | Device info tool |

### Files to Modify

| File | Changes |
|------|---------|
| `pkg/yardstick/constants.go` | Add FHSS, AES constants |
| `pkg/yardstick/device.go` | Add system commands |
| `cmd/send-recv/main.go` | Add AES, amp, long flags |

---

## Success Criteria

1. **AES**: Two devices can exchange encrypted messages
2. **FHSS**: Two devices can synchronize and hop together
3. **Long TX**: Can transmit 64KB+ without data corruption
4. **Amplifier**: Measurable power increase with SDR
5. **System**: All info commands return valid data
6. **Documentation**: All features documented with examples
7. **Tests**: All unit and integration tests pass
