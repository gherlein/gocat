// rf-scanner is a frequency scanner for the YardStick One
// It detects RF signals across sub-GHz frequencies and displays information about them.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/gousb"
	"github.com/herlein/gocat/pkg/scanner"
	"github.com/herlein/gocat/pkg/yardstick"
)

var (
	configPath  = flag.String("config", "", "Path to JSON configuration file")
	threshold   = flag.Float64("threshold", -93.0, "RSSI threshold in dBm (overrides config)")
	duration    = flag.Duration("duration", 0, "Scan duration (0 = indefinite)")
	deviceSel   = flag.String("d", "", yardstick.DeviceFlagUsage())
	listOnly    = flag.Bool("l", false, "List devices only, don't scan")
	verbose     = flag.Bool("verbose", false, "Verbose output")
	debug       = flag.Bool("debug", false, "Enable detailed debug output")
	showHistory = flag.Bool("history", false, "Show all detected signals on exit")
	continuous  = flag.Bool("continuous", true, "Continuous scan mode")
	singleShot  = flag.Bool("single", false, "Single scan only")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "RF Frequency Scanner for YardStick One\n\n")
		fmt.Fprintf(os.Stderr, "Scans sub-GHz frequencies and detects RF signals.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                           # Scan with defaults\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --config etc/scanner/433-only.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --threshold -85 --duration 30s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --single --verbose         # Single scan with details\n", os.Args[0])
	}
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Create USB context
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Handle list-only mode
	if *listOnly {
		return listDevices(ctx)
	}

	// Open YardStick One device
	fmt.Println("Opening YardStick One...")
	device, err := yardstick.SelectDevice(ctx, yardstick.DeviceSelector(*deviceSel))
	if err != nil {
		return fmt.Errorf("failed to open device: %w", err)
	}
	defer device.Close()

	fmt.Printf("Connected to: %s\n", device)

	// Load or create configuration
	var scanConfig *scanner.ScanConfig
	var configFile *scanner.ConfigFile

	if *configPath != "" {
		fmt.Printf("Loading configuration from: %s\n", *configPath)
		configFile, err = scanner.LoadConfigFile(*configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		scanConfig = configFile.ToScanConfig()
		fmt.Printf("Configuration: %s - %s\n", configFile.Name, configFile.Description)
	} else {
		scanConfig = scanner.DefaultConfig()
		fmt.Println("Using default configuration")
	}

	// Apply command-line overrides
	if *threshold != -93.0 {
		scanConfig.RSSIThreshold = float32(*threshold)
	}

	// Set up callbacks for signal events
	scanConfig.OnSignalDetected = func(info *scanner.SignalInfo) {
		fmt.Printf("\n>>> SIGNAL DETECTED: %.3f MHz @ %.1f dBm\n",
			float64(info.Frequency)/1e6, info.RSSI)
	}

	scanConfig.OnSignalLost = func(info *scanner.SignalInfo) {
		fmt.Printf("\n<<< SIGNAL LOST: %.3f MHz (seen %d times, max %.1f dBm)\n",
			float64(info.Frequency)/1e6, info.DetectionCount, info.MaxRSSI)
	}

	// Set up debug logging if enabled
	if *debug {
		scanConfig.DebugLog = func(format string, args ...interface{}) {
			fmt.Printf("[DEBUG] "+format+"\n", args...)
		}
	}

	// Validate configuration
	if err := scanConfig.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create scanner
	s := scanner.New(device, scanConfig)

	// Print configuration summary
	printConfig(scanConfig)

	// Debug: Print register state before scanning
	if *debug {
		fmt.Println("\n[DEBUG] Initial radio register state:")
		if freq2, err := device.PeekByte(0xDF09); err == nil {
			freq1, _ := device.PeekByte(0xDF0A)
			freq0, _ := device.PeekByte(0xDF0B)
			freqReg := uint32(freq2)<<16 | uint32(freq1)<<8 | uint32(freq0)
			freqHz := float64(freqReg) * 24000000.0 / 65536.0
			fmt.Printf("[DEBUG]   FREQ: 0x%02X%02X%02X = %.3f MHz\n", freq2, freq1, freq0, freqHz/1e6)
		}
		if mdmcfg4, err := device.PeekByte(0xDF0C); err == nil {
			fmt.Printf("[DEBUG]   MDMCFG4: 0x%02X\n", mdmcfg4)
		}
		if mdmcfg3, err := device.PeekByte(0xDF0D); err == nil {
			fmt.Printf("[DEBUG]   MDMCFG3: 0x%02X\n", mdmcfg3)
		}
		if mdmcfg2, err := device.PeekByte(0xDF0E); err == nil {
			fmt.Printf("[DEBUG]   MDMCFG2: 0x%02X (MOD_FORMAT=%d, SYNC_MODE=%d)\n",
				mdmcfg2, (mdmcfg2>>4)&0x07, mdmcfg2&0x07)
		}
		if agcctrl2, err := device.PeekByte(0xDF17); err == nil {
			fmt.Printf("[DEBUG]   AGCCTRL2: 0x%02X\n", agcctrl2)
		}
		if rssi, err := device.GetRSSI(); err == nil {
			fmt.Printf("[DEBUG]   Current RSSI: raw=0x%02X = %.1f dBm\n", rssi, scanner.RSSIToDBm(rssi))
		}
		if marcstate, err := device.PeekByte(0xDF3B); err == nil {
			fmt.Printf("[DEBUG]   MARCSTATE: 0x%02X\n", marcstate)
		}
		fmt.Println()
	}

	// Handle single scan mode
	if *singleShot {
		return runSingleScan(s)
	}

	// Continuous scan mode
	return runContinuousScan(s, scanConfig)
}

