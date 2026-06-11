// Package setup — initial progression reachability (keycards, generators, chokepoints).
package setup

import (
	"strings"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func isKeycardItemName(name string) bool {
	return strings.Contains(name, "Keycard")
}

// InitialReachableCellsWithExtraBlock returns init-reachable cells when candidate is also blocked.
func InitialReachableCellsWithExtraBlock(g *state.Game, extraBlocked *world.Cell) *mapset.Set[*world.Cell] {
	empty := mapset.New[*world.Cell]()
	if g == nil || g.Grid == nil {
		return &empty
	}
	entry := PlayerEntryCell(g)
	if entry == nil {
		return &empty
	}
	reachable := mapset.New[*world.Cell]()
	queue := []*world.Cell{entry}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == nil || !cur.Room || reachable.Has(cur) {
			continue
		}
		if extraBlocked != nil && cur == extraBlocked {
			continue
		}
		if cur != entry {
			ok, _ := CanEnterCellAtInit(g, cur)
			if !ok {
				continue
			}
		}
		reachable.Put(cur)
		for _, n := range cur.GetNeighbors() {
			if n != nil && n.Room && !reachable.Has(n) {
				queue = append(queue, n)
			}
		}
	}
	return &reachable
}

func cellAccessibleFromReachable(reachable *mapset.Set[*world.Cell], cell *world.Cell) bool {
	if cell == nil || reachable == nil {
		return false
	}
	if reachable.Has(cell) {
		return true
	}
	for _, n := range cell.GetNeighbors() {
		if reachable.Has(n) {
			return true
		}
	}
	return false
}

func reachableNamedRooms(reachable *mapset.Set[*world.Cell]) map[string]bool {
	rooms := make(map[string]bool)
	if reachable == nil {
		return rooms
	}
	reachable.Each(func(c *world.Cell) {
		if c.Name != "" && c.Name != "Corridor" {
			rooms[c.Name] = true
		}
	})
	return rooms
}

type keycardLocation struct {
	cell        *world.Cell
	inFurniture bool
}

func keycardLocations(g *state.Game) []keycardLocation {
	var out []keycardLocation
	if g == nil || g.Grid == nil {
		return out
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			if item != nil && isKeycardItemName(item.Name) {
				out = append(out, keycardLocation{cell: cell, inFurniture: false})
			}
		})
		data := gameworld.GetGameData(cell)
		if data.Furniture != nil && data.Furniture.ContainedItem != nil &&
			isKeycardItemName(data.Furniture.ContainedItem.Name) {
			out = append(out, keycardLocation{cell: cell, inFurniture: true})
		}
	})
	return out
}

func keycardsAccessible(g *state.Game, reachable *mapset.Set[*world.Cell]) bool {
	for _, loc := range keycardLocations(g) {
		if loc.inFurniture {
			if !cellAccessibleFromReachable(reachable, loc.cell) {
				return false
			}
			continue
		}
		if loc.cell == nil || reachable == nil || !reachable.Has(loc.cell) {
			return false
		}
	}
	return true
}

func generatorsAccessible(g *state.Game, reachable *mapset.Set[*world.Cell]) bool {
	if g == nil || g.Grid == nil {
		return true
	}
	allOK := true
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if !allOK || cell == nil {
			return
		}
		if gameworld.GetGameData(cell).Generator == nil {
			return
		}
		if !cellAccessibleFromReachable(reachable, cell) {
			allOK = false
		}
	})
	return allOK
}

// keycardsStillAccessible reports whether every init-reachable keycard stays reachable when
// extraBlocked is also treated as impassable. Keycards already unreachable at init are ignored
// so one stranded keycard does not reject all blocking placement (I3 safety net relocates them).
func keycardsStillAccessible(g *state.Game, base, with *mapset.Set[*world.Cell]) bool {
	for _, loc := range keycardLocations(g) {
		if loc.inFurniture {
			if !cellAccessibleFromReachable(base, loc.cell) {
				continue
			}
			if !cellAccessibleFromReachable(with, loc.cell) {
				return false
			}
			continue
		}
		if loc.cell == nil || base == nil || !base.Has(loc.cell) {
			continue
		}
		if with == nil || !with.Has(loc.cell) {
			return false
		}
	}
	return true
}

// InitProgressPreserved reports whether blocking candidate still leaves init keycards reachable
// and does not cut off init-reachable rooms. Generators and batteries may lie behind unpowered doors.
func InitProgressPreserved(g *state.Game, candidate *world.Cell) bool {
	if g == nil || candidate == nil {
		return true
	}
	base := InitialReachableCells(g)
	with := InitialReachableCellsWithExtraBlock(g, candidate)
	if !keycardsStillAccessible(g, base, with) {
		return false
	}
	baseRooms := reachableNamedRooms(base)
	withRooms := reachableNamedRooms(with)
	for room := range baseRooms {
		if !withRooms[room] {
			return false
		}
	}
	return true
}

