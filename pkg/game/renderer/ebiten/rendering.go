// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
package ebiten

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/leonelquinteros/gotext"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// Draw renders the game to the screen (Ebiten interface)
func (e *EbitenRenderer) Draw(screen *ebiten.Image) {
	// Fill background first
	screen.Fill(colorBackground)

	// Check if menu overlays are active - these should be drawn even without valid game state
	e.genericMenuMutex.RLock()
	genericMenuActive := e.genericMenuActive
	title := e.genericMenuTitle
	e.genericMenuMutex.RUnlock()

	// Get snapshot for consistent rendering
	e.snapshotMutex.RLock()
	snap := e.snapshot
	e.snapshotMutex.RUnlock()

	e.gameMutex.RLock()
	g := e.game
	e.gameMutex.RUnlock()

	// If menu is active but game state is invalid, draw only the menu and return
	if genericMenuActive && (!snap.valid || g == nil) {
		if e.monoFontSource == nil || e.sansFontSource == nil {
			// Can't draw menu without fonts
			return
		}

		// Draw floating tiles background for main menu (on top of background fill, before menu overlay)
		if genericMenuActive && title == "The Dark Station" {
			e.drawFloatingTilesBackground(screen)
		}

		// Draw menu overlays
		if genericMenuActive {
			e.drawGenericMenuOverlay(screen)
		}

		// Draw console overlay (always on top)
		e.drawConsole(screen)
		return
	}

	if !snap.valid || e.monoFontSource == nil || e.sansFontSource == nil {
		// Can't draw without valid snapshot or fonts
		return
	}

	if g == nil {
		return
	}

	// Get actual screen size
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Calculate font sizes for layout
	uiFontSize := e.getUIFontSize()

	// Calculate layout dimensions with dynamic spacing based on font size
	headerHeight := int(uiFontSize) + 20
	statusBarHeight := int(uiFontSize)*2 + 20

	// Consistent margin around map area (20px on all sides)
	mapMargin := 20

	// Calculate maximum available space for map (after header, with consistent margins)
	// Note: status bar and messages panel are overlays and do not reduce map height
	availableHeight := screenHeight - headerHeight - mapMargin*2
	availableWidth := screenWidth - mapMargin*2

	// Recalculate viewport to maximize based on current available space
	// This ensures the viewport uses the maximum available space
	viewportCols := availableWidth / e.tileSize
	viewportRows := availableHeight / e.tileSize

	// Ensure minimum viewport size
	if viewportCols < 15 {
		viewportCols = 15
	}
	if viewportRows < 11 {
		viewportRows = 11
	}

	// Keep odd numbers for centering
	if viewportCols%2 == 0 {
		viewportCols--
	}
	if viewportRows%2 == 0 {
		viewportRows--
	}

	// Update stored viewport (will be used in next frame's recalculateViewport)
	e.viewportCols = viewportCols
	e.viewportRows = viewportRows

	// Calculate map dimensions to fill available space
	mapAreaWidth := viewportCols * e.tileSize
	mapAreaHeight := viewportRows * e.tileSize

	// Center the map horizontally and vertically with consistent 20px margins
	mapX := (screenWidth - mapAreaWidth) / 2
	mapY := headerHeight + mapMargin

	// Draw header (empty now - deck number moved to objectives panel)
	e.drawHeaderFromSnapshot(screen, &snap, screenWidth, headerHeight)

	// Draw map background with consistent 20px margins
	vector.DrawFilledRect(screen, float32(mapX-mapMargin), float32(mapY-mapMargin),
		float32(mapAreaWidth+mapMargin*2), float32(mapAreaHeight+mapMargin*2),
		colorMapBackground, false)

	// Draw the map using snapshot for player position
	e.drawMap(screen, g, mapX, mapY, &snap)

	// Draw status bar (overlay on top left of map) - use snapshot data
	statusY := mapY + mapMargin // Consistent margin from top of map
	e.drawStatusBarFromSnapshot(screen, &snap, mapX+mapMargin, statusY, mapAreaWidth, statusBarHeight)

	// Draw messages panel as a bottom‑aligned overlay, limited to a few lines
	e.drawMessagesFromSnapshot(screen, &snap, screenWidth, screenHeight)

	// Draw menu overlays on top of everything (covers most of the screen)
	if genericMenuActive {
		e.drawGenericMenuOverlay(screen)
	}

	// Draw console overlay (always on top)
	e.drawConsole(screen)
}

