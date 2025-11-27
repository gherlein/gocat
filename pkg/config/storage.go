package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func SaveToFile(configuration *DeviceConfig, path string) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(configuration, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func LoadFromFile(path string) (*DeviceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var configuration DeviceConfig
	if err := json.Unmarshal(data, &configuration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return &configuration, nil
}

func GetConfigPath(serial string) string {
	return filepath.Join("etc", "yardsticks", fmt.Sprintf("%s.json", serial))
}
