package state

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/unlocks"
)

func TestInitRunUnlocks_StartRouting(t *testing.T) {
	g := NewGame()
	g.InitRunUnlocks(4242)
	if g.UnlockPlan == nil {
		t.Fatal("nil unlock plan")
	}
	if len(g.DeckThemes) != 10 {
		t.Fatalf("themes len = %d, want 10", len(g.DeckThemes))
	}
	if !g.LiftRoutingPowered[0] || !g.LiftRoutingPowered[1] {
		t.Fatal("decks 1–2 routing should be powered at start")
	}
	if g.LiftRoutingPowered[2] {
		t.Fatal("deck 3 routing should not start powered")
	}
}

func TestIsRunWideKeycardName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"Security Keycard", true},
		{"Deck 5 Access Keycard", true},
		{"Reactor Authorization — Observatory", true},
		{"Crew Override Authorization", false},
		{"Map", false},
	}
	for _, tc := range tests {
		if got := IsRunWideKeycardName(tc.name); got != tc.want {
			t.Errorf("IsRunWideKeycardName(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestPromoteOwnedRunKeycards(t *testing.T) {
	g := NewGame()
	g.InitRunUnlocks(99)
	authName := "Reactor Authorization — Observatory"
	g.OwnedItems.Put(world.NewItem(authName))
	g.PromoteOwnedRunKeycards()
	if !g.HasRunKeycard(authName) {
		t.Fatal("expected keycard in run inventory")
	}
	found := false
	g.OwnedItems.Each(func(item *world.Item) {
		if item != nil && item.Name == authName {
			found = true
		}
	})
	if found {
		t.Fatal("keycard should be removed from deck inventory")
	}
}

func TestAddRunKeycard_SatisfiesUnlock(t *testing.T) {
	g := NewGame()
	g.InitRunUnlocks(1)
	var reqID, keycardName string
	for _, req := range g.UnlockPlan.Requirements {
		if req.Kind == unlocks.KindSecurityKeycard && req.KeycardName != "" {
			reqID = req.ID
			keycardName = req.KeycardName
			break
		}
	}
	if keycardName == "" {
		t.Fatal("no keycard requirement in plan")
	}
	g.AddRunKeycard(world.NewItem(keycardName))
	if !g.UnlockSatisfied[reqID] {
		t.Fatalf("expected requirement %q satisfied after pickup", reqID)
	}
}
