// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
// Ebiten is a 2D game library for Go: https://ebiten.org/
package ebiten

import (
	"bytes"
	"fmt"
	"image/color"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/leonelquinteros/gotext"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/config"
	"darkstation/pkg/game/entities"
	gamemenu "darkstation/pkg/game/menu"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
	"darkstation/pkg/resources"
)

// Color palette for the game - brighter colors for visibility
var (
	colorBackground        = color.RGBA{26, 26, 46, 255}    // Dark blue-gray
	colorMapBackground     = color.RGBA{15, 15, 26, 255}    // Darker for map area
	colorPlayer            = color.RGBA{0, 255, 0, 255}     // Bright green
	colorWall              = color.RGBA{180, 180, 200, 255} // Light gray-blue for wall text
	colorWallBg            = color.RGBA{60, 60, 80, 255}    // Darker background for walls
	colorWallBgPowered     = color.RGBA{40, 80, 40, 255}    // Dark green background for walls in powered rooms
	colorFloor             = color.RGBA{100, 100, 120, 255} // Medium gray for undiscovered
	colorFloorVisited      = color.RGBA{160, 160, 180, 255} // Lighter gray for visited
	colorDoorLocked        = color.RGBA{255, 255, 0, 255}   // Bright yellow
	colorDoorUnlocked      = color.RGBA{0, 220, 0, 255}     // Bright green
	colorKeycard           = color.RGBA{100, 150, 255, 255} // Bright blue
	colorItem              = color.RGBA{220, 170, 255, 255} // Bright purple
	colorBattery           = color.RGBA{255, 200, 100, 255} // Orange for batteries
	colorHazard            = color.RGBA{255, 80, 80, 255}   // Bright red
	colorHazardCtrl        = color.RGBA{0, 255, 255, 255}   // Bright cyan
	colorGeneratorOff      = color.RGBA{255, 100, 100, 255} // Bright red
	colorGeneratorOn       = color.RGBA{0, 255, 100, 255}   // Bright green
	colorTerminal          = color.RGBA{100, 150, 255, 255} // Bright blue
	colorTerminalUsed      = color.RGBA{120, 120, 140, 255} // Medium gray
	colorMaintenance       = color.RGBA{255, 165, 0, 255}   // Orange for maintenance terminals
	colorFurniture         = color.RGBA{255, 150, 255, 255} // Bright pink
	colorFurnitureCheck    = color.RGBA{200, 180, 100, 255} // Tan/brown
	colorExitLocked        = color.RGBA{255, 100, 100, 255} // Bright red
	colorExitUnlocked      = color.RGBA{100, 255, 100, 255} // Bright green
	colorSubtle            = color.RGBA{120, 120, 140, 255} // Medium gray
	colorText              = color.RGBA{240, 240, 255, 255} // Bright off-white
	colorAction            = color.RGBA{220, 170, 255, 255} // Bright purple
	colorDenied            = color.RGBA{255, 100, 100, 255} // Bright red
	colorPanelBackground   = color.RGBA{30, 30, 50, 220}    // Semi-transparent dark
	colorFocusBackground   = color.RGBA{60, 80, 100, 200}   // Dark blue-gray for focused/interacted cell (darker than cell text)
	colorBlockedBackground = color.RGBA{100, 100, 130, 220} // Brighter background for hazards and locked doors that need to be cleared

	// Callout colors
	ColorCalloutInfo    = color.RGBA{200, 200, 255, 255} // Light blue for info
	ColorCalloutSuccess = color.RGBA{100, 255, 150, 255} // Green for success
	ColorCalloutWarning = color.RGBA{255, 220, 100, 255} // Yellow for warnings
	ColorCalloutDanger  = color.RGBA{255, 120, 120, 255} // Red for danger/blocked
	ColorCalloutItem    = color.RGBA{220, 170, 255, 255} // Purple for items
)

// Icon constants - Unicode characters for proper font rendering
// Icon constants matching TUI renderer
const (
	PlayerIcon             = "@"
	IconWall               = "▒"
	IconUnvisited          = "●"
	IconVisited            = "○"
	IconVoid               = " "
	IconExitLocked         = "▲" // Locked lift (unpowered)
	IconExitUnlocked       = "△" // Unlocked lift (powered)
	IconKey                = "⚷" // Key item on floor
	IconItem               = "?" // Generic item on floor
	IconBattery            = "■" // Battery on floor
	IconGeneratorUnpowered = "◇" // Unpowered generator
	IconGeneratorPowered   = "◆" // Powered generator
	IconDoorLocked         = "▣" // Locked door
	IconDoorUnlocked       = "□" // Unlocked door
	IconTerminalUnused     = "▫" // Unused CCTV terminal
	IconTerminalUsed       = "▪" // Used CCTV terminal
	IconMaintenance        = "▤" // Maintenance terminal
)

// Floor icons for different room types (visited/unvisited pairs)
var roomFloorIcons = map[string][2]string{
	"Bridge":          {"◎", "◉"}, // Command areas
	"Command Center":  {"◎", "◉"},
	"Communications":  {"◎", "◉"},
	"Security":        {"◎", "◉"},
	"Engineering":     {"▫", "▪"}, // Technical areas
	"Reactor Core":    {"▫", "▪"},
	"Server Room":     {"▫", "▪"},
	"Maintenance Bay": {"▫", "▪"},
	"Life Support":    {"▫", "▪"},
	"Cargo Bay":       {"□", "▣"}, // Storage areas
	"Storage":         {"□", "▣"},
	"Hangar":          {"□", "▣"},
	"Armory":          {"□", "▣"},
	"Med Bay":         {"◇", "◆"}, // Science/medical areas
	"Lab":             {"◇", "◆"},
	"Hydroponics":     {"◇", "◆"},
	"Observatory":     {"◇", "◆"},
	"Crew Quarters":   {"○", "●"}, // Living areas (using larger circles for visibility)
	"Mess Hall":       {"○", "●"},
	"Airlock":         {"╳", "╳"}, // Special areas
	"Corridor":        {"░", "░"}, // Corridors
}

// Tile size constraints
const (
	minTileSize  = 12
	maxTileSize  = 144 // Increased by 3x for higher zoom levels
	tileSizeStep = 4
	baseFontSize = 16.0 // Base font size at default tile size
)

// Callout represents a floating message displayed near a cell
type Callout struct {
	Row       int    // Cell row
	Col       int    // Cell col
	Message   string // Message to display
	Color     color.Color
	ExpiresAt int64 // Unix timestamp when callout expires (0 = never)
	CreatedAt int64 // Unix timestamp when callout was created (for animations)
}

// messageEntry represents a message with timestamp for fade-out
type messageEntry struct {
	Text      string
	Timestamp int64 // Unix timestamp in milliseconds when message was added
}

// roomLabel represents a persistent label for a room, positioned at the leftmost point
type roomLabel struct {
	RoomName string
	Row      int // Grid row of the label position (room interior row)
	StartCol int // Grid column index (leftmost point)
	EndCol   int // Same as StartCol (kept for compatibility)
}

// renderSnapshot holds a consistent snapshot of game state for rendering
// This prevents jitter from race conditions between game logic and rendering
type renderSnapshot struct {
	valid             bool
	level             int
	playerRow         int
	playerCol         int
	cellName          string
	hasMap            bool
	batteries         int
	messages          []messageEntry
	ownedItems        []string
	generators        []generatorState
	gridRows          int
	gridCols          int
	callouts          []Callout
	roomLabels        []roomLabel
	objectives        []string // Current level objectives
	exitAnimating     bool     // True when exit animation is playing
	exitAnimStartTime int64    // Timestamp when exit animation started
	focusedCellRow    int      // Row of cell with active callout (for focus background)
	focusedCellCol    int      // Col of cell with active callout (for focus background)
	interactableCells []struct {
		row int
		col int
	} // Cells with interactable objects (for focus background)
}

// generatorState holds generator info for rendering
type generatorState struct {
	powered           bool
	batteriesInserted int
	batteriesRequired int
}

// EbitenRenderer is the Ebiten-based graphical renderer
type EbitenRenderer struct {
	// Window dimensions
	windowWidth  int
	windowHeight int

	// Tile size for rendering (adjustable with +/-)
	tileSize int

	// Viewport dimensions (in tiles) - recalculated based on window and tile size
	viewportRows int
	viewportCols int

	// Font sources for text rendering
	monoFontSource *text.GoTextFaceSource // Monospace font for map tiles
	sansFontSource *text.GoTextFaceSource // Sans-serif font for UI text

	// Cached font faces (recreated when tile size changes)
	cachedTileFontSize float64
	cachedUIFontSize   float64
	cachedMonoFace     *text.GoTextFace
	cachedSansFace     *text.GoTextFace

	// Current game state (set by RenderFrame)
	game      *state.Game
	gameMutex sync.RWMutex

	// Cached render snapshot for consistent drawing
	snapshot      renderSnapshot
	snapshotMutex sync.RWMutex

	// Active callouts (floating messages near cells)
	callouts      []Callout
	calloutsMutex sync.RWMutex

	// Track last player position to clear callouts on move
	lastPlayerRow      int
	lastPlayerCol      int
	lastRoomName       string
	lastPosInitialized bool

	// Input channel for communication between Ebiten and game loop
	inputChan chan engineinput.Intent

	// Flag indicating renderer is running
	running bool

	// Messages to display with timestamps for fade-out
	trackedMessages []messageEntry
	messagesMutex   sync.RWMutex

	// Bindings menu overlay state (deprecated - use generic menu state)
	menuActive        bool
	menuActions       []engineinput.Action
	menuSelected      int
	menuHelpText      string
	menuNonRebindable map[engineinput.Action]bool
	menuMutex         sync.RWMutex

	// Generic menu overlay state
	genericMenuActive   bool
	genericMenuItems    []gamemenu.MenuItem
	genericMenuSelected int
	genericMenuHelpText string
	genericMenuTitle    string
	genericMenuMutex    sync.RWMutex

	// Analog stick state tracking (for edge detection)
	// Maps gamepad ID to previous stick state (x, y values)
	stickState      map[ebiten.GamepadID]struct{ x, y float64 }
	stickStateMutex sync.RWMutex

	// Key repeat state tracking
	// Maps key/button codes to their repeat state
	keyRepeatState      map[string]keyRepeatInfo
	keyRepeatStateMutex sync.RWMutex

	// Debounce animation state (for failed movement attempts)
	debounceDirection string // "north", "south", "east", "west"
	debounceStartTime int64  // Timestamp when debounce started (milliseconds)
	debounceMutex     sync.RWMutex
}

// keyRepeatInfo tracks the repeat state for a key or button
type keyRepeatInfo struct {
	firstPressed int64 // Timestamp when first pressed (milliseconds)
	lastRepeat   int64 // Timestamp when last repeat event was sent (milliseconds)
}

const (
	keyRepeatInitialDelay = 500 // Initial delay before first repeat (milliseconds)
	keyRepeatInterval     = 100 // Interval between repeat events (milliseconds)
)

// New creates a new Ebiten renderer
func New() *EbitenRenderer {
	return &EbitenRenderer{
		windowWidth:    1024,
		windowHeight:   768,
		tileSize:       24,
		viewportRows:   21,
		viewportCols:   35,
		inputChan:      make(chan engineinput.Intent, 10),
		running:        false,
		stickState:     make(map[ebiten.GamepadID]struct{ x, y float64 }),
		keyRepeatState: make(map[string]keyRepeatInfo),
	}
}

// Init initializes the Ebiten renderer
func (e *EbitenRenderer) Init() {
	ebiten.SetWindowSize(e.windowWidth, e.windowHeight)
	ebiten.SetWindowTitle("The Dark Station")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Enable VSync for smooth rendering (prevents tearing and reduces jitter)
	ebiten.SetVsyncEnabled(true)

	// Set TPS to match typical display refresh rate for smoother updates
	ebiten.SetTPS(60)

	// Load saved preferences
	cfg := config.Current()
	if cfg.TileSize >= minTileSize && cfg.TileSize <= maxTileSize {
		e.tileSize = cfg.TileSize
	}

	// Load the monospace font for map tiles (embedded Cascadia Code NF)
	monoSrc, err := text.NewGoTextFaceSource(bytes.NewReader(resources.CascadiaCodeNFRegular))
	if err != nil {
		// Fall back to embedded Go Mono if Cascadia Code NF fails to load
		fmt.Println("[Font] Monospace: Cascadia Code NF failed to load, using Go Mono (embedded)")
		monoSrc, err = text.NewGoTextFaceSource(bytes.NewReader(gomono.TTF))
		if err != nil {
			panic(fmt.Sprintf("failed to load mono font: %v", err))
		}
	} else {
		fmt.Println("[Font] Monospace: Cascadia Code NF (embedded)")
	}
	e.monoFontSource = monoSrc

	// Load the sans-serif font for UI text
	sansSrc, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(fmt.Sprintf("failed to load sans font: %v", err))
	}
	fmt.Println("[Font] Sans-serif: Go Regular (embedded)")
	e.sansFontSource = sansSrc

	// Calculate initial viewport based on window and tile size
	e.recalculateViewport()
}

