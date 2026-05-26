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
	InitRoomPower(g)
	EnsureGeneratorRoomBootstrap(g)

	placeBatteries(g, &avoid)

	batteryCount := countItemsOnGrid(g.Grid, "Battery")
	if batteryCount == 0 {
		t.Error("level 5: no batteries placed for 2 additional generators")
	}
	if !allBatteriesInitReachable(g) {
		t.Error("level 5: all placed batteries should be init-reachable")
	}
}

func TestEnsureBatteryReachability_relocatesUnreachableBattery(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 4)
	grid.MarkAsRoomWithName(0, 0, "Start", "desc")
	grid.MarkAsRoomWithName(0, 1, "Start", "desc")
	grid.MarkAsRoomWithName(0, 2, "Corridor", "desc")
	grid.MarkAsRoomWithName(0, 3, "Far", "desc")
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(0, 1)
	grid.BuildAllCellConnections()
	for c := 0; c < 4; c++ {
		gameworld.InitGameData(grid.GetCell(0, c))
	}
	doorCell := grid.GetCell(0, 2)
	gameworld.GetGameData(doorCell).Door = &entities.Door{RoomName: "Far", Locked: false}
	g.Grid = grid
	InitRoomPower(g)

	stranded := grid.GetCell(0, 3)
	stranded.ItemsOnFloor.Put(world.NewItem("Battery"))
	stranded.ItemsOnFloor.Put(world.NewItem("Battery"))

	EnsureBatteryReachability(g)

	reachable := InitialReachableCells(g)
	if reachable.Has(stranded) {
		t.Fatal("precondition: Far room should be unreachable at init")
	}
	strandedCount := 0
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			if item != nil && item.Name == "Battery" && !reachable.Has(cell) {
				strandedCount++
			}
		})
	})
	if strandedCount != 0 {
		t.Fatalf("expected 0 unreachable batteries after EnsureBatteryReachability, got %d", strandedCount)
	}
	if stranded.ItemsOnFloor.Size() != 0 {
		t.Fatal("stranded cell should no longer hold batteries after relocation")
	}
}

func allBatteriesInitReachable(g *state.Game) bool {
	reachable := InitialReachableCells(g)
	ok := true
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			if item != nil && item.Name == "Battery" && !reachable.Has(cell) {
				ok = false
			}
		})
	})
	return ok
}

func TestGetSpawnGeneratorBatteries(t *testing.T) {
	g := state.NewGame()
	if got := getSpawnGeneratorBatteries(g); got != 0 {
		t.Errorf("no generators: getSpawnGeneratorBatteries = %d, want 0", got)
	}

	gen := entities.NewGenerator("G1", 3)
	gen.InsertBatteries(3)
	gen.BringOnline()
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
