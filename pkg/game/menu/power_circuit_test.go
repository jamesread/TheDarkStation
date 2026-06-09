package menu

import (
	"strings"
	"testing"

	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestApplyCircuitPreset_OffAndOn(t *testing.T) {
	g, _ := makeMenuTestGame(t)
	g.RoomDoorsPowered["RoomA"] = false
	g.RoomCCTVPowered["RoomA"] = false

	ApplyCircuitPreset(g, "RoomA", CircuitFull)
	if !g.RoomDoorsPowered["RoomA"] || !g.RoomCCTVPowered["RoomA"] {
		t.Fatal("ON: expected doors and cctv on")
	}

	help := ApplyCircuitPreset(g, "RoomA", CircuitOff)
	if g.RoomDoorsPowered["RoomA"] || g.RoomCCTVPowered["RoomA"] {
		t.Fatal("OFF: expected doors and cctv off immediately")
	}
	if help == "" {
		t.Fatal("expected shutdown help text")
	}
}

func TestApplyCircuitPreset_OnCancelsPendingOff(t *testing.T) {
	g, _ := makeMenuTestGame(t)
	ApplyCircuitPreset(g, "RoomA", CircuitFull)
	setup.ScheduleRoomPowerOff(g, "RoomA", setup.PowerNowMs())
	if !setup.RoomPowerOffScheduled(g, "RoomA") {
		t.Fatal("expected pending shutdown")
	}
	ApplyCircuitPreset(g, "RoomA", CircuitFull)
	if setup.RoomPowerOffScheduled(g, "RoomA") {
		t.Fatal("ON should cancel pending shutdown")
	}
	if !g.RoomDoorsPowered["RoomA"] || !g.RoomCCTVPowered["RoomA"] {
		t.Fatal("expected circuits on after cancel")
	}
}

func TestApplyCircuitPreset_EssentialStillSupportedInternally(t *testing.T) {
	g, _ := makeMenuTestGame(t)
	ApplyCircuitPreset(g, "RoomA", CircuitEssential)
	if !g.RoomDoorsPowered["RoomA"] || g.RoomCCTVPowered["RoomA"] {
		t.Fatalf("ESSENTIAL: doors on cctv off, got doors=%v cctv=%v", g.RoomDoorsPowered["RoomA"], g.RoomCCTVPowered["RoomA"])
	}
}

func TestCircuitPresetMarkup(t *testing.T) {
	if CircuitPresetMarkup(CircuitOff) != "UNPOWERED{OFF}" {
		t.Fatalf("OFF markup = %q", CircuitPresetMarkup(CircuitOff))
	}
	if CircuitPresetMarkup(CircuitFull) != "POWERED{ON}" {
		t.Fatalf("ON markup = %q", CircuitPresetMarkup(CircuitFull))
	}
}

func TestCircuitPreset_PrevAndNext_offOnOnly(t *testing.T) {
	if CircuitOff.NextPreset() != CircuitFull {
		t.Fatalf("OFF next = %s", CircuitOff.NextPreset())
	}
	if CircuitFull.NextPreset() != CircuitOff {
		t.Fatalf("ON next = %s", CircuitFull.NextPreset())
	}
	if CircuitOff.PrevPreset() != CircuitFull {
		t.Fatalf("OFF prev = %s", CircuitOff.PrevPreset())
	}
	if CircuitFull.PrevPreset() != CircuitOff {
		t.Fatalf("ON prev = %s", CircuitFull.PrevPreset())
	}
}

func TestCurrentCircuitPreset_doorsOnlyReadsAsOn(t *testing.T) {
	g := state.NewGame()
	g.RoomDoorsPowered = map[string]bool{"RoomA": true}
	g.RoomCCTVPowered = map[string]bool{"RoomA": false}
	if CurrentCircuitPreset(g, "RoomA") != CircuitFull {
		t.Fatalf("doors-only should display as ON, got %s", CurrentCircuitPreset(g, "RoomA"))
	}
}

func TestPreviewCircuitShed_matchesShortOut(t *testing.T) {
	g, _ := makeMenuTestGame(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": false}
	g.RoomCCTVPowered = map[string]bool{"RoomA": false, "RoomB": false}

	preview := setup.PreviewShortOutIfOverload(g, "RoomB", true, false)
	_ = preview
	text := PreviewCircuitShed(g, "RoomB", CircuitFull)
	if text == "" {
		t.Fatal("expected preview text")
	}
	if !strings.Contains(text, "Preview:") {
		t.Fatalf("unexpected preview: %q", text)
	}
}

func TestPreviewCircuitToggleImpact_onOffAndOverload(t *testing.T) {
	g, _ := makeMenuTestGame(t)
	gameworld.GetGameData(g.Grid.GetCell(0, 0)).Door = entities.NewDoor("RoomA")
	g.RoomPowerOnline = map[string]bool{"RoomA": false, "RoomB": false}
	g.PowerConsumption = g.CalculatePowerConsumption()

	offImpact := PreviewCircuitToggleImpact(g, "RoomA", CircuitOff)
	if offImpact != "No change to station load" {
		t.Fatalf("OFF from already off: %q", offImpact)
	}

	g.RoomDoorsPowered["RoomA"] = true
	g.RoomCCTVPowered["RoomA"] = true
	g.RoomPowerOnline["RoomA"] = true
	g.PowerConsumption = g.CalculatePowerConsumption()

	offImpact = PreviewCircuitToggleImpact(g, "RoomA", CircuitOff)
	if !strings.Contains(offImpact, "-") || !strings.Contains(offImpact, "usage") {
		t.Fatalf("turn off impact = %q", offImpact)
	}

	g.RoomDoorsPowered["RoomA"] = false
	g.RoomCCTVPowered["RoomA"] = false
	g.RoomPowerOnline["RoomA"] = false
	g.PowerSupply = 100
	g.PowerConsumption = g.CalculatePowerConsumption()

	onImpact := PreviewCircuitToggleImpact(g, "RoomA", CircuitFull)
	if onImpact != "+10w usage" {
		t.Fatalf("turn on impact = %q, want +10w usage", onImpact)
	}

	gen := gameworld.GetGameData(g.Grid.GetCell(0, 0)).Generator
	gen.Trip()
	g.UpdatePowerSupply()
	onImpact = PreviewCircuitToggleImpact(g, "RoomA", CircuitFull)
	want := "+10w, 10w over supply - will trigger overload!"
	if onImpact != want {
		t.Fatalf("overload impact = %q, want %q", onImpact, want)
	}
}

func TestRoomCircuitPresetMenuItem_showsToggleImpact(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	gameworld.GetGameData(g.Grid.GetCell(0, 0)).Door = entities.NewDoor("RoomA")
	g.RoomPowerOnline = map[string]bool{"RoomA": false, "RoomB": false}
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)
	item := &RoomCircuitPresetMenuItem{Parent: h}

	label := item.GetLabel()
	if !strings.Contains(label, "Power Grid:") {
		t.Fatalf("label = %q", label)
	}
	if !strings.Contains(label, "+10w usage") {
		t.Fatalf("label should preview turn-on impact, got %q", label)
	}
	if item.GetHelpText() != "+10w usage" {
		t.Fatalf("help = %q", item.GetHelpText())
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
	if strings.Contains(controls, "Refresh power grid") {
		t.Fatal("controls should not include Refresh power grid")
	}
	if !strings.Contains(controls, "Power Grid") {
		t.Fatal("controls should include Power Grid row")
	}
	if !strings.Contains(controls, "Delayed shutdown") {
		t.Fatal("controls should include Delayed shutdown row")
	}
	if strings.Contains(controls, "Circuit preset") {
		t.Fatal("controls should not use old Circuit preset label")
	}
	if !strings.Contains(controls, "1/2=apply") {
		t.Fatal("Power Grid row should show 1/2 apply hint")
	}

	h.mode = maintModeDiagnostics
	diag := strings.Join(labels(h.getDiagnosticsMenuItems()), "\n")
	if !strings.Contains(diag, "LOG\tT+") {
		t.Fatal("diagnostics should include instrument strata")
	}
	if !strings.Contains(diag, "Refresh power grid") {
		t.Fatal("diagnostics should include Refresh power grid")
	}
}

func labels(items []MenuItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.GetLabel()
	}
	return out
}
