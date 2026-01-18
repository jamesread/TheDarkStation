package generator

import (
	"fmt"
	"math/rand"

	"darkstation/pkg/engine/world"
)

// BSPGenerator generates maps using Binary Space Partitioning
type BSPGenerator struct{}

// Name returns the name of this generator
func (g *BSPGenerator) Name() string {
	return "BSP Tree"
}

// bspNode represents a node in the BSP tree
type bspNode struct {
	x, y, width, height int
	left, right         *bspNode
	room                *bspRoom
}

// bspRoom represents a room within a BSP leaf node
type bspRoom struct {
	x, y, width, height int
	name, description   string
}

// Room name templates - Space Station theme
var roomNames = []string{
	"Bridge", "Cargo Bay", "Engineering", "Med Bay", "Crew Quarters",
	"Airlock", "Server Room", "Reactor Core", "Armory", "Lab",
	"Hangar", "Command Center", "Life Support", "Mess Hall", "Storage",
	"Observatory", "Communications", "Maintenance Bay", "Hydroponics", "Security",
}

var roomAdjectives = []string{
	"Abandoned", "Damaged", "Dark", "Derelict", "Emergency",
	"Flickering", "Isolated", "Sealed", "Depressurized", "Overgrown",
}

// Global room counter for unique names
var roomCounter int

// Constants for BSP generation
const (
	minNodeSize   = 8 // Minimum size of a BSP node
	minRoomSize   = 4 // Minimum size of a room
	roomPadding   = 2 // Padding between room and node edge
	corridorWidth = 3 // Width of corridors (wall + path + wall)
)

// Generate creates a new grid using BSP algorithm
func (g *BSPGenerator) Generate(level int) *world.Grid {
	grid := &world.Grid{}

	// Reset room counter for this level
	roomCounter = 0

	// Start small and scale grid size with level (add 2 for perimeter)
	// Level 1: 14x28, Level 5: 30x58, Level 10: 50x88
	baseRows := 12 + 2
	baseCols := 24 + 2
	rows := baseRows + (level * 4)
	cols := baseCols + (level * 6)

	// Cap maximum size
	if rows > 60 {
		rows = 60
	}
	if cols > 100 {
		cols = 100
	}

	grid.Build(rows, cols)

	// Create BSP tree (leaving 1 cell border for perimeter walls)
	root := &bspNode{
		x:      1,
		y:      1,
		width:  cols - 2,
		height: rows - 2,
	}

	// Split the BSP tree
	// More splits at higher levels for more rooms
	minSize := minNodeSize - (level / 3)
	if minSize < 6 {
		minSize = 6
	}
	splitBSP(root, minSize)

	// Create rooms in leaf nodes
	createRooms(root)

	// Carve rooms into the grid
	carveRooms(grid, root)

	// Connect rooms with corridors
	connectRooms(grid, root)

	// Build cell connections first so we can calculate path distances
	grid.BuildAllCellConnections()

	// Set start and exit cells using actual path distance
	rooms := collectRooms(root)
	if len(rooms) >= 1 {
		// Start in a random room
		startRoom := rooms[rand.Intn(len(rooms))]
		startRow := startRoom.y + startRoom.height/2
		startCol := startRoom.x + startRoom.width/2
		grid.SetStartCellAt(startRow, startCol)

		// Find the cell with the longest path distance from start using BFS
		startCell := grid.StartCell()
		exitCell := findFurthestCell(grid, startCell)
		if exitCell != nil {
			grid.SetExitCellAt(exitCell.Row, exitCell.Col)
		} else {
			// Fallback: exit in opposite corner of start room
			grid.SetExitCellAt(startRoom.y+startRoom.height-2, startRoom.x+startRoom.width-2)
		}
	} else {
		// Fallback to center
		centerRow, centerCol := grid.CenterPosition()
		grid.MarkAsRoom(centerRow, centerCol)
		grid.SetStartCellAt(centerRow, centerCol)
		grid.SetExitCellAt(centerRow, centerCol)
	}

	// Rebuild connections after setting start/exit
	grid.BuildAllCellConnections()

	// Validate the generated grid
	if err := grid.Validate(); err != "" {
		panic("Generated invalid grid: " + err)
	}

	return grid
}

