package menu

import (
	"testing"

	"darkstation/pkg/game/state"
)

func TestMainMenuHandler_PerfMapShortcut(t *testing.T) {
	h := NewMainMenuHandler()
	if !h.HandlePerfMapShortcut(state.NewGame(), "entities") {
		t.Fatal("perf map shortcut should close the main menu")
	}
	if h.GetSelectedAction() != MainMenuActionPerfMap {
		t.Fatalf("selected action = %v, want MainMenuActionPerfMap", h.GetSelectedAction())
	}
	if h.GetPerfMapScenario() != "entities" {
		t.Fatalf("perf map scenario = %q, want entities", h.GetPerfMapScenario())
	}
}
