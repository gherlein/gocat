// profile-test runs loopback tests for radio configuration profiles
// It transmits from one YS1 and receives on another to verify configuration works
package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/gousb"
	"github.com/herlein/gocat/pkg/config"
	"github.com/herlein/gocat/pkg/profiles"
	"github.com/herlein/gocat/pkg/registers"
	"github.com/herlein/gocat/pkg/yardstick"
)

var (
	profileName  = flag.String("profile", "", "Profile name to test (e.g., 315-ook-low-1k2)")
	generateAll  = flag.Bool("generate", false, "Generate all profile configs for specified band")
	generateBand = flag.String("band", "315", "Band to generate: 315, 433, 868, 915, or all")
	listDevices  = flag.Bool("list", false, "List available YS1 devices")
	txDevice     = flag.String("tx", "", "TX device selector (index, bus:addr, or serial)")
	rxDevice     = flag.String("rx", "", "RX device selector (index, bus:addr, or serial)")
	configDir    = flag.String("config-dir", "tests/etc", "Directory for config files")
	verbose      = flag.Bool("v", false, "Verbose output")
	timeout      = flag.Duration("timeout", 5*time.Second, "Receive timeout")
	repeat       = flag.Int("repeat", 3, "Number of times to repeat each test")
	validateOnly = flag.Bool("validate", false, "Only validate config (single device, no RF test)")
)

