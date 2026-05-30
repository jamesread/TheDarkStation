package setup

import (
	"testing"

	"darkstation/pkg/game/state"
)

func TestScheduleRoomPowerOff_delayedShutdown(t *testing.T) {
	g := state.NewGame()
	g.RoomDoorsPowered = map[string]bool{"RoomA": true}
	g.RoomCCTVPowered = map[string]bool{"RoomA": true}

	const now int64 = 1_000_000
	ScheduleRoomPowerOff(g, "RoomA", now)

	pending, remaining := RoomPowerOffPending(g, "RoomA", now)
	if !pending {
		t.Fatal("expected pending shutdown")
	}
	if remaining != RoomPowerOffDelay.Milliseconds() {
		t.Fatalf("remaining = %d, want %d", remaining, RoomPowerOffDelay.Milliseconds())
	}

	AdvanceRoomPowerOff(g, now+RoomPowerOffDelay.Milliseconds()-1)
	if !g.RoomDoorsPowered["RoomA"] || !g.RoomCCTVPowered["RoomA"] {
		t.Fatal("circuits should stay on before delay elapses")
	}

	AdvanceRoomPowerOff(g, now+RoomPowerOffDelay.Milliseconds())
	if g.RoomDoorsPowered["RoomA"] || g.RoomCCTVPowered["RoomA"] {
		t.Fatal("circuits should be off after delay")
	}
	if pending, _ := RoomPowerOffPending(g, "RoomA", now+RoomPowerOffDelay.Milliseconds()); pending {
		t.Fatal("pending entry should be cleared")
	}
}

func TestCancelRoomPowerOff_clearsPending(t *testing.T) {
	g := state.NewGame()
	g.RoomDoorsPowered = map[string]bool{"RoomA": true}
	ScheduleRoomPowerOff(g, "RoomA", 0)
	CancelRoomPowerOff(g, "RoomA")
	if pending, _ := RoomPowerOffPending(g, "RoomA", RoomPowerOffDelay.Milliseconds()); pending {
		t.Fatal("expected no pending shutdown after cancel")
	}
}
