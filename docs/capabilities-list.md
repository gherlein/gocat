# YardStick One (CC1111) Capabilities Master List

This document enumerates the complete set of configurations and capabilities of the YardStick One hardware based on the CC1111 transceiver. It serves as a reference for designing higher-level features and tools.

---

## 1. Frequency Bands

The CC1111 operates across three distinct frequency bands:

| Band | Range (Spec) | Extended Range | Common Frequencies |
|------|--------------|----------------|-------------------|
| Low | 300-348 MHz | 281-361 MHz | 315 MHz |
| Mid | 391-464 MHz | 378-481 MHz | 433.92 MHz, 418 MHz |
| High | 782-928 MHz | 749-962 MHz | 868 MHz, 915 MHz, 902-928 MHz |

**Notes:**
- Frequency resolution: ~396 Hz steps (with 24 MHz crystal)
- VCO selection required based on frequency:
  - Below 318/424/848 MHz: FSCAL2=0x0A
  - Above 318/424/848 MHz: FSCAL2=0x2A

---

## 2. Modulation Types

| Modulation | Description | Max Data Rate | Notes |
|------------|-------------|---------------|-------|
| 2-FSK | 2-level Frequency Shift Keying | 500 kBaud | Most common for digital links |
| GFSK | Gaussian FSK (filtered 2-FSK) | 500 kBaud | Reduced bandwidth, better spectral efficiency |
| ASK/OOK | On-Off Keying / Amplitude Shift Keying | 250 kBaud | Simple, common for key fobs/remotes |
| 4-FSK | 4-level FSK | 300 kBaud | 2 bits per symbol, no Manchester support |
| MSK | Minimum Shift Keying | 500 kBaud | Only for data rates >26 kBaud |

**Encoding Options (combinable with modulations):**
- Manchester encoding (not compatible with 4-FSK)
- Data whitening (PN9 sequence)

---

## 3. Data Rates

| Category | Range | Typical Uses |
|----------|-------|--------------|
| Very Low | 600-1200 baud | Long range, maximum sensitivity (-110 dBm @ 1.2k) |
| Low | 1.2-4.8 kBaud | Key fobs, simple remotes, weather stations |
| Standard | 4.8-19.2 kBaud | Legacy protocols, Manchester-encoded systems |
| Medium | 19.2-76.8 kBaud | Wireless sensors, smart home devices |
| High | 76.8-250 kBaud | High-throughput applications |
| Very High | 250-500 kBaud | Maximum throughput (limited modulation options) |

---

## 4. Channel Bandwidth

| Bandwidth | Best For | Register Notes |
|-----------|----------|----------------|
| 54-62 kHz | Low data rate signals, narrow-band | If BW ≤102 kHz: FREND1=0x56 |
| 62-102 kHz | Standard signals | If BW ≤102 kHz: FREND1=0x56 |
| 102-200 kHz | Medium data rates | If BW >102 kHz: FREND1=0xB6 |
| 200-325 kHz | High data rates | TEST2/1 settings vary |
| 325-750 kHz | Wideband signals, FHSS | If BW >325 kHz: TEST2=0x88, TEST1=0x31 |

**Guideline:** Signal should occupy ≤80% of configured channel bandwidth.

---

## 5. Transmit Power

| Frequency Band | Maximum Power | PA_TABLE Value |
|----------------|---------------|----------------|
| ≤400 MHz | +10 dBm | 0xC2 |
| 401-464 MHz | +10 dBm | 0xC0 |
| 465-849 MHz | +10 dBm | 0xC2 |
| ≥850 MHz | +10 dBm | 0xC0 |

**Range:** Configurable from approximately -30 dBm to +10 dBm.

---

## 6. Packet Configurations

### 6.1 Packet Length Modes

| Mode | Max Size | Description |
|------|----------|-------------|
| Fixed Length | 512 bytes | PKTLEN defines exact packet size |
| Variable Length | 255 bytes | First byte after sync word = length |

### 6.2 Sync Word Options

| Sync Mode | Description | Use Case |
|-----------|-------------|----------|
| SYNCM_NONE | No sync word | ASK/OOK without preamble |
| SYNCM_15_of_16 | 15 of 16 bits match | Error tolerant |
| SYNCM_16_of_16 | All 16 bits match | Standard digital links |
| SYNCM_30_of_32 | 30 of 32 bits match | Extended sync word |
| SYNCM_CARRIER | Carrier detect only | Spectrum monitoring |
| Carrier + Sync | Combined modes | High reliability links |

### 6.3 Preamble Options

Available lengths: 2, 3, 4, 6, 8, 12, 16, or 24 bytes

### 6.4 Error Handling

| Feature | Description |
|---------|-------------|
| CRC-16 | Hardware CRC checking |
| FEC | Convolutional Forward Error Correction |
| Address Filtering | Hardware address matching |
| PQT | Preamble Quality Threshold (0-7) |

---

## 7. Advanced Features

### 7.1 Hardware AES-128 Encryption

