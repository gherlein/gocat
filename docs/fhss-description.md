# FHSS (Frequency Hopping Spread Spectrum) in gocat

This document provides a detailed description of how FHSS works in gocat, including the theory of operation, the firmware mechanisms, and practical usage of the `fhss-demo` program.

## Table of Contents

1. [What is FHSS?](#what-is-fhss)
2. [How Frequency Hopping Works](#how-frequency-hopping-works)
3. [CC1111 Hardware Support](#cc1111-hardware-support)
4. [The Channel Sequence](#the-channel-sequence)
5. [Firmware FHSS State Machine](#firmware-fhss-state-machine)
6. [Timer-Driven Hopping](#timer-driven-hopping)
7. [Synchronization Process](#synchronization-process)
8. [gocat FHSS Package API](#gocat-fhss-package-api)
9. [Using fhss-demo](#using-fhss-demo)
10. [Practical Considerations](#practical-considerations)

---

## What is FHSS?

Frequency Hopping Spread Spectrum (FHSS) is a wireless transmission technique where the carrier frequency changes ("hops") according to a predetermined sequence. Instead of transmitting on a single frequency, the radio rapidly switches between many frequencies in a pattern known to both transmitter and receiver.

**Benefits of FHSS:**
- **Interference resistance** - If one frequency is jammed or has interference, only a small portion of data is affected
- **Security** - Without knowing the hop sequence, it's difficult to intercept or jam communications
- **Regulatory compliance** - FHSS allows higher power in ISM bands (FCC Part 15.247)
- **Multi-user support** - Multiple FHSS systems can coexist with different hop sequences

---

## How Frequency Hopping Works

### The Basic Concept

```
Time →
        ┌─────┐     ┌─────┐     ┌─────┐     ┌─────┐
Ch 0    │ TX  │     │     │     │     │     │ TX  │
        └─────┘     └─────┘     └─────┘     └─────┘
        ┌─────┐     ┌─────┐     ┌─────┐     ┌─────┐
Ch 1    │     │     │     │     │ TX  │     │     │
        └─────┘     └─────┘     └─────┘     └─────┘
        ┌─────┐     ┌─────┐     ┌─────┐     ┌─────┐
Ch 2    │     │     │ TX  │     │     │     │     │
        └─────┘     └─────┘     └─────┘     └─────┘
           ↑           ↑           ↑           ↑
         Dwell      Dwell       Dwell       Dwell
         Period     Period      Period      Period
```

1. **Dwell Time** - The radio stays on each frequency for a fixed period (e.g., 100-400 ms)
2. **Hop Sequence** - An ordered list of channel numbers that determines the hopping pattern
3. **Synchronization** - Both devices must be on the same channel at the same time

### Frequency Calculation

The CC1111 radio calculates the actual frequency as:

```
Actual Frequency = Base Frequency + (Channel Number × Channel Spacing)
```

Where:
- **Base Frequency** is set via FREQ2/FREQ1/FREQ0 registers
- **Channel Number** is the value in the CHANNR register (0-255)
- **Channel Spacing** is derived from MDMCFG0/MDMCFG1 registers

For example, with:
- Base frequency: 433.0 MHz
- Channel spacing: 200 kHz (0.2 MHz)
- Channel number: 5

The actual frequency would be: 433.0 + (5 × 0.2) = 434.0 MHz

---

## CC1111 Hardware Support

The CC1111 chip (used in YardStick One) has built-in support that makes FHSS efficient:

### CHANNR Register (0xF6)

The CHANNR register selects the current channel. The radio synthesizer automatically adjusts to:
```
f_carrier = f_base + CHANNR × f_channel_spacing
```

Changing CHANNR only requires a brief synthesizer calibration (~100 µs), making rapid hopping possible.

### Timer T2

The CC1111's Timer T2 is used for automatic hop timing:
- Generates periodic interrupts at the dwell time interval
- When the interrupt fires, the firmware changes to the next channel
- This provides precise, consistent timing independent of USB communication

---

## The Channel Sequence

### How Channels Are Decided

The hop sequence is configured via the `FHSS_SET_CHANNELS` command, which takes:

```
[num_channels_lo][num_channels_hi][ch0][ch1][ch2]...[chN]
```

The firmware stores this as an array `g_Channels[]` of up to 880 entries. When hopping:

1. A channel index counter `curChanIdx` starts at 0
2. On each hop, `curChanIdx` increments (wrapping at `NumChannels`)
3. The actual channel is `g_Channels[curChanIdx]`
4. This value is written to the CHANNR register

### Sequence Types

**Sequential:** Channels in order (0, 1, 2, 3, ...)
```go
channels := make([]uint8, 20)
for i := range channels {
    channels[i] = uint8(i)  // 0, 1, 2, ..., 19
}
```

**Pseudo-random:** Channels in a scrambled order for better interference rejection
```go
// Simple pseudo-random sequence using LFSR
channels := make([]uint8, 20)
lfsr := uint8(0xACE1)
for i := range channels {
    channels[i] = lfsr % 50  // Map to 50 physical channels
    lfsr = (lfsr >> 1) ^ (-(lfsr & 1) & 0xB400)
}
```

**Custom:** Application-specific patterns
```go
// Skip known interference frequencies
channels := []uint8{0, 2, 4, 6, 8, 10, 12, 14, 16, 18}  // Only even channels
```

---

## Firmware FHSS State Machine

The rfcat firmware implements an FHSS MAC layer with these states:

| State | Value | Description |
|-------|-------|-------------|
| `MAC_STATE_NONHOPPING` | 0x00 | Normal operation, no hopping |
| `MAC_STATE_DISCOVERY` | 0x01 | Scanning for FHSS networks |
| `MAC_STATE_SYNCHING` | 0x02 | Attempting to sync with master |
| `MAC_STATE_SYNCHED` | 0x03 | Successfully synchronized, hopping |
| `MAC_STATE_SYNC_MASTER` | 0x04 | Acting as sync master |
| `MAC_STATE_SYNCINGMASTER` | 0x05 | Actively transmitting beacons |

### State Transitions

```
                    ┌─────────────────────┐
                    │  MAC_STATE_NONHOPPING│
                    └──────────┬──────────┘
                               │
          ┌────────────────────┼────────────────────┐
          │                    │                    │
          ▼                    ▼                    ▼
┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐
│MAC_STATE_DISCOVERY│ │MAC_STATE_SYNCHING│ │MAC_STATE_SYNC_MASTER│
└────────┬─────────┘ └────────┬─────────┘ └────────┬─────────┘
         │                    │                    │
         │                    ▼                    ▼
         │           ┌──────────────────┐ ┌──────────────────────┐
         └──────────▶│MAC_STATE_SYNCHED │ │MAC_STATE_SYNCINGMASTER│
                     └──────────────────┘ └──────────────────────┘
```

---

## Timer-Driven Hopping

### How the Timer Works

When `FHSS_START_HOPPING` is called:

1. Timer T2 is configured with the dwell time period
2. T2 interrupt is enabled
3. On each T2 interrupt, the ISR:
   ```c
   void T2_ISR(void) {
       curChanIdx = (curChanIdx + 1) % NumChannels;
       CHANNR = g_Channels[curChanIdx];
       NumChannelHops++;
       // Re-calibrate synthesizer
       RFST = SIDLE;
       RFST = SRX;  // or STX depending on mode
   }
   ```

### Dwell Time Configuration

The dwell time is set via `FHSS_SET_MAC_PERIOD`:

```go
// Set 100ms dwell time
fh.SetMACPeriod(100)  // Value in milliseconds
```

Typical dwell times:
- **Fast hopping:** 10-50 ms (requires good sync)
- **Standard:** 100-200 ms (good balance)
- **Slow hopping:** 200-400 ms (easier sync, more vulnerable to interference)

---

## Synchronization Process

For two devices to communicate via FHSS, they must be synchronized - on the same channel at the same time.

### Master/Client Roles

**Master (Beacon Transmitter):**
1. Enters `MAC_STATE_SYNC_MASTER` or `MAC_STATE_SYNCINGMASTER`
2. Starts hopping through the channel sequence
3. Transmits beacon packets on each channel containing:
   - Current channel index
   - Timing information
   - Optional cell ID

**Client (Synchronized Receiver):**
1. Enters `MAC_STATE_SYNCHING`
2. Listens on channel 0 (or scans all channels)
3. When beacon received:
   - Extracts channel index from beacon
   - Calculates timer offset to align with master
   - Adjusts T2 counter
   - Enters `MAC_STATE_SYNCHED`
   - Begins hopping in sync with master

### Maintaining Synchronization

Once synchronized:
- Both devices hop through the same sequence at the same rate
- Small timing drifts are corrected by periodic beacon reception
- If sync is lost, client reverts to `MAC_STATE_SYNCHING`

---

## gocat FHSS Package API

The `pkg/fhss` package provides a high-level interface:

### Creating an FHSS Controller

```go
import "github.com/herlein/gocat/pkg/fhss"

fh := fhss.New(device)
```

### Configuring the Hop Sequence

```go
// Create a 20-channel sequence
channels := make([]uint8, 20)
for i := range channels {
    channels[i] = uint8(i)
}

err := fh.SetChannels(channels)
```

### Starting/Stopping Hopping

```go
// Start automatic hopping
err := fh.StartHopping()

// Stop hopping
err := fh.StopHopping()
```

### Manual Channel Control

```go
// Hop to next channel in sequence
nextCh, err := fh.NextChannel()

// Jump to specific channel
err := fh.ChangeChannel(5)
```

### MAC State Control

```go
// Become the master
err := fh.BecomeMaster()

// Start client synchronization
err := fh.StartSync(cellID)

// Get current state
state, err := fh.GetState()
fmt.Println(state)  // "SyncMaster", "Synching", etc.
```

### Transmitting During FHSS

```go
// FHSS_XMIT - transmit while hopping
err := fh.Transmit([]byte("Hello, FHSS!"))
```

---

## Using fhss-demo

The `fhss-demo` program demonstrates FHSS with three modes:

### Quick Start

**Terminal 1 - Start Master:**
```bash
./bin/fhss-demo -mode master -d '#0' -c tests/etc/433-2fsk-std-4.8k.json
```

**Terminal 2 - Start Client:**
```bash
./bin/fhss-demo -mode client -d '#1' -c tests/etc/433-2fsk-std-4.8k.json
```

### Command-Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `-mode` | (required) | `master`, `client`, or `manual` |
| `-c` | (required) | Radio configuration JSON file |
| `-d` | first device | Device selector (`#0`, `#1`, serial, bus:addr) |
| `-channels` | 20 | Number of channels in hop sequence |
| `-dwell` | 100 | Dwell time per channel (milliseconds) |
| `-cell` | 0 | Cell ID for synchronization |
| `-v` | false | Verbose output |

### Mode Descriptions

**Master Mode (`-mode master`):**
- Sets device as sync master
- Starts automatic frequency hopping
- Transmits beacon messages on each channel
- Output shows state and transmitted beacons:
  ```
  [SyncMaster] TX: BEACON:000001
  [SyncMaster] TX: BEACON:000002
  ```

**Client Mode (`-mode client`):**
- Attempts to synchronize with a master
- Listens for beacon messages
- Shows synchronization state and received data:
  ```
  [Synching] Waiting...
  [Synched] RX: BEACON:000001
  ```

**Manual Mode (`-mode manual`):**
- No synchronization - just manual channel hopping
- Useful for testing hop mechanics
- Shows each channel hop:
  ```
  Hop #1 -> Channel 0
  Hop #2 -> Channel 1
  Hop #3 -> Channel 2
  ```

### Example Session

**Window 1 (Master on device #0):**
```
$ ./bin/fhss-demo -mode master -d '#0' -c tests/etc/433-2fsk-std-4.8k.json -channels 10 -dwell 200
Connected to: YardStick One (Serial: 009a)
Setting up 10-channel hop sequence
=== FHSS Master Mode ===
Dwell time: 200 ms
Press Ctrl+C to stop

Master started - hopping and transmitting beacons
[SyncMaster] TX: BEACON:000000
[SyncMaster] TX: BEACON:000001
[SyncMaster] TX: BEACON:000002
...
```

**Window 2 (Client on device #1):**
```
$ ./bin/fhss-demo -mode client -d '#1' -c tests/etc/433-2fsk-std-4.8k.json -channels 10 -dwell 200
Connected to: YardStick One (Serial: 00b2)
Setting up 10-channel hop sequence
=== FHSS Client Mode ===
Cell ID: 0
Press Ctrl+C to stop

Attempting to synchronize with master...
Client started - listening for beacons
[Synching] Waiting...
[Synching] Waiting...
[Synched] RX: BEACON:000005
[Synched] RX: BEACON:000006
...
```

---

## Practical Considerations

### Channel Selection

**Avoid interference:**
- Stay within the ISM band for your region
- Don't use channels near WiFi or other active devices
- Consider using a subset of available channels

**Channel spacing:**
- Wider spacing (e.g., 500 kHz) = better channel isolation
- Narrower spacing (e.g., 100 kHz) = more channels available
- Must match the radio's configured channel spacing (MDMCFG0/1)

### Dwell Time Selection

| Dwell Time | Pros | Cons |
|------------|------|------|
| Short (10-50 ms) | Better interference rejection | Harder to sync, more overhead |
| Medium (100-200 ms) | Good balance | Standard choice |
| Long (200-400 ms) | Easy to sync | More vulnerable to jamming |

### Synchronization Tips

1. **Same channel sequence** - Both devices must have identical `SetChannels()` calls
2. **Same dwell time** - Both must use the same `-dwell` value
3. **Start master first** - Client needs beacons to sync to
4. **Allow sync time** - Client may take several seconds to acquire sync
5. **Use same config** - Both devices need identical radio settings

### Debugging FHSS Issues

**No synchronization:**
- Verify both devices have same channel sequence
- Check that master is actually transmitting (use SDR to verify)
- Try longer dwell times
- Verify radio configurations match

**Intermittent sync:**
- Increase dwell time
- Check for interference on some channels
- Verify timing parameters match

**Get MAC data for debugging:**
```go
macData, err := fh.GetMACData()
if err == nil {
    fmt.Printf("State: %s\n", macData.State)
    fmt.Printf("Channel Index: %d/%d\n", macData.CurChanIdx, macData.NumChannels)
    fmt.Printf("Total Hops: %d\n", macData.NumChannelHops)
}
```

---

## References

- CC1111 Datasheet (Texas Instruments SWRS033)
- rfcat firmware source: `appFHSSNIC.c`, `cc1111rf.c`
- gocat FHSS package: `pkg/fhss/fhss.go`
- gocat constants: `pkg/yardstick/constants.go` (FHSS commands and MAC states)
