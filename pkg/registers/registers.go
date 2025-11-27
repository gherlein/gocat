package registers

// RegisterMap holds all CC1111 radio configuration registers
type RegisterMap struct {
	// Sync word
	SYNC1 uint8 `json:"sync1"` // 0xDF00
	SYNC0 uint8 `json:"sync0"` // 0xDF01

	// Packet control
	PKTLEN   uint8 `json:"pktlen"`   // 0xDF02
	PKTCTRL1 uint8 `json:"pktctrl1"` // 0xDF03
	PKTCTRL0 uint8 `json:"pktctrl0"` // 0xDF04
	ADDR     uint8 `json:"addr"`     // 0xDF05
	CHANNR   uint8 `json:"channr"`   // 0xDF06

	// Frequency synthesizer
	FSCTRL1 uint8 `json:"fsctrl1"` // 0xDF07
	FSCTRL0 uint8 `json:"fsctrl0"` // 0xDF08

	// Frequency control
	FREQ2 uint8 `json:"freq2"` // 0xDF09
	FREQ1 uint8 `json:"freq1"` // 0xDF0A
	FREQ0 uint8 `json:"freq0"` // 0xDF0B

	// Modem configuration
	MDMCFG4 uint8 `json:"mdmcfg4"` // 0xDF0C
	MDMCFG3 uint8 `json:"mdmcfg3"` // 0xDF0D
	MDMCFG2 uint8 `json:"mdmcfg2"` // 0xDF0E
	MDMCFG1 uint8 `json:"mdmcfg1"` // 0xDF0F
	MDMCFG0 uint8 `json:"mdmcfg0"` // 0xDF10
	DEVIATN uint8 `json:"deviatn"` // 0xDF11

	// Main radio control state machine
	MCSM2 uint8 `json:"mcsm2"` // 0xDF12
	MCSM1 uint8 `json:"mcsm1"` // 0xDF13
	MCSM0 uint8 `json:"mcsm0"` // 0xDF14

	// Frequency offset compensation
	FOCCFG uint8 `json:"foccfg"` // 0xDF15
	BSCFG  uint8 `json:"bscfg"`  // 0xDF16

	// AGC control
	AGCCTRL2 uint8 `json:"agcctrl2"` // 0xDF17
	AGCCTRL1 uint8 `json:"agcctrl1"` // 0xDF18
	AGCCTRL0 uint8 `json:"agcctrl0"` // 0xDF19

	// Front end configuration
	FREND1 uint8 `json:"frend1"` // 0xDF1A
	FREND0 uint8 `json:"frend0"` // 0xDF1B

	// Frequency synthesizer calibration
	FSCAL3 uint8 `json:"fscal3"` // 0xDF1C
	FSCAL2 uint8 `json:"fscal2"` // 0xDF1D
	FSCAL1 uint8 `json:"fscal1"` // 0xDF1E
	FSCAL0 uint8 `json:"fscal0"` // 0xDF1F

	// Test registers
	TEST2 uint8 `json:"test2"` // 0xDF23
	TEST1 uint8 `json:"test1"` // 0xDF24
	TEST0 uint8 `json:"test0"` // 0xDF25

	// Power amplifier table (note: reversed order in memory)
	PA_TABLE [8]uint8 `json:"pa_table"` // 0xDF27-0xDF2E (PA_TABLE7-PA_TABLE0)

	// GPIO configuration
	IOCFG2 uint8 `json:"iocfg2"` // 0xDF2F
	IOCFG1 uint8 `json:"iocfg1"` // 0xDF30
	IOCFG0 uint8 `json:"iocfg0"` // 0xDF31

	// Read-only status registers
	PARTNUM    uint8 `json:"partnum"`     // 0xDF36
	CHIPID     uint8 `json:"chipid"`      // 0xDF37 (VERSION in some docs)
	FREQEST    uint8 `json:"freqest"`     // 0xDF38
	LQI        uint8 `json:"lqi"`         // 0xDF39
	RSSI       uint8 `json:"rssi"`        // 0xDF3A
	MARCSTATE  uint8 `json:"marcstate"`   // 0xDF3B
	PKTSTATUS  uint8 `json:"pktstatus"`   // 0xDF3C
	VCO_VC_DAC uint8 `json:"vco_vc_dac"` // 0xDF3D
}

