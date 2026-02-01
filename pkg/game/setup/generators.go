// Package setup provides level setup functionality for The Dark Station.
package setup

import (
	"fmt"
	"math/rand"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// placeGenerators places generators in the level
func placeGenerators(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	// Place spawn room generator
	placeSpawnGenerator(g, avoid)

	// Place additional generators for levels 3+
	if g.Level >= 3 {
		placeAdditionalGenerators(g, avoid)
	}
}

// placeSpawnGenerator places the generator in the spawn room
func placeSpawnGenerator(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	spawnCell := g.Grid.StartCell()
	spawnRoomName := spawnCell.Name

	// Find a valid cell in the spawn room for the generator (avoid chokepoints)
	spawnRoomCell := findValidGeneratorCell(g, spawnRoomName, spawnCell, avoid)
	if spawnRoomCell == nil {
		return
	}

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

// findValidGeneratorCell finds a valid cell for generator placement
func findValidGeneratorCell(g *state.Game, roomName string, startCell *world.Cell, avoid *mapset.Set[*world.Cell]) *world.Cell {
	var validCell *world.Cell

	// First pass: prefer non-chokepoint cells
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.Room && cell.Name == roomName && validCell == nil {
			if isValidForGenerator(cell, avoid) && !isChokepoint(g.Grid, cell, startCell) {
				validCell = cell
			}
		}
	})

	// Second pass: if no non-chokepoint cell found, use any valid cell
	if validCell == nil {
		g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
			if cell != nil && cell.Room && cell.Name == roomName && validCell == nil {
				if isValidForGenerator(cell, avoid) {
					validCell = cell
				}
			}
		})
	}

	return validCell
}

// isValidForGenerator checks if a cell is valid for generator placement
func isValidForGenerator(cell *world.Cell, avoid *mapset.Set[*world.Cell]) bool {
	if avoid.Has(cell) || cell.ExitCell {
		return false
	}

	data := gameworld.GetGameData(cell)
	return data.Generator == nil && data.Door == nil && data.Terminal == nil &&
		data.Puzzle == nil && data.Furniture == nil && data.Hazard == nil &&
		data.HazardControl == nil && data.MaintenanceTerm == nil
}

// placeAdditionalGenerators places additional generators for levels 3+
func placeAdditionalGenerators(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	g.Grid.ExitCell().Locked = true

	// Place additional generators: level 3 = 1 more (total 2), level 4 = 2 more (total 3), etc.
	numAdditionalGenerators := g.Level - 3

	for i := 0; i < numAdditionalGenerators; i++ {
		batteriesRequired := calculateBatteriesForGenerator(g.Level)
		gen := entities.NewGenerator(fmt.Sprintf("Generator #%d", i+2), batteriesRequired)

		// Find a room and valid cell
		genRoom := findRoom(g, g.Grid.StartCell(), avoid)
		if genRoom == nil {
			continue
		}

		validGenCell := findValidGeneratorCell(g, genRoom.Name, g.Grid.StartCell(), avoid)
		if validGenCell != nil {
			gameworld.GetGameData(validGenCell).Generator = gen
			g.AddGenerator(gen)
			avoid.Put(validGenCell)
			g.AddHint("A generator is in " + renderer.StyledCell(genRoom.Name))
		}
	}
}

// calculateBatteriesForGenerator calculates battery requirements for a generator
func calculateBatteriesForGenerator(level int) int {
	minBatteries := 1 + (level-3)/3
	maxBatteries := 2 + (level-3)/2
	if minBatteries > 5 {
		minBatteries = 5
	}
	if maxBatteries > 5 {
		maxBatteries = 5
	}
	if maxBatteries < minBatteries {
		maxBatteries = minBatteries
	}
	return minBatteries + rand.Intn(maxBatteries-minBatteries+1)
}
