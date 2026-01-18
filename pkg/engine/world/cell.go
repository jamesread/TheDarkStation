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

	ExitCell  bool
	Locked    bool          // Whether the exit is locked (requires generators)
	Generator *Generator    // Generator in this cell (if any)
	Door      *Door         // Door in this cell (if any)
	Terminal  *CCTVTerminal // CCTV terminal in this cell (if any)
	Furniture *Furniture    // Furniture in this cell (if any)
}

// HasFurniture returns true if this cell contains furniture
func (c *Cell) HasFurniture() bool {
	return c.Furniture != nil
}

// HasTerminal returns true if this cell contains a CCTV terminal
func (c *Cell) HasTerminal() bool {
	return c.Terminal != nil
}

// HasUnusedTerminal returns true if this cell has an unused CCTV terminal
func (c *Cell) HasUnusedTerminal() bool {
	return c.Terminal != nil && !c.Terminal.Used
}

// HasUsedTerminal returns true if this cell has a used CCTV terminal
func (c *Cell) HasUsedTerminal() bool {
	return c.Terminal != nil && c.Terminal.Used
}

// HasGenerator returns true if this cell contains a generator
func (c *Cell) HasGenerator() bool {
	return c.Generator != nil
}

// HasUnpoweredGenerator returns true if this cell has an unpowered generator
func (c *Cell) HasUnpoweredGenerator() bool {
	return c.Generator != nil && !c.Generator.IsPowered()
}

// HasPoweredGenerator returns true if this cell has a powered generator
func (c *Cell) HasPoweredGenerator() bool {
	return c.Generator != nil && c.Generator.IsPowered()
}

// HasDoor returns true if this cell contains a door
func (c *Cell) HasDoor() bool {
	return c.Door != nil
}

// HasLockedDoor returns true if this cell has a locked door
func (c *Cell) HasLockedDoor() bool {
	return c.Door != nil && c.Door.Locked
}

// HasUnlockedDoor returns true if this cell has an unlocked door
func (c *Cell) HasUnlockedDoor() bool {
	return c.Door != nil && !c.Door.Locked
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
