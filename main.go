package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/zyedidia/generic/mapset"
	gettext "github.com/gosexy/gettext"

	"darkcastle/pkg/engine/input"
	"darkcastle/pkg/engine/world"
	"darkcastle/pkg/game/generator"
	"darkcastle/pkg/game/renderer"
	"darkcastle/pkg/game/state"
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

// placeItem places an item in a random reachable room at an appropriate distance based on level
func placeItem(g *state.Game, start *world.Cell, item *world.Item, avoid *mapset.Set[*world.Cell]) *world.Cell {
	rooms := collectReachableRooms(start, avoid)

	if len(rooms) == 0 {
		// Fallback: place on start cell
		start.ItemsOnFloor.Put(item)
		g.AddHint("The " + renderer.ColorDenied.Sprintf(item.Name) + " is in " + renderer.ColorCell.Sprintf(start.Name))
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
		// Sort by distance and take the furthest half
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
	room := farRooms[rand.Intn(len(farRooms))]
	room.ItemsOnFloor.Put(item)

	g.AddHint("The " + renderer.ColorDenied.Sprintf(item.Name) + " is in " + renderer.ColorCell.Sprintf(room.Name))

	return room
}

// buildGame creates a new game instance
func buildGame() *state.Game {
	g := state.NewGame()

	g.Grid = generateGrid(g.Level)
	setupLevel(g)

	// Clear the initial "entered room" message
	g.ClearMessages()
	logMessage(g, "Welcome to The Dark Castle!")
	logMessage(g, "You are on level %d.", g.Level)
	if g.Grid.ExitCell().Locked {
		logMessage(g, "Find the Red Key to unlock the exit.")
	} else {
		logMessage(g, "Find the exit to proceed.")
	}

	return g
}

// setupLevel configures the current level with items and keys
func setupLevel(g *state.Game) {
	// Cells to avoid placing items on
	avoid := mapset.New[*world.Cell]()
	avoid.Put(g.Grid.StartCell())
	avoid.Put(g.Grid.ExitCell())

	// Levels 1-2: Exit is unlocked (tutorial levels)
	// Levels 3+: Exit is locked and requires a key
	if g.Level >= 3 {
		exitKey := world.NewItem("Red Key")
		g.Grid.ExitCell().RequiredItems.Put(exitKey)
		g.Grid.ExitCell().Locked = true
		keyRoom := placeItem(g, g.Grid.StartCell(), exitKey, &avoid)
		avoid.Put(keyRoom) // Don't place map in same room as key
	} else {
		g.Grid.ExitCell().Locked = false
	}

	// Always place a map
	placeItem(g, g.Grid.StartCell(), world.NewItem("Map"), &avoid)

	g.CurrentCell = g.Grid.GetCenterCell()

	moveCell(g, g.Grid.StartCell())
}

// advanceLevel generates a new map and advances to the next level
func advanceLevel(g *state.Game) {
	g.AdvanceLevel()
	g.Grid = generateGrid(g.Level)

	setupLevel(g)

	// Clear movement messages and show level info
	g.ClearMessages()
	logMessage(g, "You descended to level %d!", g.Level)
	if g.Grid.ExitCell().Locked {
		logMessage(g, "Find the Red Key to unlock the exit.")
	} else {
		logMessage(g, "Find the exit to proceed.")
	}
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

	r.RequiredItems.Each(func(reqItem *world.Item) {
		if !g.OwnedItems.Has(reqItem) {
			missingItems.Put(reqItem)
		}
	})

	if missingItems.Size() > 0 && logReason {
		missingItems.Each(func(i *world.Item) {
			logMessage(g, "To enter, you need: %v", renderer.ColorDenied.Sprintf(i.Name))
		})
	}

	return missingItems.Size() == 0, &missingItems
}

// moveCell moves the player to a new cell
func moveCell(g *state.Game, requestedCell *world.Cell) {
	if res, _ := canEnter(g, requestedCell, true); res {
		logMessage(g, gettext.Gettext("OPEN_DOOR")+"%v", renderer.ColorCell.Sprintf(gettext.Gettext(requestedCell.Description)))

		requestedCell.Visited = true

		for _, dir := range world.AllDirections() {
			adj := g.Grid.GetCellRelative(requestedCell, dir)
			if adj != nil {
				adj.Discovered = true
			}
		}

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
	initGettext()
	renderer.InitColors()

	rand.Seed(time.Now().UnixNano())

	g := buildGame()

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
	renderer.ColorAction.Printf("Level %d\n\n", g.Level)

	renderer.PrintString("GT{IN_ROOM} ROOM{%v} (ROOM{%v})\n\n", g.CurrentCell.Description, g.CurrentCell.Name)

	g.CurrentCell.ItemsOnFloor.Each(func(item *world.Item) {
		g.OwnedItems.Put(item)
		g.CurrentCell.ItemsOnFloor.Remove(item)

		if item.Name == "Map" {
			g.HasMap = true
		}

		logMessage(g, "Picked up: ITEM{%v}", item.Name)
	})

	renderer.PrintMap(g)

	renderer.PrintStatusBar(g)

	renderer.PrintPossibleActions()

	renderer.PrintMessagesPane(g)

	fmt.Printf("\n> ")

	processInput(g, input.GetInputWithArrows())
}
