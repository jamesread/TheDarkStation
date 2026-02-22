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
