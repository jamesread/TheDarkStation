// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
package ebiten

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/leonelquinteros/gotext"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// calculateObjectives calculates the current level objectives based on game state
func (e *EbitenRenderer) calculateObjectives(g *state.Game) []string {
	if g == nil || g.Grid == nil {
		return nil
	}

	var objectives []string

	// Count unpowered generators (show remaining, not total)
	unpoweredGenerators := g.UnpoweredGeneratorCount()
	if unpoweredGenerators > 0 {
		if unpoweredGenerators == 1 {
			objectives = append(objectives, "POWER_UP_ONE_GENERATOR") // Will be translated in drawColoredTextSegments
		} else {
			formatStr := gotext.Get("POWER_UP_GENERATORS")
			objectives = append(objectives, fmt.Sprintf(formatStr, unpoweredGenerators))
		}
	}

	// Count hazards (matching showLevelObjectives logic - count remaining active hazards)
	numHazards := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if gameworld.HasBlockingHazard(cell) {
			numHazards++
		}
	})

	if numHazards > 0 {
		formatStr := gotext.Get("CLEAR_HAZARDS")
		objectives = append(objectives, fmt.Sprintf(formatStr, numHazards))
	}

	// If all objectives are complete, show exit message
	if unpoweredGenerators == 0 && numHazards == 0 {
		objectives = append(objectives, "FIND_LIFT") // Will be translated in drawColoredTextSegments
	}

	return objectives
}

// RenderFrame stores the game state and captures a snapshot for the next Draw call
func (e *EbitenRenderer) RenderFrame(g *state.Game) {
	e.gameMutex.Lock()
	e.game = g
	e.gameMutex.Unlock()

	// Capture a consistent snapshot of critical render state
	e.snapshotMutex.Lock()
	defer e.snapshotMutex.Unlock()

	if g == nil || g.CurrentCell == nil || g.Grid == nil {
		e.snapshot.valid = false
		return
	}

	// Update tracked player position (clearing is now done via ClearCalloutsIfMoved)
	e.lastPlayerRow = g.CurrentCell.Row
	e.lastPlayerCol = g.CurrentCell.Col
	e.lastPosInitialized = true

	e.snapshot.valid = true
	e.snapshot.level = g.Level
	e.snapshot.playerRow = g.CurrentCell.Row
	e.snapshot.playerCol = g.CurrentCell.Col
	e.snapshot.cellName = g.CurrentCell.Name
	e.snapshot.hasMap = g.HasMap
	e.snapshot.batteries = g.Batteries
	e.snapshot.gridRows = g.Grid.Rows()
	e.snapshot.gridCols = g.Grid.Cols()

	// Compute persistent room labels (for rooms the player has visited)
	e.snapshot.roomLabels = e.computeRoomLabels(g)

	// Track messages with timestamps and handle fade-out
	e.messagesMutex.Lock()
	now := time.Now().UnixMilli()
	const messageLifetime = 10000 // 10 seconds in milliseconds

	// Create a map of current game messages for quick lookup
	currentMessages := make(map[string]bool)
	for _, msg := range g.Messages {
		currentMessages[msg.Text] = true
	}

	// Update tracked messages: add new ones, keep existing ones (even if removed from game), remove expired ones
	updatedMessages := make([]messageEntry, 0)

	// Keep existing tracked messages that are not expired (even if removed from g.Messages)
	for _, tracked := range e.trackedMessages {
		age := now - tracked.Timestamp
		if age < messageLifetime {
			updatedMessages = append(updatedMessages, tracked)
		}
		// Messages older than 10 seconds are discarded (not added to updatedMessages)
	}

	// Add new messages from game that aren't already tracked
	for _, msg := range g.Messages {
		found := false
		for _, tracked := range e.trackedMessages {
			if tracked.Text == msg.Text {
				found = true
				break
			}
		}
		if !found {
			updatedMessages = append(updatedMessages, messageEntry{
				Text:      msg.Text,
				Timestamp: msg.Timestamp,
			})
		}
	}

	// Sort messages by timestamp (oldest first) to ensure chronological ordering
	// This ensures consistent ordering regardless of when messages were added
	sort.Slice(updatedMessages, func(i, j int) bool {
		return updatedMessages[i].Timestamp < updatedMessages[j].Timestamp
	})

	e.trackedMessages = updatedMessages

	// Copy to snapshot (only non-expired messages)
	e.snapshot.messages = make([]messageEntry, len(e.trackedMessages))
	copy(e.snapshot.messages, e.trackedMessages)
	e.messagesMutex.Unlock()

	// Copy owned items
	// Collect and sort items deterministically
	e.snapshot.ownedItems = make([]string, 0)
	g.OwnedItems.Each(func(item *world.Item) {
		e.snapshot.ownedItems = append(e.snapshot.ownedItems, item.Name)
	})
	// Sort items for deterministic display order
	sort.Strings(e.snapshot.ownedItems)

	// Copy generator states
	e.snapshot.generators = make([]generatorState, len(g.Generators))
	for i, gen := range g.Generators {
		e.snapshot.generators[i] = generatorState{
			powered:           gen.IsPowered(),
			batteriesInserted: gen.BatteriesInserted,
			batteriesRequired: gen.BatteriesRequired,
		}
	}

	// Calculate objectives
	e.snapshot.objectives = e.calculateObjectives(g)

	// Copy exit animation state
	e.snapshot.exitAnimating = g.ExitAnimating
	e.snapshot.exitAnimStartTime = g.ExitAnimStartTime

	// Find the cell with the most recent active callout (for focus background)
	e.snapshot.focusedCellRow = -1
	e.snapshot.focusedCellCol = -1
	e.calloutsMutex.RLock()
	nowUnixMilli := time.Now().UnixMilli()
	var mostRecentCallout *Callout
	for i := range e.callouts {
		callout := &e.callouts[i]
		// Check if callout is active (not expired)
		if callout.ExpiresAt == 0 || callout.ExpiresAt > nowUnixMilli {
			if mostRecentCallout == nil || callout.CreatedAt > mostRecentCallout.CreatedAt {
				mostRecentCallout = callout
			}
		}
	}
	if mostRecentCallout != nil {
		e.snapshot.focusedCellRow = mostRecentCallout.Row
		e.snapshot.focusedCellCol = mostRecentCallout.Col
	}
	e.calloutsMutex.RUnlock()

	// Find interactable cells adjacent to player (for focus background)
	e.snapshot.interactableCells = make([]struct {
		row int
		col int
	}, 0)
	if g.CurrentCell != nil {
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
			// Check if cell has interactable objects
			if gameworld.HasFurniture(cell) ||
				gameworld.HasMaintenanceTerminal(cell) ||
				gameworld.HasUnusedTerminal(cell) ||
				gameworld.HasUnsolvedPuzzle(cell) ||
				gameworld.HasInactiveHazardControl(cell) {
				e.snapshot.interactableCells = append(e.snapshot.interactableCells, struct {
					row int
					col int
				}{cell.Row, cell.Col})
			}
		}
	}

	// Copy active callouts (with expiration filtering)
	e.calloutsMutex.Lock()
	// Reuse nowUnixMilli from above (already calculated)
	activeCallouts := make([]Callout, 0)
	for _, c := range e.callouts {
		if c.ExpiresAt == 0 || c.ExpiresAt > nowUnixMilli {
			activeCallouts = append(activeCallouts, c)
		}
	}
	e.callouts = activeCallouts // Remove expired callouts
	e.snapshot.callouts = make([]Callout, len(activeCallouts))
	copy(e.snapshot.callouts, activeCallouts)
	e.calloutsMutex.Unlock()
}

