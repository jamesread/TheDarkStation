package renderer

// LevelGenReporter is implemented by renderers that show deck generation progress.
type LevelGenReporter interface {
	BeginLevelGen(level, totalSteps int)
	ReportLevelGenProgress(step, totalSteps int, label string)
	ClearLevelGenProgress()
}

// BeginLevelGen starts the level-generation loading overlay.
func BeginLevelGen(level, totalSteps int) {
	if r, ok := Current.(LevelGenReporter); ok {
		r.BeginLevelGen(level, totalSteps)
	}
}

// ReportLevelGenProgress updates the loading overlay (step is 1-based).
func ReportLevelGenProgress(step, totalSteps int, label string) {
	if r, ok := Current.(LevelGenReporter); ok {
		r.ReportLevelGenProgress(step, totalSteps, label)
	}
}

// ClearLevelGenProgress hides the loading overlay.
func ClearLevelGenProgress() {
	if r, ok := Current.(LevelGenReporter); ok {
		r.ClearLevelGenProgress()
	}
}
