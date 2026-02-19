package world

// FOVRadius is the default field of view radius (Chebyshev/diamond distance).
// Symmetric in all 8 directions; increased from 3 to 4 for better corner coverage.
const FOVRadius = 4

// CalculateFOV calculates which cells are visible from a given cell within a radius.
// Uses a symmetric diamond (Chebyshev) shape with Bresenham line-of-sight for natural,
// equal coverage in all directions. Walls (non-room cells) block visibility.
func CalculateFOV(grid *Grid, center *Cell, radius int) []*Cell {
	if center == nil || grid == nil {
		return nil
	}

	visible := make(map[*Cell]bool)
	visible[center] = true

	centerRow, centerCol := center.Row, center.Col

	// Iterate over diamond (Chebyshev) - symmetric in all 8 directions
	for dr := -radius; dr <= radius; dr++ {
		for dc := -radius; dc <= radius; dc++ {
			// Chebyshev distance: max(|dr|, |dc|) <= radius
			if chebyshevDist(dr, dc) > radius {
				continue
			}

			row := centerRow + dr
			col := centerCol + dc

			cell := grid.GetCell(row, col)
			if cell == nil {
				continue
			}

			// Bresenham line-of-sight: trace from center to cell
			if hasLineOfSight(grid, centerRow, centerCol, row, col) {
				visible[cell] = true
			}
		}
	}

	result := make([]*Cell, 0, len(visible))
	for cell := range visible {
		result = append(result, cell)
	}
	return result
}

// chebyshevDist returns Chebyshev (chessboard) distance for (dr, dc).
func chebyshevDist(dr, dc int) int {
	absDr := dr
	if absDr < 0 {
		absDr = -absDr
	}
	absDc := dc
	if absDc < 0 {
		absDc = -absDc
	}
	if absDr > absDc {
		return absDr
	}
	return absDc
}

// hasLineOfSight returns true if there's a clear path from (r0,c0) to (r1,c1).
// Uses Bresenham's line algorithm; vision is blocked by non-room cells.
func hasLineOfSight(grid *Grid, r0, c0, r1, c1 int) bool {
	dr := r1 - r0
	dc := c1 - c0

	if dr == 0 && dc == 0 {
		return true
	}

	absDr := dr
	if absDr < 0 {
		absDr = -absDr
	}
	absDc := dc
	if absDc < 0 {
		absDc = -absDc
	}

	// Bresenham: step along the longer axis
	var stepR, stepC int
	if dr > 0 {
		stepR = 1
	} else if dr < 0 {
		stepR = -1
	}
	if dc > 0 {
		stepC = 1
	} else if dc < 0 {
		stepC = -1
	}

	r, c := r0, c0

	if absDr >= absDc {
		// Step along rows
		err := 2*absDc - absDr
		for r != r1 {
			r += stepR
			if err > 0 {
				c += stepC
				err -= 2 * absDr
			}
			err += 2 * absDc

			cell := grid.GetCell(r, c)
			if cell == nil {
				return false
			}
			if !cell.Room {
				return false // blocked before reaching target
			}
		}
	} else {
		// Step along cols
		err := 2*absDr - absDc
		for c != c1 {
			c += stepC
			if err > 0 {
				r += stepR
				err -= 2 * absDc
			}
			err += 2 * absDr

			cell := grid.GetCell(r, c)
			if cell == nil {
				return false
			}
			if !cell.Room {
				return false
			}
		}
	}

	return true
}

// RevealFOV marks all cells within FOV of the center cell as discovered
// Floor cells (rooms) within FOV are also marked as visited
func RevealFOV(grid *Grid, center *Cell, radius int) {
	visibleCells := CalculateFOV(grid, center, radius)
	for _, cell := range visibleCells {
		cell.Discovered = true
		// Mark floor cells as visited (player can see them clearly)
		if cell.Room {
			cell.Visited = true
		}
	}
}

// RevealFOVDefault reveals cells using the default FOV radius
func RevealFOVDefault(grid *Grid, center *Cell) {
	RevealFOV(grid, center, FOVRadius)
}
