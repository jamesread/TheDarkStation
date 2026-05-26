package ebiten

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	gamemenu "darkstation/pkg/game/menu"
	"darkstation/pkg/game/state"
)

type textInputResult struct {
	value string
	ok    bool
}

// RunTextInputDialog implements gamemenu.TextInputDialogRenderer.
func (e *EbitenRenderer) RunTextInputDialog(g *state.Game, opts gamemenu.TextInputOptions) (string, bool) {
	ch := make(chan textInputResult, 1)

	e.textInputMutex.Lock()
	e.textInputActive = true
	e.textInputHex = opts.Hex
	e.textInputTitle = opts.Title
	e.textInputPrompt = opts.Prompt
	e.textInputText = opts.Initial
	if opts.Hex {
		e.textInputText = strings.ToUpper(strings.TrimSpace(opts.Initial))
	}
	e.textInputResultCh = ch
	e.textInputMutex.Unlock()

	defer func() {
		e.textInputMutex.Lock()
		e.textInputActive = false
		e.textInputHex = false
		e.textInputResultCh = nil
		e.textInputText = ""
		e.textInputMutex.Unlock()
	}()

	res := <-ch
	return res.value, res.ok
}

func (e *EbitenRenderer) isTextInputDialogActive() bool {
	e.textInputMutex.RLock()
	defer e.textInputMutex.RUnlock()
	return e.textInputActive
}

func (e *EbitenRenderer) handleTextInputDialogInput() {
	e.textInputMutex.Lock()
	defer e.textInputMutex.Unlock()

	if !e.textInputActive || e.textInputResultCh == nil {
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		select {
		case e.textInputResultCh <- textInputResult{ok: false}:
		default:
		}
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if len(e.textInputText) > 0 {
			e.textInputText = e.textInputText[:len(e.textInputText)-1]
		}
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) {
		select {
		case e.textInputResultCh <- textInputResult{value: e.textInputText, ok: true}:
		default:
		}
		return
	}

	appendChar := func(c byte) {
		maxLen := 20
		if e.textInputHex {
			maxLen = 18 // optional "0x" + 16 hex digits
		}
		if len(e.textInputText) >= maxLen {
			return
		}
		e.textInputText += string(c)
	}

	for k := ebiten.Key0; k <= ebiten.Key9; k++ {
		if inpututil.IsKeyJustPressed(k) {
			appendChar(byte('0' + (k - ebiten.Key0)))
			return
		}
	}
	for k := ebiten.KeyNumpad0; k <= ebiten.KeyNumpad9; k++ {
		if inpututil.IsKeyJustPressed(k) {
			appendChar(byte('0' + (k - ebiten.KeyNumpad0)))
			return
		}
	}

	if e.textInputHex {
		if inpututil.IsKeyJustPressed(ebiten.KeyX) && e.textInputText == "0" {
			appendChar('X')
			return
		}
		for k := ebiten.KeyA; k <= ebiten.KeyF; k++ {
			if inpututil.IsKeyJustPressed(k) {
				appendChar(byte('A' + (k - ebiten.KeyA)))
				return
			}
		}
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) {
		if len(e.textInputText) == 0 {
			e.textInputText = "-"
		}
	}
}

func (e *EbitenRenderer) drawTextInputDialog(screen *ebiten.Image) {
	e.textInputMutex.RLock()
	active := e.textInputActive
	hexMode := e.textInputHex
	title := e.textInputTitle
	prompt := e.textInputPrompt
	inputText := e.textInputText
	e.textInputMutex.RUnlock()

	if hexMode {
		inputText = strings.ToUpper(inputText)
	}

	if !active {
		return
	}

	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Dim the screen behind the dialog.
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
	promptWidth := int(e.getTextWidth(prompt))
	fieldLabel := "Seed: "
	fieldPreview := fieldLabel + inputText + "_"
	fieldWidth := int(e.getTextWidth(fieldPreview))
	help := "Hex digits, Enter: load | Esc: cancel"
	if hexMode {
		help = "Hex seed (0-9, A-F), Enter: load | Esc: cancel"
	}
	helpWidth := int(e.getTextWidth(help))

	contentWidth := titleWidth
	for _, w := range []int{promptWidth, fieldWidth, helpWidth} {
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

	lines := 4
	if strings.TrimSpace(prompt) == "" {
		lines = 3
	}
	panelH := padding*2 + lineHeight*lines
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
	if strings.TrimSpace(prompt) != "" {
		drawLine(prompt, colorSubtle)
	}
	drawLine(fieldPreview, colorText)
	drawLine(help, colorSubtle)
}
