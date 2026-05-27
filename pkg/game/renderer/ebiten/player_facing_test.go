package ebiten

import (
	"math"
	"testing"

	"darkstation/pkg/game/state"
)

func TestPlayerFacingAngle(t *testing.T) {
	tests := []struct {
		facing state.PlayerFacing
		want   float64
	}{
		{state.FaceNorth, 0},
		{state.FaceEast, math.Pi / 2},
		{state.FaceSouth, math.Pi},
		{state.FaceWest, -math.Pi / 2},
	}
	for _, tt := range tests {
		if got := playerFacingAngle(tt.facing); got != tt.want {
			t.Errorf("playerFacingAngle(%v) = %v, want %v", tt.facing, got, tt.want)
		}
	}
}

func TestLerpAngleShortest(t *testing.T) {
	const eps = 1e-9
	got := lerpAngleShortest(-math.Pi/2, 0, 0.5)
	want := -math.Pi / 4
	if math.Abs(got-want) > eps {
		t.Fatalf("west→north halfway = %v, want %v", got, want)
	}

	got = lerpAngleShortest(0, math.Pi, 0.5)
	if math.Abs(math.Abs(got)-math.Pi/2) > eps {
		t.Fatalf("north→south halfway = %v, want ±π/2", got)
	}
}

func TestPlayerFacingRotation_drawAngle_initializesToSnap(t *testing.T) {
	var rot playerFacingRotation
	if angle := rot.drawAngle(state.FaceEast); angle != math.Pi/2 {
		t.Fatalf("initial angle = %v, want π/2", angle)
	}
}
