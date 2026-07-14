package setup

import (
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/state"
)

// IsAlwaysArmedOverlayRoom reports deck 1 Ship room circuits that must stay powered.
func IsAlwaysArmedOverlayRoom(roomName string) bool {
	return roomName == generator.ShipRoomName
}

// EnsureAlwaysArmedOverlayRoomPower forces Ship room circuits on.
func EnsureAlwaysArmedOverlayRoomPower(g *state.Game) {
	if g == nil {
		return
	}
	EnsureAlwaysArmedRoomPower(g, generator.ShipRoomName)
}

// EnsureAlwaysArmedRoomPower arms doors, CCTV, lights, and online state for one overlay room.
func EnsureAlwaysArmedRoomPower(g *state.Game, roomName string) {
	if g == nil || !IsAlwaysArmedOverlayRoom(roomName) {
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
	g.RoomDoorsPowered[roomName] = true
	g.RoomCCTVPowered[roomName] = true
	g.RoomLightsPowered[roomName] = true
	g.RoomPowerOnline[roomName] = true
	CancelRoomPowerOff(g, roomName)
}