// Clear clears the display (no-op for Ebiten, clearing happens in Draw)
func (e *EbitenRenderer) Clear() {
	// In Ebiten, clearing happens automatically in Draw
}

// GetInput gets user input from Ebiten (blocking)
func (e *EbitenRenderer) GetInput() engineinput.Intent {
	// Wait for input from the Ebiten game loop
	return <-e.inputChan
}

// StyleText applies a style to text
// For Ebiten, we return the text as-is since styling is done during rendering
func (e *EbitenRenderer) StyleText(text string, style renderer.TextStyle) string {
	// For Ebiten, styling is applied during rendering, not in the text itself
	return text
}

// FormatText formats a message with the markup system
// For Ebiten, we preserve the markup so it can be parsed and colored when displaying
func (e *EbitenRenderer) FormatText(msg string, args ...any) string {
	// Format with arguments, but preserve markup tags for later parsing
	return fmt.Sprintf(msg, args...)
}

// ShowMessage displays a message to the user
func (e *EbitenRenderer) ShowMessage(msg string) {
	e.messagesMutex.Lock()
	defer e.messagesMutex.Unlock()
	now := time.Now().UnixMilli()
	e.trackedMessages = append(e.trackedMessages, messageEntry{
		Text:      msg,
		Timestamp: now,
	})
	// Keep only the last 5 messages
	if len(e.trackedMessages) > 5 {
		e.trackedMessages = e.trackedMessages[len(e.trackedMessages)-5:]
	}
}

// GetViewportSize returns the current viewport dimensions
func (e *EbitenRenderer) GetViewportSize() (rows, cols int) {
	return e.viewportRows, e.viewportCols
}

// calculateObjectives calculates the current level objectives based on game state
func (e *EbitenRenderer) calculateObjectives(g *state.Game) []string {
	if g == nil || g.Grid == nil {
		return nil
	}

	var objectives []string

	// Count unpowered generators (show remaining, not total)
	unpoweredGenerators := g.UnpoweredGeneratorCount()
	if unpoweredGenerators > 0 {
		if unpoweredGenerators == 1 {
			objectives = append(objectives, "Power up ACTION{1} more generator with batteries.")
		} else {
			objectives = append(objectives, fmt.Sprintf("Power up ACTION{%d} more generators with batteries.", unpoweredGenerators))
		}
	}

	// Count hazards (matching showLevelObjectives logic - count remaining active hazards)
	numHazards := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if gameworld.HasBlockingHazard(cell) {
			numHazards++
		}
	})

	if numHazards > 0 {
		objectives = append(objectives, fmt.Sprintf("Clear ACTION{%d} environmental hazard(s).", numHazards))
	}

	// If all objectives are complete, show exit message
	if len(g.Generators) == 0 && numHazards == 0 {
		objectives = append(objectives, "Find the EXIT{lift} to the next deck.")
	}

	return objectives
}

// RenderFrame stores the game state and captures a snapshot for the next Draw call
func (e *EbitenRenderer) RenderFrame(g *state.Game) {
	e.gameMutex.Lock()
	e.game = g
	e.gameMutex.Unlock()

	// Capture a consistent snapshot of critical render state
	e.snapshotMutex.Lock()
	defer e.snapshotMutex.Unlock()

	if g == nil || g.CurrentCell == nil || g.Grid == nil {
		e.snapshot.valid = false
		return
	}

	// Update tracked player position (clearing is now done via ClearCalloutsIfMoved)
	e.lastPlayerRow = g.CurrentCell.Row
	e.lastPlayerCol = g.CurrentCell.Col
	e.lastPosInitialized = true

	e.snapshot.valid = true
	e.snapshot.level = g.Level
	e.snapshot.playerRow = g.CurrentCell.Row
	e.snapshot.playerCol = g.CurrentCell.Col
	e.snapshot.cellName = g.CurrentCell.Name
	e.snapshot.hasMap = g.HasMap
	e.snapshot.batteries = g.Batteries
	e.snapshot.gridRows = g.Grid.Rows()
	e.snapshot.gridCols = g.Grid.Cols()

	// Compute persistent room labels (for rooms the player has visited)
	e.snapshot.roomLabels = e.computeRoomLabels(g)

	// Track messages with timestamps and handle fade-out
	e.messagesMutex.Lock()
	now := time.Now().UnixMilli()
	const messageLifetime = 10000 // 10 seconds in milliseconds

	// Create a map of current game messages for quick lookup
	currentMessages := make(map[string]bool)
	for _, msg := range g.Messages {
		currentMessages[msg.Text] = true
	}

	// Update tracked messages: add new ones, keep existing ones (even if removed from game), remove expired ones
	updatedMessages := make([]messageEntry, 0)

	// Keep existing tracked messages that are not expired (even if removed from g.Messages)
	for _, tracked := range e.trackedMessages {
		age := now - tracked.Timestamp
		if age < messageLifetime {
			updatedMessages = append(updatedMessages, tracked)
		}
		// Messages older than 10 seconds are discarded (not added to updatedMessages)
	}

	// Add new messages from game that aren't already tracked
	for _, msg := range g.Messages {
		found := false
		for _, tracked := range e.trackedMessages {
			if tracked.Text == msg.Text {
				found = true
				break
			}
		}
		if !found {
			updatedMessages = append(updatedMessages, messageEntry{
				Text:      msg.Text,
				Timestamp: msg.Timestamp,
			})
		}
	}

	// Sort messages by timestamp (oldest first) to ensure chronological ordering
	// This ensures consistent ordering regardless of when messages were added
	sort.Slice(updatedMessages, func(i, j int) bool {
		return updatedMessages[i].Timestamp < updatedMessages[j].Timestamp
	})

	e.trackedMessages = updatedMessages

	// Copy to snapshot (only non-expired messages)
	e.snapshot.messages = make([]messageEntry, len(e.trackedMessages))
	copy(e.snapshot.messages, e.trackedMessages)
	e.messagesMutex.Unlock()

	// Copy owned items
	// Collect and sort items deterministically
	e.snapshot.ownedItems = make([]string, 0)
	g.OwnedItems.Each(func(item *world.Item) {
		e.snapshot.ownedItems = append(e.snapshot.ownedItems, item.Name)
	})
	// Sort items for deterministic display order
	sort.Strings(e.snapshot.ownedItems)

	// Copy generator states
	e.snapshot.generators = make([]generatorState, len(g.Generators))
	for i, gen := range g.Generators {
		e.snapshot.generators[i] = generatorState{
			powered:           gen.IsPowered(),
			batteriesInserted: gen.BatteriesInserted,
			batteriesRequired: gen.BatteriesRequired,
		}
	}

	// Calculate objectives
	e.snapshot.objectives = e.calculateObjectives(g)

	// Copy exit animation state
	e.snapshot.exitAnimating = g.ExitAnimating
	e.snapshot.exitAnimStartTime = g.ExitAnimStartTime

	// Find the cell with the most recent active callout (for focus background)
	e.snapshot.focusedCellRow = -1
	e.snapshot.focusedCellCol = -1
	e.calloutsMutex.RLock()
	nowUnixMilli := time.Now().UnixMilli()
	var mostRecentCallout *Callout
	for i := range e.callouts {
		callout := &e.callouts[i]
		// Check if callout is active (not expired)
		if callout.ExpiresAt == 0 || callout.ExpiresAt > nowUnixMilli {
			if mostRecentCallout == nil || callout.CreatedAt > mostRecentCallout.CreatedAt {
				mostRecentCallout = callout
			}
		}
	}
	if mostRecentCallout != nil {
		e.snapshot.focusedCellRow = mostRecentCallout.Row
		e.snapshot.focusedCellCol = mostRecentCallout.Col
	}
	e.calloutsMutex.RUnlock()

	// Find interactable cells adjacent to player (for focus background)
	e.snapshot.interactableCells = make([]struct {
		row int
		col int
	}, 0)
	if g.CurrentCell != nil {
		neighbors := []*world.Cell{
			g.CurrentCell.North,
			g.CurrentCell.South,
			g.CurrentCell.East,
			g.CurrentCell.West,
		}
		for _, cell := range neighbors {
			if cell == nil {
				continue
			}
			// Check if cell has interactable objects
			if gameworld.HasFurniture(cell) ||
				gameworld.HasUnusedTerminal(cell) ||
				gameworld.HasUnsolvedPuzzle(cell) ||
				gameworld.HasInactiveHazardControl(cell) {
				e.snapshot.interactableCells = append(e.snapshot.interactableCells, struct {
					row int
					col int
				}{cell.Row, cell.Col})
			}
		}
	}

	// Copy active callouts (with expiration filtering)
	e.calloutsMutex.Lock()
	// Reuse nowUnixMilli from above (already calculated)
	activeCallouts := make([]Callout, 0)
	for _, c := range e.callouts {
		if c.ExpiresAt == 0 || c.ExpiresAt > nowUnixMilli {
			activeCallouts = append(activeCallouts, c)
		}
	}
	e.callouts = activeCallouts // Remove expired callouts
	e.snapshot.callouts = make([]Callout, len(activeCallouts))
	copy(e.snapshot.callouts, activeCallouts)
	e.calloutsMutex.Unlock()
}

// computeRoomLabels finds the leftmost valid position for each visited room's label,
// avoiding gaps (corridor cells) and ensuring the label starts at the leftmost point of the room.
func (e *EbitenRenderer) computeRoomLabels(g *state.Game) []roomLabel {
	if g == nil || g.Grid == nil {
		return nil
	}

	rows := g.Grid.Rows()
	cols := g.Grid.Cols()

	// Track which rooms have been visited (player has stepped inside)
	roomVisited := make(map[string]bool)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			cell := g.Grid.GetCell(row, col)
			if cell == nil || !cell.Room || cell.Name == "" {
				continue
			}
			// Never label corridors
			if strings.Contains(strings.ToLower(cell.Name), "corridor") {
				continue
			}
			if cell.Visited {
				roomVisited[cell.Name] = true
			}
		}
	}

	if len(roomVisited) == 0 {
		return nil
	}

	// For each room, find the leftmost valid position for the label
	// The label should be placed above a room cell, avoiding gaps (corridors) above
	type labelPos struct {
		row int
		col int
	}
	leftmostByRoom := make(map[string]labelPos)

	// First pass: find the leftmost column for each room
	leftmostColByRoom := make(map[string]int)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			cell := g.Grid.GetCell(row, col)
			if cell == nil || !cell.Room || cell.Name == "" {
				continue
			}

			roomName := cell.Name
			// Skip corridors and unvisited rooms
			if strings.Contains(strings.ToLower(roomName), "corridor") || !roomVisited[roomName] {
				continue
			}

			// Track the leftmost column for this room
			if leftmostCol, ok := leftmostColByRoom[roomName]; !ok || col < leftmostCol {
				leftmostColByRoom[roomName] = col
			}
		}
	}

	// Second pass: for each room, find the best row at the leftmost column
	// Prefer positions where the cell above (where label renders) is not a gap/corridor
	for roomName, leftmostCol := range leftmostColByRoom {
		bestRow := -1
		bestHasGap := true // Track if best position has a gap above

		// Scan rows at the leftmost column
		for row := 0; row < rows; row++ {
			cell := g.Grid.GetCell(row, leftmostCol)
			if cell == nil || !cell.Room || cell.Name != roomName {
				continue
			}

			// Check if the cell above (where label would render) is a gap/corridor
			labelRow := row - 1
			hasGap := false
			if labelRow < 0 {
				hasGap = true // Edge of map
			} else {
				aboveCell := g.Grid.GetCell(labelRow, leftmostCol)
				if aboveCell == nil || !aboveCell.Room || strings.Contains(strings.ToLower(aboveCell.Name), "corridor") {
					hasGap = true
				}
			}

			// Prefer positions without gaps, or if all have gaps, use the first (topmost) one
			if bestRow == -1 {
				bestRow = row
				bestHasGap = hasGap
			} else if !hasGap && bestHasGap {
				// This position has no gap, current best has gap - prefer this
				bestRow = row
				bestHasGap = false
			} else if hasGap == bestHasGap && row < bestRow {
				// Both have same gap status - prefer higher (topmost) row
				bestRow = row
			}
		}

		if bestRow >= 0 {
			leftmostByRoom[roomName] = labelPos{row: bestRow, col: leftmostCol}
		}
	}

	if len(leftmostByRoom) == 0 {
		return nil
	}

	labels := make([]roomLabel, 0, len(leftmostByRoom))
	for roomName, pos := range leftmostByRoom {
		// Use the leftmost column as both start and end (single cell position)
		// The drawing code will position the label starting from this point
		labels = append(labels, roomLabel{
			RoomName: roomName,
			Row:      pos.row,
			StartCol: pos.col,
			EndCol:   pos.col, // Single cell position
		})
	}
	return labels
}

// AddCallout adds a floating message callout near a specific cell
func (e *EbitenRenderer) AddCallout(row, col int, message string, col_color color.Color, durationMs int) {
	e.calloutsMutex.Lock()
	defer e.calloutsMutex.Unlock()

	var expiresAt int64
	if durationMs > 0 {
		expiresAt = time.Now().UnixMilli() + int64(durationMs)
	}

	// Remove any existing callout at the same position
	filtered := make([]Callout, 0)
	for _, c := range e.callouts {
		if c.Row != row || c.Col != col {
			filtered = append(filtered, c)
		}
	}

	now := time.Now().UnixMilli()
	filtered = append(filtered, Callout{
		Row:       row,
		Col:       col,
		Message:   message,
		Color:     col_color,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	})
	e.callouts = filtered
}

