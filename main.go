package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	gettext "github.com/gosexy/gettext"
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/input"
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
)

func initGettext() {
	gettext.BindTextdomain("default", "mo/")
	gettext.Textdomain("default")
	gettext.SetLocale(gettext.LcAll, "en_GB.utf8")
}

// logMessage adds a formatted message to the game's message log
func logMessage(g *state.Game, msg string, a ...any) {
	formatted := renderer.FormatString(msg, a...)
	g.AddMessage(formatted)
}

// generateGrid creates a new grid using the default generator
func generateGrid(level int) *world.Grid {
	return generator.DefaultGenerator.Generate(level)
}

// addCandidateIfNotVisited adds a candidate cell for DFS if valid
// collectReachableRooms collects all reachable rooms from a starting cell using BFS
func collectReachableRooms(start *world.Cell, avoid *mapset.Set[*world.Cell]) []*world.Cell {
	var rooms []*world.Cell
	visited := mapset.New[*world.Cell]()
	queue := []*world.Cell{start}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == nil || !current.Room || visited.Has(current) {
			continue
		}

		visited.Put(current)

		if !avoid.Has(current) {
			rooms = append(rooms, current)
		}

		// Add neighbors to queue
		neighbors := []*world.Cell{current.North, current.East, current.South, current.West}
		for _, n := range neighbors {
			if n != nil && n.Room && !visited.Has(n) {
				queue = append(queue, n)
			}
		}
	}

	return rooms
}

// manhattanDistance calculates the Manhattan distance between two cells
func manhattanDistance(a, b *world.Cell) int {
	rowDist := a.Row - b.Row
	colDist := a.Col - b.Col
	if rowDist < 0 {
		rowDist = -rowDist
	}
	if colDist < 0 {
		colDist = -colDist
	}
	return rowDist + colDist
}

// findRoom finds a random reachable room at an appropriate distance based on level
func findRoom(g *state.Game, start *world.Cell, avoid *mapset.Set[*world.Cell]) *world.Cell {
	rooms := collectReachableRooms(start, avoid)

	if len(rooms) == 0 {
		return start
	}

	// Calculate minimum distance based on level
	// Level 1: min 2, Level 5: min 6, Level 10: min 11
	minDistance := 1 + g.Level

	// Filter rooms by minimum distance
	var farRooms []*world.Cell
	for _, room := range rooms {
		if manhattanDistance(start, room) >= minDistance {
			farRooms = append(farRooms, room)
		}
	}

	// If no rooms are far enough, use all rooms and pick the furthest ones
	if len(farRooms) == 0 {
		farRooms = rooms
		if len(farRooms) > 2 {
			// Simple selection: keep only rooms in the further half
			var maxDist int
			for _, room := range rooms {
				d := manhattanDistance(start, room)
				if d > maxDist {
					maxDist = d
				}
			}
			threshold := maxDist / 2
			farRooms = nil
			for _, room := range rooms {
				if manhattanDistance(start, room) >= threshold {
					farRooms = append(farRooms, room)
				}
			}
			if len(farRooms) == 0 {
				farRooms = rooms
			}
		}
	}

	// Pick a random room from the candidates
	return farRooms[rand.Intn(len(farRooms))]
}

// placeItem places an item in a random reachable room at an appropriate distance based on level
func placeItem(g *state.Game, start *world.Cell, item *world.Item, avoid *mapset.Set[*world.Cell]) *world.Cell {
	room := findRoom(g, start, avoid)

	if item != nil {
		room.ItemsOnFloor.Put(item)
		g.AddHint("The " + renderer.ColorDenied.Sprintf(item.Name) + " is in " + renderer.ColorCell.Sprintf(room.Name))
	}

	return room
}

// buildGame creates a new game instance with optional starting level
func buildGame(startLevel int) *state.Game {
	g := state.NewGame()

	// Set starting level if specified (for developer testing)
	if startLevel > 1 {
		g.Level = startLevel
	}

	g.Grid = generateGrid(g.Level)
	setupLevel(g)

	// Clear the initial "entered room" message
	g.ClearMessages()
	logMessage(g, "Welcome to the Abandoned Station!")
	logMessage(g, "You are on deck %d.", g.Level)
	showLevelObjectives(g)

	return g
}

// findCorridorCells returns all corridor cells in the grid
func findCorridorCells(grid *world.Grid) []*world.Cell {
	var corridors []*world.Cell
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell.Room && cell.Name == "Corridor" {
			corridors = append(corridors, cell)
		}
	})
	return corridors
}

