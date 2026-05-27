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
