// Package levelgen provides level generation utilities for placing entities.
package levelgen

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// RoomEntryPoints represents all corridor cells that provide access to a specific room
type RoomEntryPoints struct {
	RoomName   string
	EntryCells []*world.Cell
}

// CollectReachableRooms collects all reachable rooms from a starting cell using BFS
func CollectReachableRooms(start *world.Cell, avoid *mapset.Set[*world.Cell]) []*world.Cell {
	var rooms []*world.Cell
	visited := mapset.New[*world.Cell]()
	queue := []*world.Cell{start}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == nil || !current.Room || visited.Has(current) {
			continue
		}

		visited.Put(current)

		if !avoid.Has(current) {
			rooms = append(rooms, current)
		}

		// Add neighbors to queue
		neighbors := []*world.Cell{current.North, current.East, current.South, current.West}
		for _, n := range neighbors {
			if n != nil && n.Room && !visited.Has(n) {
				queue = append(queue, n)
			}
		}
	}

	return rooms
}

// ManhattanDistance calculates the Manhattan distance between two cells
func ManhattanDistance(a, b *world.Cell) int {
	rowDist := a.Row - b.Row
	colDist := a.Col - b.Col
	if rowDist < 0 {
		rowDist = -rowDist
	}
	if colDist < 0 {
		colDist = -colDist
	}
	return rowDist + colDist
}

// FindRoom finds a random reachable room at an appropriate distance based on level
func FindRoom(g *state.Game, start *world.Cell, avoid *mapset.Set[*world.Cell]) *world.Cell {
	rooms := CollectReachableRooms(start, avoid)

	if len(rooms) == 0 {
		return start
	}

	// Calculate minimum distance based on level
	// Level 1: min 2, Level 5: min 6, Level 10: min 11
	minDistance := 1 + g.Level

	// Filter rooms by minimum distance
	var farRooms []*world.Cell
	for _, room := range rooms {
		if ManhattanDistance(start, room) >= minDistance {
			farRooms = append(farRooms, room)
		}
	}

	// If no rooms are far enough, use all rooms and pick the furthest ones
	if len(farRooms) == 0 {
		farRooms = rooms
		if len(farRooms) > 2 {
			// Simple selection: keep only rooms in the further half
			var maxDist int
			for _, room := range rooms {
				d := ManhattanDistance(start, room)
				if d > maxDist {
					maxDist = d
				}
			}
			threshold := maxDist / 2
			farRooms = nil
			for _, room := range rooms {
				if ManhattanDistance(start, room) >= threshold {
					farRooms = append(farRooms, room)
				}
			}
			if len(farRooms) == 0 {
				farRooms = rooms
			}
		}
	}

	// Pick a random room from the candidates
	return farRooms[rand.Intn(len(farRooms))]
}

// FindRoomEntryPoints finds all corridor cells that serve as entry points to each room
// Groups them by room so we can door ALL entries to fully block a room
func FindRoomEntryPoints(grid *world.Grid) map[string]*RoomEntryPoints {
	entries := make(map[string]*RoomEntryPoints)
	seenCells := mapset.New[string]() // Track cells we've already assigned

	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		// Only look at corridor cells
		if !cell.Room || cell.Name != "Corridor" {
			return
		}

		// Check adjacent cells for rooms (not corridors)
		neighbors := []*world.Cell{cell.North, cell.East, cell.South, cell.West}
		for _, neighbor := range neighbors {
			if neighbor != nil && neighbor.Room && neighbor.Name != "Corridor" && neighbor.Name != "" {
				roomName := neighbor.Name
				cellKey := fmt.Sprintf("%d,%d-%s", cell.Row, cell.Col, roomName)

				if seenCells.Has(cellKey) {
					continue
				}
				seenCells.Put(cellKey)

				// Initialize entry points for this room if needed
				if entries[roomName] == nil {
					entries[roomName] = &RoomEntryPoints{
						RoomName:   roomName,
						EntryCells: make([]*world.Cell, 0),
					}
				}

				// Add this corridor cell as an entry point
				// Only add if not already in the list (a corridor might touch multiple cells of same room)
				alreadyAdded := false
				for _, existing := range entries[roomName].EntryCells {
					if existing == cell {
						alreadyAdded = true
						break
					}
				}
				if !alreadyAdded {
					entries[roomName].EntryCells = append(entries[roomName].EntryCells, cell)
				}
			}
		}
	})

	return entries
}

