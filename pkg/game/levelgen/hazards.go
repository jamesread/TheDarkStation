// Package levelgen provides level generation utilities for placing entities.
package levelgen

import (
	"darkstation/pkg/game/levelrand"
	"fmt"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// PlaceHazards places environmental hazards on corridor chokepoints or room passages.
// Each hazard's fix (control panel or item) is always placed in the area reachable before
// crossing the hazard, so the puzzle remains solvable.
func PlaceHazards(g *state.Game, avoid *mapset.Set[*world.Cell], lockedDoorCells *mapset.Set[*world.Cell]) {
	if g == nil || g.Grid == nil || setup.PlayerEntryCell(g) == nil {
		return
	}

	numHazards := hazardCountForLevel(g.Level)
	hazardTypes := hazardTypesForLevel(g.Level)

	blocked := mapset.New[*world.Cell]()
	lockedDoorCells.Each(func(c *world.Cell) { blocked.Put(c) })

	hazardsPlaced := 0
	for hazardsPlaced < numHazards {
		candidates := collectHazardCandidateCells(g, lockedDoorCells, &blocked)
		if len(candidates) == 0 {
			break
		}
		placed := false
		for _, cell := range candidates {
			if tryPlaceHazardAt(g, cell, hazardTypes, lockedDoorCells, &blocked, avoid) {
				hazardsPlaced++
				placed = true
				break
			}
		}
		if !placed {
			break
		}
	}
}

func hazardCountForLevel(level int) int {
	if level >= 4 {
		return 2 + levelrand.Intn(2)
	}
	if level >= 3 {
		return 1 + levelrand.Intn(2)
	}
	return 1
}

func hazardTypesForLevel(level int) []entities.HazardType {
	types := []entities.HazardType{
		entities.HazardCoolant,
		entities.HazardElectrical,
		entities.HazardGas,
	}
	if level >= 3 {
		types = append(types, entities.HazardVacuum)
	}
	if level >= 5 {
		types = append(types, entities.HazardRadiation)
	}
	return types
}

func collectHazardCandidateCells(g *state.Game, lockedDoorCells, blocked *mapset.Set[*world.Cell]) []*world.Cell {
	currentlyReachable := GetReachableCells(g.Grid, setup.PlayerEntryCell(g), blocked)
	reachableSize := currentlyReachable.Size()
	var corridorCandidates, roomCandidates []*world.Cell

	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if !isValidHazardHostCell(g, cell, lockedDoorCells, currentlyReachable) {
			return
		}
		if !blockingCellReducesReachability(g.Grid, setup.PlayerEntryCell(g), blocked, cell, reachableSize) {
			return
		}
		if !setup.InitProgressPreserved(g, cell) {
			return
		}
		if cell.Name == "Corridor" {
			corridorCandidates = append(corridorCandidates, cell)
		} else {
			roomCandidates = append(roomCandidates, cell)
		}
	})

	levelrand.Shuffle(len(corridorCandidates), func(i, j int) {
		corridorCandidates[i], corridorCandidates[j] = corridorCandidates[j], corridorCandidates[i]
	})
	levelrand.Shuffle(len(roomCandidates), func(i, j int) {
		roomCandidates[i], roomCandidates[j] = roomCandidates[j], roomCandidates[i]
	})

	out := make([]*world.Cell, 0, len(corridorCandidates)+len(roomCandidates))
	out = append(out, corridorCandidates...)
	out = append(out, roomCandidates...)
	return out
}

func isValidHazardHostCell(g *state.Game, cell *world.Cell, lockedDoorCells, currentlyReachable *mapset.Set[*world.Cell]) bool {
	if cell == nil || !cell.Room {
		return false
	}
	entry := setup.PlayerEntryCell(g)
	if cell == entry {
		return false
	}
	if lockedDoorCells.Has(cell) || !currentlyReachable.Has(cell) {
		return false
	}
	if setup.IsAdjacentToExit(g, cell) {
		return false
	}
	data := gameworld.GetGameData(cell)
	return data.Generator == nil && data.Furniture == nil && data.Terminal == nil &&
		data.Puzzle == nil && data.MaintenanceTerm == nil &&
		data.Hazard == nil && data.HazardControl == nil
}

func blockingCellReducesReachability(grid *world.Grid, start *world.Cell, blocked *mapset.Set[*world.Cell], cell *world.Cell, beforeSize int) bool {
	after := GetReachableCellsExcluding(grid, start, blocked, cell)
	return after.Size() < beforeSize
}

