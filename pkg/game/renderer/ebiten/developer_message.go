// Package ebiten provides developer-only on-screen messages (e.g. map dump confirmation).
package ebiten

import (
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	developerMessageLifetimeMs = 10000
	developerMessageMargin     = 12
)

// ShowDeveloperMessage displays a short message at the bottom-left of the window.
// The message fades out after developerMessageLifetimeMs.
func (e *EbitenRenderer) ShowDeveloperMessage(msg string) {
	e.developerMessageMutex.Lock()
	defer e.developerMessageMutex.Unlock()
	e.developerMessage = msg
	e.developerMessageAt = time.Now().UnixMilli()
}

// drawDeveloperMessage renders the active developer message at the bottom-left.
func (e *EbitenRenderer) drawDeveloperMessage(screen *ebiten.Image, _screenWidth, screenHeight int) {
	e.developerMessageMutex.RLock()
	msg := e.developerMessage
	msgAt := e.developerMessageAt
	e.developerMessageMutex.RUnlock()

	if msg == "" {
		return
	}

	now := time.Now().UnixMilli()
	age := now - msgAt
	if age >= developerMessageLifetimeMs {
		return
	}

	fadeStart := int64(developerMessageLifetimeMs * 7 / 10)
	alpha := 1.0
	if age > fadeStart {
		fadeProgress := float64(age-fadeStart) / float64(developerMessageLifetimeMs-fadeStart)
		alpha = 1.0 - fadeProgress
		if alpha < 0 {
			alpha = 0
		}
	}

	segments := e.parseMarkup(msg)
	fadedSegments := make([]textSegment, len(segments))
	for i, seg := range segments {
		fadedSegments[i] = textSegment{
			text:  seg.text,
			color: e.applyAlpha(seg.color, alpha),
		}
	}

	face := e.getSansFontFace()
	var plain strings.Builder
	for _, seg := range segments {
		plain.WriteString(seg.text)
	}
	_, textHeight := text.Measure(plain.String(), face, 0)
	textWidth := e.getMarkupWidth(msg)

	padding := 8
	panelW := int(textWidth) + padding*2
	if panelW < 80 {
		panelW = 80
	}
	panelH := int(textHeight) + padding*2
	if panelH < int(face.Size)+padding {
		panelH = int(face.Size) + padding*2
	}

	x := developerMessageMargin
	bgY := float32(screenHeight - developerMessageMargin - panelH)
	if bgY < 0 {
		bgY = 0
	}

	bgColor := e.applyAlpha(colorPanelBackground, alpha)
	borderColor := e.applyAlpha(color.RGBA{80, 80, 100, 255}, alpha)
	vector.DrawFilledRect(screen, float32(x-1), bgY-1, float32(panelW+2), float32(panelH+2), borderColor, false)
	vector.DrawFilledRect(screen, float32(x), bgY, float32(panelW), float32(panelH), bgColor, false)

	msgY := screenHeight - developerMessageMargin - int(textHeight*2)
	e.drawColoredTextSegments(screen, fadedSegments, x+padding, msgY)
}
