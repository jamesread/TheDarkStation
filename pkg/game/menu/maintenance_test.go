package menu

import (
	"strings"
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// makeMenuTestGame creates a game with a 3x3 grid, two rooms ("RoomA", "RoomB") and a corridor.
// RoomA at (0,0)-(0,1), Corridor at (1,0)-(1,1), RoomB at (2,0)-(2,1).
// Each room has a powered maintenance terminal. Generator in RoomA provides 100W supply.
func makeMenuTestGame(t *testing.T) (*state.Game, *world.Cell) {
	t.Helper()
	g := state.NewGame()
	grid := world.NewGrid(3, 2)
	grid.MarkAsRoomWithName(0, 0, "RoomA", "desc")
	grid.MarkAsRoomWithName(0, 1, "RoomA", "desc")
	grid.MarkAsRoomWithName(1, 0, "Corridor", "desc")
	grid.MarkAsRoomWithName(1, 1, "Corridor", "desc")
	grid.MarkAsRoomWithName(2, 0, "RoomB", "desc")
	grid.MarkAsRoomWithName(2, 1, "RoomB", "desc")
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(2, 1)
	grid.BuildAllCellConnections()
	g.Grid = grid

	for r := 0; r < 3; r++ {
		for c := 0; c < 2; c++ {
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}

	// Maintenance terminals (powered)
	termA := entities.NewMaintenanceTerminal("MT-A", "RoomA")
	termA.Powered = true
	gameworld.GetGameData(grid.GetCell(0, 1)).MaintenanceTerm = termA

	termB := entities.NewMaintenanceTerminal("MT-B", "RoomB")
	termB.Powered = true
	gameworld.GetGameData(grid.GetCell(2, 1)).MaintenanceTerm = termB

	// Generator in RoomA for 100W supply and power grid feed
	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteriesAndStart(1)
	g.AddGenerator(gen)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.CurrentDeckID = 0
	g.UpdatePowerSupply()

	g.RoomDoorsPowered = map[string]bool{"RoomA": false, "RoomB": false}
	g.RoomCCTVPowered = map[string]bool{"RoomA": false, "RoomB": false}
	g.RoomLightsPowered = map[string]bool{"RoomA": true, "RoomB": true}

	termCell := grid.GetCell(0, 1)
	return g, termCell
}

func TestToggleDoors_OnOff(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)

	toggle := &RoomPowerToggleMenuItem{G: g, RoomName: "RoomA", PowerType: "doors"}

	// Toggle ON
	h.OnActivate(toggle, 0)
	if !g.RoomDoorsPowered["RoomA"] {
		t.Error("after toggle ON: RoomDoorsPowered[RoomA] should be true")
	}

	// Toggle OFF
	h.OnActivate(toggle, 0)
	if g.RoomDoorsPowered["RoomA"] {
		t.Error("after toggle OFF: RoomDoorsPowered[RoomA] should be false")
	}
}

func TestToggleCCTV_OnOff(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)

	toggle := &RoomPowerToggleMenuItem{G: g, RoomName: "RoomA", PowerType: "cctv"}

	h.OnActivate(toggle, 0)
	if !g.RoomCCTVPowered["RoomA"] {
		t.Error("after toggle ON: RoomCCTVPowered[RoomA] should be true")
	}

	h.OnActivate(toggle, 0)
	if g.RoomCCTVPowered["RoomA"] {
		t.Error("after toggle OFF: RoomCCTVPowered[RoomA] should be false")
	}
}

