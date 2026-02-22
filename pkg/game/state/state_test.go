package state

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	gameworld "darkstation/pkg/game/world"
)

func TestAddBatteries(t *testing.T) {
	g := NewGame()
	if g.Batteries != 0 {
		t.Fatalf("initial Batteries = %d, want 0", g.Batteries)
	}
	g.AddBatteries(3)
	if g.Batteries != 3 {
		t.Errorf("after AddBatteries(3), Batteries = %d, want 3", g.Batteries)
	}
	g.AddBatteries(2)
	if g.Batteries != 5 {
		t.Errorf("after AddBatteries(2), Batteries = %d, want 5", g.Batteries)
	}
}

func TestUseBatteries(t *testing.T) {
	tests := []struct {
		name      string
		initial   int
		use       int
		wantUsed  int
		wantAfter int
	}{
		{"use some", 5, 3, 3, 2},
		{"use all", 5, 5, 5, 0},
		{"use more than have", 3, 10, 3, 0},
		{"use zero", 5, 0, 0, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGame()
			g.Batteries = tt.initial
			got := g.UseBatteries(tt.use)
			if got != tt.wantUsed {
				t.Errorf("UseBatteries(%d) = %d, want %d", tt.use, got, tt.wantUsed)
			}
			if g.Batteries != tt.wantAfter {
				t.Errorf("Batteries after = %d, want %d", g.Batteries, tt.wantAfter)
			}
		})
	}
}

func TestAddGenerator(t *testing.T) {
	g := NewGame()
	if len(g.Generators) != 0 {
		t.Fatalf("initial Generators len = %d, want 0", len(g.Generators))
	}
	gen := entities.NewGenerator("G1", 2)
	g.AddGenerator(gen)
	if len(g.Generators) != 1 {
		t.Fatalf("after AddGenerator, len = %d, want 1", len(g.Generators))
	}
	if g.Generators[0] != gen {
		t.Error("stored generator pointer mismatch")
	}
}

func TestAllGeneratorsPowered(t *testing.T) {
	g := NewGame()
	if !g.AllGeneratorsPowered() {
		t.Error("AllGeneratorsPowered() = false with no generators, want true")
	}

	gen1 := entities.NewGenerator("G1", 1)
	gen2 := entities.NewGenerator("G2", 1)
	g.AddGenerator(gen1)
	g.AddGenerator(gen2)

	if g.AllGeneratorsPowered() {
		t.Error("AllGeneratorsPowered() = true with 2 unpowered, want false")
	}

	gen1.InsertBatteries(1)
	if g.AllGeneratorsPowered() {
		t.Error("AllGeneratorsPowered() = true with 1 unpowered, want false")
	}

	gen2.InsertBatteries(1)
	if !g.AllGeneratorsPowered() {
		t.Error("AllGeneratorsPowered() = false with all powered, want true")
	}
}

func TestUnpoweredGeneratorCount(t *testing.T) {
	g := NewGame()
	if g.UnpoweredGeneratorCount() != 0 {
		t.Error("UnpoweredGeneratorCount() != 0 with no generators")
	}

	gen1 := entities.NewGenerator("G1", 1)
	gen2 := entities.NewGenerator("G2", 2)
	g.AddGenerator(gen1)
	g.AddGenerator(gen2)

	if got := g.UnpoweredGeneratorCount(); got != 2 {
		t.Errorf("UnpoweredGeneratorCount() = %d, want 2", got)
	}

	gen1.InsertBatteries(1)
	if got := g.UnpoweredGeneratorCount(); got != 1 {
		t.Errorf("UnpoweredGeneratorCount() after powering G1 = %d, want 1", got)
	}

	gen2.InsertBatteries(2)
	if got := g.UnpoweredGeneratorCount(); got != 0 {
		t.Errorf("UnpoweredGeneratorCount() after powering all = %d, want 0", got)
	}
}

func TestUpdatePowerSupply_Deck0(t *testing.T) {
	g := NewGame()
	g.CurrentDeckID = 0

	g.UpdatePowerSupply()
	if g.PowerSupply != 0 {
		t.Errorf("PowerSupply with no generators = %d, want 0", g.PowerSupply)
	}

	gen1 := entities.NewGenerator("G1", 1)
	gen1.InsertBatteries(1)
	g.AddGenerator(gen1)

	g.UpdatePowerSupply()
	if g.PowerSupply != 100 {
		t.Errorf("PowerSupply with 1 powered gen on deck 0 = %d, want 100", g.PowerSupply)
	}

	gen2 := entities.NewGenerator("G2", 1)
	gen2.InsertBatteries(1)
	g.AddGenerator(gen2)

	g.UpdatePowerSupply()
	if g.PowerSupply != 200 {
		t.Errorf("PowerSupply with 2 powered gens on deck 0 = %d, want 200", g.PowerSupply)
	}
}