// roomEntryPoints represents all corridor cells that provide access to a specific room
type roomEntryPoints struct {
	roomName   string
	entryCells []*world.Cell
}

// findRoomEntryPoints finds all corridor cells that serve as entry points to each room
// Groups them by room so we can door ALL entries to fully block a room
func findRoomEntryPoints(grid *world.Grid) map[string]*roomEntryPoints {
	entries := make(map[string]*roomEntryPoints)
	seenCells := mapset.New[string]() // Track cells we've already assigned

	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		// Only look at corridor cells
		if !cell.Room || cell.Name != "Corridor" {
			return
		}

		// Check adjacent cells for rooms (not corridors)
		neighbors := []*world.Cell{cell.North, cell.East, cell.South, cell.West}
		for _, neighbor := range neighbors {
			if neighbor != nil && neighbor.Room && neighbor.Name != "Corridor" && neighbor.Name != "" {
				roomName := neighbor.Name
				cellKey := fmt.Sprintf("%d,%d-%s", cell.Row, cell.Col, roomName)

				if seenCells.Has(cellKey) {
					continue
				}
				seenCells.Put(cellKey)

				// Initialize entry points for this room if needed
				if entries[roomName] == nil {
					entries[roomName] = &roomEntryPoints{
						roomName:   roomName,
						entryCells: make([]*world.Cell, 0),
					}
				}

				// Add this corridor cell as an entry point
				// Only add if not already in the list (a corridor might touch multiple cells of same room)
				alreadyAdded := false
				for _, existing := range entries[roomName].entryCells {
					if existing == cell {
						alreadyAdded = true
						break
					}
				}
				if !alreadyAdded {
					entries[roomName].entryCells = append(entries[roomName].entryCells, cell)
				}
			}
		}
	})

	return entries
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// isChokepoint checks if a cell is a chokepoint (removing it would disconnect parts of the map)
func isChokepoint(grid *world.Grid, cell *world.Cell, start *world.Cell) bool {
	if cell == nil || !cell.Room {
		return false
	}

	// Count reachable rooms from start without going through this cell
	reachableWithout := countReachableRooms(grid, start, cell)

	// Count total rooms
	totalRooms := 0
	grid.ForEachCell(func(row, col int, c *world.Cell) {
		if c.Room {
			totalRooms++
		}
	})

	// If removing this cell reduces reachability, it's a chokepoint
	return reachableWithout < totalRooms-1 // -1 because we exclude the cell itself
}

// countReachableRooms counts rooms reachable from start, optionally excluding a cell
func countReachableRooms(grid *world.Grid, start *world.Cell, exclude *world.Cell) int {
	visited := mapset.New[*world.Cell]()
	queue := []*world.Cell{start}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == nil || !current.Room || visited.Has(current) || current == exclude {
			continue
		}

		visited.Put(current)

		neighbors := []*world.Cell{current.North, current.East, current.South, current.West}
		for _, n := range neighbors {
			if n != nil && n.Room && !visited.Has(n) && n != exclude {
				queue = append(queue, n)
			}
		}
	}

	return visited.Size()
}

// getReachableCells returns all cells reachable from start without passing through locked doors
func getReachableCells(grid *world.Grid, start *world.Cell, lockedDoors *mapset.Set[*world.Cell]) *mapset.Set[*world.Cell] {
	reachable := mapset.New[*world.Cell]()
	queue := []*world.Cell{start}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == nil || !current.Room || reachable.Has(current) {
			continue
		}

		// Can't pass through locked doors
		if lockedDoors.Has(current) {
			continue
		}

		reachable.Put(current)

		neighbors := []*world.Cell{current.North, current.East, current.South, current.West}
		for _, n := range neighbors {
			if n != nil && n.Room && !reachable.Has(n) {
				queue = append(queue, n)
			}
		}
	}

	return &reachable
}

// findRoomInReachable finds a random room cell within the reachable set
func findRoomInReachable(reachable *mapset.Set[*world.Cell], avoid *mapset.Set[*world.Cell]) *world.Cell {
	var candidates []*world.Cell
	reachable.Each(func(cell *world.Cell) {
		if cell.Name != "Corridor" && !avoid.Has(cell) {
			candidates = append(candidates, cell)
		}
	})

	if len(candidates) == 0 {
		// Fallback to any reachable cell
		reachable.Each(func(cell *world.Cell) {
			if !avoid.Has(cell) {
				candidates = append(candidates, cell)
			}
		})
	}

	if len(candidates) == 0 {
		return nil
	}

	return candidates[rand.Intn(len(candidates))]
}

