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

	ExitCell      bool
	Locked        bool           // Whether the exit is locked (requires generators)
	Generator     *Generator     // Generator in this cell (if any)
	Door          *Door          // Door in this cell (if any)
	Terminal      *CCTVTerminal  // CCTV terminal in this cell (if any)
	Furniture     *Furniture     // Furniture in this cell (if any)
	Hazard        *Hazard        // Environmental hazard in this cell (if any)
	HazardControl *HazardControl // Hazard control panel in this cell (if any)
}

// HasFurniture returns true if this cell contains furniture
func (c *Cell) HasFurniture() bool {
	return c.Furniture != nil
}

// HasUncheckedFurniture returns true if this cell has furniture that hasn't been examined
func (c *Cell) HasUncheckedFurniture() bool {
	return c.Furniture != nil && !c.Furniture.Checked
}

// HasCheckedFurniture returns true if this cell has furniture that has been examined
func (c *Cell) HasCheckedFurniture() bool {
	return c.Furniture != nil && c.Furniture.Checked
}

// HasHazard returns true if this cell contains a hazard
func (c *Cell) HasHazard() bool {
	return c.Hazard != nil
}

// HasBlockingHazard returns true if this cell has an unfixed hazard
func (c *Cell) HasBlockingHazard() bool {
	return c.Hazard != nil && c.Hazard.IsBlocking()
}

// HasFixedHazard returns true if this cell has a fixed hazard
func (c *Cell) HasFixedHazard() bool {
	return c.Hazard != nil && !c.Hazard.IsBlocking()
}

// HasHazardControl returns true if this cell contains a hazard control
func (c *Cell) HasHazardControl() bool {
	return c.HazardControl != nil
}

// HasInactiveHazardControl returns true if this cell has an unactivated control
func (c *Cell) HasInactiveHazardControl() bool {
	return c.HazardControl != nil && !c.HazardControl.Activated
}

// HasActiveHazardControl returns true if this cell has an activated control
func (c *Cell) HasActiveHazardControl() bool {
	return c.HazardControl != nil && c.HazardControl.Activated
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
