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
	Grid                 *world.Grid
	LevelSeed            int64
	RoomDoorsPowered     map[string]bool
	RoomCCTVPowered      map[string]bool
	RoomLightsPowered    map[string]bool
	RoomPowerOnline      map[string]bool
	PowerPropPending     []PowerPropEntry
	Generators           []*entities.Generator
	ManualEgressReleased map[string]bool // room name -> manual door release active (no grid power)
	OwnedItems           world.ItemSet   // keycards and other deck-local pickup inventory
}

// Game represents the game state for Abandoned Station
type Game struct {
	CurrentCell *world.Cell

	PlayerFacing PlayerFacing // Direction the player is looking (map triangle icon)

	Hints []string

	Grid *world.Grid

	HasMap bool

	OwnedItems world.ItemSet

	Messages []MessageEntry

	NavStyle NavStyle

	Level int // Current deck level (1-based display): Level = CurrentDeckID + 1

	CurrentDeckID int                // 0-based deck index (source of truth for which deck we're in)
	DeckStates    map[int]*DeckState // Per-deck generated state; key = deck ID (0-based)

	Batteries                int                   // Number of batteries in inventory
	Generators               []*entities.Generator // All generators on this level
	FoundCodes               map[string]bool       // Puzzle codes found by the player (code -> found)
	ExitAnimating            bool                  // True when exit animation is playing
	ExitAnimStartTime        int64                 // Timestamp when exit animation started (milliseconds)
	LastInteractedRow        int                   // Row of last cell interacted with (for cycling)
	LastInteractedCol        int                   // Col of last cell interacted with (for cycling)
	InteractionPlayerRow     int                   // Player row when interaction order was established
	InteractionPlayerCol     int                   // Player col when interaction order was established
	InteractionsCount        int                   // Number of objects the player has interacted with (for hint system)
	MovementCount            int                   // Number of times the player has moved (for movement hint)
	LevelSeed                int64                 // Random seed used for current level generation (for reset)
	PowerSupply              int                   // Total available power from generators
	PowerConsumption         int                   // Total power being consumed by active devices
	PowerOverloadWarned      bool                  // Whether we've warned about power overload this cycle
	QuitToTitle              bool                  // Set to true to quit to main menu
	GameComplete             bool                  // True when player reached final deck and lift has no destination (completion)
	RunStartedAt             int64                 // Unix ms when the current run began
	CompletionPhase          CompletionPhase       // Summary stats or credits roll
	RunStatsSnapshot         RunStats              // Stats frozen at completion
	CreditsLineIndex         int                   // Current credits line during CompletionPhaseCredits
	CreditsLineStartMs       int64                 // When the current credits line slide-in began
	CreditsExitStartMs       int64                 // Non-zero while the current line is sliding out (manual/auto advance)
	CreditsTransitionStartMs int64                 // Non-zero during summary→credits crossfade

	// Room power: doors and CCTV/hazard controls are unpowered by default.
	// Start room's doors are powered so the player can leave.
	RoomDoorsPowered     map[string]bool // room name -> power grid armed (player enabled circuit at maint terminal)
	RoomCCTVPowered      map[string]bool // room name -> CCTV requested when room is online
	RoomLightsPowered    map[string]bool // room name -> lights enabled (0w; toggled at maintenance terminal)
	RoomPowerOnline      map[string]bool // room name -> propagated power has reached the room
	ManualEgressReleased map[string]bool // room name -> hold-to-release bypass active (routing still offline)

	// PowerPropPending schedules room activations when propagation is staggered (unused at 0 delay).
	PowerPropPending []PowerPropEntry

	// ObservationCueVisited prevents duplicate corridor-stamp callouts per cell (Story 5.2).
	ObservationCueVisited map[string]struct{}

	// LinkageTokensSeen records cross-room relay tokens inferred by the player (Story 5.3).
	LinkageTokensSeen map[string]struct{}

	// LinkageCueVisited suppresses repeat callouts for linkage corridor stamps (Story 5.3).
	LinkageCueVisited map[string]struct{}

	// MaintenanceMenuRoom is set while the maintenance menu is open; the room whose
	// maintenance view is displayed. Used to highlight that room's wall cells on the map.
	MaintenanceMenuRoom string

	// MaintenanceMenuMode is "controls" or "diagnostics" while the maintenance menu is open.
	MaintenanceMenuMode string

	// MaintenanceSelectableRooms lists rooms the player can target from the open terminal (for map overlay).
	MaintenanceSelectableRooms []string

	// MaintenanceMenuTerminalRow/Col identify the active maintenance terminal while its menu is open (-1 when unset).
	MaintenanceMenuTerminalRow int
	MaintenanceMenuTerminalCol int

	// PowerGridOverlayActive shows the power grid from the seed cell (generator use-key toggle).
	PowerGridOverlayActive  bool
	PowerGridOverlaySeedRow int
	PowerGridOverlaySeedCol int

	// LongUse holds an in-progress hold-to-use interaction (nil when inactive).
	LongUse *LongUseSession
}

