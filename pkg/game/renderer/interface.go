package renderer

import (
	"darkstation/pkg/game/state"
)

// TextStyle represents different text styling options
type TextStyle int

const (
	StyleNormal TextStyle = iota
	StyleCell
	StyleCellText
	StyleItem
	StyleAction
	StyleActionShort
	StyleDenied
	StyleKeycard
	StyleDoor
	StyleHazard
	StyleHazardCtrl
	StyleFurniture
	StyleFurnitureChecked
	StyleSubtle
	StylePlayer
	StyleExitOpen
)

// Renderer defines the interface for game rendering backends
// Implementations can include TUI (terminal), SDL, Ebiten, etc.
type Renderer interface {
	// Init initializes the renderer (colors, fonts, window, etc.)
	Init()

	// Clear clears the display
	Clear()

	// RenderFrame renders a complete game frame
	// This includes the map, status bar, messages, and input prompt
	RenderFrame(g *state.Game)

	// GetInput gets user input (blocking for TUI, event-based for GUI)
	GetInput() string

	// StyleText applies a style to text and returns the styled string
	// For TUI this applies ANSI colors, for GUI it may return markup
	StyleText(text string, style TextStyle) string

	// FormatText formats a message with the renderer's markup system
	FormatText(msg string, args ...any) string

	// ShowMessage displays a message to the user
	ShowMessage(msg string)

	// GetViewportSize returns the current viewport dimensions (rows, cols)
	GetViewportSize() (rows, cols int)
}

// Current holds the active renderer instance
var Current Renderer

// SetRenderer sets the active renderer
func SetRenderer(r Renderer) {
	Current = r
}

// Init initializes the current renderer
func Init() {
	if Current != nil {
		Current.Init()
	}
}

// Clear clears the display using the current renderer
func Clear() {
	if Current != nil {
		Current.Clear()
	}
}

// RenderFrame renders a complete game frame
func RenderFrame(g *state.Game) {
	if Current != nil {
		Current.RenderFrame(g)
	}
}

// GetInput gets user input from the current renderer
func GetInput() string {
	if Current != nil {
		return Current.GetInput()
	}
	return ""
}

// StyleText applies a style to text
func StyleText(text string, style TextStyle) string {
	if Current != nil {
		return Current.StyleText(text, style)
	}
	return text
}

// FormatText formats a message with markup
func FormatText(msg string, args ...any) string {
	if Current != nil {
		return Current.FormatText(msg, args...)
	}
	return msg
}

// GetViewportSize returns viewport dimensions
func GetViewportSize() (rows, cols int) {
	if Current != nil {
		return Current.GetViewportSize()
	}
	return 15, 30 // sensible defaults
}