// drawHeaderFromSnapshot draws the header (currently empty - deck number moved to objectives panel)
func (e *EbitenRenderer) drawHeaderFromSnapshot(screen *ebiten.Image, snap *renderSnapshot, screenWidth int, headerHeight int) {
	// Header is now empty - deck number has been moved to the objectives panel
}

// drawMap renders the game map
func (e *EbitenRenderer) drawMap(screen *ebiten.Image, g *state.Game, mapX, mapY int, snap *renderSnapshot) {
	if g.CurrentCell == nil || g.Grid == nil {
		return
	}

	// Use snapshot for player position to prevent jitter
	playerRow := snap.playerRow
	playerCol := snap.playerCol

	// Calculate viewport bounds centered on player
	startRow := playerRow - e.viewportRows/2
	startCol := playerCol - e.viewportCols/2

	// Render each tile in the viewport
	for vRow := 0; vRow < e.viewportRows; vRow++ {
		for vCol := 0; vCol < e.viewportCols; vCol++ {
			e.predrawTile(startRow, startCol, mapX, mapY, vRow, vCol, g, snap, screen)
		}
	}

	e.drawRoomLabels(screen, snap, mapX, mapY, startRow, startCol)
	e.drawCallouts(screen, snap, mapX, mapY, startRow, startCol)
	e.drawPlayerWithDebounce(screen, g, snap, mapX, mapY, startRow, startCol)
	e.drawExitAnimation(screen, snap, mapX, mapY, startRow, startCol)
}

func (e *EbitenRenderer) predrawTile(startRow int, startCol int, mapX int, mapY int, vRow int, vCol int, g *state.Game, snap *renderSnapshot, screen *ebiten.Image) {
	mapRow := startRow + vRow
	mapCol := startCol + vCol

	x := mapX + vCol*e.tileSize
	y := mapY + vRow*e.tileSize

	cell := g.Grid.GetCell(mapRow, mapCol)

	cellRenderOptions := e.getCellRenderOptions(g, cell, snap)

	if cell != nil && cell.Row == snap.playerRow && cell.Col == snap.playerCol {
		// Skip drawing player here - it will be drawn separately with debounce animation
		return
	}

	// Check if this is the focused cell (has active callout) or an interactable cell - use focus background
	var customBg color.Color
	isFocused := cell != nil && cell.Row == snap.focusedCellRow && cell.Col == snap.focusedCellCol
	isInteractable := false
	if cell != nil {
		for _, ic := range snap.interactableCells {
			if cell.Row == ic.row && cell.Col == ic.col {
				isInteractable = true
				break
			}
		}
	}

	// Check for blocking hazards or locked doors that need to be cleared - use brighter background
	needsClearing := false
	if cell != nil && (g.HasMap || cell.Discovered) {
		if gameworld.HasBlockingHazard(cell) || gameworld.HasLockedDoor(cell) {
			needsClearing = true
		}
	}

	// Set background color with priority order
	if cell != nil {
		if (g.HasMap || cell.Discovered) && gameworld.HasBlockingHazard(cell) {
			// Dark red background for impassable hazards (e.g. sparks)
			customBg = colorHazardBackground
		} else if (g.HasMap || cell.Discovered) && gameworld.HasDoor(cell) {
			roomName := gameworld.GetGameData(cell).Door.RoomName
			if !g.RoomDoorsPowered[roomName] {
				// Unpowered doors use same red background as unsolved hazards
				customBg = colorHazardBackground
			}
		} else if (g.HasMap || cell.Discovered) && gameworld.HasMaintenanceTerminal(cell) {
			data := gameworld.GetGameData(cell)
			if data.MaintenanceTerm != nil && !data.MaintenanceTerm.Powered {
				customBg = colorHazardBackground
			}
		} else if (g.HasMap || cell.Discovered) && gameworld.HasTerminal(cell) {
			if cell.Room && !g.RoomCCTVPowered[cell.Name] {
				customBg = colorHazardBackground
			}
		} else if (g.HasMap || cell.Discovered) && gameworld.HasHazardControl(cell) {
			if cell.Room && !g.RoomCCTVPowered[cell.Name] {
				customBg = colorHazardBackground
			}
		}
		if customBg == nil && needsClearing {
			// Brighter background for locked doors that need to be cleared (powered rooms)
			customBg = colorBlockedBackground
		} else if gameworld.HasPoweredGenerator(cell) {
			// Powered generators show green background
			customBg = colorWallBgPowered
		} else if isFocused || isInteractable {
			// Focus/interactable color for non-powered walls
			customBg = colorFocusBackground
		} else if cell != nil && cell.ExitCell && (g.HasMap || cell.Discovered) && !cell.Locked && g.AllGeneratorsPowered() && g.AllHazardsCleared() {
			// Unlocked exit cell - use pulsing background (requires generators powered and hazards cleared)
			customBg = e.getPulsingExitBackgroundColor()
		}
	}

	// Draw the tile character with optional background
	e.drawTileWithBg(screen, cellRenderOptions.Icon, x, y, cellRenderOptions.Color, cellRenderOptions.HasBackground, customBg)
}

