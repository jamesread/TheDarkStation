// Package gameplay provides core game logic for player movement and interactions.
package gameplay

import (
	"fmt"
	"strings"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/features"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// CanEnter checks if the player can enter a cell
func CanEnter(g *state.Game, r *world.Cell, logReason bool) (bool, *world.ItemSet) {
	missingItems := mapset.New[*world.Item]()

	if r == nil || !r.Room {
		return false, &missingItems
	}

	// Door checks: power first, then keycard. Keycard overrides unpowered locked doors;
	// keycard-gated doors stay passable without power once unlocked.
	if gameworld.HasDoor(r) {
		rData := gameworld.GetGameData(r)
		roomName := rData.Door.RoomName
		unpowered := !setup.CellHasLivePower(g, r) && !manualEgressReleased(g, roomName)

		if unpowered {
			if gameworld.HasLockedDoor(r) {
				if !unlockDoorWithKeycard(g, r, rData, logReason) {
					return false, &missingItems
				}
			} else if !rData.Door.KeycardGated {
				if logReason {
					showUnpoweredDoorCallout(g, r, rData, roomName)
				}
				return false, &missingItems
			}
		} else if gameworld.HasLockedDoor(r) {
			if !unlockDoorWithKeycard(g, r, rData, logReason) {
				return false, &missingItems
			}
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

	// Check for repair devices (blocks movement)
	if gameworld.HasRepairDevice(r) {
		return false, &missingItems
	}

	// Check for hazard controls / circuit breaker switches (blocks movement, like furniture)
	if gameworld.HasHazardControl(r) {
		return false, &missingItems
	}

	if gameworld.HasBlockingRepairBlocker(r) {
		if logReason {
			repair := gameworld.GetGameData(r).RepairBlocker
			renderer.AddCallout(r.Row, r.Col, repairBlockerCallout(repair), renderer.CalloutColorHazard, 0)
		}
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
				if StartHazardClearFromItem(g, r, hazard, fixItem.Name) {
					return false, &missingItems
				}
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

	// Check for exit lift readiness (only for exit cell)
	if r.ExitCell && !setup.ExitLiftReady(g) {
		if logReason {
			switch setup.ExitLiftState(g) {
			case state.ExitLiftLockedUnpowered:
				exit := setup.ExitCell(g)
				roomName := ""
				if exit != nil {
					roomName = exit.Name
				}
				if roomName != "" && g.RoomDoorsPowered != nil && g.RoomDoorsPowered[roomName] {
					logMessage(g, "The lift has no routing power.")
				} else {
					logMessage(g, "The lift room has no door power.")
				}
			case state.ExitLiftLockedIncomplete:
				numHazards := 0
				g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
					if gameworld.HasBlockingHazard(cell) {
						numHazards++
					}
				})
				repairs := g.IncompleteRepairCount()
				if numHazards > 0 {
					logMessage(g, "The lift requires all environmental hazards to be cleared!")
					logMessage(g, "ACTION{%d} environmental hazard(s) remain.", numHazards)
				}
				if repairs > 0 {
					logMessage(g, "The lift is locked until deck repairs are complete.")
					logMessage(g, "ACTION{%d} repair objective(s) remain.", repairs)
				}
			}
		}
		return false, &missingItems
	}

	return true, &missingItems
}

func showUnpoweredDoorCallout(g *state.Game, r *world.Cell, rData *gameworld.GameCellData, roomName string) {
	msg := "UNPOWERED{Unpowered door}"
	if g.RoomDoorsPowered[roomName] {
		msg = "UNPOWERED{Door — power routing}"
	}
	renderer.AddCallout(r.Row, r.Col, fmt.Sprintf("%s\n%s\nSUBTLE{Hold USE — manual egress release}", msg, rData.Door.DoorName()), renderer.CalloutColorDoor, 0)
}

// unlockDoorWithKeycard unlocks all doors for the keycard on r when the player has it.
// Returns false when the door is locked and the player lacks the keycard.
func unlockDoorWithKeycard(g *state.Game, r *world.Cell, rData *gameworld.GameCellData, logReason bool) bool {
	keycardName := rData.Door.KeycardName()
	var keycardItem *world.Item

	g.OwnedItems.Each(func(item *world.Item) {
		if item.Name == keycardName {
			keycardItem = item
		}
	})

	if keycardItem == nil {
		if logReason {
			logMessage(g, "This door requires a %s", renderer.StyledKeycard(keycardName))
			renderer.AddCallout(r.Row, r.Col, fmt.Sprintf("TITLE{Door Locked}\nNeeds: KEYCARD{%s}", keycardName), renderer.CalloutColorDoor, 0)
		}
		return false
	}

	doorsUnlocked := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		cellData := gameworld.GetGameData(cell)
		if gameworld.HasLockedDoor(cell) && cellData.Door.KeycardName() == keycardName {
			cellData.Door.Unlock()
			doorsUnlocked++
		}
	})
	g.OwnedItems.Remove(keycardItem)

	var calloutMsg string
	if doorsUnlocked > 1 {
		calloutMsg = fmt.Sprintf("Used KEYCARD{%s} to unlock ACTION{%d} doors to ROOM{%s}!", keycardName, doorsUnlocked, rData.Door.RoomName)
	} else {
		calloutMsg = fmt.Sprintf("Used KEYCARD{%s} to unlock the %s!", keycardName, rData.Door.DoorName())
	}
	renderer.AddCallout(r.Row, r.Col, calloutMsg, renderer.CalloutColorKeycard, 0)
	return true
}

