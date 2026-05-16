package ebiten

import (
	"image/color"
	"testing"
)

func TestFocusPlateForForeground_maintenanceOrange_isWarmDarkPlate(t *testing.T) {
	fg := color.RGBA{R: 255, G: 165, B: 0, A: 255}
	plate := focusPlateForForeground(fg)
	r32, g32, b32, a32 := plate.RGBA()
	r := uint8(r32 >> 8)
	g := uint8(g32 >> 8)
	b := uint8(b32 >> 8)
	if a32>>8 != 220 {
		t.Fatalf("alpha = %d, want 220", a32>>8)
	}
	// Warm dark amber (maintenance theme), not complementary blue
	if int(b) >= int(r) || int(b) >= int(g) {
		t.Fatalf("plate should be warm (b lowest): got (%d,%d,%d)", r, g, b)
	}
	if r < 85 || g < 55 || b > 25 {
		t.Fatalf("unexpected plate RGB (%d,%d,%d): want dark amber-brown focus for orange fg", r, g, b)
	}
}

func TestFocusPlateForForeground_doorYellow_isWarmGoldPlate(t *testing.T) {
	// Bright yellow locked-door glyph: plate stays in the warm family (spec: map-tile-focus-and-contrast).
	fg := color.RGBA{R: 255, G: 255, B: 0, A: 255}
	plate := focusPlateForForeground(fg)
	r32, g32, b32, a32 := plate.RGBA()
	r := uint8(r32 >> 8)
	g := uint8(g32 >> 8)
	b := uint8(b32 >> 8)
	if a32>>8 != 220 {
		t.Fatalf("alpha = %d, want 220", a32>>8)
	}
	if int(b) >= int(r) || int(b) >= int(g) {
		t.Fatalf("yellow lock plate should be warm (b lowest): got (%d,%d,%d)", r, g, b)
	}
}

func TestFocusPlateForForeground_hazardRed_keepsRedHue(t *testing.T) {
	plate := focusPlateForForeground(colorHazard)
	r32, g32, b32, _ := plate.RGBA()
	r := uint8(r32 >> 8)
	g := uint8(g32 >> 8)
	b := uint8(b32 >> 8)
	if int(r) < int(g)+10 || int(r) < int(b)+10 {
		t.Fatalf("hazard red fg should use red-family plate (r highest): got (%d,%d,%d)", r, g, b)
	}
}
