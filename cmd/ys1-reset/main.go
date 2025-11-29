// ys1-reset resets YardStick One devices to recover from USB errors
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/google/gousb"
)

func main() {
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Try multiple times to find devices
	for attempt := 0; attempt < 3; attempt++ {
		devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
			return desc.Vendor == 0x1d50 && desc.Product == 0x605b
		})

		if err != nil {
			fmt.Printf("Attempt %d: Error finding devices: %v\n", attempt+1, err)
			time.Sleep(time.Second)
			continue
		}

		if len(devs) == 0 {
			fmt.Printf("Attempt %d: No devices found\n", attempt+1)
			time.Sleep(time.Second)
			continue
		}

		fmt.Printf("Found %d device(s)\n", len(devs))
		for i, dev := range devs {
			serial, _ := dev.SerialNumber()
			fmt.Printf("  Device %d: %s\n", i, serial)

			// Reset the device
			if err := dev.Reset(); err != nil {
				fmt.Printf("    Reset failed: %v\n", err)
			} else {
				fmt.Printf("    Reset OK\n")
			}
			dev.Close()
		}
		os.Exit(0)
	}

	fmt.Println("Failed to find/reset devices after 3 attempts")
	os.Exit(1)
}
