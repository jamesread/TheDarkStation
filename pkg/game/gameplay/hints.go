// Package gameplay provides core game logic for player movement and interactions.
package gameplay

import (
	"fmt"

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
		renderer.AddCallout(g.CurrentCell.Row, g.CurrentCell.Col, "Press WASD or arrow keys to move", renderer.CalloutColorInfo, 0)
	}
}

// ShowInteractableHints shows callout hints for interactable objects adjacent to the player
// Only shows hints if the player has interacted with fewer than 3 objects
func ShowInteractableHints(g *state.Game) {
	// Only show hints for the first 3 interactions
	if g.InteractionsCount >= 3 {
		return
	}

	// Check adjacent cells for interactable objects
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

		// Check for interactables that are still interactable (not already used/checked)
		// Priority order: furniture, terminals, puzzles, hazard controls
		if gameworld.HasFurniture(cell) {
			furniture := gameworld.GetGameData(cell).Furniture
			if furniture.IsChecked() {
				// Furniture already checked: show normal description tooltip
				calloutText := fmt.Sprintf("FURNITURE{%s}\n%s", furniture.Name, furniture.Description)
				renderer.AddCallout(cell.Row, cell.Col, calloutText, renderer.CalloutColorFurnitureChecked, 0)
			} else {
				// Furniture not checked yet: show interaction hint (only for first 3 interactions)
				renderer.AddCallout(cell.Row, cell.Col, "Press E/Enter to interact", renderer.CalloutColorInfo, 3000)
			}
			return // Only show one hint at a time
		}
		if gameworld.HasUnusedTerminal(cell) {
			// Terminal is unused, show hint
			renderer.AddCallout(cell.Row, cell.Col, "Press E/Enter to interact", renderer.CalloutColorInfo, 3000)
			return
		}
		if gameworld.HasUnsolvedPuzzle(cell) {
			// Puzzle is unsolved, show hint
			renderer.AddCallout(cell.Row, cell.Col, "Press E/Enter to interact", renderer.CalloutColorInfo, 3000)
			return
		}
		if gameworld.HasGenerator(cell) {
			// Generator, show hint
			renderer.AddCallout(cell.Row, cell.Col, "Press E/Enter to interact", renderer.CalloutColorInfo, 3000)
			return
		}
		if gameworld.HasInactiveHazardControl(cell) {
			// Hazard control is inactive, show hint
			renderer.AddCallout(cell.Row, cell.Col, "Press E/Enter to interact", renderer.CalloutColorInfo, 3000)
			return
		}
		if gameworld.HasMaintenanceTerminal(cell) {
			// Maintenance terminal, show hint
			renderer.AddCallout(cell.Row, cell.Col, "Press E/Enter to interact", renderer.CalloutColorInfo, 3000)
			return
		}
	}
}
