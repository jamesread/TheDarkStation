package state

import "testing"

func TestLiftRideSession_CameraAt_swapsAnchor(t *testing.T) {
	s := &LiftRideSession{
		FromExitRow: 4,
		FromExitCol: 10,
		ToExitRow:   7,
		ToExitCol:   13,
		Direction:   1,
	}
	r, c, ok := s.CameraAt(0)
	if !ok || r != 4 || c != 10 {
		t.Fatalf("before swap: cam=(%v,%v) ok=%v, want (4,10) true", r, c, ok)
	}
	s.Swapped = true
	s.SwapInProgress = true
	r, c, ok = s.CameraAt(0)
	if !ok || r != 4 || c != 10 {
		t.Fatalf("during swap freeze: cam=(%v,%v) ok=%v, want source (4,10)", r, c, ok)
	}
	s.SwapInProgress = false
	r, c, ok = s.CameraAt(0)
	if !ok || r != 7 || c != 13 {
		t.Fatalf("after swap: cam=(%v,%v) ok=%v, want (7,13) true", r, c, ok)
	}
}

func TestLiftRideSession_UpdateVisual_departDims(t *testing.T) {
	s := &LiftRideSession{
		Phase:           LiftRideDepart,
		PhaseStartMs:    1000,
		RideStartMs:     1000,
		Direction:       1,
		PeripheralAlpha: 1,
	}
	s.UpdateVisual(1000)
	if s.PeripheralAlpha != 1 {
		t.Fatalf("t=0 alpha=%v, want 1", s.PeripheralAlpha)
	}
	s.UpdateVisual(1000 + LiftRideDepartMs)
	if s.PeripheralAlpha > 0.01 {
		t.Fatalf("end depart alpha=%v, want ~0", s.PeripheralAlpha)
	}
	if s.ScrollOffset <= 0 {
		t.Fatalf("depart scroll=%v, want > 0 for ascending", s.ScrollOffset)
	}
}

func TestLiftRideSession_UpdateVisual_constantScrollSpeed(t *testing.T) {
	s := &LiftRideSession{
		Phase:        LiftRideTransit,
		PhaseStartMs: 0,
		RideStartMs:  0,
		Direction:    1,
	}
	s.UpdateVisual(1000)
	got := s.ScrollOffset
	want := LiftRideScrollTilesPerSec
	if got < want-0.01 || got > want+0.01 {
		t.Fatalf("scroll after 1s = %v, want %v", got, want)
	}
}

func TestLiftRideSession_ScrollPixelOffset_wraps(t *testing.T) {
	s := &LiftRideSession{Direction: 1, ScrollOffset: 2.25}
	got := s.ScrollPixelOffset(40)
	if got < 0 || got >= 40 {
		t.Fatalf("ScrollPixelOffset=%v, want in [0,40)", got)
	}
}
