// Package gameplay tests lifecycle functions: BuildGame, AdvanceLevel, TriggerGameComplete.
package gameplay

import (
	"testing"

	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/state"
)

func TestBuildGame_GeneratesOnlyStartingDeck(t *testing.T) {
	// BuildGame must generate only the starting deck; no pre-generation of all decks.
	g := BuildGame(1)
	if g == nil {
		t.Fatal("BuildGame(1) returned nil")
	}
	if g.Grid == nil {
		t.Fatal("BuildGame(1): Grid is nil")
	}
	if g.CurrentDeckID != 0 || g.Level != 1 {
		t.Errorf("BuildGame(1): CurrentDeckID=%d Level=%d, want 0,1", g.CurrentDeckID, g.Level)
	}
	if len(g.DeckStates) != 0 {
		t.Errorf("BuildGame(1): DeckStates should be empty (no decks stored yet), got %d entries", len(g.DeckStates))
	}
}

func TestBuildGame_ClampsStartLevelToValidRange(t *testing.T) {
	// BuildGame clamps startLevel to [1, TotalDecks]; no panic or out-of-range state.
	// startLevel <= 0 → clamp to 1 (first deck)
	g0 := BuildGame(0)
	if g0 == nil || g0.Grid == nil {
		t.Fatal("BuildGame(0) returned nil or nil Grid")
	}
	if g0.CurrentDeckID != 0 || g0.Level != 1 {
		t.Errorf("BuildGame(0): CurrentDeckID=%d Level=%d, want 0,1", g0.CurrentDeckID, g0.Level)
	}
	gNeg := BuildGame(-1)
	if gNeg == nil || gNeg.Grid == nil {
		t.Fatal("BuildGame(-1) returned nil or nil Grid")
	}
	if gNeg.CurrentDeckID != 0 || gNeg.Level != 1 {
		t.Errorf("BuildGame(-1): CurrentDeckID=%d Level=%d, want 0,1", gNeg.CurrentDeckID, gNeg.Level)
	}
	// startLevel > TotalDecks → clamp to final deck
	gOver := BuildGame(deck.TotalDecks + 1)
	if gOver == nil || gOver.Grid == nil {
		t.Fatal("BuildGame(TotalDecks+1) returned nil or nil Grid")
	}
	if gOver.CurrentDeckID != deck.FinalDeckIndex || gOver.Level != deck.TotalDecks {
		t.Errorf("BuildGame(TotalDecks+1): CurrentDeckID=%d Level=%d, want %d,%d",
			gOver.CurrentDeckID, gOver.Level, deck.FinalDeckIndex, deck.TotalDecks)
	}
}

func TestDeck_FixedCountAndFinalDeck(t *testing.T) {
	// deck.TotalDecks and deck.Graph define fixed count; final deck has empty Connections.
	if deck.TotalDecks < 1 {
		t.Fatal("TotalDecks must be >= 1")
	}
	if len(deck.Graph) != deck.TotalDecks {
		t.Errorf("Graph length %d != TotalDecks %d", len(deck.Graph), deck.TotalDecks)
	}
	// Final deck (index TotalDecks-1) must have no Connections
	finalIdx := deck.FinalDeckIndex
	if finalIdx != deck.TotalDecks-1 {
		t.Errorf("FinalDeckIndex=%d, want TotalDecks-1=%d", finalIdx, deck.TotalDecks-1)
	}
	if len(deck.Graph[finalIdx].Connections) != 0 {
		t.Errorf("Final deck Connections must be empty, got %v", deck.Graph[finalIdx].Connections)
	}
	// Non-final decks must have exactly one connection (next deck)
	for i := 0; i < finalIdx; i++ {
		if len(deck.Graph[i].Connections) != 1 || deck.Graph[i].Connections[0] != i+1 {
			t.Errorf("Deck %d Connections=%v, want [%d]", i, deck.Graph[i].Connections, i+1)
		}
	}
}

func TestNextDeckID_FinalDeckReturnsFalse(t *testing.T) {
	// NextDeckID on final deck returns false; no advance possible.
	_, ok := deck.NextDeckID(deck.FinalDeckIndex)
	if ok {
		t.Error("NextDeckID(FinalDeckIndex) ok=true, want false")
	}
	_, ok = deck.NextDeckID(deck.TotalDecks) // out of range
	if ok {
		t.Error("NextDeckID(TotalDecks) ok=true, want false (out of range)")
	}
}

func TestAdvanceLevel_GeneratesOnFirstEntry(t *testing.T) {
	// When DeckStates[nextID] has no grid, AdvanceLevel generates the next deck.
	g := BuildGame(1)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame(1) setup failed")
	}
	// AdvanceLevel(1) should generate deck 2 (no stored state)
	AdvanceLevel(g)
	if g.CurrentDeckID != 1 || g.Level != 2 {
		t.Errorf("after AdvanceLevel: CurrentDeckID=%d Level=%d, want 1,2", g.CurrentDeckID, g.Level)
	}
	if g.Grid == nil {
		t.Fatal("AdvanceLevel: Grid is nil after first entry")
	}
	if len(g.DeckStates) < 1 {
		t.Error("AdvanceLevel: DeckStates should have deck 0 saved")
	}
}

