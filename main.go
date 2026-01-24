package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/zyedidia/generic/mapset"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	gamemenu "darkstation/pkg/game/menu"
	"darkstation/pkg/game/renderer"
	ebitenRenderer "darkstation/pkg/game/renderer/ebiten"
	"darkstation/pkg/game/renderer/tui"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// Version information set via ldflags during build
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func initGettext() {
	gotext.Configure("mo", "en_GB.utf8", "default")
}

// logMessage adds a formatted message to the game's message log
func logMessage(g *state.Game, msg string, a ...any) {
	formatted := renderer.ApplyMarkup(msg, a...)
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
		g.AddHint("The " + renderer.StyledDenied(item.Name) + " is in " + renderer.StyledCell(room.Name))
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

	// Store seed before generation for reset functionality
	// Use level number as deterministic seed, or time-based for variety
	seed := time.Now().UnixNano()
	g.LevelSeed = seed
	rand.Seed(seed)

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
	// Level 1: 0 rooms (simplified), Level 2: 2 rooms, Level 3+: scales up
	numLockedRooms := 0
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
		door := entities.NewDoor(roomName)
		keycardName := door.KeycardName()

		keycard := world.NewItem(keycardName)
		keycardRoom.ItemsOnFloor.Put(keycard)
		avoid.Put(keycardRoom)
		g.AddHint("The " + renderer.StyledKeycard(keycardName) + " is in " + renderer.StyledCell(keycardRoom.Name))

		// Place doors on ALL entry cells (they share the same keycard)
		for _, cell := range entryCells {
			cellDoor := entities.NewDoor(roomName)
			gameworld.GetGameData(cell).Door = cellDoor
			avoid.Put(cell)
			lockedDoorCells.Put(cell)
		}

		roomsWithDoors.Put(roomName)
		lockedRoomsPlaced++

		if len(entryCells) == 1 {
			g.AddHint("The " + renderer.StyledDoor(door.DoorName()) + " blocks access to " + renderer.StyledCell(roomName))
		} else {
			g.AddHint(fmt.Sprintf("%d doors block access to %s", len(entryCells), renderer.StyledCell(roomName)))
		}
	}

	// Always place at least one generator in spawn room
	// Levels 3+: Exit requires generators (more generators needed)
	spawnCell := g.Grid.StartCell()
	spawnRoomName := spawnCell.Name

	// Find a valid cell in the spawn room for the generator (avoid chokepoints)
	var spawnRoomCell *world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.Room && cell.Name == spawnRoomName && spawnRoomCell == nil {
			data := gameworld.GetGameData(cell)
			// Check if cell is valid for generator placement and not a chokepoint
			if !avoid.Has(cell) && !cell.ExitCell &&
				data.Generator == nil && data.Door == nil && data.Terminal == nil &&
				data.Puzzle == nil && data.Furniture == nil && data.Hazard == nil &&
				data.HazardControl == nil && data.MaintenanceTerm == nil {
				// Prefer non-chokepoint cells to avoid blocking pathfinding
				if !isChokepoint(g.Grid, cell, spawnCell) {
					spawnRoomCell = cell
				}
			}
		}
	})

	// If no non-chokepoint cell found, use any valid cell
	if spawnRoomCell == nil {
		g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
			if cell != nil && cell.Room && cell.Name == spawnRoomName && spawnRoomCell == nil {
				data := gameworld.GetGameData(cell)
				if !avoid.Has(cell) && !cell.ExitCell &&
					data.Generator == nil && data.Door == nil && data.Terminal == nil &&
					data.Puzzle == nil && data.Furniture == nil && data.Hazard == nil &&
					data.HazardControl == nil && data.MaintenanceTerm == nil {
					spawnRoomCell = cell
				}
			}
		})
	}

	// Place generator in spawn room (always)
	if spawnRoomCell != nil {
		// Level 1-2: 1 battery, Level 3+: 1-3 batteries
		batteriesRequired := 1
		if g.Level >= 3 {
			batteriesRequired = 1 + rand.Intn(3) // 1-3 batteries
		}

		gen := entities.NewGenerator("Generator #1", batteriesRequired)
		// Auto-power the spawn room generator
		gen.InsertBatteries(batteriesRequired)
		gameworld.GetGameData(spawnRoomCell).Generator = gen
		g.AddGenerator(gen)
		avoid.Put(spawnRoomCell)

		// Update power supply immediately
		g.UpdatePowerSupply()

		g.AddHint("A generator is in " + renderer.StyledCell(spawnRoomName))
	}

	// Levels 3+: Exit requires additional generators
	if g.Level >= 3 {
		g.Grid.ExitCell().Locked = true

		// Place additional generators: level 3 = 1 more (total 2), level 4 = 2 more (total 3), etc.
		numAdditionalGenerators := g.Level - 3
		totalBatteriesNeeded := 0

		// Count batteries needed for spawn room generator (but it's already powered, so don't count it)
		// Spawn generator is already powered, so we don't need batteries for it

		for i := 0; i < numAdditionalGenerators; i++ {
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

			gen := entities.NewGenerator(fmt.Sprintf("Generator #%d", i+2), batteriesRequired)
			// Find a room and valid cell that won't block pathfinding
			genRoom := findRoom(g, g.Grid.StartCell(), &avoid)

			if genRoom != nil {
				// Find a valid cell in the room that's not a chokepoint
				var validGenCell *world.Cell
				g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
					if cell != nil && cell.Room && cell.Name == genRoom.Name && validGenCell == nil {
						data := gameworld.GetGameData(cell)
						// Check if cell is valid and not a chokepoint
						if !avoid.Has(cell) && !cell.ExitCell &&
							data.Generator == nil && data.Door == nil && data.Terminal == nil &&
							data.Puzzle == nil && data.Furniture == nil && data.Hazard == nil &&
							data.HazardControl == nil && data.MaintenanceTerm == nil {
							// Check if this is a chokepoint (would block pathfinding)
							if !isChokepoint(g.Grid, cell, g.Grid.StartCell()) {
								validGenCell = cell
							}
						}
					}
				})

				// If no non-chokepoint cell found, use the room center or any valid cell
				if validGenCell == nil {
					g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
						if cell != nil && cell.Room && cell.Name == genRoom.Name && validGenCell == nil {
							data := gameworld.GetGameData(cell)
							if !avoid.Has(cell) && !cell.ExitCell &&
								data.Generator == nil && data.Door == nil && data.Terminal == nil &&
								data.Puzzle == nil && data.Furniture == nil && data.Hazard == nil &&
								data.HazardControl == nil && data.MaintenanceTerm == nil {
								validGenCell = cell
							}
						}
					})
				}

				if validGenCell != nil {
					gameworld.GetGameData(validGenCell).Generator = gen
					g.AddGenerator(gen)
					avoid.Put(validGenCell)

					g.AddHint("A generator is in " + renderer.StyledCell(genRoom.Name))
				}
			}
		}

		// Place batteries: total needed (excluding already-powered spawn generator) + 1-2 extra per level for some buffer
		// Don't count spawn generator batteries since it's already powered
		spawnGenBatteries := 0
		if len(g.Generators) > 0 && g.Generators[0].IsPowered() {
			// Spawn generator is already powered, don't count its batteries
			spawnGenBatteries = g.Generators[0].BatteriesRequired
		}
		batteriesNeededForUnpowered := totalBatteriesNeeded - spawnGenBatteries
		extraBatteries := 1 + rand.Intn(2)
		totalBatteries := batteriesNeededForUnpowered + extraBatteries

		for i := 0; i < totalBatteries; i++ {
			battery := world.NewItem("Battery")
			placeItem(g, g.Grid.StartCell(), battery, &avoid)
		}
	} else {
		g.Grid.ExitCell().Locked = false

		// Level 1-2: Spawn generator is already powered, so no batteries needed
		// Batteries can be found in furniture for other uses
	}

	// Place environmental hazards (level 2+)
	if g.Level >= 2 {
		placeHazards(g, &avoid, &lockedDoorCells)
	}

	// Map is no longer automatically placed - it's now a puzzle reward in later levels

	// Place CCTV terminals (level 2+, 1-3 based on level)
	var numTerminals int
	if g.Level >= 2 {
		numTerminals = 1 + (g.Level-1)/3 // Level 2 = 1, Level 3 = 1, Level 4 = 2, etc.
		if numTerminals > 3 {
			numTerminals = 3
		}
	} else {
		numTerminals = 0 // No terminals on level 1
	}

	// Collect all unique room names (excluding corridors)
	roomNames := collectUniqueRoomNames(g.Grid)

	// roomEntries is already available from earlier in setupLevel (line 379)

	for i := 0; i < numTerminals; i++ {
		terminalRoom := findRoom(g, g.Grid.StartCell(), &avoid)
		if terminalRoom != nil && len(roomNames) > 0 {
			terminal := entities.NewCCTVTerminal(fmt.Sprintf("CCTV Terminal #%d", i+1))

			// Assign a random room for this terminal to reveal
			targetIdx := rand.Intn(len(roomNames))
			terminal.TargetRoom = roomNames[targetIdx]
			// Remove this room from the list so each terminal reveals a different room
			roomNames = append(roomNames[:targetIdx], roomNames[targetIdx+1:]...)

			// Find a valid cell in the room that doesn't block entrances
			roomName := terminalRoom.Name
			entryPoints := mapset.New[*world.Cell]()
			if entryData, ok := roomEntries[roomName]; ok {
				for _, entryCell := range entryData.entryCells {
					// Mark the room cells adjacent to entry points as blocked
					neighbors := []*world.Cell{entryCell.North, entryCell.East, entryCell.South, entryCell.West}
					for _, neighbor := range neighbors {
						if neighbor != nil && neighbor.Room && neighbor.Name == roomName {
							entryPoints.Put(neighbor)
						}
					}
				}
			}

			// Collect all cells in this room
			var roomCells []*world.Cell
			g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
				if cell != nil && cell.Room && cell.Name == roomName {
					roomCells = append(roomCells, cell)
				}
			})

			// Find valid cells (not entry points, not already used, not exit cells)
			var validCells []*world.Cell
			for _, cell := range roomCells {
				data := gameworld.GetGameData(cell)
				// Don't place terminal on entry points, exit cells, or cells with other entities
				if !avoid.Has(cell) && !cell.ExitCell && !entryPoints.Has(cell) &&
					data.Generator == nil && data.Door == nil && data.Terminal == nil &&
					data.Furniture == nil && data.Hazard == nil && data.HazardControl == nil {
					validCells = append(validCells, cell)
				}
			}

			// If no valid cells found, fall back to the original room (but this shouldn't happen)
			if len(validCells) == 0 {
				validCells = []*world.Cell{terminalRoom}
			}

			// Pick a random valid cell
			selectedCell := validCells[rand.Intn(len(validCells))]
			gameworld.GetGameData(selectedCell).Terminal = terminal
			avoid.Put(selectedCell)
		}
	}

	// Place furniture in rooms (1-2 pieces per unique room type)
	placeFurniture(g, &avoid)

	// Place puzzle terminals (level 2+)
	if g.Level >= 2 {
		placePuzzles(g, &avoid)
	}

	// Place maintenance terminals in every room (one per room, against walls)
	placeMaintenanceTerminals(g, &avoid)

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
	hazardTypes := []entities.HazardType{
		entities.HazardCoolant,
		entities.HazardElectrical,
		entities.HazardGas,
	}
	if g.Level >= 3 {
		hazardTypes = append(hazardTypes, entities.HazardVacuum)
	}
	if g.Level >= 5 {
		hazardTypes = append(hazardTypes, entities.HazardRadiation)
	}

	// Find room entry points (like doors, hazards block entry to rooms)
	roomEntries := findRoomEntryPoints(g.Grid)

	// Build list of candidate rooms (rooms with 1-3 entry points that we can fully block)
	type roomCandidate struct {
		name    string
		entries *roomEntryPoints
	}
	var candidates []roomCandidate
	for roomName, entries := range roomEntries {
		// Only consider rooms with 1-3 entry points (manageable to block)
		if len(entries.entryCells) >= 1 && len(entries.entryCells) <= 3 {
			candidates = append(candidates, roomCandidate{name: roomName, entries: entries})
		}
	}

	// Shuffle candidates for variety
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	// Track which rooms have hazards
	roomsWithHazards := mapset.New[string]()
	hazardsPlaced := 0

	// Place hazards to fully block selected rooms (like doors)
	for _, candidate := range candidates {
		if hazardsPlaced >= numHazards {
			break
		}

		roomName := candidate.name
		entryCells := candidate.entries.entryCells

		// Skip if already has hazards or doors
		if roomsWithHazards.Has(roomName) {
			continue
		}

		// Check if all entry cells are available and reachable
		currentlyReachable := getReachableCells(g.Grid, g.Grid.StartCell(), lockedDoorCells)
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
		testBlocked := mapset.New[*world.Cell]()
		lockedDoorCells.Each(func(c *world.Cell) { testBlocked.Put(c) })
		for _, cell := range entryCells {
			testBlocked.Put(cell)
		}
		reachableWithHazard := getReachableCells(g.Grid, g.Grid.StartCell(), &testBlocked)

		// Must actually block something
		if reachableWithHazard.Size() >= currentlyReachable.Size() {
			continue
		}

		// Choose a random hazard type
		hazardType := hazardTypes[rand.Intn(len(hazardTypes))]
		hazard := entities.NewHazard(hazardType)
		info := entities.HazardTypes[hazardType]

		// Place hazards on ALL entry cells (they share the same solution)
		for _, cell := range entryCells {
			gameworld.GetGameData(cell).Hazard = hazard
			avoid.Put(cell)
		}

		roomsWithHazards.Put(roomName)
		hazardsPlaced++

		// Place the solution (item or control) in the area reachable BEFORE these hazards
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
				g.AddHint("A " + renderer.StyledItem(info.ItemName) + " is in " + renderer.StyledCell(itemRoom.Name))
			}
		} else {
			// Place the control panel in a reachable area
			controlRoom := findRoomInReachable(reachableWithHazard, avoid)
			if controlRoom == nil {
				controlRoom = findRoomInReachable(currentlyReachable, avoid)
			}
			if controlRoom != nil {
				control := entities.NewHazardControl(hazardType, hazard)
				gameworld.GetGameData(controlRoom).HazardControl = control
				avoid.Put(controlRoom)
				g.AddHint("The " + renderer.StyledHazardCtrl(info.ControlName) + " is in " + renderer.StyledCell(controlRoom.Name))
			}
		}

		if len(entryCells) == 1 {
			g.AddHint(fmt.Sprintf("A %s blocks access to %s", info.Name, renderer.StyledCell(roomName)))
		} else {
			g.AddHint(fmt.Sprintf("%d %s hazards block access to %s", len(entryCells), info.Name, renderer.StyledCell(roomName)))
		}
	}
}

