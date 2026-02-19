// Package deck defines the fixed deck count, final deck, and functional layer
// types for the station. The player never sees the total; they discover the end
// by reaching the final deck.
package deck

import (
	"github.com/leonelquinteros/gotext"
)

// Type is the functional layer type of a deck (GDD §3.1).
type Type int

const (
	Habitation       Type = iota // Crew rest, atmosphere, habitation
	Research                    // Labs, medical, experiments
	Logistics                   // Cargo, storage, distribution
	PowerDistribution           // Substations, relays, grid
	EmergencySystems            // Shelters, lifeboats, crisis
	CoreInfrastructure          // Central monitoring, primary conduits
)

// typeCount is the number of functional layer types (for cycling).
const typeCount = 6

// FunctionalType returns the functional layer type for the given deck level (1-based).
// Types cycle so each deck has an identity; final deck uses (TotalDecks-1) % typeCount.
func FunctionalType(level int) Type {
	if level <= 0 {
		return Habitation
	}
	deckIndex := level - 1 // 0-based
	return Type(deckIndex % typeCount)
}

// TotalDecks is the fixed number of decks on the station (never shown to player).
const TotalDecks = 10

// FinalDeckIndex is the deck index of the final deck (0-based; last deck).
const FinalDeckIndex = TotalDecks - 1

// IsFinalDeck returns true if the given level (1-based) is the final deck.
// Level 1 = first deck, Level TotalDecks = final deck.
func IsFinalDeck(level int) bool {
	return level >= TotalDecks
}

// NextDeckLevel returns the next deck level (1-based) for the given current level,
// or 0 if there is no next deck (current is final).
func NextDeckLevel(currentLevel int) int {
	if currentLevel <= 0 || currentLevel >= TotalDecks {
		return 0
	}
	return currentLevel + 1
}

// Descriptor describes one deck in the graph (GDD §4.1).
type Descriptor struct {
	ID          int   // 0-based deck index
	Type        Type  // Functional layer type
	Connections []int // Deck IDs reachable from this deck (e.g. next deck(s))
	Depth       int   // Depth for ordering/decay (0 = start, FinalDeckIndex = deepest)
}

// Graph is the deck graph: linear path 0 → 1 → … → FinalDeckIndex (Phase 3.1).
// Index is deck ID (0-based). Final deck has empty Connections.
var Graph []Descriptor

func init() {
	Graph = make([]Descriptor, TotalDecks)
	for i := 0; i < TotalDecks; i++ {
		conn := []int{}
		if i < FinalDeckIndex {
			conn = append(conn, i+1)
		}
		Graph[i] = Descriptor{
			ID:          i,
			Type:        FunctionalType(i + 1),
			Connections: conn,
			Depth:       i,
		}
	}
}

// NextDeckID returns the next deck ID (0-based) from the graph and true,
// or 0 and false if current is final or out of range (Phase 3.3).
func NextDeckID(deckID int) (nextID int, ok bool) {
	if deckID < 0 || deckID >= TotalDecks {
		return 0, false
	}
	conn := Graph[deckID].Connections
	if len(conn) == 0 {
		return 0, false
	}
	return conn[0], true
}

// DecayParams holds per-deck decay multipliers for power (GDD §9.1, Phase 4.1).
// Tuning (exact curves) is separate; this ensures the data path exists.
type DecayParams struct {
	GeneratorOutputMultiplier float64 // e.g. 1.0 on deck 1, lower on deeper decks
	PowerCostMultiplier       float64 // e.g. 1.0 on deck 1, higher on deeper decks
}

// DecayParamsForDeck returns decay parameters for the given deck ID (0-based).
// Deeper decks have reduced generator output and increased power costs (Phase 4.2).
func DecayParamsForDeck(deckID int) DecayParams {
	if deckID < 0 || deckID >= TotalDecks {
		return DecayParams{1.0, 1.0}
	}
	depth := float64(deckID)
	// Placeholder curve: output drops ~4% per deck, cost rises ~8% per deck (tuning later)
	outputMult := 1.0 - 0.04*depth
	if outputMult < 0.5 {
		outputMult = 0.5
	}
	costMult := 1.0 + 0.08*depth
	return DecayParams{GeneratorOutputMultiplier: outputMult, PowerCostMultiplier: costMult}
}

