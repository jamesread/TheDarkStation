package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// makeSolvabilityGrid: Lift Shaft -- Corridor -- Door(Lab) -- Lab
// Lab is a gatekeeper because every path from the shaft entry into the deck goes through it.
func makeSolvabilityGrid() (*world.Grid, *world.Cell) {
	grid := world.NewGrid(1, 5)
	grid.MarkAsRoomWithName(0, 0, "Lift Shaft", "desc")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "desc")
	grid.MarkAsRoomWithName(0, 2, "Corridor", "desc") // door cell
	grid.MarkAsRoomWithName(0, 3, "Lab", "desc")
	grid.MarkAsRoomWithName(0, 4, "Lab", "desc")
	grid.SetExitCellAt(0, 0)
	grid.SetStartCellAt(0, 3)
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
	// Create a grid where the deck is reachable without entering "Lab".
	// Row 0: Shaft(0,0) -- Corridor(0,1) -- DoorLab(0,2) -- Lab(0,3)
	grid := world.NewGrid(2, 4)
	grid.MarkAsRoomWithName(0, 0, "Lift Shaft", "desc")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "desc")
	grid.MarkAsRoomWithName(0, 2, "Corridor", "desc")
	grid.MarkAsRoomWithName(0, 3, "Lab", "desc")
	// Row 1: Shaft(1,0) -- Alt(1,1) -- Alt(1,2) -- Alt(1,3)
	grid.MarkAsRoomWithName(1, 0, "Lift Shaft", "desc")
	grid.MarkAsRoomWithName(1, 1, "Alt", "desc")
	grid.MarkAsRoomWithName(1, 2, "Alt", "desc")
	grid.MarkAsRoomWithName(1, 3, "Alt", "desc")
	grid.SetExitCellAt(0, 0)
	grid.SetStartCellAt(1, 3)
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
	// Lab is gatekeeper BUT Lift Shaft has a maintenance terminal adjacent to Lab → no deadlock.
	g := state.NewGame()
	grid, _ := makeSolvabilityGrid()
	g.Grid = grid
	InitRoomPower(g)

	shaftCell := grid.GetCell(0, 0)
	gameworld.GetGameData(shaftCell).MaintenanceTerm = entities.NewMaintenanceTerminal("MT-Shaft", "Lift Shaft")

	EnsureSolvabilityDoorPower(g)

	if g.RoomDoorsPowered["Lab"] {
		t.Error("gatekeeper Lab with adjacent terminal in Lift Shaft: doors should NOT be powered")
	}
}

func TestEnsureSolvabilityDoorPower_LockedDoorImpassable(t *testing.T) {
	// Locked door blocks a side branch with the only adjacent terminal for gatekeeper Lab.
	grid := world.NewGrid(2, 5)
	grid.MarkAsRoomWithName(0, 0, "Lift Shaft", "desc")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "desc")
	grid.MarkAsRoomWithName(0, 2, "Corridor", "desc") // door into Lab
	grid.MarkAsRoomWithName(0, 3, "Lab", "desc")
	grid.MarkAsRoomWithName(0, 4, "Lab", "desc")
	grid.MarkAsRoomWithName(1, 1, "Lift Shaft", "desc") // locked door into Aux
	grid.MarkAsRoomWithName(1, 2, "Aux", "desc")
	grid.SetExitCellAt(0, 0)
	grid.SetStartCellAt(0, 3)
	grid.BuildAllCellConnections()
	for r := 0; r < 2; r++ {
		for c := 0; c < 5; c++ {
			if cell := grid.GetCell(r, c); cell != nil {
				gameworld.InitGameData(cell)
			}
		}
	}
	gameworld.GetGameData(grid.GetCell(1, 1)).Door = &entities.Door{RoomName: "Aux", Locked: true}
	gameworld.GetGameData(grid.GetCell(1, 2)).MaintenanceTerm = entities.NewMaintenanceTerminal("MT-Aux", "Aux")
	gameworld.GetGameData(grid.GetCell(0, 2)).Door = &entities.Door{RoomName: "Lab", Locked: false}

	g := state.NewGame()
	g.Grid = grid
	InitRoomPower(g)

	EnsureSolvabilityDoorPower(g)

	if !g.RoomDoorsPowered["Lab"] {
		t.Error("locked door blocks Aux terminal: Lab should be powered")
	}
}

func TestEnsureSolvabilityDoorPower_NilGridNoPanic(t *testing.T) {
	g := state.NewGame()
	g.Grid = nil
	EnsureSolvabilityDoorPower(g) // must not panic
}

func TestEnsureSolvabilityDoorPower_StartRoomNotAutoPowered(t *testing.T) {
	g := state.NewGame()
	grid, _ := makeSolvabilityGrid()
	g.Grid = grid
	InitRoomPower(g)

	if g.RoomDoorsPowered["Lift Shaft"] {
		t.Fatal("precondition: Lift Shaft should not be auto-powered by InitRoomPower")
	}

	EnsureSolvabilityDoorPower(g)

	if g.RoomDoorsPowered["Lift Shaft"] {
		t.Error("Lift Shaft doors should only be powered when gatekeeper solvability requires it")
	}
}
