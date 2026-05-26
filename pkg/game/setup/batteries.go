// Package setup provides level setup functionality for The Dark Station.
package setup

import (
	"darkstation/pkg/game/levelrand"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// placeBatteries places batteries in the level.
// Call after EnsureGeneratorRoomBootstrap so placement respects init reachability.
func placeBatteries(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	// Levels 3+: Exit requires generators, so place batteries
	if g.Level >= 3 {
		placeBatteriesForGenerators(g, avoid)
	} else {
		// Level 1-2: Spawn generator is already powered, so no batteries needed
		// Batteries can be found in furniture for other uses
		g.Grid.ExitCell().Locked = false
	}
}

// placeBatteriesForGenerators places batteries needed for generators
func placeBatteriesForGenerators(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	// Calculate total batteries needed (excluding already-powered spawn generator)
	totalBatteriesNeeded := calculateTotalBatteriesNeeded(g)
	spawnGenBatteries := getSpawnGeneratorBatteries(g)
	batteriesNeededForUnpowered := totalBatteriesNeeded - spawnGenBatteries

	// Add 1-2 extra batteries per level for buffer
	extraBatteries := 1 + levelrand.Intn(2)
	totalBatteries := batteriesNeededForUnpowered + extraBatteries

	// Place batteries on init-reachable floor cells only
	for i := 0; i < totalBatteries; i++ {
		battery := world.NewItem("Battery")
		placeReachableItem(g, battery, avoid)
	}
}

// calculateTotalBatteriesNeeded calculates total batteries needed for all generators
func calculateTotalBatteriesNeeded(g *state.Game) int {
	total := 0
	numAdditionalGenerators := g.Level - 3
	for i := 0; i < numAdditionalGenerators; i++ {
		total += calculateBatteriesForGenerator(g.Level)
	}
	return total
}

// getSpawnGeneratorBatteries gets the battery count for the spawn generator
func getSpawnGeneratorBatteries(g *state.Game) int {
	if len(g.Generators) > 0 && g.Generators[0].IsPowered() {
		return g.Generators[0].BatteriesRequired
	}
	return 0
}

func placeReachableItem(g *state.Game, item *world.Item, avoid *mapset.Set[*world.Cell]) *world.Cell {
	if g == nil || g.Grid == nil || item == nil {
		return nil
	}
	candidates := initReachableItemCells(g, avoid)
	if len(candidates) == 0 {
		return nil
	}
	levelrand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })
	cell := candidates[0]
	cell.ItemsOnFloor.Put(item)
	if avoid != nil {
		avoid.Put(cell)
	}
	return cell
}

func initReachableItemCells(g *state.Game, avoid *mapset.Set[*world.Cell]) []*world.Cell {
	reachable := InitialReachableCells(g)
	var candidates []*world.Cell
	reachable.Each(func(cell *world.Cell) {
		if isValidForFloorItem(g, cell, avoid) {
			candidates = append(candidates, cell)
		}
	})
	return candidates
}

func isValidForFloorItem(g *state.Game, cell *world.Cell, avoid *mapset.Set[*world.Cell]) bool {
	if g == nil || cell == nil || !cell.Room || cell.Name == "" || cell.Name == "Corridor" || cell.ExitCell {
		return false
	}
	if avoid != nil && avoid.Has(cell) {
		return false
	}
	data := gameworld.GetGameData(cell)
	return data.Generator == nil && data.Door == nil && data.Terminal == nil &&
		data.Puzzle == nil && data.Furniture == nil && data.Hazard == nil &&
		data.HazardControl == nil && data.MaintenanceTerm == nil
}
