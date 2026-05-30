package state

import "testing"

func TestHazardClearSession_UpdateVisualAlpha_flashPulse(t *testing.T) {
	s := &HazardClearSession{
		Phase:        HazardClearFlash,
		PhaseStartMs: 1000,
	}
	s.UpdateVisualAlpha(1000)
	if s.VisualAlpha != 1 {
		t.Fatalf("flash start alpha = %v, want 1", s.VisualAlpha)
	}
	s.UpdateVisualAlpha(1000 + hazardClearFlashHalfPeriodMs)
	if s.VisualAlpha >= 1 || s.VisualAlpha <= 0 {
		t.Fatalf("flash dim alpha = %v, want between 0 and 1", s.VisualAlpha)
	}
}

func TestHazardClearSession_UpdateVisualAlpha_fade(t *testing.T) {
	s := &HazardClearSession{
		Phase:        HazardClearFade,
		PhaseStartMs: 1000,
		FadeProgress: 0.5,
	}
	s.UpdateVisualAlpha(1500)
	if s.VisualAlpha != 0.5 {
		t.Fatalf("fade alpha = %v, want 0.5", s.VisualAlpha)
	}
}
