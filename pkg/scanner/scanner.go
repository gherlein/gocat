package scanner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/herlein/gocat/pkg/registers"
	"github.com/herlein/gocat/pkg/yardstick"
)

// Scanner provides frequency scanning capabilities
type Scanner interface {
	// Lifecycle
	Start() error
	Stop() error
	IsRunning() bool

	// Configuration
	SetConfig(config *ScanConfig) error
	GetConfig() *ScanConfig

	// Scanning
	ScanOnce() (*ScanResult, error)
	ScanContinuous(ctx context.Context, results chan<- *ScanResult) error

	// Signal tracking
	GetActiveSignals() []*SignalInfo
	ClearSignalHistory()
}

// scanner implements the Scanner interface
type scanner struct {
	device *yardstick.Device
	config *ScanConfig

	// State
	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}

	// Signal tracking
	tracker *SignalTracker

	// Smoothing
	smoother *FrequencySmoother

	// Radio preset values (from config or defaults)
	coarsePreset RegisterOverridesJSON
	finePreset   RegisterOverridesJSON

	// Saved radio config (to restore after scanning)
	savedConfig *registers.RegisterMap
}

// New creates a new Scanner with the given device and configuration
func New(device *yardstick.Device, config *ScanConfig) Scanner {
	if config == nil {
		config = DefaultConfig()
	}

	s := &scanner{
		device:   device,
		config:   config,
		stopChan: make(chan struct{}),
		tracker: NewSignalTracker(
			config.HoldMax,
			config.LostThreshold,
			config.FrequencyResolution,
		),
	}

	// Set up smoother
	if config.SmoothingEnabled {
		s.smoother = NewFrequencySmootherWithParams(
			config.SmoothThreshold,
			config.SmoothKFast,
			config.SmoothKSlow,
		)
	}

	// Set up callbacks
	s.tracker.SetCallbacks(config.OnSignalDetected, config.OnSignalLost)

	// Set default presets
	s.setDefaultPresets()

	return s
}

// debug logs a debug message if the debug callback is set
func (s *scanner) debug(format string, args ...interface{}) {
	if s.config.DebugLog != nil {
		s.config.DebugLog(format, args...)
	}
}

// NewFromConfigFile creates a Scanner from a JSON configuration file
func NewFromConfigFile(device *yardstick.Device, configPath string) (Scanner, error) {
	configFile, err := LoadConfigFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	config := configFile.ToScanConfig()

	s := &scanner{
		device:   device,
		config:   config,
		stopChan: make(chan struct{}),
		tracker: NewSignalTracker(
			config.HoldMax,
			config.LostThreshold,
			config.FrequencyResolution,
		),
	}

	// Set up smoother
	if config.SmoothingEnabled {
		s.smoother = NewFrequencySmootherWithParams(
			config.SmoothThreshold,
			config.SmoothKFast,
			config.SmoothKSlow,
		)
	}

	// Set up callbacks
	s.tracker.SetCallbacks(config.OnSignalDetected, config.OnSignalLost)

	// Apply presets from config file
	s.coarsePreset = *configFile.GetCoarsePreset()
	s.finePreset = *configFile.GetFinePreset()

	return s, nil
}

// setDefaultPresets sets the default radio presets for scanning
func (s *scanner) setDefaultPresets() {
	// Coarse preset defaults
	mdmcfg4 := CoarseMDMCFG4
	mdmcfg3 := CoarseMDMCFG3
	mdmcfg2 := CoarseMDMCFG2
	agcctrl2 := CoarseAGCCTRL2
	agcctrl1 := CoarseAGCCTRL1
	agcctrl0 := CoarseAGCCTRL0
	frend1 := CoarseFREND1
	frend0 := CoarseFREND0

	s.coarsePreset = RegisterOverridesJSON{
		MDMCFG4:  &mdmcfg4,
		MDMCFG3:  &mdmcfg3,
		MDMCFG2:  &mdmcfg2,
		AGCCTRL2: &agcctrl2,
		AGCCTRL1: &agcctrl1,
		AGCCTRL0: &agcctrl0,
		FREND1:   &frend1,
		FREND0:   &frend0,
	}

	// Fine preset defaults
	fMdmcfg4 := FineMDMCFG4
	fMdmcfg3 := FineMDMCFG3
	fMdmcfg2 := FineMDMCFG2
	fAgcctrl2 := FineAGCCTRL2
	fAgcctrl1 := FineAGCCTRL1
	fAgcctrl0 := FineAGCCTRL0
	fFrend1 := FineFREND1
	fFrend0 := FineFREND0

	s.finePreset = RegisterOverridesJSON{
		MDMCFG4:  &fMdmcfg4,
		MDMCFG3:  &fMdmcfg3,
		MDMCFG2:  &fMdmcfg2,
		AGCCTRL2: &fAgcctrl2,
		AGCCTRL1: &fAgcctrl1,
		AGCCTRL0: &fAgcctrl0,
		FREND1:   &fFrend1,
		FREND0:   &fFrend0,
	}
}