// RadioState represents the main radio control state
type RadioState uint8

const (
	StateSLEEP       RadioState = 0x00
	StateIDLE        RadioState = 0x01
	StateXOFF        RadioState = 0x02
	StateVCOON_MC    RadioState = 0x03
	StateREGON_MC    RadioState = 0x04
	StateMAN_CAL     RadioState = 0x05
	StateVCOON       RadioState = 0x06
	StateREGON       RadioState = 0x07
	StateSTARTCAL    RadioState = 0x08
	StateBWBOOST     RadioState = 0x09
	StateFS_LOCK     RadioState = 0x0A
	StateIFADCON     RadioState = 0x0B
	StateENDCAL      RadioState = 0x0C
	StateRX          RadioState = 0x0D
	StateRX_END      RadioState = 0x0E
	StateRX_RST      RadioState = 0x0F
	StateTXRX_SWITCH RadioState = 0x10
	StateRXFIFO_OVF  RadioState = 0x11
	StateFSTXON      RadioState = 0x12
	StateTX          RadioState = 0x13
	StateTX_END      RadioState = 0x14
	StateRXTX_SWITCH RadioState = 0x15
	StateTXFIFO_UNF  RadioState = 0x16
)

// String returns a human-readable name for the radio state
func (s RadioState) String() string {
	names := map[RadioState]string{
		StateSLEEP:       "SLEEP",
		StateIDLE:        "IDLE",
		StateXOFF:        "XOFF",
		StateVCOON_MC:    "VCOON_MC",
		StateREGON_MC:    "REGON_MC",
		StateMAN_CAL:     "MANCAL",
		StateVCOON:       "VCOON",
		StateREGON:       "REGON",
		StateSTARTCAL:    "STARTCAL",
		StateBWBOOST:     "BWBOOST",
		StateFS_LOCK:     "FS_LOCK",
		StateIFADCON:     "IFADCON",
		StateENDCAL:      "ENDCAL",
		StateRX:          "RX",
		StateRX_END:      "RX_END",
		StateRX_RST:      "RX_RST",
		StateTXRX_SWITCH: "TXRX_SWITCH",
		StateRXFIFO_OVF:  "RXFIFO_OVERFLOW",
		StateFSTXON:      "FSTXON",
		StateTX:          "TX",
		StateTX_END:      "TX_END",
		StateRXTX_SWITCH: "RXTX_SWITCH",
		StateTXFIFO_UNF:  "TXFIFO_UNDERFLOW",
	}
	if name, ok := names[s]; ok {
		return name
	}
	return "UNKNOWN"
}

