package state

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	gameworld "darkstation/pkg/game/world"
)

func TestPreviewShortOutIfOverload_matchesApply(t *testing.T) {
	g := NewGame()
	grid := world.NewGrid(6, 2)
	for r := 0; r < 3; r++ {
		for c := 0; c < 2; c++ {
			grid.MarkAsRoomWithName(r, c, "RoomA", "")
		}
	}
	for r := 3; r < 6; r++ {
		for c := 0; c < 2; c++ {
			grid.MarkAsRoomWithName(r, c, "RoomB", "")
		}
	}
	grid.BuildAllCellConnections()
	g.Grid = grid
	for r := 0; r < 6; r++ {
		for c := 0; c < 2; c++ {
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}
	gameworld.GetGameData(grid.GetCell(0, 0)).Door = entities.NewDoor("RoomA")
	gameworld.GetGameData(grid.GetCell(3, 0)).Door = entities.NewDoor("RoomB")

	g.PowerSupply = 10
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": false}
	g.RoomCCTVPowered = map[string]bool{}
	g.PowerConsumption = g.CalculatePowerConsumption()

	preview := g.PreviewShortOutIfOverload("RoomB", true, false)
	g.RoomDoorsPowered["RoomB"] = true
	shorted := g.ShortOutIfOverload("RoomB")

	if shorted && len(preview) == 0 {
		t.Fatal("preview should list shed when short-out occurs")
	}
	if !shorted {
		t.Fatal("expected short-out with limited supply")
	}
	if !g.RoomDoorsPowered["RoomB"] {
		t.Fatal("protected room should stay powered")
	}
	if g.RoomDoorsPowered["RoomA"] {
		t.Fatal("RoomA should have been shed")
	}
}

func TestPreviewShortOutIfOverload_noShedWhenWithinBudget(t *testing.T) {
	g := NewGame()
	g.PowerSupply = 500
	g.RoomDoorsPowered = map[string]bool{"RoomA": false}
	g.RoomCCTVPowered = map[string]bool{}
	shed := g.PreviewShortOutIfOverload("RoomA", true, false)
	if len(shed) != 0 {
		t.Fatalf("expected no shed, got %v", shed)
	}
}