func TestAdvanceLevel_LoadsStoredDeckWhenPresent(t *testing.T) {
	// When DeckStates[nextID] has grid, AdvanceLevel loads it (no re-generation).
	g := BuildGame(1)
	g.SaveCurrentDeckState()
	// Advance to deck 2
	AdvanceLevel(g)
	if g.CurrentDeckID != 1 {
		t.Fatalf("CurrentDeckID=%d, want 1", g.CurrentDeckID)
	}
	secondGrid := g.Grid
	// Save deck 2, go back to deck 1, then advance again - should load stored deck 2
	g.SaveCurrentDeckState()
	g.LoadDeckState(0) // back to deck 1
	AdvanceLevel(g)
	if g.Grid != secondGrid {
		t.Error("AdvanceLevel should load stored deck 2, not regenerate")
	}
}

func TestAdvanceLevel_FinalDeckNoAdvance(t *testing.T) {
	// AdvanceLevel on final deck does nothing (NextDeckID returns false).
	g := BuildGame(deck.TotalDecks)
	if g == nil {
		t.Fatal("BuildGame(deck.TotalDecks) returned nil")
	}
	AdvanceLevel(g)
	if g.CurrentDeckID != deck.FinalDeckIndex || g.Level != deck.TotalDecks {
		t.Errorf("AdvanceLevel on final deck changed state: CurrentDeckID=%d Level=%d", g.CurrentDeckID, g.Level)
	}
}

func TestResetLevel_DoesNotAdvanceDeck(t *testing.T) {
	// ResetLevel regenerates the current deck only; CurrentDeckID and Level must not change.
	g := BuildGame(1)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame(1) setup failed")
	}
	ResetLevel(g)
	if g.CurrentDeckID != 0 || g.Level != 1 {
		t.Errorf("after ResetLevel: CurrentDeckID=%d Level=%d, want 0,1 (must not advance)", g.CurrentDeckID, g.Level)
	}
	if g.Grid == nil {
		t.Fatal("ResetLevel: Grid is nil after reset")
	}
	// After advancing once, reset should keep us on deck 2 (reset same deck, not go back)
	AdvanceLevel(g)
	if g.CurrentDeckID != 1 || g.Level != 2 {
		t.Fatalf("after AdvanceLevel: CurrentDeckID=%d Level=%d, want 1,2", g.CurrentDeckID, g.Level)
	}
	ResetLevel(g)
	if g.CurrentDeckID != 1 || g.Level != 2 {
		t.Errorf("after ResetLevel on deck 2: CurrentDeckID=%d Level=%d, want 1,2 (must not change deck)", g.CurrentDeckID, g.Level)
	}
}

func TestTriggerGameComplete_SetsGameComplete(t *testing.T) {
	g := state.NewGame()
	if g.GameComplete {
		t.Fatal("new game should not be complete")
	}
	TriggerGameComplete(g)
	if !g.GameComplete {
		t.Error("TriggerGameComplete should set GameComplete=true")
	}
}

func TestIsFinalDeck_MatchesTotalDecks(t *testing.T) {
	if !deck.IsFinalDeck(deck.TotalDecks) {
		t.Errorf("IsFinalDeck(TotalDecks)=false, want true")
	}
	if deck.IsFinalDeck(deck.TotalDecks - 1) {
		t.Errorf("IsFinalDeck(TotalDecks-1)=true, want false")
	}
}

func TestAdvanceThroughAllDecks_FinalDeckReachable(t *testing.T) {
	// Advances through all decks; each is generated once; final deck is reachable.
	// Note: runs 10 full BSP+SetupLevel generations; may be slow under -race or on constrained CI.
	g := BuildGame(1)
	seenDecks := make(map[int]bool)
	seenDecks[0] = true
	for g.CurrentDeckID < deck.FinalDeckIndex {
		AdvanceLevel(g)
		seenDecks[g.CurrentDeckID] = true
		if g.Grid == nil {
			t.Fatalf("deck %d: Grid is nil after AdvanceLevel", g.CurrentDeckID)
		}
	}
	// Should be on final deck
	if g.CurrentDeckID != deck.FinalDeckIndex || g.Level != deck.TotalDecks {
		t.Errorf("expected final deck: CurrentDeckID=%d Level=%d", g.CurrentDeckID, g.Level)
	}
	// All decks should have been visited
	for i := 0; i < deck.TotalDecks; i++ {
		if !seenDecks[i] {
			t.Errorf("deck %d was never visited", i)
		}
	}
	// TriggerGameComplete on final deck
	TriggerGameComplete(g)
	if !g.GameComplete {
		t.Error("TriggerGameComplete should set GameComplete on final deck")
	}
}
