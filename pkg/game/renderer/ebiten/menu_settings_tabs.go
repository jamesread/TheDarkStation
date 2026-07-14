package ebiten

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	gamemenu "darkstation/pkg/game/menu"
)

const settingsTabGap = 24

func settingsTabLabelsMaxHeight(labels []string, stripLen int, face *text.GoTextFace) float64 {
	var maxH float64
	for i := 0; i < stripLen && i < len(labels); i++ {
		_, h := text.Measure(labels[i], face, 0)
		if h > maxH {
			maxH = h
		}
	}
	return maxH
}

// settingsTabStripBlockHeight is the vertical space for tab labels plus the rule beneath them.
func settingsTabStripBlockHeight(textHeight float64) int {
	const ruleGap = 6
	const ruleBelow = 6
	return int(math.Ceil(textHeight)) + ruleGap + 2 + ruleBelow
}

func settingsTabStripLength(items []gamemenu.MenuItem) int {
	return gamemenu.SettingsTabStripLength(items)
}

func settingsTabStripRows(labels []string, title string, items []gamemenu.MenuItem) int {
	if settingsTabStripLength(items) >= 2 {
		return 1
	}
	if title == "Settings" && len(labels) >= 2 && labels[0] == "Bindings" && labels[1] == "Video" {
		return 1
	}
	return 0
}

func settingsTabStripItemCount(labels []string, title string, items []gamemenu.MenuItem) int {
	if n := settingsTabStripLength(items); n >= 2 {
		return n
	}
	if settingsTabStripRows(labels, title, items) == 1 {
		return 2
	}
	return 0
}

func (e *EbitenRenderer) settingsTabHighlightRect(p menuPanelContentParams, tabIndex int) (x, width float64) {
	stripLen := settingsTabStripLength(p.items)
	if stripLen < 2 || tabIndex < 0 || tabIndex >= stripLen {
		return 0, 0
	}
	paddingX := 24
	x = float64(p.panelX + paddingX + int(math.Round(p.contentSlideX)))
	for i := 0; i < tabIndex; i++ {
		if i < len(p.labels) {
			x += e.getMenuLabelWidth(p.labels[i]) + settingsTabGap
		}
	}
	if tabIndex < len(p.labels) {
		width = e.getMenuLabelWidth(p.labels[tabIndex])
	}
	return x, width
}

func (e *EbitenRenderer) drawSettingsTabStrip(
	screen *ebiten.Image,
	p menuPanelContentParams,
	rowY int,
	stripLen int,
	rowFace *text.GoTextFace,
	highlightCol color.Color,
) int {
	paddingX := 24
	x := p.panelX + paddingX + int(math.Round(p.contentSlideX))
	tabX := x

	labelHeight := settingsTabLabelsMaxHeight(p.labels, stripLen, rowFace)
	if labelHeight <= 0 {
		_, labelHeight = text.Measure("Ag", rowFace, 0)
	}

	var highlightX, highlightW float64
	var highlightY float64
	var highlightH float64
	useTabHighlight := false

	if p.highlightAnimating {
		startOnStrip := p.highlightAnimStartIndex >= 0 && p.highlightAnimStartIndex < stripLen
		targetOnStrip := p.highlightTarget >= 0 && p.highlightTarget < stripLen
		if startOnStrip && targetOnStrip {
			const animDuration = 150
			now := e.menuAnimClockMilli
			elapsed := now - p.highlightAnimStartTime
			startX, startW := e.settingsTabHighlightRect(p, p.highlightAnimStartIndex)
			targetX, targetW := e.settingsTabHighlightRect(p, p.highlightTarget)
			progress := 1.0
			if elapsed < animDuration {
				progress = easeInOut(float64(elapsed) / float64(animDuration))
			}
			highlightX = startX + (targetX-startX)*progress
			highlightW = startW + (targetW-startW)*progress
			highlightY = float64(rowY)
			highlightH = labelHeight + 4
			useTabHighlight = true
		}
	} else if p.selected >= 0 && p.selected < stripLen {
		highlightX, highlightW = e.settingsTabHighlightRect(p, p.selected)
		highlightY = float64(rowY)
		highlightH = labelHeight + 4
		useTabHighlight = highlightW > 0
	}

	if useTabHighlight {
		const padX = 8.0
		vector.DrawFilledRect(
			screen,
			float32(highlightX-padX),
			float32(highlightY),
			float32(highlightW+padX*2),
			float32(highlightH),
			highlightCol,
			false,
		)
	}

	for i := 0; i < stripLen; i++ {
		if i >= len(p.labels) {
			break
		}
		label := p.labels[i]
		labelColor := colorText
		if p.selected != i {
			labelColor = colorSubtle
			if tab, ok := p.items[i].(*gamemenu.SettingsTabItem); ok && tab.IsActiveContentTab() {
				labelColor = colorText
			}
		}
		e.drawUILeftTextTop(screen, label, tabX, rowY, colorWithAlpha(labelColor, p.alpha), rowFace)
		tabX += int(e.getMenuLabelWidth(label)) + settingsTabGap
	}

	const ruleGap = 6.0
	const titleBlockPad = 4.0 // matches credits spacing below title text
	ruleY := float64(rowY) + labelHeight + titleBlockPad + ruleGap
	ruleX1 := float64(p.panelX + paddingX + int(math.Round(p.contentSlideX)))
	ruleX2 := float64(p.panelX + p.panelW - paddingX + int(math.Round(p.contentSlideX)))
	ruleColor := colorWithAlpha(colorAction, p.alpha*0.85)
	vector.StrokeLine(screen, float32(ruleX1), float32(ruleY), float32(ruleX2), float32(ruleY), 1.5, ruleColor, false)

	return settingsTabStripBlockHeight(labelHeight)
}