// Start begins continuous scanning in the background
func (s *scanner) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrScannerRunning
	}

	s.running = true
	s.stopChan = make(chan struct{})
	return nil
}

// Stop stops the scanner
func (s *scanner) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return ErrScannerNotRunning
	}

	close(s.stopChan)
	s.running = false
	return nil
}

// IsRunning returns true if the scanner is running
func (s *scanner) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// SetConfig updates the scanner configuration
func (s *scanner) SetConfig(config *ScanConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = config

	// Update tracker
	s.tracker = NewSignalTracker(
		config.HoldMax,
		config.LostThreshold,
		config.FrequencyResolution,
	)
	s.tracker.SetCallbacks(config.OnSignalDetected, config.OnSignalLost)

	// Update smoother
	if config.SmoothingEnabled {
		s.smoother = NewFrequencySmootherWithParams(
			config.SmoothThreshold,
			config.SmoothKFast,
			config.SmoothKSlow,
		)
	} else {
		s.smoother = nil
	}

	return nil
}

// GetConfig returns the current configuration
func (s *scanner) GetConfig() *ScanConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// ScanOnce performs a single scan cycle (coarse + fine if signal detected)
func (s *scanner) ScanOnce() (*ScanResult, error) {
	s.mu.RLock()
	config := s.config
	s.mu.RUnlock()

	s.debug("ScanOnce: starting scan cycle")

	// Perform coarse scan
	result, err := s.coarseScan(config)
	if err != nil {
		s.debug("ScanOnce: coarse scan failed: %v", err)
		return nil, fmt.Errorf("coarse scan failed: %w", err)
	}

	// If signal detected, perform fine scan
	if result.SignalDetected {
		s.debug("ScanOnce: signal detected at %.3f MHz, starting fine scan", float64(result.CoarseFrequency)/1e6)
		result, err = s.fineScan(config, result)
		if err != nil {
			s.debug("ScanOnce: fine scan failed: %v", err)
			return nil, fmt.Errorf("fine scan failed: %w", err)
		}

		// Apply smoothing if enabled
		if s.smoother != nil && result.FineFrequency > 0 {
			smoothed := s.smoother.Update(float64(result.FineFrequency))
			s.debug("ScanOnce: smoothed frequency %.3f -> %.3f MHz", float64(result.FineFrequency)/1e6, smoothed/1e6)
			result.FineFrequency = uint32(smoothed)
		}
	}

	// Update signal tracker
	s.tracker.Update(result)

	s.debug("ScanOnce: complete - detected=%v, freq=%.3f MHz, rssi=%.1f dBm",
		result.SignalDetected, float64(result.CoarseFrequency)/1e6, result.CoarseRSSI)

	return result, nil
}

// ScanContinuous performs continuous scanning until context is cancelled
func (s *scanner) ScanContinuous(ctx context.Context, results chan<- *ScanResult) error {
	if err := s.Start(); err != nil {
		return err
	}
	defer s.Stop()

	ticker := time.NewTicker(s.config.ScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(results)
			return ctx.Err()
		case <-s.stopChan:
			close(results)
			return nil
		case <-ticker.C:
			result, err := s.ScanOnce()
			if err != nil {
				// Log error but continue scanning
				continue
			}

			// Non-blocking send
			select {
			case results <- result:
			default:
				// Channel full, skip this result
			}
		}
	}
}

// GetActiveSignals returns all tracked signals
func (s *scanner) GetActiveSignals() []*SignalInfo {
	return s.tracker.GetAllSignals()
}

