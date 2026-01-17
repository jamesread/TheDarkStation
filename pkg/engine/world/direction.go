package world

// Direction represents a cardinal direction
type Direction int

// Direction constants
const (
	North Direction = iota
	East
	South
	West
)

// AllDirections returns all valid directions for iteration
func AllDirections() []Direction {
	return []Direction{North, East, South, West}
}

// String returns the string representation of a direction
func (d Direction) String() string {
	switch d {
	case North:
		return "North"
	case East:
		return "East"
	case South:
		return "South"
	case West:
		return "West"
	default:
		return "Unknown"
	}
}

// IsValid returns true if the direction is a valid cardinal direction
func (d Direction) IsValid() bool {
	return d >= North && d <= West
}

// Opposite returns the opposite direction
func (d Direction) Opposite() Direction {
	switch d {
	case North:
		return South
	case South:
		return North
	case East:
		return West
	case West:
		return East
	default:
		return d
	}
}

// Delta returns the row and column offsets for this direction
func (d Direction) Delta() (rowDelta, colDelta int) {
	switch d {
	case North:
		return -1, 0
	case East:
		return 0, 1
	case South:
		return 1, 0
	case West:
		return 0, -1
	default:
		return 0, 0
	}
}