// setupLevel configures the current level with items and keys
func setupLevel(g *state.Game) {
	// Cells to avoid placing items on
	avoid := mapset.New[*world.Cell]()
	avoid.Put(g.Grid.StartCell())
	avoid.Put(g.Grid.ExitCell())

	// Track which cells have locked doors for reachability calculations
	lockedDoorCells := mapset.New[*world.Cell]()

	// Determine number of locked rooms based on level
	// Level 1: 1 room, Level 2: 2 rooms, Level 3+: scales up
	numLockedRooms := 1
	if g.Level >= 2 {
		numLockedRooms = 2
	}
	if g.Level >= 4 {
		numLockedRooms = 3
	}
	if g.Level >= 6 {
		numLockedRooms = 4
	}

	// Find all room entry points (corridor cells that provide access to each room)
	roomEntries := findRoomEntryPoints(g.Grid)

	// Build list of candidate rooms (rooms with 1-3 entry points that we can fully block)
	type roomCandidate struct {
		name    string
		entries *roomEntryPoints
	}
	var candidates []roomCandidate
	for roomName, entries := range roomEntries {
		// Only consider rooms with 1-3 entry points (manageable to door)
		if len(entries.entryCells) >= 1 && len(entries.entryCells) <= 3 {
			candidates = append(candidates, roomCandidate{name: roomName, entries: entries})
		}
	}

	// Shuffle candidates for variety
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	// Track which rooms have doors
	roomsWithDoors := mapset.New[string]()
	lockedRoomsPlaced := 0

	// Place doors to fully block selected rooms
	for _, candidate := range candidates {
		if lockedRoomsPlaced >= numLockedRooms {
			break
		}

		roomName := candidate.name
		entryCells := candidate.entries.entryCells

		// Skip if already has doors
		if roomsWithDoors.Has(roomName) {
			continue
		}

		// Check if all entry cells are available and reachable
		currentlyReachable := getReachableCells(g.Grid, g.Grid.StartCell(), &lockedDoorCells)
		allValid := true
		for _, cell := range entryCells {
			if avoid.Has(cell) || lockedDoorCells.Has(cell) || !currentlyReachable.Has(cell) {
				allValid = false
				break
			}
		}
		if !allValid {
			continue
		}

		// Test if blocking ALL entry cells reduces reachability
		testLocked := mapset.New[*world.Cell]()
		lockedDoorCells.Each(func(c *world.Cell) { testLocked.Put(c) })
		for _, cell := range entryCells {
			testLocked.Put(cell)
		}
		reachableWithDoors := getReachableCells(g.Grid, g.Grid.StartCell(), &testLocked)

		// Must actually block something
		if reachableWithDoors.Size() >= currentlyReachable.Size() {
			continue
		}

		// Place the keycard in the area reachable BEFORE these doors
		keycardRoom := findRoomInReachable(reachableWithDoors, &avoid)
		if keycardRoom == nil {
			continue
		}

		// Create the keycard (one keycard opens all doors to this room)
		door := world.NewDoor(roomName)
		keycardName := door.KeycardName()

		keycard := world.NewItem(keycardName)
		keycardRoom.ItemsOnFloor.Put(keycard)
		avoid.Put(keycardRoom)
		g.AddHint("The " + renderer.ColorKeycard.Sprintf(keycardName) + " is in " + renderer.ColorCell.Sprintf(keycardRoom.Name))

		// Place doors on ALL entry cells (they share the same keycard)
		for _, cell := range entryCells {
			cellDoor := world.NewDoor(roomName)
			cell.Door = cellDoor
			avoid.Put(cell)
			lockedDoorCells.Put(cell)
		}

		roomsWithDoors.Put(roomName)
		lockedRoomsPlaced++

		if len(entryCells) == 1 {
			g.AddHint("The " + renderer.ColorDoor.Sprintf(door.DoorName()) + " blocks access to " + renderer.ColorCell.Sprintf(roomName))
		} else {
			g.AddHint(fmt.Sprintf("%d doors block access to %s", len(entryCells), renderer.ColorCell.Sprintf(roomName)))
		}
	}

	// Levels 3+: Exit requires generators
	if g.Level >= 3 {
		g.Grid.ExitCell().Locked = true

		// Place generators: level 3 = 1 generator, level 4 = 2, etc.
		numGenerators := g.Level - 2
		totalBatteriesNeeded := 0

		for i := 0; i < numGenerators; i++ {
			// Each generator needs 1-5 batteries, scaling with level
			minBatteries := 1 + (g.Level-3)/3
			maxBatteries := 2 + (g.Level-3)/2
			if minBatteries > 5 {
				minBatteries = 5
			}
			if maxBatteries > 5 {
				maxBatteries = 5
			}
			if maxBatteries < minBatteries {
				maxBatteries = minBatteries
			}

			batteriesRequired := minBatteries + rand.Intn(maxBatteries-minBatteries+1)
			totalBatteriesNeeded += batteriesRequired

			gen := world.NewGenerator(fmt.Sprintf("Generator #%d", i+1), batteriesRequired)
			genRoom := findRoom(g, g.Grid.StartCell(), &avoid)
			if genRoom != nil {
				genRoom.Generator = gen
				gen.Cell = genRoom
				g.AddGenerator(gen)
				avoid.Put(genRoom)

				g.AddHint("A generator is in " + renderer.ColorCell.Sprintf(genRoom.Name))
			}
		}

		// Place batteries: total needed + 1-2 extra per level for some buffer
		extraBatteries := 1 + rand.Intn(2)
		totalBatteries := totalBatteriesNeeded + extraBatteries

		for i := 0; i < totalBatteries; i++ {
			battery := world.NewItem("Battery")
			placeItem(g, g.Grid.StartCell(), battery, &avoid)
		}
	} else {
		g.Grid.ExitCell().Locked = false
	}

	// Place environmental hazards (level 2+)
	if g.Level >= 2 {
		placeHazards(g, &avoid, &lockedDoorCells)
	}

	// Always place a map in a reachable area
	reachable := getReachableCells(g.Grid, g.Grid.StartCell(), &lockedDoorCells)
	mapRoom := findRoomInReachable(reachable, &avoid)
	if mapRoom != nil {
		mapRoom.ItemsOnFloor.Put(world.NewItem("Map"))
		avoid.Put(mapRoom)
	}

	// Place CCTV terminals (1-3 based on level)
	numTerminals := 1 + g.Level/3
	if numTerminals > 3 {
		numTerminals = 3
	}

	// Collect all unique room names (excluding corridors)
	roomNames := collectUniqueRoomNames(g.Grid)

	for i := 0; i < numTerminals; i++ {
		terminalRoom := findRoom(g, g.Grid.StartCell(), &avoid)
		if terminalRoom != nil && len(roomNames) > 0 {
			terminal := world.NewCCTVTerminal(fmt.Sprintf("CCTV Terminal #%d", i+1))

			// Assign a random room for this terminal to reveal
			targetIdx := rand.Intn(len(roomNames))
			terminal.TargetRoom = roomNames[targetIdx]
			// Remove this room from the list so each terminal reveals a different room
			roomNames = append(roomNames[:targetIdx], roomNames[targetIdx+1:]...)

			terminal.Cell = terminalRoom
			terminalRoom.Terminal = terminal
			avoid.Put(terminalRoom)
		}
	}

	// Place furniture in rooms (1-2 pieces per unique room type)
	placeFurniture(g, &avoid)

	g.CurrentCell = g.Grid.GetCenterCell()

	moveCell(g, g.Grid.StartCell())
}

