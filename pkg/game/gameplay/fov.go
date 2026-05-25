package gameplay

import (
	engworld "darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// unpoweredDoorSightBlocker stops FOV rays at doors whose room power is off.
func unpoweredDoorSightBlocker(g *state.Game) engworld.SightBlocker {
	if g == nil {
		return nil
	}
	return func(cell *engworld.Cell) bool {
		if !gameworld.HasDoor(cell) {
			return false
		}
		roomName := gameworld.GetGameData(cell).Door.RoomName
		return !g.RoomDoorsPowered[roomName]
	}
}