// LongUseSession tracks a hold-to-use interaction in progress.
type LongUseSession struct {
	Kind          string
	TargetRow     int
	TargetCol     int
	DurationMs    int64
	StartedAtMs   int64
	AccumulatedMs int64 // Milliseconds USE was held (progress only advances while held)
	LastAdvanceMs int64 // Last Update tick while held (0 until first held frame)
}

// MessageEntry represents a message with a timestamp
type MessageEntry struct {
	Text      string
	Timestamp int64 // Unix timestamp in milliseconds when message was added
}

// PowerPropEntry schedules when an armed room should receive propagated power.
type PowerPropEntry struct {
	RoomName   string
	ActivateAt int64
}

// NewGame creates a new game instance
func NewGame() *Game {
	return &Game{
		OwnedItems:            mapset.New[*world.Item](),
		PlayerFacing:          FaceNorth,
		HasMap:                false,
		Messages:              make([]MessageEntry, 0),
		Level:                 1,
		CurrentDeckID:         0,
		DeckStates:            make(map[int]*DeckState),
		Batteries:             0,
		Generators:            make([]*entities.Generator, 0),
		FoundCodes:            make(map[string]bool),
		LastInteractedRow:     -1,
		LastInteractedCol:     -1,
		InteractionPlayerRow:  -1,
		InteractionPlayerCol:  -1,
		PowerSupply:           0,
		PowerConsumption:      0,
		PowerOverloadWarned:   false,
		RoomDoorsPowered:      make(map[string]bool),
		RoomCCTVPowered:       make(map[string]bool),
		RoomLightsPowered:     make(map[string]bool),
		RoomPowerOnline:       make(map[string]bool),
		ManualEgressReleased:  make(map[string]bool),
		ObservationCueVisited: make(map[string]struct{}),
		LinkageTokensSeen:     make(map[string]struct{}),
		LinkageCueVisited:     make(map[string]struct{}),
	}
}

// ResetObservationCueAnnounced clears one-shot Story 5.2 movement callout state (new deck / reset).
func (g *Game) ResetObservationCueAnnounced() {
	if g == nil {
		return
	}
	g.ObservationCueVisited = make(map[string]struct{})
}

// ResetLinkageTokensSeen clears Story 5.3 relay attribution (new deck / load / reset).
func (g *Game) ResetLinkageTokensSeen() {
	if g == nil {
		return
	}
	g.LinkageTokensSeen = make(map[string]struct{})
	g.LinkageCueVisited = make(map[string]struct{})
}

// RecordLinkageToken notes that the player has correlated a linkage token across readings.
func (g *Game) RecordLinkageToken(token string) {
	if g == nil || token == "" {
		return
	}
	if g.LinkageTokensSeen == nil {
		g.LinkageTokensSeen = make(map[string]struct{})
	}
	g.LinkageTokensSeen[token] = struct{}{}
}

// HasLinkageToken reports whether linkage has been satisfied for token.
func (g *Game) HasLinkageToken(token string) bool {
	if g == nil || token == "" || g.LinkageTokensSeen == nil {
		return false
	}
	_, ok := g.LinkageTokensSeen[token]
	return ok
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

// RebuildGeneratorsFromGrid repopulates g.Generators from generator entities on the grid.
// Grid cells are the source of truth for per-deck generator fuel/power state.
func (g *Game) RebuildGeneratorsFromGrid() {
	if g == nil || g.Grid == nil {
		g.Generators = nil
		return
	}
	g.Generators = g.generatorsOnGrid()
}

// syncGeneratorsFromGrid updates g.Generators from the grid when generators are placed on cells.
func (g *Game) syncGeneratorsFromGrid() {
	if g == nil || g.Grid == nil {
		return
	}
	if gens := g.generatorsOnGrid(); len(gens) > 0 {
		g.Generators = gens
	}
}

func (g *Game) generatorsOnGrid() []*entities.Generator {
	if g == nil || g.Grid == nil {
		return nil
	}
	var gens []*entities.Generator
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen != nil {
			gens = append(gens, gen)
		}
	})
	return gens
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
	g.ResetLinkageTokensSeen()
	g.PowerSupply = 0
	g.PowerConsumption = 0
	g.PowerOverloadWarned = false
	g.RoomDoorsPowered = make(map[string]bool)
	g.RoomCCTVPowered = make(map[string]bool)
	g.RoomLightsPowered = make(map[string]bool)
	g.RoomPowerOnline = make(map[string]bool)
	g.PowerPropPending = nil
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

