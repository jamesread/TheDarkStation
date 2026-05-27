package ebiten

// VisitedSystemEnabled implements renderer.FeatureFlags (cvar gameplay.visited).
func (e *EbitenRenderer) VisitedSystemEnabled() bool {
	return cvarEnabled("gameplay.visited")
}

// EnvPlaquesEnabled reports whether corridor environmental plaques are drawn (cvar draw.env_plaques).
func (e *EbitenRenderer) EnvPlaquesEnabled() bool {
	return cvarEnabled("draw.env_plaques")
}
