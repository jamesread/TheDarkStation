// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
package ebiten

import (
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	gamemenu "darkstation/pkg/game/menu"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
)

// appendRoundedRect adds a rounded rectangle to the path. (x, y) is top-left; w, h are size; r is corner radius.
// Uses clockwise arcs so the path winds correctly for fill.
func appendRoundedRect(p *vector.Path, x, y, w, h, r float32) {
	appendRoundedRectDir(p, x, y, w, h, r, vector.Clockwise)
}

// appendRoundedRectDir adds a rounded rectangle with the given winding direction.
// CounterClockwise creates a hole when combined with an outer clockwise rect.
func appendRoundedRectDir(p *vector.Path, x, y, w, h, r float32, dir vector.Direction) {
	if r <= 0 {
		p.MoveTo(x, y)
		p.LineTo(x, y+h)
		p.LineTo(x+w, y+h)
		p.LineTo(x+w, y)
		p.Close()
		return
	}
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	halfPi := float32(math.Pi / 2)
	pi := float32(math.Pi)
	p.MoveTo(x+r, y)
	p.LineTo(x+w-r, y)
	p.Arc(x+w-r, y+r, r, 3*halfPi, 0, dir)
	p.LineTo(x+w, y+h-r)
	p.Arc(x+w-r, y+h-r, r, 0, halfPi, dir)
	p.LineTo(x+r, y+h)
	p.Arc(x+r, y+h-r, r, halfPi, pi, dir)
	p.LineTo(x, y+r)
	p.Arc(x+r, y+r, r, pi, 3*halfPi, dir)
	p.Close()
}

// drawRoundedRectWithShadow draws a rounded rectangle with drop shadow, fill, and border.
// Used by menus and tooltips. alpha scales shadow opacity (1.0 = full, used for callout fade).
// Shadow color is derived from borderColor (darkened to ~15% brightness).
func drawRoundedRectWithShadow(screen *ebiten.Image, x, y, w, h, cornerRadius, borderWidth float32, bgColor, borderColor color.Color, alpha float32) {
	const shadowSpread = 8
	// Derive shadow from border color (darkened)
	bor, bog, bob, _ := borderColor.RGBA()
	shadowR := uint8((bor >> 8) * 15 / 255)
	shadowG := uint8((bog >> 8) * 15 / 255)
	shadowB := uint8((bob >> 8) * 15 / 255)
	if shadowR < 8 {
		shadowR = 8
	}
	if shadowG < 8 {
		shadowG = 8
	}
	if shadowB < 8 {
		shadowB = 8
	}

	var path vector.Path
	for i := shadowSpread; i >= 1; i-- {
		ringAlpha := uint8(12 + i*8)
		if ringAlpha > 55 {
			ringAlpha = 55
		}
		ringAlpha = uint8(float32(ringAlpha) * alpha)
		path.Reset()
		appendRoundedRect(&path,
			x-float32(i), y-float32(i),
			w+float32(i*2), h+float32(i*2),
			cornerRadius+float32(i))
		appendRoundedRectDir(&path,
			x-float32(i-1), y-float32(i-1),
			w+float32((i-1)*2), h+float32((i-1)*2),
			cornerRadius+float32(i-1), vector.CounterClockwise)
		drawOpts := &vector.DrawPathOptions{AntiAlias: true}
		drawOpts.ColorScale.ScaleWithColor(color.RGBA{shadowR, shadowG, shadowB, ringAlpha})
		vector.FillPath(screen, &path, nil, drawOpts)
	}

	path.Reset()
	appendRoundedRect(&path, x, y, w, h, cornerRadius)
	drawOpts := &vector.DrawPathOptions{AntiAlias: true}
	drawOpts.ColorScale.ScaleWithColor(bgColor)
	vector.FillPath(screen, &path, nil, drawOpts)

	path.Reset()
	appendRoundedRect(&path, x, y, w, h, cornerRadius)
	strokeOpts := &vector.StrokeOptions{Width: borderWidth, MiterLimit: 10}
	drawOpts = &vector.DrawPathOptions{AntiAlias: true}
	drawOpts.ColorScale.ScaleWithColor(borderColor)
	vector.StrokePath(screen, &path, strokeOpts, drawOpts)
}