// computeRoomLabels finds the leftmost valid position for each visited room's label,
// avoiding gaps (corridor cells) and ensuring the label starts at the leftmost point of the room.
func (e *EbitenRenderer) computeRoomLabels(g *state.Game) []roomLabel {
	if g == nil || g.Grid == nil {
		return nil
	}

	rows := g.Grid.Rows()
	cols := g.Grid.Cols()

	// Track which rooms have been visited (player has stepped inside)
	roomVisited := make(map[string]bool)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			cell := g.Grid.GetCell(row, col)
			if cell == nil || !cell.Room || cell.Name == "" {
				continue
			}
			// Never label corridors
			if strings.Contains(strings.ToLower(cell.Name), "corridor") {
				continue
			}
			if cell.Visited {
				roomVisited[cell.Name] = true
			}
		}
	}

	if len(roomVisited) == 0 {
		return nil
	}

	// For each room, find the leftmost valid position for the label
	// The label should be placed above a room cell, avoiding gaps (corridors) above
	type labelPos struct {
		row int
		col int
	}
	leftmostByRoom := make(map[string]labelPos)

	// First pass: find the leftmost column for each room
	leftmostColByRoom := make(map[string]int)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			cell := g.Grid.GetCell(row, col)
			if cell == nil || !cell.Room || cell.Name == "" {
				continue
			}

			roomName := cell.Name
			// Skip corridors and unvisited rooms
			if strings.Contains(strings.ToLower(roomName), "corridor") || !roomVisited[roomName] {
				continue
			}

			// Track the leftmost column for this room
			if leftmostCol, ok := leftmostColByRoom[roomName]; !ok || col < leftmostCol {
				leftmostColByRoom[roomName] = col
			}
		}
	}

	// Second pass: for each room, find the best row at the leftmost column
	// Prefer positions where the cell above (where label renders) is not a gap/corridor
	for roomName, leftmostCol := range leftmostColByRoom {
		bestRow := -1
		bestHasGap := true // Track if best position has a gap above

		// Scan rows at the leftmost column
		for row := 0; row < rows; row++ {
			cell := g.Grid.GetCell(row, leftmostCol)
			if cell == nil || !cell.Room || cell.Name != roomName {
				continue
			}

			// Check if the cell above (where label would render) is a gap/corridor
			labelRow := row - 1
			hasGap := false
			if labelRow < 0 {
				hasGap = true // Edge of map
			} else {
				aboveCell := g.Grid.GetCell(labelRow, leftmostCol)
				if aboveCell == nil || !aboveCell.Room || strings.Contains(strings.ToLower(aboveCell.Name), "corridor") {
					hasGap = true
				}
			}

			// Prefer positions without gaps, or if all have gaps, use the first (topmost) one
			if bestRow == -1 {
				bestRow = row
				bestHasGap = hasGap
			} else if !hasGap && bestHasGap {
				// This position has no gap, current best has gap - prefer this
				bestRow = row
				bestHasGap = false
			} else if hasGap == bestHasGap && row < bestRow {
				// Both have same gap status - prefer higher (topmost) row
				bestRow = row
			}
		}

		if bestRow >= 0 {
			leftmostByRoom[roomName] = labelPos{row: bestRow, col: leftmostCol}
		}
	}

	if len(leftmostByRoom) == 0 {
		return nil
	}

	labels := make([]roomLabel, 0, len(leftmostByRoom))
	for roomName, pos := range leftmostByRoom {
		// Use the leftmost column as both start and end (single cell position)
		// The drawing code will position the label starting from this point
		labels = append(labels, roomLabel{
			RoomName: roomName,
			Row:      pos.row,
			StartCol: pos.col,
			EndCol:   pos.col, // Single cell position
		})
	}
	return labels
}
