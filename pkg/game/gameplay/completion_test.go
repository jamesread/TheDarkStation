package gameplay

import (
	"testing"
	"time"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/state"
)

func TestTriggerGameComplete_SetsRunStatsAndPhase(t *testing.T) {
	g := state.NewGame()
	g.RunStartedAt = 1
	g.MovementCount = 42
	g.InteractionsCount = 7

	TriggerGameComplete(g)

	if !g.GameComplete {
		t.Fatal("GameComplete should be true")
	}
	if g.CompletionPhase != state.CompletionPhaseSummary {
		t.Fatalf("CompletionPhase = %v, want Summary", g.CompletionPhase)
	}
	if g.RunStatsSnapshot.DecksCompleted != deck.TotalDecks {
		t.Fatalf("DecksCompleted = %d, want %d", g.RunStatsSnapshot.DecksCompleted, deck.TotalDecks)
	}
	if g.RunStatsSnapshot.Movements != 42 {
		t.Fatalf("Movements = %d, want 42", g.RunStatsSnapshot.Movements)
	}
	if g.RunStatsSnapshot.Interactions != 7 {
		t.Fatalf("Interactions = %d, want 7", g.RunStatsSnapshot.Interactions)
	}
}

func TestProcessCompletionInput_AdvancesToCredits(t *testing.T) {
	g := state.NewGame()
	TriggerGameComplete(g)

	ProcessCompletionInput(g, engineinput.Intent{Action: engineinput.ActionMoveNorth})

	if g.CompletionPhase != state.CompletionPhaseCredits {
		t.Fatalf("CompletionPhase = %v, want Credits", g.CompletionPhase)
	}
	if g.CreditsTransitionStartMs == 0 {
		t.Fatal("CreditsTransitionStartMs should be set when entering credits")
	}
	if g.CreditsLineStartMs != 0 {
		t.Fatal("CreditsLineStartMs should wait until summary fade completes")
	}
	if g.QuitToTitle {
		t.Fatal("should not quit to title after first key on summary")
	}
}

func TestProcessCompletionInput_CreditsSkipExitAnimation(t *testing.T) {
	g := state.NewGame()
	TriggerGameComplete(g)
	g.CompletionPhase = state.CompletionPhaseCredits
	g.CreditsLineIndex = 0
	g.CreditsLineStartMs = time.Now().UnixMilli() - state.CreditsSlideEnterMs
	g.CreditsExitStartMs = time.Now().UnixMilli() - 50

	ProcessCompletionInput(g, engineinput.Intent{Action: engineinput.ActionMoveNorth})

	if g.CreditsLineIndex != 1 {
		t.Fatalf("CreditsLineIndex = %d, want 1 after skip-exit keypress", g.CreditsLineIndex)
	}
	if g.CreditsExitStartMs != 0 {
		t.Fatal("CreditsExitStartMs should reset after forced exit")
	}
}

func TestAdvanceCreditsMapTransition_StartsFirstLine(t *testing.T) {
	g := state.NewGame()
	TriggerGameComplete(g)
	g.CompletionPhase = state.CompletionPhaseCredits
	g.CreditsTransitionStartMs = time.Now().UnixMilli() - state.CreditsMapFadeMs - 1

	if !advanceCreditsMapTransition(g, time.Now().UnixMilli()) {
		t.Fatal("transition should complete after fade duration")
	}
	if g.CreditsTransitionStartMs != 0 {
		t.Fatal("CreditsTransitionStartMs should clear after fade")
	}
	if g.CreditsLineStartMs == 0 {
		t.Fatal("CreditsLineStartMs should be set after fade completes")
	}
}

func TestFinishCreditsExit_AdvancesLine(t *testing.T) {
	g := state.NewGame()
	TriggerGameComplete(g)
	g.CompletionPhase = state.CompletionPhaseCredits
	g.CreditsLineIndex = 0
	g.CreditsLineStartMs = time.Now().UnixMilli() - state.CreditsSlideEnterMs
	g.CreditsExitStartMs = time.Now().UnixMilli() - state.CreditsSlideExitMs - 1

	if !finishCreditsExit(g, time.Now().UnixMilli()) {
		t.Fatal("finishCreditsExit should handle completed exit")
	}
	if g.CreditsLineIndex != 1 {
		t.Fatalf("CreditsLineIndex = %d, want 1", g.CreditsLineIndex)
	}
	if g.CreditsExitStartMs != 0 {
		t.Fatal("CreditsExitStartMs should reset after exit completes")
	}
	if g.CreditsLineStartMs == 0 {
		t.Fatal("CreditsLineStartMs should restart slide-in for next line")
	}
}

func TestQuitToTitleMenu_SetsFlagWithoutResettingGrid(t *testing.T) {
	g := state.NewGame()
	g.Level = 10
	g.Grid = world.NewGrid(3, 3)
	g.CurrentDeckID = deck.FinalDeckIndex
	g.MovementCount = 999
	g.GameComplete = true

	QuitToTitleMenu(g)

	if !g.QuitToTitle {
		t.Fatal("QuitToTitle should be true")
	}
	if g.Grid == nil {
		t.Fatal("QuitToTitleMenu should not clear grid in place (Draw may still be rendering)")
	}
	if g.Level != 10 {
		t.Fatalf("Level = %d, want 10 until outer loop resets", g.Level)
	}
}