func printConfig(config *scanner.ScanConfig) {
	fmt.Println("\n--- Scanner Configuration ---")
	fmt.Printf("Frequencies:     %d coarse frequencies\n", len(config.CoarseFrequencies))
	fmt.Printf("RSSI Threshold:  %.1f dBm\n", config.RSSIThreshold)
	fmt.Printf("Fine Scan:       +/- %d kHz in %d kHz steps\n",
		config.FineScanRange/1000, config.FineScanStep/1000)
	fmt.Printf("Dwell Time:      %v\n", config.DwellTime)
	fmt.Printf("Smoothing:       %v\n", config.SmoothingEnabled)

	if *verbose {
		fmt.Println("\nFrequency list:")
		for i, freq := range config.CoarseFrequencies {
			fmt.Printf("  %2d. %.3f MHz (%s)\n", i+1, float64(freq)/1e6, scanner.FrequencyBand(freq))
		}
	}
	fmt.Println("-----------------------------")
	fmt.Println()
}

func runSingleScan(s scanner.Scanner) error {
	fmt.Println("Performing single scan...")
	startTime := time.Now()

	result, err := s.ScanOnce()
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	elapsed := time.Since(startTime)

	fmt.Println("\n--- Scan Result ---")
	fmt.Printf("Duration:        %v\n", elapsed)
	fmt.Printf("Signal Detected: %v\n", result.SignalDetected)
	fmt.Printf("Coarse Freq:     %.3f MHz\n", float64(result.CoarseFrequency)/1e6)
	fmt.Printf("Coarse RSSI:     %.1f dBm\n", result.CoarseRSSI)

	if result.SignalDetected {
		fmt.Printf("Fine Freq:       %.3f MHz\n", float64(result.FineFrequency)/1e6)
		fmt.Printf("Fine RSSI:       %.1f dBm\n", result.FineRSSI)
		fmt.Printf("Band:            %s\n", scanner.FrequencyBand(result.FineFrequency))
	}
	fmt.Println("-------------------")

	return nil
}

func runContinuousScan(s scanner.Scanner, config *scanner.ScanConfig) error {
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create context with optional timeout
	var scanCtx context.Context
	var cancel context.CancelFunc

	if *duration > 0 {
		scanCtx, cancel = context.WithTimeout(context.Background(), *duration)
		fmt.Printf("Scanning for %v...\n", *duration)
	} else {
		scanCtx, cancel = context.WithCancel(context.Background())
		fmt.Println("Scanning... (Press Ctrl+C to stop)")
	}
	defer cancel()

	// Create results channel
	results := make(chan *scanner.ScanResult, 10)

	// Start scanning in goroutine
	scanErrChan := make(chan error, 1)
	go func() {
		scanErrChan <- s.ScanContinuous(scanCtx, results)
	}()

	// Track statistics
	var scanCount uint64
	var signalCount uint64
	lastPrint := time.Now()
	shutdownRequested := false

	// Display header
	fmt.Println("\n Scan# | Frequency (MHz) | RSSI (dBm) | Status")
	fmt.Println("-------+-----------------+------------+--------")

	// Process results
	for {
		select {
		case <-sigChan:
			if shutdownRequested {
				// Second signal - force exit
				fmt.Println("\n\nForced exit.")
				return nil
			}
			fmt.Println("\n\nShutting down... (press Ctrl+C again to force)")
			shutdownRequested = true
			cancel()
			// Set a timeout to force exit if scanner doesn't stop
			go func() {
				time.Sleep(2 * time.Second)
				fmt.Println("\nScanner did not stop in time, forcing exit.")
				os.Exit(0)
			}()

		case result, ok := <-results:
			if !ok {
				// Channel closed, we're done
				goto done
			}

			scanCount++

			if result.SignalDetected {
				signalCount++
				freq := result.FineFrequency
				if freq == 0 {
					freq = result.CoarseFrequency
				}
				rssi := result.FineRSSI
				if rssi < -199 {
					rssi = result.CoarseRSSI
				}

				fmt.Printf(" %5d | %15.3f | %10.1f | DETECTED\n",
					scanCount, float64(freq)/1e6, rssi)
			} else if *verbose {
				// Show all scans in verbose mode
				fmt.Printf(" %5d | %15.3f | %10.1f | quiet\n",
					scanCount, float64(result.CoarseFrequency)/1e6, result.CoarseRSSI)
			} else {
				// Periodic status update
				if time.Since(lastPrint) > 2*time.Second {
					fmt.Printf(" %5d | %15s | %10s | scanning...\n",
						scanCount, "---", "---")
					lastPrint = time.Now()
				}
			}

		case err := <-scanErrChan:
			if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
				return fmt.Errorf("scan error: %w", err)
			}
			goto done
		}
	}

