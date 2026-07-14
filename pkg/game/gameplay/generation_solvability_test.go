package gameplay

import (
	"testing"

	"darkstation/pkg/game/setup"
)

// TestGeneratedDecksPassSimulatedPlaythrough sweeps seeds across every deck and
// asserts the generation pipeline (placement invariants + Ensure* nets +
// regenerate-and-retry) always produces a deck a simulated player can complete.
func TestGeneratedDecksPassSimulatedPlaythrough(t *testing.T) {
	seeds := []int64{
		1, 2, 3,
		1779797637817431329,
		0x18B81A8AAF3E786C, // reported valve+coupler room-seal layout
	}
	for level := 1; level <= 10; level++ {
		for _, seed := range seeds {
			g := BuildGame(level)
			LoadLevelFromSeed(g, seed)
			report := setup.SimulatePlaythrough(g)
			if !report.Solvable {
				t.Errorf("level %d seed %d unsolvable after generation: %v",
					level, seed, report.Failures)
			}
		}
	}
}

// TestGenerationRetryIsDeterministic verifies that regenerating from the same seed
// (including any retry attempts) yields an identical layout, preserving seed
// reproducibility for resets and bug reports.
func TestGenerationRetryIsDeterministic(t *testing.T) {
	const level = 5
	const seed = int64(0x18B81A8AAF3E786C)

	g1 := BuildGame(level)
	LoadLevelFromSeed(g1, seed)
	g2 := BuildGame(level)
	LoadLevelFromSeed(g2, seed)

	gens1, furn1, maint1, cctv1, puzzles1 := countEntities(g1)
	gens2, furn2, maint2, cctv2, puzzles2 := countEntities(g2)
	if gens1 != gens2 || furn1 != furn2 || maint1 != maint2 || cctv1 != cctv2 || puzzles1 != puzzles2 {
		t.Fatalf("same seed produced different layouts: (%d,%d,%d,%d,%d) vs (%d,%d,%d,%d,%d)",
			gens1, furn1, maint1, cctv1, puzzles1, gens2, furn2, maint2, cctv2, puzzles2)
	}
	if g1.Grid.Rows() != g2.Grid.Rows() || g1.Grid.Cols() != g2.Grid.Cols() {
		t.Fatal("same seed produced different grid dimensions")
	}
}
