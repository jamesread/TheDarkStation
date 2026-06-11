package gameplay

import (
	"testing"

	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
)

func TestTravelToDeck_spawnsOnLiftShaft(t *testing.T) {
	g := buildGameWithSeed(1, 424242)
	unlockAllDecksForTest(g)

	exitBefore := g.Grid.ExitCell()
	if exitBefore == nil {
		t.Fatal("deck 1 missing exit cell")
	}
	TeleportPlayerTo(g, exitBefore)

	if err := TravelToDeck(g, 2); err != nil {
		t.Fatalf("TravelToDeck: %v", err)
	}
	exit := g.Grid.ExitCell()
	if exit == nil {
		t.Fatal("deck 2 missing exit cell")
	}
	if g.CurrentCell != exit {
		t.Fatalf("after travel: player at (%d,%d) %q, want lift at (%d,%d) %q",
			g.CurrentCell.Row, g.CurrentCell.Col, g.CurrentCell.Name,
			exit.Row, exit.Col, exit.Name)
	}
}

func TestTravelToDeck_spawnsOnLiftWhenExitLocked(t *testing.T) {
	g := buildGameWithSeed(1, 424242)
	unlockAllDecksForTest(g)
	TeleportPlayerTo(g, g.Grid.ExitCell())

	if err := TravelToDeck(g, 2); err != nil {
		t.Fatalf("TravelToDeck: %v", err)
	}
	if setup.ExitLiftState(g) == state.ExitLiftReady {
		t.Skip("seed has deck 2 lift already ready at arrival")
	}
	exit := g.Grid.ExitCell()
	if g.CurrentCell != exit {
		t.Fatalf("locked lift should still receive player on exit cell, got (%d,%d)", g.CurrentCell.Row, g.CurrentCell.Col)
	}
}
