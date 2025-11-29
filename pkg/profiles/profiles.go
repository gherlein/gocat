// Package profiles provides pre-defined radio configuration profiles for the YardStick One.
// Each profile represents a specific combination of frequency, modulation, data rate,
// and other radio parameters optimized for particular use cases.
package profiles

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/herlein/gocat/pkg/registers"
)

// CrystalMHz is the crystal frequency for CC1111 (YardStick One)
const CrystalMHz = 24.0

// Modulation types
const (
	Mod2FSK   = 0x00 // 2-level FSK
	ModGFSK   = 0x10 // Gaussian FSK
	ModASKOOK = 0x30 // ASK/OOK
	Mod4FSK   = 0x40 // 4-level FSK
	ModMSK    = 0x70 // Minimum Shift Keying
)

// Sync modes
const (
	SyncNone          = 0x00 // No preamble/sync
	Sync15of16        = 0x01 // 15 of 16 sync word bits detected
	Sync16of16        = 0x02 // 16/16 sync word bits detected
	Sync30of32        = 0x03 // 30/32 sync word bits detected
	SyncCarrier       = 0x04 // Carrier-sense above threshold
	SyncCarrier15of16 = 0x05 // Carrier + 15/16 sync
	SyncCarrier16of16 = 0x06 // Carrier + 16/16 sync
	SyncCarrier30of32 = 0x07 // Carrier + 30/32 sync
)

// Preamble lengths (register values)
const (
	Preamble2  = 0x00 << 4 // 2 bytes
	Preamble3  = 0x01 << 4 // 3 bytes
	Preamble4  = 0x02 << 4 // 4 bytes (default)
	Preamble6  = 0x03 << 4 // 6 bytes
	Preamble8  = 0x04 << 4 // 8 bytes
	Preamble12 = 0x05 << 4 // 12 bytes
	Preamble16 = 0x06 << 4 // 16 bytes
	Preamble24 = 0x07 << 4 // 24 bytes
)

// Packet length modes
const (
	PktLenFixed    = 0x00 // Fixed packet length mode
	PktLenVariable = 0x01 // Variable packet length mode
	PktLenInfinite = 0x02 // Infinite packet length mode
)

// Profile represents a complete radio configuration profile
type Profile struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	FrequencyHz float64 `json:"frequency_hz"`

	// Modulation settings
	Modulation     uint8   `json:"modulation"`
	DataRateBaud   float64 `json:"data_rate_baud"`
	DeviationHz    float64 `json:"deviation_hz,omitempty"` // For FSK modes
	ChannelBWHz    float64 `json:"channel_bandwidth_hz"`
	ManchesterEn   bool    `json:"manchester_enabled,omitempty"`
	DataWhiteningEn bool   `json:"whitening_enabled,omitempty"`

	// Sync settings
	SyncWord uint16 `json:"sync_word,omitempty"`
	SyncMode uint8  `json:"sync_mode"`

	// Packet settings
	PktLenMode    uint8 `json:"packet_length_mode"`
	PktLen        uint8 `json:"packet_length"`
	PreambleBytes uint8 `json:"preamble_bytes"`
	CRCEn         bool  `json:"crc_enabled"`
	FECEn         bool  `json:"fec_enabled,omitempty"`

	// Power settings
	TXPowerDBm int `json:"tx_power_dbm"`
}

// ProfileConfig is the JSON format for storing profile configurations
type ProfileConfig struct {
	Profile   Profile               `json:"profile"`
	Registers registers.RegisterMap `json:"registers"`
	Timestamp time.Time             `json:"timestamp"`
}

// CalcFreqRegs calculates FREQ2/1/0 register values for a given frequency
func CalcFreqRegs(freqHz float64) (freq2, freq1, freq0 uint8) {
	freqMult := (65536.0 / 1000000.0) / CrystalMHz
	num := uint32(freqHz * freqMult)
	freq2 = uint8((num >> 16) & 0xFF)
	freq1 = uint8((num >> 8) & 0xFF)
	freq0 = uint8(num & 0xFF)
	return
}