// splitBSP recursively splits a BSP node
func splitBSP(node *bspNode, minSize int) {
	if node.width < minSize*2 && node.height < minSize*2 {
		return // Too small to split
	}

	// Decide split direction
	var splitHorizontal bool
	if node.width > node.height && node.width >= minSize*2 {
		splitHorizontal = false // Split vertically
	} else if node.height > node.width && node.height >= minSize*2 {
		splitHorizontal = true // Split horizontally
	} else if node.width >= minSize*2 && node.height >= minSize*2 {
		splitHorizontal = rand.Intn(2) == 0
	} else if node.width >= minSize*2 {
		splitHorizontal = false
	} else if node.height >= minSize*2 {
		splitHorizontal = true
	} else {
		return // Can't split
	}

	if splitHorizontal {
		// Split horizontally (top and bottom)
		splitPoint := minSize + rand.Intn(node.height-minSize*2+1)
		node.left = &bspNode{
			x:      node.x,
			y:      node.y,
			width:  node.width,
			height: splitPoint,
		}
		node.right = &bspNode{
			x:      node.x,
			y:      node.y + splitPoint,
			width:  node.width,
			height: node.height - splitPoint,
		}
	} else {
		// Split vertically (left and right)
		splitPoint := minSize + rand.Intn(node.width-minSize*2+1)
		node.left = &bspNode{
			x:      node.x,
			y:      node.y,
			width:  splitPoint,
			height: node.height,
		}
		node.right = &bspNode{
			x:      node.x + splitPoint,
			y:      node.y,
			width:  node.width - splitPoint,
			height: node.height,
		}
	}

	// Recursively split children
	splitBSP(node.left, minSize)
	splitBSP(node.right, minSize)
}

// createRooms creates rooms in leaf nodes
func createRooms(node *bspNode) {
	if node.left != nil || node.right != nil {
		// Not a leaf node, recurse
		if node.left != nil {
			createRooms(node.left)
		}
		if node.right != nil {
			createRooms(node.right)
		}
		return
	}

	// Leaf node - create a room
	roomWidth := minRoomSize + rand.Intn(node.width-minRoomSize-roomPadding+1)
	roomHeight := minRoomSize + rand.Intn(node.height-minRoomSize-roomPadding+1)

	if roomWidth > node.width-roomPadding {
		roomWidth = node.width - roomPadding
	}
	if roomHeight > node.height-roomPadding {
		roomHeight = node.height - roomPadding
	}

	roomX := node.x + rand.Intn(node.width-roomWidth)
	roomY := node.y + rand.Intn(node.height-roomHeight)

	// Generate unique room name
	roomCounter++
	adjective := roomAdjectives[rand.Intn(len(roomAdjectives))]
	baseName := roomNames[rand.Intn(len(roomNames))]
	name := fmt.Sprintf("%s %s", adjective, baseName)
	description := fmt.Sprintf("ROOM_%s", baseName)

	node.room = &bspRoom{
		x:           roomX,
		y:           roomY,
		width:       roomWidth,
		height:      roomHeight,
		name:        name,
		description: description,
	}
}

// carveRooms marks room cells as walkable in the grid
func carveRooms(grid *world.Grid, node *bspNode) {
	if node.room != nil {
		// Carve the room - all cells get the same room name
		for row := node.room.y; row < node.room.y+node.room.height; row++ {
			for col := node.room.x; col < node.room.x+node.room.width; col++ {
				grid.MarkAsRoomWithName(row, col, node.room.name, node.room.description)
			}
		}
	}

	if node.left != nil {
		carveRooms(grid, node.left)
	}
	if node.right != nil {
		carveRooms(grid, node.right)
	}
}

