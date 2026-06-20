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

	// Detect menu title change (menu switch) and start panel transition / height animation.
	var titleChanged bool
	var oldHeight float64
	var fromTitle, fromHelp string
	var fromLabels []string

	if e.genericMenuActive && e.genericMenuTitle != title {
		titleChanged = true
		fromTitle = e.genericMenuTitle
		fromLabels = e.genericMenuLabels
		fromHelp = e.genericMenuHelpText
		oldHeight = e.calculateMenuHeight(fromLabels, fromTitle, fromHelp, e.genericMenuItems)
	} else if !e.genericMenuActive && len(e.prevMenuItems) > 0 && e.prevMenuTitle != title {
		titleChanged = true
		fromTitle = e.prevMenuTitle
		fromLabels = e.prevMenuLabels
		fromHelp = e.prevMenuHelpText
		oldHeight = e.calculateMenuHeight(fromLabels, fromTitle, fromHelp, e.prevMenuItems)
	}

	toLabels := e.snapshotMenuLabels(items)
	newHeight := e.calculateMenuHeight(toLabels, title, helpText, items)
	now := time.Now().UnixMilli()

	e.finishMenuHeightAnimationIfDone(now)

	if title == "Settings" && e.genericMenuActive && e.genericMenuTitle == title &&
		!e.menuPanelTransitionAnimating {
		e.beginSettingsMenuHeightAnim(now, newHeight)
	}

	if titleChanged && titleScreenMenuTransition(fromTitle, title) {
		screenWidth, screenHeight := e.floatingTileScreenSize()
		e.beginTitleScreenMenuTransition(fromTitle, title, fromHelp, helpText, fromLabels, toLabels, items, screenWidth, screenHeight)
	} else if titleChanged {
		e.menuHeightAnimStartHeight = oldHeight
		e.menuHeightAnimTargetHeight = newHeight
		e.menuHeightAnimStartTime = now
		e.menuHeightAnimating = true
	} else if !e.genericMenuActive && len(e.prevMenuItems) == 0 {
		e.menuHeightAnimating = false
		e.menuPanelTransitionAnimating = false
		e.menuHeightAnimTargetHeight = e.calculateMenuHeight(toLabels, title, helpText, items)
	}

	const highlightAnimDuration = 150
	if e.menuPanelTransitionAnimating {
		if _, done := menuTransitionProgress(now, e.menuPanelTransitionStartMs); done {
			e.menuPanelTransitionAnimating = false
			e.prevMenuItems = nil
			e.prevMenuTitle = ""
			e.prevMenuHelpText = ""
			e.prevMenuSelected = 0
		}
	}

	if title == "Settings" {
		if !e.menuPanelTransitionAnimating && !e.menuHeightAnimating {
			e.settingsMenuHeightBaseline = newHeight
		}
	} else if titleChanged || !e.genericMenuActive {
		e.settingsMenuHeightBaseline = 0
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

	// Keep the title-screen ambient field alive across main menu sub-menus.
	if titleScreenFloatingTilesMenu(title) {
		screenWidth, screenHeight := e.floatingTileScreenSize()
		e.ensureFloatingTiles(screenWidth, screenHeight)
	}
}

