// Package gameplay provides core game logic for player movement and interactions.
package gameplay

import (
	"fmt"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// ShowMovementHint shows a callout hint next to the player for movement controls
// Only shows hint if the player has moved fewer than 3 times
func ShowMovementHint(g *state.Game) {
	// Only show hint for the first 3 movements
	if g.MovementCount >= 3 {
		return
	}

	// Show hint next to the player
	if g.CurrentCell != nil {
		renderer.AddCallout(g.CurrentCell.Row, g.CurrentCell.Col, engineinput.HintMove(), renderer.CalloutColorInfo, 0)
	}
}

// ShowInteractableHints shows callout hints for interactable objects adjacent to the player
// Only shows hints if the player has interacted with fewer than 3 objects
func ShowInteractableHints(g *state.Game) {
	// Only show hints for the first 3 interactions
	if g.InteractionsCount >= 3 {
		return
	}

	// Match CheckAdjacentInteractables: prefer generators across all directions, then other types in N,S,E,W.
	neighbors := []*world.Cell{
		g.CurrentCell.North,
		g.CurrentCell.South,
		g.CurrentCell.East,
		g.CurrentCell.West,
	}

	for _, cell := range neighbors {
		if cell == nil {
			continue
		}
		if gameworld.HasGenerator(cell) {
			renderer.AddCallout(cell.Row, cell.Col, engineinput.HintInteractPrefix(), renderer.CalloutColorInfo, 3000)
			return
		}
	}

	for _, cell := range neighbors {
		if cell == nil {
			continue
		}

		// Check for interactables that are still interactable (not already used/checked)
		if gameworld.HasFurniture(cell) {
			furniture := gameworld.GetGameData(cell).Furniture
			if furniture.IsChecked() {
				// Furniture already checked: show normal description tooltip
				calloutText := fmt.Sprintf("%s\n%s", furnitureCalloutHeading(furniture.Name), furnitureCalloutBody(furniture.Description))
				renderer.AddCallout(cell.Row, cell.Col, calloutText, renderer.CalloutColorFurnitureChecked, 0)
			} else {
				// Furniture not checked yet: show interaction hint (only for first 3 interactions)
				renderer.AddCallout(cell.Row, cell.Col, engineinput.HintInteractPrefix(), renderer.CalloutColorInfo, 3000)
			}
			return // Only show one hint at a time
		}
		if gameworld.HasUnusedTerminal(cell) {
			// Terminal is unused, show hint
			renderer.AddCallout(cell.Row, cell.Col, engineinput.HintInteractPrefix(), renderer.CalloutColorInfo, 3000)
			return
		}
		if gameworld.HasUnsolvedPuzzle(cell) {
			// Puzzle is unsolved, show hint
			renderer.AddCallout(cell.Row, cell.Col, engineinput.HintInteractPrefix(), renderer.CalloutColorInfo, 3000)
			return
		}
		if gameworld.HasInactiveHazardControl(cell) {
			// Hazard control is inactive, show hint
			renderer.AddCallout(cell.Row, cell.Col, engineinput.HintInteractPrefix(), renderer.CalloutColorInfo, 3000)
			return
		}
		if gameworld.HasMaintenanceTerminal(cell) {
			renderer.AddCallout(cell.Row, cell.Col, engineinput.HintInteractPrefix(), renderer.CalloutColorInfo, 3000)
			return
		}
	}
}
