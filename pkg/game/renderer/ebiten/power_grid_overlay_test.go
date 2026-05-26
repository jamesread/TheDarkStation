package ebiten

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func testPowerGridSnapshot(t *testing.T, g *state.Game) *powerGridSnapshot {
	t.Helper()
	var snap renderSnapshot
	capturePowerGridSnapshot(g, &snap)
	return &snap.powerGrid
}

func TestPowerGridCellBg_generatorSeedLive(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 2)
	grid.MarkAsRoomWithName(0, 0, "RoomA", "")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "ROOM_CORRIDOR")
	grid.BuildAllCellConnections()
	g.Grid = grid
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
			cell.Discovered = true
		}
	})
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.RoomDoorsPowered = map[string]bool{"RoomA": true}
	g.RoomPowerOnline = map[string]bool{"RoomA": true}
	g.PowerGridOverlayActive = true
	g.PowerGridOverlaySeedRow = 0
	g.PowerGridOverlaySeedCol = 0

	seed := grid.GetCell(0, 0)
	pg := testPowerGridSnapshot(t, g)
	if !snapshotHasCell(pg.liveCells, seed) {
		t.Fatal("powered generator seed should be on live grid")
	}
	if bg := powerGridCellBg(g, pg, seed); bg != colorPowerGridGenerator {
		t.Fatalf("live generator seed bg = %v, want %v", bg, colorPowerGridGenerator)
	}
}

func TestPowerGridCellBg_armedButOffline_isRed(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 2)
	grid.MarkAsRoomWithName(0, 0, "RoomA", "")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "ROOM_CORRIDOR")
	grid.BuildAllCellConnections()
	g.Grid = grid
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
			cell.Discovered = true
		}
	})
	g.RoomDoorsPowered = map[string]bool{"RoomA": true}
	g.PowerGridOverlayActive = true
	g.PowerGridOverlaySeedRow = 0
	g.PowerGridOverlaySeedCol = 0

	seed := grid.GetCell(0, 0)
	pg := testPowerGridSnapshot(t, g)
	if snapshotHasCell(pg.liveCells, seed) {
		t.Fatal("offline room should not be on live grid")
	}
	if !snapshotHasCell(pg.armedCells, seed) {
		t.Fatal("armed room should still appear on armed grid")
	}
	if bg := powerGridCellBg(g, pg, seed); bg != colorPowerGridGeneratorOff {
		t.Fatalf("offline armed seed bg = %v, want %v", bg, colorPowerGridGeneratorOff)
	}
	corridor := grid.GetCell(0, 1)
	if bg := powerGridCellBg(g, pg, corridor); bg != colorPowerGridCellOff {
		t.Fatalf("offline armed corridor bg = %v, want %v", bg, colorPowerGridCellOff)
	}
}

func TestPowerGridLiveCells_excludesOfflineDownstream(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 4)
	grid.MarkAsRoomWithName(0, 0, "RoomA", "")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "ROOM_CORRIDOR")
	grid.MarkAsRoomWithName(0, 2, "Corridor", "ROOM_CORRIDOR")
	grid.MarkAsRoomWithName(0, 3, "RoomB", "")
	grid.BuildAllCellConnections()
	g.Grid = grid
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
			cell.Discovered = true
		}
	})
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	doorCell := grid.GetCell(0, 2)
	gameworld.GetGameData(doorCell).Door = &entities.Door{RoomName: "RoomB", Locked: false}
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": false}
	g.RoomPowerOnline = map[string]bool{"RoomA": true}
	g.ManualEgressReleased = map[string]bool{"RoomB": true}
	g.PowerGridOverlayActive = true
	g.PowerGridOverlaySeedRow = 0
	g.PowerGridOverlaySeedCol = 0

	pg := testPowerGridSnapshot(t, g)
	roomBCell := grid.GetCell(0, 3)
	if snapshotHasCell(pg.liveCells, roomBCell) {
		t.Fatal("offline downstream room should not be on live grid")
	}
	if !snapshotHasCell(pg.armedCells, roomBCell) {
		t.Fatal("manual egress should extend overlay grid into offline downstream room")
	}
}

func TestPowerGridRoomFloorBg_poweredAndUnpowered(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 3)
	grid.MarkAsRoomWithName(0, 0, "PoweredRoom", "")
	grid.MarkAsRoomWithName(0, 1, "PoweredRoom", "")
	grid.MarkAsRoomWithName(0, 2, "DeadRoom", "")
	grid.BuildAllCellConnections()
	g.Grid = grid
	grid.ForEachCell(func(row, col int, c *world.Cell) {
		if c != nil {
			gameworld.InitGameData(c)
			c.Discovered = true
		}
	})
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.PowerGridOverlayActive = true
	g.RoomDoorsPowered = map[string]bool{"PoweredRoom": true, "DeadRoom": false}
	g.RoomPowerOnline = map[string]bool{"PoweredRoom": true}
	pg := testPowerGridSnapshot(t, g)

	on := CellRenderOptions{Icon: IconVisited, Color: colorFloor, HasBackground: true}
	floorCell := grid.GetCell(0, 1)
	dead := grid.GetCell(0, 2)
	if powerGridRoomFloorBg(g, pg, floorCell, &on) == nil {
		t.Fatal("live room floor should be tinted green")
	}
	if powerGridRoomFloorBg(g, pg, dead, &on) != colorHazardBackground {
		t.Fatal("offline room floor should be tinted red")
	}
}

func TestPowerGridWallBg_adjacentRoomPower(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 2)
	grid.MarkAsRoomWithName(0, 0, "RoomA", "")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "ROOM_CORRIDOR")
	grid.BuildAllCellConnections()
	g.Grid = grid
	grid.ForEachCell(func(row, col int, c *world.Cell) {
		if c != nil {
			gameworld.InitGameData(c)
			c.Discovered = true
		}
	})
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.PowerGridOverlayActive = true
	g.RoomDoorsPowered = map[string]bool{"RoomA": true}
	g.RoomPowerOnline = map[string]bool{"RoomA": true}
	pg := testPowerGridSnapshot(t, g)

	wall := grid.GetCell(0, 1)
	wall.Room = false
	wall.Discovered = true
	if powerGridWallBg(g, pg, wall) != colorWallBgPowered {
		t.Fatal("wall adjacent to live room should be green")
	}
	g.RoomDoorsPowered["RoomA"] = false
	g.RoomPowerOnline["RoomA"] = false
	pg = testPowerGridSnapshot(t, g)
	if powerGridWallBg(g, pg, wall) != colorHazardBackground {
		t.Fatal("wall adjacent to offline room should be red")
	}
}

func TestPowerGridOverlayActive_generatorToggle(t *testing.T) {
	g := state.NewGame()
	g.PowerGridOverlayActive = true
	if !powerGridOverlayActive(g) {
		t.Fatal("overlay should be active when generator toggle on")
	}
	g.PowerGridOverlayActive = false
	if powerGridOverlayActive(g) {
		t.Fatal("overlay should be off")
	}
}

func TestCapturePowerGridSnapshot_inactiveWhenOverlayOff(t *testing.T) {
	g := state.NewGame()
	var snap renderSnapshot
	capturePowerGridSnapshot(g, &snap)
	if snap.powerGrid.active {
		t.Fatal("snapshot should be inactive when overlay off")
	}
}
