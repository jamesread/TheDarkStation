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
		oldHeight = e.calculateMenuHeight(e.genericMenuItems, e.genericMenuTitle, e.genericMenuHelpText)
	} else if !e.genericMenuActive && len(e.prevMenuItems) > 0 && e.prevMenuTitle != title {
		// Menu was cleared but we have previous state - use it for transition
		titleChanged = true
		oldHeight = e.calculateMenuHeight(e.prevMenuItems, e.prevMenuTitle, e.prevMenuHelpText)
	}

	if titleChanged {
		// Calculate required heights for both menus
		newHeight := e.calculateMenuHeight(items, title, helpText)

		e.menuHeightAnimStartHeight = oldHeight
		e.menuHeightAnimTargetHeight = newHeight
		e.menuHeightAnimStartTime = time.Now().UnixMilli()
		e.menuHeightAnimating = true
	} else if !e.genericMenuActive && len(e.prevMenuItems) == 0 {
		// Menu just opened (no previous state) - no height animation, set target height immediately
		e.menuHeightAnimating = false
		e.menuHeightAnimTargetHeight = e.calculateMenuHeight(items, title, helpText)
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
		if e.genericMenuSelected >= 0 && e.genericMenuSelected < len(e.genericMenuItems) {
			startWidth = e.getMenuItemWidth(e.genericMenuItems[e.genericMenuSelected])
		}
		if selected >= 0 && selected < len(items) {
			targetWidth = e.getMenuItemWidth(items[selected])
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
		e.prevMenuTitle = e.genericMenuTitle
		e.prevMenuHelpText = e.genericMenuHelpText
	}
	e.genericMenuActive = false
	e.genericMenuItems = nil
	e.genericMenuHelpText = ""
	e.genericMenuTitle = ""
	e.menuHighlightAnimating = false
	// Clear previous state only after animation completes (handled in RenderMenu)
}

