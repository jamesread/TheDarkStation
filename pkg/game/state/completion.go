package state

import (
	"fmt"
	"time"

	"github.com/leonelquinteros/gotext"
)

// CompletionCreditLineIDs lists gettext msgids for the end credits roll.
var CompletionCreditLineIDs = []string{
	"CREDITS_GAME_TITLE",
	"CREDITS_CREATED_BY",
	"CREDITS_ENGINE",
	"CREDITS_THANK_YOU",
}

// Credits slide timing (enter from bottom, hold, exit through top on advance).
const (
	CreditsMapFadeMs    = 700
	CreditsSlideEnterMs = 500
	CreditsSlideExitMs  = 450
	CreditsLineHoldMs   = 2500
)

// CompletionPhase tracks which end-game screen the player is viewing.
type CompletionPhase int

const (
	CompletionPhaseSummary CompletionPhase = iota
	CompletionPhaseCredits
)

// RunStats captures a snapshot of the finished run for the end screen.
type RunStats struct {
	DecksCompleted int
	Movements      int
	Interactions   int
	ElapsedSeconds int64
}

// SnapshotRunStats records current run metrics at completion time.
func (g *Game) SnapshotRunStats() RunStats {
	elapsed := int64(0)
	if g != nil && g.RunStartedAt > 0 {
		elapsed = (time.Now().UnixMilli() - g.RunStartedAt) / 1000
	}
	stats := RunStats{ElapsedSeconds: elapsed}
	if g == nil {
		return stats
	}
	stats.DecksCompleted = g.TotalDecks()
	stats.Movements = g.MovementCount
	stats.Interactions = g.InteractionsCount
	return stats
}

// FormatRunDuration formats elapsed seconds for the end-game stats panel.
func FormatRunDuration(seconds int64) string {
	if seconds < 0 {
		seconds = 0
	}
	if seconds < 60 {
		return fmt.Sprintf(gotext.Get("STAT_TIME_SECONDS"), seconds)
	}
	minutes := seconds / 60
	secs := seconds % 60
	if minutes < 60 {
		return fmt.Sprintf(gotext.Get("STAT_TIME_MINUTES"), minutes, secs)
	}
	hours := minutes / 60
	minutes = minutes % 60
	return fmt.Sprintf(gotext.Get("STAT_TIME_HOURS"), hours, minutes)
}

// ResetAllProgress clears run and deck state for a fresh game or return to title.
func (g *Game) ResetAllProgress() {
	if g == nil {
		return
	}
	fresh := NewGame()
	*g = *fresh
}
