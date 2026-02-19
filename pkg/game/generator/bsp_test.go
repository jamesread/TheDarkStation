// Package generator tests BSP grid generation: named rooms, corridors, connectivity,
// deck functional layer naming, and final-deck minimal layout.
package generator

import (
	"math/rand"
	"strings"
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
)

// countReachableRoomCells returns the number of Room cells reachable from start via N/E/S/W.
func countReachableRoomCells(grid *world.Grid, start *world.Cell) int {
	if start == nil || !start.Room {
		return 0
	}
	visited := make(map[*world.Cell]bool)
	queue := []*world.Cell{start}
	visited[start] = true
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, n := range []*world.Cell{c.North, c.East, c.South, c.West} {
			if n != nil && n.Room && !visited[n] {
				visited[n] = true
				queue = append(queue, n)
			}
		}
	}
	return len(visited)
}

// countRoomCells returns the total number of cells with Room == true.
func countRoomCells(grid *world.Grid) int {
	n := 0
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.Room {
			n++
		}
	})
	return n
}

func TestBSPGenerate_HasNamedRooms(t *testing.T) {
	rand.Seed(1)
	grid := DefaultGenerator.Generate(1)
	if grid == nil {
		t.Fatal("Generate(1) returned nil")
	}
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.Room {
			if cell.Name == "" {
				t.Errorf("room cell at (%d,%d) has empty Name", row, col)
			}
			if cell.Description == "" {
				t.Errorf("room cell at (%d,%d) has empty Description", row, col)
			}
		}
	})
}

func TestBSPGenerate_HasCorridors(t *testing.T) {
	rand.Seed(2)
	grid := DefaultGenerator.Generate(1)
	if grid == nil {
		t.Fatal("Generate(1) returned nil")
	}
	corridorCount := 0
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.Room && cell.Name == "Corridor" {
			corridorCount++
		}
	})
	if corridorCount < 1 {
		t.Errorf("expected at least one Corridor cell, got %d", corridorCount)
	}
}

func TestBSPGenerate_AllRoomsReachable(t *testing.T) {
	rand.Seed(3)
	grid := DefaultGenerator.Generate(1)
	if grid == nil {
		t.Fatal("Generate(1) returned nil")
	}
	start := grid.StartCell()
	if start == nil {
		t.Fatal("StartCell is nil")
	}
	total := countRoomCells(grid)
	reachable := countReachableRoomCells(grid, start)
	if reachable != total {
		t.Errorf("reachable room cells %d != total room cells %d (isolated rooms)", reachable, total)
	}
}

func TestBSPGenerate_DeckFunctionalLayer(t *testing.T) {
	// Deck identity drives room naming: FunctionalType(level) → RoomNamesForType(ft).
	ft := deck.FunctionalType(1)
	bases, adjectives := deck.RoomNamesForType(ft)
	if len(bases) == 0 || len(adjectives) == 0 {
		t.Fatal("RoomNamesForType returned empty; deck functional layer not configured")
	}
	rand.Seed(4)
	grid := DefaultGenerator.Generate(1)
	if grid == nil {
		t.Fatal("Generate(1) returned nil")
	}
	// At least one room name must be built from deck thematic names (adjective + base).
	hasThematicName := false
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name == "" || cell.Name == "Corridor" {
			return
		}
		name := cell.Name
		for _, adj := range adjectives {
			if strings.Contains(name, adj) {
				hasThematicName = true
				return
			}
		}
		for _, base := range bases {
			if strings.Contains(name, base) {
				hasThematicName = true
				return
			}
		}
	})
	if !hasThematicName {
		t.Error("expected at least one room name from RoomNamesForType (adjective or base); deck thematic naming not present")
	}
}

// midDeckLevelForTest is used as a "mid" deck level for comparing grid size/room count to final deck.
const midDeckLevelForTest = 3

func TestBSPGenerate_StartAndExitSet(t *testing.T) {
	// Generator sets start and exit cells; start is in a room; exit is marked ExitCell.
	rand.Seed(7)
	grid := DefaultGenerator.Generate(1)
	if grid == nil {
		t.Fatal("Generate(1) returned nil")
	}
	start := grid.StartCell()
	exit := grid.ExitCell()
	if start == nil {
		t.Fatal("StartCell is nil")
	}
	if exit == nil {
		t.Fatal("ExitCell is nil")
	}
	if !start.Room {
		t.Error("StartCell.Room = false, want true (start must be in a room)")
	}
	if start.Name == "" {
		t.Error("StartCell.Name is empty, want non-empty (start room has a name)")
	}
	if !exit.ExitCell {
		t.Error("ExitCell.ExitCell = false, want true (exit must be marked)")
	}
	if !exit.Room {
		t.Error("ExitCell.Room = false, want true (exit must be walkable)")
	}
}

func TestBSPGenerate_FinalDeckMinimalLayout(t *testing.T) {
	// Final deck uses minimal layout (fewer rooms, smaller grid) per GDD.
	if !deck.IsFinalDeck(deck.TotalDecks) {
		t.Fatal("TotalDecks should be final deck level")
	}
	rand.Seed(5)
	gridFinal := DefaultGenerator.Generate(deck.TotalDecks)
	rand.Seed(6)
	gridMid := DefaultGenerator.Generate(midDeckLevelForTest)
	if gridFinal == nil || gridMid == nil {
		t.Fatal("Generate returned nil")
	}
	rowsFinal, colsFinal := gridFinal.Rows(), gridFinal.Cols()
	rowsMid, colsMid := gridMid.Rows(), gridMid.Cols()
	if rowsFinal > rowsMid || colsFinal > colsMid {
		t.Errorf("final deck grid (%d×%d) should not be larger than mid deck (%d×%d)", rowsFinal, colsFinal, rowsMid, colsMid)
	}
	roomsFinal := countRoomCells(gridFinal)
	roomsMid := countRoomCells(gridMid)
	if roomsFinal > roomsMid {
		t.Errorf("final deck should have fewer or equal room cells than mid deck: got %d vs %d", roomsFinal, roomsMid)
	}
}
