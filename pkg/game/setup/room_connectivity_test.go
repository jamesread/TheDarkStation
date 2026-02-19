package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// TestRoomStillConnectedIfBlock_TwoDoorwaysBlockChokepoint verifies that blocking the only
// path between two doorways returns false (room would be disconnected). R8 / I7.
func TestRoomStillConnectedIfBlock_TwoDoorwaysBlockChokepoint(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(2, 3) // 2 rows, 3 cols
	// Row 0: corridor at (0,0) and (0,2); (0,1) can be wall or corridor
	grid.MarkAsRoomWithName(0, 0, "Corridor", "corridor")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "corridor")
	grid.MarkAsRoomWithName(0, 2, "Corridor", "corridor")
	// Row 1: room "R" at (1,0), (1,1), (1,2) — doorways (1,0) and (1,2), chokepoint (1,1)
	grid.MarkAsRoomWithName(1, 0, "R", "room")
	grid.MarkAsRoomWithName(1, 1, "R", "room")
	grid.MarkAsRoomWithName(1, 2, "R", "room")
	grid.BuildAllCellConnections()
	g.Grid = grid
	for _, cell := range []*world.Cell{grid.GetCell(0, 0), grid.GetCell(0, 1), grid.GetCell(0, 2), grid.GetCell(1, 0), grid.GetCell(1, 1), grid.GetCell(1, 2)} {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	}
	entryCells := []*world.Cell{grid.GetCell(0, 0), grid.GetCell(0, 2)} // corridor cells adjacent to room R
	chokepoint := grid.GetCell(1, 1)
	if chokepoint == nil {
		t.Fatal("chokepoint cell nil")
	}
	got := RoomStillConnectedIfBlock(g, "R", entryCells, chokepoint)
	if got {
		t.Errorf("RoomStillConnectedIfBlock(..., chokepoint (1,1)) = true, want false (blocking only path between doorways)")
	}
}

// TestRoomStillConnectedIfBlock_TwoDoorwaysBlockNonChokepoint verifies that blocking a cell
// that is not the only path between doorways returns true (room stays connected).
func TestRoomStillConnectedIfBlock_TwoDoorwaysBlockNonChokepoint(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(3, 2) // 3 rows, 2 cols
	// Row 0: corridor
	grid.MarkAsRoomWithName(0, 0, "Corridor", "corridor")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "corridor")
	// Rows 1–2: room "R" — doorways (1,0),(1,1); (2,0) is not the only path between them
	grid.MarkAsRoomWithName(1, 0, "R", "room")
	grid.MarkAsRoomWithName(1, 1, "R", "room")
	grid.MarkAsRoomWithName(2, 0, "R", "room")
	grid.MarkAsRoomWithName(2, 1, "R", "room")
	grid.BuildAllCellConnections()
	g.Grid = grid
	for r := 0; r < 3; r++ {
		for c := 0; c < 2; c++ {
			if cell := grid.GetCell(r, c); cell != nil {
				gameworld.InitGameData(cell)
			}
		}
	}
	entryCells := []*world.Cell{grid.GetCell(0, 0), grid.GetCell(0, 1)}
	blockCell := grid.GetCell(2, 0) // not on the only path between (1,0) and (1,1)
	if blockCell == nil {
		t.Fatal("block cell nil")
	}
	got := RoomStillConnectedIfBlock(g, "R", entryCells, blockCell)
	if !got {
		t.Errorf("RoomStillConnectedIfBlock(..., (2,0)) = false, want true (room stays connected)")
	}
}

// TestRoomStillConnectedIfBlock_EmptyEntryCells returns true (no doorways to satisfy).
func TestRoomStillConnectedIfBlock_EmptyEntryCells(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 1)
	grid.MarkAsRoomWithName(0, 0, "R", "room")
	grid.BuildAllCellConnections()
	g.Grid = grid
	gameworld.InitGameData(grid.GetCell(0, 0))
	got := RoomStillConnectedIfBlock(g, "R", nil, nil)
	if !got {
		t.Errorf("RoomStillConnectedIfBlock(..., nil entryCells) = false, want true")
	}
	got = RoomStillConnectedIfBlock(g, "R", []*world.Cell{}, nil)
	if !got {
		t.Errorf("RoomStillConnectedIfBlock(..., empty entryCells) = false, want true")
	}
}
