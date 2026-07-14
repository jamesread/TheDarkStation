// Package gamemode defines playable game modes: deck count, unlock rules, and item placement prefs.
package gamemode

// ID identifies a game mode configuration.
type ID string

const (
	// SinglePlayerPuzzle is the full 10-deck station run (default).
	SinglePlayerPuzzle ID = "SinglePlayerPuzzle"
	// SingleDeckSandbox is a single-deck layout for experiments and quick tests.
	SingleDeckSandbox ID = "SingleDeckSandbox"
	// FindTheBatteries is a single large deck: collect scattered batteries and power the generator.
	FindTheBatteries ID = "FindTheBatteries"
)

// ItemPlacementPrefs controls procedural item placement for a mode.
type ItemPlacementPrefs struct {
	// PlaceFloorBatteries places batteries on the floor to meet generator demand (level 3+).
	PlaceFloorBatteries bool
	// ExtraBatteryMin and ExtraBatteryMax are inclusive bounds for spare batteries beyond demand.
	ExtraBatteryMin int
	ExtraBatteryMax int
	// HideItemsInFurniture moves puzzle items from the floor into room furniture.
	HideItemsInFurniture bool
	// HideInFurnitureChancePct is the percent chance (0–100) to hide each eligible floor item.
	HideInFurnitureChancePct int
	// PlaceUnlockObjectives adds cross-deck routing repairs and keycard payoffs.
	PlaceUnlockObjectives bool
	// PlaceConservationPolicies seeds deck automation rules and crew override loot (deck 4+).
	PlaceConservationPolicies bool
	// PlaceHazardSolutionItems drops items required to clear environmental hazards.
	PlaceHazardSolutionItems bool
}

// LevelGenPrefs controls which systems are generated on each deck.
type LevelGenPrefs struct {
	// PlayRows and PlayCols override playable interior size (zero = default deck sizing).
	PlayRows int
	PlayCols int
	// LayoutLevel drives BSP split density when PlayRows/PlayCols are zero (zero = use deck level).
	LayoutLevel int
	PlaceDoors                bool
	PlaceHazards              bool
	PlacePuzzles              bool
	PlaceMaintenanceTerminals bool
	PlaceFurniture            bool
	PlaceRepairObjectives     bool
	PlaceCCTV                 bool
	PlaceConduitFaults        bool
	PlaceRelays               bool
	PlaceAdditionalGenerators bool
	BootstrapDeck1Ship        bool
	RunSimulateGate           bool
	// BatteryHunt uses a stripped layout: one unpowered generator and scattered floor batteries.
	BatteryHunt            bool
	BatteryHuntMinRequired int
	BatteryHuntMaxRequired int
}

// Mode is a complete game mode definition.
type Mode struct {
	ID                   ID
	DisplayName          string
	TotalDecks           int
	UsesCrossDeckUnlocks bool
	Items                ItemPlacementPrefs
	LevelGen             LevelGenPrefs
}

var registry = map[ID]Mode{
	SinglePlayerPuzzle: singlePlayerPuzzle(),
	SingleDeckSandbox:  singleDeckSandbox(),
	FindTheBatteries:   findTheBatteries(),
}

func defaultLevelGen() LevelGenPrefs {
	return LevelGenPrefs{
		PlaceDoors:                true,
		PlaceHazards:                true,
		PlacePuzzles:                true,
		PlaceMaintenanceTerminals:   true,
		PlaceFurniture:              true,
		PlaceRepairObjectives:       true,
		PlaceCCTV:                   true,
		PlaceConduitFaults:          true,
		PlaceRelays:                 true,
		PlaceAdditionalGenerators:   true,
		BootstrapDeck1Ship:          true,
		RunSimulateGate:             true,
	}
}