// SetDebounceAnimation triggers a debounce animation in the given direction
func (e *EbitenRenderer) SetDebounceAnimation(direction string) {
	e.debounceMutex.Lock()
	defer e.debounceMutex.Unlock()
	e.debounceDirection = direction
	e.debounceStartTime = time.Now().UnixMilli()
}

// AddCalloutAtPlayer adds a callout at the player's current position
func (e *EbitenRenderer) AddCalloutAtPlayer(message string, col color.Color, durationMs int) {
	e.snapshotMutex.RLock()
	row, col_pos := e.snapshot.playerRow, e.snapshot.playerCol
	e.snapshotMutex.RUnlock()
	e.AddCallout(row, col_pos, message, col, durationMs)
}

// AddCalloutNearPlayer adds a callout at an adjacent cell (for interactions)
func (e *EbitenRenderer) AddCalloutNearPlayer(row, col int, message string, col_color color.Color, durationMs int) {
	e.AddCallout(row, col, message, col_color, durationMs)
}

// ClearCallouts removes all active callouts
func (e *EbitenRenderer) ClearCallouts() {
	e.calloutsMutex.Lock()
	defer e.calloutsMutex.Unlock()
	e.callouts = nil
}

// ClearCalloutsIfMoved clears callouts if player has moved from tracked position
// Returns true if callouts were cleared
func (e *EbitenRenderer) ClearCalloutsIfMoved(row, col int) bool {
	if !e.lastPosInitialized {
		return false
	}
	if e.lastPlayerRow != row || e.lastPlayerCol != col {
		e.calloutsMutex.Lock()
		e.callouts = nil
		e.calloutsMutex.Unlock()
		return true
	}
	return false
}

// ShowRoomEntryIfNew shows a room entry callout if the player entered a new room
// Skips corridors and returns true if a callout was shown
func (e *EbitenRenderer) ShowRoomEntryIfNew(row, col int, roomName string) bool {
	// Skip if room name hasn't changed
	if e.lastRoomName == roomName {
		return false
	}

	// Update tracked room name
	oldRoom := e.lastRoomName
	e.lastRoomName = roomName

	// Skip corridors
	lowerName := strings.ToLower(roomName)
	if strings.Contains(lowerName, "corridor") || strings.Contains(lowerName, "hallway") {
		return false
	}

	// Skip if this is the first room (game just started)
	if oldRoom == "" {
		return false
	}

	// Room entry callout removed - no longer showing room titles on entry
	return true
}

// Update handles input and game logic (Ebiten interface)
func (e *EbitenRenderer) Update() error {
	// Handle font size changes (Ctrl+= to increase, Ctrl+- to decrease)
	e.handleZoom()

	// Check for gamepad input first, then fall back to keyboard (raw layer)
	if intent := e.checkGamepadInput(); intent.Action != engineinput.ActionNone {
		// Non-blocking send to input channel
		select {
		case e.inputChan <- intent:
		default:
			// Channel full, drop input
		}
	} else if intent := e.checkInput(); intent.Action != engineinput.ActionNone {
		// Non-blocking send to input channel
		select {
		case e.inputChan <- intent:
		default:
			// Channel full, drop input
		}
	}

	return nil
}

// handleZoom handles =/- for font/tile size adjustment
func (e *EbitenRenderer) handleZoom() {
	// = or + to increase font size
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) {
		e.increaseTileSize()
	}
	// - to decrease font size
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) {
		e.decreaseTileSize()
	}
	// 0 to reset font size
	if inpututil.IsKeyJustPressed(ebiten.Key0) || inpututil.IsKeyJustPressed(ebiten.KeyNumpad0) {
		e.resetTileSize()
	}
}

// increaseTileSize increases the tile/font size
func (e *EbitenRenderer) increaseTileSize() {
	if e.tileSize < maxTileSize {
		e.tileSize += tileSizeStep
		e.recalculateViewport()
		e.saveZoomPreference()
	}
}

// decreaseTileSize decreases the tile/font size
func (e *EbitenRenderer) decreaseTileSize() {
	if e.tileSize > minTileSize {
		e.tileSize -= tileSizeStep
		e.recalculateViewport()
		e.saveZoomPreference()
	}
}

// resetTileSize resets tile size to default
func (e *EbitenRenderer) resetTileSize() {
	e.tileSize = 24
	e.recalculateViewport()
	e.saveZoomPreference()
}

// saveZoomPreference saves the current tile size to preferences
func (e *EbitenRenderer) saveZoomPreference() {
	cfg := config.Current()
	if err := cfg.SetTileSize(e.tileSize); err != nil {
		// Silently ignore save errors - not critical
		fmt.Fprintf(os.Stderr, "Warning: could not save preferences: %v\n", err)
	}
}

// recalculateViewport recalculates viewport dimensions based on current window and tile size
func (e *EbitenRenderer) recalculateViewport() {
	// Invalidate font cache since sizes may have changed
	e.invalidateFontCache()

	// Get current window size
	w, h := ebiten.WindowSize()
	if w == 0 || h == 0 {
		w, h = e.windowWidth, e.windowHeight
	}

	// Calculate available space for the map (accounting for UI elements)
	// Header height + small frame border
	uiFontSize := e.getUIFontSize()
	headerHeight := int(uiFontSize) + 20
	frameBorder := 10
	availableHeight := h - headerHeight - frameBorder*2
	availableWidth := w - frameBorder*2

	// Calculate viewport dimensions to maximize the map
	e.viewportCols = availableWidth / e.tileSize
	e.viewportRows = availableHeight / e.tileSize

	// Ensure minimum viewport size
	if e.viewportCols < 15 {
		e.viewportCols = 15
	}
	if e.viewportRows < 11 {
		e.viewportRows = 11
	}

	// Keep odd numbers for centering
	if e.viewportCols%2 == 0 {
		e.viewportCols--
	}
	if e.viewportRows%2 == 0 {
		e.viewportRows--
	}
}

// shouldRepeatKey checks if a key/button should trigger (initial press or repeat)
// Returns true if the key should trigger, false otherwise
func (e *EbitenRenderer) shouldRepeatKey(isPressed func() bool, code string) bool {
	now := time.Now().UnixMilli()

	e.keyRepeatStateMutex.Lock()
	defer e.keyRepeatStateMutex.Unlock()

	pressed := isPressed()
	state, exists := e.keyRepeatState[code]

	if pressed {
		if !exists {
			// First press - record it and trigger immediately
			e.keyRepeatState[code] = keyRepeatInfo{
				firstPressed: now,
				lastRepeat:   now,
			}
			return true
		}

		// Key is held - check if we should repeat
		timeSinceFirstPress := now - state.firstPressed
		timeSinceLastRepeat := now - state.lastRepeat

		if timeSinceFirstPress >= keyRepeatInitialDelay {
			// Initial delay has passed, check repeat interval
			if timeSinceLastRepeat >= keyRepeatInterval {
				// Update last repeat time and trigger
				state.lastRepeat = now
				e.keyRepeatState[code] = state
				return true
			}
		}
		return false
	} else {
		// Key released - clean up state
		if exists {
			delete(e.keyRepeatState, code)
		}
		return false
	}
}

// checkGamepadInput checks for controller/gamepad input and returns the corresponding Intent.
// NOTE: Button indices here are tuned for common XInput-style controllers on Linux;
// mappings may vary between devices/platforms.
func (e *EbitenRenderer) checkGamepadInput() engineinput.Intent {
	// Collect currently connected gamepads
	var ids []ebiten.GamepadID
	ids = ebiten.AppendGamepadIDs(ids[:0])

	for _, id := range ids {
		// Analog stick (left stick) movement
		// Axes: 0 = X (left = -1, right = +1), 1 = Y (up = -1, down = +1)
		const deadZone = 0.5 // Threshold to avoid drift
		const axisX = 0
		const axisY = 1

		stickX := ebiten.GamepadAxisValue(id, axisX)
		stickY := ebiten.GamepadAxisValue(id, axisY)

		// Check horizontal movement (left/right) with key repeat
		stickCodeLeft := fmt.Sprintf("gamepad_%d_stick_left", id)
		if stickX < -deadZone {
			if e.shouldRepeatKey(func() bool { return stickX < -deadZone }, stickCodeLeft) {
				return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
					Device: engineinput.DeviceGamepad,
					Code:   "gamepad_dpad_left",
				}))
			}
		}
		stickCodeRight := fmt.Sprintf("gamepad_%d_stick_right", id)
		if stickX > deadZone {
			if e.shouldRepeatKey(func() bool { return stickX > deadZone }, stickCodeRight) {
				return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
					Device: engineinput.DeviceGamepad,
					Code:   "gamepad_dpad_right",
				}))
			}
		}

		// Check vertical movement (up/down) with key repeat
		stickCodeUp := fmt.Sprintf("gamepad_%d_stick_up", id)
		if stickY < -deadZone {
			if e.shouldRepeatKey(func() bool { return stickY < -deadZone }, stickCodeUp) {
				return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
					Device: engineinput.DeviceGamepad,
					Code:   "gamepad_dpad_up",
				}))
			}
		}
		stickCodeDown := fmt.Sprintf("gamepad_%d_stick_down", id)
		if stickY > deadZone {
			if e.shouldRepeatKey(func() bool { return stickY > deadZone }, stickCodeDown) {
				return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
					Device: engineinput.DeviceGamepad,
					Code:   "gamepad_dpad_down",
				}))
			}
		}

		// Directional pad (D‑pad) movement with key repeat
		// Typical mapping on many XInput-style controllers under Ebiten:
		//  - Up:    11
		//  - Right: 12
		//  - Down:  13
		//  - Left:  14
		code := fmt.Sprintf("gamepad_%d_14", id)
		if e.shouldRepeatKey(func() bool { return ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton14) }, code) {
			return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
				Device: engineinput.DeviceGamepad,
				Code:   "gamepad_dpad_left",
			}))
		}
		code = fmt.Sprintf("gamepad_%d_12", id)
		if e.shouldRepeatKey(func() bool { return ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton12) }, code) {
			return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
				Device: engineinput.DeviceGamepad,
				Code:   "gamepad_dpad_right",
			}))
		}
		code = fmt.Sprintf("gamepad_%d_11", id)
		if e.shouldRepeatKey(func() bool { return ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton11) }, code) {
			return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
				Device: engineinput.DeviceGamepad,
				Code:   "gamepad_dpad_up",
			}))
		}
		code = fmt.Sprintf("gamepad_%d_13", id)
		if e.shouldRepeatKey(func() bool { return ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton13) }, code) {
			return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
				Device: engineinput.DeviceGamepad,
				Code:   "gamepad_dpad_down",
			}))
		}

		// Face buttons:
		// - A: show help / hint
		// - B: quit game
		// Typical mapping:
		//  - A / Cross: 0
		//  - B / Circle: 1
		if inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton0) {
			return engineinput.Intent{Action: engineinput.ActionInteract}
		}
		if inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton1) {
			return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
				Device: engineinput.DeviceGamepad,
				Code:   "gamepad_b",
			}))
		}

		// Start button opens the bindings/menu.
		// Typical mapping:
		//  - Start: 7
		if inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton7) {
			return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
				Device: engineinput.DeviceGamepad,
				Code:   "gamepad_start",
			}))
		}
	}

	return engineinput.Intent{Action: engineinput.ActionNone}
}

// checkInput checks for keyboard input and returns the corresponding Intent.
func (e *EbitenRenderer) checkInput() engineinput.Intent {
	// Arrow keys / NSEW navigation with key repeat
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyArrowUp) }, "key_arrow_up") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "arrow_up",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyArrowDown) }, "key_arrow_down") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "arrow_down",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyArrowLeft) }, "key_arrow_left") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "arrow_left",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyArrowRight) }, "key_arrow_right") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "arrow_right",
		}))
	}

	// WASD navigation (as arrow alternatives) with key repeat
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyW) }, "key_w") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "arrow_up",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyS) && !ebiten.IsKeyPressed(ebiten.KeyControl) }, "key_s") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "arrow_down",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyA) }, "key_a") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "arrow_left",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyD) }, "key_d") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "arrow_right",
		}))
	}

	// Vim-style navigation with key repeat
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyK) }, "key_k") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "k",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyJ) }, "key_j") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "j",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyH) }, "key_h") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "h",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyL) }, "key_l") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "l",
		}))
	}

	// NSEW keys with key repeat
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyN) }, "key_n") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "n",
		}))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		return engineinput.Intent{Action: engineinput.ActionInteract}
	}

	// Help
	if inpututil.IsKeyJustPressed(ebiten.KeySlash) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "?",
		}))
	}

	// Interaction (Enter)
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) {
		return engineinput.Intent{Action: engineinput.ActionInteract}
	}

	// Open menu (F10)
	if inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "f9",
		}))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF10) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "menu",
		}))
	}

	// Reset level (F5)
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		return engineinput.Intent{Action: engineinput.ActionResetLevel}
	}

	// Quit
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "quit",
		}))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "quit",
		}))
	}

	return engineinput.Intent{Action: engineinput.ActionNone}
}

