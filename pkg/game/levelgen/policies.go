package levelgen

import (
	"fmt"
	"math/rand"
	"sort"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/levelrand"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
)

// PolicyMinLevel is the first deck with conservation policies.
const PolicyMinLevel = 4

// EgressSealLevel is the first deck where the egress-seal policy appears.
const EgressSealLevel = 6

// EgressSealDelayMs is how long a manual door release lasts on an unpowered room
// while an egress-seal policy is active.
const EgressSealDelayMs = 30_000

// PlaceConservationPolicies seeds this deck's deterministic automation rules and,
// when any exist, one Crew Override Authorization item that can deprecate them.
// Policies are legible constraints, not punishments: the rules are readable at any
// maintenance terminal, and nothing here can make a deck unsolvable (shed-first only
// reorders overload shedding; egress-seal re-seals a release the player can re-pull).
func PlaceConservationPolicies(g *state.Game) {
	if g == nil || g.Grid == nil || g.Level < PolicyMinLevel {
		return
	}
	g.Policies = nil

	rng := levelrand.NewDerived(g.LevelSeed, 0x70011C)

	if target := pickShedFirstRoom(g, rng); target != "" {
		g.Policies = append(g.Policies, &entities.ConservationPolicy{
			ID:         fmt.Sprintf("deck%d-policy-shed", g.CurrentDeckID),
			Code:       "HAB-PRI",
			Kind:       entities.PolicyShedFirst,
			TargetRoom: target,
		})
	}
	if g.Level >= EgressSealLevel {
		g.Policies = append(g.Policies, &entities.ConservationPolicy{
			ID:      fmt.Sprintf("deck%d-policy-seal", g.CurrentDeckID),
			Code:    "ATMOS-SEAL",
			Kind:    entities.PolicyEgressSeal,
			DelayMs: EgressSealDelayMs,
		})
	}
	if len(g.Policies) == 0 {
		return
	}
	placeCrewOverrideItem(g, rng)
}

// pickShedFirstRoom picks a deterministic named room whose loads shed first.
func pickShedFirstRoom(g *state.Game, rng *rand.Rand) string {
	seen := map[string]bool{}
	var rooms []string
	entryRoom := ""
	if entry := setup.PlayerEntryCell(g); entry != nil {
		entryRoom = entry.Name
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name == "" || cell.Name == "Corridor" ||
			generator.IsPlacementExcludedRoom(cell.Name) || cell.Name == entryRoom || seen[cell.Name] {
			return
		}
		seen[cell.Name] = true
		rooms = append(rooms, cell.Name)
	})
	if len(rooms) == 0 {
		return ""
	}
	sort.Strings(rooms)
	return rooms[rng.Intn(len(rooms))]
}

// placeCrewOverrideItem drops one Crew Override Authorization on an empty,
// init-reachable floor cell (falling back to any empty room cell).
func placeCrewOverrideItem(g *state.Game, rng *rand.Rand) {
	reach := setup.InitialReachableCells(g)
	var reachable, fallback []*world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if !validOverrideItemCell(g, cell) {
			return
		}
		if reach != nil && reach.Has(cell) {
			reachable = append(reachable, cell)
		} else {
			fallback = append(fallback, cell)
		}
	})
	candidates := reachable
	if len(candidates) == 0 {
		candidates = fallback
	}
	if len(candidates) == 0 {
		return
	}
	setup.SortCellsByPosition(candidates)
	cell := candidates[rng.Intn(len(candidates))]
	cell.ItemsOnFloor.Put(world.NewItem(entities.CrewOverrideItemName))
}

func validOverrideItemCell(g *state.Game, cell *world.Cell) bool {
	return setup.ValidFloorLootPlacementCell(g, cell, nil)
}
