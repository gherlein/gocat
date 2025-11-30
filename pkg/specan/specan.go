// Package specan provides firmware-based spectrum analysis for YardStick One
package specan

import (
	"fmt"
	"sync"
	"time"

	"github.com/herlein/gocat/pkg/yardstick"
)

// SpecAn represents a firmware-based spectrum analyzer
type SpecAn struct {
	device      *yardstick.Device
	baseFreq    uint32 // Base frequency in Hz
	chanSpacing uint32 // Channel spacing in Hz
	numChans    uint8  // Number of channels (max 255)

	mu       sync.Mutex
	running  bool
	stopChan chan struct{}
	dataChan chan *Frame
}

// Frame represents a single spectrum sweep result
type Frame struct {
	Timestamp   time.Time
	BaseFreq    uint32    // Hz
	ChanSpacing uint32    // Hz
	NumChans    int
	RSSI        []float32 // dBm values for each channel
}

// Config holds spectrum analyzer configuration
type Config struct {
	CenterFreq uint32 // Hz - center frequency
	Bandwidth  uint32 // Hz - total bandwidth to scan
	NumChans   uint8  // Number of channels (1-255)
}

// New creates a new spectrum analyzer
func New(device *yardstick.Device) *SpecAn {
	return &SpecAn{
		device:   device,
		dataChan: make(chan *Frame, 10),
		stopChan: make(chan struct{}),
	}
}

// Configure sets up the spectrum analyzer parameters
func (s *SpecAn) Configure(cfg *Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("cannot configure while running")
	}

	if cfg.NumChans == 0 {
		return fmt.Errorf("numChans must be 1-255, got %d", cfg.NumChans)
	}

	// Calculate base frequency and channel spacing
	halfBW := cfg.Bandwidth / 2
	s.baseFreq = cfg.CenterFreq - halfBW
	s.chanSpacing = cfg.Bandwidth / uint32(cfg.NumChans)
	s.numChans = cfg.NumChans

	// Set base frequency on device
	if err := s.device.SetFrequency(s.baseFreq); err != nil {
		return fmt.Errorf("failed to set frequency: %w", err)
	}

	// Set channel spacing
	if err := s.device.SetChannelSpacing(s.chanSpacing); err != nil {
		return fmt.Errorf("failed to set channel spacing: %w", err)
	}

	return nil
}

// Start begins the firmware spectrum analyzer
func (s *SpecAn) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("already running")
	}

	// Send START_SPECAN command with channel count
	cmd := []byte{s.numChans}
	_, err := s.device.Send(yardstick.AppNIC, yardstick.SPECANStart, cmd, yardstick.USBDefaultTimeout)
	if err != nil {
		return fmt.Errorf("failed to start specan: %w", err)
	}

	s.running = true
	s.stopChan = make(chan struct{})
	s.dataChan = make(chan *Frame, 10)

	// Start receive goroutine
	go s.receiveLoop()

	return nil
}

// Stop halts the spectrum analyzer
func (s *SpecAn) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	close(s.stopChan)
	s.mu.Unlock()

	// Send STOP_SPECAN command
	_, err := s.device.Send(yardstick.AppNIC, yardstick.SPECANStop, nil, yardstick.USBDefaultTimeout)
	if err != nil {
		return fmt.Errorf("failed to stop specan: %w", err)
	}

	return nil
}

// IsRunning returns true if the analyzer is running
func (s *SpecAn) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Frames returns a channel that receives spectrum frames
func (s *SpecAn) Frames() <-chan *Frame {
	return s.dataChan
}

// receiveLoop continuously receives RSSI data from firmware
func (s *SpecAn) receiveLoop() {
	defer close(s.dataChan)

	for {
		select {
		case <-s.stopChan:
			return
		default:
		}

		// Receive from APP_SPECAN, SPECAN_QUEUE
		data, err := s.device.RecvFromApp(yardstick.AppSPECAN, yardstick.SPECANQueue, 1*time.Second)
		if err != nil {
			// Timeout is normal, check if we should stop
			s.mu.Lock()
			running := s.running
			s.mu.Unlock()
			if !running {
				return
			}
			continue
		}

		if len(data) == 0 {
			continue
		}

		// Convert raw RSSI to dBm
		// rfcat formula: (raw ^ 0x80) / 2 - 88
		rssiDBm := make([]float32, len(data))
		for i, raw := range data {
			rssiDBm[i] = float32(int8(raw^0x80))/2.0 - 88.0
		}

		frame := &Frame{
			Timestamp:   time.Now(),
			BaseFreq:    s.baseFreq,
			ChanSpacing: s.chanSpacing,
			NumChans:    len(data),
			RSSI:        rssiDBm,
		}

		// Non-blocking send
		select {
		case s.dataChan <- frame:
		default:
			// Drop if channel full
		}
	}
}

// GetFrequencyForChannel returns the frequency for a given channel index
func (s *SpecAn) GetFrequencyForChannel(chanIdx int) uint32 {
	return s.baseFreq + uint32(chanIdx)*s.chanSpacing
}

// FrequencyForChannel is a helper to calculate frequency from frame parameters
func FrequencyForChannel(frame *Frame, chanIdx int) uint32 {
	return frame.BaseFreq + uint32(chanIdx)*frame.ChanSpacing
}
