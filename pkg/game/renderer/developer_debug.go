package renderer

// DeveloperDebugRenderer is implemented by renderers that support developer debug overlays.
type DeveloperDebugRenderer interface {
	SetDrawMapAreaBorder(on bool)
	DrawMapAreaBorderEnabled() bool
	ToggleDrawMapAreaBorder() bool
	SetDrawFOVRays(on bool)
	DrawFOVRaysEnabled() bool
	ToggleDrawFOVRays() bool
	SetShowFPSCounter(on bool)
	ShowFPSCounterEnabled() bool
	ToggleShowFPSCounter() bool
}

// SetDrawMapAreaBorder enables or disables the red map viewport border overlay.
func SetDrawMapAreaBorder(on bool) {
	if dr, ok := Current.(DeveloperDebugRenderer); ok {
		dr.SetDrawMapAreaBorder(on)
	}
}

// DrawMapAreaBorderEnabled reports whether the map viewport border is shown.
func DrawMapAreaBorderEnabled() bool {
	if dr, ok := Current.(DeveloperDebugRenderer); ok {
		return dr.DrawMapAreaBorderEnabled()
	}
	return false
}

// ToggleDrawMapAreaBorder flips the map viewport border and returns the new state.
func ToggleDrawMapAreaBorder() bool {
	if dr, ok := Current.(DeveloperDebugRenderer); ok {
		return dr.ToggleDrawMapAreaBorder()
	}
	return false
}

// SetDrawFOVRays enables or disables FOV ray-cast debug lines on the map.
func SetDrawFOVRays(on bool) {
	if dr, ok := Current.(DeveloperDebugRenderer); ok {
		dr.SetDrawFOVRays(on)
	}
}

// DrawFOVRaysEnabled reports whether FOV ray debug lines are shown.
func DrawFOVRaysEnabled() bool {
	if dr, ok := Current.(DeveloperDebugRenderer); ok {
		return dr.DrawFOVRaysEnabled()
	}
	return false
}

// ToggleDrawFOVRays flips FOV ray debug lines and returns the new state.
func ToggleDrawFOVRays() bool {
	if dr, ok := Current.(DeveloperDebugRenderer); ok {
		return dr.ToggleDrawFOVRays()
	}
	return false
}

// SetShowFPSCounter enables or disables the FPS overlay via draw.fps cvar.
func SetShowFPSCounter(on bool) {
	if dr, ok := Current.(DeveloperDebugRenderer); ok {
		dr.SetShowFPSCounter(on)
	}
}

// ShowFPSCounterEnabled reports whether draw.fps enables the FPS overlay.
func ShowFPSCounterEnabled() bool {
	if dr, ok := Current.(DeveloperDebugRenderer); ok {
		return dr.ShowFPSCounterEnabled()
	}
	return true // draw.fps default is 1
}

// ToggleShowFPSCounter flips draw.fps and returns the new state.
func ToggleShowFPSCounter() bool {
	if dr, ok := Current.(DeveloperDebugRenderer); ok {
		return dr.ToggleShowFPSCounter()
	}
	return true
}
