package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/levelseed"
	"darkstation/pkg/game/setup"
)

func TestSetupLevel_mapTxtSeed_level10_batteryDemandMet(t *testing.T) {
	seed, err := levelseed.Parse("18B33D08129D5B45")
	if err != nil {
		t.Fatal(err)
	}
	g := BuildGame(deck.TotalDecks)
	LoadLevelFromSeed(g, seed)

	demand := setup.UnpoweredGeneratorBatteryDemand(g)
	batteryCount := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			if item != nil && item.Name == "Battery" {
				batteryCount++
			}
		})
	})
	if demand > 0 && batteryCount < demand {
		t.Fatalf("placed %d batteries, unpowered generators need %d", batteryCount, demand)
	}
}
