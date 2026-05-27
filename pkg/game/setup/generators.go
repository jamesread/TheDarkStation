// Package setup provides level setup functionality for The Dark Station.
package setup

import (
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/levelrand"
	"fmt"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// placeGenerators places generators in the level (spawn generator only; additional gens after bootstrap).
func placeGenerators(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	placeSpawnGenerator(g, avoid)
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
		batteriesRequired = 1 + levelrand.Intn(3) // 1-3 batteries
	}

	gen := entities.NewGenerator("Generator #1", batteriesRequired)
	// Auto-power the spawn room generator
	gen.InsertBatteriesAndStart(batteriesRequired)
	gameworld.GetGameData(spawnRoomCell).Generator = gen
	g.AddGenerator(gen)
	avoid.Put(spawnRoomCell)

	// Update power supply immediately
	g.UpdatePowerSupply()
	SchedulePowerPropagation(g, PowerNowMs())

	g.AddHint("A generator is in " + renderer.StyledCell(spawnRoomName))
}

// findValidGeneratorCell finds a valid cell for generator placement
func findValidGeneratorCell(g *state.Game, roomName string, startCell *world.Cell, avoid *mapset.Set[*world.Cell]) *world.Cell {
	var preferred, fallback []*world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name != roomName {
			return
		}
		if !isValidForGenerator(cell, avoid) || !CanPlaceBlockingEntity(g, cell) {
			return
		}
		if !isChokepoint(g.Grid, cell, startCell) {
			preferred = append(preferred, cell)
		} else {
			fallback = append(fallback, cell)
		}
	})
	pool := preferred
	if len(pool) == 0 {
		pool = fallback
	}
	if len(pool) == 0 {
		return nil
	}
	levelrand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	return pool[0]
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

// PlaceAdditionalGenerators places generators beyond the auto-started spawn generator (level 3+).
func PlaceAdditionalGenerators(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	if g == nil || g.Grid == nil || g.Level < 3 {
		return
	}
	if avoid == nil {
		a := mapset.New[*world.Cell]()
		avoid = &a
	}
	exit := g.Grid.ExitCell()
	if exit != nil {
		exit.Locked = true
	}
	placeAdditionalGenerators(g, avoid)
	g.RebuildGeneratorsFromGrid()
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
	return minBatteries + levelrand.Intn(maxBatteries-minBatteries+1)
}

func placeAdditionalGenerators(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	numAdditionalGenerators := numAdditionalGeneratorsForLevel(g.Level)
	start := g.Grid.StartCell()
	for i := 0; i < numAdditionalGenerators; i++ {
		batteriesRequired := calculateBatteriesForGenerator(g.Level)
		gen := entities.NewGenerator(fmt.Sprintf("Generator #%d", i+2), batteriesRequired)
		if placeAdditionalGenerator(g, start, avoid, gen) {
			continue
		}
		// Last resort: allow rooms that already have a generator (single-room final deck).
		if placeAdditionalGeneratorInAnyRoom(g, start, avoid, gen, true) {
			continue
		}
	}
}

// numAdditionalGeneratorsForLevel returns how many unpowered generators to place beyond the spawn gen.
func numAdditionalGeneratorsForLevel(level int) int {
	if level < 3 {
		return 0
	}
	if deck.IsFinalDeck(level) {
		return 1 // GDD §10.2: final deck minimal systems
	}
	return level - 3
}

func roomHasGenerator(g *state.Game, roomName string) bool {
	if g == nil || g.Grid == nil || roomName == "" {
		return false
	}
	found := false
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if found || cell == nil || cell.Name != roomName {
			return
		}
		if gameworld.GetGameData(cell).Generator != nil {
			found = true
		}
	})
	return found
}

func placeAdditionalGenerator(g *state.Game, start *world.Cell, avoid *mapset.Set[*world.Cell], gen *entities.Generator) bool {
	for _, roomCell := range generatorRoomCandidates(g, start, avoid, true, false) {
		if tryPlaceGeneratorInRoom(g, start, avoid, gen, roomCell) {
			return true
		}
	}
	return false
}

func placeAdditionalGeneratorInAnyRoom(g *state.Game, start *world.Cell, avoid *mapset.Set[*world.Cell], gen *entities.Generator, allowOccupiedRooms bool) bool {
	for _, roomCell := range generatorRoomCandidates(g, start, avoid, false, allowOccupiedRooms) {
		if tryPlaceGeneratorInRoom(g, start, avoid, gen, roomCell) {
			return true
		}
	}
	return false
}

func tryPlaceGeneratorInRoom(g *state.Game, start *world.Cell, avoid *mapset.Set[*world.Cell], gen *entities.Generator, roomCell *world.Cell) bool {
	if roomCell == nil || roomCell.Name == "" {
		return false
	}
	validGenCell := findValidGeneratorCell(g, roomCell.Name, start, avoid)
	if validGenCell == nil {
		return false
	}
	gameworld.GetGameData(validGenCell).Generator = gen
	g.AddGenerator(gen)
	avoid.Put(validGenCell)
	g.AddHint("A generator is in " + renderer.StyledCell(roomCell.Name))
	return true
}

// generatorRoomCandidates returns deck rooms, preferring those far from start when preferFar is true.
func generatorRoomCandidates(g *state.Game, start *world.Cell, avoid *mapset.Set[*world.Cell], preferFar, allowOccupiedRooms bool) []*world.Cell {
	byRoom := make(map[string]*world.Cell)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name == "" || cell.Name == "Corridor" {
			return
		}
		if avoid != nil && avoid.Has(cell) {
			return
		}
		if !allowOccupiedRooms && roomHasGenerator(g, cell.Name) {
			return
		}
		if _, ok := byRoom[cell.Name]; !ok {
			byRoom[cell.Name] = cell
		}
	})
	if len(byRoom) == 0 {
		return nil
	}
	minDistance := 1 + g.Level
	var all, far []*world.Cell
	for _, cell := range byRoom {
		all = append(all, cell)
		if start != nil && manhattanDistance(start, cell) >= minDistance {
			far = append(far, cell)
		}
	}
	pool := all
	if preferFar && len(far) > 0 {
		pool = far
	}
	SortCellsByPosition(pool)
	levelrand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	return pool
}