// RenderMenu implements gamemenu.MenuRenderer for Ebiten.
// It captures the current frame and marks the menu overlay as active.
func (e *EbitenRenderer) RenderMenu(g *state.Game, items []gamemenu.MenuItem, selected int, helpText string, title string) {
	// Keep the underlying game/map snapshot up to date
	e.RenderFrame(g)

	e.genericMenuMutex.Lock()
	defer e.genericMenuMutex.Unlock()

	// Detect menu title change (menu switch) and start height animation
	var titleChanged bool
	var oldHeight float64

	if e.genericMenuActive && e.genericMenuTitle != title {
		// Menu is active and title changed - use current menu state
		titleChanged = true
		oldHeight = e.calculateMenuHeight(e.genericMenuLabels, e.genericMenuTitle, e.genericMenuHelpText)
	} else if !e.genericMenuActive && len(e.prevMenuItems) > 0 && e.prevMenuTitle != title {
		// Menu was cleared but we have previous state - use it for transition
		titleChanged = true
		oldHeight = e.calculateMenuHeight(e.prevMenuLabels, e.prevMenuTitle, e.prevMenuHelpText)
	}

	if titleChanged {
		// Calculate required heights for both menus
		newHeight := e.calculateMenuHeight(e.snapshotMenuLabels(items), title, helpText)

		e.menuHeightAnimStartHeight = oldHeight
		e.menuHeightAnimTargetHeight = newHeight
		e.menuHeightAnimStartTime = time.Now().UnixMilli()
		e.menuHeightAnimating = true
	} else if !e.genericMenuActive && len(e.prevMenuItems) == 0 {
		// Menu just opened (no previous state) - no height animation, set target height immediately
		e.menuHeightAnimating = false
		e.menuHeightAnimTargetHeight = e.calculateMenuHeight(e.snapshotMenuLabels(items), title, helpText)
	}

	// Check if animations have completed and clean them up
	now := time.Now().UnixMilli()
	const heightAnimDuration = 200
	const highlightAnimDuration = 150
	if e.menuHeightAnimating {
		elapsed := now - e.menuHeightAnimStartTime
		if elapsed >= heightAnimDuration {
			e.menuHeightAnimating = false
			// Clear preserved state after animation completes
			e.prevMenuItems = nil
			e.prevMenuTitle = ""
			e.prevMenuHelpText = ""
		}
	}
	if e.menuHighlightAnimating {
		elapsed := now - e.menuHighlightAnimStartTime
		if elapsed >= highlightAnimDuration {
			e.menuHighlightAnimating = false
		}
	}

	// Detect selection change and start animation (only if menu was already active)
	if e.genericMenuActive && e.genericMenuSelected != selected && e.genericMenuSelected >= 0 {
		// Calculate widths for start and target items
		var startWidth, targetWidth float64
		if e.genericMenuSelected >= 0 && e.genericMenuSelected < len(e.genericMenuLabels) {
			startWidth = e.getMenuLabelWidth(e.genericMenuLabels[e.genericMenuSelected])
		}
		labels := e.snapshotMenuLabels(items)
		if selected >= 0 && selected < len(labels) {
			targetWidth = e.getMenuLabelWidth(labels[selected])
		}

		e.menuHighlightAnimStartIndex = e.genericMenuSelected
		e.menuHighlightAnimTargetIndex = selected
		e.menuHighlightAnimStartWidth = startWidth
		e.menuHighlightAnimTargetWidth = targetWidth
		e.menuHighlightAnimStartTime = time.Now().UnixMilli()
		e.menuHighlightAnimating = true
	} else if !e.genericMenuActive {
		// Menu just opened - no animation, start at target position
		e.menuHighlightAnimating = false
	}

	e.genericMenuActive = true
	e.genericMenuSelected = selected
	e.genericMenuHelpText = helpText
	e.genericMenuTitle = title
	e.genericMenuItems = make([]gamemenu.MenuItem, len(items))
	copy(e.genericMenuItems, items)
	e.genericMenuLabels = e.snapshotMenuLabels(items)

	// Initialize floating tiles background for main menu
	if title == "The Dark Station" {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Silently ignore initialization errors - menu should still work
				}
			}()
			e.floatingTilesMutex.Lock()
			defer e.floatingTilesMutex.Unlock()
			if len(e.floatingTiles) == 0 {
				// Try to get window size, use defaults if not available yet
				screenWidth, screenHeight := ebiten.WindowSize()
				if screenWidth <= 0 || screenHeight <= 0 {
					screenWidth = 1024
					screenHeight = 768
				}
				e.initFloatingTilesUnlocked(screenWidth, screenHeight)
			}
		}()
	}
}

