// Package setup tests level setup: InitRoomPower (start room doors powered at init), etc.
package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
)

func TestInitRoomPower_NilGridNoPanic(t *testing.T) {
	g := state.NewGame()
	g.Grid = nil
	InitRoomPower(g) // must not panic
}

func TestInitRoomPower_StartRoomDoorsPowered(t *testing.T) {
	// InitRoomPower sets RoomDoorsPowered[startRoomName] = true so the player can leave (FR23).
	g := state.NewGame()
	grid := world.NewGrid(2, 2)
	grid.MarkAsRoomWithName(0, 0, "StartRoom", "ROOM_START")
	grid.SetStartCellAt(0, 0)
	g.Grid = grid

	InitRoomPower(g)

	if !g.RoomDoorsPowered["StartRoom"] {
		t.Error("InitRoomPower: RoomDoorsPowered[StartRoom] = false, want true (start room doors powered at init)")
	}
}