func TestToggleDoorsON_ShortOutProtectsToggledRoom(t *testing.T) {
	// 100W supply (1 generator). RoomA preloaded with heavy CCTV consumption.
	// Toggle RoomB doors ON: ShortOutIfOverload protects RoomB and shorts out RoomA first.
	g := state.NewGame()
	grid := world.NewGrid(6, 10)
	for r := 0; r < 3; r++ {
		for c := 0; c < 10; c++ {
			grid.MarkAsRoomWithName(r, c, "RoomA", "desc")
		}
	}
	for r := 3; r < 6; r++ {
		for c := 0; c < 10; c++ {
			grid.MarkAsRoomWithName(r, c, "RoomB", "desc")
		}
	}
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(5, 9)
	grid.BuildAllCellConnections()
	g.Grid = grid
	for r := 0; r < 6; r++ {
		for c := 0; c < 10; c++ {
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}
	// Maintenance terminals (powered)
	termA := entities.NewMaintenanceTerminal("MT-A", "RoomA")
	termA.Powered = true
	gameworld.GetGameData(grid.GetCell(0, 0)).MaintenanceTerm = termA
	termB := entities.NewMaintenanceTerminal("MT-B", "RoomB")
	termB.Powered = true
	gameworld.GetGameData(grid.GetCell(3, 0)).MaintenanceTerm = termB

	// One door per room (10w each when powered)
	gameworld.GetGameData(grid.GetCell(0, 1)).Door = &entities.Door{RoomName: "RoomA", Locked: false}
	gameworld.GetGameData(grid.GetCell(3, 1)).Door = &entities.Door{RoomName: "RoomB", Locked: false}
	// Heavy RoomA CCTV load: 12 terminals = 120w when CCTV powered
	placed := 0
	for r := 0; r < 3 && placed < 12; r++ {
		for c := 0; c < 10 && placed < 12; c++ {
			gameworld.GetGameData(grid.GetCell(r, c)).Terminal = entities.NewCCTVTerminal("TA")
			placed++
		}
	}

	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteriesAndStart(1)
	g.AddGenerator(gen)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.UpdatePowerSupply() // 100W

	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": false}
	g.RoomCCTVPowered = map[string]bool{"RoomA": true, "RoomB": false}
	g.RoomLightsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	g.RoomPowerOnline = map[string]bool{"RoomA": true}
	g.PowerConsumption = g.CalculatePowerConsumption()

	// Arm RoomB doors; overload trips when propagated power reaches RoomB.
	termCell := grid.GetCell(3, 0)
	h := NewMaintenanceMenuHandler(g, termCell, termB)
	toggleB := &RoomPowerToggleMenuItem{G: g, RoomName: "RoomB", PowerType: "doors"}
	h.OnActivate(toggleB, 0)

	if !g.RoomDoorsPowered["RoomB"] {
		t.Error("protected room RoomB: doors should remain armed after short-out")
	}
	if g.RoomDoorsPowered["RoomA"] {
		t.Error("RoomA doors should be shorted out (overload)")
	}
	if g.RoomCCTVPowered["RoomA"] {
		t.Error("RoomA CCTV should be shorted out (overload)")
	}
	if g.PowerConsumption > g.PowerSupply {
		t.Errorf("consumption (%d) should be <= supply (%d) after short-out", g.PowerConsumption, g.PowerSupply)
	}
}

func TestToggleDoorsOFF_RecalculatesConsumption(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)

	// Place door for RoomA
	gameworld.GetGameData(g.Grid.GetCell(1, 0)).Door = &entities.Door{RoomName: "RoomA", Locked: false}

	// Power RoomA doors (armed + online)
	g.RoomDoorsPowered["RoomA"] = true
	setup.EnergizeArmedRoomsForTest(g)
	g.PowerConsumption = g.CalculatePowerConsumption()
	if g.PowerConsumption == 0 {
		t.Fatal("precondition: consumption should be > 0 with doors on")
	}

	// Toggle OFF
	toggle := &RoomPowerToggleMenuItem{G: g, RoomName: "RoomA", PowerType: "doors"}
	h.OnActivate(toggle, 0)

	if g.RoomDoorsPowered["RoomA"] {
		t.Error("RoomA doors should be OFF after toggle")
	}
	if g.PowerConsumption != 0 {
		t.Errorf("consumption should be 0 after turning off all doors, got %d", g.PowerConsumption)
	}
}

