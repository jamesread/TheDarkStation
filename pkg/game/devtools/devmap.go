// Package devtools provides developer tools for testing and debugging.
package devtools

import (
	"fmt"
	"strings"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// ContainsSubstring checks if s contains substr (case-insensitive)
func ContainsSubstring(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// SwitchToDevMap switches the game to a hard-coded 50x50 developer testing map
// All possible game cells are placed with a 3-cell margin between each, grouped by type in rows
func SwitchToDevMap(g *state.Game) {
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

// logMessage adds a formatted message to the game's message log
func logMessage(g *state.Game, msg string, a ...any) {
	formatted := renderer.ApplyMarkup(msg, a...)
	g.AddMessage(formatted)
}
