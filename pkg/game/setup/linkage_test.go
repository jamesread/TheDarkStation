package setup

import (
	"strings"
	"testing"

	engineWorld "darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// Corridor spine with two plaques: observation retargets nearest to puzzle 1234 at (10,7);
// linkage selects a DISTINCT non-OBS junction for keyed puzzle 2468 at (12,7) + furniture relay.
func TestApplyMultiHopLinkage_junctionDistinctFromObservation(t *testing.T) {
	grid := engineWorld.NewGrid(16, 16)
	for row := 3; row <= 13; row++ {
		grid.MarkAsRoomWithName(row, 7, "Corridor", "ROOM_CORRIDOR")
	}
	for col := 3; col <= 13; col++ {
		grid.MarkAsRoomWithName(7, col, "Corridor", "ROOM_CORRIDOR")
	}
	grid.MarkAsRoomWithName(10, 9, "Armory", "")
	grid.MarkAsRoomWithName(12, 9, "Locker", "")
	grid.BuildAllCellConnections()

	g := state.NewGame()
	g.Grid = grid
	g.Level = 5
	g.LevelSeed = 55_555_555

	gameworld.GetGameData(grid.GetCell(7, 7)).EnvPlaqueMsgID = "ENV_PLAQUE_HAB_ATM"
	gameworld.GetGameData(grid.GetCell(5, 7)).EnvPlaqueMsgID = "ENV_PLAQUE_PWR_PHASE"

	p1 := entities.NewPuzzleTerminal("A", entities.PuzzleSequence, "1-2-3-4", "", entities.RewardBattery, "")
	p2 := entities.NewPuzzleTerminal("B", entities.PuzzleSequence, deck.MultiHopKeyedSequenceSolution, "", entities.RewardBattery, "")
	gameworld.GetGameData(grid.GetCell(10, 7)).Puzzle = p1
	gameworld.GetGameData(grid.GetCell(12, 7)).Puzzle = p2

	f := entities.NewFurniture("Desk", "Code: "+deck.MultiHopKeyedSequenceSolution, "")
	gameworld.GetGameData(grid.GetCell(12, 9)).Furniture = f

	ApplyObservationLedPuzzleCues(g)
	ApplyMultiHopLinkage(g)

	obs := gameworld.GetGameData(grid.GetCell(7, 7)).EnvPlaqueMsgID
	if !strings.HasPrefix(obs, "ENV_PLAQUE_OBS_") {
		t.Fatalf("expected observation retarget at (7,7), got msg %q", obs)
	}

	linkCell := grid.GetCell(5, 7)
	if linkCell == nil {
		t.Fatal("missing plaque cell")
	}
	ld := gameworld.GetGameData(linkCell)
	if ld.EnvPlaqueMsgID != linkagePlaqueMsgID {
		t.Fatalf("linkage plaque msg = %q want %q", ld.EnvPlaqueMsgID, linkagePlaqueMsgID)
	}
	if ld.LinkageTag != deck.MultiHopLinkageToken {
		t.Fatalf("LinkageTag = %q want %q", ld.LinkageTag, deck.MultiHopLinkageToken)
	}
	if p2.LinkageToken != deck.MultiHopLinkageToken {
		t.Fatalf("puzzle LinkageToken = %q", p2.LinkageToken)
	}
	if !strings.Contains(f.Description, "Relay: "+deck.MultiHopLinkageToken) {
		t.Fatalf("furniture missing relay: %q", f.Description)
	}
}

func TestApplyMultiHopLinkage_earlyLevel_noChange(t *testing.T) {
	grid := engineWorld.NewGrid(12, 12)
	grid.MarkAsRoomWithName(5, 5, "Corridor", "ROOM_CORRIDOR")
	grid.BuildAllCellConnections()

	g := state.NewGame()
	g.Grid = grid
	g.Level = 4
	p := entities.NewPuzzleTerminal("B", entities.PuzzleSequence, deck.MultiHopKeyedSequenceSolution, "", entities.RewardBattery, "")
	gameworld.GetGameData(grid.GetCell(5, 5)).Puzzle = p

	ApplyMultiHopLinkage(g)
	if p.LinkageToken != "" {
		t.Fatal("tier off: puzzle should not gain linkage")
	}
}
