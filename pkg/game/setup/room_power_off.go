package setup

import (
	"time"

	"darkstation/pkg/game/state"
)

// RoomPowerOffDelay is the grace period after arming OFF at a maintenance terminal
// before door/CCTV circuits actually de-energize, so the player can leave the room.
const RoomPowerOffDelay = 5 * time.Second

// GeneratorShutdownDelay is the countdown after selecting delayed shutdown at a maintenance terminal.
const GeneratorShutdownDelay = 5 * time.Second

// ScheduleRoomPowerOff arms a delayed shutdown for roomName's door and CCTV circuits.
func ScheduleRoomPowerOff(g *state.Game, roomName string, nowMs int64) {
	if g == nil || roomName == "" || IsAlwaysArmedOverlayRoom(roomName) {
		return
	}
	if g.RoomPowerOffPending == nil {
		g.RoomPowerOffPending = make(map[string]int64)
	}
	g.RoomPowerOffPending[roomName] = nowMs + RoomPowerOffDelay.Milliseconds()
}

// CancelRoomPowerOff removes a pending shutdown for roomName.
func CancelRoomPowerOff(g *state.Game, roomName string) {
	if g == nil || roomName == "" || len(g.RoomPowerOffPending) == 0 {
		return
	}
	delete(g.RoomPowerOffPending, roomName)
}

// RoomPowerOffScheduled reports whether roomName has a pending delayed shutdown.
func RoomPowerOffScheduled(g *state.Game, roomName string) bool {
	if g == nil || roomName == "" || g.RoomPowerOffPending == nil {
		return false
	}
	_, ok := g.RoomPowerOffPending[roomName]
	return ok
}

// RoomPowerOffPending reports whether roomName is scheduled to shut down and ms until off (0 if due).
func RoomPowerOffPending(g *state.Game, roomName string, nowMs int64) (pending bool, remainingMs int64) {
	if g == nil || roomName == "" || g.RoomPowerOffPending == nil {
		return false, 0
	}
	offAt, ok := g.RoomPowerOffPending[roomName]
	if !ok {
		return false, 0
	}
	remaining := offAt - nowMs
	if remaining < 0 {
		remaining = 0
	}
	return true, remaining
}

// AdvanceRoomPowerOff applies any due delayed shutdowns.
func AdvanceRoomPowerOff(g *state.Game, nowMs int64) {
	if g == nil || len(g.RoomPowerOffPending) == 0 {
		return
	}
	var due []string
	for roomName, offAt := range g.RoomPowerOffPending {
		if nowMs >= offAt {
			due = append(due, roomName)
		}
	}
	for _, roomName := range due {
		applyRoomPowerOff(g, roomName)
		delete(g.RoomPowerOffPending, roomName)
	}
}

// ApplyRoomPowerOffNow immediately de-energizes roomName's door and CCTV circuits.
func ApplyRoomPowerOffNow(g *state.Game, roomName string) {
	if g == nil || roomName == "" {
		return
	}
	CancelRoomPowerOff(g, roomName)
	CancelGeneratorShutdownForRoom(g, roomName)
	applyRoomPowerOff(g, roomName)
}

func applyRoomPowerOff(g *state.Game, roomName string) {
	if g == nil || roomName == "" {
		return
	}
	if IsAlwaysArmedOverlayRoom(roomName) {
		EnsureAlwaysArmedRoomPower(g, roomName)
		return
	}
	if g.RoomDoorsPowered == nil {
		g.RoomDoorsPowered = make(map[string]bool)
	}
	if g.RoomCCTVPowered == nil {
		g.RoomCCTVPowered = make(map[string]bool)
	}
	g.RoomDoorsPowered[roomName] = false
	g.RoomCCTVPowered[roomName] = false
	ClearRoomPropagatedPower(g, roomName)
	NotifyPowerGridChanged(g)
}

// ScheduleGeneratorShutdown arms a delayed shutdown for roomName's door and CCTV circuits.
func ScheduleGeneratorShutdown(g *state.Game, roomName string, terminalRow, terminalCol int, nowMs int64) {
	if g == nil || roomName == "" {
		return
	}
	g.GeneratorShutdownAt = nowMs + GeneratorShutdownDelay.Milliseconds()
	g.GeneratorShutdownRow = terminalRow
	g.GeneratorShutdownCol = terminalCol
	g.GeneratorShutdownRoomName = roomName
}

// CancelGeneratorShutdown clears any pending delayed room shutdown.
func CancelGeneratorShutdown(g *state.Game) {
	if g == nil {
		return
	}
	g.GeneratorShutdownAt = 0
	g.GeneratorShutdownRow = -1
	g.GeneratorShutdownCol = -1
	g.GeneratorShutdownRoomName = ""
}

// CancelGeneratorShutdownForRoom clears a pending shutdown when it targets roomName.
func CancelGeneratorShutdownForRoom(g *state.Game, roomName string) {
	if g == nil || roomName == "" || g.GeneratorShutdownRoomName != roomName {
		return
	}
	CancelGeneratorShutdown(g)
}

// GeneratorShutdownPending reports whether a delayed room shutdown countdown is active.
func GeneratorShutdownPending(g *state.Game, nowMs int64) (pending bool, remainingMs int64, row int, col int) {
	if g == nil || g.GeneratorShutdownAt == 0 {
		return false, 0, -1, -1
	}
	remaining := g.GeneratorShutdownAt - nowMs
	if remaining < 0 {
		remaining = 0
	}
	return true, remaining, g.GeneratorShutdownRow, g.GeneratorShutdownCol
}

// GeneratorShutdownRoom reports the room targeted by an active delayed shutdown countdown.
func GeneratorShutdownRoom(g *state.Game) string {
	if g == nil {
		return ""
	}
	return g.GeneratorShutdownRoomName
}

// AdvanceGeneratorShutdown applies a due delayed room shutdown and refreshes routing.
func AdvanceGeneratorShutdown(g *state.Game, nowMs int64) bool {
	if g == nil || g.GeneratorShutdownAt == 0 || nowMs < g.GeneratorShutdownAt {
		return false
	}
	roomName := g.GeneratorShutdownRoomName
	CancelGeneratorShutdown(g)
	if roomName != "" {
		applyRoomPowerOff(g, roomName)
	}
	return true
}
