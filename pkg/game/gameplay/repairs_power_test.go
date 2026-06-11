package gameplay

import (
	"testing"

	"darkstation/pkg/game/levelseed"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
)

// Regression for map.txt seed 18B7D7DC6C3FD4D2 (deck 1): routing coupler room shows power_online
// but USE reported no power because repair-device cells block grid traversal.
func TestRoutingCouplerHasLivePower_mapTxtSeed(t *testing.T) {
	levelSeed, err := levelseed.Parse("18B7D7DC6C3FD4D2")
	if err != nil {
		t.Fatal(err)
	}

	g := state.NewGame()
	g.InitRunUnlocks(levelSeed)
	g.Level = 1
	RegenerateFromSeed(g, levelSeed)

	var couplerFound bool
	for _, repair := range g.RepairObjectives {
		if repair == nil || repair.ID != "routing-repair-deck3-0" {
			continue
		}
		couplerFound = true
		cell := g.Grid.GetCell(repair.DeviceRow, repair.DeviceCol)
		if cell == nil {
			t.Fatalf("coupler %q missing device cell", repair.Name)
		}
		if !setup.RoomConsideredPowered(g, cell.Name) {
			t.Fatalf("coupler room %q should be considered powered", cell.Name)
		}
		if !setup.CellHasLivePower(g, cell) {
			t.Fatalf("coupler at x:%d y:%d should have live power when room is online", cell.Col, cell.Row)
		}
	}
	if !couplerFound {
		t.Fatal("expected routing-repair-deck3-0 on deck 1 for this seed")
	}
}