// placeFurnitureLimited places exactly maxCount pieces of furniture across all rooms
func placeFurnitureLimited(g *state.Game, avoid *mapset.Set[*world.Cell], maxCount int) {
	// Collect all unique rooms and their cells
	roomCells := make(map[string][]*world.Cell)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell.Room && cell.Name != "Corridor" && cell.Name != "" {
			roomCells[cell.Name] = append(roomCells[cell.Name], cell)
		}
	})

	// Get room entry points to avoid blocking them
	roomEntries := findRoomEntryPoints(g.Grid)

	// Collect all valid cells across all rooms
	var allValidCells []*world.Cell
	roomNames := make([]string, 0, len(roomCells))
	for roomName := range roomCells {
		roomNames = append(roomNames, roomName)
	}

	// Shuffle room order for variety
	rand.Shuffle(len(roomNames), func(i, j int) {
		roomNames[i], roomNames[j] = roomNames[j], roomNames[i]
	})

	// Collect valid cells from all rooms
	for _, roomName := range roomNames {
		cells := roomCells[roomName]
		templates := entities.GetAllFurnitureForRoom(roomName)
		if len(templates) == 0 {
			continue
		}

		entryPoints := mapset.New[*world.Cell]()
		if entryData, ok := roomEntries[roomName]; ok {
			for _, entryCell := range entryData.entryCells {
				neighbors := []*world.Cell{entryCell.North, entryCell.East, entryCell.South, entryCell.West}
				for _, neighbor := range neighbors {
					if neighbor != nil && neighbor.Room && neighbor.Name == roomName {
						entryPoints.Put(neighbor)
					}
				}
			}
		}

		for _, cell := range cells {
			data := gameworld.GetGameData(cell)
			if !avoid.Has(cell) && !cell.ExitCell && !entryPoints.Has(cell) &&
				data.Generator == nil && data.Door == nil && data.Terminal == nil &&
				data.Furniture == nil && data.Hazard == nil && data.HazardControl == nil {
				allValidCells = append(allValidCells, cell)
			}
		}
	}

	// Shuffle all valid cells
	rand.Shuffle(len(allValidCells), func(i, j int) {
		allValidCells[i], allValidCells[j] = allValidCells[j], allValidCells[i]
	})

	// Place exactly maxCount pieces
	placed := 0
	for _, cell := range allValidCells {
		if placed >= maxCount {
			break
		}

		roomName := cell.Name
		templates := entities.GetAllFurnitureForRoom(roomName)
		if len(templates) == 0 {
			continue
		}

		// Pick a random template for this room
		template := templates[rand.Intn(len(templates))]
		furniture := entities.NewFurniture(template.Name, template.Description, template.Icon)
		gameworld.GetGameData(cell).Furniture = furniture
		avoid.Put(cell)
		placed++
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
		templates := entities.GetAllFurnitureForRoom(roomName)
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

		// Find valid cells (not already used for something else, and not blocking entrances/exits)
		// Get room entry points to avoid blocking them
		roomEntries := findRoomEntryPoints(g.Grid)
		entryPoints := mapset.New[*world.Cell]()
		if entryData, ok := roomEntries[roomName]; ok {
			for _, entryCell := range entryData.entryCells {
				// Mark the room cells adjacent to entry points as blocked
				neighbors := []*world.Cell{entryCell.North, entryCell.East, entryCell.South, entryCell.West}
				for _, neighbor := range neighbors {
					if neighbor != nil && neighbor.Room && neighbor.Name == roomName {
						entryPoints.Put(neighbor)
					}
				}
			}
		}

		var validCells []*world.Cell
		for _, cell := range cells {
			data := gameworld.GetGameData(cell)
			// Don't place furniture on entry points, exit cells, or cells with other entities
			if !avoid.Has(cell) && !cell.ExitCell && !entryPoints.Has(cell) &&
				data.Generator == nil && data.Door == nil && data.Terminal == nil &&
				data.Furniture == nil && data.Hazard == nil && data.HazardControl == nil {
				validCells = append(validCells, cell)
			}
		}

		// Shuffle valid cells
		rand.Shuffle(len(validCells), func(i, j int) {
			validCells[i], validCells[j] = validCells[j], validCells[i]
		})

		// Track furniture placed in this room for item hiding
		var placedFurniture []*entities.Furniture

		// Place furniture
		for i := 0; i < numFurniture && i < len(validCells); i++ {
			template := templates[i]
			furniture := entities.NewFurniture(template.Name, template.Description, template.Icon)
			gameworld.GetGameData(validCells[i]).Furniture = furniture
			placedFurniture = append(placedFurniture, furniture)
		}

		// Chance to hide items from the floor in furniture (40% per item)
		if len(placedFurniture) > 0 {
			hideItemsInFurniture(g, cells, placedFurniture, roomName)
		}
	}
}

