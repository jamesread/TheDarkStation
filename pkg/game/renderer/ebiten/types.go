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

// LongUseAdvanceFunc ticks an in-progress hold-to-use session from the Ebiten Update thread.
type LongUseAdvanceFunc func(g *state.Game, held, released bool, nowMs int64)

// HazardClearAdvanceFunc ticks an in-progress hazard clear cinematic from the Ebiten Update thread.
type HazardClearAdvanceFunc func(g *state.Game, nowMs int64)

// HazardTourAdvanceFunc ticks an in-progress exit hazard tour from the Ebiten Update thread.
type HazardTourAdvanceFunc func(g *state.Game, nowMs int64)

// HintRefresher updates on-map control callouts after the primary input device changes.
type HintRefresher func(g *state.Game)

// Callout represents a floating message displayed near a cell
type Callout struct {
	Row       int    // Cell row
	Col       int    // Cell col
	Message   string // Message to display
	Color     color.Color
	ExpiresAt int64 // Unix timestamp when callout expires (0 = never)
	CreatedAt int64 // Unix timestamp when callout was created (for animations)
}

// roomLabel represents a persistent label for a room, positioned at the leftmost point
type roomLabel struct {
	RoomName string
	Powered  bool // Power grid enabled and fed by a powered generator
	Row      int  // Grid row of the label position (room interior row)
	StartCol int  // Grid column index (leftmost point)
	EndCol   int  // Same as StartCol (kept for compatibility)
}

// envPlaque is diegetic corridor signage (gettext msgid); drawn small inside the tile.
type envPlaque struct {
	Row   int
	Col   int
	MsgID string
}

// renderSnapshot holds a consistent snapshot of game state for rendering
// This prevents jitter from race conditions between game logic and rendering
type renderSnapshot struct {
	valid             bool
	seq               uint64
	level             int
	playerRow         int
	playerCol         int
	playerFacing      state.PlayerFacing
	cellName          string
	hasMap            bool
	batteries         int
	ownedItems        []string
	generators        []generatorState
	gridRows          int
	gridCols          int
	callouts          []Callout
	roomLabels        []roomLabel
	envPlaques        []envPlaque
	objectives        []string // Current level objectives
	exitAnimating     bool     // True when exit animation is playing
	exitAnimStartTime int64    // Timestamp when exit animation started
	focusedCellRow    int      // Row of cell with active callout (for focus background)
	focusedCellCol    int      // Col of cell with active callout (for focus background)
	interactableCells []struct {
		row int
		col int
	} // Cells with interactable objects (for focus background)
	longUseActive    bool
	longUseProgress  float64
	longUseTargetRow int
	longUseTargetCol int
	hazardClear      *state.HazardClearSession
	hazardTour       *state.HazardTourSession
	powerGrid        powerGridSnapshot
	mapPower         mapPowerSnapshot
}

// mapPowerSnapshot holds power routing state copied on the game thread for race-free Draw.
type mapPowerSnapshot struct {
	livePowerCells             map[uint64]bool
	roomDoorsPowered           map[string]bool
	roomCCTVPowered            map[string]bool
	manualEgressReleased       map[string]bool
	maintenanceMenuRoom        string
	maintenanceSelectableRooms []string
}