// collectUniqueRoomNames returns a list of unique room names (excluding corridors)
func collectUniqueRoomNames(grid *world.Grid) []string {
	namesSet := mapset.New[string]()
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell.Room && cell.Name != "Corridor" && cell.Name != "" {
			namesSet.Put(cell.Name)
		}
	})

	var names []string
	namesSet.Each(func(name string) {
		names = append(names, name)
	})
	return names
}

// placeHazards places environmental hazards that block progress
func placeHazards(g *state.Game, avoid *mapset.Set[*world.Cell], lockedDoorCells *mapset.Set[*world.Cell]) {
	// Number of hazards scales with level: level 2 = 1, level 3 = 1-2, level 4+ = 2-3
	numHazards := 1
	if g.Level >= 4 {
		numHazards = 2 + rand.Intn(2)
	} else if g.Level >= 3 {
		numHazards = 1 + rand.Intn(2)
	}

	// Available hazard types (excluding Vacuum initially, add it at level 3+)
	hazardTypes := []world.HazardType{
		world.HazardCoolant,
		world.HazardElectrical,
		world.HazardGas,
	}
	if g.Level >= 3 {
		hazardTypes = append(hazardTypes, world.HazardVacuum)
	}
	if g.Level >= 5 {
		hazardTypes = append(hazardTypes, world.HazardRadiation)
	}

	// Find corridor cells that could block progress
	corridors := findCorridorCells(g.Grid)
	rand.Shuffle(len(corridors), func(i, j int) {
		corridors[i], corridors[j] = corridors[j], corridors[i]
	})

	hazardsPlaced := 0
	for _, corridorCell := range corridors {
		if hazardsPlaced >= numHazards {
			break
		}

		// Skip if already used
		if avoid.Has(corridorCell) || lockedDoorCells.Has(corridorCell) {
			continue
		}

		// Check if this corridor is reachable from start
		currentlyReachable := getReachableCells(g.Grid, g.Grid.StartCell(), lockedDoorCells)
		if !currentlyReachable.Has(corridorCell) {
			continue
		}

		// Test if blocking this cell would reduce reachability
		testBlocked := mapset.New[*world.Cell]()
		lockedDoorCells.Each(func(c *world.Cell) { testBlocked.Put(c) })
		testBlocked.Put(corridorCell)
		reachableWithHazard := getReachableCells(g.Grid, g.Grid.StartCell(), &testBlocked)

		// Only place hazard if it blocks something
		if reachableWithHazard.Size() >= currentlyReachable.Size() {
			continue
		}

		// Choose a random hazard type
		hazardType := hazardTypes[rand.Intn(len(hazardTypes))]
		hazard := world.NewHazard(hazardType)

		// Place the hazard
		corridorCell.Hazard = hazard
		avoid.Put(corridorCell)

		info := world.HazardTypes[hazardType]

		if hazard.RequiresItem() {
			// Place the required item (e.g., Patch Kit) in a reachable area
			itemRoom := findRoomInReachable(reachableWithHazard, avoid)
			if itemRoom == nil {
				itemRoom = findRoomInReachable(currentlyReachable, avoid)
			}
			if itemRoom != nil {
				item := world.NewItem(info.ItemName)
				itemRoom.ItemsOnFloor.Put(item)
				avoid.Put(itemRoom)
				g.AddHint("A " + renderer.ColorItem.Sprintf(info.ItemName) + " is in " + renderer.ColorCell.Sprintf(itemRoom.Name))
			}
		} else {
			// Place the control panel in a reachable area
			controlRoom := findRoomInReachable(reachableWithHazard, avoid)
			if controlRoom == nil {
				controlRoom = findRoomInReachable(currentlyReachable, avoid)
			}
			if controlRoom != nil {
				control := world.NewHazardControl(hazardType, hazard)
				control.Cell = controlRoom
				controlRoom.HazardControl = control
				avoid.Put(controlRoom)
				g.AddHint("The " + renderer.ColorHazardCtrl.Sprintf(info.ControlName) + " is in " + renderer.ColorCell.Sprintf(controlRoom.Name))
			}
		}

		hazardsPlaced++
	}
}