// connectRooms connects rooms with corridors
func connectRooms(grid *world.Grid, node *bspNode) {
	if node.left == nil || node.right == nil {
		return
	}

	// Get a room from each subtree
	leftRoom := getRoom(node.left)
	rightRoom := getRoom(node.right)

	if leftRoom != nil && rightRoom != nil {
		// Get center points of each room
		leftCenterX := leftRoom.x + leftRoom.width/2
		leftCenterY := leftRoom.y + leftRoom.height/2
		rightCenterX := rightRoom.x + rightRoom.width/2
		rightCenterY := rightRoom.y + rightRoom.height/2

		// Create L-shaped corridor with 3-cell width
		if rand.Intn(2) == 0 {
			// Horizontal first, then vertical
			carveCorridorHorizontal(grid, leftCenterY, leftCenterX, rightCenterX)
			carveCorridorVertical(grid, rightCenterX, leftCenterY, rightCenterY)
		} else {
			// Vertical first, then horizontal
			carveCorridorVertical(grid, leftCenterX, leftCenterY, rightCenterY)
			carveCorridorHorizontal(grid, rightCenterY, leftCenterX, rightCenterX)
		}
	}

	// Recursively connect subtrees
	connectRooms(grid, node.left)
	connectRooms(grid, node.right)
}

// carveCorridorHorizontal carves a horizontal corridor (3 cells wide)
func carveCorridorHorizontal(grid *world.Grid, row, startCol, endCol int) {
	if startCol > endCol {
		startCol, endCol = endCol, startCol
	}

	for col := startCol; col <= endCol; col++ {
		// Only mark as corridor if not already a room (don't overwrite room names)
		cell := grid.GetCell(row, col)
		if cell != nil && !cell.Room {
			grid.MarkAsRoomWithName(row, col, "Corridor", "ROOM_CORRIDOR")
		}
	}
}

// carveCorridorVertical carves a vertical corridor (3 cells wide)
func carveCorridorVertical(grid *world.Grid, col, startRow, endRow int) {
	if startRow > endRow {
		startRow, endRow = endRow, startRow
	}

	for row := startRow; row <= endRow; row++ {
		// Only mark as corridor if not already a room (don't overwrite room names)
		cell := grid.GetCell(row, col)
		if cell != nil && !cell.Room {
			grid.MarkAsRoomWithName(row, col, "Corridor", "ROOM_CORRIDOR")
		}
	}
}

// getRoom returns a room from a subtree (picks randomly from leaves)
func getRoom(node *bspNode) *bspRoom {
	if node.room != nil {
		return node.room
	}

	var leftRoom, rightRoom *bspRoom
	if node.left != nil {
		leftRoom = getRoom(node.left)
	}
	if node.right != nil {
		rightRoom = getRoom(node.right)
	}

	if leftRoom != nil && rightRoom != nil {
		if rand.Intn(2) == 0 {
			return leftRoom
		}
		return rightRoom
	}

	if leftRoom != nil {
		return leftRoom
	}
	return rightRoom
}

// collectRooms collects all rooms from the BSP tree
func collectRooms(node *bspNode) []*bspRoom {
	var rooms []*bspRoom

	if node.room != nil {
		rooms = append(rooms, node.room)
	}

	if node.left != nil {
		rooms = append(rooms, collectRooms(node.left)...)
	}
	if node.right != nil {
		rooms = append(rooms, collectRooms(node.right)...)
	}

	return rooms
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// findFurthestCell uses BFS to find the cell with the longest path distance from start
// Prefers non-corridor room cells for the exit placement
func findFurthestCell(grid *world.Grid, start *world.Cell) *world.Cell {
	if start == nil {
		return nil
	}

	// BFS to calculate distances
	type cellDist struct {
		cell *world.Cell
		dist int
	}

	visited := make(map[*world.Cell]bool)
	queue := []cellDist{{start, 0}}
	visited[start] = true

	var furthestCell *world.Cell
	maxDist := -1

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Track the furthest cell, preferring actual rooms over corridors
		if current.dist > maxDist ||
			(current.dist == maxDist && current.cell.Name != "Corridor" && (furthestCell == nil || furthestCell.Name == "Corridor")) {
			maxDist = current.dist
			furthestCell = current.cell
		}

		// Explore neighbors
		neighbors := []*world.Cell{current.cell.North, current.cell.East, current.cell.South, current.cell.West}
		for _, neighbor := range neighbors {
			if neighbor != nil && neighbor.Room && !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, cellDist{neighbor, current.dist + 1})
			}
		}
	}

	return furthestCell
}
