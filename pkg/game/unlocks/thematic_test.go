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
	for _, req := range plan.Requirements {
		if req.RepairID == "routing-repair-deck5-reactor" && req.SourceDeckID != 3 {
			t.Fatalf("reactor routing source = %d, want deck 4 (id 3)", req.SourceDeckID)
		}
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

func TestBuildUnlockPlan_SequentialSources(t *testing.T) {
	plan := BuildUnlockPlan(424242, deck.AssignThemes(424242))
	for _, req := range plan.Requirements {
		if req.Kind != KindRoutingRepair {
			continue
		}
		want := sequentialSourceDeck(req.TargetDeckID)
		if req.SourceDeckID != want {
			t.Fatalf("req %q target deck %d: source=%d, want %d (sequential)",
				req.ID, req.TargetDeckID+1, req.SourceDeckID, want)
		}
	}
}

func TestIsDeckTravelUnlocked_SequentialChain(t *testing.T) {
	p := RunProgress{
		LiftRoutingPowered: map[int]bool{0: true, 1: true, 9: true},
		Plan:               &Plan{},
	}
	if IsDeckTravelUnlocked(p, 9) {
		t.Fatal("deck 10 should stay locked when intermediate decks are not unlocked")
	}
	for id := 2; id <= 8; id++ {
		p.LiftRoutingPowered[id] = true
	}
	if !IsDeckTravelUnlocked(p, 9) {
		t.Fatal("deck 10 should unlock once the full chain is powered")
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
