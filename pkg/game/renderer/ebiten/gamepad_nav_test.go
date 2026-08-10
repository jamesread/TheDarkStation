package ebiten

import (
	"testing"

	engineinput "darkstation/pkg/engine/input"
)

func TestStickNavDirection_EngageRequiresStrongDeflection(t *testing.T) {
	if got := stickNavDirection(0.4, 0, ""); got != "" {
		t.Fatalf("weak deflection = %q, want empty", got)
	}
	if got := stickNavDirection(-0.65, 0, ""); got != "left" {
		t.Fatalf("strong left = %q, want left", got)
	}
}

func TestStickNavDirection_HysteresisPreventsFlicker(t *testing.T) {
	// Latched left: noise below engage but above center should stay left.
	if got := stickNavDirection(-0.45, 0, "left"); got != "left" {
		t.Fatalf("latched left with -0.45 = %q, want left", got)
	}
	// Return to center releases.
	if got := stickNavDirection(-0.1, 0.1, "left"); got != "" {
		t.Fatalf("centered stick = %q, want empty", got)
	}
}

func TestStickNavDirection_DominantAxis(t *testing.T) {
	if got := stickNavDirection(-0.7, -0.5, ""); got != "left" {
		t.Fatalf("diagonal = %q, want left (dominant axis)", got)
	}
}

func TestIsMovementIntent(t *testing.T) {
	if !isMovementIntent(engineinputIntent(engineinput.ActionMoveNorth)) {
		t.Fatal("north should be movement")
	}
	if isMovementIntent(engineinputIntent(engineinput.ActionInteract)) {
		t.Fatal("interact should not be movement")
	}
}

func TestIsPointerDeviceGamepadName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"Elan Touchpad", true},
		{"Apple Inc. Magic Trackpad", true},
		{"TPPS/2 Elan TrackPoint", true},
		{"Logitech USB Optical Mouse", true},
		{"HP, Inc HyperX QuadCast Consumer Control", true},
		{"Metadot - Das Keyboard Das Keyboard System Control", true},
		{"C-Media Electronics Inc. USB PnP Sound Device", true},
		{"Elgato Stream Deck MK.2", true},
		{"Xbox Wireless Controller", false},
		{"Wireless Controller", false}, // DualShock / DualSense style
		{"Nintendo Switch Pro Controller", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isPointerDeviceGamepadName(tc.name); got != tc.want {
			t.Errorf("isPointerDeviceGamepadName(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestIsUsableGamepadCandidate_RejectsMicVolumeKnob(t *testing.T) {
	// Reproduces HyperX QuadCast Consumer Control as seen by GLFW/Ebiten on Linux:
	// 1 axis resting at -1.0, 1 button, no standard layout → continuous MoveWest.
	if isUsableGamepadCandidate("HP, Inc HyperX QuadCast Consumer Control", false, 1, 1) {
		t.Fatal("mic consumer-control knob must not be treated as a gamepad")
	}
	if isUsableGamepadCandidate("Generic HID", false, 1, 1) {
		t.Fatal("1-axis unmapped HID must not be treated as a gamepad")
	}
}

func TestIsUsableGamepadCandidate_AcceptsRealPads(t *testing.T) {
	if !isUsableGamepadCandidate("Xbox Wireless Controller", true, 6, 14) {
		t.Fatal("standard-mapped pad should be accepted")
	}
	if !isUsableGamepadCandidate("Unknown Fight Stick", false, 2, 8) {
		t.Fatal("unmapped stick with 2 axes and enough buttons should be accepted")
	}
	if isUsableGamepadCandidate("Elan Touchpad", true, 2, 8) {
		t.Fatal("pointer device must be rejected even with a standard layout mapping")
	}
}

func engineinputIntent(action engineinput.Action) engineinput.Intent {
	return engineinput.Intent{Action: action}
}
