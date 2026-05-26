package setup

import (
	"sort"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/levelrand"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

const maxRelaysPerDeck = 8

// PowerRelayPlacementActive returns whether corridor relays are placed on this deck.
func PowerRelayPlacementActive(level int) bool {
	return level >= 3 && !deck.IsFinalDeck(level)
}

// ApplyPowerRelays places corridor routing relays on junction cells (deterministic).
func ApplyPowerRelays(g *state.Game) {
	if g == nil || g.Grid == nil || !PowerRelayPlacementActive(g.Level) {
		return
	}

	var junctions [][2]int
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if !IsCorridorJunctionLayer(cell) {
			return
		}
		if corridorNeighborCount(g.Grid, row, col) < 3 {
			return
		}
		if gameworld.HasPowerRelay(cell) {
			return
		}
		junctions = append(junctions, [2]int{row, col})
	})
	if len(junctions) == 0 {
		return
	}

	sort.Slice(junctions, func(i, j int) bool {
		if junctions[i][0] != junctions[j][0] {
			return junctions[i][0] < junctions[j][0]
		}
		return junctions[i][1] < junctions[j][1]
	})

	seed := plaqueSeed(g) ^ 0x50a7e631
	rng := levelrand.NewDerived(seed, 0x0e1a7001)
	rng.Shuffle(len(junctions), func(i, j int) {
		junctions[i], junctions[j] = junctions[j], junctions[i]
	})

	limit := maxRelaysPerDeck
	if limit > len(junctions) {
		limit = len(junctions)
	}
	// Deeper decks: place more relays; some start open on level 5+.
	openFraction := 0
	if g.Level >= 5 {
		openFraction = 1
	}
	if g.Level >= 7 {
		openFraction = 2
	}

	for i := 0; i < limit; i++ {
		row, col := junctions[i][0], junctions[i][1]
		cell := g.Grid.GetCell(row, col)
		if cell == nil {
			continue
		}
		var relay *entities.PowerRelay
		if i < openFraction {
			relay = entities.NewPowerRelayOpen()
		} else {
			relay = entities.NewPowerRelay()
		}
		gameworld.GetGameData(cell).PowerRelay = relay
	}
}
