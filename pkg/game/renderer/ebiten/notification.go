// Package ebiten provides transient on-screen notifications (e.g. input device changes).
package ebiten

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	engineinput "darkstation/pkg/engine/input"
)

const (
	notificationLifetimeMs = 3000
	notificationMargin     = 16
)

// showInputDeviceNotification displays a short message when the primary input device changes.
func (e *EbitenRenderer) showInputDeviceNotification() {
	name := engineinput.PrimaryDeviceSwitchMessage()
	e.showTransientNotification(fmt.Sprintf("Switched to %s", name))
}

func (e *EbitenRenderer) showTransientNotification(msg string) {
	e.notificationMutex.Lock()
	defer e.notificationMutex.Unlock()
	e.notificationMessage = msg
	e.notificationAt = time.Now().UnixMilli()
}

func (e *EbitenRenderer) drawTransientNotification(screen *ebiten.Image, screenWidth, _screenHeight int) {
	e.notificationMutex.RLock()
	msg := e.notificationMessage
	msgAt := e.notificationAt
	e.notificationMutex.RUnlock()

	if msg == "" {
		return
	}

	now := time.Now().UnixMilli()
	age := now - msgAt
	if age >= notificationLifetimeMs {
		return
	}

	fadeStart := int64(notificationLifetimeMs * 7 / 10)
	alpha := 1.0
	if age > fadeStart {
		fadeProgress := float64(age-fadeStart) / float64(notificationLifetimeMs-fadeStart)
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

	padding := 10
	panelW := int(textWidth) + padding*2
	if panelW < 120 {
		panelW = 120
	}
	panelH := int(textHeight) + padding*2
	if panelH < int(face.Size)+padding {
		panelH = int(face.Size) + padding*2
	}

	panelX := (screenWidth - panelW) / 2
	if panelX < notificationMargin {
		panelX = notificationMargin
	}
	panelY := notificationMargin

	bgColor := e.applyAlpha(colorPanelBackground, alpha)
	borderColor := e.applyAlpha(color.RGBA{80, 80, 100, 255}, alpha)
	vector.DrawFilledRect(screen, float32(panelX-1), float32(panelY-1), float32(panelW+2), float32(panelH+2), borderColor, false)
	vector.DrawFilledRect(screen, float32(panelX), float32(panelY), float32(panelW), float32(panelH), bgColor, false)

	msgY := panelY + padding
	e.drawColoredTextSegments(screen, fadedSegments, panelX+padding, msgY)
}
