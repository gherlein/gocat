package config

import (
	"fmt"
	"time"

	"github.com/herlein/gocat/pkg/registers"
	"github.com/herlein/gocat/pkg/yardstick"
)

// DeviceConfig holds all configuration data for a YardStick One device
type DeviceConfig struct {
	Serial       string                `json:"serial"`
	Manufacturer string                `json:"manufacturer"`
	Product      string                `json:"product"`
	BuildType    string                `json:"build_type,omitempty"`
	PartNum      uint8                 `json:"part_num,omitempty"`
	Timestamp    time.Time             `json:"timestamp"`
	Registers    registers.RegisterMap `json:"registers"`
}

// DumpFromDevice reads all configuration from a device
func DumpFromDevice(device *yardstick.Device) (*DeviceConfig, error) {
	// Get the current radio state
	originalState, err := registers.GetRadioState(device)
	if err != nil {
		return nil, fmt.Errorf("failed to get radio state: %w", err)
	}

	// Put radio in IDLE state for safe register access
	if originalState != registers.StateIDLE {
		if err := registers.SetIDLE(device); err != nil {
			return nil, fmt.Errorf("failed to set IDLE state: %w", err)
		}
		// Small delay to ensure state change
		time.Sleep(10 * time.Millisecond)
	}

	// Read all registers
	registerMap, err := registers.ReadAllRegisters(device)
	if err != nil {
		return nil, fmt.Errorf("failed to read registers: %w", err)
	}

	// Get build info
	buildType, _ := device.GetBuildType()
	partNum, _ := device.GetPartNum()

	// Restore original state
	if originalState != registers.StateIDLE {
		switch originalState {
		case registers.StateRX:
			registers.SetRX(device)
		case registers.StateTX:
			registers.SetTX(device)
		}
	}

	return &DeviceConfig{
		Serial:       device.Serial,
		Manufacturer: device.Manufacturer,
		Product:      device.Product,
		BuildType:    buildType,
		PartNum:      partNum,
		Timestamp:    time.Now(),
		Registers:    *registerMap,
	}, nil
}

// ApplyToDevice writes configuration to a device
func ApplyToDevice(device *yardstick.Device, configuration *DeviceConfig) error {
	// Get the current radio state
	originalState, err := registers.GetRadioState(device)
	if err != nil {
		return fmt.Errorf("failed to get radio state: %w", err)
	}

	// Put radio in IDLE state for safe register access
	if originalState != registers.StateIDLE {
		if err := registers.SetIDLE(device); err != nil {
			return fmt.Errorf("failed to set IDLE state: %w", err)
		}
		// Small delay to ensure state change
		time.Sleep(10 * time.Millisecond)
	}

	// Write all registers
	if err := registers.WriteAllRegisters(device, &configuration.Registers); err != nil {
		return fmt.Errorf("failed to write registers: %w", err)
	}

	// Restore original state
	if originalState != registers.StateIDLE {
		switch originalState {
		case registers.StateRX:
			registers.SetRX(device)
		case registers.StateTX:
			registers.SetTX(device)
		}
	}

	return nil
}

// GetCrystalFrequency returns the crystal frequency in MHz based on part number
func GetCrystalFrequency(partNum uint8) float64 {
	switch partNum {
	case yardstick.PartNumCC1110, yardstick.PartNumCC1111:
		return 24.0
	case yardstick.PartNumCC2510, yardstick.PartNumCC2511:
		return 26.0
	default:
		return 24.0 // Default to 24 MHz
	}
}

// GetFrequencyMHz returns the configured frequency in MHz
func (c *DeviceConfig) GetFrequencyMHz() float64 {
	crystalMHz := GetCrystalFrequency(c.PartNum)
	return registers.GetFrequency(&c.Registers, crystalMHz) / 1e6
}

// GetSyncWord returns the 16-bit sync word
func (c *DeviceConfig) GetSyncWord() uint16 {
	return registers.GetSyncWord(&c.Registers)
}

// GetModulationString returns a human-readable modulation format
func (c *DeviceConfig) GetModulationString() string {
	mod := registers.GetModulation(&c.Registers)
	switch mod {
	case registers.Mod2FSK:
		return "2-FSK"
	case registers.ModGFSK:
		return "GFSK"
	case registers.ModASKOOK:
		return "ASK/OOK"
	case registers.Mod4FSK:
		return "4-FSK"
	case registers.ModMSK:
		return "MSK"
	default:
		return fmt.Sprintf("Unknown (0x%02X)", mod)
	}
}

// GetRadioStateString returns a human-readable radio state
func (c *DeviceConfig) GetRadioStateString() string {
	return registers.RadioState(c.Registers.MARCSTATE & 0x1F).String()
}