done:
	// Print summary
	fmt.Println("\n--- Scan Summary ---")
	fmt.Printf("Total Scans:     %d\n", scanCount)
	fmt.Printf("Signals Found:   %d\n", signalCount)

	if *showHistory {
		printSignalHistory(s)
	}

	return nil
}

func printSignalHistory(s scanner.Scanner) {
	signals := s.GetActiveSignals()
	if len(signals) == 0 {
		fmt.Println("\nNo signals recorded in history.")
		return
	}

	fmt.Println("\n--- Signal History ---")
	fmt.Println(" Frequency (MHz) | Count |  Max RSSI  | First Seen          | Last Seen")
	fmt.Println("-----------------+-------+------------+---------------------+--------------------")

	for _, sig := range signals {
		fmt.Printf(" %15.3f | %5d | %8.1f dB | %s | %s\n",
			float64(sig.Frequency)/1e6,
			sig.DetectionCount,
			sig.MaxRSSI,
			sig.FirstSeen.Format("2006-01-02 15:04:05"),
			sig.LastSeen.Format("2006-01-02 15:04:05"),
		)
	}

	// Print frequency summary by band
	fmt.Println("\n--- Signals by Band ---")
	bands := make(map[string]int)
	for _, sig := range signals {
		band := scanner.FrequencyBand(sig.Frequency)
		bands[band]++
	}
	for band, count := range bands {
		fmt.Printf("  %s: %d signals\n", band, count)
	}
}

// formatFrequency formats a frequency in Hz to a human-readable string
func formatFrequency(freqHz uint32) string {
	if freqHz >= 1000000000 {
		return fmt.Sprintf("%.3f GHz", float64(freqHz)/1e9)
	}
	return fmt.Sprintf("%.3f MHz", float64(freqHz)/1e6)
}

// formatRSSI formats an RSSI value
func formatRSSI(rssi float32) string {
	if rssi < -199 {
		return "---"
	}
	return fmt.Sprintf("%.1f dBm", rssi)
}

// progressBar creates a simple ASCII progress bar
func progressBar(value, max float32, width int) string {
	if max <= 0 {
		return strings.Repeat(" ", width)
	}

	// Normalize to 0-1 range (RSSI is negative, so we invert)
	ratio := (value + 120) / 60 // -120 to -60 dBm range
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	filled := int(ratio * float32(width))
	if filled > width {
		filled = width
	}

	return strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
}

// listDevices lists all connected YardStick One devices
func listDevices(ctx *gousb.Context) error {
	devices, err := yardstick.FindAllDevices(ctx)
	if err != nil {
		return fmt.Errorf("failed to list devices: %w", err)
	}

	if len(devices) == 0 {
		fmt.Println("No YardStick One devices found")
		return nil
	}

	fmt.Printf("Found %d YardStick One device(s):\n\n", len(devices))
	for i, d := range devices {
		defer d.Close()
		fmt.Printf("  #%d  %s  %d:%d\n", i, d.Serial, d.Bus, d.Address)
	}
	fmt.Println()
	fmt.Println("Use -d to select a device:")
	fmt.Println("  -d \"#0\"      Select by index")
	fmt.Println("  -d \"1:10\"    Select by bus:address")
	fmt.Println("  -d \"009a\"    Select by serial (if unique)")
	return nil
}