| Mode | Description |
|------|-------------|
| ECB | Electronic Codebook |
| CBC | Cipher Block Chaining |
| CTR | Counter Mode |
| CFB | Cipher Feedback |
| OFB | Output Feedback |
| CBCMAC | CBC-MAC Authentication |

### 7.2 Frequency Hopping Spread Spectrum (FHSS)

- Programmable channel lists
- Configurable dwell time per channel
- Sync master/slave modes
- MAC timing threshold control

### 7.3 Clear Channel Assessment (CCA)

| Mode | Behavior |
|------|----------|
| 0 | Always transmit |
| 1 | TX only if RSSI below threshold |
| 2 | TX unless currently receiving |
| 3 | Combined RSSI + RX check |

### 7.4 Signal Quality Metrics

- RSSI (Received Signal Strength Indicator)
- LQI (Link Quality Indicator)
- Frequency offset estimation

---

## 8. Standard Configuration Profiles

The following profiles represent the key operational modes for each frequency band.

### 8.1 315 MHz Band Configurations

| Profile | Modulation | Data Rate | Bandwidth | Sync | Use Case |
|---------|------------|-----------|-----------|------|----------|
| 315-OOK-Low | ASK/OOK | 1-4.8 kBaud | 58 kHz | None | Key fobs, garage doors |
| 315-OOK-Fast | ASK/OOK | 9.6-19.2 kBaud | 100 kHz | None | Fast remotes |
| 315-FSK-Sync | 2-FSK | 4.8 kBaud | 58 kHz | 16/16 | Bidirectional sensors |

### 8.2 433 MHz Band Configurations

| Profile | Modulation | Data Rate | Bandwidth | Sync | Use Case |
|---------|------------|-----------|-----------|------|----------|
| 433-OOK-Keyfob | ASK/OOK | 1-4.8 kBaud | 58 kHz | None | Key fobs, simple remotes |
| 433-OOK-PWM | ASK/OOK | 2.4-4.8 kBaud | 58 kHz | None | PWM-encoded remotes |
| 433-OOK-Manch | ASK/OOK | 4.8-9.6 kBaud | 100 kHz | None | Manchester-encoded remotes |
| 433-2FSK-Standard | 2-FSK | 4.8-9.6 kBaud | 58 kHz | 16/16 | Digital sensors |
| 433-2FSK-Fast | 2-FSK | 38.4-100 kBaud | 200 kHz | 16/16 | High-speed links |
| 433-GFSK-CRC | GFSK | 9.6-38.4 kBaud | 100 kHz | 16/16 | Smart home devices |
| 433-4FSK | 4-FSK | 50-200 kBaud | 200 kHz | 16/16 | High-throughput |

### 8.3 868 MHz Band Configurations (European ISM)

| Profile | Modulation | Data Rate | Bandwidth | Sync | Use Case |
|---------|------------|-----------|-----------|------|----------|
| 868-OOK-Simple | ASK/OOK | 1-9.6 kBaud | 100 kHz | None | Simple remotes |
| 868-2FSK-Manch | 2-FSK+Manchester | 4.8-19.2 kBaud | 63 kHz | 16/16 | EU regulatory compliance |
| 868-2FSK-Fast | 2-FSK | 38.4-100 kBaud | 200 kHz | 16/16 | High-speed sensors |
| 868-GFSK-Smart | GFSK | 9.6-38.4 kBaud | 100 kHz | 16/16 | Smart metering |
| 868-GFSK-FEC | GFSK+FEC | 19.2-38.4 kBaud | 150 kHz | 16/16 | Robust industrial |

### 8.4 902-928 MHz Band Configurations (US ISM)

| Profile | Modulation | Data Rate | Bandwidth | Sync | Use Case |
|---------|------------|-----------|-----------|------|----------|
| 915-OOK-TPMS | ASK/OOK | 4.8-19.2 kBaud | 100 kHz | Varies | TPMS, simple sensors |
| 915-2FSK-Sensor | 2-FSK | 9.6-38.4 kBaud | 100 kHz | 16/16 | Wireless sensors |
| 915-GFSK-Standard | GFSK | 38.4 kBaud | 94 kHz | 16/16 | Standard digital links |
| 915-GFSK-CRC-FEC | GFSK+CRC+FEC | 38.4-100 kBaud | 150 kHz | 16/16 | Robust sensor networks |
| 915-FHSS | GFSK | 100-250 kBaud | 300 kHz | 16/16 | Frequency hopping systems |
| 915-Max | 2-FSK | 250-500 kBaud | 500 kHz | 16/16 | Maximum throughput |

### 8.5 Multi-Band / Special Configurations

| Profile | Frequency | Modulation | Data Rate | Notes |
|---------|-----------|------------|-----------|-------|
| LongRange-Any | Any band | 2-FSK | 1.2-2.4 kBaud | -110 dBm sensitivity |
| HighSpeed-Any | Any band | 2-FSK | 500 kBaud | Maximum throughput |
| FHSS-915 | 902-928 MHz | GFSK | 100-250 kBaud | Multi-channel hopping |
| FHSS-433 | 433 MHz | GFSK | 50-100 kBaud | Multi-channel hopping |
| AES-Encrypted | Any band | Any | Varies | Hardware AES-128 |
| SpectrumMonitor | Any band | N/A | N/A | RSSI scanning only |

