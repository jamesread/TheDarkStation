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

func TestPlaceMaintenanceTerminals_LiftShaftUsesBottomLeft(t *testing.T) {
	g := state.NewGame()
	g.Level = 2
	g.Grid = generator.DefaultGenerator.Generate(2, deck.ThemeAirlock)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})
	avoid := mapset.New[*world.Cell]()
	PlaceMaintenanceTerminals(g, &avoid)

	corner := setup.LiftShaftBottomLeftCell(g)
	if corner == nil {
		t.Fatal("missing lift shaft corner")
	}
	if gameworld.GetGameData(corner).MaintenanceTerm == nil {
		for _, cell := range setup.LiftShaftCellsFromBottomLeft(g) {
			if gameworld.GetGameData(cell).MaintenanceTerm != nil {
				t.Fatalf("maint at x:%d y:%d, want bottom-left x:%d y:%d", cell.Col, cell.Row, corner.Col, corner.Row)
			}
		}
		t.Fatalf("expected lift shaft maintenance terminal at bottom-left x:%d y:%d", corner.Col, corner.Row)
	}
}
