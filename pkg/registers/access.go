package registers

import (
	"fmt"

	"github.com/herlein/gocat/pkg/yardstick"
)

// Peek reads a single byte from device memory
func Peek(device *yardstick.Device, address uint16) (uint8, error) {
	return device.PeekByte(address)
}

// PeekMultiple reads multiple bytes from device memory
func PeekMultiple(device *yardstick.Device, address uint16, length uint16) ([]byte, error) {
	return device.Peek(address, length)
}

// Poke writes a single byte to device memory
func Poke(device *yardstick.Device, address uint16, value uint8) error {
	return device.PokeByte(address, value)
}

// PokeMultiple writes multiple bytes to device memory
func PokeMultiple(device *yardstick.Device, address uint16, data []byte) error {
	return device.Poke(address, data)
}

// Strobe sends a radio strobe command
func Strobe(device *yardstick.Device, command uint8) error {
	return device.PokeByte(RegRFST, command)
}

// GetRadioState reads the current radio state
func GetRadioState(device *yardstick.Device) (RadioState, error) {
	state, err := device.PeekByte(RegMARCSTATE)
	if err != nil {
		return 0, fmt.Errorf("failed to read radio state: %w", err)
	}
	return RadioState(state & 0x1F), nil // MARCSTATE is only 5 bits
}

// SetIDLE puts the radio in idle state
func SetIDLE(device *yardstick.Device) error {
	return Strobe(device, StrobeSIDLE)
}

// SetRX puts the radio in receive mode
func SetRX(device *yardstick.Device) error {
	return Strobe(device, StrobeSRX)
}

// SetTX puts the radio in transmit mode
func SetTX(device *yardstick.Device) error {
	return Strobe(device, StrobeSTX)
}

// ReadRadioConfig reads the radio configuration block efficiently
// This reads 62 bytes starting at 0xDF00
func ReadRadioConfig(device *yardstick.Device) ([]byte, error) {
	return device.Peek(0xDF00, 62)
}

// ReadAllRegisters reads all radio configuration registers into a RegisterMap
func ReadAllRegisters(device *yardstick.Device) (*RegisterMap, error) {
	// Read the main config block (0xDF00 - 0xDF3D)
	// We'll read in chunks to handle the gaps in the register map

	reg := &RegisterMap{}
	var err error

	// Read registers 0xDF00 - 0xDF1F (32 bytes, continuous)
	block1, err := device.Peek(0xDF00, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to read register block 1: %w", err)
	}

	reg.SYNC1 = block1[0]
	reg.SYNC0 = block1[1]
	reg.PKTLEN = block1[2]
	reg.PKTCTRL1 = block1[3]
	reg.PKTCTRL0 = block1[4]
	reg.ADDR = block1[5]
	reg.CHANNR = block1[6]
	reg.FSCTRL1 = block1[7]
	reg.FSCTRL0 = block1[8]
	reg.FREQ2 = block1[9]
	reg.FREQ1 = block1[10]
	reg.FREQ0 = block1[11]
	reg.MDMCFG4 = block1[12]
	reg.MDMCFG3 = block1[13]
	reg.MDMCFG2 = block1[14]
	reg.MDMCFG1 = block1[15]
	reg.MDMCFG0 = block1[16]
	reg.DEVIATN = block1[17]
	reg.MCSM2 = block1[18]
	reg.MCSM1 = block1[19]
	reg.MCSM0 = block1[20]
	reg.FOCCFG = block1[21]
	reg.BSCFG = block1[22]
	reg.AGCCTRL2 = block1[23]
	reg.AGCCTRL1 = block1[24]
	reg.AGCCTRL0 = block1[25]
	reg.FREND1 = block1[26]
	reg.FREND0 = block1[27]
	reg.FSCAL3 = block1[28]
	reg.FSCAL2 = block1[29]
	reg.FSCAL1 = block1[30]
	reg.FSCAL0 = block1[31]

	// Read TEST registers (0xDF23 - 0xDF25)
	block2, err := device.Peek(0xDF23, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to read TEST registers: %w", err)
	}
	reg.TEST2 = block2[0]
	reg.TEST1 = block2[1]
	reg.TEST0 = block2[2]

	// Read PA_TABLE and IOCFG (0xDF27 - 0xDF31)
	block3, err := device.Peek(0xDF27, 11)
	if err != nil {
		return nil, fmt.Errorf("failed to read PA_TABLE/IOCFG: %w", err)
	}
	// PA_TABLE7 through PA_TABLE0 (indices 7 down to 0 in struct)
	reg.PA_TABLE[7] = block3[0]
	reg.PA_TABLE[6] = block3[1]
	reg.PA_TABLE[5] = block3[2]
	reg.PA_TABLE[4] = block3[3]
	reg.PA_TABLE[3] = block3[4]
	reg.PA_TABLE[2] = block3[5]
	reg.PA_TABLE[1] = block3[6]
	reg.PA_TABLE[0] = block3[7]
	reg.IOCFG2 = block3[8]
	reg.IOCFG1 = block3[9]
	reg.IOCFG0 = block3[10]

	// Read status registers (0xDF36 - 0xDF3D)
	block4, err := device.Peek(0xDF36, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to read status registers: %w", err)
	}
	reg.PARTNUM = block4[0]
	reg.CHIPID = block4[1]
	reg.FREQEST = block4[2]
	reg.LQI = block4[3]
	reg.RSSI = block4[4]
	reg.MARCSTATE = block4[5]
	reg.PKTSTATUS = block4[6]
	reg.VCO_VC_DAC = block4[7]

	return reg, nil
}