// powerGridSnapshot holds overlay routing computed on the game thread for race-free Draw.
type powerGridSnapshot struct {
	active              bool
	maintenanceMenuRoom string
	maintTerminalRow    int
	maintTerminalCol    int
	overlaySeedRow      int
	overlaySeedCol      int
	overlayDevActive    bool
	liveCells           map[uint64]bool
	armedCells          map[uint64]bool
	fedRooms            map[string]bool
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

	// Interact hold state (updated each Update, polled during long-use hold loops).
	interactHoldMutex    sync.Mutex
	interactHeld         bool
	interactPrevHeld     bool
	interactReleasedEdge bool
	longUseAdvancer      LongUseAdvanceFunc
	longUsePrevHeld      bool
	hazardClearAdvancer  HazardClearAdvanceFunc
	hazardTourAdvancer   HazardTourAdvanceFunc
	hintRefresher        HintRefresher

	// Flag indicating renderer is running
	running bool

	// Flag to track if we've logged window opening
	windowOpenedLogged bool

	// Generic menu overlay state
	genericMenuActive   bool
	genericMenuItems    []gamemenu.MenuItem
	genericMenuLabels   []string // captured on game thread in RenderMenu; Draw must not call GetLabel()
	genericMenuSelected int
	genericMenuHelpText string
	genericMenuTitle    string
	genericMenuMutex    sync.RWMutex

	// Preserved menu state for smooth transitions
	prevMenuItems    []gamemenu.MenuItem
	prevMenuLabels   []string
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

	// menuAnimClockMilli is set once per Ebiten Update(). Menu overlay animations must not use
	// time.Now() inside Draw() — Ebiten may call Draw multiple times per Update, which caused
	// visible jitter on the maintenance terminal / room picker overlay.
	menuAnimClockMilli int64
	// menuAnimTimeNano is the same instant as menuAnimClockMilli (nanoseconds since Unix epoch).
	// Maintenance camera tween uses nanoseconds — UnixMilli quantization can stall progress for several
	// consecutive Updates → visible map hitch while FPS still averages ~60.
	menuAnimTimeNano int64

	// maintPanDrawCount resets each Update; used only when cvar debug.maint_pan is on
	maintPanDrawCount int

	// maintPanCameraTweenActive tracks an in-flight maintenance-camera ease so debug.maint_pan can
	// emit a single COMPLETE line when smootherstep finishes (or clears when exiting maintenance UI).
	maintPanCameraTweenActive bool

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

	playerFacingRot playerFacingRotation
	playerMove      playerMoveTransition

	// Camera transition state (smooth pan when focusing on room in select room dialog)
	cameraCenterRow           float64
	cameraCenterCol           float64
	cameraTargetRow           float64
	cameraTargetCol           float64
	cameraFromRow             float64
	cameraFromCol             float64
	cameraTransitionStartNano int64 // Unix ns when maintenance room pan started (with menuAnimTimeNano source)
	cameraPlaySynced          bool  // true after play-mode camera has been seeded from the player (avoids maint pan from 0,0)

	// Offscreen map buffer - render tiles here at integer coords, then blit with fractional
	// offset. Eliminates per-tile sub-pixel jitter during camera transitions.
	mapBuffer       *ebiten.Image
	mapBufferWidth  int
	mapBufferHeight int
	mapDrawCache    mapDrawCache

	// snapSeq increments each RenderFrame; map draw cache uses it to skip redundant buffer fills
	// when Ebiten calls Draw() more than once per game tick.
	snapSeq uint64

	// Cached expensive snapshot derivations (invalidated by lightweight keys; no game logic changes).
	mapPowerSnapCacheKey mapPowerSnapCacheKey
	mapPowerLiveCells    map[uint64]bool
	roomLabelsCacheKey   roomLabelsCacheKey
	roomLabelsCache      []roomLabel
	objectivesCacheKey   objectivesCacheKey
	objectivesCache      []string
	envPlaquesCacheKey   envPlaquesCacheKey
	envPlaquesCache      []envPlaque

	// Background animation for main menu (floating tiles)
	floatingTiles      []floatingTile
	floatingTilesMutex sync.RWMutex

	// Developer message (bottom-left overlay; e.g. map dump confirmation)
	developerMessage      string
	developerMessageAt    int64
	developerMessageMutex sync.RWMutex

	// Transient notification (top-center; e.g. input device switch)
	notificationMessage string
	notificationAt      int64
	notificationMutex   sync.RWMutex

	// Developer debug overlays (map area border, FOV rays, etc.)
	drawMapAreaBorder  bool
	fovRayDebugEnabled bool
	devDebugMutex      sync.RWMutex

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

	// Text input dialog (developer seed entry, etc.)
	textInputActive   bool
	textInputHex      bool
	textInputTitle    string
	textInputPrompt   string
	textInputText     string
	textInputResultCh chan textInputResult
	textInputMutex    sync.RWMutex

	// Confirmation dialog (quit confirmation, etc.)
	confirmActive   bool
	confirmTitle    string
	confirmMessage  string
	confirmResultCh chan bool
	confirmMutex    sync.RWMutex

	// Level generation loading overlay (updated on game thread, read in Draw).
	levelGen     levelGenLoading
	loadingMutex sync.RWMutex
}

// mapDrawCache records the last offscreen map build so duplicate Draw() calls in one frame
// can blit without refilling every tile.
type mapDrawCache struct {
	valid                        bool
	snapSeq                      uint64
	camRowMilli, camColMilli     int64
	startRow, startCol           int
	blitX, blitY                 float64
	bufW, bufH                   int
	tileSize, viewRows, viewCols int
}

type mapPowerSnapCacheKey struct {
	powerSupply, powerConsumption int
	generatorPoweredMask          uint64
	maintRoom                     string
	powerGridOverlay              bool
}

type roomLabelsCacheKey struct {
	level, playerRow, playerCol int
	maintRoom                   string
	maintMenuMode               bool
	powerSupply                 int
	generatorPoweredMask        uint64
}

type objectivesCacheKey struct {
	level, movementCount, interactionsCount int
	unpoweredGenerators                     int
}

type envPlaquesCacheKey struct {
	level, movementCount, interactionsCount int
	envPlaquesEnabled                       bool
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
