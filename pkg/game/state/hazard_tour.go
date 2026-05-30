package state

// HazardTourPhase identifies the stage of an exit-lift hazard tour cinematic.
type HazardTourPhase int

const (
	HazardTourPanTo HazardTourPhase = iota
	HazardTourHighlight
	HazardTourPanBack
)

// HazardTourTarget is a blocking hazard cell shown during the tour.
type HazardTourTarget struct {
	Row, Col int
}

// HazardTourSession pans the camera through remaining hazards when the exit lift is blocked.
type HazardTourSession struct {
	Targets      []HazardTourTarget
	Index        int
	ReturnCamRow float64
	ReturnCamCol float64
	Phase        HazardTourPhase
	PhaseStartMs int64
}

const HazardTourHighlightMs = 2000

// CameraAt returns the map camera center for the tour at nowMs.
func (s *HazardTourSession) CameraAt(nowMs int64) (camRow, camCol float64, ok bool) {
	if s == nil || len(s.Targets) == 0 {
		return 0, 0, false
	}

	elapsed := nowMs - s.PhaseStartMs
	if elapsed < 0 {
		elapsed = 0
	}

	switch s.Phase {
	case HazardTourPanTo:
		fromR, fromC := s.panFrom()
		to := s.Targets[s.Index]
		t := hazardClearEase(float64(elapsed) / float64(HazardClearPanMs))
		return fromR + (float64(to.Row)-fromR)*t, fromC + (float64(to.Col)-fromC)*t, true
	case HazardTourHighlight:
		t := s.Targets[s.Index]
		return float64(t.Row), float64(t.Col), true
	case HazardTourPanBack:
		last := s.Targets[len(s.Targets)-1]
		t := hazardClearEase(float64(elapsed) / float64(HazardClearPanMs))
		return float64(last.Row) + (s.ReturnCamRow-float64(last.Row))*t,
			float64(last.Col) + (s.ReturnCamCol-float64(last.Col))*t, true
	default:
		return 0, 0, false
	}
}

// HighlightCell reports the hazard cell to focus during the highlight phase.
func (s *HazardTourSession) HighlightCell() (row, col int, active bool) {
	if s == nil || s.Phase != HazardTourHighlight || s.Index < 0 || s.Index >= len(s.Targets) {
		return 0, 0, false
	}
	t := s.Targets[s.Index]
	return t.Row, t.Col, true
}

func (s *HazardTourSession) panFrom() (float64, float64) {
	if s.Index <= 0 {
		return s.ReturnCamRow, s.ReturnCamCol
	}
	prev := s.Targets[s.Index-1]
	return float64(prev.Row), float64(prev.Col)
}
