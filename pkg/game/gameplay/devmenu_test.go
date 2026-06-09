package gameplay

import (
	"strings"
	"testing"

	gamemenu "darkstation/pkg/game/menu"
	"darkstation/pkg/game/state"
)

func TestDevMenuHandler_GetMenuItems(t *testing.T) {
	h := NewDevMenuHandler(state.NewGame())
	items := h.GetMenuItems()
	if len(items) != 13 {
		t.Fatalf("expected 13 items, got %d", len(items))
	}
	if items[0].GetLabel() != "Zoom\tSUBTLE{24px (30×15 tiles)}" {
		t.Fatalf("item 0 label = %q", items[0].GetLabel())
	}
	expected := map[DevMenuAction]string{
		DevMenuActionDumpMap:              "Dump map",
		DevMenuActionListCurrentCellChars: "list current cell chars",
		DevMenuActionDevTestMap:           "Developer test map",
		DevMenuActionToggleMapAreaBorder:  "Map area border",
		DevMenuActionToggleFOVRays:        "FOV ray lines",
		DevMenuActionToggleFPSDisplay:     "FPS display",
		DevMenuActionTogglePlayerPosition: "Player position",
		DevMenuActionLoadSeed:             "Load level seed",
		DevMenuActionJumpToDeck:           "Jump to deck",
		DevMenuActionTriggerOverload:      "Trigger overload",
	}
	for action, wantPrefix := range expected {
		item := findDevMenuItem(t, items, action)
		if !strings.HasPrefix(item.GetLabel(), wantPrefix) {
			t.Fatalf("action %v label = %q, want prefix %q", action, item.GetLabel(), wantPrefix)
		}
	}
	if items[12].GetLabel() != "Close" {
		t.Fatalf("item 12 label = %q", items[12].GetLabel())
	}
}

func TestDevMenuHandler_ToggleMapAreaBorder(t *testing.T) {
	h := NewDevMenuHandler(state.NewGame())
	toggle := &DevMenuItem{Label: "Map area border: OFF", Action: DevMenuActionToggleMapAreaBorder}
	shouldClose, help := h.OnActivate(toggle, 2)
	if shouldClose {
		t.Fatal("toggle should keep menu open")
	}
	if help != "Map area border: OFF" {
		t.Fatalf("help = %q", help)
	}
	if label := findDevMenuItem(t, h.GetMenuItems(), DevMenuActionToggleMapAreaBorder).GetLabel(); label != "Map area border\tUNPOWERED{OFF}" {
		t.Fatalf("label after toggle without renderer = %q", label)
	}
}

func TestDevMenuHandler_ToggleFOVRays(t *testing.T) {
	h := NewDevMenuHandler(state.NewGame())
	toggle := &DevMenuItem{Label: "FOV ray lines: OFF", Action: DevMenuActionToggleFOVRays}
	shouldClose, help := h.OnActivate(toggle, 3)
	if shouldClose {
		t.Fatal("toggle should keep menu open")
	}
	if help != "FOV ray lines: OFF" {
		t.Fatalf("help = %q", help)
	}
	if label := findDevMenuItem(t, h.GetMenuItems(), DevMenuActionToggleFOVRays).GetLabel(); label != "FOV ray lines\tUNPOWERED{OFF}" {
		t.Fatalf("label after toggle without renderer = %q", label)
	}
}

func TestDevMenuHandler_ToggleFPSDisplay(t *testing.T) {
	h := NewDevMenuHandler(state.NewGame())
	toggle := &DevMenuItem{Label: "FPS display: ON", Action: DevMenuActionToggleFPSDisplay}
	shouldClose, help := h.OnActivate(toggle, 4)
	if shouldClose {
		t.Fatal("toggle should keep menu open")
	}
	if help != "FPS display: ON" {
		t.Fatalf("help = %q", help)
	}
	if label := findDevMenuItem(t, h.GetMenuItems(), DevMenuActionToggleFPSDisplay).GetLabel(); label != "FPS display\tPOWERED{ON}" {
		t.Fatalf("label after toggle without renderer = %q", label)
	}
}

func TestDevMenuHandler_TogglePlayerPosition(t *testing.T) {
	h := NewDevMenuHandler(state.NewGame())
	toggle := &DevMenuItem{Label: "Player position: OFF", Action: DevMenuActionTogglePlayerPosition}
	shouldClose, help := h.OnActivate(toggle, 5)
	if shouldClose {
		t.Fatal("toggle should keep menu open")
	}
	if help != "Player position: OFF" {
		t.Fatalf("help = %q", help)
	}
	if label := findDevMenuItem(t, h.GetMenuItems(), DevMenuActionTogglePlayerPosition).GetLabel(); label != "Player position\tUNPOWERED{OFF}" {
		t.Fatalf("label after toggle without renderer = %q", label)
	}
}

func TestDevMenuHandler_CloseItem(t *testing.T) {
	h := NewDevMenuHandler(state.NewGame())
	closeItem := &gamemenu.CloseMenuItem{Label: "Close"}
	shouldClose, help := h.OnActivate(closeItem, 11)
	if !shouldClose {
		t.Fatal("Close should close menu")
	}
	if help != "" {
		t.Fatalf("unexpected help: %q", help)
	}
}

func TestLevelSeedMenuLabel(t *testing.T) {
	g := state.NewGame()
	if got := levelSeedMenuLabel(g); got != "Load level seed" {
		t.Fatalf("label = %q", got)
	}
	g.LevelSeed = 42
	if got := levelSeedMenuLabel(g); got != "Load level seed\tACTION{2A}" {
		t.Fatalf("label = %q", got)
	}
}

func findDevMenuItem(t *testing.T, items []gamemenu.MenuItem, action DevMenuAction) *DevMenuItem {
	t.Helper()
	for _, item := range items {
		devItem, ok := item.(*DevMenuItem)
		if ok && devItem.Action == action {
			return devItem
		}
	}
	t.Fatalf("missing dev menu item for action %v", action)
	return nil
}
