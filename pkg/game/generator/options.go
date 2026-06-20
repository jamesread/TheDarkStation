package generator

import "darkstation/pkg/game/gamemode"

// GenerateOptions overrides layout generation for alternate game modes.
type GenerateOptions struct {
	// PlayRows and PlayCols set the playable interior size (before the wall border).
	// When zero, deckGridDimensions(level) is used.
	PlayRows int
	PlayCols int
	// LayoutLevel drives BSP split density when PlayRows/PlayCols are zero.
	// Zero uses the requested level.
	LayoutLevel int
	// SkipDeck1ShipOverlay omits the fixed deck-1 Ship room overlay.
	SkipDeck1ShipOverlay bool
}

// GenerateOptionsFromMode builds layout options from a game mode.
func GenerateOptionsFromMode(m gamemode.Mode) GenerateOptions {
	lg := m.LevelGen
	return GenerateOptions{
		PlayRows:             lg.PlayRows,
		PlayCols:             lg.PlayCols,
		LayoutLevel:          lg.LayoutLevel,
		SkipDeck1ShipOverlay: !lg.BootstrapDeck1Ship,
	}
}