func reachableWithoutCells(grid *world.Grid, start *world.Cell, blocked *mapset.Set[*world.Cell]) *mapset.Set[*world.Cell] {
	return GetReachableCells(grid, start, blocked)
}

func tryPlaceHazardAt(g *state.Game, cell *world.Cell, hazardTypes []entities.HazardType, lockedDoorCells, blocked, avoid *mapset.Set[*world.Cell]) bool {
	testBlocked := mapset.New[*world.Cell]()
	blocked.Each(func(c *world.Cell) { testBlocked.Put(c) })
	testBlocked.Put(cell)

	reachableBefore := reachableWithoutCells(g.Grid, setup.PlayerEntryCell(g), blocked)
	reachableWithHazard := reachableWithoutCells(g.Grid, setup.PlayerEntryCell(g), &testBlocked)

	hazardType := hazardTypes[levelrand.Intn(len(hazardTypes))]
	hazard := entities.NewHazard(hazardType)
	info := entities.HazardTypes[hazardType]

	gameworld.GetGameData(cell).Hazard = hazard

	if !placeHazardSolution(g, hazard, info, cell, lockedDoorCells, reachableWithHazard, reachableBefore, avoid) {
		gameworld.GetGameData(cell).Hazard = nil
		return false
	}

	blocked.Put(cell)
	avoid.Put(cell)
	addHazardHint(g, cell, info)
	return true
}

func placeHazardSolution(g *state.Game, hazard *entities.Hazard, info entities.HazardInfo, hazardCell *world.Cell, lockedDoorCells, reachableWithHazard, reachableBefore *mapset.Set[*world.Cell], avoid *mapset.Set[*world.Cell]) bool {
	if hazard.RequiresItem() {
		itemCell := findHazardItemCell(g, hazardCell, lockedDoorCells, reachableWithHazard, reachableBefore, avoid)
		if itemCell == nil || !solutionReachableWithoutHazard(reachableWithHazard, hazardCell, itemCell) {
			return false
		}
		item := world.NewItem(info.ItemName)
		itemCell.ItemsOnFloor.Put(item)
		avoid.Put(itemCell)
		g.AddHint("A " + renderer.StyledItem(info.ItemName) + " is in " + renderer.StyledCell(itemCell.Name))
		return true
	}

	controlRoom := findHazardControlCell(g, hazardCell, lockedDoorCells, reachableWithHazard, avoid)
	if controlRoom == nil {
		return false
	}
	if !canReachWithoutHazardWithGame(g, g.Grid, setup.PlayerEntryCell(g), controlRoom, hazardCell, lockedDoorCells) {
		return false
	}
	if !hazardControlReachableFromFarSide(g, hazardCell, controlRoom, lockedDoorCells) {
		return false
	}
	control := entities.NewHazardControl(hazard.Type, hazard)
	gameworld.GetGameData(controlRoom).HazardControl = control
	avoid.Put(controlRoom)
	g.AddHint("The " + renderer.StyledHazardCtrl(info.ControlName) + " is in " + renderer.StyledCell(controlRoom.Name))
	return true
}

func solutionReachableWithoutHazard(reachableWithout *mapset.Set[*world.Cell], hazardCell, solutionCell *world.Cell) bool {
	if solutionCell == nil || hazardCell == nil || reachableWithout == nil {
		return false
	}
	return reachableWithout.Has(solutionCell)
}

func findHazardControlCell(g *state.Game, hazardCell *world.Cell, lockedDoorCells, reachableWithHazard, avoid *mapset.Set[*world.Cell]) *world.Cell {
	candidates := hazardControlCandidates(g, hazardCell, lockedDoorCells, reachableWithHazard, avoid)
	if len(candidates) == 0 {
		return nil
	}
	setup.SortCellsByPosition(candidates)
	levelrand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	for _, cell := range candidates {
		if hazardControlReachableFromFarSide(g, hazardCell, cell, lockedDoorCells) {
			return cell
		}
	}
	return nil
}

