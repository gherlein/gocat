// test-configs: Load configuration to YardStick One and verify it was applied correctly
//
// This tool loads a configuration file to a YardStick One device, then reads
// back all registers and compares them to ensure the configuration was applied
// exactly as specified.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/google/gousb"
	"github.com/herlein/gocat/pkg/config"
	"github.com/herlein/gocat/pkg/registers"
	"github.com/herlein/gocat/pkg/yardstick"
)

// RegisterComparison holds the comparison result for a single register
type RegisterComparison struct {
	Name     string
	Address  uint16
	Expected uint8
	Actual   uint8
	Match    bool
}

func main() {
	// Parse command line flags
	configPath := flag.String("c", "etc/defaults.json", "Configuration file path")
	deviceSel := flag.String("d", "", yardstick.DeviceFlagUsage())
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	// Load configuration from file
	fmt.Printf("Loading configuration from: %s\n", *configPath)

	configuration, err := config.LoadFromFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		printConfigSummary(configuration)
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

	fmt.Printf("Connected to: %s\n", device)

	// Test connectivity with ping
	fmt.Print("Testing connectivity... ")
	if err := device.Ping([]byte("TEST")); err != nil {
		fmt.Fprintf(os.Stderr, "FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("OK")

	// Apply configuration
	fmt.Println("Applying configuration...")
	if err := config.ApplyToDevice(device, configuration); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to apply configuration: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Configuration applied.")

	// Read back configuration
	fmt.Println("Reading back configuration for verification...")
	readBack, err := config.DumpFromDevice(device)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to read back configuration: %v\n", err)
		os.Exit(1)
	}

	// Compare configurations
	fmt.Println("\nVerification Results:")
	fmt.Println("=====================")

	comparisons := compareRegisters(&configuration.Registers, &readBack.Registers)

	// Count matches and mismatches
	matches := 0
	mismatches := 0
	skipped := 0

	for _, cmp := range comparisons {
		if cmp.Match {
			matches++
			if *verbose {
				fmt.Printf("  [OK]   %-12s (0x%04X): expected %3d (0x%02X), got %3d (0x%02X)\n",
					cmp.Name, cmp.Address, cmp.Expected, cmp.Expected, cmp.Actual, cmp.Actual)
			}
		} else if isReadOnlyRegister(cmp.Name) {
			skipped++
			if *verbose {
				fmt.Printf("  [SKIP] %-12s (0x%04X): read-only, expected %3d (0x%02X), got %3d (0x%02X)\n",
					cmp.Name, cmp.Address, cmp.Expected, cmp.Expected, cmp.Actual, cmp.Actual)
			}
		} else {
			mismatches++
			fmt.Printf("  [FAIL] %-12s (0x%04X): expected %3d (0x%02X), got %3d (0x%02X)\n",
				cmp.Name, cmp.Address, cmp.Expected, cmp.Expected, cmp.Actual, cmp.Actual)
		}
	}

	// Print summary
	fmt.Println()
	fmt.Printf("Summary: %d matched, %d mismatched, %d skipped (read-only)\n", matches, mismatches, skipped)

	if mismatches > 0 {
		fmt.Println("\nVERIFICATION FAILED")
		os.Exit(1)
	}

	fmt.Println("\nVERIFICATION PASSED - All writable registers match!")
}

func compareRegisters(expected, actual *registers.RegisterMap) []RegisterComparison {
	comparisons := []RegisterComparison{
		// Sync word
		{"SYNC1", registers.RegSYNC1, expected.SYNC1, actual.SYNC1, expected.SYNC1 == actual.SYNC1},
		{"SYNC0", registers.RegSYNC0, expected.SYNC0, actual.SYNC0, expected.SYNC0 == actual.SYNC0},

		// Packet control
		{"PKTLEN", registers.RegPKTLEN, expected.PKTLEN, actual.PKTLEN, expected.PKTLEN == actual.PKTLEN},
		{"PKTCTRL1", registers.RegPKTCTRL1, expected.PKTCTRL1, actual.PKTCTRL1, expected.PKTCTRL1 == actual.PKTCTRL1},
		{"PKTCTRL0", registers.RegPKTCTRL0, expected.PKTCTRL0, actual.PKTCTRL0, expected.PKTCTRL0 == actual.PKTCTRL0},
		{"ADDR", registers.RegADDR, expected.ADDR, actual.ADDR, expected.ADDR == actual.ADDR},
		{"CHANNR", registers.RegCHANNR, expected.CHANNR, actual.CHANNR, expected.CHANNR == actual.CHANNR},

		// Frequency synthesizer
		{"FSCTRL1", registers.RegFSCTRL1, expected.FSCTRL1, actual.FSCTRL1, expected.FSCTRL1 == actual.FSCTRL1},
		{"FSCTRL0", registers.RegFSCTRL0, expected.FSCTRL0, actual.FSCTRL0, expected.FSCTRL0 == actual.FSCTRL0},

		// Frequency control
		{"FREQ2", registers.RegFREQ2, expected.FREQ2, actual.FREQ2, expected.FREQ2 == actual.FREQ2},
		{"FREQ1", registers.RegFREQ1, expected.FREQ1, actual.FREQ1, expected.FREQ1 == actual.FREQ1},
		{"FREQ0", registers.RegFREQ0, expected.FREQ0, actual.FREQ0, expected.FREQ0 == actual.FREQ0},

		// Modem configuration
		{"MDMCFG4", registers.RegMDMCFG4, expected.MDMCFG4, actual.MDMCFG4, expected.MDMCFG4 == actual.MDMCFG4},
		{"MDMCFG3", registers.RegMDMCFG3, expected.MDMCFG3, actual.MDMCFG3, expected.MDMCFG3 == actual.MDMCFG3},
		{"MDMCFG2", registers.RegMDMCFG2, expected.MDMCFG2, actual.MDMCFG2, expected.MDMCFG2 == actual.MDMCFG2},
		{"MDMCFG1", registers.RegMDMCFG1, expected.MDMCFG1, actual.MDMCFG1, expected.MDMCFG1 == actual.MDMCFG1},
		{"MDMCFG0", registers.RegMDMCFG0, expected.MDMCFG0, actual.MDMCFG0, expected.MDMCFG0 == actual.MDMCFG0},
		{"DEVIATN", registers.RegDEVIATN, expected.DEVIATN, actual.DEVIATN, expected.DEVIATN == actual.DEVIATN},

		// Main radio control state machine
		{"MCSM2", registers.RegMCSM2, expected.MCSM2, actual.MCSM2, expected.MCSM2 == actual.MCSM2},
		{"MCSM1", registers.RegMCSM1, expected.MCSM1, actual.MCSM1, expected.MCSM1 == actual.MCSM1},
		{"MCSM0", registers.RegMCSM0, expected.MCSM0, actual.MCSM0, expected.MCSM0 == actual.MCSM0},

		// Frequency offset compensation
		{"FOCCFG", registers.RegFOCCFG, expected.FOCCFG, actual.FOCCFG, expected.FOCCFG == actual.FOCCFG},
		{"BSCFG", registers.RegBSCFG, expected.BSCFG, actual.BSCFG, expected.BSCFG == actual.BSCFG},

		// AGC control
		{"AGCCTRL2", registers.RegAGCCTRL2, expected.AGCCTRL2, actual.AGCCTRL2, expected.AGCCTRL2 == actual.AGCCTRL2},
		{"AGCCTRL1", registers.RegAGCCTRL1, expected.AGCCTRL1, actual.AGCCTRL1, expected.AGCCTRL1 == actual.AGCCTRL1},
		{"AGCCTRL0", registers.RegAGCCTRL0, expected.AGCCTRL0, actual.AGCCTRL0, expected.AGCCTRL0 == actual.AGCCTRL0},

		// Front end configuration
		{"FREND1", registers.RegFREND1, expected.FREND1, actual.FREND1, expected.FREND1 == actual.FREND1},
		{"FREND0", registers.RegFREND0, expected.FREND0, actual.FREND0, expected.FREND0 == actual.FREND0},

		// Frequency synthesizer calibration
		{"FSCAL3", registers.RegFSCAL3, expected.FSCAL3, actual.FSCAL3, expected.FSCAL3 == actual.FSCAL3},
		{"FSCAL2", registers.RegFSCAL2, expected.FSCAL2, actual.FSCAL2, expected.FSCAL2 == actual.FSCAL2},
		{"FSCAL1", registers.RegFSCAL1, expected.FSCAL1, actual.FSCAL1, expected.FSCAL1 == actual.FSCAL1},
		{"FSCAL0", registers.RegFSCAL0, expected.FSCAL0, actual.FSCAL0, expected.FSCAL0 == actual.FSCAL0},

		// Test registers
		{"TEST2", registers.RegTEST2, expected.TEST2, actual.TEST2, expected.TEST2 == actual.TEST2},
		{"TEST1", registers.RegTEST1, expected.TEST1, actual.TEST1, expected.TEST1 == actual.TEST1},
		{"TEST0", registers.RegTEST0, expected.TEST0, actual.TEST0, expected.TEST0 == actual.TEST0},

		// GPIO configuration
		{"IOCFG2", registers.RegIOCFG2, expected.IOCFG2, actual.IOCFG2, expected.IOCFG2 == actual.IOCFG2},
		{"IOCFG1", registers.RegIOCFG1, expected.IOCFG1, actual.IOCFG1, expected.IOCFG1 == actual.IOCFG1},
		{"IOCFG0", registers.RegIOCFG0, expected.IOCFG0, actual.IOCFG0, expected.IOCFG0 == actual.IOCFG0},

		// Read-only status registers (will be skipped in verification)
		{"PARTNUM", registers.RegPARTNUM, expected.PARTNUM, actual.PARTNUM, expected.PARTNUM == actual.PARTNUM},
		{"CHIPID", registers.RegCHIPID, expected.CHIPID, actual.CHIPID, expected.CHIPID == actual.CHIPID},
		{"FREQEST", registers.RegFREQEST, expected.FREQEST, actual.FREQEST, expected.FREQEST == actual.FREQEST},
		{"LQI", registers.RegLQI, expected.LQI, actual.LQI, expected.LQI == actual.LQI},
		{"RSSI", registers.RegRSSI, expected.RSSI, actual.RSSI, expected.RSSI == actual.RSSI},
		{"MARCSTATE", registers.RegMARCSTATE, expected.MARCSTATE, actual.MARCSTATE, expected.MARCSTATE == actual.MARCSTATE},
		{"PKTSTATUS", registers.RegPKTSTATUS, expected.PKTSTATUS, actual.PKTSTATUS, expected.PKTSTATUS == actual.PKTSTATUS},
		{"VCO_VC_DAC", registers.RegVCO_VC_DAC, expected.VCO_VC_DAC, actual.VCO_VC_DAC, expected.VCO_VC_DAC == actual.VCO_VC_DAC},
	}

	// Add PA_TABLE comparisons
	for i := 0; i < 8; i++ {
		addr := uint16(registers.RegPA_TABLE7 + i)
		name := fmt.Sprintf("PA_TABLE%d", 7-i)
		comparisons = append(comparisons, RegisterComparison{
			Name:     name,
			Address:  addr,
			Expected: expected.PA_TABLE[i],
			Actual:   actual.PA_TABLE[i],
			Match:    expected.PA_TABLE[i] == actual.PA_TABLE[i],
		})
	}

	return comparisons
}

// isReadOnlyRegister returns true for registers that cannot be written
func isReadOnlyRegister(name string) bool {
	readOnlyRegs := map[string]bool{
		"PARTNUM":    true,
		"CHIPID":     true,
		"FREQEST":    true,
		"LQI":        true,
		"RSSI":       true,
		"MARCSTATE":  true,
		"PKTSTATUS":  true,
		"VCO_VC_DAC": true,
	}
	return readOnlyRegs[name]
}

func printConfigSummary(cfg *config.DeviceConfig) {
	fmt.Println("\nConfiguration Summary:")
	fmt.Printf("  Serial:       %s\n", cfg.Serial)
	fmt.Printf("  Manufacturer: %s\n", cfg.Manufacturer)
	fmt.Printf("  Product:      %s\n", cfg.Product)
	fmt.Printf("  Build Type:   %s\n", cfg.BuildType)
	fmt.Printf("  Part Number:  0x%02X\n", cfg.PartNum)
	fmt.Printf("  Frequency:    %.6f MHz\n", cfg.GetFrequencyMHz())
	fmt.Printf("  Sync Word:    0x%04X\n", cfg.GetSyncWord())
	fmt.Printf("  Modulation:   %s\n", cfg.GetModulationString())
	fmt.Printf("  Packet Len:   %d\n", cfg.Registers.PKTLEN)
	fmt.Println()
}

