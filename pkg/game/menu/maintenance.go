// Package menu provides maintenance menu implementation using the generic menu system.
package menu

import (
	"fmt"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// roomPowerSummary returns per-room supply and consumption for display.
func roomPowerSummary(g *state.Game, roomName string) (supply, consumption int) {
	if g == nil || g.Grid == nil {
		return 0, 0
	}
	params := deck.DecayParamsForDeck(g.CurrentDeckID)
	wattsPerGenerator := 100

	// Supply from generators in this room
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name != roomName {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Generator != nil && data.Generator.IsPowered() {
			supply += int(float64(wattsPerGenerator) * params.GeneratorOutputMultiplier)
		}
	})

	// Consumption from doors (10w per powered room), CCTV and puzzles (in this room)
	var rawConsumption int
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Door != nil && data.Door.RoomName == roomName && g.RoomDoorsPowered[roomName] {
			rawConsumption += 10
		}
		if cell.Room && cell.Name == roomName {
			if data.Terminal != nil && g.RoomCCTVPowered[roomName] {
				rawConsumption += 10
			}
			if data.Puzzle != nil && data.Puzzle.IsSolved() {
				rawConsumption += 3
			}
		}
	})
	consumption = int(float64(rawConsumption) * params.PowerCostMultiplier)
	return supply, consumption
}

// Ping radius in cells (Euclidean distance).
const pingRadius = 15

// DeviceMenuItem represents a menu item for a device in the maintenance terminal.
type DeviceMenuItem struct {
	Device entities.DeviceInfo
}

// GetLabel returns the display label for this device menu item.
// Uses tab so the renderer can align values in a column (maintenance terminal style).
func (d *DeviceMenuItem) GetLabel() string {
	watts := 0
	if d.Device.IsActive {
		watts = d.Device.PowerCost
	}
	return fmt.Sprintf("%s (%s) -\t%s", d.Device.Name, d.Device.Type, renderer.FormatPowerWatts(watts, false))
}

// IsSelectable returns whether this device can be selected.
func (d *DeviceMenuItem) IsSelectable() bool {
	return false // Devices are read-only information
}

// GetHelpText returns help text for this device.
func (d *DeviceMenuItem) GetHelpText() string {
	return ""
}

// RestorePowerNearbyTerminalsMenuItem powers all maintenance terminals in adjacent rooms (including own).
type RestorePowerNearbyTerminalsMenuItem struct {
	Parent *MaintenanceMenuHandler
}

// GetLabel returns the display label.
func (r *RestorePowerNearbyTerminalsMenuItem) GetLabel() string {
	return "Restore power to nearby terminals"
}

// IsSelectable returns true (menu is only open at powered terminals).
func (r *RestorePowerNearbyTerminalsMenuItem) IsSelectable() bool {
	return true
}

