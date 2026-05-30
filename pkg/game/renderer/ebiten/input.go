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
	// Single clock for menu overlays this tick (drawGenericMenuOverlay must not call time.Now).
	now := time.Now()
	e.menuAnimClockMilli = now.UnixMilli()
	e.menuAnimTimeNano = now.UnixNano()
	e.maintPanDrawCount = 0

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

	// Text input dialog (blocks game intents while open)
	if e.isTextInputDialogActive() {
		e.handleTextInputDialogInput()
		return nil
	}

	// Confirmation dialog (blocks game intents while open)
	if e.isConfirmDialogActive() {
		e.handleConfirmDialogInput()
		return nil
	}

	// Update floating tiles for main menu and completion screens.
	if e.floatingTilesAnimationActive() {
		w, h := ebiten.WindowSize()
		if w > 0 && h > 0 {
			e.ensureFloatingTiles(w, h)
			e.updateFloatingTiles(w, h)
		}
	}

	// Handle font size changes (Ctrl+= to increase, Ctrl+- to decrease)
	e.handleZoom()

	// Track hold state before dispatching interact so the long-use loop sees the key down on the same frame.
	e.trackInteractHold()

	// Keyboard before gamepad so E/Enter is not lost to stick drift or held movement.
	// (See checkInput: interact is also handled before WASD movement within keyboard.)
	if intent := e.checkInput(); intent.Action != engineinput.ActionNone {
		if intent.Action == engineinput.ActionInteract {
			log.Printf("[Interact] input: dispatch ActionInteract via keyboard path")
		}
		select {
		case e.inputChan <- intent:
		default:
			log.Printf("[Interact] input: WARNING dropped intent (inputChan full) action=%v", intent.Action)
		}
	} else if intent := e.checkGamepadInput(); intent.Action != engineinput.ActionNone {
		if intent.Action == engineinput.ActionInteract {
			log.Printf("[Interact] input: dispatch ActionInteract via gamepad path")
		}
		select {
		case e.inputChan <- intent:
		default:
			log.Printf("[Interact] input: WARNING dropped intent (inputChan full) action=%v", intent.Action)
		}
	}

	// Camera pan for maintenance menu: once per Update tick, not per Draw (Draw can run
	// multiple times per frame; time-based interpolation there caused visible jitter).
	e.advanceMaintenanceCamera()

	e.advanceLongUseFromInput()

	e.advanceHazardClear()

	e.advanceHazardTour()

	return nil
}

func (e *EbitenRenderer) advanceLongUseFromInput() {
	if e.longUseAdvancer == nil {
		return
	}
	e.gameMutex.RLock()
	g := e.game
	e.gameMutex.RUnlock()
	if g == nil || g.LongUse == nil {
		e.longUsePrevHeld = false
		return
	}
	held := isInteractPressed()
	released := e.longUsePrevHeld && !held
	e.longUsePrevHeld = held
	e.longUseAdvancer(g, held, released, e.menuAnimClockMilli)
}

func (e *EbitenRenderer) advanceHazardClear() {
	if e.hazardClearAdvancer == nil {
		return
	}
	e.gameMutex.RLock()
	g := e.game
	e.gameMutex.RUnlock()
	if g == nil || g.HazardClear == nil {
		return
	}
	e.hazardClearAdvancer(g, e.menuAnimClockMilli)
}

func (e *EbitenRenderer) advanceHazardTour() {
	if e.hazardTourAdvancer == nil {
		return
	}
	e.gameMutex.RLock()
	g := e.game
	e.gameMutex.RUnlock()
	if g == nil || g.HazardTour == nil {
		return
	}
	e.hazardTourAdvancer(g, e.menuAnimClockMilli)
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
	e.invalidateMapDrawCache()

	w, h := e.windowWidth, e.windowHeight
	if w <= 0 || h <= 0 {
		w, h = ebiten.WindowSize()
	}
	e.syncViewportForMap(w, h)
}

// viewportTilesForAxis returns how many tile columns or rows are needed so a player-centered
// viewport covers the screen axis edge-to-edge (partial tiles at the border included).
func viewportTilesForAxis(screenPx, tileSize int) int {
	if tileSize <= 0 || screenPx <= 0 {
		return 1
	}
	if screenPx <= tileSize {
		return 1
	}
	half := screenPx / 2
	span := (half + tileSize - 1) / tileSize // tiles from screen center to nearest edge
	n := span*2 + 1
	if n%2 == 0 {
		n++
	}
	return n
}

// syncViewportForMap sets e.viewportCols/Rows from the full window size and tile size.
// Odd dimensions keep the player on a center tile; Layout() calls this via recalculateViewport.
func (e *EbitenRenderer) syncViewportForMap(screenWidth, screenHeight int) {
	if e.tileSize <= 0 {
		return
	}
	e.viewportCols = viewportTilesForAxis(screenWidth, e.tileSize)
	e.viewportRows = viewportTilesForAxis(screenHeight, e.tileSize)
}

