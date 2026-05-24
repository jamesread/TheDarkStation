package menu

import (
	"fmt"
	"strings"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/setup"
	gameworld "darkstation/pkg/game/world"
)

const (
	maintModeControls    = "controls"
	maintModeDiagnostics = "diagnostics"
)

// MaintenanceMenuExtraInput is implemented by MaintenanceMenuHandler for map keys and mode toggle.
type MaintenanceMenuExtraInput interface {
	HandleMaintenanceIntent(intent engineinput.Intent) (consumed bool, helpText string)
}

// ModeToggleMenuItem switches Controls / Diagnostics.
type ModeToggleMenuItem struct {
	Parent *MaintenanceMenuHandler
}

func (m *ModeToggleMenuItem) GetLabel() string {
	if m.Parent.mode == maintModeDiagnostics {
		return "Back to controls"
	}
	return "Diagnostics…"
}

func (m *ModeToggleMenuItem) IsSelectable() bool { return true }

func (m *ModeToggleMenuItem) GetHelpText() string {
	return "Press Enter or Tab to switch panel"
}

// RoomCircuitPresetMenuItem cycles and applies OFF / ESSENTIAL / FULL for the viewed room.
type RoomCircuitPresetMenuItem struct {
	Parent *MaintenanceMenuHandler
}

func (r *RoomCircuitPresetMenuItem) GetLabel() string {
	preset := CurrentCircuitPreset(r.Parent.g, r.Parent.selectedRoomName)
	return fmt.Sprintf("Circuit preset:\tACTION{%s}\t(Enter=cycle, 1/2/3=apply)", preset)
}

func (r *RoomCircuitPresetMenuItem) IsSelectable() bool {
	return roomMaintenanceTerminalPowered(r.Parent.g, r.Parent.selectedRoomName)
}

func (r *RoomCircuitPresetMenuItem) GetHelpText() string {
	if !roomMaintenanceTerminalPowered(r.Parent.g, r.Parent.selectedRoomName) {
		return "Activate this room's maintenance terminal first"
	}
	next := CurrentCircuitPreset(r.Parent.g, r.Parent.selectedRoomName).NextPreset()
	return PreviewCircuitShed(r.Parent.g, r.Parent.selectedRoomName, next)
}

// RestoreAllAdjacentMenuItem powers terminals in all adjacent rooms (legacy bulk restore).
type RestoreAllAdjacentMenuItem struct {
	Parent *MaintenanceMenuHandler
}

func (r *RestoreAllAdjacentMenuItem) GetLabel() string {
	return "Restore routing mesh"
}

func (r *RestoreAllAdjacentMenuItem) IsSelectable() bool { return true }

func (r *RestoreAllAdjacentMenuItem) GetHelpText() string {
	return "Press Enter to restore terminals reachable via powered doors and closed relays"
}

// RestoreSelectedRoomMenuItem powers terminals only in the currently viewed room.
type RestoreSelectedRoomMenuItem struct {
	Parent *MaintenanceMenuHandler
}

func (r *RestoreSelectedRoomMenuItem) GetLabel() string {
	return fmt.Sprintf("Restore terminals in:\t%s", r.Parent.selectedRoomName)
}

func (r *RestoreSelectedRoomMenuItem) IsSelectable() bool { return true }

func (r *RestoreSelectedRoomMenuItem) GetHelpText() string {
	return "Press Enter to restore power to unpowered terminals in the viewed room only"
}

// AdvancedPowerMenuItem opens granular door/light/CCTV toggles (diagnostics).
type AdvancedPowerMenuItem struct {
	Parent *MaintenanceMenuHandler
}

func (a *AdvancedPowerMenuItem) GetLabel() string {
	return "Advanced: door / light / CCTV toggles"
}

func (a *AdvancedPowerMenuItem) IsSelectable() bool { return true }

func (a *AdvancedPowerMenuItem) GetHelpText() string {
	return "Press Enter for per-system toggles"
}

// AdvancedPowerMenuHandler is a short sub-menu for granular toggles.
type AdvancedPowerMenuHandler struct {
	parent     *MaintenanceMenuHandler
	doorCount  int
	lightCount int
}

func (h *AdvancedPowerMenuHandler) GetTitle() string {
	return fmt.Sprintf("Advanced power: %s", h.parent.selectedRoomName)
}

func (h *AdvancedPowerMenuHandler) GetInstructions(selected MenuItem) string {
	return "Press Enter to toggle. Escape or Menu to return."
}

func (h *AdvancedPowerMenuHandler) OnSelect(item MenuItem, index int) {}

func (h *AdvancedPowerMenuHandler) OnActivate(item MenuItem, index int) (bool, string) {
	return h.parent.OnActivate(item, index)
}

