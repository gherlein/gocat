package profiles

import "fmt"

// 868 MHz Band Profile Factories
// These create profiles for the 868 MHz European ISM band.

// New868OOKSimple creates an 868 MHz OOK profile for simple remotes
// dataRate: 1200, 4800, or 9600 baud
func New868OOKSimple(dataRate float64) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("868-ook-simple-%s", formatDataRate(dataRate)),
		Description:   fmt.Sprintf("868 MHz ASK/OOK at %.0f baud for simple remotes", dataRate),
		FrequencyHz:   868300000,
		Modulation:    ModASKOOK,
		DataRateBaud:  dataRate,
		ChannelBWHz:   100000,
		SyncWord:      0x0000,
		SyncMode:      SyncNone,
		PktLenMode:    PktLenFixed,
		PktLen:        64,
		PreambleBytes: 4,
		CRCEn:         false,
	}
}

// New868FSKManchester creates an 868 MHz 2-FSK profile with Manchester encoding
// For EU regulatory compliance
// dataRate: 4800, 9600, or 19200 baud
func New868FSKManchester(dataRate float64) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("868-2fsk-manch-%s", formatDataRate(dataRate)),
		Description:   fmt.Sprintf("868 MHz 2-FSK+Manchester at %.0f baud for EU compliance", dataRate),
		FrequencyHz:   868300000,
		Modulation:    Mod2FSK,
		DataRateBaud:  dataRate,
		DeviationHz:   5100, // 5.1 kHz deviation
		ChannelBWHz:   63000,
		SyncWord:      0xAAAA,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        60,
		PreambleBytes: 4,
		CRCEn:         true,
		ManchesterEn:  true,
	}
}

// New868FSKFast creates an 868 MHz 2-FSK profile for high-speed sensors
// dataRate: 38400, 76800, or 100000 baud
func New868FSKFast(dataRate float64) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("868-2fsk-fast-%s", formatDataRate(dataRate)),
		Description:   fmt.Sprintf("868 MHz 2-FSK at %.0f baud for high-speed sensors", dataRate),
		FrequencyHz:   868300000,
		Modulation:    Mod2FSK,
		DataRateBaud:  dataRate,
		DeviationHz:   25000, // 25 kHz deviation
		ChannelBWHz:   200000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        255,
		PreambleBytes: 4,
		CRCEn:         true,
	}
}

// New868GFSKSmart creates an 868 MHz GFSK profile for smart metering
// dataRate: 9600, 19200, or 38400 baud
func New868GFSKSmart(dataRate float64) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("868-gfsk-smart-%s", formatDataRate(dataRate)),
		Description:   fmt.Sprintf("868 MHz GFSK at %.0f baud for smart metering", dataRate),
		FrequencyHz:   868300000,
		Modulation:    ModGFSK,
		DataRateBaud:  dataRate,
		DeviationHz:   10000, // 10 kHz deviation
		ChannelBWHz:   100000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        60,
		PreambleBytes: 4,
		CRCEn:         true,
	}
}

// New868GFSKFEC creates an 868 MHz GFSK profile with FEC for robust industrial use
// dataRate: 19200 or 38400 baud
// whitening: enable data whitening
func New868GFSKFEC(dataRate float64, whitening bool) *Profile {
	name := fmt.Sprintf("868-gfsk-fec-%s", formatDataRate(dataRate))
	if whitening {
		name += "-white"
	}

	return &Profile{
		Name:            name,
		Description:     fmt.Sprintf("868 MHz GFSK+FEC at %.0f baud for robust industrial", dataRate),
		FrequencyHz:     868300000,
		Modulation:      ModGFSK,
		DataRateBaud:    dataRate,
		DeviationHz:     15000, // 15 kHz deviation
		ChannelBWHz:     150000,
		SyncWord:        0xD391,
		SyncMode:        Sync16of16,
		PktLenMode:      PktLenVariable,
		PktLen:          60,
		PreambleBytes:   4,
		CRCEn:           true,
		FECEn:           true,
		DataWhiteningEn: whitening,
	}
}

// Generate868Profiles generates all 868 MHz band profile configurations
func Generate868Profiles(basePath string) error {
	profiles := []*Profile{
		// 868-OOK-Simple variants
		New868OOKSimple(1200),
		New868OOKSimple(4800),
		New868OOKSimple(9600),

		// 868-2FSK-Manch variants
		New868FSKManchester(4800),
		New868FSKManchester(9600),
		New868FSKManchester(19200),

		// 868-2FSK-Fast variants
		New868FSKFast(38400),
		New868FSKFast(76800),
		New868FSKFast(100000),

		// 868-GFSK-Smart variants
		New868GFSKSmart(9600),
		New868GFSKSmart(19200),
		New868GFSKSmart(38400),

		// 868-GFSK-FEC variants
		New868GFSKFEC(19200, false),
		New868GFSKFEC(38400, false),
		New868GFSKFEC(19200, true), // With whitening
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