func TestToggleCCTVON_OverloadPersistsInProtectedRoom_ShowsWarning(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 12)
	for c := 0; c < 12; c++ {
		grid.MarkAsRoomWithName(0, c, "RoomA", "desc")
		gameworld.InitGameData(grid.GetCell(0, c))
		gameworld.GetGameData(grid.GetCell(0, c)).Terminal = entities.NewCCTVTerminal("TA")
	}
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(0, 11)
	grid.BuildAllCellConnections()
	g.Grid = grid

	termA := entities.NewMaintenanceTerminal("MT-A", "RoomA")
	termA.Powered = true
	termCell := grid.GetCell(0, 0)
	gameworld.GetGameData(termCell).MaintenanceTerm = termA

	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteriesAndStart(1)
	g.AddGenerator(gen)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	gameworld.GetGameData(grid.GetCell(0, 0)).Door = &entities.Door{RoomName: "RoomA", Locked: false}
	g.CurrentDeckID = 0
	g.UpdatePowerSupply()

	g.RoomDoorsPowered = map[string]bool{"RoomA": true}
	g.RoomCCTVPowered = map[string]bool{"RoomA": false}
	g.RoomLightsPowered = map[string]bool{"RoomA": true}
	setup.EnergizeArmedRoomsForTest(g)
	g.PowerConsumption = setup.CalculatePowerConsumption(g)

	h := NewMaintenanceMenuHandler(g, termCell, termA)
	toggle := &RoomPowerToggleMenuItem{G: g, RoomName: "RoomA", PowerType: "cctv"}
	_, helpText := h.OnActivate(toggle, 0)

	if !g.RoomCCTVPowered["RoomA"] {
		t.Error("RoomA CCTV should remain ON (protected room)")
	}
	g.PowerConsumption = setup.CalculatePowerConsumption(g)
	if g.PowerConsumption <= g.PowerSupply {
		t.Fatalf("expected persistent overload, got consumption=%d supply=%d", g.PowerConsumption, g.PowerSupply)
	}
	if helpText != "Power overload persists in this room. Reduce load." {
		t.Errorf("helpText = %q, want persistent-overload warning", helpText)
	}
}

func TestToggleIsSelectable_RequiresMaintTerminalPowered(t *testing.T) {
	g, termCell := makeMenuTestGame(t)

	// Unpower the terminal
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	term.Powered = false

	toggle := &RoomPowerToggleMenuItem{G: g, RoomName: "RoomA", PowerType: "doors"}
	if toggle.IsSelectable() {
		t.Error("toggle should NOT be selectable when room's maintenance terminal is unpowered")
	}

	// Re-power the terminal
	term.Powered = true
	if !toggle.IsSelectable() {
		t.Error("toggle should be selectable when room's maintenance terminal is powered")
	}
}

func TestToggleOnActivate_RejectsUnpoweredTerminal(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)

	// Unpower the target room's terminal
	term.Powered = false

	toggle := &RoomPowerToggleMenuItem{G: g, RoomName: "RoomA", PowerType: "doors"}
	_, helpText := h.OnActivate(toggle, 0)

	if g.RoomDoorsPowered["RoomA"] {
		t.Error("doors should NOT change when terminal is unpowered")
	}
	if helpText == "" {
		t.Error("should return help text explaining terminal must be activated")
	}
}

func TestRefreshPowerGrid_PowersAdjacentRoom(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	termA := gameworld.GetGameData(termCell).MaintenanceTerm
	termA.Powered = true // We're at RoomA's terminal
	g.RoomDoorsPowered["RoomA"] = true
	g.RoomDoorsPowered["RoomB"] = true
	g.RoomDoorsPowered["Corridor"] = true
	setup.EnergizeArmedRoomsForTest(g)

	termBCell := g.Grid.GetCell(2, 1)
	termB := gameworld.GetGameData(termBCell).MaintenanceTerm
	termB.Powered = false // RoomB terminal starts unpowered

	h := NewMaintenanceMenuHandler(g, termCell, termA)
	restoreItem := &RefreshPowerGridMenuItem{Parent: h}

	_, helpText := h.OnActivate(restoreItem, 0)

	if !termB.Powered {
		t.Error("RoomB terminal should be powered after restore")
	}
	if !strings.Contains(helpText, "1 terminal") {
		t.Errorf("helpText = %q, want message mentioning 1 terminal", helpText)
	}
}

