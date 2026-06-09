package gameplay

import (
	"sort"
	"time"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// IsHazardTourActive reports whether an exit hazard tour cinematic is running.
func IsHazardTourActive(g *state.Game) bool {
	return g != nil && g.HazardTour != nil
}

// IsGameplayCinematicActive reports whether input should be blocked for a camera cinematic.
func IsGameplayCinematicActive(g *state.Game) bool {
	return IsHazardClearActive(g) || IsHazardTourActive(g)
}

// AdvanceHazardTourIfActive ticks an in-progress exit hazard tour (Ebiten Update thread).
func AdvanceHazardTourIfActive(g *state.Game, nowMs int64) {
	if g == nil || g.HazardTour == nil {
		return
	}
	s := g.HazardTour
	elapsed := nowMs - s.PhaseStartMs
	if elapsed < 0 {
		elapsed = 0
	}

	switch s.Phase {
	case state.HazardTourPanTo:
		if elapsed >= state.HazardClearPanMs {
			s.Phase = state.HazardTourHighlight
			s.PhaseStartMs = nowMs
			showHazardTourCallout(g, s)
		}
	case state.HazardTourHighlight:
		if elapsed >= state.HazardTourHighlightMs {
			if s.Index+1 < len(s.Targets) {
				s.Index++
				s.Phase = state.HazardTourPanTo
				s.PhaseStartMs = nowMs
			} else {
				s.Phase = state.HazardTourPanBack
				s.PhaseStartMs = nowMs
			}
		}
	case state.HazardTourPanBack:
		if elapsed >= state.HazardClearPanMs {
			g.HazardTour = nil
		}
	}
}

// WaitForHazardTourComplete keeps rendering until the tour finishes.
func WaitForHazardTourComplete(g *state.Game) {
	for IsHazardTourActive(g) {
		renderer.RenderFrame(g)
		time.Sleep(16 * time.Millisecond)
	}
}

func blockingHazardTargets(g *state.Game) []state.HazardTourTarget {
	if g == nil || g.Grid == nil {
		return nil
	}
	var targets []state.HazardTourTarget
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !gameworld.HasBlockingHazard(cell) {
			return
		}
		targets = append(targets, state.HazardTourTarget{Row: row, Col: col})
	})
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Row != targets[j].Row {
			return targets[i].Row < targets[j].Row
		}
		return targets[i].Col < targets[j].Col
	})
	return targets
}

// StartExitHazardTour begins the hazard location tour when the lift is powered but blocked.
func StartExitHazardTour(g *state.Game) bool {
	if g == nil || g.Grid == nil || IsGameplayCinematicActive(g) {
		return false
	}
	if setup.ExitLiftState(g) != state.ExitLiftLockedIncomplete {
		return false
	}
	targets := blockingHazardTargets(g)
	if len(targets) == 0 {
		return false
	}
	retR, retC := 0.0, 0.0
	if g.CurrentCell != nil {
		retR = float64(g.CurrentCell.Row)
		retC = float64(g.CurrentCell.Col)
	}
	g.HazardTour = &state.HazardTourSession{
		Targets:      targets,
		Index:        0,
		ReturnCamRow: retR,
		ReturnCamCol: retC,
		Phase:        state.HazardTourPanTo,
		PhaseStartMs: time.Now().UnixMilli(),
	}
	return true
}

// CheckAdjacentExitLiftAtCell starts a hazard tour when USE targets a blocked exit lift.
func CheckAdjacentExitLiftAtCell(g *state.Game, cell *world.Cell) bool {
	if cell == nil || !cell.ExitCell {
		return false
	}
	if setup.ExitLiftState(g) != state.ExitLiftLockedIncomplete {
		return false
	}
	if StartExitHazardTour(g) {
		return true
	}
	if g != nil && g.IncompleteRepairCount() > 0 {
		renderer.AddCallout(cell.Row, cell.Col, blockedLiftRepairCallout(g), renderer.CalloutColorMaintenance, 0)
		return true
	}
	return false
}

func blockedLiftRepairCallout(g *state.Game) string {
	if g == nil {
		return "UNPOWERED{Lift locked}"
	}
	for _, repair := range g.RepairObjectives {
		if repair == nil || repair.IsComplete() {
			continue
		}
		if repair.IsDraining() {
			return repairDrainCallout(repair, time.Now().UnixMilli())
		}
		return "UNPOWERED{Lift locked}\nNeeds: ACTION{" + repair.Name + "}\nSUBTLE{" + repair.RoomName + "}"
	}
	return "UNPOWERED{Lift locked}"
}

func showHazardTourCallout(g *state.Game, s *state.HazardTourSession) {
	if g == nil || s == nil || s.Index < 0 || s.Index >= len(s.Targets) {
		return
	}
	t := s.Targets[s.Index]
	cell := g.Grid.GetCell(t.Row, t.Col)
	if cell == nil {
		return
	}
	hazard := gameworld.GetGameData(cell).Hazard
	if hazard == nil {
		return
	}
	renderer.AddCallout(t.Row, t.Col, formatHazardCallout(hazard), renderer.CalloutColorHazard, state.HazardTourHighlightMs)
}
