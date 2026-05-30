package ebiten

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	gamemenu "darkstation/pkg/game/menu"
	"darkstation/pkg/game/state"
)

// RunConfirmDialog implements gamemenu.ConfirmDialogRenderer.
func (e *EbitenRenderer) RunConfirmDialog(g *state.Game, opts gamemenu.ConfirmOptions) bool {
	ch := make(chan bool, 1)

	e.confirmMutex.Lock()
	e.confirmActive = true
	e.confirmTitle = opts.Title
	e.confirmMessage = opts.Message
	e.confirmResultCh = ch
	e.confirmMutex.Unlock()

	defer func() {
		e.confirmMutex.Lock()
		e.confirmActive = false
		e.confirmTitle = ""
		e.confirmMessage = ""
		e.confirmResultCh = nil
		e.confirmMutex.Unlock()
	}()

	return <-ch
}

func (e *EbitenRenderer) isConfirmDialogActive() bool {
	e.confirmMutex.RLock()
	defer e.confirmMutex.RUnlock()
	return e.confirmActive
}

func (e *EbitenRenderer) handleConfirmDialogInput() {
	e.confirmMutex.Lock()
	defer e.confirmMutex.Unlock()

	if !e.confirmActive || e.confirmResultCh == nil {
		return
	}

	send := func(confirmed bool) {
		select {
		case e.confirmResultCh <- confirmed:
		default:
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyY) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) {
		send(true)
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyN) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		send(false)
		return
	}

	for _, id := range ebiten.AppendGamepadIDs(nil) {
		if inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton0) {
			send(true)
			return
		}
		if inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton1) {
			send(false)
			return
		}
	}
}

func (e *EbitenRenderer) drawConfirmDialog(screen *ebiten.Image) {
	e.confirmMutex.RLock()
	active := e.confirmActive
	title := e.confirmTitle
	message := e.confirmMessage
	e.confirmMutex.RUnlock()

	if !active {
		return
	}

	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	vector.DrawFilledRect(screen, 0, 0, float32(screenWidth), float32(screenHeight),
		color.RGBA{0, 0, 0, 140}, false)

	face := e.getSansFontFace()
	if face == nil {
		return
	}
	fontSize := e.getUIFontSize()
	lineHeight := int(fontSize) + 8
	padding := 20
	minWidth := 420
	maxWidth := screenWidth * 3 / 4

	titleWidth := int(e.getTextWidth(title))
	messageWidth := int(e.getTextWidth(message))
	help := "Y or Enter: Yes | N or Esc: No"
	helpWidth := int(e.getTextWidth(help))

	contentWidth := titleWidth
	for _, w := range []int{messageWidth, helpWidth} {
		if w > contentWidth {
			contentWidth = w
		}
	}
	panelW := contentWidth + padding*2
	if panelW < minWidth {
		panelW = minWidth
	}
	if panelW > maxWidth {
		panelW = maxWidth
	}

	panelH := padding*2 + lineHeight*3
	panelX := (screenWidth - panelW) / 2
	panelY := (screenHeight - panelH) / 2

	drawRoundedRectWithShadow(screen,
		float32(panelX), float32(panelY),
		float32(panelW), float32(panelH),
		10, 2,
		color.RGBA{30, 30, 50, 240},
		color.RGBA{100, 100, 150, 220},
		1.0)

	textX := panelX + padding
	y := panelY + padding

	drawLine := func(s string, c color.RGBA) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(textX), float64(y)+fontSize)
		op.ColorScale.ScaleWithColor(c)
		text.Draw(screen, s, face, op)
		y += lineHeight
	}

	drawLine(title, colorText)
	drawLine(message, colorSubtle)
	drawLine(help, colorSubtle)
}