func TestRefreshPowerGrid_NoUnpoweredShowsMessage(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)
	restoreItem := &RefreshPowerGridMenuItem{Parent: h}

	// Both terminals already powered in makeMenuTestGame
	_, helpText := h.OnActivate(restoreItem, 0)

	if !strings.Contains(helpText, "No unpowered") {
		t.Errorf("helpText = %q, want no-unpowered message", helpText)
	}
}

func TestRefreshPowerGrid_PowersOwnRoomUnpoweredTerminal(t *testing.T) {
	// RoomA has 2 terminals: one powered (we're using it), one unpowered.
	// Refresh should power the other terminal in own room.
	g := state.NewGame()
	grid := world.NewGrid(2, 3)
	grid.MarkAsRoomWithName(0, 0, "RoomA", "desc")
	grid.MarkAsRoomWithName(0, 1, "RoomA", "desc")
	grid.MarkAsRoomWithName(0, 2, "RoomA", "desc")
	grid.SetStartCellAt(0, 0)
	grid.BuildAllCellConnections()
	g.Grid = grid

	for r := 0; r < 2; r++ {
		for c := 0; c < 3; c++ {
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}

	term1 := entities.NewMaintenanceTerminal("MT-1", "RoomA")
	term1.Powered = true
	term2 := entities.NewMaintenanceTerminal("MT-2", "RoomA")
	term2.Powered = false

	gameworld.GetGameData(grid.GetCell(0, 0)).MaintenanceTerm = term1
	gameworld.GetGameData(grid.GetCell(0, 1)).MaintenanceTerm = term2

	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.AddGenerator(gen)
	g.RoomDoorsPowered["RoomA"] = true
	setup.EnergizeArmedRoomsForTest(g)

	termCell := grid.GetCell(0, 0)
	h := NewMaintenanceMenuHandler(g, termCell, term1)
	restoreItem := &RefreshPowerGridMenuItem{Parent: h}

	_, helpText := h.OnActivate(restoreItem, 0)

	if !term2.Powered {
		t.Error("MT-2 in own room should be powered after restore")
	}
	if !strings.Contains(helpText, "1 terminal") {
		t.Errorf("helpText = %q, want message mentioning 1 terminal", helpText)
	}
}

func TestMaintenanceInstrumentMenuLines_deterministic(t *testing.T) {
	g, _ := makeMenuTestGame(t)
	g.Level = 5
	g.CurrentDeckID = 4
	g.LevelSeed = 424242

	a := strings.Join(maintenanceInstrumentMenuLines(g, "RoomA"), "\n")
	b := strings.Join(maintenanceInstrumentMenuLines(g, "RoomA"), "\n")
	if a != b {
		t.Fatalf("instrument lines not deterministic")
	}
	if !strings.Contains(a, "SUBSYS-\tBUS-") || !strings.Contains(a, "LOG\tT+") {
		t.Fatalf("expected trace lines, got:\n%s", a)
	}
}

func TestMaintenanceInstrumentMenuLines_localCorrelates(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(3, 2)
	grid.MarkAsRoomWithName(0, 0, "RoomA", "desc")
	grid.MarkAsRoomWithName(0, 1, "RoomA", "desc")
	grid.MarkAsRoomWithName(1, 0, "Corridor", "ROOM_CORRIDOR")
	grid.MarkAsRoomWithName(1, 1, "Corridor", "ROOM_CORRIDOR")
	grid.MarkAsRoomWithName(2, 0, "RoomB", "desc")
	grid.MarkAsRoomWithName(2, 1, "RoomB", "desc")
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(2, 1)
	grid.BuildAllCellConnections()
	g.Grid = grid
	g.Level = 5
	g.CurrentDeckID = 2

	for r := 0; r < 3; r++ {
		for c := 0; c < 2; c++ {
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}

	puz := entities.NewPuzzleTerminal("P", entities.PuzzleSequence, "2-4-6-8", "", entities.RewardNone, "seq")
	puz.LinkageToken = "LINK-MHOP-A"
	gameworld.GetGameData(grid.GetCell(0, 0)).Puzzle = puz

	corr := grid.GetCell(1, 0)
	cgd := gameworld.GetGameData(corr)
	cgd.LinkageTag = "LINK-MHOP-A"
	cgd.EnvPlaqueMsgID = "ENV_PLAQUE_LINK_MHOP_A"

	lines := maintenanceInstrumentMenuLines(g, "RoomA")
	got := strings.Join(lines, "\n")
	for _, needle := range []string{"XCORE-", "JNCT-", "ENVREF-", "LINK-MHOP-A", "ENV_PLAQUE_LINK_MHOP_A"} {
		if !strings.Contains(got, needle) {
			t.Fatalf("missing %q in:\n%s", needle, got)
		}
	}
}

func TestAdvancedPowerMenuItems_includeCurrentMaintTerminal(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)
	adv := &AdvancedPowerMenuHandler{parent: h, doorCount: 0, lightCount: 2}
	items := adv.GetMenuItems()
	var foundCurrent bool
	for _, it := range items {
		if termItem, ok := it.(*MaintenanceTerminalPowerMenuItem); ok && termItem.Term == term {
			foundCurrent = true
			if !strings.Contains(termItem.GetLabel(), "POWERED") {
				t.Fatalf("label = %q, want powered maint terminal", termItem.GetLabel())
			}
			break
		}
	}
	if !foundCurrent {
		t.Fatal("advanced power list should include the maintenance terminal the player is using")
	}
}

func TestMaintenanceMenuItems_includeInstrumentStrata(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	g.Level = 5
	g.CurrentDeckID = 4
	g.LevelSeed = 424242
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)
	h.mode = maintModeDiagnostics
	var joined strings.Builder
	for _, it := range h.GetMenuItems() {
		joined.WriteString(it.GetLabel())
		joined.WriteByte('\n')
	}
	s := joined.String()
	if !strings.Contains(s, "LOG\tT+") || !strings.Contains(s, "SUBSYS-\tBUS-") {
		t.Fatalf("diagnostics menu should include instrument trace lines:\n%s", s)
	}
}

