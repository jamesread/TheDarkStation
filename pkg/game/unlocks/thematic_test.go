package unlocks

import (
	"testing"

	"darkstation/pkg/game/deck"
)

func TestIsDeckAlwaysReachable(t *testing.T) {
	if !IsDeckAlwaysReachable(0) || !IsDeckAlwaysReachable(1) {
		t.Fatal("decks 1–2 should be reachable at run start")
	}
	if IsDeckAlwaysReachable(2) {
		t.Fatal("deck 3 should not be always reachable")
	}
}

func TestBuildUnlockPlan_ReactorChain(t *testing.T) {
	themes := deck.AssignThemes(99)
	plan := BuildUnlockPlan(99, themes)
	if plan == nil {
		t.Fatal("nil plan")
	}
	var authCount, reactorRouting int
	for _, req := range plan.Requirements {
		if req.TargetDeckID == 4 && req.Kind == KindSecurityKeycard {
			authCount++
		}
		if req.TargetDeckID == 4 && req.Kind == KindRoutingRepair && req.RepairID == "routing-repair-deck5-reactor" {
			reactorRouting++
		}
	}
	if authCount != 2 {
		t.Fatalf("reactor auth keycards = %d, want 2", authCount)
	}
	if reactorRouting != 1 {
		t.Fatalf("reactor routing repairs = %d, want 1", reactorRouting)
	}
}

func TestIsDeckTravelUnlocked_StartDecks(t *testing.T) {
	p := RunProgress{LiftRoutingPowered: InitialLiftRouting()}
	if !IsDeckTravelUnlocked(p, 0) || !IsDeckTravelUnlocked(p, 1) {
		t.Fatal("start decks should be unlocked")
	}
	if IsDeckTravelUnlocked(p, 2) {
		t.Fatal("deck 3 should be locked without routing")
	}
}

func TestDeckTravelBlockReason_ReactorOnline(t *testing.T) {
	plan := &Plan{
		Requirements: []Requirement{{
			ID:           "life-support-reactor-5",
			TargetDeckID: 5,
			Kind:         KindReactorOnline,
			SourceDeckID: 4,
		}},
	}
	p := RunProgress{
		Plan:               plan,
		LiftRoutingPowered: map[int]bool{5: true},
		ReactorOnline:      false,
	}
	reason := DeckTravelBlockReason(p, 5)
	if reason != "Needs: Reactor Control online" {
		t.Fatalf("block reason = %q, want reactor online message", reason)
	}
}
