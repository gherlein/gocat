package scanner

import "time"

// ScanResult holds the result of a single scan cycle
type ScanResult struct {
	// Coarse scan results
	CoarseFrequency uint32  // Hz - frequency with strongest signal in coarse scan
	CoarseRSSI      float32 // dBm - signal strength at coarse frequency

	// Fine scan results (only populated if signal detected)
	FineFrequency uint32  // Hz - refined frequency from fine scan
	FineRSSI      float32 // dBm - signal strength at fine frequency

	// Metadata
	Timestamp      time.Time
	SignalDetected bool // True if RSSI exceeded threshold
}

// SignalInfo represents a detected signal with history
type SignalInfo struct {
	Frequency      uint32    // Hz - smoothed frequency
	RawFrequency   uint32    // Hz - last measured frequency
	RSSI           float32   // dBm - current signal strength
	MaxRSSI        float32   // dBm - maximum observed RSSI
	FirstSeen      time.Time // When signal was first detected
	LastSeen       time.Time // When signal was last detected
	DetectionCount uint32    // Number of times detected
}

// RSSIToDBm converts raw CC1111 RSSI register value to dBm
// The CC1111 uses a signed value in 0.5 dBm steps with -74 dBm offset
//
// Special cases:
// - 0x80 (128): Often indicates RSSI not yet valid or AGC in transition
// - Returns RSSIInvalidValue constant for special/invalid readings
func RSSIToDBm(rssi uint8) float32 {
	// 0x80 is often a sentinel/invalid value from CC1111
	// This can happen when:
	// - Radio isn't fully in RX mode
	// - RSSI hasn't been updated yet
	// - AGC is in overflow/transition state
	if rssi == 0x80 {
		return RSSIInvalidValue
	}
	if rssi >= 128 {
		return float32(int(rssi)-256)/2.0 - 74.0
	}
	return float32(rssi)/2.0 - 74.0
}

// RSSIInvalidValue represents an invalid or not-yet-available RSSI reading
const RSSIInvalidValue float32 = -200.0

// IsValidRSSI returns true if the RSSI value is a valid measurement
func IsValidRSSI(rssi float32) bool {
	return rssi > RSSIInvalidValue
}

// IsValidFrequency checks if a frequency is within CC1111 supported bands
func IsValidFrequency(freq uint32) bool {
	// 300-348 MHz band
	if freq >= 300000000 && freq <= 348000000 {
		return true
	}
	// 387-464 MHz band
	if freq >= 387000000 && freq <= 464000000 {
		return true
	}
	// 779-928 MHz band
	if freq >= 779000000 && freq <= 928000000 {
		return true
	}
	return false
}

// FrequencyBand returns the band name for a given frequency
func FrequencyBand(freq uint32) string {
	if freq >= 300000000 && freq <= 348000000 {
		return "300MHz"
	}
	if freq >= 387000000 && freq <= 464000000 {
		return "400MHz"
	}
	if freq >= 779000000 && freq <= 928000000 {
		return "800MHz"
	}
	return "Unknown"
}
