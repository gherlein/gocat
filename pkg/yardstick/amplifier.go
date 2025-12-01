package yardstick

// EnableAmplifier enables the external TX/RX amplifier (YardStick One)
func (d *Device) EnableAmplifier() error {
	return d.SetAmpMode(AmpModeOn)
}

// DisableAmplifier disables the external amplifier
func (d *Device) DisableAmplifier() error {
	return d.SetAmpMode(AmpModeOff)
}
