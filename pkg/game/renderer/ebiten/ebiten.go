// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
// Ebiten is a 2D game library for Go: https://ebiten.org/
package ebiten

import (
	"bytes"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/config"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// Color palette for the game - brighter colors for visibility
var (
	colorBackground      = color.RGBA{26, 26, 46, 255}    // Dark blue-gray
	colorMapBackground   = color.RGBA{15, 15, 26, 255}    // Darker for map area
	colorPlayer          = color.RGBA{0, 255, 0, 255}     // Bright green
	colorWall            = color.RGBA{180, 180, 200, 255} // Light gray-blue for wall text
	colorWallBg          = color.RGBA{60, 60, 80, 255}    // Darker background for walls
	colorFloor           = color.RGBA{100, 100, 120, 255} // Medium gray for undiscovered
	colorFloorVisited    = color.RGBA{160, 160, 180, 255} // Lighter gray for visited
	colorDoorLocked      = color.RGBA{255, 255, 0, 255}   // Bright yellow
	colorDoorUnlocked    = color.RGBA{0, 220, 0, 255}     // Bright green
	colorKeycard         = color.RGBA{100, 150, 255, 255} // Bright blue
	colorItem            = color.RGBA{220, 170, 255, 255} // Bright purple
	colorBattery         = color.RGBA{255, 200, 100, 255} // Orange for batteries
	colorHazard          = color.RGBA{255, 80, 80, 255}   // Bright red
	colorHazardCtrl      = color.RGBA{0, 255, 255, 255}   // Bright cyan
	colorGeneratorOff    = color.RGBA{255, 100, 100, 255} // Bright red
	colorGeneratorOn     = color.RGBA{0, 255, 100, 255}   // Bright green
	colorTerminal        = color.RGBA{100, 150, 255, 255} // Bright blue
	colorTerminalUsed    = color.RGBA{120, 120, 140, 255} // Medium gray
	colorFurniture       = color.RGBA{255, 150, 255, 255} // Bright pink
	colorFurnitureCheck  = color.RGBA{200, 180, 100, 255} // Tan/brown
	colorExitLocked      = color.RGBA{255, 100, 100, 255} // Bright red
	colorExitUnlocked    = color.RGBA{100, 255, 100, 255} // Bright green
	colorSubtle          = color.RGBA{120, 120, 140, 255} // Medium gray
	colorText            = color.RGBA{240, 240, 255, 255} // Bright off-white
	colorAction          = color.RGBA{220, 170, 255, 255} // Bright purple
	colorDenied          = color.RGBA{255, 100, 100, 255} // Bright red
	colorPanelBackground = color.RGBA{30, 30, 50, 220}    // Semi-transparent dark

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
}

// roomLabel represents a persistent label for a room, drawn above its longest horizontal wall
type roomLabel struct {
	RoomName string
	Row      int // Grid row of the wall (room interior row)
	StartCol int // Inclusive grid column index
	EndCol   int // Inclusive grid column index
}

// renderSnapshot holds a consistent snapshot of game state for rendering
// This prevents jitter from race conditions between game logic and rendering
type renderSnapshot struct {
	valid      bool
	level      int
	playerRow  int
	playerCol  int
	cellName   string
	hasMap     bool
	batteries  int
	messages   []string
	ownedItems []string
	generators []generatorState
	gridRows   int
	gridCols   int
	callouts   []Callout
	roomLabels []roomLabel
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
	inputChan chan string

	// Flag indicating renderer is running
	running bool

	// Messages to display
	messages []string
}

// New creates a new Ebiten renderer
func New() *EbitenRenderer {
	return &EbitenRenderer{
		windowWidth:  1024,
		windowHeight: 768,
		tileSize:     24,
		viewportRows: 21,
		viewportCols: 35,
		inputChan:    make(chan string, 10),
		running:      false,
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

	// Load the monospace font for map tiles
	// Try to load Cascadia Code NF from system, fall back to Go Mono
	monoSrc, fontPath := e.loadCascadiaCodeNF()
	if monoSrc == nil {
		// Fall back to embedded Go Mono
		fmt.Println("[Font] Using fallback font: Go Mono (embedded)")
		var err error
		monoSrc, err = text.NewGoTextFaceSource(bytes.NewReader(gomono.TTF))
		if err != nil {
			panic(fmt.Sprintf("failed to load mono font: %v", err))
		}
	} else {
		fmt.Printf("[Font] Loaded: %s\n", fontPath)
	}
	e.monoFontSource = monoSrc

	// Load the sans-serif font for UI text
	sansSrc, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(fmt.Sprintf("failed to load sans font: %v", err))
	}
	e.sansFontSource = sansSrc

	// Calculate initial viewport based on window and tile size
	e.recalculateViewport()
}

// loadCascadiaCodeNF attempts to load Cascadia Code NF from common system font locations
// Returns the font source and the path it was loaded from (empty if not found)
func (e *EbitenRenderer) loadCascadiaCodeNF() (*text.GoTextFaceSource, string) {
	// Common font file names for Cascadia Code NF
	fontNames := []string{
		"CascadiaCodeNF-Regular.otf",
		"CascadiaCodeNF-Regular.ttf",
		"CascadiaCodeNFMono-Regular.ttf",
		"CaskaydiaCoveNerdFont-Regular.ttf",
		"CaskaydiaCoveNerdFontMono-Regular.ttf",
		"Caskaydia Cove Nerd Font Complete Mono.ttf",
		"CascadiaCode-Regular.ttf",
		"CascadiaMono-Regular.ttf",
	}

	// Common font directories
	fontDirs := []string{}

	// User font directories
	if home, err := os.UserHomeDir(); err == nil {
		fontDirs = append(fontDirs,
			filepath.Join(home, ".local", "share", "fonts"),
			filepath.Join(home, ".fonts"),
		)
	}

	// System font directories
	fontDirs = append(fontDirs,
		"/usr/share/fonts/truetype",
		"/usr/share/fonts/TTF",
		"/usr/share/fonts",
		"/usr/local/share/fonts",
	)

	// Search for the font
	for _, dir := range fontDirs {
		for _, name := range fontNames {
			// Try direct path
			fontPath := filepath.Join(dir, name)
			if src := e.tryLoadFont(fontPath); src != nil {
				return src, fontPath
			}

			// Try in subdirectories (fonts are often in brand-named folders)
			subdirs := []string{"cascadia-code", "cascadia-code-nf-fonts", "cascadia", "nerd-fonts", "CascadiaCode", "TTF"}
			for _, subdir := range subdirs {
				fontPath = filepath.Join(dir, subdir, name)
				if src := e.tryLoadFont(fontPath); src != nil {
					return src, fontPath
				}
			}
		}

		// Also try glob pattern for any Cascadia/Caskaydia font
		patterns := []string{
			filepath.Join(dir, "**/Caskaydia*Mono*.ttf"),
			filepath.Join(dir, "**/CascadiaCode*.ttf"),
			filepath.Join(dir, "**/CascadiaMono*.ttf"),
		}
		for _, pattern := range patterns {
			matches, _ := filepath.Glob(pattern)
			for _, match := range matches {
				if src := e.tryLoadFont(match); src != nil {
					return src, match
				}
			}
		}
	}

	return nil, ""
}

// tryLoadFont attempts to load a font from the given path
func (e *EbitenRenderer) tryLoadFont(path string) *text.GoTextFaceSource {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	src, err := text.NewGoTextFaceSource(bytes.NewReader(data))
	if err != nil {
		return nil
	}

	return src
}

// Clear clears the display (no-op for Ebiten, clearing happens in Draw)
func (e *EbitenRenderer) Clear() {
	// In Ebiten, clearing happens automatically in Draw
}

// GetInput gets user input from Ebiten (blocking)
func (e *EbitenRenderer) GetInput() string {
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
func (e *EbitenRenderer) FormatText(msg string, args ...any) string {
	// Simple formatting - strip markup tags for now
	ret := fmt.Sprintf(msg, args...)

	// Remove markup tags like GT{}, ITEM{}, ROOM{}, ACTION{}
	// These are TUI-specific; Ebiten uses its own rendering
	for {
		start := strings.Index(ret, "{")
		if start == -1 {
			break
		}
		end := strings.Index(ret[start:], "}")
		if end == -1 {
			break
		}

		// Find the function name before {
		funcStart := start
		for funcStart > 0 && ret[funcStart-1] != ' ' && ret[funcStart-1] != '\n' {
			funcStart--
		}

		// Extract content between { and }
		content := ret[start+1 : start+end]

		// Replace the whole markup with just the content
		ret = ret[:funcStart] + content + ret[start+end+1:]
	}

	return ret
}

// ShowMessage displays a message to the user
func (e *EbitenRenderer) ShowMessage(msg string) {
	e.gameMutex.Lock()
	defer e.gameMutex.Unlock()
	e.messages = append(e.messages, msg)
	if len(e.messages) > 5 {
		e.messages = e.messages[len(e.messages)-5:]
	}
}

// GetViewportSize returns the current viewport dimensions
func (e *EbitenRenderer) GetViewportSize() (rows, cols int) {
	return e.viewportRows, e.viewportCols
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

	// Copy messages
	e.snapshot.messages = make([]string, len(g.Messages))
	copy(e.snapshot.messages, g.Messages)

	// Copy owned items
	e.snapshot.ownedItems = make([]string, 0)
	g.OwnedItems.Each(func(item *world.Item) {
		e.snapshot.ownedItems = append(e.snapshot.ownedItems, item.Name)
	})

	// Copy generator states
	e.snapshot.generators = make([]generatorState, len(g.Generators))
	for i, gen := range g.Generators {
		e.snapshot.generators[i] = generatorState{
			powered:           gen.IsPowered(),
			batteriesInserted: gen.BatteriesInserted,
			batteriesRequired: gen.BatteriesRequired,
		}
	}

	// Copy active callouts (with expiration filtering)
	e.calloutsMutex.Lock()
	now := time.Now().Unix()
	activeCallouts := make([]Callout, 0)
	for _, c := range e.callouts {
		if c.ExpiresAt == 0 || c.ExpiresAt > now {
			activeCallouts = append(activeCallouts, c)
		}
	}
	e.callouts = activeCallouts // Remove expired callouts
	e.snapshot.callouts = make([]Callout, len(activeCallouts))
	copy(e.snapshot.callouts, activeCallouts)
	e.calloutsMutex.Unlock()
}

// computeRoomLabels computes the longest horizontal top wall for each visited room
// and returns label definitions for rendering.
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

	// For each room, track the best (longest) horizontal boundary segment
	type segment struct {
		row      int
		startCol int
		endCol   int
	}
	bestByRoom := make(map[string]segment)

	// Scan each row for contiguous runs of "top boundary" cells for a room.
	// A top boundary cell is a room cell whose north neighbor is not the same room.
	for row := 0; row < rows; row++ {
		currentRoom := ""
		runStartCol := -1

		flushRun := func(endCol int) {
			if currentRoom == "" || runStartCol < 0 {
				return
			}
			length := endCol - runStartCol + 1
			if length <= 0 {
				return
			}
			// Only label rooms the player has actually visited
			if !roomVisited[currentRoom] {
				return
			}
			if existing, ok := bestByRoom[currentRoom]; ok {
				existingLen := existing.endCol - existing.startCol + 1
				if length > existingLen || (length == existingLen && row < existing.row) {
					bestByRoom[currentRoom] = segment{row: row, startCol: runStartCol, endCol: endCol}
				}
			} else {
				bestByRoom[currentRoom] = segment{row: row, startCol: runStartCol, endCol: endCol}
			}
		}

		for col := 0; col < cols; col++ {
			cell := g.Grid.GetCell(row, col)
			if cell == nil || !cell.Room || cell.Name == "" {
				// End any current run
				if runStartCol >= 0 {
					flushRun(col - 1)
					currentRoom = ""
					runStartCol = -1
				}
				continue
			}

			// Only consider cells whose north neighbor is not the same room:
			// these form the top edge of the room.
			isTopBoundary := false
			if cell.North == nil || !cell.North.Room || cell.North.Name != cell.Name {
				isTopBoundary = true
			}

			if !isTopBoundary {
				// Not part of the top wall – flush any active run
				if runStartCol >= 0 {
					flushRun(col - 1)
					currentRoom = ""
					runStartCol = -1
				}
				continue
			}

			// Cell is a top boundary cell for its room
			roomName := cell.Name

			// Start a new run if necessary or if room changes
			if runStartCol < 0 || roomName != currentRoom {
				if runStartCol >= 0 {
					flushRun(col - 1)
				}
				currentRoom = roomName
				runStartCol = col
			}
		}

		// Flush run at end of row
		if runStartCol >= 0 {
			flushRun(cols - 1)
		}
	}

	if len(bestByRoom) == 0 {
		return nil
	}

	labels := make([]roomLabel, 0, len(bestByRoom))
	for roomName, seg := range bestByRoom {
		labels = append(labels, roomLabel{
			RoomName: roomName,
			Row:      seg.row,
			StartCol: seg.startCol,
			EndCol:   seg.endCol,
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

	filtered = append(filtered, Callout{
		Row:       row,
		Col:       col,
		Message:   message,
		Color:     col_color,
		ExpiresAt: expiresAt,
	})
	e.callouts = filtered
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

	// Show room entry callout
	e.AddCallout(row, col, roomName, renderer.CalloutColorRoom, 0)
	return true
}

// Update handles input and game logic (Ebiten interface)
func (e *EbitenRenderer) Update() error {
	// Handle font size changes (Ctrl+= to increase, Ctrl+- to decrease)
	e.handleZoom()

	// Check for keyboard input
	input := e.checkInput()
	if input != "" {
		// Non-blocking send to input channel
		select {
		case e.inputChan <- input:
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
	// Header: ~60px, Status bar: ~60px, Messages: ~120px, margins: ~80px
	availableHeight := h - 320
	availableWidth := w - 100

	// Calculate viewport dimensions
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

// checkInput checks for keyboard input and returns the corresponding command
func (e *EbitenRenderer) checkInput() string {
	// Arrow keys / NSEW navigation
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		return "arrow_up"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		return "arrow_down"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		return "arrow_left"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		return "arrow_right"
	}

	// WASD navigation (as arrow alternatives)
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		return "arrow_up"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) && !ebiten.IsKeyPressed(ebiten.KeyControl) {
		return "arrow_down"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		return "arrow_left"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		return "arrow_right"
	}

	// Vim-style navigation
	if inpututil.IsKeyJustPressed(ebiten.KeyK) {
		return "k"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyJ) {
		return "j"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		return "h"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		return "l"
	}

	// NSEW keys
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		return "n"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		return "e"
	}

	// Help
	if inpututil.IsKeyJustPressed(ebiten.KeySlash) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		return "?"
	}

	// Quit
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		return "quit"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return "quit"
	}

	return ""
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
	directionLabelHeight := int(uiFontSize) + 15
	statusBarHeight := int(uiFontSize)*2 + 20
	messagesHeight := int(uiFontSize)*6 + 20
	helpHeight := int(uiFontSize) + 15

	// Calculate available space for map
	availableHeight := screenHeight - headerHeight - directionLabelHeight*2 - statusBarHeight - messagesHeight - helpHeight - 40
	availableWidth := screenWidth - 200 // Leave space for east/west labels

	// Calculate map dimensions
	mapAreaWidth := e.viewportCols * e.tileSize
	mapAreaHeight := e.viewportRows * e.tileSize

	// Constrain map to available space
	if mapAreaWidth > availableWidth {
		mapAreaWidth = availableWidth
	}
	if mapAreaHeight > availableHeight {
		mapAreaHeight = availableHeight
	}

	mapX := (screenWidth - mapAreaWidth) / 2
	mapY := headerHeight + directionLabelHeight

	// Draw header (deck number and room name) - use snapshot data
	e.drawHeaderFromSnapshot(screen, &snap, screenWidth)

	// Draw map background
	vector.DrawFilledRect(screen, float32(mapX-10), float32(mapY-10),
		float32(mapAreaWidth+20), float32(mapAreaHeight+20),
		colorMapBackground, false)

	// Draw the map using snapshot for player position
	e.drawMap(screen, g, mapX, mapY, &snap)

	// Draw direction labels
	e.drawDirectionLabels(screen, g, mapX, mapY, mapAreaWidth, mapAreaHeight)

	// Draw status bar (below map) - use snapshot data
	statusY := mapY + mapAreaHeight + directionLabelHeight + 10
	e.drawStatusBarFromSnapshot(screen, &snap, mapX, statusY, mapAreaWidth)

	// Draw messages panel (below status) - use snapshot data
	messagesY := statusY + statusBarHeight + 10
	e.drawMessagesFromSnapshot(screen, &snap, mapX, messagesY, mapAreaWidth, messagesHeight)

	// Draw help text at bottom
	helpText := "Arrow/WASD/HJKL: Move | ?: Hint | +/-: Zoom | Q/Esc: Quit"
	e.drawColoredText(screen, helpText, 10, screenHeight-int(uiFontSize)-10, colorSubtle)
}

// drawHeaderFromSnapshot draws the deck number and room name using snapshot data
func (e *EbitenRenderer) drawHeaderFromSnapshot(screen *ebiten.Image, snap *renderSnapshot, screenWidth int) {
	// Deck number
	deckText := fmt.Sprintf("Deck %d", snap.level)
	e.drawColoredText(screen, deckText, 20, 20, colorAction)

	// Room name (centered)
	roomText := fmt.Sprintf("In: %s", snap.cellName)
	textWidth := e.getTextWidth(roomText)
	e.drawColoredText(screen, roomText, (screenWidth-int(textWidth))/2, 20, colorFloorVisited)
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
			icon, col, hasBg := e.getCellDisplay(g, cell, snap)

			x := mapX + vCol*e.tileSize
			y := mapY + vRow*e.tileSize

			// Draw the tile character with optional background
			e.drawTile(screen, icon, x, y, col, hasBg)
		}
	}

	// Draw persistent room labels on top of the map
	e.drawRoomLabels(screen, snap, mapX, mapY, startRow, startCol)

	// Draw callouts on top of the map
	e.drawCallouts(screen, snap, mapX, mapY, startRow, startCol)
}

