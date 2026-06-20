package setup

import (
	"testing"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestLiftShaftBottomLeftCell(t *testing.T) {
	g := state.NewGame()
	g.Level = 2
	g.Grid = generator.DefaultGenerator.Generate(2, deck.ThemeAirlock)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})

	corner := LiftShaftBottomLeftCell(g)
	if corner == nil {
		t.Fatal("expected lift shaft bottom-left cell")
	}
	if corner.Name != generator.ShaftRoomName {
		t.Fatalf("corner room = %q, want %q", corner.Name, generator.ShaftRoomName)
	}
	_, leftCol, bottomRow, _ := generator.ShaftBoundsForLevel(g.Grid.Rows(), g.Grid.Cols(), g.Level)
	if corner.Row != bottomRow || corner.Col != leftCol {
		t.Fatalf("corner at x:%d y:%d, want x:%d y:%d", corner.Col, corner.Row, leftCol, bottomRow)
	}
}

func TestPlaceSpawnGenerator_UsesLiftShaftBottomLeft(t *testing.T) {
	g := state.NewGame()
	g.Level = 2
	g.Grid = generator.DefaultGenerator.Generate(2, deck.ThemeAirlock)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})
	avoid := mapset.New[*world.Cell]()
	placeSpawnGenerator(g, &avoid)

	_, leftCol, bottomRow, _ := generator.ShaftBoundsForLevel(g.Grid.Rows(), g.Grid.Cols(), g.Level)
	genCell := g.Grid.GetCell(bottomRow, leftCol)
	if genCell == nil || gameworld.GetGameData(genCell).Generator == nil {
		t.Fatalf("expected spawn generator at lift shaft x:%d y:%d", leftCol, bottomRow)
	}
	east := LiftShaftCellEastOfBottomLeft(g)
	if east != nil && gameworld.GetGameData(east).Generator != nil {
		t.Fatal("cell east of bottom-left should be reserved for the maintenance terminal")
	}
}
