package setup

import (
	"fmt"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// ExitGatingRepairRoomAccessible reports whether roomName contains an init-reachable cell
// or its doors can be toggled from a maintenance terminal in the lift-entry pocket.
func ExitGatingRepairRoomAccessible(g *state.Game, roomName string) bool {
	if g == nil || g.Grid == nil || roomName == "" || roomName == "Corridor" ||
		generator.IsPlacementExcludedRoom(roomName) {
		return false
	}
	reach := InitialReachableCells(g)
	if roomNameInReachable(reach, roomName) {
		return true
	}
	return CanPowerRoomDoorsFromReachable(g, reach, roomName)
}

func exitGatingRepairInteractableAtInit(g *state.Game, cell *world.Cell) bool {
	if cell == nil {
		return false
	}
	return ExitGatingRepairRoomAccessible(g, cell.Name) && entityHasInitReachAdjacentStand(g, cell, nil)
}

func roomNameInReachable(reach *mapset.Set[*world.Cell], roomName string) bool {
	if reach == nil {
		return false
	}
	found := false
	reach.Each(func(c *world.Cell) {
		if c != nil && c.Name == roomName {
			found = true
		}
	})
	return found
}

// ExitGatingRepairsAccessible reports whether every local lift-gating repair sits in a
// room reachable or powerable from the player entry pocket at level init.
func ExitGatingRepairsAccessible(g *state.Game) bool {
	if g == nil || g.Grid == nil {
		return true
	}
	for _, repair := range g.RepairObjectives {
		if repair == nil || repair.SkipExitGate || repair.DeviceRow < 0 || repair.DeviceCol < 0 {
			continue
		}
		// Conduit splices live on corridor conduits by design; their reachability is
		// plain corridor walking, which the progression simulator verifies.
		if repair.Type == entities.RepairConduitSplice {
			continue
		}
		cell := g.Grid.GetCell(repair.DeviceRow, repair.DeviceCol)
		if cell == nil || !ExitGatingRepairRoomAccessible(g, cell.Name) {
			return false
		}
	}
	return true
}

// EnsureExitGatingRepairReachability relocates exit-gating repair devices that were placed
// outside the init-reachable or init-powerable pocket (I1 safety net after solvability fixes).
func EnsureExitGatingRepairReachability(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	for _, repair := range g.RepairObjectives {
		if repair == nil || repair.SkipExitGate || repair.Type == entities.RepairWastePump ||
			repair.Type == entities.RepairConduitSplice {
			continue
		}
		if repair.DeviceRow < 0 || repair.DeviceCol < 0 {
			continue
		}
		cell := g.Grid.GetCell(repair.DeviceRow, repair.DeviceCol)
		if cell == nil || exitGatingRepairInteractableAtInit(g, cell) {
			continue
		}
		replacement := findExitGatingRepairRelocationCell(g, cell, repair)
		if replacement == nil {
			continue
		}
		pendingKeycard := gameworld.GetGameData(cell).PendingUnlockKeycard
		gameworld.GetGameData(cell).RepairDevice = nil
		gameworld.GetGameData(replacement).RepairDevice = repair
		if pendingKeycard != "" {
			gameworld.GetGameData(cell).PendingUnlockKeycard = ""
			gameworld.GetGameData(replacement).PendingUnlockKeycard = pendingKeycard
		}
		repair.RoomName = replacement.Name
		repair.DeviceRow = replacement.Row
		repair.DeviceCol = replacement.Col
	}
}

func findExitGatingRepairRelocationCell(g *state.Game, avoid *world.Cell, repair *entities.RepairObjective) *world.Cell {
	if g == nil || g.Grid == nil {
		return nil
	}
	entry := PlayerEntryCell(g)
	var best *world.Cell
	bestDist := int(^uint(0) >> 1)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || cell == avoid || !validExitGatingRepairRelocationCell(g, cell, repair) {
			return
		}
		dist := manhattanDistance(entry, cell)
		if dist < bestDist || (dist == bestDist && cellLess(cell, best)) {
			bestDist = dist
			best = cell
		}
	})
	return best
}

func validExitGatingRepairRelocationCell(g *state.Game, cell *world.Cell, repair *entities.RepairObjective) bool {
	if cell == nil || !cell.Room || cell == PlayerEntryCell(g) || cell.ExitCell {
		return false
	}
	if !ExitGatingRepairRoomAccessible(g, cell.Name) {
		return false
	}
	if !entityHasInitReachAdjacentStand(g, cell, nil) {
		return false
	}
	data := gameworld.GetGameData(cell)
	if data.RepairDevice != nil && data.RepairDevice != repair {
		return false
	}
	if data.Generator != nil || data.Door != nil || data.Terminal != nil || data.Puzzle != nil ||
		data.Furniture != nil || data.Hazard != nil || data.HazardControl != nil ||
		data.MaintenanceTerm != nil || data.RepairBlocker != nil || cell.ItemsOnFloor.Size() > 0 {
		return false
	}
	return CanPlaceBlockingEntity(g, cell)
}

func cellLess(a, b *world.Cell) bool {
	if b == nil {
		return true
	}
	if a.Row != b.Row {
		return a.Row < b.Row
	}
	return a.Col < b.Col
}

func warnExitGatingRepairInaccessible(g *state.Game) []string {
	if g == nil || g.Grid == nil {
		return nil
	}
	var out []string
	for _, repair := range g.RepairObjectives {
		if repair == nil || repair.SkipExitGate || repair.DeviceRow < 0 || repair.DeviceCol < 0 ||
			repair.Type == entities.RepairConduitSplice {
			continue
		}
		cell := g.Grid.GetCell(repair.DeviceRow, repair.DeviceCol)
		if cell == nil {
			continue
		}
		if !ExitGatingRepairRoomAccessible(g, cell.Name) {
			out = append(out, fmt.Sprintf(
				"exit-gating repair %q at x:%d y:%d in %q not reachable/powerable from lift entry",
				repair.Name, cell.Col, cell.Row, cell.Name))
		}
	}
	return out
}
