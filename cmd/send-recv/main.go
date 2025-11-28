// send-recv: Example program for sending and receiving RF data with YardStick One
//
// This tool demonstrates how to configure a YardStick One for RF transmission
// and reception. It can operate in either send or receive mode.
//
// Examples:
//
//	# Receive mode - listen for packets and display them
//	./send-recv -m recv -c etc/defaults.json
//
//	# Send mode - transmit data from command line
//	./send-recv -m send -c etc/defaults.json -data "Hello World"
//
//	# Send mode - transmit hex data
//	./send-recv -m send -c etc/defaults.json -hex "DEADBEEF"
//
//	# Send mode - repeat transmission 10 times
//	./send-recv -m send -c etc/defaults.json -data "test" -repeat 10
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/gousb"
	"github.com/herlein/gocat/pkg/config"
	"github.com/herlein/gocat/pkg/yardstick"
)

func main() {
	// Parse command line flags
	mode := flag.String("m", "", "Mode: 'send' or 'recv' (required)")
	configPath := flag.String("c", "", "Configuration file path (required)")
	deviceSel := flag.String("d", "", yardstick.DeviceFlagUsage())
	verbose := flag.Bool("v", false, "Verbose output")

	// Send mode options
	dataStr := flag.String("data", "", "Data to send (ASCII string)")
	hexStr := flag.String("hex", "", "Data to send (hex encoded)")
	repeat := flag.Uint("repeat", 0, "Number of times to repeat transmission (0 = once)")
	offset := flag.Uint("offset", 0, "Offset for repeat transmissions")

	// Receive mode options
	timeout := flag.Duration("timeout", 1*time.Second, "Receive timeout per packet")
	count := flag.Int("count", 0, "Number of packets to receive (0 = infinite)")
	rawOutput := flag.Bool("raw", false, "Output raw hex only (for piping)")

	flag.Parse()

	// Validate required arguments
	if *mode == "" {
		fmt.Fprintln(os.Stderr, "Error: Mode (-m) is required. Use 'send' or 'recv'")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "Error: Configuration file (-c) is required")
		flag.PrintDefaults()
		os.Exit(1)
	}

	*mode = strings.ToLower(*mode)
	if *mode != "send" && *mode != "recv" {
		fmt.Fprintf(os.Stderr, "Error: Invalid mode '%s'. Use 'send' or 'recv'\n", *mode)
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
		fmt.Printf("  Frequency:    %.6f MHz\n", configuration.GetFrequencyMHz())
		fmt.Printf("  Modulation:   %s\n", configuration.GetModulationString())
		fmt.Printf("  Sync Word:    0x%04X\n", configuration.GetSyncWord())
		fmt.Printf("  Packet Len:   %d\n", configuration.Registers.PKTLEN)
	}

	// Create USB context
	context := gousb.NewContext()
	defer context.Close()

	// Select device
	device, err := yardstick.SelectDevice(context, yardstick.DeviceSelector(*deviceSel))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer device.Close()

	if *verbose {
		fmt.Printf("Connected to: %s (Bus %d, Addr %d)\n", device.Serial, device.Bus, device.Address)
	}

	// Test connectivity
	if err := device.Ping([]byte("TEST")); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Device ping failed: %v\n", err)
		os.Exit(1)
	}

	// Apply configuration
	if *verbose {
		fmt.Println("Applying radio configuration...")
		fmt.Println("  Setting IDLE state...")
	}

	// Force IDLE state first with direct strobe
	if err := device.PokeByte(0xDFE1, 0x04); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to strobe IDLE: %v\n", err)
	}
	time.Sleep(50 * time.Millisecond)

	if *verbose {
		fmt.Println("  Writing registers...")
	}

	if err := config.ApplyToDevice(device, configuration); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to apply configuration: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Println("  Configuration applied.")
	}

	// Enable YS1 front-end amplifiers for better TX power and RX sensitivity
	if err := device.SetAmpMode(1); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to enable amplifiers: %v\n", err)
	} else if *verbose {
		fmt.Println("Amplifiers enabled")
	}

	// Verify configuration by reading back key registers
	if *verbose {
		sync1, _ := device.PeekByte(0xDF00)
		sync0, _ := device.PeekByte(0xDF01)
		pktlen, _ := device.PeekByte(0xDF02)
		mdmcfg2, _ := device.PeekByte(0xDF0E)
		freq2, _ := device.PeekByte(0xDF09)
		freq1, _ := device.PeekByte(0xDF0A)
		freq0, _ := device.PeekByte(0xDF0B)
		pa0, _ := device.PeekByte(0xDF2E)
		fmt.Printf("Verified: SYNC=0x%02X%02X PKTLEN=%d MDMCFG2=0x%02X FREQ=0x%02X%02X%02X PA0=0x%02X\n",
			sync1, sync0, pktlen, mdmcfg2, freq2, freq1, freq0, pa0)
	}

	// Run appropriate mode
	switch *mode {
	case "send":
		runSendMode(device, *dataStr, *hexStr, uint16(*repeat), uint16(*offset), *verbose)
	case "recv":
		runRecvMode(device, *timeout, *count, *verbose, *rawOutput)
	}
}