// mapTileGridOrigin returns the top-left pixel where the tile grid is blitted so the player
// (viewport center) aligns with the window center. The map draw area is the full window.
func mapTileGridOrigin(screenWidth, screenHeight, contentWidth, contentHeight int) (int, int) {
	return (screenWidth - contentWidth) / 2, (screenHeight - contentHeight) / 2
}

// mapCameraScreenOrigin places the map so the camera center sits on the screen center.
func mapCameraScreenOrigin(screenWidth, screenHeight int, cameraRow, cameraCol float64, startRow, startCol, tileSize int) (float64, float64) {
	subCol := cameraCol - float64(startCol)
	subRow := cameraRow - float64(startRow)
	half := float64(tileSize) / 2
	mapScrX := float64(screenWidth)/2 - subCol*float64(tileSize) - half
	mapScrY := float64(screenHeight)/2 - subRow*float64(tileSize) - half
	return mapScrX, mapScrY
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
		// Face buttons and Start before analog sticks so A (interact) is not lost to drift/hold.

		// Face buttons:
		// - A / Cross: interact
		// - B / Circle: back in menus; quit (with confirmation) during gameplay
		if inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton0) {
			return engineinput.Intent{Action: engineinput.ActionInteract}
		}
		if inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton1) {
			return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
				Device: engineinput.DeviceGamepad,
				Code:   "gamepad_b",
			}))
		}

		// Start opens menu
		if inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton7) {
			return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
				Device: engineinput.DeviceGamepad,
				Code:   "gamepad_start",
			}))
		}

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
	}

	return engineinput.Intent{Action: engineinput.ActionNone}
}

// checkInput checks for keyboard input and returns the corresponding Intent.
func (e *EbitenRenderer) checkInput() engineinput.Intent {
	// Interact must win over held movement keys; otherwise walking into range of a generator
	// while holding WASD only produces movement intents and E/Enter is never reached.
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		return engineinput.Intent{Action: engineinput.ActionInteract}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) {
		return engineinput.Intent{Action: engineinput.ActionInteract}
	}

	// Maintenance menu shortcuts (consumed only while maintenance menu is open)
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "tab",
		}))
	}
	if inpututil.IsKeyJustPressed(ebiten.Key1) || inpututil.IsKeyJustPressed(ebiten.KeyDigit1) || inpututil.IsKeyJustPressed(ebiten.KeyNumpad1) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "1",
		}))
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) || inpututil.IsKeyJustPressed(ebiten.KeyDigit2) || inpututil.IsKeyJustPressed(ebiten.KeyNumpad2) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "2",
		}))
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) || inpututil.IsKeyJustPressed(ebiten.KeyDigit3) || inpututil.IsKeyJustPressed(ebiten.KeyNumpad3) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "3",
		}))
	}

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

	// Help
	if inpututil.IsKeyJustPressed(ebiten.KeySlash) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "?",
		}))
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "f9",
		}))
	}

	// Open menu (F10)
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

	// Back (Q) while a menu is open — same role as gamepad B / Escape in menus.
	if e.isGenericMenuActive() && inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "q",
		}))
	}

	// Quit (Escape; confirmation required during gameplay)
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return engineinput.MapToIntent(engineinput.NewDebouncedInput(engineinput.RawInput{
			Device: engineinput.DeviceKeyboard,
			Code:   "escape",
		}))
	}

	return engineinput.Intent{Action: engineinput.ActionNone}
}

func isInteractPressed() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeyE) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) {
		return true
	}
	if ebiten.IsKeyPressed(ebiten.KeyE) ||
		ebiten.IsKeyPressed(ebiten.KeyEnter) ||
		ebiten.IsKeyPressed(ebiten.KeyKPEnter) {
		return true
	}
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		if inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton0) ||
			ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton0) {
			return true
		}
	}
	return false
}

func (e *EbitenRenderer) trackInteractHold() {
	held := isInteractPressed()
	e.interactHoldMutex.Lock()
	defer e.interactHoldMutex.Unlock()
	if e.interactPrevHeld && !held {
		e.interactReleasedEdge = true
	}
	e.interactHeld = held
	e.interactPrevHeld = held
}

// PollInteractHold returns whether USE is held and whether it was released since the last poll.
func (e *EbitenRenderer) PollInteractHold() (held bool, released bool) {
	e.interactHoldMutex.Lock()
	defer e.interactHoldMutex.Unlock()
	held = e.interactHeld
	released = e.interactReleasedEdge
	e.interactReleasedEdge = false
	return held, released
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
