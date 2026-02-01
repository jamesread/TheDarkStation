// Package setup provides level setup functionality for The Dark Station.
package setup

import (
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// getReachableCellsBlockingDoorsInto returns all cells reachable from start by BFS,
// treating as impassable: (1) any door cell that leads into blockRoomName, and
// (2) any locked door cell (player cannot pass without keycard, so do not count
// as reachable when deciding if R is a gatekeeper).
// Used to compute "reachable without entering R" for solvability checks.
func getReachableCellsBlockingDoorsInto(grid *world.Grid, start *world.Cell, blockRoomName string) *mapset.Set[*world.Cell] {
	reachable := mapset.New[*world.Cell]()
	queue := []*world.Cell{start}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == nil || !current.Room || reachable.Has(current) {
			continue
		}

		// Treat door cells that lead into blockRoomName as blocked
		if gameworld.HasDoor(current) {
			rName := gameworld.GetGameData(current).Door.RoomName
			if rName == blockRoomName {
				continue
			}
			// Treat locked doors as blocked: player cannot pass without keycard,
			// so paths through locked doors must not count as "reachable without R".
			if gameworld.HasLockedDoor(current) {
				continue
			}
		}

		reachable.Put(current)

		neighbors := []*world.Cell{current.North, current.East, current.South, current.West}
		for _, n := range neighbors {
			if n != nil && n.Room && !reachable.Has(n) {
				queue = append(queue, n)
			}
		}
	}

	return &reachable
}

// roomsWithDoors returns the set of room names that have at least one door (cell with Door leading to that room).
func roomsWithDoors(grid *world.Grid) map[string]bool {
	out := make(map[string]bool)
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !gameworld.HasDoor(cell) {
			return
		}
		roomName := gameworld.GetGameData(cell).Door.RoomName
		out[roomName] = true
	})
	return out
}

// EnsureSolvabilityDoorPower fixes room power so that no gatekeeper room creates a control-dependency deadlock.
// Must be called after maintenance terminals are placed (so we know which rooms have a terminal).
// A gatekeeper room R is one through which every path from start to exit goes. If R's doors are unpowered and
// no room adjacent to R that has a maintenance terminal is reachable from start without entering R, the player
// could never power R's doors. This function powers R's doors initially in that case.
func EnsureSolvabilityDoorPower(g *state.Game) {
	if g.Grid == nil {
		return
	}
	start := g.Grid.StartCell()
	exit := g.Grid.ExitCell()
	if start == nil || exit == nil {
		return
	}
	startRoomName := start.Name
	if startRoomName == "" {
		return
	}

	// Which rooms have at least one maintenance terminal (only those can control other rooms' power)
	roomsWithTerminal := make(map[string]bool)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name == "" {
			return
		}
		if gameworld.GetGameData(cell).MaintenanceTerm != nil {
			roomsWithTerminal[cell.Name] = true
		}
	})

	for roomName := range roomsWithDoors(g.Grid) {
		// Start room is already powered; nothing to fix
		if roomName == startRoomName {
			continue
		}
		if g.RoomDoorsPowered[roomName] {
			// Already powered; no deadlock possible
			continue
		}

		// Reachable set when we block all doors into this room
		reachableWithoutR := getReachableCellsBlockingDoorsInto(g.Grid, start, roomName)

		// Is the exit reachable without entering R? If yes, R is not a gatekeeper.
		if reachableWithoutR.Has(exit) {
			continue
		}

		// R is a gatekeeper: every path to exit goes through R. Check if any room adjacent to R
		// that has a maintenance terminal is reachable without entering R (so the player can
		// power R's doors from that terminal).
		adjacentSet := make(map[string]bool)
		for _, q := range GetAdjacentRoomNames(g.Grid, roomName) {
			if q != roomName && roomsWithTerminal[q] {
				adjacentSet[q] = true
			}
		}
		hasReachableAdjacent := false
		reachableWithoutR.Each(func(cell *world.Cell) {
			if cell != nil && cell.Room && cell.Name != "" && adjacentSet[cell.Name] {
				hasReachableAdjacent = true
			}
		})

		if !hasReachableAdjacent {
			// Deadlock: R is gatekeeper, unpowered, and no adjacent room is reachable without entering R.
			// Power R's doors so the level is solvable.
			g.RoomDoorsPowered[roomName] = true
		}
	}
}
