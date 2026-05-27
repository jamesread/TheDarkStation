package setup

import (
	"testing"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestPlaceBatteries_Level1NoBatteries(t *testing.T) {
	g := state.NewGame()
	g.Level = 1
	g.Grid = makeGridWithRooms(3, 3, "Room")
	avoid := mapset.New[*world.Cell]()
	avoid.Put(g.Grid.StartCell())
	avoid.Put(g.Grid.ExitCell())

	placeBatteries(g, &avoid)

	if g.Grid.ExitCell().Locked {
		t.Error("level 1: exit should be unlocked after placeBatteries")
	}

	batteryCount := countItemsOnGrid(g.Grid, "Battery")
	if batteryCount != 0 {
		t.Errorf("level 1: found %d batteries on grid, want 0", batteryCount)
	}
}

func TestPlaceBatteries_Level2NoBatteries(t *testing.T) {
	g := state.NewGame()
	g.Level = 2
	g.Grid = makeGridWithRooms(3, 3, "Room")
	avoid := mapset.New[*world.Cell]()
	avoid.Put(g.Grid.StartCell())
	avoid.Put(g.Grid.ExitCell())

	placeBatteries(g, &avoid)

	if g.Grid.ExitCell().Locked {
		t.Error("level 2: exit should be unlocked after placeBatteries")
	}
}

func TestPlaceBatteries_Level3NoAdditionalGenerators(t *testing.T) {
	// Level 3: no unpowered generators on grid → only buffer if any gen placed.
	g := state.NewGame()
	g.Level = 3
	g.Grid = makeGridWithRooms(5, 5, "Room")
	avoid := mapset.New[*world.Cell]()
	avoid.Put(g.Grid.StartCell())
	avoid.Put(g.Grid.ExitCell())

	placeBatteries(g, &avoid)

	batteryCount := countItemsOnGrid(g.Grid, "Battery")
	if batteryCount != 0 {
		t.Errorf("level 3 with no unpowered grid generators: placed %d batteries, want 0", batteryCount)
	}
}

func TestPlaceBatteries_coversUnpoweredGeneratorDemand(t *testing.T) {
	g := state.NewGame()
	g.Level = 10
	g.Grid = makeGridWithRooms(6, 6, "Room")
	avoid := mapset.New[*world.Cell]()
	avoid.Put(g.Grid.StartCell())
	avoid.Put(g.Grid.ExitCell())

	cell := g.Grid.GetCell(2, 2)
	gen := entities.NewGenerator("Generator #2", 4)
	gameworld.GetGameData(cell).Generator = gen
	g.AddGenerator(gen)

	placeBatteries(g, &avoid)

	batteryCount := countItemsOnGrid(g.Grid, "Battery")
	if batteryCount < 4 {
		t.Fatalf("placed %d batteries, want at least 4 for a 4-slot unpowered generator", batteryCount)
	}
}

func TestPlaceBatteries_Level5PlacesBatteries(t *testing.T) {
	g := state.NewGame()
	g.Level = 5
	g.Grid = makeGridWithRooms(6, 6, "Room")
	avoid := mapset.New[*world.Cell]()
	avoid.Put(g.Grid.StartCell())
	avoid.Put(g.Grid.ExitCell())

	InitRoomPower(g)
	placeSpawnGenerator(g, &avoid)
	PlaceAdditionalGenerators(g, &avoid)

	placeBatteries(g, &avoid)

	demand := unpoweredGeneratorBatteryDemand(g)
	batteryCount := countItemsOnGrid(g.Grid, "Battery")
	if demand > 0 && batteryCount < demand {
		t.Fatalf("placed %d batteries, want at least %d for unpowered generators", batteryCount, demand)
	}
}

func TestUnpoweredGeneratorBatteryDemand(t *testing.T) {
	g := state.NewGame()
	if got := unpoweredGeneratorBatteryDemand(g); got != 0 {
		t.Errorf("empty grid demand = %d, want 0", got)
	}

	grid := makeGridWithRooms(1, 1, "Room")
	g.Grid = grid
	cell := grid.GetCell(0, 0)
	gen := entities.NewGenerator("G2", 4)
	gameworld.GetGameData(cell).Generator = gen

	if got := unpoweredGeneratorBatteryDemand(g); got != 4 {
		t.Errorf("unpoweredGeneratorBatteryDemand = %d, want 4", got)
	}
}

func countItemsOnGrid(grid *world.Grid, itemName string) int {
	count := 0
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			cell.ItemsOnFloor.Each(func(item *world.Item) {
				if item.Name == itemName {
					count++
				}
			})
		}
	})
	return count
}
