// ys1-dump-config: Dump YardStick One configuration to JSON file
//
// This tool connects to a YardStick One device, reads its current radio
// configuration, and saves it to a JSON file. The configuration can later
// be loaded using ys1-load-config.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/google/gousb"
	"github.com/herlein/gocat/pkg/config"
	"github.com/herlein/gocat/pkg/yardstick"
)

func main() {
	// Parse command line flags
	outputFile := flag.String("o", "", "Output file path (default: etc/yardsticks/<serial>.json)")
	deviceSel := flag.String("d", "", yardstick.DeviceFlagUsage())
	verbose := flag.Bool("v", false, "Verbose output")
	listOnly := flag.Bool("l", false, "List devices only, don't dump config")
	jsonOutput := flag.Bool("json", false, "Output config to stdout as JSON instead of file")
	flag.Parse()

	// Create USB context
	context := gousb.NewContext()
	defer context.Close()

	if *listOnly {
		listDevices(context)
		return
	}

	// Select device
	device, err := yardstick.SelectDevice(context, yardstick.DeviceSelector(*deviceSel))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer device.Close()

	if *verbose {
		fmt.Printf("Connected to: %s\n", device)
	}

	// Test connectivity with ping
	if *verbose {
		fmt.Print("Testing connectivity... ")
	}
	if err := device.Ping([]byte("PING")); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Ping failed: %v\n", err)
		os.Exit(1)
	}
	if *verbose {
		fmt.Println("OK")
	}

	// Dump configuration
	if *verbose {
		fmt.Println("Reading device configuration...")
	}

	configuration, err := config.DumpFromDevice(device)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to dump configuration: %v\n", err)
		os.Exit(1)
	}

	// Output to stdout as JSON
	if *jsonOutput {
		data, err := json.MarshalIndent(configuration, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to marshal configuration: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	// Determine output path
	path := *outputFile
	if path == "" {
		path = config.GetConfigPath(device.Serial)
	}

	// Save to file
	if err := config.SaveToFile(configuration, path); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to save configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Configuration saved to: %s\n", path)

	// Print summary
	if *verbose {
		printConfigSummary(configuration)
	}
}

func listDevices(context *gousb.Context) {
	devices, err := yardstick.FindAllDevices(context)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to enumerate devices: %v\n", err)
		os.Exit(1)
	}

	if len(devices) == 0 {
		fmt.Println("No YardStick One devices found")
		return
	}

	fmt.Printf("Found %d YardStick One device(s):\n\n", len(devices))

	for i, device := range devices {
		defer device.Close()

		fmt.Printf("Device %d:\n", i+1)
		fmt.Printf("  Manufacturer: %s\n", device.Manufacturer)
		fmt.Printf("  Product:      %s\n", device.Product)
		fmt.Printf("  Serial:       %s\n", device.Serial)

		// Try to get additional info
		if buildType, err := device.GetBuildType(); err == nil {
			fmt.Printf("  Firmware:     %s\n", buildType)
		}
		if partNum, err := device.GetPartNum(); err == nil {
			chipName := "Unknown"
			switch partNum {
			case yardstick.PartNumCC1110:
				chipName = "CC1110"
			case yardstick.PartNumCC1111:
				chipName = "CC1111"
			case yardstick.PartNumCC2510:
				chipName = "CC2510"
			case yardstick.PartNumCC2511:
				chipName = "CC2511"
			}
			fmt.Printf("  Chip:         %s (0x%02X)\n", chipName, partNum)
		}
		fmt.Println()
	}
}

func printConfigSummary(cfg *config.DeviceConfig) {
	fmt.Println("\nConfiguration Summary:")
	fmt.Printf("  Build Type:   %s\n", cfg.BuildType)
	fmt.Printf("  Frequency:    %.6f MHz\n", cfg.GetFrequencyMHz())
	fmt.Printf("  Sync Word:    0x%04X\n", cfg.GetSyncWord())
	fmt.Printf("  Modulation:   %s\n", cfg.GetModulationString())
	fmt.Printf("  Radio State:  %s\n", cfg.GetRadioStateString())
	fmt.Printf("  Packet Len:   %d\n", cfg.Registers.PKTLEN)
}
