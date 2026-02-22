package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestUpdateLightingExploration_PassiveOverloadSetsPowerOverloadWarned(t *testing.T) {
	// No generators → supply 0. 4 doors powered → consumption 40. Overload.
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

// makeLightingGrid creates a 6x6 grid of rooms for lighting tests.
// Player at (0,0); cells within 5×5 (row,col <=2) are "near", others "far".
// Includes one powered generator so GetAvailablePower() > 0 by default.
func makeLightingGrid() (*world.Grid, *state.Game) {
	grid := world.NewGrid(6, 6)
	for r := 0; r < 6; r++ {
		for c := 0; c < 6; c++ {
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
	gen.InsertBatteries(1)
	g.AddGenerator(gen)
	// UpdatePowerSupply will set PowerSupply from generators
	return grid, g
}

func TestUpdateLightingExploration_WhenPowerAndVisited_SetsLightsOnAndLighted(t *testing.T) {
	grid, g := makeLightingGrid()
	// Mark cell (1,1) as visited (within 5×5 of player at 0,0)
	cell := grid.GetCell(1, 1)
	cell.Discovered = true
	cell.Visited = true

	UpdateLightingExploration(g)

	data := gameworld.GetGameData(cell)
	if !data.LightsOn {
		t.Error("LightsOn should be true when availablePower > 0 and cell visited")
	}
	if !data.Lighted {
		t.Error("Lighted should be true when lights turn on")
	}
	if !cell.Discovered || !cell.Visited {
		t.Error("discovered/visited should be preserved when lights on")
	}
}

func TestUpdateLightingExploration_WhenNoPower_FarCellsDarkenUnlessLighted(t *testing.T) {
	grid, g := makeLightingGrid()
	g.Generators = nil // no generators → UpdatePowerSupply sets 0
	g.CurrentCell = grid.GetCell(0, 0)

	// Far cell (5,5) - beyond 5×5 radius of player at (0,0)
	farCell := grid.GetCell(5, 5)
	farCell.Discovered = true
	farCell.Visited = true
	// Not permanently lighted - should darken
	gameworld.GetGameData(farCell).Lighted = false

	UpdateLightingExploration(g)

	if farCell.Discovered || farCell.Visited {
		t.Error("far cell beyond radius should have discovered/visited cleared when power <= 0 and not Lighted")
	}
}

func TestUpdateLightingExploration_WhenNoPower_NearCellsStayVisible(t *testing.T) {
	grid, g := makeLightingGrid()
	g.Generators = nil // no generators → power 0
	g.CurrentCell = grid.GetCell(0, 0)

	// Near cell (2,2) - within 5×5 radius
	nearCell := grid.GetCell(2, 2)
	nearCell.Discovered = true
	nearCell.Visited = true
	gameworld.GetGameData(nearCell).Lighted = false

	UpdateLightingExploration(g)

	if !nearCell.Discovered {
		t.Error("cell within radius of player should stay discovered when power <= 0")
	}
}

func TestUpdateLightingExploration_WhenNoPower_LightedFarCellsStayDiscovered(t *testing.T) {
	grid, g := makeLightingGrid()
	g.Generators = nil // no generators → power 0
	g.CurrentCell = grid.GetCell(0, 0)

	// Far cell (5,5) - but was previously lighted
	farCell := grid.GetCell(5, 5)
	farCell.Discovered = true
	farCell.Visited = true
	gameworld.GetGameData(farCell).Lighted = true

	UpdateLightingExploration(g)

	if !farCell.Discovered || !farCell.Visited {
		t.Error("Lighted far cell should keep discovered/visited even when power <= 0")
	}
}

func TestUpdateLightingExploration_WhenRoomLightsOff_LightsStayOffDespitePower(t *testing.T) {
	// RoomLightsPowered gates lights-on: when false, lights stay off even with power and visited
	grid, g := makeLightingGrid()
	g.RoomLightsPowered = map[string]bool{"R": false} // lights toggled off for room
	cell := grid.GetCell(1, 1)
	cell.Discovered = true
	cell.Visited = true

	UpdateLightingExploration(g)

	data := gameworld.GetGameData(cell)
	if data.LightsOn {
		t.Error("LightsOn should be false when RoomLightsPowered[room] is false")
	}
}

func TestUpdateLightingExploration_LightingDoesNotConsumePower(t *testing.T) {
	// Verify CalculatePowerConsumption does not include lighting/cells.
	// Power consumption comes only from doors, CCTV, solved puzzles.
	_, g := makeLightingGrid()
	g.RoomDoorsPowered = map[string]bool{"R": false}
	g.RoomCCTVPowered = map[string]bool{"R": false}
	// Mark many cells as visited with lights on
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.Room {
			cell.Discovered = true
			cell.Visited = true
			gameworld.GetGameData(cell).LightsOn = true
			gameworld.GetGameData(cell).Lighted = true
		}
	})

	consumptionBefore := g.CalculatePowerConsumption()
	UpdateLightingExploration(g)
	consumptionAfter := g.PowerConsumption

	// Lighting should not add to consumption; only doors/CCTV/puzzles do
	if consumptionAfter != consumptionBefore {
		t.Errorf("lighting should not consume power; consumption changed from %d to %d", consumptionBefore, consumptionAfter)
	}
	// With no doors/CCTV/puzzles powered, consumption should be 0
	if consumptionAfter != 0 {
		t.Errorf("consumption should be 0 with no doors/CCTV/puzzles; got %d", consumptionAfter)
	}
}

func TestUpdateLightingExploration_RecalculatesPowerStateBeforeApplyingLighting(t *testing.T) {
	grid, g := makeLightingGrid()
	cell := grid.GetCell(1, 1)
	cell.Discovered = true
	cell.Visited = true

	// Seed stale values to verify UpdateLightingExploration recomputes from live state.
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
	if !data.LightsOn {
		t.Error("expected visited cell to be lit after power recalculation")
	}
}
