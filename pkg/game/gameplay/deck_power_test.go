package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	gameworld "darkstation/pkg/game/world"
)

func TestAdvanceLevel_ClearsCarriedPowerState(t *testing.T) {
	g := BuildGame(1)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame failed")
	}

	deck1Gens := len(g.Generators)
	if deck1Gens == 0 {
		t.Fatal("deck 1 should have generators")
	}
	deck1GenPtr := g.Generators[0]

	// Fuel every generator on deck 1 and stock spare batteries.
	for _, gen := range g.Generators {
		gen.InsertBatteriesAndStart(gen.BatteriesRequired)
	}
	g.Batteries = 7
	g.UpdatePowerSupply()
	deck1Supply := g.PowerSupply
	if deck1Supply == 0 {
		t.Fatal("deck 1 supply should be > 0 after fueling generators")
	}

	AdvanceLevel(g)

	if g.Batteries != 0 {
		t.Fatalf("batteries after advance = %d, want 0", g.Batteries)
	}
	if len(g.Generators) == 0 {
		t.Fatal("deck 2 should have generators after advance")
	}
	if deck1GenPtr == g.Generators[0] {
		t.Fatal("deck 2 should not reuse deck 1 generator registration")
	}

	// Supply must come only from this deck's generators (typically the auto-started spawn gen).
	gridGenCount := 0
	poweredOnGrid := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen == nil {
			return
		}
		gridGenCount++
		if gen.IsPowered() {
			poweredOnGrid++
		}
	})
	if gridGenCount != len(g.Generators) {
		t.Fatalf("g.Generators len=%d grid generators=%d (must match)", len(g.Generators), gridGenCount)
	}
	if g.PowerSupply > deck1Supply && poweredOnGrid <= 1 {
		t.Fatalf("power supply %d looks like deck 1 carried over with only %d powered generator(s) on deck 2",
			g.PowerSupply, poweredOnGrid)
	}
}

func TestTravelToDeck_PreservesRunKeycards(t *testing.T) {
	g := BuildGame(1)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame failed")
	}

	g.AddRunKeycard(world.NewItem("Security Keycard"))
	if !g.HasRunKeycard("Security Keycard") {
		t.Fatal("setup: run keycard missing")
	}

	AdvanceLevel(g)

	if !g.HasRunKeycard("Security Keycard") {
		t.Error("run keycard should persist across deck travel")
	}
	if g.OwnedItems.Size() != 0 {
		t.Fatalf("deck-local OwnedItems after travel = %d, want 0", g.OwnedItems.Size())
	}
}

func TestRebuildGeneratorsFromGrid_UsesGridPointers(t *testing.T) {
	g := BuildGame(1)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame failed")
	}
	g.RebuildGeneratorsFromGrid()
	if len(g.Generators) == 0 {
		t.Fatal("expected generators on grid")
	}

	var fromGrid bool
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if fromGrid || cell == nil {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen != nil && gen == g.Generators[0] {
			fromGrid = true
		}
	})
	if !fromGrid {
		t.Fatal("g.Generators[0] should be the same pointer as the grid cell generator")
	}
}
