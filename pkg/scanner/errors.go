package scanner

import "errors"

// Scanner errors
var (
	// ErrScannerRunning indicates the scanner is already running
	ErrScannerRunning = errors.New("scanner is already running")

	// ErrScannerNotRunning indicates the scanner is not running
	ErrScannerNotRunning = errors.New("scanner is not running")

	// ErrDeviceNotReady indicates the device is not ready for scanning
	ErrDeviceNotReady = errors.New("device is not ready")

	// ErrInvalidConfig indicates invalid scanner configuration
	ErrInvalidConfig = errors.New("invalid scanner configuration")

	// ErrFrequencyOutOfRange indicates a frequency is outside valid bands
	ErrFrequencyOutOfRange = errors.New("frequency out of valid range")

	// ErrNoFrequencies indicates no frequencies were specified for scanning
	ErrNoFrequencies = errors.New("no frequencies specified for scanning")

	// ErrInvalidThreshold indicates an invalid RSSI threshold
	ErrInvalidThreshold = errors.New("RSSI threshold must be negative (dBm)")

	// ErrInvalidDwellTime indicates an invalid dwell time
	ErrInvalidDwellTime = errors.New("dwell time must be between 1-100 ms")

	// ErrConfigVersion indicates unsupported config file version
	ErrConfigVersion = errors.New("unsupported configuration version")
)
