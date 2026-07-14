package setup

import (
	"testing"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// corridorTestGame builds a dumbbell layout:
//
//	cols:  0    1    2    3    4    5    6
//	row 1: A    A    Cor  Cor  Cor+D B    B
//
// Entry (exit cell) at (1,0). Room B sits behind an UNPOWERED unlocked door at
// (1,4), so B is NOT init-reachable. Historically that made B invisible to all
// placement validators — a permanent blocker on the 1-wide corridor passed every
// check while sealing B forever.
func corridorTestGame(t *testing.T) (*state.Game, *world.Grid) {
	t.Helper()
	g := state.NewGame()
	g.Level = 1
	grid := world.NewGrid(3, 7)
	put := func(row, col int, name string) {
		grid.MarkAsRoomWithName(row, col, name, "desc")
		gameworld.InitGameData(grid.GetCell(row, col))
	}
	put(1, 0, "Room A")
	put(1, 1, "Room A")
	put(1, 2, "Corridor")
	put(1, 3, "Corridor")
	put(1, 4, "Corridor")
	put(1, 5, "Room B")
	put(1, 6, "Room B")
	grid.SetStartCellAt(1, 0)
	grid.SetExitCellAt(1, 0)
	grid.BuildAllCellConnections()

	door := entities.NewUnlockedDoor("Room B")
	gameworld.GetGameData(grid.GetCell(1, 4)).Door = door
	g.RoomDoorsPowered["Room B"] = false

	g.Grid = grid
	return g, grid
}

func TestCompletionRegionPreserved_RejectsSealingRoomBehindUnpoweredDoor(t *testing.T) {
	g, grid := corridorTestGame(t)
	candidate := grid.GetCell(1, 2)

	if CompletionRegionPreserved(g, candidate) {
		t.Fatal("CompletionRegionPreserved allowed a blocker that seals Room B behind an unpowered door")
	}
	if CanPlaceBlockingEntity(g, candidate) {
		t.Fatal("CanPlaceBlockingEntity allowed a blocker that seals Room B behind an unpowered door")
	}
}

func TestCompletionRegionPreserved_AllowsDeadEndTip(t *testing.T) {
	g, grid := corridorTestGame(t)
	// (1,6) is the far tip of Room B; blocking it severs no other cell.
	candidate := grid.GetCell(1, 6)
	if !CompletionRegionPreserved(g, candidate) {
		t.Fatal("CompletionRegionPreserved rejected a dead-end tip that severs nothing")
	}
}

// wideCorridorTestGame builds a layout with a 2-wide corridor:
//
//	cols:  0    1    2    3    4    5    6
//	row 1: A    A    Cor  Cor  Cor  B    B
//	row 2: A    A    Cor  Cor  Cor  B    B
//
// Entry (exit cell) at (1,0). No doors: everything is init-reachable.
func wideCorridorTestGame(t *testing.T) (*state.Game, *world.Grid) {
	t.Helper()
	g := state.NewGame()
	g.Level = 1
	grid := world.NewGrid(4, 7)
	put := func(row, col int, name string) {
		grid.MarkAsRoomWithName(row, col, name, "desc")
		gameworld.InitGameData(grid.GetCell(row, col))
	}
	for _, row := range []int{1, 2} {
		put(row, 0, "Room A")
		put(row, 1, "Room A")
		put(row, 2, "Corridor")
		put(row, 3, "Corridor")
		put(row, 4, "Corridor")
		put(row, 5, "Room B")
		put(row, 6, "Room B")
	}
	grid.SetStartCellAt(1, 0)
	grid.SetExitCellAt(1, 0)
	grid.BuildAllCellConnections()
	g.Grid = grid
	return g, grid
}

// TestCanPlaceBlockingEntity_CombinedPlacementsRejected is the regression test for
// the valve+coupler class of bug: two blockers that are individually legal must not
// be allowed to jointly pinch off a corridor. The second placement must be
// re-validated against the grid containing the first.
func TestCanPlaceBlockingEntity_CombinedPlacementsRejected(t *testing.T) {
	g, grid := wideCorridorTestGame(t)
	first := grid.GetCell(1, 3)
	second := grid.GetCell(2, 3)

	if !CanPlaceBlockingEntity(g, first) {
		t.Fatal("first corridor blocker should be legal (parallel lane remains)")
	}
	// Commit the first blocker (e.g. a repair device).
	gameworld.GetGameData(first).RepairDevice = entities.NewRepairObjective(
		"r1", entities.RepairPressureValve, first.Name, first.Row, first.Col)

	if CanPlaceBlockingEntity(g, second) {
		t.Fatal("second corridor blocker seals Room B and must be rejected after the first is committed")
	}
}

func TestCompletionRegionPreservedWithSet_RejectsJointSeal(t *testing.T) {
	g, grid := wideCorridorTestGame(t)
	set := mapset.New[*world.Cell]()
	set.Put(grid.GetCell(1, 3))
	set.Put(grid.GetCell(2, 3))
	if CompletionRegionPreservedWithSet(g, &set) {
		t.Fatal("jointly sealing both corridor lanes must be rejected")
	}

	single := mapset.New[*world.Cell]()
	single.Put(grid.GetCell(1, 3))
	if !CompletionRegionPreservedWithSet(g, &single) {
		t.Fatal("blocking one of two corridor lanes must be allowed")
	}
}

func TestBlockingPlacementValidator_MarksAllRegionCuts(t *testing.T) {
	g, grid := corridorTestGame(t)
	v := NewBlockingPlacementValidator(g)
	if v.CanPlace(grid.GetCell(1, 3)) {
		t.Fatal("cached validator allowed an articulation cell that severs Room B")
	}
	if !v.CanPlace(grid.GetCell(1, 6)) {
		t.Fatal("cached validator rejected a dead-end tip that severs nothing")
	}
}
