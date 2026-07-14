package levelgen

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	"darkstation/pkg/game/unlocks"
	gameworld "darkstation/pkg/game/world"
)

func TestSyncKeycardPayoffRegistration_rebindsAfterConduitFault(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(3, 6)
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			grid.MarkAsRoomWithName(r, c, "Lab", "desc")
		}
		for c := 3; c < 6; c++ {
			grid.MarkAsRoomWithName(r, c, "Corridor", "desc")
		}
	}
	grid.SetStartCellAt(1, 0)
	grid.SetExitCellAt(1, 5)
	grid.BuildAllCellConnections()
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})
	g.Grid = grid
	setup.InitRoomPower(g)
	g.CurrentDeckID = 2
	g.UnlockPlan = &unlocks.Plan{
		Requirements: []unlocks.Requirement{{
			ID:           "reactor-auth-2",
			Kind:         unlocks.KindSecurityKeycard,
			SourceDeckID: 2,
			TargetDeckID: 4,
			KeycardName:  "Reactor Authorization — Test",
		}},
	}

	exitRepairCell := grid.GetCell(1, 1)
	exitRepair := entities.NewRepairObjective("deck3-repair1", entities.RepairPowerCoupler, "Lab", 1, 1)
	gameworld.GetGameData(exitRepairCell).RepairDevice = exitRepair
	gameworld.GetGameData(exitRepairCell).PendingUnlockKeycard = "Reactor Authorization — Test"
	g.RepairObjectives = []*entities.RepairObjective{exitRepair}

	conduitCell := grid.GetCell(1, 4)
	conduit := entities.NewRepairObjective("deck3-conduit1", entities.RepairConduitSplice, "Corridor", 1, 4)
	gameworld.GetGameData(conduitCell).RepairDevice = conduit
	g.RepairObjectives = append(g.RepairObjectives, conduit)

	SyncKeycardPayoffRegistration(g)

	if gameworld.GetGameData(exitRepairCell).PendingUnlockKeycard != "" {
		t.Fatal("pending keycard should move off earlier final repair")
	}
	if gameworld.GetGameData(conduitCell).PendingUnlockKeycard != "Reactor Authorization — Test" {
		t.Fatalf("pending keycard should bind to current final repair, got %q",
			gameworld.GetGameData(conduitCell).PendingUnlockKeycard)
	}

	drop := setup.KeycardDropCell(g, conduitCell)
	if drop != conduitCell {
		t.Fatalf("conduit splice cell should be walkable drop target, got (%d,%d)", drop.Row, drop.Col)
	}
}
