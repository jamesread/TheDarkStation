package setup

import (
	"testing"

	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/state"
)

func TestEnsureAlwaysArmedOverlayRoomPower_armsShip(t *testing.T) {
	g := state.NewGame()
	g.RoomDoorsPowered = map[string]bool{
		generator.ShipRoomName: false,
	}
	g.RoomCCTVPowered = map[string]bool{
		generator.ShipRoomName: false,
	}
	g.RoomLightsPowered = map[string]bool{
		generator.ShipRoomName: false,
	}
	g.RoomPowerOnline = map[string]bool{
		generator.ShipRoomName: false,
	}

	EnsureAlwaysArmedOverlayRoomPower(g)

	room := generator.ShipRoomName
	if !g.RoomDoorsPowered[room] || !g.RoomCCTVPowered[room] || !g.RoomLightsPowered[room] {
		t.Fatalf("%q circuits should be armed", room)
	}
	if !RoomIsOnline(g, room) || !RoomConsideredPowered(g, room) {
		t.Fatalf("%q should read as powered", room)
	}
}

func TestApplyRoomPowerOffNow_ignoredForOverlayRooms(t *testing.T) {
	g := state.NewGame()
	EnsureAlwaysArmedOverlayRoomPower(g)

	ApplyRoomPowerOffNow(g, generator.ShipRoomName)

	room := generator.ShipRoomName
	if !g.RoomDoorsPowered[room] || !g.RoomCCTVPowered[room] {
		t.Fatalf("%q should stay armed after power-off attempt", room)
	}
}

func TestScheduleRoomPowerOff_ignoredForOverlayRooms(t *testing.T) {
	g := state.NewGame()
	EnsureAlwaysArmedOverlayRoomPower(g)

	ScheduleRoomPowerOff(g, generator.ShipRoomName, PowerNowMs())
	if RoomPowerOffScheduled(g, generator.ShipRoomName) {
		t.Fatal("Ship should not accept delayed power-off scheduling")
	}
}
