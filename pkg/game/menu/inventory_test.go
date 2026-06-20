package menu

import (
	"strings"
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
)

func TestNewInventoryMenuHandler_sections(t *testing.T) {
	g := state.NewGame()
	g.InitRunUnlocks(7)
	g.AddRunKeycard(world.NewItem("Reactor Authorization — Observatory"))
	g.OwnedItems.Put(world.NewItem("Crew Override Authorization"))
	g.Batteries = 2
	g.HasMap = true

	h := NewInventoryMenuHandler(g)
	var labels []string
	for _, item := range h.items {
		labels = append(labels, item.GetLabel())
	}

	joined := strings.Join(labels, "\n")
	if !strings.Contains(joined, "TITLE{Run Inventory}") {
		t.Fatalf("missing run inventory header: %q", joined)
	}
	if !strings.Contains(joined, "TITLE{Deck Inventory}") {
		t.Fatalf("missing deck inventory header: %q", joined)
	}
	if !strings.Contains(joined, "Reactor Authorization") {
		t.Fatal("run section should list reactor authorization")
	}
	if !strings.Contains(joined, "Crew Override Authorization") {
		t.Fatal("deck section should list crew override")
	}
	if !strings.Contains(joined, "Batteries x2") {
		t.Fatal("deck section should list batteries")
	}
	if !strings.Contains(joined, "ITEM{Map}") {
		t.Fatal("run section should list map")
	}
}

func TestNewInventoryMenuHandler_sortedByTypeThenTitle(t *testing.T) {
	g := state.NewGame()
	g.InitRunUnlocks(7)
	g.AddRunKeycard(world.NewItem("Reactor Authorization — Observatory"))
	g.AddRunKeycard(world.NewItem("Deck 5 Access Keycard"))
	g.HasMap = true
	g.OwnedItems.Put(world.NewItem("Zebra Tool"))
	g.OwnedItems.Put(world.NewItem("Alpha Widget"))
	g.Batteries = 1

	h := NewInventoryMenuHandler(g)
	var runRows, deckRows []string
	section := ""
	for _, item := range h.items {
		label := item.GetLabel()
		if strings.HasPrefix(label, "TITLE{Run Inventory}") {
			section = "run"
			continue
		}
		if strings.HasPrefix(label, "TITLE{Deck Inventory}") {
			section = "deck"
			continue
		}
		if strings.HasPrefix(label, "TITLE{") {
			continue
		}
		switch section {
		case "run":
			runRows = append(runRows, label)
		case "deck":
			deckRows = append(deckRows, label)
		}
	}

	if len(runRows) != 3 {
		t.Fatalf("run rows = %d, want 3 (2 keycards + map): %v", len(runRows), runRows)
	}
	if !strings.Contains(runRows[0], "Deck 5 Access Keycard") {
		t.Fatalf("first run row should be keycard A-first: %q", runRows[0])
	}
	if !strings.Contains(runRows[1], "Reactor Authorization") {
		t.Fatalf("second run row should be keycard: %q", runRows[1])
	}
	if !strings.Contains(runRows[2], "ITEM{Map}") {
		t.Fatalf("third run row should be map: %q", runRows[2])
	}

	if len(deckRows) != 3 {
		t.Fatalf("deck rows = %d, want 3 (battery + 2 items): %v", len(deckRows), deckRows)
	}
	if !strings.Contains(deckRows[0], "Batteries") {
		t.Fatalf("first deck row should be batteries: %q", deckRows[0])
	}
	if !strings.Contains(deckRows[1], "Alpha Widget") {
		t.Fatalf("second deck row should be Alpha Widget: %q", deckRows[1])
	}
	if !strings.Contains(deckRows[2], "Zebra Tool") {
		t.Fatalf("third deck row should be Zebra Tool: %q", deckRows[2])
	}
}
