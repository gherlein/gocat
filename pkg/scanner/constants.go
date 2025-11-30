// Package scanner provides frequency scanning capabilities for the YardStick One.
package scanner

import "time"

// Hardware constants
const (
	// CrystalHz is the CC1111 crystal frequency (YardStick One)
	CrystalHz uint32 = 24000000
)

// Default scanning parameters
const (
	// DefaultRSSIThreshold is the minimum RSSI for signal detection (dBm)
	DefaultRSSIThreshold float32 = -93.0

	// DefaultFineScanRange is the range around detected signal (±Hz)
	DefaultFineScanRange uint32 = 300000

	// DefaultFineScanStep is the step size for fine scan (Hz)
	DefaultFineScanStep uint32 = 20000

	// DefaultDwellTime is the time to wait for RSSI measurement
	DefaultDwellTime = 2 * time.Millisecond

	// DefaultScanInterval is the delay between scan cycles
	DefaultScanInterval = 10 * time.Millisecond
)

// Signal tracking defaults
const (
	// DefaultHoldMax is the maximum hold counter value
	DefaultHoldMax = 20

	// DefaultLostThreshold is when signal is considered lost
	DefaultLostThreshold = 15

	// DefaultFrequencyResolution is the grouping resolution for signals (Hz)
	DefaultFrequencyResolution uint32 = 10000
)

// Frequency smoothing defaults
const (
	// DefaultSmoothThreshold is the threshold for fast/slow adaptation (Hz)
	DefaultSmoothThreshold float64 = 500000

	// DefaultKFast is the adaptation coefficient for large changes
	DefaultKFast float64 = 0.9

	// DefaultKSlow is the adaptation coefficient for small changes
	DefaultKSlow float64 = 0.03
)

// Register values for scanning presets
const (
	// Coarse scan preset - wide bandwidth (~600 kHz for CC1111)
	// MDMCFG4: CHANBW_E=0, CHANBW_M=1, DRATE_E=15
	CoarseMDMCFG4  uint8 = 0x1F
	CoarseMDMCFG3  uint8 = 0x7F
	CoarseMDMCFG2  uint8 = 0x30 // ASK/OOK, no sync
	CoarseAGCCTRL2 uint8 = 0x07
	CoarseAGCCTRL1 uint8 = 0x00
	CoarseAGCCTRL0 uint8 = 0x91
	CoarseFREND1   uint8 = 0xB6
	CoarseFREND0   uint8 = 0x10

	// Fine scan preset - narrow bandwidth (~58 kHz for CC1111)
	// MDMCFG4: CHANBW_E=3, CHANBW_M=3, DRATE_E=7
	FineMDMCFG4  uint8 = 0xF7
	FineMDMCFG3  uint8 = 0x7F
	FineMDMCFG2  uint8 = 0x30 // ASK/OOK, no sync
	FineAGCCTRL2 uint8 = 0x07
	FineAGCCTRL1 uint8 = 0x00
	FineAGCCTRL0 uint8 = 0x91
	FineFREND1   uint8 = 0x56
	FineFREND0   uint8 = 0x10
)

// MARCSTATE values (from registers package, duplicated for convenience)
const (
	MarcStateIdle = 0x01
	MarcStateRX   = 0x0D
)

// DefaultFrequencies is the standard set of frequencies for scanning
var DefaultFrequencies = []uint32{
	// 300-348 MHz band
	300000000,
	303875000, // Garage doors
	304250000,
	310000000, // US keyless entry
	315000000, // US keyless entry
	318000000,

	// 387-464 MHz band
	390000000,
	418000000,
	433075000, // LPD433 first channel
	433420000,
	433920000, // LPD433 center (most common)
	434420000,
	434775000, // LPD433 last channel
	438900000,

	// 779-928 MHz band
	868350000, // EU SRD
	915000000, // US ISM
	925000000,
}

// HopperFrequencies is a minimal set for rapid scanning
var HopperFrequencies = []uint32{
	310000000, // 300 MHz band
	315000000,
	390000000, // 400 MHz band
	433920000,
	868350000, // 800 MHz band
	915000000,
}
