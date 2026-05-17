package setup

import (
	"strings"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// gettext msgid — add string to po/default.pot (+ make mo).
const linkagePlaqueMsgID = "ENV_PLAQUE_LINK_MHOP_A"

// ApplyMultiHopLinkage binds the keyed sequence puzzle (see deck.MultiHopKeyedSequenceSolution) to:
// corridor junction stamp + relay line on correlating furniture (Story 5.3; specs/multi-hop-linkage-archetype.md).
//
// Ordering: PlacePuzzles → ApplyEnvironmentalSignage → ApplyObservationLedPuzzleCues → ApplyMultiHopLinkage.
// Observation may rewrite nearest plaque for the first OBS-mapped puzzle; this pass skips ENV_PLAQUE_OBS_* so it
// targets a distinct junction tier (Avoid stamp overwrite races with Story 5.2).
//
// Persisted fields live on DeckState.Grid (GameCellData, PuzzleTerminal) — reachable per specs/level-layout-and-solvability.md.
func ApplyMultiHopLinkage(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	minimal := deck.IsFinalDeck(g.Level)
	if !deck.MultiHopLinkageActive(g.Level, minimal) {
		return
	}

	keyed := deck.MultiHopKeyedSequenceSolution
	tok := deck.MultiHopLinkageToken
	relay := ". Relay: " + tok // '.' stops CheckForPuzzleCode from absorbing the relay on one line (Story 5.3)

	var puzzleCell *world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if puzzleCell != nil || cell == nil {
			return
		}
		d := gameworld.GetGameData(cell)
		if d.Puzzle == nil || d.Puzzle.IsSolved() {
			return
		}
		if d.Puzzle.PuzzleType != entities.PuzzleSequence || d.Puzzle.Solution != keyed {
			return
		}
		puzzleCell = cell
	})
	if puzzleCell == nil {
		return
	}

	codeMarker := "Code: " + keyed
	furnitureMarked := false
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !gameworld.HasFurniture(cell) {
			return
		}
		f := gameworld.GetGameData(cell).Furniture
		if !strings.Contains(f.Description, codeMarker) {
			return
		}
		furnitureMarked = true
		if !strings.Contains(f.Description, tok) {
			f.Description += relay
		}
	})
	if !furnitureMarked {
		return
	}

	plaqueCell := nearestNonObservationPlaqueJunction(g.Grid, puzzleCell.Row, puzzleCell.Col)
	if plaqueCell == nil {
		return
	}

	gameworld.GetGameData(plaqueCell).EnvPlaqueMsgID = linkagePlaqueMsgID
	gameworld.GetGameData(plaqueCell).LinkageTag = tok
	gameworld.GetGameData(puzzleCell).Puzzle.LinkageToken = tok
}

func nearestNonObservationPlaqueJunction(grid *world.Grid, prow, pcol int) *world.Cell {
	bestDist := int(^uint(0) >> 1)
	bestR, bestC := -1, -1

	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		gd := gameworld.GetGameData(cell)
		if gd.EnvPlaqueMsgID == "" {
			return
		}
		if strings.HasPrefix(gd.EnvPlaqueMsgID, obsPlaqueMsgPrefix) {
			return
		}
		d := manhattan(row, col, prow, pcol)
		if d < bestDist || (d == bestDist && (bestR < 0 || row < bestR || (row == bestR && col < bestC))) {
			bestDist = d
			bestR, bestC = row, col
		}
	})

	if bestR < 0 {
		return nil
	}
	return grid.GetCell(bestR, bestC)
}

const obsPlaqueMsgPrefix = "ENV_PLAQUE_OBS_"
