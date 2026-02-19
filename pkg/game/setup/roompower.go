// Package setup provides level setup functionality for The Dark Station.
package setup

import (
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// InitRoomPower initializes room power state: all rooms unpowered by default,
// with the start room's doors powered so the player can leave.
func InitRoomPower(g *state.Game) {
	if g.Grid == nil {
		return
	}
	if g.RoomDoorsPowered == nil {
		g.RoomDoorsPowered = make(map[string]bool)
	}
	if g.RoomCCTVPowered == nil {
		g.RoomCCTVPowered = make(map[string]bool)
	}
	if g.RoomLightsPowered == nil {
		g.RoomLightsPowered = make(map[string]bool)
	}

	// Discover all room names from the grid
	roomNames := make(map[string]bool)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.Room && cell.Name != "" {
			roomNames[cell.Name] = true
		}
	})

	for roomName := range roomNames {
		g.RoomDoorsPowered[roomName] = false
		g.RoomCCTVPowered[roomName] = false
		g.RoomLightsPowered[roomName] = true // lights on by default (0w; toggled at maintenance terminal)
	}

	// Start room's doors are powered so the player can leave
	startCell := g.Grid.StartCell()
	if startCell != nil && startCell.Name != "" {
		g.RoomDoorsPowered[startCell.Name] = true
	}
}

// InitMaintenanceTerminalPower sets all maintenance terminals unpowered, then powers
// the one(s) in the start room so the player can use at least one terminal to restore others.
// Call after maintenance terminals are placed (e.g. after PlaceMaintenanceTerminals).
func InitMaintenanceTerminalPower(g *state.Game) {
	if g.Grid == nil {
		return
	}
	startRoomName := ""
	if start := g.Grid.StartCell(); start != nil && start.Name != "" {
		startRoomName = start.Name
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.MaintenanceTerm == nil {
			return
		}
		data.MaintenanceTerm.Powered = cell.Name == startRoomName
	})
}
