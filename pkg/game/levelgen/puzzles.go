// Package levelgen provides level generation utilities for placing entities.
package levelgen

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// PlacePuzzles places puzzle terminals that require codes found in furniture
func PlacePuzzles(g *state.Game, avoid *mapset.Set[*world.Cell]) {
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

	lockedDoors := mapset.New[*world.Cell]() // no doors yet when placing puzzles

	for i := 0; i < numPuzzles && i < len(puzzleSolutions); i++ {
		// Find a room for the puzzle
		puzzleRoom := FindRoom(g, g.Grid.StartCell(), avoid)
		if puzzleRoom == nil {
			continue
		}

		// Place on a cell that is not an articulation point, so the puzzle doesn't block the only path to a room
		placeCell := FindNonArticulationCellInRoom(g.Grid, g.Grid.StartCell(), puzzleRoom, avoid, &lockedDoors)
		if placeCell == nil {
			placeCell = puzzleRoom
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

		gameworld.GetGameData(placeCell).Puzzle = puzzle
		avoid.Put(placeCell)

		// Place the code in a furniture description in a different room
		codeRoom := FindRoom(g, g.Grid.StartCell(), avoid)
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

		g.AddHint(fmt.Sprintf("A puzzle terminal is in %s", renderer.StyledCell(placeCell.Name)))
	}
}