// placeFurniture places thematically appropriate furniture in rooms
func placeFurniture(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	// Collect all unique rooms and their cells
	roomCells := make(map[string][]*world.Cell)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell.Room && cell.Name != "Corridor" && cell.Name != "" {
			roomCells[cell.Name] = append(roomCells[cell.Name], cell)
		}
	})

	// For each unique room, try to place 1-2 furniture pieces
	for roomName, cells := range roomCells {
		templates := world.GetAllFurnitureForRoom(roomName)
		if len(templates) == 0 {
			continue
		}

		// Shuffle templates for variety
		rand.Shuffle(len(templates), func(i, j int) {
			templates[i], templates[j] = templates[j], templates[i]
		})

		// Place 1-2 furniture pieces per room (based on room size)
		numFurniture := 1
		if len(cells) > 6 {
			numFurniture = 2
		}
		if numFurniture > len(templates) {
			numFurniture = len(templates)
		}

		// Find valid cells (not already used for something else)
		var validCells []*world.Cell
		for _, cell := range cells {
			if !avoid.Has(cell) && !cell.ExitCell && cell.Generator == nil &&
				cell.Door == nil && cell.Terminal == nil && cell.Furniture == nil {
				validCells = append(validCells, cell)
			}
		}

		// Shuffle valid cells
		rand.Shuffle(len(validCells), func(i, j int) {
			validCells[i], validCells[j] = validCells[j], validCells[i]
		})

		// Track furniture placed in this room for item hiding
		var placedFurniture []*world.Furniture

		// Place furniture
		for i := 0; i < numFurniture && i < len(validCells); i++ {
			template := templates[i]
			furniture := world.NewFurniture(template.Name, template.Description, template.Icon)
			furniture.Cell = validCells[i]
			validCells[i].Furniture = furniture
			placedFurniture = append(placedFurniture, furniture)
		}

		// Chance to hide items from the floor in furniture (40% per item)
		if len(placedFurniture) > 0 {
			hideItemsInFurniture(g, cells, placedFurniture)
		}
	}
}