func TestUpdatePowerSupply_UnpoweredGeneratorNotCounted(t *testing.T) {
	g := NewGame()
	g.CurrentDeckID = 0

	powered := entities.NewGenerator("Powered", 1)
	powered.InsertBatteries(1)
	unpowered := entities.NewGenerator("Unpowered", 2)
	unpowered.InsertBatteries(1) // partial
	g.AddGenerator(powered)
	g.AddGenerator(unpowered)

	g.UpdatePowerSupply()
	if g.PowerSupply != 100 {
		t.Errorf("PowerSupply = %d, want 100 (only 1 powered gen)", g.PowerSupply)
	}
}

func TestUpdatePowerSupply_DeckDecay(t *testing.T) {
	// Deck 5 (0-based): output multiplier = 1.0 - 0.04*5 = 0.80
	g := NewGame()
	g.CurrentDeckID = 5

	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteries(1)
	g.AddGenerator(gen)

	g.UpdatePowerSupply()
	expected := int(100.0 * 0.80) // 80
	if g.PowerSupply != expected {
		t.Errorf("PowerSupply on deck 5 = %d, want %d (80%% of 100)", g.PowerSupply, expected)
	}
}

func TestUpdatePowerSupply_DeepDeckFloor(t *testing.T) {
	// Deck 9 (final): multiplier = max(1.0 - 0.04*9, 0.5) = max(0.64, 0.5) = 0.64
	g := NewGame()
	g.CurrentDeckID = 9

	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteries(1)
	g.AddGenerator(gen)

	g.UpdatePowerSupply()
	expected := int(100.0 * 0.64) // 64
	if g.PowerSupply != expected {
		t.Errorf("PowerSupply on deck 9 = %d, want %d", g.PowerSupply, expected)
	}
}

func TestGetAvailablePower(t *testing.T) {
	g := NewGame()
	g.PowerSupply = 200
	g.PowerConsumption = 50
	if got := g.GetAvailablePower(); got != 150 {
		t.Errorf("GetAvailablePower() = %d, want 150", got)
	}

	g.PowerConsumption = 250
	if got := g.GetAvailablePower(); got != -50 {
		t.Errorf("GetAvailablePower() with deficit = %d, want -50", got)
	}
}

func TestCalculatePowerConsumption_NoPoweredDevices(t *testing.T) {
	g := NewGame()
	g.Grid = nil
	if got := g.CalculatePowerConsumption(); got != 0 {
		t.Errorf("nil grid: CalculatePowerConsumption = %d, want 0", got)
	}
}

func TestSaveAndLoadDeckState_PreservesGenerators(t *testing.T) {
	g := NewGame()
	g.CurrentDeckID = 0

	gen1 := entities.NewGenerator("G1", 2)
	gen1.InsertBatteries(2)
	gen2 := entities.NewGenerator("G2", 3)
	gen2.InsertBatteries(1) // partial
	g.AddGenerator(gen1)
	g.AddGenerator(gen2)
	g.PowerSupply = 100

	// Need a grid for SaveCurrentDeckState
	g.Grid = makeMinimalGrid()

	g.SaveCurrentDeckState()

	ds := g.DeckStates[0]
	if ds == nil {
		t.Fatal("DeckStates[0] is nil after save")
	}
	if len(ds.Generators) != 2 {
		t.Fatalf("saved generators count = %d, want 2", len(ds.Generators))
	}
	if !ds.Generators[0].IsPowered() {
		t.Error("saved gen1 should be powered")
	}
	if ds.Generators[1].IsPowered() {
		t.Error("saved gen2 should NOT be powered (partial)")
	}

	// Clear and load
	g.Generators = nil
	g.PowerSupply = 0
	g.CurrentDeckID = 1

	g.LoadDeckState(0)

	if g.CurrentDeckID != 0 {
		t.Errorf("after load, CurrentDeckID = %d, want 0", g.CurrentDeckID)
	}
	if len(g.Generators) != 2 {
		t.Fatalf("after load, generators count = %d, want 2", len(g.Generators))
	}
	if !g.Generators[0].IsPowered() {
		t.Error("after load, gen1 should be powered")
	}
	if g.Generators[1].IsPowered() {
		t.Error("after load, gen2 should NOT be powered")
	}
	if g.PowerSupply != 0 {
		t.Errorf("after load, PowerSupply = %d, want 0 (caller must recalculate)", g.PowerSupply)
	}
}

