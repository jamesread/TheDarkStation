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

func TestRevealLiftShaftStructure_marksWallsAndCorners(t *testing.T) {
	g := state.NewGame()
	g.Level = 2
	g.Grid = generator.DefaultGenerator.Generate(2, deck.ThemeAirlock)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})

	RevealLiftShaftStructure(g)

	top, left, _, _ := generator.ShaftBoundsForLevel(g.Grid.Rows(), g.Grid.Cols(), g.Level)
	// Interior
	interior := g.Grid.GetCell(top, left)
	if interior == nil || !interior.Discovered {
		t.Fatal("expected shaft interior discovered")
	}
	// Corner wall (diagonally adjacent — not revealed by adjacent-room alone)
	corner := g.Grid.GetCell(top-1, left-1)
	if corner == nil || corner.Room {
		t.Fatal("expected non-room corner wall cell")
	}
	if !corner.Discovered {
		t.Fatal("expected lift corner wall discovered")
	}
	// Orthogonal wall
	north := g.Grid.GetCell(top-1, left)
	if north == nil || north.Room || !north.Discovered {
		t.Fatal("expected lift edge wall discovered")
	}
	// Outside the ring stays undiscovered
	outside := g.Grid.GetCell(top-2, left)
	if outside != nil && !outside.Room && outside.Discovered {
		t.Fatal("wall outside lift ring should not be force-discovered")
	}
}

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