// hideItemsInFurniture moves items from floor cells into furniture with a chance
func hideItemsInFurniture(g *state.Game, roomCells []*world.Cell, furniture []*world.Furniture) {
	// Find items on the floor in this room (keycards, patch kits - not batteries or maps)
	for _, cell := range roomCells {
		if cell.ItemsOnFloor.Size() == 0 {
			continue
		}

		var itemsToMove []*world.Item
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			// Only hide keycards and patch kits - items that are part of puzzles
			if containsSubstring(item.Name, "Keycard") || item.Name == "Patch Kit" {
				// 50% chance to hide in furniture
				if rand.Intn(100) < 50 {
					itemsToMove = append(itemsToMove, item)
				}
			}
		})

		// Move items to furniture
		for _, item := range itemsToMove {
			// Find furniture without an item
			for _, f := range furniture {
				if f.ContainedItem == nil {
					cell.ItemsOnFloor.Remove(item)
					f.ContainedItem = item

					// Update hint to mention furniture instead of room
					updateHintForFurnitureItem(g, item, f)
					break
				}
			}
		}
	}
}

// updateHintForFurnitureItem updates the hint for an item to mention it's in furniture
func updateHintForFurnitureItem(g *state.Game, item *world.Item, furniture *world.Furniture) {
	roomName := furniture.Cell.Name

	// Find and update the existing hint for this item
	for i, hint := range g.Hints {
		if containsSubstring(hint, item.Name) && containsSubstring(hint, roomName) {
			// Replace the hint with one mentioning the furniture
			g.Hints[i] = "The " + renderer.ColorKeycard.Sprintf(item.Name) + " is hidden in the " +
				renderer.ColorFurniture.Sprintf(furniture.Name) + " in " + renderer.ColorCell.Sprintf(roomName)
			return
		}
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

// advanceLevel generates a new map and advances to the next level
func advanceLevel(g *state.Game) {
	g.AdvanceLevel()
	g.Grid = generateGrid(g.Level)

	setupLevel(g)

	// Clear movement messages and show level info
	g.ClearMessages()
	logMessage(g, "You moved to deck %d!", g.Level)
	showLevelObjectives(g)
}

// showLevelObjectives displays the objectives for the current level
func showLevelObjectives(g *state.Game) {
	// Count doors
	numDoors := countDoors(g)
	if numDoors > 0 {
		logMessage(g, "Find keycards to unlock ACTION{%d} door(s).", numDoors)
	}
	if len(g.Generators) > 0 {
		logMessage(g, "Power up ACTION{%d} generator(s) with batteries.", len(g.Generators))
	}
	// Count hazards
	numHazards := countHazards(g)
	if numHazards > 0 {
		logMessage(g, "Clear ACTION{%d} environmental hazard(s).", numHazards)
	}
	if numDoors == 0 && len(g.Generators) == 0 && numHazards == 0 {
		logMessage(g, "Find the lift to the next deck.")
	}
}

// countDoors counts the number of locked doors on the map
func countDoors(g *state.Game) int {
	count := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell.HasLockedDoor() {
			count++
		}
	})
	return count
}

// countHazards counts the number of active hazards on the map
func countHazards(g *state.Game) int {
	count := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell.HasBlockingHazard() {
			count++
		}
	})
	return count
}

