package gameplay

import (
	"testing"
	"time"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/devtools"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestUpdateLightingExploration_PassiveOverloadSetsPowerOverloadWarned(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(2, 2)
	for r := 0; r < 2; r++ {
		for c := 0; c < 2; c++ {
			grid.MarkAsRoomWithName(r, c, "R", "desc")
			gd := gameworld.InitGameData(grid.GetCell(r, c))
			gd.Door = &entities.Door{RoomName: "R", Locked: false}
		}
	}
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(1, 1)
	grid.BuildAllCellConnections()

	g.Grid = grid
	g.CurrentCell = grid.GetCell(0, 0)
	g.RoomDoorsPowered = map[string]bool{"R": true}
	g.RoomCCTVPowered = map[string]bool{"R": false}
	g.RoomPowerOnline = map[string]bool{"R": true}
	g.PowerOverloadWarned = false

	UpdateLightingExploration(g)

	if !g.PowerOverloadWarned {
		t.Error("PowerOverloadWarned should be true when consumption > supply (no generators, doors on)")
	}
}

func TestUpdateLightingExploration_ResetsPowerOverloadWarnedWhenWithinSupply(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(2, 2)
	for r := 0; r < 2; r++ {
		for c := 0; c < 2; c++ {
			grid.MarkAsRoomWithName(r, c, "R", "desc")
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(1, 1)
	grid.BuildAllCellConnections()

	g.Grid = grid
	g.CurrentCell = grid.GetCell(0, 0)
	g.PowerSupply = 200
	g.RoomDoorsPowered = map[string]bool{"R": false}
	g.RoomCCTVPowered = map[string]bool{"R": false}
	g.PowerConsumption = 0
	g.PowerOverloadWarned = true

	UpdateLightingExploration(g)

	if g.PowerOverloadWarned {
		t.Error("PowerOverloadWarned should be false when consumption <= supply")
	}
}

func makeLightingGrid() (*world.Grid, *state.Game) {
	grid := world.NewGrid(6, 6)
	for r := 0; r < 6; r++ {
		for _, c := range []int{0, 1, 4, 5} {
			grid.MarkAsRoomWithName(r, c, "R", "desc")
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(5, 5)
	grid.BuildAllCellConnections()

	g := state.NewGame()
	g.Grid = grid
	g.CurrentCell = grid.GetCell(0, 0)
	g.RoomDoorsPowered = map[string]bool{"R": false}
	g.RoomCCTVPowered = map[string]bool{"R": false}
	g.RoomLightsPowered = map[string]bool{"R": true}
	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteriesAndStart(1)
	g.AddGenerator(gen)
	return grid, g
}

func TestUpdateLightingExploration_AlwaysLitRoomCells(t *testing.T) {
	grid, g := makeLightingGrid()
	g.Generators = nil
	g.RoomLightsPowered = map[string]bool{"R": false}

	cell := grid.GetCell(1, 1)
	cell.Discovered = true
	cell.Visited = true
	gameworld.GetGameData(cell).LightsOn = false
	gameworld.GetGameData(cell).Lighted = false

	UpdateLightingExploration(g)

	data := gameworld.GetGameData(cell)
	if !data.LightsOn || !data.Lighted {
		t.Error("room cells should always be lit while lighting system is disabled")
	}
}

func TestUpdateLightingExploration_DiscoveredCellsNotDarkenedWithoutPower(t *testing.T) {
	grid, g := makeLightingGrid()
	g.Generators = nil
	g.CurrentCell = grid.GetCell(0, 0)

	farCell := grid.GetCell(5, 5)
	farCell.Discovered = true
	farCell.Visited = false

	UpdateLightingExploration(g)

	if !farCell.Discovered {
		t.Error("discovered cells should not be darkened while lighting is disabled")
	}
	data := gameworld.GetGameData(farCell)
	if !data.LightsOn || !data.Lighted {
		t.Error("discovered room cells should be marked lit")
	}
}

func TestUpdateLightingExploration_LightingDoesNotConsumePower(t *testing.T) {
	_, g := makeLightingGrid()
	g.RoomDoorsPowered = map[string]bool{"R": false}
	g.RoomCCTVPowered = map[string]bool{"R": false}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.Room {
			cell.Discovered = true
			cell.Visited = true
		}
	})

	consumptionBefore := g.CalculatePowerConsumption()
	UpdateLightingExploration(g)
	consumptionAfter := g.PowerConsumption

	if consumptionAfter != consumptionBefore {
		t.Errorf("lighting should not consume power; consumption changed from %d to %d", consumptionBefore, consumptionAfter)
	}
	if consumptionAfter != 0 {
		t.Errorf("consumption should be 0 with no doors/CCTV/puzzles; got %d", consumptionAfter)
	}
}

func TestUpdateLightingExploration_RecalculatesPowerStateBeforeApplyingLighting(t *testing.T) {
	grid, g := makeLightingGrid()
	cell := grid.GetCell(1, 1)
	cell.Discovered = true
	cell.Visited = true

	g.PowerSupply = 0
	g.PowerConsumption = 999

	UpdateLightingExploration(g)

	if g.PowerSupply <= 0 {
		t.Errorf("expected power supply recalculated from generators, got %d", g.PowerSupply)
	}
	if g.PowerConsumption != 0 {
		t.Errorf("expected consumption recalculated from active devices, got %d", g.PowerConsumption)
	}
	data := gameworld.GetGameData(cell)
	if !data.LightsOn || !data.Lighted {
		t.Error("expected room cell to be lit while lighting system is disabled")
	}
}

func TestUpdateLightingExploration_entitiesGeneratorsPerfMapCompletesQuickly(t *testing.T) {
	g := state.NewGame()
	devtools.SwitchToPerfMap(g, "entities_generators")
	if len(g.Generators) == 0 {
		t.Fatal("perf map should register generators for realistic power simulation")
	}

	start := time.Now()
	UpdateLightingExploration(g)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("UpdateLightingExploration took %v, want under 2s on dense generator perf map", elapsed)
	}
}
