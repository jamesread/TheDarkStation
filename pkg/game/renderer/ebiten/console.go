// Package ebiten provides console implementation for the Ebiten renderer.
package ebiten

import (
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/game/renderer"
)

// cvarMap stores configuration variables
var cvarMap = make(map[string]string)
var cvarMutex sync.RWMutex

// initCvars initializes configuration variables on startup
func initCvars() {
	cvarMutex.Lock()
	cvarMap["version"] = renderer.Version
	if renderer.Commit != "unknown" && len(renderer.Commit) > 0 {
		cvarMap["commit"] = renderer.Commit
	}
	initColorCvarsLocked()
	cvarMutex.Unlock()
	loadColorsFromCvars()
}

// initColorCvarsLocked populates cvarMap with default color values (R,G,B,A).
// Caller must hold cvarMutex.
func initColorCvarsLocked() {
	// Ebiten renderer colors
	cvarMap["colors.background"] = "26,26,46,255"
	cvarMap["colors.map_background"] = "15,15,26,255"
	cvarMap["colors.player"] = "0,255,0,255"
	cvarMap["colors.wall.fg"] = "180,180,200,255"
	cvarMap["colors.wall.bg"] = "60,60,80,255"
	cvarMap["colors.wall.bg_powered"] = "40,80,40,255"
	cvarMap["colors.floor"] = "100,100,120,255"
	cvarMap["colors.floor_visited"] = "160,160,180,255"
	cvarMap["colors.door_locked"] = "255,255,0,255"
	cvarMap["colors.door_unlocked"] = "0,220,0,255"
	cvarMap["colors.keycard"] = "100,150,255,255"
	cvarMap["colors.item"] = "220,170,255,255"
	cvarMap["colors.battery"] = "255,200,100,255"
	cvarMap["colors.hazard"] = "255,80,80,255"
	cvarMap["colors.hazard_ctrl"] = "255,150,200,255"
	cvarMap["colors.generator_off"] = "255,100,100,255"
	cvarMap["colors.generator_on"] = "0,255,100,255"
	cvarMap["colors.terminal"] = "100,150,255,255"
	cvarMap["colors.terminal_used"] = "120,120,140,255"
	cvarMap["colors.maintenance"] = "255,165,0,255"
	cvarMap["colors.furniture"] = "255,150,255,255"
	cvarMap["colors.furniture_check"] = "200,180,100,255"
	cvarMap["colors.exit_locked"] = "255,100,100,255"
	cvarMap["colors.exit_unlocked"] = "100,255,100,255"
	cvarMap["colors.subtle"] = "120,130,180,255"
	cvarMap["colors.text"] = "200,210,245,255"
	cvarMap["colors.action"] = "180,150,250,255"
	cvarMap["colors.denied"] = "255,100,100,255"
	cvarMap["colors.panel_background"] = "30,30,50,220"
	cvarMap["colors.focus_background"] = "120,80,150,255"
	cvarMap["colors.blocked_background"] = "100,100,130,220"
	cvarMap["colors.hazard_background"] = "80,30,30,220"
	// Callout colors (used by renderer.AddCallout and gameplay)
	cvarMap["colors.callout_info"] = "200,200,255,255"
	cvarMap["colors.callout_success"] = "100,255,150,255"
	cvarMap["colors.callout_warning"] = "255,220,100,255"
	cvarMap["colors.callout_danger"] = "255,120,120,255"
	cvarMap["colors.callout_item"] = "220,170,255,255"
	cvarMap["colors.callout_generator"] = "255,100,100,255"
	cvarMap["colors.callout_generator_on"] = "0,255,100,255"
	cvarMap["colors.callout_terminal"] = "100,150,255,255"
	cvarMap["colors.callout_furniture"] = "255,150,255,255"
	cvarMap["colors.callout_furniture_checked"] = "200,180,100,255"
	cvarMap["colors.callout_hazard_ctrl"] = "0,255,255,255"
	cvarMap["colors.callout_hazard"] = "255,80,80,255"
	cvarMap["colors.callout_room"] = "180,180,220,255"
	cvarMap["colors.callout_door"] = "255,255,0,255"
	cvarMap["colors.callout_maintenance"] = "255,165,0,255"
}

