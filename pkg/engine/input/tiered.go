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
	ActionCancel
	ActionScreenshot
	ActionOpenMenu
	ActionOpenInventory // Open run-wide inventory overlay (in-game)
	ActionAction        // Generic "action/confirm" (e.g., Enter/A)
	ActionInteract // Interact with furniture/objects (E, Enter, A button)
	ActionDevMenu  // Open developer menu (F9)
	ActionDevMap   // Switch to developer testing map (menu / console)
	ActionMaintPanTestMap
	ActionPerfTestMap
	ActionDebugMapDump // Dump revealed map to map.txt (F8)
	ActionResetLevel   // Reset current level (F5)
	ActionZoomIn       // Zoom in (increase font/tile size)
	ActionZoomOut      // Zoom out (decrease font/tile size)

	// Maintenance menu (only consumed while maintenance menu is open)
	ActionMaintModeToggle  // Tab: switch Controls / Diagnostics
	ActionCircuitOff       // 1: apply OFF preset to viewed room
	ActionCircuitEssential // reserved; not used in menu (essential mode disabled)
	ActionCircuitFull      // 2: apply ON preset (doors + CCTV)
)

// Intent is the 4th‑layer, high‑level description of what the player wants to do.
type Intent struct {
	Action Action
	Code   string // device-specific binding code (used during rebinding capture)
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

	// Quit is keyboard-only; controller buttons must not prompt for game exit.
	"escape": ActionQuit,
	// Cancel / back in menus.
	"q":         ActionCancel,
	"gamepad_b": ActionCancel,

	// Screenshot
	"screenshot": ActionScreenshot,

	// Menu
	"menu":        ActionOpenMenu,
	"f":           ActionOpenInventory,
	"f9":          ActionDevMenu,
	"f8":          ActionDebugMapDump,

	// Controller/gamepad specific bindings
	"gamepad_dpad_up":    ActionMoveNorth,
	"gamepad_dpad_down":  ActionMoveSouth,
	"gamepad_dpad_left":  ActionMoveWest,
	"gamepad_dpad_right": ActionMoveEast,

	// Interaction (E, Enter, A button)
	"e":         ActionInteract,
	"enter":     ActionInteract,
	"gamepad_a": ActionInteract, // A button / Cross
	"gamepad_y": ActionOpenInventory,

	// Zoom (fixed bindings, not rebindable)
	"=":               ActionZoomIn,
	"+":               ActionZoomIn,
	"numpad_add":      ActionZoomIn,
	"-":               ActionZoomOut,
	"numpad_subtract": ActionZoomOut,

	// Generic action/confirm inputs (reserved, not unbindable)
	"action": ActionAction,

	"gamepad_start": ActionOpenMenu, // Start button

	// Maintenance menu shortcuts (consumed only while maintenance menu is open)
	"tab":     ActionMaintModeToggle,
	"1":       ActionCircuitOff,
	"digit1":  ActionCircuitOff,
	"numpad1": ActionCircuitOff,
	"2":       ActionCircuitFull,
	"digit2":  ActionCircuitFull,
	"numpad2": ActionCircuitFull,
	"3":       ActionCircuitFull,
	"digit3":  ActionCircuitFull,
	"numpad3": ActionCircuitFull,
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
	case ActionCancel:
		return "Cancel"
	case ActionScreenshot:
		return "Screenshot"
	case ActionOpenMenu:
		return "Open Menu"
	case ActionOpenInventory:
		return "Open Inventory"
	case ActionDevMenu:
		return "Developer Menu"
	case ActionDevMap:
		return "Dev Map"
	case ActionMaintPanTestMap:
		return "Maint pan test map"
	case ActionDebugMapDump:
		return "Debug Map Dump"
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

// SetSingleBinding replaces keyboard or gamepad bindings for the given action with a single code.
// Keyboard and gamepad bindings are updated independently so players can configure both.
func SetSingleBinding(action Action, code string) {
	if code == "" || isReservedBindingCode(code) {
		return
	}
	for c, a := range bindings {
		if isReservedBindingCode(c) {
			continue
		}
		if a == ActionAction || a == ActionInteract || a == ActionCancel {
			continue
		}
		if a == action && sameBindingDevice(c, code) {
			delete(bindings, c)
		}
	}
	bindings[code] = action
}
