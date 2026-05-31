// Package ebiten provides binding capture for the rebinding menu.
package ebiten

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	engineinput "darkstation/pkg/engine/input"
)

func (e *EbitenRenderer) setBindingCapture(active bool) {
	e.bindingCaptureMutex.Lock()
	e.bindingCaptureActive = active
	if !active {
		e.bindingCaptureStick = make(map[ebiten.GamepadID]struct{ x, y float64 })
	}
	e.bindingCaptureMutex.Unlock()
}

func (e *EbitenRenderer) isBindingCaptureActive() bool {
	e.bindingCaptureMutex.Lock()
	defer e.bindingCaptureMutex.Unlock()
	return e.bindingCaptureActive
}

// CaptureBindingCode blocks until the player presses a key or controller button.
// Returns an empty string when capture is cancelled (Escape / B).
func (e *EbitenRenderer) CaptureBindingCode() string {
	e.setBindingCapture(true)
	defer e.setBindingCapture(false)

	intent := <-e.inputChan
	if intent.Code != "" {
		return intent.Code
	}
	return bindingCodeFromIntent(intent)
}

func (e *EbitenRenderer) pollBindingCapture() (code string, done bool) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return "", true
	}
	if code := pollKeyboardBindingCapture(); code != "" {
		return code, true
	}
	return pollGamepadBindingCapture(e)
}

func pollKeyboardBindingCapture() string {
	for key := ebiten.Key(0); key <= ebiten.KeyMax; key++ {
		if !inpututil.IsKeyJustPressed(key) {
			continue
		}
		if code := ebitenKeyToBindingCode(key); code != "" {
			return code
		}
	}
	return ""
}

func pollGamepadBindingCapture(e *EbitenRenderer) (string, bool) {
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		if inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton1) {
			return "", true
		}
		for btn := ebiten.GamepadButton0; btn <= ebiten.GamepadButtonMax; btn++ {
			if inpututil.IsGamepadButtonJustPressed(id, btn) {
				if code := gamepadButtonToBindingCode(btn); code != "" {
					return code, true
				}
			}
		}
		if code := pollGamepadDirectionCapture(e, id); code != "" {
			return code, true
		}
	}
	return "", false
}

func pollGamepadDirectionCapture(e *EbitenRenderer, id ebiten.GamepadID) string {
	const deadZone = 0.5
	stickX := ebiten.GamepadAxisValue(id, 0)
	stickY := ebiten.GamepadAxisValue(id, 1)

	prev, ok := e.bindingCaptureStick[id]
	if !ok {
		e.bindingCaptureStick[id] = struct{ x, y float64 }{stickX, stickY}
	}
	defer func() {
		e.bindingCaptureStick[id] = struct{ x, y float64 }{stickX, stickY}
	}()

	if !ok || prev.x >= -deadZone {
		if stickX < -deadZone {
			return "gamepad_dpad_left"
		}
	}
	if !ok || prev.x <= deadZone {
		if stickX > deadZone {
			return "gamepad_dpad_right"
		}
	}
	if !ok || prev.y >= -deadZone {
		if stickY < -deadZone {
			return "gamepad_dpad_up"
		}
	}
	if !ok || prev.y <= deadZone {
		if stickY > deadZone {
			return "gamepad_dpad_down"
		}
	}

	for _, spec := range []struct {
		btn  ebiten.GamepadButton
		code string
	}{
		{ebiten.GamepadButton14, "gamepad_dpad_left"},
		{ebiten.GamepadButton12, "gamepad_dpad_right"},
		{ebiten.GamepadButton11, "gamepad_dpad_up"},
		{ebiten.GamepadButton13, "gamepad_dpad_down"},
	} {
		if inpututil.IsGamepadButtonJustPressed(id, spec.btn) {
			return spec.code
		}
	}
	return ""
}

func gamepadButtonToBindingCode(btn ebiten.GamepadButton) string {
	switch btn {
	case ebiten.GamepadButton0:
		return "gamepad_a"
	case ebiten.GamepadButton1:
		return "gamepad_b"
	case ebiten.GamepadButton2:
		return "gamepad_x"
	case ebiten.GamepadButton3:
		return "gamepad_y"
	case ebiten.GamepadButton4:
		return "gamepad_lb"
	case ebiten.GamepadButton5:
		return "gamepad_rb"
	case ebiten.GamepadButton6:
		return "gamepad_back"
	case ebiten.GamepadButton7:
		return "gamepad_start"
	case ebiten.GamepadButton8:
		return "gamepad_ls"
	case ebiten.GamepadButton9:
		return "gamepad_rs"
	default:
		return ""
	}
}

func ebitenKeyToBindingCode(key ebiten.Key) string {
	switch key {
	case ebiten.KeyArrowUp:
		return "arrow_up"
	case ebiten.KeyArrowDown:
		return "arrow_down"
	case ebiten.KeyArrowLeft:
		return "arrow_left"
	case ebiten.KeyArrowRight:
		return "arrow_right"
	case ebiten.KeyEnter, ebiten.KeyKPEnter:
		return "enter"
	case ebiten.KeyEscape:
		return "escape"
	case ebiten.KeyEqual, ebiten.KeyNumpadAdd:
		return "+"
	case ebiten.KeyMinus, ebiten.KeyNumpadSubtract:
		return "-"
	case ebiten.KeySlash:
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			return "?"
		}
		return "/"
	case ebiten.KeyF5:
		return "f5"
	case ebiten.KeyF8:
		return "f8"
	case ebiten.KeyF9:
		return "f9"
	case ebiten.KeyF10:
		return "menu"
	case ebiten.KeyTab:
		return "tab"
	case ebiten.Key1, ebiten.KeyNumpad1:
		return "1"
	case ebiten.Key2, ebiten.KeyNumpad2:
		return "2"
	case ebiten.Key3, ebiten.KeyNumpad3:
		return "3"
	default:
		name := key.String()
		if strings.HasPrefix(name, "Key") && len(name) == 4 {
			return strings.ToLower(name[3:])
		}
		if strings.HasPrefix(name, "Digit") && len(name) == 6 {
			return strings.ToLower(name[5:])
		}
		return strings.ToLower(name)
	}
}

func bindingCodeFromIntent(intent engineinput.Intent) string {
	switch intent.Action {
	case engineinput.ActionMoveNorth:
		return "arrow_up"
	case engineinput.ActionMoveSouth:
		return "arrow_down"
	case engineinput.ActionMoveWest:
		return "arrow_left"
	case engineinput.ActionMoveEast:
		return "arrow_right"
	case engineinput.ActionHint:
		return "?"
	case engineinput.ActionQuit:
		return "quit"
	case engineinput.ActionScreenshot:
		return "screenshot"
	case engineinput.ActionAction, engineinput.ActionInteract:
		return "enter"
	default:
		return ""
	}
}
