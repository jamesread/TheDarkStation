package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func conduitFaults(g *state.Game) []*entities.RepairObjective {
	var out []*entities.RepairObjective
	for _, rep := range g.RepairObjectives {
		if rep != nil && rep.Type == entities.RepairConduitSplice {
			out = append(out, rep)
		}
	}
	return out
}

// One sweep over mid-depth seeds: faults must appear regularly, always on corridor
// cells with segment labels, decks must stay simulator-solvable, and splicing must
// extend live power coverage.
func TestConduitFaults_placementSweep(t *testing.T) {
	seedsWithFaults := 0
	spliceVerified := false
	for seed := int64(1); seed <= 12; seed++ {
		g := state.NewGame()
		g.Level = 5
		RegenerateFromSeed(g, seed)

		faults := conduitFaults(g)
		if len(faults) == 0 {
			continue
		}
		seedsWithFaults++

		for _, rep := range faults {
			cell := g.Grid.GetCell(rep.DeviceRow, rep.DeviceCol)
			if cell == nil || cell.Name != "Corridor" {
				t.Fatalf("seed %d: conduit fault %q not on a corridor cell", seed, rep.ID)
			}
			if gameworld.GetGameData(cell).RepairDevice != rep {
				t.Fatalf("seed %d: fault %q not registered on its cell", seed, rep.ID)
			}
			if rep.SegmentLabel == "" {
				t.Fatalf("seed %d: conduit fault %q missing segment label", seed, rep.ID)
			}
			if rep.RequiresPower {
				t.Fatalf("seed %d: conduit splice %q must be repairable without power", seed, rep.ID)
			}
		}

		res := setup.SimulatePlaythrough(g)
		if !res.Solvable {
			t.Fatalf("seed %d: deck with conduit faults not solvable: %v", seed, res.Failures)
		}

		if !spliceVerified {
			before := poweredCellCount(g)
			for _, rep := range faults {
				rep.Complete()
			}
			g.InvalidateLivePowerCache()
			if after := poweredCellCount(g); after <= before {
				t.Fatalf("seed %d: splicing all conduits should extend live power (before=%d after=%d)",
					seed, before, after)
			}
			spliceVerified = true
		}
	}
	if seedsWithFaults < 7 {
		t.Fatalf("expected conduit faults on most level-5 seeds, got %d/12", seedsWithFaults)
	}
	if !spliceVerified {
		t.Fatal("never verified splice power restoration")
	}
}

// The same seed must always produce identical fault placements.
func TestConduitFaults_deterministicPerSeed(t *testing.T) {
	type faultPos struct{ row, col int }
	gen := func() []faultPos {
		g := state.NewGame()
		g.Level = 6
		RegenerateFromSeed(g, 99)
		var out []faultPos
		for _, rep := range conduitFaults(g) {
			out = append(out, faultPos{rep.DeviceRow, rep.DeviceCol})
		}
		return out
	}
	a, b := gen(), gen()
	if len(a) != len(b) {
		t.Fatalf("fault counts differ between runs: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("fault %d differs between runs: %v vs %v", i, a[i], b[i])
		}
	}
}

func poweredCellCount(g *state.Game) int {
	count := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && setup.CellHasLivePower(g, cell) {
			count++
		}
	})
	return count
}
