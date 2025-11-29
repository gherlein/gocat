package profiles

import "fmt"

// Packet Format Profile Factories
// These test different packet configurations: length modes, CRC, addressing, etc.

// NewFixedLengthVariant creates profiles with fixed packet length
// pktLen: packet length in bytes (1-255)
func NewFixedLengthVariant(pktLen uint8) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("pkt-fixed-%d", pktLen),
		Description:   fmt.Sprintf("Fixed packet length: %d bytes", pktLen),
		FrequencyHz:   433920000,
		Modulation:    ModGFSK,
		DataRateBaud:  38400,
		DeviationHz:   10000,
		ChannelBWHz:   100000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenFixed,
		PktLen:        pktLen,
		PreambleBytes: 4,
		CRCEn:         true,
	}
}

// NewVariableLengthVariant creates profiles with variable packet length
// maxLen: maximum packet length in bytes
func NewVariableLengthVariant(maxLen uint8) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("pkt-var-max%d", maxLen),
		Description:   fmt.Sprintf("Variable packet length: max %d bytes", maxLen),
		FrequencyHz:   433920000,
		Modulation:    ModGFSK,
		DataRateBaud:  38400,
		DeviationHz:   10000,
		ChannelBWHz:   100000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        maxLen,
		PreambleBytes: 4,
		CRCEn:         true,
	}
}

// NewInfiniteLengthVariant creates a profile with infinite packet length mode
// Useful for streaming data
func NewInfiniteLengthVariant() *Profile {
	return &Profile{
		Name:          "pkt-infinite",
		Description:   "Infinite packet length mode for streaming",
		FrequencyHz:   433920000,
		Modulation:    ModGFSK,
		DataRateBaud:  38400,
		DeviationHz:   10000,
		ChannelBWHz:   100000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenInfinite,
		PktLen:        0, // Not used in infinite mode
		PreambleBytes: 4,
		CRCEn:         false, // CRC usually disabled for infinite
	}
}

// NewCRCVariant creates profiles with different CRC settings
// crcEnabled: whether CRC is enabled
func NewCRCVariant(crcEnabled bool) *Profile {
	name := "pkt-crc-off"
	desc := "CRC disabled"
	if crcEnabled {
		name = "pkt-crc-on"
		desc = "CRC enabled"
	}

	return &Profile{
		Name:          name,
		Description:   desc,
		FrequencyHz:   433920000,
		Modulation:    ModGFSK,
		DataRateBaud:  38400,
		DeviationHz:   10000,
		ChannelBWHz:   100000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        60,
		PreambleBytes: 4,
		CRCEn:         crcEnabled,
	}
}

// NewSyncWordVariant creates profiles with different sync words
// syncWord: the 16-bit sync word
// name: descriptive name for the sync word
func NewSyncWordVariant(syncWord uint16, name string) *Profile {
	return &Profile{
		Name:          fmt.Sprintf("pkt-sync-%s", name),
		Description:   fmt.Sprintf("Sync word test: 0x%04X (%s)", syncWord, name),
		FrequencyHz:   433920000,
		Modulation:    ModGFSK,
		DataRateBaud:  38400,
		DeviationHz:   10000,
		ChannelBWHz:   100000,
		SyncWord:      syncWord,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        60,
		PreambleBytes: 4,
		CRCEn:         true,
	}
}

// NewMaxPacketSize creates a profile for maximum packet size testing
func NewMaxPacketSize() *Profile {
	return &Profile{
		Name:          "pkt-max-size",
		Description:   "Maximum packet size: 255 bytes",
		FrequencyHz:   433920000,
		Modulation:    ModGFSK,
		DataRateBaud:  100000, // Higher rate for large packets
		DeviationHz:   25000,
		ChannelBWHz:   200000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenVariable,
		PktLen:        255,
		PreambleBytes: 4,
		CRCEn:         true,
	}
}

// NewMinPacketSize creates a profile for minimum packet size testing
func NewMinPacketSize() *Profile {
	return &Profile{
		Name:          "pkt-min-size",
		Description:   "Minimum packet size: 1 byte",
		FrequencyHz:   433920000,
		Modulation:    ModGFSK,
		DataRateBaud:  9600,
		DeviationHz:   5000,
		ChannelBWHz:   58000,
		SyncWord:      0xD391,
		SyncMode:      Sync16of16,
		PktLenMode:    PktLenFixed,
		PktLen:        1,
		PreambleBytes: 4,
		CRCEn:         true,
	}
}

// GeneratePacketProfiles generates all packet format profiles
func GeneratePacketProfiles(basePath string) error {
	profiles := []*Profile{
		// Fixed length variants
		NewFixedLengthVariant(8),
		NewFixedLengthVariant(32),
		NewFixedLengthVariant(64),
		NewFixedLengthVariant(128),

		// Variable length variants
		NewVariableLengthVariant(60),
		NewVariableLengthVariant(128),
		NewVariableLengthVariant(255),

		// Infinite length
		NewInfiniteLengthVariant(),

		// CRC variants
		NewCRCVariant(false),
		NewCRCVariant(true),

		// Sync word variants
		NewSyncWordVariant(0xAAAA, "alternating"),
		NewSyncWordVariant(0xD391, "standard"),
		NewSyncWordVariant(0x7E7E, "hdlc-like"),

		// Size extremes
		NewMaxPacketSize(),
		NewMinPacketSize(),
	}

	if err := EnsureDir(basePath + "/dummy"); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	for _, p := range profiles {
		filename := fmt.Sprintf("%s/%s.json", basePath, p.Name)
		if err := p.SaveToFile(filename); err != nil {
			return fmt.Errorf("failed to save profile %s: %w", p.Name, err)
		}
	}

	return nil
}