// ClearMenu hides the generic menu overlay.
func (e *EbitenRenderer) ClearMenu() {
	e.genericMenuMutex.Lock()
	defer e.genericMenuMutex.Unlock()
	// Preserve menu state before clearing (for smooth transitions)
	if e.genericMenuActive {
		if e.genericMenuTitle == "Settings" {
			e.settingsMenuHeightBaseline = 0
		}
		e.prevMenuItems = make([]gamemenu.MenuItem, len(e.genericMenuItems))
		copy(e.prevMenuItems, e.genericMenuItems)
		e.prevMenuLabels = append([]string(nil), e.genericMenuLabels...)
		e.prevMenuTitle = e.genericMenuTitle
		e.prevMenuHelpText = e.genericMenuHelpText
		e.prevMenuSelected = e.genericMenuSelected
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
	panelTransition := e.menuPanelTransitionAnimating
	panelTransitionForward := e.menuPanelTransitionForward
	panelTransitionStart := e.menuPanelTransitionStartMs
	panelTransitionFromW := e.menuPanelTransitionFromW
	panelTransitionToW := e.menuPanelTransitionToW
	panelTransitionFromH := e.menuPanelTransitionFromH
	panelTransitionToH := e.menuPanelTransitionToH
	prevItems := make([]gamemenu.MenuItem, len(e.prevMenuItems))
	copy(prevItems, e.prevMenuItems)
	prevLabels := append([]string(nil), e.prevMenuLabels...)
	prevTitle := e.prevMenuTitle
	prevHelp := e.prevMenuHelpText
	prevSelected := e.prevMenuSelected
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

	requiredHeight := e.calculateMenuHeight(labels, title, helpText, items)

	var panelH float64
	var panelResizeEased float64
	var contentFadeEased float64

	if panelTransition {
		now := e.menuAnimClockMilli
		if now == 0 {
			now = time.Now().UnixMilli()
		}
		e.finishPanelTransitionIfDone(now)
		panelResizeEased = menuTransitionPanelProgress(now, panelTransitionStart)
		contentFadeEased = menuTransitionContentProgress(now, panelTransitionStart)
		panelH = lerpFloat(panelTransitionFromH, panelTransitionToH, panelResizeEased)
	} else if heightAnimating {
		// Orange maintenance menus: skip fractional height tween — vector AA + non-integer rect height
		// showed as border shimmer while the map behind was panning.
		if maintOverlayStable {
			panelH = heightTargetHeight
		} else {
			now := e.menuAnimClockMilli
			if now == 0 {
				now = time.Now().UnixMilli()
			}
			e.finishMenuHeightAnimationIfDone(now)
			e.genericMenuMutex.RLock()
			heightAnimating = e.menuHeightAnimating
			heightStartHeight = e.menuHeightAnimStartHeight
			heightTargetHeight = e.menuHeightAnimTargetHeight
			heightStartTime = e.menuHeightAnimStartTime
			e.genericMenuMutex.RUnlock()
			var easedProgress float64
			if title == "Settings" {
				easedProgress = settingsMenuHeightAnimProgress(now, heightStartTime)
			} else {
				elapsed := now - heightStartTime
				heightDur := menuHeightAnimDurationMs(title)
				if elapsed >= heightDur {
					easedProgress = 1
				} else {
					easedProgress = easeInOut(float64(elapsed) / float64(heightDur))
				}
			}
			if heightAnimating {
				panelH = heightStartHeight + (heightTargetHeight-heightStartHeight)*easedProgress
			} else {
				panelH = requiredHeight
			}
		}
	} else {
		// No animation - use required height
		panelH = requiredHeight
	}

	if maintOverlayStable {
		panelH = float64(int(math.Round(panelH)))
	}

	const menuSideMargin = 16

	panelW := menuPanelWidthForTitle(title, screenWidth, maintOverlayStable, devMenuRight)
	if panelTransition {
		panelW = lerpInt(panelTransitionFromW, panelTransitionToW, panelResizeEased)
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
	if settingsMenuUsesFixedTop(title) {
		panelY = settingsMenuPanelTopY
	}

	// Determine background transparency based on menu type
	// Main menu and bindings menu from main menu: more transparent (180)
	// All other menus (in-game menus): less transparent/more opaque (220)
	var bgAlpha uint8 = 220     // Default: more opaque for in-game menus
	var borderAlpha uint8 = 200 // Default: more opaque border for in-game menus

	if title == "The Dark Station" {
		// Main menu: same dark background as in-game menus
		bgAlpha = 220
		borderAlpha = 200
	} else if title == "Settings" {
		e.snapshotMutex.RLock()
		snapValid := e.snapshot.valid
		e.snapshotMutex.RUnlock()
		e.gameMutex.RLock()
		gameValid := e.game != nil
		e.gameMutex.RUnlock()
		if !snapValid || !gameValid {
			bgAlpha = 180
			borderAlpha = 180
		}
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

	baseContent := menuPanelContentParams{
		screenWidth:          screenWidth,
		screenHeight:         screenHeight,
		maintOverlayStable:   maintOverlayStable,
		devMenuRight:         devMenuRight,
		currentCellCharsMenu: currentCellCharsMenu,
		useVectorAA:          useVectorAA,
		panelX:               panelX,
		panelY:               panelY,
		panelW:               panelW,
		panelH:               panelH,
	}

	if panelTransition && len(prevItems) > 0 {
		outSlide := menuTransitionOutgoingSlide(panelTransitionForward, panelResizeEased)
		inSlide := menuTransitionIncomingSlide(panelTransitionForward, contentFadeEased)
		outAlpha := 1 - panelResizeEased
		inAlpha := contentFadeEased
		prevParams := baseContent
		prevParams.items = prevItems
		prevParams.labels = prevLabels
		prevParams.title = prevTitle
		prevParams.helpText = prevHelp
		prevParams.selected = prevSelected
		prevParams.contentSlideX = outSlide
		prevParams.alpha = outAlpha
		e.drawMenuPanelContent(screen, prevParams)

		currParams := baseContent
		currParams.items = items
		currParams.labels = labels
		currParams.title = title
		currParams.helpText = helpText
		currParams.selected = selected
		currParams.contentSlideX = inSlide
		currParams.alpha = inAlpha
		currParams.highlightAnimating = animating
		currParams.highlightAnimStartIndex = animStartIndex
		currParams.highlightTarget = animTargetIndex
		currParams.highlightAnimStartWidth = animStartWidth
		currParams.highlightTargetWidth = animTargetWidth
		currParams.highlightAnimStartTime = animStartTime
		e.drawMenuPanelContent(screen, currParams)
		return
	}

	currParams := baseContent
	currParams.items = items
	currParams.labels = labels
	currParams.title = title
	currParams.helpText = helpText
	currParams.selected = selected
	currParams.alpha = 1
	currParams.highlightAnimating = animating
	currParams.highlightAnimStartIndex = animStartIndex
	currParams.highlightTarget = animTargetIndex
	currParams.highlightAnimStartWidth = animStartWidth
	currParams.highlightTargetWidth = animTargetWidth
	currParams.highlightAnimStartTime = animStartTime
	e.drawMenuPanelContent(screen, currParams)
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
	if renderer.IsInventoryRowLine(label) {
		return e.inventoryMenuLabelWidth(label)
	}
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
func (e *EbitenRenderer) calculateMenuHeight(labels []string, title string, helpText string, items []gamemenu.MenuItem) float64 {
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

	// Menu items (power bar rows are taller). Settings tabs share one row.
	stripLen := settingsTabStripItemCount(labels, title, items)
	start := 0
	if stripLen >= 2 {
		tabTextH := settingsTabLabelsMaxHeight(labels, stripLen, face)
		if tabTextH <= 0 {
			_, tabTextH = text.Measure("Ag", face, 0)
		}
		height += float64(settingsTabStripBlockHeight(tabTextH))
		start = stripLen
	}
	for i := start; i < len(labels); i++ {
		label := labels[i]
		if renderer.IsPowerBarLine(label) {
			height += float64(powerBarMenuRowHeight(int(fontSize)))
		} else if renderer.IsInventoryRowLine(label) {
			height += float64(inventoryMenuRowHeight(int(fontSize)))
	} else {
			height += lineHeight
			height += float64(inventorySectionHeaderExtraGap(title, labels, i, int(lineHeight)))
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
