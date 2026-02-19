package state

import (
	"time"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/entities"
	gameworld "darkstation/pkg/game/world"
)

// NavStyle represents the navigation key style
type NavStyle int

// Navigation styles
const (
	NavStyleNSEW NavStyle = iota
	NavStyleVim
)

// DeckState holds generated state for one deck (GDD §4.2). Used for generation on first entry and revisit.
type DeckState struct {
	Grid               *world.Grid
	LevelSeed          int64
	RoomDoorsPowered   map[string]bool
	RoomCCTVPowered    map[string]bool
	RoomLightsPowered  map[string]bool
	Generators         []*entities.Generator
}

// Game represents the game state for Abandoned Station
type Game struct {
	CurrentCell *world.Cell

	Hints []string

	Grid *world.Grid

	HasMap bool

	OwnedItems world.ItemSet

	Messages []MessageEntry

	NavStyle NavStyle

	Level int // Current deck level (1-based display): Level = CurrentDeckID + 1

	CurrentDeckID int             // 0-based deck index (source of truth for which deck we're in)
	DeckStates    map[int]*DeckState // Per-deck generated state; key = deck ID (0-based)

	Batteries            int                   // Number of batteries in inventory
	Generators           []*entities.Generator // All generators on this level
	FoundCodes           map[string]bool       // Puzzle codes found by the player (code -> found)
	ExitAnimating        bool                  // True when exit animation is playing
	ExitAnimStartTime    int64                 // Timestamp when exit animation started (milliseconds)
	LastInteractedRow    int                   // Row of last cell interacted with (for cycling)
	LastInteractedCol    int                   // Col of last cell interacted with (for cycling)
	InteractionPlayerRow int                   // Player row when interaction order was established
	InteractionPlayerCol int                   // Player col when interaction order was established
	InteractionsCount    int                   // Number of objects the player has interacted with (for hint system)
	MovementCount        int                   // Number of times the player has moved (for movement hint)
	LevelSeed            int64                 // Random seed used for current level generation (for reset)
	PowerSupply          int                   // Total available power from generators
	PowerConsumption     int                   // Total power being consumed by active devices
	PowerOverloadWarned  bool                  // Whether we've warned about power overload this cycle
	QuitToTitle          bool                  // Set to true to quit to main menu
	GameComplete         bool                  // True when player reached final deck and lift has no destination (completion)

	// Room power: doors and CCTV/hazard controls are unpowered by default.
	// Start room's doors are powered so the player can leave.
	RoomDoorsPowered  map[string]bool // room name -> doors powered
	RoomCCTVPowered   map[string]bool // room name -> CCTV terminals and hazard controls powered
	RoomLightsPowered map[string]bool // room name -> lights enabled (0w; toggled at maintenance terminal)

	// MaintenanceMenuRoom is set while the maintenance menu is open; the room whose
	// maintenance view is displayed. Used to highlight that room's wall cells on the map.
	MaintenanceMenuRoom string
}

// MessageEntry represents a message with a timestamp
type MessageEntry struct {
	Text      string
	Timestamp int64 // Unix timestamp in milliseconds when message was added
}

// NewGame creates a new game instance
func NewGame() *Game {
	return &Game{
		OwnedItems:            mapset.New[*world.Item](),
		HasMap:                false,
		Messages:              make([]MessageEntry, 0),
		Level:                 1,
		CurrentDeckID:         0,
		DeckStates:            make(map[int]*DeckState),
		Batteries:             0,
		Generators:             make([]*entities.Generator, 0),
		FoundCodes:            make(map[string]bool),
		LastInteractedRow:     -1,
		LastInteractedCol:     -1,
		InteractionPlayerRow:  -1,
		InteractionPlayerCol:  -1,
		PowerSupply:           0,
		PowerConsumption:      0,
		PowerOverloadWarned:    false,
		RoomDoorsPowered:      make(map[string]bool),
		RoomCCTVPowered:       make(map[string]bool),
		RoomLightsPowered:     make(map[string]bool),
	}
}

// AddBatteries adds batteries to the player's inventory
func (g *Game) AddBatteries(count int) {
	g.Batteries += count
}

// UseBatteries removes batteries from inventory, returns actual amount used
func (g *Game) UseBatteries(count int) int {
	if count > g.Batteries {
		count = g.Batteries
	}
	g.Batteries -= count
	return count
}

// AddGenerator registers a generator for this level
func (g *Game) AddGenerator(gen *entities.Generator) {
	g.Generators = append(g.Generators, gen)
}

// AllGeneratorsPowered returns true if all generators are powered
func (g *Game) AllGeneratorsPowered() bool {
	if len(g.Generators) == 0 {
		return true
	}
	for _, gen := range g.Generators {
		if !gen.IsPowered() {
			return false
		}
	}
	return true
}

