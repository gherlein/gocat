# RFCat Default Radio Configuration

This document details the default register values used by the RFCat firmware
and Python library for CC1111-based devices (including YardStick One).

## Overview

RFCat initializes the CC1111 radio registers in firmware during `appInitRf()`.
Different default values are used depending on the radio region:

- **US Region** (default): 902 MHz, ISM band
- **EU Region** (`RADIO_EU`): 868 MHz, EU ISM band
- **CC2511**: 2.4 GHz settings (different chip)

This document focuses on the **US Region defaults** as used by YardStick One.

## Source Files

The defaults are defined in:
- **Firmware**: `/external/rfcat/firmware/application.c` - `appInitRf()` function
- **Python**: `/external/rfcat/rflib/const.py` - `FAKE_MEM_DF00` byte array

---

## Default Register Values (US Region / YardStick One)

### Sync Word Registers

| Register | Address | Default | Hex  | Description |
|----------|---------|---------|------|-------------|
| SYNC1    | 0xDF00  | 12      | 0x0C | Sync word high byte |
| SYNC0    | 0xDF01  | 78      | 0x4E | Sync word low byte |

**Default Sync Word**: `0x0C4E`

---

### Packet Control Registers

| Register | Address | Default | Hex  | Description |
|----------|---------|---------|------|-------------|
| PKTLEN   | 0xDF02  | 255     | 0xFF | Maximum packet length |
| PKTCTRL1 | 0xDF03  | 64      | 0x40 | PQT threshold = 2, no address check, no status append |
| PKTCTRL0 | 0xDF04  | 0       | 0x00 | Fixed length, no CRC, no whitening |
| ADDR     | 0xDF05  | 0       | 0x00 | Device address (not used) |
| CHANNR   | 0xDF06  | 0       | 0x00 | Channel number |

**Note**: `PKTCTRL0 = 0x00` means Fixed-Length packet mode. The firmware
initially sets `0x01` (Variable Length) but this is the post-configuration
state captured in `FAKE_MEM_DF00`.

#### PKTCTRL1 Breakdown (0x40)
- Bits 7:5 (PQT): `010` = Preamble Quality Threshold of 2
- Bit 3 (CRC_AUTOFLUSH): 0 = Disabled
- Bit 2 (APPEND_STATUS): 0 = Disabled
- Bits 1:0 (ADR_CHK): `00` = No address check

#### PKTCTRL0 Breakdown (0x00)
- Bit 6 (WHITE_DATA): 0 = Disabled
- Bits 5:4 (PKT_FORMAT): `00` = Normal mode
- Bit 2 (CRC_EN): 0 = Disabled
- Bits 1:0 (LENGTH_CONFIG): `00` = Fixed packet length

---

### Frequency Synthesizer Registers

| Register | Address | Default | Hex  | Description |
|----------|---------|---------|------|-------------|
| FSCTRL1  | 0xDF07  | 12      | 0x0C | IF frequency |
| FSCTRL0  | 0xDF08  | 0       | 0x00 | Frequency offset |

**Intermediate Frequency**: ~381 kHz at 24 MHz crystal
```
IF = (FSCTRL1 × f_XOSC) / 2^10
IF = (12 × 24,000,000) / 1024 = 281,250 Hz
```

---

### Frequency Control Registers

| Register | Address | Default | Hex  | Description |
|----------|---------|---------|------|-------------|
| FREQ2    | 0xDF09  | 37      | 0x25 | Frequency high byte |
| FREQ1    | 0xDF0A  | 149     | 0x95 | Frequency middle byte |
| FREQ0    | 0xDF0B  | 85      | 0x55 | Frequency low byte |

**Default Frequency**: ~902.299 MHz
```
FREQ_REG = (0x25 << 16) | (0x95 << 8) | 0x55 = 0x259555 = 2,462,037
f_carrier = (2,462,037 × 24,000,000) / 2^16 = 902,299,316 Hz ≈ 902.3 MHz
```

---

### Modem Configuration Registers