// Draw renders the game to the screen (Ebiten interface)
func (e *EbitenRenderer) Draw(screen *ebiten.Image) {
	// Fill background
	screen.Fill(colorBackground)

	// Get snapshot for consistent rendering
	e.snapshotMutex.RLock()
	snap := e.snapshot
	e.snapshotMutex.RUnlock()

	if !snap.valid || e.monoFontSource == nil || e.sansFontSource == nil {
		// Can't draw without valid snapshot or fonts
		return
	}

	e.gameMutex.RLock()
	g := e.game
	e.gameMutex.RUnlock()

	if g == nil {
		return
	}

	// Get actual screen size
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Calculate font sizes for layout
	uiFontSize := e.getUIFontSize()

	// Calculate layout dimensions with dynamic spacing based on font size
	headerHeight := int(uiFontSize) + 20
	statusBarHeight := int(uiFontSize)*2 + 20

	// Small frame border around the window
	frameBorder := 10

	// Calculate maximum available space for map (after header, with frame border)
	// Note: status bar and messages panel are overlays and do not reduce map height
	availableHeight := screenHeight - headerHeight - frameBorder*2
	availableWidth := screenWidth - frameBorder*2

	// Recalculate viewport to maximize based on current available space
	// This ensures the viewport uses the maximum available space
	viewportCols := availableWidth / e.tileSize
	viewportRows := availableHeight / e.tileSize

	// Ensure minimum viewport size
	if viewportCols < 15 {
		viewportCols = 15
	}
	if viewportRows < 11 {
		viewportRows = 11
	}

	// Keep odd numbers for centering
	if viewportCols%2 == 0 {
		viewportCols--
	}
	if viewportRows%2 == 0 {
		viewportRows--
	}

	// Update stored viewport (will be used in next frame's recalculateViewport)
	e.viewportCols = viewportCols
	e.viewportRows = viewportRows

	// Calculate map dimensions to fill available space
	mapAreaWidth := viewportCols * e.tileSize
	mapAreaHeight := viewportRows * e.tileSize

	// Center the map horizontally, position it right after header with frame border
	mapX := (screenWidth - mapAreaWidth) / 2
	mapY := headerHeight + frameBorder

	// Draw header (empty now - deck number moved to objectives panel)
	e.drawHeaderFromSnapshot(screen, &snap, screenWidth, headerHeight)

	// Draw map background
	vector.DrawFilledRect(screen, float32(mapX-10), float32(mapY-10),
		float32(mapAreaWidth+20), float32(mapAreaHeight+20),
		colorMapBackground, false)

	// Draw the map using snapshot for player position
	e.drawMap(screen, g, mapX, mapY, &snap)

	// Draw status bar (overlay on top left of map) - use snapshot data
	statusY := mapY + 10 // Small padding from top of map
	e.drawStatusBarFromSnapshot(screen, &snap, mapX+10, statusY, mapAreaWidth, statusBarHeight)

	// Draw messages panel as a bottom‑aligned overlay, limited to a few lines
	e.drawMessagesFromSnapshot(screen, &snap, screenWidth, screenHeight)

	// Draw bindings menu overlay if active (covers most of the screen on top of the map)
	e.menuMutex.RLock()
	menuActive := e.menuActive
	e.menuMutex.RUnlock()
	if menuActive {
		e.drawBindingsMenuOverlay(screen)
	}
	// Draw generic menu overlay if active
	e.genericMenuMutex.RLock()
	genericMenuActive := e.genericMenuActive
	e.genericMenuMutex.RUnlock()
	if genericMenuActive {
		e.drawGenericMenuOverlay(screen)
	}
}

// RenderBindingsMenu implements renderer.BindingsMenuRenderer for Ebiten.
// It captures the current frame and marks the menu overlay as active.
func (e *EbitenRenderer) RenderBindingsMenu(g *state.Game, actions []engineinput.Action, selected int, helpText string, nonRebindable map[engineinput.Action]bool) {
	// Keep the underlying game/map snapshot up to date
	e.RenderFrame(g)

	e.menuMutex.Lock()
	defer e.menuMutex.Unlock()
	e.menuActive = true
	e.menuSelected = selected
	e.menuHelpText = helpText
	e.menuActions = make([]engineinput.Action, len(actions))
	copy(e.menuActions, actions)
	e.menuNonRebindable = make(map[engineinput.Action]bool)
	for act, val := range nonRebindable {
		e.menuNonRebindable[act] = val
	}
}

// ClearBindingsMenu hides the bindings menu overlay.
func (e *EbitenRenderer) ClearBindingsMenu() {
	e.menuMutex.Lock()
	defer e.menuMutex.Unlock()
	e.menuActive = false
	e.menuActions = nil
	e.menuHelpText = ""
	e.menuNonRebindable = nil
}

// drawBindingsMenuOverlay draws a semi-transparent panel over most of the screen
// with the bindings list and a clear highlight for the selected entry.
func (e *EbitenRenderer) drawBindingsMenuOverlay(screen *ebiten.Image) {
	e.menuMutex.RLock()
	actions := make([]engineinput.Action, len(e.menuActions))
	copy(actions, e.menuActions)
	selected := e.menuSelected
	helpText := e.menuHelpText
	nonRebindable := make(map[engineinput.Action]bool)
	for act, val := range e.menuNonRebindable {
		nonRebindable[act] = val
	}
	e.menuMutex.RUnlock()

	if len(actions) == 0 {
		return
	}

	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Panel covers ~70% of screen, centered
	panelW := int(float32(screenWidth) * 0.7)
	panelH := int(float32(screenHeight) * 0.7)
	panelX := (screenWidth - panelW) / 2
	panelY := (screenHeight - panelH) / 2

	bg := color.RGBA{12, 12, 24, 230}
	border := color.RGBA{180, 180, 220, 255}

	// Border
	vector.DrawFilledRect(screen,
		float32(panelX-2), float32(panelY-2),
		float32(panelW+4), float32(panelH+4),
		border, false)
	// Background
	vector.DrawFilledRect(screen,
		float32(panelX), float32(panelY),
		float32(panelW), float32(panelH),
		bg, false)

	paddingX := 24
	paddingY := 24
	x := panelX + paddingX
	y := panelY + paddingY

	fontSize := e.getUIFontSize()
	lineHeight := int(fontSize) + 6

	// Use UI font metrics so the highlight rectangle can tightly wrap the text.
	face := e.getSansFontFace()
	_, textHeight := text.Measure("Ag", face, 0)

	// Title and help text
	e.drawColoredText(screen, "Bindings", x, y-int(fontSize), colorAction)
	y += lineHeight

	// Show version information
	versionText := fmt.Sprintf("Version: %s", renderer.Version)
	if renderer.Commit != "unknown" && len(renderer.Commit) > 0 {
		versionText += fmt.Sprintf(" (%s)", renderer.Commit[:7])
	}
	e.drawColoredText(screen, versionText, x, y-int(fontSize), colorSubtle)
	y += lineHeight

	// Show help text if provided (e.g., when editing a binding), otherwise show default instructions
	// Don't show "?: edit binding" hint if selected action is non-rebindable
	if helpText != "" {
		e.drawColoredText(screen, helpText, x, y-int(fontSize), colorAction)
	} else {
		selectedAction := engineinput.ActionNone
		if selected >= 0 && selected < len(actions) {
			selectedAction = actions[selected]
		}
		if nonRebindable[selectedAction] {
			e.drawColoredText(screen, "Up/Down: select    F10/Start or q: close", x, y-int(fontSize), colorSubtle)
		} else {
			e.drawColoredText(screen, "Up/Down: select    ?: edit binding    F10/Start or q: close", x, y-int(fontSize), colorSubtle)
		}
	}
	y += lineHeight * 2

	byAction := engineinput.GetBindingsByAction()

	// First pass: draw highlight rectangles (so they are always below text)
	for i := range actions {
		if i != selected {
			continue
		}

		// Match the vertical span of the text: from (baseline - textHeight) to baseline.
		rowParamY := y + i*lineHeight              // y passed into drawColoredText
		baselineY := float64(rowParamY) + fontSize // actual baseline y used by text.Draw
		rectTop := baselineY                       // top of glyph box
		rectHeight := textHeight + 4               // small padding below glyphs
		highlight := color.RGBA{40, 60, 120, 255}
		vector.DrawFilledRect(screen,
			float32(panelX+8), float32(rectTop),
			float32(panelW-16), float32(rectHeight),
			highlight, false)
	}

	// Second pass: draw action names and bindings on top of the highlights
	for i, act := range actions {
		name := engineinput.ActionName(act)
		codes := byAction[act]
		codeText := strings.Join(codes, ", ")
		if codeText == "" {
			codeText = "(unbound)"
		}
		// Add "(fixed)" indicator for non-rebindable actions
		if nonRebindable[act] {
			codeText += " (fixed)"
		}

		// Use a shared origin for text and rectangle calculations (see above).
		rowParamY := y + i*lineHeight

		// Use different color for non-rebindable actions
		nameColor := colorText
		if nonRebindable[act] {
			nameColor = colorSubtle // Use subtle color for non-rebindable actions
		}
		e.drawColoredText(screen, name, x, rowParamY, nameColor)
		//		codeX := x + int(e.getTextWidth(name)) + 32
		codeX := x + 200
		e.drawColoredText(screen, codeText, codeX, rowParamY, colorSubtle)
	}
}

// RenderMenu implements gamemenu.MenuRenderer for Ebiten.
// It captures the current frame and marks the menu overlay as active.
func (e *EbitenRenderer) RenderMenu(g *state.Game, items []gamemenu.MenuItem, selected int, helpText string, title string) {
	// Keep the underlying game/map snapshot up to date
	e.RenderFrame(g)

	e.genericMenuMutex.Lock()
	defer e.genericMenuMutex.Unlock()
	e.genericMenuActive = true
	e.genericMenuSelected = selected
	e.genericMenuHelpText = helpText
	e.genericMenuTitle = title
	e.genericMenuItems = make([]gamemenu.MenuItem, len(items))
	copy(e.genericMenuItems, items)
}

// ClearMenu hides the generic menu overlay.
func (e *EbitenRenderer) ClearMenu() {
	e.genericMenuMutex.Lock()
	defer e.genericMenuMutex.Unlock()
	e.genericMenuActive = false
	e.genericMenuItems = nil
	e.genericMenuHelpText = ""
	e.genericMenuTitle = ""
}

// drawGenericMenuOverlay draws a semi-transparent panel over most of the screen
// with the menu list and a clear highlight for the selected entry.
func (e *EbitenRenderer) drawGenericMenuOverlay(screen *ebiten.Image) {
	e.genericMenuMutex.RLock()
	items := make([]gamemenu.MenuItem, len(e.genericMenuItems))
	copy(items, e.genericMenuItems)
	selected := e.genericMenuSelected
	helpText := e.genericMenuHelpText
	title := e.genericMenuTitle
	e.genericMenuMutex.RUnlock()

	if len(items) == 0 {
		return
	}

	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Panel covers ~70% of screen, centered
	panelW := int(float32(screenWidth) * 0.7)
	panelH := int(float32(screenHeight) * 0.7)
	panelX := (screenWidth - panelW) / 2
	panelY := (screenHeight - panelH) / 2

	bg := color.RGBA{12, 12, 24, 230}
	border := color.RGBA{180, 180, 220, 255}

	// Border
	vector.DrawFilledRect(screen,
		float32(panelX-2), float32(panelY-2),
		float32(panelW+4), float32(panelH+4),
		border, false)
	// Background
	vector.DrawFilledRect(screen,
		float32(panelX), float32(panelY),
		float32(panelW), float32(panelH),
		bg, false)

	paddingX := 24
	paddingY := 24
	x := panelX + paddingX
	y := panelY + paddingY

	fontSize := e.getUIFontSize()
	lineHeight := int(fontSize) + 6

	// Use UI font metrics so the highlight rectangle can tightly wrap the text.
	face := e.getSansFontFace()
	_, textHeight := text.Measure("Ag", face, 0)

	// Title
	if title != "" {
		e.drawColoredText(screen, title, x, y-int(fontSize), colorAction)
		y += lineHeight
	}

	// Show version information
	versionText := fmt.Sprintf("Version: %s", renderer.Version)
	if renderer.Commit != "unknown" && len(renderer.Commit) > 0 {
		versionText += fmt.Sprintf(" (%s)", renderer.Commit[:7])
	}
	e.drawColoredText(screen, versionText, x, y-int(fontSize), colorSubtle)
	y += lineHeight

	// Show help text if provided (parse markup for proper colors)
	if helpText != "" {
		helpSegments := e.parseMarkup(helpText)
		if len(helpSegments) > 0 {
			e.drawColoredTextSegments(screen, helpSegments, x, y-int(fontSize))
		} else {
			e.drawColoredText(screen, helpText, x, y-int(fontSize), colorAction)
		}
		y += lineHeight
	}

	y += lineHeight

	// First pass: draw highlight rectangles (so they are always below text)
	for i := range items {
		if i != selected || !items[i].IsSelectable() {
			continue
		}

		// Match the vertical span of the text: from (baseline - textHeight) to baseline.
		rowParamY := y + i*lineHeight              // y passed into drawColoredText
		baselineY := float64(rowParamY) + fontSize // actual baseline y used by text.Draw
		rectTop := baselineY                       // top of glyph box
		rectHeight := textHeight + 4               // small padding below glyphs
		highlight := color.RGBA{40, 60, 120, 255}
		vector.DrawFilledRect(screen,
			float32(panelX+8), float32(rectTop),
			float32(panelW-16), float32(rectHeight),
			highlight, false)
	}

	// Second pass: draw menu items on top of the highlights
	for i, item := range items {
		label := item.GetLabel()

		// Use a shared origin for text and rectangle calculations (see above).
		rowParamY := y + i*lineHeight

		// Parse markup and draw with proper colors
		segments := e.parseMarkup(label)
		if len(segments) > 0 {
			e.drawColoredTextSegments(screen, segments, x, rowParamY)
		} else {
			// Fallback: use different color for non-selectable items
			labelColor := colorText
			if !item.IsSelectable() {
				labelColor = colorSubtle
			}
			e.drawColoredText(screen, label, x, rowParamY, labelColor)
		}
	}
}

