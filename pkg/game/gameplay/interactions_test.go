package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func makeTestGame(rows, cols int) *state.Game {
	g := state.NewGame()
	grid := world.NewGrid(rows, cols)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			grid.MarkAsRoomWithName(r, c, "Room", "desc")
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(rows-1, cols-1)
	grid.BuildAllCellConnections()
	g.Grid = grid
	g.CurrentCell = grid.GetCell(0, 0)
	return g
}

func TestCheckAdjacentGenerators_InsertsBatteries(t *testing.T) {
	g := makeTestGame(2, 2)
	gen := entities.NewGenerator("G1", 2)
	gameworld.GetGameData(g.Grid.GetCell(0, 1)).Generator = gen
	g.AddGenerator(gen)
	g.Batteries = 5

	CheckAdjacentGenerators(g)

	if gen.BatteriesInserted != 2 {
		t.Errorf("BatteriesInserted = %d, want 2", gen.BatteriesInserted)
	}
	if !gen.IsPowered() {
		t.Error("generator should be powered after inserting enough batteries")
	}
	if g.Batteries != 3 {
		t.Errorf("remaining batteries = %d, want 3", g.Batteries)
	}
}

func TestCheckAdjacentGenerators_UpdatesPowerSupply(t *testing.T) {
	g := makeTestGame(2, 2)
	g.CurrentDeckID = 0
	gen := entities.NewGenerator("G1", 1)
	gameworld.GetGameData(g.Grid.GetCell(0, 1)).Generator = gen
	g.AddGenerator(gen)
	g.Batteries = 1

	if g.PowerSupply != 0 {
		t.Fatalf("initial PowerSupply = %d, want 0", g.PowerSupply)
	}

	CheckAdjacentGenerators(g)

	if g.PowerSupply != 100 {
		t.Errorf("PowerSupply after powering generator = %d, want 100", g.PowerSupply)
	}
}

func TestCheckAdjacentGenerators_NoBatteriesNoInsert(t *testing.T) {
	g := makeTestGame(2, 2)
	gen := entities.NewGenerator("G1", 2)
	gameworld.GetGameData(g.Grid.GetCell(0, 1)).Generator = gen
	g.AddGenerator(gen)
	g.Batteries = 0

	CheckAdjacentGenerators(g)

	if gen.BatteriesInserted != 0 {
		t.Errorf("BatteriesInserted = %d, want 0 (no batteries)", gen.BatteriesInserted)
	}
}

func TestCheckAdjacentGenerators_PartialInsert(t *testing.T) {
	g := makeTestGame(2, 2)
	gen := entities.NewGenerator("G1", 5)
	gameworld.GetGameData(g.Grid.GetCell(0, 1)).Generator = gen
	g.AddGenerator(gen)
	g.Batteries = 2

	CheckAdjacentGenerators(g)

	if gen.BatteriesInserted != 2 {
		t.Errorf("BatteriesInserted = %d, want 2 (partial)", gen.BatteriesInserted)
	}
	if gen.IsPowered() {
		t.Error("generator should NOT be powered after partial insert")
	}
	if g.Batteries != 0 {
		t.Errorf("remaining batteries = %d, want 0", g.Batteries)
	}
}

func TestCheckAdjacentGenerators_AlreadyPowered(t *testing.T) {
	g := makeTestGame(2, 2)
	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteries(1)
	gameworld.GetGameData(g.Grid.GetCell(0, 1)).Generator = gen
	g.AddGenerator(gen)
	g.Batteries = 3

	CheckAdjacentGenerators(g)

	if g.Batteries != 3 {
		t.Errorf("batteries should not change for already-powered gen: got %d, want 3", g.Batteries)
	}
}

func TestCheckAdjacentGeneratorAtCell_Powered(t *testing.T) {
	g := makeTestGame(2, 2)
	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteries(1)
	cell := g.Grid.GetCell(0, 1)
	gameworld.GetGameData(cell).Generator = gen
	g.AddGenerator(gen)
	g.PowerSupply = 100

	got := CheckAdjacentGeneratorAtCell(g, cell)
	if !got {
		t.Error("CheckAdjacentGeneratorAtCell returned false for cell with generator")
	}
}

func TestCheckAdjacentGeneratorAtCell_Unpowered(t *testing.T) {
	g := makeTestGame(2, 2)
	gen := entities.NewGenerator("G1", 3)
	cell := g.Grid.GetCell(0, 1)
	gameworld.GetGameData(cell).Generator = gen
	g.AddGenerator(gen)

	got := CheckAdjacentGeneratorAtCell(g, cell)
	if !got {
		t.Error("CheckAdjacentGeneratorAtCell returned false for cell with unpowered generator")
	}
}

func TestCheckAdjacentGeneratorAtCell_NilCell(t *testing.T) {
	g := makeTestGame(2, 2)
	got := CheckAdjacentGeneratorAtCell(g, nil)
	if got {
		t.Error("CheckAdjacentGeneratorAtCell should return false for nil cell")
	}
}

