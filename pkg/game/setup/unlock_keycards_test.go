package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestKeycardDropCell_adjacentWhenRepairDeviceBlocks(t *testing.T) {
	grid := world.NewGrid(3, 4)
	for r := 0; r < 3; r++ {
		for c := 0; c < 4; c++ {
			grid.MarkAsRoomWithName(r, c, "Lab", "desc")
		}
	}
	grid.SetStartCellAt(1, 0)
	grid.SetExitCellAt(1, 3)
	grid.BuildAllCellConnections()
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})

	deviceCell := grid.GetCell(1, 2)
	adjCell := grid.GetCell(1, 1)
	repair := entities.NewRepairObjective("deck1-repair1", entities.RepairPowerCoupler, "Lab", 1, 2)
	gameworld.GetGameData(deviceCell).RepairDevice = repair

	g := state.NewGame()
	g.Grid = grid

	drop := KeycardDropCell(g, deviceCell)
	if drop != adjCell {
		t.Fatalf("KeycardDropCell = (%d,%d), want adjacent walkable (%d,%d)",
			drop.Row, drop.Col, adjCell.Row, adjCell.Col)
	}
}

func TestDropPendingUnlockKeycards_usesAdjacentCell(t *testing.T) {
	grid := world.NewGrid(3, 4)
	for r := 0; r < 3; r++ {
		for c := 0; c < 4; c++ {
			grid.MarkAsRoomWithName(r, c, "Lab", "desc")
		}
	}
	grid.SetStartCellAt(1, 0)
	grid.SetExitCellAt(1, 3)
	grid.BuildAllCellConnections()
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})

	deviceCell := grid.GetCell(1, 2)
	adjCell := grid.GetCell(1, 1)
	repair := entities.NewRepairObjective("deck1-repair1", entities.RepairPowerCoupler, "Lab", 1, 2)
	gameworld.GetGameData(deviceCell).RepairDevice = repair
	gameworld.GetGameData(deviceCell).PendingUnlockKeycard = "Reactor Authorization — Test"

	g := state.NewGame()
	g.Grid = grid
	DropPendingUnlockKeycards(g)

	found := false
	adjCell.ItemsOnFloor.Each(func(item *world.Item) {
		if item != nil && item.Name == "Reactor Authorization — Test" {
			found = true
		}
	})
	if !found {
		t.Fatal("expected keycard on adjacent walkable cell")
	}
	if gameworld.GetGameData(deviceCell).PendingUnlockKeycard != "" {
		t.Fatal("pending keycard should be cleared from repair cell")
	}
}

func TestEnsureExitGatingRepairReachability_movesPendingKeycard(t *testing.T) {
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
	gameworld.GetGameData(repairCell).PendingUnlockKeycard = "Reactor Authorization — Test"
	g.RepairObjectives = []*entities.RepairObjective{repair}

	EnsureExitGatingRepairReachability(g)

	newCell := g.Grid.GetCell(repair.DeviceRow, repair.DeviceCol)
	if gameworld.GetGameData(newCell).PendingUnlockKeycard != "Reactor Authorization — Test" {
		t.Fatalf("pending keycard not moved with repair; got %q on relocated cell",
			gameworld.GetGameData(newCell).PendingUnlockKeycard)
	}
	if gameworld.GetGameData(repairCell).PendingUnlockKeycard != "" {
		t.Fatal("pending keycard should be cleared from old repair cell")
	}
}
