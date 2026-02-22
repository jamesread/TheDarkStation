package menu

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// makeMenuTestGame creates a game with a 3x3 grid, two rooms ("RoomA", "RoomB") and a corridor.
// RoomA at (0,0)-(0,1), Corridor at (1,0)-(1,1), RoomB at (2,0)-(2,1).
// Each room has a powered maintenance terminal. Generator in RoomA provides 100W supply.
func makeMenuTestGame(t *testing.T) (*state.Game, *world.Cell) {
	t.Helper()
	g := state.NewGame()
	grid := world.NewGrid(3, 2)
	grid.MarkAsRoomWithName(0, 0, "RoomA", "desc")
	grid.MarkAsRoomWithName(0, 1, "RoomA", "desc")
	grid.MarkAsRoomWithName(1, 0, "Corridor", "desc")
	grid.MarkAsRoomWithName(1, 1, "Corridor", "desc")
	grid.MarkAsRoomWithName(2, 0, "RoomB", "desc")
	grid.MarkAsRoomWithName(2, 1, "RoomB", "desc")
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(2, 1)
	grid.BuildAllCellConnections()
	g.Grid = grid

	for r := 0; r < 3; r++ {
		for c := 0; c < 2; c++ {
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}

	// Maintenance terminals (powered)
	termA := entities.NewMaintenanceTerminal("MT-A", "RoomA")
	termA.Powered = true
	gameworld.GetGameData(grid.GetCell(0, 1)).MaintenanceTerm = termA

	termB := entities.NewMaintenanceTerminal("MT-B", "RoomB")
	termB.Powered = true
	gameworld.GetGameData(grid.GetCell(2, 1)).MaintenanceTerm = termB

	// Generator in RoomA for 100W supply
	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteries(1)
	g.AddGenerator(gen)
	g.CurrentDeckID = 0
	g.UpdatePowerSupply()

	g.RoomDoorsPowered = map[string]bool{"RoomA": false, "RoomB": false}
	g.RoomCCTVPowered = map[string]bool{"RoomA": false, "RoomB": false}
	g.RoomLightsPowered = map[string]bool{"RoomA": true, "RoomB": true}

	termCell := grid.GetCell(0, 1)
	return g, termCell
}

func TestToggleDoors_OnOff(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)

	toggle := &RoomPowerToggleMenuItem{G: g, RoomName: "RoomA", PowerType: "doors"}

	// Toggle ON
	h.OnActivate(toggle, 0)
	if !g.RoomDoorsPowered["RoomA"] {
		t.Error("after toggle ON: RoomDoorsPowered[RoomA] should be true")
	}

	// Toggle OFF
	h.OnActivate(toggle, 0)
	if g.RoomDoorsPowered["RoomA"] {
		t.Error("after toggle OFF: RoomDoorsPowered[RoomA] should be false")
	}
}

func TestToggleCCTV_OnOff(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)

	toggle := &RoomPowerToggleMenuItem{G: g, RoomName: "RoomA", PowerType: "cctv"}

	h.OnActivate(toggle, 0)
	if !g.RoomCCTVPowered["RoomA"] {
		t.Error("after toggle ON: RoomCCTVPowered[RoomA] should be true")
	}

	h.OnActivate(toggle, 0)
	if g.RoomCCTVPowered["RoomA"] {
		t.Error("after toggle OFF: RoomCCTVPowered[RoomA] should be false")
	}
}

