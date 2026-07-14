package state

import (
	"testing"

	"darkstation/pkg/game/gamemode"
)

func TestGame_TotalDecks_DefaultMode(t *testing.T) {
	g := NewGame()
	if got := g.TotalDecks(); got != 10 {
		t.Fatalf("TotalDecks() = %d, want 10", got)
	}
}

func TestGame_SetMode_SingleDeckSandbox(t *testing.T) {
	g := NewGame()
	g.SetMode(gamemode.SingleDeckSandbox)
	if got := g.TotalDecks(); got != 1 {
		t.Fatalf("TotalDecks() = %d, want 1", got)
	}
	if g.Mode().ID != gamemode.SingleDeckSandbox {
		t.Fatalf("Mode().ID = %q", g.Mode().ID)
	}
	if !g.IsFinalDeckLevel(1) {
		t.Fatal("IsFinalDeckLevel(1) = false, want true for single-deck mode")
	}
}

func TestInitRunUnlocks_SingleDeckSkipsCrossDeckPlan(t *testing.T) {
	g := NewGame()
	g.SetMode(gamemode.SingleDeckSandbox)
	g.InitRunUnlocks(42)
	if len(g.UnlockPlan.Requirements) != 0 {
		t.Fatalf("requirements = %d, want 0 for sandbox mode", len(g.UnlockPlan.Requirements))
	}
	if len(g.LiftRoutingPowered) != 1 || !g.LiftRoutingPowered[0] {
		t.Fatalf("LiftRoutingPowered = %#v, want only deck 0", g.LiftRoutingPowered)
	}
}
