package entities

// HazardType represents different types of environmental hazards
type HazardType int

const (
	HazardVacuum     HazardType = iota // Depressurized area - needs breach sealed with Patch Kit
	HazardCoolant                      // Coolant leak - needs Shutoff Valve activated
	HazardElectrical                   // Electrical fault - needs Circuit Breaker reset
	HazardGas                          // Gas leak - needs Vent Control activated
	HazardRadiation                    // Radiation leak - needs Containment Field activated
)

// Hazard represents an environmental hazard blocking a cell
type Hazard struct {
	Type        HazardType
	Name        string         // Display name
	Description string         // Description shown when trying to enter
	Fixed       bool           // Whether the hazard has been cleared
	Control     *HazardControl // The control that fixes this hazard (nil for item-based fixes)
}

// HazardControl represents a control panel that can fix a hazard
type HazardControl struct {
	Type        HazardType
	Name        string  // Display name (e.g., "Coolant Shutoff Valve")
	Description string  // Description when activated
	Activated   bool    // Whether this control has been used
	Hazard      *Hazard // The hazard this control fixes
}

// HazardInfo contains display information for each hazard type
type HazardInfo struct {
	Name           string
	BlockedMessage string
	FixedMessage   string
	Icon           string
	IconFixed      string
	ControlName    string
	ControlIcon    string
	RequiresItem   bool   // If true, needs an item instead of a control
	ItemName       string // Item needed (if RequiresItem is true)
}

// HazardTypes maps hazard types to their display information
var HazardTypes = map[HazardType]HazardInfo{
	HazardVacuum: {
		Name:           "Vacuum",
		BlockedMessage: "This section is depressurized. You need a Patch Kit to seal the breach.",
		FixedMessage:   "You seal the breach with the Patch Kit. Atmosphere restored.",
		Icon:           "◊",
		IconFixed:      "·",
		RequiresItem:   true,
		ItemName:       "Patch Kit",
	},
	HazardCoolant: {
		Name:           "Coolant Leak",
		BlockedMessage: "Supercooled coolant sprays across the passage. Find the Shutoff Valve.",
		FixedMessage:   "The coolant flow stops. Passage is clear.",
		Icon:           "≋",
		IconFixed:      "·",
		ControlName:    "Coolant Shutoff",
		ControlIcon:    "⊗",
	},
	HazardElectrical: {
		Name:           "Electrical Fault",
		BlockedMessage: "Sparks arc across the corridor. Find the Circuit Breaker.",
		FixedMessage:   "Power rerouted. The sparking stops.",
		Icon:           "⚡",
		IconFixed:      "·",
		ControlName:    "Circuit Breaker",
		ControlIcon:    "⊠",
	},
	HazardGas: {
		Name:           "Gas Leak",
		BlockedMessage: "Toxic gas fills the area. Find the Vent Control.",
		FixedMessage:   "Vents engage. The gas dissipates.",
		Icon:           "☁",
		IconFixed:      "·",
		ControlName:    "Vent Control",
		ControlIcon:    "⊞",
	},
	HazardRadiation: {
		Name:           "Radiation Leak",
		BlockedMessage: "Dangerous radiation levels detected. Find the Containment Control.",
		FixedMessage:   "Containment field activated. Radiation contained.",
		Icon:           "☢",
		IconFixed:      "·",
		ControlName:    "Containment Control",
		ControlIcon:    "⊛",
	},
}

// NewHazard creates a new hazard of the given type
func NewHazard(hazardType HazardType) *Hazard {
	info := HazardTypes[hazardType]
	return &Hazard{
		Type:        hazardType,
		Name:        info.Name,
		Description: info.BlockedMessage,
		Fixed:       false,
	}
}

// NewHazardControl creates a control panel for a hazard
func NewHazardControl(hazardType HazardType, hazard *Hazard) *HazardControl {
	info := HazardTypes[hazardType]
	control := &HazardControl{
		Type:        hazardType,
		Name:        info.ControlName,
		Description: info.FixedMessage,
		Activated:   false,
		Hazard:      hazard,
	}
	hazard.Control = control
	return control
}

// Activate activates the control and fixes the linked hazard
func (c *HazardControl) Activate() {
	if c.Activated {
		return
	}
	c.Activated = true
	if c.Hazard != nil {
		c.Hazard.Fixed = true
	}
}

// Fix marks the hazard as fixed (used for item-based fixes)
func (h *Hazard) Fix() {
	h.Fixed = true
}

// IsBlocking returns true if this hazard is currently blocking passage
func (h *Hazard) IsBlocking() bool {
	return !h.Fixed
}

// RequiresItem returns true if this hazard type needs an item to fix
func (h *Hazard) RequiresItem() bool {
	info := HazardTypes[h.Type]
	return info.RequiresItem
}

// RequiredItemName returns the name of the item needed to fix this hazard
func (h *Hazard) RequiredItemName() string {
	info := HazardTypes[h.Type]
	return info.ItemName
}

// GetIcon returns the appropriate icon for this hazard's current state
func (h *Hazard) GetIcon() string {
	info := HazardTypes[h.Type]
	if h.Fixed {
		return info.IconFixed
	}
	return info.Icon
}

// GetControlIcon returns the icon for this hazard type's control
func GetControlIcon(hazardType HazardType) string {
	return HazardTypes[hazardType].ControlIcon
}
