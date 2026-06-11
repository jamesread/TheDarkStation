// Package setup — adjacent navigation space for interactable blockers (generators, furniture, etc.).
package setup

import (
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// RequiresAdjacentNavSpace reports whether the player must stand on a neighbor to use this cell.
func RequiresAdjacentNavSpace(cell *world.Cell) bool {
	return IsPermanentlyBlockingCell(cell)
}

// isNavigableStandCell reports whether the player can stand on cell for interaction/movement
// during layout validation (doors/hazards ignored; those are handled elsewhere).
func isNavigableStandCell(g *state.Game, cell *world.Cell, extraBlocked *mapset.Set[*world.Cell]) bool {
	if cell == nil || !cell.Room {
		return false
	}
	if extraBlocked != nil && extraBlocked.Has(cell) {
		return false
	}
	if RequiresAdjacentNavSpace(cell) {
		return false
	}
	if gameworld.HasBlockingHazard(cell) {
		return false
	}
	return true
}

// EntityHasAdjacentNavSpace reports whether an existing blocking entity has at least one adjacent stand tile.
func EntityHasAdjacentNavSpace(g *state.Game, entityCell *world.Cell, extraBlocked *mapset.Set[*world.Cell]) bool {
	if g == nil || entityCell == nil || !RequiresAdjacentNavSpace(entityCell) {
		return true
	}
	for _, n := range entityCell.GetNeighbors() {
		if isNavigableStandCell(g, n, extraBlocked) {
			return true
		}
	}
	return false
}

// CandidateBlockingCellHasAdjacentNavSpace reports whether a cell would have adjacent stand
// space if a blocking entity were placed there (for pre-placement validation).
func CandidateBlockingCellHasAdjacentNavSpace(g *state.Game, candidate *world.Cell, extraBlocked *mapset.Set[*world.Cell]) bool {
	if g == nil || candidate == nil {
		return false
	}
	blocked := mapset.New[*world.Cell]()
	if extraBlocked != nil {
		extraBlocked.Each(func(c *world.Cell) { blocked.Put(c) })
	}
	blocked.Put(candidate)
	for _, n := range candidate.GetNeighbors() {
		if isNavigableStandCell(g, n, &blocked) {
			return true
		}
	}
	return false
}

// BlockingPlacementPreservesNavAccess reports whether extraBlocked still leaves every blocking
// entity with at least one adjacent free floor cell for player navigation.
func BlockingPlacementPreservesNavAccess(g *state.Game, extraBlocked *world.Cell) bool {
	if g == nil || g.Grid == nil {
		return false
	}
	extra := mapset.New[*world.Cell]()
	if extraBlocked != nil {
		extra.Put(extraBlocked)
	}
	allOK := true
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if !allOK || cell == nil || !RequiresAdjacentNavSpace(cell) {
			return
		}
		if !EntityHasAdjacentNavSpace(g, cell, &extra) {
			allOK = false
		}
	})
	if !allOK {
		return false
	}
	if extraBlocked != nil && !CandidateBlockingCellHasAdjacentNavSpace(g, extraBlocked, nil) {
		return false
	}
	return true
}

func entitiesLackingNavAccess(g *state.Game) []*world.Cell {
	var out []*world.Cell
	if g == nil || g.Grid == nil {
		return out
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && RequiresAdjacentNavSpace(cell) && !EntityHasAdjacentNavSpace(g, cell, nil) {
			out = append(out, cell)
		}
	})
	return out
}

func pickFurnitureToClearForNav(g *state.Game, furniture []*world.Cell) *world.Cell {
	blocked := entitiesLackingNavAccess(g)
	for _, f := range furniture {
		for _, b := range blocked {
			for _, n := range b.GetNeighbors() {
				if n == f {
					return f
				}
			}
		}
	}
	return nil
}

// EnsureInteractableNavAccess removes furniture that leaves generators/terminals without
// an adjacent stand tile (safety net after procedural placement).
func EnsureInteractableNavAccess(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	for attempt := 0; attempt < 64 && !BlockingPlacementPreservesNavAccess(g, nil); attempt++ {
		var furnitureCells []*world.Cell
		g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
			if cell != nil && gameworld.GetGameData(cell).Furniture != nil {
				furnitureCells = append(furnitureCells, cell)
			}
		})
		if len(furnitureCells) == 0 {
			return
		}
		target := pickFurnitureToClearForNav(g, furnitureCells)
		if target == nil {
			target = furnitureCells[0]
		}
		data := gameworld.GetGameData(target)
		if data.Furniture != nil && data.Furniture.ContainedItem != nil {
			target.ItemsOnFloor.Put(data.Furniture.ContainedItem)
		}
		data.Furniture = nil
	}
}
