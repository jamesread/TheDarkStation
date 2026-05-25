package gameplay

import (
	"testing"

	gamemenu "darkstation/pkg/game/menu"
	"darkstation/pkg/game/state"
)

func TestDevMenuHandler_GetMenuItems(t *testing.T) {
	h := NewDevMenuHandler(state.NewGame())
	items := h.GetMenuItems()
	if len(items) != 9 {
		t.Fatalf("expected 9 items, got %d", len(items))
	}
	if items[0].GetLabel() != "Zoom: 24px (30×15 tiles)" {
		t.Fatalf("item 0 label = %q", items[0].GetLabel())
	}
	if items[2].GetLabel() != "Dump map" {
		t.Fatalf("item 2 label = %q", items[2].GetLabel())
	}
	if items[3].GetLabel() != "Developer test map" {
		t.Fatalf("item 3 label = %q", items[3].GetLabel())
	}
	if items[4].GetLabel() != "Map area border: OFF" {
		t.Fatalf("item 4 label = %q", items[4].GetLabel())
	}
	if items[5].GetLabel() != "FOV ray lines: OFF" {
		t.Fatalf("item 5 label = %q", items[5].GetLabel())
	}
	if items[6].GetLabel() != "FPS display: ON" {
		t.Fatalf("item 6 label = %q", items[6].GetLabel())
	}
	if items[7].GetLabel() != "Player position: OFF" {
		t.Fatalf("item 7 label = %q", items[7].GetLabel())
	}
	if items[8].GetLabel() != "Close" {
		t.Fatalf("item 8 label = %q", items[8].GetLabel())
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
	if h.GetMenuItems()[4].GetLabel() != "Map area border: OFF" {
		t.Fatalf("label after toggle without renderer = %q", h.GetMenuItems()[4].GetLabel())
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
	if h.GetMenuItems()[5].GetLabel() != "FOV ray lines: OFF" {
		t.Fatalf("label after toggle without renderer = %q", h.GetMenuItems()[5].GetLabel())
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
	if h.GetMenuItems()[6].GetLabel() != "FPS display: ON" {
		t.Fatalf("label after toggle without renderer = %q", h.GetMenuItems()[6].GetLabel())
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
	if h.GetMenuItems()[7].GetLabel() != "Player position: OFF" {
		t.Fatalf("label after toggle without renderer = %q", h.GetMenuItems()[7].GetLabel())
	}
}

func TestDevMenuHandler_CloseItem(t *testing.T) {
	h := NewDevMenuHandler(state.NewGame())
	closeItem := &gamemenu.CloseMenuItem{Label: "Close"}
	shouldClose, help := h.OnActivate(closeItem, 8)
	if !shouldClose {
		t.Fatal("Close should close menu")
	}
	if help != "" {
		t.Fatalf("unexpected help: %q", help)
	}
}
