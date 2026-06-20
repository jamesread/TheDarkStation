package ebiten

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"darkstation/pkg/game/renderer"
)

func colorWithAlpha(c color.Color, alpha float64) color.Color {
	if alpha >= 1 {
		return c
	}
	if alpha <= 0 {
		return color.RGBA{0, 0, 0, 0}
	}
	r, g, b, a := c.RGBA()
	return color.RGBA{
		uint8(float64(r>>8) * alpha),
		uint8(float64(g>>8) * alpha),
		uint8(float64(b>>8) * alpha),
		uint8(float64(a>>8) * alpha),
	}
}

// drawMenuPanelContent draws title, help, highlights, and rows inside an existing panel chrome.
func (e *EbitenRenderer) drawMenuPanelContent(screen *ebiten.Image, p menuPanelContentParams) {
	if p.alpha <= 0 || len(p.items) == 0 {
		return
	}

	paddingX := 24
	paddingY := 24
	x := p.panelX + paddingX + int(math.Round(p.contentSlideX))
	y := p.panelY + paddingY

	fontSize := e.getUIFontSize()
	rowFace := e.getSansFontFace()
	if p.currentCellCharsMenu {
		rowFace = e.getMonoUIFontFace()
		fontSize = rowFace.Size
	}
	lineHeight := int(fontSize) + 6
	const titleToContentSpacing = 4

	_, textHeight := text.Measure("Ag", rowFace, 0)

	var titleColor color.Color = colorAction
	if strings.Contains(p.title, "Maintenance Terminal") || p.title == "Select room" {
		titleColor = colorMaintenance
	}
	highlightColor := color.RGBA{100, 60, 160, 255}
	if titleColor == colorMaintenance {
		highlightColor = color.RGBA{100, 65, 0, 255}
	}
	titleColor = colorWithAlpha(titleColor, p.alpha)
	highlightCol := colorWithAlpha(highlightColor, p.alpha)

	if p.title != "" {
		titleFace := e.getSansBoldTitleFontFace()
		e.drawColoredTextWithFace(screen, p.title, x, y-int(fontSize), titleColor, titleFace)
		y += int(titleFace.Size) + titleToContentSpacing
	}

	if p.helpText != "" {
		helpSegments := e.parseMarkup(p.helpText)
		if len(helpSegments) > 0 {
			e.drawColoredTextSegmentsAlpha(screen, helpSegments, x, y-int(fontSize), p.alpha)
		} else {
			e.drawColoredTextWithFace(screen, p.helpText, x, y-int(fontSize), colorWithAlpha(colorAction, p.alpha), rowFace)
		}
		y += lineHeight
	}

	y += lineHeight

	stripLen := settingsTabStripLength(p.items)
	if stripLen == 0 {
		stripLen = settingsTabStripItemCount(p.labels, p.title, p.items)
	}

	menuRowOffset := func(index int) float64 {
		if index <= 0 {
			return 0
		}
		off := 0.0
		fs := int(e.getUIFontSize())
		start := 0
		if stripLen >= 2 {
			if index < stripLen {
				return 0
			}
			tabTextH := settingsTabLabelsMaxHeight(p.labels, stripLen, rowFace)
			if tabTextH <= 0 {
				_, tabTextH = text.Measure("Ag", rowFace, 0)
			}
			off += float64(settingsTabStripBlockHeight(tabTextH))
			start = stripLen
		}
		for i := start; i < index && i < len(p.labels); i++ {
			if renderer.IsPowerBarLine(p.labels[i]) {
				off += float64(powerBarMenuRowHeight(fs))
			} else if renderer.IsInventoryRowLine(p.labels[i]) {
				off += float64(inventoryMenuRowHeight(fs))
			} else {
				off += float64(lineHeight)
				off += float64(inventorySectionHeaderExtraGap(p.title, p.labels, i, lineHeight))
			}
		}
		return off
	}

	const animDuration = 150
	var highlightY float64
	var highlightWidth float64
	var highlightIndex int

	if p.highlightAnimating {
		now := e.menuAnimClockMilli
		if now == 0 {
			now = 0
		}
		elapsed := now - p.highlightAnimStartTime
		if elapsed >= animDuration {
			highlightIndex = p.highlightTarget
			highlightY = float64(y) + menuRowOffset(p.highlightTarget) + fontSize
			highlightWidth = p.highlightTargetWidth
		} else {
			progress := float64(elapsed) / float64(animDuration)
			easedProgress := easeInOut(progress)
			startY := float64(y) + menuRowOffset(p.highlightAnimStartIndex) + fontSize
			targetY := float64(y) + menuRowOffset(p.highlightTarget) + fontSize
			highlightY = startY + (targetY-startY)*easedProgress
			highlightWidth = p.highlightAnimStartWidth + (p.highlightTargetWidth-p.highlightAnimStartWidth)*easedProgress
			highlightIndex = p.highlightTarget
		}
	} else {
		highlightIndex = p.selected
		highlightY = float64(y) + menuRowOffset(p.selected) + fontSize
		if p.selected >= 0 && p.selected < len(p.labels) {
			highlightWidth = e.getMenuLabelWidth(p.labels[p.selected])
		}
	}
	tabHighlight := stripLen >= 2 && p.title == "Settings" &&
		((highlightIndex >= 0 && highlightIndex < stripLen) ||
			(p.highlightAnimating && p.highlightAnimStartIndex < stripLen && p.highlightTarget < stripLen))

	if (p.title == "Settings" && !tabHighlight) || p.title == "Bindings Menu" {
		highlightWidth = float64(p.panelW - paddingX*2)
	}

	if !tabHighlight && highlightIndex >= 0 && highlightIndex < len(p.items) && p.items[highlightIndex].IsSelectable() && highlightWidth > 0 {
		const padX = 8.0
		rx := float32(float64(x) - padX)
		ry := float32(highlightY)
		rw := float32(highlightWidth + padX*2)
		rh := float32(textHeight + 4)
		if p.maintOverlayStable && !p.highlightAnimating {
			rx = float32(math.Round(float64(rx)))
			ry = float32(math.Round(float64(ry)))
			rw = float32(math.Round(float64(rw)))
			rh = float32(math.Round(float64(rh)))
		}
		vector.DrawFilledRect(screen, rx, ry, rw, rh, highlightCol, false)
	}

	var valueColumnX, wattsColumnX int
	rightAlignPowerColumn := p.title == "Select room"
	tableMenu := strings.Contains(p.title, "Maintenance Terminal") ||
		p.title == "Select room" ||
		p.title == "Settings" ||
		p.title == "Bindings Menu" ||
		p.title == "Developer Menu" ||
		p.currentCellCharsMenu ||
		p.title == "Video" ||
		p.title == "Lift Routing"
	if tableMenu {
		var maxLabelW, maxValueW float64
		for _, label := range p.labels {
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
		valueColumnGap := 8
		if p.title == "Settings" || p.title == "Bindings Menu" {
			valueColumnGap = 32
		}
		valueColumnX = x + int(maxLabelW) + valueColumnGap
		if rightAlignPowerColumn {
			wattsColumnX = p.panelX + p.panelW - paddingX
		} else {
			wattsColumnX = valueColumnX + int(maxValueW) + valueColumnGap
		}
	}

	rowY := y
	if stripLen >= 2 && p.title == "Settings" {
		rowY += e.drawSettingsTabStrip(screen, p, rowY, stripLen, rowFace, highlightCol)
	}

	for i, item := range p.items {
		if stripLen >= 2 && p.title == "Settings" && i < stripLen {
			continue
		}
		if i >= len(p.labels) {
			break
		}
		label := p.labels[i]
		rowParamY := rowY

		if renderer.IsPowerBarLine(label) {
			rowH := e.drawPowerBarMenuRow(screen, x, rowParamY, p.panelW, paddingX, label, p.alpha)
			rowY += rowH
			continue
		}

		if renderer.IsInventoryRowLine(label) {
			rowH := e.drawInventoryMenuRow(screen, x, rowParamY, label, p.alpha)
			rowY += rowH
			continue
		}

		if (p.title == "Settings" || p.title == "Bindings Menu" || p.title == "Inventory") && !item.IsSelectable() && strings.HasPrefix(label, "TITLE{") {
			e.drawColoredTextSegmentsAlphaWithFace(screen, e.parseMarkup(label), x, rowParamY, p.alpha, e.getSansBoldTitleFontFace())
			rowY += lineHeight
			rowY += inventorySectionHeaderExtraGap(p.title, p.labels, i, lineHeight)
			continue
		}

		if valueColumnX > x {
			if before, after, ok := strings.Cut(label, "\t"); ok {
				if before != "" {
					if strings.Contains(before, "{") {
						e.drawColoredTextSegmentsAlphaWithFace(screen, e.parseMarkup(before), x, rowParamY, p.alpha, rowFace)
					} else {
						labelColor := colorText
						if !item.IsSelectable() {
							labelColor = colorSubtle
						}
						e.drawColoredTextWithFace(screen, before, x, rowParamY, colorWithAlpha(labelColor, p.alpha), rowFace)
					}
				}
				valuePart, wattsPart := after, ""
				if middle, w, hasWatts := strings.Cut(after, "\t"); hasWatts {
					valuePart, wattsPart = middle, w
				}
				if valuePart != "" {
					segments := e.parseMarkup(valuePart)
					if len(segments) > 0 {
						e.drawColoredTextSegmentsAlphaWithFace(screen, segments, valueColumnX, rowParamY, p.alpha, rowFace)
					} else {
						labelColor := colorText
						if !item.IsSelectable() {
							labelColor = colorSubtle
						}
						e.drawColoredTextWithFace(screen, valuePart, valueColumnX, rowParamY, colorWithAlpha(labelColor, p.alpha), rowFace)
					}
				}
				if wattsPart != "" && wattsColumnX > valueColumnX {
					wattsX := wattsColumnX
					if rightAlignPowerColumn {
						wattsX = wattsColumnX - int(e.getMarkupWidthWithFace(wattsPart, rowFace))
					}
					segments := e.parseMarkup(wattsPart)
					if len(segments) > 0 {
						e.drawColoredTextSegmentsAlphaWithFace(screen, segments, wattsX, rowParamY, p.alpha, rowFace)
					} else {
						labelColor := colorText
						if !item.IsSelectable() {
							labelColor = colorSubtle
						}
						e.drawColoredTextWithFace(screen, wattsPart, wattsX, rowParamY, colorWithAlpha(labelColor, p.alpha), rowFace)
					}
				}
				rowY += lineHeight
				continue
			}
		}

		segments := e.parseMarkup(label)
		if len(segments) > 0 {
			e.drawColoredTextSegmentsAlphaWithFace(screen, segments, x, rowParamY, p.alpha, rowFace)
		} else {
			labelColor := colorText
			if !item.IsSelectable() {
				labelColor = colorSubtle
			}
			e.drawColoredTextWithFace(screen, label, x, rowParamY, colorWithAlpha(labelColor, p.alpha), rowFace)
		}
		rowY += lineHeight
	}
}

func (e *EbitenRenderer) drawColoredTextSegmentsAlpha(screen *ebiten.Image, segments []textSegment, x, y int, alpha float64) {
	e.drawColoredTextSegmentsAlphaWithFace(screen, segments, x, y, alpha, e.getSansFontFace())
}

func (e *EbitenRenderer) drawColoredTextSegmentsAlphaWithFace(screen *ebiten.Image, segments []textSegment, x, y int, alpha float64, face *text.GoTextFace) {
	if alpha <= 0 {
		return
	}
	scaled := make([]textSegment, len(segments))
	for i, seg := range segments {
		scaled[i] = textSegment{text: seg.text, color: colorWithAlpha(seg.color, alpha)}
	}
	e.drawColoredTextSegmentsWithFace(screen, scaled, x, y, face)
}
