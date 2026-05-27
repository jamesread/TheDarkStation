package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/levelseed"
	gameworld "darkstation/pkg/game/world"
)

func TestSetupLevel_finalDeckSeed_notSingleGeneratorCluster(t *testing.T) {
	seed, err := levelseed.Parse("18B33B30C5578C35")
	if err != nil {
		t.Fatal(err)
	}
	g := BuildGame(deck.TotalDecks)
	LoadLevelFromSeed(g, seed)

	roomNames := map[string]bool{}
	gridGens := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		if cell.Name != "" && cell.Name != "Corridor" {
			roomNames[cell.Name] = true
		}
		if gameworld.GetGameData(cell).Generator != nil {
			gridGens++
		}
	})
	if len(roomNames) < 2 {
		t.Fatalf("final deck should have multiple rooms, got %d (%v)", len(roomNames), roomNames)
	}
	wantGens := 2 // spawn + one additional on final deck
	if gridGens != wantGens {
		t.Fatalf("grid generators = %d, want %d on final deck", gridGens, wantGens)
	}
}
