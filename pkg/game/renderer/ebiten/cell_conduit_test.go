package ebiten

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestGetCellRenderOptions_unfixedConduitIsYellow(t *testing.T) {
	e := &EbitenRenderer{}
	g := state.NewGame()
	grid := world.NewGrid(2, 2)
	grid.MarkAsRoomWithName(0, 0, "Corridor", "")
	grid.BuildAllCellConnections()
	cell := grid.GetCell(0, 0)
	if cell == nil {
		t.Fatal("cell nil")
	}
	cell.Discovered = true
	gameworld.InitGameData(cell)
	gameworld.GetGameData(cell).LightsOn = true
	repair := entities.NewRepairObjective("c1", entities.RepairConduitSplice, "Corridor", 0, 0)
	gameworld.GetGameData(cell).RepairDevice = repair
	g.Grid = grid

	snap := &renderSnapshot{playerRow: -1, playerCol: -1}
	opts := e.getCellRenderOptions(g, cell, snap, false)
	if opts.Icon != IconRepairConduit {
		t.Fatalf("icon = %q, want conduit splice", opts.Icon)
	}
	if opts.Color != colorRepairConduit {
		t.Fatalf("unfixed conduit color = %v, want yellow %v", opts.Color, colorRepairConduit)
	}
	if opts.BackgroundColor != colorRepairConduitBg {
		t.Fatalf("unfixed conduit bg = %v, want %v", opts.BackgroundColor, colorRepairConduitBg)
	}

	repair.Complete()
	opts = e.getCellRenderOptions(g, cell, snap, false)
	if opts.Color != colorSubtle {
		t.Fatalf("fixed conduit color = %v, want subtle %v", opts.Color, colorSubtle)
	}
}

func TestGetCellRenderOptions_otherRepairsStayCyan(t *testing.T) {
	e := &EbitenRenderer{}
	g := state.NewGame()
	grid := world.NewGrid(2, 2)
	grid.MarkAsRoomWithName(0, 0, "Pump", "")
	grid.BuildAllCellConnections()
	cell := grid.GetCell(0, 0)
	cell.Discovered = true
	gameworld.InitGameData(cell)
	gameworld.GetGameData(cell).LightsOn = true
	gameworld.GetGameData(cell).RepairDevice = entities.NewRepairObjective("p1", entities.RepairWastePump, "Pump", 0, 0)
	g.Grid = grid

	snap := &renderSnapshot{
		playerRow: -1,
		playerCol: -1,
		mapPower: mapPowerSnapshot{
			livePowerCells: map[uint64]bool{cellCoordKey(0, 0): true},
		},
	}
	opts := e.getCellRenderOptions(g, cell, snap, false)
	if opts.Color != colorRepair {
		t.Fatalf("pump repair color = %v, want cyan %v", opts.Color, colorRepair)
	}
}