| Register | Address | Default | Hex  | Description |
|----------|---------|---------|------|-------------|
| MDMCFG4  | 0xDF0C  | 202     | 0xCA | Channel bandwidth / Data rate exponent |
| MDMCFG3  | 0xDF0D  | 163     | 0xA3 | Data rate mantissa |
| MDMCFG2  | 0xDF0E  | 1       | 0x01 | Modulation format / Sync mode |
| MDMCFG1  | 0xDF0F  | 35      | 0x23 | Preamble / Channel spacing exponent |
| MDMCFG0  | 0xDF10  | 17      | 0x11 | Channel spacing mantissa |

#### MDMCFG4 Breakdown (0xCA)
- Bits 7:6 (CHANBW_E): `11` = 3
- Bits 5:4 (CHANBW_M): `00` = 0
- Bits 3:0 (DRATE_E): `1010` = 10

**Channel Bandwidth**: ~58.0 kHz
```
BW = f_XOSC / (8 × (4 + CHANBW_M) × 2^CHANBW_E)
BW = 24,000,000 / (8 × 4 × 8) = 93,750 Hz
```

#### MDMCFG3 Breakdown (0xA3)
- DRATE_M = 163

**Data Rate**: ~38.4 kBaud
```
R_DATA = ((256 + 163) × 2^10 × 24,000,000) / 2^28
R_DATA ≈ 38,383 baud
```

#### MDMCFG2 Breakdown (0x01)
- Bit 7 (DEM_DCFILT_OFF): 0 = DC blocking filter enabled
- Bits 6:4 (MOD_FORMAT): `000` = 2-FSK
- Bit 3 (MANCHESTER_EN): 0 = Disabled
- Bits 2:0 (SYNC_MODE): `001` = 15/16 sync word bits detected

**Modulation**: 2-FSK with 15/16 sync word detection

#### MDMCFG1 Breakdown (0x23)
- Bit 7: 0 (unused)
- Bits 6:4 (NUM_PREAMBLE): `010` = 4 preamble bytes
- Bits 3:2: 00 (unused)
- Bits 1:0 (CHANSPC_E): `11` = 3

#### MDMCFG0 Breakdown (0x11)
- CHANSPC_M = 17

**Channel Spacing**: ~99.9 kHz
```
CHANSPC = (f_XOSC / 2^18) × (256 + CHANSPC_M) × 2^CHANSPC_E
CHANSPC = (24,000,000 / 262,144) × 273 × 8 = 199,951 Hz ≈ 200 kHz
```

---

### Deviation Register

| Register | Address | Default | Hex  | Description |
|----------|---------|---------|------|-------------|
| DEVIATN  | 0xDF11  | 54      | 0x36 | FSK deviation |

#### DEVIATN Breakdown (0x36)
- Bits 6:4 (DEVIATION_E): `011` = 3
- Bits 2:0 (DEVIATION_M): `110` = 6

**Frequency Deviation**: ~20.5 kHz
```
DEVIATION = (8 + 6) × 2^3 × 24,000,000 / 2^17
DEVIATION = 14 × 8 × 183.1 = 20,508 Hz ≈ 20.5 kHz
```

---

### Main Radio Control State Machine (MCSM)

| Register | Address | Default | Hex  | Description |
|----------|---------|---------|------|-------------|
| MCSM2    | 0xDF12  | 7       | 0x07 | RX timeout |
| MCSM1    | 0xDF13  | 15      | 0x0F | CCA mode / RX/TX actions |
| MCSM0    | 0xDF14  | 24      | 0x18 | Main state machine config |

#### MCSM2 Breakdown (0x07)
- Bit 4 (RX_TIME_RSSI): 0 = Don't check RSSI for RX timeout
- Bit 3 (RX_TIME_QUAL): 0 = Don't check sync word for RX timeout
- Bits 2:0 (RX_TIME): `111` = RX timeout disabled

#### MCSM1 Breakdown (0x0F)
- Bits 5:4 (CCA_MODE): `00` = Always clear (no CCA)
- Bits 3:2 (RXOFF_MODE): `11` = Stay in RX after packet
- Bits 1:0 (TXOFF_MODE): `11` = Stay in RX after TX

**Note**: The firmware initially sets MCSM1 to 0x3F for CCA with RSSI
threshold, but the captured default shows 0x0F.

