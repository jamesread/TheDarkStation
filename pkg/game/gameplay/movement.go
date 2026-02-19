// Package gameplay provides core game logic for player movement and interactions.
package gameplay

import (
	"fmt"
	"strings"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// CanEnter checks if the player can enter a cell
func CanEnter(g *state.Game, r *world.Cell, logReason bool) (bool, *world.ItemSet) {
	missingItems := mapset.New[*world.Item]()

	if r == nil || !r.Room {
		return false, &missingItems
	}

	// Check for door (block if room's doors are unpowered)
	if gameworld.HasDoor(r) {
		rData := gameworld.GetGameData(r)
		roomName := rData.Door.RoomName
		if !g.RoomDoorsPowered[roomName] {
			if logReason {
				renderer.AddCallout(r.Row, r.Col, fmt.Sprintf("UNPOWERED{Unpowered door}\n%s", rData.Door.DoorName()), renderer.CalloutColorDoor, 0)
			}
			return false, &missingItems
		}
	}

	// Check for locked door
	if gameworld.HasLockedDoor(r) {
		rData := gameworld.GetGameData(r)
		keycardName := rData.Door.KeycardName()
		hasKeycard := false
		var keycardItem *world.Item

		g.OwnedItems.Each(func(item *world.Item) {
			if item.Name == keycardName {
				hasKeycard = true
				keycardItem = item
			}
		})

		if hasKeycard {
			// Unlock ALL doors that require this keycard (a room may have multiple entry points)
			doorsUnlocked := 0
			g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
				cellData := gameworld.GetGameData(cell)
				if gameworld.HasLockedDoor(cell) && cellData.Door.KeycardName() == keycardName {
					cellData.Door.Unlock()
					doorsUnlocked++
				}
			})
			g.OwnedItems.Remove(keycardItem)
			// Show unlock message as callout with proper item colors
			var calloutMsg string
			if doorsUnlocked > 1 {
				calloutMsg = fmt.Sprintf("Used ITEM{%s} to unlock ACTION{%d} doors to ROOM{%s}!", keycardName, doorsUnlocked, rData.Door.RoomName)
			} else {
				calloutMsg = fmt.Sprintf("Used ITEM{%s} to unlock the %s!", keycardName, rData.Door.DoorName())
			}
			renderer.AddCallout(r.Row, r.Col, calloutMsg, renderer.CalloutColorItem, 0)
		} else {
			if logReason {
				logMessage(g, "This door requires a %s", renderer.StyledKeycard(keycardName))
				// Contextual tooltip next to the locked door
				renderer.AddCallout(r.Row, r.Col, fmt.Sprintf("TITLE{Door Locked}\nNeeds: ITEM{%s}", keycardName), renderer.CalloutColorDoor, 0)
			}
			return false, &missingItems
		}
	}

	// Check for generator (blocks movement)
	if gameworld.HasGenerator(r) {
		return false, &missingItems
	}

	// Check for furniture (blocks movement)
	if gameworld.HasFurniture(r) {
		return false, &missingItems
	}

	// Check for CCTV terminals (blocks movement)
	if gameworld.HasTerminal(r) {
		return false, &missingItems
	}

	// Check for puzzle terminals (blocks movement)
	if gameworld.HasPuzzle(r) {
		return false, &missingItems
	}

	// Check for maintenance terminals (blocks movement)
	if gameworld.HasMaintenanceTerminal(r) {
		return false, &missingItems
	}

	// Check for hazard controls / circuit breaker switches (blocks movement, like furniture)
	if gameworld.HasHazardControl(r) {
		return false, &missingItems
	}

	// Check for environmental hazard
	if gameworld.HasBlockingHazard(r) {
		hazard := gameworld.GetGameData(r).Hazard
		if hazard.RequiresItem() {
			// Check if player has the required item
			itemName := hazard.RequiredItemName()
			var fixItem *world.Item
			g.OwnedItems.Each(func(item *world.Item) {
				if item.Name == itemName {
					fixItem = item
				}
			})

			if fixItem != nil {
				// Use the item to fix the hazard
				hazard.Fix()
				g.OwnedItems.Remove(fixItem)
				info := entities.HazardTypes[hazard.Type]
				logMessage(g, "%s", info.FixedMessage)
			} else {
				if logReason {
					// Show hazard description as 2-line callout: first line in hazard color, second line with hint in normal color
					hazardCallout := formatHazardCallout(hazard)
					renderer.AddCallout(r.Row, r.Col, hazardCallout, renderer.CalloutColorHazard, 0)
				}
				return false, &missingItems
			}
		} else {
			// Hazard requires a control panel to be activated
			if logReason {
				// Show hazard description as 2-line callout: first line in hazard color, second line with hint in normal color
				hazardCallout := formatHazardCallout(hazard)
				renderer.AddCallout(r.Row, r.Col, hazardCallout, renderer.CalloutColorHazard, 0)
			}
			return false, &missingItems
		}
	}

	// Check for powered generators and cleared hazards (only for exit cell)
	if r.ExitCell {
		if !g.AllGeneratorsPowered() {
			if logReason {
				unpowered := g.UnpoweredGeneratorCount()
				logMessage(g, "The lift requires all generators to be powered!")
				logMessage(g, "ACTION{%d} generator(s) still need power.", unpowered)
			}
			return false, &missingItems
		}
		if !g.AllHazardsCleared() {
			if logReason {
				numHazards := 0
				g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
					if gameworld.HasBlockingHazard(cell) {
						numHazards++
					}
				})
				logMessage(g, "The lift requires all environmental hazards to be cleared!")
				logMessage(g, "ACTION{%d} environmental hazard(s) remain.", numHazards)
			}
			return false, &missingItems
		}
	}

	return true, &missingItems
}

