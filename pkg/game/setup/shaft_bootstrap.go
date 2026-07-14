package setup

import (
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/state"
)

// IsLiftShaftBoundsCell reports whether cell lies inside the centered lift shaft footprint.
// On grids smaller than the fixed 5×5 shaft, only cells named Lift Shaft are treated as shaft.
func IsLiftShaftBoundsCell(g *state.Game, cell *world.Cell) bool {
	if g == nil || g.Grid == nil || cell == nil {
		return false
	}
	rows, cols := g.Grid.Rows(), g.Grid.Cols()
	const shaftSize = 5
	if rows < shaftSize || cols < shaftSize {
		return cell.Name == generator.ShaftRoomName
	}
	top, left, bottom, right := generator.ShaftBoundsForLevel(rows, cols, g.Level)
	return cell.Row >= top && cell.Row <= bottom && cell.Col >= left && cell.Col <= right
}
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

// LiftShaftCellEastOfBottomLeft returns the cell immediately east of the south-west corner.
func LiftShaftCellEastOfBottomLeft(g *state.Game) *world.Cell {
	cells := LiftShaftCellsFromBottomLeft(g)
	if len(cells) >= 2 {
		return cells[1]
	}
	return nil
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

// liftShaftGeneratorCell places the bootstrap generator in the south-west corner,
// leaving the cell east of the corner for the bootstrap maintenance terminal.
func liftShaftGeneratorCell(g *state.Game, avoid *mapset.Set[*world.Cell]) *world.Cell {
	cells := LiftShaftCellsFromBottomLeft(g)
	if len(cells) >= 1 {
		if cell := liftShaftCellIfValid(cells[0], avoid, isValidForGenerator); cell != nil {
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
