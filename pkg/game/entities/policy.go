package entities

import "fmt"

// PolicyKind classifies a deck conservation policy (station automation rule).
type PolicyKind string

const (
	// PolicyShedFirst: under overload, the target room's loads are shed before any other.
	PolicyShedFirst PolicyKind = "shed_first"
	// PolicyEgressSeal: manual egress releases on unpowered rooms re-seal after a delay.
	PolicyEgressSeal PolicyKind = "egress_seal"
)

// ConservationPolicy is a deterministic, legible station automation rule. The
// station never acts randomly: the player can read the rule at any maintenance
// terminal, plan around it, or deprecate it with a Crew Override Authorization.
type ConservationPolicy struct {
	ID         string
	Code       string // diegetic short code, e.g. "HAB-PRI", "ATMOS-SEAL"
	Kind       PolicyKind
	TargetRoom string // shed_first: room whose loads shed first
	DelayMs    int64  // egress_seal: ms until a manual release re-seals
	Overridden bool   // deprecated via Crew Override Authorization (permanent)
}

// Active reports whether the policy is still enforced.
func (p *ConservationPolicy) Active() bool {
	return p != nil && !p.Overridden
}

// RuleText is the player-readable rule, shown at maintenance terminals.
func (p *ConservationPolicy) RuleText() string {
	if p == nil {
		return ""
	}
	switch p.Kind {
	case PolicyShedFirst:
		return fmt.Sprintf("under overload, sheds %s loads first", p.TargetRoom)
	case PolicyEgressSeal:
		return fmt.Sprintf("manual egress releases re-seal after %ds without power", p.DelayMs/1000)
	default:
		return "unknown automation rule"
	}
}

// CrewOverrideItemName is the inventory item that deprecates deck policies.
const CrewOverrideItemName = "Crew Override Authorization"