// parseColorRGBA parses "R,G,B,A" into color.RGBA. Values 0-255.
func parseColorRGBA(s string) (color.RGBA, bool) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return color.RGBA{}, false
	}
	var vals [4]uint8
	for i := 0; i < 4; i++ {
		n, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil || n < 0 || n > 255 {
			return color.RGBA{}, false
		}
		vals[i] = uint8(n)
	}
	return color.RGBA{R: vals[0], G: vals[1], B: vals[2], A: vals[3]}, true
}

// loadColorsFromCvars reads all colors.* from cvarMap and assigns to the in-memory color variables.
// Ebiten renderer colors and renderer.CalloutColor* are updated.
func loadColorsFromCvars() {
	assign := func(key string, dst *color.RGBA) {
		if s, ok := getCvar(key); ok {
			if c, ok := parseColorRGBA(s); ok {
				*dst = c
			}
		}
	}
	// Ebiten package color vars (same package)
	assign("colors.background", &colorBackground)
	assign("colors.map_background", &colorMapBackground)
	assign("colors.player", &colorPlayer)
	assign("colors.wall.fg", &colorWall)
	assign("colors.wall.bg", &colorWallBg)
	assign("colors.wall.bg_powered", &colorWallBgPowered)
	assign("colors.floor", &colorFloor)
	assign("colors.floor_visited", &colorFloorVisited)
	assign("colors.door_locked", &colorDoorLocked)
	assign("colors.door_unlocked", &colorDoorUnlocked)
	assign("colors.keycard", &colorKeycard)
	assign("colors.item", &colorItem)
	assign("colors.battery", &colorBattery)
	assign("colors.hazard", &colorHazard)
	assign("colors.hazard_ctrl", &colorHazardCtrl)
	assign("colors.generator_off", &colorGeneratorOff)
	assign("colors.generator_on", &colorGeneratorOn)
	assign("colors.terminal", &colorTerminal)
	assign("colors.terminal_used", &colorTerminalUsed)
	assign("colors.maintenance", &colorMaintenance)
	assign("colors.furniture", &colorFurniture)
	assign("colors.furniture_check", &colorFurnitureCheck)
	assign("colors.exit_locked", &colorExitLocked)
	assign("colors.exit_unlocked", &colorExitUnlocked)
	assign("colors.subtle", &colorSubtle)
	assign("colors.text", &colorText)
	assign("colors.action", &colorAction)
	assign("colors.denied", &colorDenied)
	assign("colors.panel_background", &colorPanelBackground)
	assign("colors.focus_background", &colorFocusBackground)
	assign("colors.blocked_background", &colorBlockedBackground)
	assign("colors.hazard_background", &colorHazardBackground)
	assign("colors.callout_info", &ColorCalloutInfo)
	assign("colors.callout_success", &ColorCalloutSuccess)
	assign("colors.callout_warning", &ColorCalloutWarning)
	assign("colors.callout_danger", &ColorCalloutDanger)
	assign("colors.callout_item", &ColorCalloutItem)
	// Renderer package CalloutColor* (used by gameplay)
	assign("colors.callout_info", &renderer.CalloutColorInfo)
	assign("colors.callout_success", &renderer.CalloutColorSuccess)
	assign("colors.callout_warning", &renderer.CalloutColorWarning)
	assign("colors.callout_danger", &renderer.CalloutColorDanger)
	assign("colors.callout_item", &renderer.CalloutColorItem)
	assign("colors.callout_generator", &renderer.CalloutColorGenerator)
	assign("colors.callout_generator_on", &renderer.CalloutColorGeneratorOn)
	assign("colors.callout_terminal", &renderer.CalloutColorTerminal)
	assign("colors.callout_furniture", &renderer.CalloutColorFurniture)
	assign("colors.callout_furniture_checked", &renderer.CalloutColorFurnitureChecked)
	assign("colors.callout_hazard_ctrl", &renderer.CalloutColorHazardCtrl)
	assign("colors.callout_hazard", &renderer.CalloutColorHazard)
	assign("colors.callout_room", &renderer.CalloutColorRoom)
	assign("colors.callout_door", &renderer.CalloutColorDoor)
	assign("colors.callout_maintenance", &renderer.CalloutColorMaintenance)
}