// ClearMenu hides the generic menu overlay.
func (e *EbitenRenderer) ClearMenu() {
	e.genericMenuMutex.Lock()
	defer e.genericMenuMutex.Unlock()
	// Preserve menu state before clearing (for smooth transitions)
	if e.genericMenuActive {
		e.prevMenuItems = make([]gamemenu.MenuItem, len(e.genericMenuItems))
		copy(e.prevMenuItems, e.genericMenuItems)
		e.prevMenuLabels = append([]string(nil), e.genericMenuLabels...)
		e.prevMenuTitle = e.genericMenuTitle
		e.prevMenuHelpText = e.genericMenuHelpText
	}
	e.genericMenuActive = false
	e.genericMenuItems = nil
	e.genericMenuLabels = nil
	e.genericMenuHelpText = ""
	e.genericMenuTitle = ""
	e.menuHighlightAnimating = false
	// Clear previous state only after animation completes (handled in RenderMenu)
}

func (e *EbitenRenderer) isGenericMenuActive() bool {
	e.genericMenuMutex.RLock()
	defer e.genericMenuMutex.RUnlock()
	return e.genericMenuActive
}

// drawGenericMenuOverlay draws a semi-transparent panel over most of the screen
// with the menu list and a clear highlight for the selected entry.
func (e *EbitenRenderer) drawGenericMenuOverlay(screen *ebiten.Image) {
	e.genericMenuMutex.RLock()
	items := make([]gamemenu.MenuItem, len(e.genericMenuItems))
	copy(items, e.genericMenuItems)
	labels := append([]string(nil), e.genericMenuLabels...)
	selected := e.genericMenuSelected
	helpText := e.genericMenuHelpText
	title := e.genericMenuTitle
	animating := e.menuHighlightAnimating
	animStartIndex := e.menuHighlightAnimStartIndex
	animTargetIndex := e.menuHighlightAnimTargetIndex
	animStartWidth := e.menuHighlightAnimStartWidth
	animTargetWidth := e.menuHighlightAnimTargetWidth
	animStartTime := e.menuHighlightAnimStartTime
	heightAnimating := e.menuHeightAnimating
	heightStartHeight := e.menuHeightAnimStartHeight
	heightTargetHeight := e.menuHeightAnimTargetHeight
	heightStartTime := e.menuHeightAnimStartTime
	e.genericMenuMutex.RUnlock()

	if len(items) == 0 {
		return
	}

	e.gameMutex.RLock()
	pg := e.game
	e.gameMutex.RUnlock()

	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	maintOverlayStable := strings.Contains(title, "Maintenance Terminal") || title == "Select room"
	devMenuRight := title == "Developer Menu"
	currentCellCharsMenu := title == "Current Cell Chars"
	skipMenuDropShadow := maintOverlayStable &&
		maintCameraPanTweening(pg, e.cameraCenterRow, e.cameraCenterCol, e.cameraTargetRow, e.cameraTargetCol)
	// Default true so callouts/main menu stay soft; maint/room picker uses crisp vector edges (less LCD shimmer).
	useVectorAA := !maintOverlayStable

	// Calculate required height based on menu content
	requiredHeight := e.calculateMenuHeight(labels, title, helpText)

	// Animate height transition if menu changed
	const heightAnimDuration = 200 // milliseconds
	var panelH float64

	if heightAnimating {
		// Orange maintenance menus: skip fractional height tween — vector AA + non-integer rect height
		// showed as border shimmer while the map behind was panning.
		if maintOverlayStable {
			panelH = heightTargetHeight
		} else {
			now := e.menuAnimClockMilli
			if now == 0 {
				now = time.Now().UnixMilli()
			}
			elapsed := now - heightStartTime

			if elapsed >= heightAnimDuration {
				// Animation complete - use target height
				panelH = heightTargetHeight
				// Don't mark complete here - let RenderMenu handle it on next update to avoid deadlock
			} else {
				// Interpolate between start and target heights
				progress := float64(elapsed) / float64(heightAnimDuration)
				easedProgress := easeInOut(progress)
				panelH = heightStartHeight + (heightTargetHeight-heightStartHeight)*easedProgress
			}
		}
	} else {
		// No animation - use required height
		panelH = requiredHeight
	}

	if maintOverlayStable {
		panelH = float64(int(math.Round(panelH)))
	}

	// Panel width/position: maintenance menus are left-aligned at half the default width.
	const defaultPanelWidthFrac = 0.7
	const maintPanelWidthFrac = defaultPanelWidthFrac / 2
	const menuSideMargin = 16

	panelW := int(float32(screenWidth) * defaultPanelWidthFrac)
	if maintOverlayStable {
		panelW = int(float32(screenWidth) * maintPanelWidthFrac)
	}
	panelHInt := int(panelH)
	if panelHInt < 1 {
		panelHInt = 1
	}
	panelX := (screenWidth - panelW) / 2
	switch {
	case maintOverlayStable:
		panelX = menuSideMargin
	case devMenuRight:
		panelX = screenWidth - panelW - menuSideMargin
	}
	panelY := (screenHeight - panelHInt) / 2

	// Determine background transparency based on menu type
	// Main menu and bindings menu from main menu: more transparent (180)
	// All other menus (in-game menus): less transparent/more opaque (220)
	var bgAlpha uint8 = 220     // Default: more opaque for in-game menus
	var borderAlpha uint8 = 200 // Default: more opaque border for in-game menus

	if title == "The Dark Station" {
		// Main menu: same dark background as in-game menus
		bgAlpha = 220
		borderAlpha = 200
	} else if title == "Bindings Menu" {
		// Check if we're in main menu context (game state invalid)
		e.snapshotMutex.RLock()
		snapValid := e.snapshot.valid
		e.snapshotMutex.RUnlock()
		e.gameMutex.RLock()
		gameValid := e.game != nil
		e.gameMutex.RUnlock()

		// If game state is invalid, we're in main menu context
		if !snapValid || !gameValid {
			// Bindings menu from main menu: more transparent
			bgAlpha = 180
			borderAlpha = 180
		}
		// Otherwise, use default (more opaque) for bindings menu from in-game
	}

	bg := color.RGBA{10, 6, 16, bgAlpha} // Dark purple panel background

	// Border derives from title color (maintenance/select room = orange, others = purple)
	borderBase := colorAction
	if strings.Contains(title, "Maintenance Terminal") || title == "Select room" {
		borderBase = colorMaintenance
	}
	br, bgVal, bb, _ := borderBase.RGBA()
	border := color.RGBA{uint8(br >> 8), uint8(bgVal >> 8), uint8(bb >> 8), borderAlpha}

	panelHFloat := float32(panelH)
	panelWFloat := float32(panelW)
	const menuCornerRadius = 12
	const borderWidth = 2

	var path vector.Path

	// Drop shadow: drawn FIRST (behind panel), like CSS box-shadow. Ring only (never overlaps panel).
	// 0 offset, 8px spread, fade from outer (darker) to inner (lighter). Shadow derived from border color.
	// While the maintenance map ease runs, omit these rings — they add many vector fills on top of a
	// full viewport repaint and worsen uneven frame pacing (missed draws read as hitch).
	const shadowSpread = 8
	bor, bog, bob, _ := border.RGBA()
	shadowR := uint8((bor >> 8) * 15 / 255)
	shadowG := uint8((bog >> 8) * 15 / 255)
	shadowB := uint8((bob >> 8) * 15 / 255)
	if shadowR < 8 {
		shadowR = 8
	}
	if shadowG < 8 {
		shadowG = 8
	}
	if shadowB < 8 {
		shadowB = 8
	}
	if !skipMenuDropShadow {
		for i := shadowSpread; i >= 1; i-- {
			alpha := uint8(12 + i*8)
			if alpha > 55 {
				alpha = 55
			}
			path.Reset()
			appendRoundedRect(&path,
				float32(panelX-i), float32(panelY-i),
				panelWFloat+float32(i*2), panelHFloat+float32(i*2),
				menuCornerRadius+float32(i))
			appendRoundedRectDir(&path,
				float32(panelX-(i-1)), float32(panelY-(i-1)),
				panelWFloat+float32((i-1)*2), panelHFloat+float32((i-1)*2),
				menuCornerRadius+float32(i-1), vector.CounterClockwise)
			drawOpts := &vector.DrawPathOptions{AntiAlias: useVectorAA}
			drawOpts.ColorScale.ScaleWithColor(color.RGBA{shadowR, shadowG, shadowB, alpha})
			vector.FillPath(screen, &path, nil, drawOpts)
		}
	}

	// Panel background and border (drawn on top of shadow, fully covering center)
	path.Reset()
	appendRoundedRect(&path, float32(panelX), float32(panelY), panelWFloat, panelHFloat, menuCornerRadius)

	drawOpts := &vector.DrawPathOptions{AntiAlias: useVectorAA}
	drawOpts.ColorScale.ScaleWithColor(bg)
	vector.FillPath(screen, &path, nil, drawOpts)

	path.Reset()
	appendRoundedRect(&path, float32(panelX), float32(panelY), panelWFloat, panelHFloat, menuCornerRadius)
	strokeOpts := &vector.StrokeOptions{Width: borderWidth, MiterLimit: 10}
	drawOpts = &vector.DrawPathOptions{AntiAlias: useVectorAA}
	drawOpts.ColorScale.ScaleWithColor(border)
	vector.StrokePath(screen, &path, strokeOpts, drawOpts)

	paddingX := 24
	paddingY := 24
	x := panelX + paddingX
	y := panelY + paddingY

	fontSize := e.getUIFontSize()
	rowFace := e.getSansFontFace()
	if currentCellCharsMenu {
		rowFace = e.getMonoUIFontFace()
		fontSize = rowFace.Size
	}
	lineHeight := int(fontSize) + 6
	// Tighter spacing between title and the first line below it (help text or menu items)
	const titleToContentSpacing = 4

	// Use UI font metrics so the highlight rectangle can tightly wrap the text.
	_, textHeight := text.Measure("Ag", rowFace, 0)

	// Title color and highlight derive from menu type (maintenance/select room = orange, others = purple)
	titleColor := colorAction
	if strings.Contains(title, "Maintenance Terminal") || title == "Select room" {
		titleColor = colorMaintenance
	}
	highlightColor := color.RGBA{100, 60, 160, 255} // Dark purple for default menus
	if titleColor == colorMaintenance {
		highlightColor = color.RGBA{100, 65, 0, 255} // Dark orange for maintenance menus
	}

	// Title (bold font, 2pt larger than body text)
	if title != "" {
		titleFace := e.getSansBoldTitleFontFace()
		e.drawColoredTextWithFace(screen, title, x, y-int(fontSize), titleColor, titleFace)
		// Advance by full title line height (not just titleToContentSpacing) so help text does not overlap.
		y += int(titleFace.Size) + titleToContentSpacing
	}

	// Show help text if provided (parse markup for proper colors)
	if helpText != "" {
		helpSegments := e.parseMarkup(helpText)
		if len(helpSegments) > 0 {
			e.drawColoredTextSegments(screen, helpSegments, x, y-int(fontSize))
		} else {
			e.drawColoredText(screen, helpText, x, y-int(fontSize), colorAction)
		}
		y += lineHeight
	}

	y += lineHeight

	menuRowOffset := func(index int) float64 {
		if index <= 0 {
			return 0
		}
		off := 0.0
		fs := int(e.getUIFontSize())
		for i := 0; i < index && i < len(labels); i++ {
			if renderer.IsPowerBarLine(labels[i]) {
				off += float64(powerBarMenuRowHeight(fs))
			} else {
				off += float64(lineHeight)
			}
		}
		return off
	}

	// Calculate animated highlight position and width
	const animDuration = 150 // milliseconds
	var highlightY float64
	var highlightWidth float64
	var highlightIndex int

	if animating {
		now := e.menuAnimClockMilli
		if now == 0 {
			now = time.Now().UnixMilli()
		}
		elapsed := now - animStartTime

		if elapsed >= animDuration {
			// Animation complete - use target position and width
			highlightIndex = animTargetIndex
			highlightY = float64(y) + menuRowOffset(animTargetIndex) + fontSize
			highlightWidth = animTargetWidth
			// Don't mark complete here - let RenderMenu handle it on next update to avoid deadlock
		} else {
			// Interpolate between start and target positions and widths
			progress := float64(elapsed) / float64(animDuration)
			// Use ease-in-out for smooth animation
			easedProgress := easeInOut(progress)

			startY := float64(y) + menuRowOffset(animStartIndex) + fontSize
			targetY := float64(y) + menuRowOffset(animTargetIndex) + fontSize
			highlightY = startY + (targetY-startY)*easedProgress
			highlightWidth = animStartWidth + (animTargetWidth-animStartWidth)*easedProgress
			highlightIndex = animTargetIndex // Use target index for item check
		}
	} else {
		// No animation - use current selected position and width
		highlightIndex = selected
		highlightY = float64(y) + menuRowOffset(selected) + fontSize
		if selected >= 0 && selected < len(labels) {
			highlightWidth = e.getMenuLabelWidth(labels[selected])
		} else {
			highlightWidth = 0
		}
	}
	if title == "Bindings Menu" {
		highlightWidth = float64(panelW - paddingX*2)
	}

	// First pass: draw highlight rectangles (so they are always below text)
	if highlightIndex >= 0 && highlightIndex < len(items) && items[highlightIndex].IsSelectable() && highlightWidth > 0 {
		rectTop := highlightY
		rectHeight := float64(textHeight + 4) // small padding below glyphs
		// Add padding on left and right sides of text
		const paddingX = 8.0
		rx := float32(x - paddingX)
		ry := float32(rectTop)
		rw := float32(highlightWidth + paddingX*2)
		rh := float32(rectHeight)
		if maintOverlayStable && !animating {
			rx = float32(math.Round(float64(rx)))
			ry = float32(math.Round(float64(ry)))
			rw = float32(math.Round(float64(rw)))
			rh = float32(math.Round(float64(rh)))
		}
		vector.DrawFilledRect(screen, rx, ry, rw, rh, highlightColor, false)
	}

	// For table-style menus: align values in columns (tab-separated: label, status, optional watts/value).
	var valueColumnX, wattsColumnX int
	rightAlignPowerColumn := title == "Select room"
	tableMenu := strings.Contains(title, "Maintenance Terminal") ||
		title == "Select room" ||
		title == "Bindings Menu" ||
		title == "Developer Menu" ||
		currentCellCharsMenu ||
		title == "Video"
	if tableMenu {
		var maxLabelW, maxValueW float64
		for _, label := range labels {
			before, after, ok := strings.Cut(label, "\t")
			if !ok || before == "" {
				continue
			}
			if w := e.getTextWidthWithFace(before, rowFace); w > maxLabelW {
				maxLabelW = w
			}
			valuePart := after
			wattsPart := ""
			if middle, w, hasWatts := strings.Cut(after, "\t"); hasWatts {
				valuePart, wattsPart = middle, w
			}
			if valuePart != "" {
				if w := e.getMarkupWidthWithFace(valuePart, rowFace); w > maxValueW {
					maxValueW = w
				}
			}
			if rightAlignPowerColumn && wattsPart != "" {
				if w := e.getMarkupWidthWithFace(wattsPart, rowFace); w > maxValueW {
					maxValueW = w
				}
			}
		}
		valueColumnGap := 8 // pixels between columns
		if title == "Bindings Menu" {
			valueColumnGap = 32
		}
		valueColumnX = x + int(maxLabelW) + valueColumnGap
		if rightAlignPowerColumn {
			// Third column right-aligned: draw at (rightEdge - width)
			wattsColumnX = panelX + panelW - paddingX
		} else {
			wattsColumnX = valueColumnX + int(maxValueW) + valueColumnGap
		}
	}

	// Second pass: draw menu items on top of the highlights
	rowY := y
	for i, item := range items {
		if i >= len(labels) {
			break
		}
		label := labels[i]
		rowParamY := rowY

		if renderer.IsPowerBarLine(label) {
			rowH := e.drawPowerBarMenuRow(screen, x, rowParamY, panelW, paddingX, label, 1.0)
			rowY += rowH
			continue
		}

		if title == "Bindings Menu" && !item.IsSelectable() && strings.HasPrefix(label, "TITLE{") {
			e.drawColoredTextSegmentsWithFace(screen, e.parseMarkup(label), x, rowParamY, e.getSansBoldTitleFontFace())
			rowY += lineHeight
			continue
		}

		// Maintenance terminal: draw label, value, and optional watts in columns if tab-separated
		if valueColumnX > x {
			if before, after, ok := strings.Cut(label, "\t"); ok {
				if before != "" {
					if strings.Contains(before, "{") {
						segments := e.parseMarkup(before)
						e.drawColoredTextSegmentsWithFace(screen, segments, x, rowParamY, rowFace)
					} else {
						labelColor := colorText
						if !item.IsSelectable() {
							labelColor = colorSubtle
						}
						e.drawColoredTextWithFace(screen, before, x, rowParamY, labelColor, rowFace)
					}
				}
				valuePart, wattsPart := after, ""
				if middle, w, hasWatts := strings.Cut(after, "\t"); hasWatts {
					valuePart, wattsPart = middle, w
				}
				if valuePart != "" {
					segments := e.parseMarkup(valuePart)
					if len(segments) > 0 {
						e.drawColoredTextSegmentsWithFace(screen, segments, valueColumnX, rowParamY, rowFace)
					} else {
						labelColor := colorText
						if !item.IsSelectable() {
							labelColor = colorSubtle
						}
						e.drawColoredTextWithFace(screen, valuePart, valueColumnX, rowParamY, labelColor, rowFace)
					}
				}
				if wattsPart != "" && wattsColumnX > valueColumnX {
					wattsX := wattsColumnX
					if rightAlignPowerColumn {
						wattsX = wattsColumnX - int(e.getMarkupWidthWithFace(wattsPart, rowFace))
					}
					segments := e.parseMarkup(wattsPart)
					if len(segments) > 0 {
						e.drawColoredTextSegmentsWithFace(screen, segments, wattsX, rowParamY, rowFace)
					} else {
						labelColor := colorText
						if !item.IsSelectable() {
							labelColor = colorSubtle
						}
						e.drawColoredTextWithFace(screen, wattsPart, wattsX, rowParamY, labelColor, rowFace)
					}
				}
				rowY += lineHeight
				continue
			}
		}

		// Parse markup and draw with proper colors
		segments := e.parseMarkup(label)
		if len(segments) > 0 {
			e.drawColoredTextSegmentsWithFace(screen, segments, x, rowParamY, rowFace)
		} else {
			// Fallback: use different color for non-selectable items
			labelColor := colorText
			if !item.IsSelectable() {
				labelColor = colorSubtle
			}
			e.drawColoredTextWithFace(screen, label, x, rowParamY, labelColor, rowFace)
		}
		rowY += lineHeight
	}
}

