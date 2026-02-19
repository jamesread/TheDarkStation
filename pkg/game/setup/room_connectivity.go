// Package setup provides level setup functionality for The Dark Station.
// This file implements R8 (prevent room disconnection) per specs/level-layout-and-solvability.md.

package setup

import (
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// RoomStillConnectedIfBlock returns true if, after treating additionalBlockedCell as impassable
// (in addition to existing blocking entities in the room), all doorways (room cells adjacent
// to corridor entries) in the room remain mutually reachable via walkable room cells.
// Used to prevent placing furniture/terminals that would disconnect a room (R8 / I7).
// entryCellsForRoom are the corridor-side entry cells for this room (e.g. from RoomEntryPoints.EntryCells).
func RoomStillConnectedIfBlock(g *state.Game, roomName string, entryCellsForRoom []*world.Cell, additionalBlockedCell *world.Cell) bool {
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
