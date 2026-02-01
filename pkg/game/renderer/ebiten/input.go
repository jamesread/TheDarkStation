// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
package ebiten

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/game/config"
)

// Update handles input and game logic (Ebiten interface)
func (e *EbitenRenderer) Update() error {
	// Log window opening on first update (confirms window is actually running)
	if !e.windowOpenedLogged {
		e.windowOpenedLogged = true
		w, h := ebiten.WindowSize()
		log.Printf("Main window opened successfully (%dx%d)", w, h)
	}

	// Check for console toggle (backtick/grave accent or Shift+:`)
	// Ebiten uses KeyGraveAccent for backtick (`)
	if inpututil.IsKeyJustPressed(ebiten.KeyGraveAccent) {
		e.ToggleConsole()
	}
	// Shift+:` (semicolon key with shift produces colon)
	if inpututil.IsKeyJustPressed(ebiten.KeySemicolon) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		e.ToggleConsole()
	}

	// Handle console input if console is active
	if e.IsConsoleActive() {
		e.HandleConsoleInput()
		// Don't process other input when console is open
		return nil
	}

	// Update floating tiles animation if main menu is active
	// Step 2: Enable update loop (tiles move but not drawn yet)
	e.genericMenuMutex.RLock()
	genericMenuActive := e.genericMenuActive
	title := e.genericMenuTitle
	e.genericMenuMutex.RUnlock()

	if genericMenuActive && title == "The Dark Station" {
		w, h := ebiten.WindowSize()
		if w > 0 && h > 0 {
			e.updateFloatingTiles(w, h)
		}
	}

	// Handle font size changes (Ctrl+= to increase, Ctrl+- to decrease)
	e.handleZoom()

	// Check for gamepad input first, then fall back to keyboard (raw layer)
	if intent := e.checkGamepadInput(); intent.Action != engineinput.ActionNone {
		// Non-blocking send to input channel
		select {
		case e.inputChan <- intent:
		default:
			// Channel full, drop input
		}
	} else if intent := e.checkInput(); intent.Action != engineinput.ActionNone {
		// Non-blocking send to input channel
		select {
		case e.inputChan <- intent:
		default:
			// Channel full, drop input
		}
	}

	return nil
}

// handleZoom handles =/- for font/tile size adjustment
func (e *EbitenRenderer) handleZoom() {
	// = or + to increase font size
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) {
		e.increaseTileSize()
	}
	// - to decrease font size
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) {
		e.decreaseTileSize()
	}
	// 0 to reset font size
	if inpututil.IsKeyJustPressed(ebiten.Key0) || inpututil.IsKeyJustPressed(ebiten.KeyNumpad0) {
		e.resetTileSize()
	}
}

// increaseTileSize increases the tile/font size
func (e *EbitenRenderer) increaseTileSize() {
	if e.tileSize < maxTileSize {
		e.tileSize += tileSizeStep
		e.recalculateViewport()
		e.saveZoomPreference()
	}
}

// decreaseTileSize decreases the tile/font size
func (e *EbitenRenderer) decreaseTileSize() {
	if e.tileSize > minTileSize {
		e.tileSize -= tileSizeStep
		e.recalculateViewport()
		e.saveZoomPreference()
	}
}

// resetTileSize resets tile size to default
func (e *EbitenRenderer) resetTileSize() {
	e.tileSize = 24
	e.recalculateViewport()
	e.saveZoomPreference()
}

// saveZoomPreference saves the current tile size to preferences
func (e *EbitenRenderer) saveZoomPreference() {
	cfg := config.Current()
	if err := cfg.SetTileSize(e.tileSize); err != nil {
		// Silently ignore save errors - not critical
		fmt.Fprintf(os.Stderr, "Warning: could not save preferences: %v\n", err)
	}
}