// drawGenericMenuOverlay draws a semi-transparent panel over most of the screen
// with the menu list and a clear highlight for the selected entry.
func (e *EbitenRenderer) drawGenericMenuOverlay(screen *ebiten.Image) {
	e.genericMenuMutex.RLock()
	items := make([]gamemenu.MenuItem, len(e.genericMenuItems))
	copy(items, e.genericMenuItems)
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

	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Calculate required height based on menu content
	requiredHeight := e.calculateMenuHeight(items, title, helpText)

	// Animate height transition if menu changed
	const heightAnimDuration = 200 // milliseconds
	var panelH float64

	if heightAnimating {
		now := time.Now().UnixMilli()
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
	} else {
		// No animation - use required height
		panelH = requiredHeight
	}

	// Panel covers ~70% of screen width, height is dynamic based on content
	panelW := int(float32(screenWidth) * 0.7)
	panelHInt := int(panelH)
	panelX := (screenWidth - panelW) / 2
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
		drawOpts := &vector.DrawPathOptions{AntiAlias: true}
		drawOpts.ColorScale.ScaleWithColor(color.RGBA{shadowR, shadowG, shadowB, alpha})
		vector.FillPath(screen, &path, nil, drawOpts)
	}

	// Panel background and border (drawn on top of shadow, fully covering center)
	path.Reset()
	appendRoundedRect(&path, float32(panelX), float32(panelY), panelWFloat, panelHFloat, menuCornerRadius)

	drawOpts := &vector.DrawPathOptions{AntiAlias: true}
	drawOpts.ColorScale.ScaleWithColor(bg)
	vector.FillPath(screen, &path, nil, drawOpts)

	path.Reset()
	appendRoundedRect(&path, float32(panelX), float32(panelY), panelWFloat, panelHFloat, menuCornerRadius)
	strokeOpts := &vector.StrokeOptions{Width: borderWidth, MiterLimit: 10}
	drawOpts = &vector.DrawPathOptions{AntiAlias: true}
	drawOpts.ColorScale.ScaleWithColor(border)
	vector.StrokePath(screen, &path, strokeOpts, drawOpts)

	paddingX := 24
	paddingY := 24
	x := panelX + paddingX
	y := panelY + paddingY

	fontSize := e.getUIFontSize()
	lineHeight := int(fontSize) + 6
	// Tighter spacing between title and the first line below it (help text or menu items)
	const titleToContentSpacing = 4

	// Use UI font metrics so the highlight rectangle can tightly wrap the text.
	face := e.getSansFontFace()
	_, textHeight := text.Measure("Ag", face, 0)

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
		e.drawColoredTextWithFace(screen, title, x, y-int(fontSize), titleColor, e.getSansBoldTitleFontFace())
		y += titleToContentSpacing
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

	// Calculate animated highlight position and width
	const animDuration = 150 // milliseconds
	var highlightY float64
	var highlightWidth float64
	var highlightIndex int

	if animating {
		now := time.Now().UnixMilli()
		elapsed := now - animStartTime

		if elapsed >= animDuration {
			// Animation complete - use target position and width
			highlightIndex = animTargetIndex
			highlightY = float64(y+animTargetIndex*lineHeight) + fontSize
			highlightWidth = animTargetWidth
			// Don't mark complete here - let RenderMenu handle it on next update to avoid deadlock
		} else {
			// Interpolate between start and target positions and widths
			progress := float64(elapsed) / float64(animDuration)
			// Use ease-in-out for smooth animation
			easedProgress := easeInOut(progress)

			startY := float64(y+animStartIndex*lineHeight) + fontSize
			targetY := float64(y+animTargetIndex*lineHeight) + fontSize
			highlightY = startY + (targetY-startY)*easedProgress
			highlightWidth = animStartWidth + (animTargetWidth-animStartWidth)*easedProgress
			highlightIndex = animTargetIndex // Use target index for item check
		}
	} else {
		// No animation - use current selected position and width
		highlightIndex = selected
		highlightY = float64(y+selected*lineHeight) + fontSize
		if selected >= 0 && selected < len(items) {
			highlightWidth = e.getMenuItemWidth(items[selected])
		} else {
			highlightWidth = 0
		}
	}

	// First pass: draw highlight rectangles (so they are always below text)
	if highlightIndex >= 0 && highlightIndex < len(items) && items[highlightIndex].IsSelectable() && highlightWidth > 0 {
		rectTop := highlightY
		rectHeight := float64(textHeight + 4) // small padding below glyphs
		// Add padding on left and right sides of text
		const paddingX = 8.0
		vector.DrawFilledRect(screen,
			float32(x-paddingX), float32(rectTop),
			float32(highlightWidth+paddingX*2), float32(rectHeight),
			highlightColor, false)
	}

	// For maintenance terminal menus: align values in columns (tab-separated: label, status, optional watts)
	var valueColumnX, wattsColumnX int
	rightAlignPowerColumn := title == "Select room"
	if strings.Contains(title, "Maintenance Terminal") || title == "Select room" {
		var maxLabelW, maxValueW float64
		for _, item := range items {
			label := item.GetLabel()
			before, after, ok := strings.Cut(label, "\t")
			if !ok || before == "" {
				continue
			}
			if w := e.getTextWidth(before); w > maxLabelW {
				maxLabelW = w
			}
			valuePart := after
			wattsPart := ""
			if middle, w, hasWatts := strings.Cut(after, "\t"); hasWatts {
				valuePart, wattsPart = middle, w
			}
			if valuePart != "" {
				if w := e.getMarkupWidth(valuePart); w > maxValueW {
					maxValueW = w
				}
			}
			if rightAlignPowerColumn && wattsPart != "" {
				if w := e.getMarkupWidth(wattsPart); w > maxValueW {
					maxValueW = w
				}
			}
		}
		const valueColumnGap = 8 // pixels between columns
		valueColumnX = x + int(maxLabelW) + valueColumnGap
		if rightAlignPowerColumn {
			// Third column right-aligned: draw at (rightEdge - width)
			wattsColumnX = panelX + panelW - paddingX
		} else {
			wattsColumnX = valueColumnX + int(maxValueW) + valueColumnGap
		}
	}

	// Second pass: draw menu items on top of the highlights
	for i, item := range items {
		label := item.GetLabel()

		// Use a shared origin for text and rectangle calculations (see above).
		rowParamY := y + i*lineHeight

		// Maintenance terminal: draw label, value, and optional watts in columns if tab-separated
		if valueColumnX > x {
			if before, after, ok := strings.Cut(label, "\t"); ok {
				if before != "" {
					if strings.Contains(before, "{") {
						segments := e.parseMarkup(before)
						e.drawColoredTextSegments(screen, segments, x, rowParamY)
					} else {
						labelColor := colorText
						if !item.IsSelectable() {
							labelColor = colorSubtle
						}
						e.drawColoredText(screen, before, x, rowParamY, labelColor)
					}
				}
				valuePart, wattsPart := after, ""
				if middle, w, hasWatts := strings.Cut(after, "\t"); hasWatts {
					valuePart, wattsPart = middle, w
				}
				if valuePart != "" {
					segments := e.parseMarkup(valuePart)
					if len(segments) > 0 {
						e.drawColoredTextSegments(screen, segments, valueColumnX, rowParamY)
					} else {
						labelColor := colorText
						if !item.IsSelectable() {
							labelColor = colorSubtle
						}
						e.drawColoredText(screen, valuePart, valueColumnX, rowParamY, labelColor)
					}
				}
				if wattsPart != "" && wattsColumnX > valueColumnX {
					wattsX := wattsColumnX
					if rightAlignPowerColumn {
						wattsX = wattsColumnX - int(e.getMarkupWidth(wattsPart))
					}
					segments := e.parseMarkup(wattsPart)
					if len(segments) > 0 {
						e.drawColoredTextSegments(screen, segments, wattsX, rowParamY)
					} else {
						labelColor := colorText
						if !item.IsSelectable() {
							labelColor = colorSubtle
						}
						e.drawColoredText(screen, wattsPart, wattsX, rowParamY, labelColor)
					}
				}
				continue
			}
		}

		// Parse markup and draw with proper colors
		segments := e.parseMarkup(label)
		if len(segments) > 0 {
			e.drawColoredTextSegments(screen, segments, x, rowParamY)
		} else {
			// Fallback: use different color for non-selectable items
			labelColor := colorText
			if !item.IsSelectable() {
				labelColor = colorSubtle
			}
			e.drawColoredText(screen, label, x, rowParamY, labelColor)
		}
	}

	// Draw version information in bottom right corner (only for main menu)
	if title == "The Dark Station" {
		versionText := fmt.Sprintf(gotext.Get("VERSION"), renderer.Version)
		if renderer.Commit != "unknown" && len(renderer.Commit) > 0 {
			versionText += fmt.Sprintf(" (%s)", renderer.Commit[:7])
		}
		versionWidth := e.getTextWidth(versionText)
		margin := 16 // Margin from screen edges
		versionX := screenWidth - int(versionWidth) - margin
		// Position version text at bottom right corner
		// Note: drawColoredText uses baseline positioning (adds fontSize to Y internally)
		// To position text at bottom of screen: y = screenHeight - margin - (textHeight * 2)
		// This accounts for baseline offset and text height below baseline
		_, textHeight := text.Measure(versionText, face, 0)
		versionY := screenHeight - margin - int(textHeight*2)
		e.drawColoredText(screen, versionText, versionX, versionY, colorSubtle)
	}
}

// getMarkupWidth returns the total width in pixels of a string that may contain markup (e.g. ACTION{}, POWERED{}).
func (e *EbitenRenderer) getMarkupWidth(s string) float64 {
	segments := e.parseMarkup(s)
	var w float64
	for _, seg := range segments {
		w += e.getTextWidth(seg.text)
	}
	return w
}

// getMenuItemWidth calculates the width of a menu item's label text (accounting for markup)
func (e *EbitenRenderer) getMenuItemWidth(item gamemenu.MenuItem) float64 {
	label := item.GetLabel()
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
func (e *EbitenRenderer) calculateMenuHeight(items []gamemenu.MenuItem, title string, helpText string) float64 {
	fontSize := e.getUIFontSize()
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

	// Menu items
	height += float64(len(items)) * lineHeight

	// Add extra space at bottom to account for text baseline and ensure last item is fully visible
	// The last menu item's text extends below its baseline, so we need a bit more room
	face := e.getSansFontFace()
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