func TestRefreshPowerGrid_PowersGridRoom(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	termA := gameworld.GetGameData(termCell).MaintenanceTerm
	g.RoomDoorsPowered["RoomA"] = true
	g.RoomDoorsPowered["RoomB"] = true
	g.RoomDoorsPowered["Corridor"] = true
	setup.EnergizeArmedRoomsForTest(g)
	termBCell := g.Grid.GetCell(2, 1)
	termB := gameworld.GetGameData(termBCell).MaintenanceTerm
	termB.Powered = false

	h := NewMaintenanceMenuHandler(g, termCell, termA)
	h.selectedRoomName = "RoomB"
	_, helpText := h.OnActivate(&RefreshPowerGridMenuItem{Parent: h}, 0)

	if !termB.Powered {
		t.Error("RoomB terminal should be powered")
	}
	if !strings.Contains(helpText, "1 terminal") {
		t.Errorf("helpText = %q", helpText)
	}
}

func TestViewingRoomMenuItem_HandleCycle(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)
	h.selectedRoomName = "RoomA"

	v := &ViewingRoomMenuItem{Parent: h}
	consumed, _ := v.HandleCycle(1)
	if !consumed {
		t.Fatal("east should cycle viewing room")
	}
	if h.selectedRoomName != "RoomB" {
		t.Fatalf("expected RoomB, got %s", h.selectedRoomName)
	}

	consumed, _ = v.HandleCycle(-1)
	if !consumed || h.selectedRoomName != "RoomA" {
		t.Fatalf("west should cycle back to RoomA, got %s", h.selectedRoomName)
	}
}

