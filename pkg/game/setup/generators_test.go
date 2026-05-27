package setup

import (
	"testing"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func makeGridWithRooms(rows, cols int, roomName string) *world.Grid {
	g := world.NewGrid(rows, cols)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			g.MarkAsRoomWithName(r, c, roomName, "desc")
			gameworld.InitGameData(g.GetCell(r, c))
		}
	}
	g.SetStartCellAt(0, 0)
	g.SetExitCellAt(rows-1, cols-1)
	g.BuildAllCellConnections()
	return g
}

func TestPlaceGenerators_SpawnGeneratorPlaced(t *testing.T) {
	g := state.NewGame()
	g.Level = 1
	g.Grid = makeGridWithRooms(3, 3, "Spawn")
	avoid := mapset.New[*world.Cell]()
	avoid.Put(g.Grid.StartCell())
	avoid.Put(g.Grid.ExitCell())

	placeGenerators(g, &avoid)

	if len(g.Generators) == 0 {
		t.Fatal("placeGenerators placed no generators on level 1")
	}
	if len(g.Generators) != 1 {
		t.Errorf("placeGenerators placed %d generators on level 1, want 1", len(g.Generators))
	}
}

func TestPlaceGenerators_SpawnGeneratorAutoPowered(t *testing.T) {
	g := state.NewGame()
	g.Level = 1
	g.Grid = makeGridWithRooms(3, 3, "Spawn")
	avoid := mapset.New[*world.Cell]()
	avoid.Put(g.Grid.StartCell())
	avoid.Put(g.Grid.ExitCell())

	placeGenerators(g, &avoid)

	if len(g.Generators) == 0 {
		t.Fatal("no generators placed")
	}
	gen := g.Generators[0]
	if !gen.IsPowered() {
		t.Errorf("spawn generator not auto-powered: inserted=%d, required=%d",
			gen.BatteriesInserted, gen.BatteriesRequired)
	}
}

func TestPlaceGenerators_PowerSupplyUpdatedAfterSpawn(t *testing.T) {
	g := state.NewGame()
	g.Level = 1
	g.CurrentDeckID = 0
	g.Grid = makeGridWithRooms(3, 3, "Spawn")
	avoid := mapset.New[*world.Cell]()
	avoid.Put(g.Grid.StartCell())
	avoid.Put(g.Grid.ExitCell())

	placeGenerators(g, &avoid)

	if g.PowerSupply != 100 {
		t.Errorf("PowerSupply after spawn generator = %d, want 100", g.PowerSupply)
	}
}

func TestPlaceGenerators_Level2OnlySpawn(t *testing.T) {
	g := state.NewGame()
	g.Level = 2
	grid := world.NewGrid(4, 4)
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			name := "Room"
			if r == 0 && c <= 1 {
				name = "Spawn"
			}
			grid.MarkAsRoomWithName(r, c, name, "desc")
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(3, 3)
	grid.BuildAllCellConnections()
	g.Grid = grid
	avoid := mapset.New[*world.Cell]()
	avoid.Put(g.Grid.StartCell())
	avoid.Put(g.Grid.ExitCell())

	placeGenerators(g, &avoid)

	if len(g.Generators) != 1 {
		t.Errorf("level 2: placed %d generators, want 1 (spawn only)", len(g.Generators))
	}
}

func TestIsValidForGenerator(t *testing.T) {
	grid := world.NewGrid(1, 3)
	grid.MarkAsRoomWithName(0, 0, "R", "desc")
	grid.MarkAsRoomWithName(0, 1, "R", "desc")
	grid.MarkAsRoomWithName(0, 2, "R", "desc")
	grid.BuildAllCellConnections()

	for _, cell := range []*world.Cell{grid.GetCell(0, 0), grid.GetCell(0, 1), grid.GetCell(0, 2)} {
		gameworld.InitGameData(cell)
	}

	avoid := mapset.New[*world.Cell]()

	cell0 := grid.GetCell(0, 0)
	if !isValidForGenerator(cell0, &avoid) {
		t.Error("empty cell should be valid for generator")
	}

	avoid.Put(cell0)
	if isValidForGenerator(cell0, &avoid) {
		t.Error("avoided cell should not be valid")
	}

	cell1 := grid.GetCell(0, 1)
	cell1.ExitCell = true
	if isValidForGenerator(cell1, &avoid) {
		t.Error("exit cell should not be valid")
	}

	cell2 := grid.GetCell(0, 2)
	gameworld.GetGameData(cell2).Furniture = &entities.Furniture{Name: "Desk"}
	if isValidForGenerator(cell2, &avoid) {
		t.Error("cell with furniture should not be valid")
	}
}

func TestCalculateBatteriesForGenerator(t *testing.T) {
	for level := 3; level <= 10; level++ {
		bat := calculateBatteriesForGenerator(level)
		if bat < 1 {
			t.Errorf("level %d: calculateBatteriesForGenerator = %d, want >= 1", level, bat)
		}
		if bat > 5 {
			t.Errorf("level %d: calculateBatteriesForGenerator = %d, want <= 5", level, bat)
		}
	}
}

func countGridGenerators(g *state.Game) int {
	if g == nil || g.Grid == nil {
		return 0
	}
	n := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && gameworld.GetGameData(cell).Generator != nil {
			n++
		}
	})
	return n
}

func TestNumAdditionalGeneratorsForLevel_finalDeckMinimal(t *testing.T) {
	if got := numAdditionalGeneratorsForLevel(10); got != 1 {
		t.Fatalf("final deck additional generators = %d, want 1", got)
	}
	if got := numAdditionalGeneratorsForLevel(7); got != 4 {
		t.Fatalf("level 7 additional generators = %d, want 4", got)
	}
}

func TestPlaceAdditionalGenerators_AfterBootstrap_mapTxtSeed(t *testing.T) {
	// Regression: level 7 map.txt seed previously placed only the spawn generator.
	g := state.NewGame()
	g.Level = 7
	g.CurrentDeckID = 6
	grid := world.NewGrid(8, 8)
	for r := 0; r < 8; r++ {
		for c := 0; c < 8; c++ {
			name := "Far"
			if r < 4 && c < 4 {
				name = "Start"
			}
			grid.MarkAsRoomWithName(r, c, name, "desc")
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}
	grid.SetStartCellAt(1, 1)
	grid.SetExitCellAt(6, 6)
	grid.BuildAllCellConnections()
	g.Grid = grid

	avoid := mapset.New[*world.Cell]()
	avoid.Put(g.Grid.StartCell())
	avoid.Put(g.Grid.ExitCell())
	InitRoomPower(g)
	placeSpawnGenerator(g, &avoid)
	InitMaintenanceTerminalPower(g)
	EnsureGeneratorRoomBootstrap(g)
	PlaceAdditionalGenerators(g, &avoid)

	want := 1 + (g.Level - 3)
	if got := countGridGenerators(g); got != want {
		t.Fatalf("grid generators = %d, want %d (spawn + level-3 additional)", got, want)
	}
	powered := 0
	unpowered := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		gen := gameworld.GetGameData(cell).Generator
		if gen == nil {
			return
		}
		if gen.IsPowered() {
			powered++
		} else {
			unpowered++
		}
	})
	if powered != 1 {
		t.Fatalf("powered generators = %d, want 1 (spawn only)", powered)
	}
	if unpowered != g.Level-3 {
		t.Fatalf("unpowered generators = %d, want %d", unpowered, g.Level-3)
	}
}