// placePuzzles places puzzle terminals that require codes found in furniture
func placePuzzles(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	// Place 1-2 puzzles per level (level 2+)
	numPuzzles := 1
	if g.Level >= 5 {
		numPuzzles = 2
	} else if g.Level >= 3 {
		numPuzzles = 2
	}

	// Generate puzzle solutions
	puzzleSolutions := []string{
		"1-2-3-4",
		"2-4-6-8",
		"up-down-left-right",
		"north-south-east-west",
		"alpha-beta-gamma-delta",
	}

	for i := 0; i < numPuzzles && i < len(puzzleSolutions); i++ {
		// Find a room for the puzzle
		puzzleRoom := findRoom(g, g.Grid.StartCell(), avoid)
		if puzzleRoom == nil {
			continue
		}

		solution := puzzleSolutions[i]
		puzzleType := entities.PuzzleSequence
		if strings.Contains(solution, "-") && !strings.ContainsAny(solution, "0123456789") {
			puzzleType = entities.PuzzlePattern
		}

		// Create puzzle with appropriate reward based on level
		reward := entities.RewardBattery
		if g.Level >= 6 && i == 0 {
			// First puzzle on level 6+ gives the map (powerful reward for complex puzzles)
			// Maps are only available as puzzle rewards, never as items
			// This makes the map a late-game reward that requires significant puzzle-solving
			reward = entities.RewardMap
		} else if g.Level >= 3 && i == 0 {
			// First puzzle on level 3 gives a keycard hint
			reward = entities.RewardKeycard
		}

		puzzle := entities.NewPuzzleTerminal(
			fmt.Sprintf("Security Terminal #%d", i+1),
			puzzleType,
			solution,
			fmt.Sprintf("Find the code in logs or furniture descriptions. Look for: Code: %s", solution),
			reward,
			"A security terminal requiring an access code.",
		)

		gameworld.GetGameData(puzzleRoom).Puzzle = puzzle
		avoid.Put(puzzleRoom)

		// Place the code in a furniture description in a different room
		codeRoom := findRoom(g, g.Grid.StartCell(), avoid)
		if codeRoom != nil && codeRoom != puzzleRoom {
			// Find furniture in this room and add code to its description
			roomCells := []*world.Cell{}
			g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
				if cell.Room && cell.Name == codeRoom.Name && gameworld.HasFurniture(cell) {
					roomCells = append(roomCells, cell)
				}
			})

			if len(roomCells) > 0 {
				// Pick a random furniture in this room
				furnitureCell := roomCells[rand.Intn(len(roomCells))]
				furniture := gameworld.GetGameData(furnitureCell).Furniture
				// Append code to description
				furniture.Description += fmt.Sprintf(" Code: %s", solution)
			}
			avoid.Put(codeRoom)
		}

		g.AddHint(fmt.Sprintf("A puzzle terminal is in %s", renderer.StyledCell(puzzleRoom.Name)))
	}
}

// placeMaintenanceTerminals places one maintenance terminal per room, aligned against walls
func placeMaintenanceTerminals(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	// Collect all unique rooms
	roomCells := make(map[string][]*world.Cell)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell.Room && cell.Name != "Corridor" && cell.Name != "" {
			roomCells[cell.Name] = append(roomCells[cell.Name], cell)
		}
	})

	// Get room entry points to avoid blocking them
	roomEntries := findRoomEntryPoints(g.Grid)

	// Place one maintenance terminal per room
	for roomName, cells := range roomCells {
		if len(cells) == 0 {
			continue
		}

		// Find cells that are against walls (have at least one non-room neighbor OR corridor neighbor)
		// Also prefer cells on the edge of the room (fewer room neighbors)
		var wallCells []*world.Cell
		var edgeCells []*world.Cell

		for _, cell := range cells {
			data := gameworld.GetGameData(cell)

			// Skip if already has entities
			if !avoid.Has(cell) && !cell.ExitCell &&
				data.Generator == nil && data.Door == nil && data.Terminal == nil &&
				data.Puzzle == nil && data.Furniture == nil && data.Hazard == nil &&
				data.HazardControl == nil && data.MaintenanceTerm == nil {

				// Check if cell is against a wall (has a non-room neighbor)
				isWallCell := false
				neighbors := []*world.Cell{cell.North, cell.East, cell.South, cell.West}
				roomNeighborCount := 0

				for _, neighbor := range neighbors {
					if neighbor == nil {
						isWallCell = true // Edge of map
					} else if !neighbor.Room {
						isWallCell = true // Wall
					} else if neighbor.Room && neighbor.Name == roomName {
						roomNeighborCount++
					}
				}

				// Check entry points
				entryPoints := mapset.New[*world.Cell]()
				if entryData, ok := roomEntries[roomName]; ok {
					for _, entryCell := range entryData.entryCells {
						entryNeighbors := []*world.Cell{entryCell.North, entryCell.East, entryCell.South, entryCell.West}
						for _, neighbor := range entryNeighbors {
							if neighbor != nil && neighbor.Room && neighbor.Name == roomName {
								entryPoints.Put(neighbor)
							}
						}
					}
				}

				if !entryPoints.Has(cell) {
					if isWallCell {
						wallCells = append(wallCells, cell)
					} else if roomNeighborCount <= 2 {
						// Edge of room (2 or fewer room neighbors)
						edgeCells = append(edgeCells, cell)
					}
				}
			}
		}

		// Prefer wall cells, fall back to edge cells, then any valid cell
		var validCells []*world.Cell
		if len(wallCells) > 0 {
			validCells = wallCells
		} else if len(edgeCells) > 0 {
			validCells = edgeCells
		} else {
			// Last resort: any valid cell in the room
			for _, cell := range cells {
				data := gameworld.GetGameData(cell)
				if !avoid.Has(cell) && !cell.ExitCell &&
					data.Generator == nil && data.Door == nil && data.Terminal == nil &&
					data.Puzzle == nil && data.Furniture == nil && data.Hazard == nil &&
					data.HazardControl == nil && data.MaintenanceTerm == nil {
					validCells = append(validCells, cell)
				}
			}
		}

		// Place maintenance terminal
		if len(validCells) > 0 {
			selectedCell := validCells[rand.Intn(len(validCells))]
			maintenanceTerm := entities.NewMaintenanceTerminal(fmt.Sprintf("Maintenance Terminal - %s", roomName), roomName)
			gameworld.GetGameData(selectedCell).MaintenanceTerm = maintenanceTerm
			avoid.Put(selectedCell)
		}
	}
}

// hideItemsInFurniture moves items from floor cells into furniture with a chance
func hideItemsInFurniture(g *state.Game, roomCells []*world.Cell, furniture []*entities.Furniture, roomName string) {
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
					updateHintForFurnitureItem(g, item, f, roomName)
					break
				}
			}
		}
	}
}

