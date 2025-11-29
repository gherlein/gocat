package profiles

import "fmt"

// Special Multi-Band Profile Factories
// These create specialized profiles for specific use cases that work across
// multiple frequency bands.

// NewLongRange creates a long-range profile for maximum distance
// Uses low data rate and narrow bandwidth for best sensitivity
// band: "315", "433", "868", or "915"
func NewLongRange(band string) *Profile {
	var freq float64
	switch band {
	case "315":
		freq = 315000000
	case "433":
		freq = 433920000
	case "868":
		freq = 868300000
	case "915":
		freq = 915000000
	default:
		freq = 433920000
	}

	return &Profile{
		Name:          fmt.Sprintf("%s-longrange", band),
		Description:   fmt.Sprintf("%s MHz long-range GFSK at 1.2k baud", band),
		FrequencyHz:   freq,
		Modulation:    ModGFSK,
		DataRateBaud:  1200, // Very low rate for best range
		DeviationHz:   5000, // Narrow deviation
		ChannelBWHz:   58000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        60,
		PreambleBytes: 8, // Extra preamble for better sync
		CRCEn:         true,
		FECEn:         true, // FEC for reliability
	}
}

// NewHighSpeed creates a high-speed profile for maximum throughput
// Uses maximum data rate with wider bandwidth
// band: "315", "433", "868", or "915"
func NewHighSpeed(band string) *Profile {
	var freq float64
	switch band {
	case "315":
		freq = 315000000
	case "433":
		freq = 433920000
	case "868":
		freq = 868300000
	case "915":
		freq = 915000000
	default:
		freq = 433920000
	}

	return &Profile{
		Name:          fmt.Sprintf("%s-highspeed", band),
		Description:   fmt.Sprintf("%s MHz high-speed 2-FSK at 500k baud", band),
		FrequencyHz:   freq,
		Modulation:    Mod2FSK,
		DataRateBaud:  500000, // Maximum rate
		DeviationHz:   150000, // Wide deviation for high rate
		ChannelBWHz:   812000, // Wide bandwidth
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        255,
		PreambleBytes: 4,
		CRCEn:         true,
	}
}

// NewRobust creates a robust profile with all error protection enabled
// Uses FEC, CRC, and data whitening for maximum reliability
// band: "315", "433", "868", or "915"
func NewRobust(band string) *Profile {
	var freq float64
	switch band {
	case "315":
		freq = 315000000
	case "433":
		freq = 433920000
	case "868":
		freq = 868300000
	case "915":
		freq = 915000000
	default:
		freq = 433920000
	}

	return &Profile{
		Name:            fmt.Sprintf("%s-robust", band),
		Description:     fmt.Sprintf("%s MHz robust GFSK with FEC+CRC+whitening", band),
		FrequencyHz:     freq,
		Modulation:      ModGFSK,
		DataRateBaud:    19200,
		DeviationHz:     10000,
		ChannelBWHz:     100000,
		SyncWord:        0xD391,
		SyncMode:        Sync16of16,
		PktLenMode:      PktLenVariable,
		PktLen:          60,
		PreambleBytes:   8,
		CRCEn:           true,
		FECEn:           true,
		DataWhiteningEn: true,
	}
}

// NewSpectrumMonitor creates a wide-bandwidth RX profile for spectrum monitoring
// Uses wide bandwidth for broader frequency monitoring
// centerFreq: center frequency in Hz (e.g., 433920000)
func NewSpectrumMonitor(centerFreq float64) *Profile {
	band := "custom"
	if centerFreq >= 300e6 && centerFreq < 350e6 {
		band = "315"
	} else if centerFreq >= 400e6 && centerFreq < 470e6 {
		band = "433"
	} else if centerFreq >= 800e6 && centerFreq < 870e6 {
		band = "868"
	} else if centerFreq >= 900e6 && centerFreq < 930e6 {
		band = "915"
	}

	return &Profile{
		Name:          fmt.Sprintf("%s-spectrum-mon", band),
		Description:   fmt.Sprintf("%.0f MHz spectrum monitor (wide BW)", centerFreq/1e6),
		FrequencyHz:   centerFreq,
		Modulation:    Mod2FSK, // FSK for carrier detection
		DataRateBaud:  100000,  // High rate for fast sampling
		DeviationHz:   50000,
		ChannelBWHz:   500000, // Wide bandwidth (but not maximum)
		SyncWord:      0xD391,
		SyncMode:      Sync15of16, // Lenient sync matching
		PktLenMode:    PktLenFixed,
		PktLen:        255,
		PreambleBytes: 2, // Minimal preamble
		CRCEn:         false,
	}
}