func TestSaveAndLoadDeckState_PermanentInsertionRoundTrip(t *testing.T) {
	g := NewGame()
	g.CurrentDeckID = 0
	g.Grid = makeMinimalGrid()

	gen := entities.NewGenerator("G1", 2)
	gen.InsertBatteries(2)
	g.AddGenerator(gen)
	g.UpdatePowerSupply()

	if g.PowerSupply != 100 {
		t.Fatalf("pre-save PowerSupply = %d, want 100", g.PowerSupply)
	}

	g.SaveCurrentDeckState()

	// Simulate deck switch: clear live state
	g.Generators = nil
	g.PowerSupply = 0
	g.CurrentDeckID = 1

	// Restore and recalculate
	g.LoadDeckState(0)
	g.UpdatePowerSupply()

	if !g.Generators[0].IsPowered() {
		t.Error("generator should remain powered after round-trip (insertion is permanent)")
	}
	if g.PowerSupply != 100 {
		t.Errorf("PowerSupply after round-trip = %d, want 100", g.PowerSupply)
	}
}

func TestSaveAndLoadDeckState_DeepCopyIsolation(t *testing.T) {
	g := NewGame()
	g.CurrentDeckID = 0
	g.Grid = makeMinimalGrid()

	gen := entities.NewGenerator("G1", 2)
	gen.InsertBatteries(2)
	g.AddGenerator(gen)

	g.SaveCurrentDeckState()

	// Mutate the live generator after saving
	gen.BatteriesInserted = 0

	g.LoadDeckState(0)

	if !g.Generators[0].IsPowered() {
		t.Error("loaded generator should be powered — deep copy must isolate from post-save mutation")
	}
	if g.Generators[0].BatteriesInserted != 2 {
		t.Errorf("loaded BatteriesInserted = %d, want 2", g.Generators[0].BatteriesInserted)
	}
}

func TestAdvanceLevel_ResetsPowerState(t *testing.T) {
	g := NewGame()
	g.CurrentDeckID = 0
	g.Level = 1
	g.Grid = makeMinimalGrid()

	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteries(1)
	g.AddGenerator(gen)
	g.Batteries = 5
	g.PowerSupply = 100
	g.PowerConsumption = 30
	g.PowerOverloadWarned = true

	g.AdvanceLevel()

	if g.Batteries != 0 {
		t.Errorf("Batteries after advance = %d, want 0", g.Batteries)
	}
	if len(g.Generators) != 0 {
		t.Errorf("Generators after advance = %d, want 0", len(g.Generators))
	}
	if g.PowerSupply != 0 {
		t.Errorf("PowerSupply after advance = %d, want 0", g.PowerSupply)
	}
	if g.PowerConsumption != 0 {
		t.Errorf("PowerConsumption after advance = %d, want 0", g.PowerConsumption)
	}
	if g.PowerOverloadWarned {
		t.Error("PowerOverloadWarned should be false after advance")
	}
}

func makeMinimalGrid() *world.Grid {
	grid := world.NewGrid(2, 2)
	grid.MarkAsRoomWithName(0, 0, "R", "desc")
	grid.MarkAsRoomWithName(0, 1, "R", "desc")
	grid.MarkAsRoomWithName(1, 0, "R", "desc")
	grid.MarkAsRoomWithName(1, 1, "R", "desc")
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(1, 1)
	grid.BuildAllCellConnections()
	return grid
}

func TestNewGame_Defaults(t *testing.T) {
	g := NewGame()
	if g.Batteries != 0 {
		t.Errorf("Batteries = %d, want 0", g.Batteries)
	}
	if len(g.Generators) != 0 {
		t.Errorf("Generators len = %d, want 0", len(g.Generators))
	}
	if g.PowerSupply != 0 {
		t.Errorf("PowerSupply = %d, want 0", g.PowerSupply)
	}
	if g.PowerConsumption != 0 {
		t.Errorf("PowerConsumption = %d, want 0", g.PowerConsumption)
	}
	if g.PowerOverloadWarned {
		t.Error("PowerOverloadWarned = true, want false")
	}
}

