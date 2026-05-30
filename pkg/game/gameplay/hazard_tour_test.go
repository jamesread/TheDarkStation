package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func hazardTourTestGame(blockingHazards int) (*state.Game, *world.Cell) {
	g := state.NewGame()
	grid := world.NewGrid(3, 3)
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			grid.MarkAsRoomWithName(r, c, "R", "")
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}
	grid.BuildAllCellConnections()
	exit := grid.GetCell(2, 2)
	grid.SetExitCell(exit)
	g.Grid = grid
	g.CurrentCell = grid.GetCell(2, 1)
	g.RoomDoorsPowered = map[string]bool{"R": true}

	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.AddGenerator(gen)
	setup.PropagateRoomPowerOnlineFromGenerators(g)

	coords := [][2]int{{0, 1}, {1, 0}, {1, 1}}
	for i := 0; i < blockingHazards && i < len(coords); i++ {
		cell := grid.GetCell(coords[i][0], coords[i][1])
		gameworld.GetGameData(cell).Hazard = entities.NewHazard(entities.HazardElectrical)
	}
	return g, exit
}

func TestStartExitHazardTour_startsWhenLiftIncomplete(t *testing.T) {
	g, _ := hazardTourTestGame(2)
	if setup.ExitLiftState(g) != state.ExitLiftLockedIncomplete {
		t.Fatalf("lift state = %v, want LockedIncomplete", setup.ExitLiftState(g))
	}
	if !StartExitHazardTour(g) {
		t.Fatal("expected hazard tour to start")
	}
	if !IsHazardTourActive(g) {
		t.Fatal("HazardTour session should be active")
	}
	if len(g.HazardTour.Targets) != 2 {
		t.Fatalf("targets = %d, want 2", len(g.HazardTour.Targets))
	}
}

func TestStartExitHazardTour_rejectsWhenLiftReady(t *testing.T) {
	g, _ := hazardTourTestGame(0)
	if StartExitHazardTour(g) {
		t.Fatal("tour should not start with no blocking hazards")
	}
}

func TestAdvanceHazardTourIfActive_visitsAllHazards(t *testing.T) {
	g, _ := hazardTourTestGame(2)
	StartExitHazardTour(g)

	start := g.HazardTour.PhaseStartMs
	AdvanceHazardTourIfActive(g, start+state.HazardClearPanMs)
	if g.HazardTour.Phase != state.HazardTourHighlight {
		t.Fatalf("phase = %v, want highlight", g.HazardTour.Phase)
	}
	if g.HazardTour.Index != 0 {
		t.Fatalf("index = %d, want 0", g.HazardTour.Index)
	}

	AdvanceHazardTourIfActive(g, g.HazardTour.PhaseStartMs+state.HazardTourHighlightMs)
	if g.HazardTour.Phase != state.HazardTourPanTo || g.HazardTour.Index != 1 {
		t.Fatalf("after first highlight: phase=%v index=%d", g.HazardTour.Phase, g.HazardTour.Index)
	}

	AdvanceHazardTourIfActive(g, g.HazardTour.PhaseStartMs+state.HazardClearPanMs)
	AdvanceHazardTourIfActive(g, g.HazardTour.PhaseStartMs+state.HazardTourHighlightMs)
	if g.HazardTour.Phase != state.HazardTourPanBack {
		t.Fatalf("phase = %v, want pan back", g.HazardTour.Phase)
	}

	AdvanceHazardTourIfActive(g, g.HazardTour.PhaseStartMs+state.HazardClearPanMs)
	if IsHazardTourActive(g) {
		t.Fatal("tour should be complete")
	}
}

func TestHazardTourSession_CameraAt(t *testing.T) {
	s := &state.HazardTourSession{
		Targets:      []state.HazardTourTarget{{Row: 2, Col: 3}, {Row: 5, Col: 7}},
		Index:        0,
		ReturnCamRow: 1, ReturnCamCol: 1,
		Phase: state.HazardTourPanTo, PhaseStartMs: 1000,
	}
	row, col, ok := s.CameraAt(1000)
	if !ok || row != 1 || col != 1 {
		t.Fatalf("pan start = (%v,%v), want (1,1)", row, col)
	}
	row, col, ok = s.CameraAt(1000 + state.HazardClearPanMs)
	if !ok || row != 2 || col != 3 {
		t.Fatalf("pan end = (%v,%v), want (2,3)", row, col)
	}

	s.Phase = state.HazardTourPanBack
	s.PhaseStartMs = 5000
	row, col, ok = s.CameraAt(5000 + state.HazardClearPanMs)
	if !ok || row != 1 || col != 1 {
		t.Fatalf("return = (%v,%v), want (1,1)", row, col)
	}
}

func TestCheckAdjacentExitLiftAtCell_startsTour(t *testing.T) {
	g, exit := hazardTourTestGame(1)
	if !CheckAdjacentExitLiftAtCell(g, exit) {
		t.Fatal("expected tour from adjacent exit USE")
	}
	if !IsHazardTourActive(g) {
		t.Fatal("tour should be active")
	}
}
