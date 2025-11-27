# RfCat Functionality Documentation

## Overview

RfCat is a comprehensive software-defined radio (SDR) framework designed for sub-GHz wireless communication research and security testing. It provides a Python-based interface to interact with Texas Instruments CC1111, CC2511, CC1110, and CC2510 chipsets, which are low-power RF transceiver chips. The framework enables researchers and security professionals to analyze, transmit, and receive radio frequency signals across multiple ISM bands.

**Primary Goal**: Reduce the time required for security researchers to create tools for analyzing unknown wireless targets and aid in hardware reverse-engineering.

## Supported Hardware

### Dongles
- **YARD Stick One** (primary recommended device)
- **CC1111EMK** (aka DONSDONGLES)
- **Chronos Watch Dongle** (aka CHRONOSDONGLE)
- **IMME** (limited support)
- **GoodFET** (for programming/debugging)

### Supported Chips
- **CC1111**: Primary 8051-based RF transceiver
- **CC2511**: USB-enabled variant
- **CC1110**: Base RF transceiver
- **CC2510**: USB-enabled variant

## Frequency Bands

RfCat supports multiple ISM frequency bands:

- **300 MHz band**: 281-361 MHz
- **400 MHz band**: 378-481 MHz
- **900 MHz band**: 749-962 MHz

Common frequencies include 315 MHz, 433 MHz, 868 MHz, and 915 MHz.

## Core Functionality

### 1. Radio Configuration

#### Frequency Control
- **`setFreq(freq)`**: Set carrier frequency in Hz (e.g., `d.setFreq(433000000)` for 433 MHz)
- **`getFreq()`**: Get current frequency setting
- **`setChannel(channr)`**: Set channel number for frequency hopping
- **`getChannel()`**: Get current channel
- **`setMdmChanSpc(chanspc)`**: Set channel spacing in Hz
- **`getMdmChanSpc()`**: Get channel spacing
- **`adjustFreqOffset()`**: Auto-adjust frequency offset based on received packet

#### Modulation Schemes
- **`setMdmModulation(mod)`**: Configure modulation type
  - `MOD_2FSK`: 2-level Frequency Shift Keying
  - `MOD_GFSK`: Gaussian FSK
  - `MOD_ASK_OOK`: Amplitude Shift Keying / On-Off Keying
  - `MOD_4FSK`: 4-level FSK
  - `MOD_MSK`: Minimum Shift Keying (data rates > 26 kBaud only)
  - `MANCHESTER`: Can be combined with modulation types (e.g., `MOD_2FSK | MANCHESTER`)
- **`getMdmModulation()`**: Get current modulation scheme
- **`setMdmDeviatn(deviatn)`**: Set frequency deviation in Hz
- **`getMdmDeviatn()`**: Get frequency deviation

#### Data Rate (Baud Rate)
- **`setMdmDRate(drate)`**: Set data rate in baud (e.g., `d.setMdmDRate(19200)`)
- **`getMdmDRate()`**: Get current data rate
- Supports rates from ~600 baud to 500 kBaud

#### Channel Bandwidth
- **`setMdmChanBW(bw)`**: Set receiver channel bandwidth in Hz
- **`getMdmChanBW()`**: Get channel bandwidth
- Range: ~54 kHz to 750 kHz
- Recommendation: Signal should occupy ≤80% of channel bandwidth

#### Power Control
- **`setPower(power)`**: Set transmit power level (0x00-0xFF)
- **`setMaxPower()`**: Set to maximum power for current frequency
  - Frequency-dependent: ≤400 MHz: 0xC2, 401-464 MHz: 0xC0, etc.

### 2. Packet Configuration

#### Packet Length
- **`makePktFLEN(flen)`**: Configure fixed-length packet mode
  - Maximum: 512 bytes (EP5OUT_BUFFER_SIZE - 4)
- **`makePktVLEN(maxlen)`**: Configure variable-length packet mode
  - Maximum: 255 bytes
  - First byte after sync word contains packet length
- **`getPktLEN()`**: Get current packet length configuration

#### Sync Word Configuration
- **`setMdmSyncWord(word)`**: Set 16-bit sync word (e.g., `0xD391`)
- **`getMdmSyncWord()`**: Get current sync word
- **`setMdmSyncMode(syncmode)`**: Set sync word detection mode
  - `SYNCM_NONE`: No sync word
  - `SYNCM_15_of_16`: 15 of 16 bits must match
  - `SYNCM_16_of_16`: All 16 bits must match
  - `SYNCM_30_of_32`: 30 of 32 bits must match
  - `SYNCM_CARRIER`: Carrier detect only
  - Combined carrier + sync modes available

