package gameplay

import (
	"testing"

	"darkstation/pkg/game/setup"
)

// Regression: routing couplers on source decks need live grid power when their room is online.
func TestRoutingCouplerHasLivePower_onSourceDeck(t *testing.T) {
	for seed := int64(1); seed <= 32; seed++ {
		g := buildGameWithSeed(2, seed)
		for _, repair := range g.RepairObjectives {
			if repair == nil || !repair.SkipExitGate {
				continue
			}
			cell := g.Grid.GetCell(repair.DeviceRow, repair.DeviceCol)
			if cell == nil || !setup.RoomConsideredPowered(g, cell.Name) {
				continue
			}
			if !setup.CellHasLivePower(g, cell) {
				t.Fatalf("seed %d: coupler %q at x:%d y:%d should have live power when room is online",
					seed, repair.Name, cell.Col, cell.Row)
			}
			return
		}
	}
	t.Fatal("no powered routing coupler found on deck 2 in seed sweep")
}
