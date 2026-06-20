package ebiten

import (
	"image/color"
	"testing"
	"time"
)

func TestScaleColor_BrightensAndClamps(t *testing.T) {
	base := color.RGBA{100, 200, 250, 255}
	got := scaleColor(base, 1.2).(color.RGBA)
	if got.R != 120 || got.G != 240 || got.B != 255 {
		t.Errorf("scaleColor 1.2 = %v, want {120 240 255}", got)
	}
	if got.A != 255 {
		t.Errorf("alpha must be preserved, got %d", got.A)
	}
	if dim := scaleColor(base, 0.5).(color.RGBA); dim.R != 50 {
		t.Errorf("scaleColor 0.5 R = %d, want 50", dim.R)
	}
	if scaleColor(nil, 1.5) != nil {
		t.Error("nil color (no plate) must stay nil")
	}
}

func TestBlendColors_Endpoints(t *testing.T) {
	a := color.RGBA{0, 0, 0, 255}
	b := color.RGBA{200, 100, 50, 255}
	if got := blendColors(a, b, 0).(color.RGBA); got.R != 0 || got.G != 0 || got.B != 0 {
		t.Errorf("t=0 should return first color, got %v", got)
	}
	if got := blendColors(a, b, 1).(color.RGBA); got.R != 200 || got.G != 100 || got.B != 50 {
		t.Errorf("t=1 should return second color, got %v", got)
	}
}

func TestDevicePulseColors_ExpiredReturnsInputsUnchanged(t *testing.T) {
	bg := color.RGBA{10, 20, 30, 255}
	fg := color.RGBA{200, 200, 200, 255}
	opts := &CellRenderOptions{HasBackground: true}
	gotBg, gotFg := devicePulseColors(bg, fg, opts, devicePulseDurationMs+1)
	if gotBg != color.Color(bg) || gotFg != color.Color(fg) {
		t.Error("expired pulse must not modify colors")
	}
}

func TestDevicePulseColors_ActivePulseBlendsTowardGlow(t *testing.T) {
	bg := color.RGBA{10, 20, 30, 255}
	fg := color.RGBA{200, 200, 200, 255}
	opts := &CellRenderOptions{HasBackground: true}
	// 125ms into the 500ms wave = sine peak: maximum glow for this envelope.
	gotBg, gotFg := devicePulseColors(bg, fg, opts, 125)
	if gotFg != color.Color(fg) {
		t.Error("pulse must not change the glyph color")
	}
	got := gotBg.(color.RGBA)
	if got.G <= bg.G {
		t.Errorf("active pulse should raise the green channel toward the glow, got %v", got)
	}
}

func TestSnapshotDevicePulses_PrunesExpired(t *testing.T) {
	e := &EbitenRenderer{}
	now := time.Now().UnixMilli()
	e.AddDevicePulse(3, 4)
	e.devicePulses[cellCoordKey(9, 9)] = now - devicePulseDurationMs - 1000

	e.snapshotDevicePulses(now)

	if len(e.snapshot.devicePulses) != 1 {
		t.Fatalf("want 1 active pulse in snapshot, got %d", len(e.snapshot.devicePulses))
	}
	p := e.snapshot.devicePulses[0]
	if p.row != 3 || p.col != 4 {
		t.Errorf("snapshot pulse at (%d,%d), want (3,4)", p.row, p.col)
	}
	if len(e.devicePulses) != 1 {
		t.Errorf("expired pulse should be removed from the registry, have %d entries", len(e.devicePulses))
	}
}