#### Preamble
- **`setMdmNumPreamble(preamble)`**: Set minimum preamble bytes
  - Options: 2, 3, 4, 6, 8, 12, 16, 24 bytes
- **`getMdmNumPreamble()`**: Get preamble configuration
- **`setPktPQT(num)`**: Set Preamble Quality Threshold (0-7)
- **`getPktPQT()`**: Get PQT setting

#### Error Detection/Correction
- **`setEnablePktCRC(enable)`**: Enable/disable CRC checking
- **`getEnablePktCRC()`**: Get CRC enable status
- **`setEnableMdmFEC(enable)`**: Enable/disable Forward Error Correction
- **`getEnableMdmFEC()`**: Get FEC status

#### Data Encoding
- **`setEnableMdmManchester(enable)`**: Enable/disable Manchester encoding
- **`getEnableMdmManchester()`**: Get Manchester encoding status
- **`setEnablePktDataWhitening(enable)`**: Enable/disable data whitening
- **`getEnablePktDataWhitening()`**: Get data whitening status

#### Status Bytes
- **`setEnablePktAppendStatus(enable)`**: Append RSSI/LQI status bytes to received packets
- **`getEnablePktAppendStatus()`**: Get append status setting

### 3. Transmit and Receive Operations

#### Basic Transmission
- **`RFxmit(data, repeat=0, offset=0)`**: Transmit data packet
  - `data`: Bytes to transmit (max 255 bytes for standard mode)
  - `repeat`: Number of times to repeat transmission
  - `offset`: Timing offset for transmission
  - Example: `d.RFxmit(b"HELLO WORLD")`

#### Long Packet Transmission
- **`RFxmitLong(data)`**: Transmit packets larger than 255 bytes
  - Uses chunked transmission
  - Maximum: 65535 bytes

#### Reception
- **`RFrecv(timeout=1000)`**: Receive a packet
  - `timeout`: Milliseconds to wait for packet
  - Returns: `(data, timestamp)` tuple
  - Raises `ChipconUsbTimeoutException` on timeout
  - Example: `data, timestamp = d.RFrecv()`

#### Bulk Operations
- **`RFdump(msg, maxnum=100, timeoutms=1000)`**: Continuously receive and print packets
  - Useful for monitoring traffic
  - Displays hex-encoded packets with timestamps

### 4. Radio State Control

#### Mode Setting
- **`setModeIDLE()`**: Set radio to IDLE state (no TX/RX)
- **`setModeTX()`**: Set radio to transmit mode
- **`setModeRX()`**: Set radio to receive mode
- **`setRfMode(rfmode)`**: Set custom RF mode

#### Transient Mode Control (Strobe)
- **`strobeModeIDLE()`**: Temporarily enter IDLE
- **`strobeModeTX()`**: Temporarily enter TX
- **`strobeModeRX()`**: Temporarily enter RX
- **`strobeModeCAL()`**: Trigger calibration
- **`strobeModeFSTXON()`**: Frequency synthesizer TX on

### 5. Spectrum Analysis

- **`specan(centfreq=915e6, inc=250e3, count=104)`**: Launch GUI spectrum analyzer
  - `centfreq`: Center frequency in Hz
  - `inc`: Frequency increment between samples
  - `count`: Number of frequency bins (max 255)
  - Opens graphical window showing real-time spectrum
  - Example: `d.specan(433e6, 250e3, 100)`

- **`scan(basefreq, inc, count, delaysec, drate)`**: Command-line frequency scanner
  - Scans across frequency range looking for signals
  - Prints detected packets with frequencies

### 6. Frequency Hopping Spread Spectrum (FHSS)

RfCat includes advanced FHSS capabilities for frequency-hopping protocols:

#### Channel Management
- **`setChannels(channels)`**: Set list of channels for hopping pattern
  - `channels`: List of channel numbers
  - Example: `d.setChannels([0, 10, 20, 30, 40])`
- **`getChannels()`**: Get current channel list
- **`changeChannel(chan)`**: Manually change to specific channel
- **`nextChannel()`**: Advance to next channel in pattern

