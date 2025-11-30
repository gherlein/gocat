package scanner

import "math"

// FrequencySmoother implements adaptive frequency smoothing to prevent display jitter
// while maintaining responsiveness to new signals.
type FrequencySmoother struct {
	value     float64 // Current smoothed value
	threshold float64 // Hz - above this difference, use fast adaptation
	kFast     float64 // Adaptation coefficient for large changes (0-1)
	kSlow     float64 // Adaptation coefficient for small changes (0-1)
}

// NewFrequencySmoother creates a new frequency smoother with default parameters
func NewFrequencySmoother() *FrequencySmoother {
	return &FrequencySmoother{
		value:     0,
		threshold: DefaultSmoothThreshold,
		kFast:     DefaultKFast,
		kSlow:     DefaultKSlow,
	}
}

// NewFrequencySmootherWithParams creates a smoother with custom parameters
func NewFrequencySmootherWithParams(threshold, kFast, kSlow float64) *FrequencySmoother {
	return &FrequencySmoother{
		value:     0,
		threshold: threshold,
		kFast:     kFast,
		kSlow:     kSlow,
	}
}

// Update applies adaptive smoothing to a new frequency value
// Returns the smoothed frequency value
func (s *FrequencySmoother) Update(newValue float64) float64 {
	// First value is returned as-is
	if s.value == 0 {
		s.value = newValue
		return newValue
	}

	// Calculate difference
	diff := math.Abs(newValue - s.value)

	// Choose adaptation coefficient based on change magnitude
	var k float64
	if diff > s.threshold {
		k = s.kFast // Fast adaptation for large changes (new signal)
	} else {
		k = s.kSlow // Slow adaptation for stability
	}

	// Apply exponential moving average
	s.value += (newValue - s.value) * k

	return s.value
}

// Value returns the current smoothed value
func (s *FrequencySmoother) Value() float64 {
	return s.value
}

// ValueHz returns the current smoothed value as uint32 Hz
func (s *FrequencySmoother) ValueHz() uint32 {
	return uint32(math.Round(s.value))
}

// Reset clears the smoother state
func (s *FrequencySmoother) Reset() {
	s.value = 0
}

// SetThreshold updates the threshold for fast/slow adaptation
func (s *FrequencySmoother) SetThreshold(threshold float64) {
	s.threshold = threshold
}

// SetCoefficients updates the adaptation coefficients
func (s *FrequencySmoother) SetCoefficients(kFast, kSlow float64) {
	s.kFast = kFast
	s.kSlow = kSlow
}