// drawTileWithBg draws a single tile with optional custom background color
func (e *EbitenRenderer) drawTileWithBg(screen *ebiten.Image, icon string, x, y int, col color.Color, hasBackground bool, bgColor color.Color) {
	// Skip completely empty/void tiles
	if icon == " " || icon == "" {
		return
	}

	// Convert color to RGBA
	r, g, b, a := col.RGBA()
	tileColor := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}

	// Skip if color is too dark (close to background)
	if tileColor.R < 30 && tileColor.G < 30 && tileColor.B < 30 {
		return
	}

	// Draw block background if requested (for walls, doors, etc.)
	if hasBackground {
		// Draw background with small margin inside the tile
		margin := float32(2)
		bgCol := colorWallBg
		if bgColor != nil {
			// Convert color.Color to color.RGBA
			r, g, b, a := bgColor.RGBA()
			bgCol = color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
		}
		vector.DrawFilledRect(screen, float32(x)+margin, float32(y)+margin,
			float32(e.tileSize)-margin*2, float32(e.tileSize)-margin*2,
			bgCol, false)
	}

	// Draw the colored character
	e.drawColoredChar(screen, icon, x, y, tileColor)
}

// drawPlayerWithDebounce draws the player icon with debounce animation if active
func (e *EbitenRenderer) drawPlayerWithDebounce(screen *ebiten.Image, g *state.Game, snap *renderSnapshot, mapX, mapY, startRow, startCol int) {
	e.debounceMutex.RLock()
	direction := e.debounceDirection
	startTime := e.debounceStartTime
	e.debounceMutex.RUnlock()

	// Calculate player position in viewport
	playerVRow := snap.playerRow - startRow
	playerVCol := snap.playerCol - startCol

	// Skip if player not in viewport
	if playerVRow < 0 || playerVRow >= e.viewportRows || playerVCol < 0 || playerVCol >= e.viewportCols {
		return
	}

	// Calculate base position
	baseX := mapX + playerVCol*e.tileSize
	baseY := mapY + playerVRow*e.tileSize

	// Calculate debounce offset
	offsetX := 0
	offsetY := 0
	if direction != "" {
		now := time.Now().UnixMilli()
		elapsed := now - startTime
		const debounceDuration = 150 // milliseconds

		if elapsed < debounceDuration {
			// Calculate bounce offset using a sine wave for smooth animation
			progress := float64(elapsed) / debounceDuration
			bounceAmount := math.Sin(progress*math.Pi) * 8.0 // Max 8 pixels offset

			switch direction {
			case "north":
				offsetY = int(-bounceAmount)
			case "south":
				offsetY = int(bounceAmount)
			case "east":
				offsetX = int(bounceAmount)
			case "west":
				offsetX = int(-bounceAmount)
			}
		} else {
			// Animation complete, clear it
			e.debounceMutex.Lock()
			e.debounceDirection = ""
			e.debounceMutex.Unlock()
		}
	}

	// Draw player icon at offset position
	playerX := baseX + offsetX
	playerY := baseY + offsetY
	e.drawTileWithBg(screen, PlayerIcon, playerX, playerY, colorPlayer, false, nil)
}

