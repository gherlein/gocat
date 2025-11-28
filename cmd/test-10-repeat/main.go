// test-10-repeat: Test RF reliability between two YardStick One devices
//
// This program opens two YS1 devices, assigns them as sender and receiver,
// and tests packet delivery reliability at progressively faster rates.
//
// Usage:
//
//	./test-10-repeat -c etc/defaults.json
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/gousb"
	"github.com/herlein/gocat/pkg/config"
	"github.com/herlein/gocat/pkg/yardstick"
)

type TestResult struct {
	Delay        time.Duration
	Sent         int
	Received     int
	Matched      int
	Mismatched   int
	SuccessRate  float64
	AvgRSSI      int
	MinRSSI      int
	MaxRSSI      int
	AvgLatency   time.Duration
	RecvTimeouts int
}

func main() {
	configPath := flag.String("c", "etc/defaults.json", "Configuration file path")
	packetCount := flag.Int("n", 10, "Number of packets per test run")
	initialDelay := flag.Duration("delay", 1*time.Second, "Initial delay between packets")
	minDelay := flag.Duration("min-delay", 10*time.Millisecond, "Minimum delay between packets")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	// Load configuration
	fmt.Printf("Loading configuration from: %s\n", *configPath)
	configuration, err := config.LoadFromFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Configuration:\n")
	fmt.Printf("  Frequency:  %.6f MHz\n", configuration.GetFrequencyMHz())
	fmt.Printf("  Modulation: %s\n", configuration.GetModulationString())
	fmt.Printf("  Sync Word:  0x%04X\n", configuration.GetSyncWord())
	fmt.Printf("  Packet Len: %d\n", configuration.Registers.PKTLEN)
	fmt.Println()

	// Create USB context
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Find all YS1 devices
	devices, err := yardstick.FindAllDevices(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to find devices: %v\n", err)
		os.Exit(1)
	}

	if len(devices) < 2 {
		fmt.Fprintf(os.Stderr, "Error: Need at least 2 YardStick One devices, found %d\n", len(devices))
		for _, d := range devices {
			d.Close()
		}
		os.Exit(1)
	}

	// Sort by bus:address to get consistent assignment
	sort.Slice(devices, func(i, j int) bool {
		if devices[i].Bus != devices[j].Bus {
			return devices[i].Bus < devices[j].Bus
		}
		return devices[i].Address < devices[j].Address
	})

	// Close any extra devices
	for i := 2; i < len(devices); i++ {
		devices[i].Close()
	}

	sender := devices[0]
	receiver := devices[1]

	fmt.Printf("Sender:   %s (Bus %d, Addr %d)\n", sender.Serial, sender.Bus, sender.Address)
	fmt.Printf("Receiver: %s (Bus %d, Addr %d)\n", receiver.Serial, receiver.Bus, receiver.Address)
	fmt.Println()

	defer sender.Close()
	defer receiver.Close()

	// Configure both devices
	fmt.Println("Configuring devices...")

	// Force IDLE state first
	sender.PokeByte(0xDFE1, 0x04)
	receiver.PokeByte(0xDFE1, 0x04)
	time.Sleep(50 * time.Millisecond)

	if err := config.ApplyToDevice(sender, configuration); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to configure sender: %v\n", err)
		os.Exit(1)
	}
	if err := config.ApplyToDevice(receiver, configuration); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to configure receiver: %v\n", err)
		os.Exit(1)
	}

	// Enable amplifiers
	if err := sender.SetAmpMode(1); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to enable sender amplifiers: %v\n", err)
	}
	if err := receiver.SetAmpMode(1); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to enable receiver amplifiers: %v\n", err)
	}

	// Verify configuration
	if *verbose {
		verifySenderConfig(sender)
		verifyReceiverConfig(receiver)
	}

	fmt.Println("Configuration complete.")
	fmt.Println()

	// Run tests at progressively faster rates
	var results []TestResult
	delay := *initialDelay

	for delay >= *minDelay {
		fmt.Printf("========================================\n")
		fmt.Printf("TEST RUN: %d packets, %v delay\n", *packetCount, delay)
		fmt.Printf("========================================\n")

		result := runTest(sender, receiver, *packetCount, delay, *verbose)
		results = append(results, result)

		fmt.Printf("\nResult: %d/%d packets received (%.1f%% success)\n",
			result.Received, result.Sent, result.SuccessRate)
		fmt.Printf("        Matched: %d, Mismatched: %d, Timeouts: %d\n",
			result.Matched, result.Mismatched, result.RecvTimeouts)
		if result.Received > 0 {
			fmt.Printf("        RSSI: avg=%d dBm, min=%d dBm, max=%d dBm\n",
				result.AvgRSSI, result.MinRSSI, result.MaxRSSI)
		}
		fmt.Println()

		// Stop if success rate drops below 50%
		if result.SuccessRate < 50.0 {
			fmt.Println("Success rate below 50%, stopping tests.")
			break
		}

		// Halve the delay for next run
		delay = delay / 2
	}

	// Print summary
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("SUMMARY")
	fmt.Println("========================================")
	fmt.Printf("%-15s %-8s %-8s %-10s %-10s\n", "Delay", "Sent", "Recv", "Success%", "Avg RSSI")
	fmt.Println("------------------------------------------------------------")
	for _, r := range results {
		fmt.Printf("%-15v %-8d %-8d %-10.1f %-10d\n",
			r.Delay, r.Sent, r.Received, r.SuccessRate, r.AvgRSSI)
	}
}