// getCvar retrieves a configuration variable value
func getCvar(name string) (string, bool) {
	cvarMutex.RLock()
	defer cvarMutex.RUnlock()
	value, exists := cvarMap[name]
	return value, exists
}

// setCvar sets a configuration variable value
func setCvar(name, value string) {
	cvarMutex.Lock()
	defer cvarMutex.Unlock()
	cvarMap[name] = value
}

// ToggleConsole toggles the console open/closed state
func (e *EbitenRenderer) ToggleConsole() {
	e.consoleMutex.Lock()
	defer e.consoleMutex.Unlock()

	if e.consoleAnimating {
		// Don't toggle while animating
		return
	}

	e.consoleActive = !e.consoleActive
	e.consoleAnimating = true
	e.consoleAnimStartTime = time.Now().UnixMilli()

	if !e.consoleActive {
		// Clear input when closing
		e.consoleText = ""
		e.consoleHistoryIndex = len(e.consoleHistory)
	}
}

// IsConsoleActive returns whether the console is currently active
func (e *EbitenRenderer) IsConsoleActive() bool {
	e.consoleMutex.RLock()
	defer e.consoleMutex.RUnlock()
	return e.consoleActive || e.consoleAnimating
}

// HandleConsoleInput processes input when console is active
func (e *EbitenRenderer) HandleConsoleInput() {
	e.consoleMutex.Lock()
	defer e.consoleMutex.Unlock()

	if !e.consoleActive && !e.consoleAnimating {
		return
	}

	// Handle backspace
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if len(e.consoleText) > 0 {
			e.consoleText = e.consoleText[:len(e.consoleText)-1]
		}
		return
	}

	// Handle Enter to execute command
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) {
		if strings.TrimSpace(e.consoleText) != "" {
			// Add to history
			e.consoleHistory = append(e.consoleHistory, e.consoleText)
			if len(e.consoleHistory) > 100 {
				e.consoleHistory = e.consoleHistory[1:] // Keep last 100 commands
			}
			e.consoleHistoryIndex = len(e.consoleHistory)

			// Execute command (must be called while holding the lock)
			// Store command text before unlocking
			cmdText := e.consoleText
			e.consoleText = ""
			// Reset scroll offset when executing a new command (show most recent output)
			e.consoleScrollOffset = 0
			// Execute command - it will use addConsoleOutputUnlocked internally
			e.executeCommandUnlocked(cmdText)
		}
		return
	}

	// Handle history navigation
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		if e.consoleHistoryIndex > 0 {
			e.consoleHistoryIndex--
			e.consoleText = e.consoleHistory[e.consoleHistoryIndex]
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		if e.consoleHistoryIndex < len(e.consoleHistory)-1 {
			e.consoleHistoryIndex++
			e.consoleText = e.consoleHistory[e.consoleHistoryIndex]
		} else {
			e.consoleHistoryIndex = len(e.consoleHistory)
			e.consoleText = ""
		}
		return
	}

	// Handle page up/down for scrollback
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		// Scroll up (show older messages)
		e.consoleScrollOffset += 10 // Scroll by 10 lines
		if e.consoleScrollOffset > len(e.consoleOutput) {
			e.consoleScrollOffset = len(e.consoleOutput)
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		// Scroll down (show newer messages)
		e.consoleScrollOffset -= 10 // Scroll by 10 lines
		if e.consoleScrollOffset < 0 {
			e.consoleScrollOffset = 0
		}
		return
	}

	// Handle text input (printable characters)
	for k := ebiten.KeyA; k <= ebiten.KeyZ; k++ {
		if inpututil.IsKeyJustPressed(k) {
			char := string(rune('a' + (k - ebiten.KeyA)))
			if ebiten.IsKeyPressed(ebiten.KeyShift) {
				char = strings.ToUpper(char)
			}
			e.consoleText += char
			return
		}
	}

	// Handle numbers
	for k := ebiten.Key0; k <= ebiten.Key9; k++ {
		if inpututil.IsKeyJustPressed(k) {
			char := string(rune('0' + (k - ebiten.Key0)))
			e.consoleText += char
			return
		}
	}

	// Handle space
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		e.consoleText += " "
		return
	}

	// Handle special characters
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			e.consoleText += "_"
		} else {
			e.consoleText += "-"
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			e.consoleText += "+"
		} else {
			e.consoleText += "="
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			e.consoleText += "{"
		} else {
			e.consoleText += "["
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketRight) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			e.consoleText += "}"
		} else {
			e.consoleText += "]"
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackslash) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			e.consoleText += "|"
		} else {
			e.consoleText += "\\"
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySemicolon) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			e.consoleText += ":"
		} else {
			e.consoleText += ";"
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyApostrophe) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			e.consoleText += "\""
		} else {
			e.consoleText += "'"
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyComma) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			e.consoleText += "<"
		} else {
			e.consoleText += ","
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPeriod) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			e.consoleText += ">"
		} else {
			e.consoleText += "."
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySlash) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			e.consoleText += "?"
		} else {
			e.consoleText += "/"
		}
		return
	}
}

