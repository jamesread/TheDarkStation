package menu

import (
	"testing"

	engineinput "darkstation/pkg/engine/input"
)

func TestSettingsMenuHandler_tabsSwitch(t *testing.T) {
	h := NewSettingsMenuHandler(true)
	items := h.GetMenuItems()
	if SettingsTabStripLength(items) != 2 {
		t.Fatalf("tab strip = %d, want 2", SettingsTabStripLength(items))
	}

	newSel, consumed, msg := h.TryHorizontalTabNav(items, 0, engineinput.Intent{Action: engineinput.ActionMoveEast})
	if !consumed || newSel != 1 || msg != "" {
		t.Fatalf("east from bindings: sel=%d consumed=%v msg=%q", newSel, consumed, msg)
	}
	if h.tab != SettingsTabVideo {
		t.Fatalf("tab = %v, want Video", h.tab)
	}
	items = h.GetMenuItems()
	if len(items) < 4 {
		t.Fatalf("video tab items = %d, want tab strip + window mode + back", len(items))
	}
	if _, ok := items[2].(*WindowModeMenuItem); !ok {
		t.Fatalf("first content item = %T, want WindowModeMenuItem", items[2])
	}

	newSel, consumed, msg = h.TryHorizontalTabNav(items, 1, engineinput.Intent{Action: engineinput.ActionMoveWest})
	if !consumed || newSel != 0 || msg != "" {
		t.Fatalf("west from video: sel=%d consumed=%v msg=%q", newSel, consumed, msg)
	}
	if _, ok := h.GetMenuItems()[2].(*BindingHeaderItem); !ok {
		t.Fatal("bindings tab should show binding column header")
	}
}

func TestSettingsMenuHandler_tabStripVerticalNav(t *testing.T) {
	h := NewSettingsMenuHandler(true)
	items := h.GetMenuItems()

	newSel, consumed := h.TryVerticalTabNav(items, 0, engineinput.Intent{Action: engineinput.ActionMoveSouth})
	if !consumed {
		t.Fatal("down from bindings tab should be consumed")
	}
	if _, ok := items[newSel].(*BindingMenuItem); !ok {
		t.Fatalf("down from tab = index %d (%T), want first binding row", newSel, items[newSel])
	}

	h.tab = SettingsTabVideo
	items = h.GetMenuItems()
	newSel, consumed = h.TryVerticalTabNav(items, 1, engineinput.Intent{Action: engineinput.ActionMoveSouth})
	if !consumed || newSel != 2 {
		t.Fatalf("down from video tab: sel=%d consumed=%v", newSel, consumed)
	}

	newSel, consumed = h.TryVerticalTabNav(items, 2, engineinput.Intent{Action: engineinput.ActionMoveNorth})
	if !consumed || newSel != 1 {
		t.Fatalf("up from video content: sel=%d consumed=%v, want video tab index 1", newSel, consumed)
	}

	newSel, consumed = h.TryVerticalTabNav(items, 1, engineinput.Intent{Action: engineinput.ActionMoveNorth})
	last := len(items) - 1
	if !consumed || newSel != last {
		t.Fatalf("up from video tab should wrap to back: sel=%d want %d", newSel, last)
	}
}

func TestSettingsMenuHandler_initialSelectionMatchesTab(t *testing.T) {
	h := NewSettingsMenuHandlerWithTab(true, SettingsTabVideo)
	items := h.GetMenuItems()
	if got := h.InitialMenuSelection(items); got != 1 {
		t.Fatalf("initial selection = %d, want 1 (Video tab)", got)
	}
}

func TestSettingsMenuHandler_fromMainMenuHasBack(t *testing.T) {
	h := NewSettingsMenuHandler(true)
	items := h.GetMenuItems()
	last := items[len(items)-1]
	if _, ok := last.(*BackMenuItem); !ok {
		t.Fatalf("last item = %T, want BackMenuItem", last)
	}
}

func TestSettingsMenuHandler_inGameHasCloseBack(t *testing.T) {
	h := NewSettingsMenuHandler(false)
	items := h.GetMenuItems()
	last := items[len(items)-1]
	if close, ok := last.(*CloseMenuItem); !ok || close.Label != "Back" {
		t.Fatalf("last item = %v, want CloseMenuItem Back", last)
	}
}
