// Package setup — progression-chain nav access (doors, repairs, hazard controls).
package setup

import (
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// ProgressionNavPreservedByPlacement reports whether placing extraBlocked would remove
// init-reachable access to bootstrap doors or critical interactables that already have it.
func ProgressionNavPreservedByPlacement(g *state.Game, extraBlocked *world.Cell) bool {
	if !bootstrapDoorNavPreservedByPlacement(g, extraBlocked) {
		return false
	}
	for _, cell := range criticalInteractableCells(g) {
		if !entityHasInitReachAdjacentStand(g, cell, nil) {
			continue
		}
		if !entityHasInitReachAdjacentStand(g, cell, extraBlocked) {
			return false
		}
	}
	return true
}

func bootstrapDoorNavPreservedByPlacement(g *state.Game, extraBlocked *world.Cell) bool {
	for _, door := range initPocketBootstrapDoors(g) {
		if !doorHasStandNavAccess(g, door, nil) {
			continue
		}
		if !doorHasStandNavAccess(g, door, extraBlocked) {
			return false
		}
	}
	return true
}

// ProgressionNavPreserved reports whether every bootstrap door and critical progression
// interactable has init-reachable adjacent stand space.
func ProgressionNavPreserved(g *state.Game, extraBlocked *world.Cell) bool {
	if !bootstrapDoorNavPreserved(g, extraBlocked) {
		return false
	}
	for _, cell := range criticalInteractableCells(g) {
		if !entityHasInitReachAdjacentStand(g, cell, extraBlocked) {
			return false
		}
	}
	return true
}

// entityHasInitReachAdjacentStand reports whether the player can stand on a neighbor
// reachable from the lift entry to interact with entityCell.
func entityHasInitReachAdjacentStand(g *state.Game, entityCell *world.Cell, extraBlocked *world.Cell) bool {
	if g == nil || entityCell == nil || !RequiresAdjacentNavSpace(entityCell) {
		return true
	}
	reach := InitialReachableCellsWithExtraBlock(g, extraBlocked)
	extra := mapset.New[*world.Cell]()
	if extraBlocked != nil {
		extra.Put(extraBlocked)
	}
	for _, n := range entityCell.GetNeighbors() {
		if isNavigableStandCell(g, n, &extra) && reach.Has(n) {
			return true
		}
	}
	return false
}

func criticalInteractableCells(g *state.Game) []*world.Cell {
	if g == nil || g.Grid == nil {
		return nil
	}
	var out []*world.Cell
	seen := make(map[*world.Cell]bool)
	add := func(cell *world.Cell) {
		if cell == nil || seen[cell] {
			return
		}
		seen[cell] = true
		out = append(out, cell)
	}
	for _, repair := range g.RepairObjectives {
		if repair == nil || repair.SkipExitGate || repair.DeviceRow < 0 || repair.DeviceCol < 0 {
			continue
		}
		add(g.Grid.GetCell(repair.DeviceRow, repair.DeviceCol))
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		h := gameworld.GetGameData(cell).Hazard
		if h == nil || h.Fixed || !h.IsBlocking() {
			return
		}
		if hc := hazardControlCellFor(g, h.Control); hc != nil {
			add(hc)
		}
	})
	return out
}

func hazardControlCellFor(g *state.Game, control *entities.HazardControl) *world.Cell {
	if g == nil || g.Grid == nil || control == nil {
		return nil
	}
	var found *world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if found != nil || cell == nil {
			return
		}
		if gameworld.GetGameData(cell).HazardControl == control {
			found = cell
		}
	})
	return found
}

// EnsureProgressionNavAccess clears furniture blocking bootstrap doors and critical interactables.
func EnsureProgressionNavAccess(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	EnsureBootstrapDoorNavAccess(g)
	armCriticalInteractableRoomPower(g)
	for attempt := 0; attempt < 64; attempt++ {
		if ProgressionNavPreserved(g, nil) {
			return
		}
		var furnitureCells []*world.Cell
		g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
			if cell != nil && gameworld.GetGameData(cell).Furniture != nil {
				furnitureCells = append(furnitureCells, cell)
			}
		})
		if len(furnitureCells) == 0 {
			return
		}
		target := pickFurnitureBlockingProgression(g, furnitureCells)
		if target == nil {
			if !progressionNavBlockedByFurniture(g, furnitureCells) {
				// Furniture is not the impediment (e.g. doors not yet powered);
				// removing it cannot help, so leave it in place.
				return
			}
			target = furnitureCells[0]
		}
		data := gameworld.GetGameData(target)
		if data.Furniture != nil && data.Furniture.ContainedItem != nil {
			target.ItemsOnFloor.Put(data.Furniture.ContainedItem)
		}
		data.Furniture = nil
	}
}

// armCriticalInteractableRoomPower arms door power for rooms hosting critical progression
// interactables that lack init-reachable stand space but whose doors can be powered from the
// entry pocket (e.g. via the lift shaft bootstrap terminal). This mirrors the player's first
// action at the shaft terminal so init-reach invariants hold for the progression chain.
func armCriticalInteractableRoomPower(g *state.Game) {
	if g.RoomDoorsPowered == nil {
		g.RoomDoorsPowered = make(map[string]bool)
	}
	changed := false
	for _, cell := range criticalInteractableCells(g) {
		if cell == nil || cell.Name == "" || cell.Name == "Corridor" {
			continue
		}
		if g.RoomDoorsPowered[cell.Name] {
			continue
		}
		if entityHasInitReachAdjacentStand(g, cell, nil) {
			continue
		}
		reach := InitialReachableCells(g)
		if !CanPowerRoomDoorsFromReachable(g, reach, cell.Name) {
			continue
		}
		g.RoomDoorsPowered[cell.Name] = true
		changed = true
	}
	if changed {
		g.InvalidateLivePowerCache()
		PropagateRoomPowerOnlineFromGenerators(g)
	}
}

// progressionNavBlockedByFurniture reports whether removing all furniture restores
// progression nav access, i.e. furniture is at least part of the impediment.
func progressionNavBlockedByFurniture(g *state.Game, furniture []*world.Cell) bool {
	saved := make(map[*world.Cell]*entities.Furniture, len(furniture))
	for _, f := range furniture {
		data := gameworld.GetGameData(f)
		saved[f] = data.Furniture
		data.Furniture = nil
	}
	ok := ProgressionNavPreserved(g, nil)
	for cell, f := range saved {
		gameworld.GetGameData(cell).Furniture = f
	}
	return ok
}

func pickFurnitureBlockingProgression(g *state.Game, furniture []*world.Cell) *world.Cell {
	for _, f := range furniture {
		data := gameworld.GetGameData(f)
		saved := data.Furniture
		data.Furniture = nil
		if ProgressionNavPreserved(g, nil) {
			return f
		}
		data.Furniture = saved
	}
	return nil
}
