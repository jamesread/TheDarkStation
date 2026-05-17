package deck

// MultiHopLinkageActive enables cross-room relay + puzzle gating (Story 5.3).
// Stricter than Story 5.2's observation stamp tier (level >= 3).
func MultiHopLinkageActive(level int, minimalSystems bool) bool {
	if minimalSystems || level < 5 {
		return false
	}
	return true
}

// MultiHopLinkageToken is the canonical relay string embedded in plaques and furniture.
const MultiHopLinkageToken = "LINK-MHOP-A"

// MultiHopKeyedSequenceSolution is the PuzzleSequence solution that wears linkage metadata (Story 5.3).
// Keep aligned with pkg/game/levelgen/puzzles.go sequence entries — second numeric puzzle when ≥2 terminals.
const MultiHopKeyedSequenceSolution = "2-4-6-8"