func TestShortOutIfOverload_NoOverloadReturnsFalse(t *testing.T) {
	g := NewGame()
	g.CurrentDeckID = 0
	g.Grid = makeMinimalGrid()
	g.PowerSupply = 200
	g.RoomDoorsPowered["R"] = true
	g.PowerConsumption = g.CalculatePowerConsumption()

	got := g.ShortOutIfOverload("R")
	if got {
		t.Error("ShortOutIfOverload with consumption <= supply should return false")
	}
	if !g.RoomDoorsPowered["R"] {
		t.Error("protected room should remain powered")
	}
}

func TestShortOutIfOverload_UnpowersOthersUntilWithinSupply(t *testing.T) {
	// Grid: Room A (6 doors = 60w), Room B (6 doors = 60w). Supply 100w. Protect B, short A.
	grid := world.NewGrid(2, 6)
	for r := 0; r < 2; r++ {
		for c := 0; c < 6; c++ {
			room := "A"
			if c >= 3 {
				room = "B"
			}
			grid.MarkAsRoomWithName(r, c, room, "desc")
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(1, 5)
	grid.BuildAllCellConnections()

	for r := 0; r < 2; r++ {
		for c := 0; c < 3; c++ {
			gameworld.GetGameData(grid.GetCell(r, c)).Door = &entities.Door{RoomName: "A", Locked: false}
		}
	}
	for r := 0; r < 2; r++ {
		for c := 3; c < 6; c++ {
			gameworld.GetGameData(grid.GetCell(r, c)).Door = &entities.Door{RoomName: "B", Locked: false}
		}
	}

	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteries(1)
	g := NewGame()
	g.CurrentDeckID = 0
	g.Grid = grid
	g.AddGenerator(gen)
	g.RoomDoorsPowered = map[string]bool{"A": true, "B": true}
	g.RoomCCTVPowered = map[string]bool{"A": false, "B": false}
	g.UpdatePowerSupply()
	g.PowerConsumption = g.CalculatePowerConsumption()
	// 6+6 = 12 doors × 10w = 120w > 100w supply

	got := g.ShortOutIfOverload("B")
	if !got {
		t.Error("ShortOutIfOverload with overload should return true")
	}
	if !g.RoomDoorsPowered["B"] {
		t.Error("protected room B should remain powered")
	}
	if g.RoomDoorsPowered["A"] {
		t.Error("room A should be shorted out")
	}
	if g.PowerConsumption > g.PowerSupply {
		t.Errorf("after short-out: consumption %d > supply %d", g.PowerConsumption, g.PowerSupply)
	}
}

func TestShortOutIfOverload_DeterministicOrder(t *testing.T) {
	// Four rooms: RA,RB,RC,RD. Total 12 doors = 120w. Supply 100w. Protect RC (30w).
	grid := world.NewGrid(2, 8)
	for r := 0; r < 2; r++ {
		for c := 0; c < 8; c++ {
			var room string
			switch {
			case c < 2:
				room = "RA"
			case c < 4:
				room = "RB"
			case c < 6:
				room = "RC"
			default:
				room = "RD"
			}
			grid.MarkAsRoomWithName(r, c, room, "desc")
			gd := gameworld.InitGameData(grid.GetCell(r, c))
			gd.Door = &entities.Door{RoomName: room, Locked: false}
		}
	}
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(1, 7)
	grid.BuildAllCellConnections()

	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteries(1)
	g := NewGame()
	g.CurrentDeckID = 0
	g.Grid = grid
	g.AddGenerator(gen)
	g.RoomDoorsPowered = map[string]bool{"RA": true, "RB": true, "RC": true, "RD": true}
	g.RoomCCTVPowered = map[string]bool{"RA": false, "RB": false, "RC": false, "RD": false}
	g.UpdatePowerSupply()
	g.PowerConsumption = g.CalculatePowerConsumption()
	// 16 doors × 10w = 160w > 100w

	g.ShortOutIfOverload("RC")
	if !g.RoomDoorsPowered["RC"] {
		t.Error("protected room RC should remain powered")
	}
	if g.PowerConsumption > g.PowerSupply {
		t.Errorf("consumption %d > supply %d after short-out", g.PowerConsumption, g.PowerSupply)
	}
}
