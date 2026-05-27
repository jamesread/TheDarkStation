package ebiten

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type levelGenLoading struct {
	active   bool
	level    int
	step     int
	total    int
	label    string
	progress float64
}

// BeginLevelGen implements renderer.LevelGenReporter.
func (e *EbitenRenderer) BeginLevelGen(level, totalSteps int) {
	if totalSteps < 1 {
		totalSteps = 1
	}
	e.loadingMutex.Lock()
	e.levelGen = levelGenLoading{
		active:   true,
		level:    level,
		total:    totalSteps,
		label:    "Preparing deck",
		progress: 0,
	}
	e.loadingMutex.Unlock()
}

// ReportLevelGenProgress implements renderer.LevelGenReporter.
func (e *EbitenRenderer) ReportLevelGenProgress(step, totalSteps int, label string) {
	if totalSteps < 1 {
		totalSteps = 1
	}
	if step < 0 {
		step = 0
	}
	if step > totalSteps {
		step = totalSteps
	}
	progress := float64(step) / float64(totalSteps)

	e.loadingMutex.Lock()
	e.levelGen.active = true
	e.levelGen.step = step
	e.levelGen.total = totalSteps
	e.levelGen.label = label
	e.levelGen.progress = progress
	e.loadingMutex.Unlock()
}

// ClearLevelGenProgress implements renderer.LevelGenReporter.
func (e *EbitenRenderer) ClearLevelGenProgress() {
	e.loadingMutex.Lock()
	e.levelGen = levelGenLoading{}
	e.loadingMutex.Unlock()
}

func (e *EbitenRenderer) levelGenSnapshot() levelGenLoading {
	e.loadingMutex.RLock()
	defer e.loadingMutex.RUnlock()
	return e.levelGen
}

func (e *EbitenRenderer) levelGenLoadingActive() bool {
	e.loadingMutex.RLock()
	defer e.loadingMutex.RUnlock()
	return e.levelGen.active
}

var _ interface {
	BeginLevelGen(level, totalSteps int)
	ReportLevelGenProgress(step, totalSteps int, label string)
	ClearLevelGenProgress()
} = (*EbitenRenderer)(nil)

func (e *EbitenRenderer) drawLevelGenLoading(screen *ebiten.Image, load levelGenLoading) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	panelW := int(float32(screenWidth) * 0.55)
	if panelW < 300 {
		panelW = 300
	}
	if panelW > screenWidth-40 {
		panelW = screenWidth - 40
	}
	panelH := 148
	panelX := (screenWidth - panelW) / 2
	panelY := (screenHeight - panelH) / 2

	bg := color.RGBA{10, 6, 16, 230}
	border := color.RGBA{80, 70, 120, 255}
	vector.FillRect(screen, float32(panelX), float32(panelY), float32(panelW), float32(panelH), bg, false)
	vector.StrokeRect(screen, float32(panelX), float32(panelY), float32(panelW), float32(panelH), 1.5, border, false)

	paddingX := 24
	paddingY := 22
	x := panelX + paddingX
	y := panelY + paddingY

	titleFace := e.getSansBoldTitleFontFace()
	uiFace := e.getSansFontFace()

	title := fmt.Sprintf("Generating deck %d", load.level)
	_, titleH := text.Measure(title, titleFace, 0)
	e.drawColoredTextWithFace(screen, title, x, y, colorAction, titleFace)
	y += int(titleH) + 10

	status := load.label
	if status == "" {
		status = "Working..."
	}
	e.drawColoredTextWithFace(screen, status, x, y, colorText, uiFace)

	barW := panelW - paddingX*2
	barH := 12
	barY := panelY + panelH - paddingY - barH

	barBG := color.RGBA{30, 30, 40, 220}
	barFill := color.RGBA{0, 220, 120, 255}
	barBorder := color.RGBA{180, 180, 200, 255}

	vector.FillRect(screen, float32(x), float32(barY), float32(barW), float32(barH), barBG, false)
	fillW := int(float32(barW) * float32(load.progress))
	if fillW > 0 {
		vector.FillRect(screen, float32(x), float32(barY), float32(fillW), float32(barH), barFill, false)
	}
	vector.StrokeRect(screen, float32(x), float32(barY), float32(barW), float32(barH), 1, barBorder, false)

	stepText := fmt.Sprintf("%d / %d", load.step, load.total)
	if load.total <= 0 {
		stepText = fmt.Sprintf("%d%%", int(load.progress*100))
	}
	stepW, stepH := text.Measure(stepText, uiFace, 0)
	stepX := x + barW - int(stepW)
	stepY := barY - int(stepH) - 6
	e.drawColoredTextWithFace(screen, stepText, stepX, stepY, colorSubtle, uiFace)
}
