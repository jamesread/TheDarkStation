package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestResolvePowerOverloadAfterToggle_tripsGenerator(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(6, 12)
	for r := 0; r < 6; r++ {
		for c := 0; c < 10; c++ {
			grid.MarkAsRoomWithName(r, c, "RoomA", "")
			gameworld.InitGameData(grid.GetCell(r, c))
			gameworld.GetGameData(grid.GetCell(r, c)).Terminal = entities.NewCCTVTerminal("TA")
		}
		for c := 10; c < 12; c++ {
			grid.MarkAsRoomWithName(r, c, "RoomB", "")
			gameworld.InitGameData(grid.GetCell(r, c))
			gameworld.GetGameData(grid.GetCell(r, c)).Terminal = entities.NewCCTVTerminal("TB")
		}
	}
	grid.BuildAllCellConnections()
	gameworld.GetGameData(grid.GetCell(0, 0)).Door = &entities.Door{RoomName: "RoomA", Locked: false}
	gameworld.GetGameData(grid.GetCell(0, 10)).Door = &entities.Door{RoomName: "RoomB", Locked: false}
	g.Grid = grid
	g.CurrentDeckID = 0

	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.AddGenerator(gen)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	g.RoomCCTVPowered = map[string]bool{"RoomA": true, "RoomB": true}
	EnergizeArmedRoomsForTest(g)
	g.UpdatePowerSupply()
	g.PowerConsumption = CalculatePowerConsumption(g)
	if g.PowerConsumption <= g.PowerSupply {
		t.Fatalf("test setup: need overload, consumption=%d supply=%d", g.PowerConsumption, g.PowerSupply)
	}

	if !ResolvePowerOverloadAfterToggle(g, "RoomB") {
		t.Fatal("expected overload resolution after shedding other rooms")
	}
	if gen.IsPowered() || !gen.Tripped {
		t.Fatal("generator should trip after shedding other rooms still overloads")
	}
	if g.PowerSupply != 0 {
		t.Fatalf("supply = %d, want 0 after trip", g.PowerSupply)
	}
}

func TestTriggerPowerOverloadForDev_tripsGenerator(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(2, 10)
	for r := 0; r < 2; r++ {
		for c := 0; c < 10; c++ {
			grid.MarkAsRoomWithName(r, c, "RoomA", "")
			gameworld.InitGameData(grid.GetCell(r, c))
			gameworld.GetGameData(grid.GetCell(r, c)).Terminal = entities.NewCCTVTerminal("TA")
		}
	}
	grid.BuildAllCellConnections()
	g.Grid = grid
	g.CurrentDeckID = 0

	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.AddGenerator(gen)

	if !TriggerPowerOverloadForDev(g, "RoomA") {
		t.Fatal("expected dev overload trigger to apply")
	}
	if gen.IsPowered() || !gen.Tripped {
		t.Fatal("generator should trip on dev overload")
	}
}
