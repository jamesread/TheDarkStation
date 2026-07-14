package levelgen

import (
	"testing"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestPlaceMaintenanceTerminals_LiftShaftUsesEastOfBottomLeft(t *testing.T) {
	g := state.NewGame()
	g.Level = 2
	g.Grid = generator.DefaultGenerator.Generate(2, deck.ThemeAirlock)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})
	avoid := mapset.New[*world.Cell]()
	setup.PlaceSpawnGeneratorForTest(g, &avoid)
	PlaceMaintenanceTerminals(g, &avoid)

	east := setup.LiftShaftCellEastOfBottomLeft(g)
	if east == nil {
		t.Fatal("missing lift shaft cell east of bottom-left")
	}
	if gameworld.GetGameData(east).MaintenanceTerm == nil {
		for _, cell := range setup.LiftShaftCellsFromBottomLeft(g) {
			if gameworld.GetGameData(cell).MaintenanceTerm != nil {
				t.Fatalf("maint at x:%d y:%d, want east of bottom-left x:%d y:%d", cell.Col, cell.Row, east.Col, east.Row)
			}
		}
		t.Fatalf("expected lift shaft maintenance terminal east of bottom-left at x:%d y:%d", east.Col, east.Row)
	}
}