func pickStartRoomFloorCell(g *state.Game) *world.Cell {
	if g == nil || g.Grid == nil {
		return nil
	}
	entry := PlayerEntryCell(g)
	if entry == nil {
		return nil
	}
	startRoom := entry.Name
	var fallback *world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name != startRoom || cell.ExitCell {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Generator != nil || data.Door != nil || data.Furniture != nil ||
			data.Terminal != nil || data.Puzzle != nil || data.MaintenanceTerm != nil ||
			data.Hazard != nil || data.HazardControl != nil || data.RepairDevice != nil ||
			data.RepairBlocker != nil || cell.ItemsOnFloor.Size() > 0 {
			return
		}
		if fallback == nil {
			fallback = cell
		}
		if cell == entry {
			fallback = cell
		}
	})
	return fallback
}

// EnsureKeycardReachability moves init-unreachable keycards into the initial reachable area
// (I3 safety net), preferring non-start-room cells so keycards are discovered through play.
func EnsureKeycardReachability(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	reachable := InitialReachableCells(g)
	avoid := keycardRelocationAvoidSet(g)
	entryRoom := ""
	entry := PlayerEntryCell(g)
	if entry != nil {
		entryRoom = entry.Name
	}
	for _, loc := range keycardLocations(g) {
		accessible := loc.cell != nil && reachable.Has(loc.cell)
		inEntryRoom := entryRoom != "" && loc.cell != nil && loc.cell.Name == entryRoom
		inShaftOffEntry := loc.cell != nil && loc.cell.Name == generator.ShaftRoomName && loc.cell != entry
		if accessible && !(inEntryRoom && loc.cell != entry) && !inShaftOffEntry {
			continue
		}
		landing := pickKeycardRelocationCell(g, reachable, &avoid, !accessible)
		if landing == nil {
			landing = pickAnyReachableKeycardCell(g, reachable, entry)
		}
		if landing == nil {
			continue
		}
		if loc.inFurniture {
			data := gameworld.GetGameData(loc.cell)
			if data.Furniture == nil || data.Furniture.ContainedItem == nil {
				continue
			}
			item := data.Furniture.ContainedItem
			data.Furniture.ContainedItem = nil
			landing.ItemsOnFloor.Put(item)
			avoid.Put(landing)
			continue
		}
		var toMove *world.Item
		loc.cell.ItemsOnFloor.Each(func(item *world.Item) {
			if toMove == nil && item != nil && isKeycardItemName(item.Name) {
				toMove = item
			}
		})
		if toMove != nil {
			loc.cell.ItemsOnFloor.Remove(toMove)
			landing.ItemsOnFloor.Put(toMove)
			avoid.Put(landing)
		}
	}
}

func keycardRelocationAvoidSet(g *state.Game) mapset.Set[*world.Cell] {
	avoid := mapset.New[*world.Cell]()
	if g == nil || g.Grid == nil {
		return avoid
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Generator != nil || data.Door != nil || data.Furniture != nil ||
			data.Terminal != nil || data.Puzzle != nil || data.MaintenanceTerm != nil ||
			data.Hazard != nil || data.HazardControl != nil || data.RepairDevice != nil ||
			data.RepairBlocker != nil || cell.ItemsOnFloor.Size() > 0 || cell.ExitCell {
			avoid.Put(cell)
		}
	})
	return avoid
}

func pickKeycardRelocationCell(g *state.Game, reachable *mapset.Set[*world.Cell], avoid *mapset.Set[*world.Cell], allowStartRoomFallback bool) *world.Cell {
	if g == nil || g.Grid == nil || reachable == nil {
		return nil
	}
	entry := PlayerEntryCell(g)
	entryRoom := ""
	if entry != nil {
		entryRoom = entry.Name
	}
	var candidates []*world.Cell
	reachable.Each(func(cell *world.Cell) {
		if cell == nil || !cell.Room || avoid.Has(cell) || cell == entry {
			return
		}
		if entryRoom != "" && cell.Name == entryRoom {
			return
		}
		if !reachable.Has(cell) {
			return
		}
		candidates = append(candidates, cell)
	})
	if len(candidates) > 0 {
		SortCellsByPosition(candidates)
		return candidates[0]
	}
	if cell := pickAdjacentToReachableKeycardCell(g, reachable, avoid, entry); cell != nil {
		return cell
	}
	if !allowStartRoomFallback {
		return nil
	}
	if cell := pickStartRoomNonSpawnFloorCell(g, reachable, avoid); cell != nil {
		return cell
	}
	return pickStartRoomFloorCell(g)
}

