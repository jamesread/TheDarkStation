package setup

import (
	"testing"
	"time"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func makePowerGridTestGrid(t *testing.T) (*state.Game, *world.Grid, *world.Cell, *world.Cell) {
	t.Helper()
	g := state.NewGame()
	grid := world.NewGrid(1, 3)
	grid.MarkAsRoomWithName(0, 0, "RoomA", "")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "ROOM_CORRIDOR")
	grid.MarkAsRoomWithName(0, 2, "RoomB", "")
	grid.BuildAllCellConnections()
	g.Grid = grid
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})
	return g, grid, grid.GetCell(0, 0), grid.GetCell(0, 1)
}

func TestCellsReachableInPowerGrid_matchesRoomSet(t *testing.T) {
	g, _, start, relayCell := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	rooms := RoomsReachableInPowerGrid(g, start)
	cells := CellsReachableInPowerGrid(g, start)
	if cells.Size() == 0 {
		t.Fatal("expected power grid cells from start")
	}
	seenRooms := make(map[string]bool)
	cells.Each(func(c *world.Cell) {
		if c.Name != "" && c.Name != "Corridor" {
			seenRooms[c.Name] = true
		}
	})
	if len(seenRooms) != len(rooms) {
		t.Fatalf("room names from cells %v != RoomsReachableInPowerGrid %v", seenRooms, rooms)
	}
	for _, r := range rooms {
		if !seenRooms[r] {
			t.Fatalf("room %q missing from cell-derived set", r)
		}
	}
	if !cells.Has(relayCell) {
		t.Fatal("relay junction cell should be on power grid when closed")
	}
}

func TestRoomsReachableInPowerGrid_blockedByOpenRelay(t *testing.T) {
	g, _, start, relayCell := makePowerGridTestGrid(t)
	if start == nil || relayCell == nil {
		t.Fatal("test grid missing cells")
	}
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	g.RoomCCTVPowered = map[string]bool{}

	gameworld.GetGameData(relayCell).PowerRelay = entities.NewPowerRelayOpen()

	rooms := RoomsReachableInPowerGrid(g, start)
	for _, r := range rooms {
		if r == "RoomB" {
			t.Fatal("open relay should block reach to RoomB")
		}
	}

	gameworld.GetGameData(relayCell).PowerRelay.Closed = true
	rooms = RoomsReachableInPowerGrid(g, start)
	foundB := false
	for _, r := range rooms {
		if r == "RoomB" {
			foundB = true
		}
	}
	if !foundB {
		t.Fatalf("closed relay should allow RoomB, got %v", rooms)
	}
}

func TestRoomsReachableInPowerGrid_requiresPoweredDoors(t *testing.T) {
	g, _, start, _ := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": false}
	g.RoomCCTVPowered = map[string]bool{}

	rooms := RoomsReachableInPowerGrid(g, start)
	for _, r := range rooms {
		if r == "RoomB" {
			t.Fatal("unpowered RoomB doors should block power grid")
		}
	}
}

func TestPowerGrid_passesThroughLockedDoor(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 5)
	grid.MarkAsRoomWithName(0, 0, "RoomA", "")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "ROOM_CORRIDOR")
	grid.MarkAsRoomWithName(0, 2, "RoomB", "")
	grid.MarkAsRoomWithName(0, 3, "Corridor", "ROOM_CORRIDOR")
	grid.MarkAsRoomWithName(0, 4, "RoomC", "")
	grid.BuildAllCellConnections()
	g.Grid = grid
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})

	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.AddGenerator(gen)

	lockedDoor := grid.GetCell(0, 2)
	gameworld.GetGameData(lockedDoor).Door = &entities.Door{RoomName: "RoomB", Locked: true}

	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomC": true}
	g.RoomCCTVPowered = map[string]bool{}

	PropagateRoomPowerOnlineFromGenerators(g)
	ApplyGridConductivePower(g)

	if !CellHasLivePower(g, lockedDoor) {
		t.Fatal("locked door should have live grid power")
	}
	if !CellHasLivePower(g, grid.GetCell(0, 4)) {
		t.Fatal("room beyond locked door should receive grid power")
	}
	if !CanTraverseCellForPowerGridArm(g, lockedDoor) {
		t.Fatal("locked door should be traversable on armed power grid")
	}
}

func TestCellsReachableInPowerGridOverlay_manualEgressDoor(t *testing.T) {
	g, grid, start, _ := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": false}
	g.ManualEgressReleased = map[string]bool{"RoomB": true}

	armed := CellsReachableInPowerGrid(g, start)
	overlay := CellsReachableInPowerGridOverlay(g, start)
	roomBCell := grid.GetCell(0, 2)
	if armed.Has(roomBCell) {
		t.Fatal("armed routing should not traverse unpowered doors without manual egress in propagation grid")
	}
	if overlay == nil || !overlay.Has(roomBCell) {
		t.Fatal("overlay grid should extend through manually released doors")
	}
}

