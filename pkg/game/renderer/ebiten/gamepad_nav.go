package ebiten

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	engineinput "darkstation/pkg/engine/input"
)

const (
	gamepadStickEngageThreshold  = 0.58
	gamepadStickReleaseThreshold = 0.42
	gamepadStickCenterThreshold  = 0.22

	// Unmapped HID devices must look like a real pad: a 2D stick plus a few buttons.
	// A single resting axis at -1 (common for volume knobs) must not drive MoveWest.
	minUnmappedGamepadAxes    = 2
	minUnmappedGamepadButtons = 4
)

// nonGamepadDeviceNameNeedles matches GLFW "gamepads" that are not controllers.
// On Linux, touchpads and USB "Consumer Control" knobs often appear as joysticks;
// a resting axis at -1 is then treated as left-stick held.
var nonGamepadDeviceNameNeedles = []string{
	"touchpad",
	"trackpad",
	"trackpoint",
	"touchscreen",
	"touch screen",
	"mouse",
	"pointer",
	"consumer control",
	"system control",
	"keyboard",
	"headset",
	"microphone",
	"sound device",
	"stream deck",
}

// isNonGamepadDeviceName reports whether a GLFW gamepad name is a non-controller HID device.
func isNonGamepadDeviceName(name string) bool {
	n := strings.ToLower(name)
	for _, needle := range nonGamepadDeviceNameNeedles {
		if strings.Contains(n, needle) {
			return true
		}
	}
	return false
}

// isPointerDeviceGamepadName reports whether a GLFW gamepad name is a pointer device.
// Kept as a thin wrapper for existing tests and call sites.
func isPointerDeviceGamepadName(name string) bool {
	return isNonGamepadDeviceName(name)
}

// isUsableGamepadCandidate reports whether a connected joystick should drive game input.
// Standard-mapped controllers are accepted unless the name is a known non-gamepad HID.
// Unmapped devices need a plausible stick + button count so volume knobs and similar
// 1-axis HID controls (often resting at -1) cannot spam MoveWest on launch.
func isUsableGamepadCandidate(name string, standardLayout bool, axisCount, buttonCount int) bool {
	if isNonGamepadDeviceName(name) {
		return false
	}
	if standardLayout {
		return true
	}
	return axisCount >= minUnmappedGamepadAxes && buttonCount >= minUnmappedGamepadButtons
}

// isUsableGamepad reports whether id should drive gamepad navigation / bindings.
func isUsableGamepad(id ebiten.GamepadID) bool {
	return isUsableGamepadCandidate(
		ebiten.GamepadName(id),
		ebiten.IsStandardGamepadLayoutAvailable(id),
		ebiten.GamepadAxisCount(id),
		ebiten.GamepadButtonCount(id),
	)
}

// appendUsableGamepadIDs appends connected controllers, skipping non-gamepad HID
// devices that GLFW exposes as joysticks (touchpads, mic volume knobs, etc.).
func appendUsableGamepadIDs(gamepadIDs []ebiten.GamepadID) []ebiten.GamepadID {
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		if isUsableGamepad(id) {
			gamepadIDs = append(gamepadIDs, id)
		}
	}
	return gamepadIDs
}

// stickNavDirection resolves the left stick to a single navigation direction with hysteresis
// so analog noise around the dead zone does not retrigger movement every frame.
func stickNavDirection(x, y float64, current string) string {
	ax, ay := abs(x), abs(y)
	if current == "" {
		if ax < gamepadStickEngageThreshold && ay < gamepadStickEngageThreshold {
			return ""
		}
		return dominantStickDirection(x, y, gamepadStickEngageThreshold)
	}

	if ax < gamepadStickCenterThreshold && ay < gamepadStickCenterThreshold {
		return ""
	}

	if dir := directionHeld(x, y, current, gamepadStickReleaseThreshold); dir != "" {
		return dir
	}
	if ax >= gamepadStickEngageThreshold || ay >= gamepadStickEngageThreshold {
		return dominantStickDirection(x, y, gamepadStickEngageThreshold)
	}
	return ""
}

func directionHeld(x, y float64, current string, threshold float64) string {
	switch current {
	case "up":
		if y < -threshold {
			return "up"
		}
	case "down":
		if y > threshold {
			return "down"
		}
	case "left":
		if x < -threshold {
			return "left"
		}
	case "right":
		if x > threshold {
			return "right"
		}
	}
	return ""
}