// GetHelpText returns help text.
func (r *RestorePowerNearbyTerminalsMenuItem) GetHelpText() string {
	return "Press Enter to restore power to terminals in adjacent rooms"
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

// MessageResultMenuHandler shows a single message on its own menu page with an OK button.
type MessageResultMenuHandler struct {
	Title   string
	Message string
}

// GetTitle returns the result page title.
func (h *MessageResultMenuHandler) GetTitle() string {
	return h.Title
}

// GetInstructions returns instructions.
func (h *MessageResultMenuHandler) GetInstructions(selected MenuItem) string {
	return "Press Enter to close. Escape or Menu to close."
}

// OnSelect is called when selection changes.
func (h *MessageResultMenuHandler) OnSelect(item MenuItem, index int) {}

// OnActivate closes when OK/Close is activated.
func (h *MessageResultMenuHandler) OnActivate(item MenuItem, index int) (shouldClose bool, helpText string) {
	if _, isClose := item.(*CloseMenuItem); isClose {
		return true, ""
	}
	return false, ""
}

// OnExit is called when the menu is exited.
func (h *MessageResultMenuHandler) OnExit() {}

// ShouldCloseOnAnyAction returns false.
func (h *MessageResultMenuHandler) ShouldCloseOnAnyAction() bool {
	return false
}

// GetMenuItems returns the message and an OK item.
func (h *MessageResultMenuHandler) GetMenuItems() []MenuItem {
	return []MenuItem{
		&InfoMenuItem{Label: h.Message},
		&InfoMenuItem{Label: ""},
		&CloseMenuItem{Label: "OK"},
	}
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

// RoomPowerToggleMenuItem is a selectable menu item that toggles room doors, CCTV, or lights.
type RoomPowerToggleMenuItem struct {
	G           *state.Game
	RoomName    string
	PowerType   string // "doors", "cctv", or "lights"
	Count       int    // optional count for label, e.g. "Doors (5)", "Lights (12 cells)"
	CountSuffix string // e.g. " cells" for lights, "" for doors
}

// Room power draw in watts when on (per specs).
const roomPowerWattsWhenOn = 10

// roomMaintenanceTerminalPowered returns true if the given room's maintenance terminal is powered.
// Doors and CCTV in a room can only be toggled when the room's maint terminal is activated first.
func roomMaintenanceTerminalPowered(g *state.Game, roomName string) bool {
	if g == nil || g.Grid == nil {
		return false
	}
	var powered bool
	g.Grid.ForEachCell(func(row, col int, c *world.Cell) {
		if c == nil || !c.Room || c.Name != roomName {
			return
		}
		data := gameworld.GetGameData(c)
		if data.MaintenanceTerm != nil {
			powered = data.MaintenanceTerm.Powered
		}
	})
	return powered
}

// GetLabel returns the current power state and watts for this room/system.
func (r *RoomPowerToggleMenuItem) GetLabel() string {
	var on bool
	switch r.PowerType {
	case "doors":
		on = r.G.RoomDoorsPowered[r.RoomName]
	case "cctv":
		on = r.G.RoomCCTVPowered[r.RoomName]
	case "lights":
		if v, ok := r.G.RoomLightsPowered[r.RoomName]; ok {
			on = v
		} else {
			on = true // default on when not yet set
		}
	default:
		on = false
	}
	maintPowered := roomMaintenanceTerminalPowered(r.G, r.RoomName)
	watts := 0
	if on && r.PowerType != "lights" {
		watts = roomPowerWattsWhenOn
	}
	var powerLabel string
	if r.PowerType == "lights" {
		if on {
			powerLabel = "POWERED{0w}"
		} else {
			powerLabel = "UNPOWERED{0w}"
		}
	} else {
		powerLabel = renderer.FormatPowerWatts(watts, !maintPowered)
	}
	name := "Doors"
	if r.PowerType == "cctv" {
		name = "CCTV"
	} else if r.PowerType == "lights" {
		name = "Lights"
	}
	if r.Count > 0 {
		name = fmt.Sprintf("%s (%d%s)", name, r.Count, r.CountSuffix)
	}
	return fmt.Sprintf("%s:\t%s", name, powerLabel)
}

// IsSelectable returns true only when the room's maintenance terminal is powered.
// Doors and CCTV require the room's maint terminal to be activated first.
func (r *RoomPowerToggleMenuItem) IsSelectable() bool {
	return roomMaintenanceTerminalPowered(r.G, r.RoomName)
}

// GetHelpText returns help text; explains dependency when maint terminal is not powered.
func (r *RoomPowerToggleMenuItem) GetHelpText() string {
	if roomMaintenanceTerminalPowered(r.G, r.RoomName) {
		return "Press Enter to toggle power"
	}
	return "Activate this room's maintenance terminal first"
}

// MaintenanceTerminalPowerMenuItem is a selectable menu item that toggles power for one maintenance terminal.
// Power is restored only via "Restore power to nearby terminals"; this toggle allows turning a terminal
// off for player convenience (e.g. testing or deliberate shutdown).
type MaintenanceTerminalPowerMenuItem struct {
	G    *state.Game
	Term *entities.MaintenanceTerminal
}

// GetLabel returns the current power state and watts for this terminal (terminals use 0w).
func (m *MaintenanceTerminalPowerMenuItem) GetLabel() string {
	powerLabel := "UNPOWERED{0w}"
	if m.Term.Powered {
		powerLabel = "POWERED{0w}"
	}
	return fmt.Sprintf("Terminal:\t%s", powerLabel)
}

// IsSelectable returns true.
func (m *MaintenanceTerminalPowerMenuItem) IsSelectable() bool {
	return true
}

// GetHelpText returns help text.
func (m *MaintenanceTerminalPowerMenuItem) GetHelpText() string {
	return "Press Enter to toggle power"
}

// RoomSelectorMenuItem opens a sub-menu to select which room's maintenance view to display.
type RoomSelectorMenuItem struct {
	Parent *MaintenanceMenuHandler
}

// GetLabel returns the menu entry label showing current selection.
func (r *RoomSelectorMenuItem) GetLabel() string {
	return fmt.Sprintf("Viewing room:\tACTION{%s}\t(select to change)", r.Parent.selectedRoomName)
}

// IsSelectable returns true.
func (r *RoomSelectorMenuItem) IsSelectable() bool {
	return true
}

// GetHelpText returns help text.
func (r *RoomSelectorMenuItem) GetHelpText() string {
	return "Press Enter to select a different room"
}

// roomSelectItem is a selectable room in the room selector sub-menu.
// hint and powerSummary appear in columns next to the room name.
type roomSelectItem struct {
	roomName     string
	hint         string
	powerSummary string
}

func (r *roomSelectItem) GetLabel() string {
	// Always use 3 columns: room name, hint (optional), power (right-aligned)
	if r.powerSummary != "" {
		return r.roomName + "\t" + r.hint + "\t" + r.powerSummary
	}
	if r.hint != "" {
		return r.roomName + "\t" + r.hint
	}
	return r.roomName
}

func (r *roomSelectItem) IsSelectable() bool {
	return true
}

func (r *roomSelectItem) GetHelpText() string {
	return "Press Enter to view this room's maintenance"
}

// RoomSelectorMenuHandler handles the room selection sub-menu.
type RoomSelectorMenuHandler struct {
	parent *MaintenanceMenuHandler
	rooms  []string
}

// GetTitle returns the sub-menu title.
func (h *RoomSelectorMenuHandler) GetTitle() string {
	return "Select room"
}

// GetInstructions returns the menu instructions.
func (h *RoomSelectorMenuHandler) GetInstructions(selected MenuItem) string {
	return "Press Enter to view that room. Escape or Menu to close."
}

// OnSelect is called when selection changes.
func (h *RoomSelectorMenuHandler) OnSelect(item MenuItem, index int) {}

// OnActivate selects the room and closes.
func (h *RoomSelectorMenuHandler) OnActivate(item MenuItem, index int) (shouldClose bool, helpText string) {
	if sel, ok := item.(*roomSelectItem); ok {
		h.parent.selectedRoomName = sel.roomName
		return true, ""
	}
	return true, ""
}

// OnExit is called when the sub-menu is exited.
func (h *RoomSelectorMenuHandler) OnExit() {}

// GetMaintenanceRoom implements MaintenanceRoomProvider - highlights the room under the selection.
func (h *RoomSelectorMenuHandler) GetMaintenanceRoom(selectedIndex int, items []MenuItem) string {
	if selectedIndex >= 0 && selectedIndex < len(items) {
		if sel, ok := items[selectedIndex].(*roomSelectItem); ok {
			return sel.roomName
		}
	}
	return ""
}

// ShouldCloseOnAnyAction returns false.
func (h *RoomSelectorMenuHandler) ShouldCloseOnAnyAction() bool {
	return false
}

// GetMenuItems returns the room list with hints and per-room power summary in columns.
// The currently selected room is always first.
func (h *RoomSelectorMenuHandler) GetMenuItems() []MenuItem {
	var items []MenuItem
	selected := h.parent.selectedRoomName
	// Add selected room first if present
	for _, name := range h.rooms {
		if name == selected {
			items = append(items, h.roomToItem(name))
			break
		}
	}
	// Add remaining rooms (excluding selected, already added)
	for _, name := range h.rooms {
		if name != selected {
			items = append(items, h.roomToItem(name))
		}
	}
	return items
}

func (h *RoomSelectorMenuHandler) roomToItem(name string) *roomSelectItem {
	var hint string
	if name == h.parent.terminalRoomName && name == h.parent.selectedRoomName {
		hint = "(current room, current selection)"
	} else if name == h.parent.terminalRoomName {
		hint = "(current room)"
	} else if name == h.parent.selectedRoomName {
		hint = "(current selection)"
	}
	supply, consumption := roomPowerSummary(h.parent.g, name)
	net := supply - consumption
	powerSummary := renderer.FormatPowerWatts(net, false)
	return &roomSelectItem{roomName: name, hint: hint, powerSummary: powerSummary}
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
	g                *state.Game
	cell             *world.Cell
	maintenanceTerm  *entities.MaintenanceTerminal
	terminalRoomName string   // room where the terminal is
	selectedRoomName string   // room currently being viewed (mutable)
	selectableRooms  []string // current + adjacent rooms
	mode             string   // maintModeControls or maintModeDiagnostics
}

// buildRoomDevices builds device list (CCTV, puzzles only), room consumption, and counts for doors/lights.
// Doors and lights are shown as toggles below; they are not in the device list.
func buildRoomDevices(g *state.Game, roomName string, maintenanceTerm *entities.MaintenanceTerminal) (devices []entities.DeviceInfo, roomConsumption, doorCount, lightCount int) {
	g.Grid.ForEachCell(func(row, col int, c *world.Cell) {
		if c == nil {
			return
		}
		data := gameworld.GetGameData(c)
		if data.Door != nil && data.Door.RoomName == roomName {
			doorCount++
		}
	})

	g.Grid.ForEachCell(func(row, col int, c *world.Cell) {
		if c == nil || !c.Room || c.Name != roomName {
			return
		}
		data := gameworld.GetGameData(c)

		lightCount++
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
	})

	// Consumption: doors (10w each when powered) + device power
	if doorCount > 0 && g.RoomDoorsPowered[roomName] {
		roomConsumption += 10
	}
	for _, d := range devices {
		roomConsumption += d.PowerCost
	}
	return devices, roomConsumption, doorCount, lightCount
}

// NewMaintenanceMenuHandler creates a new maintenance menu handler.
func NewMaintenanceMenuHandler(g *state.Game, cell *world.Cell, maintenanceTerm *entities.MaintenanceTerminal) *MaintenanceMenuHandler {
	roomName := maintenanceTerm.RoomName
	selectableRooms := setup.SelectableRoomsForTerminal(g, g.Grid, roomName)
	if selectableRooms == nil {
		selectableRooms = []string{roomName}
	}

	h := &MaintenanceMenuHandler{
		g:                g,
		cell:             cell,
		maintenanceTerm:  maintenanceTerm,
		terminalRoomName: roomName,
		selectedRoomName: roomName,
		selectableRooms:  selectableRooms,
	}
	h.initMaintenanceMenuState()
	return h
}

// GetTitle returns the menu title.
func (h *MaintenanceMenuHandler) GetTitle() string {
	return fmt.Sprintf("Maintenance Terminal: %s", h.selectedRoomName)
}

// GetMaintenanceRoom implements MaintenanceRoomProvider.
func (h *MaintenanceMenuHandler) GetMaintenanceRoom(selectedIndex int, items []MenuItem) string {
	return h.selectedRoomName
}

// GetInstructions returns the menu instructions.
func (h *MaintenanceMenuHandler) GetInstructions(selected MenuItem) string {
	base := "Up/Down: select | Enter: activate | A/D: switch room | 1/2/3: OFF/ESSENTIAL/FULL | Tab: mode | Esc: close"
	if selected != nil {
		if ht := selected.GetHelpText(); ht != "" {
			return ht + " — " + base
		}
	}
	return base
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
		if !roomMaintenanceTerminalPowered(h.g, toggle.RoomName) {
			return false, "Activate this room's maintenance terminal first"
		}
		helpText := ""
		switch toggle.PowerType {
		case "doors":
			h.g.RoomDoorsPowered[toggle.RoomName] = !h.g.RoomDoorsPowered[toggle.RoomName]
			if h.g.RoomDoorsPowered[toggle.RoomName] {
				if h.g.ShortOutIfOverload(toggle.RoomName) {
					helpText = "Power overload! Other systems shorted out."
				} else if h.g.PowerConsumption > h.g.PowerSupply {
					helpText = "Power overload persists in this room. Reduce load."
				}
			} else {
				h.g.UpdatePowerSupply()
				h.g.PowerConsumption = h.g.CalculatePowerConsumption()
			}
		case "cctv":
			h.g.RoomCCTVPowered[toggle.RoomName] = !h.g.RoomCCTVPowered[toggle.RoomName]
			if h.g.RoomCCTVPowered[toggle.RoomName] {
				if h.g.ShortOutIfOverload(toggle.RoomName) {
					helpText = "Power overload! Other systems shorted out."
				} else if h.g.PowerConsumption > h.g.PowerSupply {
					helpText = "Power overload persists in this room. Reduce load."
				}
			} else {
				h.g.UpdatePowerSupply()
				h.g.PowerConsumption = h.g.CalculatePowerConsumption()
			}
		case "lights":
			current := h.g.RoomLightsPowered[toggle.RoomName]
			if _, ok := h.g.RoomLightsPowered[toggle.RoomName]; !ok {
				current = true
			}
			h.g.RoomLightsPowered[toggle.RoomName] = !current
			// Lights use 0w, no consumption change
		}
		return false, helpText // Keep menu open so user can toggle more
	}
	if termItem, isTerm := item.(*MaintenanceTerminalPowerMenuItem); isTerm {
		termItem.Term.Powered = !termItem.Term.Powered
		return false, "" // Keep menu open like doors/CCTV
	}
	if presetItem, isPreset := item.(*RoomCircuitPresetMenuItem); isPreset {
		next := CurrentCircuitPreset(h.g, presetItem.Parent.selectedRoomName).NextPreset()
		return false, presetItem.Parent.applyCircuitPreset(next)
	}
	if _, isMode := item.(*ModeToggleMenuItem); isMode {
		h.toggleMode()
		if h.mode == maintModeDiagnostics {
			return false, "Diagnostics panel"
		}
		return false, "Controls panel"
	}
	if _, isRestoreAll := item.(*RestoreAllAdjacentMenuItem); isRestoreAll {
		return false, h.restoreAllAdjacent()
	}
	if _, isRestoreSel := item.(*RestoreSelectedRoomMenuItem); isRestoreSel {
		return false, h.restoreSelectedRoom()
	}
	// Legacy type used in tests — same as restore all adjacent.
	if restoreItem, isRestore := item.(*RestorePowerNearbyTerminalsMenuItem); isRestore {
		return false, restoreItem.Parent.restoreAllAdjacent()
	}
	if _, isAdv := item.(*AdvancedPowerMenuItem); isAdv {
		_, _, doorCount, lightCount := buildRoomDevices(h.g, h.selectedRoomName, h.maintenanceTerm)
		handler := &AdvancedPowerMenuHandler{parent: h, doorCount: doorCount, lightCount: lightCount}
		RunMenuDynamic(h.g, handler)
		return false, ""
	}
	if _, isPing := item.(*PingTerminalsMenuItem); isPing {
		return false, h.pingNearbyInline()
	}
	// Other items (info/devices) are read-only; any other activation closes the menu
	return true, ""
}

// OnExit is called when the menu is exited.
func (h *MaintenanceMenuHandler) OnExit() {
	h.clearMaintenanceMenuState()
}

// ShouldCloseOnAnyAction returns true if the menu should close on any action.
func (h *MaintenanceMenuHandler) ShouldCloseOnAnyAction() bool {
	return false // Allow activating "Ping nearby terminals"; close via menu/quit key
}

// GetMenuItems returns the menu items for the maintenance menu (Controls or Diagnostics mode).
func (h *MaintenanceMenuHandler) GetMenuItems() []MenuItem {
	if h.mode == maintModeDiagnostics {
		return h.getDiagnosticsMenuItems()
	}
	return h.getControlsMenuItems()
}