// CalcDataRateRegs calculates MDMCFG4[3:0] (DRATE_E) and MDMCFG3 (DRATE_M) for a given data rate
func CalcDataRateRegs(drateBaud float64) (drateE, drateM uint8) {
	crystalHz := CrystalMHz * 1000000.0
	for e := uint8(0); e < 16; e++ {
		m := int((drateBaud*math.Pow(2, 28)/(math.Pow(2, float64(e))*crystalHz) - 256) + 0.5)
		if m >= 0 && m < 256 {
			drateE = e
			drateM = uint8(m)
			return
		}
	}
	// Fallback to max
	return 15, 255
}

// CalcChannelBWRegs calculates MDMCFG4[7:4] for channel bandwidth
func CalcChannelBWRegs(bwHz float64) (chanbwE, chanbwM uint8) {
	crystalHz := CrystalMHz * 1000000.0
	for e := uint8(0); e < 4; e++ {
		m := int((crystalHz/(bwHz*math.Pow(2, float64(e))*8.0) - 4) + 0.5)
		if m >= 0 && m < 4 {
			chanbwE = e
			chanbwM = uint8(m)
			return
		}
	}
	// Fallback to widest bandwidth
	return 0, 0
}

// CalcDeviationRegs calculates DEVIATN register for FSK deviation
func CalcDeviationRegs(devHz float64) uint8 {
	crystalHz := CrystalMHz * 1000000.0
	for e := uint8(0); e < 8; e++ {
		m := int((devHz*math.Pow(2, 17)/(math.Pow(2, float64(e))*crystalHz) - 8) + 0.5)
		if m >= 0 && m < 8 {
			return (e << 4) | uint8(m)
		}
	}
	// Fallback
	return 0x47 // ~25 kHz at 24 MHz crystal
}

// GetMaxPower returns the maximum PA_TABLE value for a given frequency
func GetMaxPower(freqHz float64) uint8 {
	if freqHz <= 400000000 {
		return 0xC2
	} else if freqHz <= 464000000 {
		return 0xC0
	} else if freqHz <= 849000000 {
		return 0xC2
	}
	return 0xC0
}

// GetVCOSelection returns FSCAL2 value based on frequency
func GetVCOSelection(freqHz float64) uint8 {
	// VCO selection thresholds for each band
	if freqHz < 318000000 || (freqHz >= 391000000 && freqHz < 424000000) || (freqHz >= 782000000 && freqHz < 848000000) {
		return 0x0A // Low VCO
	}
	return 0x2A // High VCO
}

// PreambleBytesToReg converts preamble byte count to register value
func PreambleBytesToReg(bytes uint8) uint8 {
	switch bytes {
	case 2:
		return Preamble2
	case 3:
		return Preamble3
	case 4:
		return Preamble4
	case 6:
		return Preamble6
	case 8:
		return Preamble8
	case 12:
		return Preamble12
	case 16:
		return Preamble16
	case 24:
		return Preamble24
	default:
		return Preamble4 // Default to 4 bytes
	}
}

