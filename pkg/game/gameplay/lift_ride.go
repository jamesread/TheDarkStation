package gameplay

import (
	"fmt"
	"time"

	"darkstation/pkg/game/audio"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
)

// IsLiftRideActive reports whether a deck-to-deck lift cinematic is running.
func IsLiftRideActive(g *state.Game) bool {
	return g != nil && g.LiftRide != nil
}

// StartLiftRide begins the lift ride cinematic toward targetLevel (1-based).
// Returns an error if travel is invalid; the caller should surface it like TravelToDeck.
func StartLiftRide(g *state.Game, targetLevel int) error {
	if g == nil {
		return fmt.Errorf("no game state")
	}
	if targetLevel < 1 || targetLevel > g.TotalDecks() {
		return fmt.Errorf("deck must be between 1 and %d", g.TotalDecks())
	}
	targetID := targetLevel - 1
	if targetID == g.CurrentDeckID {
		return nil
	}
	if !g.IsDeckTravelUnlocked(targetID) {
		return fmt.Errorf("%s", g.DeckTravelBlockReason(targetID))
	}
	if IsGameplayCinematicActive(g) {
		return fmt.Errorf("cannot travel during cinematic")
	}
	if g.Grid == nil {
		return fmt.Errorf("no grid")
	}
	exit := g.Grid.ExitCell()
	if exit == nil {
		return fmt.Errorf("no lift shaft exit")
	}

	dir := 1
	if targetLevel < g.Level {
		dir = -1
	}

	TeleportPlayerTo(g, exit)
	now := time.Now().UnixMilli()
	g.LiftRide = &state.LiftRideSession{
		FromLevel:       g.Level,
		ToLevel:         targetLevel,
		FromExitRow:     exit.Row,
		FromExitCol:     exit.Col,
		ToExitRow:       exit.Row,
		ToExitCol:       exit.Col,
		Phase:           state.LiftRideDepart,
		PhaseStartMs:    now,
		RideStartMs:     now,
		Direction:       dir,
		PeripheralAlpha: 1,
	}
	g.LiftRide.UpdateVisual(now)
	audio.SetLiftWhirActive(true)
	return nil
}

// AdvanceLiftRideIfActive ticks visual phases for an in-progress lift ride (Ebiten Update thread).
// Deck swap is requested via NeedsSwap; WaitForLiftRideComplete performs TravelToDeck.
func AdvanceLiftRideIfActive(g *state.Game, nowMs int64) {
	if g == nil || g.LiftRide == nil {
		return
	}
	s := g.LiftRide
	if s.SwapInProgress {
		// Freeze phase clocks while TravelToDeck mutates the grid.
		return
	}
	elapsed := nowMs - s.PhaseStartMs
	if elapsed < 0 {
		elapsed = 0
	}

	switch s.Phase {
	case state.LiftRideDepart:
		s.UpdateVisual(nowMs)
		if elapsed >= state.LiftRideDepartMs {
			s.Phase = state.LiftRideTransit
			s.PhaseStartMs = nowMs
			s.UpdateVisual(nowMs)
		}
	case state.LiftRideTransit:
		s.UpdateVisual(nowMs)
		if !s.Swapped && !s.NeedsSwap && elapsed >= state.LiftRideTransitMs/2 {
			s.NeedsSwap = true
		}
		if s.Swapped && elapsed >= state.LiftRideTransitMs {
			s.ArriveScrollBase = s.ScrollOffset
			s.Phase = state.LiftRideArrive
			s.PhaseStartMs = nowMs
			s.UpdateVisual(nowMs)
		}
	case state.LiftRideArrive:
		s.UpdateVisual(nowMs)
		if elapsed >= state.LiftRideArriveMs {
			finishLiftRide(g)
		}
	}
}

// WaitForLiftRideComplete keeps rendering until the ride finishes, performing the
// mid-transit deck swap on the game goroutine (safe for level generation).
func WaitForLiftRideComplete(g *state.Game) {
	for IsLiftRideActive(g) {
		now := time.Now().UnixMilli()
		AdvanceLiftRideIfActive(g, now)
		if s := g.LiftRide; s != nil && s.NeedsSwap && !s.Swapped {
			applyLiftRideSwap(g)
		}
		renderer.RenderFrame(g)
		time.Sleep(8 * time.Millisecond)
	}
}

func applyLiftRideSwap(g *state.Game) {
	if g == nil || g.LiftRide == nil {
		return
	}
	s := g.LiftRide
	s.NeedsSwap = false
	s.SwapInProgress = true
	s.PeripheralAlpha = 0
	// Publish a frozen frame before mutating the grid so Draw never samples a half-built deck.
	renderer.RenderFrame(g)

	target := s.ToLevel
	if err := TravelToDeck(g, target); err != nil {
		logMessage(g, "%v", err)
		renderer.ShowDeveloperMessage(err.Error())
		finishLiftRide(g)
		return
	}
	if g.LiftRide == nil {
		return
	}
	s = g.LiftRide
	if exit := g.Grid.ExitCell(); exit != nil {
		s.ToExitRow = exit.Row
		s.ToExitCol = exit.Col
		// Leave FromExit as the freeze-frame camera anchor until SwapInProgress clears.
	}
	s.Swapped = true
	s.SwapInProgress = false
	// Hold transit a bit longer after a long generate so Arrive still has scroll runway.
	now := time.Now().UnixMilli()
	minRemain := int64(state.LiftRideTransitMs / 3)
	if now-s.PhaseStartMs > state.LiftRideTransitMs-minRemain {
		s.PhaseStartMs = now - (state.LiftRideTransitMs - minRemain)
	}
	s.UpdateVisual(now)
	// Publish destination camera + grid together so Draw never pairs a new camera
	// with the frozen source buffer.
	renderer.RenderFrame(g)
}

func finishLiftRide(g *state.Game) {
	if g == nil {
		return
	}
	audio.SetLiftWhirActive(false)
	g.LiftRide = nil
}
