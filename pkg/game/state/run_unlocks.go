package state

import (
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/unlocks"
)

// RunUnlockProgress builds unlock check state from the current game.
func (g *Game) RunUnlockProgress() unlocks.RunProgress {
	if g == nil {
		return unlocks.RunProgress{}
	}
	return unlocks.RunProgress{
		Plan:               g.UnlockPlan,
		Satisfied:          g.UnlockSatisfied,
		LiftRoutingPowered: g.LiftRoutingPowered,
		ReactorOnline:      g.ReactorOnline,
		DeckThemes:         g.DeckThemes,
		HasKeycard:         g.HasRunKeycard,
		RepairComplete:     g.IsRepairCompleteOnAnyDeck,
	}
}

// IsDeckTravelUnlocked reports whether the player may travel to deckID via the lift.
func (g *Game) IsDeckTravelUnlocked(deckID int) bool {
	return unlocks.IsDeckTravelUnlocked(g.RunUnlockProgress(), deckID)
}

// DeckTravelBlockReason returns why travel to deckID is blocked, or "" if allowed.
func (g *Game) DeckTravelBlockReason(deckID int) string {
	return unlocks.DeckTravelBlockReason(g.RunUnlockProgress(), deckID)
}

// ThemeForCurrentDeck returns the theme assigned to the active deck.
func (g *Game) ThemeForCurrentDeck() deck.Theme {
	return deck.ThemeForDeckID(g.DeckThemes, g.CurrentDeckID)
}

// ThemeForDeck returns the theme assigned to a deck ID.
func (g *Game) ThemeForDeck(deckID int) deck.Theme {
	return deck.ThemeForDeckID(g.DeckThemes, deckID)
}

// HasRunKeycard reports whether a keycard is in run-wide inventory.
func (g *Game) HasRunKeycard(name string) bool {
	if g == nil || name == "" {
		return false
	}
	found := false
	g.RunInventory.Each(func(item *world.Item) {
		if item != nil && item.Name == name {
			found = true
		}
	})
	return found
}

// HasKeycardNamed checks run-wide and deck-local inventory for a keycard.
func (g *Game) HasKeycardNamed(name string) bool {
	if g == nil || name == "" {
		return false
	}
	if g.HasRunKeycard(name) {
		return true
	}
	found := false
	g.OwnedItems.Each(func(item *world.Item) {
		if item != nil && item.Name == name {
			found = true
		}
	})
	return found
}

// IsRepairCompleteOnAnyDeck reports whether a repair ID is complete on the active deck
// or in any saved deck state.
func (g *Game) IsRepairCompleteOnAnyDeck(repairID string) bool {
	if g == nil || repairID == "" {
		return false
	}
	for _, repair := range g.RepairObjectives {
		if repair != nil && repair.ID == repairID && repair.IsComplete() {
			return true
		}
	}
	for _, ds := range g.DeckStates {
		if ds == nil {
			continue
		}
		for _, repair := range ds.RepairObjectives {
			if repair != nil && repair.ID == repairID && repair.IsComplete() {
				return true
			}
		}
	}
	return false
}

// InitRunUnlocks seeds themes, unlock plan, and starting routing for a new run.
func (g *Game) InitRunUnlocks(runSeed int64) {
	if g == nil {
		return
	}
	g.RunSeed = runSeed
	g.DeckThemes = deck.AssignThemes(runSeed)
	g.UnlockPlan = unlocks.BuildUnlockPlan(runSeed, g.DeckThemes)
	g.UnlockSatisfied = make(map[string]bool)
	g.LiftRoutingPowered = unlocks.InitialLiftRouting()
	g.ReactorOnline = false
}

// SetReactorOnline marks reactor control as operational for downstream deck gates.
func (g *Game) SetReactorOnline(online bool) {
	if g == nil {
		return
	}
	g.ReactorOnline = online
	if !online || g.UnlockPlan == nil {
		return
	}
	for _, req := range g.UnlockPlan.Requirements {
		if req.Kind == unlocks.KindReactorOnline {
			g.MarkUnlockSatisfied(req.ID)
		}
	}
}

// MarkUnlockSatisfied records a satisfied requirement by ID.
func (g *Game) MarkUnlockSatisfied(id string) {
	if g == nil || id == "" {
		return
	}
	if g.UnlockSatisfied == nil {
		g.UnlockSatisfied = make(map[string]bool)
	}
	g.UnlockSatisfied[id] = true
}

// OnRoutingRepairComplete enables lift routing for the target deck linked to repairID.
func (g *Game) OnRoutingRepairComplete(repairID string) {
	if g == nil || g.UnlockPlan == nil || repairID == "" {
		return
	}
	for _, req := range g.UnlockPlan.Requirements {
		if req.RepairID != repairID {
			continue
		}
		if g.LiftRoutingPowered == nil {
			g.LiftRoutingPowered = make(map[int]bool)
		}
		g.LiftRoutingPowered[req.TargetDeckID] = true
		g.MarkUnlockSatisfied(req.ID)
	}
}

// OnRunKeycardAcquired updates unlock flags when a keycard enters run inventory.
func (g *Game) OnRunKeycardAcquired(keycardName string) {
	if g == nil || g.UnlockPlan == nil || keycardName == "" {
		return
	}
	for _, req := range g.UnlockPlan.Requirements {
		if req.Kind != unlocks.KindSecurityKeycard || req.KeycardName != keycardName {
			continue
		}
		if g.HasRunKeycard(keycardName) {
			g.MarkUnlockSatisfied(req.ID)
		}
	}
}

// AddRunKeycard stores a keycard in run-wide inventory.
func (g *Game) AddRunKeycard(item *world.Item) {
	if g == nil || item == nil {
		return
	}
	g.RunInventory.Put(item)
	g.OnRunKeycardAcquired(item.Name)
}

// MaybeSetReactorOnlineFromDeck checks deck 5 completion and sets ReactorOnline.
func (g *Game) MaybeSetReactorOnlineFromDeck() {
	if g == nil || g.ReactorOnline || g.CurrentDeckID != 4 {
		return
	}
	for _, repair := range g.RepairObjectives {
		if repair == nil || repair.SkipExitGate || repair.IsComplete() {
			continue
		}
		return
	}
	if g.IncompleteRepairCount() > 0 {
		return
	}
	g.SetReactorOnline(true)
}