// ToRegisters converts a Profile to a RegisterMap
func (p *Profile) ToRegisters() *registers.RegisterMap {
	reg := &registers.RegisterMap{}

	// Frequency
	freq2, freq1, freq0 := CalcFreqRegs(p.FrequencyHz)
	reg.FREQ2 = freq2
	reg.FREQ1 = freq1
	reg.FREQ0 = freq0

	// VCO selection
	reg.FSCAL2 = GetVCOSelection(p.FrequencyHz)

	// Data rate and channel bandwidth
	drateE, drateM := CalcDataRateRegs(p.DataRateBaud)
	chanbwE, chanbwM := CalcChannelBWRegs(p.ChannelBWHz)
	reg.MDMCFG4 = (chanbwE << 6) | (chanbwM << 4) | drateE
	reg.MDMCFG3 = drateM

	// Modulation and sync mode
	reg.MDMCFG2 = p.Modulation | p.SyncMode
	if p.ManchesterEn {
		reg.MDMCFG2 |= 0x08 // Manchester enable bit
	}

	// Deviation (for FSK modes)
	if p.Modulation == Mod2FSK || p.Modulation == ModGFSK || p.Modulation == Mod4FSK {
		if p.DeviationHz > 0 {
			reg.DEVIATN = CalcDeviationRegs(p.DeviationHz)
		} else {
			// Default deviation based on data rate
			reg.DEVIATN = CalcDeviationRegs(p.DataRateBaud * 0.5)
		}
	}

	// Preamble and FEC
	reg.MDMCFG1 = PreambleBytesToReg(p.PreambleBytes)
	if p.FECEn {
		reg.MDMCFG1 |= 0x80 // FEC enable bit
	}

	// Channel spacing (default)
	reg.MDMCFG0 = 0xF8

	// Sync word
	reg.SYNC1 = uint8((p.SyncWord >> 8) & 0xFF)
	reg.SYNC0 = uint8(p.SyncWord & 0xFF)

	// Packet configuration
	reg.PKTLEN = p.PktLen
	reg.PKTCTRL0 = p.PktLenMode
	if p.CRCEn {
		reg.PKTCTRL0 |= 0x04 // CRC enable bit
	}
	if p.DataWhiteningEn {
		reg.PKTCTRL0 |= 0x40 // Data whitening enable bit
	}
	reg.PKTCTRL1 = 0x04 // Append status bytes (RSSI, LQI, CRC OK)

	// Power amplifier
	maxPower := GetMaxPower(p.FrequencyHz)
	if p.Modulation == ModASKOOK {
		// ASK/OOK uses PA_TABLE0=0x00, PA_TABLE1=power
		reg.PA_TABLE[0] = 0x00
		reg.PA_TABLE[1] = maxPower
		reg.FREND0 = 0x11 // Use PA_TABLE[1] for TX
	} else {
		reg.PA_TABLE[0] = maxPower
		reg.PA_TABLE[1] = 0x00
		reg.FREND0 = 0x10 // Use PA_TABLE[0] for TX
	}

	// Frontend configuration based on bandwidth
	if p.ChannelBWHz > 102000 {
		reg.FREND1 = 0xB6
	} else {
		reg.FREND1 = 0x56
	}

	// TEST registers based on bandwidth
	if p.ChannelBWHz > 325000 {
		reg.TEST2 = 0x88
		reg.TEST1 = 0x31
	} else {
		reg.TEST2 = 0x81
		reg.TEST1 = 0x35
	}
	reg.TEST0 = 0x09

	// Frequency synthesizer settings
	reg.FSCTRL1 = 0x06 // IF frequency
	reg.FSCTRL0 = 0x00 // Frequency offset

	// Calibration values (defaults)
	reg.FSCAL3 = 0xE9
	reg.FSCAL1 = 0x00
	reg.FSCAL0 = 0x1F

	// AGC settings (defaults)
	reg.AGCCTRL2 = 0x03
	reg.AGCCTRL1 = 0x40
	reg.AGCCTRL0 = 0x91

	// Frequency offset compensation
	reg.FOCCFG = 0x16
	reg.BSCFG = 0x6C

	// Main radio control state machine
	reg.MCSM0 = 0x18 // Auto-calibrate on IDLE->RX/TX
	reg.MCSM1 = 0x00 // Return to IDLE after TX/RX
	reg.MCSM2 = 0x07 // RX timeout disabled

	// GPIO configuration (defaults)
	reg.IOCFG2 = 0x29
	reg.IOCFG1 = 0x2E
	reg.IOCFG0 = 0x06

	// Address (no filtering)
	reg.ADDR = 0x00
	reg.CHANNR = 0x00

	return reg
}

// SaveToFile saves a profile configuration to a JSON file
func (p *Profile) SaveToFile(filepath string) error {
	config := ProfileConfig{
		Profile:   *p,
		Registers: *p.ToRegisters(),
		Timestamp: time.Now(),
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}

	return os.WriteFile(filepath, data, 0644)
}

// LoadProfileFromFile loads a profile configuration from a JSON file
func LoadProfileFromFile(path string) (*ProfileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile file: %w", err)
	}

	var config ProfileConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal profile: %w", err)
	}

	return &config, nil
}

// EnsureDir ensures the directory for a file path exists
func EnsureDir(filePath string) error {
	dir := filepath.Dir(filePath)
	return os.MkdirAll(dir, 0755)
}
