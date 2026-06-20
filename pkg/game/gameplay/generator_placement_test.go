package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/levelseed"
	gameworld "darkstation/pkg/game/world"
)

func TestSetupLevel_mapTxtSeed_level7_hasUnpoweredGenerators(t *testing.T) {
	seed, err := levelseed.Parse("18B338A3F4C98C87")
	if err != nil {
		t.Fatal(err)
	}
	g := BuildGame(7)
	LoadLevelFromSeed(g, seed)

	want := 1 + (g.Level - 3)
	gridGens := 0
	powered := 0
	unpowered := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen == nil {
			return
		}
		gridGens++
		if gen.IsPowered() {
			powered++
		} else {
			unpowered++
		}
	})
	if gridGens < 1 {
		t.Fatalf("grid generators = %d, want at least 1", gridGens)
	}
	if gridGens > want {
		t.Fatalf("grid generators = %d, want at most %d", gridGens, want)
	}
	if powered != 1 {
		t.Fatalf("powered generators = %d, want 1 (spawn generator)", powered)
	}
	if unpowered > g.Level-3 {
		t.Fatalf("unpowered generators = %d, want at most %d for battery use", unpowered, g.Level-3)
	}
}