// drawExitAnimation draws the exit transition animation with a meaningful message
func (e *EbitenRenderer) drawExitAnimation(screen *ebiten.Image, snap *renderSnapshot, mapX, mapY, startRow, startCol int) {
	if !snap.exitAnimating {
		return
	}

	now := time.Now().UnixMilli()
	elapsed := now - snap.exitAnimStartTime
	const exitAnimDuration = 2000 // 2 seconds for transition

	if elapsed >= exitAnimDuration {
		return // Animation complete
	}

	// Calculate fade progress (0.0 to 1.0)
	progress := float64(elapsed) / exitAnimDuration

	// Get screen dimensions - use actual screen bounds to ensure full coverage
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	if w == 0 || h == 0 {
		// Fallback to window size if screen bounds are invalid
		w, h = ebiten.WindowSize()
		if w == 0 || h == 0 {
			w, h = e.windowWidth, e.windowHeight
		}
	}

	// Phase 1: Fade to dark background (first 40% of animation)
	// Phase 2: Show message on dark background (middle 40%)
	// Phase 3: Fade message out (last 20%)
	var overlayAlpha float64
	var textAlpha float64
	var showText bool

	if progress < 0.4 {
		// Phase 1: Fade to dark background
		overlayAlpha = progress / 0.4
		textAlpha = 0
		showText = false
	} else if progress < 0.8 {
		// Phase 2: Show message on dark background
		overlayAlpha = 1.0
		textProgress := (progress - 0.4) / 0.4
		textAlpha = textProgress
		if textAlpha > 1.0 {
			textAlpha = 1.0
		}
		showText = true
	} else {
		// Phase 3: Fade message out, keep dark background
		overlayAlpha = 1.0
		fadeProgress := (progress - 0.8) / 0.2
		textAlpha = 1.0 - fadeProgress
		if textAlpha < 0 {
			textAlpha = 0
		}
		showText = textAlpha > 0
	}

	// Draw dark overlay matching the game's aesthetic
	overlayColor := color.RGBA{15, 15, 26, uint8(255 * overlayAlpha)} // Same as colorMapBackground
	vector.DrawFilledRect(screen, 0, 0, float32(w), float32(h), overlayColor, false)

	// Draw transition message
	if showText && textAlpha > 0 {
		// Format string with translation - translate the format string first
		messageFormat := gotext.Get("DECK_CLEARED")
		message := fmt.Sprintf(messageFormat, snap.level)
		subMessage := gotext.Get("PROCEEDING_TO_NEXT_LEVEL")

		// Get font size for UI (use larger size for transition screen)
		fontSize := e.getUIFontSize() * 1.5
		face := e.getSansFontFace()

		// Calculate text position (centered)
		messageWidth, _ := text.Measure(message, face, 0)
		subMessageWidth, _ := text.Measure(subMessage, face, 0)

		centerX := float64(w) / 2
		centerY := float64(h) / 2

		messageX := centerX - float64(messageWidth)/2
		messageY := centerY - fontSize/2

		subMessageX := centerX - float64(subMessageWidth)/2
		subMessageY := centerY + fontSize + 10

		// Draw main message with fade (using action color for emphasis)
		mainTextColor := color.RGBA{220, 170, 255, uint8(255 * textAlpha)} // colorAction
		op := &text.DrawOptions{}
		op.GeoM.Translate(messageX, messageY+fontSize)
		op.ColorScale.ScaleWithColor(mainTextColor)
		text.Draw(screen, message, face, op)

		// Draw sub message with fade (using text color)
		subTextColor := color.RGBA{240, 240, 255, uint8(255 * textAlpha)} // colorText
		op2 := &text.DrawOptions{}
		op2.GeoM.Translate(subMessageX, subMessageY+fontSize)
		op2.ColorScale.ScaleWithColor(subTextColor)
		text.Draw(screen, subMessage, face, op2)
	}
}