#### MCSM0 Breakdown (0x18)
- Bits 5:4 (FS_AUTOCAL): `01` = Auto-cal from IDLE to RX/TX
- Bits 3:2 (PO_TIMEOUT): `10` = Approximately 149 µs timeout
- Bits 1:0: `00` (reserved)

---

### Frequency Offset/AFC Configuration

| Register | Address | Default | Hex  | Description |
|----------|---------|---------|------|-------------|
| FOCCFG   | 0xDF15  | 23      | 0x17 | Frequency offset compensation |
| BSCFG    | 0xDF16  | 108     | 0x6C | Bit synchronization |

#### FOCCFG Breakdown (0x17)
- Bit 5 (FOC_BS_CS_GATE): 0 = FOC/BS independent of CS
- Bits 4:3 (FOC_PRE_K): `10` = 3K
- Bit 2 (FOC_POST_K): 1 = K/2
- Bits 1:0 (FOC_LIMIT): `11` = ±BW_CHANNEL/2

#### BSCFG Breakdown (0x6C)
- Bits 7:6 (BS_PRE_KI): `01` = 2KI
- Bits 5:4 (BS_PRE_KP): `10` = 3KP
- Bit 3 (BS_POST_KI): 1 = KI/2
- Bit 2 (BS_POST_KP): 1 = KP
- Bits 1:0 (BS_LIMIT): `00` = No data rate offset compensation

---

### AGC Control Registers

| Register | Address | Default | Hex  | Description |
|----------|---------|---------|------|-------------|
| AGCCTRL2 | 0xDF17  | 3       | 0x03 | AGC control |
| AGCCTRL1 | 0xDF18  | 64      | 0x40 | AGC control |
| AGCCTRL0 | 0xDF19  | 145     | 0x91 | AGC control |

#### AGCCTRL2 Breakdown (0x03)
- Bits 7:6 (MAX_DVGA_GAIN): `00` = Maximum DVGA gain
- Bits 5:3 (MAX_LNA_GAIN): `000` = Maximum LNA gain
- Bits 2:0 (MAGN_TARGET): `011` = 33 dB target amplitude

#### AGCCTRL1 Breakdown (0x40)
- Bit 6 (AGC_LNA_PRIORITY): 1 = LNA2 gain decreased first
- Bits 5:4 (CARRIER_SENSE_REL_THR): `00` = Relative CS threshold disabled
- Bits 3:0 (CARRIER_SENSE_ABS_THR): `0000` = 0 dB above MAGN_TARGET

#### AGCCTRL0 Breakdown (0x91)
- Bits 7:6 (HYST_LEVEL): `10` = Medium hysteresis
- Bits 5:4 (WAIT_TIME): `01` = 16 samples
- Bits 3:2 (AGC_FREEZE): `00` = Normal operation
- Bits 1:0 (FILTER_LENGTH): `01` = 16 samples for channel filter

---

### Front End Configuration

| Register | Address | Default | Hex  | Description |
|----------|---------|---------|------|-------------|
| FREND1   | 0xDF1A  | 182     | 0xB6 | Front end RX config |
| FREND0   | 0xDF1B  | 16      | 0x10 | Front end TX config |

#### FREND1 Breakdown (0xB6)
- Bits 7:6 (LNA_CURRENT): `10` = Nominal
- Bits 5:4 (LNA2MIX_CURRENT): `11` = Nominal
- Bits 3:2 (LODIV_BUF_CURRENT_RX): `01` = Nominal
- Bits 1:0 (MIX_CURRENT): `10` = Nominal

#### FREND0 Breakdown (0x10)
- Bits 5:4 (LODIV_BUF_CURRENT_TX): `01` = Nominal
- Bits 2:0 (PA_POWER): `000` = PA_TABLE[0] used

---

### Frequency Synthesizer Calibration

| Register | Address | Default | Hex  | Description |
|----------|---------|---------|------|-------------|
| FSCAL3   | 0xDF1C  | 234     | 0xEA | FS calibration |
| FSCAL2   | 0xDF1D  | 42      | 0x2A | FS calibration (VCO high) |
| FSCAL1   | 0xDF1E  | 0       | 0x00 | FS calibration |
| FSCAL0   | 0xDF1F  | 31      | 0x1F | FS calibration |

**Note**: FSCAL2 value of 0x2A selects the high VCO range, appropriate for
frequencies above the VCO mid-point (~848 MHz for 900 MHz band).

