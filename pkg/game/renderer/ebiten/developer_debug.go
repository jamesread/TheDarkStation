package ebiten

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var mapAreaBorderColor = color.RGBA{255, 0, 0, 255}

// SetDrawFOVRays enables or disables FOV ray-cast debug lines on the map.
func (e *EbitenRenderer) SetDrawFOVRays(on bool) {
	e.devDebugMutex.Lock()
	e.fovRayDebugEnabled = on
	e.devDebugMutex.Unlock()
}

// DrawFOVRaysEnabled reports whether FOV ray debug lines are shown.
func (e *EbitenRenderer) DrawFOVRaysEnabled() bool {
	e.devDebugMutex.RLock()
	defer e.devDebugMutex.RUnlock()
	return e.fovRayDebugEnabled
}

// ToggleDrawFOVRays flips FOV ray debug lines and returns the new state.
func (e *EbitenRenderer) ToggleDrawFOVRays() bool {
	e.devDebugMutex.Lock()
	e.fovRayDebugEnabled = !e.fovRayDebugEnabled
	on := e.fovRayDebugEnabled
	e.devDebugMutex.Unlock()
	return on
}

// SetDrawMapAreaBorder enables or disables the red border around the map drawing area.
func (e *EbitenRenderer) SetDrawMapAreaBorder(on bool) {
	e.devDebugMutex.Lock()
	e.drawMapAreaBorder = on
	e.devDebugMutex.Unlock()
}

// DrawMapAreaBorderEnabled reports whether the map area border overlay is active.
func (e *EbitenRenderer) DrawMapAreaBorderEnabled() bool {
	e.devDebugMutex.RLock()
	defer e.devDebugMutex.RUnlock()
	return e.drawMapAreaBorder
}

// ToggleDrawMapAreaBorder flips the map area border overlay.
func (e *EbitenRenderer) ToggleDrawMapAreaBorder() bool {
	e.devDebugMutex.Lock()
	e.drawMapAreaBorder = !e.drawMapAreaBorder
	on := e.drawMapAreaBorder
	e.devDebugMutex.Unlock()
	return on
}

// SetShowFPSCounter sets the draw.fps cvar.
func (e *EbitenRenderer) SetShowFPSCounter(on bool) {
	setCvarBool("draw.fps", on)
}

// ShowFPSCounterEnabled reports whether the draw.fps cvar enables the FPS overlay.
func (e *EbitenRenderer) ShowFPSCounterEnabled() bool {
	return cvarEnabled("draw.fps")
}

// ToggleShowFPSCounter flips draw.fps and returns the new state.
func (e *EbitenRenderer) ToggleShowFPSCounter() bool {
	return toggleCvarBool("draw.fps")
}

// SetShowPlayerPosition sets the draw.player_pos cvar.
func (e *EbitenRenderer) SetShowPlayerPosition(on bool) {
	setCvarBool("draw.player_pos", on)
}

// ShowPlayerPositionEnabled reports whether draw.player_pos enables the player X/Y overlay.
func (e *EbitenRenderer) ShowPlayerPositionEnabled() bool {
	return cvarEnabled("draw.player_pos")
}

// ToggleShowPlayerPosition flips draw.player_pos and returns the new state.
func (e *EbitenRenderer) ToggleShowPlayerPosition() bool {
	return toggleCvarBool("draw.player_pos")
}

func (e *EbitenRenderer) drawMapAreaBorderOutline(screen *ebiten.Image, x, y, w, h int) {
	if w <= 0 || h <= 0 {
		return
	}
	const borderWidth = 2
	var path vector.Path
	xf, yf := float32(x), float32(y)
	wf, hf := float32(w), float32(h)
	path.MoveTo(xf, yf)
	path.LineTo(xf+wf, yf)
	path.LineTo(xf+wf, yf+hf)
	path.LineTo(xf, yf+hf)
	path.Close()
	strokeOpts := &vector.StrokeOptions{Width: borderWidth, MiterLimit: 10}
	drawOpts := &vector.DrawPathOptions{}
	drawOpts.ColorScale.ScaleWithColor(mapAreaBorderColor)
	vector.StrokePath(screen, &path, strokeOpts, drawOpts)
}
