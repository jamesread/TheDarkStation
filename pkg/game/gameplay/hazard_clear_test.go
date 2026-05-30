package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func hazardClearTestGame() (*state.Game, *world.Cell, *entities.Hazard) {
	g := makeTestGame(3, 3)
	hazardCell := g.Grid.GetCell(1, 1)
	hazard := entities.NewHazard(entities.HazardElectrical)
	gameworld.GetGameData(hazardCell).Hazard = hazard
	g.CurrentCell = g.Grid.GetCell(1, 0)
	return g, hazardCell, hazard
}

func TestStartHazardClearFromControl_startsSession(t *testing.T) {
	g, hazardCell, hazard := hazardClearTestGame()
	ctrlCell := g.Grid.GetCell(0, 1)
	control := entities.NewHazardControl(entities.HazardElectrical, hazard)
	gameworld.GetGameData(ctrlCell).HazardControl = control
	g.RoomCCTVPowered = map[string]bool{ctrlCell.Name: true}

	if !StartHazardClearFromControl(g, ctrlCell, control) {
		t.Fatal("expected cinematic to start")
	}
	if !IsHazardClearActive(g) {
		t.Fatal("HazardClear session should be active")
	}
	if g.HazardClear.HazardRow != hazardCell.Row || g.HazardClear.HazardCol != hazardCell.Col {
		t.Fatalf("hazard target = (%d,%d), want (%d,%d)",
			g.HazardClear.HazardRow, g.HazardClear.HazardCol, hazardCell.Row, hazardCell.Col)
	}
	if control.Activated {
		t.Error("control should not activate until fade completes")
	}
}

func TestAdvanceHazardClearIfActive_completesPhases(t *testing.T) {
	g, _, hazard := hazardClearTestGame()
	ctrlCell := g.Grid.GetCell(0, 1)
	control := entities.NewHazardControl(entities.HazardElectrical, hazard)
	gameworld.GetGameData(ctrlCell).HazardControl = control
	g.RoomCCTVPowered = map[string]bool{ctrlCell.Name: true}
	StartHazardClearFromControl(g, ctrlCell, control)

	start := g.HazardClear.PhaseStartMs
	AdvanceHazardClearIfActive(g, start+state.HazardClearPanMs)
	if g.HazardClear.Phase != state.HazardClearFlash {
		t.Fatalf("phase = %v, want flash", g.HazardClear.Phase)
	}

	flashStart := g.HazardClear.PhaseStartMs
	AdvanceHazardClearIfActive(g, flashStart+91)
	if g.HazardClear.VisualAlpha >= 1 || g.HazardClear.VisualAlpha <= 0 {
		t.Fatalf("flash visual alpha = %v, want dimmed pulse between 0 and 1", g.HazardClear.VisualAlpha)
	}

	AdvanceHazardClearIfActive(g, flashStart+state.HazardClearFlashMs)
	if g.HazardClear.Phase != state.HazardClearFade {
		t.Fatalf("phase = %v, want fade", g.HazardClear.Phase)
	}

	mid := g.HazardClear.PhaseStartMs + hazardClearFadeMs/2
	AdvanceHazardClearIfActive(g, mid)
	if g.HazardClear.FadeProgress <= 0 {
		t.Fatalf("fade progress = %v, want > 0", g.HazardClear.FadeProgress)
	}

	endFade := g.HazardClear.PhaseStartMs + hazardClearFadeMs
	AdvanceHazardClearIfActive(g, endFade)
	if !control.Activated {
		t.Error("control should activate after fade")
	}
	if g.HazardClear.Phase != state.HazardClearPanBack {
		t.Fatalf("phase = %v, want pan back", g.HazardClear.Phase)
	}

	endPan := g.HazardClear.PhaseStartMs + state.HazardClearPanMs
	AdvanceHazardClearIfActive(g, endPan)
	if IsHazardClearActive(g) {
		t.Error("cinematic should be complete")
	}
}

func TestHazardClearSession_CameraAt(t *testing.T) {
	s := &state.HazardClearSession{
		HazardRow: 5, HazardCol: 7,
		ReturnCamRow: 1, ReturnCamCol: 2,
		Phase: state.HazardClearPanTo, PhaseStartMs: 1000,
	}
	row, col, ok := s.CameraAt(1000)
	if !ok {
		t.Fatal("expected camera")
	}
	if row != 1 || col != 2 {
		t.Fatalf("start camera = (%v,%v), want (1,2)", row, col)
	}
	row, col, ok = s.CameraAt(1000 + state.HazardClearPanMs)
	if !ok {
		t.Fatal("expected camera at pan end")
	}
	if row != 5 || col != 7 {
		t.Fatalf("end pan camera = (%v,%v), want (5,7)", row, col)
	}

	s.Phase = state.HazardClearFlash
	s.PhaseStartMs = 1500
	row, col, ok = s.CameraAt(1600)
	if !ok || row != 5 || col != 7 {
		t.Fatalf("flash camera = (%v,%v), want (5,7)", row, col)
	}

	s.Phase = state.HazardClearPanBack
	s.PhaseStartMs = 2000
	row, col, ok = s.CameraAt(2000 + state.HazardClearPanMs)
	if !ok {
		t.Fatal("expected camera at pan back end")
	}
	if row != 1 || col != 2 {
		t.Fatalf("return camera = (%v,%v), want (1,2)", row, col)
	}
}
