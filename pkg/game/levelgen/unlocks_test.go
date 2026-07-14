package levelgen

import (
	"testing"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	"darkstation/pkg/game/unlocks"
	gameworld "darkstation/pkg/game/world"
)

func TestPlaceRoutingRepair_doesNotOverwriteExitGatingDevice(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(3, 6)
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			grid.MarkAsRoomWithName(r, c, "Lab", "desc")
		}
		for c := 3; c < 6; c++ {
			grid.MarkAsRoomWithName(r, c, "Annex", "desc")
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
	g.CurrentDeckID = 1
	g.UnlockPlan = &unlocks.Plan{
		Requirements: []unlocks.Requirement{{
			ID:           "routing-test",
			Kind:         unlocks.KindRoutingRepair,
			SourceDeckID: 1,
			TargetDeckID: 2,
			RepairID:     "routing-repair-deck3-test",
		}},
	}

	exitRepairCell := grid.GetCell(1, 3)
	exitRepair := entities.NewRepairObjective("deck2-repair1", entities.RepairPressureValve, "Lab", 1, 3)
	gameworld.GetGameData(exitRepairCell).RepairDevice = exitRepair
	g.RepairObjectives = []*entities.RepairObjective{exitRepair}

	avoid := mapset.New[*world.Cell]()
	PlaceUnlockObjectives(g, &avoid)

	routing := g.RepairByID("routing-repair-deck3-test")
	if routing == nil {
		t.Fatal("expected routing coupler repair objective")
	}
	if gameworld.GetGameData(exitRepairCell).RepairDevice != exitRepair {
		t.Fatal("routing coupler placement must not replace the exit-gating repair device")
	}
	if routing.DeviceRow == exitRepair.DeviceRow && routing.DeviceCol == exitRepair.DeviceCol {
		t.Fatal("routing coupler must not share the exit-gating repair cell")
	}
	if !setup.CanPlaceBlockingEntity(g, grid.GetCell(routing.DeviceRow, routing.DeviceCol)) {
		t.Fatal("routing coupler cell should pass blocking placement validation")
	}
}

func TestCanEnterCellAtInit_blocksRepairDevice(t *testing.T) {
	g, grid, start := makeUnlockTestGrid(t)
	repairCell := grid.GetCell(1, 2)
	gameworld.GetGameData(repairCell).RepairDevice = entities.NewRepairObjective(
		"routing", entities.RepairSignalCalibrator, "Lab", 1, 2,
	)

	ok, reason := setup.CanEnterCellAtInit(g, repairCell)
	if ok || reason != setup.MovementBlockedEntity {
		t.Fatalf("repair device cell: ok=%v reason=%q, want blocked_entity", ok, reason)
	}

	reachable := setup.InitialReachableCells(g)
	if reachable.Has(repairCell) {
		t.Fatal("init reachability should not include repair device cells")
	}
	_ = start
}

func makeUnlockTestGrid(t *testing.T) (*state.Game, *world.Grid, *world.Cell) {
	t.Helper()
	grid := world.NewGrid(3, 5)
	for r := 0; r < 3; r++ {
		for c := 0; c < 5; c++ {
			grid.MarkAsRoomWithName(r, c, "Lab", "desc")
		}
	}
	grid.SetStartCellAt(1, 0)
	grid.SetExitCellAt(1, 4)
	grid.BuildAllCellConnections()
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})
	g := state.NewGame()
	g.Grid = grid
	return g, grid, grid.StartCell()
}
