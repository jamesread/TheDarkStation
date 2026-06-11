package gameplay

import (
	"fmt"
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func levelEntityDigest(g *state.Game) string {
	if g == nil || g.Grid == nil {
		return ""
	}
	var h uint64
	add := func(v uint64) { h = h*31 + v }
	add(uint64(g.Grid.Rows()))
	add(uint64(g.Grid.Cols()))
	add(uint64(len(g.Generators)))
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		flags := uint64(0)
		if data.Door != nil {
			flags |= 1
		}
		if data.Generator != nil {
			flags |= 2
		}
		if data.MaintenanceTerm != nil {
			flags |= 4
		}
		if data.Furniture != nil {
			flags |= 8
		}
		if data.Puzzle != nil {
			flags |= 16
		}
		if data.Terminal != nil {
			flags |= 32
		}
		if data.Hazard != nil {
			flags |= 64
		}
		if flags != 0 {
			add(uint64(row)*1000 + uint64(col))
			add(flags)
		}
	})
	return fmt.Sprintf("%016x", h)
}

func TestRegenerateFromSeed_Deterministic(t *testing.T) {
	const seed = int64(1779797637817431329)
	const level = 3

	g1 := state.NewGame()
	g1.InitRunUnlocks(seed)
	g1.Level = level
	g1.CurrentDeckID = level - 1
	RegenerateFromSeed(g1, seed)
	d1 := levelEntityDigest(g1)

	g2 := state.NewGame()
	g2.InitRunUnlocks(seed)
	g2.Level = level
	g2.CurrentDeckID = level - 1
	RegenerateFromSeed(g2, seed)
	d2 := levelEntityDigest(g2)

	if d1 != d2 {
		t.Fatalf("digests differ for same seed:\n  %s\n  %s", d1, d2)
	}
	if len(g1.Generators) == 0 {
		t.Fatal("expected at least one generator after regeneration")
	}
}

func TestResetLevel_MatchesRegenerateFromSeed(t *testing.T) {
	const seed = int64(424242)
	level := 2

	g1 := state.NewGame()
	g1.InitRunUnlocks(seed)
	g1.Level = level
	g1.CurrentDeckID = level - 1
	RegenerateFromSeed(g1, seed)

	g2 := state.NewGame()
	g2.InitRunUnlocks(seed)
	g2.Level = level
	g2.CurrentDeckID = level - 1
	g2.LevelSeed = seed
	ResetLevel(g2)

	if levelEntityDigest(g1) != levelEntityDigest(g2) {
		t.Fatal("ResetLevel and RegenerateFromSeed should produce identical layouts")
	}
}