// Register addresses (memory-mapped at 0xDF00)
const (
	RegSYNC1     = 0xDF00
	RegSYNC0     = 0xDF01
	RegPKTLEN    = 0xDF02
	RegPKTCTRL1  = 0xDF03
	RegPKTCTRL0  = 0xDF04
	RegADDR      = 0xDF05
	RegCHANNR    = 0xDF06
	RegFSCTRL1   = 0xDF07
	RegFSCTRL0   = 0xDF08
	RegFREQ2     = 0xDF09
	RegFREQ1     = 0xDF0A
	RegFREQ0     = 0xDF0B
	RegMDMCFG4   = 0xDF0C
	RegMDMCFG3   = 0xDF0D
	RegMDMCFG2   = 0xDF0E
	RegMDMCFG1   = 0xDF0F
	RegMDMCFG0   = 0xDF10
	RegDEVIATN   = 0xDF11
	RegMCSM2     = 0xDF12
	RegMCSM1     = 0xDF13
	RegMCSM0     = 0xDF14
	RegFOCCFG    = 0xDF15
	RegBSCFG     = 0xDF16
	RegAGCCTRL2  = 0xDF17
	RegAGCCTRL1  = 0xDF18
	RegAGCCTRL0  = 0xDF19
	RegFREND1    = 0xDF1A
	RegFREND0    = 0xDF1B
	RegFSCAL3    = 0xDF1C
	RegFSCAL2    = 0xDF1D
	RegFSCAL1    = 0xDF1E
	RegFSCAL0    = 0xDF1F
	// Reserved 0xDF20-0xDF22
	RegTEST2 = 0xDF23
	RegTEST1 = 0xDF24
	RegTEST0 = 0xDF25
	// Reserved 0xDF26
	RegPA_TABLE7 = 0xDF27
	RegPA_TABLE6 = 0xDF28
	RegPA_TABLE5 = 0xDF29
	RegPA_TABLE4 = 0xDF2A
	RegPA_TABLE3 = 0xDF2B
	RegPA_TABLE2 = 0xDF2C
	RegPA_TABLE1 = 0xDF2D
	RegPA_TABLE0 = 0xDF2E
	RegIOCFG2    = 0xDF2F
	RegIOCFG1    = 0xDF30
	RegIOCFG0    = 0xDF31
	// Reserved 0xDF32-0xDF35
	RegPARTNUM    = 0xDF36
	RegCHIPID     = 0xDF37 // Also called VERSION
	RegFREQEST    = 0xDF38
	RegLQI        = 0xDF39
	RegRSSI       = 0xDF3A
	RegMARCSTATE  = 0xDF3B
	RegPKTSTATUS  = 0xDF3C
	RegVCO_VC_DAC = 0xDF3D
)

// Radio strobe commands (RFST register values)
const (
	StrobeSFSTXON = 0x00 // Enable and calibrate frequency synthesizer
	StrobeSCAL    = 0x01 // Calibrate frequency synthesizer
	StrobeSRX     = 0x02 // Enable RX
	StrobeSTX     = 0x03 // Enable TX
	StrobeSIDLE   = 0x04 // Exit RX/TX, turn off frequency synthesizer
	StrobeSNOP    = 0x05 // No operation
)

// RFST register address
const RegRFST = 0xDFE1

// Modulation formats (MDMCFG2[6:4])
const (
	Mod2FSK   = 0x00 // 2-FSK
	ModGFSK   = 0x10 // GFSK
	ModASKOOK = 0x30 // ASK/OOK
	Mod4FSK   = 0x40 // 4-FSK
	ModMSK    = 0x70 // MSK
)

// Sync mode (MDMCFG2[2:0])
const (
	SyncNone              = 0x00 // No preamble/sync
	Sync15of16            = 0x01 // 15/16 sync word bits detected
	Sync16of16            = 0x02 // 16/16 sync word bits detected
	Sync30of32            = 0x03 // 30/32 sync word bits detected
	SyncCarrier           = 0x04 // Carrier-sense above threshold
	SyncCarrier15of16     = 0x05 // Carrier-sense + 15/16 sync
	SyncCarrier16of16     = 0x06 // Carrier-sense + 16/16 sync
	SyncCarrier30of32     = 0x07 // Carrier-sense + 30/32 sync
)

// Packet length config (PKTCTRL0[1:0])
const (
	PktLenFixed    = 0x00 // Fixed packet length mode
	PktLenVariable = 0x01 // Variable packet length mode
	PktLenInfinite = 0x02 // Infinite packet length mode
)

// CRC enable (PKTCTRL0[2])
const (
	CRCDisabled = 0x00
	CRCEnabled  = 0x04
)

// Data whitening (PKTCTRL0[6])
const (
	WhiteningDisabled = 0x00
	WhiteningEnabled  = 0x40
)

// FEC enable (PKTCTRL0[7] - CC1101 only, not CC1111)
const (
	FECDisabled = 0x00
	FECEnabled  = 0x80
)
