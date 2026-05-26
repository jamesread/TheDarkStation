package setup

import (
	"time"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// PowerPropagationDelay is the per-hop delay before a room receives propagated power (0 = instant).
const PowerPropagationDelay time.Duration = 0

// EnsureRoomPowerOnlineMap initializes RoomPowerOnline when nil.
func EnsureRoomPowerOnlineMap(g *state.Game) {
	if g == nil {
		return
	}
	if g.RoomPowerOnline == nil {
		g.RoomPowerOnline = make(map[string]bool)
	}
}

// EnergizeArmedRoomsForTest sets RoomPowerOnline for every armed room (unit tests only).
func EnergizeArmedRoomsForTest(g *state.Game) {
	if g == nil {
		return
	}
	EnsureRoomPowerOnlineMap(g)
	for name, armed := range g.RoomDoorsPowered {
		g.RoomPowerOnline[name] = armed
	}
}

// RoomIsOnline reports whether propagated power has reached the room name on the live grid.
func RoomIsOnline(g *state.Game, roomName string) bool {
	if g == nil || roomName == "" {
		return false
	}
	return g.RoomPowerOnline != nil && g.RoomPowerOnline[roomName]
}

// RoomConsideredPowered reports whether a room should display and behave as powered,
// including zero-draw circuits that are armed on a live grid.
func RoomConsideredPowered(g *state.Game, roomName string) bool {
	if g == nil || roomName == "" {
		return false
	}
	if RoomIsOnline(g, roomName) {
		return true
	}
	return g.RoomDoorsPowered != nil && g.RoomDoorsPowered[roomName] && RoomHasLivePower(g, roomName)
}

// RoomManualEgressReleased reports whether the player has manually released egress for the room.
func RoomManualEgressReleased(g *state.Game, roomName string) bool {
	if g == nil || roomName == "" {
		return false
	}
	return g.ManualEgressReleased != nil && g.ManualEgressReleased[roomName]
}

// ClearRoomPropagatedPower turns off propagated power for a room and removes pending activation.
func ClearRoomPropagatedPower(g *state.Game, roomName string) {
	if g == nil || roomName == "" {
		return
	}
	EnsureRoomPowerOnlineMap(g)
	g.RoomPowerOnline[roomName] = false
	removePendingRoom(g, roomName)
}

// ClearAllPropagatedPower clears every room's online state and pending queue.
func ClearAllPropagatedPower(g *state.Game) {
	if g == nil {
		return
	}
	EnsureRoomPowerOnlineMap(g)
	for name := range g.RoomPowerOnline {
		g.RoomPowerOnline[name] = false
	}
	g.PowerPropPending = nil
}

// PowerNowMs returns the current time in milliseconds for propagation scheduling.
func PowerNowMs() int64 {
	return time.Now().UnixMilli()
}

// NotifyPowerGridChanged reschedules propagation and refreshes supply/consumption/grid terminals.
func NotifyPowerGridChanged(g *state.Game) {
	if g == nil {
		return
	}
	SchedulePowerPropagation(g, PowerNowMs())
	g.UpdatePowerSupply()
	g.PowerConsumption = g.CalculatePowerConsumption()
	ApplyGridConductivePower(g)
}

// PropagateRoomPowerOnlineFromGenerators recomputes RoomPowerOnline from powered generators through
// live conduits only. Rooms separated by unpowered segments stay offline even when armed elsewhere.
func PropagateRoomPowerOnlineFromGenerators(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	EnsureRoomPowerOnlineMap(g)
	if !anyGeneratorPowered(g) {
		ClearAllPropagatedPower(g)
		return
	}
	for name := range g.RoomPowerOnline {
		g.RoomPowerOnline[name] = false
	}

	cellsReachableViaArmedRoutingFromGenerators(g).Each(func(cell *world.Cell) {
		if cell == nil || cell.Name == "" || cell.Name == "Corridor" {
			return
		}
		if CanTraverseCellForPowerGridArm(g, cell) {
			g.RoomPowerOnline[cell.Name] = true
		}
	})
	for roomName, armed := range g.RoomDoorsPowered {
		if armed && RoomHasLivePower(g, roomName) {
			g.RoomPowerOnline[roomName] = true
		}
	}
}

// SchedulePowerPropagation rebuilds the pending activation queue from powered generators
// through armed rooms (RoomDoorsPowered). Already-online rooms stay online.
func SchedulePowerPropagation(g *state.Game, nowMs int64) {
	if g == nil || g.Grid == nil {
		return
	}
	EnsureRoomPowerOnlineMap(g)
	if !anyGeneratorPowered(g) {
		ClearAllPropagatedPower(g)
		return
	}

	PropagateRoomPowerOnlineFromGenerators(g)
	g.PowerPropPending = nil
}

// AdvancePowerPropagation activates due rooms; returns true if a short-out occurred.
func AdvancePowerPropagation(g *state.Game, nowMs int64) bool {
	if g == nil || len(g.PowerPropPending) == 0 {
		return false
	}
	EnsureRoomPowerOnlineMap(g)
	g.UpdatePowerSupply()

	var remaining []state.PowerPropEntry
	shorted := false

	for _, p := range g.PowerPropPending {
		if p.ActivateAt > nowMs {
			remaining = append(remaining, p)
			continue
		}
		if shorted {
			continue
		}
		if !g.RoomDoorsPowered[p.RoomName] {
			continue
		}
		if g.RoomPowerOnline[p.RoomName] {
			continue
		}
		if roomActivationWouldOverload(g, p.RoomName) {
			g.RoomPowerOnline[p.RoomName] = true
			ResolvePowerOverloadAfterToggle(g, p.RoomName)
			if RoomWouldOverloadGrid(g, p.RoomName) {
				ClearRoomPropagatedPower(g, p.RoomName)
			}
			shorted = true
			continue
		}
		g.RoomPowerOnline[p.RoomName] = true
	}

	if shorted {
		g.PowerPropPending = nil
	} else {
		g.PowerPropPending = remaining
	}
	g.PowerConsumption = g.CalculatePowerConsumption()
	ApplyGridConductivePower(g)
	return shorted
}

func anyGeneratorPowered(g *state.Game) bool {
	if g == nil || g.Grid == nil {
		return false
	}
	found := false
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen != nil && gen.IsPowered() {
			found = true
		}
	})
	return found
}