// recalculateViewport recalculates viewport dimensions based on current window and tile size
func (e *EbitenRenderer) recalculateViewport() {
	// Invalidate font cache since sizes may have changed
	e.invalidateFontCache()

	// Get current window size
	w, h := ebiten.WindowSize()
	if w == 0 || h == 0 {
		w, h = e.windowWidth, e.windowHeight
	}

	// Calculate available space for the map (accounting for UI elements)
	// Header height + small frame border
	uiFontSize := e.getUIFontSize()
	headerHeight := int(uiFontSize) + 20
	frameBorder := 10
	availableHeight := h - headerHeight - frameBorder*2
	availableWidth := w - frameBorder*2

	// Calculate viewport dimensions to maximize the map
	e.viewportCols = availableWidth / e.tileSize
	e.viewportRows = availableHeight / e.tileSize

	// Ensure minimum viewport size
	if e.viewportCols < 15 {
		e.viewportCols = 15
	}
	if e.viewportRows < 11 {
		e.viewportRows = 11
	}

	// Keep odd numbers for centering
	if e.viewportCols%2 == 0 {
		e.viewportCols--
	}
	if e.viewportRows%2 == 0 {
		e.viewportRows--
	}
}

// shouldRepeatKey checks if a key/button should trigger (initial press or repeat)
// Returns true if the key should trigger, false otherwise
func (e *EbitenRenderer) shouldRepeatKey(isPressed func() bool, code string) bool {
	now := time.Now().UnixMilli()

	e.keyRepeatStateMutex.Lock()
	defer e.keyRepeatStateMutex.Unlock()

	pressed := isPressed()
	state, exists := e.keyRepeatState[code]

	if pressed {
		if !exists {
			// First press - record it and trigger immediately
			e.keyRepeatState[code] = keyRepeatInfo{
				firstPressed: now,
				lastRepeat:   now,
			}
			return true
		}

		// Key is held - check if we should repeat
		timeSinceFirstPress := now - state.firstPressed
		timeSinceLastRepeat := now - state.lastRepeat

		if timeSinceFirstPress >= keyRepeatInitialDelay {
			// Initial delay has passed, check repeat interval
			if timeSinceLastRepeat >= keyRepeatInterval {
				// Update last repeat time and trigger
				state.lastRepeat = now
				e.keyRepeatState[code] = state
				return true
			}
		}
		return false
	} else {
		// Key released - clean up state
		if exists {
			delete(e.keyRepeatState, code)
		}
		return false
	}
}

// checkGamepadInput checks for controller/gamepad input and returns the corresponding Intent.
// NOTE: Button indices here are tuned for common XInput-style controllers on Linux;
// mappings may vary between devices/platforms.
func (e *EbitenRenderer) checkGamepadInput() engineinput.Intent {
	// Collect currently connected gamepads
	var ids []ebiten.GamepadID
	ids = ebiten.AppendGamepadIDs(ids[:0])

	for _, id := range ids {
		// Analog stick (left stick) movement
		// Axes: 0 = X (left = -1, right = +1), 1 = Y (up = -1, down = +1)
		const deadZone = 0.5 // Threshold to avoid drift
		const axisX = 0
		const axisY = 1

		stickX := ebiten.GamepadAxisValue(id, axisX)
		stickY := ebiten.GamepadAxisValue(id, axisY)

		// Check horizontal movement (left/right) with key repeat
		stickCodeLeft := fmt.Sprintf("gamepad_%d_stick_left", id)
		if stickX < -deadZone {
			if e.shouldRepeatKey(func() bool { return stickX < -deadZone }, stickCodeLeft) {
				return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
					Device: engineinput.DeviceGamepad,
					Code:   "gamepad_dpad_left",
				}))
			}
		}
		stickCodeRight := fmt.Sprintf("gamepad_%d_stick_right", id)
		if stickX > deadZone {
			if e.shouldRepeatKey(func() bool { return stickX > deadZone }, stickCodeRight) {
				return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
					Device: engineinput.DeviceGamepad,
					Code:   "gamepad_dpad_right",
				}))
			}
		}

		// Check vertical movement (up/down) with key repeat
		stickCodeUp := fmt.Sprintf("gamepad_%d_stick_up", id)
		if stickY < -deadZone {
			if e.shouldRepeatKey(func() bool { return stickY < -deadZone }, stickCodeUp) {
				return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
					Device: engineinput.DeviceGamepad,
					Code:   "gamepad_dpad_up",
				}))
			}
		}
		stickCodeDown := fmt.Sprintf("gamepad_%d_stick_down", id)
		if stickY > deadZone {
			if e.shouldRepeatKey(func() bool { return stickY > deadZone }, stickCodeDown) {
				return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
					Device: engineinput.DeviceGamepad,
					Code:   "gamepad_dpad_down",
				}))
			}
		}

		// Directional pad (D‑pad) movement with key repeat
		// Typical mapping on many XInput-style controllers under Ebiten:
		//  - Up:    11
		//  - Right: 12
		//  - Down:  13
		//  - Left:  14
		code := fmt.Sprintf("gamepad_%d_14", id)
		if e.shouldRepeatKey(func() bool { return ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton14) }, code) {
			return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
				Device: engineinput.DeviceGamepad,
				Code:   "gamepad_dpad_left",
			}))
		}
		code = fmt.Sprintf("gamepad_%d_12", id)
		if e.shouldRepeatKey(func() bool { return ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton12) }, code) {
			return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
				Device: engineinput.DeviceGamepad,
				Code:   "gamepad_dpad_right",
			}))
		}
		code = fmt.Sprintf("gamepad_%d_11", id)
		if e.shouldRepeatKey(func() bool { return ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton11) }, code) {
			return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
				Device: engineinput.DeviceGamepad,
				Code:   "gamepad_dpad_up",
			}))
		}
		code = fmt.Sprintf("gamepad_%d_13", id)
		if e.shouldRepeatKey(func() bool { return ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton13) }, code) {
			return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
				Device: engineinput.DeviceGamepad,
				Code:   "gamepad_dpad_down",
			}))
		}

		// Face buttons:
		// - A: show help / hint
		// - B: quit game
		// Typical mapping:
		//  - A / Cross: 0
		//  - B / Circle: 1
		if inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton0) {
			return engineinput.Intent{Action: engineinput.ActionInteract}
		}
		if inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton1) {
			return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
				Device: engineinput.DeviceGamepad,
				Code:   "gamepad_b",
			}))
		}

		// Start button opens the bindings/menu.
		// Typical mapping:
		//  - Start: 7
		if inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton7) {
			return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
				Device: engineinput.DeviceGamepad,
				Code:   "gamepad_start",
			}))
		}
	}

	return engineinput.Intent{Action: engineinput.ActionNone}
}

