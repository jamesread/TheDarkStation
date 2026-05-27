// Package levelgen provides level generation utilities for placing entities.
package levelgen

import (
	"darkstation/pkg/game/levelrand"
	"fmt"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// PlaceHazards places environmental hazards on corridor chokepoints or room passages.
// Each hazard's fix (control panel or item) is always placed in the area reachable before
// crossing the hazard, so the puzzle remains solvable.
func PlaceHazards(g *state.Game, avoid *mapset.Set[*world.Cell], lockedDoorCells *mapset.Set[*world.Cell]) {
	if g == nil || g.Grid == nil || g.Grid.StartCell() == nil {
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
	currentlyReachable := GetReachableCells(g.Grid, g.Grid.StartCell(), blocked)
	var corridorCandidates, roomCandidates []*world.Cell

	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if !isValidHazardHostCell(g, cell, lockedDoorCells, currentlyReachable) {
			return
		}
		if !blockingCellReducesReachability(g.Grid, g.Grid.StartCell(), lockedDoorCells, blocked, cell) {
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
	start := g.Grid.StartCell()
	exit := g.Grid.ExitCell()
	if cell == start || cell == exit {
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

func blockingCellReducesReachability(grid *world.Grid, start *world.Cell, lockedDoorCells, blocked *mapset.Set[*world.Cell], cell *world.Cell) bool {
	testBlocked := mapset.New[*world.Cell]()
	blocked.Each(func(c *world.Cell) { testBlocked.Put(c) })
	testBlocked.Put(cell)

	before := GetReachableCells(grid, start, blocked)
	after := GetReachableCells(grid, start, &testBlocked)
	return after.Size() < before.Size()
}

func reachableWithoutCells(grid *world.Grid, start *world.Cell, blocked *mapset.Set[*world.Cell]) *mapset.Set[*world.Cell] {
	return GetReachableCells(grid, start, blocked)
}

func tryPlaceHazardAt(g *state.Game, cell *world.Cell, hazardTypes []entities.HazardType, lockedDoorCells, blocked, avoid *mapset.Set[*world.Cell]) bool {
	testBlocked := mapset.New[*world.Cell]()
	blocked.Each(func(c *world.Cell) { testBlocked.Put(c) })
	testBlocked.Put(cell)

	reachableBefore := reachableWithoutCells(g.Grid, g.Grid.StartCell(), blocked)
	reachableWithHazard := reachableWithoutCells(g.Grid, g.Grid.StartCell(), &testBlocked)

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
		itemRoom := FindRoomInReachable(reachableWithHazard, avoid)
		if itemRoom == nil {
			itemRoom = FindRoomInReachable(reachableBefore, avoid)
		}
		if itemRoom == nil || !solutionReachableWithoutHazard(reachableWithHazard, hazardCell, itemRoom) {
			return false
		}
		item := world.NewItem(info.ItemName)
		itemRoom.ItemsOnFloor.Put(item)
		avoid.Put(itemRoom)
		g.AddHint("A " + renderer.StyledItem(info.ItemName) + " is in " + renderer.StyledCell(itemRoom.Name))
		return true
	}

	controlRoom := findHazardControlCell(g, hazardCell, lockedDoorCells, reachableWithHazard, avoid)
	if controlRoom == nil {
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
	var preferred, fallback []*world.Cell
	reachableWithHazard.Each(func(cell *world.Cell) {
		if cell == nil || cell == hazardCell || avoid.Has(cell) {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.HazardControl != nil || data.Generator != nil || data.MaintenanceTerm != nil {
			return
		}
		if !setup.CanPlaceBlockingEntity(g, cell) {
			return
		}
		if IsArticulationPoint(g.Grid, g.Grid.StartCell(), cell, lockedDoorCells) {
			return
		}
		if cell.Name != "Corridor" {
			preferred = append(preferred, cell)
		} else {
			fallback = append(fallback, cell)
		}
	})

	candidates := preferred
	if len(candidates) == 0 {
		candidates = fallback
	}
	if len(candidates) == 0 {
		return nil
	}
	setup.SortCellsByPosition(candidates)
	return candidates[levelrand.Intn(len(candidates))]
}

func addHazardHint(g *state.Game, cell *world.Cell, info entities.HazardInfo) {
	if cell.Name == "Corridor" {
		g.AddHint(fmt.Sprintf("A %s blocks a corridor passage", info.Name))
		return
	}
	g.AddHint(fmt.Sprintf("A %s blocks passage through %s", info.Name, renderer.StyledCell(cell.Name)))
}
