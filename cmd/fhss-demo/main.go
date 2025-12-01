// fhss-demo: Demonstration of Frequency Hopping Spread Spectrum (FHSS) with YardStick One
//
// This tool demonstrates FHSS functionality using two YardStick One devices.
// One device acts as the master (beacon transmitter) and the other as a client
// that synchronizes to the master's hopping pattern.
//
// Examples:
//
//	# Run as master (beacon transmitter) on device #0
//	./fhss-demo -mode master -d '#0' -c tests/etc/433-2fsk-std-4.8k.json
//
//	# Run as client (synchronized receiver) on device #1
//	./fhss-demo -mode client -d '#1' -c tests/etc/433-2fsk-std-4.8k.json
//
//	# Custom channel sequence with 10 channels
//	./fhss-demo -mode master -d '#0' -c tests/etc/433-2fsk-std-4.8k.json -channels 10
//
//	# Manual hopping test (no sync, just hop through channels)
//	./fhss-demo -mode manual -d '#0' -c tests/etc/433-2fsk-std-4.8k.json -channels 5
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/gousb"
	"github.com/herlein/gocat/pkg/config"
	"github.com/herlein/gocat/pkg/fhss"
	"github.com/herlein/gocat/pkg/yardstick"
)

func main() {
	mode := flag.String("mode", "", "Mode: 'master', 'client', or 'manual' (required)")
	configPath := flag.String("c", "", "Configuration file path (required)")
	deviceSel := flag.String("d", "", yardstick.DeviceFlagUsage())
	verbose := flag.Bool("v", false, "Verbose output")

	// FHSS options
	numChannels := flag.Int("channels", 20, "Number of channels in hop sequence")
	dwellMs := flag.Int("dwell", 100, "Dwell time per channel in milliseconds")
	cellID := flag.Uint("cell", 0, "Cell ID for synchronization (0-65535)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -mode <master|client|manual> -c <config.json> [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "FHSS demonstration for YardStick One devices\n\n")
		fmt.Fprintf(os.Stderr, "Modes:\n")
		fmt.Fprintf(os.Stderr, "  master  - Act as sync master, transmit beacons\n")
		fmt.Fprintf(os.Stderr, "  client  - Synchronize to master and receive\n")
		fmt.Fprintf(os.Stderr, "  manual  - Manual channel hopping (no sync)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Terminal 1 - Start master\n")
		fmt.Fprintf(os.Stderr, "  %s -mode master -d '#0' -c tests/etc/433-2fsk-std-4.8k.json\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Terminal 2 - Start client\n")
		fmt.Fprintf(os.Stderr, "  %s -mode client -d '#1' -c tests/etc/433-2fsk-std-4.8k.json\n", os.Args[0])
	}
	flag.Parse()

	if *mode == "" {
		fmt.Fprintln(os.Stderr, "Error: Mode (-mode) is required")
		flag.Usage()
		os.Exit(1)
	}

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "Error: Configuration file (-c) is required")
		flag.Usage()
		os.Exit(1)
	}

	*mode = strings.ToLower(*mode)
	if *mode != "master" && *mode != "client" && *mode != "manual" {
		fmt.Fprintf(os.Stderr, "Error: Invalid mode '%s'. Use 'master', 'client', or 'manual'\n", *mode)
		os.Exit(1)
	}

	if *numChannels < 2 || *numChannels > yardstick.FHSSMaxChannels {
		fmt.Fprintf(os.Stderr, "Error: channels must be between 2 and %d\n", yardstick.FHSSMaxChannels)
		os.Exit(1)
	}

	// Load configuration
	if *verbose {
		fmt.Printf("Loading configuration from: %s\n", *configPath)
	}

	configuration, err := config.LoadFromFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Configuration loaded:\n")
		fmt.Printf("  Base Frequency: %.6f MHz\n", configuration.GetFrequencyMHz())
		fmt.Printf("  Modulation:     %s\n", configuration.GetModulationString())
	}

	// Create USB context
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Select device
	device, err := yardstick.SelectDevice(ctx, yardstick.DeviceSelector(*deviceSel))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer device.Close()

	fmt.Printf("Connected to: %s (Serial: %s)\n", device.Product, device.Serial)

	// Test connectivity
	if err := device.Ping([]byte("FHSS")); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Device ping failed: %v\n", err)
		os.Exit(1)
	}

	// Apply radio configuration
	if *verbose {
		fmt.Println("Applying radio configuration...")
	}

	// Force IDLE state first
	if err := device.PokeByte(0xDFE1, 0x04); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to strobe IDLE: %v\n", err)
	}
	time.Sleep(50 * time.Millisecond)

	if err := config.ApplyToDevice(device, configuration); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to apply configuration: %v\n", err)
		os.Exit(1)
	}

	// Enable amplifiers
	if err := device.EnableAmplifier(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to enable amplifiers: %v\n", err)
	}

	// Create FHSS controller
	fh := fhss.New(device)

	// Generate channel sequence
	channels := make([]uint8, *numChannels)
	for i := range channels {
		channels[i] = uint8(i)
	}

	if *verbose {
		fmt.Printf("Setting up %d-channel hop sequence\n", *numChannels)
	}

	if err := fh.SetChannels(channels); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to set channels: %v\n", err)
		os.Exit(1)
	}

	// Set up signal handling for clean shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	switch *mode {
	case "master":
		runMaster(fh, device, *dwellMs, *verbose, sigChan)
	case "client":
		runClient(fh, device, uint16(*cellID), *verbose, sigChan)
	case "manual":
		runManual(fh, device, *dwellMs, *verbose, sigChan)
	}
}