func TestCheckAdjacentGeneratorAtCell_NoGenerator(t *testing.T) {
	g := makeTestGame(2, 2)
	cell := g.Grid.GetCell(0, 1)
	got := CheckAdjacentGeneratorAtCell(g, cell)
	if got {
		t.Error("CheckAdjacentGeneratorAtCell should return false for cell without generator")
	}
}

func TestPickUpItemsOnFloor_Battery(t *testing.T) {
	g := makeTestGame(2, 2)
	battery := world.NewItem("Battery")
	g.CurrentCell.ItemsOnFloor.Put(battery)

	if g.Batteries != 0 {
		t.Fatalf("initial batteries = %d, want 0", g.Batteries)
	}

	PickUpItemsOnFloor(g)

	if g.Batteries != 1 {
		t.Errorf("after pickup, batteries = %d, want 1", g.Batteries)
	}
	if g.CurrentCell.ItemsOnFloor.Size() != 0 {
		t.Error("battery should be removed from floor after pickup")
	}
}

func TestPickUpItemsOnFloor_MultipleBatteries(t *testing.T) {
	g := makeTestGame(2, 2)
	g.CurrentCell.ItemsOnFloor.Put(world.NewItem("Battery"))
	g.CurrentCell.ItemsOnFloor.Put(world.NewItem("Battery"))

	PickUpItemsOnFloor(g)

	if g.Batteries != 2 {
		t.Errorf("after picking up 2 batteries, count = %d, want 2", g.Batteries)
	}
}

func TestPickUpItemsOnFloor_NonBatteryItem(t *testing.T) {
	g := makeTestGame(2, 2)
	keycard := world.NewItem("Keycard-A")
	g.CurrentCell.ItemsOnFloor.Put(keycard)

	PickUpItemsOnFloor(g)

	if g.Batteries != 0 {
		t.Errorf("non-battery item should not increase battery count: got %d", g.Batteries)
	}
	if !g.OwnedItems.Has(keycard) {
		t.Error("non-battery item should be added to OwnedItems")
	}
}

// Integration: full lifecycle of picking up batteries and powering a generator.
func TestIntegration_BatteryPickupAndGeneratorPower(t *testing.T) {
	g := makeTestGame(3, 3)
	g.CurrentDeckID = 0

	// Place an unpowered generator adjacent to player (0,0) -> at (0,1)
	gen := entities.NewGenerator("Main Generator", 2)
	gameworld.GetGameData(g.Grid.GetCell(0, 1)).Generator = gen
	g.AddGenerator(gen)

	// Place batteries on player's cell and another cell
	g.CurrentCell.ItemsOnFloor.Put(world.NewItem("Battery"))
	g.Grid.GetCell(1, 0).ItemsOnFloor.Put(world.NewItem("Battery"))

	// Step 1: Pick up first battery
	PickUpItemsOnFloor(g)
	if g.Batteries != 1 {
		t.Fatalf("step 1: batteries = %d, want 1", g.Batteries)
	}

	// Step 2: Try inserting — only 1 of 2 needed, partial insert
	CheckAdjacentGenerators(g)
	if gen.BatteriesInserted != 1 {
		t.Fatalf("step 2: inserted = %d, want 1", gen.BatteriesInserted)
	}
	if gen.IsPowered() {
		t.Fatal("step 2: generator should not be powered yet")
	}
	if g.Batteries != 0 {
		t.Fatalf("step 2: remaining batteries = %d, want 0", g.Batteries)
	}

	// Step 3: Move to (1,0), pick up second battery
	g.CurrentCell = g.Grid.GetCell(1, 0)
	PickUpItemsOnFloor(g)
	if g.Batteries != 1 {
		t.Fatalf("step 3: batteries = %d, want 1", g.Batteries)
	}

	// Step 4: Move back to (0,0), insert final battery
	g.CurrentCell = g.Grid.GetCell(0, 0)
	CheckAdjacentGenerators(g)
	if !gen.IsPowered() {
		t.Fatal("step 4: generator should be powered")
	}
	if g.PowerSupply != 100 {
		t.Errorf("step 4: PowerSupply = %d, want 100", g.PowerSupply)
	}
	if g.Batteries != 0 {
		t.Errorf("step 4: remaining batteries = %d, want 0", g.Batteries)
	}
}

