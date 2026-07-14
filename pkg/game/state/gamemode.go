package state

import (
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/gamemode"
)

// SetMode selects the active game mode for this run.
func (g *Game) SetMode(id gamemode.ID) {
	if g == nil {
		return
	}
	g.GameMode = gamemode.Get(id)
}

// Mode returns the active game mode, defaulting to SinglePlayerPuzzle.
func (g *Game) Mode() gamemode.Mode {
	if g == nil {
		return gamemode.Default()
	}
	if g.GameMode.ID == "" {
		return gamemode.Default()
	}
	return g.GameMode
}

// ItemPlacement returns item placement preferences for the active mode.
func (g *Game) ItemPlacement() gamemode.ItemPlacementPrefs {
	return g.Mode().Items
}

// LevelGen returns level generation preferences for the active mode.
func (g *Game) LevelGen() gamemode.LevelGenPrefs {
	return g.Mode().LevelGen
}

// TotalDecks returns the deck count for the active mode.
func (g *Game) TotalDecks() int {
	n := g.Mode().TotalDecks
	if n < 1 {
		return deck.TotalDecks
	}
	return n
}

// FinalDeckIndex returns the 0-based index of the last deck in this mode.
func (g *Game) FinalDeckIndex() int {
	return g.TotalDecks() - 1
}

// IsFinalDeckLevel reports whether level (1-based) is the final deck for this mode.
func (g *Game) IsFinalDeckLevel(level int) bool {
	return level >= g.TotalDecks()
}

// NextDeckID returns the next deck ID from the current deck, or false at the final deck.
func (g *Game) NextDeckID(deckID int) (nextID int, ok bool) {
	return deck.NextDeckIDFor(g.TotalDecks(), deckID)
}
