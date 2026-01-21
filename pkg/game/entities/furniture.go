package entities

import (
	"darkstation/pkg/engine/world"
)

// Furniture represents a piece of furniture in a room
type Furniture struct {
	Name          string      // Display name
	Description   string      // Hint text shown when player is adjacent
	Icon          string      // Icon to display on the map
	Checked       bool        // Whether the player has examined this furniture
	ContainedItem *world.Item // Item hidden in this furniture (if any)
}

// NewFurniture creates a new furniture piece
func NewFurniture(name, description, icon string) *Furniture {
	return &Furniture{
		Name:        name,
		Description: description,
		Icon:        icon,
		Checked:     false,
	}
}

// Check marks the furniture as examined and returns any contained item
// Can be called multiple times, but will only return an item on the first call
// Subsequent calls return nil (item already taken)
func (f *Furniture) Check() *world.Item {
	f.Checked = true
	item := f.ContainedItem
	f.ContainedItem = nil // Clear item after first check to prevent duplicates
	return item
}

// HasItem returns true if this furniture contains an item
func (f *Furniture) HasItem() bool {
	return f.ContainedItem != nil
}

// IsChecked returns true if the furniture has been examined
func (f *Furniture) IsChecked() bool {
	return f.Checked
}

// FurnitureTemplate defines a furniture type that can be placed in rooms
type FurnitureTemplate struct {
	Name        string
	Description string
	Icon        string
}

