// Package menu provides maintenance menu implementation using the generic menu system.
package menu

import (
	"fmt"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// Ping radius in cells (Euclidean distance).
const pingRadius = 15

// DeviceMenuItem represents a menu item for a device in the maintenance terminal.
type DeviceMenuItem struct {
	Device entities.DeviceInfo
}

// GetLabel returns the display label for this device menu item.
// Uses tab so the renderer can align values in a column (maintenance terminal style).
func (d *DeviceMenuItem) GetLabel() string {
	status := "OFF"
	if d.Device.IsActive {
		status = "ON"
	}
	return fmt.Sprintf("%s (%s) -\tACTION{%d} watts - %s", d.Device.Name, d.Device.Type, d.Device.PowerCost, status)
}

// IsSelectable returns whether this device can be selected.
func (d *DeviceMenuItem) IsSelectable() bool {
	return false // Devices are read-only information
}

// GetHelpText returns help text for this device.
func (d *DeviceMenuItem) GetHelpText() string {
	return ""
}

// PingTerminalsMenuItem is a selectable menu item that discovers nearby terminals in the room.
type PingTerminalsMenuItem struct{}

// GetLabel returns the display label for the ping action.
func (p *PingTerminalsMenuItem) GetLabel() string {
	return "Ping nearby terminals"
}

// IsSelectable returns true so the player can activate this action.
func (p *PingTerminalsMenuItem) IsSelectable() bool {
	return true
}

// GetHelpText returns help text for this item.
func (p *PingTerminalsMenuItem) GetHelpText() string {
	return ""
}

// CloseMenuItem is a selectable menu item that closes the current menu (e.g. ping results).
type CloseMenuItem struct {
	Label string
}

// GetLabel returns the display label.
func (c *CloseMenuItem) GetLabel() string {
	if c.Label != "" {
		return c.Label
	}
	return "Close"
}

// IsSelectable returns true so the user can activate to close.
func (c *CloseMenuItem) IsSelectable() bool {
	return true
}

// GetHelpText returns help text for this item.
func (c *CloseMenuItem) GetHelpText() string {
	return ""
}

// pingResult holds one discovered terminal for the ping results menu.
type pingResult struct {
	Name string
	Type string
}

// PingResultsMenuHandler handles the ping results sub-menu.
type PingResultsMenuHandler struct {
	items []MenuItem
}

// GetTitle returns the ping results menu title.
func (h *PingResultsMenuHandler) GetTitle() string {
	return "Ping results"
}

// GetInstructions returns the menu instructions.
func (h *PingResultsMenuHandler) GetInstructions(selected MenuItem) string {
	return "Press Enter to close. Escape or Menu to close."
}

// OnSelect is called when an item is selected.
func (h *PingResultsMenuHandler) OnSelect(item MenuItem, index int) {}

// OnActivate is called when an item is activated.
func (h *PingResultsMenuHandler) OnActivate(item MenuItem, index int) (shouldClose bool, helpText string) {
	if _, isClose := item.(*CloseMenuItem); isClose {
		return true, ""
	}
	return false, ""
}

// OnExit is called when the menu is exited.
func (h *PingResultsMenuHandler) OnExit() {}

// ShouldCloseOnAnyAction returns false so user can select Close and press Enter.
func (h *PingResultsMenuHandler) ShouldCloseOnAnyAction() bool {
	return false
}

// GetMenuItems returns the pre-built ping result items.
func (h *PingResultsMenuHandler) GetMenuItems() []MenuItem {
	return h.items
}

// RoomPowerToggleMenuItem is a selectable menu item that toggles room doors or CCTV power.
type RoomPowerToggleMenuItem struct {
	G         *state.Game
	RoomName  string
	PowerType string // "doors" or "cctv"
}

// GetLabel returns the current power state for this room/system.
func (r *RoomPowerToggleMenuItem) GetLabel() string {
	var on bool
	if r.PowerType == "doors" {
		on = r.G.RoomDoorsPowered[r.RoomName]
	} else {
		on = r.G.RoomCCTVPowered[r.RoomName]
	}
	status := "OFF"
	if on {
		status = "ON"
	}
	name := "Room doors"
	if r.PowerType == "cctv" {
		name = "Room CCTV"
	}
	return fmt.Sprintf("%s (%s):\t%s", name, r.RoomName, status)
}

// IsSelectable returns true so the player can toggle.
func (r *RoomPowerToggleMenuItem) IsSelectable() bool {
	return true
}

// GetHelpText returns help text.
func (r *RoomPowerToggleMenuItem) GetHelpText() string {
	return "Press Enter to toggle power"
}

// RestoreNearbyTerminalsMenuItem restores power to maintenance terminals in adjacent rooms.
type RestoreNearbyTerminalsMenuItem struct {
	G               *state.Game
	CurrentRoomName string
}

// GetLabel returns the menu entry label.
func (r *RestoreNearbyTerminalsMenuItem) GetLabel() string {
	return "Restore power to nearby terminals"
}

// IsSelectable returns true.
func (r *RestoreNearbyTerminalsMenuItem) IsSelectable() bool {
	return true
}

// GetHelpText returns help text.
func (r *RestoreNearbyTerminalsMenuItem) GetHelpText() string {
	return "Press Enter to power maintenance terminals in adjacent rooms"
}

// AllRoomsPowerMenuItem opens a sub-menu to control power for adjacent rooms only.
type AllRoomsPowerMenuItem struct {
	G *state.Game
}

// GetLabel returns the menu entry label.
func (a *AllRoomsPowerMenuItem) GetLabel() string {
	return "Adjacent rooms power..."
}

// IsSelectable returns true.
func (a *AllRoomsPowerMenuItem) IsSelectable() bool {
	return true
}

// GetHelpText returns help text.
func (a *AllRoomsPowerMenuItem) GetHelpText() string {
	return "Press Enter to set doors/CCTV power (adjacent rooms only)"
}

// AllRoomsPowerMenuHandler handles the adjacent-rooms power sub-menu.
type AllRoomsPowerMenuHandler struct {
	g        *state.Game
	roomName string
	items    []MenuItem
}

// GetTitle returns the sub-menu title.
func (h *AllRoomsPowerMenuHandler) GetTitle() string {
	return "Room power (adjacent rooms)"
}

// GetInstructions returns instructions.
func (h *AllRoomsPowerMenuHandler) GetInstructions(selected MenuItem) string {
	return "Press Enter to toggle. Escape or Menu to close."
}

// OnSelect is called when selection changes.
func (h *AllRoomsPowerMenuHandler) OnSelect(item MenuItem, index int) {}

// OnActivate toggles power when a RoomPowerToggleMenuItem is activated.
func (h *AllRoomsPowerMenuHandler) OnActivate(item MenuItem, index int) (shouldClose bool, helpText string) {
	if _, isClose := item.(*CloseMenuItem); isClose {
		return true, ""
	}
	if toggle, isToggle := item.(*RoomPowerToggleMenuItem); isToggle {
		helpText := ""
		if toggle.PowerType == "doors" {
			h.g.RoomDoorsPowered[toggle.RoomName] = !h.g.RoomDoorsPowered[toggle.RoomName]
			if h.g.RoomDoorsPowered[toggle.RoomName] {
				if h.g.ShortOutIfOverload(toggle.RoomName) {
					helpText = "Power overload! Other systems shorted out."
				}
			}
		} else {
			h.g.RoomCCTVPowered[toggle.RoomName] = !h.g.RoomCCTVPowered[toggle.RoomName]
			if h.g.RoomCCTVPowered[toggle.RoomName] {
				if h.g.ShortOutIfOverload(toggle.RoomName) {
					helpText = "Power overload! Other systems shorted out."
				}
			}
		}
		// Rebuild items for adjacent rooms only so labels refresh
		h.items = buildAdjacentRoomsPowerItems(h.g, h.roomName)
		return false, helpText
	}
	return true, ""
}

// OnExit is called when the sub-menu is exited.
func (h *AllRoomsPowerMenuHandler) OnExit() {}

// ShouldCloseOnAnyAction returns false.
func (h *AllRoomsPowerMenuHandler) ShouldCloseOnAnyAction() bool {
	return false
}

// GetMenuItems returns the items for the all-rooms power menu.
func (h *AllRoomsPowerMenuHandler) GetMenuItems() []MenuItem {
	return h.items
}

// buildAdjacentRoomsPowerItems builds menu items only for the current room and rooms directly adjacent to it.
func buildAdjacentRoomsPowerItems(g *state.Game, currentRoomName string) []MenuItem {
	names := setup.GetAdjacentRoomNames(g.Grid, currentRoomName)
	var items []MenuItem
	items = append(items, &InfoMenuItem{Label: "Toggle doors and CCTV (this room + adjacent only):"}, &InfoMenuItem{Label: ""})
	for _, roomName := range names {
		items = append(items, &InfoMenuItem{Label: roomName + ":"})
		items = append(items, &RoomPowerToggleMenuItem{G: g, RoomName: roomName, PowerType: "doors"})
		items = append(items, &RoomPowerToggleMenuItem{G: g, RoomName: roomName, PowerType: "cctv"})
		items = append(items, &InfoMenuItem{Label: ""})
	}
	items = append(items, &CloseMenuItem{Label: "Close"})
	return items
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
	doorCount := 0

	// Count doors that lead to this room (they are on corridor cells)
	g.Grid.ForEachCell(func(row, col int, c *world.Cell) {
		if c == nil {
			return
		}
		data := gameworld.GetGameData(c)
		if data.Door != nil && data.Door.RoomName == roomName {
			doorCount++
		}
	})

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

		// CCTV terminals (10W when room CCTV is powered)
		if data.Terminal != nil {
			powerCost := 0
			if g.RoomCCTVPowered[roomName] {
				powerCost = 10
			}
			devices = append(devices, entities.DeviceInfo{
				Name:      data.Terminal.Name,
				Type:      "Terminal",
				PowerCost: powerCost,
				IsActive:  g.RoomCCTVPowered[roomName],
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

		// Maintenance terminal (powered state; can be restored from nearby)
		if data.MaintenanceTerm != nil && data.MaintenanceTerm != maintenanceTerm {
			devices = append(devices, entities.DeviceInfo{
				Name:      data.MaintenanceTerm.Name,
				Type:      "Maintenance",
				PowerCost: 0,
				IsActive:  data.MaintenanceTerm.Powered,
				CanToggle: false,
			})
		}
	})

	// Add doors that lead to this room (10W each when powered)
	if doorCount > 0 {
		powerCost := 0
		if g.RoomDoorsPowered[roomName] {
			powerCost = doorCount * 10
		}
		devices = append(devices, entities.DeviceInfo{
			Name:      fmt.Sprintf("Doors (%d)", doorCount),
			Type:      "Doors",
			PowerCost: powerCost,
			IsActive:  g.RoomDoorsPowered[roomName],
			CanToggle: false,
		})
	}

	// Add lighting as a single aggregated entry (standard cells no longer consume power)
	if lightCount > 0 {
		devices = append([]entities.DeviceInfo{{
			Name:      fmt.Sprintf("Room Lighting (%d cells)", lightCount),
			Type:      "Light",
			PowerCost: 0,
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
	return "Press Enter to select. Escape or Menu to close."
}

// OnSelect is called when an item is selected.
func (h *MaintenanceMenuHandler) OnSelect(item MenuItem, index int) {
	// Nothing to do on selection
}

// OnActivate is called when an item is activated.
func (h *MaintenanceMenuHandler) OnActivate(item MenuItem, index int) (shouldClose bool, helpText string) {
	if _, isClose := item.(*CloseMenuItem); isClose {
		return true, ""
	}
	if toggle, isToggle := item.(*RoomPowerToggleMenuItem); isToggle {
		if toggle.PowerType == "doors" {
			h.g.RoomDoorsPowered[toggle.RoomName] = !h.g.RoomDoorsPowered[toggle.RoomName]
			if h.g.RoomDoorsPowered[toggle.RoomName] {
				if h.g.ShortOutIfOverload(toggle.RoomName) {
					return true, "Power overload! Other systems shorted out."
				}
			}
		} else {
			h.g.RoomCCTVPowered[toggle.RoomName] = !h.g.RoomCCTVPowered[toggle.RoomName]
			if h.g.RoomCCTVPowered[toggle.RoomName] {
				if h.g.ShortOutIfOverload(toggle.RoomName) {
					return true, "Power overload! Other systems shorted out."
				}
			}
		}
		return true, "" // Close so player can reopen and see updated state
	}
	if _, isAllRooms := item.(*AllRoomsPowerMenuItem); isAllRooms {
		items := buildAdjacentRoomsPowerItems(h.g, h.roomName)
		handler := &AllRoomsPowerMenuHandler{g: h.g, roomName: h.roomName, items: items}
		RunMenu(h.g, items, handler)
		return false, ""
	}
	if _, isRestore := item.(*RestoreNearbyTerminalsMenuItem); isRestore {
		// Power all maintenance terminals in rooms adjacent to this terminal's room
		adjacentNames := setup.GetAdjacentRoomNames(h.g.Grid, h.roomName)
		adjacentSet := make(map[string]bool)
		for _, name := range adjacentNames {
			adjacentSet[name] = true
		}
		restored := 0
		h.g.Grid.ForEachCell(func(row, col int, c *world.Cell) {
			if c == nil || !c.Room {
				return
			}
			data := gameworld.GetGameData(c)
			if data.MaintenanceTerm == nil || data.MaintenanceTerm == h.maintenanceTerm {
				return
			}
			if !adjacentSet[c.Name] {
				return
			}
			if !data.MaintenanceTerm.Powered {
				data.MaintenanceTerm.Powered = true
				restored++
			}
		})
		if restored > 0 {
			// Feedback could be shown via callout; for now menu stays open
			return false, fmt.Sprintf("Restored power to %d terminal(s)", restored)
		}
		return false, "No unpowered terminals in nearby rooms"
	}
	if _, isPing := item.(*PingTerminalsMenuItem); isPing {
		centerRow, centerCol := h.cell.Row, h.cell.Col
		radiusSq := pingRadius * pingRadius
		var results []pingResult

		h.g.Grid.ForEachCell(func(row, col int, c *world.Cell) {
			if c == nil {
				return
			}
			dr := row - centerRow
			dc := col - centerCol
			if dr*dr+dc*dc > radiusSq {
				return
			}
			if !gameworld.HasTerminal(c) && !gameworld.HasPuzzle(c) {
				return
			}
			if c.Discovered {
				return
			}
			c.Discovered = true
			data := gameworld.GetGameData(c)
			if gameworld.HasTerminal(c) && data.Terminal != nil {
				results = append(results, pingResult{Name: data.Terminal.Name, Type: "CCTV"})
			}
			if gameworld.HasPuzzle(c) && data.Puzzle != nil {
				results = append(results, pingResult{Name: data.Puzzle.Name, Type: "Puzzle"})
			}
		})

		// Build ping results menu items and open sub-menu
		resultItems := []MenuItem{
			&InfoMenuItem{Label: "Ping results"},
			&InfoMenuItem{Label: ""},
		}
		if len(results) == 0 {
			resultItems = append(resultItems, &InfoMenuItem{Label: "No terminals within range."})
		} else {
			resultItems = append(resultItems, &InfoMenuItem{Label: fmt.Sprintf("Discovered %d terminal(s):", len(results))})
			for _, r := range results {
				resultItems = append(resultItems, &InfoMenuItem{Label: fmt.Sprintf("  %s (%s)", r.Name, r.Type)})
			}
		}
		resultItems = append(resultItems, &InfoMenuItem{Label: ""}, &CloseMenuItem{Label: "Close"})

		resultsHandler := &PingResultsMenuHandler{items: resultItems}
		RunMenu(h.g, resultItems, resultsHandler)
		return false, ""
	}
	// Other items (info/devices) are read-only; any other activation closes the menu
	return true, ""
}

// OnExit is called when the menu is exited.
func (h *MaintenanceMenuHandler) OnExit() {
	// Nothing to do on exit
}

// ShouldCloseOnAnyAction returns true if the menu should close on any action.
func (h *MaintenanceMenuHandler) ShouldCloseOnAnyAction() bool {
	return false // Allow activating "Ping nearby terminals"; close via menu/quit key
}

// GetMenuItems returns the menu items for the maintenance menu.
// Labels use tab (\t) so the renderer can align values in a column (maintenance terminal style).
func (h *MaintenanceMenuHandler) GetMenuItems() []MenuItem {
	items := []MenuItem{
		&InfoMenuItem{Label: fmt.Sprintf("Power Supply:\tACTION{%d} watts", h.g.PowerSupply)},
		&InfoMenuItem{Label: fmt.Sprintf("Power Consumption:\tACTION{%d} watts", h.g.PowerConsumption)},
		&InfoMenuItem{Label: fmt.Sprintf("Available Power:\tACTION{%d} watts", h.g.GetAvailablePower())},
		&InfoMenuItem{Label: ""}, // Empty line
		&InfoMenuItem{Label: fmt.Sprintf("Room Devices (%d):", len(h.devices))},
		&InfoMenuItem{Label: fmt.Sprintf("Room Power Consumption:\tACTION{%d} watts", h.roomConsumption)},
	}

	if len(h.devices) == 0 {
		items = append(items, &InfoMenuItem{Label: "No active devices in this room."})
	} else {
		for _, device := range h.devices {
			items = append(items, &DeviceMenuItem{Device: device})
		}
	}

	items = append(items,
		&InfoMenuItem{Label: ""}, // Empty line
		&InfoMenuItem{Label: "Room power (10W each when on):"},
		&RoomPowerToggleMenuItem{G: h.g, RoomName: h.roomName, PowerType: "doors"},
		&RoomPowerToggleMenuItem{G: h.g, RoomName: h.roomName, PowerType: "cctv"},
		&InfoMenuItem{Label: ""},
		&AllRoomsPowerMenuItem{G: h.g},
		&InfoMenuItem{Label: ""},
		&RestoreNearbyTerminalsMenuItem{G: h.g, CurrentRoomName: h.roomName},
		&InfoMenuItem{Label: ""}, // Empty line
		&PingTerminalsMenuItem{}, // Ping discovers nearby terminals on the map
		&InfoMenuItem{Label: ""}, // Empty line
		&CloseMenuItem{Label: "Close"},
	)
	return items
}
