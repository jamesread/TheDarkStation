// Package setup tests level setup: InitRoomPower (start room doors powered at init), etc.
package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestInitRoomPower_NilGridNoPanic(t *testing.T) {
	g := state.NewGame()
	g.Grid = nil
	InitRoomPower(g) // must not panic
}

func TestInitRoomPower_StartRoomDoorsPowered(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(2, 2)
	grid.MarkAsRoomWithName(0, 0, "StartRoom", "ROOM_START")
	grid.SetStartCellAt(0, 0)
	g.Grid = grid

	InitRoomPower(g)

	if !g.RoomDoorsPowered["StartRoom"] {
		t.Error("InitRoomPower: RoomDoorsPowered[StartRoom] = false, want true (start room doors powered at init)")
	}
}

func TestInitRoomPower_MultiRoomDefaults(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(3, 3)
	grid.MarkAsRoomWithName(0, 0, "Bridge", "desc")
	grid.MarkAsRoomWithName(0, 1, "Bridge", "desc")
	grid.MarkAsRoomWithName(1, 0, "Lab", "desc")
	grid.MarkAsRoomWithName(1, 1, "Lab", "desc")
	grid.MarkAsRoomWithName(2, 0, "Cargo", "desc")
	grid.MarkAsRoomWithName(2, 1, "Cargo", "desc")
	grid.SetStartCellAt(0, 0)
	grid.BuildAllCellConnections()
	g.Grid = grid

	InitRoomPower(g)

	// Start room (Bridge) doors powered
	if !g.RoomDoorsPowered["Bridge"] {
		t.Error("start room Bridge: RoomDoorsPowered should be true")
	}
	// Other rooms unpowered
	if g.RoomDoorsPowered["Lab"] {
		t.Error("Lab: RoomDoorsPowered should be false")
	}
	if g.RoomDoorsPowered["Cargo"] {
		t.Error("Cargo: RoomDoorsPowered should be false")
	}
	// All CCTV unpowered
	for _, room := range []string{"Bridge", "Lab", "Cargo"} {
		if g.RoomCCTVPowered[room] {
			t.Errorf("%s: RoomCCTVPowered should be false", room)
		}
	}
	// All lights default on
	for _, room := range []string{"Bridge", "Lab", "Cargo"} {
		if !g.RoomLightsPowered[room] {
			t.Errorf("%s: RoomLightsPowered should be true (default on)", room)
		}
	}
}

func TestInitRoomPower_DifferentRoomNames(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(2, 3)
	grid.MarkAsRoomWithName(0, 0, "Alpha", "desc")
	grid.MarkAsRoomWithName(0, 1, "Beta", "desc")
	grid.MarkAsRoomWithName(0, 2, "Gamma", "desc")
	grid.MarkAsRoomWithName(1, 0, "Alpha", "desc")
	grid.MarkAsRoomWithName(1, 1, "Beta", "desc")
	grid.MarkAsRoomWithName(1, 2, "Gamma", "desc")
	grid.SetStartCellAt(0, 1)
	grid.BuildAllCellConnections()
	g.Grid = grid

	InitRoomPower(g)

	if !g.RoomDoorsPowered["Beta"] {
		t.Error("start room Beta: doors should be powered")
	}
	if g.RoomDoorsPowered["Alpha"] {
		t.Error("Alpha: doors should NOT be powered")
	}
	if g.RoomDoorsPowered["Gamma"] {
		t.Error("Gamma: doors should NOT be powered")
	}

	// Verify all 3 rooms have entries
	for _, room := range []string{"Alpha", "Beta", "Gamma"} {
		if _, ok := g.RoomDoorsPowered[room]; !ok {
			t.Errorf("%s: missing from RoomDoorsPowered map", room)
		}
		if _, ok := g.RoomCCTVPowered[room]; !ok {
			t.Errorf("%s: missing from RoomCCTVPowered map", room)
		}
		if _, ok := g.RoomLightsPowered[room]; !ok {
			t.Errorf("%s: missing from RoomLightsPowered map", room)
		}
	}
}

func TestInitRoomPower_Idempotent(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(2, 2)
	grid.MarkAsRoomWithName(0, 0, "Start", "desc")
	grid.MarkAsRoomWithName(0, 1, "Other", "desc")
	grid.MarkAsRoomWithName(1, 0, "Start", "desc")
	grid.MarkAsRoomWithName(1, 1, "Other", "desc")
	grid.SetStartCellAt(0, 0)
	grid.BuildAllCellConnections()
	g.Grid = grid

	InitRoomPower(g)
	InitRoomPower(g)

	if !g.RoomDoorsPowered["Start"] {
		t.Error("after double init: Start doors should be powered")
	}
	if g.RoomDoorsPowered["Other"] {
		t.Error("after double init: Other doors should NOT be powered")
	}
	if !g.RoomLightsPowered["Start"] || !g.RoomLightsPowered["Other"] {
		t.Error("after double init: lights should still default on")
	}
}