func hazardControlCandidates(g *state.Game, hazardCell *world.Cell, lockedDoorCells, reachableWithHazard, avoid *mapset.Set[*world.Cell]) []*world.Cell {
	var preferred, fallback []*world.Cell
	placement := setup.NewBlockingPlacementValidator(g)
	canPlaceCache := make(map[*world.Cell]bool)
	noLockedDoors := mapset.New[*world.Cell]()
	addCandidate := func(cell *world.Cell) {
		if cell == nil {
			return
		}
		if cell == hazardCell || avoid.Has(cell) || cell.ExitCell || generator.IsPlacementExcludedRoom(cell.Name) {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.HazardControl != nil || data.Generator != nil || data.MaintenanceTerm != nil {
			return
		}
		if cell.ItemsOnFloor.Size() > 0 {
			return
		}
		canPlace, ok := canPlaceCache[cell]
		if !ok {
			canPlace = placement.CanPlace(cell)
			canPlaceCache[cell] = canPlace
		}
		if !canPlace {
			return
		}
		if IsArticulationPoint(g.Grid, setup.PlayerEntryCell(g), cell, lockedDoorCells) {
			return
		}
		// Also reject chokepoints of the door-openable graph: locked doors open later
		// (keycards are placed on-deck), and a permanent blocker in front of a door's
		// only approach cell would wall that room off forever (e.g. rooms hosting
		// exit-gating repairs, soft-locking the deck).
		if IsArticulationPoint(g.Grid, setup.PlayerEntryCell(g), cell, &noLockedDoors) {
			return
		}
		if cell.Name != "Corridor" {
			preferred = append(preferred, cell)
		} else {
			fallback = append(fallback, cell)
		}
	}

	reachableWithHazard.Each(addCandidate)
	farAdjacent := hazardFarSideAdjacentCells(g, hazardCell, lockedDoorCells)
	for _, far := range farAdjacent {
		farSide := farSideReachableWithoutHazard(g.Grid, far, hazardCell, lockedDoorCells)
		farSide.Each(addCandidate)
	}

	if len(farAdjacent) > 0 {
		var farOnlyPreferred, farOnlyFallback []*world.Cell
		farSideAll := mapset.New[*world.Cell]()
		for _, far := range farAdjacent {
			farSideReachableWithoutHazard(g.Grid, far, hazardCell, lockedDoorCells).Each(func(c *world.Cell) {
				farSideAll.Put(c)
			})
		}
		for _, c := range preferred {
			if farSideAll.Has(c) {
				farOnlyPreferred = append(farOnlyPreferred, c)
			}
		}
		for _, c := range fallback {
			if farSideAll.Has(c) {
				farOnlyFallback = append(farOnlyFallback, c)
			}
		}
		if len(farOnlyPreferred) > 0 {
			return farOnlyPreferred
		}
		if len(farOnlyFallback) > 0 {
			return farOnlyFallback
		}
	}

	if len(preferred) > 0 {
		return preferred
	}
	return fallback
}

func farSideReachableWithoutHazard(grid *world.Grid, seed, hazard *world.Cell, lockedDoorCells *mapset.Set[*world.Cell]) *mapset.Set[*world.Cell] {
	out := mapset.New[*world.Cell]()
	if grid == nil || seed == nil || hazard == nil {
		return &out
	}
	queue := []*world.Cell{seed}
	out.Put(seed)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, n := range cur.GetNeighbors() {
			if n == nil || !n.Room || out.Has(n) || n == hazard {
				continue
			}
			if lockedDoorCells.Has(n) {
				continue
			}
			out.Put(n)
			queue = append(queue, n)
		}
	}
	return &out
}

// hazardFarSideAdjacentCells returns hazard neighbors reachable from start only by crossing hazardCell.
func hazardFarSideAdjacentCells(g *state.Game, hazardCell *world.Cell, lockedDoorCells *mapset.Set[*world.Cell]) []*world.Cell {
	if g == nil || g.Grid == nil || hazardCell == nil {
		return nil
	}
	blockedWithHazard := mapset.New[*world.Cell]()
	lockedDoorCells.Each(func(c *world.Cell) { blockedWithHazard.Put(c) })
	blockedWithHazard.Put(hazardCell)

	before := GetReachableCells(g.Grid, setup.PlayerEntryCell(g), &blockedWithHazard)
	full := GetReachableCells(g.Grid, setup.PlayerEntryCell(g), lockedDoorCells)

	var far []*world.Cell
	for _, n := range hazardCell.GetNeighbors() {
		if n == nil || !n.Room || !full.Has(n) || before.Has(n) {
			continue
		}
		far = append(far, n)
	}
	return far
}

