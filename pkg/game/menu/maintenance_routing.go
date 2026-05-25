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

// ViewingRoomMenuItem is the selectable row showing which room is targeted; A/D cycles when multiple rooms exist.
type ViewingRoomMenuItem struct {
	Parent *MaintenanceMenuHandler
}

func (v *ViewingRoomMenuItem) GetLabel() string {
	label := fmt.Sprintf("Viewing:\tACTION{%s}", v.Parent.selectedRoomName)
	if v.Parent.canCycleRooms() {
		return label + "\tSUBTLE{◀ A/D ▶}"
	}
	return label + "\tSUBTLE{(only room)}"
}

func (v *ViewingRoomMenuItem) IsSelectable() bool { return true }

func (v *ViewingRoomMenuItem) GetHelpText() string {
	if !v.Parent.canCycleRooms() {
		return "Only this room is selectable from here"
	}
	return "A/D or Enter: switch viewed room"
}

func (v *ViewingRoomMenuItem) CanCycle() bool {
	return v.Parent.canCycleRooms()
}

func (v *ViewingRoomMenuItem) HandleCycle(delta int) (bool, string) {
	msg, ok := v.Parent.cycleRoomMessage(delta)
	return ok, msg
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
	if r.IsSelectable() {
		return fmt.Sprintf("Circuit preset:\tACTION{%s}\tSUBTLE{◀ A/D ▶}\t(Enter=cycle, 1/2/3=apply)", preset)
	}
	return fmt.Sprintf("Circuit preset:\tACTION{%s}\t(1/2/3=apply)", preset)
}

func (r *RoomCircuitPresetMenuItem) IsSelectable() bool {
	return canToggleRoomPower(r.Parent.g, r.Parent.terminalRoomName, r.Parent.selectedRoomName)
}

func (r *RoomCircuitPresetMenuItem) GetHelpText() string {
	if !canToggleRoomPower(r.Parent.g, r.Parent.terminalRoomName, r.Parent.selectedRoomName) {
		if r.Parent.terminalRoomName != "" && r.Parent.terminalRoomName != r.Parent.selectedRoomName {
			return "No control path to this room from here"
		}
		return "Activate this room's maintenance terminal first"
	}
	next := CurrentCircuitPreset(r.Parent.g, r.Parent.selectedRoomName).NextPreset()
	return PreviewCircuitShed(r.Parent.g, r.Parent.selectedRoomName, next)
}

func (r *RoomCircuitPresetMenuItem) CanCycle() bool {
	return r.IsSelectable()
}

func (r *RoomCircuitPresetMenuItem) HandleCycle(delta int) (bool, string) {
	if !r.IsSelectable() {
		return false, ""
	}
	preset := CurrentCircuitPreset(r.Parent.g, r.Parent.selectedRoomName)
	if delta < 0 {
		preset = preset.PrevPreset()
	} else {
		preset = preset.NextPreset()
	}
	return true, r.Parent.applyCircuitPreset(preset)
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
		if data.MaintenanceTerm == nil {
			return
		}
		items = append(items, &MaintenanceTerminalPowerMenuItem{G: g, Term: data.MaintenanceTerm})
	})
	items = append(items,
		&RoomPowerToggleMenuItem{G: g, RoomName: room, ControllerRoom: h.parent.terminalRoomName, PowerType: "doors", Count: h.doorCount},
		&RoomPowerToggleMenuItem{G: g, RoomName: room, ControllerRoom: h.parent.terminalRoomName, PowerType: "lights", Count: h.lightCount, CountSuffix: " cells"},
		&RoomPowerToggleMenuItem{G: g, RoomName: room, ControllerRoom: h.parent.terminalRoomName, PowerType: "cctv"},
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

func (h *MaintenanceMenuHandler) canCycleRooms() bool {
	return len(h.selectableRooms) > 1
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

// cycleRoomMessage advances the viewed room when possible and returns feedback text.
func (h *MaintenanceMenuHandler) cycleRoomMessage(delta int) (string, bool) {
	if !h.canCycleRooms() {
		return "", false
	}
	h.cycleRoom(delta)
	return fmt.Sprintf("Viewing: %s", h.selectedRoomName), true
}

func (h *MaintenanceMenuHandler) applyCircuitPreset(preset CircuitPreset) string {
	if !canToggleRoomPower(h.g, h.terminalRoomName, h.selectedRoomName) {
		if h.terminalRoomName != "" && h.terminalRoomName != h.selectedRoomName {
			return "No control path to this room from here"
		}
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

// HandleMaintenanceIntent handles Tab mode toggle and 1/2/3 circuit preset shortcuts.
func (h *MaintenanceMenuHandler) HandleMaintenanceIntent(intent engineinput.Intent) (bool, string) {
	switch intent.Action {
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
		&ViewingRoomMenuItem{Parent: h},
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
		&ViewingRoomMenuItem{Parent: h},
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

// HandleMaintenanceIntent delegates room cycling to the parent maintenance handler (advanced sub-menu).
func (h *AdvancedPowerMenuHandler) HandleMaintenanceIntent(intent engineinput.Intent) (bool, string) {
	if h.parent == nil {
		return false, ""
	}
	return h.parent.HandleMaintenanceIntent(intent)
}
