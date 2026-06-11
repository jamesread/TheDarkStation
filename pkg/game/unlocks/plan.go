// Package unlocks defines procedural deck travel unlocks for a run.
package unlocks

import (
	"math/rand"
	"sort"

	"darkstation/pkg/game/deck"
)

// Kind is how a deck travel requirement is satisfied.
type Kind int

const (
	KindRoutingRepair Kind = iota
	KindSecurityKeycard
	KindReactorOnline
)

// Requirement is one condition that must be met before a target deck is reachable via the lift.
type Requirement struct {
	ID           string
	TargetDeckID int // 0-based deck index (level = TargetDeckID + 1)
	Kind         Kind
	SourceDeckID int // deck where the objective or keycard is placed
	KeycardName  string
	RepairID     string
}

// Plan is the full unlock graph for one run, derived from RunSeed.
type Plan struct {
	RunSeed      int64
	Requirements []Requirement
}

// IsDeckAlwaysReachable reports decks 1–2 (IDs 0–1) unlocked at run start.
func IsDeckAlwaysReachable(deckID int) bool {
	return deck.IsDeckAlwaysReachable(deckID)
}

func requirementCount(level int) int {
	switch {
	case level <= 3:
		return 1
	case level <= 5:
		return 1 + (level-3)/2
	case level <= 7:
		return 2
	case level == 8:
		return 2
	case level == 9:
		return 3
	default: // level 10
		return 2
	}
}

func pickSourceDeck(rng *rand.Rand, targetID int, use map[int]int) int {
	if targetID <= 0 {
		return 0
	}
	candidates := make([]int, 0, targetID)
	for d := 0; d < targetID; d++ {
		candidates = append(candidates, d)
	}
	sort.Slice(candidates, func(i, j int) bool {
		ui, uj := use[candidates[i]], use[candidates[j]]
		if ui != uj {
			return ui < uj
		}
		return candidates[i] < candidates[j]
	})
	// Weight toward less-used sources with some randomness.
	window := candidates
	if len(window) > 3 {
		window = window[:3]
	}
	return window[rng.Intn(len(window))]
}

// ForSource returns requirements whose objective is placed on sourceDeckID.
func (p *Plan) ForSource(sourceDeckID int) []Requirement {
	if p == nil {
		return nil
	}
	var out []Requirement
	for _, req := range p.Requirements {
		if req.SourceDeckID == sourceDeckID {
			out = append(out, req)
		}
	}
	return out
}

// ForTarget returns requirements gating travel to targetDeckID.
func (p *Plan) ForTarget(targetDeckID int) []Requirement {
	if p == nil {
		return nil
	}
	var out []Requirement
	for _, req := range p.Requirements {
		if req.TargetDeckID == targetDeckID {
			out = append(out, req)
		}
	}
	return out
}

// InitialLiftRouting returns deck IDs with lift routing powered at run start (decks 1–2).
func InitialLiftRouting() map[int]bool {
	m := make(map[int]bool, 2)
	m[0] = true
	m[1] = true
	return m
}