func TestArmedGridComponents_denseGeneratorsCompletesQuickly(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(72, 112)
	grid.BuildAllCellConnections()
	g.Grid = grid
	g.RoomDoorsPowered = make(map[string]bool)
	g.RoomCCTVPowered = make(map[string]bool)
	g.RoomLightsPowered = make(map[string]bool)
	g.RoomPowerOnline = make(map[string]bool)

	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		cell.Room = true
		cell.Name = "Perf Entity Floor"
		gameworld.InitGameData(cell)
		g.RoomDoorsPowered[cell.Name] = true
		g.RoomCCTVPowered[cell.Name] = true
		g.RoomLightsPowered[cell.Name] = true
		g.RoomPowerOnline[cell.Name] = true
	})
	placed := 0
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if row < 2 || col < 2 || row > grid.Rows()-3 || col > grid.Cols()-3 {
			return
		}
		diag := row - col
		if diag%3 != 0 {
			return
		}
		linePos := (row + col) / 3
		if linePos%6 == 5 {
			return
		}
		gen := entities.NewGenerator("G", 1)
		gen.InsertBatteriesAndStart(1)
		gameworld.GetGameData(cell).Generator = gen
		placed++
	})
	if placed < 1000 {
		t.Fatalf("expected dense generator layout, got %d cells", placed)
	}
	g.RebuildGeneratorsFromGrid()

	components := armedGridComponentsFromGeneratorLocations(g, nil)
	if len(components) != 1 {
		t.Fatalf("dense open floor should be one armed component, got %d", len(components))
	}

	g.InvalidateLivePowerCache()
	start := time.Now()
	_ = AnyArmedGridOverloaded(g)
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("AnyArmedGridOverloaded took %v, want <= 50ms on ~%d generators", elapsed, placed)
	}
}

func TestPowerGridComponentCount_splitByOpenRelay(t *testing.T) {
	g, _, start, relayCell := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	gameworld.GetGameData(relayCell).PowerRelay = entities.NewPowerRelayOpen()
	if PowerGridComponentCount(g) != 2 {
		t.Fatalf("open relay should split power grid into 2 components, got %d", PowerGridComponentCount(g))
	}
	gameworld.GetGameData(relayCell).PowerRelay.Closed = true
	if PowerGridComponentCount(g) != 1 {
		t.Fatalf("closed relay should unify power grid, got %d", PowerGridComponentCount(g))
	}
	_ = start
}

func TestGeneratorGridSupplyAtCell_connectedGenerators(t *testing.T) {
	g, grid, start, relayCell := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	genA := entities.NewGenerator("GA", 1)
	genA.InsertBatteriesAndStart(1)
	gameworld.GetGameData(start).Generator = genA
	genB := entities.NewGenerator("GB", 1)
	genB.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 2)).Generator = genB
	gameworld.GetGameData(relayCell).PowerRelay = entities.NewPowerRelay()

	ind, gridTotal, count := GeneratorGridSupplyAtCell(g, start)
	if ind != 100 || gridTotal != 200 {
		t.Fatalf("connected gens: individual=%d grid=%d, want 100 and 200", ind, gridTotal)
	}
	if count != 1 {
		t.Fatalf("grid count = %d, want 1", count)
	}
}

func TestGeneratorGridSupplyAtCell_splitGrids(t *testing.T) {
	g, grid, start, relayCell := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	genA := entities.NewGenerator("GA", 1)
	genA.InsertBatteriesAndStart(1)
	gameworld.GetGameData(start).Generator = genA
	genB := entities.NewGenerator("GB", 1)
	genB.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 2)).Generator = genB
	gameworld.GetGameData(relayCell).PowerRelay = entities.NewPowerRelayOpen()

	ind, gridTotal, count := GeneratorGridSupplyAtCell(g, start)
	if ind != 100 || gridTotal != 100 {
		t.Fatalf("split power grid from A: individual=%d grid=%d, want 100 and 100", ind, gridTotal)
	}
	if count != 2 {
		t.Fatalf("grid count = %d, want 2", count)
	}
}

func TestCanTraverseCellForPowerGrid_blocksFurniture(t *testing.T) {
	g, grid, start, relayCell := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	gameworld.GetGameData(relayCell).Furniture = &entities.Furniture{Name: "Locker"}

	if CanTraverseCellForPowerGrid(g, relayCell) {
		t.Fatal("furniture should block power grid traversal")
	}
	rooms := RoomsReachableInPowerGrid(g, start)
	for _, r := range rooms {
		if r == "RoomB" {
			t.Fatal("grid should not reach RoomB through furniture cell")
		}
	}
	_ = grid
}

