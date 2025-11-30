package specan

// FindPeaks finds channels with RSSI above threshold
func FindPeaks(frame *Frame, thresholdDBm float32) []Peak {
	var peaks []Peak
	for i, rssi := range frame.RSSI {
		if rssi >= thresholdDBm {
			peaks = append(peaks, Peak{
				ChannelIndex: i,
				FrequencyHz:  FrequencyForChannel(frame, i),
				RSSI:         rssi,
			})
		}
	}
	return peaks
}

// Peak represents a detected signal peak
type Peak struct {
	ChannelIndex int
	FrequencyHz  uint32
	RSSI         float32
}

// MaxRSSI returns the channel with maximum RSSI
func MaxRSSI(frame *Frame) (channelIndex int, frequencyHz uint32, rssi float32) {
	if len(frame.RSSI) == 0 {
		return -1, 0, -200.0
	}

	maxIdx := 0
	maxVal := frame.RSSI[0]

	for i, v := range frame.RSSI {
		if v > maxVal {
			maxVal = v
			maxIdx = i
		}
	}

	return maxIdx, FrequencyForChannel(frame, maxIdx), maxVal
}

// AverageRSSI calculates the average RSSI across all channels
func AverageRSSI(frame *Frame) float32 {
	if len(frame.RSSI) == 0 {
		return -200.0
	}

	var sum float32
	for _, v := range frame.RSSI {
		sum += v
	}
	return sum / float32(len(frame.RSSI))
}

// MinRSSI returns the channel with minimum RSSI (noise floor)
func MinRSSI(frame *Frame) (channelIndex int, frequencyHz uint32, rssi float32) {
	if len(frame.RSSI) == 0 {
		return -1, 0, -200.0
	}

	minIdx := 0
	minVal := frame.RSSI[0]

	for i, v := range frame.RSSI {
		if v < minVal {
			minVal = v
			minIdx = i
		}
	}

	return minIdx, FrequencyForChannel(frame, minIdx), minVal
}

// SignalToNoise returns the difference between max and min RSSI
func SignalToNoise(frame *Frame) float32 {
	_, _, maxRSSI := MaxRSSI(frame)
	_, _, minRSSI := MinRSSI(frame)
	return maxRSSI - minRSSI
}
