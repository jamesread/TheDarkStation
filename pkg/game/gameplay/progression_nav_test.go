package gameplay

import (
	"testing"

	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/levelseed"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
)

// Regression for map.txt seed 18B7DC98AB6116AE (deck 2): routing couplers stacked in
// Sealed Potable Feed blocked Pressure Valve, circuit breaker, and bootstrap doors.
// Couplers may share a room on small decks (fewer spare rooms than sourced couplers),
// so the assertions are: repair rooms accessible-or-powerable from the lift entry,
// every coupler retains adjacent stand space, and clean solvability analysis.
func TestProgressionNavAccess_mapTxtSeed(t *testing.T) {
	levelSeed, err := levelseed.Parse("18B7DC98AB6116AE")
	if err != nil {
		t.Fatal(err)
	}
	g := stateNewGameForDeck(t, levelSeed, 2)

	finalRepair := finalExitGatingRepair(g)
	if finalRepair == nil {
		t.Fatal("expected exit-gating repair on deck 2")
	}
	repairCell := g.Grid.GetCell(finalRepair.DeviceRow, finalRepair.DeviceCol)
	if repairCell == nil {
		t.Fatalf("repair device cell missing at row=%d col=%d", finalRepair.DeviceRow, finalRepair.DeviceCol)
	}
	if !setup.ExitGatingRepairRoomAccessible(g, repairCell.Name) {
		t.Fatalf("exit-gating repair %q at x:%d y:%d in %q must be reachable or powerable from lift entry",
			finalRepair.Name, repairCell.Col, repairCell.Row, repairCell.Name)
	}

	report := setup.AnalyzeSolvability(g)
	for _, w := range report.Warnings {
		t.Errorf("unexpected solvability warning: %s", w)
	}

	if !setup.BlockingPlacementPreservesNavAccess(g, nil) {
		t.Error("a blocking entity has no adjacent stand cell after setup")
	}
	for _, repair := range g.RepairObjectives {
		if repair == nil || !repair.SkipExitGate || repair.DeviceRow < 0 {
			continue
		}
		cell := g.Grid.GetCell(repair.DeviceRow, repair.DeviceCol)
		if cell == nil {
			t.Errorf("coupler %q device cell missing at x:%d y:%d", repair.Name, repair.DeviceCol, repair.DeviceRow)
			continue
		}
		if !setup.ExitGatingRepairRoomAccessible(g, cell.Name) {
			t.Errorf("coupler %q room %q not reachable or powerable from lift entry", repair.Name, cell.Name)
		}
		if !setup.EntityHasAdjacentNavSpace(g, cell, nil) {
			t.Errorf("coupler %q at x:%d y:%d has no adjacent stand cell", repair.Name, cell.Col, cell.Row)
		}
	}
}

func stateNewGameForDeck(t *testing.T, levelSeed int64, level int) *state.Game {
	t.Helper()
	g := state.NewGame()
	g.InitRunUnlocks(levelSeed - int64(level-1)*9973)
	g.Level = level
	RegenerateFromSeed(g, levelSeed)
	return g
}

func finalExitGatingRepair(g *state.Game) *entities.RepairObjective {
	for i := len(g.RepairObjectives) - 1; i >= 0; i-- {
		repair := g.RepairObjectives[i]
		if repair != nil && !repair.SkipExitGate {
			return repair
		}
	}
	return nil
}
