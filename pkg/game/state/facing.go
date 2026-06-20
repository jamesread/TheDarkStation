package state

import "darkstation/pkg/engine/world"

// PlayerFacing is the direction the player is looking (orthogonal map orientation).
type PlayerFacing int

const (
	FaceNorth PlayerFacing = iota
	FaceSouth
	FaceEast
	FaceWest
)

// Icon returns the map glyph indicating look direction.
func (f PlayerFacing) Icon() string {
	switch f {
	case FaceSouth:
		return "↓"
	case FaceEast:
		return "→"
	case FaceWest:
		return "←"
	default:
		return "↑"
	}
}

// Delta returns the (rowDelta, colDelta) unit vector for this facing
// (north = decreasing row, east = increasing col).
func (f PlayerFacing) Delta() (dRow, dCol int) {
	switch f {
	case FaceSouth:
		return 1, 0
	case FaceEast:
		return 0, 1
	case FaceWest:
		return 0, -1
	default:
		return -1, 0
	}
}

// FacingToward returns the facing from one cell to an orthogonally adjacent neighbor.
func FacingToward(from, target *world.Cell) (PlayerFacing, bool) {
	if from == nil || target == nil {
		return 0, false
	}
	switch target {
	case from.North:
		return FaceNorth, true
	case from.South:
		return FaceSouth, true
	case from.East:
		return FaceEast, true
	case from.West:
		return FaceWest, true
	default:
		return 0, false
	}
}

// AdjacentCellsClockwiseFromFacing returns orthogonal neighbors starting with the cell
// in the player's facing direction, then continuing clockwise (N→E→S→W).
func AdjacentCellsClockwiseFromFacing(from *world.Cell, facing PlayerFacing) []*world.Cell {
	if from == nil {
		return nil
	}
	start := 0
	switch facing {
	case FaceEast:
		start = 1
	case FaceSouth:
		start = 2
	case FaceWest:
		start = 3
	}
	// Clockwise from north: N, E, S, W.
	lookup := []func(*world.Cell) *world.Cell{
		func(c *world.Cell) *world.Cell { return c.North },
		func(c *world.Cell) *world.Cell { return c.East },
		func(c *world.Cell) *world.Cell { return c.South },
		func(c *world.Cell) *world.Cell { return c.West },
	}
	out := make([]*world.Cell, 0, 4)
	for i := 0; i < 4; i++ {
		out = append(out, lookup[(start+i)%4](from))
	}
	return out
}