---

### Test Registers

| Register | Address | Default | Hex  | Description |
|----------|---------|---------|------|-------------|
| TEST2    | 0xDF23  | 136     | 0x88 | Test settings |
| TEST1    | 0xDF24  | 49      | 0x31 | Test settings |
| TEST0    | 0xDF25  | 9       | 0x09 | Test settings |

These values are optimized for low data rates with increased sensitivity.

---

### Power Amplifier Table

| Register   | Address | Default | Hex  | Description |
|------------|---------|---------|------|-------------|
| PA_TABLE7  | 0xDF27  | 0       | 0x00 | PA setting 7 |
| PA_TABLE6  | 0xDF28  | 0       | 0x00 | PA setting 6 |
| PA_TABLE5  | 0xDF29  | 0       | 0x00 | PA setting 5 |
| PA_TABLE4  | 0xDF2A  | 0       | 0x00 | PA setting 4 |
| PA_TABLE3  | 0xDF2B  | 0       | 0x00 | PA setting 3 |
| PA_TABLE2  | 0xDF2C  | 0       | 0x00 | PA setting 2 |
| PA_TABLE1  | 0xDF2D  | 0       | 0x00 | PA setting 1 |
| PA_TABLE0  | 0xDF2E  | 192     | 0xC0 | PA setting 0 (active) |

**PA_TABLE0 = 0xC0**: Approximately +10 dBm output power at 900 MHz.

For ASK/OOK modulation, PA_TABLE0 and PA_TABLE1 are used for low/high states:
- PA_TABLE0 = 0x00 (off)
- PA_TABLE1 = power level (e.g., 0xC0)

---

### GPIO Configuration

| Register | Address | Default | Hex  | Description |
|----------|---------|---------|------|-------------|
| IOCFG2   | 0xDF2F  | 0       | 0x00 | GDO2 pin config |
| IOCFG1   | 0xDF30  | 0       | 0x00 | GDO1 pin config |
| IOCFG0   | 0xDF31  | 0       | 0x00 | GDO0 pin config |

All GDO pins configured for output 0 (constant low).

---

## Summary of Default Configuration

| Parameter | Value |
|-----------|-------|
| **Frequency** | 902.3 MHz |
| **Modulation** | 2-FSK |
| **Data Rate** | ~38.4 kBaud |
| **Deviation** | ~20.5 kHz |
| **Channel Bandwidth** | ~58 kHz |
| **Channel Spacing** | ~200 kHz |
| **Sync Word** | 0x0C4E |
| **Sync Mode** | 15/16 bits |
| **Preamble** | 4 bytes |
| **Packet Length** | Fixed, 255 bytes max |
| **CRC** | Disabled |
| **Data Whitening** | Disabled |
| **Manchester** | Disabled |
| **TX Power** | ~+10 dBm |
| **RX after TX** | Stay in RX |

---

## Raw Byte Dump (FAKE_MEM_DF00)

From `const.py`, the complete 0xDF00-0xDF3D memory region as a Python bytes literal:

```python
FAKE_MEM_DF00 = b'\x0cN\xff@\x00\x00\x00\x0c\x00%\x95U\xca\xa3\x01#\x116\x07\x0f\x18\x17l\x03@\x91\xb6\x10\xef*+\x1fY??\x881\t\x00\x00\x00\x00\x00\x00\x00\x00\xc0\x00\x00\x00\x00\x00\x00\x00\x11\x03\x12\x80\xaa\r\x90\xfd'
```

Decoded register-by-register (offsets 0x00-0x3D):

