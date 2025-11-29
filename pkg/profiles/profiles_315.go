package profiles

import "fmt"

// 315 MHz Band Profile Factories
// These create profiles for the 315 MHz ISM band, commonly used for key fobs and garage doors.

// New315OOKLow creates a 315 MHz OOK profile at the specified data rate
// Suitable for key fobs and garage door openers
// dataRate: 1200, 2400, or 4800 baud
func New315OOKLow(dataRate float64) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("315-ook-low-%s", formatDataRate(dataRate)),
		Description:   fmt.Sprintf("315 MHz ASK/OOK at %.0f baud for key fobs/garage doors", dataRate),
		FrequencyHz:   315000000,
		Modulation:    ModASKOOK,
		DataRateBaud:  dataRate,
		ChannelBWHz:   58000, // 58 kHz narrow bandwidth
		SyncWord:      0x0000,
		SyncMode:      SyncNone,
		PktLenMode:    PktLenFixed,
		PktLen:        64,
		PreambleBytes: 4,
		CRCEn:         false,
	}
}

// New315OOKFast creates a 315 MHz OOK profile for fast remotes
// dataRate: 9600 or 19200 baud
func New315OOKFast(dataRate float64) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("315-ook-fast-%s", formatDataRate(dataRate)),
		Description:   fmt.Sprintf("315 MHz ASK/OOK at %.0f baud for fast remotes", dataRate),
		FrequencyHz:   315000000,
		Modulation:    ModASKOOK,
		DataRateBaud:  dataRate,
		ChannelBWHz:   100000, // 100 kHz wider bandwidth for higher rate
		SyncWord:      0x0000,
		SyncMode:      SyncNone,
		PktLenMode:    PktLenFixed,
		PktLen:        64,
		PreambleBytes: 4,
		CRCEn:         false,
	}
}

// New315FSKSync creates a 315 MHz 2-FSK profile with sync word
// Suitable for bidirectional digital sensors
// dataRate: 2400, 4800, or 9600 baud
// fecEnabled: enable forward error correction
func New315FSKSync(dataRate float64, fecEnabled bool) *Profile {
	name := fmt.Sprintf("315-2fsk-sync-%s", formatDataRate(dataRate))
	if fecEnabled {
		name += "-fec"
	}

	return &Profile{
		Name:          name,
		Description:   fmt.Sprintf("315 MHz 2-FSK at %.0f baud with sync for bidirectional sensors", dataRate),
		FrequencyHz:   315000000,
		Modulation:    Mod2FSK,
		DataRateBaud:  dataRate,
		DeviationHz:   dataRate * 0.5, // Deviation = half data rate (standard for FSK)
		ChannelBWHz:   58000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        60, // Max length for variable mode
		PreambleBytes: 4,
		CRCEn:         true,
		FECEn:         fecEnabled,
	}
}

// formatDataRate formats a data rate for use in profile names
func formatDataRate(rate float64) string {
	if rate >= 1000000 {
		return fmt.Sprintf("%.0fM", rate/1000000)
	} else if rate >= 1000 {
		// Common rates like 1200, 2400, 4800, 9600, 19200, 38400, etc.
		k := rate / 1000
		if k == float64(int(k)) {
			return fmt.Sprintf("%.0fk", k)
		}
		return fmt.Sprintf("%.1fk", k)
	}
	return fmt.Sprintf("%.0f", rate)
}

// Generate315Profiles generates all 315 MHz band profile configurations
func Generate315Profiles(basePath string) error {
	profiles := []*Profile{
		// 315-OOK-Low variants
		New315OOKLow(1200),
		New315OOKLow(2400),
		New315OOKLow(4800),

		// 315-OOK-Fast variants
		New315OOKFast(9600),
		New315OOKFast(19200),

		// 315-FSK-Sync variants
		New315FSKSync(2400, false),
		New315FSKSync(4800, false),
		New315FSKSync(9600, false),
		New315FSKSync(4800, true), // With FEC
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