// Edge case: multiple generators, limited batteries
func TestEdgeCase_MultipleGeneratorsLimitedBatteries(t *testing.T) {
	g := makeTestGame(3, 3)
	g.CurrentDeckID = 0

	gen1 := entities.NewGenerator("G1", 1)
	gen2 := entities.NewGenerator("G2", 1)
	// Place gen1 at (0,1) and gen2 at (1,0) — both adjacent to player at (0,0)
	gameworld.GetGameData(g.Grid.GetCell(0, 1)).Generator = gen1
	gameworld.GetGameData(g.Grid.GetCell(1, 0)).Generator = gen2
	g.AddGenerator(gen1)
	g.AddGenerator(gen2)
	g.Batteries = 1

	CheckAdjacentGenerators(g)

	// Only one generator should get the battery (first found in NESW order)
	totalInserted := gen1.BatteriesInserted + gen2.BatteriesInserted
	if totalInserted != 1 {
		t.Errorf("total inserted = %d, want 1 (only 1 battery available)", totalInserted)
	}
	if g.Batteries != 0 {
		t.Errorf("remaining batteries = %d, want 0", g.Batteries)
	}
}

// Edge case: generator with InsertBatteries(0) is a no-op
func TestEdgeCase_InsertZeroBatteries(t *testing.T) {
	gen := entities.NewGenerator("G1", 3)
	inserted := gen.InsertBatteries(0)
	if inserted != 0 {
		t.Errorf("InsertBatteries(0) = %d, want 0", inserted)
	}
	if gen.BatteriesInserted != 0 {
		t.Errorf("BatteriesInserted = %d, want 0", gen.BatteriesInserted)
	}
}

// Edge case: InsertBatteries caps at needed amount
func TestEdgeCase_InsertBatteriesCapped(t *testing.T) {
	gen := entities.NewGenerator("G1", 2)
	gen.InsertBatteries(1)
	inserted := gen.InsertBatteries(10)
	if inserted != 1 {
		t.Errorf("InsertBatteries(10) with 1 needed = %d, want 1", inserted)
	}
	if gen.BatteriesInserted != 2 {
		t.Errorf("BatteriesInserted = %d, want 2", gen.BatteriesInserted)
	}
}

func TestCheckAdjacentTerminalsAtCell_CCTVUnpowered(t *testing.T) {
	g := makeTestGame(2, 2)
	termCell := g.Grid.GetCell(0, 1)
	gameworld.GetGameData(termCell).Terminal = entities.NewCCTVTerminal("CCTV-1")
	g.RoomCCTVPowered = map[string]bool{"Room": false}

	result := CheckAdjacentTerminalsAtCell(g, termCell)
	if !result {
		t.Error("should return true (interaction consumed) even when unpowered")
	}
	// Terminal should NOT be activated
	term := gameworld.GetGameData(termCell).Terminal
	if term.Used {
		t.Error("terminal should not be activated when CCTV unpowered")
	}
}

func TestCheckAdjacentTerminalsAtCell_CCTVPowered(t *testing.T) {
	g := makeTestGame(2, 2)
	termCell := g.Grid.GetCell(0, 1)
	gameworld.GetGameData(termCell).Terminal = entities.NewCCTVTerminal("CCTV-1")
	// Target a room that exists in the grid
	gameworld.GetGameData(termCell).Terminal.TargetRoom = "Room"
	g.RoomCCTVPowered = map[string]bool{"Room": true}

	result := CheckAdjacentTerminalsAtCell(g, termCell)
	if !result {
		t.Error("should return true when CCTV powered")
	}
	term := gameworld.GetGameData(termCell).Terminal
	if !term.Used {
		t.Error("terminal should be activated when CCTV powered")
	}
}

func TestCheckAdjacentHazardControlsAtCell_Unpowered(t *testing.T) {
	g := makeTestGame(2, 2)
	ctrlCell := g.Grid.GetCell(0, 1)
	hazard := &entities.Hazard{Type: entities.HazardElectrical}
	gameworld.GetGameData(ctrlCell).HazardControl = entities.NewHazardControl(entities.HazardElectrical, hazard)
	g.RoomCCTVPowered = map[string]bool{"Room": false}

	result := CheckAdjacentHazardControlsAtCell(g, ctrlCell)
	if !result {
		t.Error("should return true (interaction consumed) even when unpowered")
	}
	ctrl := gameworld.GetGameData(ctrlCell).HazardControl
	if ctrl.Activated {
		t.Error("hazard control should not be activated when room power off")
	}
}

func TestCheckAdjacentHazardControlsAtCell_Powered(t *testing.T) {
	g := makeTestGame(2, 2)
	ctrlCell := g.Grid.GetCell(0, 1)
	hazard := &entities.Hazard{Type: entities.HazardElectrical}
	gameworld.GetGameData(ctrlCell).HazardControl = entities.NewHazardControl(entities.HazardElectrical, hazard)
	g.RoomCCTVPowered = map[string]bool{"Room": true}

	result := CheckAdjacentHazardControlsAtCell(g, ctrlCell)
	if !result {
		t.Error("should return true when room power on")
	}
	ctrl := gameworld.GetGameData(ctrlCell).HazardControl
	if !ctrl.Activated {
		t.Error("hazard control should be activated when room power on")
	}
}