func singlePlayerPuzzle() Mode {
	return Mode{
		ID:                   SinglePlayerPuzzle,
		DisplayName:          "Single Player Puzzle",
		TotalDecks:           10,
		UsesCrossDeckUnlocks: true,
		Items: ItemPlacementPrefs{
			PlaceFloorBatteries:       true,
			ExtraBatteryMin:           1,
			ExtraBatteryMax:           2,
			HideItemsInFurniture:      true,
			HideInFurnitureChancePct:  50,
			PlaceUnlockObjectives:     true,
			PlaceConservationPolicies: true,
			PlaceHazardSolutionItems:  true,
		},
		LevelGen: defaultLevelGen(),
	}
}

func singleDeckSandbox() Mode {
	lg := defaultLevelGen()
	return Mode{
		ID:                   SingleDeckSandbox,
		DisplayName:          "Single Deck Sandbox",
		TotalDecks:           1,
		UsesCrossDeckUnlocks: false,
		Items: ItemPlacementPrefs{
			PlaceFloorBatteries:       true,
			ExtraBatteryMin:           0,
			ExtraBatteryMax:           1,
			HideItemsInFurniture:      true,
			HideInFurnitureChancePct:  40,
			PlaceUnlockObjectives:     false,
			PlaceConservationPolicies: false,
			PlaceHazardSolutionItems:  true,
		},
		LevelGen: lg,
	}
}

func findTheBatteries() Mode {
	return Mode{
		ID:                   FindTheBatteries,
		DisplayName:          "Find The Batteries",
		TotalDecks:           1,
		UsesCrossDeckUnlocks: false,
		Items: ItemPlacementPrefs{
			PlaceFloorBatteries:       false,
			HideItemsInFurniture:      false,
			PlaceUnlockObjectives:     false,
			PlaceConservationPolicies: false,
			PlaceHazardSolutionItems:  false,
		},
		LevelGen: LevelGenPrefs{
			LayoutLevel:               5, // largest standard bull-curve deck
			PlaceDoors:                false,
			PlaceHazards:              false,
			PlacePuzzles:              false,
			PlaceMaintenanceTerminals: false,
			PlaceFurniture:            false,
			PlaceRepairObjectives:     false,
			PlaceCCTV:                 false,
			PlaceConduitFaults:        false,
			PlaceRelays:               false,
			PlaceAdditionalGenerators: false,
			BootstrapDeck1Ship:        false,
			RunSimulateGate:           true,
			BatteryHunt:               true,
			BatteryHuntMinRequired:    5,
			BatteryHuntMaxRequired:    8,
		},
	}
}

// Get returns the mode for id, or SinglePlayerPuzzle when unknown.
func Get(id ID) Mode {
	if m, ok := registry[id]; ok {
		return m
	}
	return registry[SinglePlayerPuzzle]
}

// Default is the standard full-station mode.
func Default() Mode {
	return registry[SinglePlayerPuzzle]
}

// All returns every registered mode in stable ID order.
func All() []Mode {
	return []Mode{
		registry[SinglePlayerPuzzle],
		registry[SingleDeckSandbox],
		registry[FindTheBatteries],
	}
}

// ExtraBatteryRoll returns a random extra battery count within the prefs bounds.
func (p ItemPlacementPrefs) ExtraBatteryRoll(intn func(n int) int) int {
	if intn == nil {
		return p.ExtraBatteryMin
	}
	min, max := p.ExtraBatteryMin, p.ExtraBatteryMax
	if max < min {
		max = min
	}
	span := max - min + 1
	if span <= 1 {
		return min
	}
	return min + intn(span)
}

// BatteryHuntRequiredRoll returns how many batteries the generator needs in battery-hunt mode.
func (lg LevelGenPrefs) BatteryHuntRequiredRoll(intn func(n int) int) int {
	min, max := lg.BatteryHuntMinRequired, lg.BatteryHuntMaxRequired
	if max < min {
		max = min
	}
	if min <= 0 {
		min = 1
	}
	if max <= 0 {
		max = min
	}
	span := max - min + 1
	if span <= 1 || intn == nil {
		return min
	}
	return min + intn(span)
}
