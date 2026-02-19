// Package setup provides level setup functionality for The Dark Station.
package setup

import (
	"math/rand"
	"sort"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
)

// GetAdjacentRoomNames returns room names that are adjacent to the given room.
// Adjacency includes (1) direct: a cell in roomName has a N/S/E/W neighbour in room B;
// (2) corridor-mediated: a cell in roomName borders a corridor, and that corridor (or
// corridors reachable from it) borders room B. So rooms connected only by corridors
// (e.g. A-Corridor-B) are considered adjacent. The name "Corridor" is excluded from
// the result so the UI shows only named rooms. Result is sorted and includes roomName.
// Nil is returned for nil grid, empty roomName, or roomName not in grid.
func GetAdjacentRoomNames(grid *world.Grid, roomName string) []string {
	if grid == nil || roomName == "" {
		return nil
	}
	adjacent := make(map[string]bool)
	foundRoom := false
	var corridorFrontier []*world.Cell

	// Pass 1: direct neighbours and corridor cells next to this room.
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name != roomName {
			return
		}
		foundRoom = true
		for _, n := range []*world.Cell{cell.North, cell.South, cell.East, cell.West} {
			if n == nil || !n.Room {
				continue
			}
			if n.Name == "Corridor" {
				corridorFrontier = append(corridorFrontier, n)
				continue
			}
			if n.Name != "" && n.Name != roomName {
				adjacent[n.Name] = true
			}
		}
	})

	if !foundRoom {
		return nil
	}

	// Pass 2: from corridors adjacent to this room, BFS through corridors and add any room on the other side.
	if len(corridorFrontier) > 0 {
		visited := make(map[*world.Cell]bool)
		queue := append([]*world.Cell(nil), corridorFrontier...)
		for _, c := range corridorFrontier {
			visited[c] = true
		}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			for _, n := range []*world.Cell{cur.North, cur.South, cur.East, cur.West} {
				if n == nil || !n.Room {
					continue
				}
				if n.Name == "Corridor" {
					if !visited[n] {
						visited[n] = true
						queue = append(queue, n)
					}
					continue
				}
				if n.Name != "" && n.Name != roomName {
					adjacent[n.Name] = true
				}
			}
		}
	}

	adjacent[roomName] = true
	var names []string
	for n := range adjacent {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// getReachableCells finds all reachable cells from a starting point, avoiding locked doors
func getReachableCells(grid *world.Grid, start *world.Cell, lockedDoors *mapset.Set[*world.Cell]) *mapset.Set[*world.Cell] {
	reachable := mapset.New[*world.Cell]()
	queue := []*world.Cell{start}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == nil || !current.Room || reachable.Has(current) {
			continue
		}

		// Can't pass through locked doors
		if lockedDoors != nil && lockedDoors.Has(current) {
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

// findRoomInReachable finds a random room cell within the reachable set
func findRoomInReachable(reachable *mapset.Set[*world.Cell], avoid *mapset.Set[*world.Cell]) *world.Cell {
	var candidates []*world.Cell
	reachable.Each(func(cell *world.Cell) {
		if cell.Name != "Corridor" && !avoid.Has(cell) {
			candidates = append(candidates, cell)
		}
	})

	if len(candidates) == 0 {
		// Fallback to any reachable cell
		reachable.Each(func(cell *world.Cell) {
			if !avoid.Has(cell) {
				candidates = append(candidates, cell)
			}
		})
	}

	if len(candidates) == 0 {
		return nil
	}

	return candidates[rand.Intn(len(candidates))]
}

// collectReachableRooms collects all reachable rooms from a starting cell using BFS
func collectReachableRooms(start *world.Cell, avoid *mapset.Set[*world.Cell]) []*world.Cell {
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

// manhattanDistance calculates the Manhattan distance between two cells
func manhattanDistance(a, b *world.Cell) int {
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

// findRoom finds a random reachable room at an appropriate distance based on level
func findRoom(g *state.Game, start *world.Cell, avoid *mapset.Set[*world.Cell]) *world.Cell {
	rooms := collectReachableRooms(start, avoid)

	if len(rooms) == 0 {
		return start
	}

	// Calculate minimum distance based on level
	// Level 1: min 2, Level 5: min 6, Level 10: min 11
	minDistance := 1 + g.Level

	// Filter rooms by minimum distance
	var farRooms []*world.Cell
	for _, room := range rooms {
		if manhattanDistance(start, room) >= minDistance {
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
				d := manhattanDistance(start, room)
				if d > maxDist {
					maxDist = d
				}
			}
			threshold := maxDist / 2
			farRooms = nil
			for _, room := range rooms {
				if manhattanDistance(start, room) >= threshold {
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

// placeItem places an item in a random reachable room at an appropriate distance based on level
func placeItem(g *state.Game, start *world.Cell, item *world.Item, avoid *mapset.Set[*world.Cell]) *world.Cell {
	room := findRoom(g, start, avoid)
	if room != nil {
		room.ItemsOnFloor.Put(item)
		avoid.Put(room)
	}
	return room
}

// isChokepoint checks if a cell is a chokepoint (removing it would disconnect parts of the map)
func isChokepoint(grid *world.Grid, cell *world.Cell, start *world.Cell) bool {
	if cell == nil || !cell.Room {
		return false
	}

	// Temporarily mark cell as blocked
	blocked := mapset.New[*world.Cell]()
	blocked.Put(cell)

	// Check reachability without this cell
	reachable := getReachableCells(grid, start, &blocked)
	totalRooms := 0
	grid.ForEachCell(func(row, col int, c *world.Cell) {
		if c != nil && c.Room {
			totalRooms++
		}
	})

	// If removing this cell significantly reduces reachability, it's a chokepoint
	// Use a threshold: if we lose more than 10% of rooms, it's a chokepoint
	threshold := totalRooms / 10
	return reachable.Size() < totalRooms-threshold
}
