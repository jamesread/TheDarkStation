package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// simulateTestGame builds:
//
//	cols:  0    1    2    3    4
//	row 1: A    A    Cor  D    B
//
// Entry/exit at (1,0). The door at (1,3) is LOCKED (Room B keycard).
func simulateTestGame(t *testing.T) (*state.Game, *world.Grid) {
	t.Helper()
	g := state.NewGame()
	g.Level = 1
	grid := world.NewGrid(3, 5)
	put := func(row, col int, name string) {
		grid.MarkAsRoomWithName(row, col, name, "desc")
		gameworld.InitGameData(grid.GetCell(row, col))
	}
	put(1, 0, "Room A")
	put(1, 1, "Room A")
	put(1, 2, "Corridor")
	put(1, 3, "Corridor")
	put(1, 4, "Room B")
	grid.SetStartCellAt(1, 0)
	grid.SetExitCellAt(1, 0)
	grid.BuildAllCellConnections()

	gameworld.GetGameData(grid.GetCell(1, 3)).Door = entities.NewDoor("Room B")

	g.Grid = grid
	return g, grid
}

func TestSimulatePlaythrough_KeycardLockedInOwnRoomFails(t *testing.T) {
	g, grid := simulateTestGame(t)
	// Keycard to Room B locked inside Room B: the classic circular dependency.
	grid.GetCell(1, 4).ItemsOnFloor.Put(world.NewItem("Room B Keycard"))

	report := SimulatePlaythrough(g)
	if report.Solvable {
		t.Fatalf("deck with Room B keycard locked inside Room B reported solvable; trace=%v", report.Trace)
	}
}

func TestSimulatePlaythrough_KeycardOutsideRoomSucceeds(t *testing.T) {
	g, grid := simulateTestGame(t)
	grid.GetCell(1, 1).ItemsOnFloor.Put(world.NewItem("Room B Keycard"))

	report := SimulatePlaythrough(g)
	if !report.Solvable {
		t.Fatalf("solvable deck reported unsolvable: %v", report.Failures)
	}
}

func TestSimulatePlaythrough_KeycardInFurnitureChain(t *testing.T) {
	g, grid := simulateTestGame(t)
	// Keycard hidden in furniture on a side cell of Room A (furniture blocks movement,
	// so it must not sit on the only path): player opens it, then unlocks B.
	grid.MarkAsRoomWithName(0, 1, "Room A", "desc")
	gameworld.InitGameData(grid.GetCell(0, 1))
	grid.BuildAllCellConnections()
	furn := entities.NewFurniture("Crate", "desc", "▦")
	furn.ContainedItem = world.NewItem("Room B Keycard")
	gameworld.GetGameData(grid.GetCell(0, 1)).Furniture = furn

	report := SimulatePlaythrough(g)
	if !report.Solvable {
		t.Fatalf("furniture keycard chain reported unsolvable: %v", report.Failures)
	}
}

func TestSimulatePlaythrough_SealedRepairDeviceFails(t *testing.T) {
	g, grid := simulateTestGame(t)
	grid.GetCell(1, 1).ItemsOnFloor.Put(world.NewItem("Room B Keycard"))

	// A repair device walled into Room B's only approach makes the deck unsolvable:
	// block the corridor cell before the door with a second device.
	corridor := grid.GetCell(1, 2)
	blocker := entities.NewRepairObjective("seal", entities.RepairPressureValve, corridor.Name, corridor.Row, corridor.Col)
	gameworld.GetGameData(corridor).RepairDevice = blocker

	device := entities.NewRepairObjective("deck1-repair1", entities.RepairPressureValve, "Room B", 1, 4)
	gameworld.GetGameData(grid.GetCell(1, 4)).RepairDevice = device
	g.RepairObjectives = []*entities.RepairObjective{blocker, device}

	report := SimulatePlaythrough(g)
	if report.Solvable {
		t.Fatal("deck with repair device sealed behind another device reported solvable")
	}
}

func TestSimulatePlaythrough_RepairChainWithPrereqs(t *testing.T) {
	g, grid := simulateTestGame(t)
	grid.GetCell(1, 1).ItemsOnFloor.Put(world.NewItem("Room B Keycard"))
	g.RoomDoorsPowered["Room A"] = true
	g.RoomDoorsPowered["Room B"] = true
	// Side cell so the second device does not wall off the corridor.
	grid.MarkAsRoomWithName(0, 1, "Room A", "desc")
	gameworld.InitGameData(grid.GetCell(0, 1))
	grid.BuildAllCellConnections()

	// First device in Room B (behind the keycard door); second depends on it.
	first := entities.NewRepairObjective("deck1-repair1", entities.RepairPressureValve, "Room B", 1, 4)
	gameworld.GetGameData(grid.GetCell(1, 4)).RepairDevice = first

	second := entities.NewRepairObjective("deck1-repair2", entities.RepairPressureValve, "Room A", 0, 1)
	second.PrereqIDs = []string{first.ID}
	gameworld.GetGameData(grid.GetCell(0, 1)).RepairDevice = second
	g.RepairObjectives = []*entities.RepairObjective{first, second}

	report := SimulatePlaythrough(g)
	if !report.Solvable {
		t.Fatalf("repair chain (keycard -> device 1 -> device 2) reported unsolvable: %v", report.Failures)
	}
}

func TestSimulatePlaythrough_HazardWithControl(t *testing.T) {
	g, grid := simulateTestGame(t)
	grid.GetCell(1, 1).ItemsOnFloor.Put(world.NewItem("Room B Keycard"))

	hazard := entities.NewHazard(entities.HazardElectrical)
	gameworld.GetGameData(grid.GetCell(1, 2)).Hazard = hazard
	control := entities.NewHazardControl(hazard.Type, hazard)
	gameworld.GetGameData(grid.GetCell(0, 1)).HazardControl = control
	grid.MarkAsRoomWithName(0, 1, "Room A", "desc")
	grid.BuildAllCellConnections()

	report := SimulatePlaythrough(g)
	if !report.Solvable {
		t.Fatalf("hazard with reachable control reported unsolvable: %v", report.Failures)
	}
}
