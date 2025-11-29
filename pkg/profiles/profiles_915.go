package profiles

import "fmt"

// 915 MHz Band Profile Factories
// These create profiles for the 902-928 MHz US ISM band.

// New915OOKTPMS creates a 915 MHz OOK profile for TPMS and simple sensors
// dataRate: 4800, 9600, or 19200 baud
// syncEnabled: use sync word instead of no sync
func New915OOKTPMS(dataRate float64, syncEnabled bool) *Profile {
	name := fmt.Sprintf("915-ook-tpms-%s", formatDataRate(dataRate))
	syncMode := SyncNone
	if syncEnabled {
		name += "-sync"
		syncMode = Sync15of16
	} else {
		name += "-nosync"
	}

	return &Profile{
		Name:          name,
		Description:   fmt.Sprintf("915 MHz ASK/OOK at %.0f baud for TPMS", dataRate),
		FrequencyHz:   915000000,
		Modulation:    ModASKOOK,
		DataRateBaud:  dataRate,
		ChannelBWHz:   100000,
		SyncWord:      0xD391,
		SyncMode:      uint8(syncMode),
		PktLenMode:    PktLenFixed,
		PktLen:        64,
		PreambleBytes: 4,
		CRCEn:         false,
	}
}

// New915FSKSensor creates a 915 MHz 2-FSK profile for wireless sensors
// dataRate: 9600, 19200, or 38400 baud
func New915FSKSensor(dataRate float64) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("915-2fsk-sensor-%s", formatDataRate(dataRate)),
		Description:   fmt.Sprintf("915 MHz 2-FSK at %.0f baud for wireless sensors", dataRate),
		FrequencyHz:   915000000,
		Modulation:    Mod2FSK,
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

// New915GFSKStandard creates a 915 MHz GFSK profile for standard digital links
// whitening: enable data whitening
func New915GFSKStandard(whitening bool) *Profile {
	name := "915-gfsk-std-38.4k"
	if whitening {
		name += "-white"
	}

	return &Profile{
		Name:            name,
		Description:     "915 MHz GFSK at 38.4k baud for standard digital links",
		FrequencyHz:     915000000,
		Modulation:      ModGFSK,
		DataRateBaud:    38400,
		DeviationHz:     20000, // 20 kHz deviation
		ChannelBWHz:     94000,
		SyncWord:        0xD391,
		SyncMode:        Sync16of16,
		PktLenMode:      PktLenVariable,
		PktLen:          60,
		PreambleBytes:   4,
		CRCEn:           true,
		DataWhiteningEn: whitening,
	}
}

// New915GFSKCRCFEC creates a 915 MHz GFSK profile with CRC and FEC
// dataRate: 38400, 76800, or 100000 baud
func New915GFSKCRCFEC(dataRate float64) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("915-gfsk-crc-fec-%s", formatDataRate(dataRate)),
		Description:   fmt.Sprintf("915 MHz GFSK+CRC+FEC at %.0f baud for robust sensors", dataRate),
		FrequencyHz:   915000000,
		Modulation:    ModGFSK,
		DataRateBaud:  dataRate,
		DeviationHz:   25000, // 25 kHz deviation
		ChannelBWHz:   150000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        60,
		PreambleBytes: 4,
		CRCEn:         true,
		FECEn:         true,
	}
}

// New915FHSS creates a 915 MHz GFSK profile for frequency hopping
// This is a base config - FHSS channel list would be set separately
// dataRate: 100000 or 250000 baud
// isMaster: true for master, false for slave
func New915FHSS(dataRate float64, isMaster bool) *Profile {
	role := "slave"
	if isMaster {
		role = "master"
	}

	return &Profile{
		Name:          fmt.Sprintf("915-fhss-%s-%s", formatDataRate(dataRate), role),
		Description:   fmt.Sprintf("915 MHz GFSK FHSS at %.0f baud (%s)", dataRate, role),
		FrequencyHz:   915000000, // Base frequency, will hop
		Modulation:    ModGFSK,
		DataRateBaud:  dataRate,
		DeviationHz:   50000, // 50 kHz deviation for wider signal
		ChannelBWHz:   300000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        255,
		PreambleBytes: 4,
		CRCEn:         true,
	}
}

// New915Max creates a 915 MHz 2-FSK profile for maximum throughput
// dataRate: 250000 or 500000 baud
func New915Max(dataRate float64) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("915-2fsk-max-%s", formatDataRate(dataRate)),
		Description:   fmt.Sprintf("915 MHz 2-FSK at %.0f baud for max throughput", dataRate),
		FrequencyHz:   915000000,
		Modulation:    Mod2FSK,
		DataRateBaud:  dataRate,
		DeviationHz:   100000, // 100 kHz deviation
		ChannelBWHz:   500000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        255,
		PreambleBytes: 4,
		CRCEn:         true,
	}
}

// Generate915Profiles generates all 915 MHz band profile configurations
func Generate915Profiles(basePath string) error {
	profiles := []*Profile{
		// 915-OOK-TPMS variants
		New915OOKTPMS(4800, false),
		New915OOKTPMS(9600, false),
		New915OOKTPMS(19200, false),
		New915OOKTPMS(9600, true), // With sync

		// 915-2FSK-Sensor variants
		New915FSKSensor(9600),
		New915FSKSensor(19200),
		New915FSKSensor(38400),

		// 915-GFSK-Standard variants
		New915GFSKStandard(false),
		New915GFSKStandard(true), // With whitening

		// 915-GFSK-CRC-FEC variants
		New915GFSKCRCFEC(38400),
		New915GFSKCRCFEC(76800),
		New915GFSKCRCFEC(100000),

		// 915-FHSS variants
		New915FHSS(100000, true),  // Master
		New915FHSS(100000, false), // Slave
		New915FHSS(250000, true),  // Master
		New915FHSS(250000, false), // Slave

		// 915-Max variants
		New915Max(250000),
		New915Max(500000),
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