#### Hopping Control
- **`startHopping()`**: Begin automatic frequency hopping
- **`stopHopping()`**: Stop frequency hopping
- **`setMACperiod(dwell_ms)`**: Set dwell time per channel in milliseconds
- **`setMACthreshold(value)`**: Set MAC timing threshold
- **`getMACthreshold()`**: Get MAC threshold

#### FHSS State
- **`setFHSSstate(state)`**: Set FHSS state machine state
  - States: `FHSS_STATE_NONHOPPING`, `FHSS_STATE_DISCOVERY`, `FHSS_STATE_SYNCHING`, `FHSS_STATE_SYNCHED`, `FHSS_STATE_SYNC_MASTER`
- **`getFHSSstate()`**: Get current FHSS state
- **`mac_SyncCell(CellID)`**: Synchronize with a cell/network

#### FHSS Transmission
- **`FHSSxmit(data)`**: Transmit with frequency hopping
- **`getMACdata()`**: Get MAC layer state information
- **`setMACdata(data)`**: Set MAC layer parameters
- **`reprMACdata()`**: Get human-readable MAC state

### 7. AES Hardware Encryption

The CC1111 includes a hardware AES-128 encryption co-processor:

- **`setAESmode(aesmode)`**: Configure AES operation mode
  - Mode flags (bitfield):
    - `ENCCS_MODE_CBC`, `ENCCS_MODE_ECB`, `ENCCS_MODE_CTR`, `ENCCS_MODE_CFB`, `ENCCS_MODE_OFB`, `ENCCS_MODE_CBCMAC`
    - `AES_CRYPTO_IN_ON/OFF`: Enable/disable inbound encryption
    - `AES_CRYPTO_IN_ENCRYPT/DECRYPT`: Operation for inbound
    - `AES_CRYPTO_OUT_ON/OFF`: Enable/disable outbound encryption
    - `AES_CRYPTO_OUT_ENCRYPT/DECRYPT`: Operation for outbound
  - Example: `d.setAESmode(ENCCS_MODE_CBC | AES_CRYPTO_OUT_ON | AES_CRYPTO_OUT_ENCRYPT)`

- **`getAESmode()`**: Get current AES mode
- **`setAESkey(key)`**: Set 128-bit AES key (16 bytes)
- **`setAESiv(iv)`**: Set initialization vector (16 bytes)

### 8. Signal Quality and Diagnostics

#### RSSI and LQI
- **`getRSSI()`**: Get Received Signal Strength Indicator
  - Returns raw RSSI value from radio
- **`getLQI()`**: Get Link Quality Indicator
  - Provides indication of signal quality

#### Radio State
- **`getMARCSTATE()`**: Get Main Radio Control state machine state
  - Returns state name and numeric value
  - States include: IDLE, RX, TX, CALIBRATE, etc.

#### Debug and Status
- **`ping()`**: Test dongle communication
- **`debug()`**: Print debug information
- **`discover()`**: Attempt to discover radio configuration
- **`getPartNum()`**: Get chip part number
- **`getDebugCodes()`**: Retrieve debug codes from firmware

### 9. Low-Level Register Access

For advanced users who need direct hardware control:

- **`peek(addr, length=1)`**: Read from memory/register address
  - Returns raw bytes
- **`poke(addr, data)`**: Write to memory/register address
  - Direct memory manipulation
- **`setRFRegister(regaddr, value)`**: Set radio register (handles IDLE state)
  - Automatically puts radio in IDLE before writing
- **`setRFbits(addr, bitnum, bitsz, val)`**: Set individual register bits
- **`getRadioConfig()`**: Read complete radio configuration
- **`setRadioConfig(bytedef)`**: Write complete radio configuration
- **`reprRadioConfig()`**: Get human-readable radio configuration string

### 10. Advanced Features

#### Clear Channel Assessment (CCA)
- **`setEnableCCA(mode, absthresh, relthresh, magn)`**: Configure CCA for collision avoidance
  - Mode 0: Always transmit (no CCA)
  - Mode 1: TX only if RSSI below threshold
  - Mode 2: TX unless currently receiving
  - Mode 3: Combination of modes 1 and 2

#### Bit Synchronization
- **`setBSLimit(bslimit)`**: Set bit synchronization data rate offset compensation
  - Options: 0%, ±3.125%, ±6.25%, ±12.5% offset tolerance
- **`getBSLimit()`**: Get current BS limit

#### DC Filter
- **`setEnableMdmDCFilter(enable)`**: Enable/disable DC blocking filter
- **`getEnableMdmDCFilter()`**: Get DC filter status

