package setup

import (
	"fmt"
	"strings"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// BootstrapGeneratorRoom turns the room circuit ON (doors + CCTV) for a powered generator on cell.
func BootstrapGeneratorRoom(g *state.Game, cell *world.Cell) {
	if g == nil || cell == nil || !cell.Room || cell.Name == "" || cell.Name == "Corridor" {
		return
	}
	gen := gameworld.GetGameData(cell).Generator
	if gen == nil || !gen.IsPowered() {
		return
	}
	armGeneratorRoomCircuit(g, cell.Name)
	if paired := shaftFragmentPairName(cell.Name); paired != "" {
		armGeneratorRoomCircuit(g, paired)
	}
	EnsureRoomPowerOnlineMap(g)
}

const shaftFarSuffix = " Far"

// shaftFragmentPairName returns the paired base/Far room name when a generator room was split
// by the lift shaft carve (e.g. "Lab" ↔ "Lab Far").
func shaftFragmentPairName(roomName string) string {
	if roomName == "" || roomName == "Corridor" {
		return ""
	}
	if strings.HasSuffix(roomName, shaftFarSuffix) {
		return roomName[:len(roomName)-len(shaftFarSuffix)]
	}
	return roomName + shaftFarSuffix
}

func armGeneratorRoomCircuit(g *state.Game, roomName string) {
	if g == nil || roomName == "" || roomName == "Corridor" {
		return
	}
	if g.RoomDoorsPowered == nil {
		g.RoomDoorsPowered = make(map[string]bool)
	}
	if g.RoomCCTVPowered == nil {
		g.RoomCCTVPowered = make(map[string]bool)
	}
	g.RoomDoorsPowered[roomName] = true
	g.RoomCCTVPowered[roomName] = true
}

// EnsureGeneratorRoomBootstrap arms routing for every generator room, energizes rooms whose
// generators are already online, and applies conductive power to maintenance terminals on
// those generator power grids. Prevents maint-terminal bootstrap deadlocks without requiring a
// powered terminal in the start room.
func EnsureGeneratorRoomBootstrap(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	EnsureMinimumMaintenanceCoverage(g)
	EnsureGeneratorRoomMaintenanceTerminal(g)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		if gameworld.GetGameData(cell).Generator != nil {
			BootstrapGeneratorRoom(g, cell)
		}
	})
	PropagateRoomPowerOnlineFromGenerators(g)
	ApplyGridConductivePower(g)
	EnsureMaintTerminalBootstrapFallback(g)
}

// EnsureMinimumMaintenanceCoverage guarantees at least one maintenance terminal exists when
// the level would otherwise have none (all R8 placements skipped).
func EnsureMinimumMaintenanceCoverage(g *state.Game) {
	if g == nil || g.Grid == nil || countMaintenanceTerminals(g) > 0 {
		return
	}
	if entry := PlayerEntryCell(g); entry != nil && entry.Name != "" {
		for _, roomName := range GetAdjacentRoomNames(g.Grid, entry.Name) {
			if roomName == entry.Name || roomName == "Corridor" {
				continue
			}
			if placeMaintenanceTerminalInRoom(g, roomName, true) {
				return
			}
		}
	}
	placed := false
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if placed || cell == nil || !cell.Room {
			return
		}
		if gameworld.GetGameData(cell).Generator == nil {
			return
		}
		if placeMaintenanceTerminalInRoom(g, cell.Name, true) {
			placed = true
			return
		}
		for _, adj := range GetAdjacentRoomNames(g.Grid, cell.Name) {
			if placeMaintenanceTerminalInRoom(g, adj, true) {
				placed = true
				return
			}
		}
	})
}

func countMaintenanceTerminals(g *state.Game) int {
	if g == nil || g.Grid == nil {
		return 0
	}
	n := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		if gameworld.GetGameData(cell).MaintenanceTerm != nil {
			n++
		}
	})
	return n
}

// EnsureGeneratorRoomMaintenanceTerminal places a maintenance terminal in any generator room
// that lacks one (R8 can skip the start room during normal placement).
func EnsureGeneratorRoomMaintenanceTerminal(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		if gameworld.GetGameData(cell).Generator == nil {
			return
		}
		roomName := cell.Name
		if roomName == "" || roomName == "Corridor" {
			return
		}
		if roomHasMaintenanceTerminal(g, roomName) {
			return
		}
		if placeMaintenanceTerminalInRoom(g, roomName, false) {
			return
		}
		for _, adj := range GetAdjacentRoomNames(g.Grid, roomName) {
			if placeMaintenanceTerminalInRoom(g, adj, true) {
				return
			}
		}
	})
}

func roomHasMaintenanceTerminal(g *state.Game, roomName string) bool {
	if g == nil || g.Grid == nil || roomName == "" {
		return false
	}
	found := false
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if found || cell == nil || cell.Name != roomName {
			return
		}
		if gameworld.GetGameData(cell).MaintenanceTerm != nil {
			found = true
		}
	})
	return found
}

func placeMaintenanceTerminalInRoom(g *state.Game, roomName string, force bool) bool {
	if g == nil || g.Grid == nil || roomName == "" {
		return false
	}
	if roomHasMaintenanceTerminal(g, roomName) {
		return true
	}
	target := firstMaintTerminalCell(g, roomName, false)
	if target == nil && force {
		target = firstMaintTerminalCell(g, roomName, true)
	}
	if target == nil && force {
		if entry := PlayerEntryCell(g); entry != nil && entry.Name == roomName && !entry.ExitCell {
			data := gameworld.GetGameData(entry)
			if data.MaintenanceTerm == nil && data.Generator == nil {
				target = entry
			}
		}
	}
	if target == nil {
		return false
	}
	term := entities.NewMaintenanceTerminal(fmt.Sprintf("Maintenance Terminal - %s", roomName), roomName)
	gameworld.GetGameData(target).MaintenanceTerm = term
	return true
}

