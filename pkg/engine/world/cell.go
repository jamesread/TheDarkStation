package world

import (
	"github.com/zyedidia/generic/mapset"
)

// Cell represents a single cell/room in the grid
type Cell struct {
	Name        string
	Description string

	Row int
	Col int

	ItemsOnFloor  ItemSet
	RequiredItems ItemSet

	North *Cell
	East  *Cell
	South *Cell
	West  *Cell

	Visited    bool
	Discovered bool
	Room       bool

	ExitCell bool
	Locked   bool // Whether the exit is locked (requires key)
}

// IsExitLocked returns true if this is a locked exit cell
func (c *Cell) IsExitLocked() bool {
	return c.ExitCell && c.Locked
}

// IsExitUnlocked returns true if this is an unlocked exit cell
func (c *Cell) IsExitUnlocked() bool {
	return c.ExitCell && !c.Locked
}

// NewCell creates a new cell at the given position
func NewCell(row, col int, name, description string) *Cell {
	return &Cell{
		Name:          name,
		Description:   description,
		Row:           row,
		Col:           col,
		RequiredItems: mapset.New[*Item](),
		ItemsOnFloor:  mapset.New[*Item](),
	}
}

// HasConnections returns true if the cell has any nil connections
func (c *Cell) HasConnections() bool {
	return c.North == nil || c.East == nil || c.South == nil || c.West == nil
}

// GetNeighbor returns the neighboring cell in the given direction
func (c *Cell) GetNeighbor(dir Direction) *Cell {
	if c == nil {
		return nil
	}
	switch dir {
	case North:
		return c.North
	case East:
		return c.East
	case South:
		return c.South
	case West:
		return c.West
	default:
		return nil
	}
}

// SetNeighbor sets the neighboring cell in the given direction
func (c *Cell) SetNeighbor(dir Direction, neighbor *Cell) {
	if c == nil {
		return
	}
	switch dir {
	case North:
		c.North = neighbor
	case East:
		c.East = neighbor
	case South:
		c.South = neighbor
	case West:
		c.West = neighbor
	}
}