func pickAnyReachableKeycardCell(g *state.Game, reachable *mapset.Set[*world.Cell], entry *world.Cell) *world.Cell {
	if g == nil || reachable == nil {
		return nil
	}
	var candidates []*world.Cell
	reachable.Each(func(cell *world.Cell) {
		if cell == nil || !cell.Room || cell == entry || cell.ExitCell {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Generator != nil || data.Door != nil || data.Furniture != nil ||
			data.Terminal != nil || data.Puzzle != nil || data.MaintenanceTerm != nil ||
			data.Hazard != nil || data.HazardControl != nil || data.RepairDevice != nil ||
			data.RepairBlocker != nil {
			return
		}
		candidates = append(candidates, cell)
	})
	if len(candidates) == 0 {
		return nil
	}
	SortCellsByPosition(candidates)
	return candidates[0]
}

func pickAdjacentToReachableKeycardCell(g *state.Game, reachable *mapset.Set[*world.Cell], avoid *mapset.Set[*world.Cell], entry *world.Cell) *world.Cell {
	if g == nil || reachable == nil || reachable.Size() == 0 {
		return nil
	}
	reach := InitialReachableCells(g)
	var candidates []*world.Cell
	reach.Each(func(cell *world.Cell) {
		if cell == nil || !cell.Room || avoid.Has(cell) || cell == entry {
			return
		}
		candidates = append(candidates, cell)
	})
	if len(candidates) == 0 {
		reach.Each(func(base *world.Cell) {
			if base == nil {
				return
			}
			for _, n := range base.GetNeighbors() {
				if n == nil || !n.Room || avoid.Has(n) || n == entry || !reach.Has(n) {
					continue
				}
				candidates = append(candidates, n)
			}
		})
	}
	if len(candidates) == 0 {
		return nil
	}
	SortCellsByPosition(candidates)
	return candidates[0]
}

func pickStartRoomNonSpawnFloorCell(g *state.Game, reachable *mapset.Set[*world.Cell], avoid *mapset.Set[*world.Cell]) *world.Cell {
	entry := PlayerEntryCell(g)
	if entry == nil {
		return nil
	}
	var candidates []*world.Cell
	reachable.Each(func(cell *world.Cell) {
		if cell == nil || !cell.Room || cell == entry || cell.Name != entry.Name || avoid.Has(cell) {
			return
		}
		candidates = append(candidates, cell)
	})
	if len(candidates) == 0 {
		return nil
	}
	SortCellsByPosition(candidates)
	return candidates[0]
}

// generatorLocationOK reports whether an existing generator at cell preserves exit/nav paths
// and init keycard access. Generators may be unreachable at level start until power is restored.
func generatorLocationOK(g *state.Game, cell *world.Cell) bool {
	if g == nil || cell == nil || gameworld.GetGameData(cell).Generator == nil {
		return true
	}
	if !ExitReachableWhenCompletable(g, cell) {
		return false
	}
	if !BlockingPlacementPreservesNavAccess(g, cell) {
		return false
	}
	reachable := InitialReachableCells(g)
	return keycardsAccessible(g, reachable)
}

// EnsureGeneratorSafePlacement relocates generators that block init progression (e.g. corridor chokepoints).
func EnsureGeneratorSafePlacement(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	type genAt struct {
		cell *world.Cell
		gen  *entities.Generator
	}
	var bad []genAt
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen == nil || generatorLocationOK(g, cell) {
			return
		}
		bad = append(bad, genAt{cell: cell, gen: gen})
	})
	if len(bad) == 0 {
		return
	}
	avoid := mapset.New[*world.Cell]()
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Generator != nil {
			avoid.Put(cell)
		}
	})
	routingOrigin := PlayerEntryCell(g)
	for _, genEntry := range bad {
		gameworld.GetGameData(genEntry.cell).Generator = nil
		avoid.Remove(genEntry.cell)
		avoid.Put(genEntry.cell) // never relocate back onto the same chokepoint cell
		roomName := genEntry.cell.Name
		var newCell *world.Cell
		candidates := []string{roomName}
		for _, adj := range GetAdjacentRoomNames(g.Grid, roomName) {
			candidates = append(candidates, adj)
		}
		for _, candidateRoom := range candidates {
			cell := findValidGeneratorCell(g, candidateRoom, routingOrigin, &avoid)
			if cell == nil {
				continue
			}
			gameworld.GetGameData(cell).Generator = genEntry.gen
			if generatorLocationOK(g, cell) {
				newCell = cell
				break
			}
			gameworld.GetGameData(cell).Generator = nil
		}
		if newCell == nil {
			for _, candidateRoom := range collectUniqueRoomNames(g.Grid) {
				cell := findGeneratorCellInRoom(g, candidateRoom, routingOrigin, &avoid, false)
				if cell == nil {
					cell = findGeneratorCellInRoom(g, candidateRoom, routingOrigin, &avoid, true)
				}
				if cell == nil {
					continue
				}
				gameworld.GetGameData(cell).Generator = genEntry.gen
				if generatorLocationOK(g, cell) {
					newCell = cell
					break
				}
				gameworld.GetGameData(cell).Generator = nil
			}
		}
		if newCell != nil {
			gameworld.GetGameData(newCell).Generator = genEntry.gen
			avoid.Put(newCell)
			continue
		}
		// Last resort: restore at original cell so the level keeps required generator count.
		gameworld.GetGameData(genEntry.cell).Generator = genEntry.gen
		avoid.Put(genEntry.cell)
	}
}

// EnsureInitProgressReachability applies keycard and generator placement safety nets.
func EnsureInitProgressReachability(g *state.Game) {
	EnsureLiftShaftEntryClearance(g)
	EnsureGeneratorSafePlacement(g)
	EnsureKeycardReachability(g)
}
