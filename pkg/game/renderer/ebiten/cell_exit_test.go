package ebiten

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestGetCellRenderOptions_exitLiftStates(t *testing.T) {
	e := &EbitenRenderer{}
	g := state.NewGame()
	grid := world.NewGrid(2, 2)
	grid.MarkAsRoomWithName(0, 0, "Start", "")
	grid.MarkAsRoomWithName(0, 1, "Lift", "")
	grid.BuildAllCellConnections()
	exit := grid.GetCell(0, 1)
	if exit == nil {
		t.Fatal("exit cell nil")
	}
	grid.SetExitCell(exit)
	exit.Discovered = true
	gameworld.InitGameData(grid.GetCell(0, 0))
	gameworld.InitGameData(exit)
	g.Grid = grid
	g.HasMap = true
	g.RoomDoorsPowered["Start"] = true
	g.RoomDoorsPowered["Lift"] = true

	snap := &renderSnapshot{playerRow: -1, playerCol: -1}

	opts := e.getCellRenderOptions(g, exit, snap, false)
	if opts.Icon != IconExitLocked || opts.Color != colorExitLocked {
		t.Fatalf("unpowered: icon=%q color=%v, want locked red", opts.Icon, opts.Color)
	}

	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.AddGenerator(gen)
	setup.PropagateRoomPowerOnlineFromGenerators(g)

	gameworld.GetGameData(grid.GetCell(0, 0)).Hazard = entities.NewHazard(entities.HazardVacuum)

	opts = e.getCellRenderOptions(g, exit, snap, false)
	if opts.Icon != IconExitLocked || opts.Color != colorExitPending {
		t.Fatalf("grid powered with hazard: icon=%q color=%v, want locked yellow", opts.Icon, opts.Color)
	}

	gameworld.GetGameData(grid.GetCell(0, 0)).Hazard.Fix()
	opts = e.getCellRenderOptions(g, exit, snap, false)
	if opts.Icon != IconExitUnlocked {
		t.Fatalf("ready: icon=%q, want unlocked triangle", opts.Icon)
	}
}

func TestGetTileCustomBg_exitLiftPulsingBgOnlyWhenReady(t *testing.T) {
	e := &EbitenRenderer{}
	g := state.NewGame()
	grid := world.NewGrid(2, 2)
	grid.MarkAsRoomWithName(0, 0, "Start", "")
	grid.MarkAsRoomWithName(0, 1, "Lift", "")
	grid.BuildAllCellConnections()
	exit := grid.GetCell(0, 1)
	if exit == nil {
		t.Fatal("exit cell nil")
	}
	grid.SetExitCell(exit)
	exit.Discovered = true
	gameworld.InitGameData(grid.GetCell(0, 0))
	gameworld.InitGameData(exit)
	g.Grid = grid
	g.HasMap = true
	g.RoomDoorsPowered["Start"] = true
	g.RoomDoorsPowered["Lift"] = true

	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.AddGenerator(gen)
	setup.PropagateRoomPowerOnlineFromGenerators(g)

	snap := &renderSnapshot{playerRow: -1, playerCol: -1}
	opts := e.getCellRenderOptions(g, exit, snap, false)

	gameworld.GetGameData(grid.GetCell(0, 0)).Hazard = entities.NewHazard(entities.HazardVacuum)
	if bg := e.getTileCustomBg(g, exit, snap, &opts, nil); bg != nil && bg == e.getPulsingExitBackgroundColor() {
		t.Fatal("incomplete lift should not use pulsing exit background")
	}

	gameworld.GetGameData(grid.GetCell(0, 0)).Hazard.Fix()
	if bg := e.getTileCustomBg(g, exit, snap, &opts, nil); bg == nil {
		t.Fatal("ready lift should have pulsing exit background")
	}
}
