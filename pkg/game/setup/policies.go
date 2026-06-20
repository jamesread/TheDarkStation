package setup

import (
	"sort"

	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
)

// ShedFirstRooms returns rooms targeted by active shed-first policies: under
// overload these rooms lose power before any other consumer.
func ShedFirstRooms(g *state.Game) map[string]bool {
	out := map[string]bool{}
	if g == nil {
		return out
	}
	for _, p := range g.Policies {
		if p.Active() && p.Kind == entities.PolicyShedFirst && p.TargetRoom != "" {
			out[p.TargetRoom] = true
		}
	}
	return out
}

// AdvanceEgressSeal enforces active egress-seal policies: a manual door release
// on a room that still has no live power re-seals DelayMs after it was pulled.
// Returns the rooms that were re-sealed this tick (for player messaging).
func AdvanceEgressSeal(g *state.Game, nowMs int64) []string {
	if g == nil || len(g.ManualEgressReleased) == 0 {
		return nil
	}
	var policy *entities.ConservationPolicy
	for _, p := range g.Policies {
		if p.Active() && p.Kind == entities.PolicyEgressSeal {
			policy = p
			break
		}
	}
	if policy == nil {
		return nil
	}
	var sealed []string
	for roomName, released := range g.ManualEgressReleased {
		if !released {
			continue
		}
		releasedAt, ok := g.ManualEgressReleasedAtMs[roomName]
		if !ok {
			// Releases without a timestamp predate the policy system; never re-seal them.
			continue
		}
		if nowMs < releasedAt+policy.DelayMs {
			continue
		}
		if RoomIsOnline(g, roomName) || RoomHasLivePower(g, roomName) {
			// Properly powered rooms keep their doors energized; the release is moot.
			delete(g.ManualEgressReleasedAtMs, roomName)
			continue
		}
		delete(g.ManualEgressReleased, roomName)
		delete(g.ManualEgressReleasedAtMs, roomName)
		sealed = append(sealed, roomName)
	}
	sort.Strings(sealed)
	return sealed
}

// OverrideDeckPolicies deprecates every active policy on the current deck.
// Returns the number of policies overridden.
func OverrideDeckPolicies(g *state.Game) int {
	if g == nil {
		return 0
	}
	n := 0
	for _, p := range g.Policies {
		if p.Active() {
			p.Overridden = true
			n++
		}
	}
	return n
}

// ActivePolicyCount reports how many policies are still enforced on this deck.
func ActivePolicyCount(g *state.Game) int {
	if g == nil {
		return 0
	}
	n := 0
	for _, p := range g.Policies {
		if p.Active() {
			n++
		}
	}
	return n
}
