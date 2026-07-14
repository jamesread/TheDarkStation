package ebiten

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"darkstation/pkg/game/renderer"
)

func inventoryMenuCellSize(fontSize float64) int {
	return int(fontSize) + 6
}

func inventoryMenuRowHeight(fontSize int) int {
	lineHeight := fontSize + 6
	cell := inventoryMenuCellSize(float64(fontSize))
	if cell > lineHeight {
		return cell + 2
	}
	return lineHeight
}

func inventoryDepictionColors(key string) (fg color.Color, bg color.Color, hasBg bool) {
	switch renderer.InventoryDepictionKey(key) {
	case renderer.InventoryDepictionKeycard:
		return colorKeycard, colorWallBg, true
	case renderer.InventoryDepictionKeyBattery:
		return colorBattery, colorWallBg, true
	case renderer.InventoryDepictionKeyMap:
		return colorItem, colorWallBg, true
	default:
		return colorItem, colorWallBg, true
	}
}

func (e *EbitenRenderer) drawInventoryMenuRow(screen *ebiten.Image, x, y int, line string, alpha float64) int {
	icon, key, label, ok := renderer.ParseInventoryRowLine(line)
	if !ok {
		return int(e.getUIFontSize()) + 6
	}
	fontSize := e.getUIFontSize()
	rowFace := e.getSansFontFace()
	monoFace := e.getMonoUIFontFace()
	cellSize := inventoryMenuCellSize(fontSize)

	fg, bg, hasBg := inventoryDepictionColors(key)
	fg = colorWithAlpha(fg, alpha)
	if hasBg {
		bg = colorWithAlpha(bg, alpha)
	}

	if hasBg {
		vector.DrawFilledRect(screen, float32(x), float32(y), float32(cellSize), float32(cellSize), bg, false)
	}
	iconW, iconH := text.Measure(icon, monoFace, 0)
	iconX := x + (cellSize-int(iconW))/2
	iconY := y + (cellSize-int(iconH))/2
	e.drawUILeftTextTop(screen, icon, iconX, iconY, fg, monoFace)

	_, textH := text.Measure("Ay", rowFace, 0)
	textTopY := y + (cellSize-int(textH))/2
	textX := x + cellSize + 10
	segments := e.parseMarkup(label)
	if len(segments) > 0 {
		e.drawInventoryMenuRowTextSegments(screen, segments, textX, textTopY, alpha, rowFace)
	} else {
		e.drawUILeftTextTop(screen, label, textX, textTopY, colorWithAlpha(colorText, alpha), rowFace)
	}
	return inventoryMenuRowHeight(int(fontSize))
}

// inventorySectionHeaderExtraGap adds space after inventory section headers so the
// first depiction row does not overlap the title line.
func inventorySectionHeaderExtraGap(menuTitle string, labels []string, index int, lineHeight int) int {
	if menuTitle != "Inventory" || index < 0 || index >= len(labels) {
		return 0
	}
	if !strings.HasPrefix(labels[index], "TITLE{") {
		return 0
	}
	if index+1 < len(labels) && renderer.IsInventoryRowLine(labels[index+1]) {
		return lineHeight
	}
	return 0
}

func (e *EbitenRenderer) drawInventoryMenuRowTextSegments(screen *ebiten.Image, segments []textSegment, x, topY int, alpha float64, face *text.GoTextFace) {
	if alpha <= 0 {
		return
	}
	currentX := float64(x)
	for _, seg := range segments {
		if seg.text == "" {
			continue
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(currentX, float64(topY))
		op.ColorScale.ScaleWithColor(colorWithAlpha(seg.color, alpha))
		text.Draw(screen, seg.text, face, op)
		w, _ := text.Measure(seg.text, face, 0)
		currentX += w
	}
}

func (e *EbitenRenderer) inventoryMenuLabelWidth(line string) float64 {
	_, _, label, ok := renderer.ParseInventoryRowLine(line)
	if !ok {
		return e.getMenuLabelWidth(line)
	}
	fontSize := e.getUIFontSize()
	cellSize := inventoryMenuCellSize(fontSize)
	return float64(cellSize+10) + e.getMenuLabelWidth(label)
}
