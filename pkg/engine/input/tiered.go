package input

import (
	"sort"
	"time"
)

// Device represents a physical input source.
type Device int

const (
	DeviceUnknown Device = iota
	DeviceKeyboard
	DeviceGamepad
	DeviceTerminal
)

// Action represents a high‑level intent in the game.
type Action int

const (
	ActionNone Action = iota

	// Movement
	ActionMoveNorth
	ActionMoveSouth
	ActionMoveWest
	ActionMoveEast

	// Meta / UI
	ActionHint
	ActionQuit
	ActionScreenshot
	ActionOpenMenu
	ActionAction     // Generic "action/confirm" (e.g., Enter/A)
	ActionInteract   // Interact with furniture/objects (E, Enter, A button)
	ActionDevMap     // Switch to developer testing map (F9)
	ActionResetLevel // Reset current level (F5)
	ActionZoomIn     // Zoom in (increase font/tile size)
	ActionZoomOut    // Zoom out (decrease font/tile size)
)

// Intent is the 4th‑layer, high‑level description of what the player wants to do.
type Intent struct {
	Action Action
}

// RawInput is the 1st‑layer event emitted directly from an input device.
// Code is a device‑specific identifier (e.g. "KeyW", "arrow_up", "GamepadDPadUp").
type RawInput struct {
	Device    Device
	Code      string
	Timestamp time.Time
}

// DebouncedInput is the 2nd‑layer representation after debouncing/deduplication.
// For this turn‑based game, we treat each RawInput as already debounced by
// the underlying libraries (Ebiten, terminal raw mode), but keep a distinct
// type to make the layering explicit and extensible.
type DebouncedInput struct {
	Device Device
	Code   string
}

// NewDebouncedInput converts a raw event to a debounced event.
// At the moment this is a thin wrapper, but it is the right place to add
// key‑repeat suppression or timing based logic later.
func NewDebouncedInput(raw RawInput) DebouncedInput {
	return DebouncedInput{
		Device: raw.Device,
		Code:   raw.Code,
	}
}

// bindings maps raw codes to actions (3rd-layer bindings).
// Multiple codes may point to the same Action.
var bindings = map[string]Action{
	// Movement (arrows, NSEW, Vim)
	"arrow_up":    ActionMoveNorth,
	"north":       ActionMoveNorth,
	"n":           ActionMoveNorth,
	"k":           ActionMoveNorth,
	"arrow_down":  ActionMoveSouth,
	"south":       ActionMoveSouth,
	"s":           ActionMoveSouth,
	"j":           ActionMoveSouth,
	"arrow_left":  ActionMoveWest,
	"west":        ActionMoveWest,
	"w":           ActionMoveWest,
	"h":           ActionMoveWest,
	"arrow_right": ActionMoveEast,
	"east":        ActionMoveEast,
	"l":           ActionMoveEast,

	// Help / hint
	"?":    ActionHint,
	"hint": ActionHint,

	// Quit
	"quit":   ActionQuit,
	"q":      ActionQuit,
	"escape": ActionQuit,

	// Screenshot
	"screenshot": ActionScreenshot,

	// Menu
	"menu": ActionOpenMenu,
	"f9":   ActionDevMap,

	// Controller/gamepad specific bindings
	"gamepad_dpad_up":    ActionMoveNorth,
	"gamepad_dpad_down":  ActionMoveSouth,
	"gamepad_dpad_left":  ActionMoveWest,
	"gamepad_dpad_right": ActionMoveEast,

	// Interaction (E, Enter, A button)
	"e":         ActionInteract,
	"enter":     ActionInteract,
	"gamepad_a": ActionInteract, // A button / Cross

	// Zoom (fixed bindings, not rebindable)
	"=":               ActionZoomIn,
	"+":               ActionZoomIn,
	"numpad_add":      ActionZoomIn,
	"-":               ActionZoomOut,
	"numpad_subtract": ActionZoomOut,

	// Generic action/confirm inputs (reserved, not unbindable)
	"action": ActionAction,

	"gamepad_b":     ActionQuit,     // B button / Circle
	"gamepad_start": ActionOpenMenu, // Start button
}

// MapToIntent is the 3rd+4th layer: it applies the current bindings to a
// debounced input and returns a high‑level Intent.
func MapToIntent(ev DebouncedInput) Intent {
	if act, ok := bindings[ev.Code]; ok {
		return Intent{Action: act}
	}
	return Intent{Action: ActionNone}
}

// ActionName returns a human-friendly name for an action.
func ActionName(a Action) string {
	switch a {
	case ActionMoveNorth:
		return "Move North"
	case ActionMoveSouth:
		return "Move South"
	case ActionMoveWest:
		return "Move West"
	case ActionMoveEast:
		return "Move East"
	case ActionHint:
		return "Hint"
	case ActionQuit:
		return "Quit"
	case ActionScreenshot:
		return "Screenshot"
	case ActionOpenMenu:
		return "Open Menu"
	case ActionDevMap:
		return "Dev Map"
	case ActionAction:
		return "Action"
	case ActionInteract:
		return "Interact"
	case ActionResetLevel:
		return "Reset Level"
	case ActionZoomIn:
		return "Zoom In"
	case ActionZoomOut:
		return "Zoom Out"
	default:
		return "None"
	}
}

// GetBindingsByAction returns the current bindings grouped by action.
func GetBindingsByAction() map[Action][]string {
	result := make(map[Action][]string)
	for code, act := range bindings {
		result[act] = append(result[act], code)
	}
	// Ensure stable ordering of codes within each action so UI doesn't flicker.
	for act, codes := range result {
		sort.Strings(codes)
		result[act] = codes
	}
	return result
}

// SetSingleBinding replaces all bindings for the given action with a single code.
func SetSingleBinding(action Action, code string) {
	// Remove any existing code mapped to this action
	for c, a := range bindings {
		// Always keep the core arrow-key bindings so they can't be remapped away.
		if c == "arrow_up" || c == "arrow_down" || c == "arrow_left" || c == "arrow_right" {
			continue
		}
		// Keep reserved interaction bindings (E / Enter / gamepad A)
		if c == "e" || c == "enter" || c == "gamepad_a" {
			continue
		}
		// Don't allow reserved actions themselves to have their bindings cleared
		if a == ActionAction || a == ActionInteract {
			continue
		}
		if a == action {
			delete(bindings, c)
		}
	}
	// Don't allow arrows or reserved interaction codes to be rebound through the menu – they are reserved.
	if code != "" &&
		code != "arrow_up" && code != "arrow_down" &&
		code != "arrow_left" && code != "arrow_right" &&
		code != "e" && code != "enter" && code != "gamepad_a" {
		bindings[code] = action
	}
}