// canEnter checks if the player can enter a cell
func canEnter(g *state.Game, r *world.Cell, logReason bool) (bool, *world.ItemSet) {
	missingItems := mapset.New[*world.Item]()

	if r == nil || !r.Room {
		if logReason {
			logMessage(g, "There is nothing in that direction.")
		}

		return false, &missingItems
	}

	// Check for locked door
	if r.HasLockedDoor() {
		keycardName := r.Door.KeycardName()
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
				if cell.HasLockedDoor() && cell.Door.KeycardName() == keycardName {
					cell.Door.Unlock()
					doorsUnlocked++
				}
			})
			g.OwnedItems.Remove(keycardItem)
			if doorsUnlocked > 1 {
				logMessage(g, "Used %s to unlock %d doors to %s!", renderer.ColorKeycard.Sprintf(keycardName), doorsUnlocked, renderer.ColorCell.Sprintf(r.Door.RoomName))
			} else {
				logMessage(g, "Used %s to unlock the %s!", renderer.ColorKeycard.Sprintf(keycardName), renderer.ColorDoor.Sprintf(r.Door.DoorName()))
			}
		} else {
			if logReason {
				logMessage(g, "This door requires a %s", renderer.ColorKeycard.Sprintf(keycardName))
			}
			return false, &missingItems
		}
	}

	// Check for environmental hazard
	if r.HasBlockingHazard() {
		hazard := r.Hazard
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
				info := world.HazardTypes[hazard.Type]
				logMessage(g, info.FixedMessage)
			} else {
				if logReason {
					logMessage(g, renderer.ColorHazard.Sprintf(hazard.Description))
				}
				return false, &missingItems
			}
		} else {
			// Hazard requires a control panel to be activated
			if logReason {
				logMessage(g, renderer.ColorHazard.Sprintf(hazard.Description))
			}
			return false, &missingItems
		}
	}

	// Check for powered generators (only for exit cell)
	if r.ExitCell && !g.AllGeneratorsPowered() {
		if logReason {
			unpowered := g.UnpoweredGeneratorCount()
			logMessage(g, "The lift requires all generators to be powered!")
			logMessage(g, "ACTION{%d} generator(s) still need power.", unpowered)
		}
		return false, &missingItems
	}

	return true, &missingItems
}

// moveCell moves the player to a new cell
func moveCell(g *state.Game, requestedCell *world.Cell) {
	if res, _ := canEnter(g, requestedCell, true); res {
		// Only log message if entering a different named room
		if g.CurrentCell == nil || g.CurrentCell.Name != requestedCell.Name {
			logMessage(g, gettext.Gettext("OPEN_DOOR")+"%v", renderer.ColorCell.Sprintf(requestedCell.Name))
		}

		requestedCell.Visited = true

		// Reveal cells within field of view (radius 3, with line-of-sight blocking)
		world.RevealFOVDefault(g.Grid, requestedCell)

		g.CurrentCell = requestedCell
	}
}

// processInput handles player input
func processInput(g *state.Game, in string) {
	if in == "" {
		return
	}

	if in == "?" || in == "hint" {
		idx := rand.Intn(len(g.Hints))
		logMessage(g, g.Hints[idx])
		return
	}

	if in == "quit" || in == "q" {
		fmt.Println(gettext.Gettext("GOODBYE"))
		os.Exit(0)
	}

	// NSEW navigation
	if in == "east" || in == "e" || in == "arrow_right" {
		g.NavStyle = state.NavStyleNSEW
		moveCell(g, g.CurrentCell.East)
		return
	}

	if in == "west" || in == "w" || in == "arrow_left" {
		g.NavStyle = state.NavStyleNSEW
		moveCell(g, g.CurrentCell.West)
		return
	}

	if in == "north" || in == "n" || in == "arrow_up" {
		g.NavStyle = state.NavStyleNSEW
		moveCell(g, g.CurrentCell.North)
		return
	}

	if in == "south" || in == "s" || in == "arrow_down" {
		g.NavStyle = state.NavStyleNSEW
		moveCell(g, g.CurrentCell.South)
		return
	}

	// Vim-style navigation (hjkl)
	if in == "l" {
		g.NavStyle = state.NavStyleVim
		moveCell(g, g.CurrentCell.East)
		return
	}

	if in == "h" {
		g.NavStyle = state.NavStyleVim
		moveCell(g, g.CurrentCell.West)
		return
	}

	if in == "k" {
		g.NavStyle = state.NavStyleVim
		moveCell(g, g.CurrentCell.North)
		return
	}

	if in == "j" {
		g.NavStyle = state.NavStyleVim
		moveCell(g, g.CurrentCell.South)
		return
	}

	logMessage(g, gettext.Gettext("UNKNOWN_COMMAND"))
}

func main() {
	startLevel := flag.Int("level", 1, "starting level/deck number (for developer testing)")
	flag.Parse()

	initGettext()
	renderer.InitColors()

	rand.Seed(time.Now().UnixNano())

	g := buildGame(*startLevel)

	for {
		mainLoop(g)
	}
}

