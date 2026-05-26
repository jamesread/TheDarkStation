package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestSplitGrids_independentSupply(t *testing.T) {
	g, grid, start, relayCell := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	genA := entities.NewGenerator("GA", 1)
	genA.InsertBatteriesAndStart(1)
	gameworld.GetGameData(start).Generator = genA
	genB := entities.NewGenerator("GB", 1)
	genB.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 2)).Generator = genB
	gameworld.GetGameData(relayCell).PowerRelay = entities.NewPowerRelayOpen()
	EnergizeArmedRoomsForTest(g)

	if ArmedGridSupplyForRoom(g, "RoomA") != 100 {
		t.Fatalf("RoomA grid supply = %d, want 100", ArmedGridSupplyForRoom(g, "RoomA"))
	}
	if ArmedGridSupplyForRoom(g, "RoomB") != 100 {
		t.Fatalf("RoomB grid supply = %d, want 100", ArmedGridSupplyForRoom(g, "RoomB"))
	}
}

func TestShortOutIfOverload_onlyShedsSameGrid(t *testing.T) {
	grid := world.NewGrid(2, 12)
	for r := 0; r < 2; r++ {
		for c := 0; c < 12; c++ {
			room := "RoomA"
			if c >= 10 {
				room = "RoomB"
			}
			grid.MarkAsRoomWithName(r, c, room, "desc")
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}
	grid.BuildAllCellConnections()
	gameworld.GetGameData(grid.GetCell(0, 0)).Door = &entities.Door{RoomName: "RoomA", Locked: false}
	gameworld.GetGameData(grid.GetCell(0, 10)).Door = &entities.Door{RoomName: "RoomB", Locked: false}
	for r := 0; r < 2; r++ {
		for c := 0; c < 10; c++ {
			gameworld.GetGameData(grid.GetCell(r, c)).Terminal = entities.NewCCTVTerminal("TA")
		}
	}
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g := state.NewGame()
	g.Grid = grid
	g.AddGenerator(gen)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": false}
	g.RoomCCTVPowered = map[string]bool{"RoomA": true, "RoomB": false}
	g.RoomPowerOnline = map[string]bool{"RoomA": true}

	preview := PreviewShortOutIfOverload(g, "RoomB", true, false)
	g.RoomDoorsPowered["RoomB"] = true
	g.RoomPowerOnline["RoomB"] = true
	shorted := ShortOutIfOverload(g, "RoomB")

	if shorted && len(preview) == 0 {
		t.Fatal("preview should list shed when short-out occurs")
	}
	if !shorted {
		t.Fatal("expected short-out with limited grid supply")
	}
	if !g.RoomDoorsPowered["RoomB"] {
		t.Fatal("protected room should stay powered")
	}
	if g.RoomDoorsPowered["RoomA"] {
		t.Fatal("RoomA should have been shed")
	}
}

func TestPreviewRoomPresetConsumption_gridScoped(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 1)
	grid.MarkAsRoomWithName(0, 0, "RoomA", "")
	grid.BuildAllCellConnections()
	g.Grid = grid
	gameworld.InitGameData(grid.GetCell(0, 0))
	gameworld.GetGameData(grid.GetCell(0, 0)).Door = &entities.Door{RoomName: "RoomA", Locked: false}
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen

	g.RoomDoorsPowered = map[string]bool{"RoomA": false}
	g.RoomCCTVPowered = map[string]bool{"RoomA": false}
	g.RoomPowerOnline = map[string]bool{"RoomA": false}

	before, afterApply, afterShed := PreviewRoomPresetConsumption(g, "RoomA", true, false)
	if before != 0 || afterApply != 10 || afterShed != 10 {
		t.Fatalf("before=%d afterApply=%d afterShed=%d", before, afterApply, afterShed)
	}
}

func TestShortOutIfOverload_noOverloadReturnsFalse(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 1)
	grid.MarkAsRoomWithName(0, 0, "R", "")
	grid.BuildAllCellConnections()
	g.Grid = grid
	gameworld.InitGameData(grid.GetCell(0, 0))
	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.AddGenerator(gen)
	g.RoomDoorsPowered = map[string]bool{"R": true}
	g.RoomPowerOnline = map[string]bool{"R": true}

	if ShortOutIfOverload(g, "R") {
		t.Error("expected no short-out when grid load is within supply")
	}
}

func TestShortOutIfOverload_unpowersOthersOnSameGrid(t *testing.T) {
	grid := world.NewGrid(2, 12)
	for r := 0; r < 2; r++ {
		for c := 0; c < 12; c++ {
			room := "A"
			if c >= 10 {
				room = "B"
			}
			grid.MarkAsRoomWithName(r, c, room, "desc")
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}
	grid.BuildAllCellConnections()
	gameworld.GetGameData(grid.GetCell(0, 0)).Door = &entities.Door{RoomName: "A", Locked: false}
	gameworld.GetGameData(grid.GetCell(0, 10)).Door = &entities.Door{RoomName: "B", Locked: false}
	for r := 0; r < 2; r++ {
		for c := 0; c < 10; c++ {
			gameworld.GetGameData(grid.GetCell(r, c)).Terminal = entities.NewCCTVTerminal("TA")
		}
	}
	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g := state.NewGame()
	g.Grid = grid
	g.AddGenerator(gen)
	g.RoomDoorsPowered = map[string]bool{"A": true, "B": true}
	g.RoomCCTVPowered = map[string]bool{"A": true, "B": false}
	g.RoomPowerOnline = map[string]bool{"A": true, "B": true}

	if !ShortOutIfOverload(g, "B") {
		t.Fatal("expected short-out on overloaded grid")
	}
	if !g.RoomDoorsPowered["B"] || g.RoomDoorsPowered["A"] || g.RoomCCTVPowered["A"] {
		t.Fatalf("after short-out: B=%v A doors=%v A cctv=%v", g.RoomDoorsPowered["B"], g.RoomDoorsPowered["A"], g.RoomCCTVPowered["A"])
	}
	if ConsumptionOnArmedGrid(g, ArmedGridForRoom(g, "B")) > ArmedGridSupplyForRoom(g, "B") {
		t.Fatal("grid should be within supply after short-out")
	}
}
