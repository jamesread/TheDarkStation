// Package setup provides level setup functionality for The Dark Station.
package setup

import (
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
)

// SetupConfig holds configuration for level setup
type SetupConfig struct {
	Avoid           mapset.Set[*world.Cell]
	LockedDoorCells mapset.Set[*world.Cell]
}

// SetupLevel configures a level with all entities, items, and objectives.
// Returns the avoid set and locked door cells for use by other placement functions.
func SetupLevel(g *state.Game) *SetupConfig {
	// Cells to avoid placing items on
	avoid := mapset.New[*world.Cell]()
	avoid.Put(g.Grid.StartCell())
	avoid.Put(g.Grid.ExitCell())

	// Track which cells have locked doors for reachability calculations
	lockedDoorCells := mapset.New[*world.Cell]()

	// Place locked rooms with doors
	PlaceLockedRooms(g, &avoid, &lockedDoorCells)

	// Ensure every room has at least one door (unlocked for rooms without locked doors)
	roomEntries := FindRoomEntryPoints(g.Grid)
	EnsureEveryRoomHasDoor(g, &avoid, &lockedDoorCells, roomEntries)

	// Initialize room power: unpowered by default, start room doors powered
	InitRoomPower(g)

	// Place generators
	PlaceGenerators(g, &avoid)

	// Place batteries
	PlaceBatteries(g, &avoid)

	// Place CCTV terminals (level 2+)
	PlaceCCTVTerminals(g, &avoid, roomEntries)

	// Set player position to center, will be moved to start in main
	g.CurrentCell = g.Grid.GetCenterCell()

	return &SetupConfig{
		Avoid:           avoid,
		LockedDoorCells: lockedDoorCells,
	}
}

// PlaceLockedRooms places locked rooms with doors (exported for use in main)
func PlaceLockedRooms(g *state.Game, avoid *mapset.Set[*world.Cell], lockedDoorCells *mapset.Set[*world.Cell]) {
	placeLockedRooms(g, avoid, lockedDoorCells)
}

// PlaceGenerators places generators (exported for use in main)
func PlaceGenerators(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	placeGenerators(g, avoid)
}

// PlaceBatteries places batteries (exported for use in main)
func PlaceBatteries(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	placeBatteries(g, avoid)
}

// PlaceCCTVTerminals places CCTV terminals (exported for use in main)
func PlaceCCTVTerminals(g *state.Game, avoid *mapset.Set[*world.Cell], roomEntries map[string]*RoomEntryPoints) {
	placeCCTVTerminals(g, avoid, roomEntries)
}

// FindRoomEntryPoints finds room entry points (exported for use in main)
func FindRoomEntryPoints(grid *world.Grid) map[string]*RoomEntryPoints {
	return findRoomEntryPoints(grid)
}

// moveCell is a helper that will be imported from main or moved here
func moveCell(g *state.Game, target *world.Cell) {
	// This function needs to be accessible - it's currently in main.go
	// For now, we'll need to import it or move it here
	// TODO: Move moveCell to a shared location or import from main
}