// MoveCell moves the player to a new cell
func MoveCell(g *state.Game, requestedCell *world.Cell) {
	// Determine direction for debounce animation
	var direction string
	if g.CurrentCell != nil {
		if requestedCell == g.CurrentCell.North {
			direction = "north"
		} else if requestedCell == g.CurrentCell.South {
			direction = "south"
		} else if requestedCell == g.CurrentCell.East {
			direction = "east"
		} else if requestedCell == g.CurrentCell.West {
			direction = "west"
		}
	}

	if res, _ := CanEnter(g, requestedCell, true); res {
		// Check if lights are on - if not, cells won't stay explored
		cellData := gameworld.GetGameData(requestedCell)
		if cellData.LightsOn {
			requestedCell.Visited = true
			cellData.Lighted = true
		} else {
			// If lights are off, only mark as visited temporarily
			requestedCell.Visited = true
		}

		// Reveal cells within field of view (radius 3, with line-of-sight blocking)
		world.RevealFOVDefault(g.Grid, requestedCell)

		// Ensure 3x3 radius cells are always visible (even without power)
		ensureNearbyCellsVisible(g, requestedCell)

		// Update lighting-based exploration
		UpdateLightingExploration(g)

		// Reset interaction order when player moves
		if g.CurrentCell == nil || g.CurrentCell.Row != requestedCell.Row || g.CurrentCell.Col != requestedCell.Col {
			g.LastInteractedRow = -1
			g.LastInteractedCol = -1
			g.InteractionPlayerRow = requestedCell.Row
			g.InteractionPlayerCol = requestedCell.Col
			// Increment movement count for hint system (only if player actually moved from a previous position)
			if g.CurrentCell != nil {
				g.MovementCount++
			}
		}

		g.CurrentCell = requestedCell
	} else {
		// Movement failed - trigger debounce animation
		if direction != "" {
			renderer.SetDebounceAnimation(direction)
		}
	}
}

// ensureNearbyCellsVisible ensures cells within 5x5 radius of player are always visible
func ensureNearbyCellsVisible(g *state.Game, centerCell *world.Cell) {
	if g.Grid == nil || centerCell == nil {
		return
	}

	centerRow := centerCell.Row
	centerCol := centerCell.Col

	// Ensure all cells within 5x5 radius (2 cells in each direction) are visible
	for row := centerRow - 2; row <= centerRow+2; row++ {
		for col := centerCol - 2; col <= centerCol+2; col++ {
			cell := g.Grid.GetCell(row, col)
			if cell != nil && cell.Room {
				// Always keep nearby cells visible for exploration
				cell.Discovered = true
				if cell.Visited {
					// Keep visited state
				}
			}
		}
	}
}

// formatHazardCallout formats a hazard description into a 2-line callout
// First line: hazard description in hazard color (red)
// Second line: hint (e.g., "Find the Circuit Breaker") in normal text color
func formatHazardCallout(hazard *entities.Hazard) string {
	description := hazard.Description
	info := entities.HazardTypes[hazard.Type]

	// Extract hint from description - look for "Find the" or "Find" pattern
	// Or use ControlName if available
	var hint string
	var mainDescription string

	if info.ControlName != "" {
		hint = fmt.Sprintf("Find the %s", info.ControlName)
		// Remove hint from description if it's there
		if strings.Contains(description, "Find the") {
			parts := strings.Split(description, "Find the")
			mainDescription = strings.TrimSpace(parts[0])
		} else {
			mainDescription = description
		}
	} else if info.ItemName != "" {
		hint = fmt.Sprintf("Find the %s", info.ItemName)
		// Remove hint from description if it's there
		if strings.Contains(description, "Find the") {
			parts := strings.Split(description, "Find the")
			mainDescription = strings.TrimSpace(parts[0])
		} else {
			mainDescription = description
		}
	} else {
		// Try to extract from description
		if strings.Contains(description, "Find the") {
			parts := strings.Split(description, "Find the")
			if len(parts) > 1 {
				hint = "Find the" + strings.TrimSpace(parts[1])
				// Remove hint from description
				mainDescription = strings.TrimSpace(parts[0])
			} else {
				mainDescription = description
			}
		} else if strings.Contains(description, "Find ") {
			parts := strings.Split(description, "Find ")
			if len(parts) > 1 {
				hint = "Find " + strings.TrimSpace(parts[1])
				// Remove hint from description
				mainDescription = strings.TrimSpace(parts[0])
			} else {
				mainDescription = description
			}
		} else {
			mainDescription = description
		}
	}

	// Format as 2-line callout with markup
	// First line uses HAZARD{} markup for red color, second line is normal text
	if hint != "" {
		return fmt.Sprintf("HAZARD{%s}\n%s", mainDescription, hint)
	}
	// Fallback: just show description in hazard color if we can't extract hint
	return fmt.Sprintf("HAZARD{%s}", mainDescription)
}

// logMessage adds a formatted message to the game's message log
func logMessage(g *state.Game, msg string, a ...any) {
	formatted := renderer.ApplyMarkup(msg, a...)
	g.AddMessage(formatted)
}
