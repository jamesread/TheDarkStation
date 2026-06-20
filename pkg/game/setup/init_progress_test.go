package setup

import (
	"testing"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// Reproduces map.txt pattern: a corridor tile is the only init path to a keycard north;
// placing a generator on that tile traps the south locked door keycard chain.
func makeCorridorKeycardTrap(t *testing.T) (*state.Game, *world.Cell, *world.Cell, *world.Cell) {
	t.Helper()
	grid := world.NewGrid(2, 4)
	grid.MarkAsRoomWithName(0, 0, "StartRoom", "desc")
	grid.MarkAsRoomWithName(1, 0, "StartRoom", "desc")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "ROOM_CORRIDOR")
	grid.MarkAsRoomWithName(0, 2, "NorthRoom", "desc")
	grid.MarkAsRoomWithName(0, 3, "SouthRoom", "desc")
	grid.SetExitCellAt(1, 0)
	grid.SetStartCellAt(0, 3)
	grid.BuildAllCellConnections()
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})

	g := state.NewGame()
	g.Grid = grid
	g.Level = 4
	InitRoomPower(g)

	corridor := grid.GetCell(0, 1)
	north := grid.GetCell(0, 2)
	southDoor := grid.GetCell(1, 0)
	gameworld.GetGameData(southDoor).Door = entities.NewDoor("SouthRoom")
	north.ItemsOnFloor.Put(world.NewItem("SouthRoom Keycard"))

	return g, corridor, north, southDoor
}

func TestInitProgressPreserved_rejectsCorridorGeneratorTrap(t *testing.T) {
	g, corridor, _, _ := makeCorridorKeycardTrap(t)
	if InitProgressPreserved(g, corridor) {
		t.Fatal("generator on sole corridor path should break init progress")
	}
	if CanPlaceBlockingEntity(g, corridor) {
		t.Fatal("CanPlaceBlockingEntity should reject corridor generator trap")
	}
}

func TestEnsureGeneratorSafePlacement_relocatesCorridorTrap(t *testing.T) {
	g, corridor, north, _ := makeCorridorKeycardTrap(t)
	gen := entities.NewGenerator("Generator #2", 1)
	gameworld.GetGameData(corridor).Generator = gen
	g.AddGenerator(gen)

	if generatorLocationOK(g, corridor) {
		t.Fatal("precondition: corridor generator should fail location check")
	}

	EnsureGeneratorSafePlacement(g)

	if gameworld.GetGameData(corridor).Generator != nil {
		t.Fatal("generator should be relocated off corridor chokepoint")
	}
	reachable := InitialReachableCells(g)
	if !reachable.Has(north) {
		t.Fatal("north room with keycard should be init reachable after relocation")
	}
	if !keycardsAccessible(g, reachable) {
		t.Fatal("keycard should remain accessible after generator relocation")
	}
}

func TestEnsureKeycardReachability_movesUnreachableFloorKeycard(t *testing.T) {
	g, corridor, north, _ := makeCorridorKeycardTrap(t)
	gen := entities.NewGenerator("Generator #2", 1)
	gameworld.GetGameData(corridor).Generator = gen
	g.AddGenerator(gen)

	EnsureKeycardReachability(g)

	reachable := InitialReachableCells(g)
	if !keycardsAccessible(g, reachable) {
		t.Fatal("EnsureKeycardReachability should move keycard into start reach")
	}
	if north.ItemsOnFloor.Size() > 0 {
		t.Fatal("keycard should be moved off unreachable north cell")
	}
}

func TestFindValidGeneratorCell_rejectsCorridorTrap(t *testing.T) {
	g, corridor, _, _ := makeCorridorKeycardTrap(t)
	avoid := mapset.New[*world.Cell]()
	got := findValidGeneratorCell(g, corridor.Name, PlayerEntryCell(g), &avoid)
	if got == corridor {
		t.Fatal("findValidGeneratorCell should not pick corridor chokepoint")
	}
}
