package deck

import "testing"

func TestObservationLedPuzzleCuesActive_earlyDeck_off(t *testing.T) {
	if ObservationLedPuzzleCuesActive(2, false) {
		t.Fatal("tier off for level < 3")
	}
	if ObservationLedPuzzleCuesActive(3, true) {
		t.Fatal("tier off when minimal final-style layout flagged")
	}
}

func TestObservationLedPuzzleCuesActive_deckThree_on(t *testing.T) {
	if !ObservationLedPuzzleCuesActive(3, false) {
		t.Fatal("tier on from level 3 with full systems")
	}
}

func TestObservationSeqPlaqueMsgID_knownSequences(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want string
	}{
		{"1-2-3-4", "ENV_PLAQUE_OBS_SEQ_1234"},
		{"2-4-6-8", "ENV_PLAQUE_OBS_SEQ_2468"},
	} {
		got, ok := ObservationSeqPlaqueMsgID(tt.in)
		if !ok || got != tt.want {
			t.Fatalf("solution %q: got (%q,%v), want (%q,true)", tt.in, got, ok, tt.want)
		}
	}
	if _, ok := ObservationSeqPlaqueMsgID("north-south"); ok {
		t.Fatal("pattern solutions should not map to observation plaques")
	}
}
