package gameplay

import (
	"testing"

	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/levelseed"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
)

// Regression for map.txt seed 18B7D6E97660ACA5 (deck 2): Pressure Valve was placed in
// Sealed Atmospheric Mixing Bay Far, whose doors could not be powered from the lift entry pocket.
func TestExitGatingRepairAccessible_mapTxtSeed(t *testing.T) {
	levelSeed, err := levelseed.Parse("18B7D6E97660ACA5")
	if err != nil {
		t.Fatal(err)
	}
	runSeed := levelSeed - 9973

	g := state.NewGame()
	g.InitRunUnlocks(runSeed)
	g.Level = 2
	RegenerateFromSeed(g, levelSeed)

	if !setup.ExitGatingRepairsAccessible(g) {
		t.Fatal("exit-gating repairs should be reachable or powerable from lift entry")
	}

	report := setup.AnalyzeSolvability(g)
	for _, w := range report.Warnings {
		t.Errorf("unexpected solvability warning: %s", w)
	}

	var exitGating int
	for _, repair := range g.RepairObjectives {
		if repair == nil || repair.SkipExitGate {
			continue
		}
		// Conduit splices live on corridors; their reachability is plain walking,
		// verified by the progression simulator (see ExitGatingRepairsAccessible).
		if repair.Type == entities.RepairConduitSplice {
			exitGating++
			continue
		}
		exitGating++
		cell := g.Grid.GetCell(repair.DeviceRow, repair.DeviceCol)
		if cell == nil {
			t.Fatalf("repair %q missing device cell", repair.Name)
		}
		if !setup.ExitGatingRepairRoomAccessible(g, cell.Name) {
			t.Fatalf("repair %q at x:%d y:%d in %q not accessible from lift entry",
				repair.Name, cell.Col, cell.Row, cell.Name)
		}
	}
	if exitGating == 0 {
		t.Fatal("expected at least one exit-gating repair on deck 2")
	}
}
