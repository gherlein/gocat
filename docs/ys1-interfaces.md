# YardStick One (CC1111) Configuration Guide - AI Reference

## Device Overview

The CC1111 is a Texas Instruments System-on-Chip combining:
- Enhanced 8051 MCU (24/26 MHz crystal)
- Sub-1 GHz RF transceiver (300-348 MHz, 391-464 MHz, 782-928 MHz)
- USB 2.0 Full Speed interface
- 32 KB Flash, 4 KB RAM
- Hardware AES-128 encryption
- DMA controller, timers, ADC

## Critical Configuration Requirements

### Prerequisite: Radio State Management
**CRITICAL**: Radio MUST be in IDLE state before modifying most configuration registers.

```python
# Always idle before configuration changes
self.strobeModeIDLE()  # Set RFST=0x04
# Make configuration changes
# Return to desired state
self.strobeModeRX()    # Set RFST=0x02
```

### Register Access Pattern
1. Read current radio state via MARCSTATE (0xDF3B)
2. If not IDLE (0x01), strobe IDLE via RFST (X_RFST)
3. Modify configuration registers
4. Return to previous radio state

## Register Map - Core Configuration

### Memory-Mapped Radio Registers (0xDF00-0xDF3D)

```
Base Address: 0xDF00 (radio configuration starts here)

0xDF00  SYNC1       Sync word high byte
0xDF01  SYNC0       Sync word low byte
0xDF02  PKTLEN      Packet length (fixed) or max length (variable)
0xDF03  PKTCTRL1    Packet automation control
0xDF04  PKTCTRL0    Packet automation control
0xDF05  ADDR        Device address
0xDF06  CHANNR      Channel number
0xDF07  FSCTRL1     Frequency synthesizer control
0xDF08  FSCTRL0     Frequency synthesizer control (offset)
0xDF09  FREQ2       Frequency control word, high byte
0xDF0A  FREQ1       Frequency control word, middle byte
0xDF0B  FREQ0       Frequency control word, low byte
0xDF0C  MDMCFG4     Modem configuration
0xDF0D  MDMCFG3     Modem configuration
0xDF0E  MDMCFG2     Modem configuration
0xDF0F  MDMCFG1     Modem configuration
0xDF10  MDMCFG0     Modem configuration
0xDF11  DEVIATN     Modem deviation setting
0xDF12  MCSM2       Main radio control state machine configuration
0xDF13  MCSM1       Main radio control state machine configuration
0xDF14  MCSM0       Main radio control state machine configuration
0xDF15  FOCCFG      Frequency offset compensation configuration
0xDF16  BSCFG       Bit synchronization configuration
0xDF17  AGCCTRL2    AGC control
0xDF18  AGCCTRL1    AGC control
0xDF19  AGCCTRL0    AGC control
0xDF1A  FREND1      Front end RX configuration
0xDF1B  FREND0      Front end TX configuration
0xDF1C  FSCAL3      Frequency synthesizer calibration
0xDF1D  FSCAL2      Frequency synthesizer calibration
0xDF1E  FSCAL1      Frequency synthesizer calibration
0xDF1F  FSCAL0      Frequency synthesizer calibration
0xDF27-2E PA_TABLE  Power amplifier output power settings (8 bytes)
0xDF2F  IOCFG2      GDO2 output pin configuration
0xDF30  IOCFG1      GDO1 output pin configuration
0xDF31  IOCFG0      GDO0 output pin configuration
0xDF36  PARTNUM     Chip part number (read-only)
0xDF37  CHIPID      Chip ID (read-only)
0xDF38  FREQEST     Frequency offset estimate (read-only)
0xDF39  LQI         Link Quality Indicator (read-only)
0xDF3A  RSSI        Received Signal Strength Indicator (read-only)
0xDF3B  MARCSTATE   Main radio control state machine state (read-only)
0xDF3C  PKTSTATUS   Packet status (read-only)
0xDF3D  VCO_VC_DAC  VCO DAC value (read-only)
```

### Direct Registers

```
X_RFST = 0xE1   Radio strobe commands (write-only)
    0x00  SFSTXON  - Frequency synthesizer on
    0x01  SCAL     - Calibrate frequency synthesizer
    0x02  SRX      - Enable RX
    0x03  STX      - Enable TX
    0x04  SIDLE    - Exit RX/TX, turn off frequency synthesizer
```

## Configuration Formulas

### Frequency = (register_value * crystal_mhz * 10^6) / 2^16
```python
freq_reg = int((freq_hz * 2**16) / (crystal_mhz * 1000000))
FREQ2 = (freq_reg >> 16) & 0xFF
FREQ1 = (freq_reg >> 8) & 0xFF
FREQ0 = freq_reg & 0xFF
# VCO: Below 318/424/848 MHz → FSCAL2=0x0A, Above → FSCAL2=0x2A
```

### Data Rate = (256 + DRATE_M) * 2^DRATE_E * crystal_hz / 2^28
```python
for e in range(16):
    m = int((drate * 2**28) / (2**e * crystal_hz) - 256 + 0.5)
    if m < 256: break
MDMCFG3 = m
MDMCFG4[3:0] = e
```

### Deviation = (8 + DEV_M) * 2^DEV_E * crystal_hz / 2^17
```python
for e in range(8):
    m = int((dev_hz * 2**17) / (2**e * crystal_hz) - 8 + 0.5)
    if m < 8: break
DEVIATN = (e << 4) | m
```

### Channel BW = crystal_hz / (8 * (4 + CHANBW_M) * 2^CHANBW_E)
```python
for e in range(4):
    m = int((crystal_hz / (bw * 2**e * 8)) - 4 + 0.5)
    if m < 4: break
MDMCFG4[7:4] = (e << 2) | m
# If BW > 102 kHz: FREND1=0xB6, else 0x56
# If BW > 325 kHz: TEST2=0x88, TEST1=0x31, else 0x81, 0x35
```

## Minimal Configuration Example (433 MHz ASK)

```python
# IDLE
poke(0xE1, 0x04)
# Freq
val = int(433920000 * 65536 / 24000000)
poke(0xDF09, val>>16); poke(0xDF0A, val>>8); poke(0xDF0B, val)
poke(0xDF1D, 0x0A)  # VCO
# Modulation
poke(0xDF0E, 0x30)  # ASK, no sync
# Data rate (4.8k)
poke(0xDF0D, 131); poke(0xDF0C, (peek(0xDF0C)&0xF0)|8)
# Packet
poke(0xDF02, 64); poke(0xDF04, 0x00)  # Fixed 64
# Power
poke(0xDF2E, 0); poke(0xDF2D, 0xC2); poke(0xDF1B, 0x01)
# RX
poke(0xE1, 0x02)
```

## Configuration Checklist
- [ ] FREQ2/1/0, FSCAL2 (frequency + VCO)
- [ ] MDMCFG4/3 (data rate + channel BW)
- [ ] MDMCFG2 (modulation + sync mode)
- [ ] DEVIATN (FSK deviation)
- [ ] SYNC1/0 (if sync enabled)
- [ ] PKTCTRL0/1, PKTLEN (packet format)
- [ ] PA_TABLE0/1, FREND0 (power)
- [ ] FREND1, TEST2/1 (BW-dependent settings)

## Common Patterns

**ASK/OOK (key fobs)**: 433 MHz, no sync, fixed packet, max power  
**GFSK (sensors)**: 915 MHz, 16/16 sync, variable packet, CRC, FEC  
**2FSK+Manchester (EU)**: 868 MHz, 16/16 sync, 4.8-19.2k baud
