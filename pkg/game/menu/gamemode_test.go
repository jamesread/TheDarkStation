package menu

import (
	"testing"

	"darkstation/pkg/game/gamemode"
)

func TestGameModeMenuItem_LabelAndHelp(t *testing.T) {
	item := &GameModeMenuItem{Mode: gamemode.Get(gamemode.SinglePlayerPuzzle)}
	if item.GetLabel() == "" {
		t.Fatal("GetLabel returned empty string")
	}
	if item.GetHelpText() == "" {
		t.Fatal("GetHelpText returned empty string")
	}
}

func TestNewGameModeMenuHandler_ListsAllModes(t *testing.T) {
	h := NewGameModeMenuHandler(gamemode.SingleDeckSandbox)
	if len(h.items) != len(gamemode.All()) {
		t.Fatalf("items = %d, want %d", len(h.items), len(gamemode.All()))
	}
	if got := h.InitialMenuSelection(h.items); got != 1 {
		t.Fatalf("InitialMenuSelection = %d, want 1 (SingleDeckSandbox)", got)
	}
}

func TestGameModeMenuHandler_OnActivate(t *testing.T) {
	h := NewGameModeMenuHandler(gamemode.SinglePlayerPuzzle)
	item := h.items[0]
	closeMenu, _ := h.OnActivate(item, 0)
	if !closeMenu {
		t.Fatal("OnActivate should close menu")
	}
	if !h.confirmed {
		t.Fatal("confirmed = false, want true")
	}
	if h.selectedMode != gamemode.SinglePlayerPuzzle {
		t.Fatalf("selectedMode = %q", h.selectedMode)
	}
}
