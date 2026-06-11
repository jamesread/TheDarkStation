package setup

import (
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/state"
)

// LiftShaftCellsFromBottomLeft returns lift shaft cells ordered from the south-west corner
// (bottom-left on screen: max row, min col), scanning east along each row then north.
func LiftShaftCellsFromBottomLeft(g *state.Game) []*world.Cell {
	if g == nil || g.Grid == nil {
		return nil
	}
	grid := g.Grid
	topRow, leftCol, bottomRow, rightCol := generator.ShaftBoundsForLevel(grid.Rows(), grid.Cols(), g.Level)
	var out []*world.Cell
	for row := bottomRow; row >= topRow; row-- {
		for col := leftCol; col <= rightCol; col++ {
			cell := grid.GetCell(row, col)
			if cell != nil && cell.Room && cell.Name == generator.ShaftRoomName {
				out = append(out, cell)
			}
		}
	}
	return out
}

// LiftShaftBottomLeftCell returns the south-west corner cell of the centered lift shaft.
func LiftShaftBottomLeftCell(g *state.Game) *world.Cell {
	cells := LiftShaftCellsFromBottomLeft(g)
	if len(cells) == 0 {
		return nil
	}
	return cells[0]
}

func liftShaftCellIfValid(cell *world.Cell, avoid *mapset.Set[*world.Cell], valid func(*world.Cell, *mapset.Set[*world.Cell]) bool) *world.Cell {
	if cell == nil || cell.ExitCell {
		return nil
	}
	if avoid != nil && avoid.Has(cell) {
		return nil
	}
	if valid != nil && !valid(cell, avoid) {
		return nil
	}
	return cell
}

// liftShaftGeneratorCell places the bootstrap generator beside the south-west corner,
// leaving the corner cell for the bootstrap maintenance terminal.
func liftShaftGeneratorCell(g *state.Game, avoid *mapset.Set[*world.Cell]) *world.Cell {
	cells := LiftShaftCellsFromBottomLeft(g)
	if len(cells) >= 2 {
		if cell := liftShaftCellIfValid(cells[1], avoid, isValidForGenerator); cell != nil {
			return cell
		}
	}
	return liftShaftBootstrapCell(g, avoid, isValidForGenerator)
}

// liftShaftBootstrapCell picks the first valid cell in bottom-left shaft order.
func liftShaftBootstrapCell(g *state.Game, avoid *mapset.Set[*world.Cell], valid func(*world.Cell, *mapset.Set[*world.Cell]) bool) *world.Cell {
	for _, cell := range LiftShaftCellsFromBottomLeft(g) {
		if liftShaftCellIfValid(cell, avoid, valid) != nil {
			return cell
		}
	}
	return nil
}
