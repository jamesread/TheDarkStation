package setup

import (
	"testing"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// Reproduces map.txt: generator in corner with two adjacent floor cells; two furniture block all access.
func makeCornerGeneratorRoom(t *testing.T) (*state.Game, *world.Cell, *world.Cell, *world.Cell) {
	t.Helper()
	grid := world.NewGrid(4, 6)
	for r := 1; r <= 2; r++ {
		for c := 2; c <= 5; c++ {
			grid.MarkAsRoomWithName(r, c, "StartRoom", "desc")
		}
	}
	grid.SetStartCellAt(2, 4)
	grid.SetExitCellAt(2, 5)
	grid.BuildAllCellConnections()
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})
	g := state.NewGame()
	g.Grid = grid

	genCell := grid.GetCell(1, 2)
	gameworld.GetGameData(genCell).Generator = entities.NewGenerator("Generator #1", 1)
	fEast := grid.GetCell(1, 3)
	fSouth := grid.GetCell(2, 2)
	return g, genCell, fEast, fSouth
}

func TestBlockingPlacementPreservesNavAccess_rejectsSecondFurnitureTrap(t *testing.T) {
	g, genCell, fEast, fSouth := makeCornerGeneratorRoom(t)
	if !EntityHasAdjacentNavSpace(g, genCell, nil) {
		t.Fatal("generator should start with adjacent nav space")
	}
	if !CanPlaceBlockingEntity(g, fEast) {
		t.Fatal("should allow first adjacent furniture")
	}

	gameworld.GetGameData(fEast).Furniture = entities.NewFurniture("Panel", "desc", "F")
	if !EntityHasAdjacentNavSpace(g, genCell, nil) {
		t.Fatal("generator should still have nav space with one adjacent furniture")
	}
	if CanPlaceBlockingEntity(g, fSouth) {
		t.Fatal("should reject second furniture that removes all generator nav space")
	}
}

func TestCanPlaceBlockingEntity_rejectsFurnitureBlockingGenerator(t *testing.T) {
	g, _, fEast, fSouth := makeCornerGeneratorRoom(t)
	gameworld.GetGameData(fEast).Furniture = entities.NewFurniture("Panel", "desc", "F")
	if CanPlaceBlockingEntity(g, fSouth) {
		t.Fatal("should not allow second furniture that blocks generator access")
	}
}

func TestEnsureInteractableNavAccess_clearsTrappingFurniture(t *testing.T) {
	g, genCell, fEast, fSouth := makeCornerGeneratorRoom(t)
	gameworld.GetGameData(fEast).Furniture = entities.NewFurniture("Panel", "desc", "F")
	gameworld.GetGameData(fSouth).Furniture = entities.NewFurniture("Scrubber", "desc", "F")
	if EntityHasAdjacentNavSpace(g, genCell, nil) {
		t.Fatal("setup should trap generator")
	}

	EnsureInteractableNavAccess(g)
	if !EntityHasAdjacentNavSpace(g, genCell, nil) {
		t.Fatal("EnsureInteractableNavAccess should restore generator nav access")
	}
}

func TestFindValidGeneratorCell_requiresAdjacentNavSpace(t *testing.T) {
	grid := world.NewGrid(3, 3)
	grid.MarkAsRoomWithName(1, 1, "StartRoom", "desc")
	grid.SetStartCellAt(0, 0)
	grid.BuildAllCellConnections()
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})
	g := state.NewGame()
	g.Grid = grid
	avoid := mapset.New[*world.Cell]()

	lone := grid.GetCell(1, 1)
	if CandidateBlockingCellHasAdjacentNavSpace(g, lone, nil) {
		t.Fatal("isolated cell should lack adjacent nav space for a blocking placement")
	}
	got := findValidGeneratorCell(g, "StartRoom", grid.GetCell(0, 0), &avoid)
	if got != nil {
		t.Fatalf("findValidGeneratorCell = (%d,%d), want nil", got.Row, got.Col)
	}
}