func runMaster(fh *fhss.FHSS, device *yardstick.Device, dwellMs int, verbose bool, sigChan chan os.Signal) {
	fmt.Println("=== FHSS Master Mode ===")
	fmt.Printf("Dwell time: %d ms\n", dwellMs)
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Set as sync master
	if err := fh.BecomeMaster(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to become master: %v\n", err)
		os.Exit(1)
	}

	// Start hopping
	if err := fh.StartHopping(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to start hopping: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Master started - hopping and transmitting beacons")

	// Main loop - transmit beacon messages
	msgNum := 0
	ticker := time.NewTicker(time.Duration(dwellMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println("\nShutting down master...")
			fh.Stop()
			return
		case <-ticker.C:
			// Get current state
			state, err := fh.GetState()
			if err != nil {
				if verbose {
					fmt.Printf("Warning: Failed to get state: %v\n", err)
				}
				continue
			}

			// Transmit beacon
			beacon := fmt.Sprintf("BEACON:%06d", msgNum)
			if err := fh.Transmit([]byte(beacon)); err != nil {
				if verbose {
					fmt.Printf("Warning: Failed to transmit: %v\n", err)
				}
			} else {
				fmt.Printf("[%s] TX: %s\n", state, beacon)
				msgNum++
			}
		}
	}
}

func runClient(fh *fhss.FHSS, device *yardstick.Device, cellID uint16, verbose bool, sigChan chan os.Signal) {
	fmt.Println("=== FHSS Client Mode ===")
	fmt.Printf("Cell ID: %d\n", cellID)
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Start synchronization
	fmt.Println("Attempting to synchronize with master...")
	if err := fh.StartSync(cellID); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to start sync: %v\n", err)
		os.Exit(1)
	}

	// Put radio in RX mode
	if err := device.SetModeRX(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to set RX mode: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Client started - listening for beacons")

	// Main loop - receive and display
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println("\nShutting down client...")
			fh.Stop()
			return
		case <-ticker.C:
			// Check state
			state, err := fh.GetState()
			if err != nil {
				continue
			}

			// Try to receive
			data, err := device.RFRecv(50*time.Millisecond, 255)
			if err != nil {
				// Timeout is normal
				if verbose {
					fmt.Printf("[%s] Waiting...\n", state)
				}
				continue
			}

			if len(data) > 0 {
				fmt.Printf("[%s] RX: %s\n", state, string(data))
			}
		}
	}
}

func runManual(fh *fhss.FHSS, device *yardstick.Device, dwellMs int, verbose bool, sigChan chan os.Signal) {
	fmt.Println("=== FHSS Manual Mode ===")
	fmt.Printf("Dwell time: %d ms\n", dwellMs)
	fmt.Println("Manually hopping through channels (no sync)")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	hopCount := 0
	ticker := time.NewTicker(time.Duration(dwellMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println("\nStopping manual hopping...")
			return
		case <-ticker.C:
			// Hop to next channel
			ch, err := fh.NextChannel()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to hop: %v\n", err)
				continue
			}

			hopCount++
			fmt.Printf("Hop #%d -> Channel %d\n", hopCount, ch)

			// Optionally get MAC data for debugging
			if verbose {
				macData, err := fh.GetMACData()
				if err == nil {
					fmt.Printf("  State: %s, ChanIdx: %d, Hops: %d\n",
						macData.State, macData.CurChanIdx, macData.NumChannelHops)
				}
			}
		}
	}
}
