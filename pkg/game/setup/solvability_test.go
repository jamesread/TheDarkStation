package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// makeSolvabilityGrid creates a 1-row grid: Start -- Corridor -- Door(R) -- R -- R(exit)
// R is a gatekeeper because every path from start to exit goes through it.
func makeSolvabilityGrid() (*world.Grid, *world.Cell) {
	grid := world.NewGrid(1, 5)
	grid.MarkAsRoomWithName(0, 0, "Start", "desc")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "desc")
	grid.MarkAsRoomWithName(0, 2, "Corridor", "desc") // door cell
	grid.MarkAsRoomWithName(0, 3, "Lab", "desc")
	grid.MarkAsRoomWithName(0, 4, "Lab", "desc")
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(0, 4)
	grid.BuildAllCellConnections()
	for c := 0; c < 5; c++ {
		gameworld.InitGameData(grid.GetCell(0, c))
	}
	// Place door at (0,2) belonging to "Lab"
	doorCell := grid.GetCell(0, 2)
	gameworld.GetGameData(doorCell).Door = &entities.Door{RoomName: "Lab", Locked: false}
	return grid, doorCell
}

func TestEnsureSolvabilityDoorPower_GatekeeperNoTerminal(t *testing.T) {
	// Lab is gatekeeper (exit behind Lab's door) with no adjacent terminal reachable.
	// Expected: Lab's doors get powered.
	g := state.NewGame()
	grid, _ := makeSolvabilityGrid()
	g.Grid = grid
	InitRoomPower(g)

	EnsureSolvabilityDoorPower(g)

	if !g.RoomDoorsPowered["Lab"] {
		t.Error("gatekeeper Lab with no adjacent terminal: doors should be powered")
	}
}

func TestEnsureSolvabilityDoorPower_NonGatekeeperNotPowered(t *testing.T) {
	// Create a grid where the exit is reachable without entering "Lab".
	// Layout: Start -- Lab(door) -- Exit  but also Start -- Alt -- Exit (bypass)
	grid := world.NewGrid(2, 4)
	// Row 0: Start(0,0) -- Corridor(0,1) -- DoorLab(0,2) -- Lab(0,3)
	grid.MarkAsRoomWithName(0, 0, "Start", "desc")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "desc")
	grid.MarkAsRoomWithName(0, 2, "Corridor", "desc")
	grid.MarkAsRoomWithName(0, 3, "Lab", "desc")
	// Row 1: Start(1,0) -- Alt(1,1) -- Alt(1,2) -- Exit(1,3)
	grid.MarkAsRoomWithName(1, 0, "Start", "desc")
	grid.MarkAsRoomWithName(1, 1, "Alt", "desc")
	grid.MarkAsRoomWithName(1, 2, "Alt", "desc")
	grid.MarkAsRoomWithName(1, 3, "Alt", "desc")
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(1, 3)
	grid.BuildAllCellConnections()
	for r := 0; r < 2; r++ {
		for c := 0; c < 4; c++ {
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}
	// Door into Lab at (0,2)
	gameworld.GetGameData(grid.GetCell(0, 2)).Door = &entities.Door{RoomName: "Lab", Locked: false}

	g := state.NewGame()
	g.Grid = grid
	InitRoomPower(g)

	EnsureSolvabilityDoorPower(g)

	if g.RoomDoorsPowered["Lab"] {
		t.Error("non-gatekeeper Lab (exit reachable via Alt): doors should NOT be powered")
	}
}

func TestEnsureSolvabilityDoorPower_GatekeeperWithAdjacentTerminal(t *testing.T) {
	// Lab is gatekeeper BUT Start has a maintenance terminal AND Start is adjacent to Lab → no deadlock.
	g := state.NewGame()
	grid, _ := makeSolvabilityGrid()
	g.Grid = grid
	InitRoomPower(g)

	// Place maintenance terminal in Start room (adjacent to Lab via corridor)
	startCell := grid.GetCell(0, 0)
	gameworld.GetGameData(startCell).MaintenanceTerm = entities.NewMaintenanceTerminal("MT-Start", "Start")

	EnsureSolvabilityDoorPower(g)

	if g.RoomDoorsPowered["Lab"] {
		t.Error("gatekeeper Lab with adjacent terminal in Start: doors should NOT be powered")
	}
}

func TestEnsureSolvabilityDoorPower_LockedDoorImpassable(t *testing.T) {
	// Locked door cells are treated as impassable when computing reachability.
	// Layout: Start -- LockedDoor(Aux) -- Aux(terminal) -- Corridor -- Door(Lab) -- Lab -- Exit
	// The locked door blocks reachability to Aux, so Aux's terminal can't be reached.
	grid := world.NewGrid(1, 7)
	grid.MarkAsRoomWithName(0, 0, "Start", "desc")
	grid.MarkAsRoomWithName(0, 1, "Start", "desc") // locked door cell into Aux
	grid.MarkAsRoomWithName(0, 2, "Aux", "desc")
	grid.MarkAsRoomWithName(0, 3, "Corridor", "desc")
	grid.MarkAsRoomWithName(0, 4, "Corridor", "desc") // door into Lab
	grid.MarkAsRoomWithName(0, 5, "Lab", "desc")
	grid.MarkAsRoomWithName(0, 6, "Lab", "desc")
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(0, 6)
	grid.BuildAllCellConnections()
	for c := 0; c < 7; c++ {
		gameworld.InitGameData(grid.GetCell(0, c))
	}
	// Locked door into Aux at (0,1)
	gameworld.GetGameData(grid.GetCell(0, 1)).Door = &entities.Door{RoomName: "Aux", Locked: true}
	// Maintenance terminal in Aux
	gameworld.GetGameData(grid.GetCell(0, 2)).MaintenanceTerm = entities.NewMaintenanceTerminal("MT-Aux", "Aux")
	// Unlocked door into Lab at (0,4)
	gameworld.GetGameData(grid.GetCell(0, 4)).Door = &entities.Door{RoomName: "Lab", Locked: false}

	g := state.NewGame()
	g.Grid = grid
	InitRoomPower(g)

	EnsureSolvabilityDoorPower(g)

	// Lab is gatekeeper. Aux has a terminal adjacent to Lab, but Aux is behind a locked door.
	// Locked doors are impassable → Aux unreachable → no adjacent terminal reachable → Lab powered.
	if !g.RoomDoorsPowered["Lab"] {
		t.Error("locked door blocks Aux terminal: Lab should be powered")
	}
}

func TestEnsureSolvabilityDoorPower_NilGridNoPanic(t *testing.T) {
	g := state.NewGame()
	g.Grid = nil
	EnsureSolvabilityDoorPower(g) // must not panic
}

func TestEnsureSolvabilityDoorPower_StartRoomAlreadyPowered(t *testing.T) {
	// Start room doors are already powered by InitRoomPower; should not be re-evaluated.
	g := state.NewGame()
	grid, _ := makeSolvabilityGrid()
	g.Grid = grid
	InitRoomPower(g)

	if !g.RoomDoorsPowered["Start"] {
		t.Fatal("precondition: Start doors should be powered")
	}

	EnsureSolvabilityDoorPower(g)

	if !g.RoomDoorsPowered["Start"] {
		t.Error("Start room doors should remain powered")
	}
}