// RoomFurniture contains space station themed furniture templates by room type
var RoomFurniture = map[string][]FurnitureTemplate{
	"Bridge": {
		{"Captain's Chair", "A worn command chair faces the main viewscreen.", "Ω"},
		{"Navigation Console", "Star charts flicker on a dusty display.", "≡"},
		{"Helm Station", "Manual flight controls, covered in emergency overrides.", "∩"},
	},
	"Command Center": {
		{"Tactical Display", "A holographic map table, now dark.", "◈"},
		{"Communications Array", "Banks of switches and indicator lights.", "≡"},
		{"Status Board", "Crew assignments, most names crossed out.", "▤"},
	},
	"Communications": {
		{"Radio Equipment", "Long-range transmitters, all frequencies silent.", "≋"},
		{"Signal Decoder", "Encrypted message logs scroll endlessly.", "≡"},
		{"Antenna Controls", "Dish alignment controls, slightly off-calibration.", "¥"},
	},
	"Security": {
		{"Weapons Locker", "Reinforced cabinet, lock has been forced.", "▥"},
		{"Monitoring Station", "Camera feeds cycle through empty corridors.", "◫"},
		{"Detention Cell", "A small holding area, door hanging open.", "▦"},
	},
	"Engineering": {
		{"Computer Console", "Diagnostic readouts scroll past warnings.", "≡"},
		{"Tool Rack", "Wrenches and plasma cutters, some missing.", "╦"},
		{"Schematic Display", "Station blueprints, several sections highlighted red.", "▤"},
	},
	"Reactor Core": {
		{"Control Rods", "Emergency dampeners, partially deployed.", "╫"},
		{"Coolant Pipes", "Thick tubes hum with circulating fluid.", "═"},
		{"Radiation Monitor", "Geiger counter clicks occasionally.", "☢"},
	},
	"Server Room": {
		{"Server Rack", "Blinking lights indicate partial functionality.", "▥"},
		{"Terminal Bank", "Multiple screens display scrolling logs.", "≡"},
		{"Cooling Unit", "Industrial fans spin slowly.", "※"},
	},
	"Maintenance Bay": {
		{"Workbench", "Scattered parts and half-finished repairs.", "╤"},
		{"Parts Bin", "Salvaged components, poorly organized.", "▤"},
		{"Diagnostic Station", "Equipment testing rig, currently idle.", "◫"},
	},
	"Life Support": {
		{"Air Recycler", "Filters wheeze with each cycle.", "◎"},
		{"Water Reclamation", "Condensation drips into collection tanks.", "≋"},
		{"Oxygen Tanks", "Emergency reserves, gauges show half-full.", "○"},
	},
	"Cargo Bay": {
		{"Shipping Container", "Dented metal crate, manifest unreadable.", "▣"},
		{"Cargo Crane", "Overhead lifting mechanism, chains dangling.", "╥"},
		{"Loading Dolly", "Wheeled cart, one wheel broken.", "□"},
	},
	"Storage": {
		{"Supply Shelf", "Canned goods and emergency rations.", "▤"},
		{"Equipment Locker", "Personal effects, owners unknown.", "▥"},
		{"Crate Stack", "Boxes piled haphazardly.", "▣"},
	},
	"Hangar": {
		{"Landing Pad", "Scorch marks indicate recent departures.", "▭"},
		{"Fuel Pump", "Emergency shutoff engaged.", "╪"},
		{"Tool Cart", "Maintenance equipment for spacecraft.", "╤"},
	},
	"Armory": {
		{"Weapon Rack", "Empty slots where rifles once hung.", "╫"},
		{"Ammo Crate", "Heavy box, lid pried open.", "▣"},
		{"Body Armor Stand", "A single damaged suit remains.", "╥"},
	},
	"Med Bay": {
		{"Medical Bed", "Sterile sheets, hastily stripped.", "╦"},
		{"Medicine Cabinet", "Pharmaceutical supplies, mostly depleted.", "▥"},
		{"Diagnostic Scanner", "Handheld medical tricorder on the counter.", "◫"},
	},
	"Lab": {
		{"Microscope Station", "Slides still loaded, samples dried.", "◎"},
		{"Chemical Hood", "Fume extractor hums softly.", "╥"},
		{"Specimen Jars", "Preserved samples float in murky liquid.", "○"},
	},
	"Hydroponics": {
		{"Growth Bed", "Wilted plants in nutrient solution.", "≋"},
		{"UV Lamps", "Artificial sunlight, flickering.", "¤"},
		{"Seed Storage", "Labeled drawers of genetic samples.", "▤"},
	},
	"Observatory": {
		{"Telescope Mount", "Lens pointed at infinite darkness.", "◎"},
		{"Star Chart", "Constellations marked with navigation routes.", "✦"},
		{"Recording Equipment", "Astronomical data, decades of observations.", "◫"},
	},
	"Crew Quarters": {
		{"Bunk Bed", "Personal effects scattered on unmade sheets.", "╦"},
		{"Footlocker", "Lock broken, contents rifled through.", "▣"},
		{"Photo Display", "Faded images of distant families.", "▤"},
	},
	"Mess Hall": {
		{"Dining Table", "Trays of food, long since spoiled.", "╤"},
		{"Food Dispenser", "Vending machine, selections limited.", "▥"},
		{"Coffee Machine", "The pot is cold and empty.", "○"},
	},
	"Airlock": {
		{"Pressure Door", "Heavy bulkhead, seals intact.", "▥"},
		{"EVA Suit Rack", "Emergency spacesuits, some missing.", "╫"},
		{"Decompression Controls", "Warning lights flash intermittently.", "◈"},
	},
}

// GetFurnitureForRoom returns a random furniture piece appropriate for the room type
func GetFurnitureForRoom(roomName string) *FurnitureTemplate {
	// Check each room type to see if it's contained in the room name
	for baseRoom, templates := range RoomFurniture {
		if len(templates) > 0 && containsString(roomName, baseRoom) {
			return &templates[0] // Return first template, caller can randomize
		}
	}
	return nil
}

// GetAllFurnitureForRoom returns all furniture templates for a room type
func GetAllFurnitureForRoom(roomName string) []FurnitureTemplate {
	for baseRoom, templates := range RoomFurniture {
		if containsString(roomName, baseRoom) {
			return templates
		}
	}
	return nil
}

// containsString checks if haystack contains needle
func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) && findString(haystack, needle) >= 0
}

// findString finds needle in haystack, returns -1 if not found
func findString(haystack, needle string) int {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