// NewBalanced creates a balanced profile with good range and throughput
// Good middle-ground for general use
// band: "315", "433", "868", or "915"
func NewBalanced(band string) *Profile {
	var freq float64
	switch band {
	case "315":
		freq = 315000000
	case "433":
		freq = 433920000
	case "868":
		freq = 868300000
	case "915":
		freq = 915000000
	default:
		freq = 433920000
	}

	return &Profile{
		Name:          fmt.Sprintf("%s-balanced", band),
		Description:   fmt.Sprintf("%s MHz balanced GFSK at 38.4k baud", band),
		FrequencyHz:   freq,
		Modulation:    ModGFSK,
		DataRateBaud:  38400,
		DeviationHz:   20000,
		ChannelBWHz:   100000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        60,
		PreambleBytes: 4,
		CRCEn:         true,
	}
}

// New4FSKHighThroughput creates a 4-FSK profile for maximum throughput
// 4-FSK achieves 2 bits per symbol for higher data rates
// band: "433", "868", or "915" (not recommended for 315)
func New4FSKHighThroughput(band string) *Profile {
	var freq float64
	switch band {
	case "433":
		freq = 433920000
	case "868":
		freq = 868300000
	case "915":
		freq = 915000000
	default:
		freq = 433920000
	}

	return &Profile{
		Name:          fmt.Sprintf("%s-4fsk-high", band),
		Description:   fmt.Sprintf("%s MHz 4-FSK at 200k baud for high throughput", band),
		FrequencyHz:   freq,
		Modulation:    Mod4FSK,
		DataRateBaud:  200000,
		DeviationHz:   25000, // Inner deviation
		ChannelBWHz:   200000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        255,
		PreambleBytes: 4,
		CRCEn:         true,
	}
}

// NewMSKStandard creates an MSK profile for efficient spectrum use
// MSK provides continuous phase FSK with efficient bandwidth
// band: "433", "868", or "915"
func NewMSKStandard(band string) *Profile {
	var freq float64
	switch band {
	case "433":
		freq = 433920000
	case "868":
		freq = 868300000
	case "915":
		freq = 915000000
	default:
		freq = 433920000
	}

	return &Profile{
		Name:          fmt.Sprintf("%s-msk-std", band),
		Description:   fmt.Sprintf("%s MHz MSK at 100k baud", band),
		FrequencyHz:   freq,
		Modulation:    ModMSK,
		DataRateBaud:  100000,
		DeviationHz:   0, // MSK deviation is derived from data rate
		ChannelBWHz:   150000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        60,
		PreambleBytes: 4,
		CRCEn:         true,
	}
}

// GenerateSpecialProfiles generates all special profile configurations
func GenerateSpecialProfiles(basePath string) error {
	profiles := []*Profile{
		// LongRange profiles for each band
		NewLongRange("315"),
		NewLongRange("433"),
		NewLongRange("868"),
		NewLongRange("915"),

		// HighSpeed profiles for each band
		NewHighSpeed("315"),
		NewHighSpeed("433"),
		NewHighSpeed("868"),
		NewHighSpeed("915"),

		// Robust profiles for each band
		NewRobust("315"),
		NewRobust("433"),
		NewRobust("868"),
		NewRobust("915"),

		// Balanced profiles for each band
		NewBalanced("315"),
		NewBalanced("433"),
		NewBalanced("868"),
		NewBalanced("915"),

		// Spectrum monitor profiles
		NewSpectrumMonitor(315000000),
		NewSpectrumMonitor(433920000),
		NewSpectrumMonitor(868300000),
		NewSpectrumMonitor(915000000),

		// 4-FSK high throughput profiles
		New4FSKHighThroughput("433"),
		New4FSKHighThroughput("868"),
		New4FSKHighThroughput("915"),

		// MSK profiles
		NewMSKStandard("433"),
		NewMSKStandard("868"),
		NewMSKStandard("915"),
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
