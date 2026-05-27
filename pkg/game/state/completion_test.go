package state

import (
	"testing"

	"darkstation/pkg/game/deck"
)

func TestSnapshotRunStats(t *testing.T) {
	g := NewGame()
	g.RunStartedAt = 0
	g.MovementCount = 12
	g.InteractionsCount = 3

	stats := g.SnapshotRunStats()
	if stats.DecksCompleted != deck.TotalDecks {
		t.Fatalf("DecksCompleted = %d, want %d", stats.DecksCompleted, deck.TotalDecks)
	}
	if stats.Movements != 12 {
		t.Fatalf("Movements = %d, want 12", stats.Movements)
	}
	if stats.Interactions != 3 {
		t.Fatalf("Interactions = %d, want 3", stats.Interactions)
	}
}

func TestResetAllProgress(t *testing.T) {
	g := NewGame()
	g.Level = 8
	g.CurrentDeckID = 7
	g.DeckStates[2] = &DeckState{}
	g.GameComplete = true

	g.ResetAllProgress()

	if g.Level != 1 || g.CurrentDeckID != 0 {
		t.Fatalf("after reset: level=%d deck=%d, want 1/0", g.Level, g.CurrentDeckID)
	}
	if len(g.DeckStates) != 0 {
		t.Fatal("DeckStates should be empty after reset")
	}
	if g.GameComplete {
		t.Fatal("GameComplete should be false after reset")
	}
}
