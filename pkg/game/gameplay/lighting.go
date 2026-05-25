// Package gameplay provides core game logic for player movement and interactions.
package gameplay

import (
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// UpdateLightingExploration recalculates power supply/consumption and applies exploration state.
// Per-room lighting/fog is disabled for now: all room cells are treated as always lit.
func UpdateLightingExploration(g *state.Game) {
	if g.Grid == nil || g.CurrentCell == nil {
		return
	}

	totalConsumption := g.CalculatePowerConsumption()
	g.PowerConsumption = totalConsumption
	g.UpdatePowerSupply()

	if g.PowerConsumption > g.PowerSupply && !g.PowerOverloadWarned {
		logMessage(g, "WARNING: Power consumption (%dw) exceeds supply (%dw)!", g.PowerConsumption, g.PowerSupply)
		g.PowerOverloadWarned = true
	} else if g.PowerConsumption <= g.PowerSupply {
		g.PowerOverloadWarned = false
	}

	applyAlwaysLit(g)
}

func applyAlwaysLit(g *state.Game) {
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		data := gameworld.GetGameData(cell)
		data.LightsOn = true
		data.Lighted = true
	})
}