// ClearSignalHistory clears all tracked signals
func (s *scanner) ClearSignalHistory() {
	s.tracker.Clear()
	if s.smoother != nil {
		s.smoother.Reset()
	}
}

// coarseScan performs a wide-bandwidth scan across configured frequencies
func (s *scanner) coarseScan(config *ScanConfig) (*ScanResult, error) {
	result := &ScanResult{
		Timestamp:  time.Now(),
		CoarseRSSI: -200.0, // Very low initial value
	}

	s.debug("coarseScan: starting scan of %d frequencies, threshold=%.1f dBm", len(config.CoarseFrequencies), config.RSSIThreshold)

	// Load wide bandwidth preset
	if err := s.loadPreset(&s.coarsePreset); err != nil {
		s.debug("coarseScan: failed to load preset: %v", err)
		return nil, fmt.Errorf("failed to load coarse preset: %w", err)
	}
	s.debug("coarseScan: loaded coarse preset (MDMCFG4=0x%02X, MDMCFG2=0x%02X)",
		*s.coarsePreset.MDMCFG4, *s.coarsePreset.MDMCFG2)

	// Scan each frequency
	var scanErrors int
	for i, freq := range config.CoarseFrequencies {
		rssi, err := s.measureRSSI(freq, config.DwellTime)
		if err != nil {
			scanErrors++
			s.debug("coarseScan: [%d] %.3f MHz - ERROR: %v", i, float64(freq)/1e6, err)
			continue
		}

		s.debug("coarseScan: [%d] %.3f MHz = %.1f dBm", i, float64(freq)/1e6, rssi)

		if rssi > result.CoarseRSSI {
			result.CoarseRSSI = rssi
			result.CoarseFrequency = freq
		}
	}

	// Check threshold
	result.SignalDetected = result.CoarseRSSI >= config.RSSIThreshold

	s.debug("coarseScan: complete - best=%.3f MHz @ %.1f dBm, detected=%v, errors=%d",
		float64(result.CoarseFrequency)/1e6, result.CoarseRSSI, result.SignalDetected, scanErrors)

	return result, nil
}

// fineScan performs a narrow-bandwidth scan around the detected frequency
func (s *scanner) fineScan(config *ScanConfig, coarseResult *ScanResult) (*ScanResult, error) {
	if !coarseResult.SignalDetected {
		return coarseResult, nil
	}

	// Load narrow bandwidth preset
	if err := s.loadPreset(&s.finePreset); err != nil {
		return nil, fmt.Errorf("failed to load fine preset: %w", err)
	}

	center := coarseResult.CoarseFrequency
	startFreq := center - config.FineScanRange
	endFreq := center + config.FineScanRange

	var maxRSSI float32 = -200.0
	var maxFreq uint32 = 0

	// Scan the range
	for freq := startFreq; freq <= endFreq; freq += config.FineScanStep {
		// Skip invalid frequencies
		if !IsValidFrequency(freq) {
			continue
		}

		rssi, err := s.measureRSSI(freq, config.DwellTime)
		if err != nil {
			continue
		}

		if rssi > maxRSSI {
			maxRSSI = rssi
			maxFreq = freq
		}
	}

	coarseResult.FineFrequency = maxFreq
	coarseResult.FineRSSI = maxRSSI

	return coarseResult, nil
}

// measureRSSI measures the RSSI at a specific frequency
func (s *scanner) measureRSSI(freqHz uint32, dwellTime time.Duration) (float32, error) {
	// 1. Go to IDLE
	if err := s.device.StrobeModeIDLE(); err != nil {
		return 0, fmt.Errorf("failed to set IDLE: %w", err)
	}

	// 2. Set frequency
	if err := s.setFrequency(freqHz); err != nil {
		return 0, fmt.Errorf("failed to set frequency: %w", err)
	}

	// 3. Calibrate (strobe SCAL)
	if err := registers.Strobe(s.device, registers.StrobeSCAL); err != nil {
		return 0, fmt.Errorf("failed to calibrate: %w", err)
	}

	// 4. Brief wait for calibration
	time.Sleep(500 * time.Microsecond)

	// 5. Enter RX mode
	if err := s.device.StrobeModeRX(); err != nil {
		return 0, fmt.Errorf("failed to set RX: %w", err)
	}

	// 6. Wait for AGC to settle
	time.Sleep(dwellTime)

	// 7. Read RSSI
	rssiRaw, err := s.device.GetRSSI()
	if err != nil {
		return 0, fmt.Errorf("failed to read RSSI: %w", err)
	}

	// 8. Return to IDLE
	_ = s.device.StrobeModeIDLE()

	rssiDBm := RSSIToDBm(rssiRaw)
	s.debug("measureRSSI: %.3f MHz -> raw=0x%02X (%d) = %.1f dBm",
		float64(freqHz)/1e6, rssiRaw, rssiRaw, rssiDBm)

	return rssiDBm, nil
}

