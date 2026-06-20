package gameplay

import (
	"testing"

	"darkstation/pkg/game/levelseed"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	"darkstation/pkg/game/unlocks"
)

// Regression: shaft maintenance must not block routing coupler placement on the source deck.
// With sequential unlocks, deck 3's routing coupler is placed on deck 2.
func TestRoutingCouplerPlacedOnSourceDeck_mapTxtRunSeed(t *testing.T) {
	levelSeed, err := levelseed.Parse("18B81251043DDAE1") // map.txt deck-2 seed
	if err != nil {
		t.Fatal(err)
	}
	runSeed := levelSeed - 9973 // deck 2 seed = RunSeed + 1*9973

	g := state.NewGame()
	g.InitRunUnlocks(runSeed)
	g.Level = 2
	g.CurrentDeckID = 1
	RegenerateFromSeed(g, levelSeed)

	if !setup.BlockingPlacementPreservesNavAccess(g, nil) {
		t.Error("deck 2: a blocking entity has no adjacent stand cell after setup")
	}

	for _, req := range g.UnlockPlan.ForSource(g.CurrentDeckID) {
		if req.Kind != unlocks.KindRoutingRepair {
			continue
		}
		if g.RepairByID(req.RepairID) == nil {
			t.Errorf("deck 2: routing repair %q (unlocks deck %d) not placed", req.RepairID, req.TargetDeckID+1)
		}
	}
}