// AllHazardsCleared returns true if all hazards are cleared (no blocking hazards remain)
func (g *Game) AllHazardsCleared() bool {
	if g == nil || g.Grid == nil {
		return true
	}
	hasBlockingHazard := false
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && gameworld.HasBlockingHazard(cell) {
			hasBlockingHazard = true
		}
	})
	return !hasBlockingHazard
}

// UnpoweredGeneratorCount returns the number of unpowered generators
func (g *Game) UnpoweredGeneratorCount() int {
	count := 0
	for _, gen := range g.Generators {
		if !gen.IsPowered() {
			count++
		}
	}
	return count
}

// AddMessage adds a message to the game's message log
func (g *Game) AddMessage(msg string) {
	const maxMessages = 5
	now := time.Now().UnixMilli()

	// Remove messages older than 10 seconds before adding new one
	g.RemoveOldMessages()

	g.Messages = append(g.Messages, MessageEntry{
		Text:      msg,
		Timestamp: now,
	})

	// Keep only the last maxMessages
	if len(g.Messages) > maxMessages {
		g.Messages = g.Messages[len(g.Messages)-maxMessages:]
	}
}

// RemoveOldMessages removes messages older than 10 seconds from the buffer
func (g *Game) RemoveOldMessages() {
	const messageLifetime = 10000 // 10 seconds in milliseconds
	now := time.Now().UnixMilli()

	filtered := make([]MessageEntry, 0, len(g.Messages))
	for _, msg := range g.Messages {
		age := now - msg.Timestamp
		if age < messageLifetime {
			filtered = append(filtered, msg)
		}
	}
	g.Messages = filtered
}

// ClearMessages clears all messages
func (g *Game) ClearMessages() {
	g.Messages = make([]MessageEntry, 0)
}

// AddHint adds a hint to the game
func (g *Game) AddHint(hint string) {
	g.Hints = append(g.Hints, hint)
}

// PickUpItem adds an item to the player's inventory
func (g *Game) PickUpItem(item *world.Item) {
	g.OwnedItems.Put(item)
}

// HasItem checks if the player has a specific item
func (g *Game) HasItem(item *world.Item) bool {
	return g.OwnedItems.Has(item)
}

// AdvanceLevel increments the level counter and resets level-specific state.
// Does not increment past the final deck (deck.TotalDecks).
// Used when advancing without per-deck store (e.g. legacy path). Prefer gameplay.AdvanceLevel for graph-based advance.
func (g *Game) AdvanceLevel() {
	if g.Level < deck.TotalDecks {
		g.Level++
	}
	if g.CurrentDeckID < deck.FinalDeckIndex {
		g.CurrentDeckID++
	}
	g.OwnedItems = mapset.New[*world.Item]()
	g.HasMap = false
	g.Hints = nil
	g.Batteries = 0
	g.Generators = make([]*entities.Generator, 0)
	g.FoundCodes = make(map[string]bool)
	g.PowerSupply = 0
	g.PowerConsumption = 0
	g.PowerOverloadWarned = false
	g.RoomDoorsPowered = make(map[string]bool)
	g.RoomCCTVPowered = make(map[string]bool)
	g.RoomLightsPowered = make(map[string]bool)
}

// copyPowerMaps returns copies of the given power maps (for per-deck state).
func copyPowerMaps(doors, cctv, lights map[string]bool) (doorsCopy, cctvCopy, lightsCopy map[string]bool) {
	doorsCopy = make(map[string]bool, len(doors))
	for k, v := range doors {
		doorsCopy[k] = v
	}
	cctvCopy = make(map[string]bool, len(cctv))
	for k, v := range cctv {
		cctvCopy[k] = v
	}
	if lights == nil {
		lights = make(map[string]bool)
	}
	lightsCopy = make(map[string]bool, len(lights))
	for k, v := range lights {
		lightsCopy[k] = v
	}
	return doorsCopy, cctvCopy, lightsCopy
}

// SaveCurrentDeckState stores the current deck's grid and power state into DeckStates (Phase 3.4).
// Call before switching to another deck so the current deck can be restored on revisit.
func (g *Game) SaveCurrentDeckState() {
	if g.Grid == nil {
		return
	}
	doorsCopy, cctvCopy, lightsCopy := copyPowerMaps(g.RoomDoorsPowered, g.RoomCCTVPowered, g.RoomLightsPowered)
	genCopy := make([]*entities.Generator, len(g.Generators))
	copy(genCopy, g.Generators)
	g.DeckStates[g.CurrentDeckID] = &DeckState{
		Grid:              g.Grid,
		LevelSeed:         g.LevelSeed,
		RoomDoorsPowered:   doorsCopy,
		RoomCCTVPowered:    cctvCopy,
		RoomLightsPowered:  lightsCopy,
		Generators:        genCopy,
	}
}

