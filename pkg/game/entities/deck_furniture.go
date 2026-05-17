package entities

import (
	"darkstation/pkg/game/deck"
)

// FurnitureFallbackForFunctionalLayer returns cold technical templates when RoomFurniture
// has no substring match for BSP thematic room names (Story 5.1 optional alignment).
func FurnitureFallbackForFunctionalLayer(ft deck.Type) []FurnitureTemplate {
	switch ft {
	case deck.Habitation:
		return []FurnitureTemplate{
			{Name: "Atmospheric scrubber manifold", Description: "Stamped VOID — seal integrity not monitored.", Icon: "⌂"},
			{Name: "Crew bunk rack", Description: "Allocation tags faded; bedding retained per regulation.", Icon: "▭"},
			{Name: "Pressure-equalisation panel", Description: "Manual bypass sealed — electronic latch offline.", Icon: "▣"},
		}
	case deck.Research:
		return []FurnitureTemplate{
			{Name: "Specimen locker", Description: "Biohazard marking obsolete; latch responds cold.", Icon: "▦"},
			{Name: "Analysis bench", Description: "Sample trays aligned; no active incubation.", Icon: "≡"},
			{Name: "Cold-chain cabinet", Description: "Compressor idle — temperature log stale.", Icon: "~"},
		}
	case deck.Logistics:
		return []FurnitureTemplate{
			{Name: "Cargo restraint anchor", Description: "Floor bolts sheared on one corner.", Icon: "⨂"},
			{Name: "Manifest terminal cradle", Description: "Network uplink absent; cradle powered-down.", Icon: "▤"},
			{Name: "Bulkhead pallet stack", Description: "Strapping intact; inventory references deleted routes.", Icon: "▥"},
		}
	case deck.PowerDistribution:
		return []FurnitureTemplate{
			{Name: "Phase bus bar housing", Description: "Warning placard: isolate upstream before tactile inspection.", Icon: "*"},
			{Name: "Relay calibration fixture", Description: "Torque markers corroded; pins seated.", Icon: "⌁"},
			{Name: "Capacitor bleed cage", Description: "Discharge lamps dark — assumed dormant.", Icon: "▯"},
		}
	case deck.EmergencySystems:
		return []FurnitureTemplate{
			{Name: "Beacon test harness", Description: "Annual drill acknowledgment filed — never retrieved.", Icon: "◉"},
			{Name: "Shelter ration locker", Description: "Seals nominal; contents expired per manifest.", Icon: "▪"},
			{Name: "Override pull station", Description: "Guard wire intact; lever friction high.", Icon: "!"},
		}
	case deck.CoreInfrastructure:
		return []FurnitureTemplate{
			{Name: "Spine conduit hatch", Description: "Keyed maintenance only — adjacent conduit tagged CORE.", Icon: "⌗"},
			{Name: "Monitoring relay shelf", Description: "Indicator legends mismatched with downstream feeds.", Icon: "◫"},
			{Name: "Audit tape spindle", Description: "Cartridge absent; spindle rotates freely.", Icon: "◎"},
		}
	default:
		return []FurnitureTemplate{
			{Name: "Generic service shelf", Description: "Residual fasteners only.", Icon: "╬"},
		}
	}
}
