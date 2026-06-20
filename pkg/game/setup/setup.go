// Package setup provides level setup functionality for The Dark Station.
package setup

import (
	"sort"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/levelrand"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
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
	if g.Grid != nil {
		g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
			if cell != nil && generator.IsEmptyOverlayRoom(cell.Name) {
				avoid.Put(cell)
			}
		})
	}
	if entry := PlayerEntryCell(g); entry != nil {
		avoid.Put(entry)
	}

	// Track which cells have locked doors for reachability calculations
	lockedDoorCells := mapset.New[*world.Cell]()

	// Place locked rooms with doors
	PlaceLockedRooms(g, &avoid, &lockedDoorCells)

	// Ensure every room has at least one door (unlocked for rooms without locked doors)
	roomEntries := FindRoomEntryPoints(g.Grid)
	EnsureEveryRoomHasDoor(g, &avoid, &lockedDoorCells, roomEntries)

	// Initialize room power: unpowered by default; generator bootstrap arms generator rooms
	InitRoomPower(g)

	// Corridor door cells added above can strand keycards placed before them; relocate before
	// blocking entities (generators, batteries) are placed.
	EnsureKeycardReachability(g)

	// Place generators (spawn only; additional generators and batteries after bootstrap in lifecycle).
	PlaceGenerators(g, &avoid)

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
	EnsurePoweredSpawnGenerator(g, avoid)
}

// EnsurePoweredSpawnGenerator guarantees a powered spawn generator exists after placement.
func EnsurePoweredSpawnGenerator(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	if g == nil || g.Grid == nil {
		return
	}
	for _, gen := range g.Generators {
		if gen != nil && gen.IsPowered() {
			return
		}
	}
	var cell *world.Cell
	if start := g.Grid.StartCell(); start != nil && start.Name != "" &&
		!generator.IsEmptyOverlayRoom(start.Name) {
		cell = findGeneratorCellInRoom(g, start.Name, start, avoid, true)
	}
	if cell == nil {
		cell = findGeneratorCellAnywhere(g, avoid, true)
	}
	if cell == nil {
		return
	}
	batteriesRequired := 1
	if g.Level >= 3 {
		batteriesRequired = 1 + levelrand.Intn(3)
	}
	gen := entities.NewGenerator("Generator #1", batteriesRequired)
	gen.InsertBatteriesAndStart(batteriesRequired)
	gameworld.GetGameData(cell).Generator = gen
	g.AddGenerator(gen)
	if avoid != nil {
		avoid.Put(cell)
	}
	g.UpdatePowerSupply()
	SchedulePowerPropagation(g, PowerNowMs())
}

func findGeneratorCellAnywhere(g *state.Game, avoid *mapset.Set[*world.Cell], relaxed bool) *world.Cell {
	if g == nil || g.Grid == nil {
		return nil
	}
	rooms := map[string]struct{}{}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.Room && cell.Name != "" && cell.Name != "Corridor" &&
			!generator.IsPlacementExcludedRoom(cell.Name) {
			rooms[cell.Name] = struct{}{}
		}
	})
	names := make([]string, 0, len(rooms))
	for name := range rooms {
		names = append(names, name)
	}
	sort.Strings(names)
	routingOrigin := PlayerEntryCell(g)
	for _, name := range names {
		if cell := findGeneratorCellInRoom(g, name, routingOrigin, avoid, relaxed); cell != nil {
			return cell
		}
	}
	return nil
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