// drawRoomLabels renders persistent room name labels at the leftmost point of each room
func (e *EbitenRenderer) drawRoomLabels(screen *ebiten.Image, snap *renderSnapshot, mapX, mapY, startRow, startCol int) {
	if len(snap.roomLabels) == 0 {
		return
	}

	fontSize := e.getUIFontSize()

	for _, rl := range snap.roomLabels {
		// Check if the label position is visible in the viewport
		labelCol := rl.StartCol
		viewportStartCol := startCol
		viewportEndCol := startCol + e.viewportCols - 1

		// Skip if label is outside viewport
		if labelCol < viewportStartCol || labelCol > viewportEndCol {
			continue
		}

		// Convert to viewport coordinates
		vCol := labelCol - startCol
		vRow := (rl.Row - startRow) - 1

		// Skip if not in vertical range
		if vRow < 0 || vRow >= e.viewportRows {
			continue
		}

		// Compute pixel position (left edge of the cell where label should be)
		cellX := mapX + vCol*e.tileSize
		cellY := mapY + vRow*e.tileSize

		// Measure text
		textWidth := e.getTextWidth(rl.RoomName)

		// Draw background box for readability
		paddingX := 6
		paddingY := 4
		boxW := int(textWidth) + paddingX*2
		boxH := int(fontSize) + paddingY*2

		// Position box starting at the leftmost point of the room cell
		// Raise it by half its height so it sits just above the wall
		boxX := cellX + 2 // Small offset from left edge of cell
		boxY := cellY - boxH - 4 - boxH/2

		// Higher contrast colors for room labels
		bgColor := color.RGBA{15, 20, 40, 235}
		borderColor := color.RGBA{140, 140, 200, 255}

		// Border
		vector.DrawFilledRect(
			screen,
			float32(boxX-1), float32(boxY-1),
			float32(boxW+2), float32(boxH+2),
			borderColor,
			false,
		)

		// Background
		vector.DrawFilledRect(
			screen,
			float32(boxX), float32(boxY),
			float32(boxW), float32(boxH),
			bgColor,
			false,
		)

		// Position text: drawColoredText uses baseline positioning (adds fontSize to y)
		// Similar to callouts: subtract fontSize so baseline ends up inside the box
		textX := boxX + paddingX
		textY := boxY + paddingY - int(fontSize)

		// Draw bold-ish text by rendering twice with a slight offset
		labelColor := color.RGBA{230, 230, 255, 255} // High-contrast light color
		e.drawColoredText(screen, rl.RoomName, textX, textY, labelColor)
		e.drawColoredText(screen, rl.RoomName, textX+1, textY, labelColor)
	}
}

// drawDirectionLabels draws N/S/E/W labels around the map
func (e *EbitenRenderer) drawDirectionLabels(screen *ebiten.Image, g *state.Game, mapX, mapY, mapWidth, mapHeight int) {
	if g.CurrentCell == nil {
		return
	}

	fontSize := e.getUIFontSize()

	// North label (above map, centered)
	northText := e.getDirectionText(g, g.CurrentCell.North, "NORTH")
	northWidth := e.getTextWidth(northText)
	northX := mapX + mapWidth/2 - int(northWidth)/2
	e.drawColoredText(screen, northText, northX, mapY-int(fontSize)-15, colorText)

	// South label (below map, centered)
	southText := e.getDirectionText(g, g.CurrentCell.South, "SOUTH")
	southWidth := e.getTextWidth(southText)
	southX := mapX + mapWidth/2 - int(southWidth)/2
	e.drawColoredText(screen, southText, southX, mapY+mapHeight+10, colorText)

	// West label (left of map, vertically centered)
	westText := e.getDirectionText(g, g.CurrentCell.West, "WEST")
	westWidth := e.getTextWidth(westText)
	e.drawColoredText(screen, westText, mapX-int(westWidth)-20, mapY+mapHeight/2-int(fontSize)/2, colorText)

	// East label (right of map, vertically centered)
	eastText := e.getDirectionText(g, g.CurrentCell.East, "EAST")
	e.drawColoredText(screen, eastText, mapX+mapWidth+20, mapY+mapHeight/2-int(fontSize)/2, colorText)
}

// getDirectionText returns the text for a direction label
// direction should be a translation key (e.g., "NORTH") which will be translated in drawColoredText
func (e *EbitenRenderer) getDirectionText(g *state.Game, cell *world.Cell, direction string) string {
	if cell == nil || !cell.Room {
		return "WALL"
	}

	// Check if blocked
	if gameworld.HasLockedDoor(cell) {
		data := gameworld.GetGameData(cell)
		// Translate direction first, then format
		translatedDir := gotext.Get(direction)
		return fmt.Sprintf(gotext.Get("DIRECTION_NEED_KEYCARD"), translatedDir, data.Door.KeycardName())
	}

	if gameworld.HasBlockingHazard(cell) {
		// Translate direction first, then format
		translatedDir := gotext.Get(direction)
		return fmt.Sprintf(gotext.Get("DIRECTION_BLOCKED"), translatedDir)
	}

	return direction
}

