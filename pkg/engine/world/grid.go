package world

import (
	"fmt"
	"math/rand"
)

var roomDescriptions = []string{
	"ROOM_COBBLESTONE",
	"ROOM_COURTYARD",
	"ROOM_GARDEN",
	"ROOM_WORKSHOP",
	"ROOM_KITCHEN",
	"ROOM_BANQUET",
}

// Grid represents the game map with encapsulated cell storage
type Grid struct {
	roomMap map[int]map[int]*Cell
	roomDir map[string]*Cell
	rows    int
	cols    int

	startCell *Cell
	exitCell  *Cell
}

// NewGrid creates a new grid with the given dimensions
func NewGrid(rows, cols int) *Grid {
	g := &Grid{}
	g.Build(rows, cols)
	return g
}

// Rows returns the number of rows in the grid
func (g *Grid) Rows() int {
	return g.rows
}

// Cols returns the number of columns in the grid
func (g *Grid) Cols() int {
	return g.cols
}

// StartCell returns the starting cell
func (g *Grid) StartCell() *Cell {
	return g.startCell
}

// ExitCell returns the exit cell
func (g *Grid) ExitCell() *Cell {
	return g.exitCell
}

// IsValidPosition checks if a row/col position is within grid bounds
func (g *Grid) IsValidPosition(row, col int) bool {
	return row >= 0 && row < g.rows && col >= 0 && col < g.cols
}

// IsPlayablePosition checks if a position is within the playable area (not on the perimeter)
// This ensures a 1-cell wall border around the entire map
func (g *Grid) IsPlayablePosition(row, col int) bool {
	return row >= 1 && row < g.rows-1 && col >= 1 && col < g.cols-1
}

// IsOnPerimeter checks if a position is on the edge of the grid
func (g *Grid) IsOnPerimeter(row, col int) bool {
	return g.IsValidPosition(row, col) && !g.IsPlayablePosition(row, col)
}

// GetCell returns the cell at the given position, or nil if out of bounds
func (g *Grid) GetCell(row, col int) *Cell {
	if !g.IsValidPosition(row, col) {
		return nil
	}

	if g.roomMap == nil {
		return nil
	}

	rowMap, found := g.roomMap[row]
	if !found {
		return nil
	}

	return rowMap[col]
}

// GetCellByName returns a cell by its name, or nil if not found
func (g *Grid) GetCellByName(name string) *Cell {
	if g.roomDir == nil {
		return nil
	}
	return g.roomDir[name]
}

// GetCellRelative returns the cell adjacent to the given cell in the specified direction
func (g *Grid) GetCellRelative(c *Cell, dir Direction) *Cell {
	if c == nil {
		return nil
	}
	if !dir.IsValid() {
		return nil
	}
	rowRel, colRel := dir.Delta()
	return g.GetCell(c.Row+rowRel, c.Col+colRel)
}

// CenterPosition returns the row and column of the grid center
func (g *Grid) CenterPosition() (int, int) {
	return g.rows / 2, g.cols / 2
}

// GetCenterCell returns the cell at the center of the grid
func (g *Grid) GetCenterCell() *Cell {
	row, col := g.CenterPosition()
	return g.GetCell(row, col)
}

// SetStartCell sets the starting cell. Returns false if the cell is nil or not in this grid.
func (g *Grid) SetStartCell(cell *Cell) bool {
	if cell == nil {
		return false
	}
	if g.GetCell(cell.Row, cell.Col) != cell {
		return false
	}
	g.startCell = cell
	return true
}

// SetStartCellAt sets the starting cell by position. Returns false if out of bounds.
func (g *Grid) SetStartCellAt(row, col int) bool {
	cell := g.GetCell(row, col)
	if cell == nil {
		return false
	}
	g.startCell = cell
	return true
}

// SetExitCell sets the exit cell and marks it as an exit. Returns false if the cell is nil or not in this grid.
func (g *Grid) SetExitCell(cell *Cell) bool {
	if cell == nil {
		return false
	}
	if g.GetCell(cell.Row, cell.Col) != cell {
		return false
	}
	g.exitCell = cell
	cell.ExitCell = true
	return true
}

// SetExitCellAt sets the exit cell by position. Returns false if out of bounds.
func (g *Grid) SetExitCellAt(row, col int) bool {
	cell := g.GetCell(row, col)
	if cell == nil {
		return false
	}
	g.exitCell = cell
	cell.ExitCell = true
	return true
}

// MarkAsRoom marks the cell at the given position as a room. Returns false if out of bounds.
func (g *Grid) MarkAsRoom(row, col int) bool {
	cell := g.GetCell(row, col)
	if cell == nil {
		return false
	}
	cell.Room = true
	return true
}

// GenerateCellDescription returns a random room description
func GenerateCellDescription() string {
	i := rand.Intn(len(roomDescriptions))
	return roomDescriptions[i]
}

// Build initializes the grid with the given dimensions
func (g *Grid) Build(rows, cols int) {
	if rows <= 0 || cols <= 0 {
		panic("Grid dimensions must be positive")
	}

	g.rows = rows
	g.cols = cols

	g.roomMap = make(map[int]map[int]*Cell, rows)
	g.roomDir = make(map[string]*Cell)

	for currentRow := 0; currentRow < rows; currentRow++ {
		g.roomMap[currentRow] = make(map[int]*Cell)

		for currentCol := 0; currentCol < cols; currentCol++ {
			roomName := fmt.Sprintf("%v:%v", currentRow, currentCol)

			c := NewCell(currentRow, currentCol, roomName, GenerateCellDescription())

			g.roomMap[currentRow][currentCol] = c
			g.roomDir[roomName] = c
		}
	}
}

// BuildAllCellConnections connects all cells to their neighbors
func (g *Grid) BuildAllCellConnections() {
	for row := 0; row < g.rows; row++ {
		for col := 0; col < g.cols; col++ {
			cell := g.GetCell(row, col)
			if cell != nil {
				g.buildCellConnections(cell)
			}
		}
	}
}

func (g *Grid) buildCellConnections(current *Cell) {
	if current == nil {
		return
	}

	for _, dir := range AllDirections() {
		adj := g.GetCellRelative(current, dir)

		if adj == nil {
			continue
		}

		current.SetNeighbor(dir, adj)
		adj.SetNeighbor(dir.Opposite(), current)
	}
}

// ForEachCell iterates over all cells in the grid, calling the provided function for each
func (g *Grid) ForEachCell(fn func(row, col int, cell *Cell)) {
	for row := 0; row < g.rows; row++ {
		for col := 0; col < g.cols; col++ {
			cell := g.GetCell(row, col)
			if cell != nil {
				fn(row, col, cell)
			}
		}
	}
}

// Validate checks the grid for common issues and returns an error description or empty string if valid
func (g *Grid) Validate() string {
	if g.rows <= 0 || g.cols <= 0 {
		return "Grid has invalid dimensions"
	}

	if g.startCell == nil {
		return "Grid has no start cell"
	}

	if g.exitCell == nil {
		return "Grid has no exit cell"
	}

	if !g.startCell.Room {
		return "Start cell is not marked as a room"
	}

	if !g.exitCell.Room {
		return "Exit cell is not marked as a room"
	}

	return ""
}
