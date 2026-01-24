// Package menu provides maintenance menu implementation using the generic menu system.
package menu

import (
	"fmt"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// DeviceMenuItem represents a menu item for a device in the maintenance terminal.
type DeviceMenuItem struct {
	Device entities.DeviceInfo
}

// GetLabel returns the display label for this device menu item.
func (d *DeviceMenuItem) GetLabel() string {
	status := "OFF"
	if d.Device.IsActive {
		status = "ON"
	}
	return fmt.Sprintf("%s (%s) - ACTION{%d} watts - %s", d.Device.Name, d.Device.Type, d.Device.PowerCost, status)
}

// IsSelectable returns whether this device can be selected.
func (d *DeviceMenuItem) IsSelectable() bool {
	return false // Devices are read-only information
}

// GetHelpText returns help text for this device.
func (d *DeviceMenuItem) GetHelpText() string {
	return ""
}

// InfoMenuItem represents a menu item for displaying information (non-selectable).
type InfoMenuItem struct {
	Label string
}

// GetLabel returns the display label for this info menu item.
func (i *InfoMenuItem) GetLabel() string {
	return i.Label
}

// IsSelectable returns whether this info item can be selected.
func (i *InfoMenuItem) IsSelectable() bool {
	return false
}

// GetHelpText returns help text for this info item.
func (i *InfoMenuItem) GetHelpText() string {
	return ""
}

// MaintenanceMenuHandler handles the maintenance terminal menu.
type MaintenanceMenuHandler struct {
	g               *state.Game
	cell            *world.Cell
	maintenanceTerm *entities.MaintenanceTerminal
	roomName        string
	devices         []entities.DeviceInfo
	roomConsumption int
}

// NewMaintenanceMenuHandler creates a new maintenance menu handler.
func NewMaintenanceMenuHandler(g *state.Game, cell *world.Cell, maintenanceTerm *entities.MaintenanceTerminal) *MaintenanceMenuHandler {
	roomName := maintenanceTerm.RoomName

	// Collect all devices in this room
	var devices []entities.DeviceInfo
	lightCount := 0
	lightPower := 0

	// Find all cells in this room
	g.Grid.ForEachCell(func(row, col int, c *world.Cell) {
		if c == nil || !c.Room || c.Name != roomName {
			return
		}

		data := gameworld.GetGameData(c)

		// Count lights (aggregate)
		if data.LightsOn {
			lightCount++
			lightPower++
		}

		// CCTV terminals
		if data.Terminal != nil {
			powerCost := 0
			if data.Terminal.Used {
				powerCost = 5
			}
			devices = append(devices, entities.DeviceInfo{
				Name:      data.Terminal.Name,
				Type:      "Terminal",
				PowerCost: powerCost,
				IsActive:  data.Terminal.Used,
				CanToggle: false,
			})
		}

		// Puzzle terminals
		if data.Puzzle != nil {
			powerCost := 0
			if data.Puzzle.IsSolved() {
				powerCost = 3
			}
			devices = append(devices, entities.DeviceInfo{
				Name:      data.Puzzle.Name,
				Type:      "Puzzle",
				PowerCost: powerCost,
				IsActive:  data.Puzzle.IsSolved(),
				CanToggle: false,
			})
		}

		// Maintenance terminal itself (doesn't consume power - just information display)
		if data.MaintenanceTerm != nil && data.MaintenanceTerm != maintenanceTerm {
			devices = append(devices, entities.DeviceInfo{
				Name:      data.MaintenanceTerm.Name,
				Type:      "Maintenance",
				PowerCost: 0,
				IsActive:  false,
				CanToggle: false,
			})
		}
	})

	// Add lighting as a single aggregated entry
	if lightCount > 0 {
		devices = append([]entities.DeviceInfo{{
			Name:      fmt.Sprintf("Room Lighting (%d cells)", lightCount),
			Type:      "Light",
			PowerCost: lightPower,
			IsActive:  true,
			CanToggle: true,
		}}, devices...)
	}

	// Calculate room power consumption
	roomConsumption := 0
	for _, device := range devices {
		roomConsumption += device.PowerCost
	}

	return &MaintenanceMenuHandler{
		g:               g,
		cell:            cell,
		maintenanceTerm: maintenanceTerm,
		roomName:        roomName,
		devices:         devices,
		roomConsumption: roomConsumption,
	}
}

// GetTitle returns the menu title.
func (h *MaintenanceMenuHandler) GetTitle() string {
	return fmt.Sprintf("Maintenance Terminal: %s", h.roomName)
}

// GetInstructions returns the menu instructions.
func (h *MaintenanceMenuHandler) GetInstructions(selected MenuItem) string {
	return "Press any key to close..."
}

// OnSelect is called when an item is selected.
func (h *MaintenanceMenuHandler) OnSelect(item MenuItem, index int) {
	// Nothing to do on selection
}

// OnActivate is called when an item is activated.
func (h *MaintenanceMenuHandler) OnActivate(item MenuItem, index int) (shouldClose bool, helpText string) {
	// All items are read-only, so any activation closes the menu
	return true, ""
}

// OnExit is called when the menu is exited.
func (h *MaintenanceMenuHandler) OnExit() {
	// Nothing to do on exit
}

// ShouldCloseOnAnyAction returns true if the menu should close on any action.
func (h *MaintenanceMenuHandler) ShouldCloseOnAnyAction() bool {
	return true // Maintenance menu closes on any key press
}

// GetMenuItems returns the menu items for the maintenance menu.
func (h *MaintenanceMenuHandler) GetMenuItems() []MenuItem {
	items := []MenuItem{
		&InfoMenuItem{Label: fmt.Sprintf("Power Supply: ACTION{%d} watts", h.g.PowerSupply)},
		&InfoMenuItem{Label: fmt.Sprintf("Power Consumption: ACTION{%d} watts", h.g.PowerConsumption)},
		&InfoMenuItem{Label: fmt.Sprintf("Available Power: ACTION{%d} watts", h.g.GetAvailablePower())},
		&InfoMenuItem{Label: ""}, // Empty line
		&InfoMenuItem{Label: fmt.Sprintf("Room Devices (%d):", len(h.devices))},
		&InfoMenuItem{Label: fmt.Sprintf("Room Power Consumption: ACTION{%d} watts", h.roomConsumption)},
		&InfoMenuItem{Label: ""}, // Empty line
	}

	if len(h.devices) == 0 {
		items = append(items, &InfoMenuItem{Label: "No active devices in this room."})
	} else {
		for _, device := range h.devices {
			items = append(items, &DeviceMenuItem{Device: device})
		}
	}

	return items
}