func TestCanTraverseCellForPowerGrid_blocksRepairDevice(t *testing.T) {
	g, grid, start, relayCell := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	gameworld.GetGameData(relayCell).RepairDevice = entities.NewRepairObjective(
		"routing", entities.RepairSignalCalibrator, "Corridor", relayCell.Row, relayCell.Col,
	)

	if CanTraverseCellForPowerGrid(g, relayCell) {
		t.Fatal("repair device should block power grid traversal")
	}
	rooms := RoomsReachableInPowerGrid(g, start)
	for _, r := range rooms {
		if r == "RoomB" {
			t.Fatal("grid should not reach RoomB through repair device cell")
		}
	}
	_ = grid
}

func TestRoomsFedByPoweredGeneratorGrid_requiresGridAndGenerator(t *testing.T) {
	g, _, start, relayCell := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	EnergizeArmedRoomsForTest(g)
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(start).Generator = gen

	fed := RoomsFedByPoweredGeneratorGrid(g)
	if !fed["RoomA"] || !fed["RoomB"] {
		t.Fatalf("both rooms should be fed by powered generator, got %v", fed)
	}

	g.RoomDoorsPowered["RoomB"] = false
	PropagateRoomPowerOnlineFromGenerators(g)
	fed = RoomsFedByPoweredGeneratorGrid(g)
	if fed["RoomB"] {
		t.Fatal("grid-disabled room should not count as fed")
	}
	if !fed["RoomA"] {
		t.Fatal("RoomA should still be fed")
	}

	gen.BatteriesInserted = 0
	fed = RoomsFedByPoweredGeneratorGrid(g)
	if len(fed) != 0 {
		t.Fatalf("unpowered generator should feed no rooms, got %v", fed)
	}

	gen.InsertBatteriesAndStart(1)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	gameworld.GetGameData(relayCell).PowerRelay = entities.NewPowerRelayOpen()
	fed = RoomsFedByPoweredGeneratorGrid(g)
	if fed["RoomA"] && fed["RoomB"] {
		t.Fatal("open relay should split power grid; RoomB should not be fed from RoomA generator")
	}
	if !fed["RoomA"] {
		t.Fatal("RoomA should still be fed locally")
	}
}

func TestRoomPoweredOnPowerGrid(t *testing.T) {
	g, _, start, _ := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": false}
	g.RoomPowerOnline = map[string]bool{"RoomA": true}
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(start).Generator = gen
	fed := RoomsFedByPoweredGeneratorGrid(g)

	if !RoomPoweredOnPowerGrid(g, "RoomA", fed) {
		t.Fatal("RoomA should be powered on grid")
	}
	if RoomPoweredOnPowerGrid(g, "RoomB", fed) {
		t.Fatal("RoomB grid off should not be powered on grid")
	}
}

func TestTripGeneratorsFeedingRoom_tripsGridGenerators(t *testing.T) {
	g, grid, start, _ := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	EnergizeArmedRoomsForTest(g)
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(start).Generator = gen

	if TripGeneratorsFeedingRoom(g, "RoomB") == 0 {
		t.Fatal("RoomB should be on power grid from RoomA generator when doors are on")
	}
	gen.Tripped = false
	gen.BringOnline()

	if TripGeneratorsFeedingRoom(g, "RoomA") != 1 {
		t.Fatal("expected generator feeding RoomA to trip")
	}
	if gen.IsPowered() || !gen.Tripped {
		t.Fatal("generator should be tripped offline")
	}
	_ = grid
}

func TestUnpowerTerminalsOffGeneratorGrid(t *testing.T) {
	g, grid, start, _ := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(start).Generator = gen
	termB := entities.NewMaintenanceTerminal("T", "RoomB")
	termB.Powered = true
	gameworld.GetGameData(grid.GetCell(0, 2)).MaintenanceTerm = termB
	g.RoomDoorsPowered["RoomB"] = false

	if UnpowerTerminalsOffGeneratorGrid(g) != 1 {
		t.Fatal("terminal in grid-off room should lose power")
	}
	if termB.Powered {
		t.Fatal("terminal should be unpowered without generator-fed power grid")
	}
}

func TestApplyGridConductivePower_fromPoweredGenerator(t *testing.T) {
	g, grid, start, relayCell := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	EnergizeArmedRoomsForTest(g)
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(start).Generator = gen
	termB := entities.NewMaintenanceTerminal("T", "RoomB")
	termB.Powered = false
	gameworld.GetGameData(grid.GetCell(0, 2)).MaintenanceTerm = termB
	gameworld.GetGameData(relayCell).PowerRelay = entities.NewPowerRelay()

	if ApplyGridConductivePower(g) != 1 {
		t.Fatal("power grid from powered generator should power terminal in RoomB")
	}
	if !termB.Powered {
		t.Fatal("terminal on power grid should be powered")
	}
}

