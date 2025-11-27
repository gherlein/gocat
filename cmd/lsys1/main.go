// lsys1: List all connected YardStick One devices
//
// This tool enumerates all YardStick One devices connected to the system
// and displays their serial numbers and basic information.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/google/gousb"
	"github.com/herlein/gocat/pkg/yardstick"
)

func main() {
	verbose := flag.Bool("v", false, "Verbose output (show additional device details)")
	flag.Parse()

	// Create USB context
	context := gousb.NewContext()
	defer context.Close()

	// Find all YardStick One devices
	devices, err := yardstick.FindAllDevices(context)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to enumerate devices: %v\n", err)
		os.Exit(1)
	}

	if len(devices) == 0 {
		fmt.Println("No YardStick One devices found")
		os.Exit(0)
	}

	fmt.Printf("Found %d YardStick One device(s):\n", len(devices))

	for i, device := range devices {
		defer device.Close()

		if *verbose {
			fmt.Printf("\nDevice %d:\n", i+1)
			fmt.Printf("  Serial:       %s\n", device.Serial)
			fmt.Printf("  Bus:          %d\n", device.Bus)
			fmt.Printf("  Address:      %d\n", device.Address)
			fmt.Printf("  Manufacturer: %s\n", device.Manufacturer)
			fmt.Printf("  Product:      %s\n", device.Product)

			// Try to get firmware info
			if buildType, err := device.GetBuildType(); err == nil {
				fmt.Printf("  Firmware:     %s\n", buildType)
			}

			// Try to get chip info
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
		} else {
			fmt.Printf("  %s (Bus %d, Addr %d)\n", device.Serial, device.Bus, device.Address)
		}
	}
}