func TestToggleDoorsON_ShortOutProtectsToggledRoom(t *testing.T) {
	// 100W supply (1 generator). Create 11 doors in RoomA (110W) + 1 door in RoomB (10W).
	// Pre-power RoomA's doors (110W > 100W triggers overload).
	// Toggle RoomB doors ON: ShortOutIfOverload protects RoomB and shorts out RoomA.
	g := state.NewGame()
	grid := world.NewGrid(6, 4)
	for r := 0; r < 3; r++ {
		for c := 0; c < 4; c++ {
			grid.MarkAsRoomWithName(r, c, "RoomA", "desc")
		}
	}
	for r := 3; r < 6; r++ {
		for c := 0; c < 4; c++ {
			grid.MarkAsRoomWithName(r, c, "RoomB", "desc")
		}
	}
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(5, 3)
	grid.BuildAllCellConnections()
	g.Grid = grid
	for r := 0; r < 6; r++ {
		for c := 0; c < 4; c++ {
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}
	// Maintenance terminals (powered)
	termA := entities.NewMaintenanceTerminal("MT-A", "RoomA")
	termA.Powered = true
	gameworld.GetGameData(grid.GetCell(0, 0)).MaintenanceTerm = termA
	termB := entities.NewMaintenanceTerminal("MT-B", "RoomB")
	termB.Powered = true
	gameworld.GetGameData(grid.GetCell(3, 0)).MaintenanceTerm = termB

	// 11 doors in RoomA (each 10W = 110W when powered)
	for i := 0; i < 11; i++ {
		r := i / 4
		c := i % 4
		if r < 3 {
			gameworld.GetGameData(grid.GetCell(r, c)).Door = &entities.Door{RoomName: "RoomA", Locked: false}
		}
	}
	// 1 door in RoomB (10W)
	gameworld.GetGameData(grid.GetCell(3, 1)).Door = &entities.Door{RoomName: "RoomB", Locked: false}

	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteries(1)
	g.AddGenerator(gen)
	g.UpdatePowerSupply() // 100W

	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": false}
	g.RoomCCTVPowered = map[string]bool{"RoomA": false, "RoomB": false}
	g.RoomLightsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	g.PowerConsumption = g.CalculatePowerConsumption()

	// Toggle RoomB doors ON — total would be 120W > 100W supply.
	// ShortOutIfOverload protects RoomB (the toggled room) and shorts out RoomA.
	termCell := grid.GetCell(3, 0)
	h := NewMaintenanceMenuHandler(g, termCell, termB)
	toggleB := &RoomPowerToggleMenuItem{G: g, RoomName: "RoomB", PowerType: "doors"}
	h.OnActivate(toggleB, 0)

	if !g.RoomDoorsPowered["RoomB"] {
		t.Error("protected room RoomB: doors should remain ON")
	}
	if g.RoomDoorsPowered["RoomA"] {
		t.Error("RoomA doors should be shorted out (overload)")
	}
	if g.PowerConsumption > g.PowerSupply {
		t.Errorf("consumption (%d) should be <= supply (%d) after short-out", g.PowerConsumption, g.PowerSupply)
	}
}

func TestToggleDoorsOFF_RecalculatesConsumption(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)

	// Place door for RoomA
	gameworld.GetGameData(g.Grid.GetCell(1, 0)).Door = &entities.Door{RoomName: "RoomA", Locked: false}

	// Power RoomA doors
	g.RoomDoorsPowered["RoomA"] = true
	g.PowerConsumption = g.CalculatePowerConsumption()
	if g.PowerConsumption == 0 {
		t.Fatal("precondition: consumption should be > 0 with doors on")
	}

	// Toggle OFF
	toggle := &RoomPowerToggleMenuItem{G: g, RoomName: "RoomA", PowerType: "doors"}
	h.OnActivate(toggle, 0)

	if g.RoomDoorsPowered["RoomA"] {
		t.Error("RoomA doors should be OFF after toggle")
	}
	if g.PowerConsumption != 0 {
		t.Errorf("consumption should be 0 after turning off all doors, got %d", g.PowerConsumption)
	}
}

func TestToggleIsSelectable_RequiresMaintTerminalPowered(t *testing.T) {
	g, termCell := makeMenuTestGame(t)

	// Unpower the terminal
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	term.Powered = false

	toggle := &RoomPowerToggleMenuItem{G: g, RoomName: "RoomA", PowerType: "doors"}
	if toggle.IsSelectable() {
		t.Error("toggle should NOT be selectable when room's maintenance terminal is unpowered")
	}

	// Re-power the terminal
	term.Powered = true
	if !toggle.IsSelectable() {
		t.Error("toggle should be selectable when room's maintenance terminal is powered")
	}
}

func TestToggleOnActivate_RejectsUnpoweredTerminal(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)

	// Unpower the target room's terminal
	term.Powered = false

	toggle := &RoomPowerToggleMenuItem{G: g, RoomName: "RoomA", PowerType: "doors"}
	_, helpText := h.OnActivate(toggle, 0)

	if g.RoomDoorsPowered["RoomA"] {
		t.Error("doors should NOT change when terminal is unpowered")
	}
	if helpText == "" {
		t.Error("should return help text explaining terminal must be activated")
	}
}
