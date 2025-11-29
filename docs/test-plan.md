# YardStick One Test Plan

This document defines the complete test plan for validating all YS1 configuration profiles. Each test references configuration files stored in `./tests/etc/` following the naming convention defined below.

---

## Table of Contents

1. [Test Infrastructure](#1-test-infrastructure)
2. [Naming Convention](#2-naming-convention)
3. [Test Tracking Summary](#3-test-tracking-summary)
4. [315 MHz Band Tests](#4-315-mhz-band-tests)
5. [433 MHz Band Tests](#5-433-mhz-band-tests)
6. [868 MHz Band Tests](#6-868-mhz-band-tests)
7. [915 MHz Band Tests](#7-915-mhz-band-tests)
8. [Multi-Band Special Tests](#8-multi-band-special-tests)
9. [Encoding Variation Tests](#9-encoding-variation-tests)
10. [Packet Format Tests](#10-packet-format-tests)
11. [Advanced Feature Tests](#11-advanced-feature-tests)
12. [Integration Tests](#12-integration-tests)
13. [Revision History](#13-revision-history)

---

## 1. Test Infrastructure

### 1.1 Directory Structure

```
./tests/
├── etc/                          # Configuration files
│   ├── 315-ook-low-1k2.json
│   ├── 315-ook-low-2k4.json
│   ├── ...
│   └── spectrum-monitor-915.json
├── data/                         # Test data payloads
│   ├── payload-small.bin         # 16 bytes
│   ├── payload-medium.bin        # 64 bytes
│   ├── payload-large.bin         # 255 bytes
│   └── payload-max.bin           # 512 bytes (fixed mode only)
├── results/                      # Test results output
│   └── <timestamp>/
└── scripts/                      # Test automation scripts
    ├── run-loopback-test.sh
    ├── run-config-validation.sh
    └── run-all-tests.sh
```

### 1.2 Hardware Requirements

- **Minimum**: 2x YardStick One devices (TX + RX)
- **Recommended**: 3x YardStick One devices (TX + RX + monitor)
- USB 2.0 ports (USB 3.0 may cause issues)
- RF shielded enclosure for controlled testing (optional)

### 1.3 Test Types

| Type | Code | Description |
|------|------|-------------|
| Config Validation | CV | Verify registers are set correctly after load |
| Loopback TX/RX | LB | Transmit from one device, receive on another |
| Range Test | RT | Test at various distances/attenuations |
| Error Injection | EI | Verify CRC/FEC error handling |
| Stress Test | ST | Extended duration / high packet rate |

### 1.4 Test Procedure Template

For each loopback test:
1. Load TX config to Device A
2. Load RX config to Device B (same as TX unless noted)
3. Set Device B to RX mode
4. Transmit test payload from Device A
5. Verify Device B receives correct payload
6. Record RSSI/LQI values
7. Repeat with multiple payload sizes

---

## 2. Naming Convention

### 2.1 Config File Naming

Format: `<freq>-<mod>-<profile>-<rate>[-<options>].json`

**Components:**
- `<freq>`: Frequency band (315, 433, 868, 915, multi)
- `<mod>`: Modulation (ook, 2fsk, gfsk, 4fsk, msk)
- `<profile>`: Profile variant name
- `<rate>`: Data rate (1k2, 2k4, 4k8, 9k6, 19k2, 38k4, 100k, 250k, 500k)
- `<options>`: Optional features (manch, white, fec, crc, sync)

### 2.2 Examples

| Config File | Description |
|-------------|-------------|
| `315-ook-low-1k2.json` | 315 MHz OOK Low profile at 1.2 kBaud |
| `433-gfsk-crc-38k4-fec.json` | 433 MHz GFSK with CRC at 38.4k with FEC |
| `915-2fsk-sensor-19k2-manch.json` | 915 MHz 2-FSK sensor at 19.2k with Manchester |

### 2.3 TX/RX Config Pairs

Most tests use symmetric configs. When TX and RX differ, use suffixes:
- `<name>-tx.json` - Transmitter config
- `<name>-rx.json` - Receiver config

---

## 3. Test Tracking Summary

### 3.1 Overall Progress

| Category | Total Tests | Completed | Remaining |
|----------|-------------|-----------|-----------|
| 315 MHz Band | 9 | 0 | 9 |
| 433 MHz Band | 21 | 0 | 21 |
| 868 MHz Band | 15 | 0 | 15 |
| 915 MHz Band | 18 | 0 | 18 |
| Multi-Band Special | 18 | 0 | 18 |
| Encoding Variations | 12 | 0 | 12 |
| Packet Format | 12 | 0 | 12 |
| Advanced Features | 15 | 0 | 15 |
| Integration Tests | 8 | 0 | 8 |
| **TOTAL** | **128** | **0** | **128** |

### 3.2 Config File Creation Progress

| Category | Total Configs | Created | Remaining |
|----------|---------------|---------|-----------|
| Core Profiles | 81 | 0 | 81 |
| Encoding Variants | 12 | 0 | 12 |
| Packet Variants | 12 | 0 | 12 |
| Special Features | 23 | 0 | 23 |
| **TOTAL** | **128** | **0** | **128** |

---

## 4. 315 MHz Band Tests

### 4.1 Profile: 315-OOK-Low (Key Fobs, Garage Doors)

**Base Configuration:**
- Frequency: 315.000 MHz
- Modulation: ASK/OOK
- Bandwidth: 58 kHz
- Sync: None
- Packet: Fixed length, 64 bytes

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 315-OOK-L-01 | `315-ook-low-1k2.json` | 1.2 kBaud | Minimum rate |
| - [ ] | 315-OOK-L-02 | `315-ook-low-2k4.json` | 2.4 kBaud | Common key fob |
| - [ ] | 315-OOK-L-03 | `315-ook-low-4k8.json` | 4.8 kBaud | Maximum for profile |

**Test Procedure:** Standard loopback (Section 1.4)
**TX/RX Config:** Symmetric (same config for both)

### 4.2 Profile: 315-OOK-Fast (Fast Remotes)

**Base Configuration:**
- Frequency: 315.000 MHz
- Modulation: ASK/OOK
- Bandwidth: 100 kHz
- Sync: None
- Packet: Fixed length, 64 bytes

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 315-OOK-F-01 | `315-ook-fast-9k6.json` | 9.6 kBaud | Minimum for profile |
| - [ ] | 315-OOK-F-02 | `315-ook-fast-19k2.json` | 19.2 kBaud | Maximum for profile |

**Test Procedure:** Standard loopback
**TX/RX Config:** Symmetric

### 4.3 Profile: 315-FSK-Sync (Bidirectional Sensors)

**Base Configuration:**
- Frequency: 315.000 MHz
- Modulation: 2-FSK
- Bandwidth: 58 kHz
- Sync: 16/16 (0xD391)
- Packet: Variable length, max 60 bytes, CRC enabled

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 315-FSK-S-01 | `315-2fsk-sync-2k4.json` | 2.4 kBaud | Low rate |
| - [ ] | 315-FSK-S-02 | `315-2fsk-sync-4k8.json` | 4.8 kBaud | Standard |
| - [ ] | 315-FSK-S-03 | `315-2fsk-sync-9k6.json` | 9.6 kBaud | Higher rate |
| - [ ] | 315-FSK-S-04 | `315-2fsk-sync-4k8-fec.json` | 4.8 kBaud | With FEC |

**Test Procedure:** Standard loopback + CRC validation
**TX/RX Config:** Symmetric

---

## 5. 433 MHz Band Tests

### 5.1 Profile: 433-OOK-Keyfob (Key Fobs, Simple Remotes)

**Base Configuration:**
- Frequency: 433.920 MHz
- Modulation: ASK/OOK
- Bandwidth: 58 kHz
- Sync: None
- Packet: Fixed length, 64 bytes

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 433-OOK-K-01 | `433-ook-keyfob-1k2.json` | 1.2 kBaud | Minimum |
| - [ ] | 433-OOK-K-02 | `433-ook-keyfob-2k4.json` | 2.4 kBaud | Common |
| - [ ] | 433-OOK-K-03 | `433-ook-keyfob-4k8.json` | 4.8 kBaud | Maximum |

**Test Procedure:** Standard loopback
**TX/RX Config:** Symmetric

### 5.2 Profile: 433-OOK-PWM (PWM-Encoded Remotes)

**Base Configuration:**
- Frequency: 433.920 MHz
- Modulation: ASK/OOK
- Bandwidth: 58 kHz
- Sync: None
- Packet: Fixed length, 64 bytes

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 433-OOK-P-01 | `433-ook-pwm-2k4.json` | 2.4 kBaud | Low end |
| - [ ] | 433-OOK-P-02 | `433-ook-pwm-4k8.json` | 4.8 kBaud | High end |

**Test Procedure:** Standard loopback
**TX/RX Config:** Symmetric

### 5.3 Profile: 433-OOK-Manch (Manchester-Encoded Remotes)

**Base Configuration:**
- Frequency: 433.920 MHz
- Modulation: ASK/OOK + Manchester
- Bandwidth: 100 kHz
- Sync: None
- Packet: Fixed length, 64 bytes

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 433-OOK-M-01 | `433-ook-manch-4k8.json` | 4.8 kBaud | Low end |
| - [ ] | 433-OOK-M-02 | `433-ook-manch-9k6.json` | 9.6 kBaud | High end |

**Test Procedure:** Standard loopback
**TX/RX Config:** Symmetric

### 5.4 Profile: 433-2FSK-Standard (Digital Sensors)

**Base Configuration:**
- Frequency: 433.920 MHz
- Modulation: 2-FSK
- Bandwidth: 58 kHz
- Deviation: 5 kHz
- Sync: 16/16 (0xD391)
- Packet: Variable length, max 60 bytes, CRC enabled

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 433-2FSK-S-01 | `433-2fsk-std-4k8.json` | 4.8 kBaud | Minimum |
| - [ ] | 433-2FSK-S-02 | `433-2fsk-std-9k6.json` | 9.6 kBaud | Maximum |
| - [ ] | 433-2FSK-S-03 | `433-2fsk-std-4k8-fec.json` | 4.8 kBaud | With FEC |

**Test Procedure:** Standard loopback + CRC validation
**TX/RX Config:** Symmetric

### 5.5 Profile: 433-2FSK-Fast (High-Speed Links)

**Base Configuration:**
- Frequency: 433.920 MHz
- Modulation: 2-FSK
- Bandwidth: 200 kHz
- Deviation: 25 kHz
- Sync: 16/16 (0xD391)
- Packet: Variable length, max 255 bytes, CRC enabled

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 433-2FSK-F-01 | `433-2fsk-fast-38k4.json` | 38.4 kBaud | Low end |
| - [ ] | 433-2FSK-F-02 | `433-2fsk-fast-76k8.json` | 76.8 kBaud | Mid |
| - [ ] | 433-2FSK-F-03 | `433-2fsk-fast-100k.json` | 100 kBaud | Maximum |

**Test Procedure:** Standard loopback + CRC validation
**TX/RX Config:** Symmetric

### 5.6 Profile: 433-GFSK-CRC (Smart Home Devices)

**Base Configuration:**
- Frequency: 433.920 MHz
- Modulation: GFSK
- Bandwidth: 100 kHz
- Deviation: 10 kHz
- Sync: 16/16 (0xD391)
- Packet: Variable length, max 60 bytes, CRC enabled

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 433-GFSK-C-01 | `433-gfsk-crc-9k6.json` | 9.6 kBaud | Low end |
| - [ ] | 433-GFSK-C-02 | `433-gfsk-crc-19k2.json` | 19.2 kBaud | Mid |
| - [ ] | 433-GFSK-C-03 | `433-gfsk-crc-38k4.json` | 38.4 kBaud | Maximum |
| - [ ] | 433-GFSK-C-04 | `433-gfsk-crc-19k2-fec.json` | 19.2 kBaud | With FEC |

**Test Procedure:** Standard loopback + CRC validation
**TX/RX Config:** Symmetric

### 5.7 Profile: 433-4FSK (High-Throughput)

**Base Configuration:**
- Frequency: 433.920 MHz
- Modulation: 4-FSK
- Bandwidth: 200 kHz
- Sync: 16/16 (0xD391)
- Packet: Variable length, max 255 bytes, CRC enabled

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 433-4FSK-01 | `433-4fsk-50k.json` | 50 kBaud | Low end |
| - [ ] | 433-4FSK-02 | `433-4fsk-100k.json` | 100 kBaud | Mid |
| - [ ] | 433-4FSK-03 | `433-4fsk-200k.json` | 200 kBaud | Maximum |

**Test Procedure:** Standard loopback + CRC validation
**TX/RX Config:** Symmetric
**Note:** Manchester encoding NOT supported with 4-FSK

---

## 6. 868 MHz Band Tests

### 6.1 Profile: 868-OOK-Simple (Simple Remotes)

**Base Configuration:**
- Frequency: 868.300 MHz
- Modulation: ASK/OOK
- Bandwidth: 100 kHz
- Sync: None
- Packet: Fixed length, 64 bytes

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 868-OOK-S-01 | `868-ook-simple-1k2.json` | 1.2 kBaud | Minimum |
| - [ ] | 868-OOK-S-02 | `868-ook-simple-4k8.json` | 4.8 kBaud | Mid |
| - [ ] | 868-OOK-S-03 | `868-ook-simple-9k6.json` | 9.6 kBaud | Maximum |

**Test Procedure:** Standard loopback
**TX/RX Config:** Symmetric

### 6.2 Profile: 868-2FSK-Manch (EU Regulatory Compliance)

**Base Configuration:**
- Frequency: 868.300 MHz
- Modulation: 2-FSK + Manchester
- Bandwidth: 63 kHz
- Deviation: 5.1 kHz
- Sync: 16/16 (0xAAAA)
- Packet: Variable length, max 60 bytes, CRC enabled

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 868-2FSK-M-01 | `868-2fsk-manch-4k8.json` | 4.8 kBaud | Minimum |
| - [ ] | 868-2FSK-M-02 | `868-2fsk-manch-9k6.json` | 9.6 kBaud | Mid |
| - [ ] | 868-2FSK-M-03 | `868-2fsk-manch-19k2.json` | 19.2 kBaud | Maximum |

**Test Procedure:** Standard loopback + CRC validation
**TX/RX Config:** Symmetric

### 6.3 Profile: 868-2FSK-Fast (High-Speed Sensors)

**Base Configuration:**
- Frequency: 868.300 MHz
- Modulation: 2-FSK
- Bandwidth: 200 kHz
- Deviation: 25 kHz
- Sync: 16/16 (0xD391)
- Packet: Variable length, max 255 bytes, CRC enabled

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 868-2FSK-F-01 | `868-2fsk-fast-38k4.json` | 38.4 kBaud | Minimum |
| - [ ] | 868-2FSK-F-02 | `868-2fsk-fast-76k8.json` | 76.8 kBaud | Mid |
| - [ ] | 868-2FSK-F-03 | `868-2fsk-fast-100k.json` | 100 kBaud | Maximum |

**Test Procedure:** Standard loopback + CRC validation
**TX/RX Config:** Symmetric

### 6.4 Profile: 868-GFSK-Smart (Smart Metering)

**Base Configuration:**
- Frequency: 868.300 MHz
- Modulation: GFSK
- Bandwidth: 100 kHz
- Deviation: 10 kHz
- Sync: 16/16 (0xD391)
- Packet: Variable length, max 60 bytes, CRC enabled

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 868-GFSK-S-01 | `868-gfsk-smart-9k6.json` | 9.6 kBaud | Minimum |
| - [ ] | 868-GFSK-S-02 | `868-gfsk-smart-19k2.json` | 19.2 kBaud | Mid |
| - [ ] | 868-GFSK-S-03 | `868-gfsk-smart-38k4.json` | 38.4 kBaud | Maximum |

**Test Procedure:** Standard loopback + CRC validation
**TX/RX Config:** Symmetric

### 6.5 Profile: 868-GFSK-FEC (Robust Industrial)

**Base Configuration:**
- Frequency: 868.300 MHz
- Modulation: GFSK + FEC
- Bandwidth: 150 kHz
- Deviation: 15 kHz
- Sync: 16/16 (0xD391)
- Packet: Variable length, max 60 bytes, CRC enabled, FEC enabled

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 868-GFSK-FEC-01 | `868-gfsk-fec-19k2.json` | 19.2 kBaud | Minimum |
| - [ ] | 868-GFSK-FEC-02 | `868-gfsk-fec-38k4.json` | 38.4 kBaud | Maximum |
| - [ ] | 868-GFSK-FEC-03 | `868-gfsk-fec-19k2-white.json` | 19.2 kBaud | With whitening |

**Test Procedure:** Standard loopback + CRC validation + FEC error recovery test
**TX/RX Config:** Symmetric

---

## 7. 915 MHz Band Tests

### 7.1 Profile: 915-OOK-TPMS (TPMS, Simple Sensors)

**Base Configuration:**
- Frequency: 915.000 MHz
- Modulation: ASK/OOK
- Bandwidth: 100 kHz
- Sync: Varies (test both none and 15/16)
- Packet: Fixed length, 64 bytes

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 915-OOK-T-01 | `915-ook-tpms-4k8-nosync.json` | 4.8 kBaud | No sync |
| - [ ] | 915-OOK-T-02 | `915-ook-tpms-9k6-nosync.json` | 9.6 kBaud | No sync |
| - [ ] | 915-OOK-T-03 | `915-ook-tpms-19k2-nosync.json` | 19.2 kBaud | No sync |
| - [ ] | 915-OOK-T-04 | `915-ook-tpms-9k6-sync.json` | 9.6 kBaud | 15/16 sync |

**Test Procedure:** Standard loopback
**TX/RX Config:** Symmetric

### 7.2 Profile: 915-2FSK-Sensor (Wireless Sensors)

**Base Configuration:**
- Frequency: 915.000 MHz
- Modulation: 2-FSK
- Bandwidth: 100 kHz
- Deviation: 10 kHz
- Sync: 16/16 (0xD391)
- Packet: Variable length, max 60 bytes, CRC enabled

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 915-2FSK-S-01 | `915-2fsk-sensor-9k6.json` | 9.6 kBaud | Minimum |
| - [ ] | 915-2FSK-S-02 | `915-2fsk-sensor-19k2.json` | 19.2 kBaud | Mid |
| - [ ] | 915-2FSK-S-03 | `915-2fsk-sensor-38k4.json` | 38.4 kBaud | Maximum |

**Test Procedure:** Standard loopback + CRC validation
**TX/RX Config:** Symmetric

### 7.3 Profile: 915-GFSK-Standard (Standard Digital Links)

**Base Configuration:**
- Frequency: 915.000 MHz
- Modulation: GFSK
- Bandwidth: 94 kHz
- Deviation: 20 kHz
- Sync: 16/16 (0xD391)
- Packet: Variable length, max 60 bytes, CRC enabled

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 915-GFSK-S-01 | `915-gfsk-std-38k4.json` | 38.4 kBaud | Standard |
| - [ ] | 915-GFSK-S-02 | `915-gfsk-std-38k4-white.json` | 38.4 kBaud | With whitening |

**Test Procedure:** Standard loopback + CRC validation
**TX/RX Config:** Symmetric

### 7.4 Profile: 915-GFSK-CRC-FEC (Robust Sensor Networks)

**Base Configuration:**
- Frequency: 915.000 MHz
- Modulation: GFSK + FEC
- Bandwidth: 150 kHz
- Deviation: 25 kHz
- Sync: 16/16 (0xD391)
- Packet: Variable length, max 60 bytes, CRC enabled, FEC enabled

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 915-GFSK-CF-01 | `915-gfsk-crc-fec-38k4.json` | 38.4 kBaud | Minimum |
| - [ ] | 915-GFSK-CF-02 | `915-gfsk-crc-fec-76k8.json` | 76.8 kBaud | Mid |
| - [ ] | 915-GFSK-CF-03 | `915-gfsk-crc-fec-100k.json` | 100 kBaud | Maximum |

**Test Procedure:** Standard loopback + CRC validation + FEC error recovery
**TX/RX Config:** Symmetric

### 7.5 Profile: 915-FHSS (Frequency Hopping Systems)

**Base Configuration:**
- Frequency: 902-928 MHz (hopping)
- Modulation: GFSK
- Bandwidth: 300 kHz
- Sync: 16/16 (0xD391)
- Packet: Variable length, max 255 bytes, CRC enabled
- Channels: 50 channels, 500 kHz spacing

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 915-FHSS-01 | `915-fhss-100k-master.json` | 100 kBaud | Master config |
| - [ ] | 915-FHSS-01 | `915-fhss-100k-slave.json` | 100 kBaud | Slave config |
| - [ ] | 915-FHSS-02 | `915-fhss-250k-master.json` | 250 kBaud | Master config |
| - [ ] | 915-FHSS-02 | `915-fhss-250k-slave.json` | 250 kBaud | Slave config |

**Test Procedure:** FHSS sync test + loopback across multiple channels
**TX/RX Config:** Asymmetric (master/slave)

### 7.6 Profile: 915-Max (Maximum Throughput)

**Base Configuration:**
- Frequency: 915.000 MHz
- Modulation: 2-FSK
- Bandwidth: 500 kHz
- Deviation: 100 kHz
- Sync: 16/16 (0xD391)
- Packet: Variable length, max 255 bytes, CRC enabled

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | 915-MAX-01 | `915-2fsk-max-250k.json` | 250 kBaud | Lower max |
| - [ ] | 915-MAX-02 | `915-2fsk-max-500k.json` | 500 kBaud | Maximum |

**Test Procedure:** Standard loopback + throughput measurement
**TX/RX Config:** Symmetric

---

## 8. Multi-Band Special Tests

### 8.1 Profile: LongRange-Any (Maximum Sensitivity)

**Base Configuration:**
- Modulation: 2-FSK
- Bandwidth: 58 kHz
- Deviation: 2.4 kHz
- Sync: 16/16 (0xD391)
- Packet: Variable length, max 60 bytes, CRC enabled

| Status | Test ID | Config File | Data Rate | Frequency | Notes |
|--------|---------|-------------|-----------|-----------|-------|
| - [ ] | LR-315-01 | `315-2fsk-longrange-1k2.json` | 1.2 kBaud | 315 MHz | 315 band |
| - [ ] | LR-433-01 | `433-2fsk-longrange-1k2.json` | 1.2 kBaud | 433 MHz | 433 band |
| - [ ] | LR-868-01 | `868-2fsk-longrange-1k2.json` | 1.2 kBaud | 868 MHz | 868 band |
| - [ ] | LR-915-01 | `915-2fsk-longrange-1k2.json` | 1.2 kBaud | 915 MHz | 915 band |
| - [ ] | LR-315-02 | `315-2fsk-longrange-2k4.json` | 2.4 kBaud | 315 MHz | Higher rate |
| - [ ] | LR-433-02 | `433-2fsk-longrange-2k4.json` | 2.4 kBaud | 433 MHz | Higher rate |

**Test Procedure:** Standard loopback + RSSI measurement at low signal levels
**TX/RX Config:** Symmetric
**Expected Sensitivity:** -110 dBm @ 1.2 kBaud

### 8.2 Profile: HighSpeed-Any (Maximum Throughput)

**Base Configuration:**
- Modulation: 2-FSK
- Bandwidth: 500 kHz
- Deviation: 100 kHz
- Sync: 16/16 (0xD391)
- Packet: Variable length, max 255 bytes, CRC enabled

| Status | Test ID | Config File | Data Rate | Frequency | Notes |
|--------|---------|-------------|-----------|-----------|-------|
| - [ ] | HS-433-01 | `433-2fsk-highspeed-500k.json` | 500 kBaud | 433 MHz | 433 band |
| - [ ] | HS-868-01 | `868-2fsk-highspeed-500k.json` | 500 kBaud | 868 MHz | 868 band |
| - [ ] | HS-915-01 | `915-2fsk-highspeed-500k.json` | 500 kBaud | 915 MHz | 915 band |

**Test Procedure:** Standard loopback + throughput measurement
**TX/RX Config:** Symmetric

### 8.3 Profile: FHSS-433 (433 MHz Frequency Hopping)

**Base Configuration:**
- Frequency: 433-434 MHz (hopping)
- Modulation: GFSK
- Bandwidth: 200 kHz
- Sync: 16/16 (0xD391)
- Packet: Variable length, max 255 bytes, CRC enabled
- Channels: 10 channels, 100 kHz spacing

| Status | Test ID | Config File | Data Rate | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | FHSS433-01 | `433-fhss-50k-master.json` | 50 kBaud | Master |
| - [ ] | FHSS433-01 | `433-fhss-50k-slave.json` | 50 kBaud | Slave |
| - [ ] | FHSS433-02 | `433-fhss-100k-master.json` | 100 kBaud | Master |
| - [ ] | FHSS433-02 | `433-fhss-100k-slave.json` | 100 kBaud | Slave |

**Test Procedure:** FHSS sync test + loopback
**TX/RX Config:** Asymmetric (master/slave)

### 8.4 Profile: AES-Encrypted (Hardware Encryption)

**Base Configuration:**
- Various frequencies
- Modulation: GFSK
- AES-128 enabled
- Packet: Variable length, CRC enabled

| Status | Test ID | Config File | AES Mode | Frequency | Notes |
|--------|---------|-------------|----------|-----------|-------|
| - [ ] | AES-ECB-01 | `433-gfsk-aes-ecb.json` | ECB | 433 MHz | Basic encryption |
| - [ ] | AES-CBC-01 | `433-gfsk-aes-cbc.json` | CBC | 433 MHz | Chain mode |
| - [ ] | AES-CTR-01 | `915-gfsk-aes-ctr.json` | CTR | 915 MHz | Counter mode |
| - [ ] | AES-CBC-02 | `915-gfsk-aes-cbc.json` | CBC | 915 MHz | Chain mode |

**Test Procedure:** Encrypted loopback + verify decryption
**TX/RX Config:** Symmetric (same key)
**Note:** Requires AES key configuration in both devices

### 8.5 Profile: SpectrumMonitor (RSSI Scanning)

**Base Configuration:**
- Various frequencies
- RX only (no TX)
- RSSI capture mode

| Status | Test ID | Config File | Frequency Range | Notes |
|--------|---------|-------------|-----------------|-------|
| - [ ] | SPEC-315-01 | `spectrum-monitor-315.json` | 310-320 MHz | 315 band |
| - [ ] | SPEC-433-01 | `spectrum-monitor-433.json` | 430-440 MHz | 433 band |
| - [ ] | SPEC-868-01 | `spectrum-monitor-868.json` | 863-870 MHz | 868 band |
| - [ ] | SPEC-915-01 | `spectrum-monitor-915.json` | 902-928 MHz | 915 band |

**Test Procedure:** Verify RSSI readings across frequency sweep
**TX/RX Config:** RX only configuration
**Note:** No transmission, RX/spectrum analysis only

---

## 9. Encoding Variation Tests

These tests verify encoding options work correctly across different modulations.

### 9.1 Manchester Encoding Tests

| Status | Test ID | Config File | Base Profile | Notes |
|--------|---------|-------------|--------------|-------|
| - [ ] | MANCH-01 | `433-2fsk-std-9k6-manch.json` | 433-2FSK-Standard | 2-FSK + Manchester |
| - [ ] | MANCH-02 | `433-gfsk-crc-19k2-manch.json` | 433-GFSK-CRC | GFSK + Manchester |
| - [ ] | MANCH-03 | `915-2fsk-sensor-19k2-manch.json` | 915-2FSK-Sensor | 2-FSK + Manchester |
| - [ ] | MANCH-04 | `915-gfsk-std-38k4-manch.json` | 915-GFSK-Standard | GFSK + Manchester |

**Test Procedure:** Standard loopback
**TX/RX Config:** Symmetric
**Note:** Manchester NOT compatible with 4-FSK

### 9.2 Data Whitening Tests

| Status | Test ID | Config File | Base Profile | Notes |
|--------|---------|-------------|--------------|-------|
| - [ ] | WHITE-01 | `433-2fsk-std-9k6-white.json` | 433-2FSK-Standard | 2-FSK + Whitening |
| - [ ] | WHITE-02 | `433-gfsk-crc-19k2-white.json` | 433-GFSK-CRC | GFSK + Whitening |
| - [ ] | WHITE-03 | `915-2fsk-sensor-19k2-white.json` | 915-2FSK-Sensor | 2-FSK + Whitening |
| - [ ] | WHITE-04 | `915-4fsk-100k-white.json` | 915-4FSK variant | 4-FSK + Whitening |

**Test Procedure:** Standard loopback
**TX/RX Config:** Symmetric

### 9.3 FEC Tests (Standalone)

| Status | Test ID | Config File | Base Profile | Notes |
|--------|---------|-------------|--------------|-------|
| - [ ] | FEC-01 | `433-2fsk-std-9k6-fec.json` | 433-2FSK-Standard | 2-FSK + FEC |
| - [ ] | FEC-02 | `433-gfsk-crc-19k2-fec.json` | 433-GFSK-CRC | GFSK + FEC |
| - [ ] | FEC-03 | `915-2fsk-sensor-19k2-fec.json` | 915-2FSK-Sensor | 2-FSK + FEC |
| - [ ] | FEC-04 | `868-2fsk-manch-9k6-fec.json` | 868-2FSK-Manch | Manchester + FEC |

**Test Procedure:** Standard loopback + FEC error recovery verification
**TX/RX Config:** Symmetric

---

## 10. Packet Format Tests

These tests verify different packet configurations.

### 10.1 Fixed vs Variable Length

| Status | Test ID | Config File | Length Mode | Size | Notes |
|--------|---------|-------------|-------------|------|-------|
| - [ ] | PKT-FIX-01 | `433-gfsk-fixed-16.json` | Fixed | 16 bytes | Small |
| - [ ] | PKT-FIX-02 | `433-gfsk-fixed-64.json` | Fixed | 64 bytes | Medium |
| - [ ] | PKT-FIX-03 | `433-gfsk-fixed-255.json` | Fixed | 255 bytes | Large |
| - [ ] | PKT-FIX-04 | `433-gfsk-fixed-512.json` | Fixed | 512 bytes | Maximum |
| - [ ] | PKT-VAR-01 | `433-gfsk-var-16.json` | Variable | Max 16 | Small |
| - [ ] | PKT-VAR-02 | `433-gfsk-var-64.json` | Variable | Max 64 | Medium |
| - [ ] | PKT-VAR-03 | `433-gfsk-var-255.json` | Variable | Max 255 | Maximum |

**Test Procedure:** Loopback with various payload sizes
**TX/RX Config:** Symmetric

### 10.2 Sync Word Modes

| Status | Test ID | Config File | Sync Mode | Notes |
|--------|---------|-------------|-----------|-------|
| - [ ] | SYNC-01 | `433-gfsk-sync-none.json` | NONE | No sync word |
| - [ ] | SYNC-02 | `433-gfsk-sync-15of16.json` | 15/16 | Error tolerant |
| - [ ] | SYNC-03 | `433-gfsk-sync-16of16.json` | 16/16 | Exact match |
| - [ ] | SYNC-04 | `433-gfsk-sync-30of32.json` | 30/32 | Extended sync |
| - [ ] | SYNC-05 | `433-gfsk-sync-carrier.json` | CARRIER | Carrier only |

**Test Procedure:** Loopback with sync verification
**TX/RX Config:** Symmetric

---

## 11. Advanced Feature Tests

### 11.1 CCA (Clear Channel Assessment) Tests

| Status | Test ID | Config File | CCA Mode | Notes |
|--------|---------|-------------|----------|-------|
| - [ ] | CCA-01 | `433-gfsk-cca-mode0.json` | 0 | Always TX |
| - [ ] | CCA-02 | `433-gfsk-cca-mode1.json` | 1 | RSSI threshold |
| - [ ] | CCA-03 | `433-gfsk-cca-mode2.json` | 2 | Not receiving |
| - [ ] | CCA-04 | `433-gfsk-cca-mode3.json` | 3 | Combined |

**Test Procedure:** TX with channel busy simulation
**TX/RX Config:** Symmetric + interference source

### 11.2 Address Filtering Tests

| Status | Test ID | Config File | Address | Notes |
|--------|---------|-------------|---------|-------|
| - [ ] | ADDR-01 | `433-gfsk-addr-match.json` | 0x55 | Match test |
| - [ ] | ADDR-02 | `433-gfsk-addr-mismatch-tx.json` | 0x55 | TX address |
| - [ ] | ADDR-02 | `433-gfsk-addr-mismatch-rx.json` | 0xAA | RX address (should reject) |
| - [ ] | ADDR-03 | `433-gfsk-addr-broadcast.json` | 0x00 | Broadcast |

**Test Procedure:** Address filtering verification
**TX/RX Config:** Asymmetric for mismatch test

### 11.3 Preamble Length Tests

| Status | Test ID | Config File | Preamble | Notes |
|--------|---------|-------------|----------|-------|
| - [ ] | PREAM-01 | `433-gfsk-preamble-2.json` | 2 bytes | Minimum |
| - [ ] | PREAM-02 | `433-gfsk-preamble-4.json` | 4 bytes | Standard |
| - [ ] | PREAM-03 | `433-gfsk-preamble-8.json` | 8 bytes | Extended |
| - [ ] | PREAM-04 | `433-gfsk-preamble-24.json` | 24 bytes | Maximum |

**Test Procedure:** Loopback with preamble detection verification
**TX/RX Config:** Symmetric

### 11.4 Power Level Tests

| Status | Test ID | Config File | Power | Notes |
|--------|---------|-------------|-------|-------|
| - [ ] | PWR-01 | `433-gfsk-power-min.json` | -30 dBm | Minimum |
| - [ ] | PWR-02 | `433-gfsk-power-low.json` | -10 dBm | Low |
| - [ ] | PWR-03 | `433-gfsk-power-max.json` | +10 dBm | Maximum |

**Test Procedure:** Loopback with RSSI measurement correlation
**TX/RX Config:** Symmetric

---

## 12. Integration Tests

### 12.1 Cross-Configuration Compatibility

These tests verify certain configurations can still communicate when slightly mismatched.

| Status | Test ID | TX Config | RX Config | Notes |
|--------|---------|-----------|-----------|-------|
| - [ ] | XCOMPAT-01 | `433-gfsk-crc-19k2.json` | `433-gfsk-crc-19k2-fec.json` | TX no FEC, RX has FEC |
| - [ ] | XCOMPAT-02 | `433-2fsk-std-9k6.json` | `433-gfsk-crc-9k6.json` | 2-FSK TX to GFSK RX |
| - [ ] | XCOMPAT-03 | `915-gfsk-std-38k4.json` | `915-gfsk-std-38k4-white.json` | Whitening mismatch |

**Expected:** Some should fail, documenting compatibility boundaries

### 12.2 Bidirectional Communication

| Status | Test ID | Config File | Notes |
|--------|---------|-------------|-------|
| - [ ] | BIDIR-01 | `433-gfsk-crc-19k2.json` | TX/RX alternation |
| - [ ] | BIDIR-02 | `915-gfsk-crc-fec-38k4.json` | TX/RX alternation with FEC |

**Test Procedure:** Alternating TX/RX between both devices

### 12.3 Stress Tests

| Status | Test ID | Config File | Duration | Packet Rate | Notes |
|--------|---------|-------------|----------|-------------|-------|
| - [ ] | STRESS-01 | `433-gfsk-crc-38k4.json` | 10 min | 100 pkt/sec | Sustained load |
| - [ ] | STRESS-02 | `915-2fsk-max-500k.json` | 5 min | Max rate | Throughput stress |
| - [ ] | STRESS-03 | `915-fhss-100k-master.json` | 10 min | 50 pkt/sec | FHSS sustained |

**Test Procedure:** Extended duration packet transmission with error rate measurement
**TX/RX Config:** Symmetric (or master/slave for FHSS)

---

## 13. Revision History

| Date | Version | Description |
|------|---------|-------------|
| 2025-11-29 | 1.0 | Initial test plan created |

---

## Appendix A: Config File Template

All config files in `./tests/etc/` should follow this JSON structure:

```json
{
  "name": "433-gfsk-crc-19k2",
  "description": "433 MHz GFSK with CRC at 19.2 kBaud",
  "frequency_hz": 433920000,
  "modulation": "GFSK",
  "data_rate_baud": 19200,
  "deviation_hz": 10000,
  "channel_bandwidth_hz": 100000,
  "sync_word": "0xD391",
  "sync_mode": "16_of_16",
  "packet_length_mode": "variable",
  "packet_max_length": 60,
  "preamble_bytes": 4,
  "crc_enabled": true,
  "fec_enabled": false,
  "manchester_enabled": false,
  "whitening_enabled": false,
  "tx_power_dbm": 10,
  "registers": {
    "FREQ2": "0x10",
    "FREQ1": "0xB0",
    "FREQ0": "0x71",
    "MDMCFG4": "0xCA",
    "MDMCFG3": "0x83",
    "MDMCFG2": "0x13",
    "MDMCFG1": "0x22",
    "MDMCFG0": "0xF8",
    "DEVIATN": "0x34",
    "FSCAL2": "0x2A",
    "SYNC1": "0xD3",
    "SYNC0": "0x91",
    "PKTLEN": "0x3C",
    "PKTCTRL1": "0x04",
    "PKTCTRL0": "0x45",
    "PA_TABLE0": "0xC0",
    "FREND0": "0x10",
    "FREND1": "0x56"
  }
}
```

---

## Appendix B: Quick Reference - All Config Files

### Core Configs (81 files)

```
tests/etc/
├── 315-ook-low-1k2.json
├── 315-ook-low-2k4.json
├── 315-ook-low-4k8.json
├── 315-ook-fast-9k6.json
├── 315-ook-fast-19k2.json
├── 315-2fsk-sync-2k4.json
├── 315-2fsk-sync-4k8.json
├── 315-2fsk-sync-9k6.json
├── 315-2fsk-sync-4k8-fec.json
├── 433-ook-keyfob-1k2.json
├── 433-ook-keyfob-2k4.json
├── 433-ook-keyfob-4k8.json
├── 433-ook-pwm-2k4.json
├── 433-ook-pwm-4k8.json
├── 433-ook-manch-4k8.json
├── 433-ook-manch-9k6.json
├── 433-2fsk-std-4k8.json
├── 433-2fsk-std-9k6.json
├── 433-2fsk-std-4k8-fec.json
├── 433-2fsk-fast-38k4.json
├── 433-2fsk-fast-76k8.json
├── 433-2fsk-fast-100k.json
├── 433-gfsk-crc-9k6.json
├── 433-gfsk-crc-19k2.json
├── 433-gfsk-crc-38k4.json
├── 433-gfsk-crc-19k2-fec.json
├── 433-4fsk-50k.json
├── 433-4fsk-100k.json
├── 433-4fsk-200k.json
├── 868-ook-simple-1k2.json
├── 868-ook-simple-4k8.json
├── 868-ook-simple-9k6.json
├── 868-2fsk-manch-4k8.json
├── 868-2fsk-manch-9k6.json
├── 868-2fsk-manch-19k2.json
├── 868-2fsk-fast-38k4.json
├── 868-2fsk-fast-76k8.json
├── 868-2fsk-fast-100k.json
├── 868-gfsk-smart-9k6.json
├── 868-gfsk-smart-19k2.json
├── 868-gfsk-smart-38k4.json
├── 868-gfsk-fec-19k2.json
├── 868-gfsk-fec-38k4.json
├── 868-gfsk-fec-19k2-white.json
├── 915-ook-tpms-4k8-nosync.json
├── 915-ook-tpms-9k6-nosync.json
├── 915-ook-tpms-19k2-nosync.json
├── 915-ook-tpms-9k6-sync.json
├── 915-2fsk-sensor-9k6.json
├── 915-2fsk-sensor-19k2.json
├── 915-2fsk-sensor-38k4.json
├── 915-gfsk-std-38k4.json
├── 915-gfsk-std-38k4-white.json
├── 915-gfsk-crc-fec-38k4.json
├── 915-gfsk-crc-fec-76k8.json
├── 915-gfsk-crc-fec-100k.json
├── 915-fhss-100k-master.json
├── 915-fhss-100k-slave.json
├── 915-fhss-250k-master.json
├── 915-fhss-250k-slave.json
├── 915-2fsk-max-250k.json
├── 915-2fsk-max-500k.json
├── 315-2fsk-longrange-1k2.json
├── 433-2fsk-longrange-1k2.json
├── 868-2fsk-longrange-1k2.json
├── 915-2fsk-longrange-1k2.json
├── 315-2fsk-longrange-2k4.json
├── 433-2fsk-longrange-2k4.json
├── 433-2fsk-highspeed-500k.json
├── 868-2fsk-highspeed-500k.json
├── 915-2fsk-highspeed-500k.json
├── 433-fhss-50k-master.json
├── 433-fhss-50k-slave.json
├── 433-fhss-100k-master.json
├── 433-fhss-100k-slave.json
├── 433-gfsk-aes-ecb.json
├── 433-gfsk-aes-cbc.json
├── 915-gfsk-aes-ctr.json
├── 915-gfsk-aes-cbc.json
├── spectrum-monitor-315.json
├── spectrum-monitor-433.json
├── spectrum-monitor-868.json
└── spectrum-monitor-915.json
```

### Encoding Variant Configs (12 files)

```
tests/etc/
├── 433-2fsk-std-9k6-manch.json
├── 433-gfsk-crc-19k2-manch.json
├── 915-2fsk-sensor-19k2-manch.json
├── 915-gfsk-std-38k4-manch.json
├── 433-2fsk-std-9k6-white.json
├── 433-gfsk-crc-19k2-white.json
├── 915-2fsk-sensor-19k2-white.json
├── 915-4fsk-100k-white.json
├── 433-2fsk-std-9k6-fec.json
├── 433-gfsk-crc-19k2-fec.json
├── 915-2fsk-sensor-19k2-fec.json
└── 868-2fsk-manch-9k6-fec.json
```

### Packet Format Configs (12 files)

```
tests/etc/
├── 433-gfsk-fixed-16.json
├── 433-gfsk-fixed-64.json
├── 433-gfsk-fixed-255.json
├── 433-gfsk-fixed-512.json
├── 433-gfsk-var-16.json
├── 433-gfsk-var-64.json
├── 433-gfsk-var-255.json
├── 433-gfsk-sync-none.json
├── 433-gfsk-sync-15of16.json
├── 433-gfsk-sync-16of16.json
├── 433-gfsk-sync-30of32.json
└── 433-gfsk-sync-carrier.json
```

### Advanced Feature Configs (23 files)

```
tests/etc/
├── 433-gfsk-cca-mode0.json
├── 433-gfsk-cca-mode1.json
├── 433-gfsk-cca-mode2.json
├── 433-gfsk-cca-mode3.json
├── 433-gfsk-addr-match.json
├── 433-gfsk-addr-mismatch-tx.json
├── 433-gfsk-addr-mismatch-rx.json
├── 433-gfsk-addr-broadcast.json
├── 433-gfsk-preamble-2.json
├── 433-gfsk-preamble-4.json
├── 433-gfsk-preamble-8.json
├── 433-gfsk-preamble-24.json
├── 433-gfsk-power-min.json
├── 433-gfsk-power-low.json
└── 433-gfsk-power-max.json
```

---

## Appendix C: Test Execution Checklist

Use this checklist to track test execution progress:

### Phase 1: Infrastructure Setup
- [ ] Create `tests/etc/` directory
- [ ] Create `tests/data/` directory with test payloads
- [ ] Create `tests/results/` directory
- [ ] Create `tests/scripts/` directory
- [ ] Implement `run-loopback-test.sh`
- [ ] Implement `run-config-validation.sh`
- [ ] Verify 2+ YS1 devices available

### Phase 2: Core Profile Tests
- [ ] Complete all 315 MHz tests (9 tests)
- [ ] Complete all 433 MHz tests (21 tests)
- [ ] Complete all 868 MHz tests (15 tests)
- [ ] Complete all 915 MHz tests (18 tests)

### Phase 3: Special Tests
- [ ] Complete multi-band special tests (18 tests)
- [ ] Complete encoding variation tests (12 tests)
- [ ] Complete packet format tests (12 tests)
- [ ] Complete advanced feature tests (15 tests)

### Phase 4: Integration Tests
- [ ] Complete integration tests (8 tests)

### Phase 5: Documentation
- [ ] Update test results summary
- [ ] Document any failures or anomalies
- [ ] Update revision history