// updateHintForFurnitureItem updates the hint for an item to mention it's in furniture
func updateHintForFurnitureItem(g *state.Game, item *world.Item, furniture *entities.Furniture, roomName string) {
	// Find and update the existing hint for this item
	for i, hint := range g.Hints {
		if containsSubstring(hint, item.Name) && containsSubstring(hint, roomName) {
			// Replace the hint with one mentioning the furniture
			g.Hints[i] = "The " + renderer.StyledKeycard(item.Name) + " is hidden in the " +
				renderer.StyledFurniture(furniture.Name) + " in " + renderer.StyledCell(roomName)
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

// resetLevel resets the current level using the same seed/map
func resetLevel(g *state.Game) {
	currentLevel := g.Level

	// Clear inventory and level-specific state
	g.OwnedItems = mapset.New[*world.Item]()
	g.Batteries = 0
	g.HasMap = false
	g.FoundCodes = make(map[string]bool)
	g.Generators = make([]*entities.Generator, 0)
	g.Hints = nil
	g.PowerSupply = 0
	g.PowerConsumption = 0
	g.PowerOverloadWarned = false

	// Reset interaction/movement counters
	g.MovementCount = 0
	g.InteractionsCount = 0
	g.LastInteractedRow = -1
	g.LastInteractedCol = -1
	g.InteractionPlayerRow = -1
	g.InteractionPlayerCol = -1

	// Reset exit animation state
	g.ExitAnimating = false
	g.ExitAnimStartTime = 0

	// Regenerate grid with the same seed (or use level as seed if not set)
	var seed int64
	if g.LevelSeed != 0 {
		seed = g.LevelSeed
	} else {
		// Fallback: use level number as seed (deterministic)
		seed = int64(currentLevel)
	}

	// Set seed before generating to ensure same map layout
	rand.Seed(seed)
	g.Grid = generateGrid(currentLevel)

	// Setup level again (will place entities in same positions due to same seed)
	setupLevel(g)

	// Store the seed for future resets
	g.LevelSeed = seed

	// Update power and lighting after setup
	updateLightingExploration(g)

	// Clear messages and show reset message
	g.ClearMessages()
	logMessage(g, "Level reset!")
	logMessage(g, "You are on deck %d.", g.Level)
	showLevelObjectives(g)
}

// advanceLevel generates a new map and advances to the next level
func advanceLevel(g *state.Game) {
	g.AdvanceLevel()

	// Store seed for new level (for reset functionality)
	seed := time.Now().UnixNano()
	g.LevelSeed = seed
	rand.Seed(seed)

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
	// Note: Keycard message removed - players can discover locked doors naturally
	if len(g.Generators) > 0 {
		logMessage(g, "Power up ACTION{%d} generator(s) with batteries.", len(g.Generators))
	}
	// Count hazards
	numHazards := countHazards(g)
	if numHazards > 0 {
		logMessage(g, "Clear ACTION{%d} environmental hazard(s).", numHazards)
	}
	if numDoors == 0 && len(g.Generators) == 0 && numHazards == 0 {
		logMessage(g, "Find the EXIT{lift} to the next deck.")
	}
}

// countDoors counts the number of locked doors on the map
func countDoors(g *state.Game) int {
	count := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if gameworld.HasLockedDoor(cell) {
			count++
		}
	})
	return count
}

// countHazards counts the number of active hazards on the map
func countHazards(g *state.Game) int {
	count := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if gameworld.HasBlockingHazard(cell) {
			count++
		}
	})
	return count
}

