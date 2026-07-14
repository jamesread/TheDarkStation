// Package levelgen provides level generation utilities for placing entities.
package levelgen

import (
	"darkstation/pkg/game/levelrand"
	"fmt"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
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
	for _, roomName := range SortedRoomMapKeys(roomCells) {
		cells := roomCells[roomName]
		if len(cells) == 0 {
			continue
		}
		if generator.IsEmptyOverlayRoom(roomName) {
			continue
		}
		if roomName == generator.ShaftRoomName {
			placeLiftShaftMaintenanceTerminal(g, cells, roomEntries, avoid)
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
				data.HazardControl == nil && data.MaintenanceTerm == nil &&
				data.RepairDevice == nil && data.RepairBlocker == nil &&
				cell.ItemsOnFloor.Size() == 0 {

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
					data.HazardControl == nil && data.MaintenanceTerm == nil &&
					data.RepairDevice == nil && data.RepairBlocker == nil &&
					cell.ItemsOnFloor.Size() == 0 {
					validCells = append(validCells, cell)
				}
			}
		}

		// R8: only place where room stays connected (all doorways mutually reachable).
		// No fallback to validCells: if no R8-compliant candidate exists, skip this room
		// (including start room) to avoid disconnecting the room per I7.
		// The start room may therefore have zero maintenance terminals; InitMaintenanceTerminalPower
		// will then power none (accepted trade-off to preserve I7).
		var entryCells []*world.Cell
		if entryData := roomEntries[roomName]; entryData != nil {
			entryCells = entryData.EntryCells
		}
		var connectedCandidates []*world.Cell
		for _, cell := range validCells {
			if isRoomStillConnected(g, roomName, entryCells, cell) && setup.CanPlaceBlockingEntity(g, cell) {
				connectedCandidates = append(connectedCandidates, cell)
			}
		}
		if len(connectedCandidates) == 0 {
			continue
		}
		candidates := connectedCandidates
		setup.SortCellsByPosition(candidates)

		selectedCell := candidates[levelrand.Intn(len(candidates))]
		maintenanceTerm := entities.NewMaintenanceTerminal(fmt.Sprintf("Maintenance Terminal - %s", roomName), roomName)
		gameworld.GetGameData(selectedCell).MaintenanceTerm = maintenanceTerm
		avoid.Put(selectedCell)
	}
}

func placeLiftShaftMaintenanceTerminal(g *state.Game, cells []*world.Cell, _ map[string]*setup.RoomEntryPoints, avoid *mapset.Set[*world.Cell]) {
	roomName := generator.ShaftRoomName
	var selectedCell *world.Cell
	if east := setup.LiftShaftCellEastOfBottomLeft(g); liftShaftMaintFits(g, east, avoid) {
		selectedCell = east
	}
	if selectedCell == nil {
		for _, cell := range setup.LiftShaftCellsFromBottomLeft(g) {
			if cell == setup.LiftShaftBottomLeftCell(g) {
				continue
			}
			if liftShaftMaintFits(g, cell, avoid) {
				selectedCell = cell
				break
			}
		}
	}
	if selectedCell == nil && len(cells) > 0 {
		ordered := cells
		setup.SortCellsByPosition(ordered)
		for _, cell := range ordered {
			if liftShaftMaintFits(g, cell, avoid) {
				selectedCell = cell
				break
			}
		}
	}
	if selectedCell == nil {
		return
	}
	maintenanceTerm := entities.NewMaintenanceTerminal(fmt.Sprintf("Maintenance Terminal - %s", roomName), roomName)
	// The shaft bootstrap generator is online from level start; its terminal is live too,
	// so later placement passes can rely on shaft-pocket power control (e.g. exit-gating repairs).
	if liftShaftHasPoweredGenerator(g) {
		maintenanceTerm.Powered = true
	}
	gameworld.GetGameData(selectedCell).MaintenanceTerm = maintenanceTerm
	avoid.Put(selectedCell)
}

func liftShaftHasPoweredGenerator(g *state.Game) bool {
	for _, cell := range setup.LiftShaftCellsFromBottomLeft(g) {
		gen := gameworld.GetGameData(cell).Generator
		if gen != nil && gen.IsPowered() {
			return true
		}
	}
	return false
}

func liftShaftMaintFits(g *state.Game, cell *world.Cell, avoid *mapset.Set[*world.Cell]) bool {
	if g == nil || cell == nil || !cell.Room || cell.Name != generator.ShaftRoomName {
		return false
	}
	if (avoid != nil && avoid.Has(cell)) || cell.ExitCell {
		return false
	}
	data := gameworld.GetGameData(cell)
	if data.Generator != nil || data.Door != nil || data.Terminal != nil ||
		data.Puzzle != nil || data.Furniture != nil || data.Hazard != nil ||
		data.HazardControl != nil || data.MaintenanceTerm != nil ||
		data.RepairDevice != nil || data.RepairBlocker != nil ||
		cell.ItemsOnFloor.Size() > 0 {
		return false
	}
	// Must not strand other blocking entities (e.g. take the shaft bootstrap generator's
	// last adjacent stand cell): downstream placement passes (repairs, routing couplers)
	// refuse all cells once any entity loses nav access, breaking run progression.
	return setup.CandidateBlockingCellHasAdjacentNavSpace(g, cell, avoid) &&
		setup.BlockingPlacementPreservesNavAccess(g, cell) &&
		setup.CompletionRegionPreserved(g, cell)
}
