package renderer

import (
	"image/color"

	"darkstation/pkg/engine/input"
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
	// It returns a high-level Intent from the tiered input system.
	GetInput() input.Intent

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

// Version information (set by main package during initialization)
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// SetVersion sets the version information
func SetVersion(v, c, d string) {
	Version = v
	Commit = c
	Date = d
}

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
		intent := Current.GetInput()
		// Backwards-compatible helper: most callers only care about the action.
		switch intent.Action {
		case input.ActionMoveNorth:
			return "arrow_up"
		case input.ActionMoveSouth:
			return "arrow_down"
		case input.ActionMoveWest:
			return "arrow_left"
		case input.ActionMoveEast:
			return "arrow_right"
		case input.ActionHint:
			return "?"
		case input.ActionQuit:
			return "quit"
		case input.ActionScreenshot:
			return "screenshot"
		case input.ActionAction:
			return "enter"
		default:
			return ""
		}
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

// CalloutRenderer is an optional interface for renderers that support floating callouts
type CalloutRenderer interface {
	// AddCallout adds a floating message near a specific cell
	// durationMs is how long the callout should display (0 = until cleared)
	AddCallout(row, col int, message string, c color.Color, durationMs int)

	// ClearCallouts removes all active callouts
	ClearCallouts()

	// ClearCalloutsIfMoved clears callouts if player moved from the last known position
	ClearCalloutsIfMoved(row, col int) bool

	// SetDebounceAnimation triggers a debounce animation in the given direction
	SetDebounceAnimation(direction string)

	// ShowRoomEntryIfNew shows a room entry callout if the player entered a new room
	// Returns true if a callout was shown
	ShowRoomEntryIfNew(row, col int, roomName string) bool
}

// BindingsMenuRenderer is an optional interface for renderers that can draw
// a full-screen bindings menu overlay on top of the map.
type BindingsMenuRenderer interface {
	// RenderBindingsMenu draws the bindings menu overlay for the given action list
	// and currently selected index. helpText is optional instruction text to display
	// on the menu (e.g., "Type new binding code..." when editing).
	// nonRebindable is a set of actions that cannot be rebound (should be displayed differently).
	RenderBindingsMenu(g *state.Game, actions []input.Action, selected int, helpText string, nonRebindable map[input.Action]bool)

	// ClearBindingsMenu hides any active bindings menu overlay.
	ClearBindingsMenu()
}

// Callout colors for different message types (matching cell colors)
var (
	CalloutColorInfo             = color.RGBA{200, 200, 255, 255} // Light blue
	CalloutColorSuccess          = color.RGBA{100, 255, 150, 255} // Green
	CalloutColorWarning          = color.RGBA{255, 220, 100, 255} // Yellow
	CalloutColorDanger           = color.RGBA{255, 120, 120, 255} // Red
	CalloutColorItem             = color.RGBA{220, 170, 255, 255} // Purple
	CalloutColorGenerator        = color.RGBA{255, 100, 100, 255} // Red (unpowered)
	CalloutColorGeneratorOn      = color.RGBA{0, 255, 100, 255}   // Green (powered)
	CalloutColorTerminal         = color.RGBA{100, 150, 255, 255} // Blue
	CalloutColorFurniture        = color.RGBA{255, 150, 255, 255} // Pink (unchecked)
	CalloutColorFurnitureChecked = color.RGBA{200, 180, 100, 255} // Tan/brown (checked, decorative)
	CalloutColorHazardCtrl       = color.RGBA{0, 255, 255, 255}   // Cyan
	CalloutColorHazard           = color.RGBA{255, 80, 80, 255}   // Red for hazards
	CalloutColorRoom             = color.RGBA{180, 180, 220, 255} // Light gray-blue for room names
	CalloutColorDoor             = color.RGBA{255, 255, 0, 255}   // Yellow for locked doors
)

// AddCallout adds a callout if the current renderer supports it
func AddCallout(row, col int, message string, c color.Color, durationMs int) {
	if cr, ok := Current.(CalloutRenderer); ok {
		cr.AddCallout(row, col, message, c, durationMs)
	}
}

// SetDebounceAnimation triggers a debounce animation in the given direction
func SetDebounceAnimation(direction string) {
	if cr, ok := Current.(CalloutRenderer); ok {
		cr.SetDebounceAnimation(direction)
	}
}

// ClearCallouts clears callouts if the current renderer supports it
func ClearCallouts() {
	if cr, ok := Current.(CalloutRenderer); ok {
		cr.ClearCallouts()
	}
}

// ClearCalloutsIfMoved clears callouts if player moved, returns true if cleared
func ClearCalloutsIfMoved(row, col int) bool {
	if cr, ok := Current.(CalloutRenderer); ok {
		return cr.ClearCalloutsIfMoved(row, col)
	}
	return false
}

// ShowRoomEntryIfNew shows a room entry callout if the player entered a new room
func ShowRoomEntryIfNew(row, col int, roomName string) bool {
	if cr, ok := Current.(CalloutRenderer); ok {
		return cr.ShowRoomEntryIfNew(row, col, roomName)
	}
	return false
}
