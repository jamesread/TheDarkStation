// Package world provides generic 2D grid-based world primitives.
// These are engine-level constructs usable by any tile-based game.
package world

import (
	"github.com/zyedidia/generic/mapset"
)

// Cell represents a single cell/tile in the grid.
// This is a generic engine primitive that can be extended by games.
type Cell struct {
	// Basic identification
	Name        string
	Description string

	// Grid position
	Row int
	Col int

	// Item management (generic items on the floor)
	ItemsOnFloor  ItemSet
	RequiredItems ItemSet

	// Navigation - links to adjacent cells
	North *Cell
	East  *Cell
	South *Cell
	West  *Cell

	// Visibility state
	Visited    bool
	Discovered bool

	// Cell type flags
	Room     bool // Is this cell a walkable room/corridor?
	ExitCell bool // Is this the exit/goal cell?
	Locked   bool // Is this cell locked (requires condition to enter)?

	// GameData holds game-specific extensions.
	// Games should cast this to their specific type (e.g., *GameCellData).
	// This allows the engine Cell to remain generic while supporting
	// game-specific entities like doors, furniture, hazards, etc.
	GameData interface{}
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

// IsExitLocked returns true if this is a locked exit cell
func (c *Cell) IsExitLocked() bool {
	return c.ExitCell && c.Locked
}

// IsExitUnlocked returns true if this is an unlocked exit cell
func (c *Cell) IsExitUnlocked() bool {
	return c.ExitCell && !c.Locked
}

// GetNeighbors returns all non-nil adjacent cells
func (c *Cell) GetNeighbors() []*Cell {
	var neighbors []*Cell
	if c.North != nil {
		neighbors = append(neighbors, c.North)
	}
	if c.East != nil {
		neighbors = append(neighbors, c.East)
	}
	if c.South != nil {
		neighbors = append(neighbors, c.South)
	}
	if c.West != nil {
		neighbors = append(neighbors, c.West)
	}
	return neighbors
}
