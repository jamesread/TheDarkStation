package state

import (
	"darkstation/pkg/game/entities"
)

// SlimePop tracks a short pop-off animation for a drained toxic-slime cell.
type SlimePop struct {
	Row         int
	Col         int
	StartedAtMs int64
}

// AddSlimePop records a pop animation for a cell that just finished draining.
func (g *Game) AddSlimePop(row, col int, nowMs int64) {
	if g == nil {
		return
	}
	for _, pop := range g.SlimePops {
		if pop.Row == row && pop.Col == col {
			return
		}
	}
	g.SlimePops = append(g.SlimePops, SlimePop{Row: row, Col: col, StartedAtMs: nowMs})
}

// PruneSlimePops removes finished pop animations.
func (g *Game) PruneSlimePops(nowMs int64) {
	if g == nil || len(g.SlimePops) == 0 {
		return
	}
	out := g.SlimePops[:0]
	for _, pop := range g.SlimePops {
		if nowMs-pop.StartedAtMs < entities.SlimePopDurationMs {
			out = append(out, pop)
		}
	}
	g.SlimePops = out
}

// SlimePopProgress returns 0..1 for an active pop at row,col, or false if none.
func (g *Game) SlimePopProgress(row, col int, nowMs int64) (float64, bool) {
	if g == nil {
		return 0, false
	}
	for _, pop := range g.SlimePops {
		if pop.Row != row || pop.Col != col {
			continue
		}
		elapsed := nowMs - pop.StartedAtMs
		if elapsed < 0 {
			elapsed = 0
		}
		if elapsed >= entities.SlimePopDurationMs {
			return 1, true
		}
		return float64(elapsed) / float64(entities.SlimePopDurationMs), true
	}
	return 0, false
}
