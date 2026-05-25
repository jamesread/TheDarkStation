package ebiten

// VisitedSystemEnabled implements renderer.FeatureFlags (cvar gameplay.visited).
func (e *EbitenRenderer) VisitedSystemEnabled() bool {
	return cvarEnabled("gameplay.visited")
}
