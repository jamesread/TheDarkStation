// Package ebiten provides build label overlay rendering.
package ebiten

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/leonelquinteros/gotext"

	"darkstation/pkg/game/renderer"
)

// drawBuildLabel renders the friendly build stamp in the bottom-right on every screen.
func (e *EbitenRenderer) drawBuildLabel(screen *ebiten.Image, screenWidth, screenHeight int) {
	if screen == nil || e.sansFontSource == nil || renderer.BuildLabel == "" {
		return
	}
	label := fmt.Sprintf(gotext.Get("VERSION"), renderer.BuildLabel)
	face := e.getSansFontFace()
	_, textHeight := text.Measure(label, face, 0)
	margin := 16
	labelWidth := e.getTextWidth(label)
	x := screenWidth - int(labelWidth) - margin
	y := screenHeight - margin - int(textHeight*2)
	e.drawColoredText(screen, label, x, y, colorSubtle)
}
