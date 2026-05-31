// Package ebiten provides input activity detection for primary device switching.
package ebiten

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	engineinput "darkstation/pkg/engine/input"
)

func (e *EbitenRenderer) pollPrimaryDeviceActivity() {
	kb := hasKeyboardActivityNew()
	gp := e.hasGamepadActivityNew()
	switch {
	case gp:
		e.notePrimaryDevice(engineinput.DeviceGamepad)
	case kb:
		e.notePrimaryDevice(engineinput.DeviceKeyboard)
	}
}

func (e *EbitenRenderer) notePrimaryDevice(device engineinput.Device) {
	if !engineinput.NoteDeviceActivity(device) {
		return
	}
	e.showInputDeviceNotification()
	if e.isBindingCaptureActive() || e.isGenericMenuActive() {
		return
	}
	e.gameMutex.RLock()
	g := e.game
	refresher := e.hintRefresher
	e.gameMutex.RUnlock()
	if g == nil || g.CurrentCell == nil || refresher == nil {
		return
	}
	refresher(g)
}

func hasKeyboardActivityNew() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeyE) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyTab) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
		inpututil.IsKeyJustPressed(ebiten.KeyQ) ||
		inpututil.IsKeyJustPressed(ebiten.KeyF5) ||
		inpututil.IsKeyJustPressed(ebiten.KeyF8) ||
		inpututil.IsKeyJustPressed(ebiten.KeyF9) ||
		inpututil.IsKeyJustPressed(ebiten.KeyF10) {
		return true
	}
	if inpututil.IsKeyJustPressed(ebiten.Key1) || inpututil.IsKeyJustPressed(ebiten.KeyDigit1) || inpututil.IsKeyJustPressed(ebiten.KeyNumpad1) ||
		inpututil.IsKeyJustPressed(ebiten.Key2) || inpututil.IsKeyJustPressed(ebiten.KeyDigit2) || inpututil.IsKeyJustPressed(ebiten.KeyNumpad2) ||
		inpututil.IsKeyJustPressed(ebiten.Key3) || inpututil.IsKeyJustPressed(ebiten.KeyDigit3) || inpututil.IsKeyJustPressed(ebiten.KeyNumpad3) {
		return true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySlash) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		return true
	}

	movementKeys := []ebiten.Key{
		ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight,
		ebiten.KeyW, ebiten.KeyA, ebiten.KeyS, ebiten.KeyD,
		ebiten.KeyK, ebiten.KeyJ, ebiten.KeyH, ebiten.KeyL,
		ebiten.KeyN,
	}
	for _, k := range movementKeys {
		if inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return false
}

func (e *EbitenRenderer) hasGamepadActivityNew() bool {
	const deadZone = 0.5
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		for btn := ebiten.GamepadButton0; btn <= ebiten.GamepadButtonMax; btn++ {
			if inpututil.IsGamepadButtonJustPressed(id, btn) {
				return true
			}
		}
		stickX := ebiten.GamepadAxisValue(id, 0)
		stickY := ebiten.GamepadAxisValue(id, 1)
		active := abs(stickX) > deadZone || abs(stickY) > deadZone
		e.stickStateMutex.RLock()
		prev, ok := e.stickState[id]
		e.stickStateMutex.RUnlock()
		wasActive := ok && (abs(prev.x) > deadZone || abs(prev.y) > deadZone)
		if active && !wasActive {
			return true
		}
	}
	return false
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
