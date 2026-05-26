// Package setup — initial progression reachability (keycards, generators, chokepoints).
package setup

import (
	"strings"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
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
	start := g.Grid.StartCell()
	if start == nil {
		return &empty
	}
	reachable := mapset.New[*world.Cell]()
	queue := []*world.Cell{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == nil || !cur.Room || reachable.Has(cur) {
			continue
		}
		if extraBlocked != nil && cur == extraBlocked {
			continue
		}
		if cur != start {
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
		if !cellAccessibleFromReachable(reachable, loc.cell) {
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
		if !cellAccessibleFromReachable(base, loc.cell) {
			continue
		}
		if !cellAccessibleFromReachable(with, loc.cell) {
			return false
		}
	}
	return true
}

// generatorsStillAccessible reports whether every init-reachable generator stays interactable
// when extraBlocked is also impassable. Generators already unreachable at init are ignored here;
// EnsureGeneratorSafePlacement relocates those after placement.
func generatorsStillAccessible(g *state.Game, base, with *mapset.Set[*world.Cell]) bool {
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
		if !cellAccessibleFromReachable(base, cell) {
			return
		}
		if !cellAccessibleFromReachable(with, cell) {
			allOK = false
		}
	})
	return allOK
}

// generatorAdjacentReachableAtInit reports whether a generator at candidate would have at least
// one adjacent stand tile reachable from start at level init (generators block their own cell).
func generatorAdjacentReachableAtInit(g *state.Game, candidate *world.Cell) bool {
	if g == nil || candidate == nil {
		return false
	}
	with := InitialReachableCellsWithExtraBlock(g, candidate)
	return cellAccessibleFromReachable(with, candidate)
}

// InitProgressPreserved reports whether blocking candidate still leaves init keycards and generators reachable.
func InitProgressPreserved(g *state.Game, candidate *world.Cell) bool {
	if g == nil || candidate == nil {
		return true
	}
	base := InitialReachableCells(g)
	with := InitialReachableCellsWithExtraBlock(g, candidate)
	if !keycardsStillAccessible(g, base, with) {
		return false
	}
	if !generatorsStillAccessible(g, base, with) {
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
	start := g.Grid.StartCell()
	if start == nil {
		return nil
	}
	startRoom := start.Name
	var fallback *world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name != startRoom || cell.ExitCell {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Generator != nil || data.Door != nil || data.Furniture != nil ||
			data.Terminal != nil || data.Puzzle != nil || data.MaintenanceTerm != nil {
			return
		}
		if fallback == nil {
			fallback = cell
		}
		if cell == start {
			fallback = cell
		}
	})
	return fallback
}

// EnsureKeycardReachability moves init-unreachable keycards into the start room (I3 safety net).
func EnsureKeycardReachability(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	reachable := InitialReachableCells(g)
	landing := pickStartRoomFloorCell(g)
	if landing == nil {
		return
	}
	for _, loc := range keycardLocations(g) {
		if cellAccessibleFromReachable(reachable, loc.cell) {
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
		}
	}
}

// generatorLocationOK reports whether an existing generator at cell preserves init keycard
// and generator reachability (stricter than InitProgressPreserved, which ignores already-stranded items).
func generatorLocationOK(g *state.Game, cell *world.Cell) bool {
	if g == nil || cell == nil || gameworld.GetGameData(cell).Generator == nil {
		return true
	}
	if !generatorAdjacentReachableAtInit(g, cell) {
		return false
	}
	reachable := InitialReachableCells(g)
	return keycardsAccessible(g, reachable) && generatorsAccessible(g, reachable)
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
	start := g.Grid.StartCell()
	for _, entry := range bad {
		gameworld.GetGameData(entry.cell).Generator = nil
		avoid.Remove(entry.cell)
		roomName := entry.cell.Name
		newCell := findValidGeneratorCell(g, roomName, start, &avoid)
		if newCell == nil {
			for _, adj := range GetAdjacentRoomNames(g.Grid, roomName) {
				newCell = findValidGeneratorCell(g, adj, start, &avoid)
				if newCell != nil {
					break
				}
			}
		}
		if newCell != nil {
			gameworld.GetGameData(newCell).Generator = entry.gen
			avoid.Put(newCell)
			continue
		}
		// Last resort: restore at original cell so the level keeps required generator count.
		gameworld.GetGameData(entry.cell).Generator = entry.gen
		avoid.Put(entry.cell)
	}
}

// EnsureInitProgressReachability applies keycard, generator, and battery placement safety nets.
func EnsureInitProgressReachability(g *state.Game) {
	EnsureGeneratorSafePlacement(g)
	EnsureKeycardReachability(g)
}

// EnsureBatteryReachability moves init-unreachable floor batteries onto reachable tiles (I3 safety net).
func EnsureBatteryReachability(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	reachable := InitialReachableCells(g)
	landing := pickReachableItemLandingCell(g, reachable)
	if landing == nil {
		return
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		var toMove []*world.Item
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			if item != nil && item.Name == "Battery" {
				toMove = append(toMove, item)
			}
		})
		for _, item := range toMove {
			if reachable.Has(cell) {
				continue
			}
			cell.ItemsOnFloor.Remove(item)
			landing.ItemsOnFloor.Put(item)
		}
	})
}

func pickReachableItemLandingCell(g *state.Game, reachable *mapset.Set[*world.Cell]) *world.Cell {
	if g == nil || g.Grid == nil || reachable == nil {
		return nil
	}
	if start := g.Grid.StartCell(); start != nil && reachable.Has(start) && isValidForFloorItem(g, start, nil) {
		return start
	}
	var fallback *world.Cell
	reachable.Each(func(cell *world.Cell) {
		if fallback != nil || !isValidForFloorItem(g, cell, nil) {
			return
		}
		fallback = cell
	})
	return fallback
}
