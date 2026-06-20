package gameplay

import (
	"fmt"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/generator"
	gamemenu "darkstation/pkg/game/menu"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
)

// SpawnMode selects where the player stands after entering a deck.
type SpawnMode int

const (
	// SpawnModeLiftShaft places the player on the lift exit cell (default deck entry).
	SpawnModeLiftShaft SpawnMode = iota
	// SpawnModeShip places the player in the deck 1 Ship room when present.
	SpawnModeShip
)

// TravelToDeck saves the current deck and moves the player to targetLevel (1-based).
func TravelToDeck(g *state.Game, targetLevel int) error {
	if g == nil {
		return fmt.Errorf("no game state")
	}
	if targetLevel < 1 || targetLevel > deck.TotalDecks {
		return fmt.Errorf("deck must be between 1 and %d", deck.TotalDecks)
	}
	targetID := targetLevel - 1
	if targetID == g.CurrentDeckID {
		return nil
	}
	if !g.IsDeckTravelUnlocked(targetID) {
		return fmt.Errorf("%s", g.DeckTravelBlockReason(targetID))
	}

	g.SaveCurrentDeckState()
	clearCrossDeckPowerState(g)
	clearCompletionState(g)

	if ds := g.DeckStates[targetID]; ds != nil && ds.Grid != nil {
		g.LoadDeckState(targetID)
		applyLoadedDeckFixups(g)
		refreshDeckPower(g)
		UpdateLightingExploration(g)
	} else {
		g.CurrentDeckID = targetID
		g.Level = targetLevel
		seed := g.RunSeed + int64(targetID)*9973
		generateLevel(g, targetLevel, seed)
		refreshDeckPower(g)
		g.SaveCurrentDeckState()
		UpdateLightingExploration(g)
	}

	spawnOnDeckEntry(g, SpawnModeLiftShaft)
	g.ClearMessages()
	logMessage(g, "Lift routing: deck %d.", g.Level)
	return nil
}

// SpawnOnDeckEntry teleports the player to the chosen deck entry point.
func SpawnOnDeckEntry(g *state.Game, mode SpawnMode) {
	spawnOnDeckEntry(g, mode)
}

func spawnOnDeckEntry(g *state.Game, mode SpawnMode) {
	if g == nil || g.Grid == nil {
		return
	}
	if mode == SpawnModeShip {
		if start := g.Grid.StartCell(); start != nil && start.Name == generator.ShipRoomName {
			TeleportPlayerTo(g, start)
			g.PlayerFacing = state.FaceSouth
			return
		}
	}
	spawnInLiftShaft(g)
}

func spawnInLiftShaft(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	exit := g.Grid.ExitCell()
	if exit != nil {
		TeleportPlayerTo(g, exit)
		return
	}
	if start := g.Grid.StartCell(); start != nil {
		TeleportPlayerTo(g, start)
	}
}

// TryUseLift handles lift terminal interaction from the shaft (adjacent or standing on exit cell).
func TryUseLift(g *state.Game) bool {
	if g == nil || g.CurrentCell == nil {
		return false
	}
	cell := liftInteractionCell(g)
	if cell == nil {
		return false
	}

	switch setup.ExitLiftState(g) {
	case state.ExitLiftLockedUnpowered, state.ExitLiftLockedIncomplete:
		CheckAdjacentExitLiftAtCell(g, cell)
		return true
	}
	if !setup.ExitLiftReady(g) {
		return false
	}

	if deck.IsFinalDeck(g.Level) {
		TriggerGameComplete(g)
		return true
	}

	targetLevel, ok := gamemenu.RunLiftMenu(g)
	if !ok || targetLevel <= 0 {
		return true
	}
	if err := TravelToDeck(g, targetLevel); err != nil {
		logMessage(g, "%v", err)
		renderer.ShowDeveloperMessage(err.Error())
	}
	return true
}

func liftInteractionCell(g *state.Game) *world.Cell {
	if g.CurrentCell.ExitCell {
		return g.CurrentCell
	}
	for _, n := range []*world.Cell{g.CurrentCell.North, g.CurrentCell.South, g.CurrentCell.East, g.CurrentCell.West} {
		if n != nil && n.ExitCell {
			return n
		}
	}
	return nil
}
