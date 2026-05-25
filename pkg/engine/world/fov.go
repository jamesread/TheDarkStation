package world

import "math"

// SightBlocker returns true when the cell is visible but blocks sight beyond it
// (e.g. an unpowered door). Nil means only non-room cells block rays.
type SightBlocker func(cell *Cell) bool

// CalculateFOV returns cells visible from center using ray casting.
// A Bresenham ray is traced toward every grid cell; each room cell along a ray is
// visible until the first non-room (wall) cell or sight blocker, which stops the ray.
// Rays are not length-limited and do not bounce.
func CalculateFOV(grid *Grid, center *Cell, blockSight SightBlocker) []*Cell {
	if center == nil || grid == nil {
		return nil
	}

	visible := make(map[*Cell]bool)
	centerRow, centerCol := center.Row, center.Col
	markVisibleRoom(grid, centerRow, centerCol, visible, blockSight)

	grid.ForEachCell(func(row, col int, _ *Cell) {
		if row == centerRow && col == centerCol {
			return
		}
		castRay(grid, centerRow, centerCol, row, col, visible, blockSight)
	})

	result := make([]*Cell, 0, len(visible))
	for cell := range visible {
		result = append(result, cell)
	}
	return result
}

// VisibleCellSet returns a set of cells visible from center (same rules as CalculateFOV).
func VisibleCellSet(grid *Grid, center *Cell, blockSight SightBlocker) map[*Cell]bool {
	cells := CalculateFOV(grid, center, blockSight)
	set := make(map[*Cell]bool, len(cells))
	for _, c := range cells {
		set[c] = true
	}
	return set
}

// FOVRay is one ray-cast segment from a FOV center to its endpoint (last visible room cell).
type FOVRay struct {
	EndRow, EndCol int
}

// RayCastEndpoint traces a Bresenham ray from (r0,c0) toward (r1,c1) and returns the last room
// cell reached before a wall or sight blocker stops the ray.
func RayCastEndpoint(grid *Grid, r0, c0, r1, c1 int, blockSight SightBlocker) (endRow, endCol int, ok bool) {
	var lastR, lastC int
	found := false
	walkRay(grid, r0, c0, r1, c1, func(row, col int) bool {
		cell := grid.GetCell(row, col)
		if cell == nil || !cell.Room {
			return false
		}
		lastR, lastC = row, col
		found = true
		if blockSight != nil && blockSight(cell) {
			return false
		}
		return true
	})
	if !found {
		return 0, 0, false
	}
	return lastR, lastC, true
}

// CollectFOVRays returns one ray per unique endpoint used by CalculateFOV (same cast targets).
func CollectFOVRays(grid *Grid, center *Cell, blockSight SightBlocker) []FOVRay {
	if center == nil || grid == nil {
		return nil
	}
	centerRow, centerCol := center.Row, center.Col
	seen := make(map[[2]int]struct{})
	var rays []FOVRay
	grid.ForEachCell(func(row, col int, _ *Cell) {
		if row == centerRow && col == centerCol {
			return
		}
		er, ec, ok := RayCastEndpoint(grid, centerRow, centerCol, row, col, blockSight)
		if !ok {
			return
		}
		key := [2]int{er, ec}
		if _, dup := seen[key]; dup {
			return
		}
		seen[key] = struct{}{}
		rays = append(rays, FOVRay{EndRow: er, EndCol: ec})
	})
	return rays
}

// castRay traces a Bresenham line and marks room cells visible until a wall or blocker stops the ray.
func castRay(grid *Grid, r0, c0, r1, c1 int, visible map[*Cell]bool, blockSight SightBlocker) {
	walkRay(grid, r0, c0, r1, c1, func(row, col int) bool {
		return markVisibleRoom(grid, row, col, visible, blockSight)
	})
}

func walkRay(grid *Grid, r0, c0, r1, c1 int, visit func(row, col int) bool) {
	dr := r1 - r0
	dc := c1 - c0
	if dr == 0 && dc == 0 {
		visit(r0, c0)
		return
	}

	stepR := 0
	if dr > 0 {
		stepR = 1
	} else if dr < 0 {
		stepR = -1
	}
	stepC := 0
	if dc > 0 {
		stepC = 1
	} else if dc < 0 {
		stepC = -1
	}

	absDr := dr
	if absDr < 0 {
		absDr = -absDr
	}
	absDc := dc
	if absDc < 0 {
		absDc = -absDc
	}

	r, c := r0, c0

	if absDr >= absDc {
		err := 2*absDc - absDr
		for {
			if !visit(r, c) {
				return
			}
			if r == r1 && c == c1 {
				return
			}
			if err > 0 {
				c += stepC
				err -= 2 * absDr
			}
			r += stepR
			err += 2 * absDc
		}
	}

	err := 2*absDr - absDc
	for {
		if !visit(r, c) {
			return
		}
		if r == r1 && c == c1 {
			return
		}
		if err > 0 {
			r += stepR
			err -= 2 * absDc
		}
		c += stepC
		err += 2 * absDr
	}
}

// markVisibleRoom adds a room cell to visible; returns false if sight cannot continue past the cell.
func markVisibleRoom(grid *Grid, row, col int, visible map[*Cell]bool, blockSight SightBlocker) bool {
	cell := grid.GetCell(row, col)
	if cell == nil || !cell.Room {
		return false
	}
	visible[cell] = true
	if blockSight != nil && blockSight(cell) {
		return false
	}
	return true
}

// DistanceCells returns Euclidean distance between two grid cells (for tests/tools).
func DistanceCells(a, b *Cell) float64 {
	if a == nil || b == nil {
		return math.MaxFloat64
	}
	dr := float64(a.Row - b.Row)
	dc := float64(a.Col - b.Col)
	return math.Sqrt(dr*dr + dc*dc)
}

// RevealFOV marks all cells within line-of-sight of the center cell as discovered.
// Visited is set only when the player steps on a cell (see gameplay movement).
func RevealFOV(grid *Grid, center *Cell, blockSight SightBlocker) {
	visibleCells := CalculateFOV(grid, center, blockSight)
	for _, cell := range visibleCells {
		cell.Discovered = true
	}
}

// RevealFOVDefault reveals cells using ray-cast line of sight from center.
func RevealFOVDefault(grid *Grid, center *Cell, blockSight SightBlocker) {
	RevealFOV(grid, center, blockSight)
}