func runTest(sender, receiver *yardstick.Device, count int, delay time.Duration, verbose bool) TestResult {
	result := TestResult{
		Delay:   delay,
		Sent:    count,
		MinRSSI: 0,
		MaxRSSI: -200,
	}

	// Prepare test packets
	pktLen := 16 // Fixed packet length from config
	packets := make([][]byte, count)
	for i := 0; i < count; i++ {
		// Create packet with sequence number and recognizable pattern
		pkt := make([]byte, pktLen)
		pkt[0] = 0xAA                     // Header marker
		pkt[1] = byte(i)                  // Sequence number
		pkt[2] = byte(count)              // Total count
		pkt[3] = 0x55                     // Second marker
		copy(pkt[4:], []byte("TEST1234")) // Payload
		packets[i] = pkt
	}

	// Channel to collect received packets
	type recvPacket struct {
		data      []byte
		rssi      int
		timestamp time.Time
	}
	recvChan := make(chan recvPacket, count*2)
	var recvWg sync.WaitGroup
	var stopRecv atomic.Bool

	// Start receiver goroutine
	recvWg.Add(1)
	go func() {
		defer recvWg.Done()

		// Enter RX mode
		if err := receiver.SetModeRX(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Receiver failed to enter RX mode: %v\n", err)
			return
		}

		recvTimeout := 200 * time.Millisecond
		for !stopRecv.Load() {
			data, err := receiver.RFRecv(recvTimeout, 0)
			if err != nil {
				result.RecvTimeouts++
				continue
			}

			// Get RSSI
			rssi := -150
			if status, err := receiver.GetRadioStatus(); err == nil {
				rssi = status.RSSIdBm
			}

			recvChan <- recvPacket{
				data:      data,
				rssi:      rssi,
				timestamp: time.Now(),
			}
		}
	}()

	// Give receiver time to start
	time.Sleep(100 * time.Millisecond)

	// Send packets
	sendTimes := make([]time.Time, count)
	for i := 0; i < count; i++ {
		if verbose {
			fmt.Printf("  TX[%02d]: %s\n", i, hex.EncodeToString(packets[i]))
		}

		sendTimes[i] = time.Now()
		err := sender.RFXmit(packets[i], 0, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  TX[%02d] ERROR: %v\n", i, err)
		}

		if i < count-1 {
			time.Sleep(delay)
		}
	}

	// Wait for remaining packets to arrive
	time.Sleep(500 * time.Millisecond)

	// Stop receiver
	stopRecv.Store(true)
	recvWg.Wait()
	close(recvChan)

	// Analyze received packets
	received := make([]recvPacket, 0)
	for pkt := range recvChan {
		received = append(received, pkt)
	}

	result.Received = len(received)

	// Match received packets to sent packets
	matched := make(map[int]bool)
	var totalRSSI int

	for _, rpkt := range received {
		if verbose {
			fmt.Printf("  RX: %s (RSSI: %d dBm)\n", hex.EncodeToString(rpkt.data), rpkt.rssi)
		}

		// Check if this is a valid test packet
		if len(rpkt.data) >= 4 && rpkt.data[0] == 0xAA && rpkt.data[3] == 0x55 {
			seqNum := int(rpkt.data[1])
			if seqNum < count && !matched[seqNum] {
				// Verify payload matches
				if len(rpkt.data) >= pktLen {
					expectedPkt := packets[seqNum]
					match := true
					for j := 0; j < pktLen; j++ {
						if rpkt.data[j] != expectedPkt[j] {
							match = false
							break
						}
					}
					if match {
						matched[seqNum] = true
						result.Matched++
					} else {
						result.Mismatched++
						if verbose {
							fmt.Printf("       ^ Mismatch at seq %d\n", seqNum)
						}
					}
				}
			}
		} else {
			// Not a test packet (noise)
			if verbose {
				fmt.Printf("       ^ Not a test packet (noise)\n")
			}
		}

		totalRSSI += rpkt.rssi
		if rpkt.rssi < result.MinRSSI {
			result.MinRSSI = rpkt.rssi
		}
		if rpkt.rssi > result.MaxRSSI {
			result.MaxRSSI = rpkt.rssi
		}
	}

	if result.Received > 0 {
		result.AvgRSSI = totalRSSI / result.Received
	}

	result.SuccessRate = float64(result.Matched) / float64(result.Sent) * 100.0

	// Report missing packets
	if verbose {
		missing := []int{}
		for i := 0; i < count; i++ {
			if !matched[i] {
				missing = append(missing, i)
			}
		}
		if len(missing) > 0 {
			fmt.Printf("  Missing packets: %v\n", missing)
		}
	}

	return result
}

