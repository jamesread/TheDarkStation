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

func TestUpdateLightingExploration_HeadlampLightsNearbyCells(t *testing.T) {
	grid, g := makeLightingGrid()
	g.Generators = nil
	g.RoomLightsPowered = map[string]bool{"R": false}

	cell := grid.GetCell(1, 1) // within HeadlampRadius of the player at (0,0)
	gameworld.GetGameData(cell).LightsOn = false
	gameworld.GetGameData(cell).Lighted = false

	UpdateLightingExploration(g)

	data := gameworld.GetGameData(cell)
	if !data.LightsOn || !data.Lighted {
		t.Error("cells within headlamp radius should be illuminated even with no grid power")
	}
	if !cell.Discovered {
		t.Error("headlamp-lit cells should be discovered")
	}
}

func TestApplyHeadlamp_ConeFollowsFacing(t *testing.T) {
	grid, g := makeLightingGrid()
	g.Generators = nil
	g.RoomLightsPowered = map[string]bool{"R": false}
	g.CurrentCell = grid.GetCell(2, 0)

	ahead := grid.GetCell(0, 0)  // 2 north of the player
	behind := grid.GetCell(4, 0) // 2 south of the player

	g.PlayerFacing = state.FaceNorth
	UpdateLightingExploration(g)
	if !gameworld.GetGameData(ahead).LightsOn {
		t.Error("cell 2 ahead of facing should be inside the headlamp cone")
	}
	if gameworld.GetGameData(behind).LightsOn {
		t.Error("cell 2 behind facing should be outside the headlamp cone")
	}

	// Turn around: the beam swings with the player.
	g.PlayerFacing = state.FaceSouth
	UpdateLightingExploration(g)
	if gameworld.GetGameData(ahead).LightsOn {
		t.Error("after turning south, the cell 2 north should fall out of the beam")
	}
	if !gameworld.GetGameData(behind).LightsOn {
		t.Error("after turning south, the cell 2 south should be lit")
	}
	if !gameworld.GetGameData(ahead).Lighted {
		t.Error("a cell seen lit must keep remembered knowledge after the beam moves off it")
	}
}

func TestApplyHeadlamp_BackRadiusKeepsAdjacentGlow(t *testing.T) {
	grid, g := makeLightingGrid()
	g.Generators = nil
	g.RoomLightsPowered = map[string]bool{"R": false}
	g.CurrentCell = grid.GetCell(2, 0)
	g.PlayerFacing = state.FaceNorth

	UpdateLightingExploration(g)

	adjacentBehind := grid.GetCell(3, 0) // 1 south of the player
	if !gameworld.GetGameData(adjacentBehind).LightsOn {
		t.Error("cells at Chebyshev distance 1 should stay lit regardless of facing (residual glow)")
	}
}

func TestRefreshHeadlampCone_RestoresGridLitCells(t *testing.T) {
	grid, g := makeLightingGrid()
	g.Generators = nil
	g.RoomLightsPowered = map[string]bool{"R": false}
	g.CurrentCell = grid.GetCell(2, 0)
	g.PlayerFacing = state.FaceNorth
	UpdateLightingExploration(g)

	// Simulate a grid-lit cell inside the cone window, then swing the cone away.
	gridLitCell := grid.GetCell(0, 0)
	gameworld.GetGameData(gridLitCell).GridLit = true
	g.PlayerFacing = state.FaceSouth
	RefreshHeadlampCone(g)

	if !gameworld.GetGameData(gridLitCell).LightsOn {
		t.Error("grid-lit cell must stay lit when the headlamp cone swings off it")
	}
	if !gameworld.GetGameData(grid.GetCell(4, 0)).LightsOn {
		t.Error("cone refresh should light the new facing direction")
	}
}

func TestHeadlampConeCovers(t *testing.T) {
	faceRow, faceCol := state.FaceNorth.Delta()
	cases := []struct {
		dr, dc int
		want   bool
	}{
		{0, 0, true},   // player cell
		{-2, 0, true},  // straight ahead
		{-2, 2, true},  // forward diagonal
		{0, 2, true},   // perpendicular at full radius (dot == 0 counts as forward)
		{1, 0, true},   // 1 behind: residual glow
		{2, 0, false},  // 2 behind
		{2, -2, false}, // rear diagonal
		{-3, 0, false}, // beyond radius
	}
	for _, c := range cases {
		if got := headlampConeCovers(c.dr, c.dc, faceRow, faceCol); got != c.want {
			t.Errorf("headlampConeCovers(%d,%d facing north) = %v, want %v", c.dr, c.dc, got, c.want)
		}
	}
}

func TestUpdateLightingExploration_FarDiscoveredCellGoesDarkWithoutPower(t *testing.T) {
	grid, g := makeLightingGrid()
	g.Generators = nil
	g.CurrentCell = grid.GetCell(0, 0)

	farCell := grid.GetCell(5, 5)
	farCell.Discovered = true
	farCell.Visited = false

	UpdateLightingExploration(g)

	if !farCell.Discovered {
		t.Error("going dark must not lose discovery (layout knowledge is sticky)")
	}
	data := gameworld.GetGameData(farCell)
	if data.LightsOn {
		t.Error("far cell with no live power and no headlamp should be dark")
	}
	if data.Lighted {
		t.Error("cell never seen lit should not gain remembered knowledge")
	}
}

func TestUpdateLightingExploration_LiveConduitLightsRoom(t *testing.T) {
	grid, g := makeLightingGrid()

	// Powered generator on the grid inside room "R" (left block).
	genCell := grid.GetCell(5, 0)
	gen := entities.NewGenerator("G-grid", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(genCell).Generator = gen

	farSameRoom := grid.GetCell(3, 1) // outside headlamp range, same conductive pocket
	farSameRoom.Discovered = true
	disconnected := grid.GetCell(5, 5) // right block: no conduit from the generator
	disconnected.Discovered = true

	UpdateLightingExploration(g)

	if !gameworld.GetGameData(farSameRoom).LightsOn {
		t.Error("cell on a live conduit with lights enabled should be lit")
	}
	if !gameworld.GetGameData(farSameRoom).Lighted {
		t.Error("discovered cell seen lit should gain remembered knowledge")
	}
	if gameworld.GetGameData(disconnected).LightsOn {
		t.Error("cell disconnected from every powered generator should stay dark")
	}
}

func TestUpdateLightingExploration_RoomLightsToggleGatesConduitLighting(t *testing.T) {
	grid, g := makeLightingGrid()
	g.RoomLightsPowered = map[string]bool{"R": false}

	genCell := grid.GetCell(5, 0)
	gen := entities.NewGenerator("G-grid", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(genCell).Generator = gen

	farSameRoom := grid.GetCell(3, 1)
	farSameRoom.Discovered = true

	UpdateLightingExploration(g)

	if gameworld.GetGameData(farSameRoom).LightsOn {
		t.Error("room with lights circuit off should stay dark even on a live conduit")
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
		t.Error("expected cell within headlamp radius to be lit")
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