func firstMaintTerminalCell(g *state.Game, roomName string, force bool) *world.Cell {
	for _, cell := range maintTerminalCandidateCells(g, roomName) {
		if cell == nil || !cell.Room || cell.Name != roomName || cell.ExitCell {
			continue
		}
		if cell == PlayerEntryCell(g) {
			continue
		}
		data := gameworld.GetGameData(cell)
		if data.Generator != nil || data.Door != nil || data.MaintenanceTerm != nil ||
			data.Terminal != nil || data.Puzzle != nil || data.Furniture != nil ||
			data.Hazard != nil || data.HazardControl != nil || data.RepairDevice != nil ||
			data.RepairBlocker != nil || cell.ItemsOnFloor.Size() > 0 {
			continue
		}
		if !force && !CanPlaceBlockingEntity(g, cell) {
			continue
		}
		return cell
	}
	return nil
}

func maintTerminalCandidateCells(g *state.Game, roomName string) []*world.Cell {
	if g == nil || g.Grid == nil {
		return nil
	}
	if roomName == generator.ShaftRoomName {
		return LiftShaftCellsFromBottomLeft(g)
	}
	var out []*world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.Room && cell.Name == roomName {
			out = append(out, cell)
		}
	})
	SortCellsByPosition(out)
	return out
}

// ForceMaintBootstrapOK guarantees a powered maintenance terminal when generators are online.
func ForceMaintBootstrapOK(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	hasPoweredGen := false
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if hasPoweredGen || cell == nil || !cell.Room {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen != nil && gen.IsPowered() {
			hasPoweredGen = true
		}
	})
	if !hasPoweredGen {
		return
	}
	if CountPoweredMaintenanceTerminals(g) > 0 {
		return
	}
	EnsureMinimumMaintenanceCoverage(g)
	EnsureMaintTerminalBootstrapFallback(g)
	if CountPoweredMaintenanceTerminals(g) == 0 {
		powerFirstUnpoweredMaintenanceTerminal(g)
	}
}

// EnsureMaintTerminalBootstrapFallback powers at least one maintenance terminal reachable from
// each powered generator when conductive local feed alone is insufficient (e.g. generator
// room has no terminal). Uses armed power grid first, then adjacent rooms.
func EnsureMaintTerminalBootstrapFallback(g *state.Game) {
	if g == nil || g.Grid == nil || MaintBootstrapOK(g) {
		return
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen == nil || !gen.IsPowered() {
			return
		}
		if powerMaintTerminalsInGrid(g, ConductiveGridFromSeed(g, cell)) > 0 {
			return
		}
		if powerMaintTerminalsInGrid(g, CellsReachableInPowerGrid(g, cell)) > 0 {
			return
		}
		for _, adj := range GetAdjacentRoomNames(g.Grid, cell.Name) {
			if powerFirstMaintTerminalInRoom(g, adj) {
				return
			}
		}
	})
	if !MaintBootstrapOK(g) {
		powerFirstUnpoweredMaintenanceTerminal(g)
	}
}

func powerFirstUnpoweredMaintenanceTerminal(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	powered := false
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if powered || cell == nil {
			return
		}
		mt := gameworld.GetGameData(cell).MaintenanceTerm
		if mt == nil || mt.Powered || mt.Disabled {
			return
		}
		mt.Powered = true
		powered = true
	})
}

func powerFirstMaintTerminalInRoom(g *state.Game, roomName string) bool {
	if g == nil || g.Grid == nil || roomName == "" {
		return false
	}
	powered := false
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if powered || cell == nil || cell.Name != roomName {
			return
		}
		mt := gameworld.GetGameData(cell).MaintenanceTerm
		if mt == nil || mt.Powered || mt.Disabled {
			return
		}
		mt.Powered = true
		powered = true
	})
	return powered
}

// CountPoweredMaintenanceTerminals returns maintenance terminals with Powered==true.
func CountPoweredMaintenanceTerminals(g *state.Game) int {
	if g == nil || g.Grid == nil {
		return 0
	}
	n := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		mt := gameworld.GetGameData(cell).MaintenanceTerm
		if mt != nil && mt.Powered {
			n++
		}
	})
	return n
}

// MaintBootstrapOK reports whether at least one maintenance terminal is powered on a
// conductive generator power grid while any generator is online.
func MaintBootstrapOK(g *state.Game) bool {
	if g == nil || g.Grid == nil {
		return true
	}
	hasPoweredGen := false
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen != nil && gen.IsPowered() {
			hasPoweredGen = true
		}
	})
	if !hasPoweredGen {
		return true
	}
	return CountPoweredMaintenanceTerminals(g) > 0
}

// BootstrapPoweredGenerators refreshes routing and conductive terminal feed after a generator
// comes online mid-game. When poweredCell is set, that generator room's circuit is turned ON.
func BootstrapPoweredGenerators(g *state.Game, poweredCell *world.Cell) {
	if g == nil || g.Grid == nil {
		return
	}
	if poweredCell != nil {
		BootstrapGeneratorRoom(g, poweredCell)
	}
	NotifyPowerGridChanged(g)
}
