package state

// ExitLiftState describes lift readiness for rendering and movement gating.
type ExitLiftState int

const (
	// ExitLiftLockedUnpowered — exit room has no grid power; red locked icon.
	ExitLiftLockedUnpowered ExitLiftState = iota
	// ExitLiftLockedIncomplete — exit room has grid power but hazards remain; yellow locked icon.
	ExitLiftLockedIncomplete
	// ExitLiftReady — lift is usable; green pulsing icon and background.
	ExitLiftReady
)