func TestViewingRoomMenuItem_singleRoomDoesNotCycle(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)
	h.selectableRooms = []string{"RoomA"}
	h.selectedRoomName = "RoomA"

	v := &ViewingRoomMenuItem{Parent: h}
	consumed, _ := v.HandleCycle(-1)
	if consumed {
		t.Fatal("should not cycle when only one selectable room")
	}
}

func TestRoomCircuitPresetMenuItem_HandleCycle(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)
	h.selectedRoomName = "RoomA"
	g.RoomDoorsPowered["RoomA"] = false
	g.RoomCCTVPowered["RoomA"] = false

	preset := &RoomCircuitPresetMenuItem{Parent: h}
	consumed, _ := preset.HandleCycle(1)
	if !consumed {
		t.Fatal("east should cycle power grid")
	}
	if CurrentCircuitPreset(g, "RoomA") != CircuitFull {
		t.Fatalf("expected ON, got %s", CurrentCircuitPreset(g, "RoomA"))
	}

	consumed, _ = preset.HandleCycle(1)
	if !consumed || CurrentCircuitPreset(g, "RoomA") != CircuitOff {
		t.Fatalf("expected OFF, got %s", CurrentCircuitPreset(g, "RoomA"))
	}

	consumed, _ = preset.HandleCycle(-1)
	if !consumed || CurrentCircuitPreset(g, "RoomA") != CircuitFull {
		t.Fatalf("west should step back to ON, got %s", CurrentCircuitPreset(g, "RoomA"))
	}
}

func TestRoomLabelWithPowerDraw(t *testing.T) {
	g, _ := makeMenuTestGame(t)
	gameworld.GetGameData(g.Grid.GetCell(0, 0)).Door = entities.NewDoor("RoomA")
	gameworld.GetGameData(g.Grid.GetCell(0, 1)).Terminal = entities.NewCCTVTerminal("CCTV-A")
	g.RoomDoorsPowered["RoomA"] = true
	g.RoomCCTVPowered["RoomA"] = true
	setup.EnergizeArmedRoomsForTest(g)

	label := RoomLabelWithPowerDraw(g, "RoomA")
	if label != "RoomA: 20w" {
		t.Fatalf("label = %q, want RoomA: 20w (doors + CCTV)", label)
	}

	g.RoomDoorsPowered["RoomA"] = false
	g.RoomCCTVPowered["RoomA"] = false
	setup.ClearRoomPropagatedPower(g, "RoomA")
	if RoomLabelWithPowerDraw(g, "RoomA") != "RoomA: 0w" {
		t.Fatal("all systems off should show 0w draw")
	}
}

func TestViewingRoomMenuItem_labelShowsSwitchHint(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)

	v := &ViewingRoomMenuItem{Parent: h}
	if !strings.Contains(v.GetLabel(), "A/D") {
		t.Fatalf("multi-room label should show A/D hint: %q", v.GetLabel())
	}

	h.selectableRooms = []string{"RoomA"}
	if strings.Contains((&ViewingRoomMenuItem{Parent: h}).GetLabel(), "A/D") {
		t.Fatal("single-room label should not show A/D hint")
	}
}

func TestOnActivate_viewingRoomCycles(t *testing.T) {
	g, termCell := makeMenuTestGame(t)
	term := gameworld.GetGameData(termCell).MaintenanceTerm
	h := NewMaintenanceMenuHandler(g, termCell, term)
	h.selectedRoomName = "RoomA"

	_, help := h.OnActivate(&ViewingRoomMenuItem{Parent: h}, 0)
	if h.selectedRoomName != "RoomB" {
		t.Fatalf("Enter on viewing row should cycle room, got %s", h.selectedRoomName)
	}
	if help == "" {
		t.Fatal("expected help text after cycle")
	}
}
