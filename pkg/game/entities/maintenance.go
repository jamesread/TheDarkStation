package entities

// MaintenanceTerminal represents a maintenance terminal for managing room power
type MaintenanceTerminal struct {
	Name     string
	RoomName string // Name of the room this terminal controls
	Used     bool   // Whether the terminal has been accessed
}

// NewMaintenanceTerminal creates a new maintenance terminal
func NewMaintenanceTerminal(name string, roomName string) *MaintenanceTerminal {
	return &MaintenanceTerminal{
		Name:     name,
		RoomName: roomName,
		Used:     false,
	}
}

// Activate marks the terminal as used
func (m *MaintenanceTerminal) Activate() {
	m.Used = true
}

// IsUsed returns whether the terminal has been used
func (m *MaintenanceTerminal) IsUsed() bool {
	return m.Used
}

// DeviceInfo represents information about a power-consuming device
type DeviceInfo struct {
	Name      string
	Type      string // "Light", "Terminal", "Puzzle", etc.
	PowerCost int    // Power consumption
	IsActive  bool   // Whether the device is currently on
	CanToggle bool   // Whether the device can be toggled
}