// canEnter checks if the player can enter a cell
func canEnter(g *state.Game, r *world.Cell, logReason bool) (bool, *world.ItemSet) {
	missingItems := mapset.New[*world.Item]()

	if r == nil || !r.Room {
		return false, &missingItems
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
				renderer.AddCallout(r.Row, r.Col, fmt.Sprintf("Locked: %s", keycardName), renderer.CalloutColorDoor, 0)
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

// moveCell moves the player to a new cell
func moveCell(g *state.Game, requestedCell *world.Cell) {
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

	if res, _ := canEnter(g, requestedCell, true); res {
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
		updateLightingExploration(g)

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

// processIntent handles a high-level input intent from the tiered input system.
func processIntent(g *state.Game, intent engineinput.Intent) {
	switch intent.Action {
	case engineinput.ActionNone:
		return

	case engineinput.ActionOpenMenu:
		runBindingsMenu(g)
		return

	case engineinput.ActionHint:
		idx := rand.Intn(len(g.Hints))
		logMessage(g, "%s", g.Hints[idx])
		return

	case engineinput.ActionQuit:
		fmt.Println(gotext.Get("GOODBYE"))
		os.Exit(0)

	case engineinput.ActionScreenshot:
		filename := saveScreenshotHTML(g)
		logMessage(g, "Screenshot saved to ITEM{%s}", filename)
		return

	case engineinput.ActionDevMap:
		switchToDevMap(g)
		return

	case engineinput.ActionResetLevel:
		resetLevel(g)
		return

	case engineinput.ActionMoveEast:
		g.NavStyle = state.NavStyleNSEW
		moveCell(g, g.CurrentCell.East)
		return

	case engineinput.ActionMoveWest:
		g.NavStyle = state.NavStyleNSEW
		moveCell(g, g.CurrentCell.West)
		return

	case engineinput.ActionMoveNorth:
		g.NavStyle = state.NavStyleNSEW
		moveCell(g, g.CurrentCell.North)
		return

	case engineinput.ActionMoveSouth:
		g.NavStyle = state.NavStyleNSEW
		moveCell(g, g.CurrentCell.South)
		return

	case engineinput.ActionInteract:
		// Check for adjacent interactables in NSEW priority order, cycling through them
		interacted := checkAdjacentInteractables(g)
		if !interacted {
			logMessage(g, "Nothing to interact with here.")
		}
		return
	}

	logMessage(g, "%s", gotext.Get("UNKNOWN_COMMAND"))
}

func main() {
	startLevel := flag.Int("level", 1, "starting level/deck number (for developer testing)")
	useEbiten := flag.Bool("ebiten", false, "use Ebiten graphical renderer instead of TUI")
	flag.Parse()

	// Check for LEVEL environment variable (takes precedence over flag)
	if envLevel := os.Getenv("LEVEL"); envLevel != "" {
		if parsedLevel, err := strconv.Atoi(envLevel); err == nil && parsedLevel > 0 {
			*startLevel = parsedLevel
		}
	}

	initGettext()
	rand.Seed(time.Now().UnixNano())

	// Set version information for renderers
	renderer.SetVersion(version, commit, date)

	if *useEbiten {
		// Initialize the Ebiten renderer
		ebitRenderer := ebitenRenderer.New()
		renderer.SetRenderer(ebitRenderer)
		renderer.Init()

		g := buildGame(*startLevel)

		// Run the game with Ebiten's game loop
		// Ebiten's RunGame blocks, so we run the game logic in Update
		if err := ebitRenderer.RunWithGameLoop(func() {
			for {
				mainLoop(g)
			}
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error running Ebiten: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Initialize the TUI renderer
		tuiRenderer := tui.New()
		renderer.SetRenderer(tuiRenderer)
		renderer.Init()

		g := buildGame(*startLevel)

		for {
			mainLoop(g)
		}
	}
}

func mainLoop(g *state.Game) {
	renderer.Clear()

	// Clear callouts if player moved (before adding new ones)
	renderer.ClearCalloutsIfMoved(g.CurrentCell.Row, g.CurrentCell.Col)

	// Show room entry callout if player entered a new room (but not corridors)
	renderer.ShowRoomEntryIfNew(g.CurrentCell.Row, g.CurrentCell.Col, g.CurrentCell.Name)

	// Check exit animation state
	if g.ExitAnimating {
		elapsed := time.Now().UnixMilli() - g.ExitAnimStartTime
		const exitAnimDuration = 2000 // 2 seconds (matches drawExitAnimation)
		if elapsed >= exitAnimDuration {
			// Animation complete, advance to next level
			g.ExitAnimating = false
			advanceLevel(g)
		}
	} else if g.CurrentCell.ExitCell {
		// Start exit animation when player enters exit
		if !g.ExitAnimating {
			g.ExitAnimating = true
			g.ExitAnimStartTime = time.Now().UnixMilli()
		}
	}

	// Pick up items on the floor
	g.CurrentCell.ItemsOnFloor.Each(func(item *world.Item) {
		g.CurrentCell.ItemsOnFloor.Remove(item)

		if item.Name == "Map" {
			// Maps should no longer be found as items - they're puzzle rewards only
			// But handle it gracefully if one somehow appears
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

	// Check adjacent cells for unpowered generators and auto-insert batteries
	checkAdjacentGenerators(g)

	// Update lighting and power consumption
	updateLightingExploration(g)

	// Remove messages older than 10 seconds from the buffer
	g.RemoveOldMessages()

	// Show hints for interactable objects (only for first 3 interactions)
	showInteractableHints(g)

	// Show movement hint (only for first 3 movements)
	showMovementHint(g)

	// Hazard controls, terminals, and puzzles now require explicit interaction (handled in ActionInteract)

	// Check adjacent cells for furniture and show hints
	// Furniture interaction is now explicit (handled in processIntent)

	// Render the complete game frame
	renderer.RenderFrame(g)

	// If exit animation is running, continue loop without waiting for input
	// This allows the animation to complete automatically
	if g.ExitAnimating {
		// Small delay to allow animation to render smoothly
		time.Sleep(16 * time.Millisecond) // ~60 FPS
		return
	}

	// Get and process input (tiered input system -> Intent -> game logic)
	processIntent(g, renderer.Current.GetInput())
}

// isNonRebindable checks if an action cannot be rebound
func isNonRebindable(action engineinput.Action) bool {
	return action == engineinput.ActionInteract ||
		action == engineinput.ActionZoomIn ||
		action == engineinput.ActionZoomOut
}

// runBindingsMenu presents a simple bindings configuration menu using the generic menu system.
func runBindingsMenu(g *state.Game) {
	handler := gamemenu.NewBindingsMenuHandler()
	items := handler.GetMenuItems()
	gamemenu.RunMenu(g, items, handler)
}

// checkAdjacentGeneratorAtCell checks a specific cell for generator and shows power info
// Returns true if generator was interacted with
func checkAdjacentGeneratorAtCell(g *state.Game, cell *world.Cell) bool {
	if cell == nil || !gameworld.HasGenerator(cell) {
		return false
	}

	gen := gameworld.GetGameData(cell).Generator

	// Build tooltip message with generator status and power information
	var calloutText strings.Builder
	calloutText.WriteString(fmt.Sprintf("=== %s ===\n", gen.Name))

	if gen.IsPowered() {
		calloutText.WriteString("Status: POWERED\n")
		calloutText.WriteString(fmt.Sprintf("Batteries: ACTION{%d}/ACTION{%d}\n", gen.BatteriesInserted, gen.BatteriesRequired))
	} else {
		calloutText.WriteString("Status: UNPOWERED\n")
		calloutText.WriteString(fmt.Sprintf("Batteries: ACTION{%d}/ACTION{%d}\n", gen.BatteriesInserted, gen.BatteriesRequired))
		calloutText.WriteString(fmt.Sprintf("Needs: ACTION{%d} more batteries\n", gen.BatteriesNeeded()))
	}
	calloutText.WriteString("\n")
	calloutText.WriteString(fmt.Sprintf("Power Supply: ACTION{%d} watts\n", g.PowerSupply))
	calloutText.WriteString(fmt.Sprintf("Power Consumption: ACTION{%d} watts\n", g.PowerConsumption))
	calloutText.WriteString(fmt.Sprintf("Available Power: ACTION{%d} watts", g.GetAvailablePower()))

	// Use appropriate color based on power status
	calloutColor := renderer.CalloutColorGenerator
	if gen.IsPowered() {
		calloutColor = renderer.CalloutColorGeneratorOn
	}

	renderer.AddCallout(cell.Row, cell.Col, calloutText.String(), calloutColor, 0)

	return true
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
				renderer.AddCallout(cell.Row, cell.Col, fmt.Sprintf("%s POWERED!", gen.Name), renderer.CalloutColorGeneratorOn, 0)
				// Update power supply when generator is powered
				g.UpdatePowerSupply()
				// Update lighting based on new power availability
				updateLightingExploration(g)
				logMessage(g, "Power supply: ACTION{%d} watts available", g.GetAvailablePower())
			} else {
				logMessage(g, "%s needs ACTION{%d} more batteries", gen.Name, gen.BatteriesNeeded())
				renderer.AddCallout(cell.Row, cell.Col, fmt.Sprintf("+%d batteries (%d more needed)", inserted, gen.BatteriesNeeded()), renderer.CalloutColorGenerator, 0)
			}
		}
	}
}

// checkAdjacentTerminalsAtCell checks a specific cell for terminals and interacts with it
// Returns true if a terminal was interacted with
func checkAdjacentTerminalsAtCell(g *state.Game, cell *world.Cell) bool {
	if cell == nil || !gameworld.HasUnusedTerminal(cell) {
		return false
	}

	terminal := gameworld.GetGameData(cell).Terminal
	targetRoom := terminal.TargetRoom

	// Check if the room is already fully revealed
	alreadyRevealed := isRoomFullyRevealed(g.Grid, targetRoom)

	if alreadyRevealed {
		logMessage(g, "Accessed %s - ROOM{%s} already explored.", terminal.Name, targetRoom)
		terminal.Activate()
		renderer.AddCallout(cell.Row, cell.Col, fmt.Sprintf("%s already explored", targetRoom), renderer.CalloutColorTerminal, 0)
	} else {
		// Reveal the target room
		if revealRoomByName(g.Grid, targetRoom) {
			terminal.Activate()
			logMessage(g, "Accessed %s - revealed ROOM{%s} on security feed!", terminal.Name, targetRoom)
			renderer.AddCallout(cell.Row, cell.Col, fmt.Sprintf("Revealed: %s", targetRoom), renderer.CalloutColorTerminal, 0)
		}
	}
	return true
}

// checkAdjacentTerminals checks adjacent cells for unused CCTV terminals and activates them
// DEPRECATED: Use checkAdjacentInteractables instead for priority-based interaction
// Returns true if a terminal was interacted with
func checkAdjacentTerminals(g *state.Game) bool {
	neighbors := []*world.Cell{
		g.CurrentCell.North,
		g.CurrentCell.East,
		g.CurrentCell.South,
		g.CurrentCell.West,
	}

	for _, cell := range neighbors {
		if checkAdjacentTerminalsAtCell(g, cell) {
			return true
		}
	}
	return false
}

// checkAdjacentPuzzlesAtCell checks a specific cell for puzzles and interacts with it
// Returns true if a puzzle was interacted with
func checkAdjacentPuzzlesAtCell(g *state.Game, cell *world.Cell) bool {
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
			renderer.AddCallout(cell.Row, cell.Col, "Puzzle solved!", renderer.CalloutColorTerminal, 0)
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

// checkAdjacentPuzzles checks adjacent cells for unsolved puzzles and allows interaction
// DEPRECATED: Use checkAdjacentInteractables instead for priority-based interaction
// Returns true if a puzzle was interacted with
func checkAdjacentPuzzles(g *state.Game) bool {
	neighbors := []*world.Cell{
		g.CurrentCell.North,
		g.CurrentCell.East,
		g.CurrentCell.South,
		g.CurrentCell.West,
	}

	for _, cell := range neighbors {
		if checkAdjacentPuzzlesAtCell(g, cell) {
			return true
		}
	}
	return false
}

// checkForPuzzleCode extracts puzzle codes from text and adds them to found codes
func checkForPuzzleCode(g *state.Game, text string) {
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
		renderer.AddCallout(cell.Row, cell.Col, "Battery received!", renderer.CalloutColorItem, 0)
	case entities.RewardRevealRoom:
		// Reveal a random room
		logMessage(g, "Security feed activated - a new area is revealed.")
	case entities.RewardUnlockArea:
		// Unlock a previously locked area
		logMessage(g, "Access granted to a previously locked section.")
	case entities.RewardMap:
		// Give the player the map - powerful reward!
		g.HasMap = true
		renderer.AddCallout(cell.Row, cell.Col, "Map acquired!", renderer.CalloutColorItem, 0)
		logMessage(g, "Received: ITEM{Map}")
	}
}

// showMovementHint shows a callout hint next to the player for movement controls
// Only shows hint if the player has moved fewer than 3 times
func showMovementHint(g *state.Game) {
	// Only show hint for the first 3 movements
	if g.MovementCount >= 3 {
		return
	}

	// Show hint next to the player
	if g.CurrentCell != nil {
		renderer.AddCallout(g.CurrentCell.Row, g.CurrentCell.Col, "Press WASD or arrow keys to move", renderer.CalloutColorInfo, 0)
	}
}

// showInteractableHints shows callout hints for interactable objects adjacent to the player
// Only shows hints if the player has interacted with fewer than 3 objects
func showInteractableHints(g *state.Game) {
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

// checkAdjacentInteractables checks adjacent cells in NSEW priority order for interactables
// Cycles through interactables when player hasn't moved, skipping previously interacted cells
// Returns true if an interaction occurred
func checkAdjacentInteractables(g *state.Game) bool {
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
			if checkAdjacentGeneratorAtCell(g, cell) {
				g.LastInteractedRow = cell.Row
				g.LastInteractedCol = cell.Col
				g.InteractionsCount++
				return true
			}
		}
		if gameworld.HasFurniture(cell) {
			if checkAdjacentFurnitureAtCell(g, cell) {
				g.LastInteractedRow = cell.Row
				g.LastInteractedCol = cell.Col
				g.InteractionsCount++
				return true
			}
		}
		if gameworld.HasUnusedTerminal(cell) {
			if checkAdjacentTerminalsAtCell(g, cell) {
				g.LastInteractedRow = cell.Row
				g.LastInteractedCol = cell.Col
				g.InteractionsCount++
				return true
			}
		}
		if gameworld.HasUnsolvedPuzzle(cell) {
			if checkAdjacentPuzzlesAtCell(g, cell) {
				g.LastInteractedRow = cell.Row
				g.LastInteractedCol = cell.Col
				g.InteractionsCount++
				return true
			}
		}
		if gameworld.HasInactiveHazardControl(cell) {
			if checkAdjacentHazardControlsAtCell(g, cell) {
				g.LastInteractedRow = cell.Row
				g.LastInteractedCol = cell.Col
				g.InteractionsCount++
				return true
			}
		}
		if gameworld.HasMaintenanceTerminal(cell) {
			if checkAdjacentMaintenanceTerminalAtCell(g, cell) {
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

// checkAdjacentFurnitureAtCell checks a specific cell for furniture and interacts with it
// Returns true if furniture was interacted with
// Furniture can be interacted with multiple times, but items are only given once
func checkAdjacentFurnitureAtCell(g *state.Game, cell *world.Cell) bool {
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
		checkForPuzzleCode(g, furniture.Description)
	}

	// If furniture contained an item, give it to the player and show callout
	if item != nil {
		if item.Name == "Battery" {
			g.AddBatteries(1)
			// Show furniture name in furniture color, then "Found: Battery!" in item color
			calloutText := fmt.Sprintf("FURNITURE{%s}\nFound: ACTION{Battery}!", furniture.Name)
			renderer.AddCallout(cell.Row, cell.Col, calloutText, renderer.CalloutColorFurnitureChecked, 0)
		} else {
			g.OwnedItems.Put(item)
			// Show furniture name in furniture color, then "Found: [Item]!" in item color
			calloutText := fmt.Sprintf("FURNITURE{%s}\nFound: ITEM{%s}!", furniture.Name, item.Name)
			renderer.AddCallout(cell.Row, cell.Col, calloutText, renderer.CalloutColorFurnitureChecked, 0)
		}
	} else {
		// Furniture already checked or decorative-only: show description
		// Show furniture name on first line, description on second line
		calloutText := fmt.Sprintf("FURNITURE{%s}\n%s", furniture.Name, furniture.Description)
		renderer.AddCallout(cell.Row, cell.Col, calloutText, renderer.CalloutColorFurnitureChecked, 0)
	}
	return true
}

// checkAdjacentFurniture checks adjacent cells for furniture and allows interaction
// DEPRECATED: Use checkAdjacentInteractables instead for priority-based interaction
// Returns true if furniture was interacted with
// Furniture can be interacted with multiple times, but items are only given once
func checkAdjacentFurniture(g *state.Game) bool {
	neighbors := []*world.Cell{
		g.CurrentCell.North,
		g.CurrentCell.East,
		g.CurrentCell.South,
		g.CurrentCell.West,
	}

	for _, cell := range neighbors {
		if cell == nil || !gameworld.HasFurniture(cell) {
			continue
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
			checkForPuzzleCode(g, furniture.Description)
		}

		// If furniture contained an item, give it to the player and show callout
		if item != nil {
			if item.Name == "Battery" {
				g.AddBatteries(1)
				// Show furniture name in furniture color, then "Found: Battery!" in item color
				calloutText := fmt.Sprintf("FURNITURE{%s}\nFound: ACTION{Battery}!", furniture.Name)
				renderer.AddCallout(cell.Row, cell.Col, calloutText, renderer.CalloutColorFurnitureChecked, 0)
			} else {
				g.OwnedItems.Put(item)
				// Show furniture name in furniture color, then "Found: [Item]!" in item color
				calloutText := fmt.Sprintf("FURNITURE{%s}\nFound: ITEM{%s}!", furniture.Name, item.Name)
				renderer.AddCallout(cell.Row, cell.Col, calloutText, renderer.CalloutColorFurnitureChecked, 0)
			}
		} else {
			// Furniture already checked or decorative-only: show description
			// Show furniture name on first line, description on second line
			calloutText := fmt.Sprintf("FURNITURE{%s}\n%s", furniture.Name, furniture.Description)
			renderer.AddCallout(cell.Row, cell.Col, calloutText, renderer.CalloutColorFurnitureChecked, 0)
		}
		return true
	}
	return false
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

// checkAdjacentHazardControlsAtCell checks a specific cell for hazard controls and interacts with it
// Returns true if a hazard control was interacted with
func checkAdjacentHazardControlsAtCell(g *state.Game, cell *world.Cell) bool {
	if cell == nil || !gameworld.HasInactiveHazardControl(cell) {
		return false
	}

	control := gameworld.GetGameData(cell).HazardControl
	control.Activate()

	info := entities.HazardTypes[control.Type]
	logMessage(g, "Activated %s: %s", renderer.StyledHazardCtrl(control.Name), info.FixedMessage)
	renderer.AddCallout(cell.Row, cell.Col, fmt.Sprintf("%s activated!", control.Name), renderer.CalloutColorHazardCtrl, 0)
	return true
}

// checkAdjacentHazardControls checks adjacent cells for inactive hazard controls and activates them
// DEPRECATED: Use checkAdjacentInteractables instead for priority-based interaction
// Returns true if a hazard control was interacted with
func checkAdjacentHazardControls(g *state.Game) bool {
	neighbors := []*world.Cell{
		g.CurrentCell.North,
		g.CurrentCell.East,
		g.CurrentCell.South,
		g.CurrentCell.West,
	}

	for _, cell := range neighbors {
		if checkAdjacentHazardControlsAtCell(g, cell) {
			return true
		}
	}
	return false
}

// checkAdjacentMaintenanceTerminalAtCell checks a specific cell for maintenance terminal and opens menu
// Returns true if maintenance terminal was interacted with
func checkAdjacentMaintenanceTerminalAtCell(g *state.Game, cell *world.Cell) bool {
	if cell == nil || !gameworld.HasMaintenanceTerminal(cell) {
		return false
	}

	maintenanceTerm := gameworld.GetGameData(cell).MaintenanceTerm
	// Don't mark as used - allow multiple interactions
	// maintenanceTerm.Activate()

	// Open maintenance terminal menu
	runMaintenanceMenu(g, cell, maintenanceTerm)
	return true
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

// updateLightingExploration updates cell exploration based on lighting
func updateLightingExploration(g *state.Game) {
	if g.Grid == nil || g.CurrentCell == nil {
		return
	}

	// Calculate total power consumption
	totalConsumption := calculatePowerConsumption(g)
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

// calculatePowerConsumption calculates total power consumption from all active devices
func calculatePowerConsumption(g *state.Game) int {
	if g.Grid == nil {
		return 0
	}

	totalConsumption := 0

	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}

		data := gameworld.GetGameData(cell)

		// Lights consume power if on
		if data.LightsOn {
			totalConsumption += 1 // 1 watt per lit cell
		}

		// CCTV terminals consume power if used
		if data.Terminal != nil && data.Terminal.Used {
			totalConsumption += 5 // 5 watts per active terminal
		}

		// Puzzle terminals consume power if solved
		if data.Puzzle != nil && data.Puzzle.IsSolved() {
			totalConsumption += 3 // 3 watts per solved puzzle
		}

		// Maintenance terminals don't consume power - they're just information displays
	})

	return totalConsumption
}

// runMaintenanceMenu shows the maintenance terminal menu with room devices and power consumption using the generic menu system.
func runMaintenanceMenu(g *state.Game, cell *world.Cell, maintenanceTerm *entities.MaintenanceTerminal) {
	handler := gamemenu.NewMaintenanceMenuHandler(g, cell, maintenanceTerm)
	items := handler.GetMenuItems()
	gamemenu.RunMenu(g, items, handler)
}

// saveScreenshotHTML saves the current map view as an HTML file
func saveScreenshotHTML(g *state.Game) string {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("screenshot-%s.html", timestamp)

	viewportRows, viewportCols := renderer.GetViewportSize()

	// Calculate viewport bounds centered on player
	playerRow := g.CurrentCell.Row
	playerCol := g.CurrentCell.Col
	startRow := playerRow - viewportRows/2
	startCol := playerCol - viewportCols/2

	// Build the HTML
	var html strings.Builder

	html.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>The Dark Station - Screenshot</title>
    <style>
        body {
            background-color: #1a1a2e;
            color: #eee;
            font-family: 'Courier New', monospace;
            padding: 20px;
        }
        .header {
            color: #bb86fc;
            font-size: 18px;
            margin-bottom: 10px;
        }
        .room-name {
            color: #888;
            margin-bottom: 20px;
        }
        .map-container {
            background-color: #0f0f1a;
            padding: 20px;
            border-radius: 8px;
            display: inline-block;
            margin: 20px 0;
        }
        .map-row {
            white-space: pre;
            line-height: 1.2;
            font-size: 16px;
        }
        .player { color: #00ff00; font-weight: bold; }
        .wall { color: #666; }
        .floor { color: #888; }
        .floor-visited { color: #aaa; }
        .door-locked { color: #ffff00; font-weight: bold; }
        .door-unlocked { color: #00aa00; }
        .keycard { color: #4444ff; }
        .item { color: #bb86fc; }
        .battery { color: #bb86fc; font-weight: bold; }
        .hazard { color: #ff4444; }
        .hazard-ctrl { color: #00ffff; }
        .generator-off { color: #ff4444; font-weight: bold; }
        .generator-on { color: #00aa00; }
        .terminal { color: #4444ff; }
        .terminal-used { color: #666; }
        .furniture { color: #ff66ff; font-weight: bold; }
        .furniture-checked { color: #aaaa00; }
        .exit-locked { color: #ff4444; font-weight: bold; }
        .exit-unlocked { color: #00aa00; }
        .void { color: #1a1a2e; }
        .inventory {
            margin-top: 20px;
            color: #888;
        }
        .inventory-item { color: #bb86fc; }
        .messages {
            margin-top: 20px;
            border-top: 1px solid #333;
            padding-top: 10px;
        }
        .message { color: #ccc; margin: 5px 0; }
    </style>
</head>
<body>
`)

	// Header
	html.WriteString(fmt.Sprintf(`    <div class="header">Deck %d</div>`+"\n", g.Level))
	html.WriteString(fmt.Sprintf(`    <div class="room-name">In: %s</div>`+"\n", g.CurrentCell.Name))

	// Map container
	html.WriteString(`    <div class="map-container">` + "\n")

	// Render the viewport
	for vRow := 0; vRow < viewportRows; vRow++ {
		mapRow := startRow + vRow
		html.WriteString(`        <div class="map-row">`)

		for vCol := 0; vCol < viewportCols; vCol++ {
			mapCol := startCol + vCol
			cell := g.Grid.GetCell(mapRow, mapCol)
			icon, class := getCellHTMLInfo(g, cell)
			html.WriteString(fmt.Sprintf(`<span class="%s">%s</span>`, class, icon))
		}

		html.WriteString("</div>\n")
	}

	html.WriteString(`    </div>` + "\n")

	// Inventory
	html.WriteString(`    <div class="inventory">Inventory: `)
	if g.OwnedItems.Size() == 0 && g.Batteries == 0 {
		html.WriteString(`<span style="color:#666">(empty)</span>`)
	} else {
		first := true
		g.OwnedItems.Each(func(item *world.Item) {
			if !first {
				html.WriteString(", ")
			}
			html.WriteString(fmt.Sprintf(`<span class="inventory-item">%s</span>`, item.Name))
			first = false
		})
		if g.Batteries > 0 {
			if !first {
				html.WriteString(", ")
			}
			html.WriteString(fmt.Sprintf(`<span class="battery">Batteries x%d</span>`, g.Batteries))
		}
	}
	html.WriteString(`</div>` + "\n")

	// Generator status
	if len(g.Generators) > 0 {
		html.WriteString(`    <div class="inventory">Generators: `)
		for i, gen := range g.Generators {
			if i > 0 {
				html.WriteString(", ")
			}
			if gen.IsPowered() {
				html.WriteString(fmt.Sprintf(`<span class="generator-on">#%d POWERED</span>`, i+1))
			} else {
				html.WriteString(fmt.Sprintf(`<span class="generator-off">#%d %d/%d</span>`, i+1, gen.BatteriesInserted, gen.BatteriesRequired))
			}
		}
		html.WriteString(`</div>` + "\n")
	}

	// Messages
	if len(g.Messages) > 0 {
		html.WriteString(`    <div class="messages">` + "\n")
		for _, msg := range g.Messages {
			// Strip ANSI codes for HTML output
			cleanMsg := stripANSI(msg.Text)
			html.WriteString(fmt.Sprintf(`        <div class="message">%s</div>`+"\n", cleanMsg))
		}
		html.WriteString(`    </div>` + "\n")
	}

	html.WriteString(`</body>
</html>
`)

	// Write to file
	os.WriteFile(filename, []byte(html.String()), 0644)
	return filename
}

// getCellHTMLInfo returns the icon and CSS class for a cell
func getCellHTMLInfo(g *state.Game, r *world.Cell) (string, string) {
	if r == nil {
		return " ", "void"
	}

	// Player position
	if g.CurrentCell == r {
		return "@", "player"
	}

	// Get game-specific data for this cell
	data := gameworld.GetGameData(r)

	// Hazard (show if has map or discovered)
	if gameworld.HasHazard(r) && (g.HasMap || r.Discovered) {
		if data.Hazard.IsBlocking() {
			return data.Hazard.GetIcon(), "hazard"
		}
	}

	// Hazard Control (show if has map or discovered)
	if gameworld.HasHazardControl(r) && (g.HasMap || r.Discovered) {
		if !data.HazardControl.Activated {
			return entities.GetControlIcon(data.HazardControl.Type), "hazard-ctrl"
		}
		return entities.GetControlIcon(data.HazardControl.Type), "terminal-used"
	}

	// Door (show if has map or discovered)
	if gameworld.HasDoor(r) && (g.HasMap || r.Discovered) {
		if data.Door.Locked {
			return "▣", "door-locked"
		}
		return "□", "door-unlocked"
	}

	// Generator (show if has map or discovered)
	if gameworld.HasGenerator(r) && (g.HasMap || r.Discovered) {
		if data.Generator.IsPowered() {
			return "◆", "generator-on"
		}
		return "◇", "generator-off"
	}

	// CCTV Terminal (show if has map or discovered)
	if gameworld.HasTerminal(r) && (g.HasMap || r.Discovered) {
		if data.Terminal.IsUsed() {
			return "▪", "terminal-used"
		}
		return "▫", "terminal"
	}

	// Furniture (show if has map or discovered)
	if gameworld.HasFurniture(r) && (g.HasMap || r.Discovered) {
		if data.Furniture.IsChecked() {
			return data.Furniture.Icon, "furniture-checked"
		}
		return data.Furniture.Icon, "furniture"
	}

	// Exit cell (show if has map or discovered)
	if r.ExitCell && (g.HasMap || r.Discovered) {
		if r.Locked && !g.AllGeneratorsPowered() {
			return "▲", "exit-locked"
		}
		return "△", "exit-unlocked"
	}

	// Items on floor (show if has map or discovered)
	if r.ItemsOnFloor.Size() > 0 && (g.HasMap || r.Discovered) {
		if cellHasKeycard(r) {
			return "⚷", "keycard"
		}
		if cellHasBattery(r) {
			return "■", "battery"
		}
		return "?", "item"
	}

	// Visited rooms
	if r.Visited {
		return getFloorIconHTML(r.Name, true), "floor-visited"
	}

	// Discovered but not visited
	if r.Discovered {
		if r.Room {
			return getFloorIconHTML(r.Name, false), "floor"
		}
		return "▒", "wall"
	}

	// Has map - show rooms faintly
	if g.HasMap && r.Room {
		return getFloorIconHTML(r.Name, false), "floor"
	}

	// Non-room cells adjacent to discovered/visited rooms render as walls
	if !r.Room && hasAdjacentDiscoveredRoomHTML(r) {
		return "▒", "wall"
	}

	// Unknown/void
	return " ", "void"
}

// getFloorIconHTML returns floor icons for HTML output
func getFloorIconHTML(roomName string, visited bool) string {
	roomFloorIcons := map[string][2]string{
		"Bridge":          {"◎", "◉"},
		"Command Center":  {"◎", "◉"},
		"Communications":  {"◎", "◉"},
		"Security":        {"◎", "◉"},
		"Engineering":     {"▫", "▪"},
		"Reactor Core":    {"▫", "▪"},
		"Server Room":     {"▫", "▪"},
		"Maintenance Bay": {"▫", "▪"},
		"Life Support":    {"▫", "▪"},
		"Cargo Bay":       {"□", "▣"},
		"Storage":         {"□", "▣"},
		"Hangar":          {"□", "▣"},
		"Armory":          {"□", "▣"},
		"Med Bay":         {"◇", "◆"},
		"Lab":             {"◇", "◆"},
		"Hydroponics":     {"◇", "◆"},
		"Observatory":     {"◇", "◆"},
		"Crew Quarters":   {"·", "•"},
		"Mess Hall":       {"·", "•"},
		"Airlock":         {"╳", "╳"},
		"Corridor":        {"░", "░"},
	}

	for baseRoom, icons := range roomFloorIcons {
		if containsSubstring(roomName, baseRoom) {
			if visited {
				return icons[0]
			}
			return icons[1]
		}
	}
	if visited {
		return "○"
	}
	return "●"
}

// hasAdjacentDiscoveredRoomHTML checks if any adjacent cell is a discovered or visited room
func hasAdjacentDiscoveredRoomHTML(c *world.Cell) bool {
	neighbors := []*world.Cell{c.North, c.East, c.South, c.West}
	for _, n := range neighbors {
		if n != nil && n.Room && (n.Discovered || n.Visited) {
			return true
		}
	}
	return false
}

// cellHasKeycard checks if a cell has a keycard item on the floor
func cellHasKeycard(c *world.Cell) bool {
	hasKeycard := false
	c.ItemsOnFloor.Each(func(item *world.Item) {
		if containsSubstring(item.Name, "Keycard") || containsSubstring(item.Name, "keycard") {
			hasKeycard = true
		}
	})
	return hasKeycard
}

// switchToDevMap switches the game to a hard-coded 50x50 developer testing map
// All possible game cells are placed with a 3-cell margin between each, grouped by type in rows
func switchToDevMap(g *state.Game) {
	// Create a 50x50 grid
	grid := world.NewGrid(50, 50)

	// Build cell connections for navigation
	grid.BuildAllCellConnections()

	// Initialize all cells as floor cells
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		cell.Room = true
		cell.Name = "Dev Test Floor"
		cell.Discovered = true
		cell.Visited = true
	})

	// Define spacing: 3-cell margin between items, items placed in rows
	const margin = 3
	currentRow := 2
	currentCol := 2

	// Row 1: Doors (locked and unlocked)
	doorRow := currentRow
	doorCol := currentCol
	doorNames := []string{"Test Room A", "Test Room B", "Test Room C"}
	for i, roomName := range doorNames {
		cell := grid.GetCell(doorRow, doorCol+i*(margin+1))
		if cell != nil {
			data := gameworld.InitGameData(cell)
			door := entities.NewDoor(roomName)
			if i == 0 {
				door.Unlock() // First door unlocked
			}
			data.Door = door
		}
	}
	currentRow += margin + 1

	// Row 2: Generators (unpowered and powered)
	genRow := currentRow
	genCol := currentCol
	for i := 0; i < 3; i++ {
		cell := grid.GetCell(genRow, genCol+i*(margin+1))
		if cell != nil {
			data := gameworld.InitGameData(cell)
			gen := entities.NewGenerator(fmt.Sprintf("Generator %d", i+1), 2)
			if i == 2 {
				// Third generator is powered
				gen.BatteriesInserted = 2
			}
			data.Generator = gen
			g.AddGenerator(gen)
		}
	}
	currentRow += margin + 1

	// Row 3: CCTV Terminals (unused and used)
	termRow := currentRow
	termCol := currentCol
	for i := 0; i < 2; i++ {
		cell := grid.GetCell(termRow, termCol+i*(margin+1))
		if cell != nil {
			data := gameworld.InitGameData(cell)
			terminal := entities.NewCCTVTerminal(fmt.Sprintf("CCTV Terminal %d", i+1))
			if i == 1 {
				terminal.Activate() // Second terminal is used
			}
			terminal.TargetRoom = fmt.Sprintf("Target Room %d", i+1)
			data.Terminal = terminal
		}
	}
	currentRow += margin + 1

	// Row 4: Puzzle Terminals
	puzzleRow := currentRow
	puzzleCol := currentCol
	for i := 0; i < 2; i++ {
		cell := grid.GetCell(puzzleRow, puzzleCol+i*(margin+1))
		if cell != nil {
			data := gameworld.InitGameData(cell)
			puzzleType := entities.PuzzleSequence
			if i == 1 {
				puzzleType = entities.PuzzlePattern
			}
			puzzle := entities.NewPuzzleTerminal(
				fmt.Sprintf("Puzzle Terminal %d", i+1),
				puzzleType,
				fmt.Sprintf("SOL-%d", i+1),
				fmt.Sprintf("Hint for puzzle %d", i+1),
				entities.RewardBattery,
				fmt.Sprintf("Test puzzle %d", i+1),
			)
			data.Puzzle = puzzle
		}
	}
	currentRow += margin + 1

	// Row 5: Furniture
	furnRow := currentRow
	furnCol := currentCol
	furnitureTypes := []struct {
		name, desc, icon string
		hasItem          bool
		itemName         string
	}{
		{"Desk", "A standard desk", "D", false, ""},
		{"Cabinet", "Storage cabinet", "C", true, "Battery"},
		{"Locker", "Personal locker", "L", true, "Test Keycard"},
		{"Table", "Work table", "T", false, ""},
	}
	for i, furn := range furnitureTypes {
		cell := grid.GetCell(furnRow, furnCol+i*(margin+1))
		if cell != nil {
			data := gameworld.InitGameData(cell)
			furniture := entities.NewFurniture(furn.name, furn.desc, furn.icon)
			if furn.hasItem {
				furniture.ContainedItem = world.NewItem(furn.itemName)
			}
			data.Furniture = furniture
		}
	}
	currentRow += margin + 1

	// Row 6: Hazards (all types)
	hazardRow := currentRow
	hazardCol := currentCol
	hazardTypes := []entities.HazardType{
		entities.HazardVacuum,
		entities.HazardCoolant,
		entities.HazardElectrical,
		entities.HazardGas,
		entities.HazardRadiation,
	}
	for i, hazType := range hazardTypes {
		cell := grid.GetCell(hazardRow, hazardCol+i*(margin+1))
		if cell != nil {
			data := gameworld.InitGameData(cell)
			hazard := entities.NewHazard(hazType)
			data.Hazard = hazard
		}
	}
	currentRow += margin + 1

	// Row 7: Hazard Controls
	controlRow := currentRow
	controlCol := currentCol
	for i, hazType := range hazardTypes {
		cell := grid.GetCell(controlRow, controlCol+i*(margin+1))
		if cell != nil {
			// Create hazard first
			hazardCell := grid.GetCell(hazardRow, hazardCol+i*(margin+1))
			var hazard *entities.Hazard
			if hazardCell != nil {
				hazardData := gameworld.GetGameData(hazardCell)
				hazard = hazardData.Hazard
			}
			if hazard == nil {
				hazard = entities.NewHazard(hazType)
			}

			data := gameworld.InitGameData(cell)
			control := entities.NewHazardControl(hazType, hazard)
			if i == 0 {
				control.Activate() // First control is activated
			}
			data.HazardControl = control
		}
	}
	currentRow += margin + 1

	// Row 8: Items on floor
	itemRow := currentRow
	itemCol := currentCol
	items := []string{"Battery", "Test Keycard", "Patch Kit", "Map"}
	for i, itemName := range items {
		cell := grid.GetCell(itemRow, itemCol+i*(margin+1))
		if cell != nil {
			item := world.NewItem(itemName)
			cell.ItemsOnFloor.Put(item)
		}
	}
	currentRow += margin + 1

	// Row 9: Exit (unlocked, all generators powered, all hazards cleared)
	exitRow := currentRow
	exitCol := currentCol
	exitCell := grid.GetCell(exitRow, exitCol)
	if exitCell != nil {
		exitCell.ExitCell = true
		exitCell.Locked = false
	}

	// Set player start position (top-left, away from entities)
	startCell := grid.GetCell(1, 1)
	if startCell != nil {
		startCell.Room = true
		startCell.Name = "Dev Test Floor"
		startCell.Discovered = true
		startCell.Visited = true
		grid.SetStartCell(startCell)
		g.CurrentCell = startCell
		world.RevealFOVDefault(grid, startCell)
	}

	// Set exit cell
	if exitCell != nil {
		exitCell.Room = true
		exitCell.Name = "Dev Test Floor"
		exitCell.Discovered = true
		grid.SetExitCell(exitCell)
	}

	// Update game state
	g.Grid = grid
	g.Level = 999 // Mark as dev map
	g.ClearMessages()
	logMessage(g, "Switched to developer testing map!")
	logMessage(g, "All entity types are placed in rows with 3-cell margins.")
}

// cellHasBattery checks if a cell has a battery item on the floor
func cellHasBattery(c *world.Cell) bool {
	hasBattery := false
	c.ItemsOnFloor.Each(func(item *world.Item) {
		if containsSubstring(item.Name, "Battery") || containsSubstring(item.Name, "battery") {
			hasBattery = true
		}
	})
	return hasBattery
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
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
