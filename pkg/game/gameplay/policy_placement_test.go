package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/levelgen"
	"darkstation/pkg/game/state"
)

func overrideItemCount(g *state.Game) int {
	count := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			if item != nil && item.Name == entities.CrewOverrideItemName {
				count++
			}
		})
	})
	return count
}

func TestConservationPolicies_placementByDepth(t *testing.T) {
	// Shallow decks have no policies and no override item.
	g := state.NewGame()
	g.Level = 2
	RegenerateFromSeed(g, 7)
	if len(g.Policies) != 0 {
		t.Fatalf("deck 2 should have no policies, got %d", len(g.Policies))
	}
	if overrideItemCount(g) != 0 {
		t.Fatal("deck 2 should not carry a Crew Override Authorization")
	}

	// Deep decks get shed-first + egress-seal and exactly one override item.
	g = state.NewGame()
	g.Level = levelgen.EgressSealLevel
	RegenerateFromSeed(g, 7)
	var kinds []entities.PolicyKind
	for _, p := range g.Policies {
		kinds = append(kinds, p.Kind)
		if p.Overridden {
			t.Fatalf("policy %q must start active", p.ID)
		}
		if p.Kind == entities.PolicyShedFirst && p.TargetRoom == "" {
			t.Fatal("shed-first policy missing target room")
		}
	}
	hasShed, hasSeal := false, false
	for _, k := range kinds {
		switch k {
		case entities.PolicyShedFirst:
			hasShed = true
		case entities.PolicyEgressSeal:
			hasSeal = true
		}
	}
	if !hasShed || !hasSeal {
		t.Fatalf("deck %d policies = %v, want shed_first and egress_seal", g.Level, kinds)
	}
	if n := overrideItemCount(g); n != 1 {
		t.Fatalf("override items on deck = %d, want 1", n)
	}
}

func TestConservationPolicies_deterministicPerSeed(t *testing.T) {
	gen := func() (string, int) {
		g := state.NewGame()
		g.Level = 6
		RegenerateFromSeed(g, 42)
		target := ""
		for _, p := range g.Policies {
			if p.Kind == entities.PolicyShedFirst {
				target = p.TargetRoom
			}
		}
		return target, len(g.Policies)
	}
	t1, n1 := gen()
	t2, n2 := gen()
	if t1 != t2 || n1 != n2 {
		t.Fatalf("policy generation not deterministic: (%q,%d) vs (%q,%d)", t1, n1, t2, n2)
	}
}