func (h *AdvancedPowerMenuHandler) OnExit() {}

func (h *AdvancedPowerMenuHandler) ShouldCloseOnAnyAction() bool { return false }

func (h *AdvancedPowerMenuHandler) GetMaintenanceRoom(selectedIndex int, items []MenuItem) string {
	return h.parent.selectedRoomName
}

func (h *AdvancedPowerMenuHandler) GetMenuItems() []MenuItem {
	room := h.parent.selectedRoomName
	g := h.parent.g
	items := []MenuItem{
		&InfoMenuItem{Label: fmt.Sprintf("Room: %s", room)},
		&InfoMenuItem{Label: ""},
	}
	g.Grid.ForEachCell(func(row, col int, c *world.Cell) {
		if c == nil || !c.Room || c.Name != room {
			return
		}
		data := gameworld.GetGameData(c)
		if data.MaintenanceTerm == nil || data.MaintenanceTerm == h.parent.maintenanceTerm {
			return
		}
		items = append(items, &MaintenanceTerminalPowerMenuItem{G: g, Term: data.MaintenanceTerm})
	})
	items = append(items,
		&RoomPowerToggleMenuItem{G: g, RoomName: room, PowerType: "doors", Count: h.doorCount},
		&RoomPowerToggleMenuItem{G: g, RoomName: room, PowerType: "lights", Count: h.lightCount, CountSuffix: " cells"},
		&RoomPowerToggleMenuItem{G: g, RoomName: room, PowerType: "cctv"},
		&InfoMenuItem{Label: ""},
		&CloseMenuItem{Label: "Back"},
	)
	return items
}

func (h *MaintenanceMenuHandler) initMaintenanceMenuState() {
	h.mode = maintModeControls
	h.g.MaintenanceMenuMode = maintModeControls
	h.g.MaintenanceSelectableRooms = append([]string(nil), h.selectableRooms...)
}

func (h *MaintenanceMenuHandler) clearMaintenanceMenuState() {
	h.g.MaintenanceMenuMode = ""
	h.g.MaintenanceSelectableRooms = nil
}

func (h *MaintenanceMenuHandler) toggleMode() {
	if h.mode == maintModeDiagnostics {
		h.mode = maintModeControls
	} else {
		h.mode = maintModeDiagnostics
	}
	h.g.MaintenanceMenuMode = h.mode
}

func (h *MaintenanceMenuHandler) cycleRoom(delta int) {
	if len(h.selectableRooms) == 0 {
		return
	}
	idx := 0
	for i, name := range h.selectableRooms {
		if name == h.selectedRoomName {
			idx = i
			break
		}
	}
	idx += delta
	n := len(h.selectableRooms)
	idx = ((idx % n) + n) % n
	h.selectedRoomName = h.selectableRooms[idx]
}

func (h *MaintenanceMenuHandler) applyCircuitPreset(preset CircuitPreset) string {
	if !roomMaintenanceTerminalPowered(h.g, h.selectedRoomName) {
		return "Activate this room's maintenance terminal first"
	}
	return ApplyCircuitPreset(h.g, h.selectedRoomName, preset)
}

func (h *MaintenanceMenuHandler) restoreAllAdjacent() string {
	rooms := setup.RoomsReachableInPowerMesh(h.g, h.cell)
	if len(rooms) == 0 {
		rooms = []string{h.terminalRoomName}
	}
	roomSet := make(map[string]bool)
	for _, r := range rooms {
		roomSet[r] = true
	}
	_, msg := setup.RestoreTerminalsInRooms(h.g, roomSet)
	return msg
}

func (h *MaintenanceMenuHandler) restoreSelectedRoom() string {
	reachable := setup.RoomsReachableInPowerMesh(h.g, h.cell)
	for _, r := range reachable {
		if r == h.selectedRoomName {
			_, msg := setup.RestoreTerminalsInRooms(h.g, map[string]bool{h.selectedRoomName: true})
			return msg
		}
	}
	return "Selected room not on routing mesh — power doors and close relays first"
}

func (h *MaintenanceMenuHandler) pingNearbyInline() string {
	centerRow, centerCol := h.cell.Row, h.cell.Col
	radiusSq := pingRadius * pingRadius
	var names []string

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
			names = append(names, fmt.Sprintf("%s (CCTV)", data.Terminal.Name))
		}
		if gameworld.HasPuzzle(c) && data.Puzzle != nil {
			names = append(names, fmt.Sprintf("%s (Puzzle)", data.Puzzle.Name))
		}
	})
	if len(names) == 0 {
		return "Ping: no new terminals within range"
	}
	return fmt.Sprintf("Ping: discovered %d — %s", len(names), strings.Join(names, "; "))
}