| Offset | Register | Value | Hex |
|--------|----------|-------|-----|
| 0x00 | SYNC1 | 12 | 0x0C |
| 0x01 | SYNC0 | 78 | 0x4E |
| 0x02 | PKTLEN | 255 | 0xFF |
| 0x03 | PKTCTRL1 | 64 | 0x40 |
| 0x04 | PKTCTRL0 | 0 | 0x00 |
| 0x05 | ADDR | 0 | 0x00 |
| 0x06 | CHANNR | 0 | 0x00 |
| 0x07 | FSCTRL1 | 12 | 0x0C |
| 0x08 | FSCTRL0 | 0 | 0x00 |
| 0x09 | FREQ2 | 37 | 0x25 |
| 0x0A | FREQ1 | 149 | 0x95 |
| 0x0B | FREQ0 | 85 | 0x55 |
| 0x0C | MDMCFG4 | 202 | 0xCA |
| 0x0D | MDMCFG3 | 163 | 0xA3 |
| 0x0E | MDMCFG2 | 1 | 0x01 |
| 0x0F | MDMCFG1 | 35 | 0x23 |
| 0x10 | MDMCFG0 | 17 | 0x11 |
| 0x11 | DEVIATN | 54 | 0x36 |
| 0x12 | MCSM2 | 7 | 0x07 |
| 0x13 | MCSM1 | 15 | 0x0F |
| 0x14 | MCSM0 | 24 | 0x18 |
| 0x15 | FOCCFG | 23 | 0x17 |
| 0x16 | BSCFG | 108 | 0x6C |
| 0x17 | AGCCTRL2 | 3 | 0x03 |
| 0x18 | AGCCTRL1 | 64 | 0x40 |
| 0x19 | AGCCTRL0 | 145 | 0x91 |
| 0x1A | FREND1 | 182 | 0xB6 |
| 0x1B | FREND0 | 16 | 0x10 |
| 0x1C | FSCAL3 | 239 | 0xEF |
| 0x1D | FSCAL2 | 42 | 0x2A |
| 0x1E | FSCAL1 | 43 | 0x2B |
| 0x1F | FSCAL0 | 31 | 0x1F |
| 0x20-0x22 | (reserved) | - | - |
| 0x23 | TEST2 | 136 | 0x88 |
| 0x24 | TEST1 | 49 | 0x31 |
| 0x25 | TEST0 | 9 | 0x09 |
| 0x26 | (reserved) | - | - |
| 0x27 | PA_TABLE7 | 0 | 0x00 |
| 0x28 | PA_TABLE6 | 0 | 0x00 |
| 0x29 | PA_TABLE5 | 0 | 0x00 |
| 0x2A | PA_TABLE4 | 0 | 0x00 |
| 0x2B | PA_TABLE3 | 0 | 0x00 |
| 0x2C | PA_TABLE2 | 0 | 0x00 |
| 0x2D | PA_TABLE1 | 0 | 0x00 |
| 0x2E | PA_TABLE0 | 192 | 0xC0 |
| 0x2F | IOCFG2 | 0 | 0x00 |
| 0x30 | IOCFG1 | 0 | 0x00 |
| 0x31 | IOCFG0 | 0 | 0x00 |
| 0x32-0x35 | (reserved) | - | - |
| 0x36 | PARTNUM | 17 | 0x11 |
| 0x37 | CHIPID | 3 | 0x03 |
| 0x38 | FREQEST | 18 | 0x12 |
| 0x39 | LQI | 128 | 0x80 |
| 0x3A | RSSI | 170 | 0xAA |
| 0x3B | MARCSTATE | 13 | 0x0D |
| 0x3C | PKTSTATUS | 144 | 0x90 |
| 0x3D | VCO_VC_DAC | 253 | 0xFD |

---

## Differences: Firmware Init vs FAKE_MEM_DF00

The firmware initializes some registers differently than what appears in
`FAKE_MEM_DF00`. This is because `FAKE_MEM_DF00` represents a snapshot
after the radio has been configured and used:

| Register | Firmware Init | FAKE_MEM_DF00 | Notes |
|----------|---------------|---------------|-------|
| PKTCTRL0 | 0x01 | 0x00 | VLEN → FLEN mode switch |
| MCSM1 | 0x3F | 0x0F | CCA mode changed |
| FSCAL3 | 0xEA | 0xEF | Calibration updated |
| FSCAL1 | 0x00 | 0x2B | Calibration updated |
| MARCSTATE | (varies) | 0x0D (RX) | Current state |

---

## References

- [CC1110/CC1111 Datasheet](https://www.ti.com/lit/ds/symlink/cc1111.pdf)
- [docs/cc1110-cc1111.md](cc1110-cc1111.md)
- [docs/configuration.md](configuration.md)
- [docs/rfcat-packet-format.md](rfcat-packet-format.md)
