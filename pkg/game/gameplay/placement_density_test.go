package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/levelseed"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func entityCounts(g *state.Game) (generators, furniture, maint int) {
	if g == nil || g.Grid == nil {
		return 0, 0, 0
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Generator != nil {
			generators++
		}
		if data.Furniture != nil {
			furniture++
		}
		if data.MaintenanceTerm != nil {
			maint++
		}
	})
	return generators, furniture, maint
}

func TestSetupLevel_mapTxtSeed_hasGeneratorsAndFurniture(t *testing.T) {
	seed, err := levelseed.Parse("18B3227D4725F32A")
	if err != nil {
		t.Fatal(err)
	}
	g := BuildGame(5)
	LoadLevelFromSeed(g, seed)

	gen, furn, maint := entityCounts(g)
	wantGens := 1 + (g.Level - 3)
	if gen < 1 {
		t.Fatalf("expected at least one generator on level %d map.txt seed, got %d", g.Level, gen)
	}
	if gen > wantGens {
		t.Fatalf("expected at most %d generators on level %d map.txt seed, got %d", wantGens, g.Level, gen)
	}
	if len(g.Generators) == 0 {
		t.Fatal("expected g.Generators populated")
	}
	if furn == 0 {
		t.Fatal("expected furniture on level 5 map.txt seed")
	}
	if maint == 0 {
		t.Fatal("expected at least one maintenance terminal")
	}
}

func TestSetupLevel_level5Seeds_haveGeneratorsAndFurniture(t *testing.T) {
	bare := 0
	for seed := int64(1); seed <= 20; seed++ {
		g := state.NewGame()
		g.InitRunUnlocks(seed * 99991)
		g.CurrentDeckID = 4
		g.Level = 5
		RegenerateFromSeed(g, seed)
		gen, furn, _ := entityCounts(g)
		if gen == 0 {
			bare++
			continue
		}
		if furn == 0 {
			bare++
		}
	}
	if bare > 3 {
		t.Fatalf("%d/20 level-5 seeds still missing generators or furniture (tolerance 3)", bare)
	}
}
