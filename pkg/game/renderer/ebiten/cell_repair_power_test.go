package ebiten

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestGetCellRenderOptions_unpoweredValveIsOffline(t *testing.T) {
	e := &EbitenRenderer{}
	g := state.NewGame()
	grid := world.NewGrid(1, 2)
	grid.MarkAsRoomWithName(0, 0, "Pump", "")
	grid.BuildAllCellConnections()
	cell := grid.GetCell(0, 0)
	cell.Discovered = true
	gameworld.InitGameData(cell)
	gameworld.GetGameData(cell).LightsOn = true
	repair := entities.NewRepairObjective("v1", entities.RepairPressureValve, "Pump", 0, 0)
	gameworld.GetGameData(cell).RepairDevice = repair
	g.Grid = grid

	snap := &renderSnapshot{playerRow: -1, playerCol: -1}
	opts := e.getCellRenderOptions(g, cell, snap, false)
	if opts.Icon != IconRepairValve {
		t.Fatalf("icon = %q, want valve", opts.Icon)
	}
	if opts.Color != colorGeneratorOff {
		t.Fatalf("unpowered valve color = %v, want offline %v", opts.Color, colorGeneratorOff)
	}
	if opts.BackgroundColor != colorHazardBackground {
		t.Fatalf("unpowered valve bg = %v, want %v", opts.BackgroundColor, colorHazardBackground)
	}

	snap.mapPower.livePowerCells = map[uint64]bool{cellCoordKey(0, 0): true}
	opts = e.getCellRenderOptions(g, cell, snap, false)
	if opts.Color != colorRepair {
		t.Fatalf("powered valve color = %v, want cyan %v", opts.Color, colorRepair)
	}
}

func TestGetCellRenderOptions_unpoweredSignalAndCouplerOffline(t *testing.T) {
	e := &EbitenRenderer{}
	g := state.NewGame()
	grid := world.NewGrid(1, 3)
	grid.MarkAsRoomWithName(0, 0, "A", "")
	grid.MarkAsRoomWithName(0, 1, "B", "")
	grid.MarkAsRoomWithName(0, 2, "C", "")
	grid.BuildAllCellConnections()
	g.Grid = grid

	cases := []struct {
		col  int
		typ  entities.RepairType
		icon string
	}{
		{0, entities.RepairSignalCalibrator, IconRepairSignal},
		{1, entities.RepairPowerCoupler, IconRepairCoupler},
	}
	for _, tc := range cases {
		cell := grid.GetCell(0, tc.col)
		cell.Discovered = true
		gameworld.InitGameData(cell)
		gameworld.GetGameData(cell).LightsOn = true
		gameworld.GetGameData(cell).RepairDevice = entities.NewRepairObjective("r", tc.typ, cell.Name, 0, tc.col)
		snap := &renderSnapshot{playerRow: -1, playerCol: -1}
		opts := e.getCellRenderOptions(g, cell, snap, false)
		if opts.Icon != tc.icon || opts.Color != colorGeneratorOff {
			t.Fatalf("%s at col %d: icon=%q color=%v, want offline", tc.typ, tc.col, opts.Icon, opts.Color)
		}
	}
}
