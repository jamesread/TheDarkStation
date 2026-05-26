package gameplay

import (
	"testing"

	engworld "darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestUnpoweredDoorSightBlocker_blocksFOVBeyondDoor(t *testing.T) {
	grid := engworld.NewGrid(11, 11)
	for c := 3; c <= 7; c++ {
		cell := grid.GetCell(5, c)
		if cell != nil {
			cell.Room = true
			cell.Name = "Room"
		}
	}
	grid.BuildAllCellConnections()
	doorCell := grid.GetCell(5, 5)
	gameworld.GetGameData(doorCell).Door = &entities.Door{RoomName: "Beyond", Locked: false}

	g := state.NewGame()
	g.RoomDoorsPowered = map[string]bool{"Beyond": false}

	center := grid.GetCell(5, 3)
	center.Room = true
	visible := engworld.VisibleCellSet(grid, center, unpoweredDoorSightBlocker(g))

	if !visible[doorCell] {
		t.Fatal("unpowered door cell should still be visible")
	}
	if visible[grid.GetCell(5, 6)] {
		t.Fatal("cells beyond unpowered door should not be visible")
	}
}

func TestUnpoweredDoorSightBlocker_poweredDoorAllowsSight(t *testing.T) {
	grid := engworld.NewGrid(11, 11)
	for c := 3; c <= 7; c++ {
		cell := grid.GetCell(5, c)
		if cell != nil {
			cell.Room = true
			cell.Name = "Room"
		}
	}
	grid.BuildAllCellConnections()
	gameworld.GetGameData(grid.GetCell(5, 5)).Door = &entities.Door{RoomName: "Beyond", Locked: false}

	g := state.NewGame()
	g.Grid = grid
	g.RoomDoorsPowered = map[string]bool{"Beyond": true}
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(5, 7)).Generator = gen
	setup.PropagateRoomPowerOnlineFromGenerators(g)

	center := grid.GetCell(5, 3)
	center.Room = true
	visible := engworld.VisibleCellSet(grid, center, unpoweredDoorSightBlocker(g))

	if !visible[grid.GetCell(5, 7)] {
		t.Fatal("cells beyond powered door should be visible along corridor")
	}
}
