package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ScanConfig defines runtime scanning parameters
type ScanConfig struct {
	// Frequency lists
	CoarseFrequencies []uint32 // Hz - frequencies for coarse scan

	// Scan parameters
	RSSIThreshold float32       // dBm - minimum signal detection threshold
	FineScanRange uint32        // Hz - range around detected signal (± this value)
	FineScanStep  uint32        // Hz - step size for fine scan
	DwellTime     time.Duration // Time to wait for RSSI measurement
	ScanInterval  time.Duration // Delay between scan cycles

	// Signal tracking
	HoldMax             int    // Maximum hold counter value
	LostThreshold       int    // Counter value when signal is considered lost
	FrequencyResolution uint32 // Hz - grouping resolution for signals

	// Smoothing
	SmoothingEnabled bool
	SmoothThreshold  float64
	SmoothKFast      float64
	SmoothKSlow      float64

	// Callbacks (optional, not serialized)
	OnSignalDetected func(info *SignalInfo) `json:"-"`
	OnSignalLost     func(info *SignalInfo) `json:"-"`

	// Debug callback (optional)
	DebugLog func(format string, args ...interface{}) `json:"-"`
}

// DefaultConfig returns a ScanConfig with default values
func DefaultConfig() *ScanConfig {
	return &ScanConfig{
		CoarseFrequencies:   DefaultFrequencies,
		RSSIThreshold:       DefaultRSSIThreshold,
		FineScanRange:       DefaultFineScanRange,
		FineScanStep:        DefaultFineScanStep,
		DwellTime:           DefaultDwellTime,
		ScanInterval:        DefaultScanInterval,
		HoldMax:             DefaultHoldMax,
		LostThreshold:       DefaultLostThreshold,
		FrequencyResolution: DefaultFrequencyResolution,
		SmoothingEnabled:    true,
		SmoothThreshold:     DefaultSmoothThreshold,
		SmoothKFast:         DefaultKFast,
		SmoothKSlow:         DefaultKSlow,
	}
}

// Validate checks the configuration for errors
func (c *ScanConfig) Validate() error {
	if len(c.CoarseFrequencies) == 0 {
		return ErrNoFrequencies
	}

	for _, freq := range c.CoarseFrequencies {
		if !IsValidFrequency(freq) {
			return fmt.Errorf("%w: %d Hz", ErrFrequencyOutOfRange, freq)
		}
	}

	if c.RSSIThreshold > 0 {
		return ErrInvalidThreshold
	}

	if c.DwellTime < time.Millisecond || c.DwellTime > 100*time.Millisecond {
		return ErrInvalidDwellTime
	}

	return nil
}

// --- JSON Configuration File Types ---

// ConfigFile represents the JSON configuration file structure
type ConfigFile struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Created     time.Time `json:"created"`

	Frequencies    FrequencyConfigJSON `json:"frequencies"`
	ScanParameters ScanParametersJSON  `json:"scan_parameters"`
	SignalTracking SignalTrackingJSON  `json:"signal_tracking"`
	Smoothing      SmoothingJSON       `json:"smoothing"`
	RadioPresets   RadioPresetsJSON    `json:"radio_presets"`
	Output         OutputConfigJSON    `json:"output"`
}

// FrequencyConfigJSON defines frequency lists and bands in JSON
type FrequencyConfigJSON struct {
	Coarse []uint32         `json:"coarse"`
	Hopper []uint32         `json:"hopper,omitempty"`
	Bands  []BandConfigJSON `json:"bands,omitempty"`
}

// BandConfigJSON defines a frequency band for scanning
type BandConfigJSON struct {
	Name    string `json:"name"`
	StartHz uint32 `json:"start_hz"`
	EndHz   uint32 `json:"end_hz"`
	StepHz  uint32 `json:"step_hz"`
	Enabled bool   `json:"enabled"`
}

// ScanParametersJSON holds scan timing and threshold settings
type ScanParametersJSON struct {
	RSSIThresholdDBm float32 `json:"rssi_threshold_dbm"`
	FineScanRangeHz  uint32  `json:"fine_scan_range_hz"`
	FineScanStepHz   uint32  `json:"fine_scan_step_hz"`
	DwellTimeMs      uint32  `json:"dwell_time_ms"`
	ScanIntervalMs   uint32  `json:"scan_interval_ms"`
}

// SignalTrackingJSON holds signal detection hysteresis settings
type SignalTrackingJSON struct {
	HoldMax               int    `json:"hold_max"`
	LostThreshold         int    `json:"lost_threshold"`
	FrequencyResolutionHz uint32 `json:"frequency_resolution_hz"`
}

// SmoothingJSON holds frequency smoothing algorithm settings
type SmoothingJSON struct {
	Enabled     bool    `json:"enabled"`
	ThresholdHz float64 `json:"threshold_hz"`
	KFast       float64 `json:"k_fast"`
	KSlow       float64 `json:"k_slow"`
}

// RadioPresetsJSON holds register values for scan presets
type RadioPresetsJSON struct {
	Coarse RegisterOverridesJSON `json:"coarse"`
	Fine   RegisterOverridesJSON `json:"fine"`
}

