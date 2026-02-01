// Package setup provides level setup functionality for The Dark Station.
package setup

import (
	"fmt"
	"math/rand"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// RoomEntryPoints holds entry points for a room
type RoomEntryPoints struct {
	RoomName   string
	EntryCells []*world.Cell
}

// findRoomEntryPoints finds all room entry points (corridor cells that provide access to each room)
func findRoomEntryPoints(grid *world.Grid) map[string]*RoomEntryPoints {
	entries := make(map[string]*RoomEntryPoints)
	seenCells := mapset.New[string]()

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

// getNumLockedRooms determines the number of locked rooms based on level
func getNumLockedRooms(level int) int {
	if level >= 6 {
		return 4
	}
	if level >= 4 {
		return 3
	}
	if level >= 2 {
		return 2
	}
	return 0
}

// placeLockedRooms places doors to lock rooms based on level requirements
func placeLockedRooms(g *state.Game, avoid *mapset.Set[*world.Cell], lockedDoorCells *mapset.Set[*world.Cell]) {
	numLockedRooms := getNumLockedRooms(g.Level)
	if numLockedRooms == 0 {
		return
	}

	// Find all room entry points
	roomEntries := findRoomEntryPoints(g.Grid)

	// Build list of candidate rooms
	candidates := buildRoomCandidates(roomEntries)

	// Shuffle candidates for variety
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	// Track which rooms have doors
	roomsWithDoors := mapset.New[string]()
	lockedRoomsPlaced := 0

	// Place doors to fully block selected rooms
	for _, candidate := range candidates {
		if lockedRoomsPlaced >= numLockedRooms {
			break
		}

		if !canPlaceDoorsForRoom(g, candidate, avoid, lockedDoorCells) {
			continue
		}

		// Place the doors and keycard
		placeRoomDoors(g, candidate, avoid, lockedDoorCells, &roomsWithDoors)
		lockedRoomsPlaced++
	}
}

// roomCandidate represents a room that could be locked
type roomCandidate struct {
	name    string
	entries *RoomEntryPoints
}

// buildRoomCandidates builds a list of candidate rooms for locking
func buildRoomCandidates(roomEntries map[string]*RoomEntryPoints) []roomCandidate {
	var candidates []roomCandidate
	for roomName, entries := range roomEntries {
		// Only consider rooms with 1-3 entry points (manageable to door)
		if len(entries.EntryCells) >= 1 && len(entries.EntryCells) <= 3 {
			candidates = append(candidates, roomCandidate{
				name:    roomName,
				entries: entries,
			})
		}
	}
	return candidates
}

// canPlaceDoorsForRoom checks if doors can be placed for a room
func canPlaceDoorsForRoom(g *state.Game, candidate roomCandidate, avoid *mapset.Set[*world.Cell], lockedDoorCells *mapset.Set[*world.Cell]) bool {
	entryCells := candidate.entries.EntryCells

	// Skip if already has doors
	// This check is done by the caller with roomsWithDoors

	// Check if all entry cells are available and reachable
	currentlyReachable := getReachableCells(g.Grid, g.Grid.StartCell(), lockedDoorCells)
	for _, cell := range entryCells {
		if avoid.Has(cell) || lockedDoorCells.Has(cell) || !currentlyReachable.Has(cell) {
			return false
		}
	}

	// Test if blocking ALL entry cells reduces reachability
	testLocked := mapset.New[*world.Cell]()
	lockedDoorCells.Each(func(c *world.Cell) { testLocked.Put(c) })
	for _, cell := range entryCells {
		testLocked.Put(cell)
	}
	reachableWithDoors := getReachableCells(g.Grid, g.Grid.StartCell(), &testLocked)

	// Must actually block something
	if reachableWithDoors.Size() >= currentlyReachable.Size() {
		return false
	}

	return true
}

// placeRoomDoors places doors and keycard for a room
func placeRoomDoors(g *state.Game, candidate roomCandidate, avoid *mapset.Set[*world.Cell], lockedDoorCells *mapset.Set[*world.Cell], roomsWithDoors *mapset.Set[string]) {
	roomName := candidate.name
	entryCells := candidate.entries.EntryCells

	// Test reachability with doors
	testLocked := mapset.New[*world.Cell]()
	lockedDoorCells.Each(func(c *world.Cell) { testLocked.Put(c) })
	for _, cell := range entryCells {
		testLocked.Put(cell)
	}
	reachableWithDoors := getReachableCells(g.Grid, g.Grid.StartCell(), &testLocked)

	// Place the keycard in the area reachable BEFORE these doors
	keycardRoom := findRoomInReachable(reachableWithDoors, avoid)
	if keycardRoom == nil {
		return
	}

	// Create the keycard (one keycard opens all doors to this room)
	door := entities.NewDoor(roomName)
	keycardName := door.KeycardName()

	keycard := world.NewItem(keycardName)
	keycardRoom.ItemsOnFloor.Put(keycard)
	avoid.Put(keycardRoom)
	g.AddHint("The " + renderer.StyledKeycard(keycardName) + " is in " + renderer.StyledCell(keycardRoom.Name))

	// Place doors on ALL entry cells (they share the same keycard)
	for _, cell := range entryCells {
		cellDoor := entities.NewDoor(roomName)
		gameworld.GetGameData(cell).Door = cellDoor
		avoid.Put(cell)
		lockedDoorCells.Put(cell)
	}

	roomsWithDoors.Put(roomName)

	// Add hint about the doors
	if len(entryCells) == 1 {
		g.AddHint("The " + renderer.StyledDoor(door.DoorName()) + " blocks access to " + renderer.StyledCell(roomName))
	} else {
		g.AddHint(fmt.Sprintf("%d doors block access to %s", len(entryCells), renderer.StyledCell(roomName)))
	}
}

// EnsureEveryRoomHasDoor places one unlocked door for each room that has no door yet.
func EnsureEveryRoomHasDoor(g *state.Game, avoid *mapset.Set[*world.Cell], lockedDoorCells *mapset.Set[*world.Cell], roomEntries map[string]*RoomEntryPoints) {
	roomsWithDoors := mapset.New[string]()
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if gameworld.HasDoor(cell) {
			roomName := gameworld.GetGameData(cell).Door.RoomName
			roomsWithDoors.Put(roomName)
		}
	})

	for roomName, entries := range roomEntries {
		if roomsWithDoors.Has(roomName) || len(entries.EntryCells) == 0 {
			continue
		}
		cell := entries.EntryCells[0]
		if avoid.Has(cell) || lockedDoorCells.Has(cell) {
			continue
		}
		door := entities.NewDoor(roomName)
		door.Unlock()
		gameworld.GetGameData(cell).Door = door
		avoid.Put(cell)
		roomsWithDoors.Put(roomName)
	}
}
