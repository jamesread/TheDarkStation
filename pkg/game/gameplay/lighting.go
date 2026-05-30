// Package gameplay provides core game logic for player movement and interactions.
package gameplay

import (
	"time"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// UpdateLightingExploration recalculates power supply/consumption and applies exploration state.
// Per-room lighting/fog is disabled for now: all room cells are treated as always lit.
func UpdateLightingExploration(g *state.Game) {
	if g.Grid == nil || g.CurrentCell == nil {
		return
	}

	nowMs := time.Now().UnixMilli()
	if len(g.Generators) > 0 {
		setup.AdvancePowerPropagation(g, nowMs)
	}
	setup.AdvanceRoomPowerOff(g, nowMs)

	totalConsumption := g.CalculatePowerConsumption()
	g.PowerConsumption = totalConsumption
	g.UpdatePowerSupply()
	setup.ApplyGridConductivePower(g)

	if setup.AnyArmedGridOverloaded(g) && !g.PowerOverloadWarned {
		logMessage(g, "WARNING: Power consumption exceeds supply on a power grid!")
		g.PowerOverloadWarned = true
	} else if !setup.AnyArmedGridOverloaded(g) {
		g.PowerOverloadWarned = false
	}

	applyAlwaysLit(g)
}

func applyAlwaysLit(g *state.Game) {
	if g.AlwaysLitApplied || g.Grid == nil {
		return
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		data := gameworld.GetGameData(cell)
		data.LightsOn = true
		data.Lighted = true
	})
	g.AlwaysLitApplied = true
}
