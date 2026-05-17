package setup

import (
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// ApplyObservationLedPuzzleCues retargets one corridor junction plaque so its
// stamp echoes the earliest row-major PuzzleSequence puzzle whose solution maps
// to a plaque msgid on this deck (Story 5.2).
// Runs after ApplyEnvironmentalSignage; does not change puzzle reachability
// or furniture code placement (see specs/level-layout-and-solvability.md).
func ApplyObservationLedPuzzleCues(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	minimal := deck.IsFinalDeck(g.Level)
	if !deck.ObservationLedPuzzleCuesActive(g.Level, minimal) {
		return
	}

	var puzzleCell *world.Cell
	var puzzleSolution string

	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if puzzleCell != nil || cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Puzzle == nil || data.Puzzle.IsSolved() {
			return
		}
		if data.Puzzle.PuzzleType != entities.PuzzleSequence {
			return
		}
		_, ok := deck.ObservationSeqPlaqueMsgID(data.Puzzle.Solution)
		if !ok {
			return
		}
		puzzleCell = cell
		puzzleSolution = data.Puzzle.Solution
	})

	if puzzleCell == nil || puzzleSolution == "" {
		return
	}

	msgid, ok := deck.ObservationSeqPlaqueMsgID(puzzleSolution)
	if !ok {
		return
	}

	bestR, bestC := -1, -1
	bestD := int(^uint(0) >> 1)

	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		if gameworld.GetGameData(cell).EnvPlaqueMsgID == "" {
			return
		}
		d := manhattan(row, col, puzzleCell.Row, puzzleCell.Col)
		if d < bestD || (d == bestD && (bestR < 0 || row < bestR || (row == bestR && col < bestC))) {
			bestD = d
			bestR, bestC = row, col
		}
	})

	if bestR < 0 {
		return
	}
	plaque := g.Grid.GetCell(bestR, bestC)
	if plaque == nil {
		return
	}
	gameworld.GetGameData(plaque).EnvPlaqueMsgID = msgid
}

func manhattan(r1, c1, r2, c2 int) int {
	dr := r1 - r2
	if dr < 0 {
		dr = -dr
	}
	dc := c1 - c2
	if dc < 0 {
		dc = -dc
	}
	return dr + dc
}
