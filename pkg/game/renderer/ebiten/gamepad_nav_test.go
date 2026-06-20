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

func engineinputIntent(action engineinput.Action) engineinput.Intent {
	return engineinput.Intent{Action: action}
}
