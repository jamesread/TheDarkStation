package ebiten

import (
	"image/color"
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
	want := pulsePowerGridEdgeColor(colorPowerGridGenerator, powerGridEdgePulseIntensity(0, 0, 0, 0, 0, 0, 0))
	if bg := powerGridCellBgAt(g, pg, seed, 0); bg != want {
		t.Fatalf("live generator seed bg = %v, want %v", bg, want)
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
	wantSeed := pulsePowerGridEdgeColor(colorPowerGridGeneratorOff, powerGridEdgePulseIntensity(0, 0, 0, 0, 0, 0, 0)*0.55)
	if bg := powerGridCellBgAt(g, pg, seed, 0); bg != wantSeed {
		t.Fatalf("offline armed seed bg = %v, want %v", bg, wantSeed)
	}
	corridor := grid.GetCell(0, 1)
	wantCorr := pulsePowerGridEdgeColor(colorPowerGridCellOff, powerGridEdgePulseIntensity(0, 0, 1, 0, 1, 0, 0)*0.55)
	if bg := powerGridCellBgAt(g, pg, corridor, 0); bg != wantCorr {
		t.Fatalf("offline armed corridor bg = %v, want %v", bg, wantCorr)
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

func TestPowerGridPulseIntensity_peaksAtPhaseZero(t *testing.T) {
	// With no audio epoch, phase uses absolute time; intensity helpers are pure enough to
	// exercise color/width scaling independently.
	peak := pulsePowerGridEdgeColor(color.RGBA{10, 200, 10, 150}, 1)
	dim := pulsePowerGridEdgeColor(color.RGBA{10, 200, 10, 150}, 0)
	_, _, _, peakA := peak.RGBA()
	_, _, _, dimA := dim.RGBA()
	if peakA <= dimA {
		t.Fatalf("peak alpha %d should exceed dim alpha %d", peakA, dimA)
	}
	if powerGridPulseLineWidth(1) <= powerGridPulseLineWidth(0) {
		t.Fatal("peak pulse line width should exceed dim width")
	}
	hi := powerGridPulseIntensity(0)
	if hi < 0.9 {
		t.Fatalf("intensity at phase-aligned 0 = %v, want near peak", hi)
	}
}

func TestPowerGridEdgePulseIntensity_travelsOutward(t *testing.T) {
	// At phase 0, an edge at the seed is brightest; an edge far from the seed is dimmer.
	near := powerGridEdgePulseIntensity(0, 0, 0, 0, 1, 0, 0)
	far := powerGridEdgePulseIntensity(0, 0, 9, 0, 10, 0, 0)
	if near <= far {
		t.Fatalf("near intensity %v should exceed far %v at phase 0", near, far)
	}
}
