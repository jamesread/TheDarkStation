package menu

import (
	"testing"

	engineinput "darkstation/pkg/engine/input"
)

func TestSettingsMenuHandler_tabsSwitch(t *testing.T) {
	h := NewSettingsMenuHandler(true)
	items := h.GetMenuItems()
	if SettingsTabStripLength(items) != 3 {
		t.Fatalf("tab strip = %d, want 3", SettingsTabStripLength(items))
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
	if _, ok := items[3].(*WindowModeMenuItem); !ok {
		t.Fatalf("first video content item = %T, want WindowModeMenuItem", items[3])
	}

	newSel, consumed, msg = h.TryHorizontalTabNav(items, 1, engineinput.Intent{Action: engineinput.ActionMoveEast})
	if !consumed || newSel != 2 || msg != "" {
		t.Fatalf("east from video: sel=%d consumed=%v msg=%q", newSel, consumed, msg)
	}
	if h.tab != SettingsTabAudio {
		t.Fatalf("tab = %v, want Audio", h.tab)
	}
	items = h.GetMenuItems()
	if _, ok := items[3].(*SFXVolumeMenuItem); !ok {
		t.Fatalf("first audio content item = %T, want SFXVolumeMenuItem", items[3])
	}
	if _, ok := items[4].(*MusicVolumeMenuItem); !ok {
		t.Fatalf("second audio content item = %T, want MusicVolumeMenuItem", items[4])
	}
	if _, ok := items[5].(*MuteMenuItem); !ok {
		t.Fatalf("third audio content item = %T, want MuteMenuItem", items[5])
	}

	newSel, consumed, msg = h.TryHorizontalTabNav(items, 2, engineinput.Intent{Action: engineinput.ActionMoveWest})
	if !consumed || newSel != 1 || msg != "" {
		t.Fatalf("west from audio: sel=%d consumed=%v msg=%q", newSel, consumed, msg)
	}
	newSel, consumed, msg = h.TryHorizontalTabNav(h.GetMenuItems(), 1, engineinput.Intent{Action: engineinput.ActionMoveWest})
	if !consumed || newSel != 0 || msg != "" {
		t.Fatalf("west from video: sel=%d consumed=%v msg=%q", newSel, consumed, msg)
	}
	if _, ok := h.GetMenuItems()[3].(*BindingHeaderItem); !ok {
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
	if !consumed || newSel != 3 {
		t.Fatalf("down from video tab: sel=%d consumed=%v, want first content index 3", newSel, consumed)
	}

	newSel, consumed = h.TryVerticalTabNav(items, 3, engineinput.Intent{Action: engineinput.ActionMoveNorth})
	if !consumed || newSel != 1 {
		t.Fatalf("up from video content: sel=%d consumed=%v, want video tab index 1", newSel, consumed)
	}

	newSel, consumed = h.TryVerticalTabNav(items, 1, engineinput.Intent{Action: engineinput.ActionMoveNorth})
	last := len(items) - 1
	if !consumed || newSel != last {
		t.Fatalf("up from video tab should wrap to back: sel=%d want %d", newSel, last)
	}

	h.tab = SettingsTabAudio
	items = h.GetMenuItems()
	newSel, consumed = h.TryVerticalTabNav(items, 2, engineinput.Intent{Action: engineinput.ActionMoveSouth})
	if !consumed || newSel != 3 {
		t.Fatalf("down from audio tab: sel=%d consumed=%v, want volume row", newSel, consumed)
	}
	if _, ok := items[newSel].(*SFXVolumeMenuItem); !ok {
		t.Fatalf("audio content = %T, want SFXVolumeMenuItem", items[newSel])
	}
}

func TestSettingsMenuHandler_initialSelectionMatchesTab(t *testing.T) {
	h := NewSettingsMenuHandlerWithTab(true, SettingsTabVideo)
	items := h.GetMenuItems()
	if got := h.InitialMenuSelection(items); got != 1 {
		t.Fatalf("initial selection = %d, want 1 (Video tab)", got)
	}

	h = NewSettingsMenuHandlerWithTab(true, SettingsTabAudio)
	items = h.GetMenuItems()
	if got := h.InitialMenuSelection(items); got != 2 {
		t.Fatalf("initial selection = %d, want 2 (Audio tab)", got)
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
