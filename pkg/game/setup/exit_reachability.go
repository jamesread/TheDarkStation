// Package setup — exit reachability at level completion (R7 / I1).
package setup

import (
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// IsPermanentlyBlockingCell reports whether a cell always blocks movement even when
// doors are powered, keycards used, and hazards
// hazards cleared (generators, furniture, terminals, etc.).
func IsPermanentlyBlockingCell(cell *world.Cell) bool {
	if cell == nil || !cell.Room {
		return false
	}
	data := gameworld.GetGameData(cell)
	return data.Generator != nil || data.Furniture != nil || data.Terminal != nil ||
		data.Puzzle != nil || data.MaintenanceTerm != nil || data.HazardControl != nil ||
		data.RepairDevice != nil
}

// isPassableAtLevelCompletion reports whether the player could step on the cell once all win
// conditions are satisfied (hazards cleared; doors powered and unlocked).
func isPassableAtLevelCompletion(cell *world.Cell, extraBlocked *mapset.Set[*world.Cell]) bool {
	if cell == nil || !cell.Room {
		return false
	}
	if extraBlocked != nil && extraBlocked.Has(cell) {
		return false
	}
	if IsPermanentlyBlockingCell(cell) {
		return false
	}
	// Hazards are cleared before the exit is usable; treat hazard cells as passable here.
	return true
}

// ExitReachableWhenCompletable reports whether the exit cell can be reached from the player entry
// assuming all doors are powered/unlocked and all hazards are cleared.
// extraBlocked treats additional cells as permanently blocked (for placement checks).
func ExitReachableWhenCompletable(g *state.Game, extraBlocked *world.Cell) bool {
	if g == nil || g.Grid == nil {
		return false
	}
	entry := PlayerEntryCell(g)
	exit := g.Grid.ExitCell()
	if entry == nil || exit == nil {
		return true // no layout constraint when entry/exit not set (unit tests)
	}
	extra := mapset.New[*world.Cell]()
	if extraBlocked != nil {
		extra.Put(extraBlocked)
	}
	reachable := mapset.New[*world.Cell]()
	queue := []*world.Cell{entry}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == nil || !cur.Room || reachable.Has(cur) {
			continue
		}
		if !isPassableAtLevelCompletion(cur, &extra) {
			continue
		}
		reachable.Put(cur)
		for _, n := range cur.GetNeighbors() {
			if n != nil && n.Room && !reachable.Has(n) {
				queue = append(queue, n)
			}
		}
	}
	return reachable.Has(exit)
}

// CanPlaceBlockingEntity reports whether placing a permanent blocker at candidate still
// leaves a completable path to the exit (R7), preserves adjacent nav space for interactables,
// and does not cut off init-reachable keycards or rooms.
func CanPlaceBlockingEntity(g *state.Game, candidate *world.Cell) bool {
	if g == nil || candidate == nil {
		return false
	}
	if candidate.ItemsOnFloor.Size() > 0 {
		return false
	}
	if !ExitReachableWhenCompletable(g, candidate) {
		return false
	}
	if !BlockingPlacementPreservesNavAccess(g, candidate) {
		return false
	}
	if !bootstrapDoorNavPreserved(g, candidate) {
		return false
	}
	return InitProgressPreserved(g, candidate)
}

// IsAdjacentToExit returns true when cell shares an edge with the exit cell.
func IsAdjacentToExit(g *state.Game, cell *world.Cell) bool {
	if g == nil || g.Grid == nil || cell == nil {
		return false
	}
	exit := g.Grid.ExitCell()
	if exit == nil {
		return false
	}
	for _, n := range cell.GetNeighbors() {
		if n == exit {
			return true
		}
	}
	return false
}

// EnsureExitReachability removes blocking entities that make the exit permanently unreachable
// (safety net after placement). Implements R7 remediation.
func EnsureExitReachability(g *state.Game) {
	if g == nil || g.Grid == nil || ExitReachableWhenCompletable(g, nil) {
		return
	}
	// Prefer clearing blockers closest to the exit first.
	for attempt := 0; attempt < 64 && !ExitReachableWhenCompletable(g, nil); attempt++ {
		exit := g.Grid.ExitCell()
		if exit == nil {
			return
		}
		var candidates []*world.Cell
		g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
			if cell == nil || !IsPermanentlyBlockingCell(cell) {
				return
			}
			candidates = append(candidates, cell)
		})
		if len(candidates) == 0 {
			return
		}
		best := candidates[0]
		bestDist := distToExit(best, exit)
		for _, cell := range candidates[1:] {
			if d := distToExit(cell, exit); d < bestDist {
				best = cell
				bestDist = d
			}
		}
		clearPermanentBlocker(g, best)
	}
}

func distToExit(cell, exit *world.Cell) int {
	dr := cell.Row - exit.Row
	if dr < 0 {
		dr = -dr
	}
	dc := cell.Col - exit.Col
	if dc < 0 {
		dc = -dc
	}
	return dr + dc
}

func clearPermanentBlocker(g *state.Game, cell *world.Cell) {
	if cell == nil {
		return
	}
	data := gameworld.GetGameData(cell)
	data.Furniture = nil
	data.MaintenanceTerm = nil
	data.Terminal = nil
	data.Puzzle = nil
	data.HazardControl = nil
	data.RepairDevice = nil
	// Generators must remain; exit routing should not require stepping on them.
}
