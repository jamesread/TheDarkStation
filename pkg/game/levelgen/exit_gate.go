package levelgen

import (
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/levelrand"
)

// ExitGateKind identifies a physical puzzle that may block access to the lift shaft.
type ExitGateKind string

const (
	// ExitGateNone means no extra tiles block the shaft; the lift still requires repairs/hazards/power.
	ExitGateNone ExitGateKind = "none"
	// ExitGateSlime places a waste pump and toxic-slime tiles around the exit.
	ExitGateSlime ExitGateKind = "slime"
)

// PickExitGateKind chooses an exit-gate puzzle for this deck using the level RNG.
func PickExitGateKind(level int) ExitGateKind {
	pool := exitGatePoolForLevel(level)
	if len(pool) == 0 {
		return ExitGateNone
	}
	return pool[levelrand.Intn(len(pool))]
}

func exitGatePoolForLevel(level int) []ExitGateKind {
	if level < 2 || deck.IsFinalDeck(level) {
		return nil
	}
	// Add new exit-gate puzzle types here as they are implemented.
	return []ExitGateKind{ExitGateNone, ExitGateSlime}
}
