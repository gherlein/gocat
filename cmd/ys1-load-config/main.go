// ys1-load-config: Load configuration to YardStick One from JSON file
//
// This tool reads a previously saved configuration file and applies it
// to a YardStick One device.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/google/gousb"
	"github.com/herlein/gocat/pkg/config"
	"github.com/herlein/gocat/pkg/yardstick"
)

func main() {
	// Parse command line flags
	serial := flag.String("s", "", "Device serial number (optional, uses first device if not specified)")
	verbose := flag.Bool("v", false, "Verbose output")
	verify := flag.Bool("verify", false, "Verify configuration after writing")
	flag.Parse()

	// Get config file path from arguments
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <config-file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s etc/yardsticks/ABC123.json\n", os.Args[0])
		os.Exit(1)
	}

	configPath := args[0]

	// Load configuration from file
	if *verbose {
		fmt.Printf("Loading configuration from: %s\n", configPath)
	}

	configuration, err := config.LoadFromFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Configuration loaded:\n")
		fmt.Printf("  Original Serial:    %s\n", configuration.Serial)
		fmt.Printf("  Original Product:   %s %s\n", configuration.Manufacturer, configuration.Product)
		fmt.Printf("  Original Timestamp: %s\n", configuration.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Build Type:         %s\n", configuration.BuildType)
		fmt.Printf("  Frequency:          %.6f MHz\n", configuration.GetFrequencyMHz())
		fmt.Printf("  Sync Word:          0x%04X\n", configuration.GetSyncWord())
		fmt.Printf("  Modulation:         %s\n", configuration.GetModulationString())
	}

	// Create USB context
	context := gousb.NewContext()
	defer context.Close()

	// Find or open specific device
	var device *yardstick.Device

	if *serial != "" {
		device, err = yardstick.OpenDevice(context, *serial)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to open device with serial %s: %v\n", *serial, err)
			os.Exit(1)
		}
	} else {
		devices, err := yardstick.FindAllDevices(context)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to enumerate devices: %v\n", err)
			os.Exit(1)
		}
		if len(devices) == 0 {
			fmt.Fprintln(os.Stderr, "Error: No YardStick One devices found")
			os.Exit(1)
		}
		device = devices[0]
		// Close other devices
		for i := 1; i < len(devices); i++ {
			devices[i].Close()
		}
	}
	defer device.Close()

	if *verbose {
		fmt.Printf("\nConnected to: %s\n", device)
	}

	// Test connectivity with ping
	if *verbose {
		fmt.Print("Testing connectivity... ")
	}
	if err := device.Ping([]byte("TEST")); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Ping failed: %v\n", err)
		os.Exit(1)
	}
	if *verbose {
		fmt.Println("OK")
	}

	// Apply configuration
	if *verbose {
		fmt.Println("Applying configuration...")
	}

	if err := config.ApplyToDevice(device, configuration); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to apply configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration applied successfully")

	// Verify if requested
	if *verify {
		if *verbose {
			fmt.Println("\nVerifying configuration...")
		}

		readBack, err := config.DumpFromDevice(device)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to read back configuration for verification: %v\n", err)
		} else {
			errors := verifyConfig(configuration, readBack)
			if len(errors) > 0 {
				fmt.Fprintf(os.Stderr, "Verification failed with %d error(s):\n", len(errors))
				for _, e := range errors {
					fmt.Fprintf(os.Stderr, "  - %s\n", e)
				}
				os.Exit(1)
			}
			fmt.Println("Verification: OK")
		}
	}
}

func verifyConfig(expected, actual *config.DeviceConfig) []string {
	var errors []string

	// Compare key registers (not all, since some are read-only or volatile)
	e := &expected.Registers
	a := &actual.Registers

	if e.SYNC1 != a.SYNC1 || e.SYNC0 != a.SYNC0 {
		errors = append(errors, fmt.Sprintf("SYNC mismatch: expected 0x%02X%02X, got 0x%02X%02X",
			e.SYNC1, e.SYNC0, a.SYNC1, a.SYNC0))
	}

	if e.FREQ2 != a.FREQ2 || e.FREQ1 != a.FREQ1 || e.FREQ0 != a.FREQ0 {
		errors = append(errors, fmt.Sprintf("FREQ mismatch: expected 0x%02X%02X%02X, got 0x%02X%02X%02X",
			e.FREQ2, e.FREQ1, e.FREQ0, a.FREQ2, a.FREQ1, a.FREQ0))
	}

	if e.MDMCFG2 != a.MDMCFG2 {
		errors = append(errors, fmt.Sprintf("MDMCFG2 mismatch: expected 0x%02X, got 0x%02X",
			e.MDMCFG2, a.MDMCFG2))
	}

	if e.PKTLEN != a.PKTLEN {
		errors = append(errors, fmt.Sprintf("PKTLEN mismatch: expected %d, got %d",
			e.PKTLEN, a.PKTLEN))
	}

	if e.PKTCTRL0 != a.PKTCTRL0 {
		errors = append(errors, fmt.Sprintf("PKTCTRL0 mismatch: expected 0x%02X, got 0x%02X",
			e.PKTCTRL0, a.PKTCTRL0))
	}

	if e.PKTCTRL1 != a.PKTCTRL1 {
		errors = append(errors, fmt.Sprintf("PKTCTRL1 mismatch: expected 0x%02X, got 0x%02X",
			e.PKTCTRL1, a.PKTCTRL1))
	}

	return errors
}
