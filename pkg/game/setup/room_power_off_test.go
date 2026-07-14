package setup

import (
	"testing"

	"darkstation/pkg/game/entities"
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

func TestScheduleGeneratorShutdown_delayedRoomShutdown(t *testing.T) {
	g := state.NewGame()
	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteriesAndStart(1)
	g.Generators = []*entities.Generator{gen}
	g.RoomDoorsPowered = map[string]bool{"RoomA": true}
	g.RoomCCTVPowered = map[string]bool{"RoomA": true}

	const now int64 = 2_000_000
	ScheduleGeneratorShutdown(g, "RoomA", 3, 4, now)

	pending, remaining, row, col := GeneratorShutdownPending(g, now)
	if !pending {
		t.Fatal("expected pending room shutdown")
	}
	if remaining != GeneratorShutdownDelay.Milliseconds() {
		t.Fatalf("remaining = %d, want %d", remaining, GeneratorShutdownDelay.Milliseconds())
	}
	if row != 3 || col != 4 {
		t.Fatalf("countdown cell = (%d,%d), want (3,4)", row, col)
	}
	if GeneratorShutdownRoom(g) != "RoomA" {
		t.Fatalf("shutdown room = %q, want RoomA", GeneratorShutdownRoom(g))
	}

	AdvanceGeneratorShutdown(g, now+GeneratorShutdownDelay.Milliseconds()-1)
	if !gen.IsPowered() {
		t.Fatal("generator should stay powered before delay elapses")
	}
	if !g.RoomDoorsPowered["RoomA"] || !g.RoomCCTVPowered["RoomA"] {
		t.Fatal("room circuits should stay armed before delay elapses")
	}

	AdvanceGeneratorShutdown(g, now+GeneratorShutdownDelay.Milliseconds())
	if !gen.IsPowered() {
		t.Fatal("generator should stay online after room shutdown")
	}
	if g.RoomDoorsPowered["RoomA"] || g.RoomCCTVPowered["RoomA"] {
		t.Fatal("room circuits should be off after delay")
	}
	if pending, _, _, _ := GeneratorShutdownPending(g, now+GeneratorShutdownDelay.Milliseconds()); pending {
		t.Fatal("pending room shutdown should be cleared")
	}
}

func TestCancelGeneratorShutdownForRoom_clearsMatchingCountdown(t *testing.T) {
	g := state.NewGame()
	ScheduleGeneratorShutdown(g, "RoomA", 1, 2, 0)
	CancelGeneratorShutdownForRoom(g, "RoomB")
	if pending, _, _, _ := GeneratorShutdownPending(g, 0); !pending {
		t.Fatal("non-matching cancel should leave countdown active")
	}
	CancelGeneratorShutdownForRoom(g, "RoomA")
	if pending, _, _, _ := GeneratorShutdownPending(g, 0); pending {
		t.Fatal("matching cancel should clear countdown")
	}
}