func TestApplyGridConductivePower_splitGrids(t *testing.T) {
	g, grid, start, relayCell := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	EnergizeArmedRoomsForTest(g)
	genA := entities.NewGenerator("GA", 1)
	genA.InsertBatteriesAndStart(1)
	gameworld.GetGameData(start).Generator = genA
	genB := entities.NewGenerator("GB", 1)
	genB.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 2)).Generator = genB
	termB := entities.NewMaintenanceTerminal("T", "RoomB")
	termB.Powered = false
	gameworld.GetGameData(grid.GetCell(0, 2)).MaintenanceTerm = termB
	gameworld.GetGameData(relayCell).PowerRelay = entities.NewPowerRelayOpen()

	ApplyGridConductivePower(g)
	if !termB.Powered {
		t.Fatal("terminal B should power from its local generator power grid")
	}
	for _, r := range RoomsReachableInPowerGrid(g, start) {
		if r == "RoomB" {
			t.Fatal("open relay should keep RoomA grid separate from RoomB")
		}
	}
}

func TestApplyGridConductivePower_sameRoomAsGenerator(t *testing.T) {
	g, grid, start, _ := makePowerGridTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": false, "RoomB": false}
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(start).Generator = gen
	term := entities.NewMaintenanceTerminal("T", "RoomA")
	term.Powered = false
	gameworld.GetGameData(grid.GetCell(0, 0)).MaintenanceTerm = term

	if ApplyGridConductivePower(g) != 1 {
		t.Fatal("powered generator should feed terminals in its room even when doors are off")
	}
	if !term.Powered {
		t.Fatal("same-room terminal should be powered")
	}
}

func TestRestoreTerminalsInRooms_powerGrid(t *testing.T) {
	g, grid, _, _ := makePowerGridTestGrid(t)
	termB := entities.NewMaintenanceTerminal("T", "RoomB")
	termB.Powered = false
	gameworld.GetGameData(grid.GetCell(0, 2)).MaintenanceTerm = termB
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}

	n, msg := RestoreTerminalsInRooms(g, map[string]bool{"RoomB": true})
	if n != 1 {
		t.Fatalf("restored %d, msg %q", n, msg)
	}
	if !termB.Powered {
		t.Fatal("terminal should be powered")
	}
}

func TestCellHasLivePower_repairDeviceAdjacentToLiveCell(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(3, 3)
	g.Grid = grid
	roomName := "Workshop"
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			c := grid.GetCell(row, col)
			c.Room, c.Name, c.Discovered = true, roomName, true
			gameworld.InitGameData(c)
		}
	}
	grid.BuildAllCellConnections()

	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(1, 0)).Generator = gen

	repair := entities.NewRepairObjective("routing-repair-deck3-0", entities.RepairSignalCalibrator, roomName, 1, 1)
	repair.RequiresPower = true
	gameworld.GetGameData(grid.GetCell(1, 1)).RepairDevice = repair

	g.RoomDoorsPowered = map[string]bool{roomName: true}
	PropagateRoomPowerOnlineFromGenerators(g)
	ApplyGridConductivePower(g)

	if !CellHasLivePower(g, grid.GetCell(1, 0)) {
		t.Fatal("generator cell should have live power")
	}
	if !CellHasLivePower(g, grid.GetCell(1, 1)) {
		t.Fatal("repair device cell adjacent to live pocket should have live power")
	}
}

func TestCellHasLivePower_repairDeviceIsolatedPocketNotPowered(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 7)
	g.Grid = grid
	roomName := "SplitRoom"
	for _, col := range []int{0, 1, 2} {
		c := grid.GetCell(0, col)
		c.Room, c.Name, c.Discovered = true, roomName, true
		gameworld.InitGameData(c)
	}
	for _, col := range []int{4, 5, 6} {
		c := grid.GetCell(0, col)
		c.Room, c.Name, c.Discovered = true, roomName, true
		gameworld.InitGameData(c)
	}
	block := grid.GetCell(0, 3)
	block.Room, block.Name, block.Discovered = true, "Corridor", true
	gameworld.InitGameData(block)
	gameworld.GetGameData(block).PowerRelay = entities.NewPowerRelayOpen()
	grid.BuildAllCellConnections()

	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen

	repair := entities.NewRepairObjective("waste", entities.RepairWastePump, roomName, 0, 5)
	repair.RequiresPower = true
	gameworld.GetGameData(grid.GetCell(0, 5)).RepairDevice = repair

	g.RoomDoorsPowered = map[string]bool{roomName: true}
	PropagateRoomPowerOnlineFromGenerators(g)
	ApplyGridConductivePower(g)

	if CellHasLivePower(g, grid.GetCell(0, 5)) {
		t.Fatal("repair device in isolated pocket should not have live power")
	}
}
