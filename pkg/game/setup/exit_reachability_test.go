package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// Reproduces map.txt: exit corner pocket blocked by furniture west and maint terminal south.
func TestExitReachableWhenCompletable_blockedExitCorner(t *testing.T) {
	grid := world.NewGrid(10, 18)
	for r := 3; r <= 6; r++ {
		for c := 2; c <= 13; c++ {
			room := "Depressurized Crew Quarters"
			if c >= 11 {
				room = "Derelict Cryogenic Habitation Block"
			}
			grid.MarkAsRoomWithName(r, c, room, "desc")
		}
	}
	grid.SetExitCellAt(5, 4)
	grid.SetStartCellAt(3, 13)
	grid.BuildAllCellConnections()
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})
	g := state.NewGame()
	g.Grid = grid

	exitCell := grid.ExitCell()
	gameworld.GetGameData(exitCell).Furniture = entities.NewFurniture("Blocked lift pad", "desc", "F")

	if ExitReachableWhenCompletable(g, nil) {
		t.Fatal("expected exit unreachable when lift cell is blocked")
	}
	if CanPlaceBlockingEntity(g, exitCell) {
		t.Error("blocker on exit cell should fail placement check")
	}

	EnsureExitReachability(g)
	if !ExitReachableWhenCompletable(g, nil) {
		t.Fatal("EnsureExitReachability should clear blockers on exit")
	}
}

func TestCanPlaceBlockingEntity_allowsNonExitPathBlock(t *testing.T) {
	grid := world.NewGrid(5, 5)
	for r := 0; r < 5; r++ {
		for c := 0; c < 5; c++ {
			grid.MarkAsRoomWithName(r, c, "R", "desc")
		}
	}
	grid.SetExitCellAt(2, 4)
	grid.SetStartCellAt(2, 0)
	grid.BuildAllCellConnections()
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})
	g := state.NewGame()
	g.Grid = grid

	// Block (2,2) but alternate path exists via (1,*) and (3,*)
	center := grid.GetCell(2, 2)
	if !CanPlaceBlockingEntity(g, center) {
		t.Fatal("blocking center should be ok when side paths exist")
	}
}

func TestIsAdjacentToExit(t *testing.T) {
	grid := world.NewGrid(3, 3)
	grid.MarkAsRoomWithName(1, 1, "R", "desc")
	grid.SetExitCellAt(1, 2)
	grid.BuildAllCellConnections()
	g := state.NewGame()
	g.Grid = grid
	if !IsAdjacentToExit(g, grid.GetCell(1, 1)) {
		t.Error("cell west of exit should be adjacent")
	}
	if IsAdjacentToExit(g, grid.GetCell(0, 0)) {
		t.Error("far cell should not be adjacent to exit")
	}
}
