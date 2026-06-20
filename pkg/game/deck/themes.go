// Package deck: run-seeded deck themes for room naming and furniture.
package deck

import (
	"fmt"
	"math/rand"
	"sort"
)

// Theme identifies a deck's functional identity for naming and content.
type Theme string

const (
	ThemeAirlock          Theme = "airlock"
	ThemeReactorControl   Theme = "reactor_control"
	ThemeExitDeck         Theme = "exit_deck"
	ThemeHydroponics      Theme = "hydroponics"
	ThemeDormitories      Theme = "dormitories"
	ThemeLifeSupport      Theme = "life_support"
	ThemeCargoLogistics   Theme = "cargo_logistics"
	ThemeMedicalBay       Theme = "medical_bay"
	ThemeResearchLabs     Theme = "research_labs"
	ThemeCommunications   Theme = "communications"
	ThemeNavigation       Theme = "navigation"
	ThemeSanitation       Theme = "sanitation"
	ThemeWaterReclamation Theme = "water_reclamation"
	ThemeCryogenicStorage Theme = "cryogenic_storage"
	ThemeManufacturing    Theme = "manufacturing"
	ThemeObservatory      Theme = "observatory"
	ThemeSecurityArmory   Theme = "security_armory"
	ThemeDataArchive      Theme = "data_archive"
	ThemeMessHall         Theme = "mess_hall"
	ThemeRecreation       Theme = "recreation"
	ThemeEVAMaintenance   Theme = "eva_maintenance"
	ThemeDockingRing      Theme = "docking_ring"
	ThemeChemicalProcess  Theme = "chemical_processing"
	ThemeParticlePhysics  Theme = "particle_physics"
	ThemeThermalReg       Theme = "thermal_regulation"
	ThemeAtmosphericProc  Theme = "atmospheric_processing"
)

// assignableThemes are shuffled onto decks 2–4 and 6–9 (Life Support post-deck-5 only).
var assignableThemes = []Theme{
	ThemeHydroponics,
	ThemeDormitories,
	ThemeLifeSupport,
	ThemeCargoLogistics,
	ThemeMedicalBay,
	ThemeResearchLabs,
	ThemeCommunications,
	ThemeNavigation,
	ThemeSanitation,
	ThemeWaterReclamation,
	ThemeCryogenicStorage,
	ThemeManufacturing,
	ThemeObservatory,
	ThemeSecurityArmory,
	ThemeDataArchive,
	ThemeMessHall,
	ThemeRecreation,
	ThemeEVAMaintenance,
	ThemeDockingRing,
	ThemeChemicalProcess,
	ThemeParticlePhysics,
	ThemeThermalReg,
	ThemeAtmosphericProc,
}

// IsDeckAlwaysReachable reports decks unlocked at run start (airlock + first significant deck).
func IsDeckAlwaysReachable(deckID int) bool {
	return deckID == 0 || deckID == 1
}

// ThemeDisplayName returns a human-readable deck theme label.
func ThemeDisplayName(t Theme) string {
	switch t {
	case ThemeAirlock:
		return "Airlock"
	case ThemeReactorControl:
		return "Reactor Control"
	case ThemeExitDeck:
		return "Exit Deck"
	case ThemeHydroponics:
		return "Hydroponics"
	case ThemeDormitories:
		return "Dormitories"
	case ThemeLifeSupport:
		return "Life Support"
	case ThemeCargoLogistics:
		return "Cargo & Logistics"
	case ThemeMedicalBay:
		return "Medical Bay"
	case ThemeResearchLabs:
		return "Research Laboratories"
	case ThemeCommunications:
		return "Communications Array"
	case ThemeNavigation:
		return "Navigation"
	case ThemeSanitation:
		return "Sanitation & Waste Processing"
	case ThemeWaterReclamation:
		return "Water Reclamation"
	case ThemeCryogenicStorage:
		return "Cryogenic Storage"
	case ThemeManufacturing:
		return "Manufacturing Bay"
	case ThemeObservatory:
		return "Observatory"
	case ThemeSecurityArmory:
		return "Security & Armory"
	case ThemeDataArchive:
		return "Data Archive"
	case ThemeMessHall:
		return "Mess Hall"
	case ThemeRecreation:
		return "Recreation Commons"
	case ThemeEVAMaintenance:
		return "EVA Suit Maintenance"
	case ThemeDockingRing:
		return "Docking Ring"
	case ThemeChemicalProcess:
		return "Chemical Processing"
	case ThemeParticlePhysics:
		return "Particle Physics"
	case ThemeThermalReg:
		return "Thermal Regulation"
	case ThemeAtmosphericProc:
		return "Atmospheric Processing"
	default:
		return string(t)
	}
}