// hazardControlReachableFromFarSide ensures players who reach the hazard from the far side can
// still reach the control without crossing the hazard (avoids start-side-only control traps).
func hazardControlReachableFromFarSide(g *state.Game, hazardCell, controlCell *world.Cell, lockedDoorCells *mapset.Set[*world.Cell]) bool {
	if hazardCell == nil || controlCell == nil {
		return false
	}
	for _, far := range hazardFarSideAdjacentCells(g, hazardCell, lockedDoorCells) {
		if !canReachWithoutHazardWithGame(g, g.Grid, far, controlCell, hazardCell, lockedDoorCells) {
			return false
		}
	}
	return true
}

func canReachWithoutHazard(grid *world.Grid, from, to, hazard *world.Cell, lockedDoorCells *mapset.Set[*world.Cell]) bool {
	return canReachWithoutHazardWithGame(nil, grid, from, to, hazard, lockedDoorCells)
}

func canReachWithoutHazardWithGame(g *state.Game, grid *world.Grid, from, to, hazard *world.Cell, lockedDoorCells *mapset.Set[*world.Cell]) bool {
	if from == nil || to == nil || hazard == nil || grid == nil {
		return false
	}
	visited := mapset.New[*world.Cell]()
	queue := []*world.Cell{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == nil || !cur.Room || visited.Has(cur) {
			continue
		}
		if cur == hazard {
			continue
		}
		if cur == to {
			return true
		}
		visited.Put(cur)
		for _, n := range cur.GetNeighbors() {
			if n == nil || !n.Room || visited.Has(n) || n == hazard {
				continue
			}
			if lockedDoorCells.Has(n) {
				continue
			}
			if g != nil {
				ok, _ := setup.CanEnterCellAtInit(g, n)
				if !ok {
					continue
				}
			}
			queue = append(queue, n)
		}
	}
	return false
}

func addHazardHint(g *state.Game, cell *world.Cell, info entities.HazardInfo) {
	if cell.Name == "Corridor" {
		g.AddHint(fmt.Sprintf("A %s blocks a corridor passage", info.Name))
		return
	}
	g.AddHint(fmt.Sprintf("A %s blocks passage through %s", info.Name, renderer.StyledCell(cell.Name)))
}

// EnsureHazardControlsSolvable relocates hazard controls that are only reachable from the
// start side of a chokepoint, which traps players who approach from the far side.
func EnsureHazardControlsSolvable(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	locked := collectLockedDoorCells(g)
	avoid := collectHazardSolutionItemCells(g)

	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !gameworld.HasBlockingHazard(cell) {
			return
		}
		hazard := gameworld.GetGameData(cell).Hazard
		if hazard == nil || hazard.RequiresItem() || hazard.Control == nil {
			return
		}
		controlCell := hazardControlCell(g, hazard.Control)
		if controlCell == nil {
			return
		}
		if canReachWithoutHazardWithGame(g, g.Grid, setup.PlayerEntryCell(g), controlCell, cell, &locked) &&
			hazardControlReachableFromFarSide(g, cell, controlCell, &locked) {
			return
		}

		gameworld.GetGameData(controlCell).HazardControl = nil
		blocked := mapset.New[*world.Cell]()
		locked.Each(func(c *world.Cell) { blocked.Put(c) })
		blocked.Put(cell)
		reachableWithHazard := GetReachableCells(g.Grid, setup.PlayerEntryCell(g), &blocked)

		replacement := findHazardControlCell(g, cell, &locked, reachableWithHazard, &avoid)
		if replacement == nil {
			return
		}
		gameworld.GetGameData(replacement).HazardControl = hazard.Control
		avoid.Put(replacement)
	})
}

// EnsureHazardSolutionsDisjoint moves hazard fix items off cells that also host hazard controls.
// Hazard controls block movement and render above floor items, which makes patch kits unreachable.
func EnsureHazardSolutionsDisjoint(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	locked := collectLockedDoorCells(g)
	avoid := collectHazardSolutionItemCells(g)

	var conflictCells []*world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		if gameworld.GetGameData(cell).HazardControl != nil && cell.ItemsOnFloor.Size() > 0 {
			conflictCells = append(conflictCells, cell)
		}
	})

	for _, cell := range conflictCells {
		relocateHazardSolutionItems(g, cell, &locked, &avoid)
	}
}

