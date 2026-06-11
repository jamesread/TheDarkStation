package gameplay

import (
	"testing"

	"darkstation/pkg/game/levelseed"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	"darkstation/pkg/game/unlocks"
)

// Regression: on this run seed, deck 1's shaft maintenance terminal used to take the
// shaft bootstrap generator's last adjacent stand cell (furniture occupied the others).
// That made BlockingPlacementPreservesNavAccess fail deck-wide, so no repair objectives
// and no routing couplers were placed on deck 1 — leaving deck 3 permanently locked
// ("Lift routing offline") for the whole run.
func TestRoutingCouplerPlacedOnSourceDeck_mapTxtRunSeed(t *testing.T) {
	levelSeed, err := levelseed.Parse("18B81251043DDAE1") // map.txt deck-2 seed
	if err != nil {
		t.Fatal(err)
	}
	runSeed := levelSeed - 9973 // deck 2 seed = RunSeed + 1*9973

	g := state.NewGame()
	g.InitRunUnlocks(runSeed)
	g.Level = 1
	RegenerateFromSeed(g, runSeed)

	if !setup.BlockingPlacementPreservesNavAccess(g, nil) {
		t.Error("deck 1: a blocking entity has no adjacent stand cell after setup")
	}

	exitGating := 0
	for _, r := range g.RepairObjectives {
		if r != nil && !r.SkipExitGate {
			exitGating++
		}
	}
	if exitGating == 0 {
		t.Error("deck 1: no exit-gating repair objectives placed")
	}

	for _, req := range g.UnlockPlan.ForSource(g.CurrentDeckID) {
		if req.Kind != unlocks.KindRoutingRepair {
			continue
		}
		if g.RepairByID(req.RepairID) == nil {
			t.Errorf("deck 1: routing repair %q (unlocks deck %d) not placed", req.RepairID, req.TargetDeckID+1)
		}
	}
}
