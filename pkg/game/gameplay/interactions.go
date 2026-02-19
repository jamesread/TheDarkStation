// Package gameplay provides core game logic for player movement and interactions.
package gameplay

import (
	"fmt"
	"strings"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// CheckAdjacentInteractables checks adjacent cells in NSEW priority order for interactables
// Cycles through interactables when player hasn't moved, skipping previously interacted cells
// Returns true if an interaction occurred
func CheckAdjacentInteractables(g *state.Game) bool {
	// Check if player has moved since last interaction (reset order if moved)
	if g.InteractionPlayerRow != g.CurrentCell.Row || g.InteractionPlayerCol != g.CurrentCell.Col {
		g.LastInteractedRow = -1
		g.LastInteractedCol = -1
		g.InteractionPlayerRow = g.CurrentCell.Row
		g.InteractionPlayerCol = g.CurrentCell.Col
	}

	// Check cells in NSEW priority order
	neighbors := []struct {
		cell      *world.Cell
		direction string
	}{
		{g.CurrentCell.North, "north"},
		{g.CurrentCell.South, "south"},
		{g.CurrentCell.East, "east"},
		{g.CurrentCell.West, "west"},
	}

	// Find first interactable cell, skipping the last interacted one
	for _, neighbor := range neighbors {
		cell := neighbor.cell
		if cell == nil {
			continue
		}

		// Skip if this is the cell we just interacted with
		if cell.Row == g.LastInteractedRow && cell.Col == g.LastInteractedCol {
			continue
		}

		// Check for interactables in priority order: generators, furniture, terminals, puzzles, hazard controls
		if gameworld.HasGenerator(cell) {
			if CheckAdjacentGeneratorAtCell(g, cell) {
				g.LastInteractedRow = cell.Row
				g.LastInteractedCol = cell.Col
				g.InteractionsCount++
				return true
			}
		}
		if gameworld.HasFurniture(cell) {
			if CheckAdjacentFurnitureAtCell(g, cell) {
				g.LastInteractedRow = cell.Row
				g.LastInteractedCol = cell.Col
				g.InteractionsCount++
				return true
			}
		}
		if gameworld.HasUnusedTerminal(cell) {
			if CheckAdjacentTerminalsAtCell(g, cell) {
				g.LastInteractedRow = cell.Row
				g.LastInteractedCol = cell.Col
				g.InteractionsCount++
				return true
			}
		}
		if gameworld.HasUnsolvedPuzzle(cell) {
			if CheckAdjacentPuzzlesAtCell(g, cell) {
				g.LastInteractedRow = cell.Row
				g.LastInteractedCol = cell.Col
				g.InteractionsCount++
				return true
			}
		}
		if gameworld.HasInactiveHazardControl(cell) {
			if CheckAdjacentHazardControlsAtCell(g, cell) {
				g.LastInteractedRow = cell.Row
				g.LastInteractedCol = cell.Col
				g.InteractionsCount++
				return true
			}
		}
		if gameworld.HasMaintenanceTerminal(cell) {
			if CheckAdjacentMaintenanceTerminalAtCell(g, cell) {
				// Reset last interacted cell so maintenance terminal can be reopened immediately
				g.LastInteractedRow = -1
				g.LastInteractedCol = -1
				g.InteractionsCount++
				return true
			}
		}
	}

	return false
}

// CheckAdjacentGeneratorAtCell checks a specific cell for generator and shows power info
// Returns true if generator was interacted with
func CheckAdjacentGeneratorAtCell(g *state.Game, cell *world.Cell) bool {
	if cell == nil || !gameworld.HasGenerator(cell) {
		return false
	}

	gen := gameworld.GetGameData(cell).Generator

	// Build tooltip message with generator status and power information
	var calloutText strings.Builder
	calloutText.WriteString(fmt.Sprintf("TITLE{%s}\n", gen.Name))

	if gen.IsPowered() {
		calloutText.WriteString("Status: POWERED{POWERED}\n")
		calloutText.WriteString(fmt.Sprintf("Batteries: ACTION{%d}/ACTION{%d}\n", gen.BatteriesInserted, gen.BatteriesRequired))
	} else {
		calloutText.WriteString("Status: UNPOWERED\n")
		calloutText.WriteString(fmt.Sprintf("Batteries: ACTION{%d}/ACTION{%d}\n", gen.BatteriesInserted, gen.BatteriesRequired))
		calloutText.WriteString(fmt.Sprintf("Needs: ACTION{%d} more batteries\n", gen.BatteriesNeeded()))
	}
	calloutText.WriteString("\n")
	calloutText.WriteString(fmt.Sprintf("Power Supply: %s\n", renderer.FormatPowerWatts(g.PowerSupply, false)))
	calloutText.WriteString(fmt.Sprintf("Power Consumption: %s\n", renderer.FormatPowerWatts(g.PowerConsumption, false)))
	calloutText.WriteString(fmt.Sprintf("Available Power: %s", renderer.FormatPowerWatts(g.GetAvailablePower(), false)))

	// Use appropriate color based on power status
	calloutColor := renderer.CalloutColorGenerator
	if gen.IsPowered() {
		calloutColor = renderer.CalloutColorGeneratorOn
	}

	renderer.AddCallout(cell.Row, cell.Col, calloutText.String(), calloutColor, 0)

	return true
}

func PickUpItemsOnFloor(g *state.Game) {
	g.CurrentCell.ItemsOnFloor.Each(func(item *world.Item) {
		g.CurrentCell.ItemsOnFloor.Remove(item)

		if item.Name == "Map" {
			g.HasMap = true
			g.OwnedItems.Put(item)
			renderer.AddCallout(g.CurrentCell.Row, g.CurrentCell.Col, "Picked up: ITEM{Map}", renderer.CalloutColorItem, 0)
		} else if item.Name == "Battery" {
			g.AddBatteries(1)
			renderer.AddCallout(g.CurrentCell.Row, g.CurrentCell.Col, "Picked up: ACTION{Battery}", renderer.CalloutColorItem, 0)
		} else {
			g.OwnedItems.Put(item)
			renderer.AddCallout(g.CurrentCell.Row, g.CurrentCell.Col, fmt.Sprintf("Picked up: ITEM{%s}", item.Name), renderer.CalloutColorItem, 0)
		}
	})

}

// CheckAdjacentGenerators checks adjacent cells for unpowered generators and inserts batteries
func CheckAdjacentGenerators(g *state.Game) {
	if g.Batteries == 0 {
		return
	}

	neighbors := []*world.Cell{
		g.CurrentCell.North,
		g.CurrentCell.East,
		g.CurrentCell.South,
		g.CurrentCell.West,
	}

	for _, cell := range neighbors {
		if cell == nil || !gameworld.HasUnpoweredGenerator(cell) {
			continue
		}

		gen := gameworld.GetGameData(cell).Generator
		needed := gen.BatteriesNeeded()
		if needed == 0 {
			continue
		}

		// Transfer batteries
		toInsert := needed
		if toInsert > g.Batteries {
			toInsert = g.Batteries
		}

		inserted := gen.InsertBatteries(g.UseBatteries(toInsert))
		if inserted > 0 {
			logMessage(g, "Inserted ACTION{%d} batteries into ROOM{%s}", inserted, gen.Name)

			if gen.IsPowered() {
				logMessage(g, "ITEM{%s} is now powered!", gen.Name)
				renderer.AddCallout(cell.Row, cell.Col, fmt.Sprintf("TITLE{%s} POWERED{ONLINE}", gen.Name), renderer.CalloutColorGeneratorOn, 0)
				// Update power supply when generator is powered
				g.UpdatePowerSupply()
				// Update lighting based on new power availability
				UpdateLightingExploration(g)
				logMessage(g, "Power supply: %dw available", g.GetAvailablePower())
			} else {
				logMessage(g, "%s needs ACTION{%d} more batteries", gen.Name, gen.BatteriesNeeded())
				renderer.AddCallout(cell.Row, cell.Col, fmt.Sprintf("+%d batteries (%d more needed)", inserted, gen.BatteriesNeeded()), renderer.CalloutColorGenerator, 0)
			}
		}
	}
}

// CheckAdjacentTerminalsAtCell checks a specific cell for terminals and interacts with it
// Returns true if a terminal was interacted with
func CheckAdjacentTerminalsAtCell(g *state.Game, cell *world.Cell) bool {
	if cell == nil || !gameworld.HasUnusedTerminal(cell) {
		return false
	}

	// CCTV terminal requires room power to operate
	if !g.RoomCCTVPowered[cell.Name] {
		logMessage(g, "CCTV terminal has no power. Restore power via the maintenance terminal.")
		renderer.AddCallout(cell.Row, cell.Col, "TITLE{Terminal has no power}", renderer.CalloutColorTerminal, 0)
		return true // consumed interaction, but no effect
	}

	terminal := gameworld.GetGameData(cell).Terminal
	targetRoom := terminal.TargetRoom

	// Check if the room is already fully revealed
	alreadyRevealed := isRoomFullyRevealed(g.Grid, targetRoom)

	if alreadyRevealed {
		logMessage(g, "Accessed %s - ROOM{%s} already explored.", terminal.Name, targetRoom)
		terminal.Activate()
		renderer.AddCallout(cell.Row, cell.Col, fmt.Sprintf("TITLE{%s already explored}", targetRoom), renderer.CalloutColorTerminal, 0)
	} else {
		// Reveal the target room
		if revealRoomByName(g.Grid, targetRoom) {
			terminal.Activate()
			logMessage(g, "Accessed %s - revealed ROOM{%s} on security feed!", terminal.Name, targetRoom)
			renderer.AddCallout(cell.Row, cell.Col, fmt.Sprintf("TITLE{Revealed: %s}", targetRoom), renderer.CalloutColorTerminal, 0)
		}
	}
	return true
}

// CheckAdjacentPuzzlesAtCell checks a specific cell for puzzles and interacts with it
// Returns true if a puzzle was interacted with
func CheckAdjacentPuzzlesAtCell(g *state.Game, cell *world.Cell) bool {
	if cell == nil || !gameworld.HasUnsolvedPuzzle(cell) {
		return false
	}

	puzzle := gameworld.GetGameData(cell).Puzzle

	// Show puzzle description and hint
	logMessage(g, "Puzzle Terminal: %s", puzzle.Description)
	if puzzle.Hint != "" {
		logMessage(g, "Hint: %s", puzzle.Hint)
	}

	// Check if player has found the solution code
	if g.HasFoundCode(puzzle.Solution) {
		// Player has the code, solve the puzzle
		if !puzzle.IsSolved() {
			puzzle.Solve()
			logMessage(g, "Puzzle solved! Solution: %s", puzzle.Solution)
			applyPuzzleReward(g, puzzle, cell)
			renderer.AddCallout(cell.Row, cell.Col, "TITLE{Puzzle solved!}", renderer.CalloutColorTerminal, 0)
		} else {
			logMessage(g, "This puzzle has already been solved.")
		}
	} else {
		// Show the puzzle challenge
		if puzzle.PuzzleType == entities.PuzzleSequence {
			logMessage(g, "Sequence Puzzle: Enter the correct sequence.")
		} else {
			logMessage(g, "Pattern Puzzle: Enter the correct pattern.")
		}
		logMessage(g, "Look for the solution code in logs and furniture descriptions.")
	}
	return true
}

// CheckAdjacentFurnitureAtCell checks a specific cell for furniture and interacts with it
// Returns true if furniture was interacted with
// Furniture can be interacted with multiple times, but items are only given once
func CheckAdjacentFurnitureAtCell(g *state.Game, cell *world.Cell) bool {
	if cell == nil || !gameworld.HasFurniture(cell) {
		return false
	}

	furniture := gameworld.GetGameData(cell).Furniture

	// Track if this is the first time checking this furniture
	wasChecked := furniture.IsChecked()

	// Check the furniture and get any contained item (if not already taken)
	// Check() sets ContainedItem to nil after first check, preventing duplicate items
	item := furniture.Check()

	// Check if description contains a puzzle code (format: "Code: X-Y-Z" or "Sequence: 1-2-3")
	// Only check for codes on first interaction
	if !wasChecked {
		CheckForPuzzleCode(g, furniture.Description)
	}

	// If furniture contained an item, give it to the player and show callout
	if item != nil {
		if item.Name == "Battery" {
			g.AddBatteries(1)
			calloutText := fmt.Sprintf("TITLE{%s}\nFound: ACTION{Battery}!", furniture.Name)
			renderer.AddCallout(cell.Row, cell.Col, calloutText, renderer.CalloutColorFurnitureChecked, 0)
		} else {
			g.OwnedItems.Put(item)
			calloutText := fmt.Sprintf("TITLE{%s}\nFound: ITEM{%s}!", furniture.Name, item.Name)
			renderer.AddCallout(cell.Row, cell.Col, calloutText, renderer.CalloutColorFurnitureChecked, 0)
		}
	} else {
		calloutText := fmt.Sprintf("TITLE{%s}\n%s", furniture.Name, furniture.Description)
		renderer.AddCallout(cell.Row, cell.Col, calloutText, renderer.CalloutColorFurnitureChecked, 0)
	}
	return true
}

// CheckAdjacentHazardControlsAtCell checks a specific cell for hazard controls and interacts with it
// Returns true if a hazard control was interacted with
func CheckAdjacentHazardControlsAtCell(g *state.Game, cell *world.Cell) bool {
	if cell == nil || !gameworld.HasInactiveHazardControl(cell) {
		return false
	}

	// Hazard control (circuit breaker) requires room power to operate
	if !g.RoomCCTVPowered[cell.Name] {
		logMessage(g, "Circuit breaker has no power. Restore power via the maintenance terminal.")
		renderer.AddCallout(cell.Row, cell.Col, "TITLE{No power}", renderer.CalloutColorHazardCtrl, 0)
		return true
	}

	control := gameworld.GetGameData(cell).HazardControl
	control.Activate()

	info := entities.HazardTypes[control.Type]
	logMessage(g, "Activated %s: %s", renderer.StyledHazardCtrl(control.Name), info.FixedMessage)
	renderer.AddCallout(cell.Row, cell.Col, fmt.Sprintf("TITLE{%s activated!}", control.Name), renderer.CalloutColorHazardCtrl, 0)
	return true
}

// CheckAdjacentMaintenanceTerminalAtCell checks a specific cell for maintenance terminal and opens menu
// Returns true if maintenance terminal was interacted with
func CheckAdjacentMaintenanceTerminalAtCell(g *state.Game, cell *world.Cell) bool {
	if cell == nil || !gameworld.HasMaintenanceTerminal(cell) {
		return false
	}

	maintenanceTerm := gameworld.GetGameData(cell).MaintenanceTerm
	if !maintenanceTerm.Powered {
		logMessage(g, "Terminal has no power. Restore power from another maintenance terminal.")
		return true
	}

	// Open maintenance terminal menu
	RunMaintenanceMenu(g, cell, maintenanceTerm)
	return true
}

// CheckForPuzzleCode extracts puzzle codes from text and adds them to found codes
func CheckForPuzzleCode(g *state.Game, text string) {
	// Look for patterns like "Code: 1-2-3-4" or "Sequence: up-down-left-right"
	// Simple pattern matching for codes
	lowerText := strings.ToLower(text)

	// Check for "code:" or "sequence:" followed by the actual code
	codePrefixes := []string{"code:", "sequence:", "pattern:", "solution:"}
	for _, prefix := range codePrefixes {
		if idx := strings.Index(lowerText, prefix); idx != -1 {
			// Extract the code after the prefix
			codeStart := idx + len(prefix)
			codeText := strings.TrimSpace(text[codeStart:])
			// Take up to the next sentence or line break
			if endIdx := strings.IndexAny(codeText, ".,;!?\n"); endIdx != -1 {
				codeText = codeText[:endIdx]
			}
			codeText = strings.TrimSpace(codeText)
			if codeText != "" {
				g.AddFoundCode(codeText)
				logMessage(g, "Discovered code: %s", codeText)
			}
			break
		}
	}
}

// applyPuzzleReward applies the reward for solving a puzzle
func applyPuzzleReward(g *state.Game, puzzle *entities.PuzzleTerminal, cell *world.Cell) {
	switch puzzle.Reward {
	case entities.RewardKeycard:
		// Find a locked door and unlock it
		// This would be set up during level generation
		logMessage(g, "A door unlocks somewhere on the station...")
	case entities.RewardBattery:
		g.AddBatteries(1)
		logMessage(g, "Received: ACTION{Battery}")
		renderer.AddCallout(cell.Row, cell.Col, "TITLE{Battery received!}", renderer.CalloutColorItem, 0)
	case entities.RewardRevealRoom:
		// Reveal a random room
		logMessage(g, "Security feed activated - a new area is revealed.")
	case entities.RewardUnlockArea:
		// Unlock a previously locked area
		logMessage(g, "Access granted to a previously locked section.")
	case entities.RewardMap:
		// Give the player the map - powerful reward!
		g.HasMap = true
		renderer.AddCallout(cell.Row, cell.Col, "TITLE{Map acquired!}", renderer.CalloutColorItem, 0)
		logMessage(g, "Received: ITEM{Map}")
	}
}

// revealRoomByName reveals all cells with the given room name
func revealRoomByName(grid *world.Grid, roomName string) bool {
	revealed := false
	alreadyRevealed := true

	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell.Name == roomName {
			if !cell.Visited {
				alreadyRevealed = false
			}
			cell.Discovered = true
			cell.Visited = true
			revealed = true
		}
	})

	// Return false if the room was already fully revealed
	if alreadyRevealed {
		return false
	}
	return revealed
}

// isRoomFullyRevealed checks if all cells with the given room name are visited
func isRoomFullyRevealed(grid *world.Grid, roomName string) bool {
	allVisited := true
	found := false

	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell.Name == roomName {
			found = true
			if !cell.Visited {
				allVisited = false
			}
		}
	})

	return found && allVisited
}
