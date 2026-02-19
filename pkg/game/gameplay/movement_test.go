// Package gameplay provides core game logic for player movement and interactions.
package gameplay

import (
	"testing"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// makeMinimalGameWithGrid creates a Game with a 2x2 grid (minimal two adjacent cells):
// (0,0) and (0,1) marked as room, connected, and RoomDoorsPowered set so movement is allowed.
func makeMinimalGameWithGrid(t *testing.T) (*state.Game, *world.Cell, *world.Cell) {
	t.Helper()
	g := state.NewGame()
	grid := world.NewGrid(2, 2) // 2x2 = two adjacent walkable cells
	grid.MarkAsRoom(0, 0)
	grid.MarkAsRoom(0, 1)
	grid.BuildAllCellConnections()
	g.Grid = grid
	cellLeft := grid.GetCell(0, 0)
	cellRight := grid.GetCell(0, 1)
	if cellLeft == nil || cellRight == nil {
		t.Fatal("grid cells nil")
	}
	gameworld.InitGameData(cellLeft)
	gameworld.InitGameData(cellRight)
	// RoomDoorsPowered: cell names from Build are "row:col"
	g.RoomDoorsPowered["0:0"] = true
	g.RoomDoorsPowered["0:1"] = true
	g.RoomCCTVPowered["0:0"] = false
	g.RoomCCTVPowered["0:1"] = false
	return g, cellLeft, cellRight
}

func TestCanEnter_NilCell(t *testing.T) {
	g := state.NewGame()
	ok, _ := CanEnter(g, nil, false)
	if ok {
		t.Error("CanEnter(g, nil, false) = true, want false")
	}
}

func TestCanEnter_NonRoomCell(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 1)
	cell := grid.GetCell(0, 0)
	if cell == nil {
		t.Fatal("cell nil")
	}
	// Room not set (default false)
	ok, _ := CanEnter(g, cell, false)
	if ok {
		t.Error("CanEnter(g, non-room cell, false) = true, want false")
	}
}

func TestCanEnter_EmptyRoomCell(t *testing.T) {
	g, cellLeft, _ := makeMinimalGameWithGrid(t)
	ok, _ := CanEnter(g, cellLeft, false)
	if !ok {
		t.Error("CanEnter(g, empty room cell, false) = false, want true")
	}
}

func TestCanEnter_GeneratorBlocksMovement(t *testing.T) {
	g, _, cellRight := makeMinimalGameWithGrid(t)
	data := gameworld.GetGameData(cellRight)
	data.Generator = entities.NewGenerator("Gen", 0)
	ok, _ := CanEnter(g, cellRight, false)
	if ok {
		t.Error("CanEnter(g, cell with generator, false) = true, want false (generator blocks)")
	}
}

func TestCanEnter_FurnitureBlocksMovement(t *testing.T) {
	g, _, cellRight := makeMinimalGameWithGrid(t)
	data := gameworld.GetGameData(cellRight)
	data.Furniture = entities.NewFurniture("Desk", "A metal desk", "desk")
	ok, _ := CanEnter(g, cellRight, false)
	if ok {
		t.Error("CanEnter(g, cell with furniture, false) = true, want false (furniture blocks)")
	}
}

func TestProcessIntent_NilCurrentCellNoPanic(t *testing.T) {
	// Movement with nil CurrentCell must not panic and must not change state.
	g, cellLeft, cellRight := makeMinimalGameWithGrid(t)
	g.Grid = g.Grid // ensure grid is set
	g.CurrentCell = nil
	ProcessIntent(g, engineinput.Intent{Action: engineinput.ActionMoveEast})
	if g.CurrentCell != nil {
		t.Errorf("ProcessIntent(MoveEast) with nil CurrentCell changed CurrentCell to %v", g.CurrentCell)
	}
	// Set CurrentCell and confirm normal move still works
	g.CurrentCell = cellLeft
	ProcessIntent(g, engineinput.Intent{Action: engineinput.ActionMoveEast})
	if g.CurrentCell != cellRight {
		t.Errorf("after setting CurrentCell, MoveEast: CurrentCell = %v, want east cell", g.CurrentCell)
	}
}

func TestProcessIntent_ValidMoveUpdatesCurrentCell(t *testing.T) {
	g, cellLeft, cellRight := makeMinimalGameWithGrid(t)
	g.CurrentCell = cellLeft
	ProcessIntent(g, engineinput.Intent{Action: engineinput.ActionMoveEast})
	if g.CurrentCell != cellRight {
		t.Errorf("after MoveEast: CurrentCell = %v, want %v (east cell)", g.CurrentCell, cellRight)
	}
}

func TestProcessIntent_BlockedMoveDoesNotUpdateCurrentCell(t *testing.T) {
	g, cellLeft, _ := makeMinimalGameWithGrid(t)
	g.CurrentCell = cellLeft
	// cellLeft.East is cellRight which is enterable - so move would succeed. To test blocked:
	// use a cell that has no East neighbor (or East is wall). 1x1 grid so East is nil.
	g2 := state.NewGame()
	grid := world.NewGrid(1, 1) // 1x1 = no neighbors, move blocked
	grid.MarkAsRoom(0, 0)
	grid.BuildAllCellConnections()
	g2.Grid = grid
	c := grid.GetCell(0, 0)
	gameworld.InitGameData(c)
	g2.RoomDoorsPowered["0:0"] = true
	g2.RoomCCTVPowered["0:0"] = false
	g2.CurrentCell = c
	ProcessIntent(g2, engineinput.Intent{Action: engineinput.ActionMoveEast})
	if g2.CurrentCell != c {
		t.Errorf("blocked move (no east): CurrentCell changed to %v, want unchanged %v", g2.CurrentCell, c)
	}
}

func TestProcessIntent_AllFourDirections(t *testing.T) {
	// 3x3 grid, center (1,1), move N/S/E/W and assert CurrentCell
	g := state.NewGame()
	grid := world.NewGrid(3, 3) // 3x3 = center has all four neighbors
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			grid.MarkAsRoomWithName(r, c, "R", "desc")
		}
	}
	grid.BuildAllCellConnections()
	g.Grid = grid
	center := grid.GetCell(1, 1)
	cells := map[string]*world.Cell{
		"center": center,
		"north":  grid.GetCell(0, 1),
		"south":  grid.GetCell(2, 1),
		"east":   grid.GetCell(1, 2),
		"west":   grid.GetCell(1, 0),
	}
	for _, cell := range cells {
		gameworld.InitGameData(cell)
	}
	g.RoomDoorsPowered["R"] = true
	g.RoomCCTVPowered["R"] = false

	dirs := []struct {
		name   string
		action engineinput.Action
		want   *world.Cell
	}{
		{"North", engineinput.ActionMoveNorth, cells["north"]},
		{"South", engineinput.ActionMoveSouth, cells["south"]},
		{"East", engineinput.ActionMoveEast, cells["east"]},
		{"West", engineinput.ActionMoveWest, cells["west"]},
	}
	for _, d := range dirs {
		t.Run(d.name, func(t *testing.T) {
			g.CurrentCell = center
			ProcessIntent(g, engineinput.Intent{Action: d.action})
			if g.CurrentCell != d.want {
				t.Errorf("after Move%s: CurrentCell = %v, want %s", d.name, g.CurrentCell, d.name)
			}
		})
	}
}
