package menu

import (
	"strings"
	"testing"

	"darkstation/pkg/game/state"
)

func TestLiftMenuHandler_initialSelectionCurrentDeck(t *testing.T) {
	g := state.NewGame()
	g.InitRunUnlocks(42)
	g.CurrentDeckID = 1

	handler := NewLiftMenuHandler(g)
	got := handler.InitialMenuSelection(handler.items)
	if got != 1 {
		t.Fatalf("InitialMenuSelection = %d, want 1 (current deck)", got)
	}
	current := handler.items[1].(*LiftDeckItem)
	if !current.IsSelectable() {
		t.Fatal("current deck row should be selectable for highlight")
	}
	closeMenu, help := handler.OnActivate(current, 1)
	if closeMenu || help == "" {
		t.Fatalf("activate current deck: closeMenu=%v help=%q, want stay open with message", closeMenu, help)
	}
}

func TestLiftDeckItem_includesThemeTitle(t *testing.T) {
	g := state.NewGame()
	g.InitRunUnlocks(42)
	g.CurrentDeckID = 0

	item := &LiftDeckItem{
		DeckID: 2,
		Level:  3,
		G:      g,
	}
	label := item.GetLabel()
	if !strings.Contains(label, "Deck 3") {
		t.Fatalf("label %q should include deck number", label)
	}
	theme := g.ThemeForDeck(2)
	if theme == "" {
		t.Fatal("expected theme for deck 3")
	}
	if !strings.Contains(label, "—") {
		t.Fatalf("label %q should include theme separator", label)
	}
}
