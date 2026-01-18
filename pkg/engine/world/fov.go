package world

import (
	"math"
)

// FOVRadius is the default field of view radius
const FOVRadius = 3

// CalculateFOV calculates which cells are visible from a given cell within a radius
// Uses raycasting to determine line of sight - walls block visibility
func CalculateFOV(grid *Grid, center *Cell, radius int) []*Cell {
	if center == nil || grid == nil {
		return nil
	}

	visible := make(map[*Cell]bool)
	visible[center] = true

	// Cast rays in all directions
	// Use enough rays to cover all cells at the edge of the radius
	numRays := int(2 * math.Pi * float64(radius) * 2) // Roughly 2 rays per cell at perimeter
	if numRays < 36 {
		numRays = 36 // Minimum 36 rays (every 10 degrees)
	}

	for i := 0; i < numRays; i++ {
		angle := (2 * math.Pi * float64(i)) / float64(numRays)
		castRay(grid, center, angle, radius, visible)
	}

	// Convert map to slice
	result := make([]*Cell, 0, len(visible))
	for cell := range visible {
		result = append(result, cell)
	}

	return result
}

// castRay casts a single ray from center at the given angle
func castRay(grid *Grid, center *Cell, angle float64, radius int, visible map[*Cell]bool) {
	dx := math.Cos(angle)
	dy := math.Sin(angle)

	// Step along the ray
	for dist := 1; dist <= radius; dist++ {
		// Calculate cell position
		col := center.Col + int(math.Round(dx*float64(dist)))
		row := center.Row + int(math.Round(dy*float64(dist)))

		cell := grid.GetCell(row, col)
		if cell == nil {
			break // Out of bounds
		}

		// Mark as visible
		visible[cell] = true

		// If this cell blocks vision (not a room/walkable), stop the ray
		if !cell.Room {
			break
		}
	}
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
