package generator

import (
	"darkstation/pkg/engine/world"
)

const (
	// ShaftRoomName is the fixed room name for the core lift shaft on every deck.
	ShaftRoomName = "Lift Shaft"
	// shaftSize is the lift shaft width and height on every deck (square).
	shaftSize = 5
)

// shaftDimensions returns the fixed lift shaft size (5×5 on all decks).
func shaftDimensions(_ int, _ int, _ int) (w, h int) {
	return shaftSize, shaftSize
}

func shaftBounds(rows, cols, w, h int) (topRow, leftCol, bottomRow, rightCol int) {
	leftCol = (cols - w) / 2
	topRow = (rows - h) / 2
	rightCol = leftCol + w - 1
	bottomRow = topRow + h - 1
	return
}

// ShaftBounds returns inclusive row/col bounds for the centered 5×5 shaft.
func ShaftBounds(rows, cols int) (topRow, leftCol, bottomRow, rightCol int) {
	return shaftBounds(rows, cols, shaftSize, shaftSize)
}

// ShaftBoundsForLevel returns centered shaft bounds sized for the given deck level.
func ShaftBoundsForLevel(rows, cols, level int) (topRow, leftCol, bottomRow, rightCol int) {
	w, h := shaftDimensions(rows, cols, level)
	return shaftBounds(rows, cols, w, h)
}

// minShaftStripExtent is the smallest width/height of a BSP strip beside the shaft
// region that can still host a room (minRoomSize plus padding). Thinner strips are
// absorbed into the shaft leaf and stay walls.
const minShaftStripExtent = minRoomSize + roomPadding

// reserveShaftLeaf force-splits the BSP root so the centered shaft plus a one-cell
// wall ring becomes its own leaf with a fixed Lift Shaft room. The shaft then takes
// part in normal room carving and corridor connection like any other BSP room.
func reserveShaftLeaf(root *bspNode, rows, cols, level int) *bspNode {
	topRow, leftCol, bottomRow, rightCol := ShaftBoundsForLevel(rows, cols, level)

	node := root
	if leftCol-1-node.x >= minShaftStripExtent {
		_, node = splitNodeVertical(node, leftCol-1)
	}
	if node.x+node.width-(rightCol+2) >= minShaftStripExtent {
		node, _ = splitNodeVertical(node, rightCol+2)
	}
	if topRow-1-node.y >= minShaftStripExtent {
		_, node = splitNodeHorizontal(node, topRow-1)
	}
	if node.y+node.height-(bottomRow+2) >= minShaftStripExtent {
		node, _ = splitNodeHorizontal(node, bottomRow+2)
	}

	node.room = &bspRoom{
		x:           leftCol,
		y:           topRow,
		width:       rightCol - leftCol + 1,
		height:      bottomRow - topRow + 1,
		name:        ShaftRoomName,
		description: "ROOM_LIFT_SHAFT",
	}
	return node
}

// splitNodeVertical splits a leaf node at absolute column splitX into left/right children.
func splitNodeVertical(node *bspNode, splitX int) (left, right *bspNode) {
	node.left = &bspNode{x: node.x, y: node.y, width: splitX - node.x, height: node.height}
	node.right = &bspNode{x: splitX, y: node.y, width: node.x + node.width - splitX, height: node.height}
	return node.left, node.right
}

// splitNodeHorizontal splits a leaf node at absolute row splitY into top/bottom children.
func splitNodeHorizontal(node *bspNode, splitY int) (top, bottom *bspNode) {
	node.left = &bspNode{x: node.x, y: node.y, width: node.width, height: splitY - node.y}
	node.right = &bspNode{x: node.x, y: splitY, width: node.width, height: node.y + node.height - splitY}
	return node.left, node.right
}

// MarkShaftExit marks the shaft center as the exit/lift cell.
func MarkShaftExit(grid *world.Grid, level int) {
	if grid == nil {
		return
	}
	topRow, leftCol, bottomRow, rightCol := ShaftBoundsForLevel(grid.Rows(), grid.Cols(), level)
	grid.SetExitCellAt((topRow+bottomRow)/2, (leftCol+rightCol)/2)
}

func roomNameComponents(cells []*world.Cell) [][]*world.Cell {
	if len(cells) == 0 {
		return nil
	}
	inSet := make(map[*world.Cell]bool, len(cells))
	for _, c := range cells {
		inSet[c] = true
	}
	visited := make(map[*world.Cell]bool)
	var components [][]*world.Cell
	for _, start := range cells {
		if visited[start] {
			continue
		}
		var component []*world.Cell
		queue := []*world.Cell{start}
		visited[start] = true
		for len(queue) > 0 {
			c := queue[0]
			queue = queue[1:]
			component = append(component, c)
			for _, n := range []*world.Cell{c.North, c.East, c.South, c.West} {
				if n == nil || !inSet[n] || visited[n] {
					continue
				}
				visited[n] = true
				queue = append(queue, n)
			}
		}
		components = append(components, component)
	}
	return components
}

// pickStartCellOutsideShaft chooses a start position in the largest non-shaft room when possible.
func pickStartCellOutsideShaft(grid *world.Grid, rooms []*bspRoom, level int) {
	if grid == nil || len(rooms) == 0 {
		return
	}
	var candidates []*bspRoom
	for _, room := range rooms {
		if room == nil || roomOverlapsShaft(grid, room, level) {
			continue
		}
		if roomCellsOutsideShaft(grid, room, level) < 2 {
			continue
		}
		candidates = append(candidates, room)
	}
	if len(candidates) == 0 {
		for _, room := range rooms {
			if room != nil && roomCellsOutsideShaft(grid, room, level) > 0 {
				candidates = append(candidates, room)
			}
		}
	}
	if len(candidates) == 0 {
		candidates = rooms
	}

	startRoom := candidates[0]
	bestSize := roomCellsOutsideShaft(grid, startRoom, level)
	for _, room := range candidates[1:] {
		if size := roomCellsOutsideShaft(grid, room, level); size > bestSize {
			bestSize = size
			startRoom = room
		}
	}
	startRow := startRoom.y + startRoom.height/2
	startCol := startRoom.x + startRoom.width/2
	grid.SetStartCellAt(startRow, startCol)
}

func roomOverlapsShaft(grid *world.Grid, room *bspRoom, level int) bool {
	topRow, leftCol, bottomRow, rightCol := ShaftBoundsForLevel(grid.Rows(), grid.Cols(), level)
	return room.x <= rightCol && room.x+room.width-1 >= leftCol &&
		room.y <= bottomRow && room.y+room.height-1 >= topRow
}

func roomCellsOutsideShaft(grid *world.Grid, room *bspRoom, level int) int {
	if grid == nil || room == nil {
		return 0
	}
	topRow, leftCol, bottomRow, rightCol := ShaftBoundsForLevel(grid.Rows(), grid.Cols(), level)
	n := 0
	for row := room.y; row < room.y+room.height; row++ {
		for col := room.x; col < room.x+room.width; col++ {
			if row >= topRow && row <= bottomRow && col >= leftCol && col <= rightCol {
				continue
			}
			if grid.IsPlayablePosition(row, col) {
				n++
			}
		}
	}
	return n
}
