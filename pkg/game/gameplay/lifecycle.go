// Package gameplay provides core game logic for player movement and interactions.
package gameplay

import (
	"math/rand"
	"time"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/levelgen"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// GenerateGrid creates a new grid using the default generator
func GenerateGrid(level int) *world.Grid {
	return generator.DefaultGenerator.Generate(level)
}

// BuildGame creates a new game instance with optional starting level
func BuildGame(startLevel int) *state.Game {
	g := state.NewGame()

	// Set starting level if specified (for developer testing)
	if startLevel > 1 {
		g.Level = startLevel
	}

	// Store seed before generation for reset functionality
	// Use level number as deterministic seed, or time-based for variety
	seed := time.Now().UnixNano()
	g.LevelSeed = seed
	rand.Seed(seed)

	g.Grid = GenerateGrid(g.Level)
	SetupLevel(g)

	// Clear the initial "entered room" message
	g.ClearMessages()
	logMessage(g, "Welcome to the Abandoned Station!")
	logMessage(g, "You are on deck %d.", g.Level)
	ShowLevelObjectives(g)

	return g
}

// SetupLevel configures the current level with items and keys
func SetupLevel(g *state.Game) {
	// Use the setup package to configure the level
	config := setup.SetupLevel(g)
	avoid := &config.Avoid
	lockedDoorCells := &config.LockedDoorCells

	// Place environmental hazards (level 2+)
	if g.Level >= 2 {
		levelgen.PlaceHazards(g, avoid, lockedDoorCells)
	}

	// Place furniture in rooms (1-2 pieces per unique room type)
	levelgen.PlaceFurniture(g, avoid)

	// Place puzzle terminals (level 2+)
	if g.Level >= 2 {
		levelgen.PlacePuzzles(g, avoid)
	}

	// Place maintenance terminals in every room (one per room, against walls)
	levelgen.PlaceMaintenanceTerminals(g, avoid)

	// Ensure no control-dependency deadlock: gatekeeper rooms with unpowered doors
	// must be powerable from an adjacent room that has a terminal; otherwise power them initially.
	setup.EnsureSolvabilityDoorPower(g)

	// Only the start room's maintenance terminal(s) start powered; others can be restored from nearby.
	setup.InitMaintenanceTerminalPower(g)

	// Move player to start cell (setup package sets current cell to center)
	MoveCell(g, g.Grid.StartCell())
}

// ResetLevel resets the current level using the same seed/map
func ResetLevel(g *state.Game) {
	currentLevel := g.Level

	// Clear inventory and level-specific state
	g.OwnedItems = mapset.New[*world.Item]()
	g.Batteries = 0
	g.HasMap = false
	g.FoundCodes = make(map[string]bool)
	g.Generators = make([]*entities.Generator, 0)
	g.Hints = nil
	g.PowerSupply = 0
	g.PowerConsumption = 0
	g.PowerOverloadWarned = false
	g.RoomDoorsPowered = make(map[string]bool)
	g.RoomCCTVPowered = make(map[string]bool)

	// Reset interaction/movement counters
	g.MovementCount = 0
	g.InteractionsCount = 0
	g.LastInteractedRow = -1
	g.LastInteractedCol = -1
	g.InteractionPlayerRow = -1
	g.InteractionPlayerCol = -1

	// Reset exit animation state
	g.ExitAnimating = false
	g.ExitAnimStartTime = 0

	// Regenerate grid with the same seed (or use level as seed if not set)
	var seed int64
	if g.LevelSeed != 0 {
		seed = g.LevelSeed
	} else {
		// Fallback: use level number as seed (deterministic)
		seed = int64(currentLevel)
	}

	// Set seed before generating to ensure same map layout
	rand.Seed(seed)
	g.Grid = GenerateGrid(currentLevel)

	// Setup level again (will place entities in same positions due to same seed)
	SetupLevel(g)

	// Store the seed for future resets
	g.LevelSeed = seed

	// Update power and lighting after setup
	UpdateLightingExploration(g)

	// Clear messages and show reset message
	g.ClearMessages()
	logMessage(g, "Level reset!")
	logMessage(g, "You are on deck %d.", g.Level)
	ShowLevelObjectives(g)
}

// AdvanceLevel generates a new map and advances to the next level
func AdvanceLevel(g *state.Game) {
	g.AdvanceLevel()

	// Store seed for new level (for reset functionality)
	seed := time.Now().UnixNano()
	g.LevelSeed = seed
	rand.Seed(seed)

	g.Grid = GenerateGrid(g.Level)
	SetupLevel(g)

	// Clear movement messages and show level info
	g.ClearMessages()
	logMessage(g, "You moved to deck %d!", g.Level)
	ShowLevelObjectives(g)
}

// ShowLevelObjectives displays the objectives for the current level
func ShowLevelObjectives(g *state.Game) {
	// Count doors
	numDoors := countDoors(g)
	// Note: Keycard message removed - players can discover locked doors naturally
	if len(g.Generators) > 0 {
		logMessage(g, "Power up ACTION{%d} generator(s) with batteries.", len(g.Generators))
	}
	// Count hazards
	numHazards := countHazards(g)
	if numHazards > 0 {
		logMessage(g, "Clear ACTION{%d} environmental hazard(s).", numHazards)
	}
	if numDoors == 0 && len(g.Generators) == 0 && numHazards == 0 {
		logMessage(g, "Find the EXIT{lift} to the next deck.")
	}
}

// countDoors counts the number of locked doors on the map
func countDoors(g *state.Game) int {
	count := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if gameworld.HasLockedDoor(cell) {
			count++
		}
	})
	return count
}

// countHazards counts the number of active hazards on the map
func countHazards(g *state.Game) int {
	count := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if gameworld.HasBlockingHazard(cell) {
			count++
		}
	})
	return count
}
