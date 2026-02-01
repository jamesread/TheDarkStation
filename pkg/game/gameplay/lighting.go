// Package gameplay provides core game logic for player movement and interactions.
package gameplay

import (
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// UpdateLightingExploration updates cell exploration based on lighting
func UpdateLightingExploration(g *state.Game) {
	if g.Grid == nil || g.CurrentCell == nil {
		return
	}

	// Calculate total power consumption
	totalConsumption := g.CalculatePowerConsumption()
	g.PowerConsumption = totalConsumption

	// Update power supply from generators
	g.UpdatePowerSupply()

	availablePower := g.GetAvailablePower()

	// Check if power consumption exceeds supply and warn the player
	if g.PowerConsumption > g.PowerSupply && !g.PowerOverloadWarned {
		logMessage(g, "WARNING: Power consumption (ACTION{%d} watts) exceeds supply (ACTION{%d} watts)!", g.PowerConsumption, g.PowerSupply)
		g.PowerOverloadWarned = true
	} else if g.PowerConsumption <= g.PowerSupply {
		// Reset warning flag when power is sufficient
		g.PowerOverloadWarned = false
	}
	playerRow := g.CurrentCell.Row
	playerCol := g.CurrentCell.Col

	// If we have power, turn on lights in visited cells
	// If no power, turn off lights (cells will fade from explored)
	// Exception: cells within 3x3 radius of player always stay visible
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}

		data := gameworld.GetGameData(cell)

		// Calculate distance from player (Manhattan distance for 5x5 radius)
		rowDist := row - playerRow
		colDist := col - playerCol
		if rowDist < 0 {
			rowDist = -rowDist
		}
		if colDist < 0 {
			colDist = -colDist
		}
		// 5x5 radius means max distance of 2 in each direction
		isNearPlayer := rowDist <= 2 && colDist <= 2

		// If cell was visited and we have power, lights should be on
		if cell.Visited && availablePower > 0 {
			if !data.LightsOn {
				data.LightsOn = true
				data.Lighted = true
				// Ensure cell stays explored when lights are on
				cell.Discovered = true
				cell.Visited = true
			}
		} else if availablePower <= 0 {
			// No power - lights off
			data.LightsOn = false

			// Cells near player always stay visible (3x3 radius)
			if isNearPlayer {
				// Keep nearby cells visible even without power
				cell.Discovered = true
				if cell.Visited {
					// Mark as temporarily visible (not permanently lighted)
					// This allows exploration without power
				}
			} else {
				// Far cells fade if not permanently lighted
				if !data.Lighted {
					cell.Discovered = false
					cell.Visited = false
				}
			}
		}
	})
}
