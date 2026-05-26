package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func countEntities(g *state.Game) (generators, furniture, maint, cctv, puzzles int) {
	if g == nil || g.Grid == nil {
		return
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
		if data.Terminal != nil {
			cctv++
		}
		if data.Puzzle != nil {
			puzzles++
		}
	})
	return
}

func TestSeed1779797637817431329_Level3_HasFullLayout(t *testing.T) {
	const seed = int64(1779797637817431329)
	const level = 3

	g := state.NewGame()
	g.Level = level
	RegenerateFromSeed(g, seed)

	gens, furn, maint, cctv, puzzles := countEntities(g)
	if gens == 0 {
		t.Fatal("expected at least one generator")
	}
	if furn == 0 {
		t.Fatal("expected furniture")
	}
	if maint == 0 {
		t.Fatal("expected maintenance terminals")
	}
	t.Logf("entities: generators=%d furniture=%d maint=%d cctv=%d puzzles=%d", gens, furn, maint, cctv, puzzles)
}
