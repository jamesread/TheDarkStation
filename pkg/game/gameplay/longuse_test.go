package gameplay

import (
	"testing"
	"time"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func makeLongUseTestGame() (*state.Game, *world.Cell) {
	g := state.NewGame()
	grid := world.NewGrid(2, 2)
	grid.MarkAsRoomWithName(0, 0, "R", "desc")
	grid.MarkAsRoomWithName(0, 1, "R", "desc")
	grid.SetStartCellAt(0, 0)
	grid.BuildAllCellConnections()
	g.Grid = grid
	g.CurrentCell = grid.GetCell(0, 0)

	genCell := grid.GetCell(0, 1)
	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteries(1)
	gen.Trip()
	gameworld.InitGameData(genCell)
	gameworld.GetGameData(genCell).Generator = gen
	g.AddGenerator(gen)
	return g, genCell
}

func TestGeneratorNeedsLongUsePowerUp_fueledAwaitingStartup(t *testing.T) {
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteries(1)
	if !GeneratorNeedsLongUsePowerUp(gen) {
		t.Fatal("fueled offline generator should need long use startup")
	}
	gen.BringOnline()
	if GeneratorNeedsLongUsePowerUp(gen) {
		t.Fatal("online generator should not need long use")
	}
}

func TestGeneratorNeedsLongUsePowerUp_trippedWithBatteries(t *testing.T) {
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteries(1)
	gen.Trip()
	if !GeneratorNeedsLongUsePowerUp(gen) {
		t.Fatal("tripped generator with batteries should need long use")
	}
	gen.Restart()
	if GeneratorNeedsLongUsePowerUp(gen) {
		t.Fatal("powered generator should not need long use")
	}
}

func TestTryBeginLongUseOnAdjacent_startsSession(t *testing.T) {
	g, genCell := makeLongUseTestGame()
	if !TryBeginLongUseOnAdjacent(g) {
		t.Fatal("expected long use to start for tripped adjacent generator")
	}
	if g.LongUse == nil {
		t.Fatal("LongUse session should be set")
	}
	if g.LongUse.TargetRow != genCell.Row || g.LongUse.TargetCol != genCell.Col {
		t.Fatalf("target = (%d,%d), want (%d,%d)", g.LongUse.TargetRow, g.LongUse.TargetCol, genCell.Row, genCell.Col)
	}
	if g.LongUse.DurationMs != GeneratorPowerUpDuration.Milliseconds() {
		t.Fatalf("duration = %d, want %d", g.LongUse.DurationMs, GeneratorPowerUpDuration.Milliseconds())
	}
}

func TestTickLongUse_completesAfterDuration(t *testing.T) {
	g, _ := makeLongUseTestGame()
	TryBeginLongUseOnAdjacent(g)
	start := g.LongUse.StartedAtMs
	if !TickLongUse(g, start+GeneratorPowerUpDuration.Milliseconds()) {
		t.Fatal("should complete after full duration")
	}
}

func TestCompleteLongUse_powersGenerator(t *testing.T) {
	g, genCell := makeLongUseTestGame()
	gen := gameworld.GetGameData(genCell).Generator
	g.RoomDoorsPowered = map[string]bool{"R": false}
	g.RoomCCTVPowered = map[string]bool{"R": false}
	TryBeginLongUseOnAdjacent(g)
	g.LongUse.StartedAtMs = time.Now().UnixMilli() - GeneratorPowerUpDuration.Milliseconds()
	CompleteLongUse(g)
	if !gen.IsPowered() {
		t.Fatal("generator should be powered after long use completes")
	}
	if !g.RoomDoorsPowered["R"] {
		t.Fatal("generator room circuit should turn ON when generator powers up")
	}
	if !g.RoomCCTVPowered["R"] {
		t.Fatal("generator room CCTV should turn ON when generator powers up")
	}
	if IsLongUseActive(g) {
		t.Fatal("session should be cleared after complete")
	}
}

func TestCompleteLongUse_startsWastePumpDrain(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 3)
	grid.MarkAsRoomWithName(0, 0, "A", "")
	grid.MarkAsRoomWithName(0, 1, "Pump", "")
	grid.MarkAsRoomWithName(0, 2, "Lift", "")
	grid.BuildAllCellConnections()
	grid.SetStartCellAt(0, 0)
	g.Grid = grid
	g.CurrentCell = grid.GetCell(0, 0)

	repair := entities.NewRepairObjective("pump", entities.RepairWastePump, "Pump", 0, 1)
	repair.BlockerName = "Toxic Slime"
	repair.BlockerRow = 0
	repair.BlockerCol = 2
	gameworld.GetGameData(grid.GetCell(0, 1)).RepairDevice = repair
	gameworld.GetGameData(grid.GetCell(0, 2)).RepairBlocker = repair
	g.RepairObjectives = []*entities.RepairObjective{repair}

	if !TryBeginLongUseOnAdjacent(g) {
		t.Fatal("expected adjacent waste pump repair to start long use")
	}
	CompleteLongUse(g)
	if !repair.IsDraining() {
		t.Fatal("waste pump should start timed draining after long use")
	}
	if !gameworld.HasBlockingRepairBlocker(grid.GetCell(0, 2)) {
		t.Fatal("toxic slime should block while draining")
	}
	if !g.AdvanceRepairTimers(time.Now().UnixMilli() + entities.WastePumpDrainDurationMs + 1) {
		t.Fatal("drain timer should complete")
	}
	if gameworld.HasBlockingRepairBlocker(grid.GetCell(0, 2)) {
		t.Fatal("toxic slime should stop blocking after drain completes")
	}
}