// HandleMaintenanceIntent handles A/D room cycle, Tab mode toggle, and 1/2/3 circuit presets.
func (h *MaintenanceMenuHandler) HandleMaintenanceIntent(intent engineinput.Intent) (bool, string) {
	switch intent.Action {
	case engineinput.ActionMoveWest:
		h.cycleRoom(-1)
		return true, fmt.Sprintf("Viewing: %s (A/D or arrows to switch)", h.selectedRoomName)
	case engineinput.ActionMoveEast:
		h.cycleRoom(1)
		return true, fmt.Sprintf("Viewing: %s (A/D or arrows to switch)", h.selectedRoomName)
	case engineinput.ActionMaintModeToggle:
		h.toggleMode()
		if h.mode == maintModeDiagnostics {
			return true, "Diagnostics panel — Tab or Back to return to controls"
		}
		return true, "Controls panel — Tab for diagnostics"
	case engineinput.ActionCircuitOff:
		return true, h.applyCircuitPreset(CircuitOff)
	case engineinput.ActionCircuitEssential:
		return true, h.applyCircuitPreset(CircuitEssential)
	case engineinput.ActionCircuitFull:
		return true, h.applyCircuitPreset(CircuitFull)
	default:
		return false, ""
	}
}

func (h *MaintenanceMenuHandler) getControlsMenuItems() []MenuItem {
	flavourLine := deck.TerminalFlavourText(h.g.CurrentDeckID)
	_, roomConsumption, _, _ := buildRoomDevices(h.g, h.selectedRoomName, h.maintenanceTerm)

	items := []MenuItem{
		&InfoMenuItem{Label: "SUBTLE{" + flavourLine + "}"},
		&InfoMenuItem{Label: fmt.Sprintf("Viewing:\tACTION{%s}\t(A/D switch room)", h.selectedRoomName)},
		&InfoMenuItem{Label: ""},
		&InfoMenuItem{Label: fmt.Sprintf("Supply:\t%s", renderer.FormatPowerWatts(h.g.PowerSupply, false))},
		&InfoMenuItem{Label: fmt.Sprintf("Used:\t%s", renderer.FormatPowerWatts(h.g.PowerConsumption, false))},
		&InfoMenuItem{Label: fmt.Sprintf("Free:\t%s", renderer.FormatPowerWatts(h.g.GetAvailablePower(), false))},
		&InfoMenuItem{Label: fmt.Sprintf("Room load:\t%s", renderer.FormatPowerWatts(roomConsumption, false))},
		&InfoMenuItem{Label: ""},
		&RoomCircuitPresetMenuItem{Parent: h},
		&RestoreAllAdjacentMenuItem{Parent: h},
		&RestoreSelectedRoomMenuItem{Parent: h},
		&PingTerminalsMenuItem{},
		&ModeToggleMenuItem{Parent: h},
		&InfoMenuItem{Label: ""},
		&CloseMenuItem{Label: "Close"},
	}
	return items
}

func (h *MaintenanceMenuHandler) getDiagnosticsMenuItems() []MenuItem {
	devices, roomConsumption, _, _ := buildRoomDevices(h.g, h.selectedRoomName, h.maintenanceTerm)
	flavourLine := deck.TerminalFlavourText(h.g.CurrentDeckID)
	instrLines := maintenanceInstrumentMenuLines(h.g, h.selectedRoomName)

	items := []MenuItem{
		&InfoMenuItem{Label: "SUBTLE{" + flavourLine + "}"},
		&InfoMenuItem{Label: fmt.Sprintf("Viewing:\t%s", h.selectedRoomName)},
		&InfoMenuItem{Label: ""},
	}
	for _, line := range instrLines {
		items = append(items, &InfoMenuItem{Label: line})
	}
	if len(instrLines) > 0 {
		items = append(items, &InfoMenuItem{Label: ""})
	}
	items = append(items,
		&InfoMenuItem{Label: fmt.Sprintf("Room (%d devices):", len(devices))},
	)
	if len(devices) == 0 {
		items = append(items, &InfoMenuItem{Label: "No active devices in this room."})
	} else {
		for _, device := range devices {
			items = append(items, &DeviceMenuItem{Device: device})
		}
	}
	items = append(items,
		&InfoMenuItem{Label: fmt.Sprintf("Consumption:\t%s", renderer.FormatPowerWatts(roomConsumption, false))},
		&InfoMenuItem{Label: ""},
		&AdvancedPowerMenuItem{Parent: h},
		&ModeToggleMenuItem{Parent: h},
		&InfoMenuItem{Label: ""},
		&CloseMenuItem{Label: "Close"},
	)
	return items
}
