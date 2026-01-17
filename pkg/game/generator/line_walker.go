package generator

import (
	"math/rand"

	"darkcastle/pkg/engine/world"
)

// LineWalkerGenerator generates maps by walking lines in random directions
// with branching probability
type LineWalkerGenerator struct{}

// Name returns the name of this generator
func (g *LineWalkerGenerator) Name() string {
	return "Line Walker"
}

// Generate creates a new grid for the given level
func (g *LineWalkerGenerator) Generate(level int) *world.Grid {
	grid := &world.Grid{}

	// Scale grid size with level (add 2 extra for perimeter walls)
	// Level 1: 12x22 (10x20 playable), Level 10: ~30x58 playable
	baseRows := 8 + 2 // +2 for perimeter
	baseCols := 16 + 2
	rows := baseRows + (level * 2)
	cols := baseCols + (level * 4)

	grid.Build(rows, cols)

	// Start in the center (which is always in playable area)
	row, col := grid.CenterPosition()

	grid.SetStartCellAt(row, col)

	// Scale branch probability with level (more complex layouts)
	// Level 1: 0.28, Level 10: 0.55
	branchProb := float32(0.25) + float32(level)*0.03
	if branchProb > 0.65 {
		branchProb = 0.65
	}

	// Scale corridor length with level
	// Level 1: 2-4, Level 10: 4-9
	minDist := 2 + (level / 4)
	maxDist := 4 + (level / 2)

	// Build main corridors in all four directions
	g.buildLineOfRooms(grid, row, col, world.North, branchProb, minDist, maxDist)
	g.buildLineOfRooms(grid, row, col, world.East, branchProb, minDist, maxDist)
	g.buildLineOfRooms(grid, row, col, world.South, branchProb, minDist, maxDist)
	exitCellRow, exitCellCol := g.buildLineOfRooms(grid, row, col, world.West, branchProb, minDist, maxDist)

	// Add extra corridors at higher levels for more complexity
	extraCorridors := level / 2
	for i := 0; i < extraCorridors; i++ {
		// Start from a random position near center
		randRow := row + rand.Intn(5) - 2
		randCol := col + rand.Intn(5) - 2
		if grid.IsPlayablePosition(randRow, randCol) {
			g.buildLineOfRoomsRandom(grid, randRow, randCol, branchProb, minDist, maxDist)
		}
	}

	grid.SetExitCellAt(exitCellRow, exitCellCol)

	grid.BuildAllCellConnections()

	// Validate the generated grid
	if err := grid.Validate(); err != "" {
		panic("Generated invalid grid: " + err)
	}

	return grid
}

// randomDirection returns a random cardinal direction
func (g *LineWalkerGenerator) randomDirection() world.Direction {
	return world.Direction(rand.Intn(4))
}

// buildLineOfRoomsRandom creates a line of rooms in a random direction
func (g *LineWalkerGenerator) buildLineOfRoomsRandom(grid *world.Grid, row, col int, branchProbability float32, minDist, maxDist int) (int, int) {
	return g.buildLineOfRooms(grid, row, col, g.randomDirection(), branchProbability, minDist, maxDist)
}

// buildLineOfRooms creates a line of rooms starting from (row, col) in the given direction
// Rooms are only placed within the playable area (not on the perimeter)
func (g *LineWalkerGenerator) buildLineOfRooms(grid *world.Grid, row, col int, dir world.Direction, branchProbability float32, minDist, maxDist int) (int, int) {
	if !dir.IsValid() {
		dir = g.randomDirection()
	}

	rowDelta, colDelta := dir.Delta()

	distance := minDist + rand.Intn(maxDist-minDist+1)

	for segment := 0; segment < distance; segment++ {
		// Only mark as room if within playable area (not on perimeter)
		if grid.IsPlayablePosition(row, col) {
			grid.MarkAsRoom(row, col)
		}

		// If the next cell would be outside playable area, stop here
		if !grid.IsPlayablePosition(row+rowDelta, col+colDelta) {
			return row, col
		}

		if rand.Float32() < branchProbability {
			g.buildLineOfRoomsRandom(grid, row, col, branchProbability-.1, minDist, maxDist)
		}

		row += rowDelta
		col += colDelta
	}

	// Mark the final cell we moved to as a room (if in playable area)
	if grid.IsPlayablePosition(row, col) {
		grid.MarkAsRoom(row, col)
	}

	return row, col
}
