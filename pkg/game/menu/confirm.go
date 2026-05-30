package menu

import (
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
)

// ConfirmOptions configures a blocking yes/no confirmation dialog.
type ConfirmOptions struct {
	Title   string
	Message string
}

// ConfirmDialogRenderer is implemented by renderers that support modal confirmation.
type ConfirmDialogRenderer interface {
	RunConfirmDialog(g *state.Game, opts ConfirmOptions) bool
}

// RunConfirmDialog opens a modal confirmation dialog. Returns true when confirmed.
func RunConfirmDialog(g *state.Game, opts ConfirmOptions) bool {
	if g == nil {
		return false
	}
	if mr, ok := renderer.Current.(ConfirmDialogRenderer); ok {
		return mr.RunConfirmDialog(g, opts)
	}
	return false
}

// ConfirmQuitGame asks the player to confirm exiting the application.
func ConfirmQuitGame(g *state.Game) bool {
	return RunConfirmDialog(g, ConfirmOptions{
		Title:   "Quit?",
		Message: "Are you sure you want to quit?",
	})
}