// IsLifeSupportTheme reports whether the theme is Life Support.
func IsLifeSupportTheme(t Theme) bool {
	return t == ThemeLifeSupport
}

// AssignThemes returns the theme for each deck ID (0-based) for a run seed.
func AssignThemes(runSeed int64) map[int]Theme {
	return AssignThemesFor(runSeed, TotalDecks)
}

// AssignThemesFor returns deck themes for a run with the given deck count.
func AssignThemesFor(runSeed int64, totalDecks int) map[int]Theme {
	if totalDecks < 1 {
		totalDecks = 1
	}
	themes := make(map[int]Theme, totalDecks)
	themes[0] = ThemeAirlock
	finalIdx := FinalDeckIndexFor(totalDecks)
	if finalIdx > 0 {
		themes[finalIdx] = ThemeExitDeck
	}
	if totalDecks > 4 {
		themes[4] = ThemeReactorControl // deck 5
	}

	assignable := make([]Theme, 0, len(assignableThemes))
	for _, t := range assignableThemes {
		if t == ThemeLifeSupport {
			continue
		}
		assignable = append(assignable, t)
	}
	rng := rand.New(rand.NewSource(runSeed))

	earlySlots := filterDeckSlots([]int{1, 2, 3}, totalDecks)
	lateSlots := filterDeckSlots([]int{5, 6, 7, 8}, totalDecks)

	shuffleThemes(rng, assignable)
	earlyPool := append([]Theme(nil), assignable...)
	if len(earlyPool) > len(earlySlots) {
		earlyPool = earlyPool[:len(earlySlots)]
	}
	for i, id := range earlySlots {
		if i < len(earlyPool) {
			themes[id] = earlyPool[i]
		}
	}

	remaining := assignable[len(earlyPool):]
	remaining = append(remaining, ThemeLifeSupport)
	shuffleThemes(rng, remaining)
	latePool := remaining
	if len(latePool) > len(lateSlots) {
		latePool = latePool[:len(lateSlots)]
	}
	for i, id := range lateSlots {
		if i < len(latePool) {
			themes[id] = latePool[i]
		}
	}

	for id := 0; id < totalDecks; id++ {
		if themes[id] == "" {
			themes[id] = ThemeCargoLogistics
		}
	}
	return themes
}

func filterDeckSlots(slots []int, totalDecks int) []int {
	var out []int
	for _, id := range slots {
		if id >= 0 && id < totalDecks {
			out = append(out, id)
		}
	}
	return out
}

func shuffleThemes(rng *rand.Rand, themes []Theme) {
	rng.Shuffle(len(themes), func(i, j int) { themes[i], themes[j] = themes[j], themes[i] })
}

// ThemeForDeckID returns the assigned theme for a deck ID from a theme map.
func ThemeForDeckID(themes map[int]Theme, deckID int) Theme {
	if themes == nil {
		return ThemeAirlock
	}
	if t, ok := themes[deckID]; ok && t != "" {
		return t
	}
	return ThemeCargoLogistics
}

// ReactorAuthKeycardName returns the keycard name for reactor authorization from a source deck.
func ReactorAuthKeycardName(sourceDeckID int, themes map[int]Theme) string {
	return fmt.Sprintf("Reactor Authorization — %s", ThemeDisplayName(ThemeForDeckID(themes, sourceDeckID)))
}

