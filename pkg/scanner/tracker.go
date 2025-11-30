package scanner

import (
	"sync"
	"time"
)

// SignalTracker manages detected signals with hysteresis
type SignalTracker struct {
	signals     map[uint32]*SignalInfo // Key: rounded frequency
	mu          sync.RWMutex
	holdCounter int    // Counts down when signal lost
	holdMax     int    // Maximum hold count
	lostAt      int    // Counter value when "lost" callback fires
	resolution  uint32 // Frequency resolution for grouping (Hz)

	// Current active signal
	activeFrequency uint32
	activeSignal    *SignalInfo

	// Callbacks
	onDetected func(*SignalInfo)
	onLost     func(*SignalInfo)
}

// NewSignalTracker creates a new signal tracker with the given parameters
func NewSignalTracker(holdMax, lostAt int, resolution uint32) *SignalTracker {
	return &SignalTracker{
		signals:    make(map[uint32]*SignalInfo),
		holdMax:    holdMax,
		lostAt:     lostAt,
		resolution: resolution,
	}
}

// SetCallbacks sets the signal detection callbacks
func (t *SignalTracker) SetCallbacks(onDetected, onLost func(*SignalInfo)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onDetected = onDetected
	t.onLost = onLost
}

// Update processes a scan result and updates signal tracking state
func (t *SignalTracker) Update(result *ScanResult) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if result.SignalDetected {
		// Reset hold counter
		t.holdCounter = t.holdMax

		// Round frequency for lookup
		key := t.roundFrequency(result.FineFrequency)

		info, exists := t.signals[key]
		if !exists {
			// New signal detected
			info = &SignalInfo{
				Frequency:      result.FineFrequency,
				RawFrequency:   result.FineFrequency,
				RSSI:           result.FineRSSI,
				MaxRSSI:        result.FineRSSI,
				FirstSeen:      result.Timestamp,
				LastSeen:       result.Timestamp,
				DetectionCount: 1,
			}
			t.signals[key] = info

			// Check if this is a new active signal
			if t.activeSignal == nil || key != t.activeFrequency {
				t.activeFrequency = key
				t.activeSignal = info
				if t.onDetected != nil {
					// Copy to avoid race conditions
					infoCopy := *info
					go t.onDetected(&infoCopy)
				}
			}
		} else {
			// Update existing signal
			info.RawFrequency = result.FineFrequency
			info.RSSI = result.FineRSSI
			info.LastSeen = result.Timestamp
			info.DetectionCount++
			if result.FineRSSI > info.MaxRSSI {
				info.MaxRSSI = result.FineRSSI
			}
		}

		// Update active signal reference
		t.activeSignal = info
		t.activeFrequency = key
	} else {
		// No signal detected - decrement hold counter
		if t.holdCounter > 0 {
			t.holdCounter--

			if t.holdCounter == t.lostAt && t.activeSignal != nil {
				// Signal considered lost - trigger callback
				if t.onLost != nil {
					infoCopy := *t.activeSignal
					go t.onLost(&infoCopy)
				}
			}

			if t.holdCounter == 0 {
				// Signal completely gone
				t.activeSignal = nil
				t.activeFrequency = 0
			}
		}
	}
}

// roundFrequency rounds a frequency to the configured resolution
func (t *SignalTracker) roundFrequency(freq uint32) uint32 {
	if t.resolution == 0 {
		return freq
	}
	return (freq / t.resolution) * t.resolution
}

// GetActiveSignal returns the currently active signal, if any
func (t *SignalTracker) GetActiveSignal() *SignalInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.activeSignal == nil {
		return nil
	}

	// Return a copy
	info := *t.activeSignal
	return &info
}

// GetAllSignals returns all tracked signals
func (t *SignalTracker) GetAllSignals() []*SignalInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	signals := make([]*SignalInfo, 0, len(t.signals))
	for _, info := range t.signals {
		infoCopy := *info
		signals = append(signals, &infoCopy)
	}
	return signals
}

// GetSignalCount returns the number of tracked signals
func (t *SignalTracker) GetSignalCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.signals)
}

// Clear removes all tracked signals
func (t *SignalTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.signals = make(map[uint32]*SignalInfo)
	t.activeSignal = nil
	t.activeFrequency = 0
	t.holdCounter = 0
}

// PruneOld removes signals not seen since the given time
func (t *SignalTracker) PruneOld(since time.Time) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	count := 0
	for key, info := range t.signals {
		if info.LastSeen.Before(since) {
			delete(t.signals, key)
			count++
		}
	}
	return count
}

// IsActive returns true if a signal is currently being tracked
func (t *SignalTracker) IsActive() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.activeSignal != nil && t.holdCounter > 0
}

// HoldCounter returns the current hold counter value
func (t *SignalTracker) HoldCounter() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.holdCounter
}
