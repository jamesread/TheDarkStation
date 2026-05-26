// Package gameplay provides core game logic for player movement and interactions.
package gameplay

import (
	"time"

	"darkstation/pkg/game/levelrand"

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
	generateLevel(g, startLevel, seed)

	g.ClearMessages()

	return g
}

// generateLevel rebuilds grid and setup from a seed (deterministic when seed is fixed).
func generateLevel(g *state.Game, level int, seed int64) {
	levelrand.Seed(seed)
	g.LevelSeed = seed
	g.Level = level
	g.CurrentDeckID = level - 1
	g.Grid = GenerateGrid(level)
	SetupLevel(g)
}

// RegenerateFromSeed rebuilds the current level from seed (for reset / debug reproduction).
func RegenerateFromSeed(g *state.Game, seed int64) {
	if g == nil {
		return
	}
	level := g.Level
	if level < 1 {
		level = 1
	}
	generateLevel(g, level, seed)
}

// LoadLevelFromSeed clears deck progress and rebuilds the current level from seed.
func LoadLevelFromSeed(g *state.Game, seed int64) {
	if g == nil {
		return
	}
	clearLevelProgress(g)
	generateLevel(g, g.Level, seed)
	g.SaveCurrentDeckState()
	UpdateLightingExploration(g)
	setup.EnsureSolvabilityDoorPower(g)
	setup.ApplyGridConductivePower(g)
	g.ClearMessages()
}

func clearLevelProgress(g *state.Game) {
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
	g.RoomPowerOnline = make(map[string]bool)
	g.ManualEgressReleased = make(map[string]bool)
	g.PowerPropPending = nil
	g.LongUse = nil

	g.MovementCount = 0
	g.InteractionsCount = 0
	g.LastInteractedRow = -1
	g.LastInteractedCol = -1
	g.InteractionPlayerRow = -1
	g.InteractionPlayerCol = -1

	g.ExitAnimating = false
	g.ExitAnimStartTime = 0
	g.GameComplete = false
}

// clearCrossDeckPowerState resets player-carried power state when entering a different deck.
// Each deck keeps its own grid-attached generators and saved room power maps.
func clearCrossDeckPowerState(g *state.Game) {
	if g == nil {
		return
	}
	g.Batteries = 0
	g.OwnedItems = mapset.New[*world.Item]()
	g.Generators = make([]*entities.Generator, 0)
	g.PowerSupply = 0
	g.PowerConsumption = 0
	g.PowerOverloadWarned = false
	g.PowerPropPending = nil
	g.LongUse = nil
	ClearGeneratorPowerGridOverlay(g)
}

// refreshDeckPower rebuilds generator registration and recalculates supply/consumption for the active deck.
func refreshDeckPower(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	g.RebuildGeneratorsFromGrid()
	g.UpdatePowerSupply()
	setup.SchedulePowerPropagation(g, setup.PowerNowMs())
	setup.ApplyGridConductivePower(g)
	g.PowerConsumption = g.CalculatePowerConsumption()
}

// SetupLevel configures the current level with items and keys
func SetupLevel(g *state.Game) {
	g.ResetObservationCueAnnounced()
	g.ResetLinkageTokensSeen()
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

	// Relocate init-trapped keycards/generators and clear furniture that blocks interact access.
	setup.EnsureInitProgressReachability(g)
	setup.EnsureInteractableNavAccess(g)

	// Ensure no control-dependency deadlock: gatekeeper rooms with unpowered doors
	// must be powerable from an adjacent room that has a terminal; otherwise power them initially.
	setup.EnsureSolvabilityDoorPower(g)

	setup.InitMaintenanceTerminalPower(g)
	setup.EnsureGeneratorRoomBootstrap(g)
	setup.PlaceAdditionalGenerators(g, avoid)
	setup.PlaceBatteries(g, avoid)
	setup.EnsureBatteryReachability(g)

	// Remove or avoid blockers that would make the exit permanently unreachable (R7).
	setup.EnsureExitReachability(g)

	// Diegetic corridor signage tied to functional deck layer (Story 5.1).
	setup.ApplyEnvironmentalSignage(g)
	setup.ApplyObservationLedPuzzleCues(g)
	setup.ApplyMultiHopLinkage(g)

	// Corridor power relays for power grid (power-routing Phase 3).
	setup.ApplyPowerRelays(g)

	// Move player to start cell (setup package sets current cell to center)
	MoveCell(g, g.Grid.StartCell())
}

// ResetLevel resets the current deck using the same seed; updates per-deck store (Phase 3.4).
func ResetLevel(g *state.Game) {
	currentLevel := g.Level

	clearLevelProgress(g)

	var seed int64
	if g.LevelSeed != 0 {
		seed = g.LevelSeed
	} else {
		seed = int64(currentLevel)
	}

	generateLevel(g, currentLevel, seed)

	// Update store so revisit uses reset layout (Phase 3.4)
	g.SaveCurrentDeckState()

	UpdateLightingExploration(g)
	setup.EnsureSolvabilityDoorPower(g)
	setup.ApplyGridConductivePower(g)

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

	clearCrossDeckPowerState(g)

	// Load stored state or generate on first entry (Phase 3.4)
	if ds := g.DeckStates[nextID]; ds != nil && ds.Grid != nil {
		g.LoadDeckState(nextID)
		refreshDeckPower(g)
		UpdateLightingExploration(g)
	} else {
		g.CurrentDeckID = nextID
		g.Level = g.CurrentDeckID + 1
		seed := time.Now().UnixNano()
		generateLevel(g, g.Level, seed)
		refreshDeckPower(g)
		g.SaveCurrentDeckState() // Store for potential revisit
		UpdateLightingExploration(g)
	}

	g.ClearMessages()
	logMessage(g, "You moved to deck %d!", g.Level)
}