func TestInitMaintenanceTerminalPower_StartRoomPowered(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(2, 2)
	grid.MarkAsRoomWithName(0, 0, "Start", "desc")
	grid.MarkAsRoomWithName(0, 1, "Other", "desc")
	grid.MarkAsRoomWithName(1, 0, "Start", "desc")
	grid.MarkAsRoomWithName(1, 1, "Other", "desc")
	grid.SetStartCellAt(0, 0)
	grid.BuildAllCellConnections()
	g.Grid = grid

	for r := 0; r < 2; r++ {
		for c := 0; c < 2; c++ {
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}

	// Place maintenance terminals
	startTerm := entities.NewMaintenanceTerminal("MT-Start", "Start")
	startTerm.Powered = true // will be overridden
	gameworld.GetGameData(grid.GetCell(1, 0)).MaintenanceTerm = startTerm

	otherTerm := entities.NewMaintenanceTerminal("MT-Other", "Other")
	otherTerm.Powered = true // will be overridden
	gameworld.GetGameData(grid.GetCell(0, 1)).MaintenanceTerm = otherTerm

	InitMaintenanceTerminalPower(g)

	if !startTerm.Powered {
		t.Error("start room terminal should be powered after InitMaintenanceTerminalPower")
	}
	if otherTerm.Powered {
		t.Error("non-start room terminal should NOT be powered after InitMaintenanceTerminalPower")
	}
}

func TestInitMaintenanceTerminalPower_NilGridNoPanic(t *testing.T) {
	g := state.NewGame()
	g.Grid = nil
	InitMaintenanceTerminalPower(g) // must not panic
}

func TestInitMaintenanceTerminalPower_MultipleTerminalsInStartRoom(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(3, 3)
	grid.MarkAsRoomWithName(0, 0, "Start", "desc")
	grid.MarkAsRoomWithName(0, 1, "Start", "desc")
	grid.MarkAsRoomWithName(1, 0, "Start", "desc")
	grid.MarkAsRoomWithName(1, 1, "Other", "desc")
	grid.MarkAsRoomWithName(2, 0, "Start", "desc")
	grid.MarkAsRoomWithName(2, 1, "Other", "desc")
	grid.SetStartCellAt(0, 0)
	grid.BuildAllCellConnections()
	g.Grid = grid

	for r := 0; r < 3; r++ {
		for c := 0; c < 2; c++ {
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}

	// Two terminals in start room, one in Other
	term1 := entities.NewMaintenanceTerminal("MT1", "Start")
	term2 := entities.NewMaintenanceTerminal("MT2", "Start")
	termOther := entities.NewMaintenanceTerminal("MT-Other", "Other")
	gameworld.GetGameData(grid.GetCell(0, 1)).MaintenanceTerm = term1
	gameworld.GetGameData(grid.GetCell(1, 0)).MaintenanceTerm = term2
	gameworld.GetGameData(grid.GetCell(1, 1)).MaintenanceTerm = termOther

	InitMaintenanceTerminalPower(g)

	if !term1.Powered || !term2.Powered {
		t.Error("all start room terminals should be powered")
	}
	if termOther.Powered {
		t.Error("Other room terminal should NOT be powered")
	}
}

func TestInitMaintenanceTerminalPower_Idempotent(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(2, 2)
	grid.MarkAsRoomWithName(0, 0, "Start", "desc")
	grid.MarkAsRoomWithName(0, 1, "Other", "desc")
	grid.MarkAsRoomWithName(1, 0, "Start", "desc")
	grid.MarkAsRoomWithName(1, 1, "Other", "desc")
	grid.SetStartCellAt(0, 0)
	grid.BuildAllCellConnections()
	g.Grid = grid

	for r := 0; r < 2; r++ {
		for c := 0; c < 2; c++ {
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}

	startTerm := entities.NewMaintenanceTerminal("MT-Start", "Start")
	otherTerm := entities.NewMaintenanceTerminal("MT-Other", "Other")
	gameworld.GetGameData(grid.GetCell(1, 0)).MaintenanceTerm = startTerm
	gameworld.GetGameData(grid.GetCell(0, 1)).MaintenanceTerm = otherTerm

	InitMaintenanceTerminalPower(g)
	InitMaintenanceTerminalPower(g)

	if !startTerm.Powered {
		t.Error("after double init: start room terminal should be powered")
	}
	if otherTerm.Powered {
		t.Error("after double init: other room terminal should NOT be powered")
	}
}

// TestSaveLoadDeckState_MaintenanceTerminalPowerPreserved verifies that terminal Powered state
// persists across SaveCurrentDeckState/LoadDeckState (state package). Placed here as integration
// test of InitMaintenanceTerminalPower + deck state round-trip.
func TestSaveLoadDeckState_MaintenanceTerminalPowerPreserved(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(2, 2)
	grid.MarkAsRoomWithName(0, 0, "Start", "desc")
	grid.MarkAsRoomWithName(0, 1, "Other", "desc")
	grid.MarkAsRoomWithName(1, 0, "Start", "desc")
	grid.MarkAsRoomWithName(1, 1, "Other", "desc")
	grid.SetStartCellAt(0, 0)
	grid.BuildAllCellConnections()
	g.Grid = grid

	for r := 0; r < 2; r++ {
		for c := 0; c < 2; c++ {
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}

	startTerm := entities.NewMaintenanceTerminal("MT-Start", "Start")
	otherTerm := entities.NewMaintenanceTerminal("MT-Other", "Other")
	gameworld.GetGameData(grid.GetCell(1, 0)).MaintenanceTerm = startTerm
	gameworld.GetGameData(grid.GetCell(0, 1)).MaintenanceTerm = otherTerm

	InitMaintenanceTerminalPower(g)
	// Simulate "Restore power" - power the Other room terminal
	otherTerm.Powered = true

	g.CurrentDeckID = 0
	g.SaveCurrentDeckState()

	// Simulate overwriting grid (e.g. advancing and coming back)
	g.Grid = nil
	g.LoadDeckState(0)

	if g.Grid == nil {
		t.Fatal("Grid should be restored after LoadDeckState")
	}
	// The grid in DeckState has the same cells - terminals are in that grid
	// Find the Other room terminal and verify it's still powered
	found := false
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.MaintenanceTerm != nil && data.MaintenanceTerm.Name == "MT-Other" {
			found = true
			if !data.MaintenanceTerm.Powered {
				t.Error("after Save/Load: MT-Other terminal should still be powered")
			}
		}
	})
	if !found {
		t.Error("could not find MT-Other terminal in restored grid")
	}
}
