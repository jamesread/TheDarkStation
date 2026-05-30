package setup

import (
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
)

// ExitCellHasLivePower reports whether the exit lift cell has propagated grid power,
// or the exit room has manual egress release (same rules as powered doors).
func ExitCellHasLivePower(g *state.Game) bool {
	if g == nil || g.Grid == nil {
		return false
	}
	exit := g.Grid.ExitCell()
	if exit == nil || !exit.Room {
		return false
	}
	if CellHasLivePower(g, exit) {
		return true
	}
	if exit.Name != "" && exit.Name != "Corridor" {
		return RoomManualEgressReleased(g, exit.Name)
	}
	return false
}

// ExitLiftState returns the current lift readiness for this deck.
func ExitLiftState(g *state.Game) state.ExitLiftState {
	if g == nil {
		return state.ExitLiftLockedUnpowered
	}
	if !ExitCellHasLivePower(g) {
		return state.ExitLiftLockedUnpowered
	}
	if !g.AllHazardsCleared() {
		return state.ExitLiftLockedIncomplete
	}
	return state.ExitLiftReady
}

// ExitLiftReady reports whether the player may enter and use the exit lift.
func ExitLiftReady(g *state.Game) bool {
	return ExitLiftState(g) == state.ExitLiftReady
}

// ExitCell returns the exit cell when the grid is present.
func ExitCell(g *state.Game) *world.Cell {
	if g == nil || g.Grid == nil {
		return nil
	}
	return g.Grid.ExitCell()
}