func copyBoolMap(m map[string]bool) map[string]bool {
	if m == nil {
		return make(map[string]bool)
	}
	out := make(map[string]bool, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func copyOwnedItems(items world.ItemSet) world.ItemSet {
	out := mapset.New[*world.Item]()
	items.Each(func(item *world.Item) {
		if item != nil {
			out.Put(world.NewItem(item.Name))
		}
	})
	return out
}

// SaveCurrentDeckState stores the current deck's grid and power state into DeckStates (Phase 3.4).
// Call before switching to another deck so the current deck can be restored on revisit.
func (g *Game) SaveCurrentDeckState() {
	if g.Grid == nil {
		return
	}
	g.syncGeneratorsFromGrid()
	doorsCopy, cctvCopy, lightsCopy := copyPowerMaps(g.RoomDoorsPowered, g.RoomCCTVPowered, g.RoomLightsPowered)
	onlineCopy := copyBoolMap(g.RoomPowerOnline)
	manualEgressCopy := copyBoolMap(g.ManualEgressReleased)
	pendingCopy := append([]PowerPropEntry(nil), g.PowerPropPending...)
	genCopy := make([]*entities.Generator, len(g.Generators))
	for i, gen := range g.Generators {
		genCopy[i] = &entities.Generator{
			Name:              gen.Name,
			BatteriesRequired: gen.BatteriesRequired,
			BatteriesInserted: gen.BatteriesInserted,
			Online:            gen.Online,
			Tripped:           gen.Tripped,
		}
	}
	g.DeckStates[g.CurrentDeckID] = &DeckState{
		Grid:                 g.Grid,
		LevelSeed:            g.LevelSeed,
		RoomDoorsPowered:     doorsCopy,
		RoomCCTVPowered:      cctvCopy,
		RoomLightsPowered:    lightsCopy,
		RoomPowerOnline:      onlineCopy,
		PowerPropPending:     pendingCopy,
		ManualEgressReleased: manualEgressCopy,
		Generators:           genCopy,
		OwnedItems:           copyOwnedItems(g.OwnedItems),
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
	if ds.RoomPowerOnline != nil {
		g.RoomPowerOnline = copyBoolMap(ds.RoomPowerOnline)
	} else {
		g.RoomPowerOnline = make(map[string]bool)
	}
	if ds.ManualEgressReleased != nil {
		g.ManualEgressReleased = copyBoolMap(ds.ManualEgressReleased)
	} else {
		g.ManualEgressReleased = make(map[string]bool)
	}
	g.PowerPropPending = append([]PowerPropEntry(nil), ds.PowerPropPending...)
	g.OwnedItems = copyOwnedItems(ds.OwnedItems)
	g.RebuildGeneratorsFromGrid()
	g.HasMap = false
	g.Hints = nil
	g.FoundCodes = make(map[string]bool)
	g.ResetLinkageTokensSeen()
	g.PowerSupply = 0
	g.PowerConsumption = 0
	g.PowerOverloadWarned = false
	g.ResetObservationCueAnnounced()
	g.CurrentCell = g.Grid.StartCell()
}

// GetAvailablePower returns the available power (supply - consumption)
func (g *Game) GetAvailablePower() int {
	return g.PowerSupply - g.PowerConsumption
}

// UpdatePowerSupply recalculates total deck power generation from all powered generators.
func (g *Game) UpdatePowerSupply() {
	totalPower := 0
	for _, gen := range g.Generators {
		if gen.IsPowered() {
			totalPower += 100
		}
	}
	g.PowerSupply = totalPower
}

// CalculatePowerConsumption returns total power draw from online rooms (propagated power).
func (g *Game) CalculatePowerConsumption() int {
	if g == nil || g.RoomPowerOnline == nil {
		return 0
	}
	cctv := make(map[string]bool, len(g.RoomCCTVPowered))
	for k, v := range g.RoomCCTVPowered {
		cctv[k] = v
	}
	return calculateConsumptionFromMaps(g, g.RoomPowerOnline, cctv)
}

func calculateConsumptionFromMaps(g *Game, online, cctv map[string]bool) int {
	if g == nil || g.Grid == nil {
		return 0
	}
	rawConsumption := 0
	doorRoomCounted := make(map[string]bool)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Terminal != nil && cctv[cell.Name] && online[cell.Name] {
			rawConsumption += 10
		}
		if data.Door != nil && online[data.Door.RoomName] && !doorRoomCounted[data.Door.RoomName] {
			rawConsumption += 10
			doorRoomCounted[data.Door.RoomName] = true
		}
		if data.Puzzle != nil && data.Puzzle.IsSolved() {
			rawConsumption += 3
		}
	})
	params := deck.DecayParamsForDeck(g.CurrentDeckID)
	return int(float64(rawConsumption) * params.PowerCostMultiplier)
}

// AddFoundCode records that the player has found a puzzle code
func (g *Game) AddFoundCode(code string) {
	g.FoundCodes[code] = true
}

// HasFoundCode checks if the player has found a specific code
func (g *Game) HasFoundCode(code string) bool {
	return g.FoundCodes[code]
}
