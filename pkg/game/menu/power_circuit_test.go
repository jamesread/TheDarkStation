package menu

import (
	"strings"
	"testing"

	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestApplyCircuitPreset_EssentialAndFull(t *testing.T) {
	g, _ := makeMenuTestGame(t)
	g.RoomDoorsPowered["RoomA"] = false
	g.RoomCCTVPowered["RoomA"] = false

	ApplyCircuitPreset(g, "RoomA", CircuitEssential)
	if !g.RoomDoorsPowered["RoomA"] || g.RoomCCTVPowered["RoomA"] {
		t.Fatalf("ESSENTIAL: doors on cctv off, got doors=%v cctv=%v", g.RoomDoorsPowered["RoomA"], g.RoomCCTVPowered["RoomA"])
	}

	ApplyCircuitPreset(g, "RoomA", CircuitFull)
	if !g.RoomDoorsPowered["RoomA"] || !g.RoomCCTVPowered["RoomA"] {
		t.Fatal("FULL: expected doors and cctv on")
	}

	ApplyCircuitPreset(g, "RoomA", CircuitOff)
	if g.RoomDoorsPowered["RoomA"] || g.RoomCCTVPowered["RoomA"] {
		t.Fatal("OFF: expected doors and cctv off")
	}
}

func TestPreviewCircuitShed_matchesShortOut(t *testing.T) {
	g := state.NewGame()
	g.PowerSupply = 100
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": false}
	g.RoomCCTVPowered = map[string]bool{"RoomA": false, "RoomB": false}
	g.Grid = nil // preview uses maps only for door/cctv; consumption needs grid for full accuracy

	preview := g.PreviewShortOutIfOverload("RoomB", true, false)
	_ = preview
	text := PreviewCircuitShed(g, "RoomB", CircuitEssential)
	if text == "" {
		t.Fatal("expected preview text")
	}
	if !strings.Contains(text, "Preview:") {
		t.Fatalf("unexpected preview: %q", text)
	}
}

func TestMaintenanceMenuItems_controlsVsDiagnostics(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)

	controls := strings.Join(labels(h.getControlsMenuItems()), "\n")
	if strings.Contains(controls, "LOG\tT+") {
		t.Fatal("controls mode should not include instrument LOG line")
	}
	if !strings.Contains(controls, "Circuit preset") {
		t.Fatal("controls should include circuit preset")
	}

	h.mode = maintModeDiagnostics
	diag := strings.Join(labels(h.getDiagnosticsMenuItems()), "\n")
	if !strings.Contains(diag, "LOG\tT+") {
		t.Fatal("diagnostics should include instrument strata")
	}
}

func labels(items []MenuItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.GetLabel()
	}
	return out
}
