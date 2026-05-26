package gameplay

import (
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
)

// ClearGeneratorPowerGridOverlay hides the generator-toggled power grid overlay.
func ClearGeneratorPowerGridOverlay(g *state.Game) {
	if g == nil {
		return
	}
	g.PowerGridOverlayActive = false
	g.PowerGridOverlaySeedRow = -1
	g.PowerGridOverlaySeedCol = -1
}

// ToggleGeneratorPowerGridOverlay shows or hides the power grid from the given generator cell.
func ToggleGeneratorPowerGridOverlay(g *state.Game, cell *world.Cell) {
	if g == nil || cell == nil {
		return
	}
	if g.PowerGridOverlayActive &&
		g.PowerGridOverlaySeedRow == cell.Row &&
		g.PowerGridOverlaySeedCol == cell.Col {
		ClearGeneratorPowerGridOverlay(g)
		return
	}
	g.PowerGridOverlayActive = true
	g.PowerGridOverlaySeedRow = cell.Row
	g.PowerGridOverlaySeedCol = cell.Col
}
