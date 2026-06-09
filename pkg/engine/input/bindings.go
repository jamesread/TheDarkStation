package input

import "strings"

// IsGamepadCode reports whether code is a controller binding identifier.
func IsGamepadCode(code string) bool {
	return strings.HasPrefix(code, "gamepad_")
}

// IsKeyboardCode reports whether code is a keyboard binding identifier.
func IsKeyboardCode(code string) bool {
	return code != "" && !IsGamepadCode(code)
}

// FormatBindingCode returns a player-facing label for a binding code.
func FormatBindingCode(code string) string {
	switch code {
	case "gamepad_a":
		return "A"
	case "gamepad_b":
		return "B"
	case "gamepad_x":
		return "X"
	case "gamepad_y":
		return "Y"
	case "gamepad_lb":
		return "LB"
	case "gamepad_rb":
		return "RB"
	case "gamepad_back":
		return "Back"
	case "gamepad_start":
		return "Start"
	case "gamepad_ls":
		return "L3"
	case "gamepad_rs":
		return "R3"
	case "gamepad_dpad_up":
		return "D-pad Up"
	case "gamepad_dpad_down":
		return "D-pad Down"
	case "gamepad_dpad_left":
		return "D-pad Left"
	case "gamepad_dpad_right":
		return "D-pad Right"
	case "arrow_up":
		return "Up Arrow"
	case "arrow_down":
		return "Down Arrow"
	case "arrow_left":
		return "Left Arrow"
	case "arrow_right":
		return "Right Arrow"
	case "escape":
		return "Escape"
	case "q":
		return "Q"
	case "enter":
		return "Enter"
	case "menu":
		return "F10"
	default:
		return code
	}
}

func isReservedKeyboardBinding(code string) bool {
	return code == "arrow_up" || code == "arrow_down" ||
		code == "arrow_left" || code == "arrow_right" ||
		code == "e" || code == "enter" || code == "q"
}

func isReservedGamepadBinding(code string) bool {
	return code == "gamepad_a" || code == "gamepad_b"
}

func isReservedBindingCode(code string) bool {
	if IsGamepadCode(code) {
		return isReservedGamepadBinding(code)
	}
	return isReservedKeyboardBinding(code)
}

func sameBindingDevice(a, b string) bool {
	return IsGamepadCode(a) == IsGamepadCode(b)
}