// drawHeaderFromSnapshot draws the header (currently empty - deck number moved to objectives panel)
func (e *EbitenRenderer) drawHeaderFromSnapshot(screen *ebiten.Image, snap *renderSnapshot, screenWidth int, headerHeight int) {
	// Header is now empty - deck number has been moved to the objectives panel
}

// drawMap renders the game map
func (e *EbitenRenderer) drawMap(screen *ebiten.Image, g *state.Game, mapX, mapY int, snap *renderSnapshot) {
	if g.CurrentCell == nil || g.Grid == nil {
		return
	}

	// Use snapshot for player position to prevent jitter
	playerRow := snap.playerRow
	playerCol := snap.playerCol

	// Calculate viewport bounds centered on player
	startRow := playerRow - e.viewportRows/2
	startCol := playerCol - e.viewportCols/2

	// Render each tile in the viewport
	for vRow := 0; vRow < e.viewportRows; vRow++ {
		for vCol := 0; vCol < e.viewportCols; vCol++ {
			mapRow := startRow + vRow
			mapCol := startCol + vCol

			cell := g.Grid.GetCell(mapRow, mapCol)

			// Skip drawing player here - it will be drawn separately with debounce animation
			if cell != nil && cell.Row == snap.playerRow && cell.Col == snap.playerCol {
				// Draw background only, player icon will be drawn separately
				_, _, hasBg := e.getCellDisplay(g, cell, snap)
				if hasBg {
					x := mapX + vCol*e.tileSize
					y := mapY + vRow*e.tileSize
					e.drawTile(screen, " ", x, y, colorBackground, true)
				}
				continue
			}

			icon, col, hasBg := e.getCellDisplay(g, cell, snap)

			x := mapX + vCol*e.tileSize
			y := mapY + vRow*e.tileSize

			// Check if this is the focused cell (has active callout) or an interactable cell - use focus background
			var customBg color.Color
			isFocused := cell != nil && cell.Row == snap.focusedCellRow && cell.Col == snap.focusedCellCol
			isInteractable := false
			if cell != nil {
				for _, ic := range snap.interactableCells {
					if cell.Row == ic.row && cell.Col == ic.col {
						isInteractable = true
						break
					}
				}
			}

			// Check for blocking hazards or locked doors that need to be cleared - use brighter background
			needsClearing := false
			if cell != nil && (g.HasMap || cell.Discovered) {
				if gameworld.HasBlockingHazard(cell) || gameworld.HasLockedDoor(cell) {
					needsClearing = true
				}
			}

			// Check if this is a wall or corner in a powered room - use dark green background
			// This should persist even when focused/interactable
			isPoweredWall := false
			if cell != nil && !cell.Room {
				if e.roomHasPower(g, cell) {
					// For corners and walls (non-exit cells), always show powered color
					// For exit cells, only show powered color if locked or not all generators/hazards cleared
					if !needsClearing {
						if !cell.ExitCell {
							// Corners and walls always get powered background
							isPoweredWall = true
						} else if cell.Locked || !g.AllGeneratorsPowered() || !g.AllHazardsCleared() {
							// Exit cell only gets powered background if locked or not ready
							isPoweredWall = true
						}
					}
				}
			}

			// Check if this cell has a powered generator - use dark green background
			hasPoweredGenerator := false
			if cell != nil && (g.HasMap || cell.Discovered) {
				if gameworld.HasPoweredGenerator(cell) {
					hasPoweredGenerator = true
				}
			}

			// Set background color with priority order
			if needsClearing {
				// Brighter background to indicate this should be made passable
				customBg = colorBlockedBackground
			} else if isPoweredWall || hasPoweredGenerator {
				// Powered walls and powered generators always show green background, even when focused/interactable
				customBg = colorWallBgPowered
			} else if isFocused || isInteractable {
				// Focus/interactable color for non-powered walls
				customBg = colorFocusBackground
			} else if cell != nil && cell.ExitCell && (g.HasMap || cell.Discovered) && !cell.Locked && g.AllGeneratorsPowered() && g.AllHazardsCleared() {
				// Unlocked exit cell - use pulsing background (requires generators powered and hazards cleared)
				customBg = e.getPulsingExitBackgroundColor()
			}

			// Draw the tile character with optional background
			e.drawTileWithBg(screen, icon, x, y, col, hasBg, customBg)
		}
	}

	// Draw persistent room labels on top of the map
	e.drawRoomLabels(screen, snap, mapX, mapY, startRow, startCol)

	// Draw callouts on top of the map
	e.drawCallouts(screen, snap, mapX, mapY, startRow, startCol)

	// Draw player with debounce animation if active
	e.drawPlayerWithDebounce(screen, g, snap, mapX, mapY, startRow, startCol)

	// Draw exit animation overlay if active
	if snap.exitAnimating {
		e.drawExitAnimation(screen, snap, mapX, mapY, startRow, startCol)
	}
}

// drawPlayerWithDebounce draws the player icon with debounce animation if active
func (e *EbitenRenderer) drawPlayerWithDebounce(screen *ebiten.Image, g *state.Game, snap *renderSnapshot, mapX, mapY, startRow, startCol int) {
	e.debounceMutex.RLock()
	direction := e.debounceDirection
	startTime := e.debounceStartTime
	e.debounceMutex.RUnlock()

	// Calculate player position in viewport
	playerVRow := snap.playerRow - startRow
	playerVCol := snap.playerCol - startCol

	// Skip if player not in viewport
	if playerVRow < 0 || playerVRow >= e.viewportRows || playerVCol < 0 || playerVCol >= e.viewportCols {
		return
	}

	// Calculate base position
	baseX := mapX + playerVCol*e.tileSize
	baseY := mapY + playerVRow*e.tileSize

	// Calculate debounce offset
	offsetX := 0
	offsetY := 0
	if direction != "" {
		now := time.Now().UnixMilli()
		elapsed := now - startTime
		const debounceDuration = 150 // milliseconds

		if elapsed < debounceDuration {
			// Calculate bounce offset using a sine wave for smooth animation
			progress := float64(elapsed) / debounceDuration
			bounceAmount := math.Sin(progress*math.Pi) * 8.0 // Max 8 pixels offset

			switch direction {
			case "north":
				offsetY = int(-bounceAmount)
			case "south":
				offsetY = int(bounceAmount)
			case "east":
				offsetX = int(bounceAmount)
			case "west":
				offsetX = int(-bounceAmount)
			}
		} else {
			// Animation complete, clear it
			e.debounceMutex.Lock()
			e.debounceDirection = ""
			e.debounceMutex.Unlock()
		}
	}

	// Draw player icon at offset position
	playerX := baseX + offsetX
	playerY := baseY + offsetY
	e.drawTile(screen, PlayerIcon, playerX, playerY, colorPlayer, false)
}

// drawExitAnimation draws the exit transition animation with a meaningful message
func (e *EbitenRenderer) drawExitAnimation(screen *ebiten.Image, snap *renderSnapshot, mapX, mapY, startRow, startCol int) {
	now := time.Now().UnixMilli()
	elapsed := now - snap.exitAnimStartTime
	const exitAnimDuration = 2000 // 2 seconds for transition

	if elapsed >= exitAnimDuration {
		return // Animation complete
	}

	// Calculate fade progress (0.0 to 1.0)
	progress := float64(elapsed) / exitAnimDuration

	// Get screen dimensions - use actual screen bounds to ensure full coverage
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	if w == 0 || h == 0 {
		// Fallback to window size if screen bounds are invalid
		w, h = ebiten.WindowSize()
		if w == 0 || h == 0 {
			w, h = e.windowWidth, e.windowHeight
		}
	}

	// Phase 1: Fade to dark background (first 40% of animation)
	// Phase 2: Show message on dark background (middle 40%)
	// Phase 3: Fade message out (last 20%)
	var overlayAlpha float64
	var textAlpha float64
	var showText bool

	if progress < 0.4 {
		// Phase 1: Fade to dark background
		overlayAlpha = progress / 0.4
		textAlpha = 0
		showText = false
	} else if progress < 0.8 {
		// Phase 2: Show message on dark background
		overlayAlpha = 1.0
		textProgress := (progress - 0.4) / 0.4
		textAlpha = textProgress
		if textAlpha > 1.0 {
			textAlpha = 1.0
		}
		showText = true
	} else {
		// Phase 3: Fade message out, keep dark background
		overlayAlpha = 1.0
		fadeProgress := (progress - 0.8) / 0.2
		textAlpha = 1.0 - fadeProgress
		if textAlpha < 0 {
			textAlpha = 0
		}
		showText = textAlpha > 0
	}

	// Draw dark overlay matching the game's aesthetic
	overlayColor := color.RGBA{15, 15, 26, uint8(255 * overlayAlpha)} // Same as colorMapBackground
	vector.DrawFilledRect(screen, 0, 0, float32(w), float32(h), overlayColor, false)

	// Draw transition message
	if showText && textAlpha > 0 {
		message := fmt.Sprintf("DECK %d CLEARED", snap.level)
		subMessage := "Proceeding to next level..."

		// Get font size for UI (use larger size for transition screen)
		fontSize := e.getUIFontSize() * 1.5
		face := e.getSansFontFace()

		// Calculate text position (centered)
		messageWidth, _ := text.Measure(message, face, 0)
		subMessageWidth, _ := text.Measure(subMessage, face, 0)

		centerX := float64(w) / 2
		centerY := float64(h) / 2

		messageX := centerX - float64(messageWidth)/2
		messageY := centerY - fontSize/2

		subMessageX := centerX - float64(subMessageWidth)/2
		subMessageY := centerY + fontSize + 10

		// Draw main message with fade (using action color for emphasis)
		mainTextColor := color.RGBA{220, 170, 255, uint8(255 * textAlpha)} // colorAction
		op := &text.DrawOptions{}
		op.GeoM.Translate(messageX, messageY+fontSize)
		op.ColorScale.ScaleWithColor(mainTextColor)
		text.Draw(screen, message, face, op)

		// Draw sub message with fade (using text color)
		subTextColor := color.RGBA{240, 240, 255, uint8(255 * textAlpha)} // colorText
		op2 := &text.DrawOptions{}
		op2.GeoM.Translate(subMessageX, subMessageY+fontSize)
		op2.ColorScale.ScaleWithColor(subTextColor)
		text.Draw(screen, subMessage, face, op2)
	}
}

// drawRoomLabels renders persistent room name labels at the leftmost point of each room
func (e *EbitenRenderer) drawRoomLabels(screen *ebiten.Image, snap *renderSnapshot, mapX, mapY, startRow, startCol int) {
	if len(snap.roomLabels) == 0 {
		return
	}

	fontSize := e.getUIFontSize()

	for _, rl := range snap.roomLabels {
		// Check if the label position is visible in the viewport
		labelCol := rl.StartCol
		viewportStartCol := startCol
		viewportEndCol := startCol + e.viewportCols - 1

		// Skip if label is outside viewport
		if labelCol < viewportStartCol || labelCol > viewportEndCol {
			continue
		}

		// Convert to viewport coordinates
		vCol := labelCol - startCol
		vRow := (rl.Row - startRow) - 1

		// Skip if not in vertical range
		if vRow < 0 || vRow >= e.viewportRows {
			continue
		}

		// Compute pixel position (left edge of the cell where label should be)
		cellX := mapX + vCol*e.tileSize
		cellY := mapY + vRow*e.tileSize

		// Measure text
		textWidth := e.getTextWidth(rl.RoomName)

		// Draw background box for readability
		paddingX := 6
		paddingY := 4
		boxW := int(textWidth) + paddingX*2
		boxH := int(fontSize) + paddingY*2

		// Position box starting at the leftmost point of the room cell
		// Raise it by half its height so it sits just above the wall
		boxX := cellX + 2 // Small offset from left edge of cell
		boxY := cellY - boxH - 4 - boxH/2

		// Higher contrast colors for room labels
		bgColor := color.RGBA{15, 20, 40, 235}
		borderColor := color.RGBA{140, 140, 200, 255}

		// Border
		vector.DrawFilledRect(
			screen,
			float32(boxX-1), float32(boxY-1),
			float32(boxW+2), float32(boxH+2),
			borderColor,
			false,
		)

		// Background
		vector.DrawFilledRect(
			screen,
			float32(boxX), float32(boxY),
			float32(boxW), float32(boxH),
			bgColor,
			false,
		)

		// Position text: drawColoredText uses baseline positioning (adds fontSize to y)
		// Similar to callouts: subtract fontSize so baseline ends up inside the box
		textX := boxX + paddingX
		textY := boxY + paddingY - int(fontSize)

		// Draw bold-ish text by rendering twice with a slight offset
		labelColor := color.RGBA{230, 230, 255, 255} // High-contrast light color
		e.drawColoredText(screen, rl.RoomName, textX, textY, labelColor)
		e.drawColoredText(screen, rl.RoomName, textX+1, textY, labelColor)
	}
}

