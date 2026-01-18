package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/zyedidia/generic/mapset"
	gettext "github.com/gosexy/gettext"

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

// roomBoundary represents a corridor cell that connects to a room
type roomBoundary struct {
	corridorCell *world.Cell
	roomName     string
}

// findRoomBoundaries finds corridor cells that are adjacent to room cells
// These are ideal locations for doors (at the entrance to rooms)
func findRoomBoundaries(grid *world.Grid) []roomBoundary {
	var boundaries []roomBoundary
	seenPairs := mapset.New[string]() // Track corridor-room pairs we've already added

	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		// Only look at corridor cells
		if !cell.Room || cell.Name != "Corridor" {
			return
		}

		// Check adjacent cells for rooms (not corridors)
		neighbors := []*world.Cell{cell.North, cell.East, cell.South, cell.West}
		for _, neighbor := range neighbors {
			if neighbor != nil && neighbor.Room && neighbor.Name != "Corridor" && neighbor.Name != "" {
				// Create a unique key for this corridor-room pair
				pairKey := fmt.Sprintf("%d,%d-%s", cell.Row, cell.Col, neighbor.Name)
				if !seenPairs.Has(pairKey) {
					seenPairs.Put(pairKey)
					boundaries = append(boundaries, roomBoundary{
						corridorCell: cell,
						roomName:     neighbor.Name,
					})
				}
			}
		}
	})

	return boundaries
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

	// Determine number of doors based on level
	// More doors as levels progress - most rooms should have doors
	// Level 1: 1 door
	// Level 2: 2 doors
	// Level 3+: 2 + (level-2) doors, scaling with available rooms
	var numDoors int
	if g.Level == 1 {
		numDoors = 1
	} else if g.Level == 2 {
		numDoors = 2
	} else {
		numDoors = 2 + (g.Level - 2)
	}
	// Cap at reasonable maximum
	if numDoors > 10 {
		numDoors = 10
	}

	// Find room-corridor boundary cells (corridor cells that connect to rooms)
	// These are ideal locations for doors
	roomBoundaries := findRoomBoundaries(g.Grid)

	// Shuffle for random selection
	rand.Shuffle(len(roomBoundaries), func(i, j int) {
		roomBoundaries[i], roomBoundaries[j] = roomBoundaries[j], roomBoundaries[i]
	})

	// Track which rooms already have doors
	roomsWithDoors := mapset.New[string]()

	// Place doors at room-corridor boundaries, blocking the path to the exit
	// Each door is named after the room it guards
	for i := 0; i < numDoors; i++ {
		// Calculate currently reachable area (with all previously placed doors)
		currentlyReachable := getReachableCells(g.Grid, g.Grid.StartCell(), &lockedDoorCells)

		// Find a boundary cell that blocks the exit when locked
		var doorCell *world.Cell
		var targetRoomName string

		for _, boundary := range roomBoundaries {
			if avoid.Has(boundary.corridorCell) || lockedDoorCells.Has(boundary.corridorCell) {
				continue
			}
			// Don't put multiple doors for the same room
			if roomsWithDoors.Has(boundary.roomName) {
				continue
			}
			// Must be reachable from spawn
			if !currentlyReachable.Has(boundary.corridorCell) {
				continue
			}
			// Test if locking this cell blocks the exit
			testLocked := mapset.New[*world.Cell]()
			lockedDoorCells.Each(func(c *world.Cell) { testLocked.Put(c) })
			testLocked.Put(boundary.corridorCell)
			reachableWithDoor := getReachableCells(g.Grid, g.Grid.StartCell(), &testLocked)

			// Prefer cells that block the exit, but accept any that reduce reachability
			if !reachableWithDoor.Has(g.Grid.ExitCell()) || reachableWithDoor.Size() < currentlyReachable.Size() {
				doorCell = boundary.corridorCell
				targetRoomName = boundary.roomName
				break
			}
		}

		if doorCell == nil || targetRoomName == "" {
			continue // No valid door location
		}

		// Calculate what's reachable with this door in place
		testLocked := mapset.New[*world.Cell]()
		lockedDoorCells.Each(func(c *world.Cell) { testLocked.Put(c) })
		testLocked.Put(doorCell)
		reachableBeforeDoor := getReachableCells(g.Grid, g.Grid.StartCell(), &testLocked)

		// Place the keycard in the area reachable BEFORE this door
		keycardRoom := findRoomInReachable(reachableBeforeDoor, &avoid)
		if keycardRoom == nil {
			continue // No valid keycard location
		}

		// Create the door and keycard
		door := world.NewDoor(targetRoomName)
		keycardName := door.KeycardName()

		keycard := world.NewItem(keycardName)
		keycardRoom.ItemsOnFloor.Put(keycard)
		avoid.Put(keycardRoom)
		g.AddHint("The " + renderer.ColorKeycard.Sprintf(keycardName) + " is in " + renderer.ColorCell.Sprintf(keycardRoom.Name))

		// Place the door
		doorCell.Door = door
		avoid.Put(doorCell)
		lockedDoorCells.Put(doorCell)
		roomsWithDoors.Put(targetRoomName)
		g.AddHint("The " + renderer.ColorDoor.Sprintf(door.DoorName()) + " blocks access to " + renderer.ColorCell.Sprintf(targetRoomName))
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

		// Place furniture
		for i := 0; i < numFurniture && i < len(validCells); i++ {
			template := templates[i]
			furniture := world.NewFurniture(template.Name, template.Description, template.Icon)
			furniture.Cell = validCells[i]
			validCells[i].Furniture = furniture
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
	if numDoors == 0 && len(g.Generators) == 0 {
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
			// Unlock the door and consume the keycard
			r.Door.Unlock()
			g.OwnedItems.Remove(keycardItem)
			logMessage(g, "Used %s to unlock the %s!", renderer.ColorKeycard.Sprintf(keycardName), renderer.ColorDoor.Sprintf(r.Door.DoorName()))
		} else {
			if logReason {
				logMessage(g, "This door requires a %s", renderer.ColorKeycard.Sprintf(keycardName))
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

// checkAdjacentFurniture checks adjacent cells for furniture and displays hints
func checkAdjacentFurniture(g *state.Game) {
	neighbors := []*world.Cell{
		g.CurrentCell.North,
		g.CurrentCell.East,
		g.CurrentCell.South,
		g.CurrentCell.West,
	}

	for _, cell := range neighbors {
		if cell == nil || !cell.HasFurniture() {
			continue
		}

		furniture := cell.Furniture

		// Only show hint if we haven't shown it recently (check if cell was just entered)
		// We track this by checking if the furniture description is already in recent messages
		alreadyShown := false
		for _, msg := range g.Messages {
			if containsSubstring(msg, furniture.Name) {
				alreadyShown = true
				break
			}
		}

		if !alreadyShown {
			logMessage(g, "%s: %s", renderer.ColorFurniture.Sprintf(furniture.Name), furniture.Description)
		}
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
