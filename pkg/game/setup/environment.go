package setup

import (
	"sort"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/levelrand"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

const envMaxPlaquesPerDeck = 14

func isCorridorCell(c *world.Cell) bool {
	return c != nil && c.Room && c.Name == "Corridor" && c.Description == "ROOM_CORRIDOR"
}

// IsCorridorJunctionLayer returns true for corridor cells used as junction plaque / linkage anchors (Stories 5.1–5.3).
func IsCorridorJunctionLayer(c *world.Cell) bool {
	return isCorridorCell(c)
}

func corridorNeighborCount(grid *world.Grid, row, col int) int {
	cell := grid.GetCell(row, col)
	if cell == nil {
		return 0
	}
	n := 0
	for _, nb := range cell.GetNeighbors() {
		if isCorridorCell(nb) {
			n++
		}
	}
	return n
}

// plaqueSeed returns a reproducible seed when LevelSeed is unset.
func plaqueSeed(g *state.Game) int64 {
	if g.LevelSeed != 0 {
		return g.LevelSeed
	}
	return int64(g.Level)*48264817 + int64(g.CurrentDeckID)*77773313
}

// ApplyEnvironmentalSignage assigns gettext-backed corridor junction plaques on GameCellData.
// Junction rule: Corridor cell with ≥3 corridor neighbors (typical T/cross).
func ApplyEnvironmentalSignage(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}

	ft := deck.FunctionalType(g.Level)
	keys := deck.EnvironmentalPlaqueKeys(ft)
	if len(keys) == 0 {
		return
	}

	var junctions [][2]int
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if !isCorridorCell(cell) {
			return
		}
		if corridorNeighborCount(g.Grid, row, col) < 3 {
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

	rng := levelrand.NewDerived(plaqueSeed(g), 0xe0516e01)
	rng.Shuffle(len(junctions), func(i, j int) {
		junctions[i], junctions[j] = junctions[j], junctions[i]
	})

	max := envMaxPlaquesPerDeck
	if len(junctions) < max {
		max = len(junctions)
	}
	for i := 0; i < max; i++ {
		pos := junctions[i]
		cell := g.Grid.GetCell(pos[0], pos[1])
		if cell == nil {
			continue
		}
		idx := (int(ft) + pos[0]*31 + pos[1]*17 + i*13) % len(keys)
		if idx < 0 {
			idx = -idx
		}
		gameworld.GetGameData(cell).EnvPlaqueMsgID = keys[idx]
	}
}