#### Intermediate Frequency
- **`setFsIF(freq_if)`**: Set intermediate frequency
- **`getFsIF()`**: Get IF setting
- **`setFsOffset(if_off)`**: Set frequency offset
- **`getFsOffset()`**: Get frequency offset

#### LED Control
- **`setLedMode(ledmode)`**: Control LED behavior on dongle

#### Amplifier Control
- **`setAmpMode(ampmode)`**: Control external RF amplifier (hardware-dependent)
- **`getAmpMode()`**: Get amplifier mode

### 11. Automatic Configuration Helpers

RfCat includes experimental helpers that calculate optimal settings:

- **`calculateMdmDeviatn()`**: Auto-calculate deviation for current baud rate
- **`calculatePktChanBW()`**: Auto-calculate optimal channel bandwidth
- **`calculateFsIF()`**: Calculate optimal intermediate frequency
- **`setRFparameters()`**: Convenience function to configure multiple parameters

### 12. Configuration Management

#### Save/Load
- **`reprRadioConfig()`**: Get complete configuration as human-readable string
  - Shows all register values and their interpretations
- Radio configs can be saved to variables and restored
- **`lowball(lowball)`**: Set radio to low-power listening mode
- **`lowballRestore()`**: Restore from low-power mode

### 13. Interactive Mode Features

When launched with `rfcat -r`, provides interactive Python shell with:

- Tab completion for all functions
- `d` object pre-configured for dongle access
- IPython integration (if available)
- Direct Python scripting capabilities
- Example workflow:
  ```python
  d.setFreq(433000000)
  d.setMdmModulation(MOD_ASK_OOK)
  d.setMdmDRate(4800)
  d.RFxmit(b"TEST")
  data, time = d.RFrecv()
  print(d.reprRadioConfig())
  ```

## Operating Modes

### 1. Research Mode (`rfcat -r`)
Interactive Python shell for experimentation and research. Direct access to all rfcat functions through the `d` dongle object.

### 2. Spectrum Analyzer Mode (`rfcat -s`)
Launches GUI spectrum analyzer for visualizing RF spectrum in real-time.

### 3. Network Mode (rf_redirection)
- **`rf_redirection(fdtup)`**: Bridge RF to network socket or file descriptors
- Enables transparent RF packet forwarding to/from network
- Supports both sockets and file I/O

### 4. Bootloader Mode (`rfcat --bootloader --force`)
Enter bootloader mode for firmware updates.

### 5. Server Mode (`rfcat_server`)
Run as network server for remote access to dongle functionality.

## Integration Capabilities

### Metasploit Framework Integration

**rfcat_msfrelay** provides Metasploit Hardware Bridge support:

- Exposes rfcat functionality through Metasploit's hwbridge
- Enables use of Metasploit modules for RF attacks
- RESTful API for remote control
- Authentication: Username/password (default: `msf_relay:rfcat_relaypass`)

#### Available through Metasploit:
- Frequency configuration
- Modulation control
- Transmit/receive operations
- Channel bandwidth settings
- Power control
- Sync word configuration
- CRC and Manchester encoding control

Example Metasploit workflow:
```
use auxiliary/client/hwbridge/connect
set httpusername msf_relay
set httppassword rfcat_relaypass
run
sessions -i 1
# Now have access to RFtransceiver commands
```

### Network Bridging
- Can forward RF packets to TCP sockets
- Enables remote RF operations
- Packet format includes timestamp and length headers

## Special Implementations

### InverseCat Class
A variant of RfCat that automatically inverts all transmitted and received bits:
- **`InverseCat`**: Useful for protocols that use inverted encoding
- Automatically handles bit inversion on TX and RX

### IMME Sniff Mode
Special firmware and tools for the IMME girls toy:
- Interactive display on IMME screen
- Keyboard controls for frequency/modulation
- Real-time packet display
- Key bindings for rapid configuration changes

## Use Cases

### Security Research
- Wireless protocol reverse engineering
- Replay attacks on RF systems
- Jamming and denial-of-service testing
- Wireless key fob analysis
- Tire Pressure Monitoring System (TPMS) research
- Wireless sensor network analysis

### Protocol Analysis
- Unknown protocol discovery
- Packet capture and analysis
- Timing analysis
- Modulation identification
- Frequency hopping pattern analysis

