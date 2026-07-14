// Package setup — completion-region preservation (the I-Rooms invariant).
//
// A permanent blocker (generator, furniture, terminal, repair device, …) must never
// sever any cell that is currently reachable from the player entry under
// level-completion passability (all doors powered/unlocked, hazards cleared,
// clearable blockers drained). This is stronger than only protecting the exit
// path or the init-reachable pocket: it covers rooms behind unpowered doors and
// corridor pockets, so combinations of individually-legal placements can no
// longer seal off parts of the deck.
package setup

import (
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
)

// CompletionReachableFrom returns every cell reachable from start under
// level-completion passability, treating cells in extraBlocked as walls.
func CompletionReachableFrom(g *state.Game, start *world.Cell, extraBlocked *mapset.Set[*world.Cell]) *mapset.Set[*world.Cell] {
	reachable := mapset.New[*world.Cell]()
	if g == nil || g.Grid == nil || start == nil {
		return &reachable
	}
	if !isPassableAtLevelCompletion(start, extraBlocked) {
		return &reachable
	}
	queue := []*world.Cell{start}
	reachable.Put(start)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, n := range cur.GetNeighbors() {
			if n == nil || reachable.Has(n) || !isPassableAtLevelCompletion(n, extraBlocked) {
				continue
			}
			reachable.Put(n)
			queue = append(queue, n)
		}
	}
	return &reachable
}

// CompletionRegionPreserved reports whether a permanent blocker at candidate severs
// no completion-reachable cell other than candidate itself (I-Rooms invariant).
func CompletionRegionPreserved(g *state.Game, candidate *world.Cell) bool {
	if g == nil || g.Grid == nil || candidate == nil {
		return true
	}
	entry := PlayerEntryCell(g)
	if entry == nil {
		return true
	}
	if candidate == entry {
		return false
	}
	base := CompletionReachableFrom(g, entry, nil)
	if !base.Has(candidate) {
		// Candidate is already outside the playable region; it cannot sever it.
		return true
	}
	extra := mapset.New[*world.Cell]()
	extra.Put(candidate)
	with := CompletionReachableFrom(g, entry, &extra)
	return with.Size() == base.Size()-1
}

// CompletionRegionPreservedWithSet is CompletionRegionPreserved for a multi-cell
// placement: blocking every candidate cell must lose exactly those cells from the
// completion-reachable region.
func CompletionRegionPreservedWithSet(g *state.Game, candidates *mapset.Set[*world.Cell]) bool {
	if g == nil || g.Grid == nil || candidates == nil || candidates.Size() == 0 {
		return true
	}
	entry := PlayerEntryCell(g)
	if entry == nil {
		return true
	}
	if candidates.Has(entry) {
		return false
	}
	base := CompletionReachableFrom(g, entry, nil)
	inRegion := 0
	candidates.Each(func(c *world.Cell) {
		if base.Has(c) {
			inRegion++
		}
	})
	if inRegion == 0 {
		return true
	}
	with := CompletionReachableFrom(g, entry, candidates)
	return with.Size() == base.Size()-inRegion
}
