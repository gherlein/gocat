# YardStick One Configuration File Reference

This document provides a complete reference for the JSON configuration files produced by `ys1-dump-config` and consumed by `ys1-load-config`. Understanding this configuration is essential for programming the CC1111 radio transceiver in the YardStick One.

## Table of Contents

1. [Overview](#overview)
2. [File Structure](#file-structure)
3. [Device Metadata](#device-metadata)
4. [Register Reference](#register-reference)
   - [Sync Word Registers](#sync-word-registers)
   - [Packet Control Registers](#packet-control-registers)
   - [Frequency Control Registers](#frequency-control-registers)
   - [Modem Configuration Registers](#modem-configuration-registers)
   - [Main Radio Control State Machine](#main-radio-control-state-machine)
   - [Frequency Offset and Bit Synchronization](#frequency-offset-and-bit-synchronization)
   - [AGC Control Registers](#agc-control-registers)
   - [Front End Configuration](#front-end-configuration)
   - [Frequency Synthesizer Calibration](#frequency-synthesizer-calibration)
   - [Test Registers](#test-registers)
   - [Power Amplifier Table](#power-amplifier-table)
   - [GPIO Configuration](#gpio-configuration)
   - [Status Registers (Read-Only)](#status-registers-read-only)
5. [Configuration Examples](#configuration-examples)
6. [Common Calculations](#common-calculations)

---

## Overview

The configuration file captures the complete state of a YardStick One's CC1111 radio transceiver. The CC1111 is a Texas Instruments System-on-Chip featuring:

- **MCU**: Enhanced 8051 core running at 24 MHz
- **Frequency Bands**: 300-348 MHz, 391-464 MHz, 782-928 MHz
- **Data Rates**: 1.2 to 500 kBaud
- **Modulation**: 2-FSK, GFSK, MSK, ASK/OOK, 4-FSK
- **Features**: Hardware AES-128, FEC, Manchester encoding, data whitening

The configuration represents 62 bytes of memory-mapped registers starting at address `0xDF00`.

---

## File Structure

```json
{
  "serial": "009a",
  "manufacturer": "Great Scott Gadgets",
  "product": "YARD Stick One",
  "build_type": "YARDSTICKONE r0606",
  "part_num": 17,
  "timestamp": "2025-11-27T07:12:43.07391761-06:00",
  "registers": {
    // ... radio configuration registers ...
  }
}
```

---

## Device Metadata

### serial
- **Type**: String
- **Description**: Unique serial number of the YardStick One device
- **Example**: `"009a"`
- **Notes**: Used to identify specific devices when multiple are connected; also used as the default filename

### manufacturer
- **Type**: String
- **Description**: USB manufacturer string from the device descriptor
- **Example**: `"Great Scott Gadgets"`

### product
- **Type**: String
- **Description**: USB product string from the device descriptor
- **Example**: `"YARD Stick One"`

### build_type
- **Type**: String
- **Description**: Firmware build identifier returned by the device
- **Example**: `"YARDSTICKONE r0606"`
- **Notes**: Format is `<DEVICE_TYPE> r<VERSION>`. The version indicates the RfCat firmware revision

### part_num
- **Type**: Integer (uint8)
- **Description**: CC1111 chip part number read from PARTNUM register (0xDF36)
- **Values**:
  | Value | Chip | Crystal |
  |-------|------|---------|
  | 0x01 (1) | CC1110 | 24 MHz |
  | 0x11 (17) | CC1111 | 24 MHz |
  | 0x81 (129) | CC2510 | 26 MHz |
  | 0x91 (145) | CC2511 | 26 MHz |
- **Example**: `17` (CC1111)
- **Notes**: The crystal frequency affects all frequency calculations

### timestamp
- **Type**: String (RFC 3339 format)
- **Description**: When the configuration was captured
- **Example**: `"2025-11-27T07:12:43.07391761-06:00"`

---

## Register Reference

All registers are in the `registers` object. Values are decimal integers (0-255) unless otherwise noted.

### Sync Word Registers

The sync word is a 16-bit pattern used for packet synchronization. The receiver searches for this pattern to identify the start of a packet.

#### sync1
- **Address**: 0xDF00
- **Description**: Sync word high byte
- **Example**: `12` (0x0C)
- **Calculation**: Combined with sync0 to form 16-bit sync word: `(sync1 << 8) | sync0`

#### sync0
- **Address**: 0xDF01
- **Description**: Sync word low byte
- **Example**: `78` (0x4E)
- **Notes**: In the example, sync word = 0x0C4E (3150 decimal)

---

### Packet Control Registers

These registers control packet formatting, length handling, address filtering, and channel selection.

#### pktlen
- **Address**: 0xDF02
- **Description**: Packet length configuration
- **Range**: 1-255
- **Example**: `255`
- **Behavior**:
  - **Fixed length mode**: Exact packet length in bytes
  - **Variable length mode**: Maximum allowed packet length
  - **Infinite length mode**: Not used

#### pktctrl1
- **Address**: 0xDF03
- **Description**: Packet automation control 1
- **Example**: `64` (0x40)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 7:5 | PQT | Preamble quality threshold (0-7). Sync word accepted if PQT quality is above threshold |
  | 4 | - | Reserved |
  | 3 | CRC_AUTOFLUSH | Auto-flush RX FIFO on CRC error (1=enable) |
  | 2 | APPEND_STATUS | Append 2 status bytes (RSSI, LQI) to payload (1=enable) |
  | 1:0 | ADR_CHK | Address check mode |

  **ADR_CHK Values**:
  | Value | Mode |
  |-------|------|
  | 0 | No address check |
  | 1 | Address check, no broadcast |
  | 2 | Address check, 0x00 broadcast |
  | 3 | Address check, 0x00 and 0xFF broadcast |

- **Example Decode**: `64 = 0x40 = 0b01000000` → PQT=2, no autoflush, no status append, no address check

#### pktctrl0
- **Address**: 0xDF04
- **Description**: Packet automation control 0
- **Example**: `0` (0x00)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 7 | - | Reserved |
  | 6 | WHITE_DATA | Data whitening enable (1=enable) |
  | 5:4 | PKT_FORMAT | Packet format |
  | 3 | - | Reserved |
  | 2 | CRC_EN | CRC calculation enable (1=enable) |
  | 1:0 | LENGTH_CONFIG | Packet length mode |

  **PKT_FORMAT Values**:
  | Value | Mode |
  |-------|------|
  | 0 | Normal mode (FIFOs) |
  | 1 | Synchronous serial mode |
  | 2 | Random TX mode |
  | 3 | Asynchronous serial mode |

  **LENGTH_CONFIG Values**:
  | Value | Mode | Description |
  |-------|------|-------------|
  | 0 | Fixed | PKTLEN specifies exact length |
  | 1 | Variable | First byte after sync word is length |
  | 2 | Infinite | Packet length manually controlled |
  | 3 | Reserved | - |

- **Example Decode**: `0 = 0x00` → No whitening, normal mode, no CRC, fixed length

#### addr
- **Address**: 0xDF05
- **Description**: Device address for packet filtering
- **Range**: 0-255
- **Example**: `0`
- **Notes**: Used when address checking is enabled in PKTCTRL1

#### channr
- **Address**: 0xDF06
- **Description**: Channel number multiplier
- **Range**: 0-255
- **Example**: `0`
- **Calculation**: Actual frequency = base_freq + (channr × channel_spacing)
- **Notes**: Channel spacing is configured in MDMCFG1/MDMCFG0

---

### Frequency Control Registers

These registers set the carrier frequency and frequency synthesizer parameters.

#### fsctrl1
- **Address**: 0xDF07
- **Description**: Frequency synthesizer control - IF frequency
- **Example**: `12` (0x0C)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 4:0 | FREQ_IF | IF frequency setting |

- **Calculation**: `IF_freq = (FREQ_IF × crystal_freq) / 2^10`
- **Example**: With crystal=24MHz, FREQ_IF=12: IF = (12 × 24MHz) / 1024 = 281.25 kHz

#### fsctrl0
- **Address**: 0xDF08
- **Description**: Frequency synthesizer control - frequency offset
- **Example**: `0`
- **Notes**: Signed 8-bit offset added to frequency word. Used for fine frequency adjustment.

#### freq2
- **Address**: 0xDF09
- **Description**: Frequency control word, high byte (bits 23:16)
- **Example**: `37` (0x25)

#### freq1
- **Address**: 0xDF0A
- **Description**: Frequency control word, middle byte (bits 15:8)
- **Example**: `149` (0x95)

#### freq0
- **Address**: 0xDF0B
- **Description**: Frequency control word, low byte (bits 7:0)
- **Example**: `85` (0x55)

**Frequency Calculation**:
```
FREQ_REG = (freq2 << 16) | (freq1 << 8) | freq0
carrier_freq = (FREQ_REG × crystal_freq) / 2^16
```

**Example Calculation**:
```
FREQ_REG = (37 << 16) | (149 << 8) | 85 = 0x259555 = 2462037
crystal_freq = 24 MHz
carrier_freq = (2462037 × 24,000,000) / 65536 = 901,499,938 Hz ≈ 901.5 MHz
```

**Reverse Calculation** (setting frequency):
```
FREQ_REG = (desired_freq × 65536) / crystal_freq
```

---

### Modem Configuration Registers

These registers control data rate, modulation format, channel bandwidth, and sync word behavior.

#### mdmcfg4
- **Address**: 0xDF0C
- **Description**: Modem configuration 4 - channel bandwidth and data rate exponent
- **Example**: `202` (0xCA)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 7:6 | CHANBW_E | Channel bandwidth exponent |
  | 5:4 | CHANBW_M | Channel bandwidth mantissa |
  | 3:0 | DRATE_E | Data rate exponent |

**Channel Bandwidth Calculation**:
```
BW = crystal_freq / (8 × (4 + CHANBW_M) × 2^CHANBW_E)
```

**Example**: `0xCA = 0b11001010` → CHANBW_E=3, CHANBW_M=0, DRATE_E=10
```
BW = 24MHz / (8 × 4 × 8) = 93.75 kHz
```

#### mdmcfg3
- **Address**: 0xDF0D
- **Description**: Modem configuration 3 - data rate mantissa
- **Example**: `163` (0xA3)
- **Notes**: Full 8 bits used for DRATE_M

**Data Rate Calculation**:
```
data_rate = ((256 + DRATE_M) × 2^DRATE_E × crystal_freq) / 2^28
```

**Example**: DRATE_E=10, DRATE_M=163, crystal=24MHz
```
data_rate = ((256 + 163) × 1024 × 24,000,000) / 268,435,456 = 38,383 baud ≈ 38.4 kBaud
```

#### mdmcfg2
- **Address**: 0xDF0E
- **Description**: Modem configuration 2 - modulation format and sync mode
- **Example**: `1` (0x01)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 7 | DEM_DCFILT_OFF | Disable digital DC filter (0=enable) |
  | 6:4 | MOD_FORMAT | Modulation format |
  | 3 | MANCHESTER_EN | Manchester encoding enable |
  | 2:0 | SYNC_MODE | Sync word detection mode |

  **MOD_FORMAT Values**:
  | Value | Modulation |
  |-------|------------|
  | 0 | 2-FSK |
  | 1 | GFSK |
  | 3 | ASK/OOK |
  | 4 | 4-FSK |
  | 7 | MSK |

  **SYNC_MODE Values**:
  | Value | Mode | Description |
  |-------|------|-------------|
  | 0 | SYNCM_NONE | No preamble/sync |
  | 1 | SYNCM_15_16 | 15/16 sync word bits match |
  | 2 | SYNCM_16_16 | 16/16 sync word bits match |
  | 3 | SYNCM_30_32 | 30/32 sync word bits match |
  | 4 | SYNCM_CARRIER | Carrier-sense above threshold |
  | 5 | SYNCM_CARRIER_15_16 | Carrier-sense + 15/16 |
  | 6 | SYNCM_CARRIER_16_16 | Carrier-sense + 16/16 |
  | 7 | SYNCM_CARRIER_30_32 | Carrier-sense + 30/32 |

- **Example Decode**: `1 = 0x01` → 2-FSK modulation, no Manchester, 15/16 sync match

#### mdmcfg1
- **Address**: 0xDF0F
- **Description**: Modem configuration 1 - FEC, preamble, channel spacing exponent
- **Example**: `35` (0x23)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 7 | FEC_EN | Forward Error Correction enable |
  | 6:4 | NUM_PREAMBLE | Minimum preamble bytes to transmit |
  | 3:2 | - | Reserved |
  | 1:0 | CHANSPC_E | Channel spacing exponent |

  **NUM_PREAMBLE Values**:
  | Value | Preamble Bytes |
  |-------|----------------|
  | 0 | 2 |
  | 1 | 3 |
  | 2 | 4 |
  | 3 | 6 |
  | 4 | 8 |
  | 5 | 12 |
  | 6 | 16 |
  | 7 | 24 |

- **Example Decode**: `35 = 0x23 = 0b00100011` → No FEC, 4 preamble bytes, CHANSPC_E=3

#### mdmcfg0
- **Address**: 0xDF10
- **Description**: Modem configuration 0 - channel spacing mantissa
- **Example**: `17` (0x11)
- **Notes**: Full 8 bits used for CHANSPC_M

**Channel Spacing Calculation**:
```
channel_spacing = ((256 + CHANSPC_M) × 2^CHANSPC_E × crystal_freq) / 2^18
```

#### deviatn
- **Address**: 0xDF11
- **Description**: Modem deviation setting (FSK only)
- **Example**: `54` (0x36)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 6:4 | DEVIATION_E | Deviation exponent |
  | 2:0 | DEVIATION_M | Deviation mantissa |

**Deviation Calculation** (for 2-FSK/GFSK):
```
deviation = ((8 + DEVIATION_M) × 2^DEVIATION_E × crystal_freq) / 2^17
```

**Example**: `0x36 = 0b00110110` → DEVIATION_E=3, DEVIATION_M=6
```
deviation = ((8 + 6) × 8 × 24,000,000) / 131,072 = 20,508 Hz ≈ 20.5 kHz
```

---

### Main Radio Control State Machine

These registers control automatic radio state transitions.

#### mcsm2
- **Address**: 0xDF12
- **Description**: Main Radio Control State Machine 2
- **Example**: `7` (0x07)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 4 | RX_TIME_RSSI | Terminate RX on RSSI measurement |
  | 3 | RX_TIME_QUAL | Check sync word quality |
  | 2:0 | RX_TIME | RX timeout (WOR) |

#### mcsm1
- **Address**: 0xDF13
- **Description**: Main Radio Control State Machine 1
- **Example**: `15` (0x0F)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 5:4 | CCA_MODE | Clear Channel Assessment mode |
  | 3:2 | RXOFF_MODE | State after RX complete |
  | 1:0 | TXOFF_MODE | State after TX complete |

  **CCA_MODE Values**:
  | Value | Mode |
  |-------|------|
  | 0 | Always clear |
  | 1 | RSSI below threshold |
  | 2 | Receiving packet |
  | 3 | RSSI below OR receiving |

  **RXOFF_MODE / TXOFF_MODE Values**:
  | Value | Next State |
  |-------|------------|
  | 0 | IDLE |
  | 1 | FSTXON |
  | 2 | TX (for RXOFF) / Stay in TX (for TXOFF) |
  | 3 | RX |

- **Example Decode**: `15 = 0x0F` → CCA=always clear, after RX→RX, after TX→RX

#### mcsm0
- **Address**: 0xDF14
- **Description**: Main Radio Control State Machine 0
- **Example**: `24` (0x18)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 5:4 | FS_AUTOCAL | Auto calibration mode |
  | 3:2 | PO_TIMEOUT | Power-on timeout |
  | 1 | PIN_CTRL_EN | Pin radio control enable |
  | 0 | XOSC_FORCE_ON | Force oscillator on in SLEEP |

  **FS_AUTOCAL Values**:
  | Value | Calibration |
  |-------|-------------|
  | 0 | Never |
  | 1 | When going from IDLE to RX/TX |
  | 2 | When going from RX/TX to IDLE |
  | 3 | Every 4th time going to RX/TX |

- **Example Decode**: `24 = 0x18 = 0b00011000` → Auto-cal from IDLE→RX/TX, timeout=2, no pin ctrl

---

### Frequency Offset and Bit Synchronization

#### foccfg
- **Address**: 0xDF15
- **Description**: Frequency Offset Compensation Configuration
- **Example**: `23` (0x17)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 5 | FOC_BS_CS_GATE | Freeze offset compensation until CS |
  | 4:3 | FOC_PRE_K | Pre-demod compensation gain |
  | 2 | FOC_POST_K | Post-demod compensation gain |
  | 1:0 | FOC_LIMIT | Max frequency offset compensation |

#### bscfg
- **Address**: 0xDF16
- **Description**: Bit Synchronization Configuration
- **Example**: `108` (0x6C)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 7:6 | BS_PRE_KI | Pre-sync clock recovery KI |
  | 5:4 | BS_PRE_KP | Pre-sync clock recovery KP |
  | 3 | BS_POST_KI | Post-sync clock recovery KI |
  | 2 | BS_POST_KP | Post-sync clock recovery KP |
  | 1:0 | BS_LIMIT | Data rate offset compensation limit |

---

### AGC Control Registers

Automatic Gain Control optimizes receiver sensitivity.

#### agcctrl2
- **Address**: 0xDF17
- **Description**: AGC Control 2
- **Example**: `3` (0x03)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 7:6 | MAX_DVGA_GAIN | Maximum DVGA gain |
  | 5:3 | MAX_LNA_GAIN | Maximum LNA gain |
  | 2:0 | MAGN_TARGET | Target amplitude for AGC loop |

#### agcctrl1
- **Address**: 0xDF18
- **Description**: AGC Control 1
- **Example**: `64` (0x40)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 6 | AGC_LNA_PRIORITY | LNA2 priority |
  | 5:4 | CARRIER_SENSE_REL_THR | Relative carrier sense threshold |
  | 3:0 | CARRIER_SENSE_ABS_THR | Absolute carrier sense threshold |

#### agcctrl0
- **Address**: 0xDF19
- **Description**: AGC Control 0
- **Example**: `145` (0x91)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 7:6 | HYST_LEVEL | Hysteresis level |
  | 5:4 | WAIT_TIME | Wait time before AGC update |
  | 3:2 | AGC_FREEZE | AGC freeze behavior |
  | 1:0 | FILTER_LENGTH | Averaging length for amplitude |

---

### Front End Configuration

#### frend1
- **Address**: 0xDF1A
- **Description**: Front End RX Configuration
- **Example**: `182` (0xB6)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 7:6 | LNA_CURRENT | LNA bias current |
  | 5:4 | LNA2MIX_CURRENT | LNA2 to mixer bias |
  | 3:2 | LODIV_BUF_CURRENT_RX | LO buffer current (RX) |
  | 1:0 | MIX_CURRENT | Mixer bias current |

- **Bandwidth Dependency**: For channel BW > 101.5 kHz, use `0xB6`; otherwise `0x56`

#### frend0
- **Address**: 0xDF1B
- **Description**: Front End TX Configuration
- **Example**: `16` (0x10)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 5:4 | LODIV_BUF_CURRENT_TX | LO buffer current (TX) |
  | 2:0 | PA_POWER | PA power index (selects PA_TABLE entry) |

- **Notes**: PA_POWER selects which PA_TABLE entry to use for transmit power

---

### Frequency Synthesizer Calibration

These registers store calibration values. Generally set automatically by calibration or from SmartRF Studio.

#### fscal3
- **Address**: 0xDF1C
- **Description**: Frequency Synthesizer Calibration 3
- **Example**: `234` (0xEA)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 7:6 | FSCAL3 | Charge pump current |
  | 5:4 | CHP_CURR_CAL_EN | Charge pump cal enable |
  | 3:0 | FSCAL3 | (continued) |

#### fscal2
- **Address**: 0xDF1D
- **Description**: Frequency Synthesizer Calibration 2 - VCO Selection
- **Example**: `42` (0x2A)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 5 | VCO_CORE_H_EN | High VCO enable |
  | 4:0 | FSCAL2 | VCO current calibration |

- **Critical**: VCO selection depends on frequency:
  | Frequency | FSCAL2 |
  |-----------|--------|
  | < 318 MHz | 0x0A |
  | ≥ 318 MHz & < 424 MHz | 0x0A |
  | ≥ 424 MHz & < 848 MHz | 0x2A |
  | ≥ 848 MHz | 0x2A |

#### fscal1
- **Address**: 0xDF1E
- **Description**: Frequency Synthesizer Calibration 1
- **Example**: `0` (0x00)

#### fscal0
- **Address**: 0xDF1F
- **Description**: Frequency Synthesizer Calibration 0
- **Example**: `31` (0x1F)

---

### Test Registers

Test registers for optimal radio performance. Values depend on channel bandwidth.

#### test2
- **Address**: 0xDF23
- **Description**: Test Register 2
- **Example**: `136` (0x88)
- **Bandwidth Dependency**:
  | Channel BW | TEST2 |
  |------------|-------|
  | ≤ 325 kHz | 0x81 |
  | > 325 kHz | 0x88 |

#### test1
- **Address**: 0xDF24
- **Description**: Test Register 1
- **Example**: `49` (0x31)
- **Bandwidth Dependency**:
  | Channel BW | TEST1 |
  |------------|-------|
  | ≤ 325 kHz | 0x35 |
  | > 325 kHz | 0x31 |

#### test0
- **Address**: 0xDF25
- **Description**: Test Register 0
- **Example**: `9` (0x09)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 7:2 | TEST0 | Test value |
  | 1 | VCO_SEL_CAL_EN | Enable VCO cal on every TX/RX |
  | 0 | - | Reserved |

---

### Power Amplifier Table

The PA_TABLE contains up to 8 power settings. FREND0.PA_POWER selects which entry is used.

#### pa_table
- **Address**: 0xDF27-0xDF2E (PA_TABLE7 through PA_TABLE0)
- **Type**: Array of 8 uint8 values
- **Example**: `[192, 0, 0, 0, 0, 0, 0, 0]`
- **Index Mapping**:
  | Array Index | Register | Address |
  |-------------|----------|---------|
  | 0 | PA_TABLE7 | 0xDF27 |
  | 1 | PA_TABLE6 | 0xDF28 |
  | 2 | PA_TABLE5 | 0xDF29 |
  | 3 | PA_TABLE4 | 0xDF2A |
  | 4 | PA_TABLE3 | 0xDF2B |
  | 5 | PA_TABLE2 | 0xDF2C |
  | 6 | PA_TABLE1 | 0xDF2D |
  | 7 | PA_TABLE0 | 0xDF2E |

**Power Settings** (vary by frequency band):

For 915 MHz band:
| PA_TABLE Value | Approx. Power |
|----------------|---------------|
| 0x00 | -30 dBm |
| 0x0D | -20 dBm |
| 0x34 | -10 dBm |
| 0x60 | 0 dBm |
| 0x84 | +5 dBm |
| 0xC0 | +10 dBm |
| 0xC3 | +11 dBm (max) |

- **Example**: `pa_table[0] = 192 (0xC0)` → ~+10 dBm output power
- **ASK/OOK Note**: For ASK/OOK modulation, index 0 is "off" power, index 1 is "on" power

---

### GPIO Configuration

Configure GDO0, GDO1, GDO2 pins for various signals.

#### iocfg2
- **Address**: 0xDF2F
- **Description**: GDO2 Pin Configuration
- **Example**: `0`
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 6 | GDO2_INV | Invert output |
  | 5:0 | GDO2_CFG | Signal selection |

#### iocfg1
- **Address**: 0xDF30
- **Description**: GDO1 Pin Configuration
- **Example**: `0`
- **Notes**: Same format as IOCFG2. GDO1 is shared with SO (serial out) in SPI mode.

#### iocfg0
- **Address**: 0xDF31
- **Description**: GDO0 Pin Configuration
- **Example**: `0`
- **Notes**: Same format as IOCFG2

**Common GDOx_CFG Values**:
| Value | Signal |
|-------|--------|
| 0x00 | RX FIFO threshold |
| 0x01 | RX FIFO threshold or end of packet |
| 0x02 | TX FIFO threshold |
| 0x06 | Sync word sent/received |
| 0x07 | Packet received with CRC OK |
| 0x09 | Clear channel assessment |
| 0x0E | Carrier sense |
| 0x29 | CHIP_RDYn |
| 0x2E | HW to 0 (GDO0 default) |
| 0x2F | CLK_XOSC/1 |
| 0x3F | CLK_XOSC/192 |

---

### Status Registers (Read-Only)

These registers provide real-time device status. They are captured at dump time but cannot be restored.

#### partnum
- **Address**: 0xDF36
- **Description**: Chip Part Number
- **Example**: `17` (0x11 = CC1111)
- **Values**: See [part_num](#part_num) in Device Metadata

#### chipid
- **Address**: 0xDF37
- **Description**: Chip Revision/Version
- **Example**: `4`
- **Notes**: Silicon revision number

#### freqest
- **Address**: 0xDF38
- **Description**: Frequency Offset Estimate
- **Example**: `0`
- **Notes**: Signed estimate of carrier frequency offset. Updated during RX.

#### lqi
- **Address**: 0xDF39
- **Description**: Link Quality Indicator
- **Example**: `255`
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 7 | CRC_OK | CRC check passed |
  | 6:0 | LQI | Link quality (lower = better) |

- **Notes**: Best possible LQI is 0, worst is 127

#### rssi
- **Address**: 0xDF3A
- **Description**: Received Signal Strength Indicator
- **Example**: `128` (0x80)
- **Calculation**:
```
RSSI_dBm = (RSSI_dec - 128) / 2 - RSSI_offset
```
Where RSSI_offset ≈ 74 dB for CC1111

- **Example**: `128 → (128-128)/2 - 74 = -74 dBm` (approximately noise floor)

#### marcstate
- **Address**: 0xDF3B
- **Description**: Main Radio Control State Machine State
- **Example**: `1`
- **Values**:
  | Value | State | Description |
  |-------|-------|-------------|
  | 0x00 | SLEEP | Sleep |
  | 0x01 | IDLE | Idle |
  | 0x02 | XOFF | XOSC off |
  | 0x03-0x0C | Various | Calibration/startup states |
  | 0x0D | RX | Receiving |
  | 0x0E | RX_END | RX end |
  | 0x0F | RX_RST | RX reset |
  | 0x10 | TXRX_SWITCH | Switching TX→RX |
  | 0x11 | RXFIFO_OVERFLOW | RX FIFO overflow |
  | 0x12 | FSTXON | Fast TX ready |
  | 0x13 | TX | Transmitting |
  | 0x14 | TX_END | TX end |
  | 0x15 | RXTX_SWITCH | Switching RX→TX |
  | 0x16 | TXFIFO_UNDERFLOW | TX FIFO underflow |

- **Example**: `1` = IDLE state

#### pktstatus
- **Address**: 0xDF3C
- **Description**: Packet Status
- **Example**: `144` (0x90)
- **Bit Fields**:
  | Bits | Name | Description |
  |------|------|-------------|
  | 7 | CRC_OK | Last CRC comparison matched |
  | 6 | CS | Carrier sense |
  | 5 | PQT_REACHED | Preamble quality reached |
  | 4 | CCA | Channel is clear |
  | 3 | SFD | Sync word found |
  | 2 | GDO2 | Current GDO2 value |
  | 1 | - | Reserved |
  | 0 | GDO0 | Current GDO0 value |

- **Example Decode**: `144 = 0x90 = 0b10010000` → CRC_OK=1, CS=0, PQT=0, CCA=1

#### vco_vc_dac
- **Address**: 0xDF3D
- **Description**: VCO DAC Output Value
- **Example**: `164` (0xA4)
- **Notes**: Current DAC setting for VCO. Useful for debugging frequency synthesis.

---

## Configuration Examples

### Example 1: Decode the Sample Configuration

From `etc/yardsticks/009a.json`:

| Parameter | Value | Decoded |
|-----------|-------|---------|
| Sync Word | 0x0C4E | `(12 << 8) \| 78 = 3150` |
| Frequency | 0x259555 | 901.5 MHz |
| Data Rate | E=10, M=163 | 38.4 kBaud |
| Modulation | 0x01 | 2-FSK, 15/16 sync |
| Deviation | E=3, M=6 | 20.5 kHz |
| TX Power | 0xC0 | ~+10 dBm |
| Radio State | 1 | IDLE |

### Example 2: 433 MHz ASK Configuration

For receiving key fobs:
```json
{
  "freq2": 16, "freq1": 167, "freq0": 98,  // 433.92 MHz
  "mdmcfg2": 48,                            // ASK/OOK, no sync
  "mdmcfg4": 200, "mdmcfg3": 131,           // ~4.8 kBaud
  "pktctrl0": 0, "pktlen": 64,              // Fixed 64 bytes
  "fscal2": 10                              // Low VCO
}
```

### Example 3: 915 MHz GFSK Configuration

For wireless sensors:
```json
{
  "freq2": 35, "freq1": 0, "freq0": 0,      // 915 MHz
  "mdmcfg2": 19,                            // GFSK, 30/32 sync
  "mdmcfg4": 202, "mdmcfg3": 163,           // 38.4 kBaud
  "deviatn": 54,                            // 20.5 kHz deviation
  "sync1": 211, "sync0": 145,               // Sync word 0xD391
  "pktctrl0": 5,                            // Variable length, CRC
  "fscal2": 42                              // High VCO
}
```

---

## Common Calculations

### Frequency Setting

```go
// Set carrier frequency
func setFrequency(freqHz float64, crystalMHz float64) (freq2, freq1, freq0 uint8) {
    freqReg := uint32(freqHz * 65536 / (crystalMHz * 1e6))
    return uint8(freqReg >> 16), uint8(freqReg >> 8), uint8(freqReg)
}

// Get carrier frequency
func getFrequency(freq2, freq1, freq0 uint8, crystalMHz float64) float64 {
    freqReg := uint32(freq2)<<16 | uint32(freq1)<<8 | uint32(freq0)
    return float64(freqReg) * crystalMHz * 1e6 / 65536
}
```

### Data Rate Calculation

```go
// Calculate data rate from registers
func getDataRate(mdmcfg4, mdmcfg3 uint8, crystalMHz float64) float64 {
    drateE := mdmcfg4 & 0x0F
    drateM := mdmcfg3
    return float64((256+uint32(drateM))) * math.Pow(2, float64(drateE)) *
           crystalMHz * 1e6 / math.Pow(2, 28)
}
```

### Channel Bandwidth Calculation

```go
// Calculate channel bandwidth from MDMCFG4
func getChannelBW(mdmcfg4 uint8, crystalMHz float64) float64 {
    chanbwE := (mdmcfg4 >> 6) & 0x03
    chanbwM := (mdmcfg4 >> 4) & 0x03
    return crystalMHz * 1e6 / (8 * (4 + float64(chanbwM)) * math.Pow(2, float64(chanbwE)))
}
```

### Deviation Calculation

```go
// Calculate FSK deviation from DEVIATN
func getDeviation(deviatn uint8, crystalMHz float64) float64 {
    devE := (deviatn >> 4) & 0x07
    devM := deviatn & 0x07
    return float64(8+devM) * math.Pow(2, float64(devE)) * crystalMHz * 1e6 / 131072
}
```

---

## References

- **CC1111 Datasheet**: [SWRS033H](https://www.ti.com/lit/ds/symlink/cc1111.pdf) - Texas Instruments
- **RfCat Protocol**: See `docs/rfcat-packet-format.md`
- **GoCat Design**: See `docs/DESIGN.md`
- **SmartRF Studio**: TI's official RF configuration tool
