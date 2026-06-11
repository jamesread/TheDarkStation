package generator

import (
	"strings"
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/levelrand"
)

func TestDeckGridDimensions_BullCurve(t *testing.T) {
	rows2, cols2 := deckGridDimensions(2)
	rows5, cols5 := deckGridDimensions(5)
	rows9, cols9 := deckGridDimensions(9)
	if rows5 <= rows2 || cols5 <= cols2 {
		t.Fatalf("deck 5 (%dx%d) should exceed deck 2 (%dx%d)", rows5, cols5, rows2, cols2)
	}
	if rows5 < rows9 || cols5 < cols9 {
		t.Fatalf("deck 5 (%dx%d) should be >= deck 9 (%dx%d)", rows5, cols5, rows9, cols9)
	}
}

func TestShaftBounds_Centered(t *testing.T) {
	rows, cols := 40, 60
	top, left, bottom, right := ShaftBounds(rows, cols)
	if top <= 0 || bottom >= rows-1 || left <= 0 || right >= cols-1 {
		t.Fatalf("shaft not inset from perimeter: top=%d left=%d bottom=%d right=%d", top, left, bottom, right)
	}
	midRow := (top + bottom) / 2
	midCol := (left + right) / 2
	if abs(midRow-rows/2) > 2 || abs(midCol-cols/2) > 2 {
		t.Fatalf("shaft center (%d,%d) not near grid center (%d,%d)", midRow, midCol, rows/2, cols/2)
	}
}

func TestShaftBoundsForLevel_AlwaysFiveByFive(t *testing.T) {
	for _, level := range []int{1, 3, 5, 10} {
		rows, cols := deckGridDimensions(level)
		top, left, bottom, right := ShaftBoundsForLevel(rows, cols, level)
		w := right - left + 1
		h := bottom - top + 1
		if w != 5 || h != 5 {
			t.Fatalf("deck %d shaft = %dx%d, want 5x5", level, w, h)
		}
		if top <= 0 || bottom >= rows-1 || left <= 0 || right >= cols-1 {
			t.Fatalf("deck %d shaft not inset from perimeter", level)
		}
	}
}

func TestGenerate_Deck1_NoAnnexExplosion(t *testing.T) {
	grid := DefaultGenerator.Generate(1, deck.ThemeAirlock)
	names := make(map[string]int)
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name == "" || cell.Name == "Corridor" {
			return
		}
		names[cell.Name]++
	})
	if len(names) > 8 {
		t.Fatalf("deck 1 has %d distinct room names (want <=8): %v", len(names), names)
	}
	for name := range names {
		if strings.Contains(name, "Annex") {
			t.Fatalf("unexpected annex room name %q after shaft carve", name)
		}
	}
}

func TestGenerate_NoShaftSplitFarNames(t *testing.T) {
	levelrand.Seed(0x18B7D890DF002802)
	grid := DefaultGenerator.Generate(2, deck.ThemeCargoLogistics)
	if grid == nil {
		t.Fatal("Generate(2) returned nil")
	}
	for name := range roomNamesOnGrid(grid) {
		if strings.HasSuffix(name, " Far") {
			t.Fatalf("unexpected shaft-split room name %q", name)
		}
	}
}

func roomNamesOnGrid(grid *world.Grid) map[string]int {
	names := make(map[string]int)
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name == "" || cell.Name == "Corridor" || cell.Name == ShaftRoomName {
			return
		}
		names[cell.Name]++
	})
	return names
}

func TestGenerate_IncludesShaftExit(t *testing.T) {
	grid := DefaultGenerator.Generate(3, deck.ThemeAirlock)
	exit := grid.ExitCell()
	if exit == nil || !exit.ExitCell {
		t.Fatal("expected exit cell in shaft")
	}
	if exit.Name != ShaftRoomName {
		t.Fatalf("exit room = %q, want %q", exit.Name, ShaftRoomName)
	}
}