// executeCommand parses and executes a console command
// This function locks the mutex itself - use executeCommandUnlocked if you already hold the lock
func (e *EbitenRenderer) executeCommand(cmd string) {
	e.consoleMutex.Lock()
	defer e.consoleMutex.Unlock()
	e.executeCommandUnlocked(cmd)
}

// executeCommandUnlocked parses and executes a console command without locking
// Caller must hold consoleMutex
func (e *EbitenRenderer) executeCommandUnlocked(cmd string) {
	cmd = strings.TrimSpace(cmd)
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "bind":
		if len(parts) < 3 {
			e.addConsoleOutputUnlocked("Usage: bind <key> <action>")
			return
		}
		key := strings.ToLower(parts[1])
		actionName := strings.Join(parts[2:], " ")
		e.handleBindCommandUnlocked(key, actionName)

	case "get":
		if len(parts) < 2 {
			e.addConsoleOutputUnlocked("Usage: get <cvar>")
			return
		}
		cvarName := strings.ToLower(parts[1])
		if value, exists := getCvar(cvarName); exists {
			e.addConsoleOutputUnlocked(fmt.Sprintf("%s = \"%s\"", cvarName, value))
		} else {
			e.addConsoleOutputUnlocked(fmt.Sprintf("Unknown cvar: %s", cvarName))
		}

	case "set":
		if len(parts) < 3 {
			e.addConsoleOutputUnlocked("Usage: set <cvar> <value>")
			return
		}
		cvarName := strings.ToLower(parts[1])
		value := strings.Join(parts[2:], " ")
		setCvar(cvarName, value)
		e.addConsoleOutputUnlocked(fmt.Sprintf("%s = \"%s\"", cvarName, value))

	case "clear":
		e.consoleOutput = nil
		e.consoleScrollOffset = 0

	case "color_update":
		loadColorsFromCvars()
		e.addConsoleOutputUnlocked("Colors reloaded from cvars")

	case "list":
		// List all cvars in alphabetical order
		cvarMutex.RLock()
		cvarNames := make([]string, 0, len(cvarMap))
		for name := range cvarMap {
			cvarNames = append(cvarNames, name)
		}
		cvarMutex.RUnlock()

		// Sort alphabetically
		sort.Strings(cvarNames)

		if len(cvarNames) == 0 {
			e.addConsoleOutputUnlocked("No cvars defined")
		} else {
			e.addConsoleOutputUnlocked(fmt.Sprintf("Cvars (%d):", len(cvarNames)))
			for _, name := range cvarNames {
				value, _ := getCvar(name)
				e.addConsoleOutputUnlocked(fmt.Sprintf("  %s = \"%s\"", name, value))
			}
		}

	case "help":
		e.addConsoleOutputUnlocked("Commands:")
		e.addConsoleOutputUnlocked("  bind <key> <action>  - Bind a key to an action")
		e.addConsoleOutputUnlocked("  get <cvar>          - Get a configuration variable")
		e.addConsoleOutputUnlocked("  set <cvar> <value>  - Set a configuration variable")
		e.addConsoleOutputUnlocked("  list                - List all cvars")
		e.addConsoleOutputUnlocked("  color_update        - Reload colors from cvars")
		e.addConsoleOutputUnlocked("  clear               - Clear console output")
		e.addConsoleOutputUnlocked("  help                - Show this help")

	default:
		e.addConsoleOutputUnlocked(fmt.Sprintf("Unknown command: %s (type 'help' for commands)", command))
	}
}

