package menu

import (
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
)

// TextInputOptions configures a blocking text-entry dialog.
type TextInputOptions struct {
	Title   string
	Prompt  string
	Initial string
	// Hex enables hexadecimal entry (0-9, A-F, optional 0x prefix; displayed uppercase).
	Hex bool
}

// TextInputDialogRenderer is implemented by renderers that support modal text entry.
type TextInputDialogRenderer interface {
	RunTextInputDialog(g *state.Game, opts TextInputOptions) (value string, ok bool)
}

// RunTextInputDialog opens a modal text-entry dialog. Returns ok=false when cancelled or unsupported.
func RunTextInputDialog(g *state.Game, opts TextInputOptions) (string, bool) {
	if g == nil {
		return "", false
	}
	if mr, ok := renderer.Current.(TextInputDialogRenderer); ok {
		return mr.RunTextInputDialog(g, opts)
	}
	return "", false
}