func dominantStickDirection(x, y float64, threshold float64) string {
	ax, ay := abs(x), abs(y)
	if ax >= ay {
		switch {
		case x < -threshold:
			return "left"
		case x > threshold:
			return "right"
		}
	} else {
		switch {
		case y < -threshold:
			return "up"
		case y > threshold:
			return "down"
		}
	}
	return ""
}

func gamepadDPadDirection(id ebiten.GamepadID) string {
	if inputPressed(id, ebiten.GamepadButton11) || standardPressed(id, ebiten.StandardGamepadButtonLeftTop) {
		return "up"
	}
	if inputPressed(id, ebiten.GamepadButton13) || standardPressed(id, ebiten.StandardGamepadButtonLeftBottom) {
		return "down"
	}
	if inputPressed(id, ebiten.GamepadButton14) || standardPressed(id, ebiten.StandardGamepadButtonLeftLeft) {
		return "left"
	}
	if inputPressed(id, ebiten.GamepadButton12) || standardPressed(id, ebiten.StandardGamepadButtonLeftRight) {
		return "right"
	}
	return ""
}

func inputPressed(id ebiten.GamepadID, btn ebiten.GamepadButton) bool {
	return ebiten.IsGamepadButtonPressed(id, btn)
}

func standardPressed(id ebiten.GamepadID, btn ebiten.StandardGamepadButton) bool {
	return ebiten.IsStandardGamepadButtonPressed(id, btn)
}

func gamepadNavBindingCode(dir string) string {
	switch dir {
	case "up":
		return "gamepad_dpad_up"
	case "down":
		return "gamepad_dpad_down"
	case "left":
		return "gamepad_dpad_left"
	case "right":
		return "gamepad_dpad_right"
	default:
		return ""
	}
}

func isMovementIntent(intent engineinput.Intent) bool {
	switch intent.Action {
	case engineinput.ActionMoveNorth, engineinput.ActionMoveSouth, engineinput.ActionMoveEast, engineinput.ActionMoveWest:
		return true
	default:
		return false
	}
}

func (e *EbitenRenderer) pollGamepadNavigation(id ebiten.GamepadID) engineinput.Intent {
	dir := gamepadDPadDirection(id)
	if dir == "" {
		e.gamepadNavMutex.Lock()
		current := e.gamepadNavDir[id]
		e.gamepadNavMutex.Unlock()

		stickX := ebiten.GamepadAxisValue(id, 0)
		stickY := ebiten.GamepadAxisValue(id, 1)
		dir = stickNavDirection(stickX, stickY, current)
	}

	e.gamepadNavMutex.Lock()
	if dir == "" {
		delete(e.gamepadNavDir, id)
	} else {
		e.gamepadNavDir[id] = dir
	}
	e.gamepadNavMutex.Unlock()

	if dir == "" {
		e.clearGamepadNavRepeat(id)
		return engineinput.Intent{Action: engineinput.ActionNone}
	}

	code := gamepadNavBindingCode(dir)
	repeatKey := fmt.Sprintf("gamepad_%d_nav_%s", id, dir)
	held := func() bool {
		if gamepadDPadDirection(id) == dir {
			return true
		}
		e.gamepadNavMutex.Lock()
		current := e.gamepadNavDir[id]
		e.gamepadNavMutex.Unlock()
		stickX := ebiten.GamepadAxisValue(id, 0)
		stickY := ebiten.GamepadAxisValue(id, 1)
		return stickNavDirection(stickX, stickY, current) == dir
	}

	if !e.shouldRepeatKeyWithTiming(held, repeatKey, keyRepeatInitialDelay, keyRepeatInterval) {
		return engineinput.Intent{Action: engineinput.ActionNone}
	}
	return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
		Device: engineinput.DeviceGamepad,
		Code:   code,
	}))
}

func (e *EbitenRenderer) clearGamepadNavRepeat(id ebiten.GamepadID) {
	prefix := fmt.Sprintf("gamepad_%d_nav_", id)
	e.keyRepeatStateMutex.Lock()
	defer e.keyRepeatStateMutex.Unlock()
	for key := range e.keyRepeatState {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(e.keyRepeatState, key)
		}
	}
}