// TerminalFlavourKey returns the gettext message key for the maintenance terminal
// status line for the given deck ID (Phase 5.2). Later decks use more obsolete or
// contradictory lines (GDD §6).
func TerminalFlavourKey(deckID int) string {
	if deckID < 0 || deckID >= TotalDecks {
		return "TERMINAL_STATUS_NOMINAL"
	}
	// Bands: early (0-2), mid (3-5), late (6-8), final (9)
	switch {
	case deckID <= 2:
		return "TERMINAL_STATUS_NOMINAL"
	case deckID <= 5:
		return "TERMINAL_STATUS_LEGACY"
	case deckID <= 8:
		return "TERMINAL_STATUS_ANOMALY"
	default:
		return "TERMINAL_STATUS_FINAL"
	}
}

// TerminalFlavourText returns the translated flavour text for the maintenance terminal
// status line for the given deck ID. Uses gotext.Get with constant keys to satisfy vet.
func TerminalFlavourText(deckID int) string {
	switch TerminalFlavourKey(deckID) {
	case "TERMINAL_STATUS_LEGACY":
		return gotext.Get("TERMINAL_STATUS_LEGACY")
	case "TERMINAL_STATUS_ANOMALY":
		return gotext.Get("TERMINAL_STATUS_ANOMALY")
	case "TERMINAL_STATUS_FINAL":
		return gotext.Get("TERMINAL_STATUS_FINAL")
	default:
		return gotext.Get("TERMINAL_STATUS_NOMINAL")
	}
}

// RoomNamesForType returns thematic room base names and adjectives for the given
// functional type. Naming is cold, functional, slightly outdated (GDD §7).
func RoomNamesForType(t Type) (bases []string, adjectives []string) {
	adjectives = []string{
		"Abandoned", "Damaged", "Derelict", "Emergency",
		"Isolated", "Sealed", "Depressurized", "Overgrown",
	}
	switch t {
	case Habitation:
		bases = []string{
			"Cryogenic Habitation Block", "Atmospheric Recycling Annex", "Crew Rest Module",
			"Personnel Bay", "Dormitory Section", "Habitation Unit", "Rest Bay",
			"Recycling Station", "Quarantine Dorm", "Crew Quarters",
		}
	case Research:
		bases = []string{
			"Organic Nutrient Synthesis", "Emergency Medical Wing", "Sample Analysis Lab",
			"Experiment Bay", "Data Review Chamber", "Specimen Holding", "Med Bay",
			"Research Module", "Culture Lab", "Diagnostics Suite",
		}
	case Logistics:
		bases = []string{
			"Cargo Hold", "Loading Bay", "Supply Locker", "Distribution Hub",
			"Storage Annex", "Receiving", "Parts Depot", "Logistics Node",
			"Transfer Bay", "Inventory Control",
		}
	case PowerDistribution:
		bases = []string{
			"Substation", "Relay Room", "Capacitor Bank", "Distribution Node",
			"Grid Junction", "Power Conduit", "Transfer Station", "Regulator Bay",
			"Primary Feed", "Auxiliary Grid",
		}
	case EmergencySystems:
		bases = []string{
			"Emergency Shelter", "Lifeboat Bay", "Evacuation Assembly", "Crisis Command",
			"Backup Life Support", "Emergency Medical", "Shelter Module", "Escape Pod Bay",
			"Contingency Station", "Emergency Power",
		}
	case CoreInfrastructure:
		bases = []string{
			"Central Monitoring", "Primary Conduit", "Core Junction", "Station Spine",
			"Maintenance Conduit", "Command Node", "Core Access", "Primary Hub",
			"Infrastructure Node", "Control Conduit",
		}
	default:
		bases = []string{"Bridge", "Cargo Bay", "Engineering", "Med Bay", "Crew Quarters",
			"Airlock", "Server Room", "Reactor Core", "Armory", "Lab"}
	}
	return bases, adjectives
}
