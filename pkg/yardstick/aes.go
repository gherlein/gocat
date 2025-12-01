package yardstick

import "fmt"

// AESConfig holds AES encryption configuration
type AESConfig struct {
	Mode      uint8    // AES mode (ECB, CBC, etc.)
	Key       [16]byte // 128-bit encryption key
	IV        [16]byte // 128-bit initialization vector
	EncryptTX bool     // Encrypt outgoing packets
	DecryptRX bool     // Decrypt incoming packets
}

// SetAESMode configures the AES crypto mode
func (d *Device) SetAESMode(mode uint8) error {
	_, err := d.Send(AppNIC, NICSetAESMode, []byte{mode}, USBDefaultTimeout)
	return err
}

// GetAESMode returns the current AES mode
func (d *Device) GetAESMode() (uint8, error) {
	resp, err := d.Send(AppNIC, NICGetAESMode, nil, USBDefaultTimeout)
	if err != nil {
		return 0, err
	}
	if len(resp) < 1 {
		return 0, fmt.Errorf("empty response")
	}
	return resp[0], nil
}

// SetAESKey sets the 128-bit AES encryption key
func (d *Device) SetAESKey(key [16]byte) error {
	_, err := d.Send(AppNIC, NICSetAESKey, key[:], USBDefaultTimeout)
	return err
}

// SetAESIV sets the 128-bit initialization vector
func (d *Device) SetAESIV(iv [16]byte) error {
	_, err := d.Send(AppNIC, NICSetAESIV, iv[:], USBDefaultTimeout)
	return err
}

// ConfigureAES is a convenience function to set up AES in one call
func (d *Device) ConfigureAES(cfg *AESConfig) error {
	// Set key first
	if err := d.SetAESKey(cfg.Key); err != nil {
		return fmt.Errorf("set key: %w", err)
	}

	// Set IV (used for CBC and other modes)
	if err := d.SetAESIV(cfg.IV); err != nil {
		return fmt.Errorf("set IV: %w", err)
	}

	// Build mode byte
	mode := cfg.Mode
	if cfg.EncryptTX {
		mode |= AESCryptoOutEnable | AESCryptoOutEncrypt
	}
	if cfg.DecryptRX {
		mode |= AESCryptoInEnable
	}

	return d.SetAESMode(mode)
}

// DisableAES turns off all AES encryption
func (d *Device) DisableAES() error {
	return d.SetAESMode(AESCryptoNone)
}