// RegisterOverridesJSON allows partial register configuration
type RegisterOverridesJSON struct {
	MDMCFG4  *uint8 `json:"mdmcfg4,omitempty"`
	MDMCFG3  *uint8 `json:"mdmcfg3,omitempty"`
	MDMCFG2  *uint8 `json:"mdmcfg2,omitempty"`
	MDMCFG1  *uint8 `json:"mdmcfg1,omitempty"`
	MDMCFG0  *uint8 `json:"mdmcfg0,omitempty"`
	AGCCTRL2 *uint8 `json:"agcctrl2,omitempty"`
	AGCCTRL1 *uint8 `json:"agcctrl1,omitempty"`
	AGCCTRL0 *uint8 `json:"agcctrl0,omitempty"`
	FREND1   *uint8 `json:"frend1,omitempty"`
	FREND0   *uint8 `json:"frend0,omitempty"`
	FOCCFG   *uint8 `json:"foccfg,omitempty"`
	BSCFG    *uint8 `json:"bscfg,omitempty"`
}

// OutputConfigJSON defines signal logging options
type OutputConfigJSON struct {
	LogSignals bool   `json:"log_signals"`
	LogPath    string `json:"log_path,omitempty"`
	LogFormat  string `json:"log_format,omitempty"` // csv, json, text
}

// LoadConfigFile loads scanner configuration from a JSON file
func LoadConfigFile(path string) (*ConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ConfigFile
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// Validate checks the configuration file for errors
func (c *ConfigFile) Validate() error {
	if c.Version != "1.0" {
		return fmt.Errorf("%w: %s", ErrConfigVersion, c.Version)
	}

	if len(c.Frequencies.Coarse) == 0 && len(c.Frequencies.Bands) == 0 {
		return ErrNoFrequencies
	}

	for _, freq := range c.Frequencies.Coarse {
		if !IsValidFrequency(freq) {
			return fmt.Errorf("%w: %d Hz", ErrFrequencyOutOfRange, freq)
		}
	}

	if c.ScanParameters.RSSIThresholdDBm > 0 {
		return ErrInvalidThreshold
	}

	if c.ScanParameters.DwellTimeMs < 1 || c.ScanParameters.DwellTimeMs > 100 {
		return ErrInvalidDwellTime
	}

	return nil
}

// ToScanConfig converts JSON config to runtime ScanConfig
func (c *ConfigFile) ToScanConfig() *ScanConfig {
	frequencies := c.Frequencies.Coarse
	if len(frequencies) == 0 {
		frequencies = c.expandBands()
	}

	dwellTime := time.Duration(c.ScanParameters.DwellTimeMs) * time.Millisecond
	if dwellTime == 0 {
		dwellTime = DefaultDwellTime
	}

	scanInterval := time.Duration(c.ScanParameters.ScanIntervalMs) * time.Millisecond
	if scanInterval == 0 {
		scanInterval = DefaultScanInterval
	}

	holdMax := c.SignalTracking.HoldMax
	if holdMax == 0 {
		holdMax = DefaultHoldMax
	}

	lostThreshold := c.SignalTracking.LostThreshold
	if lostThreshold == 0 {
		lostThreshold = DefaultLostThreshold
	}

	freqResolution := c.SignalTracking.FrequencyResolutionHz
	if freqResolution == 0 {
		freqResolution = DefaultFrequencyResolution
	}

	smoothThreshold := c.Smoothing.ThresholdHz
	if smoothThreshold == 0 {
		smoothThreshold = DefaultSmoothThreshold
	}

	kFast := c.Smoothing.KFast
	if kFast == 0 {
		kFast = DefaultKFast
	}

	kSlow := c.Smoothing.KSlow
	if kSlow == 0 {
		kSlow = DefaultKSlow
	}

	return &ScanConfig{
		CoarseFrequencies:   frequencies,
		RSSIThreshold:       c.ScanParameters.RSSIThresholdDBm,
		FineScanRange:       c.ScanParameters.FineScanRangeHz,
		FineScanStep:        c.ScanParameters.FineScanStepHz,
		DwellTime:           dwellTime,
		ScanInterval:        scanInterval,
		HoldMax:             holdMax,
		LostThreshold:       lostThreshold,
		FrequencyResolution: freqResolution,
		SmoothingEnabled:    c.Smoothing.Enabled,
		SmoothThreshold:     smoothThreshold,
		SmoothKFast:         kFast,
		SmoothKSlow:         kSlow,
	}
}

// expandBands generates frequency list from band definitions
func (c *ConfigFile) expandBands() []uint32 {
	var freqs []uint32
	for _, band := range c.Frequencies.Bands {
		if !band.Enabled {
			continue
		}
		for freq := band.StartHz; freq <= band.EndHz; freq += band.StepHz {
			if IsValidFrequency(freq) {
				freqs = append(freqs, freq)
			}
		}
	}
	return freqs
}

// GetCoarsePreset returns register values for coarse scan from config or defaults
func (c *ConfigFile) GetCoarsePreset() *RegisterOverridesJSON {
	return &c.RadioPresets.Coarse
}

// GetFinePreset returns register values for fine scan from config or defaults
func (c *ConfigFile) GetFinePreset() *RegisterOverridesJSON {
	return &c.RadioPresets.Fine
}

// SaveConfigFile saves scanner configuration to a JSON file
func SaveConfigFile(config *ConfigFile, path string) error {
	config.Created = time.Now()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
