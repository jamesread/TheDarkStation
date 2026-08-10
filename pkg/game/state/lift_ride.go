package state

import "math"

// LiftRidePhase identifies the stage of a deck-to-deck lift cinematic.
type LiftRidePhase int

const (
	LiftRideDepart LiftRidePhase = iota
	LiftRideTransit
	LiftRideArrive
)

const (
	LiftRideDepartMs  = 400
	LiftRideTransitMs = 1400
	LiftRideArriveMs  = 450

	// LiftRideScrollTilesPerSec is constant shaft scroll speed during the ride.
	LiftRideScrollTilesPerSec = 3.2
)

// LiftRideSession runs a shaft-anchored ride cinematic between decks.
type LiftRideSession struct {
	FromLevel int
	ToLevel   int

	FromExitRow int
	FromExitCol int
	ToExitRow   int
	ToExitCol   int

	Phase        LiftRidePhase
	PhaseStartMs int64
	// RideStartMs is the wall-clock start of the cinematic (scroll is absolute from this).
	RideStartMs int64

	// NeedsSwap is set by the Update-thread advancer at mid-transit; the game
	// loop performs TravelToDeck so level generation does not freeze rendering.
	NeedsSwap bool
	Swapped   bool
	// SwapInProgress freezes the map buffer while TravelToDeck mutates the grid.
	SwapInProgress bool

	// Direction is +1 when traveling to a higher deck number, -1 when lower.
	Direction int

	// ScrollOffset is cumulative vertical scroll in tile units (signed by Direction).
	ScrollOffset float64
	// ArriveScrollBase is ScrollOffset captured when entering Arrive (for ease-to-rest).
	ArriveScrollBase float64
	// PeripheralAlpha is 1 for a fully visible map, 0 when only the shaft shows.
	PeripheralAlpha float64
}

// CameraAt returns the map camera center for the ride at nowMs.
// While SwapInProgress, keep the source exit so a frozen buffer is never blitted
// with the destination camera (which shifts the shaft toward the bottom-right).
func (s *LiftRideSession) CameraAt(nowMs int64) (camRow, camCol float64, ok bool) {
	if s == nil {
		return 0, 0, false
	}
	_ = nowMs
	if s.Swapped && !s.SwapInProgress {
		return float64(s.ToExitRow), float64(s.ToExitCol), true
	}
	return float64(s.FromExitRow), float64(s.FromExitCol), true
}

// UpdateVisual refreshes scroll and periphery alpha from the current phase clock.
func (s *LiftRideSession) UpdateVisual(nowMs int64) {
	if s == nil {
		return
	}
	elapsed := nowMs - s.PhaseStartMs
	if elapsed < 0 {
		elapsed = 0
	}
	dir := float64(s.Direction)
	if dir == 0 {
		dir = 1
	}

	switch s.Phase {
	case LiftRideDepart:
		t := float64(elapsed) / float64(LiftRideDepartMs)
		if t > 1 {
			t = 1
		}
		s.PeripheralAlpha = 1 - liftRideEase(t)
		s.ScrollOffset = dir * LiftRideScrollTilesPerSec * float64(nowMs-s.RideStartMs) / 1000
	case LiftRideTransit:
		s.PeripheralAlpha = 0
		s.ScrollOffset = dir * LiftRideScrollTilesPerSec * float64(nowMs-s.RideStartMs) / 1000
	case LiftRideArrive:
		t := float64(elapsed) / float64(LiftRideArriveMs)
		if t > 1 {
			t = 1
		}
		ease := liftRideEase(t)
		s.PeripheralAlpha = ease
		// Ease residual fractional scroll toward a resting integer offset.
		base := s.ArriveScrollBase
		frac := base - math.Trunc(base)
		s.ScrollOffset = base - frac*ease
	default:
		s.PeripheralAlpha = 1
	}
	if s.PeripheralAlpha < 0 {
		s.PeripheralAlpha = 0
	}
	if s.PeripheralAlpha > 1 {
		s.PeripheralAlpha = 1
	}
}

// ScrollPixelOffset returns the vertical pixel shift for shaft tiles (wrap-friendly, sub-pixel).
func (s *LiftRideSession) ScrollPixelOffset(tileSize int) float64 {
	if s == nil || tileSize <= 0 {
		return 0
	}
	px := s.ScrollOffset * float64(tileSize)
	mod := math.Mod(px, float64(tileSize))
	if mod < 0 {
		mod += float64(tileSize)
	}
	// Positive Direction scrolls content down the screen.
	if s.Direction < 0 {
		return -mod
	}
	return mod
}

func liftRideEase(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	// Smoothstep.
	return t * t * (3 - 2*t)
}
