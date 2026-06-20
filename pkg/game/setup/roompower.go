// Package setup provides level setup functionality for The Dark Station.
package setup

import (
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// InitRoomPower initializes room power state: all rooms unpowered by default.
// Generator bootstrap and player terminal actions arm routing when power reaches a room.
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
	EnsureRoomPowerOnlineMap(g)

	// Discover all room names from the grid
	roomNames := make(map[string]bool)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.Room && cell.Name != "" {
			roomNames[cell.Name] = true
		}
	})

	for _, roomName := range sortedRoomNames(roomNames) {
		if IsAlwaysArmedOverlayRoom(roomName) {
			EnsureAlwaysArmedRoomPower(g, roomName)
			continue
		}
		g.RoomDoorsPowered[roomName] = false
		g.RoomCCTVPowered[roomName] = false
		g.RoomLightsPowered[roomName] = true // lights on by default (0w; toggled at maintenance terminal)
	}
}

// InitMaintenanceTerminalPower sets all maintenance terminals unpowered.
// Call EnsureGeneratorRoomBootstrap after generators are placed to feed terminals from
// powered generators on the conductive power grid (including same-room local feed).
func InitMaintenanceTerminalPower(g *state.Game) {
	if g.Grid == nil {
		return
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.MaintenanceTerm == nil {
			return
		}
		data.MaintenanceTerm.Powered = false
	})
}
