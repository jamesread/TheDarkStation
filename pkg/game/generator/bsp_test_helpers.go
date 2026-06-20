package generator

import "darkstation/pkg/game/deck"

func testThemeForLevel(level int) deck.Theme {
	themes := deck.AssignThemes(42)
	return deck.ThemeForDeckID(themes, level-1)
}