// drawRoomLabels renders persistent room name labels above the longest horizontal wall of each room
func (e *EbitenRenderer) drawRoomLabels(screen *ebiten.Image, snap *renderSnapshot, mapX, mapY, startRow, startCol int) {
	if len(snap.roomLabels) == 0 {
		return
	}

	fontSize := e.getUIFontSize()

	for _, rl := range snap.roomLabels {
		// Determine visible portion of this wall segment in the current viewport
		wallStartCol := rl.StartCol
		wallEndCol := rl.EndCol

		viewportStartCol := startCol
		viewportEndCol := startCol + e.viewportCols - 1

		// Intersection of wall segment with viewport columns
		visStartCol := wallStartCol
		if visStartCol < viewportStartCol {
			visStartCol = viewportStartCol
		}
		visEndCol := wallEndCol
		if visEndCol > viewportEndCol {
			visEndCol = viewportEndCol
		}

		if visStartCol > visEndCol {
			continue // not visible in this viewport
		}

		// Compute center column of the visible part
		centerCol := (visStartCol + visEndCol) / 2

		// Convert to viewport coordinates
		vCol := centerCol - startCol
		vRow := rl.Row - startRow

		// Skip if not in vertical range
		if vRow < 0 || vRow >= e.viewportRows {
			continue
		}

		// Compute pixel position
		cellX := mapX + vCol*e.tileSize
		cellY := mapY + vRow*e.tileSize

		// Measure text
		textWidth := e.getTextWidth(rl.RoomName)

		// Draw background box for readability
		paddingX := 6
		paddingY := 4
		boxW := int(textWidth) + paddingX*2
		boxH := int(fontSize) + paddingY*2

		// Position box centered horizontally over the wall
		// Raise it by half its height so it sits just above the wall
		boxX := cellX + e.tileSize/2 - boxW/2
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

	for _, callout := range snap.callouts {
		// Calculate screen position from cell position
		vRow := callout.Row - startRow
		vCol := callout.Col - startCol

		// Skip if outside viewport
		if vRow < 0 || vRow >= e.viewportRows || vCol < 0 || vCol >= e.viewportCols {
			continue
		}

		// Calculate pixel position (center of the cell)
		cellX := mapX + vCol*e.tileSize
		cellY := mapY + vRow*e.tileSize

		// Measure text for callout box
		textWidth := e.getTextWidth(callout.Message)
		boxHeight := int(fontSize) + padding*2

		// Position callout to the right and vertically centered with the cell
		calloutX := cellX + e.tileSize + 8
		calloutY := cellY + (e.tileSize-boxHeight)/2

		// If callout would go off right edge, position to the left instead
		boxWidth := int(textWidth) + padding*2
		if calloutX+boxWidth > mapX+e.viewportCols*e.tileSize {
			calloutX = cellX - boxWidth - 8
		}

		// Keep callout within vertical bounds
		if calloutY < mapY {
			calloutY = mapY
		}
		if calloutY+boxHeight > mapY+e.viewportRows*e.tileSize {
			calloutY = mapY + e.viewportRows*e.tileSize - boxHeight
		}

		// Draw callout background
		bgColor := color.RGBA{15, 15, 25, 240}
		borderColor := color.RGBA{80, 80, 100, 255}

		// Border
		vector.DrawFilledRect(screen,
			float32(calloutX-1), float32(calloutY-1),
			float32(boxWidth+2), float32(boxHeight+2),
			borderColor, false)

		// Background
		vector.DrawFilledRect(screen,
			float32(calloutX), float32(calloutY),
			float32(boxWidth), float32(boxHeight),
			bgColor, false)

		// Draw pointer/arrow toward the cell
		arrowSize := float32(6)
		arrowY := float32(calloutY + boxHeight/2)
		if calloutX > cellX+e.tileSize {
			// Arrow pointing left
			arrowX := float32(calloutX - 1)
			vector.DrawFilledRect(screen, arrowX-arrowSize, arrowY-2, arrowSize, 4, borderColor, false)
		} else {
			// Arrow pointing right
			arrowX := float32(calloutX + boxWidth + 1)
			vector.DrawFilledRect(screen, arrowX, arrowY-2, arrowSize, 4, borderColor, false)
		}

		// Draw text - position so baseline is vertically centered in box
		// drawColoredText adds fontSize to y for baseline, so we need to offset
		textY := calloutY + padding - int(fontSize)
		e.drawColoredText(screen, callout.Message, calloutX+padding, textY, callout.Color)
	}
}

// drawTile draws a single tile at the given position
func (e *EbitenRenderer) drawTile(screen *ebiten.Image, icon string, x, y int, col color.Color, hasBackground bool) {
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
		vector.DrawFilledRect(screen, float32(x)+margin, float32(y)+margin,
			float32(e.tileSize)-margin*2, float32(e.tileSize)-margin*2,
			colorWallBg, false)
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

// getTextWidth returns the width of a string in pixels at UI font size
func (e *EbitenRenderer) getTextWidth(str string) float64 {
	face := e.getSansFontFace()
	w, _ := text.Measure(str, face, 0)
	return w
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

	// CCTV Terminal (show if has map or discovered)
	if gameworld.HasTerminal(r) && (g.HasMap || r.Discovered) {
		if data.Terminal.IsUsed() {
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
		return IconExitUnlocked, colorExitUnlocked, true
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
func (e *EbitenRenderer) drawStatusBarFromSnapshot(screen *ebiten.Image, snap *renderSnapshot, x, y, width int) {
	// Draw panel background
	vector.DrawFilledRect(screen, float32(x-10), float32(y-5),
		float32(width+20), 50, colorPanelBackground, false)

	// Inventory line
	invText := "Inventory: "
	if len(snap.ownedItems) == 0 && snap.batteries == 0 {
		invText += "(empty)"
	} else {
		items := make([]string, len(snap.ownedItems))
		copy(items, snap.ownedItems)
		if snap.batteries > 0 {
			items = append(items, fmt.Sprintf("Batteries x%d", snap.batteries))
		}
		invText += strings.Join(items, ", ")
	}
	e.drawColoredText(screen, invText, x, y, colorText)

	// Generator status (if applicable)
	if len(snap.generators) > 0 {
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
		e.drawColoredText(screen, genText, x, y+20, colorText)
	}
}

// drawMessagesFromSnapshot draws the messages panel using snapshot data
func (e *EbitenRenderer) drawMessagesFromSnapshot(screen *ebiten.Image, snap *renderSnapshot, x, y, width, maxHeight int) {
	// Draw panel background
	panelHeight := 120
	if panelHeight > maxHeight {
		panelHeight = maxHeight
	}
	vector.DrawFilledRect(screen, float32(x-10), float32(y-5),
		float32(width+20), float32(panelHeight), colorPanelBackground, false)

	// Header
	e.drawColoredText(screen, "─── Messages ───", x, y, colorSubtle)

	// Messages
	if len(snap.messages) == 0 {
		e.drawColoredText(screen, "(no messages)", x+10, y+20, colorSubtle)
	} else {
		for i, msg := range snap.messages {
			// Strip any ANSI codes or markup
			cleanMsg := e.FormatText("%s", msg)
			e.drawColoredText(screen, cleanMsg, x+10, y+20+i*16, colorText)
		}
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