// getMarkupWidth returns the total width in pixels of a string that may contain markup (e.g. ACTION{}, POWERED{}).
func (e *EbitenRenderer) getMarkupWidth(s string) float64 {
	return e.getMarkupWidthWithFace(s, e.getSansFontFace())
}

func (e *EbitenRenderer) getMarkupWidthWithFace(s string, face *text.GoTextFace) float64 {
	segments := e.parseMarkup(s)
	var w float64
	for _, seg := range segments {
		w += e.getTextWidthWithFace(seg.text, face)
	}
	return w
}

func (e *EbitenRenderer) snapshotMenuLabels(items []gamemenu.MenuItem) []string {
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.GetLabel()
	}
	return labels
}

// getMenuLabelWidth calculates the width of a menu label (accounting for markup).
func (e *EbitenRenderer) getMenuLabelWidth(label string) float64 {
	if renderer.IsPowerBarLine(label) {
		return 220
	}
	// Parse markup to get actual text segments
	segments := e.parseMarkup(label)

	// Sum up the width of all segments
	var totalWidth float64
	for _, seg := range segments {
		totalWidth += e.getTextWidth(seg.text)
	}
	return totalWidth
}

// calculateMenuHeight calculates the required height for a menu based on its content
func (e *EbitenRenderer) calculateMenuHeight(labels []string, title string, helpText string) float64 {
	fontSize := e.getUIFontSize()
	face := e.getSansFontFace()
	if title == "Current Cell Chars" {
		face = e.getMonoUIFontFace()
		fontSize = face.Size
	}
	lineHeight := float64(int(fontSize) + 6)
	paddingY := 24.0 * 2            // Top and bottom padding
	const titleToContentSpacing = 4 // Must match drawGenericMenuOverlay

	height := paddingY

	// Title (bold 2pt larger; uses tighter spacing to first line below)
	if title != "" {
		titleFontSize := fontSize + 2
		height += titleFontSize + titleToContentSpacing
	}

	// Help text
	if helpText != "" {
		height += lineHeight
	}

	// Spacing before menu items (this is an extra lineHeight added after help text)
	height += lineHeight

	// Menu items (power bar rows are taller)
	for _, label := range labels {
		if renderer.IsPowerBarLine(label) {
			height += float64(powerBarMenuRowHeight(int(fontSize)))
		} else {
			height += lineHeight
		}
	}

	// Add extra space at bottom to account for text baseline and ensure last item is fully visible
	// The last menu item's text extends below its baseline, so we need a bit more room
	_, textHeight := text.Measure("Ag", face, 0)
	// Add space for text height below baseline (textHeight includes the full glyph box)
	height += textHeight - fontSize + 8 // Extra buffer for comfortable spacing

	return height
}

// easeInOut provides smooth easing for animations (ease-in-out cubic)
func easeInOut(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	return 1 - math.Pow(-2*t+2, 3)/2
}
