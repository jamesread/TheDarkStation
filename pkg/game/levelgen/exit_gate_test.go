package levelgen

import (
	"testing"

	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/levelrand"
)

func TestPickExitGateKind_deck1AndFinalNeverSlime(t *testing.T) {
	for seed := int64(0); seed < 50; seed++ {
		levelrand.Seed(seed)
		if got := PickExitGateKind(1); got != ExitGateNone {
			t.Fatalf("deck 1 seed %d: got %q, want none", seed, got)
		}
		levelrand.Seed(seed + 999)
		if got := PickExitGateKind(deck.TotalDecks); got != ExitGateNone {
			t.Fatalf("final deck seed %d: got %q, want none", seed, got)
		}
	}
}

func TestPickExitGateKind_midDecksUsePool(t *testing.T) {
	sawNone := false
	sawSlime := false
	for seed := int64(0); seed < 200; seed++ {
		levelrand.Seed(seed)
		switch PickExitGateKind(5) {
		case ExitGateNone:
			sawNone = true
		case ExitGateSlime:
			sawSlime = true
		default:
			t.Fatalf("seed %d: unexpected exit gate", seed)
		}
	}
	if !sawNone || !sawSlime {
		t.Fatalf("expected both none and slime across seeds, got none=%v slime=%v", sawNone, sawSlime)
	}
}

func seedForExitGate(level int, want ExitGateKind) int64 {
	for seed := int64(0); seed < 500; seed++ {
		levelrand.Seed(seed)
		if PickExitGateKind(level) == want {
			return seed
		}
	}
	return -1
}
