package gameplay

import (
	"testing"

	engineWorld "darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestCheckAdjacentPuzzlesAtCell_linkageGate(t *testing.T) {
	grid := engineWorld.NewGrid(7, 7)
	for r := 1; r <= 5; r++ {
		grid.MarkAsRoomWithName(r, 3, "Hall", "ROOM_CORRIDOR")
	}
	grid.BuildAllCellConnections()

	puz := entities.NewPuzzleTerminal("T", entities.PuzzleSequence, deck.MultiHopKeyedSequenceSolution, "hint", entities.RewardBattery, "desc")
	puz.LinkageToken = deck.MultiHopLinkageToken
	puzzleCell := grid.GetCell(3, 3)
	playerCell := grid.GetCell(2, 3)
	gameworld.GetGameData(puzzleCell).Puzzle = puz

	g := state.NewGame()
	g.Grid = grid
	g.CurrentCell = playerCell
	g.AddFoundCode(deck.MultiHopKeyedSequenceSolution)

	if !CheckAdjacentPuzzlesAtCell(g, puzzleCell) {
		t.Fatal("expected interaction consumed")
	}
	if puz.IsSolved() {
		t.Fatal("puzzle must not solve without linkage token")
	}

	g.RecordLinkageToken(deck.MultiHopLinkageToken)
	if !CheckAdjacentPuzzlesAtCell(g, puzzleCell) {
		t.Fatal("expected interaction consumed")
	}
	if !puz.IsSolved() {
		t.Fatal("puzzle should solve after linkage + code")
	}
}
