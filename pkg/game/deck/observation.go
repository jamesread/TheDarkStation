package deck

// ObservationLedPuzzleCuesActive returns true when corridor signage may carry
// sequence fingerprints matching security puzzles (Story 5.2 / FR26 tiering).
// Early decks (level < 3) and the minimal final-deck layout stay unchanged.
func ObservationLedPuzzleCuesActive(level int, minimalSystems bool) bool {
	if minimalSystems || level < 3 {
		return false
	}
	return true
}

// ObservationSeqPlaqueMsgID maps known numeric sequence puzzle solutions to
// diegetic plaque gettext msgids. Keep in sync with levelgen.PlacePuzzles
// puzzleSolutions order for sequence-type entries.
func ObservationSeqPlaqueMsgID(solution string) (msgid string, ok bool) {
	switch solution {
	case "1-2-3-4":
		return "ENV_PLAQUE_OBS_SEQ_1234", true
	case "2-4-6-8":
		return "ENV_PLAQUE_OBS_SEQ_2468", true
	default:
		return "", false
	}
}