// RoomNamesForTheme returns room base names and adjectives for BSP generation.
func RoomNamesForTheme(t Theme) (bases []string, adjectives []string) {
	adjectives = []string{
		"Abandoned", "Damaged", "Derelict", "Emergency",
		"Isolated", "Sealed", "Depressurized", "Overgrown",
	}
	switch t {
	case ThemeAirlock:
		bases = []string{
			"Transit Airlock", "Docking Seal", "Pressure Vestibule", "Egress Chamber",
			"External Seal Gate", "Entry Lock", "Depressurization Bay",
		}
	case ThemeReactorControl:
		bases = []string{
			"Primary Reactor Bay", "Containment Monitor", "Plasma Regulator", "Core Control Room",
			"Fuel Rod Handling", "Reactor Startup Console", "Neutron Flux Monitor", "Scram Override",
		}
	case ThemeExitDeck:
		bases = []string{
			"Terminal Vestibule", "Final Relay", "Shutdown Annex", "End-of-Line Monitor",
		}
	case ThemeHydroponics:
		bases = []string{
			"Nutrient Synthesis Bay", "Grow Light Array", "Irrigation Manifold", "Crop Storage",
			"Algae Cultivation Tank", "Hydroponic Rack",
		}
	case ThemeDormitories:
		bases = []string{
			"Crew Rest Module", "Personnel Bay", "Dormitory Section", "Rest Bay",
			"Cryogenic Habitation Block", "Quarantine Dorm", "Crew Quarters",
		}
	case ThemeLifeSupport:
		bases = []string{
			"Atmospheric Recycling Annex", "Oxygen Generation Module", "CO2 Scrubber Bank",
			"Life Support Monitor", "Breathable Atmosphere Control", "Emergency Life Support",
		}
	case ThemeCargoLogistics:
		bases = []string{
			"Cargo Hold", "Loading Bay", "Supply Locker", "Distribution Hub",
			"Storage Annex", "Parts Depot", "Logistics Node",
		}
	case ThemeMedicalBay:
		bases = []string{
			"Emergency Medical Wing", "Med Bay", "Diagnostics Suite", "Surgical Prep",
			"Specimen Holding", "Triage Station",
		}
	case ThemeResearchLabs:
		bases = []string{
			"Sample Analysis Lab", "Experiment Bay", "Culture Lab", "Data Review Chamber",
			"Organic Nutrient Synthesis", "Research Module",
		}
	case ThemeCommunications:
		bases = []string{
			"Signal Relay Room", "Antenna Control", "Transmission Hub", "Comm Array Monitor",
		}
	case ThemeNavigation:
		bases = []string{
			"Astrogation Plot Room", "Star Tracker Bay", "Course Correction Node", "Nav Console",
		}
	case ThemeSanitation:
		bases = []string{
			"Waste Processing Bay", "Recycling Station", "Contaminant Filter", "Sludge Pump Room",
		}
	case ThemeWaterReclamation:
		bases = []string{
			"Water Purification Node", "Reclamation Tank", "Condensate Recovery", "Potable Feed",
		}
	case ThemeCryogenicStorage:
		bases = []string{
			"Cryo Vault", "Cold Storage Lock", "Preservation Bay", "Thermal Isolation Chamber",
		}
	case ThemeManufacturing:
		bases = []string{
			"Fabrication Bay", "Assembly Line Node", "Tool Crib", "Parts Fabricator",
		}
	case ThemeObservatory:
		bases = []string{
			"Sensor Dome Control", "Telescope Relay", "Observation Deck", "Spectral Analysis",
		}
	case ThemeSecurityArmory:
		bases = []string{
			"Security Checkpoint", "Armory Vault", "Access Control Node", "Perimeter Monitor",
		}
	case ThemeDataArchive:
		bases = []string{
			"Server Room", "Archive Vault", "Data Review Chamber", "Tape Library",
		}
	case ThemeMessHall:
		bases = []string{
			"Galley Prep", "Mess Hall", "Food Synthesizer Bay", "Provisions Storage",
		}
	case ThemeRecreation:
		bases = []string{
			"Recreation Commons", "Exercise Module", "Lounge Bay", "Holodeck Stub",
		}
	case ThemeEVAMaintenance:
		bases = []string{
			"Suit Maintenance Bay", "EVA Prep Lock", "Tool Seal Check", "External Gear Store",
		}
	case ThemeDockingRing:
		bases = []string{
			"Docking Clamps Control", "Berth Monitor", "External Hatch Relay", "Ring Junction",
		}
	case ThemeChemicalProcess:
		bases = []string{
			"Chemical Reactor Bay", "Solvent Storage", "Catalyst Handling", "Process Monitor",
		}
	case ThemeParticlePhysics:
		bases = []string{
			"Collider Monitor", "Particle Trap", "Beam Alignment Room", "Detector Array",
		}
	case ThemeThermalReg:
		bases = []string{
			"Heat Exchange Node", "Coolant Manifold", "Thermal Monitor", "Radiator Control",
		}
	case ThemeAtmosphericProc:
		bases = []string{
			"Atmospheric Mixing Bay", "Pressure Equalization", "Gas Separation Node", "Vent Control",
		}
	default:
		bases, adjectives = RoomNamesForType(Habitation)
	}
	return bases, adjectives
}

