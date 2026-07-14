package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// makeShedPolicyGrid: one conductive strip with a generator and three rooms,
// each with a door (10w). Supply tuned so two rooms must shed.
func makeShedPolicyGrid(t *testing.T) (*state.Game, *world.Grid) {
	t.Helper()
	grid := world.NewGrid(1, 9)
	names := []string{"Alpha", "Alpha", "Alpha", "Beta", "Beta", "Beta", "Gamma", "Gamma", "Gamma"}
	for c := 0; c < 9; c++ {
		grid.MarkAsRoomWithName(0, c, names[c], "")
		gameworld.InitGameData(grid.GetCell(0, c))
	}
	grid.BuildAllCellConnections()
	g := state.NewGame()
	g.Grid = grid
	return g, grid
}

func TestSortShedQueue_policyTargetShedsFirst(t *testing.T) {
	g, _ := makeShedPolicyGrid(t)
	list := []shedConsumer{
		{"Alpha", "doors"},
		{"Beta", "cctv"},
		{"Beta", "doors"},
		{"Gamma", "doors"},
	}

	sortShedQueue(g, list)
	if list[0].room != "Alpha" {
		t.Fatalf("without policy, queue head = %q, want alphabetical Alpha", list[0].room)
	}

	g.Policies = []*entities.ConservationPolicy{{
		ID: "p1", Code: "HAB-PRI", Kind: entities.PolicyShedFirst, TargetRoom: "Gamma",
	}}
	sortShedQueue(g, list)
	if list[0].room != "Gamma" {
		t.Fatalf("with shed-first policy on Gamma, queue head = %q, want Gamma", list[0].room)
	}
	if list[1].room != "Alpha" || list[2].room != "Beta" || list[2].kind != "doors" {
		t.Fatalf("remaining queue should stay name-deterministic, got %v", list)
	}

	g.Policies[0].Overridden = true
	sortShedQueue(g, list)
	if list[0].room != "Alpha" {
		t.Fatalf("deprecated policy must not bias the queue, head = %q", list[0].room)
	}
}

func TestAdvanceEgressSeal_resealsAfterDelay(t *testing.T) {
	g, _ := makeShedPolicyGrid(t)
	g.Policies = []*entities.ConservationPolicy{{
		ID: "p1", Code: "ATMOS-SEAL", Kind: entities.PolicyEgressSeal, DelayMs: 30_000,
	}}
	g.ManualEgressReleased = map[string]bool{"Alpha": true}
	g.ManualEgressReleasedAtMs = map[string]int64{"Alpha": 1_000}

	if sealed := AdvanceEgressSeal(g, 20_000); len(sealed) != 0 {
		t.Fatalf("sealed %v before delay elapsed", sealed)
	}
	sealed := AdvanceEgressSeal(g, 31_001)
	if len(sealed) != 1 || sealed[0] != "Alpha" {
		t.Fatalf("sealed = %v, want [Alpha]", sealed)
	}
	if g.ManualEgressReleased["Alpha"] {
		t.Fatal("release flag should be cleared after re-seal")
	}
}

func TestAdvanceEgressSeal_skipsPoweredRoomsAndNoPolicy(t *testing.T) {
	g, _ := makeShedPolicyGrid(t)
	g.ManualEgressReleased = map[string]bool{"Alpha": true}
	g.ManualEgressReleasedAtMs = map[string]int64{"Alpha": 0}

	if sealed := AdvanceEgressSeal(g, 1_000_000); len(sealed) != 0 {
		t.Fatalf("no policy: sealed %v", sealed)
	}

	g.Policies = []*entities.ConservationPolicy{{
		ID: "p1", Code: "ATMOS-SEAL", Kind: entities.PolicyEgressSeal, DelayMs: 30_000,
	}}
	g.RoomPowerOnline = map[string]bool{"Alpha": true}
	if sealed := AdvanceEgressSeal(g, 1_000_000); len(sealed) != 0 {
		t.Fatalf("powered room must not re-seal, got %v", sealed)
	}
	if !g.ManualEgressReleased["Alpha"] {
		t.Fatal("release flag must survive on powered room")
	}
}

func TestOverrideDeckPolicies_permanent(t *testing.T) {
	g := state.NewGame()
	g.Policies = []*entities.ConservationPolicy{
		{ID: "a", Kind: entities.PolicyShedFirst, TargetRoom: "X"},
		{ID: "b", Kind: entities.PolicyEgressSeal, DelayMs: 30_000},
		{ID: "c", Kind: entities.PolicyShedFirst, TargetRoom: "Y", Overridden: true},
	}
	if n := OverrideDeckPolicies(g); n != 2 {
		t.Fatalf("overrode %d policies, want 2", n)
	}
	if ActivePolicyCount(g) != 0 {
		t.Fatal("all policies should be deprecated after override")
	}
}
