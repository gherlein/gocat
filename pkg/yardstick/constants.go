package yardstick

import "time"

// USB Device Identifiers
const (
	VendorID  = 0x1D50
	ProductID = 0x605B // YardStick One

	// Alternative product IDs for other devices
	ProductIDDonsDongle    = 0x6048
	ProductIDChronosDongle = 0x6047
	ProductIDSRFStick      = 0xECC1

	// Bootloader IDs
	ProductIDBootloader     = 0x6049
	ProductIDBootloaderAlt  = 0x604A
	ProductIDBootloaderAlt2 = 0xECC0
)

// USB Endpoint Configuration
const (
	EP5InAddr           = 0x85 // EP5 IN (device to host)
	EP5OutAddr          = 0x05 // EP5 OUT (host to device)
	EP5MaxPacketSize    = 64
	EP5OutBufferSize    = 516
	EP0MaxPacketSize    = 32
	ResponseMarker      = 0x40 // '@' character marks start of response
)

// USB Timeouts
const (
	USBDefaultTimeout = 1000 * time.Millisecond
	USBRXWaitTimeout  = 1000 * time.Millisecond
	USBTXWaitTimeout  = 10000 * time.Millisecond
)

// Application IDs for EP5 protocol
const (
	AppGeneric = 0x01 // Generic application (reserved)
	AppNIC     = 0x42 // Radio NIC operations
	AppSPECAN  = 0x43 // Spectrum analyzer
	AppDebug   = 0xFE // Debug output
	AppSystem  = 0xFF // System/administrative commands
)

// System Commands (APP_SYSTEM = 0xFF)
const (
	SysCmdPeek              = 0x80 // Read memory
	SysCmdPoke              = 0x81 // Write memory
	SysCmdPing              = 0x82 // Echo test
	SysCmdStatus            = 0x83 // Get status
	SysCmdPokeReg           = 0x84 // Write to register
	SysCmdGetClock          = 0x85 // Get clock value
	SysCmdBuildType         = 0x86 // Get firmware build info
	SysCmdBootloader        = 0x87 // Enter bootloader
	SysCmdRFMode            = 0x88 // Set radio mode
	SysCmdCompiler          = 0x89 // Get compiler info
	SysCmdPartNum           = 0x8E // Get chip part number
	SysCmdReset             = 0x8F // Reset device
	SysCmdClearCodes        = 0x90 // Clear debug codes
	SysCmdDeviceSerialNum   = 0x91 // Get device serial number
	SysCmdLEDMode           = 0x93 // Set LED mode
)

// NIC Commands (APP_NIC = 0x42)
const (
	NICRecv         = 0x01 // Receive RF data
	NICXmit         = 0x02 // Transmit RF data
	NICSetID        = 0x03 // Set network/device ID
	NICSetRecvLarge = 0x05 // Configure large packet receive
	NICSetAESMode   = 0x06 // Set AES crypto mode
	NICGetAESMode   = 0x07 // Get AES crypto mode
	NICSetAESIV     = 0x08 // Set AES initialization vector
	NICSetAESKey    = 0x09 // Set AES key
	NICSetAmpMode   = 0x0A // Set amplifier mode
	NICGetAmpMode   = 0x0B // Get amplifier mode
	NICLongXmit     = 0x0C // Start long transmission
	NICLongXmitMore = 0x0D // Continue long transmission
)

// EP0 Vendor Commands (control transfers)
const (
	EP0CmdGetDebugCodes = 0x00 // Get debug/error codes
	EP0CmdPokeX         = 0x01 // Write to XDATA memory
	EP0CmdPeekX         = 0x02 // Read from XDATA memory
	EP0CmdPing0         = 0x03 // Ping (echo request)
	EP0CmdPing1         = 0x04 // Ping (echo EP0 OUT buffer)
	EP0CmdWCID          = 0xFE // Windows Compatible ID
	EP0CmdReset         = 0xFF // Reset device
)

// USB Request Types
const (
	RequestTypeVendorIn  = 0xC0 // Vendor request, device to host
	RequestTypeVendorOut = 0x40 // Vendor request, host to device
)

// Radio Strobe Commands (RFST register values)
const (
	RFSTSfstxon = 0x00 // Enable and calibrate
	RFSTScal    = 0x01 // Calibrate
	RFSTSrx     = 0x02 // Enable RX
	RFSTStx     = 0x03 // Enable TX
	RFSTSidle   = 0x04 // Idle mode
	RFSTSnop    = 0x05 // No operation
)

// LED Mode values
const (
	LEDModeOff = 0x00
	LEDModeOn  = 0x01
)

// Chip Part Numbers
const (
	PartNumCC1110 = 0x01
	PartNumCC1111 = 0x11
	PartNumCC2510 = 0x81
	PartNumCC2511 = 0x91
)

// RF Constants
const (
	RFMaxTXBlock = 255   // Maximum standard TX block size
	RFMaxTXLong  = 65535 // Maximum long TX size
	RFMaxTXChunk = 240   // Chunk size for long transmit
	RFMaxRXBlock = 512   // Maximum RX block size
)

// Error/Return Codes
const (
	RCNoError                    = 0x00
	RCTXDroppedPacket            = 0xEC
	RCTXError                    = 0xED
	RCRFBlocksizeIncompat        = 0xEE
	RCRFModeIncompat             = 0xEF
	RCTempErrBufferNotAvailable  = 0xFE
	RCErrBufferSizeExceeded      = 0xFF
)

// Last Code Error values (LCE_*)
const (
	LCENoError                      = 0x00
	LCEUSBEP5TXWhileInbufWritten    = 0x01
	LCEUSBEP0SentStall              = 0x04
	LCEUSBEP5OutWhileOutbufWritten  = 0x05
	LCEUSBEP5LenTooBig              = 0x06
	LCEUSBEP5GotCrap                = 0x07
	LCEUSBEP5Stall                  = 0x08
	LCERFRXOverflow                 = 0x10
	LCERFTXUnderflow                = 0x11
)