// drawCallouts renders floating message callouts near cells
func (e *EbitenRenderer) drawCallouts(screen *ebiten.Image, snap *renderSnapshot, mapX, mapY, startRow, startCol int) {
	if len(snap.callouts) == 0 {
		return
	}

	fontSize := e.getUIFontSize()
	padding := 6
	now := time.Now().UnixMilli()

	// Animation timing constants
	const (
		entranceDuration = 200 // milliseconds for entrance animation
		exitDuration     = 200 // milliseconds for exit animation
	)

	for _, callout := range snap.callouts {
		// Calculate screen position from cell position
		vRow := callout.Row - startRow
		vCol := callout.Col - startCol

		// Skip if outside viewport
		if vRow < 0 || vRow >= e.viewportRows || vCol < 0 || vCol >= e.viewportCols {
			continue
		}

		// Calculate animation progress
		age := now - callout.CreatedAt
		var alpha float64 = 1.0
		var slideOffsetY float32 = 0.0

		// Entrance animation (fade in from black + slide in from top)
		if age < entranceDuration {
			progress := float64(age) / entranceDuration
			alpha = progress                               // Fade in from 0 to 1
			slideOffsetY = float32(-20 * (1.0 - progress)) // Slide in from 20px above
		}

		// Exit animation (fade out to black + slide out to bottom)
		if callout.ExpiresAt > 0 {
			timeUntilExpiry := callout.ExpiresAt - now
			if timeUntilExpiry < exitDuration && timeUntilExpiry > 0 {
				progress := float64(timeUntilExpiry) / exitDuration
				alpha = progress                              // Fade out from 1 to 0
				slideOffsetY = float32(20 * (1.0 - progress)) // Slide out to 20px below
			} else if timeUntilExpiry <= 0 {
				continue // Skip expired callouts
			}
		}

		// Calculate pixel position (center of the cell)
		cellX := mapX + vCol*e.tileSize
		cellY := mapY + vRow*e.tileSize

		// Parse markup to get actual text segments (for width calculation)
		// Split message by newlines to handle multi-line callouts
		lines := strings.Split(callout.Message, "\n")
		maxTextWidth := 0.0
		for _, line := range lines {
			lineSegments := e.parseMarkup(line)
			lineWidth := 0.0
			for _, seg := range lineSegments {
				lineWidth += e.getTextWidth(seg.text)
			}
			if lineWidth > maxTextWidth {
				maxTextWidth = lineWidth
			}
		}
		textWidth := maxTextWidth
		boxHeight := (int(fontSize)+4)*len(lines) + padding*2 // Height for all lines

		// Determine base position (to the right or left of cell)
		boxWidth := int(textWidth) + padding*2
		baseCalloutX := cellX + e.tileSize + 8

		// If callout would go off right edge, position to the left instead
		if baseCalloutX+boxWidth > mapX+e.viewportCols*e.tileSize {
			baseCalloutX = cellX - boxWidth - 8
		}

		// Check if callout would overlap with player icon
		// Player position in viewport coordinates
		playerVRow := snap.playerRow - startRow
		playerVCol := snap.playerCol - startCol
		playerX := mapX + playerVCol*e.tileSize
		playerY := mapY + playerVRow*e.tileSize

		// Calculate callout box bounds
		baseCalloutY := cellY + (e.tileSize-boxHeight)/2
		calloutBoxLeft := float32(baseCalloutX)
		calloutBoxRight := float32(baseCalloutX + boxWidth)
		calloutBoxTop := float32(baseCalloutY)
		calloutBoxBottom := float32(baseCalloutY + boxHeight)

		// Check if callout overlaps with player icon (player icon is roughly centered in its tile)
		playerIconLeft := float32(playerX + e.tileSize/4)
		playerIconRight := float32(playerX + e.tileSize*3/4)
		playerIconTop := float32(playerY + e.tileSize/4)
		playerIconBottom := float32(playerY + e.tileSize*3/4)

		overlapsPlayer := calloutBoxLeft < playerIconRight && calloutBoxRight > playerIconLeft &&
			calloutBoxTop < playerIconBottom && calloutBoxBottom > playerIconTop

		// If overlapping, move callout back a column and down a row
		if overlapsPlayer {
			// Move back a column (left if on right side, right if on left side)
			if baseCalloutX > cellX+e.tileSize {
				// Callout is on the right, move it further right (back a column)
				//baseCalloutX = cellX + e.tileSize*2 + 8
				baseCalloutX = cellX // - e.tileSize*2 + 8
			} else {
				// Callout is on the left, move it further left (back a column)
				//baseCalloutX = cellX - boxWidth - e.tileSize - 8
				//baseCalloutX = cellX + e.tileSize*2 + 8
			}
			// Move down a row
			baseCalloutY = cellY + e.tileSize + (e.tileSize-boxHeight)/2
		}

		// Apply slide animation offset (vertical only)
		calloutX := float32(baseCalloutX)
		calloutY := float32(baseCalloutY) + slideOffsetY

		// Keep callout within vertical bounds (check after applying slide offset)
		if calloutY < float32(mapY) {
			calloutY = float32(mapY)
		}
		if calloutY+float32(boxHeight) > float32(mapY+e.viewportRows*e.tileSize) {
			calloutY = float32(mapY + e.viewportRows*e.tileSize - boxHeight)
		}

		// Skip drawing if alpha is too low (avoid rendering artifacts)
		if alpha < 0.01 {
			continue
		}

		// Apply alpha to colors (fade from black/transparent, not white)
		// The applyAlpha function multiplies the alpha channel, so colors fade to transparent black
		bgColor := e.applyAlpha(color.RGBA{15, 15, 25, 240}, alpha)
		borderColor := e.applyAlpha(color.RGBA{80, 80, 100, 255}, alpha)

		// Border
		vector.DrawFilledRect(screen,
			calloutX-1, calloutY-1,
			float32(boxWidth+2), float32(boxHeight+2),
			borderColor, false)

		// Background
		vector.DrawFilledRect(screen,
			calloutX, calloutY,
			float32(boxWidth), float32(boxHeight),
			bgColor, false)

		// Draw pointer/arrow toward the cell
		arrowSize := float32(6)
		arrowY := calloutY + float32(boxHeight/2)
		if calloutX > float32(cellX+e.tileSize) {
			// Arrow pointing left
			arrowX := calloutX - 1
			vector.DrawFilledRect(screen, arrowX-arrowSize, arrowY-2, arrowSize, 4, borderColor, false)
		} else {
			// Arrow pointing right
			arrowX := calloutX + float32(boxWidth) + 1
			vector.DrawFilledRect(screen, arrowX, arrowY-2, arrowSize, 4, borderColor, false)
		}

		// Draw text - position so baseline is vertically centered in box
		// drawColoredText adds fontSize to y for baseline, so we need to offset
		lineHeight := int(fontSize) + 4
		startY := int(calloutY) + padding - int(fontSize)

		// Draw each line of the callout
		for i, line := range lines {
			lineSegments := e.parseMarkup(line)
			// Apply alpha to all segment colors
			fadedSegments := make([]textSegment, len(lineSegments))
			for j, seg := range lineSegments {
				fadedSegments[j] = textSegment{
					text:  seg.text,
					color: e.applyAlpha(seg.color, alpha),
				}
			}
			textY := startY + i*lineHeight
			e.drawColoredTextSegments(screen, fadedSegments, int(calloutX)+padding, textY)
		}
	}
}

// drawTile draws a single tile at the given position
func (e *EbitenRenderer) drawTile(screen *ebiten.Image, icon string, x, y int, col color.Color, hasBackground bool) {
	e.drawTileWithBg(screen, icon, x, y, col, hasBackground, nil)
}

// drawTileWithBg draws a single tile with optional custom background color
func (e *EbitenRenderer) drawTileWithBg(screen *ebiten.Image, icon string, x, y int, col color.Color, hasBackground bool, bgColor color.Color) {
	// Skip completely empty/void tiles
	if icon == " " || icon == "" {
		return
	}

	// Convert color to RGBA
	r, g, b, a := col.RGBA()
	tileColor := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}

	// Skip if color is too dark (close to background)
	if tileColor.R < 30 && tileColor.G < 30 && tileColor.B < 30 {
		return
	}

	// Draw block background if requested (for walls, doors, etc.)
	if hasBackground {
		// Draw background with small margin inside the tile
		margin := float32(2)
		bgCol := colorWallBg
		if bgColor != nil {
			// Convert color.Color to color.RGBA
			r, g, b, a := bgColor.RGBA()
			bgCol = color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
		}
		vector.DrawFilledRect(screen, float32(x)+margin, float32(y)+margin,
			float32(e.tileSize)-margin*2, float32(e.tileSize)-margin*2,
			bgCol, false)
	}

	// Draw the colored character
	e.drawColoredChar(screen, icon, x, y, tileColor)
}

// getTileFontSize returns the font size for map tiles, scaled to the current tile size
func (e *EbitenRenderer) getTileFontSize() float64 {
	// Scale font size based on tile size (default tile size is 24)
	return baseFontSize * float64(e.tileSize) / 24.0
}

// getUIFontSize returns the font size for UI text (50% of tile size)
func (e *EbitenRenderer) getUIFontSize() float64 {
	size := e.getTileFontSize() * 0.5
	if size < 10 {
		size = 10
	}
	return size
}

// getMonoFontFace returns a cached monospace font face for map tiles
func (e *EbitenRenderer) getMonoFontFace() *text.GoTextFace {
	size := e.getTileFontSize()
	if e.cachedMonoFace == nil || e.cachedTileFontSize != size {
		e.cachedTileFontSize = size
		e.cachedMonoFace = &text.GoTextFace{
			Source: e.monoFontSource,
			Size:   size,
		}
	}
	return e.cachedMonoFace
}

// getSansFontFace returns a cached sans-serif font face for UI text
func (e *EbitenRenderer) getSansFontFace() *text.GoTextFace {
	size := e.getUIFontSize()
	if e.cachedSansFace == nil || e.cachedUIFontSize != size {
		e.cachedUIFontSize = size
		e.cachedSansFace = &text.GoTextFace{
			Source: e.sansFontSource,
			Size:   size,
		}
	}
	return e.cachedSansFace
}

// invalidateFontCache clears cached font faces (call when tile size changes)
func (e *EbitenRenderer) invalidateFontCache() {
	e.cachedMonoFace = nil
	e.cachedSansFace = nil
}

// drawColoredChar draws a character with color at the given tile position (uses mono font)
func (e *EbitenRenderer) drawColoredChar(screen *ebiten.Image, char string, x, y int, col color.Color) {
	face := e.getMonoFontFace()

	// Calculate position to center the character in the tile
	// text.Measure returns the bounding box width and height
	w, h := text.Measure(char, face, 0)

	// Center horizontally and vertically within the tile
	// text/v2 Draw uses top-left as the origin point
	offsetX := (float64(e.tileSize) - w) / 2
	offsetY := (float64(e.tileSize) - h) / 2

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x)+offsetX, float64(y)+offsetY)
	op.ColorScale.ScaleWithColor(col)

	text.Draw(screen, char, face, op)
}

// drawColoredText draws text with a specific color using sans-serif font for UI
func (e *EbitenRenderer) drawColoredText(screen *ebiten.Image, str string, x, y int, col color.Color) {
	face := e.getSansFontFace()
	fontSize := e.getUIFontSize()

	op := &text.DrawOptions{}
	// Adjust y position since text.Draw uses baseline
	op.GeoM.Translate(float64(x), float64(y)+fontSize)
	op.ColorScale.ScaleWithColor(col)

	text.Draw(screen, str, face, op)
}

// textSegment represents a segment of text with a specific color
type textSegment struct {
	text  string
	color color.Color
}

