package renderer

// FeatureFlags exposes runtime feature toggles backed by console cvars.
type FeatureFlags interface {
	VisitedSystemEnabled() bool
}

// VisitedSystemEnabled reports whether visited-cell tracking and visited floor styling are active.
// Default is false when the active renderer does not implement FeatureFlags.
func VisitedSystemEnabled() bool {
	if Current == nil {
		return false
	}
	if ff, ok := Current.(FeatureFlags); ok {
		return ff.VisitedSystemEnabled()
	}
	return false
}