// drawStatusBarFromSnapshot draws the inventory and generator status using snapshot data
func (e *EbitenRenderer) drawStatusBarFromSnapshot(screen *ebiten.Image, snap *renderSnapshot, x, y, width, height int) {
	// Check if there's anything to show
	hasObjectives := len(snap.objectives) > 0
	hasInventory := len(snap.ownedItems) > 0 || snap.batteries > 0
	hasGenerators := len(snap.generators) > 0

	// Always show at least the deck number
	hasDeckNumber := true

	// Don't draw anything if everything is empty (but we always have deck number)
	if !hasDeckNumber && !hasObjectives && !hasInventory && !hasGenerators {
		return
	}

	fontSize := e.getUIFontSize()
	lineHeight := int(fontSize) + 4

	// Calculate how many lines we need (always include deck number)
	linesNeeded := 0
	if hasDeckNumber {
		linesNeeded++ // Deck number is always first
	}
	if hasObjectives {
		linesNeeded += len(snap.objectives)
	}
	if hasInventory {
		linesNeeded++
	}
	if hasGenerators {
		linesNeeded++
	}

	// Calculate the maximum width needed for all text lines
	maxTextWidth := 0.0
	// Deck number text - translate format string first, then format
	deckTextFormat := gotext.Get("DECK_NUMBER")
	deckText := fmt.Sprintf(deckTextFormat, snap.level)
	deckWidth := e.getTextWidth(deckText)
	if deckWidth > maxTextWidth {
		maxTextWidth = deckWidth
	}
	if hasObjectives {
		for _, objective := range snap.objectives {
			// Translate objective if it's a translation key for width calculation
			translatedObjective := gotext.Get(objective)
			// Parse markup to get actual text width (not markup)
			segments := e.parseMarkup(translatedObjective)
			textWidth := 0.0
			for _, seg := range segments {
				textWidth += e.getTextWidth(seg.text)
			}
			if textWidth > maxTextWidth {
				maxTextWidth = textWidth
			}
		}
	}
	if hasInventory {
		// Build inventory text with markup for width calculation (same format as rendering)
		invLabel := gotext.Get("INVENTORY")
		invParts := []string{invLabel}
		for i, itemName := range snap.ownedItems {
			if i > 0 {
				invParts = append(invParts, ",")
			}
			invParts = append(invParts, fmt.Sprintf("ITEM{%s}", itemName))
		}
		if snap.batteries > 0 {
			if len(snap.ownedItems) > 0 {
				invParts = append(invParts, ",")
			}
			invParts = append(invParts, fmt.Sprintf("ACTION{Batteries x%d}", snap.batteries))
		}
		invText := strings.Join(invParts, " ")
		// Calculate width using parsed segments (actual text width, not markup)
		segments := e.parseMarkup(invText)
		textWidth := 0.0
		for _, seg := range segments {
			textWidth += e.getTextWidth(seg.text)
		}
		if textWidth > maxTextWidth {
			maxTextWidth = textWidth
		}
	}
	if hasGenerators {
		genLabel := gotext.Get("GENERATORS")
		genText := genLabel + " "
		genParts := []string{}
		for i, gen := range snap.generators {
			if gen.powered {
				genParts = append(genParts, fmt.Sprintf("#%d POWERED{ONLINE}", i+1))
			} else {
				genParts = append(genParts, fmt.Sprintf("#%d %d/%d", i+1, gen.batteriesInserted, gen.batteriesRequired))
			}
		}
		genText += strings.Join(genParts, ", ")
		w := e.getTextWidth(genText)
		if w > maxTextWidth {
			maxTextWidth = w
		}
	}

	// Adjust panel height based on actual content
	panelHeight := lineHeight*linesNeeded + 10
	if panelHeight < int(fontSize)+10 {
		panelHeight = int(fontSize) + 10
	}

	// Calculate panel width based on widest text, with padding
	panelWidth := int(maxTextWidth) + 20 // 10px padding on each side
	if panelWidth < 100 {
		panelWidth = 100 // Minimum width
	}

	// Draw panel background with border (more opaque for overlay on map)
	bgX := float32(x - 10)
	bgY := float32(y - 5)
	bgW := float32(panelWidth)
	bgH := float32(panelHeight)
	borderColor := color.RGBA{80, 80, 100, 255}
	// More opaque background for overlay readability
	overlayBackground := color.RGBA{20, 20, 35, 250} // More opaque than colorPanelBackground

	// Border
	vector.DrawFilledRect(screen, bgX-1, bgY-1, bgW+2, bgH+2, borderColor, false)
	// Background
	vector.DrawFilledRect(screen, bgX, bgY, bgW, bgH, overlayBackground, false)

	// Calculate vertical center for first line
	// Since drawColoredText adds fontSize for baseline, we need to adjust
	firstLineY := y + (panelHeight / 2) - (lineHeight * linesNeeded / 2) - int(fontSize)

	currentY := firstLineY

	// Deck number (always first line)
	if hasDeckNumber {
		deckTextFormat := gotext.Get("DECK_NUMBER")
		deckText := fmt.Sprintf(deckTextFormat, snap.level)
		e.drawColoredText(screen, deckText, x, currentY, colorAction)
		currentY += lineHeight
		// Add a small gap between deck number and objectives
		if hasObjectives {
			currentY += 2
		}
	}

	// Objectives (displayed after deck number)
	if hasObjectives {
		for _, objective := range snap.objectives {
			// Translate objective if it's a translation key, then parse markup
			translatedObjective := gotext.Get(objective)
			// Parse markup to properly color ACTION{} segments
			segments := e.parseMarkup(translatedObjective)
			e.drawColoredTextSegments(screen, segments, x, currentY)
			currentY += lineHeight
		}
		// Add a small gap between objectives and inventory
		if hasInventory || hasGenerators {
			currentY += 2
		}
	}

	// Inventory line (only if not empty)
	if hasInventory {
		// Build inventory text with item colors using markup, commas in default color
		invLabel := gotext.Get("INVENTORY")
		invParts := []string{invLabel}
		for i, itemName := range snap.ownedItems {
			if i > 0 {
				invParts = append(invParts, ",") // Comma in default text color
			}
			invParts = append(invParts, fmt.Sprintf("ITEM{%s}", itemName))
		}
		if snap.batteries > 0 {
			if len(snap.ownedItems) > 0 {
				invParts = append(invParts, ",") // Comma in default text color
			}
			invParts = append(invParts, fmt.Sprintf("ACTION{Batteries x%d}", snap.batteries))
		}
		invText := strings.Join(invParts, " ")

		// Parse markup to apply item colors (commas will be in default color)
		segments := e.parseMarkup(invText)
		e.drawColoredTextSegments(screen, segments, x, currentY)
		currentY += lineHeight
	}

	// Generator status (if applicable)
	if hasGenerators {
		genLabel := gotext.Get("GENERATORS")
		genText := genLabel + ": "
		genParts := []string{}
		for i, gen := range snap.generators {
			if gen.powered {
				genParts = append(genParts, fmt.Sprintf("#%d POWERED{ONLINE}", i+1))
			} else {
				genParts = append(genParts, fmt.Sprintf("#%d %d/%d", i+1, gen.batteriesInserted, gen.batteriesRequired))
			}
		}
		genText += strings.Join(genParts, ", ")
		e.drawColoredTextSegments(screen, e.parseMarkup(genText), x, currentY)
	}
}

