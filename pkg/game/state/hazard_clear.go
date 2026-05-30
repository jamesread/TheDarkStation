package state

import "darkstation/pkg/game/entities"

// HazardClearPhase identifies the stage of a hazard shutdown cinematic.
type HazardClearPhase int

const (
	HazardClearPanTo HazardClearPhase = iota
	HazardClearFlash
	HazardClearFade
	HazardClearPanBack
)

// HazardClearPending describes the fix applied when the fade completes.
type HazardClearPending struct {
	Hazard *entities.Hazard
	// Control is activated on completion when non-nil (circuit breaker path).
	Control *entities.HazardControl
	// ItemName is removed from inventory on completion when non-empty (patch kit path).
	ItemName string
	// Callout shown when the fix completes.
	CalloutRow, CalloutCol int
	CalloutMessage         string
	CalloutStyle           string // renderer style name; resolved in gameplay
	LogMessage             string
}

// HazardClearSession runs a camera pan/flash/fade cinematic before a hazard is cleared.
type HazardClearSession struct {
	HazardRow, HazardCol int
	ReturnCamRow         float64
	ReturnCamCol         float64
	Phase                HazardClearPhase
	PhaseStartMs         int64
	FadeProgress         float64 // 0..1 during HazardClearFade
	VisualAlpha          float64 // hazard icon/background alpha during flash and fade (1 = opaque)
	Pending              HazardClearPending
}

const (
	HazardClearPanMs   = 800
	HazardClearFlashMs = 500 // rapid blink before fade-out
)

const hazardClearFlashHalfPeriodMs = 90

// CameraAt returns the map camera center for the cinematic at nowMs.
func (s *HazardClearSession) CameraAt(nowMs int64) (camRow, camCol float64, ok bool) {
	if s == nil {
		return 0, 0, false
	}
	hazardR := float64(s.HazardRow)
	hazardC := float64(s.HazardCol)
	retR := s.ReturnCamRow
	retC := s.ReturnCamCol

	elapsed := nowMs - s.PhaseStartMs
	if elapsed < 0 {
		elapsed = 0
	}

	switch s.Phase {
	case HazardClearPanTo:
		t := hazardClearEase(float64(elapsed) / float64(HazardClearPanMs))
		return retR + (hazardR-retR)*t, retC + (hazardC-retC)*t, true
	case HazardClearFlash, HazardClearFade:
		return hazardR, hazardC, true
	case HazardClearPanBack:
		t := hazardClearEase(float64(elapsed) / float64(HazardClearPanMs))
		return hazardR + (retR-hazardR)*t, hazardC + (retC-hazardC)*t, true
	default:
		return 0, 0, false
	}
}

func hazardClearEase(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * t * (t*(t*6-15) + 10)
}

// UpdateVisualAlpha sets VisualAlpha for hazard tile rendering during flash and fade phases.
func (s *HazardClearSession) UpdateVisualAlpha(nowMs int64) {
	if s == nil {
		return
	}
	elapsed := nowMs - s.PhaseStartMs
	if elapsed < 0 {
		elapsed = 0
	}
	switch s.Phase {
	case HazardClearFlash:
		if (elapsed/hazardClearFlashHalfPeriodMs)%2 == 0 {
			s.VisualAlpha = 1
		} else {
			s.VisualAlpha = 0.35
		}
	case HazardClearFade:
		alpha := 1 - s.FadeProgress
		if alpha < 0 {
			alpha = 0
		}
		s.VisualAlpha = alpha
	default:
		s.VisualAlpha = 1
	}
}