func runSendMode(device *yardstick.Device, dataStr, hexStr string, repeat, offset uint16, verbose bool) {
	// Determine data to send
	var data []byte

	if hexStr != "" {
		var err error
		data, err = hex.DecodeString(hexStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid hex string: %v\n", err)
			os.Exit(1)
		}
	} else if dataStr != "" {
		data = []byte(dataStr)
	} else {
		fmt.Fprintln(os.Stderr, "Error: Must specify -data or -hex for send mode")
		os.Exit(1)
	}

	if len(data) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No data to send")
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Transmitting %d bytes", len(data))
		if repeat > 0 {
			fmt.Printf(" (repeat %d times, offset %d)", repeat, offset)
		}
		fmt.Println()
		fmt.Printf("Data (hex): %s\n", hex.EncodeToString(data))
	}

	// Transmit data
	err := device.RFXmit(data, repeat, offset)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Transmit failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Transmission complete")
}

func runRecvMode(device *yardstick.Device, timeout time.Duration, count int, verbose, rawOutput bool) {
	// Set up signal handler for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Enter receive mode
	if verbose {
		fmt.Println("Entering receive mode...")
	}

	if err := device.SetModeRX(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to enter RX mode: %v\n", err)
		os.Exit(1)
	}

	// Show initial radio status in verbose mode
	if verbose {
		status, err := device.GetRadioStatus()
		if err == nil {
			fmt.Printf("Initial radio state: MARCSTATE=0x%02X RSSI=%d dBm\n",
				status.MARCSTATE, status.RSSIdBm)
		}
	}

	if !rawOutput {
		fmt.Println("Listening for packets (Ctrl+C to stop)...")
		fmt.Println()
	}

	packetsReceived := 0
	timeouts := 0
	startTime := time.Now()

	// Use shorter internal timeout for more responsive signal handling
	recvTimeout := 200 * time.Millisecond
	if timeout < recvTimeout {
		recvTimeout = timeout
	}

	for {
		// Check for shutdown signal (non-blocking)
		select {
		case <-sigChan:
			if !rawOutput {
				fmt.Printf("\n\nReceived %d packets, %d timeouts in %v\n",
					packetsReceived, timeouts, time.Since(startTime).Round(time.Second))
			}
			return
		default:
		}

		// Try to receive a packet with short timeout for responsive Ctrl+C
		data, err := device.RFRecv(recvTimeout, 0)
		if err != nil {
			// Timeout is normal, continue
			timeouts++
			if verbose && timeouts%5 == 0 {
				// Periodic status update every 5 timeouts (1 second)
				status, serr := device.GetRadioStatus()
				if serr == nil {
					fmt.Printf("  [waiting] timeouts=%d MARCSTATE=0x%02X RSSI=%d dBm PKTSTATUS=0x%02X\n",
						timeouts, status.MARCSTATE, status.RSSIdBm, status.PKTSTATUS)
				}
			}
			continue
		}

		// Get radio status immediately after receiving
		status, _ := device.GetRadioStatus()

		// Filter out packets with bad CRC (software filter since hardware doesn't always filter)
		if status != nil && !status.CRCOk {
			if verbose {
				fmt.Printf("  [dropped] CRC failed, %d bytes\n", len(data))
			}
			continue
		}

		packetsReceived++
		timestamp := time.Now()

		if rawOutput {
			// Raw hex output for piping
			fmt.Println(hex.EncodeToString(data))
		} else {
			// Formatted output with radio diagnostics
			fmt.Printf("[%s] Packet #%d (%d bytes):\n",
				timestamp.Format("15:04:05.000"),
				packetsReceived,
				len(data))

			if status != nil {
				crcStr := "NO"
				if status.CRCOk {
					crcStr = "OK"
				}
				fmt.Printf("  RSSI: %d dBm, LQI: %d, CRC: %s, PKTSTATUS: 0x%02X\n",
					status.RSSIdBm, status.LQI, crcStr, status.PKTSTATUS)
			}

			fmt.Printf("  Hex: %s\n", hex.EncodeToString(data))
			if len(data) <= 64 {
				fmt.Printf("  ASCII: %s\n", makePrintable(data))
			} else {
				fmt.Printf("  ASCII: %s... (truncated)\n", makePrintable(data[:64]))
			}
			fmt.Println()
		}

		// Check packet count limit
		if count > 0 && packetsReceived >= count {
			if !rawOutput {
				fmt.Printf("Received requested %d packets\n", count)
			}
			return
		}
	}
}

// makePrintable converts bytes to a printable string, replacing non-printable characters
func makePrintable(data []byte) string {
	result := make([]byte, len(data))
	for i, b := range data {
		if b >= 32 && b < 127 {
			result[i] = b
		} else {
			result[i] = '.'
		}
	}
	return string(result)
}
