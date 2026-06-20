package unlocks

import (
	"fmt"
	"math/rand"

	"darkstation/pkg/game/deck"
)

// BuildUnlockPlan creates a theme-aware unlock graph for a run.
func BuildUnlockPlan(runSeed int64, themes map[int]deck.Theme) *Plan {
	return BuildUnlockPlanFor(runSeed, themes, deck.TotalDecks)
}

// BuildUnlockPlanFor creates an unlock graph sized for totalDecks.
func BuildUnlockPlanFor(runSeed int64, themes map[int]deck.Theme, totalDecks int) *Plan {
	if totalDecks < 1 {
		totalDecks = 1
	}
	rng := rand.New(rand.NewSource(runSeed ^ 0x5851f42d4c957f2d))
	p := &Plan{RunSeed: runSeed}

	if totalDecks > 4 {
		addReactorAuthorizationChain(p, rng, themes)
	}
	addProceduralRequirements(p, rng, themes, totalDecks)
	addLifeSupportGates(p, themes)

	return p
}

func addReactorAuthorizationChain(p *Plan, rng *rand.Rand, themes map[int]deck.Theme) {
	const reactorDeckID = 4 // deck 5
	earlyCandidates := []int{1, 2, 3}
	rng.Shuffle(len(earlyCandidates), func(i, j int) {
		earlyCandidates[i], earlyCandidates[j] = earlyCandidates[j], earlyCandidates[i]
	})
	authCount := 2
	if authCount > len(earlyCandidates) {
		authCount = len(earlyCandidates)
	}
	for i := 0; i < authCount; i++ {
		sourceID := earlyCandidates[i]
		req := Requirement{
			ID:           fmt.Sprintf("reactor-auth-%d", sourceID),
			TargetDeckID: reactorDeckID,
			Kind:         KindSecurityKeycard,
			SourceDeckID: sourceID,
			KeycardName:  deck.ReactorAuthKeycardName(sourceID, themes),
		}
		p.Requirements = append(p.Requirements, req)
	}

	routingSource := reactorDeckID - 1 // deck 4 pays off routing to deck 5
	p.Requirements = append(p.Requirements, Requirement{
		ID:           fmt.Sprintf("routing-reactor-deck%d", reactorDeckID+1),
		TargetDeckID: reactorDeckID,
		Kind:         KindRoutingRepair,
		SourceDeckID: routingSource,
		RepairID:     fmt.Sprintf("routing-repair-deck%d-reactor", reactorDeckID+1),
	})
}

func addProceduralRequirements(p *Plan, rng *rand.Rand, themes map[int]deck.Theme, totalDecks int) {
	hasRouting := map[int]bool{}
	if totalDecks > 4 {
		hasRouting[4] = true // deck 5 from reactor chain
	}
	for targetLevel := 3; targetLevel <= totalDecks; targetLevel++ {
		targetID := targetLevel - 1
		if deck.IsDeckAlwaysReachable(targetID) {
			continue
		}
		numReqs := requirementCount(targetLevel)
		sourceUse := make(map[int]int)
		routingAdded := hasRouting[targetID]

		for i := 0; i < numReqs; i++ {
			kind := KindRoutingRepair
			if !routingAdded {
				kind = KindRoutingRepair
			} else if rng.Intn(100) < 40 {
				kind = KindSecurityKeycard
			}

			sourceID := pickSourceDeck(rng, targetID, sourceUse)
			sourceUse[sourceID]++

			req := Requirement{
				ID:           fmt.Sprintf("unlock-%d-%d", targetID, i),
				TargetDeckID: targetID,
				Kind:         kind,
				SourceDeckID: sourceID,
			}
			switch kind {
			case KindSecurityKeycard:
				req.KeycardName = fmt.Sprintf("Deck %d Access Keycard", targetLevel)
			case KindRoutingRepair:
				req.RepairID = fmt.Sprintf("routing-repair-deck%d-%d", targetLevel, i)
				hasRouting[targetID] = true
				routingAdded = true
			}
			if duplicateRepairID(p, req.RepairID) {
				continue
			}
			p.Requirements = append(p.Requirements, req)
			if kind == KindRoutingRepair {
				routingAdded = true
			}
		}
		if !hasRouting[targetID] {
			sourceID := pickSourceDeck(rng, targetID, sourceUse)
			req := Requirement{
				ID:           fmt.Sprintf("unlock-routing-%d", targetID),
				TargetDeckID: targetID,
				Kind:         KindRoutingRepair,
				SourceDeckID: sourceID,
				RepairID:     fmt.Sprintf("routing-repair-deck%d-fallback", targetLevel),
			}
			if !duplicateRepairID(p, req.RepairID) {
				p.Requirements = append(p.Requirements, req)
				hasRouting[targetID] = true
			}
		}
		_ = themes
	}
}

func duplicateRepairID(p *Plan, repairID string) bool {
	for _, req := range p.Requirements {
		if req.RepairID == repairID {
			return true
		}
	}
	return false
}

func addLifeSupportGates(p *Plan, themes map[int]deck.Theme) {
	for _, deckID := range deck.LifeSupportDeckIDs(themes) {
		if deckID <= 4 {
			continue
		}
		p.Requirements = append(p.Requirements, Requirement{
			ID:           fmt.Sprintf("life-support-reactor-%d", deckID),
			TargetDeckID: deckID,
			Kind:         KindReactorOnline,
			SourceDeckID: 4,
		})
	}
}

// Generate is deprecated; use BuildUnlockPlan with assigned themes.
func Generate(runSeed int64) *Plan {
	themes := deck.AssignThemes(runSeed)
	return BuildUnlockPlan(runSeed, themes)
}