### Hardware Reverse Engineering
- Identify operating frequency
- Determine modulation scheme
- Extract sync words and packet structure
- Analyze error correction mechanisms

### Penetration Testing
- Wireless building access systems
- Garage door openers
- Remote controls
- Industrial wireless sensors
- Smart home devices

### Wireless Communication Development
- Protocol prototyping
- Range testing
- Interference analysis
- Custom wireless link development

## Performance Characteristics

- **USB Interface**: USB 2.0 Full Speed
- **Maximum Packet Size**: 512 bytes (receive buffer), 255 bytes (standard TX), 65535 bytes (long TX)
- **USB Transfer Block**: 64 bytes (EP5)
- **Frequency Resolution**: ~396 Hz steps (24 MHz crystal)
- **Data Rate Range**: ~600 baud to 500 kBaud
- **TX Power Range**: Configurable from ~-30 dBm to +10 dBm (frequency-dependent)

## Important Considerations

### Critical Requirements
1. **IDLE Before Configure**: Radio must be in IDLE state before changing most configuration registers
2. **TX Before Write**: Must be in TX mode before writing to RFD register (firmware handles this)
3. **USB Port Compatibility**: USB3 ports can cause issues; use USB2 ports when possible
4. **Root/Permissions**: Requires root access or udev rules configuration for non-root access

### Limitations
- Python 2.7 primarily (Python 3 support in progress)
- MSK modulation only for data rates > 26 kBaud
- Manchester encoding not supported with 4FSK
- ASK/OOK only up to 250 kBaud
- Frequency range limited by hardware (no 2.4 GHz support)

### Best Practices
1. Always call `d.ping()` to verify dongle connection
2. Use `d.setModeIDLE()` before extensive configuration changes
3. Read current config with `d.getRadioConfig()` before modifications
4. Use `d.reprRadioConfig()` to document working configurations
5. Handle `ChipconUsbTimeoutException` in receive loops
6. Match channel bandwidth to signal requirements (signal ≤80% of BW)
7. Configure CRC and FEC appropriately for your protocol
8. Test with different preamble/sync settings for reliability

### Common Gotchas
- RAM is limited on CC1111 (use sparingly)
- Different assembly for RAM vs XDATA variables
- Radio must be in correct state for operations
- USB timeouts vary between IDLE and ACTIVE states
- Some USB3 ports are incompatible

## Configuration Examples

### 433 MHz ASK/OOK (Typical for key fobs)
```python
d.setFreq(433920000)
d.setMdmModulation(MOD_ASK_OOK)
d.setMdmDRate(4800)
d.setMdmSyncMode(SYNCM_NONE)
d.makePktFLEN(64)
d.setMaxPower()
```

### 915 MHz GFSK (Typical for wireless sensors)
```python
d.setFreq(915000000)
d.setMdmModulation(MOD_GFSK)
d.setMdmDRate(38400)
d.setMdmDeviatn(20000)
d.setMdmChanBW(94000)
d.setMdmSyncWord(0xD391)
d.setMdmSyncMode(SYNCM_16_of_16)
d.makePktVLEN(60)
d.setEnablePktCRC(True)
d.setEnableMdmFEC(True)
```

### 868 MHz FSK with Manchester (Typical for European ISM)
```python
d.setFreq(868300000)
d.setMdmModulation(MOD_2FSK | MANCHESTER)
d.setMdmDRate(19200)
d.setMdmDeviatn(5100)
d.setMdmChanBW(63000)
d.setMdmSyncWord(0xAAAA)
```

## External Projects Using RfCat

- **Z-Wave Attack Framework**: Z-Wave security research tools
- Various penetration testing tools
- Academic research projects
- Industrial wireless security assessments

## Summary

RfCat is a powerful, flexible framework for sub-GHz wireless research that provides:

- **Comprehensive radio control**: Full access to all CC1111 radio parameters
- **Multiple modulation schemes**: Support for FSK, GFSK, ASK/OOK, MSK, and Manchester encoding
- **Advanced features**: FHSS, hardware AES encryption, spectrum analysis
- **Flexible operation modes**: Interactive research, spectrum analyzer, network bridging, Metasploit integration
- **Low-level access**: Direct register manipulation for maximum control
- **High-level convenience**: Pre-configured functions for common operations

The framework excels at rapid prototyping, reverse engineering, and security research for wireless protocols operating below 1 GHz. Its Python-based API makes it accessible while still providing the low-level control necessary for advanced RF work.
