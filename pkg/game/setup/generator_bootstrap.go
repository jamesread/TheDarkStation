package setup

import (
	"fmt"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// BootstrapGeneratorRoom arms routing and energizes the room containing a powered generator.
func BootstrapGeneratorRoom(g *state.Game, cell *world.Cell) {
	if g == nil || cell == nil || !cell.Room || cell.Name == "" || cell.Name == "Corridor" {
		return
	}
	gen := gameworld.GetGameData(cell).Generator
	if gen == nil || !gen.IsPowered() {
		return
	}
	g.RoomDoorsPowered[cell.Name] = true
	EnsureRoomPowerOnlineMap(g)
	// RoomPowerOnline is derived from live conduit reachability (see PropagateRoomPowerOnlineFromGenerators).
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
	if start := g.Grid.StartCell(); start != nil && start.Name != "" {
		if placeMaintenanceTerminalInRoom(g, start.Name, true) {
			return
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
	var target *world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if target != nil || cell == nil || !cell.Room || cell.Name != roomName || cell.ExitCell {
			return
		}
		if cell == g.Grid.StartCell() {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Generator != nil || data.Door != nil || data.MaintenanceTerm != nil {
			return
		}
		if !force && !CanPlaceBlockingEntity(g, cell) {
			return
		}
		target = cell
	})
	if target == nil && force {
		g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
			if target != nil || cell == nil || !cell.Room || cell.Name != roomName || cell.ExitCell {
				return
			}
			if cell == g.Grid.StartCell() {
				return
			}
			data := gameworld.GetGameData(cell)
			if data.Generator != nil || data.Door != nil || data.MaintenanceTerm != nil {
				return
			}
			target = cell
		})
	}
	if target == nil {
		return false
	}
	term := entities.NewMaintenanceTerminal(fmt.Sprintf("Maintenance Terminal - %s", roomName), roomName)
	gameworld.GetGameData(target).MaintenanceTerm = term
	return true
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
		if mt == nil || mt.Powered {
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
		if mt == nil || mt.Powered {
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

// BootstrapPoweredGenerators refreshes generator-room routing and conductive terminal feed
// after a generator comes online mid-game (does not re-arm doors the player turned off).
func BootstrapPoweredGenerators(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	PropagateRoomPowerOnlineFromGenerators(g)
	ApplyGridConductivePower(g)
}
