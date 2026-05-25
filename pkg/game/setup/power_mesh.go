package setup

import (
	"fmt"
	"sort"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// CanTraverseCellForPowerMesh reports whether terminal control power may pass through a cell.
// Requires powered doors on door cells, blocks locked doors and open (non-conducting) relays.
func CanTraverseCellForPowerMesh(g *state.Game, cell *world.Cell) bool {
	if g == nil || cell == nil || !cell.Room {
		return false
	}
	if gameworld.RelayBlocksMesh(cell) {
		return false
	}
	if gameworld.HasLockedDoor(cell) {
		return false
	}
	if gameworld.HasDoor(cell) {
		roomName := gameworld.GetGameData(cell).Door.RoomName
		if roomName == "" || !g.RoomDoorsPowered[roomName] {
			return false
		}
	}
	if cell.Name != "" && cell.Name != "Corridor" && !g.RoomDoorsPowered[cell.Name] {
		return false
	}
	return true
}

// RoomsReachableInPowerMeshExcluding is like RoomsReachableInPowerMesh but does not traverse cells in excludeRoom.
func RoomsReachableInPowerMeshExcluding(g *state.Game, startCell *world.Cell, excludeRoom string) []string {
	if excludeRoom == "" {
		return RoomsReachableInPowerMesh(g, startCell)
	}
	if g == nil || g.Grid == nil || startCell == nil {
		return nil
	}
	if !CanTraverseCellForPowerMesh(g, startCell) {
		return nil
	}

	visited := mapset.New[*world.Cell]()
	rooms := make(map[string]bool)
	queue := []*world.Cell{startCell}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == nil || visited.Has(cur) {
			continue
		}
		if cur.Name == excludeRoom {
			continue
		}
		if !CanTraverseCellForPowerMesh(g, cur) {
			continue
		}
		visited.Put(cur)
		if cur.Name != "" && cur.Name != "Corridor" {
			rooms[cur.Name] = true
		}
		for _, n := range cur.GetNeighbors() {
			if n == nil || !n.Room || visited.Has(n) || n.Name == excludeRoom {
				continue
			}
			if CanTraverseCellForPowerMesh(g, n) {
				queue = append(queue, n)
			}
		}
	}

	names := make([]string, 0, len(rooms))
	for name := range rooms {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RoomsReachableInPowerMesh returns sorted room names reachable from startCell via powered doors and closed relays.
func RoomsReachableInPowerMesh(g *state.Game, startCell *world.Cell) []string {
	if g == nil || g.Grid == nil || startCell == nil {
		return nil
	}
	if !CanTraverseCellForPowerMesh(g, startCell) {
		return nil
	}

	visited := mapset.New[*world.Cell]()
	rooms := make(map[string]bool)
	queue := []*world.Cell{startCell}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == nil || visited.Has(cur) {
			continue
		}
		if !CanTraverseCellForPowerMesh(g, cur) {
			continue
		}
		visited.Put(cur)
		if cur.Name != "" && cur.Name != "Corridor" {
			rooms[cur.Name] = true
		}
		for _, n := range cur.GetNeighbors() {
			if n == nil || !n.Room || visited.Has(n) {
				continue
			}
			if CanTraverseCellForPowerMesh(g, n) {
				queue = append(queue, n)
			}
		}
	}

	names := make([]string, 0, len(rooms))
	for name := range rooms {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RoomsReachableFromPoweredTerminals unions mesh reachability from every powered maintenance terminal.
func RoomsReachableFromPoweredTerminals(g *state.Game) []string {
	if g == nil || g.Grid == nil {
		return nil
	}
	roomSet := make(map[string]bool)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.MaintenanceTerm == nil || !data.MaintenanceTerm.Powered {
			return
		}
		for _, name := range RoomsReachableInPowerMesh(g, cell) {
			roomSet[name] = true
		}
	})
	names := make([]string, 0, len(roomSet))
	for name := range roomSet {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SelectableRoomsForTerminal returns rooms the player may target from a maintenance terminal:
// union of mesh-reachable rooms and spec-adjacent rooms (§2.2: own room + directly adjacent).
func SelectableRoomsForTerminal(g *state.Game, grid *world.Grid, terminalRoom string) []string {
	if g == nil || grid == nil || terminalRoom == "" {
		return GetAdjacentRoomNames(grid, terminalRoom)
	}
	roomSet := make(map[string]bool)
	hasPoweredTerminal := false
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name != terminalRoom {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.MaintenanceTerm == nil || !data.MaintenanceTerm.Powered {
			return
		}
		hasPoweredTerminal = true
		for _, name := range RoomsReachableInPowerMesh(g, cell) {
			roomSet[name] = true
		}
	})
	for _, name := range GetAdjacentRoomNames(grid, terminalRoom) {
		roomSet[name] = true
	}
	if !hasPoweredTerminal && len(roomSet) == 0 {
		return GetAdjacentRoomNames(grid, terminalRoom)
	}
	names := make([]string, 0, len(roomSet))
	for name := range roomSet {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RestoreTerminalsInRooms powers unpowered maintenance terminals in the given rooms.
func RestoreTerminalsInRooms(g *state.Game, roomSet map[string]bool) (restored int, message string) {
	if g == nil || g.Grid == nil || len(roomSet) == 0 {
		return 0, "No target rooms"
	}
	g.Grid.ForEachCell(func(row, col int, c *world.Cell) {
		if c == nil || !c.Room || !roomSet[c.Name] {
			return
		}
		data := gameworld.GetGameData(c)
		if data.MaintenanceTerm == nil || data.MaintenanceTerm.Powered {
			return
		}
		data.MaintenanceTerm.Powered = true
		restored++
	})
	if restored > 0 {
		return restored, fmt.Sprintf("Restored power to %d terminal(s) via routing mesh", restored)
	}
	return 0, "No unpowered terminals on routing mesh"
}
