// Package gameplay provides core game logic for player movement and interactions.
package gameplay

import (
	"fmt"
	"image/color"
	"log"
	"strings"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/features"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

var runMaintenanceMenu = RunMaintenanceMenu

// logInteractDebugSnapshot prints adjacent-cell flags and cycling state (prefix [Interact]).
func logInteractDebugSnapshot(g *state.Game, phase string) {
	if g == nil || g.CurrentCell == nil {
		log.Printf("[Interact] %s: player cell is nil", phase)
		return
	}
	pc := g.CurrentCell
	log.Printf("[Interact] %s: player=(%d,%d) lastInteracted=(%d,%d) interactionAnchor=(%d,%d) interactionsCount=%d",
		phase, pc.Row, pc.Col, g.LastInteractedRow, g.LastInteractedCol,
		g.InteractionPlayerRow, g.InteractionPlayerCol, g.InteractionsCount)

	dirs := []string{"N", "S", "E", "W"}
	cells := []*world.Cell{pc.North, pc.South, pc.East, pc.West}
	for i, cell := range cells {
		if cell == nil {
			log.Printf("[Interact] %s: neighbor %s: nil", phase, dirs[i])
			continue
		}
		var tags []string
		if cell.Discovered {
			tags = append(tags, "discovered")
		}
		if cell.Visited {
			tags = append(tags, "visited")
		}
		if gameworld.HasGenerator(cell) {
			tags = append(tags, "generator")
		}
		if gameworld.HasFurniture(cell) {
			tags = append(tags, "furniture")
		}
		if gameworld.HasUnusedTerminal(cell) {
			tags = append(tags, "cctv")
		}
		if gameworld.HasUnsolvedPuzzle(cell) {
			tags = append(tags, "puzzle")
		}
		if gameworld.HasInactiveHazardControl(cell) {
			tags = append(tags, "hazardCtrl")
		}
		if gameworld.HasMaintenanceTerminal(cell) {
			tags = append(tags, "maint")
		}
		log.Printf("[Interact] %s: neighbor %s: (%d,%d) name=%q flags=[%s]", phase, dirs[i], cell.Row, cell.Col, cell.Name, strings.Join(tags, ","))
	}
}

// countAdjacentInteractionCandidates returns how many orthogonal neighbors have at least one
// interaction type that CheckAdjacentInteractables considers (same Has* gates as the scan).
func countAdjacentInteractionCandidates(neighbors []*world.Cell) int {
	n := 0
	for _, cell := range neighbors {
		if cell == nil {
			continue
		}
		if gameworld.HasGenerator(cell) ||
			gameworld.HasFurniture(cell) ||
			gameworld.HasUnusedTerminal(cell) ||
			gameworld.HasUnsolvedPuzzle(cell) ||
			gameworld.HasInactiveHazardControl(cell) ||
			gameworld.HasPowerRelay(cell) ||
			gameworld.HasMaintenanceTerminal(cell) {
			n++
		}
	}
	return n
}

// CheckAdjacentInteractables checks adjacent cells for interactables.
// Generators are handled in a first pass (any direction) so a generator is not skipped when
// another direction has e.g. a maintenance terminal or CCTV that would come earlier in plain N,S,E,W order.
// Cycles through interactables when player hasn't moved, skipping the previously interacted cell.
// If there is only one adjacent interactable cell, last-interacted cycling is cleared so the first
// scan always hits that target (no "empty" pass that relies on a second scan).
// If the only adjacent interactable is the same cell as last time but multiple targets exist elsewhere,
// a second scan ignores that skip so e.g. the generator callout can open again without moving.
// Returns true if an interaction occurred
func CheckAdjacentInteractables(g *state.Game) bool {
	if g == nil || g.CurrentCell == nil {
		log.Printf("[Interact] CheckAdjacentInteractables: abort (nil game or current cell)")
		return false
	}

	// Check if player has moved since last interaction (reset order if moved)
	if g.InteractionPlayerRow != g.CurrentCell.Row || g.InteractionPlayerCol != g.CurrentCell.Col {
		g.LastInteractedRow = -1
		g.LastInteractedCol = -1
		g.InteractionPlayerRow = g.CurrentCell.Row
		g.InteractionPlayerCol = g.CurrentCell.Col
	}

	neighbors := []*world.Cell{
		g.CurrentCell.North,
		g.CurrentCell.South,
		g.CurrentCell.East,
		g.CurrentCell.West,
	}

	if countAdjacentInteractionCandidates(neighbors) <= 1 {
		g.LastInteractedRow = -1
		g.LastInteractedCol = -1
	}

	logInteractDebugSnapshot(g, "before_scan")

	for _, honorLastSkip := range []bool{true, false} {
		if tryAdjacentInteractableScan(g, neighbors, honorLastSkip) {
			return true
		}
	}

	log.Printf("[Interact] no handler matched (see before_scan neighbor lines)")
	return false
}

// tryAdjacentInteractableScan runs the two-pass adjacent scan. When honorLastInteractedSkip is true,
// the cell matching LastInteractedRow/Col is skipped so the player can cycle other adjacent targets.
func tryAdjacentInteractableScan(g *state.Game, neighbors []*world.Cell, honorLastInteractedSkip bool) bool {
	skipCell := func(cell *world.Cell) bool {
		if cell == nil {
			return true
		}
		if honorLastInteractedSkip && cell.Row == g.LastInteractedRow && cell.Col == g.LastInteractedCol {
			return true
		}
		return false
	}

	// Pass 1: generators only (highest priority across adjacency)
	for _, cell := range neighbors {
		if skipCell(cell) {
			continue
		}
		if gameworld.HasGenerator(cell) && CheckAdjacentGeneratorAtCell(g, cell) {
			FaceTowardAdjacentCell(g, cell)
			g.LastInteractedRow = cell.Row
			g.LastInteractedCol = cell.Col
			g.InteractionsCount++
			log.Printf("[Interact] handled: generator at (%d,%d)", cell.Row, cell.Col)
			return true
		}
	}

	// Pass 2: furniture, terminals, puzzles, hazard controls, maintenance
	for _, cell := range neighbors {
		if skipCell(cell) {
			continue
		}

		if gameworld.HasFurniture(cell) {
			if CheckAdjacentFurnitureAtCell(g, cell) {
				FaceTowardAdjacentCell(g, cell)
				g.LastInteractedRow = cell.Row
				g.LastInteractedCol = cell.Col
				g.InteractionsCount++
				log.Printf("[Interact] handled: furniture at (%d,%d)", cell.Row, cell.Col)
				return true
			}
		}
		if gameworld.HasUnusedTerminal(cell) {
			if CheckAdjacentTerminalsAtCell(g, cell) {
				FaceTowardAdjacentCell(g, cell)
				g.LastInteractedRow = cell.Row
				g.LastInteractedCol = cell.Col
				g.InteractionsCount++
				log.Printf("[Interact] handled: CCTV terminal at (%d,%d)", cell.Row, cell.Col)
				return true
			}
		}
		if gameworld.HasUnsolvedPuzzle(cell) {
			if CheckAdjacentPuzzlesAtCell(g, cell) {
				FaceTowardAdjacentCell(g, cell)
				g.LastInteractedRow = cell.Row
				g.LastInteractedCol = cell.Col
				g.InteractionsCount++
				log.Printf("[Interact] handled: puzzle at (%d,%d)", cell.Row, cell.Col)
				return true
			}
		}
		if gameworld.HasInactiveHazardControl(cell) {
			if CheckAdjacentHazardControlsAtCell(g, cell) {
				FaceTowardAdjacentCell(g, cell)
				g.LastInteractedRow = cell.Row
				g.LastInteractedCol = cell.Col
				g.InteractionsCount++
				log.Printf("[Interact] handled: hazard control at (%d,%d)", cell.Row, cell.Col)
				return true
			}
		}
		if gameworld.HasPowerRelay(cell) {
			if CheckAdjacentPowerRelayAtCell(g, cell) {
				FaceTowardAdjacentCell(g, cell)
				g.LastInteractedRow = cell.Row
				g.LastInteractedCol = cell.Col
				g.InteractionsCount++
				log.Printf("[Interact] handled: power relay at (%d,%d)", cell.Row, cell.Col)
				return true
			}
		}
		if gameworld.HasMaintenanceTerminal(cell) {
			if CheckAdjacentMaintenanceTerminalAtCell(g, cell) {
				FaceTowardAdjacentCell(g, cell)
				// Reset last interacted cell so maintenance terminal can be reopened immediately
				g.LastInteractedRow = -1
				g.LastInteractedCol = -1
				g.InteractionsCount++
				log.Printf("[Interact] handled: maintenance terminal at (%d,%d)", cell.Row, cell.Col)
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

	// Build tooltip message with generator status and power information (UNPOWERED{}/POWERED{} for headline + border tint).
	var calloutText strings.Builder
	if gen.IsPowered() {
		calloutText.WriteString(fmt.Sprintf("POWERED{%s}\n", gen.Name))
		calloutText.WriteString("SUBTLE{Status: }POWERED{Online}\n")
		calloutText.WriteString(fmt.Sprintf("Batteries: ACTION{%d}/ACTION{%d}\n", gen.BatteriesInserted, gen.BatteriesRequired))
	} else {
		calloutText.WriteString(fmt.Sprintf("UNPOWERED{%s}\n", gen.Name))
		if GeneratorNeedsLongUsePowerUp(gen) {
			calloutText.WriteString("SUBTLE{Status: }UNPOWERED{Waiting for startup sequence}\n")
			if gen.Tripped {
				calloutText.WriteString("SUBTLE{Tripped by overload — hold USE to restart}\n")
			} else {
				calloutText.WriteString("SUBTLE{Hold USE to start}\n")
			}
		} else {
			calloutText.WriteString("SUBTLE{Status: }UNPOWERED{Unpowered}\n")
		}
		calloutText.WriteString(fmt.Sprintf("Batteries: ACTION{%d}/ACTION{%d}\n", gen.BatteriesInserted, gen.BatteriesRequired))
		if gen.BatteriesNeeded() > 0 {
			calloutText.WriteString(fmt.Sprintf("Needs: ACTION{%d} more batteries\n", gen.BatteriesNeeded()))
		}
	}
	individual, gridTotal, gridCount := setup.GeneratorGridSupplyAtCell(g, cell)
	_, gridUsed, _ := setup.GridPowerSummary(g, cell)
	calloutText.WriteString("\n")
	calloutText.WriteString(fmt.Sprintf("Generator output: ACTION{%s}\n", renderer.FormatPowerWatts(individual, false)))
	calloutText.WriteString("\n")
	calloutText.WriteString(renderer.FormatPowerBarLine("Grid power", gridTotal, gridUsed))
	calloutText.WriteString("\n")
	if gridCount > 1 {
		calloutText.WriteString(fmt.Sprintf("SUBTLE{Separate power grids on deck: }ACTION{%d}", gridCount))
	}

	// Use appropriate color based on power status
	calloutColor := renderer.CalloutColorGenerator
	if gen.IsPowered() {
		calloutColor = renderer.CalloutColorGeneratorOn
	}

	renderer.AddCallout(cell.Row, cell.Col, calloutText.String(), calloutColor, 0)

	if gen.IsPowered() {
		ToggleGeneratorPowerGridOverlay(g, cell)
	}

	return true
}

func PickUpItemsOnFloor(g *state.Game) {
	g.CurrentCell.ItemsOnFloor.Each(func(item *world.Item) {
		g.CurrentCell.ItemsOnFloor.Remove(item)

		if item.Name == "Map" {
			g.HasMap = true
			g.OwnedItems.Put(item)
			renderer.AddCallout(g.CurrentCell.Row, g.CurrentCell.Col, "Picked up: ITEM{Map}", renderer.CalloutColorItem, 0)
		} else if strings.Contains(strings.ToLower(item.Name), "battery") {
			g.AddBatteries(1)
			msg := fmt.Sprintf("Picked up: BATTERY{%s}", item.Name)
			renderer.AddCallout(g.CurrentCell.Row, g.CurrentCell.Col, msg, renderer.CalloutColorBattery, 0)
		} else {
			g.OwnedItems.Put(item)
			msg, c := floorPickupOwnedItemCallout(item.Name)
			renderer.AddCallout(g.CurrentCell.Row, g.CurrentCell.Col, msg, c, 0)
		}
	})

}

// floorPickupOwnedItemCallout returns markup and AddCallout color for a carried item (not Map/Battery pickup paths).
func floorPickupOwnedItemCallout(itemName string) (string, color.RGBA) {
	l := strings.ToLower(itemName)
	switch {
	case strings.Contains(l, "keycard"):
		return fmt.Sprintf("Picked up: KEYCARD{%s}", itemName), renderer.CalloutColorKeycard
	default:
		return fmt.Sprintf("Picked up: ITEM{%s}", itemName), renderer.CalloutColorItem
	}
}

func furnitureFoundItemSegment(itemName string) string {
	l := strings.ToLower(itemName)
	switch {
	case strings.Contains(l, "battery"):
		return fmt.Sprintf("BATTERY{%s}", itemName)
	case strings.Contains(l, "keycard"):
		return fmt.Sprintf("KEYCARD{%s}", itemName)
	default:
		return fmt.Sprintf("ITEM{%s}", itemName)
	}
}

func furnitureCalloutHeading(name string) string {
	return fmt.Sprintf("FURNITURE_CHECKED{%s}", name)
}

func furnitureCalloutBody(text string) string {
	return text
}

func furnitureCalloutFoundWithItem(itemName string) string {
	return fmt.Sprintf("FURNITURE_CHECKED{Found: }%s!", furnitureFoundItemSegment(itemName))
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
				renderer.AddCallout(cell.Row, cell.Col, fmt.Sprintf("POWERED{%s - online}", gen.Name), renderer.CalloutColorGeneratorOn, 0)
				setup.NotifyPowerGridChanged(g)
				UpdateLightingExploration(g)
				logMessage(g, "Power supply: %dw available", g.GetAvailablePower())
			} else if GeneratorNeedsLongUsePowerUp(gen) {
				logMessage(g, "%s is waiting for startup — hold USE to power it up", gen.Name)
				renderer.AddCallout(cell.Row, cell.Col, fmt.Sprintf("UNPOWERED{%s — waiting for startup sequence}", gen.Name), renderer.CalloutColorGenerator, 0)
			} else {
				logMessage(g, "%s needs ACTION{%d} more batteries", gen.Name, gen.BatteriesNeeded())
				renderer.AddCallout(cell.Row, cell.Col, fmt.Sprintf("UNPOWERED{+%d batteries - %d more needed}", inserted, gen.BatteriesNeeded()), renderer.CalloutColorGenerator, 0)
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
		renderer.AddCallout(cell.Row, cell.Col, "UNPOWERED{Terminal has no power}", renderer.CalloutColorTerminal, 0)
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
// Observation-first beats (Story 5.2) add diegetic corridor stamps; solves still require FoundCodes here.
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
		// Story 5.3: correlation token must be inferred from another reachable surface before admit.
		if puzzle.LinkageToken != "" && !g.HasLinkageToken(puzzle.LinkageToken) {
			logMessage(g, "The terminal accepts your fragment, but a relay identifier is still unresolved.")
			return true
		}
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

	noteLinkageRelaysFromText(g, furniture.Description)

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
		if strings.Contains(strings.ToLower(item.Name), "battery") {
			g.AddBatteries(1)
			calloutText := fmt.Sprintf("%s\n%s", furnitureCalloutHeading(furniture.Name), furnitureCalloutFoundWithItem(item.Name))
			renderer.AddCallout(cell.Row, cell.Col, calloutText, renderer.CalloutColorFurnitureChecked, 0)
		} else {
			g.OwnedItems.Put(item)
			calloutText := fmt.Sprintf("%s\n%s", furnitureCalloutHeading(furniture.Name), furnitureCalloutFoundWithItem(item.Name))
			renderer.AddCallout(cell.Row, cell.Col, calloutText, renderer.CalloutColorFurnitureChecked, 0)
		}
	} else {
		calloutText := fmt.Sprintf("%s\n%s", furnitureCalloutHeading(furniture.Name), furnitureCalloutBody(furniture.Description))
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
		renderer.AddCallout(cell.Row, cell.Col, "UNPOWERED{No power}", renderer.CalloutColorHazardCtrl, 0)
		return true
	}

	control := gameworld.GetGameData(cell).HazardControl
	control.Activate()

	info := entities.HazardTypes[control.Type]
	logMessage(g, "Activated %s: %s", renderer.StyledHazardCtrl(control.Name), info.FixedMessage)
	renderer.AddCallout(cell.Row, cell.Col, fmt.Sprintf("TITLE{%s activated!}", control.Name), renderer.CalloutColorHazardCtrl, 0)
	return true
}

// CheckAdjacentPowerRelayAtCell toggles a corridor routing relay (Phase 3 power grid).
func CheckAdjacentPowerRelayAtCell(g *state.Game, cell *world.Cell) bool {
	if cell == nil || !gameworld.HasPowerRelay(cell) {
		return false
	}
	relay := gameworld.GetGameData(cell).PowerRelay
	relay.Closed = !relay.Closed
	stateLabel := "OPEN"
	if relay.Closed {
		stateLabel = "CLOSED"
	}
	msg := fmt.Sprintf("RELAY{%s}\nSUBTLE{Routing: }ACTION{%s}", "Power routing relay", stateLabel)
	renderer.AddCallout(cell.Row, cell.Col, msg, renderer.CalloutColorMaintenance, 0)
	setup.NotifyPowerGridChanged(g)
	return true
}

// CheckAdjacentMaintenanceTerminalAtCell checks a specific cell for maintenance terminal and opens menu
// Returns true if maintenance terminal was interacted with
func CheckAdjacentMaintenanceTerminalAtCell(g *state.Game, cell *world.Cell) bool {
	if cell == nil || !gameworld.HasMaintenanceTerminal(cell) {
		return false
	}

	g.MaintenanceMenuTerminalRow = cell.Row
	g.MaintenanceMenuTerminalCol = cell.Col
	setup.ApplyGridConductivePower(g)
	maintenanceTerm := gameworld.GetGameData(cell).MaintenanceTerm
	if !maintenanceTerm.Powered {
		g.MaintenanceMenuTerminalRow = -1
		g.MaintenanceMenuTerminalCol = -1
		logMessage(g, "Terminal has no power. Restore power from another maintenance terminal.")
		renderer.AddCallout(cell.Row, cell.Col, "UNPOWERED{Terminal has no power}", renderer.CalloutColorMaintenance, 0)
		return true
	}

	// Open maintenance terminal menu
	runMaintenanceMenu(g, cell, maintenanceTerm)
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
		renderer.AddCallout(cell.Row, cell.Col, "TITLE{Battery received!}", renderer.CalloutColorBattery, 0)
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
			if features.VisitedSystemEnabled() {
				if !cell.Visited {
					alreadyRevealed = false
				}
			} else if !cell.Discovered {
				alreadyRevealed = false
			}
			cell.Discovered = true
			features.MarkVisited(cell)
			revealed = true
		}
	})

	// Return false if the room was already fully revealed
	if alreadyRevealed {
		return false
	}
	return revealed
}

// isRoomFullyRevealed checks if all cells with the given room name are visited (or discovered when visited system is off).
func isRoomFullyRevealed(grid *world.Grid, roomName string) bool {
	allRevealed := true
	found := false

	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell.Name == roomName {
			found = true
			if features.VisitedSystemEnabled() {
				if !cell.Visited {
					allRevealed = false
				}
			} else if !cell.Discovered {
				allRevealed = false
			}
		}
	})

	return found && allRevealed
}
