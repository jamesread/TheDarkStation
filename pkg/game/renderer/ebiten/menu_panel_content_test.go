package ebiten

import "testing"

func TestMenuLabelsUseTabColumns(t *testing.T) {
	if menuLabelsUseTabColumns(nil) {
		t.Fatal("nil labels should not use tab columns")
	}
	if menuLabelsUseTabColumns([]string{"Generate", "Settings", "Quit"}) {
		t.Fatal("plain labels should not use tab columns")
	}
	// Titles that previously needed an explicit whitelist — any tabbed label enables columns.
	for _, labels := range [][]string{
		{"SFX Volume\tACTION{100%}\tSUBTLE{< left/right >}"},
		{"Single Player Puzzle\tSUBTLE{10 decks}"},
		{"Zoom\tSUBTLE{24px (30×15 tiles)}"},
		{"TRACE\tPOWERED{BUS OK}\tsegment energized"},
		{"Action\tKeyboard\tController"},
	} {
		if !menuLabelsUseTabColumns(labels) {
			t.Fatalf("expected tab columns for %q", labels[0])
		}
	}
}

func TestMenuTabColumnGap(t *testing.T) {
	if got := menuTabColumnGap("Settings"); got != 32 {
		t.Fatalf("Settings gap = %d, want 32", got)
	}
	if got := menuTabColumnGap("Audio"); got != 32 {
		t.Fatalf("Audio gap = %d, want 32", got)
	}
	if got := menuTabColumnGap("Maintenance Terminal — Habitation"); got != 8 {
		t.Fatalf("maintenance gap = %d, want 8", got)
	}
}
