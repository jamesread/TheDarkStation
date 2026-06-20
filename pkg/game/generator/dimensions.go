package generator

import "darkstation/pkg/game/deck"

// deckGridDimensions returns outer grid rows/cols for a deck level.
// Deck 1 is a small airlock; decks 2–9 follow a bull curve (largest at deck 5);
// the final deck is minimal.
func deckGridDimensions(level int) (rows, cols int) {
	const wallBorder = 2

	if level <= 0 {
		level = 1
	}
	// Playable area must fit the centered 5×5 lift shaft plus perimeter rooms.
	const minPlayForShaft = 8
	if level == 1 {
		return minPlayForShaft + wallBorder, 20 + wallBorder
	}
	if deck.IsFinalDeck(level) {
		return minPlayForShaft + wallBorder, 20 + wallBorder
	}
	if level < 2 || level > 9 {
		return 16 + wallBorder, 28 + wallBorder
	}

	// Bull curve: deck 5 largest; decks 2 and 9 smallest in the 2–9 band.
	const (
		maxPlayRows = 24
		maxPlayCols = 40
		minPlayRows = 14
		minPlayCols = 26
		centerLevel = 5
		maxDist     = 3 // |2-5| and |9-5|
	)
	dist := centerLevel - level
	if dist < 0 {
		dist = -dist
	}
	t := float64(dist) / float64(maxDist)
	playRows := int(float64(maxPlayRows) - t*float64(maxPlayRows-minPlayRows))
	playCols := int(float64(maxPlayCols) - t*float64(maxPlayCols-minPlayCols))
	return playRows + wallBorder, playCols + wallBorder
}
