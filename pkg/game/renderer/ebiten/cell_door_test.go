package ebiten

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestGetCellRenderOptions_doorManualRelease_isYellow(t *testing.T) {
	e := &EbitenRenderer{}
	g := state.NewGame()
	grid := world.NewGrid(1, 2)
	grid.MarkAsRoomWithName(0, 0, "RoomA", "")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "")
	grid.BuildAllCellConnections()
	g.Grid = grid
	g.HasMap = true
	g.CurrentCell = grid.GetCell(0, 0)

	doorCell := grid.GetCell(0, 1)
	doorCell.Discovered = true
	gameworld.InitGameData(doorCell)
	gameworld.GetGameData(doorCell).Door = &entities.Door{RoomName: "RoomB", Locked: false}
	g.ManualEgressReleased["RoomB"] = true

	snap := &renderSnapshot{playerRow: 0, playerCol: 0}
	opts := e.getCellRenderOptions(g, doorCell, snap, false)
	if opts.Color != colorDoorLocked {
		t.Fatalf("manual release door color = %v, want yellow %v", opts.Color, colorDoorLocked)
	}
	if opts.Color == colorHazard {
		t.Fatal("manual release door should not use unpowered red")
	}
}

func TestGetCellRenderOptions_unpoweredDoor_isRed(t *testing.T) {
	e := &EbitenRenderer{}
	g := state.NewGame()
	grid := world.NewGrid(1, 2)
	grid.MarkAsRoomWithName(0, 0, "RoomA", "")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "")
	grid.BuildAllCellConnections()
	g.Grid = grid
	g.HasMap = true
	g.CurrentCell = grid.GetCell(0, 0)

	doorCell := grid.GetCell(0, 1)
	doorCell.Discovered = true
	gameworld.InitGameData(doorCell)
	gameworld.GetGameData(doorCell).Door = &entities.Door{RoomName: "RoomB", Locked: false}

	snap := &renderSnapshot{playerRow: 0, playerCol: 0}
	opts := e.getCellRenderOptions(g, doorCell, snap, false)
	if opts.Color != colorHazard {
		t.Fatalf("unpowered door color = %v, want red %v", opts.Color, colorHazard)
	}
}
