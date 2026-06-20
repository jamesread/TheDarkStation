package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func powerCouplerTestGame(t *testing.T) (*state.Game, *world.Cell, *entities.RepairObjective) {
	t.Helper()
	g := state.NewGame()
	grid := world.NewGrid(1, 2)
	roomName := "CouplerRoom"
	grid.MarkAsRoomWithName(0, 0, roomName, "")
	grid.MarkAsRoomWithName(0, 1, roomName, "")
	grid.BuildAllCellConnections()
	grid.SetStartCellAt(0, 0)
	g.Grid = grid
	g.CurrentCell = grid.GetCell(0, 0)

	repair := entities.NewRepairObjective("coupler", entities.RepairPowerCoupler, roomName, 0, 1)
	gameworld.GetGameData(grid.GetCell(0, 1)).RepairDevice = repair
	g.RepairObjectives = []*entities.RepairObjective{repair}

	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.AddGenerator(gen)
	g.RoomDoorsPowered = map[string]bool{roomName: true}
	setup.PropagateRoomPowerOnlineFromGenerators(g)
	setup.ApplyGridConductivePower(g)
	if !setup.CellHasLivePower(g, grid.GetCell(0, 1)) {
		t.Fatal("coupler repair cell should have live power for test setup")
	}
	return g, grid.GetCell(0, 1), repair
}

func TestTryPowerCouplerCrank_startsProgressBarSession(t *testing.T) {
	g, cell, _ := powerCouplerTestGame(t)
	if !TryPowerCouplerCrank(g, cell, gameworld.GetGameData(cell).RepairDevice) {
		t.Fatal("expected crank to start")
	}
	if !IsCouplerCrankActive(g) {
		t.Fatal("coupler crank session should be active")
	}
	if g.LongUse.AccumulatedMs <= 0 {
		t.Fatal("first USE should pump crank progress")
	}
}

func TestCouplerCrank_drainsWithoutRapidUse(t *testing.T) {
	g, cell, repair := powerCouplerTestGame(t)
	TryPowerCouplerCrank(g, cell, repair)
	start := g.LongUse.AccumulatedMs
	AdvanceCouplerCrankIfActive(g, g.LongUse.LastAdvanceMs+800)
	if IsCouplerCrankActive(g) && g.LongUse.AccumulatedMs >= start {
		t.Fatalf("progress should drain over time: before=%d after=%d", start, g.LongUse.AccumulatedMs)
	}
	if repair.IsComplete() {
		t.Fatal("drain alone should not complete coupler")
	}
}

func TestCouplerCrank_completesWithRapidUse(t *testing.T) {
	g, cell, repair := powerCouplerTestGame(t)
	TryPowerCouplerCrank(g, cell, repair)
	now := g.LongUse.LastAdvanceMs
	for i := 0; i < 5; i++ {
		TryPowerCouplerCrank(g, cell, repair)
		now += 80
		AdvanceCouplerCrankIfActive(g, now)
	}
	if !repair.IsComplete() {
		t.Fatal("rapid USE should fill crank bar and complete coupler")
	}
	if IsCouplerCrankActive(g) {
		t.Fatal("session should clear after completion")
	}
}

func TestCouplerCrank_abandonOnMoveDrainsToZero(t *testing.T) {
	g, cell, repair := powerCouplerTestGame(t)
	TryPowerCouplerCrank(g, cell, repair)
	if g.LongUse.AccumulatedMs <= 0 {
		t.Fatal("expected progress after first crank")
	}
	AbandonCouplerCrank(g)
	if !g.LongUse.Abandoning {
		t.Fatal("expected abandoning flag")
	}
	now := g.LongUse.LastAdvanceMs
	AdvanceCouplerCrankIfActive(g, now+600)
	if IsCouplerCrankActive(g) {
		t.Fatal("abandoned crank should clear after draining to zero")
	}
	if repair.IsComplete() {
		t.Fatal("abandoned crank should not complete repair")
	}
}

func TestIsHoldLongUseActive_excludesCouplerCrank(t *testing.T) {
	g, cell, repair := powerCouplerTestGame(t)
	if IsHoldLongUseActive(g) {
		t.Fatal("no session should be active")
	}
	TryPowerCouplerCrank(g, cell, repair)
	if !IsCouplerCrankActive(g) {
		t.Fatal("expected coupler crank session")
	}
	if IsHoldLongUseActive(g) {
		t.Fatal("coupler crank should not count as blocking hold long use")
	}
}

func TestCouplerCrank_idleDrainClearsEmptyBar(t *testing.T) {
	g, cell, repair := powerCouplerTestGame(t)
	TryPowerCouplerCrank(g, cell, repair)
	now := g.LongUse.LastAdvanceMs
	AdvanceCouplerCrankIfActive(g, now+2500)
	if IsCouplerCrankActive(g) {
		t.Fatal("crank session should end when progress drains to zero")
	}
	if repair.IsComplete() {
		t.Fatal("idle drain should not complete repair")
	}
}

func TestCouplerCrank_slowUseDoesNotComplete(t *testing.T) {
	g, cell, repair := powerCouplerTestGame(t)
	TryPowerCouplerCrank(g, cell, repair)
	now := g.LongUse.LastAdvanceMs
	for i := 0; i < 5; i++ {
		TryPowerCouplerCrank(g, cell, repair)
		now += 600
		AdvanceCouplerCrankIfActive(g, now)
	}
	if repair.IsComplete() {
		t.Fatal("slow USE should not complete coupler before bar drains away")
	}
}
