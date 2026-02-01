// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
// Ebiten is a 2D game library for Go: https://ebiten.org/
package ebiten

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/game/config"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/resources"
)

// Constants, types, and variables are now in separate files:
// - constants.go: color palette, icons, constants
// - types.go: type definitions
// - input.go: input handling and Update method
// - rendering.go: main Draw function and drawing functions
// - text.go: text rendering functions
// - cell.go: cell rendering logic
// - callouts.go: callout management
// - menu.go: menu rendering
// - snapshot.go: frame rendering and snapshot management
// - font.go: font management
// - animation.go: animation utilities

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
		log.Printf("[Font] Monospace: Cascadia Code NF failed to load, using Go Mono (embedded)")
		monoSrc, err = text.NewGoTextFaceSource(bytes.NewReader(gomono.TTF))
		if err != nil {
			panic(fmt.Sprintf("failed to load mono font: %v", err))
		}
	} else {
		log.Printf("[Font] Monospace: Cascadia Code NF (embedded)")
	}
	e.monoFontSource = monoSrc

	// Load the sans-serif font for UI text
	sansSrc, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(fmt.Sprintf("failed to load sans font: %v", err))
	}
	log.Printf("[Font] Sans-serif: Go Regular (embedded)")
	e.sansFontSource = sansSrc

	// Load the sans-serif bold font for menu titles
	boldSrc, err := text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))
	if err != nil {
		panic(fmt.Sprintf("failed to load sans bold font: %v", err))
	}
	log.Printf("[Font] Sans-serif bold: Go Bold (embedded)")
	e.sansBoldFontSource = boldSrc

	// Calculate initial viewport based on window and tile size
	e.recalculateViewport()

	// Initialize console cvars
	initCvars()
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