// LoadDeckState restores deck state for the given deck ID into g and sets CurrentDeckID/Level (Phase 3.4).
// Clears per-deck UI state (HasMap, Hints, FoundCodes, power recalc). Caller must set CurrentCell and update lighting.
func (g *Game) LoadDeckState(deckID int) {
	ds, ok := g.DeckStates[deckID]
	if !ok || ds == nil || ds.Grid == nil {
		return
	}
	g.CurrentDeckID = deckID
	g.Level = deckID + 1
	g.Grid = ds.Grid
	g.LevelSeed = ds.LevelSeed
	g.RoomDoorsPowered, g.RoomCCTVPowered, g.RoomLightsPowered = copyPowerMaps(ds.RoomDoorsPowered, ds.RoomCCTVPowered, ds.RoomLightsPowered)
	g.Generators = make([]*entities.Generator, len(ds.Generators))
	copy(g.Generators, ds.Generators)
	g.HasMap = false
	g.Hints = nil
	g.FoundCodes = make(map[string]bool)
	g.PowerSupply = 0
	g.PowerConsumption = 0
	g.PowerOverloadWarned = false
	g.CurrentCell = g.Grid.StartCell()
}

// GetAvailablePower returns the available power (supply - consumption)
func (g *Game) GetAvailablePower() int {
	return g.PowerSupply - g.PowerConsumption
}

// UpdatePowerSupply recalculates power supply from powered generators.
// Uses per-deck decay: generator output is 100 W × deck's output multiplier (Phase 4.3).
func (g *Game) UpdatePowerSupply() {
	params := deck.DecayParamsForDeck(g.CurrentDeckID)
	wattsPerGenerator := 100
	totalPower := 0
	for _, gen := range g.Generators {
		if gen.IsPowered() {
			totalPower += int(float64(wattsPerGenerator) * params.GeneratorOutputMultiplier)
		}
	}
	g.PowerSupply = totalPower
}

// CalculatePowerConsumption returns total power consumption from all active devices
// (doors, CCTV, solved puzzles), scaled by per-deck cost multiplier (Phase 4.3).
func (g *Game) CalculatePowerConsumption() int {
	if g.Grid == nil {
		return 0
	}
	rawConsumption := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Terminal != nil && g.RoomCCTVPowered[cell.Name] {
			rawConsumption += 10
		}
		if data.Door != nil && g.RoomDoorsPowered[data.Door.RoomName] {
			rawConsumption += 10
		}
		if data.Puzzle != nil && data.Puzzle.IsSolved() {
			rawConsumption += 3
		}
	})
	params := deck.DecayParamsForDeck(g.CurrentDeckID)
	return int(float64(rawConsumption) * params.PowerCostMultiplier)
}

// ShortOutIfOverload runs after a room power toggle to ON: if consumption exceeds supply,
// "shorts out" by unpowering other rooms' doors and CCTV (never protectedRoomName)
// until consumption <= supply. Caller must have already applied the toggle.
// Updates g.PowerConsumption. Returns true if any systems were unpowered.
func (g *Game) ShortOutIfOverload(protectedRoomName string) bool {
	g.UpdatePowerSupply()
	consumption := g.CalculatePowerConsumption()
	g.PowerConsumption = consumption
	if consumption <= g.PowerSupply {
		return false
	}
	type consumer struct{ room, kind string }
	var list []consumer
	for roomName := range g.RoomDoorsPowered {
		if roomName == protectedRoomName {
			continue
		}
		if g.RoomDoorsPowered[roomName] {
			list = append(list, consumer{roomName, "doors"})
		}
	}
	for roomName := range g.RoomCCTVPowered {
		if roomName == protectedRoomName {
			continue
		}
		if g.RoomCCTVPowered[roomName] {
			list = append(list, consumer{roomName, "cctv"})
		}
	}
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].room < list[i].room || (list[j].room == list[i].room && list[j].kind == "doors" && list[i].kind == "cctv") {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	shortOut := false
	for _, c := range list {
		if consumption <= g.PowerSupply {
			break
		}
		if c.kind == "doors" && g.RoomDoorsPowered[c.room] {
			g.RoomDoorsPowered[c.room] = false
			shortOut = true
		} else if c.kind == "cctv" && g.RoomCCTVPowered[c.room] {
			g.RoomCCTVPowered[c.room] = false
			shortOut = true
		}
		consumption = g.CalculatePowerConsumption()
		g.PowerConsumption = consumption
	}
	return shortOut
}

// AddFoundCode records that the player has found a puzzle code
func (g *Game) AddFoundCode(code string) {
	g.FoundCodes[code] = true
}

// HasFoundCode checks if the player has found a specific code
func (g *Game) HasFoundCode(code string) bool {
	return g.FoundCodes[code]
}
