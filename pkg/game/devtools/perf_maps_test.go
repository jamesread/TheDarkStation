package devtools

import (
	"testing"
	"time"

	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
)

func TestSwitchToPerfMap_scenariosLoad(t *testing.T) {
	for _, scenario := range PerfMapScenarios {
		t.Run(scenario, func(t *testing.T) {
			g := state.NewGame()
			got := SwitchToPerfMap(g, scenario)
			if got != scenario {
				t.Fatalf("scenario = %q, want %q", got, scenario)
			}
			if g.Grid == nil {
				t.Fatal("grid is nil")
			}
			if g.CurrentCell == nil {
				t.Fatal("current cell is nil")
			}
			if !g.CurrentCell.Room {
				t.Fatal("current cell should be walkable")
			}
			if !g.HasMap {
				t.Fatal("performance maps should be fully visible")
			}
		})
	}
}

func TestSwitchToPerfMap_unknownFallsBackToOpen(t *testing.T) {
	g := state.NewGame()
	if got := SwitchToPerfMap(g, "not-a-map"); got != "open" {
		t.Fatalf("unknown scenario = %q, want open", got)
	}
}

func TestSwitchToPerfMap_entitiesSyncsGeneratorsFromGrid(t *testing.T) {
	g := state.NewGame()
	SwitchToPerfMap(g, "entities")
	if len(g.Generators) == 0 {
		t.Fatal("mixed entity perf map should register grid generators for power simulation")
	}
	if len(g.RepairObjectives) != 0 {
		t.Fatalf("registered repairs = %d, want 0 for mixed entity stress map", len(g.RepairObjectives))
	}
}

func TestSwitchToPerfMap_entitiesGenerators_registersAndSimulatesPower(t *testing.T) {
	g := state.NewGame()
	SwitchToPerfMap(g, "entities_generators")
	if len(g.Generators) == 0 {
		t.Fatal("entities_generators should register generators like normal gameplay")
	}
	for _, gen := range g.Generators {
		if gen == nil || !gen.IsPowered() {
			t.Fatal("perf generators should be fueled and online for power-grid stress")
		}
	}

	start := time.Now()
	if setup.AnyArmedGridOverloaded(g) {
		// Dense powered generators on one armed floor typically overload; still must finish quickly.
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("AnyArmedGridOverloaded took %v, want under 2s on dense generator perf map", elapsed)
	}
}
