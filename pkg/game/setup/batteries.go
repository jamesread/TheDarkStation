// Package setup provides level setup functionality for The Dark Station.
package setup

import (
	"darkstation/pkg/game/gamemode"
	"darkstation/pkg/game/levelrand"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// placeBatteries places batteries in the level.
func placeBatteries(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	prefs := g.ItemPlacement()
	// Levels 3+: Exit requires generators, so place batteries
	if g.Level >= 3 {
		if prefs.PlaceFloorBatteries {
			placeBatteriesForGenerators(g, avoid, prefs)
		}
	} else {
		// Level 1-2: Spawn generator is already powered, so no batteries needed
		// Batteries can be found in furniture for other uses
		g.Grid.ExitCell().Locked = false
	}
}

// placeBatteriesForGenerators places batteries needed for generators
func placeBatteriesForGenerators(g *state.Game, avoid *mapset.Set[*world.Cell], prefs gamemode.ItemPlacementPrefs) {
	demand := unpoweredGeneratorBatteryDemand(g)
	if demand == 0 {
		return
	}

	totalBatteries := demand + prefs.ExtraBatteryRoll(levelrand.Intn)

	for i := 0; i < totalBatteries; i++ {
		battery := world.NewItem("Battery")
		placeItem(g, PlayerEntryCell(g), battery, avoid)
	}
}

// UnpoweredGeneratorBatteryDemand sums remaining battery slots on unpowered grid generators.
func UnpoweredGeneratorBatteryDemand(g *state.Game) int {
	return unpoweredGeneratorBatteryDemand(g)
}

func unpoweredGeneratorBatteryDemand(g *state.Game) int {
	if g == nil || g.Grid == nil {
		return 0
	}
	total := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen == nil || gen.IsPowered() {
			return
		}
		total += gen.BatteriesNeeded()
	})
	return total
}