// parseMarkup parses a message string with markup (ITEM{}, ROOM{}, ACTION{}, GT{}) and returns colored segments
func (e *EbitenRenderer) parseMarkup(msg string) []textSegment {
	var segments []textSegment
	// Regex to match markup: FUNCTION{content}
	markupRegex := regexp.MustCompile(`([A-Z]+)\{([^}]*)\}`)

	lastIndex := 0
	matches := markupRegex.FindAllStringSubmatchIndex(msg, -1)

	for _, match := range matches {
		// Add text before the markup
		if match[0] > lastIndex {
			plainText := msg[lastIndex:match[0]]
			if plainText != "" {
				segments = append(segments, textSegment{text: plainText, color: colorText})
			}
		}

		// Extract function name and content
		function := msg[match[2]:match[3]]
		content := msg[match[4]:match[5]]

		var segColor color.Color
		switch function {
		case "ITEM":
			segColor = colorItem
		case "ROOM":
			segColor = colorFloorVisited // Light gray-blue for room names
		case "ACTION":
			segColor = colorAction
		case "EXIT":
			segColor = colorExitUnlocked // Bright green for exit/lift
		case "GT":
			// GT{} is for translations - look up the translation
			content = gotext.Get(content)
			segColor = colorText
		case "FURNITURE":
			// FURNITURE{} uses the furniture callout color (tan/brown for checked furniture)
			segColor = renderer.CalloutColorFurnitureChecked
		case "HAZARD":
			// HAZARD{} uses the hazard color (red)
			segColor = colorHazard
		default:
			segColor = colorText
		}

		segments = append(segments, textSegment{text: content, color: segColor})
		lastIndex = match[1]
	}

	// Add remaining text after last markup
	if lastIndex < len(msg) {
		plainText := msg[lastIndex:]
		if plainText != "" {
			segments = append(segments, textSegment{text: plainText, color: colorText})
		}
	}

	// If no markup found, return the whole message as a single segment
	if len(segments) == 0 {
		segments = append(segments, textSegment{text: msg, color: colorText})
	}

	return segments
}

// applyAlpha applies an alpha value to a color
func (e *EbitenRenderer) applyAlpha(c color.Color, alpha float64) color.Color {
	if alpha <= 0 {
		alpha = 0
	}
	if alpha > 1.0 {
		alpha = 1.0
	}

	r, g, b, a := c.RGBA()
	// RGBA returns values in 0-65535 range, convert to 0-255
	r8 := uint8(r >> 8)
	g8 := uint8(g >> 8)
	b8 := uint8(b >> 8)
	a8 := uint8(a >> 8)

	// Apply alpha to both RGB and alpha channel for proper fade from black
	// This ensures colors fade to transparent black, not transparent bright colors
	newR := uint8(float64(r8) * alpha)
	newG := uint8(float64(g8) * alpha)
	newB := uint8(float64(b8) * alpha)
	newAlpha := uint8(float64(a8) * alpha)

	return color.RGBA{newR, newG, newB, newAlpha}
}

// drawColoredTextSegments draws multiple text segments with different colors
func (e *EbitenRenderer) drawColoredTextSegments(screen *ebiten.Image, segments []textSegment, x, y int) {
	face := e.getSansFontFace()
	fontSize := e.getUIFontSize()
	currentX := float64(x)

	for _, seg := range segments {
		if seg.text == "" {
			continue
		}

		op := &text.DrawOptions{}
		op.GeoM.Translate(currentX, float64(y)+fontSize)
		op.ColorScale.ScaleWithColor(seg.color)

		text.Draw(screen, seg.text, face, op)

		// Move x position for next segment
		w, _ := text.Measure(seg.text, face, 0)
		currentX += w
	}
}

// getTextWidth returns the width of a string in pixels at UI font size
func (e *EbitenRenderer) getTextWidth(str string) float64 {
	face := e.getSansFontFace()
	w, _ := text.Measure(str, face, 0)
	return w
}

// getPulsingExitColor returns a pulsing color for the unlocked exit icon
// Uses a sine wave to create a smooth pulsing effect
func (e *EbitenRenderer) getPulsingExitColor() color.Color {
	// Pulse period: 2 seconds (2000ms)
	const pulsePeriod = 2000.0
	now := time.Now().UnixMilli()

	// Calculate pulse value (0.0 to 1.0) using sine wave
	// This creates a smooth oscillation
	pulsePhase := float64(now%int64(pulsePeriod)) / pulsePeriod
	pulseValue := (math.Sin(pulsePhase*2*math.Pi) + 1.0) / 2.0 // 0.0 to 1.0

	// Pulse between 50% and 100% brightness
	minBrightness := 0.5
	maxBrightness := 1.0
	brightness := minBrightness + (maxBrightness-minBrightness)*pulseValue

	// Apply brightness to the base exit unlocked color (bright green)
	baseR, baseG, baseB, baseA := colorExitUnlocked.RGBA()
	r8 := uint8(float64(baseR>>8) * brightness)
	g8 := uint8(float64(baseG>>8) * brightness)
	b8 := uint8(float64(baseB>>8) * brightness)
	a8 := uint8(baseA >> 8)

	return color.RGBA{r8, g8, b8, a8}
}

// getPulsingExitBackgroundColor returns a pulsing background color for the unlocked exit
// Uses a distinct color (cyan/blue) that pulses
func (e *EbitenRenderer) getPulsingExitBackgroundColor() color.Color {
	// Pulse period: 2 seconds (2000ms)
	const pulsePeriod = 2000.0
	now := time.Now().UnixMilli()

	// Calculate pulse value (0.0 to 1.0) using sine wave
	pulsePhase := float64(now%int64(pulsePeriod)) / pulsePeriod
	pulseValue := (math.Sin(pulsePhase*2*math.Pi) + 1.0) / 2.0 // 0.0 to 1.0

	// Pulse between 30% and 70% brightness for background (distinct from icon)
	minBrightness := 0.3
	maxBrightness := 0.7
	brightness := minBrightness + (maxBrightness-minBrightness)*pulseValue

	// Use a distinct cyan/blue color for the background
	baseColor := color.RGBA{50, 150, 255, 255} // Cyan-blue
	r8 := uint8(float64(baseColor.R) * brightness)
	g8 := uint8(float64(baseColor.G) * brightness)
	b8 := uint8(float64(baseColor.B) * brightness)

	return color.RGBA{r8, g8, b8, baseColor.A}
}

// getCellDisplay returns the icon, color, and whether to draw a background for a cell
func (e *EbitenRenderer) getCellDisplay(g *state.Game, r *world.Cell, snap *renderSnapshot) (string, color.Color, bool) {
	if r == nil {
		return IconVoid, colorBackground, false
	}

	// Player position - use snapshot coordinates for consistency
	if r.Row == snap.playerRow && r.Col == snap.playerCol {
		return PlayerIcon, colorPlayer, false
	}

	// Get game-specific data for this cell
	data := gameworld.GetGameData(r)

	// Hazard (show if has map or discovered)
	if gameworld.HasHazard(r) && (g.HasMap || r.Discovered) {
		if data.Hazard.IsBlocking() {
			return data.Hazard.GetIcon(), colorHazard, true
		}
	}

	// Hazard Control (show if has map or discovered)
	if gameworld.HasHazardControl(r) && (g.HasMap || r.Discovered) {
		if !data.HazardControl.Activated {
			return entities.GetControlIcon(data.HazardControl.Type), colorHazardCtrl, true
		}
		return entities.GetControlIcon(data.HazardControl.Type), colorSubtle, false
	}

	// Door (show if has map or discovered)
	if gameworld.HasDoor(r) && (g.HasMap || r.Discovered) {
		if data.Door.Locked {
			return IconDoorLocked, colorDoorLocked, true
		}
		return IconDoorUnlocked, colorDoorUnlocked, true
	}

	// Generator (show if has map or discovered)
	if gameworld.HasGenerator(r) && (g.HasMap || r.Discovered) {
		if data.Generator.IsPowered() {
			return IconGeneratorPowered, colorGeneratorOn, true
		}
		return IconGeneratorUnpowered, colorGeneratorOff, true
	}

	// Maintenance Terminal (always visible, distinctive orange color) - high priority
	if gameworld.HasMaintenanceTerminal(r) {
		return IconMaintenance, colorMaintenance, true
	}

	// CCTV Terminal (show if has map or discovered)
	if gameworld.HasTerminal(r) && (g.HasMap || r.Discovered) {
		if data.Terminal.IsUsed() {
			return IconTerminalUsed, colorTerminalUsed, false
		}
		return IconTerminalUnused, colorTerminal, true
	}

	// Puzzle Terminal (show if has map or discovered)
	if gameworld.HasPuzzle(r) && (g.HasMap || r.Discovered) {
		if data.Puzzle.IsSolved() {
			return IconTerminalUsed, colorTerminalUsed, false
		}
		return IconTerminalUnused, colorTerminal, true
	}

	// Furniture (show if has map or discovered)
	if gameworld.HasFurniture(r) && (g.HasMap || r.Discovered) {
		if data.Furniture.IsChecked() {
			return data.Furniture.Icon, colorFurnitureCheck, false
		}
		return data.Furniture.Icon, colorFurniture, true
	}

	// Exit cell (show if has map or discovered)
	if r.ExitCell && (g.HasMap || r.Discovered) {
		if r.Locked && !g.AllGeneratorsPowered() {
			return IconExitLocked, colorExitLocked, true
		}
		// Unlocked exit - apply continuous pulsing animation for icon
		pulseColor := e.getPulsingExitColor()
		// Background will be drawn with pulsing color separately
		return IconExitUnlocked, pulseColor, true
	}

	// Items on floor (show if has map or discovered)
	if r.ItemsOnFloor.Size() > 0 && (g.HasMap || r.Discovered) {
		if cellHasKeycard(r) {
			return IconKey, colorKeycard, true
		}
		if cellHasBattery(r) {
			return IconBattery, colorBattery, true
		}
		return IconItem, colorItem, true
	}

	// Visited rooms
	if r.Visited {
		return getFloorIcon(r.Name, true), colorFloorVisited, false
	}

	// Discovered but not visited
	if r.Discovered {
		if r.Room {
			return getFloorIcon(r.Name, false), colorFloor, false
		}
		return IconWall, colorWall, true // Walls get background
	}

	// Has map - show rooms faintly
	if g.HasMap && r.Room {
		return getFloorIcon(r.Name, false), colorSubtle, false
	}

	// Non-room cells adjacent to discovered/visited rooms render as walls
	if !r.Room && hasAdjacentDiscoveredRoom(r) {
		return IconWall, colorWall, true // Walls get background
	}

	// Unknown/void
	return IconVoid, colorBackground, false
}

// getFloorIcon returns the appropriate floor icon for a room
func getFloorIcon(roomName string, visited bool) string {
	for baseRoom, icons := range roomFloorIcons {
		if strings.Contains(roomName, baseRoom) {
			if visited {
				return icons[0]
			}
			return icons[1]
		}
	}
	if visited {
		return IconVisited
	}
	return IconUnvisited
}

// cellHasKeycard checks if a cell has a keycard item
func cellHasKeycard(c *world.Cell) bool {
	hasKeycard := false
	c.ItemsOnFloor.Each(func(item *world.Item) {
		if strings.Contains(strings.ToLower(item.Name), "keycard") {
			hasKeycard = true
		}
	})
	return hasKeycard
}

// cellHasBattery checks if a cell has a battery item
func cellHasBattery(c *world.Cell) bool {
	hasBattery := false
	c.ItemsOnFloor.Each(func(item *world.Item) {
		if strings.Contains(strings.ToLower(item.Name), "battery") {
			hasBattery = true
		}
	})
	return hasBattery
}

// hasAdjacentDiscoveredRoom checks if any adjacent cell is discovered
func hasAdjacentDiscoveredRoom(c *world.Cell) bool {
	neighbors := []*world.Cell{c.North, c.East, c.South, c.West}
	for _, n := range neighbors {
		if n != nil && n.Room && (n.Discovered || n.Visited) {
			return true
		}
	}
	return false
}

// roomHasPower checks if the room adjacent to a wall cell has power
func (e *EbitenRenderer) roomHasPower(g *state.Game, wallCell *world.Cell) bool {
	if wallCell == nil || g == nil || g.Grid == nil {
		return false
	}

	// Check if there's available power (power supply > consumption)
	availablePower := g.GetAvailablePower()
	if availablePower <= 0 {
		return false
	}

	// If there's available power, check if any adjacent room exists
	// (if there's power, rooms should be considered powered)
	neighbors := []*world.Cell{wallCell.North, wallCell.East, wallCell.South, wallCell.West}
	for _, neighbor := range neighbors {
		if neighbor != nil && neighbor.Room {
			// Room exists and there's available power - room is powered
			return true
		}
	}
	return false
}

// drawDirectionLabels draws N/S/E/W labels around the map
func (e *EbitenRenderer) drawDirectionLabels(screen *ebiten.Image, g *state.Game, mapX, mapY, mapWidth, mapHeight int) {
	if g.CurrentCell == nil {
		return
	}

	fontSize := e.getUIFontSize()

	// North label (above map, centered)
	northText := e.getDirectionText(g, g.CurrentCell.North, "North")
	northWidth := e.getTextWidth(northText)
	northX := mapX + mapWidth/2 - int(northWidth)/2
	e.drawColoredText(screen, northText, northX, mapY-int(fontSize)-15, colorText)

	// South label (below map, centered)
	southText := e.getDirectionText(g, g.CurrentCell.South, "South")
	southWidth := e.getTextWidth(southText)
	southX := mapX + mapWidth/2 - int(southWidth)/2
	e.drawColoredText(screen, southText, southX, mapY+mapHeight+10, colorText)

	// West label (left of map, vertically centered)
	westText := e.getDirectionText(g, g.CurrentCell.West, "West")
	westWidth := e.getTextWidth(westText)
	e.drawColoredText(screen, westText, mapX-int(westWidth)-20, mapY+mapHeight/2-int(fontSize)/2, colorText)

	// East label (right of map, vertically centered)
	eastText := e.getDirectionText(g, g.CurrentCell.East, "East")
	e.drawColoredText(screen, eastText, mapX+mapWidth+20, mapY+mapHeight/2-int(fontSize)/2, colorText)
}

