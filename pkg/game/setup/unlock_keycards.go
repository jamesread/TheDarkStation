package setup

import (
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// KeycardDropCell returns the cell where an unlock keycard should appear. When the
// registered repair cell still hosts a movement-blocking device, the keycard drops
// on an adjacent walkable cell so the player can pick it up after completion.
func KeycardDropCell(g *state.Game, registered *world.Cell) *world.Cell {
	if registered == nil {
		return nil
	}
	if keycardDropCellWalkable(g, registered) {
		return registered
	}
	entry := PlayerEntryCell(g)
	dist := initPathDistances(g, entry)
	var best *world.Cell
	bestDist := -1
	for _, n := range registered.GetNeighbors() {
		if n == nil || !keycardDropCellWalkable(g, n) {
			continue
		}
		d := dist[n]
		if d > bestDist {
			bestDist = d
			best = n
		}
	}
	if best != nil {
		return best
	}
	return registered
}

func keycardDropCellWalkable(g *state.Game, cell *world.Cell) bool {
	if cell == nil || !cell.Room || cell == PlayerEntryCell(g) || cell.ExitCell {
		return false
	}
	data := gameworld.GetGameData(cell)
	if data.Generator != nil || data.Door != nil || data.Terminal != nil || data.Puzzle != nil ||
		data.Furniture != nil || data.Hazard != nil || data.HazardControl != nil ||
		data.MaintenanceTerm != nil || data.RepairBlocker != nil ||
		gameworld.RepairDeviceBlocksMovement(cell) {
		return false
	}
	return true
}

// DropPendingUnlockKeycards spawns registered unlock keycards onto pickup-reachable cells.
func DropPendingUnlockKeycards(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.PendingUnlockKeycard == "" {
			return
		}
		name := data.PendingUnlockKeycard
		data.PendingUnlockKeycard = ""
		dropCell := KeycardDropCell(g, cell)
		if dropCell == nil {
			dropCell = cell
		}
		already := false
		dropCell.ItemsOnFloor.Each(func(item *world.Item) {
			if item != nil && item.Name == name {
				already = true
			}
		})
		if !already {
			dropCell.ItemsOnFloor.Put(world.NewItem(name))
		}
	})
}

func initPathDistances(g *state.Game, from *world.Cell) map[*world.Cell]int {
	if g == nil || from == nil {
		return nil
	}
	visited := map[*world.Cell]int{from: 0}
	queue := []*world.Cell{from}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, n := range c.GetNeighbors() {
			if n == nil || !n.Room {
				continue
			}
			if _, ok := visited[n]; ok {
				continue
			}
			if n != from {
				ok, _ := CanEnterCellAtInit(g, n)
				if !ok {
					continue
				}
			}
			visited[n] = visited[c] + 1
			queue = append(queue, n)
		}
	}
	return visited
}

// UnlockKeycardOnFloor reports whether name appears on any cell floor.
func UnlockKeycardOnFloor(g *state.Game, name string) bool {
	found := false
	if g == nil || g.Grid == nil || name == "" {
		return false
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if found || cell == nil {
			return
		}
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			if item != nil && item.Name == name {
				found = true
			}
		})
	})
	return found
}

// PendingUnlockKeycardRegistered reports whether name is registered on a repair cell.
func PendingUnlockKeycardRegistered(g *state.Game, name string) bool {
	found := false
	if g == nil || g.Grid == nil || name == "" {
		return false
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if found || cell == nil {
			return
		}
		if gameworld.GetGameData(cell).PendingUnlockKeycard == name {
			found = true
		}
	})
	return found
}
