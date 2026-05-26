package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/levelseed"
	"darkstation/pkg/game/setup"
)

func TestSetupLevel_mapTxtSeed_batteriesInitReachable(t *testing.T) {
	seed, err := levelseed.Parse("18B3398076C57641")
	if err != nil {
		t.Fatal(err)
	}
	g := BuildGame(7)
	LoadLevelFromSeed(g, seed)

	reachable := setup.InitialReachableCells(g)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			if item == nil || item.Name != "Battery" {
				return
			}
			if !reachable.Has(cell) {
				t.Fatalf("battery at (%d,%d) room %q is not init-reachable", row, col, cell.Name)
			}
		})
	})
}