// getDirectionText returns the text for a direction label
func (e *EbitenRenderer) getDirectionText(g *state.Game, cell *world.Cell, direction string) string {
	if cell == nil || !cell.Room {
		return "# Wall #"
	}

	// Check if blocked
	if gameworld.HasLockedDoor(cell) {
		data := gameworld.GetGameData(cell)
		return fmt.Sprintf("%s (need %s)", direction, data.Door.KeycardName())
	}

	if gameworld.HasBlockingHazard(cell) {
		return fmt.Sprintf("%s (blocked)", direction)
	}

	return direction
}

// drawStatusBarFromSnapshot draws the inventory and generator status using snapshot data
func (e *EbitenRenderer) drawStatusBarFromSnapshot(screen *ebiten.Image, snap *renderSnapshot, x, y, width, height int) {
	// Check if there's anything to show
	hasObjectives := len(snap.objectives) > 0
	hasInventory := len(snap.ownedItems) > 0 || snap.batteries > 0
	hasGenerators := len(snap.generators) > 0

	// Always show at least the deck number
	hasDeckNumber := true

	// Don't draw anything if everything is empty (but we always have deck number)
	if !hasDeckNumber && !hasObjectives && !hasInventory && !hasGenerators {
		return
	}

	fontSize := e.getUIFontSize()
	lineHeight := int(fontSize) + 4

	// Calculate how many lines we need (always include deck number)
	linesNeeded := 0
	if hasDeckNumber {
		linesNeeded++ // Deck number is always first
	}
	if hasObjectives {
		linesNeeded += len(snap.objectives)
	}
	if hasInventory {
		linesNeeded++
	}
	if hasGenerators {
		linesNeeded++
	}

	// Calculate the maximum width needed for all text lines
	maxTextWidth := 0.0
	// Deck number text
	deckText := fmt.Sprintf("Deck %d", snap.level)
	deckWidth := e.getTextWidth(deckText)
	if deckWidth > maxTextWidth {
		maxTextWidth = deckWidth
	}
	if hasObjectives {
		for _, objective := range snap.objectives {
			w := e.getTextWidth(objective)
			if w > maxTextWidth {
				maxTextWidth = w
			}
		}
	}
	if hasInventory {
		// Build inventory text with markup for width calculation (same format as rendering)
		invParts := []string{"Inventory:"}
		for i, itemName := range snap.ownedItems {
			if i > 0 {
				invParts = append(invParts, ",")
			}
			invParts = append(invParts, fmt.Sprintf("ITEM{%s}", itemName))
		}
		if snap.batteries > 0 {
			if len(snap.ownedItems) > 0 {
				invParts = append(invParts, ",")
			}
			invParts = append(invParts, fmt.Sprintf("ACTION{Batteries x%d}", snap.batteries))
		}
		invText := strings.Join(invParts, " ")
		// Calculate width using parsed segments (actual text width, not markup)
		segments := e.parseMarkup(invText)
		textWidth := 0.0
		for _, seg := range segments {
			textWidth += e.getTextWidth(seg.text)
		}
		if textWidth > maxTextWidth {
			maxTextWidth = textWidth
		}
	}
	if hasGenerators {
		genText := "Generators: "
		genParts := []string{}
		for i, gen := range snap.generators {
			if gen.powered {
				genParts = append(genParts, fmt.Sprintf("#%d POWERED", i+1))
			} else {
				genParts = append(genParts, fmt.Sprintf("#%d %d/%d", i+1, gen.batteriesInserted, gen.batteriesRequired))
			}
		}
		genText += strings.Join(genParts, ", ")
		w := e.getTextWidth(genText)
		if w > maxTextWidth {
			maxTextWidth = w
		}
	}

	// Adjust panel height based on actual content
	panelHeight := lineHeight*linesNeeded + 10
	if panelHeight < int(fontSize)+10 {
		panelHeight = int(fontSize) + 10
	}

	// Calculate panel width based on widest text, with padding
	panelWidth := int(maxTextWidth) + 20 // 10px padding on each side
	if panelWidth < 100 {
		panelWidth = 100 // Minimum width
	}

	// Draw panel background with border (more opaque for overlay on map)
	bgX := float32(x - 10)
	bgY := float32(y - 5)
	bgW := float32(panelWidth)
	bgH := float32(panelHeight)
	borderColor := color.RGBA{80, 80, 100, 255}
	// More opaque background for overlay readability
	overlayBackground := color.RGBA{20, 20, 35, 250} // More opaque than colorPanelBackground

	// Border
	vector.DrawFilledRect(screen, bgX-1, bgY-1, bgW+2, bgH+2, borderColor, false)
	// Background
	vector.DrawFilledRect(screen, bgX, bgY, bgW, bgH, overlayBackground, false)

	// Calculate vertical center for first line
	// Since drawColoredText adds fontSize for baseline, we need to adjust
	firstLineY := y + (panelHeight / 2) - (lineHeight * linesNeeded / 2) - int(fontSize)

	currentY := firstLineY

	// Deck number (always first line)
	if hasDeckNumber {
		deckText := fmt.Sprintf("Deck %d", snap.level)
		e.drawColoredText(screen, deckText, x, currentY, colorAction)
		currentY += lineHeight
		// Add a small gap between deck number and objectives
		if hasObjectives {
			currentY += 2
		}
	}

	// Objectives (displayed after deck number)
	if hasObjectives {
		for _, objective := range snap.objectives {
			// Parse markup to properly color ACTION{} segments
			segments := e.parseMarkup(objective)
			e.drawColoredTextSegments(screen, segments, x, currentY)
			currentY += lineHeight
		}
		// Add a small gap between objectives and inventory
		if hasInventory || hasGenerators {
			currentY += 2
		}
	}

	// Inventory line (only if not empty)
	if hasInventory {
		// Build inventory text with item colors using markup, commas in default color
		invParts := []string{"Inventory:"}
		for i, itemName := range snap.ownedItems {
			if i > 0 {
				invParts = append(invParts, ",") // Comma in default text color
			}
			invParts = append(invParts, fmt.Sprintf("ITEM{%s}", itemName))
		}
		if snap.batteries > 0 {
			if len(snap.ownedItems) > 0 {
				invParts = append(invParts, ",") // Comma in default text color
			}
			invParts = append(invParts, fmt.Sprintf("ACTION{Batteries x%d}", snap.batteries))
		}
		invText := strings.Join(invParts, " ")

		// Parse markup to apply item colors (commas will be in default color)
		segments := e.parseMarkup(invText)
		e.drawColoredTextSegments(screen, segments, x, currentY)
		currentY += lineHeight
	}

	// Generator status (if applicable)
	if hasGenerators {
		genText := "Generators: "
		genParts := []string{}
		for i, gen := range snap.generators {
			if gen.powered {
				genParts = append(genParts, fmt.Sprintf("#%d POWERED", i+1))
			} else {
				genParts = append(genParts, fmt.Sprintf("#%d %d/%d", i+1, gen.batteriesInserted, gen.batteriesRequired))
			}
		}
		genText += strings.Join(genParts, ", ")
		e.drawColoredText(screen, genText, x, currentY, colorText)
	}
}

// drawMessagesFromSnapshot draws the messages panel as a bottom‑aligned overlay using snapshot data.
// The background panel is only drawn when there are visible (non‑expired) messages and shows at most 4 lines.
func (e *EbitenRenderer) drawMessagesFromSnapshot(screen *ebiten.Image, snap *renderSnapshot, screenWidth, screenHeight int) {
	const maxVisibleLines = 4

	fontSize := e.getUIFontSize()
	lineHeight := int(fontSize) + 4 // Font size plus padding for proper line spacing

	if len(snap.messages) == 0 {
		// No messages to show, so don't draw any panel background
		return
	}

	now := time.Now().UnixMilli()
	const messageLifetime = 10000 // 10 seconds in milliseconds

	// Collect visible messages (messages are already sorted chronologically in snapshot)
	type visibleMessage struct {
		segments []textSegment
	}
	visible := make([]visibleMessage, 0, maxVisibleLines)

	// Iterate through messages in chronological order (oldest first)
	// Take the last maxVisibleLines messages (most recent)
	startIdx := len(snap.messages) - maxVisibleLines
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(snap.messages) && len(visible) < maxVisibleLines; i++ {
		msgEntry := snap.messages[i]
		age := now - msgEntry.Timestamp
		if age >= messageLifetime {
			continue // Skip fully faded/expired messages (shouldn't happen, but double-check)
		}

		// Calculate alpha: 1.0 at start, 0.0 at messageLifetime
		// Fade starts at 7 seconds (70% of lifetime), fully transparent at 10 seconds
		fadeStart := int64(messageLifetime * 7 / 10) // Start fading at 7 seconds
		alpha := 1.0
		if age > fadeStart {
			// Fade from 1.0 to 0.0 over the last 3 seconds
			fadeProgress := float64(age-fadeStart) / float64(messageLifetime-fadeStart)
			alpha = 1.0 - fadeProgress
			if alpha < 0 {
				alpha = 0
			}
		}

		// Parse markup and apply alpha to segment colors
		segments := e.parseMarkup(msgEntry.Text)
		fadedSegments := make([]textSegment, len(segments))
		for j, seg := range segments {
			fadedSegments[j] = textSegment{
				text:  seg.text,
				color: e.applyAlpha(seg.color, alpha),
			}
		}

		visible = append(visible, visibleMessage{segments: fadedSegments})
	}

	// If no messages are actually visible after fading, don't draw anything
	if len(visible) == 0 {
		return
	}

	// Calculate the maximum width needed for all messages
	maxTextWidth := 0.0
	// Include header width
	headerText := "─── Messages ───"
	headerWidth := e.getTextWidth(headerText)
	if headerWidth > maxTextWidth {
		maxTextWidth = headerWidth
	}
	// Calculate width for each visible message (sum of all segments)
	for _, vm := range visible {
		msgWidth := 0.0
		for _, seg := range vm.segments {
			msgWidth += e.getTextWidth(seg.text)
		}
		if msgWidth > maxTextWidth {
			maxTextWidth = msgWidth
		}
	}

	// Calculate dynamic panel height based on number of visible messages
	headerHeight := int(fontSize) + 8
	bodyHeight := len(visible) * lineHeight
	panelHeight := headerHeight + bodyHeight + 10 // Extra padding

	// Calculate panel width based on widest text, with padding
	panelWidth := int(maxTextWidth) + 20 // 10px padding on each side
	if panelWidth < 100 {
		panelWidth = 100 // Minimum width
	}
	// Don't exceed screen width
	if panelWidth > screenWidth-40 {
		panelWidth = screenWidth - 40
	}

	// Position panel aligned to the bottom of the window, centered horizontally
	marginBottom := 20
	bgX := float32((screenWidth - panelWidth) / 2)
	bgY := float32(screenHeight - marginBottom - panelHeight)
	if bgY < 0 {
		bgY = 0
	}
	bgW := float32(panelWidth)
	bgH := float32(panelHeight)

	borderColor := color.RGBA{80, 80, 100, 255}

	// Border
	vector.DrawFilledRect(screen, bgX-1, bgY-1, bgW+2, bgH+2, borderColor, false)
	// Background
	vector.DrawFilledRect(screen, bgX, bgY, bgW, bgH, colorPanelBackground, false)

	// Header - position at top with proper padding (centered in panel)
	x := int(bgX) + 10
	headerY := int(bgY) + 8 - int(fontSize) // Small padding from top, account for baseline
	e.drawColoredText(screen, headerText, x, headerY, colorSubtle)

	// Messages - start below header with proper spacing
	messageStartY := int(bgY) + headerHeight + 4
	for i, vm := range visible {
		msgY := messageStartY + i*lineHeight - int(fontSize)
		e.drawColoredTextSegments(screen, vm.segments, x, msgY)
	}
}

// Layout returns the game's logical screen size (Ebiten interface)
func (e *EbitenRenderer) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Recalculate viewport when window size changes
	if outsideWidth != e.windowWidth || outsideHeight != e.windowHeight {
		e.windowWidth = outsideWidth
		e.windowHeight = outsideHeight
		e.recalculateViewport()
	}
	return outsideWidth, outsideHeight
}

// Run starts the Ebiten game loop
func (e *EbitenRenderer) Run() error {
	e.running = true
	return ebiten.RunGame(e)
}

// IsRunning returns whether the renderer is running
func (e *EbitenRenderer) IsRunning() bool {
	return e.running
}

// RunWithGameLoop starts the Ebiten game loop in a goroutine and returns
// This allows the main game loop to continue running
func (e *EbitenRenderer) RunWithGameLoop(gameLoop func()) error {
	// Start the game loop in a goroutine
	go gameLoop()

	// Run Ebiten (this blocks until the window is closed)
	return e.Run()
}
