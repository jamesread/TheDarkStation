package gameplay

import (
	"time"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/game/state"
)

// InitRunTracking marks the start of a new run for elapsed-time statistics.
func InitRunTracking(g *state.Game) {
	if g == nil {
		return
	}
	g.RunStartedAt = time.Now().UnixMilli()
	g.CompletionPhase = state.CompletionPhaseSummary
	g.CreditsLineIndex = 0
	g.CreditsLineStartMs = 0
	g.CreditsExitStartMs = 0
	g.CreditsTransitionStartMs = 0
}

// TriggerGameComplete marks the run finished and snapshots statistics for the end screen.
func TriggerGameComplete(g *state.Game) {
	if g == nil {
		return
	}
	g.RunStatsSnapshot = g.SnapshotRunStats()
	g.GameComplete = true
	g.CompletionPhase = state.CompletionPhaseSummary
	g.CreditsLineIndex = 0
	g.CreditsLineStartMs = 0
	g.CreditsExitStartMs = 0
	g.CreditsTransitionStartMs = 0
}

// UpdateCompletionSequence auto-advances the credits roll after hold + slide-out.
func UpdateCompletionSequence(g *state.Game) {
	if g == nil || !g.GameComplete || g.CompletionPhase != state.CompletionPhaseCredits {
		return
	}
	now := time.Now().UnixMilli()
	if advanceCreditsMapTransition(g, now) {
		return
	}
	if g.CreditsLineStartMs == 0 {
		return
	}
	if finishCreditsExit(g, now) {
		return
	}
	if g.CreditsExitStartMs != 0 {
		return
	}
	enterDone := g.CreditsLineStartMs + state.CreditsSlideEnterMs
	if now < enterDone {
		return
	}
	if now-enterDone < state.CreditsLineHoldMs {
		return
	}
	if g.CreditsLineIndex >= len(state.CompletionCreditLineIDs)-1 {
		QuitToTitleMenu(g)
		return
	}
	g.CreditsExitStartMs = now
}

func advanceCreditsMapTransition(g *state.Game, now int64) bool {
	if g == nil || g.CreditsTransitionStartMs == 0 {
		return false
	}
	if now-g.CreditsTransitionStartMs < state.CreditsMapFadeMs {
		return true
	}
	// First credits line is already centered when the map fade completes.
	g.CreditsLineStartMs = now - state.CreditsSlideEnterMs
	g.CreditsTransitionStartMs = 0
	return true
}

// ProcessCompletionInput handles input on the completion / credits screens.
func ProcessCompletionInput(g *state.Game, intent engineinput.Intent) {
	if g == nil || !g.GameComplete {
		return
	}
	if intent.Action == engineinput.ActionNone {
		return
	}
	switch g.CompletionPhase {
	case state.CompletionPhaseSummary:
		now := time.Now().UnixMilli()
		g.CompletionPhase = state.CompletionPhaseCredits
		g.CreditsLineIndex = 0
		g.CreditsTransitionStartMs = now
		g.CreditsLineStartMs = 0
		g.CreditsExitStartMs = 0
	case state.CompletionPhaseCredits:
		if g.CreditsTransitionStartMs != 0 || g.CreditsLineStartMs == 0 {
			return
		}
		if g.CreditsExitStartMs != 0 {
			// Keypress during slide-out skips the remaining exit animation.
			finishCreditsExit(g, g.CreditsExitStartMs+state.CreditsSlideExitMs)
			return
		}
		if g.CreditsLineIndex >= len(state.CompletionCreditLineIDs)-1 {
			QuitToTitleMenu(g)
			return
		}
		g.CreditsExitStartMs = time.Now().UnixMilli()
	}
}

func finishCreditsExit(g *state.Game, now int64) bool {
	if g == nil || g.CreditsExitStartMs == 0 {
		return false
	}
	if now-g.CreditsExitStartMs < state.CreditsSlideExitMs {
		return true
	}
	if !advanceCreditsLine(g) {
		QuitToTitleMenu(g)
		return true
	}
	g.CreditsExitStartMs = 0
	g.CreditsLineStartMs = now
	return true
}

func advanceCreditsLine(g *state.Game) bool {
	if g == nil {
		return false
	}
	if g.CreditsLineIndex >= len(state.CompletionCreditLineIDs)-1 {
		return false
	}
	g.CreditsLineIndex++
	return true
}

// QuitToTitleMenu signals return to the main menu. The outer loop discards this
// Game and BuildGame starts a fresh run — do not reset in place here (Draw may
// still be rendering the same *Game on another thread).
func QuitToTitleMenu(g *state.Game) {
	if g == nil {
		return
	}
	g.QuitToTitle = true
}
