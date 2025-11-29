package profiles

import "fmt"

// Encoding Variation Profile Factories
// These test different encoding options: Manchester, whitening, sync modes, etc.

// NewManchesterVariant creates profiles with different Manchester encoding settings
// modType: "ook", "2fsk", or "gfsk"
// dataRate: data rate in baud
func NewManchesterVariant(modType string, dataRate float64) *Profile {
	var mod uint8
	switch modType {
	case "ook":
		mod = ModASKOOK
	case "2fsk":
		mod = Mod2FSK
	case "gfsk":
		mod = ModGFSK
	default:
		mod = Mod2FSK
	}

	return &Profile{
		Name:          fmt.Sprintf("enc-manch-%s-%s", modType, formatDataRate(dataRate)),
		Description:   fmt.Sprintf("Manchester encoding test with %s at %.0f baud", modType, dataRate),
		FrequencyHz:   433920000,
		Modulation:    mod,
		DataRateBaud:  dataRate,
		DeviationHz:   10000,
		ChannelBWHz:   100000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        60,
		PreambleBytes: 4,
		CRCEn:         true,
		ManchesterEn:  true,
	}
}

// NewWhiteningVariant creates profiles with data whitening enabled
// modType: "2fsk" or "gfsk"
// dataRate: data rate in baud
func NewWhiteningVariant(modType string, dataRate float64) *Profile {
	var mod uint8
	switch modType {
	case "2fsk":
		mod = Mod2FSK
	case "gfsk":
		mod = ModGFSK
	default:
		mod = ModGFSK
	}

	return &Profile{
		Name:            fmt.Sprintf("enc-white-%s-%s", modType, formatDataRate(dataRate)),
		Description:     fmt.Sprintf("Data whitening test with %s at %.0f baud", modType, dataRate),
		FrequencyHz:     433920000,
		Modulation:      mod,
		DataRateBaud:    dataRate,
		DeviationHz:     10000,
		ChannelBWHz:     100000,
		SyncWord:        0xD391,
		SyncMode:        Sync16of16,
		PktLenMode:      PktLenVariable,
		PktLen:          60,
		PreambleBytes:   4,
		CRCEn:           true,
		DataWhiteningEn: true,
	}
}

// NewSyncModeVariant creates profiles with different sync word matching modes
// syncMode: one of the Sync* constants
// description: human-readable sync mode name
func NewSyncModeVariant(syncMode uint8, syncModeName string) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("enc-sync-%s", syncModeName),
		Description:   fmt.Sprintf("Sync mode test: %s", syncModeName),
		FrequencyHz:   433920000,
		Modulation:    ModGFSK,
		DataRateBaud:  38400,
		DeviationHz:   10000,
		ChannelBWHz:   100000,
		SyncWord:      0xD391,
		SyncMode:      syncMode,
		PktLenMode:    PktLenVariable,
		PktLen:        60,
		PreambleBytes: 4,
		CRCEn:         true,
	}
}

// NewPreambleLengthVariant creates profiles with different preamble lengths
// preambleBytes: number of preamble bytes (0-8)
func NewPreambleLengthVariant(preambleBytes uint8) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("enc-preamble-%d", preambleBytes),
		Description:   fmt.Sprintf("Preamble length test: %d bytes", preambleBytes),
		FrequencyHz:   433920000,
		Modulation:    ModGFSK,
		DataRateBaud:  38400,
		DeviationHz:   10000,
		ChannelBWHz:   100000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        60,
		PreambleBytes: preambleBytes,
		CRCEn:         true,
	}
}

// NewFECVariant creates profiles with FEC enabled
// dataRate: data rate in baud
func NewFECVariant(dataRate float64) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("enc-fec-%s", formatDataRate(dataRate)),
		Description:   fmt.Sprintf("FEC enabled test at %.0f baud", dataRate),
		FrequencyHz:   433920000,
		Modulation:    ModGFSK,
		DataRateBaud:  dataRate,
		DeviationHz:   10000,
		ChannelBWHz:   100000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        60,
		PreambleBytes: 4,
		CRCEn:         true,
		FECEn:         true,
	}
}

// NewFullEncodingStack creates a profile with all encoding features enabled
// Tests Manchester + Whitening + FEC + CRC together
func NewFullEncodingStack() *Profile {
	return &Profile{
		Name:            "enc-full-stack",
		Description:     "Full encoding stack: Manchester + Whitening + FEC + CRC",
		FrequencyHz:     433920000,
		Modulation:      Mod2FSK, // Note: Manchester compatible with 2-FSK
		DataRateBaud:    9600,    // Lower rate for full stack
		DeviationHz:     5000,
		ChannelBWHz:     58000,
		SyncWord:        0xD391,
		SyncMode:        Sync16of16,
		PktLenMode:      PktLenVariable,
		PktLen:          60,
		PreambleBytes:   8, // Extra preamble for reliability
		CRCEn:           true,
		FECEn:           true,
		DataWhiteningEn: true,
		ManchesterEn:    true,
	}
}

// GenerateEncodingProfiles generates all encoding variation profiles
func GenerateEncodingProfiles(basePath string) error {
	profiles := []*Profile{
		// Manchester encoding variants
		NewManchesterVariant("ook", 4800),
		NewManchesterVariant("2fsk", 9600),
		NewManchesterVariant("gfsk", 19200),

		// Data whitening variants
		NewWhiteningVariant("2fsk", 19200),
		NewWhiteningVariant("gfsk", 38400),

		// Sync mode variants
		NewSyncModeVariant(Sync15of16, "15of16"),
		NewSyncModeVariant(Sync16of16, "16of16"),
		NewSyncModeVariant(Sync30of32, "30of32"),

		// Preamble length variants
		NewPreambleLengthVariant(2),
		NewPreambleLengthVariant(4),
		NewPreambleLengthVariant(8),

		// FEC variants
		NewFECVariant(9600),
		NewFECVariant(38400),

		// Full encoding stack
		NewFullEncodingStack(),
	}

	if err := EnsureDir(basePath + "/dummy"); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	for _, p := range profiles {
		filename := fmt.Sprintf("%s/%s.json", basePath, p.Name)
		if err := p.SaveToFile(filename); err != nil {
			return fmt.Errorf("failed to save profile %s: %w", p.Name, err)
		}
	}

	return nil
}
