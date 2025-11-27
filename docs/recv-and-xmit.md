# YardStick One Receive and Transmit Operations

This document describes the detailed sequence of steps required to configure a
YardStick One (YS1) device for receiving or transmitting RF data, and how to
exchange data between the host application and the device.

## Overview

The YS1 uses a CC1111 radio transceiver with the following key characteristics:

- **Communication**: USB bulk transfers on Endpoint 5 (EP5)
- **Radio States**: IDLE, RX, TX (controlled via RFST strobe register)
- **Data Flow**: Asynchronous receive, synchronous transmit
- **Buffering**: Double-buffered receive, single transmit buffer

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Receive Mode Configuration](#receive-mode-configuration)
3. [Transmit Mode Configuration](#transmit-mode-configuration)
4. [Receive Data Flow](#receive-data-flow)
5. [Transmit Data Flow](#transmit-data-flow)
6. [State Machine Transitions](#state-machine-transitions)
7. [Error Handling](#error-handling)
8. [Code Examples](#code-examples)

---

## Prerequisites

Before any RF operation, ensure:

1. **USB Connection Established**
   - Device opened via gousb
   - EP5 IN and OUT endpoints claimed
   - Background receive thread running (optional but recommended)

2. **Radio Configured**
   - Frequency set (FREQ2/FREQ1/FREQ0)
   - Modulation configured (MDMCFG2)
   - Data rate set (MDMCFG4/MDMCFG3)
   - Sync word configured (SYNC1/SYNC0)
   - Packet length mode set (PKTCTRL0/PKTLEN)

3. **Radio in Known State**
   - Verify MARCSTATE register
   - Typically start from IDLE state

---

## Receive Mode Configuration

### Step-by-Step Sequence

#### Step 1: Set Radio to IDLE (if not already)

Before changing configuration, ensure radio is idle:

```
Command: POKE to RFST register (0xDFE1)
Value:   0x04 (SIDLE strobe)
Wait:    Poll MARCSTATE (0xDF3B) until value = 0x01 (IDLE)
```

**EP5 Packet:**
```
Host → Device:
  App:     0xFF (APP_SYSTEM)
  Cmd:     0x81 (SYS_CMD_POKE)
  Payload: [0xE1, 0xDF] (address LE) + [0x04] (SIDLE)
```

#### Step 2: Configure Packet Parameters

Set packet control registers for receive:

| Register | Address | Typical Value | Purpose |
|----------|---------|---------------|---------|
| PKTLEN   | 0xDF02  | 255 (0xFF)    | Max packet length |
| PKTCTRL0 | 0xDF04  | 0x00 or 0x01  | Fixed (0) or Variable (1) length |
| PKTCTRL1 | 0xDF03  | 0x04 or 0x00  | Append status (0x04) or not |

**Variable Length Mode (PKTCTRL0 = 0x01):**
- First byte of received data is length
- More flexible for unknown protocols

**Fixed Length Mode (PKTCTRL0 = 0x00):**
- PKTLEN defines exact packet size
- More efficient for known protocols

#### Step 3: Configure MCSM1 for RX Behavior

The MCSM1 register controls what happens after receiving a packet:

```
MCSM1 (0xDF13) bit fields:
  Bits 5:4 - CCA_MODE (Clear Channel Assessment)
  Bits 3:2 - RXOFF_MODE (State after RX)
  Bits 1:0 - TXOFF_MODE (State after TX)

RXOFF_MODE values:
  0x00 = IDLE after RX
  0x01 = FSTXON after RX
  0x02 = TX after RX
  0x03 = Stay in RX (continuous receive)
```

**For continuous receive:**
```
MCSM1 = 0x0F  // CCA=00, RXOFF=11 (stay RX), TXOFF=11 (go RX)
```

#### Step 4: Set Receive Mode via SYS_CMD_RFMODE

Send the RFMODE command to enter receive:

```
Host → Device:
  App:     0xFF (APP_SYSTEM)
  Cmd:     0x88 (SYS_CMD_RFMODE)
  Payload: [0x02] (RFST_SRX)
```

This command:
1. Sets MCSM1 to stay in RX mode
2. Issues RFST = SRX strobe
3. Waits for MARCSTATE = RX (0x0D)

**Alternative: Direct Strobe**
```
POKE 0xDFE1 = 0x02 (SRX)
Poll MARCSTATE until 0x0D
```

#### Step 5: Optionally Configure Large Receive Blocks

For packets larger than 255 bytes:

```
Host → Device:
  App:     0x42 (APP_NIC)
  Cmd:     0x05 (NIC_SET_RECV_LARGE)
  Payload: [blocksize_lo, blocksize_hi] (max 512)
```

---

## Transmit Mode Configuration

### Step-by-Step Sequence

#### Step 1: Set Radio to IDLE

Same as receive - ensure clean state:

```
POKE 0xDFE1 = 0x04 (SIDLE)
Wait for MARCSTATE = 0x01
```

#### Step 2: Configure Packet Parameters

| Register | Address | Purpose |
|----------|---------|---------|
| PKTLEN   | 0xDF02  | Packet length (fixed mode) |
| PKTCTRL0 | 0xDF04  | Length mode, CRC, whitening |
| PA_TABLE0| 0xDF2E  | TX power level |

**PA_TABLE0 Power Levels (900 MHz band):**
| Value | Approx. Power |
|-------|---------------|
| 0x00  | -30 dBm (off) |
| 0x50  | 0 dBm         |
| 0x8E  | +5 dBm        |
| 0xC0  | +10 dBm       |

#### Step 3: Configure MCSM1 for TX Behavior

```
MCSM1 for transmit with return to RX:
  MCSM1 = 0x0F  // After TX, go to RX

MCSM1 for transmit with return to IDLE:
  MCSM1 = 0x00  // After TX, go to IDLE
```

#### Step 4: (Optional) Set TX Mode via RFMODE

For continuous TX (jamming mode only):

```
Host → Device:
  App:     0xFF (APP_SYSTEM)
  Cmd:     0x88 (SYS_CMD_RFMODE)
  Payload: [0x03] (RFST_STX)
```

**Note:** Normal transmit is done via NIC_XMIT command, not by setting TX mode.

---

## Receive Data Flow

### Firmware Receive Process

1. **Radio receives packet** (handled by CC1111 hardware)
2. **RFIF_IRQ_DONE interrupt** fires when complete packet received
3. **Firmware interrupt handler:**
   - Reads packet from RF RX buffer
   - Sends to host via `txdata(APP_NIC, NIC_RECV, len, data)`
4. **Marks buffer processed** for reuse

### Host Receive Process

#### Polling Method

1. **Send EP5 IN request** with timeout
2. **Parse response packet:**
   ```
   Byte 0:   '@' (0x40) marker
   Byte 1:   App ID (0x42 for NIC)
   Byte 2:   Cmd (0x01 for NIC_RECV)
   Bytes 3-4: Length (little-endian)
   Bytes 5+:  RF data payload
   ```
3. **Check for timeout** - no data available

#### Threaded Method (Recommended)

1. **Background thread** continuously reads EP5 IN
2. **Accumulates data** in receive queue
3. **Parses packets** by finding '@' markers
4. **Sorts into mailboxes** by App ID and Command
5. **Application calls recv()** to get next packet from mailbox

### RFrecv Function Flow

```
RFrecv(timeout, blocksize):
    1. If blocksize specified and > 255:
       - Send NIC_SET_RECV_LARGE command

    2. Call recv(APP_NIC, NIC_RECV, timeout):
       - Check recv_mbox[APP_NIC][NIC_RECV] queue
       - If empty, wait on recv_event with timeout
       - If packet available, pop and return
       - If timeout, raise exception

    3. If encoder configured, decode data

    4. Return (data, timestamp)
```

### Receive Data Format

**Variable Length Mode:**
```
Received packet structure:
  Byte 0:     Packet length (N)
  Bytes 1-N:  Actual RF data

Firmware sends: bytes 1-N (length byte stripped)
Host receives:  bytes 1-N
```

**Fixed Length Mode:**
```
Received packet structure:
  Bytes 0-(PKTLEN-1): RF data (exactly PKTLEN bytes)

Firmware sends: all PKTLEN bytes
Host receives:  all PKTLEN bytes
```

---

## Transmit Data Flow

### Standard Transmit (≤255 bytes)

#### NIC_XMIT Command Format

```
Host → Device:
  App:     0x42 (APP_NIC)
  Cmd:     0x02 (NIC_XMIT)
  Payload:
    Bytes 0-1:  data_len (little-endian, actual RF data length)
    Bytes 2-3:  repeat (0 = once, 65535 = forever)
    Bytes 4-5:  offset (for repeat, start offset within data)
    Bytes 6+:   RF data to transmit
```

**Example: Transmit 4 bytes once**
```
42 02 0A 00 04 00 00 00 00 00 DE AD BE EF
│  │  │     │     │     │     └─ RF data
│  │  │     │     │     └─ offset = 0
│  │  │     │     └─ repeat = 0 (once)
│  │  │     └─ data_len = 4
│  │  └─ payload length = 10
│  └─ NIC_XMIT
└─ APP_NIC
```

#### Firmware Transmit Process

1. **Wait for previous TX complete** (MARCSTATE != TX)
2. **Configure repeat parameters** if specified
3. **Handle packet length:**
   - Variable mode: prepend length byte
   - Fixed mode: set PKTLEN register
4. **Copy data to TX buffer**
5. **Issue STX strobe** (RFST = 0x03)
6. **Wait for transmission complete**
7. **Send acknowledgment** to host
8. **Return to configured state** (RX/IDLE per MCSM1)

### Long Transmit (>255 bytes)

For data exceeding RF_MAX_TX_BLOCK (255 bytes), use chunked transfer:

#### Phase 1: Initialize Long Transmit

```
Host → Device:
  App:     0x42 (APP_NIC)
  Cmd:     0x0C (NIC_LONG_XMIT)
  Payload:
    Bytes 0-1:  total_len (total data length, up to 65535)
    Byte 2:     preload (number of 240-byte chunks to preload)
    Bytes 3+:   first chunks of data
```

**Response:**
```
Device → Host:
  Payload: [error_code] (0 = success)
```

#### Phase 2: Send Additional Chunks

Repeat until all data sent:

```
Host → Device:
  App:     0x42 (APP_NIC)
  Cmd:     0x0D (NIC_LONG_XMIT_MORE)
  Payload:
    Byte 0:     chunk_len (length of this chunk, or 0 to finish)
    Bytes 1+:   chunk data
```

**Response:**
```
Device → Host:
  Payload: [error_code]
    0x00 = success
    0xFE = retry (buffer not available)
```

#### Phase 3: Signal Completion

```
Host → Device:
  App:     0x42 (APP_NIC)
  Cmd:     0x0D (NIC_LONG_XMIT_MORE)
  Payload: [0x00] (chunk_len = 0 signals done)
```

### RFxmit Function Flow

```
RFxmit(data, repeat=0, offset=0):
    1. If encoder configured, encode data

    2. If len(data) > 255:
       - If repeat or offset specified, return error
       - Call RFxmitLong(data)

    3. Calculate wait time based on data length + repeats

    4. Build NIC_XMIT packet:
       - Pack: data_len (2), repeat (2), offset (2)
       - Append: RF data

    5. Send via EP5 and wait for response
```

---

## State Machine Transitions

### MARCSTATE Values

| Value | State | Description |
|-------|-------|-------------|
| 0x00  | SLEEP | Sleep mode |
| 0x01  | IDLE | Idle, ready for commands |
| 0x02  | XOFF | Crystal off |
| 0x03-0x07 | Calibration | Various calibration states |
| 0x08-0x0C | Startup | Frequency synthesizer startup |
| 0x0D  | RX | Receiving |
| 0x0E-0x0F | RX_END/RST | RX completion states |
| 0x10  | TXRX_SWITCH | Switching from TX to RX |
| 0x11  | RXFIFO_OVF | RX FIFO overflow (error) |
| 0x12  | FSTXON | Frequency synth on, ready for TX |
| 0x13  | TX | Transmitting |
| 0x14-0x15 | TX_END/SWITCH | TX completion states |
| 0x16  | TXFIFO_UNF | TX FIFO underflow (error) |

### State Transition Diagram

```
                    ┌──────────────────────────────────────┐
                    │                                      │
                    ▼                                      │
              ┌─────────┐                                  │
    ┌────────▶│  IDLE   │◀────────┐                       │
    │         └────┬────┘         │                       │
    │              │              │                       │
    │    SIDLE     │   SRX/STX    │    SIDLE             │
    │              │              │                       │
    │              ▼              │                       │
    │    ┌─────────────────┐      │                       │
    │    │  Calibration    │      │                       │
    │    │  (automatic)    │      │                       │
    │    └────────┬────────┘      │                       │
    │             │               │                       │
    │             ▼               │                       │
    │    ┌────────┴────────┐      │                       │
    │    │                 │      │                       │
    │    ▼                 ▼      │                       │
    │  ┌────┐           ┌────┐   │                       │
    │  │ RX │           │ TX │───┘                       │
    │  └──┬─┘           └──┬─┘     (MCSM1 TXOFF_MODE)    │
    │     │                │                              │
    │     │ Packet         │ Packet                       │
    │     │ received       │ sent                         │
    │     │                │                              │
    │     ▼                ▼                              │
    │  ┌──────────────────────┐                          │
    │  │  MCSM1 determines    │──────────────────────────┘
    │  │  next state          │
    │  └──────────────────────┘
    │
    └─── RXOFF_MODE = IDLE or TXOFF_MODE = IDLE
```

### MCSM1 Configuration

```
MCSM1 = (CCA_MODE << 4) | (RXOFF_MODE << 2) | TXOFF_MODE

Common configurations:
  0x0F = Continuous RX (RX→RX, TX→RX)
  0x00 = Return to IDLE (RX→IDLE, TX→IDLE)
  0x0C = TX after RX, then RX
  0x30 = RSSI-based CCA, return to IDLE
```

---

## Error Handling

### Receive Errors

| Error | MARCSTATE | Cause | Recovery |
|-------|-----------|-------|----------|
| RX Overflow | 0x11 | Buffer full, data lost | Flush RX, restart RX mode |
| Timeout | - | No packet received | Retry or continue polling |
| CRC Error | - | Corrupted packet | Automatic discard (if CRC enabled) |

**RX Overflow Recovery:**
```
1. Issue SIDLE strobe
2. Issue SFRX strobe (flush RX FIFO) - if available
3. Re-enter RX mode
```

### Transmit Errors

| Error | Code | Cause | Recovery |
|-------|------|-------|----------|
| TX Underflow | 0x11 (RC_TX_ERROR) | Data not supplied fast enough | Retry transmission |
| Buffer Full | 0xFE | Device buffer busy | Retry after delay |
| Size Exceeded | 0xFF | Data too large | Use long transmit |
| CCA Fail | 0xEC | Channel busy | Retry later |

**TX Underflow Recovery:**
```
1. Issue SIDLE strobe
2. Issue SFTX strobe (flush TX FIFO) - if available
3. Retry transmission
```

### USB Errors

| Error | Cause | Recovery |
|-------|-------|----------|
| Timeout | Device not responding | Retry, check connection |
| Stall | Protocol error | Clear stall, reset endpoint |
| Disconnect | USB disconnected | Re-enumerate, reconnect |

---

## Code Examples

### Go: Basic Receive Loop

```go
func receivePackets(device *yardstick.Device, timeout time.Duration) error {
    // Set radio to RX mode
    if err := device.SetModeRX(); err != nil {
        return fmt.Errorf("failed to set RX mode: %w", err)
    }

    for {
        // Wait for packet (blocking with timeout)
        data, err := device.Recv(yardstick.AppNIC, yardstick.NICRecv, timeout)
        if err != nil {
            if errors.Is(err, yardstick.ErrTimeout) {
                continue // No packet, keep waiting
            }
            return fmt.Errorf("receive error: %w", err)
        }

        // Process received data
        fmt.Printf("Received %d bytes: %x\n", len(data), data)
    }
}
```

### Go: Basic Transmit

```go
func transmitPacket(device *yardstick.Device, data []byte) error {
    if len(data) > 255 {
        return fmt.Errorf("data too large for standard transmit")
    }

    // Build NIC_XMIT payload
    payload := make([]byte, 6+len(data))
    binary.LittleEndian.PutUint16(payload[0:2], uint16(len(data))) // data_len
    binary.LittleEndian.PutUint16(payload[2:4], 0)                  // repeat
    binary.LittleEndian.PutUint16(payload[4:6], 0)                  // offset
    copy(payload[6:], data)

    // Send and wait for response
    _, err := device.Send(yardstick.AppNIC, yardstick.NICXmit, payload, 10*time.Second)
    return err
}
```

### Go: Configure for 433 MHz ASK

```go
func configure433ASK(device *yardstick.Device) error {
    // Ensure IDLE state
    if err := registers.SetIDLE(device); err != nil {
        return err
    }

    // Set frequency to 433.92 MHz
    // FREQ = 433920000 * 65536 / 24000000 = 0x10B1A9
    device.Poke(registers.RegFREQ2, []byte{0x10})
    device.Poke(registers.RegFREQ1, []byte{0xB1})
    device.Poke(registers.RegFREQ0, []byte{0xA9})

    // Set ASK/OOK modulation, no sync
    device.Poke(registers.RegMDMCFG2, []byte{0x30}) // ASK, no sync

    // Fixed packet length, no CRC
    device.Poke(registers.RegPKTCTRL0, []byte{0x00})
    device.Poke(registers.RegPKTLEN, []byte{0x20}) // 32 bytes

    // PA table for ASK: index 0 = off, index 1 = on
    device.Poke(registers.RegPA_TABLE0, []byte{0x00}) // Off state
    device.Poke(registers.RegPA_TABLE1, []byte{0xC0}) // On state (+10 dBm)

    // FREND0 to use PA_TABLE1 for high state
    device.Poke(registers.RegFREND0, []byte{0x11}) // PA_POWER = 1

    return nil
}
```

### Go: Threaded Receive Architecture

```go
type RadioReceiver struct {
    device    *yardstick.Device
    recvQueue chan []byte
    stopChan  chan struct{}
}

func (r *RadioReceiver) Start() {
    go r.recvLoop()
}

func (r *RadioReceiver) recvLoop() {
    for {
        select {
        case <-r.stopChan:
            return
        default:
            data, err := r.device.Recv(yardstick.AppNIC, yardstick.NICRecv, 100*time.Millisecond)
            if err == nil {
                select {
                case r.recvQueue <- data:
                default:
                    // Queue full, drop packet
                }
            }
        }
    }
}

func (r *RadioReceiver) Receive(timeout time.Duration) ([]byte, error) {
    select {
    case data := <-r.recvQueue:
        return data, nil
    case <-time.After(timeout):
        return nil, ErrTimeout
    }
}
```

---

## References

- [docs/rfcat-packet-format.md](rfcat-packet-format.md) - Complete USB protocol specification
- [docs/configuration.md](configuration.md) - Register configuration details
- [docs/defaults-in-rfcat.md](defaults-in-rfcat.md) - Default register values
- [docs/cc1110-cc1111.md](cc1110-cc1111.md) - CC1111 datasheet reference
- RFCat source: `/external/rfcat/rflib/chipcon_nic.py`
- Firmware source: `/external/rfcat/firmware/cc1111rf.c`
