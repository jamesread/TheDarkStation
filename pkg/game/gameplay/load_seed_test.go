package gameplay

import (
	"testing"

	"darkstation/pkg/game/state"
)

func TestLoadLevelFromSeed_UpdatesLevelSeed(t *testing.T) {
	const seed = int64(424242)
	g := state.NewGame()
	g.Level = 2
	LoadLevelFromSeed(g, seed)
	if g.LevelSeed != seed {
		t.Fatalf("LevelSeed = %d, want %d", g.LevelSeed, seed)
	}
	if g.Grid == nil {
		t.Fatal("expected grid after load")
	}
}
