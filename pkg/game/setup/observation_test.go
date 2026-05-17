package setup

import (
	"testing"

	engineWorld "darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// Corridor cross + southern armory: puzzle cell below junction; observation pass
// should retarget nearest junction plaque to the sequence fingerprint (Story 5.2).
func TestApplyObservationLedPuzzleCues_retargetsNearestJunctionStamp(t *testing.T) {
	grid := engineWorld.NewGrid(14, 14)

	for row := 4; row <= 10; row++ {
		grid.MarkAsRoomWithName(row, 7, "Corridor", "ROOM_CORRIDOR")
	}
	for col := 4; col <= 10; col++ {
		grid.MarkAsRoomWithName(7, col, "Corridor", "ROOM_CORRIDOR")
	}
	grid.MarkAsRoomWithName(10, 7, "Armory", "")
	grid.BuildAllCellConnections()

	g := state.NewGame()
	g.Grid = grid
	g.Level = 3
	g.LevelSeed = 102_938_475

	ApplyEnvironmentalSignage(g)

	puz := entities.NewPuzzleTerminal(
		"Test Puzzle",
		entities.PuzzleSequence,
		"1-2-3-4",
		"",
		entities.RewardBattery,
		"test",
	)
	gameworld.GetGameData(grid.GetCell(10, 7)).Puzzle = puz

	ApplyObservationLedPuzzleCues(g)

	plaqueCell := grid.GetCell(7, 7)
	if plaqueCell == nil {
		t.Fatal("missing junction")
	}
	msg := gameworld.GetGameData(plaqueCell).EnvPlaqueMsgID
	if msg != "ENV_PLAQUE_OBS_SEQ_1234" {
		t.Fatalf("junction plaque msg = %q, want ENV_PLAQUE_OBS_SEQ_1234", msg)
	}
}

func TestApplyObservationLedPuzzleCues_earlyLevel_noChange(t *testing.T) {
	grid := engineWorld.NewGrid(14, 14)
	for row := 4; row <= 10; row++ {
		grid.MarkAsRoomWithName(row, 7, "Corridor", "ROOM_CORRIDOR")
	}
	for col := 4; col <= 10; col++ {
		grid.MarkAsRoomWithName(7, col, "Corridor", "ROOM_CORRIDOR")
	}
	grid.MarkAsRoomWithName(10, 7, "Armory", "")
	grid.BuildAllCellConnections()

	g := state.NewGame()
	g.Grid = grid
	g.Level = 2
	g.LevelSeed = 1

	ApplyEnvironmentalSignage(g)
	baseline := gameworld.GetGameData(grid.GetCell(7, 7)).EnvPlaqueMsgID

	puz := entities.NewPuzzleTerminal("P", entities.PuzzleSequence, "1-2-3-4", "", entities.RewardBattery, "")
	gameworld.GetGameData(grid.GetCell(10, 7)).Puzzle = puz

	ApplyObservationLedPuzzleCues(g)
	if gameworld.GetGameData(grid.GetCell(7, 7)).EnvPlaqueMsgID != baseline {
		t.Fatal("early deck should keep generic environmental plaque")
	}
}