// WriteAllRegisters writes all writable radio configuration registers from a RegisterMap
func WriteAllRegisters(device *yardstick.Device, reg *RegisterMap) error {
	// Write registers 0xDF00 - 0xDF1F (32 bytes)
	block1 := []byte{
		reg.SYNC1, reg.SYNC0,
		reg.PKTLEN, reg.PKTCTRL1, reg.PKTCTRL0, reg.ADDR, reg.CHANNR,
		reg.FSCTRL1, reg.FSCTRL0,
		reg.FREQ2, reg.FREQ1, reg.FREQ0,
		reg.MDMCFG4, reg.MDMCFG3, reg.MDMCFG2, reg.MDMCFG1, reg.MDMCFG0,
		reg.DEVIATN,
		reg.MCSM2, reg.MCSM1, reg.MCSM0,
		reg.FOCCFG, reg.BSCFG,
		reg.AGCCTRL2, reg.AGCCTRL1, reg.AGCCTRL0,
		reg.FREND1, reg.FREND0,
		reg.FSCAL3, reg.FSCAL2, reg.FSCAL1, reg.FSCAL0,
	}
	if err := device.Poke(0xDF00, block1); err != nil {
		return fmt.Errorf("failed to write register block 1: %w", err)
	}

	// Write TEST registers (0xDF23 - 0xDF25)
	block2 := []byte{reg.TEST2, reg.TEST1, reg.TEST0}
	if err := device.Poke(0xDF23, block2); err != nil {
		return fmt.Errorf("failed to write TEST registers: %w", err)
	}

	// Write PA_TABLE and IOCFG (0xDF27 - 0xDF31)
	block3 := []byte{
		reg.PA_TABLE[7], reg.PA_TABLE[6], reg.PA_TABLE[5], reg.PA_TABLE[4],
		reg.PA_TABLE[3], reg.PA_TABLE[2], reg.PA_TABLE[1], reg.PA_TABLE[0],
		reg.IOCFG2, reg.IOCFG1, reg.IOCFG0,
	}
	if err := device.Poke(0xDF27, block3); err != nil {
		return fmt.Errorf("failed to write PA_TABLE/IOCFG: %w", err)
	}

	// Note: Status registers (0xDF36 - 0xDF3D) are read-only

	return nil
}

// GetFrequency calculates the carrier frequency in Hz from the register values
// crystalMHz should be 24 for CC1110/CC1111, 26 for CC2510/CC2511
func GetFrequency(reg *RegisterMap, crystalMHz float64) float64 {
	freq := uint32(reg.FREQ2)<<16 | uint32(reg.FREQ1)<<8 | uint32(reg.FREQ0)
	return float64(freq) * (crystalMHz * 1e6 / 65536.0)
}

// SetFrequency calculates and sets the FREQ registers for a given frequency
// crystalMHz should be 24 for CC1110/CC1111, 26 for CC2510/CC2511
func SetFrequency(reg *RegisterMap, frequencyHz float64, crystalMHz float64) {
	freq := uint32(frequencyHz * 65536.0 / (crystalMHz * 1e6))
	reg.FREQ2 = uint8((freq >> 16) & 0xFF)
	reg.FREQ1 = uint8((freq >> 8) & 0xFF)
	reg.FREQ0 = uint8(freq & 0xFF)
}

// GetSyncWord returns the 16-bit sync word from the register map
func GetSyncWord(reg *RegisterMap) uint16 {
	return uint16(reg.SYNC1)<<8 | uint16(reg.SYNC0)
}

// SetSyncWord sets the 16-bit sync word in the register map
func SetSyncWord(reg *RegisterMap, syncWord uint16) {
	reg.SYNC1 = uint8((syncWord >> 8) & 0xFF)
	reg.SYNC0 = uint8(syncWord & 0xFF)
}

// GetModulation returns the modulation format from MDMCFG2
func GetModulation(reg *RegisterMap) uint8 {
	return reg.MDMCFG2 & 0x70
}

// SetModulation sets the modulation format in MDMCFG2
func SetModulation(reg *RegisterMap, mod uint8) {
	reg.MDMCFG2 = (reg.MDMCFG2 & 0x8F) | (mod & 0x70)
}

// GetSyncMode returns the sync mode from MDMCFG2
func GetSyncMode(reg *RegisterMap) uint8 {
	return reg.MDMCFG2 & 0x07
}

// SetSyncMode sets the sync mode in MDMCFG2
func SetSyncMode(reg *RegisterMap, mode uint8) {
	reg.MDMCFG2 = (reg.MDMCFG2 & 0xF8) | (mode & 0x07)
}
