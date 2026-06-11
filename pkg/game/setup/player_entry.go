package setup

import (
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// PlayerEntryCellForGrid returns where the player begins on this deck layout.
func PlayerEntryCellForGrid(grid *world.Grid) *world.Cell {
	if grid == nil {
		return nil
	}
	if entry := grid.ExitCell(); entry != nil {
		return entry
	}
	return grid.StartCell()
}

// PlayerEntryCell returns where the player begins on this deck (lift shaft center when carved).
func PlayerEntryCell(g *state.Game) *world.Cell {
	if g == nil || g.Grid == nil {
		return nil
	}
	return PlayerEntryCellForGrid(g.Grid)
}

// EnsureEntryAdjacentDoorPower powers doors on rooms bordering the lift entry when those
// doors are the only init egress from the shaft pocket.
func EnsureEntryAdjacentDoorPower(g *state.Game) {
	entry := PlayerEntryCell(g)
	if g == nil || g.Grid == nil || entry == nil {
		return
	}
	for _, n := range entry.GetNeighbors() {
		if n == nil || !gameworld.HasDoor(n) {
			continue
		}
		ok, reason := CanEnterCellAtInit(g, n)
		if ok || reason != MovementUnpoweredDoor {
			continue
		}
		roomName := gameworld.GetGameData(n).Door.RoomName
		if roomName != "" {
			g.RoomDoorsPowered[roomName] = true
		}
	}
}

// can step off the exit cell into the shaft at level start.
func EnsureLiftShaftEntryClearance(g *state.Game) {
	entry := PlayerEntryCell(g)
	if g == nil || g.Grid == nil || entry == nil {
		return
	}
	for _, n := range entry.GetNeighbors() {
		if n == nil || n.Name != generator.ShaftRoomName {
			continue
		}
		if IsPermanentlyBlockingCell(n) {
			clearPermanentBlocker(g, n)
		}
		data := gameworld.GetGameData(n)
		if data.RepairBlocker != nil {
			data.RepairBlocker = nil
		}
	}
}
