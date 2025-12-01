# rfcat Firmware Features Analysis

This document analyzes the capabilities of the rfcat firmware for YardStick One, beyond the spectrum analyzer functionality that has already been implemented in gocat.

## Overview

The rfcat firmware for the CC1111 chip provides a comprehensive set of RF transceiver capabilities. The firmware is organized around several "applications" (APP_*) that handle different functionality, with commands sent over USB.

## Already Implemented in gocat

- **Spectrum Analyzer (SPECAN)** - Firmware-based RF spectrum analysis
  - `RFCAT_START_SPECAN` (0x40) - Start spectrum analyzer mode
  - `RFCAT_STOP_SPECAN` (0x41) - Stop spectrum analyzer mode
  - `APP_SPECAN` (0x43) with `SPECAN_QUEUE` (0x01) for receiving RSSI data

## Additional Firmware Capabilities

### 1. Hardware AES Encryption/Decryption

The CC1111 has a built-in AES crypto co-processor that the firmware exposes:

**Commands (via APP_NIC = 0x42):**
- `NIC_SET_AES_MODE` (0x06) - Set AES encryption mode
- `NIC_GET_AES_MODE` (0x07) - Get current AES mode
- `NIC_SET_AES_IV` (0x08) - Set 128-bit initialization vector
- `NIC_SET_AES_KEY` (0x09) - Set 128-bit encryption key

**Supported AES Modes:**
- ECB (Electronic Codebook)
- CBC (Cipher Block Chaining)
- CBC-MAC (Message Authentication Code)

**Features:**
- Hardware DMA-accelerated encryption/decryption
- Transparent encryption of TX data
- Automatic decryption of RX data
- In-band crypto mode configuration

**Use Cases:**
- Encrypted RF communication between devices
- Secure key fob protocols
- Authentication systems

### 2. Frequency Hopping Spread Spectrum (FHSS)

Full FHSS MAC layer implementation for frequency-agile communications:

**Commands:**
- `FHSS_SET_CHANNELS` (0x10) - Define hop sequence (up to 880 channels)
- `FHSS_GET_CHANNELS` (0x18) - Get current channel list
- `FHSS_NEXT_CHANNEL` (0x11) - Manually hop to next channel
- `FHSS_CHANGE_CHANNEL` (0x12) - Jump to specific channel
- `FHSS_START_HOPPING` (0x23) - Begin automatic hopping
- `FHSS_STOP_HOPPING` (0x24) - Stop automatic hopping
- `FHSS_XMIT` (0x17) - Transmit during FHSS operation
- `FHSS_SET_STATE` (0x20) / `FHSS_GET_STATE` (0x21) - MAC state control
- `FHSS_START_SYNC` (0x22) - Synchronize to a hopping network
- `FHSS_SET_MAC_THRESHOLD` (0x13) - Set timing threshold
- `FHSS_SET_MAC_DATA` (0x15) / `FHSS_GET_MAC_DATA` (0x16) - Raw MAC data

**MAC States:**
- `MAC_STATE_NONHOPPING` (0) - Standard non-hopping mode
- `MAC_STATE_DISCOVERY` (1) - Searching for networks
- `MAC_STATE_SYNCHING` (2) - Synchronizing to a master
- `MAC_STATE_SYNCHED` (3) - Synchronized and hopping
- `MAC_STATE_SYNC_MASTER` (4) - Operating as sync master
- `MAC_STATE_SYNCINGMASTER` (5) - Actively beaconing as master

**Features:**
- Timer-driven automatic channel hopping (T2 interrupt)
- Configurable dwell time per channel
- Network discovery and synchronization
- Master/slave operation modes
- Beacon transmission for sync

**Use Cases:**
- Analyzing proprietary FHSS protocols
- Building FHSS communication systems
- Reverse engineering spread spectrum devices

### 3. Long/Infinite Packet Transmission

Support for transmitting packets larger than the 255-byte hardware limit:

**Commands:**
- `NIC_LONG_XMIT` (0x0c) - Start long transmission
- `NIC_LONG_XMIT_MORE` (0x0d) - Continue sending chunks

**Features:**
- Multi-buffer queue (2 buffers × 240 bytes each)
- Automatic buffer management during TX
- Infinite mode for continuous transmission
- USB streaming to RF with flow control

**Use Cases:**
- Transmitting large payloads
- Continuous wave/jamming (research)
- Protocol testing with long packets

### 4. Large Packet Reception

Receive packets larger than standard buffer size:

**Command:**
- `NIC_SET_RECV_LARGE` (0x05) - Configure large block receive

**Features:**
- Configurable receive buffer size
- Infinite RX mode for continuous reception
- Automatic packet reassembly

