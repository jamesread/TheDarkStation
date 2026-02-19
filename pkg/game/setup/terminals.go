// Package setup provides level setup functionality for The Dark Station.
package setup

import (
	"fmt"
	"math/rand"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// collectUniqueRoomNames returns a list of unique room names (excluding corridors)
func collectUniqueRoomNames(grid *world.Grid) []string {
	namesSet := mapset.New[string]()
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell.Room && cell.Name != "Corridor" && cell.Name != "" {
			namesSet.Put(cell.Name)
		}
	})

	var names []string
	namesSet.Each(func(name string) {
		names = append(names, name)
	})
	return names
}

// placeCCTVTerminals places CCTV terminals in the level
func placeCCTVTerminals(g *state.Game, avoid *mapset.Set[*world.Cell], roomEntries map[string]*RoomEntryPoints) {
	// Calculate number of terminals based on level
	numTerminals := calculateNumTerminals(g.Level)
	if numTerminals == 0 {
		return
	}

	// Collect all unique room names (excluding corridors)
	roomNames := collectUniqueRoomNames(g.Grid)

	for i := 0; i < numTerminals; i++ {
		terminalRoom := findRoom(g, g.Grid.StartCell(), avoid)
		if terminalRoom == nil || len(roomNames) == 0 {
			continue
		}

		terminal := entities.NewCCTVTerminal(fmt.Sprintf("CCTV Terminal #%d", i+1))

		// Assign a random room for this terminal to reveal
		targetIdx := rand.Intn(len(roomNames))
		terminal.TargetRoom = roomNames[targetIdx]
		// Remove this room from the list so each terminal reveals a different room
		roomNames = append(roomNames[:targetIdx], roomNames[targetIdx+1:]...)

		// Find a valid cell in the room that doesn't block entrances
		placeTerminalInRoom(g, terminal, terminalRoom, roomEntries, avoid)
	}
}

// calculateNumTerminals calculates the number of CCTV terminals to place
func calculateNumTerminals(level int) int {
	if level >= 2 {
		numTerminals := 1 + (level-1)/3 // Level 2 = 1, Level 3 = 1, Level 4 = 2, etc.
		if numTerminals > 3 {
			numTerminals = 3
		}
		return numTerminals
	}
	return 0 // No terminals on level 1
}

// placeTerminalInRoom places a terminal in a specific room.
// R8: only places on a cell where the room stays connected (all doorways mutually reachable).
func placeTerminalInRoom(g *state.Game, terminal *entities.CCTVTerminal, terminalRoom *world.Cell, roomEntries map[string]*RoomEntryPoints, avoid *mapset.Set[*world.Cell]) {
	roomName := terminalRoom.Name
	entryPoints := getRoomEntryPoints(terminalRoom, roomEntries)

	// Collect all cells in this room
	roomCells := collectRoomCells(g.Grid, roomName)

	// Find valid cells (not entry points, not already used, not exit cells)
	validCells := filterValidTerminalCells(roomCells, entryPoints, avoid)

	// If no valid cells found, skip placement (cannot satisfy R8)
	if len(validCells) == 0 {
		return
	}

	// R8: filter to cells where room stays connected after placing terminal
	var entryCells []*world.Cell
	if ep := roomEntries[roomName]; ep != nil {
		entryCells = ep.EntryCells
	}
	var connectedCandidates []*world.Cell
	for _, cell := range validCells {
		if RoomStillConnectedIfBlock(g, roomName, entryCells, cell) {
			connectedCandidates = append(connectedCandidates, cell)
		}
	}
	if len(connectedCandidates) == 0 {
		// No cell keeps room connected; skip placement to satisfy R8
		return
	}

	selectedCell := connectedCandidates[rand.Intn(len(connectedCandidates))]
	gameworld.GetGameData(selectedCell).Terminal = terminal
	avoid.Put(selectedCell)
}

// getRoomEntryPoints gets the entry point cells for a room
func getRoomEntryPoints(room *world.Cell, roomEntries map[string]*RoomEntryPoints) *mapset.Set[*world.Cell] {
	entryPoints := mapset.New[*world.Cell]()
	if entryData, ok := roomEntries[room.Name]; ok {
		for _, entryCell := range entryData.EntryCells {
			// Mark the room cells adjacent to entry points as blocked
			neighbors := []*world.Cell{entryCell.North, entryCell.East, entryCell.South, entryCell.West}
			for _, neighbor := range neighbors {
				if neighbor != nil && neighbor.Room && neighbor.Name == room.Name {
					entryPoints.Put(neighbor)
				}
			}
		}
	}
	return &entryPoints
}

// collectRoomCells collects all cells in a room
func collectRoomCells(grid *world.Grid, roomName string) []*world.Cell {
	var roomCells []*world.Cell
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.Room && cell.Name == roomName {
			roomCells = append(roomCells, cell)
		}
	})
	return roomCells
}

// filterValidTerminalCells filters cells that are valid for terminal placement
func filterValidTerminalCells(roomCells []*world.Cell, entryPoints *mapset.Set[*world.Cell], avoid *mapset.Set[*world.Cell]) []*world.Cell {
	var validCells []*world.Cell
	for _, cell := range roomCells {
		data := gameworld.GetGameData(cell)
		// Don't place terminal on entry points, exit cells, or cells with other entities
		if !avoid.Has(cell) && !cell.ExitCell && !entryPoints.Has(cell) &&
			data.Generator == nil && data.Door == nil && data.Terminal == nil &&
			data.Furniture == nil && data.Hazard == nil && data.HazardControl == nil {
			validCells = append(validCells, cell)
		}
	}
	return validCells
}
