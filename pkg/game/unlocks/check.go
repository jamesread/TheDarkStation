package unlocks

import (
	"fmt"

	"darkstation/pkg/game/deck"
)

// RunProgress holds run-wide unlock progress passed into check helpers.
type RunProgress struct {
	Plan               *Plan
	Satisfied          map[string]bool
	LiftRoutingPowered map[int]bool
	ReactorOnline      bool
	DeckThemes         map[int]deck.Theme
	HasKeycard         func(name string) bool
	RepairComplete     func(repairID string) bool
}

// IsRequirementSatisfied checks run-wide progress for one requirement.
func IsRequirementSatisfied(p RunProgress, req Requirement) bool {
	if p.Satisfied != nil && p.Satisfied[req.ID] {
		return true
	}
	switch req.Kind {
	case KindSecurityKeycard:
		return p.HasKeycard != nil && p.HasKeycard(req.KeycardName)
	case KindRoutingRepair:
		if p.LiftRoutingPowered != nil && p.LiftRoutingPowered[req.TargetDeckID] {
			return true
		}
		return p.RepairComplete != nil && p.RepairComplete(req.RepairID)
	case KindReactorOnline:
		return p.ReactorOnline
	default:
		return false
	}
}

// IsDeckTravelUnlocked reports whether the lift menu may travel to deckID.
func IsDeckTravelUnlocked(p RunProgress, deckID int) bool {
	if deckID < 0 || deckID >= deck.TotalDecks {
		return false
	}
	if IsDeckAlwaysReachable(deckID) {
		return true
	}
	if p.LiftRoutingPowered == nil || !p.LiftRoutingPowered[deckID] {
		return false
	}
	if p.Plan == nil {
		return false
	}
	for _, req := range p.Plan.ForTarget(deckID) {
		if !IsRequirementSatisfied(p, req) {
			return false
		}
	}
	return true
}

// DeckTravelBlockReason returns a short reason when travel to deckID is locked, or "" if unlocked.
func DeckTravelBlockReason(p RunProgress, deckID int) string {
	if IsDeckTravelUnlocked(p, deckID) {
		return ""
	}
	if deckID < 0 || deckID >= deck.TotalDecks {
		return "Invalid deck"
	}
	if IsDeckAlwaysReachable(deckID) {
		return ""
	}
	if p.LiftRoutingPowered == nil || !p.LiftRoutingPowered[deckID] {
		return "Lift routing offline"
	}
	if p.Plan == nil {
		return "Routing table unavailable"
	}
	for _, req := range p.Plan.ForTarget(deckID) {
		if IsRequirementSatisfied(p, req) {
			continue
		}
		switch req.Kind {
		case KindSecurityKeycard:
			return fmt.Sprintf("Needs: %s", req.KeycardName)
		case KindRoutingRepair:
			return fmt.Sprintf("Needs: routing repair on deck %d", req.SourceDeckID+1)
		case KindReactorOnline:
			return "Needs: Reactor Control online"
		default:
			return "Requirements incomplete"
		}
	}
	return "Requirements incomplete"
}

// RoutingRepairName returns the display name for a routing repair objective.
func RoutingRepairName(targetLevel int) string {
	return fmt.Sprintf("Lift Routing Coupler (Deck %d)", targetLevel)
}
