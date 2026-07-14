// Ambient map feedback: live conduits shimmer faintly, headlamp-only light
// flickers like a battery lamp, and devices the player just changed pulse for a
// moment ("the station noticed"). All effects are presentation-only — they read
// existing snapshot state and never touch game state.
package ebiten

import (
	"image/color"
	"math"
	"time"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

const (
	// devicePulseDurationMs is how long a "station noticed" pulse glows.
	devicePulseDurationMs = 2500

	// conduitShimmerPeriodMs is the period of the brightness wave that travels
	// along live (powered) cells; the per-cell phase offset makes it read as
	// flow along the run rather than a screen-wide pulse.
	conduitShimmerPeriodMs = 1800.0
	conduitShimmerAmp      = 0.07

	// Headlamp flicker: two incommensurate sines give an organic battery-lamp
	// wobble on cells lit only by the player's own lamp.
	headlampFlickerAmp1     = 0.05
	headlampFlickerAmp2     = 0.03
	headlampFlickerPeriod1  = 1100.0
	headlampFlickerPeriod2  = 370.0
	headlampFlickerCellSeed = 0.9
)

// devicePulseSnapshot is one active "station noticed" highlight copied for Draw.
type devicePulseSnapshot struct {
	row     int
	col     int
	startMs int64
}

// AddDevicePulse implements renderer.DevicePulseRenderer: marks a cell the
// player just changed so the map briefly acknowledges the action.
func (e *EbitenRenderer) AddDevicePulse(row, col int) {
	e.devicePulseMutex.Lock()
	defer e.devicePulseMutex.Unlock()
	if e.devicePulses == nil {
		e.devicePulses = make(map[uint64]int64)
	}
	e.devicePulses[cellCoordKey(row, col)] = time.Now().UnixMilli()
}

// snapshotDevicePulses prunes expired pulses and copies the rest for Draw.
func (e *EbitenRenderer) snapshotDevicePulses(nowMs int64) {
	e.devicePulseMutex.Lock()
	defer e.devicePulseMutex.Unlock()
	e.snapshot.devicePulses = e.snapshot.devicePulses[:0]
	for key, startMs := range e.devicePulses {
		if nowMs-startMs > devicePulseDurationMs {
			delete(e.devicePulses, key)
			continue
		}
		e.snapshot.devicePulses = append(e.snapshot.devicePulses, devicePulseSnapshot{
			row:     int(int32(key >> 32)),
			col:     int(int32(uint32(key))),
			startMs: startMs,
		})
	}
}

// ambientTileColors applies idle-world modulation to a tile's plate and glyph:
// device pulse > headlamp flicker > conduit shimmer. Returns the colors to draw.
func (e *EbitenRenderer) ambientTileColors(g *state.Game, cell *world.Cell, snap *renderSnapshot,
	opts *CellRenderOptions, customBg color.Color) (bg, fg color.Color) {
	bg, fg = customBg, opts.Color
	if cell == nil || !cell.Room || snap == nil {
		return bg, fg
	}
	nowMs := time.Now().UnixMilli()

	if startMs, ok := snapDevicePulseAt(snap, cell.Row, cell.Col); ok {
		return devicePulseColors(bg, fg, opts, nowMs-startMs)
	}

	if cellKnowledgeTier(g, cell) != knowledgeLive {
		return bg, fg
	}
	if snapCellHasLivePower(snap, cell) {
		// Powered conduit: a faint wave travels along the run.
		phase := 2*math.Pi*float64(nowMs)/conduitShimmerPeriodMs - 0.6*float64(cell.Row+cell.Col)
		factor := 1 + conduitShimmerAmp*math.Sin(phase)
		return scaleColor(bg, factor), fg
	}
	if gameworld.GetGameData(cell).LightsOn {
		// Lit only by the player's own lamp: battery flicker on plate and glyph.
		cellPhase := headlampFlickerCellSeed * float64((cell.Row*31+cell.Col*17)%64)
		factor := 1 +
			headlampFlickerAmp1*math.Sin(2*math.Pi*float64(nowMs)/headlampFlickerPeriod1+cellPhase) +
			headlampFlickerAmp2*math.Sin(2*math.Pi*float64(nowMs)/headlampFlickerPeriod2+1.7*cellPhase)
		return scaleColor(bg, factor), scaleColor(fg, factor)
	}
	return bg, fg
}

func snapDevicePulseAt(snap *renderSnapshot, row, col int) (int64, bool) {
	for _, p := range snap.devicePulses {
		if p.row == row && p.col == col {
			return p.startMs, true
		}
	}
	return 0, false
}

// devicePulseColors blends a decaying green-cyan glow over the tile plate.
func devicePulseColors(bg, fg color.Color, opts *CellRenderOptions, elapsedMs int64) (color.Color, color.Color) {
	progress := float64(elapsedMs) / float64(devicePulseDurationMs)
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		return bg, fg
	}
	envelope := 1 - progress
	wave := 0.5 + 0.5*math.Sin(2*math.Pi*float64(elapsedMs)/500.0)
	strength := 0.45 * envelope * wave

	base := bg
	if base == nil {
		if opts != nil && opts.HasBackground {
			base = colorWallBg
		} else {
			base = colorBackground
		}
	}
	glow := color.RGBA{80, 200, 140, 255}
	return blendColors(base, glow, strength), fg
}

// scaleColor multiplies a color's RGB channels by factor, clamping to [0,255].
// A nil color stays nil (no plate).
func scaleColor(c color.Color, factor float64) color.Color {
	if c == nil {
		return nil
	}
	r, g, b, a := c.RGBA()
	return color.RGBA{
		clampChannel(float64(r>>8) * factor),
		clampChannel(float64(g>>8) * factor),
		clampChannel(float64(b>>8) * factor),
		uint8(a >> 8),
	}
}

// blendColors linearly interpolates from a to b by t (0..1).
func blendColors(a, b color.Color, t float64) color.Color {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	return color.RGBA{
		clampChannel(float64(ar>>8) + (float64(br>>8)-float64(ar>>8))*t),
		clampChannel(float64(ag>>8) + (float64(bg>>8)-float64(ag>>8))*t),
		clampChannel(float64(ab>>8) + (float64(bb>>8)-float64(ab>>8))*t),
		uint8(aa >> 8),
	}
}

func clampChannel(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