func findHazardItemCell(g *state.Game, hazardCell *world.Cell, lockedDoorCells, reachableWithHazard, reachableBefore, avoid *mapset.Set[*world.Cell]) *world.Cell {
	reachables := []*mapset.Set[*world.Cell]{reachableWithHazard, reachableBefore}
	for _, reach := range reachables {
		if reach == nil {
			continue
		}
		var candidates []*world.Cell
		reach.Each(func(cell *world.Cell) {
			if !isValidHazardItemCell(g, cell, hazardCell, lockedDoorCells, avoid) {
				return
			}
			if !solutionReachableWithoutHazard(reach, hazardCell, cell) {
				return
			}
			candidates = append(candidates, cell)
		})
		if len(candidates) == 0 {
			continue
		}
		setup.SortCellsByPosition(candidates)
		levelrand.Shuffle(len(candidates), func(i, j int) {
			candidates[i], candidates[j] = candidates[j], candidates[i]
		})
		return candidates[0]
	}
	return nil
}

func isValidHazardItemCell(g *state.Game, cell, hazardCell *world.Cell, lockedDoorCells, avoid *mapset.Set[*world.Cell]) bool {
	if cell == nil || cell == hazardCell {
		return false
	}
	if lockedDoorCells != nil && lockedDoorCells.Has(cell) {
		return false
	}
	return setup.ValidFloorLootPlacementCell(g, cell, avoid)
}

func collectHazardSolutionItemCells(g *state.Game) mapset.Set[*world.Cell] {
	cells := mapset.New[*world.Cell]()
	if g == nil || g.Grid == nil {
		return cells
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || cell.ItemsOnFloor.Size() == 0 {
			return
		}
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			if isHazardSolutionItemName(item.Name) {
				cells.Put(cell)
			}
		})
	})
	return cells
}

func isHazardSolutionItemName(name string) bool {
	for _, info := range entities.HazardTypes {
		if info.RequiresItem && info.ItemName == name {
			return true
		}
	}
	return false
}

func relocateHazardSolutionItems(g *state.Game, conflictCell *world.Cell, lockedDoorCells, avoid *mapset.Set[*world.Cell]) {
	if g == nil || conflictCell == nil {
		return
	}
	var items []*world.Item
	conflictCell.ItemsOnFloor.Each(func(item *world.Item) {
		if item != nil && isHazardSolutionItemName(item.Name) {
			items = append(items, item)
		}
	})
	for _, item := range items {
		conflictCell.ItemsOnFloor.Remove(item)
		dest := findHazardItemRelocationCell(g, conflictCell, lockedDoorCells, avoid)
		if dest == nil {
			conflictCell.ItemsOnFloor.Put(item)
			continue
		}
		dest.ItemsOnFloor.Put(item)
		avoid.Put(dest)
	}
}

func findHazardItemRelocationCell(g *state.Game, near *world.Cell, lockedDoorCells, avoid *mapset.Set[*world.Cell]) *world.Cell {
	if g == nil || near == nil {
		return nil
	}
	var candidates []*world.Cell
	add := func(cell *world.Cell) {
		if cell == nil || avoid.Has(cell) {
			return
		}
		if lockedDoorCells.Has(cell) {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.HazardControl != nil || data.Generator != nil || data.MaintenanceTerm != nil {
			return
		}
		if data.Hazard != nil && data.Hazard.IsBlocking() {
			return
		}
		if cell.ItemsOnFloor.Size() > 0 {
			return
		}
		ok, _ := setup.CanEnterCellAtInit(g, cell)
		if !ok {
			return
		}
		candidates = append(candidates, cell)
	}
	for _, n := range near.GetNeighbors() {
		add(n)
	}
	if len(candidates) == 0 {
		reach := GetReachableCells(g.Grid, setup.PlayerEntryCell(g), lockedDoorCells)
		reach.Each(func(cell *world.Cell) { add(cell) })
	}
	if len(candidates) == 0 {
		return nil
	}
	setup.SortCellsByPosition(candidates)
	levelrand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	return candidates[0]
}

func collectLockedDoorCells(g *state.Game) mapset.Set[*world.Cell] {
	locked := mapset.New[*world.Cell]()
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && gameworld.HasLockedDoor(cell) {
			locked.Put(cell)
		}
	})
	return locked
}

func hazardControlCell(g *state.Game, control *entities.HazardControl) *world.Cell {
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