### 5. External Amplifier Control (YardStick One)

Control the YardStick One's external TX/RX amplifiers:

**Commands:**
- `NIC_SET_AMP_MODE` (0x0a) - Enable/disable amplifier
- `NIC_GET_AMP_MODE` (0x0b) - Get amplifier state

**Hardware Pins (YS1):**
- `TX_AMP_EN` (P2_0) - TX amplifier enable
- `RX_AMP_EN` (P2_4) - RX amplifier enable
- `AMP_BYPASS_EN` (P2_3) - Amplifier bypass

### 6. Direct Register Access

Low-level hardware control via USB EP0:

**Commands:**
- `EP0_CMD_PEEKX` (0x02) - Read arbitrary memory/SFR
- `EP0_CMD_POKEX` (0x01) - Write arbitrary memory/SFR

**Features:**
- Read/write any CC1111 register
- Direct radio configuration
- Debug and development access

### 7. System Commands

**Via EP0:**
- `EP0_CMD_GET_DEBUG_CODES` (0x00) - Get debug status
- `EP0_CMD_RESET` (0xfe) - Trigger device reset via watchdog
- `EP0_CMD_PING0/PING1` (0x03/0x04) - Connectivity test

**Via APP_SYSTEM:**
- `CMD_PEEK` (0x80) - Memory peek
- `CMD_POKE` (0x81) - Memory poke
- `CMD_PING` (0x82) - Ping
- `CMD_STATUS` (0x83) - Get status
- `CMD_RFMODE` (0x88) - Set RF mode (RX/TX/IDLE)
- `CMD_BUILDTYPE` (0x86) - Get firmware build info
- `CMD_BOOTLOADER` (0x87) - Enter bootloader
- `CMD_PARTNUM` (0x8e) - Get chip part number
- `CMD_RESET` (0x8f) - Software reset
- `CMD_LEDMODE` (0x93) - Set LED behavior

### 8. Radio Configuration

The firmware provides extensive radio configuration through direct register access:

**Modulation:**
- 2-FSK, GFSK, ASK/OOK, MSK
- Manchester encoding support
- FEC (Forward Error Correction)
- Data whitening

**Packet Handling:**
- Variable and fixed length packets
- Configurable preamble (2, 3, 4, 6, 8, 12, 16, 24 bytes)
- Sync word configuration (16-bit, 32-bit, carrier sense)
- CRC generation/checking
- Address filtering
- PQT (Preamble Quality Threshold)

**RF Parameters:**
- Frequency: 300-348 MHz, 391-464 MHz, 782-928 MHz
- Data rate: 0.6 - 500 kbaud
- Channel bandwidth: configurable
- Deviation: configurable for FSK
- Output power: multiple levels

### 9. Repeater Mode

Receive-and-retransmit functionality:

**Functions:**
- `RepeaterStart()` - Begin repeater mode
- `RepeaterStop()` - End repeater mode

**Features:**
- Dual DMA channel operation (RX→TX)
- Automatic retransmission of received packets

## Implementation Priority for gocat

### High Priority (Most Useful)

1. **AES Encryption** - Enables secure communications and analyzing encrypted protocols
2. **FHSS Support** - Essential for analyzing spread spectrum systems
3. **External Amplifier Control** - Already partially implemented, needs completion

### Medium Priority

4. **Long Packet TX/RX** - Useful for certain protocols
5. **Repeater Mode** - Niche but useful for range extension testing

### Lower Priority (Already Accessible via Register Poke)

6. **Direct Register Access** - Already possible with existing peek/poke
7. **Additional System Commands** - Mostly debugging utilities

## Protocol Reference

### USB Endpoints
- **EP0**: Control transfers, system commands, peek/poke
- **EP5 IN**: Data from device (RX packets, SPECAN data)
- **EP5 OUT**: Data to device (TX packets, commands)

### Message Format
```
EP5 OUT: [APP][CMD][DATA...]
EP5 IN:  [APP][CMD][LEN_LOW][LEN_HIGH][DATA...]
```

### Application IDs
- `APP_NIC` (0x42) - Primary radio application
- `APP_SPECAN` (0x43) - Spectrum analyzer data
- `APP_SYSTEM` (0x00) - System commands

## Source Files Reference

- `firmware/appFHSSNIC.c` - Main application, FHSS, SPECAN
- `firmware/cc1111rf.c` - RF driver, TX/RX handling
- `firmware/cc1111_aes.c` - AES encryption support
- `firmware/include/FHSS.h` - FHSS/SPECAN constants
- `firmware/include/nic.h` - NIC command constants
- `firmware/include/global.h` - Global definitions, AES modes
- `rflib/chipcon_nic.py` - Python API reference