// GetReachableCells returns all cells reachable from start without passing through locked doors
func GetReachableCells(grid *world.Grid, start *world.Cell, lockedDoors *mapset.Set[*world.Cell]) *mapset.Set[*world.Cell] {
	return GetReachableCellsExcluding(grid, start, lockedDoors, nil)
}

// GetReachableCellsExcluding returns all cells reachable from start without passing through locked doors or the exclude cell.
// Exclude is treated as impassable (e.g. to test if a cell is an articulation point). Pass nil for no exclusion.
func GetReachableCellsExcluding(grid *world.Grid, start *world.Cell, lockedDoors *mapset.Set[*world.Cell], exclude *world.Cell) *mapset.Set[*world.Cell] {
	reachable := mapset.New[*world.Cell]()
	queue := []*world.Cell{start}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == nil || !current.Room || reachable.Has(current) {
			continue
		}

		if current == exclude {
			continue // treat as impassable
		}

		// Can't pass through locked doors
		if lockedDoors.Has(current) {
			continue
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

// IsArticulationPoint returns true if blocking this cell would disconnect the reachable set (i.e. the cell is a chokepoint).
// Hazard controls and other blocking entities should not be placed on articulation points.
func IsArticulationPoint(grid *world.Grid, start *world.Cell, cell *world.Cell, lockedDoors *mapset.Set[*world.Cell]) bool {
	fullReach := GetReachableCells(grid, start, lockedDoors)
	if !fullReach.Has(cell) {
		return false
	}
	reachWithoutCell := GetReachableCellsExcluding(grid, start, lockedDoors, cell)
	// If blocking this cell loses more than just the cell itself, it's an articulation point
	return reachWithoutCell.Size() < fullReach.Size()-1
}

// FindRoomInReachable finds a random room cell within the reachable set
func FindRoomInReachable(reachable *mapset.Set[*world.Cell], avoid *mapset.Set[*world.Cell]) *world.Cell {
	return FindNonArticulationCellInReachable(nil, nil, nil, reachable, avoid)
}

// FindNonArticulationCellInReachable finds a random cell within the reachable set that is NOT an articulation point,
// so placing a blocking entity (e.g. hazard control) there won't disconnect rooms. Pass grid, start, lockedDoors
// as nil to skip articulation-point check (same behavior as old FindRoomInReachable).
func FindNonArticulationCellInReachable(grid *world.Grid, start *world.Cell, lockedDoors *mapset.Set[*world.Cell], reachable *mapset.Set[*world.Cell], avoid *mapset.Set[*world.Cell]) *world.Cell {
	var candidates []*world.Cell
	reachable.Each(func(cell *world.Cell) {
		if cell.Name != "Corridor" && !avoid.Has(cell) {
			candidates = append(candidates, cell)
		}
	})

	if len(candidates) == 0 {
		reachable.Each(func(cell *world.Cell) {
			if !avoid.Has(cell) {
				candidates = append(candidates, cell)
			}
		})
	}

	if len(candidates) == 0 {
		return nil
	}

	// Filter out articulation points so we don't block paths
	if grid != nil && start != nil && lockedDoors != nil {
		var safe []*world.Cell
		for _, cell := range candidates {
			if !IsArticulationPoint(grid, start, cell, lockedDoors) {
				safe = append(safe, cell)
			}
		}
		if len(safe) > 0 {
			candidates = safe
		}
	}

	return candidates[rand.Intn(len(candidates))]
}

// FindNonArticulationCellInRoom finds a cell in the same room as roomCell that is NOT an articulation point,
// so placing a blocking entity (e.g. puzzle terminal) there won't disconnect rooms. Returns nil if none found;
// callers can fall back to roomCell.
func FindNonArticulationCellInRoom(grid *world.Grid, start *world.Cell, roomCell *world.Cell, avoid *mapset.Set[*world.Cell], lockedDoors *mapset.Set[*world.Cell]) *world.Cell {
	if grid == nil || start == nil || roomCell == nil || lockedDoors == nil {
		return nil
	}
	reachable := GetReachableCells(grid, start, lockedDoors)
	var candidates []*world.Cell
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.Room && cell.Name == roomCell.Name && reachable.Has(cell) && !avoid.Has(cell) {
			candidates = append(candidates, cell)
		}
	})
	if len(candidates) == 0 {
		return nil
	}
	var safe []*world.Cell
	for _, cell := range candidates {
		if !IsArticulationPoint(grid, start, cell, lockedDoors) {
			safe = append(safe, cell)
		}
	}
	if len(safe) == 0 {
		return nil
	}
	return safe[rand.Intn(len(safe))]
}