func main() {
	flag.Parse()

	if *listDevices {
		doListDevices()
		return
	}

	if *generateAll {
		if err := doGenerateProfiles(); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating profiles: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *profileName == "" {
		fmt.Fprintln(os.Stderr, "Usage: profile-test -profile <name> [-tx <device>] [-rx <device>]")
		fmt.Fprintln(os.Stderr, "       profile-test -profile <name> -validate  (config validation only)")
		fmt.Fprintln(os.Stderr, "       profile-test -generate  (generate all 315 MHz configs)")
		fmt.Fprintln(os.Stderr, "       profile-test -list      (list available devices)")
		flag.PrintDefaults()
		os.Exit(1)
	}

	var err error
	if *validateOnly {
		err = doConfigValidation()
	} else {
		err = doProfileTest()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Test FAILED: %v\n", err)
		os.Exit(1)
	}
}

func doListDevices() {
	ctx := gousb.NewContext()
	defer ctx.Close()

	devices, err := yardstick.FindAllDevices(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding devices: %v\n", err)
		os.Exit(1)
	}

	if len(devices) == 0 {
		fmt.Println("No YardStick One devices found")
		return
	}

	fmt.Printf("Found %d YardStick One device(s):\n\n", len(devices))
	for i, dev := range devices {
		fmt.Printf("  #%d  %s  %d:%d\n", i, dev.Serial, dev.Bus, dev.Address)
		dev.Close()
	}
}

func doConfigValidation() error {
	// Load profile config
	configPath := filepath.Join(*configDir, *profileName+".json")
	fmt.Printf("Loading profile: %s\n", configPath)

	profileCfg, err := profiles.LoadProfileFromFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to load profile: %w", err)
	}

	fmt.Printf("Profile: %s\n", profileCfg.Profile.Name)
	fmt.Printf("  Frequency: %.3f MHz\n", profileCfg.Profile.FrequencyHz/1e6)
	fmt.Printf("  Data Rate: %.0f baud\n", profileCfg.Profile.DataRateBaud)
	fmt.Printf("  Modulation: 0x%02X\n", profileCfg.Profile.Modulation)

	// Open USB context
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Find a single device - use retry logic
	var dev *yardstick.Device
	for attempt := 0; attempt < 3; attempt++ {
		devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
			return desc.Vendor == gousb.ID(0x1d50) && desc.Product == gousb.ID(0x605b)
		})
		if err != nil {
			fmt.Printf("Attempt %d: enumeration error: %v\n", attempt+1, err)
			time.Sleep(time.Second)
		}
		if len(devs) > 0 {
			// Wrap the first device
			usbDev := devs[0]
			serial, _ := usbDev.SerialNumber()
			fmt.Printf("Found device: %s\n", serial)

			// Use the existing device opening function
			for _, d := range devs[1:] {
				d.Close() // Close extra devices
			}

			// Open the device properly using SelectDevice
			usbDev.Close()
			dev, err = yardstick.SelectDevice(ctx, yardstick.DeviceSelector(""))
			if err == nil {
				break
			}
			fmt.Printf("Failed to open device: %v\n", err)
		}
	}

	if dev == nil {
		return fmt.Errorf("could not find any YS1 device")
	}
	defer dev.Close()

	fmt.Printf("Using device: %s (%d:%d)\n", dev.Serial, dev.Bus, dev.Address)

	// Test connectivity
	fmt.Println("Testing connectivity...")
	if err := dev.Ping([]byte("TEST")); err != nil {
		return fmt.Errorf("device ping failed: %w", err)
	}
	fmt.Println("Ping OK")

	// Force IDLE state
	fmt.Println("Setting device to IDLE state...")
	if err := dev.PokeByte(0xDFE1, 0x04); err != nil {
		fmt.Printf("Warning: IDLE strobe failed: %v\n", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Apply configuration
	fmt.Println("Applying configuration...")
	devCfg := &config.DeviceConfig{
		Serial:    dev.Serial,
		Timestamp: time.Now(),
		Registers: profileCfg.Registers,
	}

	if err := config.ApplyToDevice(dev, devCfg); err != nil {
		return fmt.Errorf("failed to configure device: %w", err)
	}

	// Enable amplifiers
	fmt.Println("Enabling amplifiers...")
	if err := dev.SetAmpMode(1); err != nil {
		fmt.Printf("Warning: amplifier enable failed: %v\n", err)
	}

	// Verify configuration
	fmt.Println("Verifying configuration...")
	if err := verifyConfig(dev, &profileCfg.Registers); err != nil {
		return fmt.Errorf("config verification failed: %w", err)
	}

	// Test mode transitions
	fmt.Println("Testing mode transitions...")

	// Test IDLE -> RX
	fmt.Println("  Testing RX mode...")
	if err := dev.SetModeRX(); err != nil {
		return fmt.Errorf("failed to enter RX mode: %w", err)
	}
	state, err := dev.GetMARCSTATE()
	if err != nil {
		return fmt.Errorf("failed to read MARCSTATE: %w", err)
	}
	if state != 0x0D {
		return fmt.Errorf("not in RX mode: MARCSTATE=0x%02X", state)
	}
	fmt.Printf("    MARCSTATE=0x%02X (RX) OK\n", state)

	// Test RX -> IDLE
	fmt.Println("  Testing IDLE mode...")
	if err := dev.SetModeIDLE(); err != nil {
		return fmt.Errorf("failed to enter IDLE mode: %w", err)
	}
	time.Sleep(10 * time.Millisecond)
	state, err = dev.GetMARCSTATE()
	if err != nil {
		return fmt.Errorf("failed to read MARCSTATE: %w", err)
	}
	if state != 0x01 {
		return fmt.Errorf("not in IDLE mode: MARCSTATE=0x%02X", state)
	}
	fmt.Printf("    MARCSTATE=0x%02X (IDLE) OK\n", state)

	fmt.Println("\n=== Config Validation PASSED ===")
	fmt.Printf("Profile: %s\n", profileCfg.Profile.Name)
	return nil
}

func doGenerateProfiles() error {
	absPath, err := filepath.Abs(*configDir)
	if err != nil {
		return fmt.Errorf("invalid config directory: %w", err)
	}

	band := *generateBand
	totalCount := 0

	switch band {
	case "315":
		fmt.Printf("Generating 315 MHz profiles to %s\n", absPath)
		if err := profiles.Generate315Profiles(absPath); err != nil {
			return err
		}
		totalCount += 9
	case "433":
		fmt.Printf("Generating 433 MHz profiles to %s\n", absPath)
		if err := profiles.Generate433Profiles(absPath); err != nil {
			return err
		}
		totalCount += 21
	case "868":
		fmt.Printf("Generating 868 MHz profiles to %s\n", absPath)
		if err := profiles.Generate868Profiles(absPath); err != nil {
			return err
		}
		totalCount += 15
	case "915":
		fmt.Printf("Generating 915 MHz profiles to %s\n", absPath)
		if err := profiles.Generate915Profiles(absPath); err != nil {
			return err
		}
		totalCount += 18
	case "special":
		fmt.Printf("Generating special profiles to %s\n", absPath)
		if err := profiles.GenerateSpecialProfiles(absPath); err != nil {
			return err
		}
		totalCount += 26
	case "encoding":
		fmt.Printf("Generating encoding profiles to %s\n", absPath)
		if err := profiles.GenerateEncodingProfiles(absPath); err != nil {
			return err
		}
		totalCount += 14
	case "packet":
		fmt.Printf("Generating packet profiles to %s\n", absPath)
		if err := profiles.GeneratePacketProfiles(absPath); err != nil {
			return err
		}
		totalCount += 15
	case "all":
		fmt.Printf("Generating all profiles to %s\n", absPath)
		if err := profiles.Generate315Profiles(absPath); err != nil {
			return fmt.Errorf("315 MHz: %w", err)
		}
		totalCount += 9
		if err := profiles.Generate433Profiles(absPath); err != nil {
			return fmt.Errorf("433 MHz: %w", err)
		}
		totalCount += 21
		if err := profiles.Generate868Profiles(absPath); err != nil {
			return fmt.Errorf("868 MHz: %w", err)
		}
		totalCount += 15
		if err := profiles.Generate915Profiles(absPath); err != nil {
			return fmt.Errorf("915 MHz: %w", err)
		}
		totalCount += 18
		if err := profiles.GenerateSpecialProfiles(absPath); err != nil {
			return fmt.Errorf("special: %w", err)
		}
		totalCount += 26
		if err := profiles.GenerateEncodingProfiles(absPath); err != nil {
			return fmt.Errorf("encoding: %w", err)
		}
		totalCount += 14
		if err := profiles.GeneratePacketProfiles(absPath); err != nil {
			return fmt.Errorf("packet: %w", err)
		}
		totalCount += 15
	default:
		return fmt.Errorf("unknown band: %s (use 315, 433, 868, 915, special, encoding, packet, or all)", band)
	}

	// List generated files
	var pattern string
	if band == "all" {
		pattern = absPath + "/*.json"
	} else {
		pattern = absPath + "/" + band + "-*.json"
	}
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	fmt.Printf("Generated %d profile configs:\n", len(files))
	for _, f := range files {
		fmt.Printf("  %s\n", filepath.Base(f))
	}

	return nil
}

func doProfileTest() error {
	// Load profile config
	configPath := filepath.Join(*configDir, *profileName+".json")
	fmt.Printf("Loading profile: %s\n", configPath)

	profileCfg, err := profiles.LoadProfileFromFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to load profile: %w", err)
	}

	if *verbose {
		fmt.Printf("Profile: %s\n", profileCfg.Profile.Name)
		fmt.Printf("  Frequency: %.3f MHz\n", profileCfg.Profile.FrequencyHz/1e6)
		fmt.Printf("  Data Rate: %.0f baud\n", profileCfg.Profile.DataRateBaud)
		fmt.Printf("  Modulation: 0x%02X\n", profileCfg.Profile.Modulation)
	}

	// Open USB context
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Find devices
	devices, err := yardstick.FindAllDevices(ctx)
	if err != nil {
		return fmt.Errorf("failed to find devices: %w", err)
	}
	if len(devices) < 2 {
		for _, d := range devices {
			d.Close()
		}
		return fmt.Errorf("need at least 2 YS1 devices, found %d", len(devices))
	}

	// Select TX and RX devices
	txDev := devices[0]
	rxDev := devices[1]

	// If specific devices were requested, find them
	if *txDevice != "" || *rxDevice != "" {
		txDev, rxDev, err = selectDevices(devices, *txDevice, *rxDevice)
		if err != nil {
			for _, d := range devices {
				d.Close()
			}
			return err
		}
	}

	// Close any unused devices
	for _, d := range devices {
		if d != txDev && d != rxDev {
			d.Close()
		}
	}

	defer txDev.Close()
	defer rxDev.Close()

	fmt.Printf("TX Device: %s (%d:%d)\n", txDev.Serial, txDev.Bus, txDev.Address)
	fmt.Printf("RX Device: %s (%d:%d)\n", rxDev.Serial, rxDev.Bus, rxDev.Address)

	// Test connectivity
	fmt.Println("Testing device connectivity...")
	if err := txDev.Ping([]byte("TX")); err != nil {
		return fmt.Errorf("TX device ping failed: %w", err)
	}
	if err := rxDev.Ping([]byte("RX")); err != nil {
		return fmt.Errorf("RX device ping failed: %w", err)
	}

	// Force IDLE state on both devices first
	fmt.Println("Setting devices to IDLE state...")
	if err := txDev.PokeByte(0xDFE1, 0x04); err != nil {
		fmt.Printf("Warning: TX IDLE strobe failed: %v\n", err)
	}
	if err := rxDev.PokeByte(0xDFE1, 0x04); err != nil {
		fmt.Printf("Warning: RX IDLE strobe failed: %v\n", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Apply configuration to both devices
	fmt.Println("Applying configuration to devices...")

	devCfg := &config.DeviceConfig{
		Serial:    txDev.Serial,
		Timestamp: time.Now(),
		Registers: profileCfg.Registers,
	}

	if err := config.ApplyToDevice(txDev, devCfg); err != nil {
		return fmt.Errorf("failed to configure TX device: %w", err)
	}

	devCfg.Serial = rxDev.Serial
	if err := config.ApplyToDevice(rxDev, devCfg); err != nil {
		return fmt.Errorf("failed to configure RX device: %w", err)
	}

	// Enable amplifiers for better TX power and RX sensitivity
	fmt.Println("Enabling amplifiers...")
	if err := txDev.SetAmpMode(1); err != nil {
		fmt.Printf("Warning: TX amplifier enable failed: %v\n", err)
	}
	if err := rxDev.SetAmpMode(1); err != nil {
		fmt.Printf("Warning: RX amplifier enable failed: %v\n", err)
	}

	// Verify configuration was applied
	if *verbose {
		fmt.Println("Verifying TX device configuration...")
		if err := verifyConfig(txDev, &profileCfg.Registers); err != nil {
			return fmt.Errorf("TX config verification failed: %w", err)
		}
		fmt.Println("Verifying RX device configuration...")
		if err := verifyConfig(rxDev, &profileCfg.Registers); err != nil {
			return fmt.Errorf("RX config verification failed: %w", err)
		}
	}

	// Run loopback test
	fmt.Println("\nRunning loopback test...")
	return runLoopbackTest(txDev, rxDev, &profileCfg.Profile)
}

func selectDevices(devices []*yardstick.Device, txSel, rxSel string) (*yardstick.Device, *yardstick.Device, error) {
	var txDev, rxDev *yardstick.Device

	for _, d := range devices {
		selector := fmt.Sprintf("%d:%d", d.Bus, d.Address)
		if txSel != "" && (d.Serial == txSel || selector == txSel) {
			txDev = d
		}
		if rxSel != "" && (d.Serial == rxSel || selector == rxSel) {
			rxDev = d
		}
	}

	// If TX not specified but RX is, use other device for TX
	if txDev == nil && rxDev != nil {
		for _, d := range devices {
			if d != rxDev {
				txDev = d
				break
			}
		}
	}

	// If RX not specified but TX is, use other device for RX
	if rxDev == nil && txDev != nil {
		for _, d := range devices {
			if d != txDev {
				rxDev = d
				break
			}
		}
	}

	// If neither specified, use first two
	if txDev == nil && rxDev == nil {
		if len(devices) >= 2 {
			txDev = devices[0]
			rxDev = devices[1]
		}
	}

	if txDev == nil {
		return nil, nil, fmt.Errorf("TX device not found: %s", txSel)
	}
	if rxDev == nil {
		return nil, nil, fmt.Errorf("RX device not found: %s", rxSel)
	}
	if txDev == rxDev {
		return nil, nil, fmt.Errorf("TX and RX must be different devices")
	}

	return txDev, rxDev, nil
}

func verifyConfig(dev *yardstick.Device, expected *registers.RegisterMap) error {
	actual, err := registers.ReadAllRegisters(dev)
	if err != nil {
		return fmt.Errorf("failed to read registers: %w", err)
	}

	// Compare key registers
	checks := []struct {
		name     string
		expected uint8
		actual   uint8
	}{
		{"FREQ2", expected.FREQ2, actual.FREQ2},
		{"FREQ1", expected.FREQ1, actual.FREQ1},
		{"FREQ0", expected.FREQ0, actual.FREQ0},
		{"MDMCFG4", expected.MDMCFG4, actual.MDMCFG4},
		{"MDMCFG3", expected.MDMCFG3, actual.MDMCFG3},
		{"MDMCFG2", expected.MDMCFG2, actual.MDMCFG2},
		{"MDMCFG1", expected.MDMCFG1, actual.MDMCFG1},
		{"PKTCTRL0", expected.PKTCTRL0, actual.PKTCTRL0},
		{"PKTLEN", expected.PKTLEN, actual.PKTLEN},
	}

	for _, c := range checks {
		if c.expected != c.actual {
			return fmt.Errorf("%s mismatch: expected 0x%02X, got 0x%02X", c.name, c.expected, c.actual)
		}
		if *verbose {
			fmt.Printf("  %s: 0x%02X OK\n", c.name, c.actual)
		}
	}

	return nil
}

func runLoopbackTest(txDev, rxDev *yardstick.Device, profile *profiles.Profile) error {
	// Create test payload based on packet configuration
	payloadLen := int(profile.PktLen)
	if profile.PktLenMode == profiles.PktLenVariable {
		// For variable length, use a smaller test payload
		payloadLen = 16
	}
	if payloadLen > 64 {
		payloadLen = 64 // Keep tests reasonable
	}

	// For OOK/ASK without sync, we need to add our own structure
	var testPayload []byte
	if profile.SyncMode == profiles.SyncNone {
		// Create a recognizable pattern with preamble and pattern
		testPayload = make([]byte, payloadLen)
		// Start with alternating bits (preamble)
		for i := 0; i < 8 && i < payloadLen; i++ {
			testPayload[i] = 0xAA
		}
		// Add recognizable pattern
		pattern := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE}
		for i := 0; i < len(pattern) && i+8 < payloadLen; i++ {
			testPayload[8+i] = pattern[i]
		}
		// Fill rest with counter
		for i := 14; i < payloadLen; i++ {
			testPayload[i] = uint8(i)
		}
	} else {
		// For sync mode, just use recognizable data
		testPayload = make([]byte, payloadLen)
		for i := range testPayload {
			testPayload[i] = uint8((i + 0x42) & 0xFF)
		}
	}

	fmt.Printf("Test payload (%d bytes): %s\n", len(testPayload), hex.EncodeToString(testPayload[:min(16, len(testPayload))]))

	// Put RX device in receive mode
	fmt.Println("Setting RX device to receive mode...")
	if err := rxDev.SetModeRX(); err != nil {
		return fmt.Errorf("failed to set RX mode: %w", err)
	}

	// Run multiple test iterations
	successCount := 0
	for i := 0; i < *repeat; i++ {
		fmt.Printf("\nTest iteration %d/%d\n", i+1, *repeat)

		// Small delay before transmit
		time.Sleep(100 * time.Millisecond)

		// Transmit
		fmt.Printf("  Transmitting %d bytes...\n", len(testPayload))
		if err := txDev.RFXmit(testPayload, 0, 0); err != nil {
			fmt.Printf("  TX Error: %v\n", err)
			continue
		}

		// Receive
		fmt.Printf("  Waiting for RX (timeout: %v)...\n", *timeout)
		rxData, err := rxDev.RFRecv(*timeout, 0)
		if err != nil {
			fmt.Printf("  RX Error: %v\n", err)
			// Re-enter RX mode for next iteration
			rxDev.SetModeRX()
			continue
		}

		// Check received data
		fmt.Printf("  Received %d bytes: %s\n", len(rxData), hex.EncodeToString(rxData[:min(16, len(rxData))]))

		// Get RSSI/LQI
		status, err := rxDev.GetRadioStatus()
		if err == nil {
			fmt.Printf("  RSSI: %d dBm, LQI: %d, CRC OK: %v\n", status.RSSIdBm, status.LQI, status.CRCOk)
		}

		// Compare payloads
		if comparePayloads(testPayload, rxData, profile) {
			fmt.Println("  PASS: Payload matched!")
			successCount++
		} else {
			fmt.Println("  FAIL: Payload mismatch")
		}

		// Re-enter RX mode for next iteration
		rxDev.SetModeRX()
	}

	// Summary
	fmt.Printf("\n=== Test Summary ===\n")
	fmt.Printf("Profile: %s\n", profile.Name)
	fmt.Printf("Passed: %d/%d iterations\n", successCount, *repeat)

	if successCount == 0 {
		return fmt.Errorf("all test iterations failed")
	}
	if successCount < *repeat {
		return fmt.Errorf("%d/%d iterations failed", *repeat-successCount, *repeat)
	}

	fmt.Println("All tests PASSED!")
	return nil
}

func comparePayloads(sent, received []byte, profile *profiles.Profile) bool {
	// For sync mode with CRC, expect exact match (possibly with status bytes appended)
	if profile.SyncMode != profiles.SyncNone && profile.CRCEn {
		// Received data may have status bytes appended
		if len(received) >= len(sent) {
			return bytes.Equal(sent, received[:len(sent)])
		}
		return false
	}

	// For OOK without sync, look for our pattern within received data
	if profile.SyncMode == profiles.SyncNone {
		// Look for our marker pattern
		pattern := []byte{0xDE, 0xAD, 0xBE, 0xEF}
		return bytes.Contains(received, pattern)
	}

	// Default: exact match
	return bytes.Equal(sent, received)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
