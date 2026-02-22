package setup

import (
	"testing"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
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
	// Level 3: numAdditionalGenerators = 3-3 = 0, so no batteries needed beyond spawn.
	// placeBatteries still runs but calculateTotalBatteriesNeeded returns 0.
	g := state.NewGame()
	g.Level = 3
	g.Grid = makeGridWithRooms(5, 5, "Room")
	avoid := mapset.New[*world.Cell]()
	avoid.Put(g.Grid.StartCell())
	avoid.Put(g.Grid.ExitCell())

	gen1 := entities.NewGenerator("G1", 1)
	gen1.InsertBatteries(1)
	g.AddGenerator(gen1)

	placeBatteries(g, &avoid)

	// With 0 additional generators, formula yields at most 1 extra battery
	// (totalBatteriesNeeded=0, spawnGenBatteries=1, extras=1-2 → total 0-1).
	batteryCount := countItemsOnGrid(g.Grid, "Battery")
	if batteryCount > 2 {
		t.Errorf("level 3 with no additional generators: placed %d batteries, want <= 2", batteryCount)
	}
}

func TestPlaceBatteries_Level5PlacesBatteries(t *testing.T) {
	// Level 5: numAdditionalGenerators = 5-3 = 2; batteries are placed for them.
	g := state.NewGame()
	g.Level = 5
	g.Grid = makeGridWithRooms(6, 6, "Room")
	avoid := mapset.New[*world.Cell]()
	avoid.Put(g.Grid.StartCell())
	avoid.Put(g.Grid.ExitCell())

	gen1 := entities.NewGenerator("G1", 1)
	gen1.InsertBatteries(1)
	g.AddGenerator(gen1)

	placeBatteries(g, &avoid)

	batteryCount := countItemsOnGrid(g.Grid, "Battery")
	if batteryCount == 0 {
		t.Error("level 5: no batteries placed for 2 additional generators")
	}
}

func TestGetSpawnGeneratorBatteries(t *testing.T) {
	g := state.NewGame()
	if got := getSpawnGeneratorBatteries(g); got != 0 {
		t.Errorf("no generators: getSpawnGeneratorBatteries = %d, want 0", got)
	}

	gen := entities.NewGenerator("G1", 3)
	gen.InsertBatteries(3)
	g.AddGenerator(gen)

	if got := getSpawnGeneratorBatteries(g); got != 3 {
		t.Errorf("powered spawn gen: getSpawnGeneratorBatteries = %d, want 3", got)
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
