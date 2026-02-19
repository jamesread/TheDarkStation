// Package gameplay provides core game logic for player movement and interactions.
package gameplay

import (
	"math/rand"
	"time"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/levelgen"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
)

// GenerateGrid creates a new grid using the default generator
func GenerateGrid(level int) *world.Grid {
	return generator.DefaultGenerator.Generate(level)
}

// BuildGame creates a new game instance with optional starting level (Phase 3.2, 3.4).
// Current deck is set from startLevel; deck is generated on first entry (no load).
func BuildGame(startLevel int) *state.Game {
	g := state.NewGame()

	// Current deck by ID; Level = 1-based display (Phase 3.2)
	if startLevel < 1 {
		startLevel = 1
	}
	if startLevel > deck.TotalDecks {
		startLevel = deck.TotalDecks
	}
	g.CurrentDeckID = startLevel - 1
	g.Level = g.CurrentDeckID + 1

	// Generate current deck on first entry (no stored state yet)
	seed := time.Now().UnixNano()
	g.LevelSeed = seed
	rand.Seed(seed)
	g.Grid = GenerateGrid(g.Level)
	SetupLevel(g)

	g.ClearMessages()

	return g
}

// SetupLevel configures the current level with items and keys
func SetupLevel(g *state.Game) {
	config := setup.SetupLevel(g)
	avoid := &config.Avoid
	lockedDoorCells := &config.LockedDoorCells
	minimalSystems := deck.IsFinalDeck(g.Level) // Final deck: minimal rooms/systems (GDD §10.2)

	// Place environmental hazards (level 2+), skip on final deck
	if g.Level >= 2 && !minimalSystems {
		levelgen.PlaceHazards(g, avoid, lockedDoorCells)
	}

	// Place furniture in rooms (1-2 per room); none on final deck
	if !minimalSystems {
		levelgen.PlaceFurniture(g, avoid)
	}

	// Place puzzle terminals (level 2+), skip on final deck
	if g.Level >= 2 && !minimalSystems {
		levelgen.PlacePuzzles(g, avoid)
	}

	// Maintenance terminals (one per room); keep on final deck for barely functional power
	levelgen.PlaceMaintenanceTerminals(g, avoid)

	// Ensure no control-dependency deadlock: gatekeeper rooms with unpowered doors
	// must be powerable from an adjacent room that has a terminal; otherwise power them initially.
	setup.EnsureSolvabilityDoorPower(g)

	// Only the start room's maintenance terminal(s) start powered; others can be restored from nearby.
	setup.InitMaintenanceTerminalPower(g)

	// Move player to start cell (setup package sets current cell to center)
	MoveCell(g, g.Grid.StartCell())
}

// ResetLevel resets the current deck using the same seed; updates per-deck store (Phase 3.4).
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
	g.RoomLightsPowered = make(map[string]bool)

	g.MovementCount = 0
	g.InteractionsCount = 0
	g.LastInteractedRow = -1
	g.LastInteractedCol = -1
	g.InteractionPlayerRow = -1
	g.InteractionPlayerCol = -1

	g.ExitAnimating = false
	g.ExitAnimStartTime = 0
	g.GameComplete = false

	var seed int64
	if g.LevelSeed != 0 {
		seed = g.LevelSeed
	} else {
		seed = int64(currentLevel)
	}

	rand.Seed(seed)
	g.Grid = GenerateGrid(currentLevel)
	SetupLevel(g)
	g.LevelSeed = seed

	// Update store so revisit uses reset layout (Phase 3.4)
	g.SaveCurrentDeckState()

	UpdateLightingExploration(g)

	g.ClearMessages()
	logMessage(g, "Level reset!")
}

// TriggerGameComplete is called when the player reaches the exit on the final deck.
// The lift has no destination; the game is complete (ending per GDD §11).
func TriggerGameComplete(g *state.Game) {
	g.GameComplete = true
	logMessage(g, "No further work requests detected.")
}

// AdvanceLevel moves to the next deck via the graph: saves current deck state,
// loads or generates the next deck, sets CurrentDeckID/Level (Phase 3.3, 3.4).
// Does nothing if already at or past the final deck.
func AdvanceLevel(g *state.Game) {
	nextID, ok := deck.NextDeckID(g.CurrentDeckID)
	if !ok {
		return
	}

	// Save current deck so we can revisit (Phase 3.4)
	g.SaveCurrentDeckState()

	// Load stored state or generate on first entry (Phase 3.4)
	if ds := g.DeckStates[nextID]; ds != nil && ds.Grid != nil {
		g.LoadDeckState(nextID)
		UpdateLightingExploration(g)
	} else {
		g.CurrentDeckID = nextID
		g.Level = g.CurrentDeckID + 1
		seed := time.Now().UnixNano()
		g.LevelSeed = seed
		rand.Seed(seed)
		g.Grid = GenerateGrid(g.Level)
		SetupLevel(g)
		g.SaveCurrentDeckState() // Store for potential revisit
	}

	g.ClearMessages()
	logMessage(g, "You moved to deck %d!", g.Level)
}
