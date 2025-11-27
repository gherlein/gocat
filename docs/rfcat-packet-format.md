# RFCat Packet Format Specification

This document describes the USB communication protocol used by RFCat to communicate with YardStick One (YS1) and other compatible dongles (CC1111-based devices). This specification is reverse-engineered from the RFCat Python library and firmware source code.

## Table of Contents

1. [Overview](#overview)
2. [USB Configuration](#usb-configuration)
3. [Packet Structure](#packet-structure)
4. [Application IDs](#application-ids)
5. [System Commands (APP_SYSTEM)](#system-commands-app_system)
6. [NIC Commands (APP_NIC)](#nic-commands-app_nic)
7. [FHSS Commands](#fhss-commands)
8. [Spectrum Analyzer Commands](#spectrum-analyzer-commands)
9. [EP0 Vendor Commands](#ep0-vendor-commands)
10. [Radio Configuration Registers](#radio-configuration-registers)
11. [Error Codes](#error-codes)
12. [Example Transactions](#example-transactions)

---

## Overview

RFCat uses USB bulk transfers on Endpoint 5 (EP5) for the primary command/response protocol. Control transfers on Endpoint 0 (EP0) are used for low-level operations like peek/poke memory access.

The communication model is request-response: the host sends a command packet, and the device responds with a response packet containing the same application ID and command byte.

---

## USB Configuration

### USB Identifiers

| Device           | Vendor ID | Product ID |
|------------------|-----------|------------|
| YardStick One    | 0x1D50    | 0x605B     |
| Dons Dongle      | 0x1D50    | 0x6048     |
| Chronos Dongle   | 0x1D50    | 0x6047     |
| SRF Stick        | 0x1D50    | 0xECC1     |
| Legacy TI        | 0x0451    | 0x4715     |

### Bootloader Mode IDs

| Device Type      | Vendor ID | Product ID |
|------------------|-----------|------------|
| Bootloader       | 0x1D50    | 0x6049     |
| Bootloader Alt   | 0x1D50    | 0x604A     |
| Bootloader Alt2  | 0x1D50    | 0xECC0     |

### Endpoint Configuration

| Endpoint | Direction | Type | Max Packet Size | Buffer Size |
|----------|-----------|------|-----------------|-------------|
| EP0      | IN/OUT    | Control | 32 bytes    | 32 bytes    |
| EP5      | IN        | Bulk    | 64 bytes    | N/A         |
| EP5      | OUT       | Bulk    | 64 bytes    | 516 bytes   |

### Timeouts

| Operation | Default Timeout (ms) |
|-----------|---------------------|
| USB Default | 1000              |
| RX Wait     | 1000              |
| TX Wait     | 10000             |
| EP Idle     | 400               |
| EP Active   | 10                |

---

## Packet Structure

### Host-to-Device (EP5 OUT)

Commands sent from host to device use the following format:

```
Offset  Size    Field       Description
------  ----    -----       -----------
0       1       app         Application ID (see Application IDs)
1       1       cmd         Command byte (application-specific)
2       2       length      Payload length (little-endian, uint16)
4       N       payload     Command-specific data (N = length)
```

**Total packet size:** 4 + length bytes

**Example:** Ping command
```
Bytes: [0xFF] [0x82] [0x04, 0x00] [0x41, 0x42, 0x43, 0x44]
        app    cmd    length=4     payload="ABCD"
```

### Device-to-Host (EP5 IN)

Responses from device to host use the following format:

```
Offset  Size    Field       Description
------  ----    -----       -----------
0       1       marker      Always 0x40 ('@')
1       1       app         Application ID (echoed from request)
2       1       cmd         Command byte (echoed from request)
3       2       length      Payload length (little-endian, uint16)
5       N       payload     Response data (N = length)
```

**Total packet size:** 5 + length bytes

**Example:** Ping response
```
Bytes: [0x40] [0xFF] [0x82] [0x04, 0x00] [0x41, 0x42, 0x43, 0x44]
        '@'    app    cmd    length=4     payload="ABCD"
```

### Multi-Packet Messages

For messages larger than 64 bytes (EP5_MAX_PACKET_SIZE):

1. First packet contains the header (marker + app + cmd + length)
2. Subsequent packets contain only payload data
3. The `length` field indicates total payload size across all packets
4. Maximum total message size: 516 bytes (EP5OUT_BUFFER_SIZE)

---

## Application IDs

| Name        | Value | Description                              |
|-------------|-------|------------------------------------------|
| APP_GENERIC | 0x01  | Generic application (reserved)           |
| APP_NIC     | 0x42  | Radio NIC operations (TX/RX)             |
| APP_SPECAN  | 0x43  | Spectrum analyzer                        |
| APP_DEBUG   | 0xFE  | Debug output from firmware               |
| APP_SYSTEM  | 0xFF  | System/administrative commands           |

---

## System Commands (APP_SYSTEM)

Application ID: `0xFF`

### SYS_CMD_PING (0x82)

Echo back the payload data. Used to verify communication.

**Request:**
- Payload: Arbitrary data to echo

**Response:**
- Payload: Same data echoed back

### SYS_CMD_PEEK (0x80)

Read memory from device.

**Request:**
```
Offset  Size    Field       Description
------  ----    -----       -----------
0       2       bytecount   Number of bytes to read (little-endian)
2       2       address     Memory address to read from (little-endian)
```

**Response:**
- Payload: `bytecount` bytes of memory data

### SYS_CMD_POKE (0x81)

Write data to memory.

**Request:**
```
Offset  Size    Field       Description
------  ----    -----       -----------
0       2       address     Memory address to write to (little-endian)
2       N       data        Bytes to write
```

**Response:**
- Payload: 2 bytes (bytes left, should be 0 on success)

### SYS_CMD_POKE_REG (0x84)

Write data to a register (similar to POKE but with register semantics).

**Request:**
```
Offset  Size    Field       Description
------  ----    -----       -----------
0       2       address     Register address (little-endian)
2       N       data        Bytes to write
```

**Response:**
- Payload: 2 bytes (bytes left)

### SYS_CMD_STATUS (0x83)

Get device status.

**Request:**
- Payload: Empty

**Response:**
- Payload: Status string (implementation-specific)

### SYS_CMD_GET_CLOCK (0x85)

Get current clock value.

**Request:**
- Payload: Empty

**Response:**
- Payload: 4 bytes clock value (little-endian uint32)

### SYS_CMD_BUILDTYPE (0x86)

Get firmware build information.

**Request:**
- Payload: Empty

**Response:**
- Payload: Null-terminated build string (e.g., "YARDSTICKONE r0001\0")

### SYS_CMD_BOOTLOADER (0x87)

Enter bootloader mode for firmware updates.

**Request:**
- Payload: Empty

**Response:**
- Payload: Echoed (device will reset into bootloader)

### SYS_CMD_RFMODE (0x88)

Set radio mode (RX/TX/IDLE).

**Request:**
```
Offset  Size    Field       Description
------  ----    -----       -----------
0       1       mode        Radio mode (see RFST values)
```

**RFST Mode Values:**
| Name        | Value | Description            |
|-------------|-------|------------------------|
| RFST_SFSTXON| 0x00  | Enable and calibrate   |
| RFST_SCAL   | 0x01  | Calibrate              |
| RFST_SRX    | 0x02  | Enable RX              |
| RFST_STX    | 0x03  | Enable TX              |
| RFST_SIDLE  | 0x04  | Idle mode              |
| RFST_SNOP   | 0x05  | No operation           |

**Response:**
- Payload: Echoed data

### SYS_CMD_COMPILER (0x89)

Get compiler information.

**Request:**
- Payload: Empty

**Response:**
- Payload: Compiler version string (e.g., "SDCCvXXX")

### SYS_CMD_PARTNUM (0x8E)

Get chip part number.

**Request:**
- Payload: Empty

**Response:**
- Payload: 1 byte part number

**Part Number Values:**
| Value | Chip    |
|-------|---------|
| 0x01  | CC1110  |
| 0x11  | CC1111  |
| 0x81  | CC2510  |
| 0x91  | CC2511  |

### SYS_CMD_RESET (0x8F)

Reset the device.

**Request:**
- Payload: "RESET_NOW\0" (must match exactly)

**Response:**
- Payload: Echoed (device will reset)

### SYS_CMD_CLEAR_CODES (0x90)

Clear debug/error codes.

**Request:**
- Payload: 2 bytes (any value)

**Response:**
- Payload: 2 bytes

### SYS_CMD_DEVICE_SERIAL_NUMBER (0x91)

Get device serial number.

**Request:**
- Payload: Empty

**Response:**
- Payload: 16 bytes serial number

### SYS_CMD_LED_MODE (0x93)

Set LED mode.

**Request:**
```
Offset  Size    Field       Description
------  ----    -----       -----------
0       1       mode        0x00 = OFF, 0x01 = ON
```

**Response:**
- Payload: Echoed

---

## NIC Commands (APP_NIC)

Application ID: `0x42`

### NIC_RECV (0x01)

Receive RF data. This is the response command for received packets.

**Request:**
- Not sent by host; this is a device-initiated response

**Response (device sends when data received):**
- Payload: Received RF data

### NIC_XMIT (0x02)

Transmit RF data.

**Request:**
```
Offset  Size    Field       Description
------  ----    -----       -----------
0       2       data_len    Length of RF data (little-endian)
2       2       repeat      Number of times to repeat (0 = once, 65535 = forever)
4       2       offset      Offset within data for repeat
6       N       data        RF data to transmit
```

**Response:**
- Payload: Empty on success

**Maximum data length:** 255 bytes for standard transmit

### NIC_SET_ID (0x03)

Set network/device ID.

**Request:**
- Payload: ID data (implementation-specific)

**Response:**
- Payload: Confirmation

### NIC_SET_RECV_LARGE (0x05)

Configure large packet receive mode.

**Request:**
```
Offset  Size    Field       Description
------  ----    -----       -----------
0       2       blocksize   Max receive block size (little-endian, max 512)
```

**Response:**
- Payload: Confirmation

### NIC_SET_AES_MODE (0x06)

Configure AES crypto mode.

**Request:**
```
Offset  Size    Field       Description
------  ----    -----       -----------
0       1       aesmode     AES mode bitfield
```

**AES Mode Bitfield:**
```
Bits 7:4 - CC1111 AES mode (ENCCS_MODE_*)
Bit 3    - Outbound ON (1) / OFF (0)
Bit 2    - Outbound Encrypt (1) / Decrypt (0)
Bit 1    - Inbound ON (1) / OFF (0)
Bit 0    - Inbound Encrypt (1) / Decrypt (0)
```

**ENCCS_MODE Values:**
| Name             | Value | Description                    |
|------------------|-------|--------------------------------|
| ENCCS_MODE_CBC   | 0x00  | Cipher Block Chaining          |
| ENCCS_MODE_CFB   | 0x10  | Cipher Feedback                |
| ENCCS_MODE_OFB   | 0x20  | Output Feedback                |
| ENCCS_MODE_CTR   | 0x30  | Counter                        |
| ENCCS_MODE_ECB   | 0x40  | Electronic Codebook            |
| ENCCS_MODE_CBCMAC| 0x50  | CBC Message Authentication Code|

**Response:**
- Payload: Confirmation

### NIC_GET_AES_MODE (0x07)

Get current AES crypto mode.

**Request:**
- Payload: Empty

**Response:**
- Payload: 1 byte AES mode

### NIC_SET_AES_IV (0x08)

Set AES initialization vector.

**Request:**
- Payload: 16 bytes IV

**Response:**
- Payload: Confirmation

### NIC_SET_AES_KEY (0x09)

Set AES encryption key.

**Request:**
- Payload: 16 bytes key

**Response:**
- Payload: Confirmation

### NIC_SET_AMP_MODE (0x0A)

Set external amplifier mode.

**Request:**
```
Offset  Size    Field       Description
------  ----    -----       -----------
0       1       mode        Amplifier mode
```

**Response:**
- Payload: Confirmation

### NIC_GET_AMP_MODE (0x0B)

Get current amplifier mode.

**Request:**
- Payload: Empty

**Response:**
- Payload: 1 byte amplifier mode

### NIC_LONG_XMIT (0x0C)

Start long (>255 byte) transmission.

**Request:**
```
Offset  Size    Field       Description
------  ----    -----       -----------
0       2       total_len   Total data length (little-endian, max 65535)
2       1       preload     Number of chunks to preload
3       N       data        First chunk(s) of data (up to preload * 240 bytes)
```

**Response:**
- Payload: 1 byte error code (0 = success)

### NIC_LONG_XMIT_MORE (0x0D)

Continue long transmission with more data.

**Request:**
```
Offset  Size    Field       Description
------  ----    -----       -----------
0       1       chunk_len   Length of this chunk (0 = finished)
1       N       data        Chunk data
```

**Response:**
- Payload: 1 byte error code (0 = success, 0xFE = retry)

---

## FHSS Commands

Application ID: `0x42` (same as NIC)

These commands control Frequency Hopping Spread Spectrum functionality.

### FHSS_SET_CHANNELS (0x10)

Set the channel hop list.

**Request:**
- Payload: Array of channel numbers

### FHSS_NEXT_CHANNEL (0x11)

Advance to next channel in hop sequence.

### FHSS_CHANGE_CHANNEL (0x12)

Change to a specific channel.

**Request:**
- Payload: Channel number

### FHSS_SET_MAC_THRESHOLD (0x13)

Set MAC threshold value.

### FHSS_GET_MAC_THRESHOLD (0x14)

Get current MAC threshold.

### FHSS_SET_MAC_DATA (0x15)

Set MAC layer data.

### FHSS_GET_MAC_DATA (0x16)

Get MAC layer data.

### FHSS_XMIT (0x17)

Transmit using FHSS.

### FHSS_GET_CHANNELS (0x18)

Get current channel hop list.

### FHSS_SET_STATE (0x20)

Set FHSS state machine state.

**FHSS States:**
| Name                    | Value | Description              |
|-------------------------|-------|--------------------------|
| FHSS_STATE_NONHOPPING   | 0x00  | Not hopping              |
| FHSS_STATE_DISCOVERY    | 0x01  | Discovering              |
| FHSS_STATE_SYNCHING     | 0x02  | Synchronizing            |
| FHSS_STATE_SYNCHED      | 0x03  | Synchronized             |
| FHSS_STATE_SYNC_MASTER  | 0x04  | Sync master              |
| FHSS_STATE_SYNCINGMASTER| 0x05  | Syncing to master        |

### FHSS_GET_STATE (0x21)

Get current FHSS state.

### FHSS_START_SYNC (0x22)

Start synchronization.

### FHSS_START_HOPPING (0x23)

Start frequency hopping.

### FHSS_STOP_HOPPING (0x24)

Stop frequency hopping.

---

## Spectrum Analyzer Commands

Application ID: `0x43`

### SPECAN_QUEUE (0x01)

Queue spectrum data. Device sends this to host with spectrum data.

**Response:**
- Payload: Spectrum data points

### RFCAT_START_SPECAN (0x40)

Start spectrum analyzer mode (sent via NIC app).

### RFCAT_STOP_SPECAN (0x41)

Stop spectrum analyzer mode (sent via NIC app).

---

## EP0 Vendor Commands

These commands use USB control transfers on EP0 with vendor request type.

### EP0_CMD_GET_DEBUG_CODES (0x00)

Get last debug/error codes.

**Request Type:** IN
**Response:** 2 bytes [lastCode[0], lastCode[1]]

### EP0_CMD_PEEKX (0x02)

Read from XDATA memory.

**Request Type:** IN
**wValue:** Address to read
**wLength:** Number of bytes to read
**Response:** Memory contents

### EP0_CMD_POKEX (0x01)

Write to XDATA memory.

**Request Type:** OUT
**wValue:** Address to write
**Data:** Bytes to write

### EP0_CMD_PING0 (0x03)

Ping (echo request packet).

### EP0_CMD_PING1 (0x04)

Ping (echo EP0 OUT buffer).

### EP0_CMD_RESET (0xFE)

Reset device.

**wValue:** 0x5352 ('SR')
**wIndex:** 0x4E54 ('NT')

---

## Radio Configuration Registers

The CC1111 radio configuration is stored at address `0xDF00`. Reading 0x3E (62) bytes from this address retrieves the full radio configuration.

### Register Map (0xDF00 - 0xDF3D)

| Offset | Address | Register   | Description                    |
|--------|---------|------------|--------------------------------|
| 0x00   | 0xDF00  | SYNC1      | Sync word, high byte           |
| 0x01   | 0xDF01  | SYNC0      | Sync word, low byte            |
| 0x02   | 0xDF02  | PKTLEN     | Packet length                  |
| 0x03   | 0xDF03  | PKTCTRL1   | Packet automation control 1    |
| 0x04   | 0xDF04  | PKTCTRL0   | Packet automation control 0    |
| 0x05   | 0xDF05  | ADDR       | Device address                 |
| 0x06   | 0xDF06  | CHANNR     | Channel number                 |
| 0x07   | 0xDF07  | FSCTRL1    | Frequency synthesizer control 1|
| 0x08   | 0xDF08  | FSCTRL0    | Frequency synthesizer control 0|
| 0x09   | 0xDF09  | FREQ2      | Frequency control word, high   |
| 0x0A   | 0xDF0A  | FREQ1      | Frequency control word, mid    |
| 0x0B   | 0xDF0B  | FREQ0      | Frequency control word, low    |
| 0x0C   | 0xDF0C  | MDMCFG4    | Modem configuration 4          |
| 0x0D   | 0xDF0D  | MDMCFG3    | Modem configuration 3          |
| 0x0E   | 0xDF0E  | MDMCFG2    | Modem configuration 2          |
| 0x0F   | 0xDF0F  | MDMCFG1    | Modem configuration 1          |
| 0x10   | 0xDF10  | MDMCFG0    | Modem configuration 0          |
| 0x11   | 0xDF11  | DEVIATN    | Modem deviation setting        |
| 0x12   | 0xDF12  | MCSM2      | Main radio control state machine 2 |
| 0x13   | 0xDF13  | MCSM1      | Main radio control state machine 1 |
| 0x14   | 0xDF14  | MCSM0      | Main radio control state machine 0 |
| 0x15   | 0xDF15  | FOCCFG     | Frequency offset compensation  |
| 0x16   | 0xDF16  | BSCFG      | Bit synchronization config     |
| 0x17   | 0xDF17  | AGCCTRL2   | AGC control 2                  |
| 0x18   | 0xDF18  | AGCCTRL1   | AGC control 1                  |
| 0x19   | 0xDF19  | AGCCTRL0   | AGC control 0                  |
| 0x1A   | 0xDF1A  | FREND1     | Front end RX configuration     |
| 0x1B   | 0xDF1B  | FREND0     | Front end TX configuration     |
| 0x1C   | 0xDF1C  | FSCAL3     | Frequency synthesizer cal 3    |
| 0x1D   | 0xDF1D  | FSCAL2     | Frequency synthesizer cal 2    |
| 0x1E   | 0xDF1E  | FSCAL1     | Frequency synthesizer cal 1    |
| 0x1F   | 0xDF1F  | FSCAL0     | Frequency synthesizer cal 0    |
| 0x23   | 0xDF23  | TEST2      | Test register 2                |
| 0x24   | 0xDF24  | TEST1      | Test register 1                |
| 0x25   | 0xDF25  | TEST0      | Test register 0                |
| 0x27   | 0xDF27  | PA_TABLE7  | Power amplifier table 7        |
| 0x28   | 0xDF28  | PA_TABLE6  | Power amplifier table 6        |
| 0x29   | 0xDF29  | PA_TABLE5  | Power amplifier table 5        |
| 0x2A   | 0xDF2A  | PA_TABLE4  | Power amplifier table 4        |
| 0x2B   | 0xDF2B  | PA_TABLE3  | Power amplifier table 3        |
| 0x2C   | 0xDF2C  | PA_TABLE2  | Power amplifier table 2        |
| 0x2D   | 0xDF2D  | PA_TABLE1  | Power amplifier table 1        |
| 0x2E   | 0xDF2E  | PA_TABLE0  | Power amplifier table 0        |
| 0x2F   | 0xDF2F  | IOCFG2     | GPIO configuration 2           |
| 0x30   | 0xDF30  | IOCFG1     | GPIO configuration 1           |
| 0x31   | 0xDF31  | IOCFG0     | GPIO configuration 0           |
| 0x36   | 0xDF36  | PARTNUM    | Part number (read-only)        |
| 0x37   | 0xDF37  | CHIPID     | Chip ID (read-only)            |
| 0x38   | 0xDF38  | FREQEST    | Frequency offset estimate      |
| 0x39   | 0xDF39  | LQI        | Link quality indicator         |
| 0x3A   | 0xDF3A  | RSSI       | Received signal strength       |
| 0x3B   | 0xDF3B  | MARCSTATE  | Main radio control state       |

### Frequency Calculation

The carrier frequency is calculated as:

```
frequency = (FREQ2 << 16 | FREQ1 << 8 | FREQ0) * (crystal_freq / 65536)
```

Where `crystal_freq` is typically 26 MHz for CC2510/CC2511 and 24 MHz for CC1110/CC1111.

### Modulation Modes (MDMCFG2)

| Value | Mode     | Description                |
|-------|----------|----------------------------|
| 0x00  | MOD_2FSK | 2-FSK modulation           |
| 0x10  | MOD_GFSK | GFSK modulation            |
| 0x30  | MOD_ASK_OOK | ASK/OOK modulation      |
| 0x40  | MOD_4FSK | 4-FSK modulation           |
| 0x70  | MOD_MSK  | MSK modulation             |

Add 0x08 for Manchester encoding.

### Sync Modes (MDMCFG2)

| Value | Mode                    |
|-------|-------------------------|
| 0     | No sync                 |
| 1     | 15 of 16 bits match     |
| 2     | 16 of 16 bits match     |
| 3     | 30 of 32 bits match     |
| 4     | Carrier detect          |
| 5     | Carrier + 15/16 match   |
| 6     | Carrier + 16/16 match   |
| 7     | Carrier + 30/32 match   |

### MARCSTATE Values (0xDF3B)

| Value | State              |
|-------|--------------------|
| 0x00  | SLEEP              |
| 0x01  | IDLE               |
| 0x0D  | RX                 |
| 0x13  | TX                 |
| 0x11  | RX_OVERFLOW        |
| 0x16  | TX_UNDERFLOW       |

---

## Error Codes

### Firmware Error Codes (LCE_*)

| Name                              | Value | Description                    |
|-----------------------------------|-------|--------------------------------|
| LCE_NO_ERROR                      | 0x00  | No error                       |
| LCE_USB_EP5_TX_WHILE_INBUF_WRITTEN| 0x01  | TX while input buffer written  |
| LCE_USB_EP0_SENT_STALL            | 0x04  | EP0 sent stall                 |
| LCE_USB_EP5_OUT_WHILE_OUTBUF_WRITTEN| 0x05| EP5 OUT while buffer written   |
| LCE_USB_EP5_LEN_TOO_BIG           | 0x06  | EP5 packet too large           |
| LCE_USB_EP5_GOT_CRAP              | 0x07  | Invalid EP5 data               |
| LCE_USB_EP5_STALL                 | 0x08  | EP5 stalled                    |
| LCE_RF_RXOVF                      | 0x10  | RF RX overflow                 |
| LCE_RF_TXUNF                      | 0x11  | RF TX underflow                |

### Return Codes (RC_*)

| Name                              | Value | Description                    |
|-----------------------------------|-------|--------------------------------|
| RC_NO_ERROR                       | 0x00  | Success                        |
| RC_TX_DROPPED_PACKET              | 0xEC  | TX packet dropped              |
| RC_TX_ERROR                       | 0xED  | TX error                       |
| RC_RF_BLOCKSIZE_INCOMPAT          | 0xEE  | Block size incompatible        |
| RC_RF_MODE_INCOMPAT               | 0xEF  | RF mode incompatible           |
| RC_TEMP_ERR_BUFFER_NOT_AVAILABLE  | 0xFE  | Buffer not available (retry)   |
| RC_ERR_BUFFER_SIZE_EXCEEDED       | 0xFF  | Buffer size exceeded           |

---

## Example Transactions

### Example 1: Ping

**Host sends:**
```hex
FF 82 04 00 50 49 4E 47
```
- App: 0xFF (SYSTEM)
- Cmd: 0x82 (PING)
- Length: 0x0004
- Data: "PING"

**Device responds:**
```hex
40 FF 82 04 00 50 49 4E 47
```
- Marker: '@'
- App: 0xFF
- Cmd: 0x82
- Length: 0x0004
- Data: "PING"

### Example 2: Get Part Number

**Host sends:**
```hex
FF 8E 00 00
```
- App: 0xFF (SYSTEM)
- Cmd: 0x8E (PARTNUM)
- Length: 0x0000

**Device responds:**
```hex
40 FF 8E 01 00 11
```
- Part number: 0x11 (CC1111)

### Example 3: RF Transmit

**Host sends:**
```hex
42 02 0C 00 04 00 00 00 00 00 DE AD BE EF
```
- App: 0x42 (NIC)
- Cmd: 0x02 (XMIT)
- Length: 0x000C (12 bytes)
- Data length: 0x0004 (4 bytes RF data)
- Repeat: 0x0000 (once)
- Offset: 0x0000
- RF Data: DE AD BE EF

### Example 4: Set Frequency (via POKE)

To set frequency to 433 MHz (assuming 24 MHz crystal):
```
FREQ = 433000000 * 65536 / 24000000 = 0x10A762
FREQ2 = 0x10, FREQ1 = 0xA7, FREQ0 = 0x62
```

**Host sends:**
```hex
FF 81 05 00 09 DF 10 A7 62
```
- App: 0xFF (SYSTEM)
- Cmd: 0x81 (POKE)
- Length: 0x0005
- Address: 0xDF09 (FREQ2)
- Data: 0x10 0xA7 0x62

### Example 5: Receive RF Data

The host polls for received data by calling `recv()` with APP_NIC. When the device receives RF data, it sends:

**Device sends:**
```hex
40 42 01 04 00 DE AD BE EF
```
- Marker: '@'
- App: 0x42 (NIC)
- Cmd: 0x01 (RECV)
- Length: 0x0004
- Data: Received RF bytes

---

## Implementation Notes

1. **Byte Order:** All multi-byte integers are little-endian.

2. **Packet Parsing:** When receiving, look for the '@' (0x40) marker to find packet boundaries. The receive thread should accumulate data until a complete packet is received.

3. **Thread Safety:** The original rfcat uses separate threads for send and receive. Proper synchronization is required when accessing shared buffers.

4. **Timeouts:** Different operations require different timeouts. TX operations may need longer timeouts for large transmissions.

5. **Radio State:** Before changing radio configuration, ensure the radio is in IDLE state (MARCSTATE = 0x01).

6. **Buffer Sizes:**
   - Maximum EP5 OUT buffer: 516 bytes
   - Maximum standard RF TX block: 255 bytes
   - Maximum long TX block: 65535 bytes
   - Maximum RF RX block: 512 bytes
   - TX chunk size for long transmit: 240 bytes

7. **Response Matching:** Commands are matched by application ID and command byte. The response will echo these values.

---

## References

- RFCat Python library: `/external/rfcat/rflib/`
- RFCat firmware: `/external/rfcat/firmware/`
- CC1111 Datasheet (Texas Instruments)
- USB 2.0 Specification

---

*Document generated from reverse engineering of RFCat source code.*
