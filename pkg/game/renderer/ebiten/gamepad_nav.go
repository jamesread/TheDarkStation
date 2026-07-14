package ebiten

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	engineinput "darkstation/pkg/engine/input"
)

const (
	gamepadStickEngageThreshold  = 0.58
	gamepadStickReleaseThreshold = 0.42
	gamepadStickCenterThreshold  = 0.22
)

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
