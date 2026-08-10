package gameplay

import (
	"testing"

	"darkstation/pkg/game/state"
)

func TestStartLiftRide_setsSession(t *testing.T) {
	g := buildGameWithSeed(1, 424242)
	unlockAllDecksForTest(g)
	TeleportPlayerTo(g, g.Grid.ExitCell())

	if err := StartLiftRide(g, 2); err != nil {
		t.Fatalf("StartLiftRide: %v", err)
	}
	if g.LiftRide == nil {
		t.Fatal("expected LiftRide session")
	}
	if g.LiftRide.FromLevel != 1 || g.LiftRide.ToLevel != 2 {
		t.Fatalf("from/to = %d/%d, want 1/2", g.LiftRide.FromLevel, g.LiftRide.ToLevel)
	}
	if g.LiftRide.Phase != state.LiftRideDepart {
		t.Fatalf("phase = %v, want Depart", g.LiftRide.Phase)
	}
	if !IsGameplayCinematicActive(g) {
		t.Fatal("expected cinematic active during lift ride")
	}
	finishLiftRide(g)
}

func TestAdvanceLiftRide_swapsCachedDeck(t *testing.T) {
	g := buildGameWithSeed(1, 424242)
	unlockAllDecksForTest(g)
	TeleportPlayerTo(g, g.Grid.ExitCell())

	// Prefetch destination so mid-ride swap avoids a full generate in this unit test.
	if err := TravelToDeck(g, 2); err != nil {
		t.Fatalf("prefetch deck 2: %v", err)
	}
	if err := TravelToDeck(g, 1); err != nil {
		t.Fatalf("return deck 1: %v", err)
	}
	TeleportPlayerTo(g, g.Grid.ExitCell())

	if err := StartLiftRide(g, 2); err != nil {
		t.Fatalf("StartLiftRide: %v", err)
	}
	now := g.LiftRide.PhaseStartMs

	// Depart → Transit
	AdvanceLiftRideIfActive(g, now+state.LiftRideDepartMs)
	if g.LiftRide.Phase != state.LiftRideTransit {
		t.Fatalf("phase after depart = %v, want Transit", g.LiftRide.Phase)
	}

	// Mid-transit requests swap
	AdvanceLiftRideIfActive(g, g.LiftRide.PhaseStartMs+state.LiftRideTransitMs/2)
	if !g.LiftRide.NeedsSwap {
		t.Fatal("expected NeedsSwap at mid-transit")
	}
	applyLiftRideSwap(g)
	if g.Level != 2 {
		t.Fatalf("after swap Level=%d, want 2", g.Level)
	}
	if g.LiftRide == nil || !g.LiftRide.Swapped {
		t.Fatal("expected Swapped after applyLiftRideSwap")
	}
	if g.LiftRide.SwapInProgress {
		t.Fatal("expected SwapInProgress cleared after applyLiftRideSwap")
	}
	exit := g.Grid.ExitCell()
	if exit == nil || g.CurrentCell != exit {
		t.Fatal("expected player on destination exit after swap")
	}

	// Finish transit → arrive → clear
	AdvanceLiftRideIfActive(g, g.LiftRide.PhaseStartMs+state.LiftRideTransitMs)
	if g.LiftRide == nil || g.LiftRide.Phase != state.LiftRideArrive {
		t.Fatalf("expected Arrive phase, got %#v", g.LiftRide)
	}
	AdvanceLiftRideIfActive(g, g.LiftRide.PhaseStartMs+state.LiftRideArriveMs)
	if g.LiftRide != nil {
		t.Fatal("expected LiftRide cleared after Arrive")
	}
}
