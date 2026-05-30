package gameplay

import (
	"fmt"
	"time"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

const hazardClearFadeMs = 3000

// IsHazardClearActive reports whether a hazard shutdown cinematic is running.
func IsHazardClearActive(g *state.Game) bool {
	return g != nil && g.HazardClear != nil
}

// AdvanceHazardClearIfActive ticks an in-progress hazard clear cinematic (Ebiten Update thread).
func AdvanceHazardClearIfActive(g *state.Game, nowMs int64) {
	if g == nil || g.HazardClear == nil {
		return
	}
	s := g.HazardClear
	elapsed := nowMs - s.PhaseStartMs
	if elapsed < 0 {
		elapsed = 0
	}

	switch s.Phase {
	case state.HazardClearPanTo:
		if elapsed >= state.HazardClearPanMs {
			s.Phase = state.HazardClearFlash
			s.PhaseStartMs = nowMs
			s.FadeProgress = 0
			s.VisualAlpha = 1
		}
	case state.HazardClearFlash:
		if elapsed >= state.HazardClearFlashMs {
			s.Phase = state.HazardClearFade
			s.PhaseStartMs = nowMs
			s.FadeProgress = 0
			s.VisualAlpha = 1
		}
	case state.HazardClearFade:
		p := float64(elapsed) / float64(hazardClearFadeMs)
		if p > 1 {
			p = 1
		}
		s.FadeProgress = p
		if elapsed >= hazardClearFadeMs {
			completeHazardClearFix(g, s)
			s.Phase = state.HazardClearPanBack
			s.PhaseStartMs = nowMs
			s.FadeProgress = 1
		}
	case state.HazardClearPanBack:
		if elapsed >= state.HazardClearPanMs {
			g.HazardClear = nil
			return
		}
	}
	s.UpdateVisualAlpha(nowMs)
}

// WaitForHazardClearComplete keeps rendering until the cinematic finishes.
func WaitForHazardClearComplete(g *state.Game) {
	for IsHazardClearActive(g) {
		renderer.RenderFrame(g)
		time.Sleep(16 * time.Millisecond)
	}
}

func hazardCellFor(g *state.Game, hazard *entities.Hazard) *world.Cell {
	if g == nil || g.Grid == nil || hazard == nil {
		return nil
	}
	var found *world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if found != nil || cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Hazard == hazard {
			found = cell
		}
	})
	return found
}

func startHazardClear(g *state.Game, hazardCell *world.Cell, pending state.HazardClearPending) bool {
	if g == nil || hazardCell == nil || IsGameplayCinematicActive(g) {
		return false
	}
	retR, retC := 0.0, 0.0
	if g.CurrentCell != nil {
		retR = float64(g.CurrentCell.Row)
		retC = float64(g.CurrentCell.Col)
	}
	g.HazardClear = &state.HazardClearSession{
		HazardRow:    hazardCell.Row,
		HazardCol:    hazardCell.Col,
		ReturnCamRow: retR,
		ReturnCamCol: retC,
		Phase:        state.HazardClearPanTo,
		PhaseStartMs: time.Now().UnixMilli(),
		VisualAlpha:  1,
		Pending:      pending,
	}
	return true
}

// StartHazardClearFromControl begins the cinematic for an adjacent hazard control activation.
func StartHazardClearFromControl(g *state.Game, controlCell *world.Cell, control *entities.HazardControl) bool {
	if g == nil || control == nil || controlCell == nil || control.Hazard == nil {
		return false
	}
	hazardCell := hazardCellFor(g, control.Hazard)
	if hazardCell == nil {
		control.Activate()
		return false
	}
	info := entities.HazardTypes[control.Type]
	pending := state.HazardClearPending{
		Hazard:         control.Hazard,
		Control:        control,
		CalloutRow:     controlCell.Row,
		CalloutCol:     controlCell.Col,
		CalloutMessage: fmt.Sprintf("TITLE{%s activated!}", control.Name),
		LogMessage:     fmt.Sprintf("Activated %s: %s", renderer.StyledHazardCtrl(control.Name), info.FixedMessage),
	}
	if !startHazardClear(g, hazardCell, pending) {
		return false
	}
	return true
}

// StartHazardClearFromItem begins the cinematic for fixing a hazard with a carried item.
func StartHazardClearFromItem(g *state.Game, hazardCell *world.Cell, hazard *entities.Hazard, itemName string) bool {
	if g == nil || hazardCell == nil || hazard == nil || itemName == "" {
		return false
	}
	info := entities.HazardTypes[hazard.Type]
	pending := state.HazardClearPending{
		Hazard:         hazard,
		ItemName:       itemName,
		CalloutRow:     hazardCell.Row,
		CalloutCol:     hazardCell.Col,
		CalloutMessage: info.FixedMessage,
		LogMessage:     info.FixedMessage,
	}
	return startHazardClear(g, hazardCell, pending)
}

func completeHazardClearFix(g *state.Game, s *state.HazardClearSession) {
	p := s.Pending
	if p.Control != nil {
		p.Control.Activate()
	} else if p.ItemName != "" && p.Hazard != nil {
		p.Hazard.Fix()
		var remove *world.Item
		g.OwnedItems.Each(func(item *world.Item) {
			if remove == nil && item != nil && item.Name == p.ItemName {
				remove = item
			}
		})
		if remove != nil {
			g.OwnedItems.Remove(remove)
		}
	} else if p.Hazard != nil {
		p.Hazard.Fix()
	}

	if p.LogMessage != "" {
		logMessage(g, "%s", p.LogMessage)
	}
	if p.CalloutMessage != "" {
		style := renderer.CalloutColorHazardCtrl
		if p.Control == nil {
			style = renderer.CalloutColorHazard
		}
		renderer.AddCallout(p.CalloutRow, p.CalloutCol, p.CalloutMessage, style, 0)
	}
}
