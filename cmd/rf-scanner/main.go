// rf-scanner is a firmware-based frequency scanner for the YardStick One
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/google/gousb"
	"github.com/herlein/gocat/pkg/specan"
	"github.com/herlein/gocat/pkg/yardstick"
)

var (
	centerFreq = flag.Float64("center", 433.92, "Center frequency in MHz")
	bandwidth  = flag.Float64("bw", 2.0, "Bandwidth in MHz")
	numChans   = flag.Int("chans", 100, "Number of channels (1-255)")
	threshold  = flag.Float64("threshold", -70.0, "RSSI threshold in dBm for peak detection")
	duration   = flag.Duration("duration", 0, "Scan duration (0 = indefinite)")
	deviceSel  = flag.String("d", "", yardstick.DeviceFlagUsage())
	listOnly   = flag.Bool("l", false, "List devices only")
	verbose    = flag.Bool("v", false, "Verbose output - show all frames")
	quiet      = flag.Bool("q", false, "Quiet mode - only show detected signals")
	csvOut     = flag.String("csv", "", "Output CSV file for spectrogram data")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Firmware-based RF Spectrum Analyzer for YardStick One\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -center 433.92 -bw 2           # Scan 432.92-434.92 MHz\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -center 915 -bw 10 -chans 200  # Wide scan at 915 MHz\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -threshold -80 -q              # Only show signals above -80 dBm\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -csv spectrum.csv -duration 10s # Save spectrogram data to CSV\n", os.Args[0])
	}
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := gousb.NewContext()
	defer ctx.Close()

	if *listOnly {
		return listDevices(ctx)
	}

	// Validate parameters
	if *numChans < 1 || *numChans > 255 {
		return fmt.Errorf("chans must be 1-255")
	}

	// Open device
	fmt.Println("Opening YardStick One...")
	device, err := yardstick.SelectDevice(ctx, yardstick.DeviceSelector(*deviceSel))
	if err != nil {
		return fmt.Errorf("failed to open device: %w", err)
	}
	defer device.Close()

	fmt.Printf("Connected to: %s\n", device)

	// Create spectrum analyzer
	sa := specan.New(device)

	// Configure
	cfg := &specan.Config{
		CenterFreq: uint32(*centerFreq * 1e6),
		Bandwidth:  uint32(*bandwidth * 1e6),
		NumChans:   uint8(*numChans),
	}

	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  Center:     %.3f MHz\n", *centerFreq)
	fmt.Printf("  Bandwidth:  %.3f MHz\n", *bandwidth)
	fmt.Printf("  Channels:   %d\n", *numChans)
	fmt.Printf("  Range:      %.3f - %.3f MHz\n",
		*centerFreq-*bandwidth/2, *centerFreq+*bandwidth/2)
	fmt.Printf("  Resolution: %.3f kHz per channel\n", *bandwidth*1000/float64(*numChans))
	fmt.Printf("  Threshold:  %.1f dBm\n", *threshold)
	if *csvOut != "" {
		fmt.Printf("  CSV Output: %s\n", *csvOut)
	}
	fmt.Println()

	if err := sa.Configure(cfg); err != nil {
		return fmt.Errorf("configure failed: %w", err)
	}

	// Set up CSV output if requested
	var csvFile *os.File
	var csvWriter *bufio.Writer
	if *csvOut != "" {
		var err error
		csvFile, err = os.Create(*csvOut)
		if err != nil {
			return fmt.Errorf("failed to create CSV file: %w", err)
		}
		defer csvFile.Close()
		csvWriter = bufio.NewWriter(csvFile)
		defer csvWriter.Flush()

		// Write header: timestamp_ms, freq1, freq2, freq3, ...
		freqs := make([]string, *numChans)
		baseFreq := *centerFreq - *bandwidth/2
		chanSpacing := *bandwidth / float64(*numChans)
		for i := 0; i < *numChans; i++ {
			freqs[i] = fmt.Sprintf("%.6f", baseFreq+float64(i)*chanSpacing)
		}
		fmt.Fprintf(csvWriter, "timestamp_ms,%s\n", strings.Join(freqs, ","))
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start analyzer
	if err := sa.Start(); err != nil {
		return fmt.Errorf("start failed: %w", err)
	}
	defer sa.Stop()

	// Set up timeout if specified
	var timeoutCtx context.Context
	var cancel context.CancelFunc
	if *duration > 0 {
		timeoutCtx, cancel = context.WithTimeout(context.Background(), *duration)
		fmt.Printf("Scanning for %v...\n", *duration)
	} else {
		timeoutCtx, cancel = context.WithCancel(context.Background())
		fmt.Println("Scanning... (Press Ctrl+C to stop)")
	}
	defer cancel()

	// Display header
	if !*quiet {
		fmt.Println("\n Frame | Max Freq (MHz) | Max RSSI | Avg RSSI | Peaks")
		fmt.Println("-------+----------------+----------+----------+-------")
	}

	frameCount := 0
	peakCount := 0

	for {
		select {
		case <-sigChan:
			fmt.Println("\n\nStopping...")
			goto done

		case <-timeoutCtx.Done():
			goto done

		case frame, ok := <-sa.Frames():
			if !ok {
				goto done
			}

			frameCount++
			maxIdx, maxFreq, maxRSSI := specan.MaxRSSI(frame)
			avgRSSI := specan.AverageRSSI(frame)
			peaks := specan.FindPeaks(frame, float32(*threshold))

			// Write CSV row if output file specified
			if csvWriter != nil {
				// timestamp in milliseconds since Unix epoch
				tsMs := frame.Timestamp.UnixMilli()
				rssiStrs := make([]string, len(frame.RSSI))
				for i, rssi := range frame.RSSI {
					rssiStrs[i] = fmt.Sprintf("%.1f", rssi)
				}
				fmt.Fprintf(csvWriter, "%d,%s\n", tsMs, strings.Join(rssiStrs, ","))
			}

			if len(peaks) > 0 {
				peakCount += len(peaks)
				if *quiet {
					// Quiet mode: only show peaks
					for _, p := range peaks {
						fmt.Printf("SIGNAL: %.3f MHz @ %.1f dBm\n",
							float64(p.FrequencyHz)/1e6, p.RSSI)
					}
				}
			}

			if !*quiet {
				if *verbose || len(peaks) > 0 {
					fmt.Printf(" %5d | %14.3f | %8.1f | %8.1f | %d\n",
						frameCount, float64(maxFreq)/1e6, maxRSSI, avgRSSI, len(peaks))
				} else if frameCount%50 == 0 {
					// Periodic status update
					fmt.Printf(" %5d | %14.3f | %8.1f | %8.1f | scanning...\n",
						frameCount, float64(maxFreq)/1e6, maxRSSI, avgRSSI)
				}
			}

			// Debug: print full spectrum on verbose with signal
			if *verbose && len(peaks) > 0 && maxIdx >= 0 {
				fmt.Printf("        Channel %d: raw index in spectrum\n", maxIdx)
			}
		}
	}

done:
	fmt.Printf("\n--- Summary ---\n")
	fmt.Printf("Frames:  %d\n", frameCount)
	fmt.Printf("Signals: %d (above %.1f dBm)\n", peakCount, *threshold)
	return nil
}

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
	return nil
}
