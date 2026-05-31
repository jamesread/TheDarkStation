package input

import "sync"

// PrimaryDevice is the input source used for on-screen control hints and intent priority.
type PrimaryDevice int

const (
	PrimaryKeyboard PrimaryDevice = iota
	PrimaryGamepad
)

var (
	primaryMu     sync.RWMutex
	primaryDevice = PrimaryKeyboard
)

// GetPrimaryDevice returns the active primary input device.
func GetPrimaryDevice() PrimaryDevice {
	primaryMu.RLock()
	defer primaryMu.RUnlock()
	return primaryDevice
}

// NoteDeviceActivity records use of a gameplay input device. Returns true when the primary device changed.
func NoteDeviceActivity(device Device) bool {
	if device != DeviceKeyboard && device != DeviceGamepad {
		return false
	}
	next := PrimaryKeyboard
	if device == DeviceGamepad {
		next = PrimaryGamepad
	}
	primaryMu.Lock()
	defer primaryMu.Unlock()
	if primaryDevice == next {
		return false
	}
	primaryDevice = next
	return true
}

// PrimaryDeviceSwitchMessage returns a short player-facing message after a primary device change.
func PrimaryDeviceSwitchMessage() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "Controller"
	}
	return "Keyboard"
}
