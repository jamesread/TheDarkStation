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
	if start := grid.StartCell(); start != nil && start.Name == generator.ShipRoomName {
		return start
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
	if g == nil || g.Grid == nil {
		return
	}
	seen := map[*world.Cell]bool{}
	for _, entry := range entryDoorPowerSeeds(g) {
		if entry == nil || seen[entry] {
			continue
		}
		seen[entry] = true
		powerUnpoweredEgressDoorsAdjacent(g, entry)
	}
}

func entryDoorPowerSeeds(g *state.Game) []*world.Cell {
	var seeds []*world.Cell
	if entry := PlayerEntryCell(g); entry != nil {
		seeds = append(seeds, entry)
	}
	if g == nil || g.Grid == nil {
		return seeds
	}
	if exit := g.Grid.ExitCell(); exit != nil {
		dup := false
		for _, s := range seeds {
			if s == exit {
				dup = true
				break
			}
		}
		if !dup {
			seeds = append(seeds, exit)
		}
	}
	return seeds
}

func powerUnpoweredEgressDoorsAdjacent(g *state.Game, entry *world.Cell) {
	if g == nil || entry == nil {
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
	if g == nil || g.Grid == nil {
		return
	}
	entry := g.Grid.ExitCell()
	if entry == nil {
		entry = PlayerEntryCell(g)
	}
	if entry == nil {
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