// ContainsSubstring checks if s contains substr
func ContainsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// isRoomStillConnected returns true if, after treating additionalBlockedCell as impassable
// (in addition to existing blocking entities in the room), all doorways (room cells adjacent
// to corridor entries) in the room remain mutually reachable via walkable room cells.
// Used to prevent placing furniture/terminals that would disconnect a room (R8 / I7).
func isRoomStillConnected(g *state.Game, roomName string, entryCellsForRoom []*world.Cell, additionalBlockedCell *world.Cell) bool {
	if len(entryCellsForRoom) == 0 {
		return true
	}
	// Collect room cells and doorway cells (room cells adjacent to an entry)
	roomCells := make([]*world.Cell, 0)
	doorwaySet := mapset.New[*world.Cell]()
	entrySet := mapset.New[*world.Cell]()
	for _, c := range entryCellsForRoom {
		entrySet.Put(c)
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name != roomName {
			return
		}
		roomCells = append(roomCells, cell)
		for _, n := range []*world.Cell{cell.North, cell.East, cell.South, cell.West} {
			if n != nil && entrySet.Has(n) {
				doorwaySet.Put(cell)
				break
			}
		}
	})
	doorways := doorwaySet.Size()
	if doorways == 0 {
		return true
	}
	// Blocked = room cells with blocking entity or the candidate cell
	blocked := mapset.New[*world.Cell]()
	for _, cell := range roomCells {
		if cell == additionalBlockedCell {
			blocked.Put(cell)
			continue
		}
		data := gameworld.GetGameData(cell)
		if data.Generator != nil || data.Door != nil || data.Terminal != nil ||
			data.Puzzle != nil || data.Furniture != nil || data.Hazard != nil ||
			data.HazardControl != nil || data.MaintenanceTerm != nil {
			blocked.Put(cell)
		}
	}
	// BFS from first doorway; count how many doorways we reach
	firstDoorway := (*world.Cell)(nil)
	doorwaySet.Each(func(c *world.Cell) {
		if firstDoorway == nil {
			firstDoorway = c
		}
	})
	if firstDoorway == nil {
		return true
	}
	visited := mapset.New[*world.Cell]()
	queue := []*world.Cell{firstDoorway}
	doorwaysReached := 0
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == nil || visited.Has(current) || blocked.Has(current) {
			continue
		}
		if !current.Room || current.Name != roomName {
			continue
		}
		visited.Put(current)
		if doorwaySet.Has(current) {
			doorwaysReached++
		}
		for _, n := range []*world.Cell{current.North, current.East, current.South, current.West} {
			if n != nil && n.Room && n.Name == roomName && !visited.Has(n) && !blocked.Has(n) {
				queue = append(queue, n)
			}
		}
	}
	return doorwaysReached == doorways
}