// PlaqueLayer maps a deck theme to the environmental signage functional layer.
func PlaqueLayer(t Theme) Type {
	switch t {
	case ThemeAirlock, ThemeDormitories, ThemeMessHall, ThemeRecreation, ThemeEVAMaintenance, ThemeDockingRing:
		return Habitation
	case ThemeResearchLabs, ThemeMedicalBay, ThemeParticlePhysics, ThemeObservatory, ThemeCryogenicStorage:
		return Research
	case ThemeCargoLogistics, ThemeManufacturing, ThemeChemicalProcess:
		return Logistics
	case ThemeReactorControl, ThemeThermalReg, ThemeNavigation:
		return PowerDistribution
	case ThemeExitDeck, ThemeSecurityArmory:
		return EmergencySystems
	default:
		return CoreInfrastructure
	}
}

// ThemeAbbrev returns a short diagnostic label for maintenance instrument traces.
func ThemeAbbrev(t Theme) string {
	switch t {
	case ThemeAirlock:
		return "AIR"
	case ThemeReactorControl:
		return "RCT"
	case ThemeExitDeck:
		return "EXT"
	case ThemeLifeSupport:
		return "LIF"
	case ThemeHydroponics:
		return "HYD"
	case ThemeDormitories:
		return "DRM"
	case ThemeCargoLogistics:
		return "LOG"
	case ThemeMedicalBay:
		return "MED"
	case ThemeResearchLabs:
		return "RES"
	case ThemeCommunications:
		return "COM"
	case ThemeNavigation:
		return "NAV"
	case ThemeSanitation:
		return "SAN"
	case ThemeWaterReclamation:
		return "H2O"
	case ThemeCryogenicStorage:
		return "CRY"
	case ThemeManufacturing:
		return "MFG"
	case ThemeObservatory:
		return "OBS"
	case ThemeSecurityArmory:
		return "SEC"
	case ThemeDataArchive:
		return "DAT"
	case ThemeMessHall:
		return "MES"
	case ThemeRecreation:
		return "REC"
	case ThemeEVAMaintenance:
		return "EVA"
	case ThemeDockingRing:
		return "DOC"
	case ThemeChemicalProcess:
		return "CHM"
	case ThemeParticlePhysics:
		return "PHY"
	case ThemeThermalReg:
		return "THM"
	case ThemeAtmosphericProc:
		return "ATM"
	default:
		return "SYS"
	}
}

// LifeSupportDeckIDs returns deck IDs assigned Life Support in themes.
func LifeSupportDeckIDs(themes map[int]Theme) []int {
	if themes == nil {
		return nil
	}
	var ids []int
	for id, t := range themes {
		if IsLifeSupportTheme(t) {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	return ids
}
