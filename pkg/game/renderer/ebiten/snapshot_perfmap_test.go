package ebiten

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
)

func TestCalculateObjectives_perfMapSkipsObjectives(t *testing.T) {
	e := &EbitenRenderer{}
	g := state.NewGame()
	g.Grid = world.NewGrid(3, 3)
	g.PerfMapScenario = "open"

	if objectives := e.calculateObjectives(g); objectives != nil {
		t.Fatalf("objectives = %v, want nil on perf map", objectives)
	}
}

func TestDeckHeaderText_perfMap(t *testing.T) {
	snap := &renderSnapshot{perfMapScenario: "entities_generators"}
	if got := deckHeaderText(snap); got != "perfmap entities_generators" {
		t.Fatalf("header = %q, want perfmap entities_generators", got)
	}
}

func TestRenderFrameSnapshot_perfMapHeader(t *testing.T) {
	e := &EbitenRenderer{}
	g := state.NewGame()
	g.Grid = world.NewGrid(3, 3)
	g.CurrentCell = g.Grid.GetCell(1, 1)
	g.PerfMapScenario = "mixed"

	e.RenderFrame(g)

	e.snapshotMutex.Lock()
	defer e.snapshotMutex.Unlock()
	if !e.snapshot.valid {
		t.Fatal("snapshot should be valid")
	}
	if e.snapshot.perfMapScenario != "mixed" {
		t.Fatalf("perfMapScenario = %q, want mixed", e.snapshot.perfMapScenario)
	}
	if len(e.snapshot.objectives) != 0 {
		t.Fatalf("objectives = %v, want none on perf map", e.snapshot.objectives)
	}
	if got := deckHeaderText(&e.snapshot); got != "perfmap mixed" {
		t.Fatalf("header = %q, want perfmap mixed", got)
	}
}