---

## 9. Capability Matrix Summary

### 9.1 Modulation vs Data Rate Support

| Modulation | 600-4.8k | 4.8-26k | 26-250k | 250-500k |
|------------|----------|---------|---------|----------|
| 2-FSK | Yes | Yes | Yes | Yes |
| GFSK | Yes | Yes | Yes | Yes |
| ASK/OOK | Yes | Yes | Yes | No |
| 4-FSK | Yes | Yes | Yes | Limited |
| MSK | No | No | Yes | Yes |

### 9.2 Encoding vs Modulation Compatibility

| Encoding | 2-FSK | GFSK | ASK/OOK | 4-FSK | MSK |
|----------|-------|------|---------|-------|-----|
| None | Yes | Yes | Yes | Yes | Yes |
| Manchester | Yes | Yes | Yes | No | Yes |
| Whitening | Yes | Yes | Yes | Yes | Yes |
| FEC | Yes | Yes | Yes | Yes | Yes |

### 9.3 Features by Use Case

| Use Case | Typical Profile | Key Features |
|----------|-----------------|--------------|
| Key Fobs/Remotes | 433-OOK-Keyfob | ASK/OOK, no sync, fixed length |
| Wireless Sensors | 915-GFSK-Standard | GFSK, sync, variable length, CRC |
| Smart Home | 433-GFSK-CRC | GFSK, CRC, bidirectional |
| Industrial | 868-GFSK-FEC | GFSK, FEC, robust error handling |
| TPMS Analysis | 915-OOK-TPMS | ASK/OOK or FSK, various sync |
| Long Range | LongRange-Any | Low baud, narrow bandwidth |
| High Throughput | HighSpeed-Any | High baud, wide bandwidth |
| Secure Links | AES-Encrypted | Any modulation + HW AES |
| Protocol Discovery | SpectrumMonitor | RSSI scanning, no demodulation |

---

## 10. Hardware Constraints and Limitations

| Parameter | Limit | Notes |
|-----------|-------|-------|
| Frequency Range | 281-962 MHz | Three bands with gaps |
| Max TX Power | +10 dBm | Frequency-dependent PA tables |
| Max RX Sensitivity | -110 dBm | At 1.2 kBaud |
| Max Data Rate | 500 kBaud | Modulation-dependent |
| Max Packet (fixed) | 512 bytes | EP5OUT_BUFFER_SIZE - 4 |
| Max Packet (variable) | 255 bytes | Length byte is 8 bits |
| Max Long TX | 65535 bytes | Chunked transmission |
| USB Speed | Full Speed (12 Mbps) | USB 2.0 |
| RAM | 4 KB | Limited buffer space |
| Flash | 32 KB | Firmware storage |
| AES Key | 128 bits | Hardware accelerated |

---

## 11. Complete Configuration Checklist

When configuring a radio profile, ensure all these parameters are set:

**Frequency:**
- [ ] FREQ2/1/0 (carrier frequency)
- [ ] FSCAL2 (VCO selection)
- [ ] CHANNR (channel number if using channel spacing)
- [ ] FSCTRL1/0 (frequency synthesizer control)

**Modulation:**
- [ ] MDMCFG2 (modulation type + sync mode)
- [ ] MDMCFG4/3 (data rate + channel bandwidth)
- [ ] DEVIATN (FSK deviation)
- [ ] Manchester enable (if needed)

**Packet:**
- [ ] PKTCTRL0 (packet format: fixed/variable)
- [ ] PKTCTRL1 (status bytes, address check)
- [ ] PKTLEN (length or max length)
- [ ] SYNC1/0 (sync word)
- [ ] ADDR (device address if filtering)
- [ ] MDMCFG1/0 (preamble length, channel spacing)

**Error Handling:**
- [ ] CRC enable
- [ ] FEC enable
- [ ] PQT (preamble quality threshold)

**Power/Frontend:**
- [ ] PA_TABLE0/1 (power levels)
- [ ] FREND0 (TX frontend: PA table index)
- [ ] FREND1 (RX frontend: LNA settings)
- [ ] TEST2/1 (bandwidth-dependent)

**State Machine:**
- [ ] MCSM0/1/2 (auto-calibration, CCA mode, timeouts)

**Optional:**
- [ ] AGCCTRL0/1/2 (AGC settings)
- [ ] FOCCFG (frequency offset compensation)
- [ ] BSCFG (bit synchronization)
- [ ] IOCFG0/1/2 (GPIO pin configurations)

---

## 12. Revision History

| Date | Description |
|------|-------------|
| 2025-11-29 | Initial capabilities list created from CC1111 datasheet and RfCat documentation |