// MoveCell moves the player to a new cell
func MoveCell(g *state.Game, requestedCell *world.Cell) {
	// Determine direction for debounce animation and facing
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
	if direction != "" {
		setPlayerFacingFromDirection(g, direction)
	}

	if res, _ := CanEnter(g, requestedCell, true); res {
		features.MarkVisited(requestedCell)
		cellData := gameworld.GetGameData(requestedCell)
		cellData.LightsOn = true
		cellData.Lighted = true

		// Reveal cells within field of view (ray-cast; walls block sight).
		world.RevealFOVDefault(g.Grid, requestedCell, unpoweredDoorSightBlocker(g))

		// Update lighting-based exploration
		UpdateLightingExploration(g)

		// Reset interaction order when player moves
		if g.CurrentCell == nil || g.CurrentCell.Row != requestedCell.Row || g.CurrentCell.Col != requestedCell.Col {
			g.LastInteractedRow = -1
			g.LastInteractedCol = -1
			g.InteractionPlayerRow = requestedCell.Row
			g.InteractionPlayerCol = requestedCell.Col
			ClearGeneratorPowerGridOverlay(g)
			// Increment movement count for hint system (only if player actually moved from a previous position)
			if g.CurrentCell != nil {
				g.MovementCount++
			}
		}

		g.CurrentCell = requestedCell
		maybeAnnounceObservationCueOnMove(g, requestedCell)
		maybeAnnounceLinkageCueOnMove(g, requestedCell)
		if features.VisitedSystemEnabled() {
			noteLinkageTagFromVisitedCell(g, requestedCell)
		}
	} else {
		// Movement failed - trigger debounce animation
		if direction != "" {
			renderer.SetDebounceAnimation(direction)
		}
	}
}

func setPlayerFacingFromDirection(g *state.Game, direction string) {
	switch direction {
	case "north":
		g.PlayerFacing = state.FaceNorth
	case "south":
		g.PlayerFacing = state.FaceSouth
	case "east":
		g.PlayerFacing = state.FaceEast
	case "west":
		g.PlayerFacing = state.FaceWest
	}
}

// FaceTowardAdjacentCell updates player facing toward an orthogonally adjacent cell (e.g. on USE).
func FaceTowardAdjacentCell(g *state.Game, target *world.Cell) {
	if g == nil || g.CurrentCell == nil {
		return
	}
	if facing, ok := state.FacingToward(g.CurrentCell, target); ok {
		g.PlayerFacing = facing
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