// drawMessagesFromSnapshot draws the messages panel as a bottom‑aligned overlay using snapshot data.
// The background panel is only drawn when there are visible (non‑expired) messages and shows at most 4 lines.
func (e *EbitenRenderer) drawMessagesFromSnapshot(screen *ebiten.Image, snap *renderSnapshot, screenWidth, screenHeight int) {
	const maxVisibleLines = 4

	fontSize := e.getUIFontSize()
	lineHeight := int(fontSize) + 4 // Font size plus padding for proper line spacing

	if len(snap.messages) == 0 {
		// No messages to show, so don't draw any panel background
		return
	}

	now := time.Now().UnixMilli()
	const messageLifetime = 10000 // 10 seconds in milliseconds

	// Collect visible messages (messages are already sorted chronologically in snapshot)
	type visibleMessage struct {
		segments []textSegment
	}
	visible := make([]visibleMessage, 0, maxVisibleLines)

	// Iterate through messages in chronological order (oldest first)
	// Take the last maxVisibleLines messages (most recent)
	startIdx := len(snap.messages) - maxVisibleLines
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(snap.messages) && len(visible) < maxVisibleLines; i++ {
		msgEntry := snap.messages[i]
		age := now - msgEntry.Timestamp
		if age >= messageLifetime {
			continue // Skip fully faded/expired messages (shouldn't happen, but double-check)
		}

		// Calculate alpha: 1.0 at start, 0.0 at messageLifetime
		// Fade starts at 7 seconds (70% of lifetime), fully transparent at 10 seconds
		fadeStart := int64(messageLifetime * 7 / 10) // Start fading at 7 seconds
		alpha := 1.0
		if age > fadeStart {
			// Fade from 1.0 to 0.0 over the last 3 seconds
			fadeProgress := float64(age-fadeStart) / float64(messageLifetime-fadeStart)
			alpha = 1.0 - fadeProgress
			if alpha < 0 {
				alpha = 0
			}
		}

		// Parse markup and apply alpha to segment colors
		segments := e.parseMarkup(msgEntry.Text)
		fadedSegments := make([]textSegment, len(segments))
		for j, seg := range segments {
			fadedSegments[j] = textSegment{
				text:  seg.text,
				color: e.applyAlpha(seg.color, alpha),
			}
		}

		visible = append(visible, visibleMessage{segments: fadedSegments})
	}

	// If no messages are actually visible after fading, don't draw anything
	if len(visible) == 0 {
		return
	}

	// Calculate the maximum width needed for all messages
	maxTextWidth := 0.0
	// Include header width
	headerText := "─── Messages ───"
	headerWidth := e.getTextWidth(headerText)
	if headerWidth > maxTextWidth {
		maxTextWidth = headerWidth
	}
	// Calculate width for each visible message (sum of all segments)
	for _, vm := range visible {
		msgWidth := 0.0
		for _, seg := range vm.segments {
			msgWidth += e.getTextWidth(seg.text)
		}
		if msgWidth > maxTextWidth {
			maxTextWidth = msgWidth
		}
	}

	// Calculate dynamic panel height based on number of visible messages
	headerHeight := int(fontSize) + 8
	bodyHeight := len(visible) * lineHeight
	panelHeight := headerHeight + bodyHeight + 10 // Extra padding

	// Calculate panel width based on widest text, with padding
	panelWidth := int(maxTextWidth) + 20 // 10px padding on each side
	if panelWidth < 100 {
		panelWidth = 100 // Minimum width
	}
	// Don't exceed screen width
	if panelWidth > screenWidth-40 {
		panelWidth = screenWidth - 40
	}

	// Position panel aligned to the bottom of the window, centered horizontally
	marginBottom := 20
	bgX := float32((screenWidth - panelWidth) / 2)
	bgY := float32(screenHeight - marginBottom - panelHeight)
	if bgY < 0 {
		bgY = 0
	}
	bgW := float32(panelWidth)
	bgH := float32(panelHeight)

	borderColor := color.RGBA{80, 80, 100, 255}

	// Border
	vector.DrawFilledRect(screen, bgX-1, bgY-1, bgW+2, bgH+2, borderColor, false)
	// Background
	vector.DrawFilledRect(screen, bgX, bgY, bgW, bgH, colorPanelBackground, false)

	// Header - position at top with proper padding (centered in panel)
	x := int(bgX) + 10
	headerY := int(bgY) + 8 - int(fontSize) // Small padding from top, account for baseline
	e.drawColoredText(screen, headerText, x, headerY, colorSubtle)

	// Messages - start below header with proper spacing
	messageStartY := int(bgY) + headerHeight + 4
	for i, vm := range visible {
		msgY := messageStartY + i*lineHeight - int(fontSize)
		e.drawColoredTextSegments(screen, vm.segments, x, msgY)
	}
}
