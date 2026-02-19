// Package levelgen provides level generation utilities for placing entities.
package levelgen

import (
	"fmt"
	"math/rand"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// PlaceMaintenanceTerminals places one maintenance terminal per room, aligned against walls
func PlaceMaintenanceTerminals(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	// Collect all unique rooms
	roomCells := make(map[string][]*world.Cell)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell.Room && cell.Name != "Corridor" && cell.Name != "" {
			roomCells[cell.Name] = append(roomCells[cell.Name], cell)
		}
	})

	// Get room entry points to avoid blocking them
	roomEntries := setup.FindRoomEntryPoints(g.Grid)

	// Place one maintenance terminal per room
	for roomName, cells := range roomCells {
		if len(cells) == 0 {
			continue
		}

		// Find cells that are against walls (have at least one non-room neighbor OR corridor neighbor)
		// Also prefer cells on the edge of the room (fewer room neighbors)
		var wallCells []*world.Cell
		var edgeCells []*world.Cell

		for _, cell := range cells {
			data := gameworld.GetGameData(cell)

			// Skip if already has entities
			if !avoid.Has(cell) && !cell.ExitCell &&
				data.Generator == nil && data.Door == nil && data.Terminal == nil &&
				data.Puzzle == nil && data.Furniture == nil && data.Hazard == nil &&
				data.HazardControl == nil && data.MaintenanceTerm == nil {

				// Check if cell is against a wall (has a non-room neighbor)
				isWallCell := false
				neighbors := []*world.Cell{cell.North, cell.East, cell.South, cell.West}
				roomNeighborCount := 0

				for _, neighbor := range neighbors {
					if neighbor == nil {
						isWallCell = true // Edge of map
					} else if !neighbor.Room {
						isWallCell = true // Wall
					} else if neighbor.Room && neighbor.Name == roomName {
						roomNeighborCount++
					}
				}

				// Check entry points
				entryPoints := mapset.New[*world.Cell]()
				if entryData, ok := roomEntries[roomName]; ok {
					for _, entryCell := range entryData.EntryCells {
						entryNeighbors := []*world.Cell{entryCell.North, entryCell.East, entryCell.South, entryCell.West}
						for _, neighbor := range entryNeighbors {
							if neighbor != nil && neighbor.Room && neighbor.Name == roomName {
								entryPoints.Put(neighbor)
							}
						}
					}
				}

				if !entryPoints.Has(cell) {
					if isWallCell {
						wallCells = append(wallCells, cell)
					} else if roomNeighborCount <= 2 {
						// Edge of room (2 or fewer room neighbors)
						edgeCells = append(edgeCells, cell)
					}
				}
			}
		}

		// Prefer wall cells, fall back to edge cells, then any valid cell
		var validCells []*world.Cell
		if len(wallCells) > 0 {
			validCells = wallCells
		} else if len(edgeCells) > 0 {
			validCells = edgeCells
		} else {
			// Last resort: any valid cell in the room
			for _, cell := range cells {
				data := gameworld.GetGameData(cell)
				if !avoid.Has(cell) && !cell.ExitCell &&
					data.Generator == nil && data.Door == nil && data.Terminal == nil &&
					data.Puzzle == nil && data.Furniture == nil && data.Hazard == nil &&
					data.HazardControl == nil && data.MaintenanceTerm == nil {
					validCells = append(validCells, cell)
				}
			}
		}

		// R8: only place where room stays connected (all doorways mutually reachable)
		var entryCells []*world.Cell
		if entryData := roomEntries[roomName]; entryData != nil {
			entryCells = entryData.EntryCells
		}
		var connectedCandidates []*world.Cell
		for _, cell := range validCells {
			if isRoomStillConnected(g, roomName, entryCells, cell) {
				connectedCandidates = append(connectedCandidates, cell)
			}
		}
		// For start room: must have a terminal (InitMaintenanceTerminalPower only powers start room terminals).
		// Without one, the level would have 0 accessible powered terminals and be unsolvable.
		// Fall back to validCells if no R8-compliant candidate exists.
		candidates := connectedCandidates
		if len(candidates) == 0 && len(validCells) > 0 {
			startCell := g.Grid.StartCell()
			if startCell != nil && startCell.Name == roomName {
				candidates = validCells
			} else {
				continue
			}
		} else if len(candidates) == 0 {
			continue
		}

		selectedCell := candidates[rand.Intn(len(candidates))]
		maintenanceTerm := entities.NewMaintenanceTerminal(fmt.Sprintf("Maintenance Terminal - %s", roomName), roomName)
		gameworld.GetGameData(selectedCell).MaintenanceTerm = maintenanceTerm
		avoid.Put(selectedCell)
	}
}
