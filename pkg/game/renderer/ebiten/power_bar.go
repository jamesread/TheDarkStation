package ebiten

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"darkstation/pkg/game/renderer"
)

const (
	powerBarHeight       = 8
	powerBarYOffset      = powerBarHeight * 2
	powerBarLabelGap     = 4
	powerBarWattsGap     = 6
	powerBarCalloutExtra = powerBarHeight + powerBarLabelGap + powerBarYOffset
)

func (e *EbitenRenderer) drawPowerBar(screen *ebiten.Image, x, y, width int, supply, consumption, highlight int, alpha float64) int {
	if width < 12 {
		width = 12
	}
	h := powerBarHeight
	track := e.applyAlpha(renderer.PowerBarColorTrack, alpha)
	border := e.applyAlpha(renderer.PowerBarColorBorder, alpha)
	fill := e.applyAlpha(renderer.PowerBarUsageColor(supply, consumption), alpha)
	highlightFill := e.applyAlpha(renderer.PowerBarColorOrange, alpha)

	vector.FillRect(screen, float32(x), float32(y), float32(width), float32(h), track, false)
	fillW := int(float32(width) * float32(renderer.PowerBarUsageFraction(supply, consumption)))
	if fillW > 0 {
		vector.FillRect(screen, float32(x), float32(y), float32(fillW), float32(h), fill, false)
	}
	highlightW := int(float32(width) * float32(renderer.PowerBarHighlightFraction(supply, highlight)))
	if highlightW > fillW {
		highlightW = fillW
	}
	if highlightW > 0 {
		highlightX := x + fillW - highlightW
		vector.FillRect(screen, float32(highlightX), float32(y), float32(highlightW), float32(h), highlightFill, false)
	}
	vector.StrokeRect(screen, float32(x), float32(y), float32(width), float32(h), 1, border, false)
	return h
}

func (e *EbitenRenderer) drawPowerBarRow(screen *ebiten.Image, x, y, maxWidth int, label string, supply, consumption, highlight int, alpha float64, subtleLabel bool) int {
	labelColor := colorText
	if subtleLabel {
		labelColor = colorSubtle
	}
	labelW := 0
	if label != "" {
		e.drawColoredText(screen, label, x, y, labelColor)
		labelW = int(e.getTextWidth(label))
	}
	barX := x
	if labelW > 0 {
		barX = x + labelW + powerBarLabelGap
	}
	watts := renderer.FormatPowerBarWattsSuffix(supply, consumption)
	wattsW := int(e.getTextWidth(watts))
	barW := maxWidth - (barX - x) - wattsW - powerBarWattsGap
	if barW < 40 {
		barW = 40
	}
	barY := y + int(e.getUIFontSize()) - powerBarHeight + 2 + powerBarYOffset
	if barY < y {
		barY = y + 2
	}
	e.drawPowerBar(screen, barX, barY, barW, supply, consumption, highlight, alpha)
	wattsX := barX + barW + powerBarWattsGap
	e.drawColoredText(screen, watts, wattsX, y, colorSubtle)
	return powerBarRowHeight(int(e.getUIFontSize()))
}

func (e *EbitenRenderer) powerBarLineExtraHeight() int {
	return powerBarCalloutExtra
}

func powerBarRowHeight(fontSize int) int {
	return fontSize + 6 + powerBarCalloutExtra
}

func powerBarMenuRowHeight(fontSize int) int {
	return powerBarRowHeight(fontSize)
}

func (e *EbitenRenderer) drawPowerBarMenuRow(screen *ebiten.Image, x, y, panelW, paddingX int, line string, alpha float64) int {
	label, supply, consumption, highlight, ok := renderer.ParsePowerBarLine(line)
	if !ok {
		return int(e.getUIFontSize()) + 6
	}
	maxWidth := panelW - paddingX*2
	return e.drawPowerBarRow(screen, x, y, maxWidth, label, supply, consumption, highlight, alpha, label != "")
}