func TestCheckAdjacentRepair_signalCalibrationCompletesInSequence(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 2)
	grid.MarkAsRoomWithName(0, 0, "A", "")
	grid.MarkAsRoomWithName(0, 1, "Signal", "")
	grid.BuildAllCellConnections()
	grid.SetStartCellAt(0, 0)
	g.Grid = grid
	g.CurrentCell = grid.GetCell(0, 0)

	repair := entities.NewRepairObjective("signal", entities.RepairSignalCalibrator, "Signal", 0, 1)
	gameworld.GetGameData(grid.GetCell(0, 1)).RepairDevice = repair
	g.RepairObjectives = []*entities.RepairObjective{repair}

	for i := 0; i < entities.SignalCalibrationSteps; i++ {
		if !CheckAdjacentRepairAtCell(g, grid.GetCell(0, 1)) {
			t.Fatalf("calibration step %d did not consume interaction", i+1)
		}
	}
	if !repair.IsComplete() {
		t.Fatal("signal calibrator should complete after calibration sequence")
	}
}

func TestCancelLongUse_clearsSession(t *testing.T) {
	g, _ := makeLongUseTestGame()
	TryBeginLongUseOnAdjacent(g)
	CancelLongUse(g)
	if IsLongUseActive(g) {
		t.Fatal("session should be cleared")
	}
}

func TestAdvanceLongUseIfActive_completesWhenHeld(t *testing.T) {
	g, genCell := makeLongUseTestGame()
	gen := gameworld.GetGameData(genCell).Generator
	TryBeginLongUseOnAdjacent(g)
	start := int64(1000)

	AdvanceLongUseIfActive(g, true, false, start)
	if gen.IsPowered() {
		t.Fatal("should not complete before full hold duration")
	}
	AdvanceLongUseIfActive(g, true, false, start+100)
	AdvanceLongUseIfActive(g, true, false, start+GeneratorPowerUpDuration.Milliseconds())
	if !gen.IsPowered() {
		t.Fatal("generator should be powered after held for full duration")
	}
}

func TestAdvanceLongUseIfActive_cancelsOnRelease(t *testing.T) {
	g, _ := makeLongUseTestGame()
	TryBeginLongUseOnAdjacent(g)
	AdvanceLongUseIfActive(g, false, true, time.Now().UnixMilli())
	if IsLongUseActive(g) {
		t.Fatal("release should cancel long use")
	}
}

func TestAdvanceLongUseIfActive_pausesWhenNotHeld(t *testing.T) {
	g, _ := makeLongUseTestGame()
	TryBeginLongUseOnAdjacent(g)
	start := int64(1000)

	AdvanceLongUseIfActive(g, true, false, start)
	AdvanceLongUseIfActive(g, true, false, start+500)
	if g.LongUse.AccumulatedMs == 0 {
		t.Fatal("expected accumulated hold time while USE is held")
	}
	AdvanceLongUseIfActive(g, false, false, start+2000)
	if !IsLongUseActive(g) {
		t.Fatal("brief !held without release should not cancel long use")
	}
	AdvanceLongUseIfActive(g, false, true, start+2100)
	if IsLongUseActive(g) {
		t.Fatal("release should cancel long use")
	}
}

func TestLongUseProgress_clamped(t *testing.T) {
	g, _ := makeLongUseTestGame()
	TryBeginLongUseOnAdjacent(g)
	g.LongUse.AccumulatedMs = 0
	if p := LongUseProgress(g, 0); p != 0 {
		t.Fatalf("progress at start = %f, want 0", p)
	}
	g.LongUse.AccumulatedMs = GeneratorPowerUpDuration.Milliseconds() / 2
	if p := LongUseProgress(g, 0); p < 0.45 || p > 0.55 {
		t.Fatalf("progress at half = %f, want ~0.5", p)
	}
	g.LongUse.AccumulatedMs = GeneratorPowerUpDuration.Milliseconds()
	if p := LongUseProgress(g, 0); p != 1 {
		t.Fatalf("progress at end = %f, want 1", p)
	}
}