// handleBindCommand handles the bind command
// This function locks the mutex itself - use handleBindCommandUnlocked if you already hold the lock
func (e *EbitenRenderer) handleBindCommand(key, actionName string) {
	e.consoleMutex.Lock()
	defer e.consoleMutex.Unlock()
	e.handleBindCommandUnlocked(key, actionName)
}

// handleBindCommandUnlocked handles the bind command without locking
// Caller must hold consoleMutex
func (e *EbitenRenderer) handleBindCommandUnlocked(key, actionName string) {
	// Map action name to Action enum
	var action engineinput.Action
	actionNameLower := strings.ToLower(actionName)

	switch actionNameLower {
	case "movenorth", "moveup", "north", "up", "n":
		action = engineinput.ActionMoveNorth
	case "movesouth", "movedown", "south", "down", "s":
		action = engineinput.ActionMoveSouth
	case "movewest", "moveleft", "west", "left", "w":
		action = engineinput.ActionMoveWest
	case "moveeast", "moveright", "east", "right", "e":
		action = engineinput.ActionMoveEast
	case "hint":
		action = engineinput.ActionHint
	case "quit":
		action = engineinput.ActionQuit
	case "screenshot":
		action = engineinput.ActionScreenshot
	case "openmenu", "menu":
		action = engineinput.ActionOpenMenu
	case "devmap":
		action = engineinput.ActionDevMap
	case "action", "interact":
		action = engineinput.ActionInteract
	case "resetlevel":
		action = engineinput.ActionResetLevel
	case "zoomin":
		action = engineinput.ActionZoomIn
	case "zoomout":
		action = engineinput.ActionZoomOut
	default:
		e.addConsoleOutputUnlocked(fmt.Sprintf("Unknown action: %s", actionName))
		return
	}

	// Set the binding
	engineinput.SetSingleBinding(action, key)
	e.addConsoleOutputUnlocked(fmt.Sprintf("Bound '%s' to %s", key, engineinput.ActionName(action)))
}

// addConsoleOutput adds a line to the console output
// This function locks the mutex itself - use addConsoleOutputUnlocked if you already hold the lock
func (e *EbitenRenderer) addConsoleOutput(line string) {
	e.consoleMutex.Lock()
	defer e.consoleMutex.Unlock()
	e.addConsoleOutputUnlocked(line)
}

// addConsoleOutputUnlocked adds a line to the console output without locking
// Caller must hold consoleMutex
func (e *EbitenRenderer) addConsoleOutputUnlocked(line string) {
	e.consoleOutput = append(e.consoleOutput, line)
	// Keep last 50 lines
	if len(e.consoleOutput) > 50 {
		e.consoleOutput = e.consoleOutput[len(e.consoleOutput)-50:]
	}
}

