// Package world provides game-specific world extensions for The Dark Station.
// It extends the generic engine/world primitives with space station themed entities.
package world

import (
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
)

// GameCellData holds game-specific entity references for a cell.
// This is stored in the engine Cell's GameData field.
type GameCellData struct {
	Generator       *entities.Generator           // Power generator in this cell (if any)
	Door            *entities.Door                // Keycard door in this cell (if any)
	Terminal        *entities.CCTVTerminal        // CCTV terminal in this cell (if any)
	Puzzle          *entities.PuzzleTerminal      // Puzzle terminal in this cell (if any)
	Furniture       *entities.Furniture           // Furniture in this cell (if any)
	Hazard          *entities.Hazard              // Environmental hazard in this cell (if any)
	HazardControl   *entities.HazardControl       // Hazard control panel in this cell (if any)
	MaintenanceTerm *entities.MaintenanceTerminal // Maintenance terminal in this cell (if any)
	LightsOn        bool                          // Whether lights are on in this cell
	Lighted         bool                          // Whether this cell has been lit (stays explored)
}

// InitGameData initializes game data for a cell if not already set
func InitGameData(cell *world.Cell) *GameCellData {
	if cell.GameData == nil {
		cell.GameData = &GameCellData{}
	}
	return cell.GameData.(*GameCellData)
}

// GetGameData retrieves game data from a cell, initializing if needed
func GetGameData(cell *world.Cell) *GameCellData {
	return InitGameData(cell)
}

// Helper functions for checking entity presence on cells

// HasFurniture returns true if this cell contains furniture
func HasFurniture(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Furniture != nil
}

// HasUncheckedFurniture returns true if this cell has furniture that hasn't been examined
func HasUncheckedFurniture(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Furniture != nil && !data.Furniture.Checked
}

// HasCheckedFurniture returns true if this cell has furniture that has been examined
func HasCheckedFurniture(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Furniture != nil && data.Furniture.Checked
}

// HasHazard returns true if this cell contains a hazard
func HasHazard(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Hazard != nil
}

// HasBlockingHazard returns true if this cell has an unfixed hazard
func HasBlockingHazard(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Hazard != nil && data.Hazard.IsBlocking()
}

// HasFixedHazard returns true if this cell has a fixed hazard
func HasFixedHazard(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Hazard != nil && !data.Hazard.IsBlocking()
}

// HasHazardControl returns true if this cell contains a hazard control
func HasHazardControl(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.HazardControl != nil
}

// HasInactiveHazardControl returns true if this cell has an unactivated control
func HasInactiveHazardControl(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.HazardControl != nil && !data.HazardControl.Activated
}

// HasActiveHazardControl returns true if this cell has an activated control
func HasActiveHazardControl(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.HazardControl != nil && data.HazardControl.Activated
}

// HasTerminal returns true if this cell contains a CCTV terminal
func HasTerminal(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Terminal != nil
}

// HasUnusedTerminal returns true if this cell has an unused CCTV terminal
func HasUnusedTerminal(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Terminal != nil && !data.Terminal.Used
}

// HasUsedTerminal returns true if this cell has a used CCTV terminal
func HasUsedTerminal(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Terminal != nil && data.Terminal.Used
}

// HasGenerator returns true if this cell contains a generator
func HasGenerator(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Generator != nil
}

// HasUnpoweredGenerator returns true if this cell has an unpowered generator
func HasUnpoweredGenerator(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Generator != nil && !data.Generator.IsPowered()
}

// HasPoweredGenerator returns true if this cell has a powered generator
func HasPoweredGenerator(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Generator != nil && data.Generator.IsPowered()
}

// HasDoor returns true if this cell contains a door
func HasDoor(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Door != nil
}

// HasLockedDoor returns true if this cell has a locked door
func HasLockedDoor(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Door != nil && data.Door.Locked
}

// HasUnlockedDoor returns true if this cell has an unlocked door
func HasUnlockedDoor(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Door != nil && !data.Door.Locked
}

// HasPuzzle returns true if this cell contains a puzzle terminal
func HasPuzzle(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Puzzle != nil
}

// HasUnsolvedPuzzle returns true if this cell has an unsolved puzzle
func HasUnsolvedPuzzle(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Puzzle != nil && !data.Puzzle.IsSolved()
}

// HasMaintenanceTerminal returns true if this cell contains a maintenance terminal
func HasMaintenanceTerminal(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.MaintenanceTerm != nil
}

// AreLightsOn returns true if lights are on in this cell
func AreLightsOn(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.LightsOn
}

// SetLightsOn sets the lighting state for this cell
func SetLightsOn(cell *world.Cell, on bool) {
	data := GetGameData(cell)
	data.LightsOn = on
	if on {
		data.Lighted = true // Once lit, stays lit
		// If lights are on, mark as discovered and visited
		cell.Discovered = true
		cell.Visited = true
	}
}

// IsLighted returns true if this cell has been lit (stays explored)
func IsLighted(cell *world.Cell) bool {
	data := GetGameData(cell)
	return data.Lighted
}
