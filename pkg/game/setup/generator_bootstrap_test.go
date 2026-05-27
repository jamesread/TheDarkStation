package setup

import (
	"testing"

	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestEnsureGeneratorRoomBootstrap_powersSameRoomTerminal(t *testing.T) {
	g, grid, start, _ := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": false, "RoomB": false}
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(start).Generator = gen
	g.AddGenerator(gen)
	term := entities.NewMaintenanceTerminal("T", "RoomA")
	gameworld.GetGameData(grid.GetCell(0, 0)).MaintenanceTerm = term

	InitMaintenanceTerminalPower(g)
	EnsureGeneratorRoomBootstrap(g)

	if !g.RoomDoorsPowered["RoomA"] {
		t.Fatal("generator room doors should be armed")
	}
	if !RoomIsOnline(g, "RoomA") {
		t.Fatal("generator room should be online")
	}
	if !term.Powered {
		t.Fatal("same-room maintenance terminal should be powered via conductive power grid")
	}
	if !MaintBootstrapOK(g) {
		t.Fatal("maint bootstrap should be OK")
	}
}

func TestRoomsOnConductiveGeneratorGrid_includesLocalFeedWithoutPropagation(t *testing.T) {
	g, grid, start, _ := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": false, "RoomB": false}
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(start).Generator = gen
	term := entities.NewMaintenanceTerminal("T", "RoomA")
	gameworld.GetGameData(grid.GetCell(0, 0)).MaintenanceTerm = term

	rooms := RoomsOnConductiveGeneratorGrid(g)
	if !rooms["RoomA"] {
		t.Fatal("conductive power grid should include generator room via local feed")
	}
	_ = grid
}

func TestMaintBootstrapOK_noGenerator(t *testing.T) {
	g := state.NewGame()
	if !MaintBootstrapOK(g) {
		t.Fatal("empty game should pass bootstrap check")
	}
}

func TestBootstrapGeneratorRoom_armsAndEnergizes(t *testing.T) {
	g, grid, start, _ := makePowerGridTestGrid(t)
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(start).Generator = gen
	g.RoomDoorsPowered = map[string]bool{"RoomA": false}

	BootstrapGeneratorRoom(g, start)
	PropagateRoomPowerOnlineFromGenerators(g)

	if !g.RoomDoorsPowered["RoomA"] {
		t.Error("doors should be armed")
	}
	if !g.RoomCCTVPowered["RoomA"] {
		t.Error("CCTV circuit should be armed")
	}
	if !RoomIsOnline(g, "RoomA") {
		t.Error("room should be online")
	}
	_ = grid
}

func TestBootstrapPoweredGenerators_armsPoweredCellRoom(t *testing.T) {
	g, grid, start, _ := makePowerGridTestGrid(t)
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(start).Generator = gen
	g.RoomDoorsPowered = map[string]bool{"RoomA": false}
	g.RoomCCTVPowered = map[string]bool{"RoomA": false}

	BootstrapPoweredGenerators(g, start)

	if !g.RoomDoorsPowered["RoomA"] {
		t.Fatal("generator room doors should be armed when generator comes online")
	}
	if !g.RoomCCTVPowered["RoomA"] {
		t.Fatal("generator room CCTV should be armed when generator comes online")
	}
	if !RoomIsOnline(g, "RoomA") {
		t.Fatal("generator room should be online")
	}
	_ = grid
}
