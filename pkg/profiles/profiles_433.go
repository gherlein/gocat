package profiles

import "fmt"

// 433 MHz Band Profile Factories
// These create profiles for the 433 MHz ISM band, commonly used for key fobs,
// remotes, and wireless sensors.

// New433OOKKeyfob creates a 433 MHz OOK profile at the specified data rate
// Suitable for key fobs and simple remotes
// dataRate: 1200, 2400, or 4800 baud
func New433OOKKeyfob(dataRate float64) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("433-ook-keyfob-%s", formatDataRate(dataRate)),
		Description:   fmt.Sprintf("433 MHz ASK/OOK at %.0f baud for key fobs", dataRate),
		FrequencyHz:   433920000,
		Modulation:    ModASKOOK,
		DataRateBaud:  dataRate,
		ChannelBWHz:   58000,
		SyncWord:      0x0000,
		SyncMode:      SyncNone,
		PktLenMode:    PktLenFixed,
		PktLen:        64,
		PreambleBytes: 4,
		CRCEn:         false,
	}
}

// New433OOKPWM creates a 433 MHz OOK profile for PWM-encoded remotes
// dataRate: 2400 or 4800 baud
func New433OOKPWM(dataRate float64) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("433-ook-pwm-%s", formatDataRate(dataRate)),
		Description:   fmt.Sprintf("433 MHz ASK/OOK at %.0f baud for PWM remotes", dataRate),
		FrequencyHz:   433920000,
		Modulation:    ModASKOOK,
		DataRateBaud:  dataRate,
		ChannelBWHz:   58000,
		SyncWord:      0x0000,
		SyncMode:      SyncNone,
		PktLenMode:    PktLenFixed,
		PktLen:        64,
		PreambleBytes: 4,
		CRCEn:         false,
	}
}

// New433OOKManchester creates a 433 MHz OOK profile with Manchester encoding
// dataRate: 4800 or 9600 baud
func New433OOKManchester(dataRate float64) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("433-ook-manch-%s", formatDataRate(dataRate)),
		Description:   fmt.Sprintf("433 MHz ASK/OOK at %.0f baud with Manchester encoding", dataRate),
		FrequencyHz:   433920000,
		Modulation:    ModASKOOK,
		DataRateBaud:  dataRate,
		ChannelBWHz:   100000, // Wider bandwidth for higher rate
		SyncWord:      0x0000,
		SyncMode:      SyncNone,
		PktLenMode:    PktLenFixed,
		PktLen:        64,
		PreambleBytes: 4,
		CRCEn:         false,
		ManchesterEn:  true,
	}
}

// New433FSKStandard creates a 433 MHz 2-FSK profile with sync word
// Suitable for digital sensors
// dataRate: 4800 or 9600 baud
// fecEnabled: enable forward error correction
func New433FSKStandard(dataRate float64, fecEnabled bool) *Profile {
	name := fmt.Sprintf("433-2fsk-std-%s", formatDataRate(dataRate))
	if fecEnabled {
		name += "-fec"
	}

	return &Profile{
		Name:          name,
		Description:   fmt.Sprintf("433 MHz 2-FSK at %.0f baud for digital sensors", dataRate),
		FrequencyHz:   433920000,
		Modulation:    Mod2FSK,
		DataRateBaud:  dataRate,
		DeviationHz:   5000, // 5 kHz deviation
		ChannelBWHz:   58000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        60,
		PreambleBytes: 4,
		CRCEn:         true,
		FECEn:         fecEnabled,
	}
}

// New433FSKFast creates a 433 MHz 2-FSK profile for high-speed links
// dataRate: 38400, 76800, or 100000 baud
func New433FSKFast(dataRate float64) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("433-2fsk-fast-%s", formatDataRate(dataRate)),
		Description:   fmt.Sprintf("433 MHz 2-FSK at %.0f baud for high-speed links", dataRate),
		FrequencyHz:   433920000,
		Modulation:    Mod2FSK,
		DataRateBaud:  dataRate,
		DeviationHz:   25000, // 25 kHz deviation for higher rates
		ChannelBWHz:   200000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        255,
		PreambleBytes: 4,
		CRCEn:         true,
	}
}

// New433GFSKCRC creates a 433 MHz GFSK profile for smart home devices
// dataRate: 9600, 19200, or 38400 baud
// fecEnabled: enable forward error correction
func New433GFSKCRC(dataRate float64, fecEnabled bool) *Profile {
	name := fmt.Sprintf("433-gfsk-crc-%s", formatDataRate(dataRate))
	if fecEnabled {
		name += "-fec"
	}

	return &Profile{
		Name:          name,
		Description:   fmt.Sprintf("433 MHz GFSK at %.0f baud for smart home devices", dataRate),
		FrequencyHz:   433920000,
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
		FECEn:         fecEnabled,
	}
}

// New4334FSK creates a 433 MHz 4-FSK profile for high-throughput
// dataRate: 50000, 100000, or 200000 baud
func New4334FSK(dataRate float64) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("433-4fsk-%s", formatDataRate(dataRate)),
		Description:   fmt.Sprintf("433 MHz 4-FSK at %.0f baud for high-throughput", dataRate),
		FrequencyHz:   433920000,
		Modulation:    Mod4FSK,
		DataRateBaud:  dataRate,
		DeviationHz:   25000, // Inner deviation
		ChannelBWHz:   200000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        255,
		PreambleBytes: 4,
		CRCEn:         true,
		// Note: Manchester encoding NOT supported with 4-FSK
	}
}

// Generate433Profiles generates all 433 MHz band profile configurations
func Generate433Profiles(basePath string) error {
	profiles := []*Profile{
		// 433-OOK-Keyfob variants
		New433OOKKeyfob(1200),
		New433OOKKeyfob(2400),
		New433OOKKeyfob(4800),

		// 433-OOK-PWM variants
		New433OOKPWM(2400),
		New433OOKPWM(4800),

		// 433-OOK-Manch variants
		New433OOKManchester(4800),
		New433OOKManchester(9600),

		// 433-2FSK-Standard variants
		New433FSKStandard(4800, false),
		New433FSKStandard(9600, false),
		New433FSKStandard(4800, true), // With FEC

		// 433-2FSK-Fast variants
		New433FSKFast(38400),
		New433FSKFast(76800),
		New433FSKFast(100000),

		// 433-GFSK-CRC variants
		New433GFSKCRC(9600, false),
		New433GFSKCRC(19200, false),
		New433GFSKCRC(38400, false),
		New433GFSKCRC(19200, true), // With FEC

		// 433-4FSK variants
		New4334FSK(50000),
		New4334FSK(100000),
		New4334FSK(200000),
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
