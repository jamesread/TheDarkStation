package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestEnsureExitGatingRepairReachability_relocatesInaccessibleRepair(t *testing.T) {
	grid := world.NewGrid(3, 5)
	for c := 0; c <= 2; c++ {
		grid.MarkAsRoomWithName(0, c, "Start", "desc")
		grid.MarkAsRoomWithName(1, c, "Start", "desc")
	}
	grid.MarkAsRoomWithName(1, 3, "Corridor", "desc")
	grid.MarkAsRoomWithName(0, 4, "Blocked", "desc")
	grid.MarkAsRoomWithName(1, 4, "Blocked", "desc")
	grid.SetStartCellAt(1, 2)
	grid.SetExitCellAt(1, 1)
	grid.BuildAllCellConnections()
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})
	doorCell := grid.GetCell(1, 3)
	gameworld.GetGameData(doorCell).Door = &entities.Door{RoomName: "Blocked", Locked: false}

	g := state.NewGame()
	g.Grid = grid
	InitRoomPower(g)

	repairCell := grid.GetCell(1, 4)
	repair := entities.NewRepairObjective("test-repair", entities.RepairPressureValve, "Blocked", 1, 4)
	gameworld.GetGameData(repairCell).RepairDevice = repair
	g.RepairObjectives = []*entities.RepairObjective{repair}

	if ExitGatingRepairRoomAccessible(g, "Blocked") {
		t.Fatal("precondition: Blocked room should not be accessible from lift entry")
	}

	EnsureExitGatingRepairReachability(g)

	if repair.RoomName == "Blocked" {
		t.Fatalf("repair should relocate out of Blocked, still in %q", repair.RoomName)
	}
	if !ExitGatingRepairRoomAccessible(g, repair.RoomName) {
		t.Fatalf("relocated repair room %q still not accessible", repair.RoomName)
	}
}