func mainLoop(g *state.Game) {
	renderer.Clear()

	if g.CurrentCell.ExitCell {
		logMessage(g, gettext.Gettext("EXIT"))
		advanceLevel(g)
	}

	// Level indicator in top left
	renderer.ColorAction.Printf("Deck %d\n\n", g.Level)

	renderer.PrintString("GT{IN_ROOM} ROOM{%v}\n\n", g.CurrentCell.Name)

	// Pick up items on the floor
	g.CurrentCell.ItemsOnFloor.Each(func(item *world.Item) {
		g.CurrentCell.ItemsOnFloor.Remove(item)

		if item.Name == "Map" {
			g.HasMap = true
			g.OwnedItems.Put(item)
			logMessage(g, "Picked up: ITEM{%v}", item.Name)
		} else if item.Name == "Battery" {
			g.AddBatteries(1)
			logMessage(g, "Picked up: ACTION{Battery}")
		} else {
			g.OwnedItems.Put(item)
			logMessage(g, "Picked up: ITEM{%v}", item.Name)
		}
	})

	// Check adjacent cells for unpowered generators and auto-insert batteries
	checkAdjacentGenerators(g)

	// Check adjacent cells for unused CCTV terminals
	checkAdjacentTerminals(g)

	// Check adjacent cells for inactive hazard controls
	checkAdjacentHazardControls(g)

	// Check adjacent cells for furniture and show hints
	checkAdjacentFurniture(g)

	renderer.PrintMap(g)

	renderer.PrintStatusBar(g)

	renderer.PrintPossibleActions()

	renderer.PrintMessagesPane(g)

	fmt.Printf("\n> ")

	processInput(g, input.GetInputWithArrows())
}

// checkAdjacentGenerators checks adjacent cells for unpowered generators and inserts batteries
func checkAdjacentGenerators(g *state.Game) {
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
		if cell == nil || !cell.HasUnpoweredGenerator() {
			continue
		}

		gen := cell.Generator
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
			} else {
				logMessage(g, "%s needs ACTION{%d} more batteries", gen.Name, gen.BatteriesNeeded())
			}
		}
	}
}

// checkAdjacentTerminals checks adjacent cells for unused CCTV terminals and activates them
func checkAdjacentTerminals(g *state.Game) {
	neighbors := []*world.Cell{
		g.CurrentCell.North,
		g.CurrentCell.East,
		g.CurrentCell.South,
		g.CurrentCell.West,
	}

	for _, cell := range neighbors {
		if cell == nil || !cell.HasUnusedTerminal() {
			continue
		}

		terminal := cell.Terminal
		targetRoom := terminal.TargetRoom

		// Check if the room is already fully revealed
		alreadyRevealed := isRoomFullyRevealed(g.Grid, targetRoom)

		if alreadyRevealed {
			logMessage(g, "Accessed %s - ROOM{%s} already explored.", terminal.Name, targetRoom)
			terminal.Activate()
		} else {
			// Reveal the target room
			if revealRoomByName(g.Grid, targetRoom) {
				terminal.Activate()
				logMessage(g, "Accessed %s - revealed ROOM{%s} on security feed!", terminal.Name, targetRoom)
			}
		}
	}
}

// checkAdjacentFurniture checks adjacent cells for unchecked furniture and examines them
func checkAdjacentFurniture(g *state.Game) {
	neighbors := []*world.Cell{
		g.CurrentCell.North,
		g.CurrentCell.East,
		g.CurrentCell.South,
		g.CurrentCell.West,
	}

	for _, cell := range neighbors {
		if cell == nil || !cell.HasUncheckedFurniture() {
			continue
		}

		furniture := cell.Furniture

		// Check the furniture and get any contained item
		item := furniture.Check()

		// Show the description
		logMessage(g, "%s: %s", renderer.ColorFurnitureCheck.Sprintf(furniture.Name), furniture.Description)

		// If furniture contained an item, give it to the player
		if item != nil {
			if item.Name == "Battery" {
				g.AddBatteries(1)
				logMessage(g, "Found inside: ACTION{Battery}")
			} else {
				g.OwnedItems.Put(item)
				logMessage(g, "Found inside: ITEM{%s}", item.Name)
			}
		}
	}
}

// checkAdjacentHazardControls checks adjacent cells for inactive hazard controls and activates them
func checkAdjacentHazardControls(g *state.Game) {
	neighbors := []*world.Cell{
		g.CurrentCell.North,
		g.CurrentCell.East,
		g.CurrentCell.South,
		g.CurrentCell.West,
	}

	for _, cell := range neighbors {
		if cell == nil || !cell.HasInactiveHazardControl() {
			continue
		}

		control := cell.HazardControl
		control.Activate()

		info := world.HazardTypes[control.Type]
		logMessage(g, "Activated %s: %s", renderer.ColorHazardCtrl.Sprintf(control.Name), info.FixedMessage)
	}
}

// containsSubstring checks if s contains substr
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

// findSubstring finds substr in s, returns -1 if not found
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
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
