package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
)

func loadLevelFromSeedForTest(t *testing.T, level int, seed int64) *state.Game {
	t.Helper()
	g := state.NewGame()
	g.CurrentDeckID = level - 1
	g.Level = level
	g.InitRunUnlocks(seed ^ 0x4b1d)
	LoadLevelFromSeed(g, seed)
	return g
}

func TestLiftShaftItemsNotPlacedAfterReachability_seed18B915A36BFF907C(t *testing.T) {
	const seed = int64(0x18B915A36BFF907C)
	g := loadLevelFromSeedForTest(t, 6, seed)

	report := setup.SimulatePlaythrough(g)
	if !report.Solvable {
		t.Fatalf("fresh generation unsolvable: %v", report.Failures)
	}

	reach := setup.InitialReachableCells(g)
	nonShaftReach := 0
	reach.Each(func(cell *world.Cell) {
		if cell != nil && cell.Name != generator.ShaftRoomName {
			nonShaftReach++
		}
	})
	t.Logf("init reachable=%d non-shaft cells=%d", reach.Size(), nonShaftReach)

	setup.EnsureFloorLootReachability(g)

	top, left, bottom, right := generator.ShaftBoundsForLevel(g.Grid.Rows(), g.Grid.Cols(), g.Level)
	_ = top
	_ = left
	_ = bottom
	_ = right
	exit := g.Grid.ExitCell()

	var shaftItems []string
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !setup.IsLiftShaftBoundsCell(g, cell) {
			return
		}
		if cell == exit {
			return
		}
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			if item != nil {
				shaftItems = append(shaftItems, item.Name)
				t.Errorf("floor loot %q in lift shaft at x:%d y:%d", item.Name, col, row)
			}
		})
	})
	if len(shaftItems) > 0 {
		t.Fatalf("expected no floor loot in lift shaft, got %v", shaftItems)
	}
}
