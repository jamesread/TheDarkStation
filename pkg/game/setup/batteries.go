// Package setup provides level setup functionality for The Dark Station.
package setup

import (
	"math/rand"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
)

// placeBatteries places batteries in the level
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
	extraBatteries := 1 + rand.Intn(2)
	totalBatteries := batteriesNeededForUnpowered + extraBatteries

	// Place batteries
	for i := 0; i < totalBatteries; i++ {
		battery := world.NewItem("Battery")
		placeItem(g, g.Grid.StartCell(), battery, avoid)
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