// roomDepthsFromPoweredGenerators returns BFS hop count from any powered generator through armed power grid cells.
func roomDepthsFromPoweredGenerators(g *state.Game) map[string]int {
	depths := make(map[string]int)
	if g == nil || g.Grid == nil {
		return depths
	}

	type queueEntry struct {
		cell  *world.Cell
		depth int
	}
	visited := make(map[*world.Cell]bool)
	var queue []queueEntry

	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen == nil || !gen.IsPowered() {
			return
		}
		if visited[cell] {
			return
		}
		visited[cell] = true
		queue = append(queue, queueEntry{cell, 0})
	})

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.cell == nil {
			continue
		}
		if cur.depth > 0 && !CanTraverseCellForPowerGridArm(g, cur.cell) {
			continue
		}
		if cur.cell.Name != "" && cur.cell.Name != "Corridor" {
			if d, ok := depths[cur.cell.Name]; !ok || cur.depth < d {
				depths[cur.cell.Name] = cur.depth
			}
		}
		for _, n := range cur.cell.GetNeighbors() {
			if n == nil || !n.Room || visited[n] {
				continue
			}
			if !CanTraverseCellForPowerGridArm(g, n) {
				continue
			}
			visited[n] = true
			queue = append(queue, queueEntry{n, cur.depth + 1})
		}
	}
	return depths
}

func pruneStaleOnlineRooms(g *state.Game) {
	if g == nil || g.RoomPowerOnline == nil {
		return
	}
	depths := roomDepthsFromPoweredGenerators(g)
	for roomName, on := range g.RoomPowerOnline {
		if !on {
			continue
		}
		if !g.RoomDoorsPowered[roomName] {
			ClearRoomPropagatedPower(g, roomName)
			continue
		}
		if _, reachable := depths[roomName]; !reachable {
			ClearRoomPropagatedPower(g, roomName)
		}
	}
}

func removePendingRoom(g *state.Game, roomName string) {
	if g == nil || len(g.PowerPropPending) == 0 {
		return
	}
	var kept []state.PowerPropEntry
	for _, p := range g.PowerPropPending {
		if p.RoomName != roomName {
			kept = append(kept, p)
		}
	}
	g.PowerPropPending = kept
}

func roomActivationWouldOverload(g *state.Game, roomName string) bool {
	if g == nil {
		return false
	}
	return ConsumptionIfRoomCameOnline(g, roomName) > ArmedGridSupplyForRoom(g, roomName)
}