// checkInput checks for keyboard input and returns the corresponding Intent.
func (e *EbitenRenderer) checkInput() engineinput.Intent {
	// Arrow keys / NSEW navigation with key repeat
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyArrowUp) }, "key_arrow_up") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "arrow_up",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyArrowDown) }, "key_arrow_down") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "arrow_down",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyArrowLeft) }, "key_arrow_left") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "arrow_left",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyArrowRight) }, "key_arrow_right") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "arrow_right",
		}))
	}

	// WASD navigation (as arrow alternatives) with key repeat
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyW) }, "key_w") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "arrow_up",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyS) && !ebiten.IsKeyPressed(ebiten.KeyControl) }, "key_s") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "arrow_down",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyA) }, "key_a") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "arrow_left",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyD) }, "key_d") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "arrow_right",
		}))
	}

	// Vim-style navigation with key repeat
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyK) }, "key_k") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "k",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyJ) }, "key_j") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "j",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyH) }, "key_h") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "h",
		}))
	}
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyL) }, "key_l") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "l",
		}))
	}

	// NSEW keys with key repeat
	if e.shouldRepeatKey(func() bool { return ebiten.IsKeyPressed(ebiten.KeyN) }, "key_n") {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "n",
		}))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		return engineinput.Intent{Action: engineinput.ActionInteract}
	}

	// Help
	if inpututil.IsKeyJustPressed(ebiten.KeySlash) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "?",
		}))
	}

	// Interaction (Enter)
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) {
		return engineinput.Intent{Action: engineinput.ActionInteract}
	}

	// Open menu (F10)
	if inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "f9",
		}))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF10) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "menu",
		}))
	}

	// Debug map dump (F8)
	if inpututil.IsKeyJustPressed(ebiten.KeyF8) {
		return engineinput.Intent{Action: engineinput.ActionDebugMapDump}
	}

	// Reset level (F5)
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		return engineinput.Intent{Action: engineinput.ActionResetLevel}
	}

	// Quit
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "quit",
		}))
	}

	return engineinput.Intent{Action: engineinput.ActionNone}
}

// Layout returns the game's logical screen size (Ebiten interface)
func (e *EbitenRenderer) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Recalculate viewport when window size changes
	if outsideWidth != e.windowWidth || outsideHeight != e.windowHeight {
		e.windowWidth = outsideWidth
		e.windowHeight = outsideHeight
		e.recalculateViewport()
	}
	return outsideWidth, outsideHeight
}
