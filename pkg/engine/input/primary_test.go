package input

import "testing"

func TestNoteDeviceActivity(t *testing.T) {
	primaryMu.Lock()
	primaryDevice = PrimaryKeyboard
	primaryMu.Unlock()

	if NoteDeviceActivity(DeviceKeyboard) {
		t.Fatal("keyboard when already keyboard should not change")
	}
	if !NoteDeviceActivity(DeviceGamepad) {
		t.Fatal("gamepad should switch primary")
	}
	if GetPrimaryDevice() != PrimaryGamepad {
		t.Fatalf("primary = %v, want gamepad", GetPrimaryDevice())
	}
	if PrimaryDeviceSwitchMessage() != "Controller" {
		t.Fatalf("message = %q", PrimaryDeviceSwitchMessage())
	}
	if !NoteDeviceActivity(DeviceKeyboard) {
		t.Fatal("keyboard should switch primary back")
	}
	if PrimaryDeviceSwitchMessage() != "Keyboard" {
		t.Fatalf("message = %q", PrimaryDeviceSwitchMessage())
	}
}

func TestHintMoveSwitchesWithPrimary(t *testing.T) {
	primaryMu.Lock()
	primaryDevice = PrimaryGamepad
	primaryMu.Unlock()
	if HintMove() != "Use left stick or D-pad to move" {
		t.Fatalf("gamepad move hint: %q", HintMove())
	}
	primaryMu.Lock()
	primaryDevice = PrimaryKeyboard
	primaryMu.Unlock()
	if HintMove() != "Press WASD or arrow keys to move" {
		t.Fatalf("keyboard move hint: %q", HintMove())
	}
}