// setFrequency sets the radio frequency
func (s *scanner) setFrequency(freqHz uint32) error {
	// Calculate FREQ registers for 24 MHz crystal
	// FREQ = (freq_hz * 65536) / 24000000
	freq := uint32((uint64(freqHz) * 65536) / uint64(CrystalHz))

	freq2 := uint8((freq >> 16) & 0xFF)
	freq1 := uint8((freq >> 8) & 0xFF)
	freq0 := uint8(freq & 0xFF)

	// Write FREQ2, FREQ1, FREQ0 registers
	if err := s.device.PokeByte(registers.RegFREQ2, freq2); err != nil {
		return err
	}
	if err := s.device.PokeByte(registers.RegFREQ1, freq1); err != nil {
		return err
	}
	if err := s.device.PokeByte(registers.RegFREQ0, freq0); err != nil {
		return err
	}

	return nil
}

// loadPreset loads radio register values from a preset
func (s *scanner) loadPreset(preset *RegisterOverridesJSON) error {
	// Apply each non-nil register value
	if preset.MDMCFG4 != nil {
		if err := s.device.PokeByte(registers.RegMDMCFG4, *preset.MDMCFG4); err != nil {
			return err
		}
	}
	if preset.MDMCFG3 != nil {
		if err := s.device.PokeByte(registers.RegMDMCFG3, *preset.MDMCFG3); err != nil {
			return err
		}
	}
	if preset.MDMCFG2 != nil {
		if err := s.device.PokeByte(registers.RegMDMCFG2, *preset.MDMCFG2); err != nil {
			return err
		}
	}
	if preset.MDMCFG1 != nil {
		if err := s.device.PokeByte(registers.RegMDMCFG1, *preset.MDMCFG1); err != nil {
			return err
		}
	}
	if preset.MDMCFG0 != nil {
		if err := s.device.PokeByte(registers.RegMDMCFG0, *preset.MDMCFG0); err != nil {
			return err
		}
	}
	if preset.AGCCTRL2 != nil {
		if err := s.device.PokeByte(registers.RegAGCCTRL2, *preset.AGCCTRL2); err != nil {
			return err
		}
	}
	if preset.AGCCTRL1 != nil {
		if err := s.device.PokeByte(registers.RegAGCCTRL1, *preset.AGCCTRL1); err != nil {
			return err
		}
	}
	if preset.AGCCTRL0 != nil {
		if err := s.device.PokeByte(registers.RegAGCCTRL0, *preset.AGCCTRL0); err != nil {
			return err
		}
	}
	if preset.FREND1 != nil {
		if err := s.device.PokeByte(registers.RegFREND1, *preset.FREND1); err != nil {
			return err
		}
	}
	if preset.FREND0 != nil {
		if err := s.device.PokeByte(registers.RegFREND0, *preset.FREND0); err != nil {
			return err
		}
	}
	if preset.FOCCFG != nil {
		if err := s.device.PokeByte(registers.RegFOCCFG, *preset.FOCCFG); err != nil {
			return err
		}
	}
	if preset.BSCFG != nil {
		if err := s.device.PokeByte(registers.RegBSCFG, *preset.BSCFG); err != nil {
			return err
		}
	}

	return nil
}

// GetTracker returns the signal tracker (for advanced usage)
func (s *scanner) GetTracker() *SignalTracker {
	return s.tracker
}

// GetSmoother returns the frequency smoother (for advanced usage)
func (s *scanner) GetSmoother() *FrequencySmoother {
	return s.smoother
}
