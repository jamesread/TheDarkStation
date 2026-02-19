// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
package ebiten

import (
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	engineinput "darkstation/pkg/engine/input"
	gamemenu "darkstation/pkg/game/menu"
	"darkstation/pkg/game/state"
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

// CellRenderOptions describes how a cell should be drawn on the map.
type CellRenderOptions struct {
	Icon            string
	Color           color.Color
	HasBackground   bool
	BackgroundColor color.Color // optional; used when HasBackground is true (overrides default wall bg)
}

// keyRepeatInfo tracks the repeat state for a key or button
type keyRepeatInfo struct {
	firstPressed int64 // Timestamp when first pressed (milliseconds)
	lastRepeat   int64 // Timestamp when last repeat event was sent (milliseconds)
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
	monoFontSource     *text.GoTextFaceSource // Monospace font for map tiles
	sansFontSource     *text.GoTextFaceSource // Sans-serif font for UI text
	sansBoldFontSource *text.GoTextFaceSource // Sans-serif bold for menu titles

	// Cached font faces (recreated when tile size changes)
	cachedTileFontSize      float64
	cachedUIFontSize        float64
	cachedMonoUIFontSize    float64
	cachedMonoFace          *text.GoTextFace
	cachedSansFace          *text.GoTextFace
	cachedSansBoldFace      *text.GoTextFace // Sans bold for menu titles (same size as UI)
	cachedSansBoldTitleFace *text.GoTextFace // Sans bold 2pt larger for menu title text
	cachedSansBoldTitleSize float64          // Size used for cachedSansBoldTitleFace
	cachedMonoUIFace        *text.GoTextFace // Monospace font with UI size (for console)

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

	// Flag to track if we've logged window opening
	windowOpenedLogged bool

	// Messages to display with timestamps for fade-out
	trackedMessages []messageEntry
	messagesMutex   sync.RWMutex

	// Generic menu overlay state
	genericMenuActive   bool
	genericMenuItems    []gamemenu.MenuItem
	genericMenuSelected int
	genericMenuHelpText string
	genericMenuTitle    string
	genericMenuMutex    sync.RWMutex

	// Preserved menu state for smooth transitions
	prevMenuItems    []gamemenu.MenuItem
	prevMenuTitle    string
	prevMenuHelpText string

	// Menu highlight animation state
	menuHighlightAnimStartIndex  int
	menuHighlightAnimTargetIndex int
	menuHighlightAnimStartWidth  float64 // Width at start of animation
	menuHighlightAnimTargetWidth float64 // Width at target of animation
	menuHighlightAnimStartTime   int64   // Timestamp when animation started (milliseconds)
	menuHighlightAnimating       bool

	// Menu height animation state
	menuHeightAnimStartHeight  float64 // Height at start of animation
	menuHeightAnimTargetHeight float64 // Height at target of animation
	menuHeightAnimStartTime    int64   // Timestamp when height animation started (milliseconds)
	menuHeightAnimating        bool

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

	// Camera transition state (smooth pan when focusing on room in select room dialog)
	cameraCenterRow       float64
	cameraCenterCol       float64
	cameraTargetRow       float64
	cameraTargetCol       float64
	cameraFromRow         float64
	cameraFromCol         float64
	cameraTransitionStart int64 // Timestamp when transition started (nanoseconds, for sub-ms precision)

	// Offscreen map buffer - render tiles here at integer coords, then blit with fractional
	// offset. Eliminates per-tile sub-pixel jitter during camera transitions.
	mapBuffer        *ebiten.Image
	mapBufferWidth   int
	mapBufferHeight int

	// Background animation for main menu (floating tiles)
	floatingTiles      []floatingTile
	floatingTilesMutex sync.RWMutex

	// Console state
	consoleActive        bool
	consoleText          string   // Current input text
	consoleHistory       []string // Command history
	consoleHistoryIndex  int      // Current position in history (for up/down navigation)
	consoleOutput        []string // Console output lines
	consoleScrollOffset  int      // Scroll offset for page up/down (0 = showing most recent)
	consoleAnimProgress  float64  // 0.0 (closed) to 1.0 (open)
	consoleAnimating     bool
	consoleAnimStartTime int64 // Timestamp when animation started
	consoleMutex         sync.RWMutex
}

// floatingTile represents a single tile in the background animation
type floatingTile struct {
	x, y          float64 // Position
	vx, vy        float64 // Velocity
	icon          string
	color         color.Color
	alpha         float64 // Opacity (0.0 to 1.0)
	rotation      float64 // Rotation angle in radians
	rotationSpeed float64 // Rotation speed
}
