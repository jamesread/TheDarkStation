package setup

import (
	"testing"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/levelrand"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestBootstrapDeck1ShipSystems_placesFusionReactorAndConduits(t *testing.T) {
	levelrand.Seed(42)
	g := state.NewGame()
	g.Level = 1
	g.Grid = generator.BSP.Generate(1, deck.ThemeAirlock)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})

	avoid := mapset.New[*world.Cell]()
	placeSpawnGenerator(g, &avoid)
	BootstrapDeck1ShipSystems(g, &avoid)

	fusionCell := g.Grid.GetCell(generator.Deck1FusionReactorRow, generator.Deck1FusionReactorCol)
	if fusionCell == nil {
		t.Fatal("fusion reactor cell missing")
	}
	gen := gameworld.GetGameData(fusionCell).Generator
	if gen == nil || !gen.Permanent || !gen.IsPowered() {
		t.Fatalf("fusion reactor = %+v, want permanent powered generator", gen)
	}
	if gen.Name != generator.ShipFusionReactorName {
		t.Fatalf("name = %q, want %q", gen.Name, generator.ShipFusionReactorName)
	}

	shaftGen := findLiftShaftBootstrapGeneratorCell(g)
	if shaftGen == nil {
		t.Fatal("lift shaft bootstrap generator missing")
	}

	conduitCount := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		furn := gameworld.GetGameData(cell).Furniture
		if furn != nil && furn.PowerConduit {
			conduitCount++
		}
	})
	if conduitCount == 0 {
		t.Fatal("expected emergency power conduits on path to shaft generator")
	}

	if !RoomsOnConductiveGeneratorGrid(g)[generator.ShipRoomName] {
		t.Fatal("ship should be on conductive generator grid")
	}
}

func TestBootstrapDeck1ShipSystems_skipsOtherDecks(t *testing.T) {
	g := state.NewGame()
	g.Level = 2
	g.Grid = generator.DefaultGenerator.Generate(2, deck.ThemeAirlock)
	BootstrapDeck1ShipSystems(g, nil)

	var fusionCount int
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen != nil && gen.Permanent {
			fusionCount++
		}
	})
	if fusionCount != 0 {
		t.Fatalf("deck 2 should not get fusion reactor, found %d", fusionCount)
	}
}

func TestPermanentFusionReactor_immuneToTripAndBatteryInsert(t *testing.T) {
	gen := entities.NewPermanentFusionReactor(generator.ShipFusionReactorName)
	gen.Trip()
	if !gen.IsPowered() {
		t.Fatal("permanent reactor should stay powered after Trip")
	}
	if gen.InsertBatteriesAndStart(3) != 0 {
		t.Fatal("permanent reactor should ignore battery insertion")
	}
}