// drawConsole draws the console overlay with animation
func (e *EbitenRenderer) drawConsole(screen *ebiten.Image) {
	e.consoleMutex.RLock()
	active := e.consoleActive
	consoleText := e.consoleText
	output := make([]string, len(e.consoleOutput))
	copy(output, e.consoleOutput)
	scrollOffset := e.consoleScrollOffset
	animating := e.consoleAnimating
	animStartTime := e.consoleAnimStartTime
	currentProgress := e.consoleAnimProgress
	e.consoleMutex.RUnlock()

	// Update animation
	const animDuration = 200 // milliseconds
	var progress float64

	if animating {
		now := time.Now().UnixMilli()
		elapsed := now - animStartTime

		if elapsed >= animDuration {
			// Animation complete
			if active {
				progress = 1.0
			} else {
				progress = 0.0
			}

			e.consoleMutex.Lock()
			e.consoleAnimating = false
			e.consoleAnimProgress = progress
			e.consoleMutex.Unlock()
		} else {
			// Interpolate
			animProgress := float64(elapsed) / float64(animDuration)
			easedProgress := easeInOut(animProgress)

			if active {
				progress = easedProgress
			} else {
				progress = 1.0 - easedProgress
			}

			e.consoleMutex.Lock()
			e.consoleAnimProgress = progress
			e.consoleMutex.Unlock()
		}
	} else {
		progress = currentProgress
		if !active && progress <= 0 {
			return // Console fully closed
		}
	}

	if progress <= 0 {
		return // Console fully closed
	}

	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Console takes up bottom portion of screen
	consoleHeight := int(float64(screenHeight) * 0.4 * progress) // 40% of screen, animated
	consoleY := screenHeight - consoleHeight

	// Draw console background (semi-transparent dark)
	bgColor := color.RGBA{0, 0, 0, uint8(220 * progress)}
	vector.DrawFilledRect(screen, 0, float32(consoleY), float32(screenWidth), float32(consoleHeight), bgColor, false)

	// Draw border at top
	borderColor := color.RGBA{100, 100, 150, uint8(255 * progress)}
	vector.DrawFilledRect(screen, 0, float32(consoleY), float32(screenWidth), 2, borderColor, false)

	// Draw console content
	if consoleHeight > 20 {
		// Use monospace font with UI size (matches menu font size but monospace for console)
		face := e.getMonoUIFontFace()
		if face == nil {
			return
		}

		fontSize := e.getUIFontSize()
		lineHeight := int(fontSize) + 6 // Match menu system line spacing
		paddingX := 10
		paddingY := 10

		// Measure text height for proper positioning (text.Draw uses baseline positioning)
		//		_, textHeight := text.Measure("Ag", face, 0)

		// Draw output lines (scrollable, show last few lines that fit)
		// Start from top of console area
		// text.Draw uses baseline positioning, so we add fontSize to position the baseline correctly
		outputY := consoleY + paddingY
		linesToShow := (consoleHeight - paddingY*2 - lineHeight*2) / lineHeight // Reserve space for input line
		if linesToShow > 0 && len(output) > 0 {
			// Calculate start index based on scroll offset
			// scrollOffset 0 = show most recent, higher values = show older messages
			startIdx := len(output) - linesToShow - scrollOffset
			if startIdx < 0 {
				startIdx = 0
			}
			if startIdx >= len(output) {
				startIdx = len(output) - 1
			}

			// Draw visible lines
			for i := startIdx; i < len(output) && i < startIdx+linesToShow; i++ {
				if outputY+lineHeight > consoleY+consoleHeight-lineHeight {
					break
				}
				// text.Draw uses baseline positioning, so add fontSize to Y coordinate
				op := &text.DrawOptions{}
				op.GeoM.Translate(float64(paddingX), float64(outputY)+fontSize)
				textColor := color.RGBA{200, 200, 200, uint8(255 * progress)}
				op.ColorScale.ScaleWithColor(textColor)
				text.Draw(screen, output[i], face, op)
				outputY += lineHeight
			}
		}

		// Draw input line at bottom
		// Position input line near bottom of console
		// Similar to output lines, but positioned from bottom
		prompt := "> "
		inputText := prompt + consoleText

		// Measure input text height for proper bottom alignment (per AGENTS.md)
		_, inputTextHeight := text.Measure(inputText+"_", face, 0)
		// Use formula from AGENTS.md for bottom-aligned text:
		// y = screenHeight - margin - (textHeight * 2)
		// In console coordinates: consoleY + consoleHeight - paddingY - (textHeight * 2)
		// This gives us the Y position where we want the text to start (top of text)
		inputY := consoleY + consoleHeight - paddingY - int(inputTextHeight*2)

		// Draw cursor (blinking)
		cursor := "_"
		if int(time.Now().UnixMilli()/500)%2 == 0 {
			cursor = " "
		}

		// text.Draw uses baseline positioning, so add fontSize to Y coordinate
		// (same as drawColoredText does internally - it adds fontSize to position baseline)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(paddingX), float64(inputY)+fontSize)
		textColor := color.RGBA{255, 255, 255, uint8(255 * progress)}
		op.ColorScale.ScaleWithColor(textColor)
		text.Draw(screen, inputText+cursor, face, op)
	}
}