func verifySenderConfig(device *yardstick.Device) {
	sync1, _ := device.PeekByte(0xDF00)
	sync0, _ := device.PeekByte(0xDF01)
	pktlen, _ := device.PeekByte(0xDF02)
	mdmcfg2, _ := device.PeekByte(0xDF0E)
	freq2, _ := device.PeekByte(0xDF09)
	freq1, _ := device.PeekByte(0xDF0A)
	freq0, _ := device.PeekByte(0xDF0B)
	pa0, _ := device.PeekByte(0xDF2E)
	fmt.Printf("Sender verified: SYNC=0x%02X%02X PKTLEN=%d MDMCFG2=0x%02X FREQ=0x%02X%02X%02X PA0=0x%02X\n",
		sync1, sync0, pktlen, mdmcfg2, freq2, freq1, freq0, pa0)
}

func verifyReceiverConfig(device *yardstick.Device) {
	sync1, _ := device.PeekByte(0xDF00)
	sync0, _ := device.PeekByte(0xDF01)
	pktlen, _ := device.PeekByte(0xDF02)
	mdmcfg2, _ := device.PeekByte(0xDF0E)
	freq2, _ := device.PeekByte(0xDF09)
	freq1, _ := device.PeekByte(0xDF0A)
	freq0, _ := device.PeekByte(0xDF0B)
	pa0, _ := device.PeekByte(0xDF2E)
	fmt.Printf("Receiver verified: SYNC=0x%02X%02X PKTLEN=%d MDMCFG2=0x%02X FREQ=0x%02X%02X%02X PA0=0x%02X\n",
		sync1, sync0, pktlen, mdmcfg2, freq2, freq1, freq0, pa0)
}
