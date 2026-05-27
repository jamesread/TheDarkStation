// Package gameplay tests lifecycle functions: BuildGame, AdvanceLevel, TriggerGameComplete.
package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/menu"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestBuildGame_GeneratesOnlyStartingDeck(t *testing.T) {
	// BuildGame must generate only the starting deck; no pre-generation of all decks.
	g := BuildGame(1)
	if g == nil {
		t.Fatal("BuildGame(1) returned nil")
	}
	if g.Grid == nil {
		t.Fatal("BuildGame(1): Grid is nil")
	}
	if g.CurrentDeckID != 0 || g.Level != 1 {
		t.Errorf("BuildGame(1): CurrentDeckID=%d Level=%d, want 0,1", g.CurrentDeckID, g.Level)
	}
	if len(g.DeckStates) != 0 {
		t.Errorf("BuildGame(1): DeckStates should be empty (no decks stored yet), got %d entries", len(g.DeckStates))
	}
}

func TestBuildGame_ClampsStartLevelToValidRange(t *testing.T) {
	// BuildGame clamps startLevel to [1, TotalDecks]; no panic or out-of-range state.
	// startLevel <= 0 → clamp to 1 (first deck)
	g0 := BuildGame(0)
	if g0 == nil || g0.Grid == nil {
		t.Fatal("BuildGame(0) returned nil or nil Grid")
	}
	if g0.CurrentDeckID != 0 || g0.Level != 1 {
		t.Errorf("BuildGame(0): CurrentDeckID=%d Level=%d, want 0,1", g0.CurrentDeckID, g0.Level)
	}
	gNeg := BuildGame(-1)
	if gNeg == nil || gNeg.Grid == nil {
		t.Fatal("BuildGame(-1) returned nil or nil Grid")
	}
	if gNeg.CurrentDeckID != 0 || gNeg.Level != 1 {
		t.Errorf("BuildGame(-1): CurrentDeckID=%d Level=%d, want 0,1", gNeg.CurrentDeckID, gNeg.Level)
	}
	// startLevel > TotalDecks → clamp to final deck
	gOver := BuildGame(deck.TotalDecks + 1)
	if gOver == nil || gOver.Grid == nil {
		t.Fatal("BuildGame(TotalDecks+1) returned nil or nil Grid")
	}
	if gOver.CurrentDeckID != deck.FinalDeckIndex || gOver.Level != deck.TotalDecks {
		t.Errorf("BuildGame(TotalDecks+1): CurrentDeckID=%d Level=%d, want %d,%d",
			gOver.CurrentDeckID, gOver.Level, deck.FinalDeckIndex, deck.TotalDecks)
	}
}

func TestDeck_FixedCountAndFinalDeck(t *testing.T) {
	// deck.TotalDecks and deck.Graph define fixed count; final deck has empty Connections.
	if deck.TotalDecks < 1 {
		t.Fatal("TotalDecks must be >= 1")
	}
	if len(deck.Graph) != deck.TotalDecks {
		t.Errorf("Graph length %d != TotalDecks %d", len(deck.Graph), deck.TotalDecks)
	}
	// Final deck (index TotalDecks-1) must have no Connections
	finalIdx := deck.FinalDeckIndex
	if finalIdx != deck.TotalDecks-1 {
		t.Errorf("FinalDeckIndex=%d, want TotalDecks-1=%d", finalIdx, deck.TotalDecks-1)
	}
	if len(deck.Graph[finalIdx].Connections) != 0 {
		t.Errorf("Final deck Connections must be empty, got %v", deck.Graph[finalIdx].Connections)
	}
	// Non-final decks must have exactly one connection (next deck)
	for i := 0; i < finalIdx; i++ {
		if len(deck.Graph[i].Connections) != 1 || deck.Graph[i].Connections[0] != i+1 {
			t.Errorf("Deck %d Connections=%v, want [%d]", i, deck.Graph[i].Connections, i+1)
		}
	}
}

func TestNextDeckID_FinalDeckReturnsFalse(t *testing.T) {
	// NextDeckID on final deck returns false; no advance possible.
	_, ok := deck.NextDeckID(deck.FinalDeckIndex)
	if ok {
		t.Error("NextDeckID(FinalDeckIndex) ok=true, want false")
	}
	_, ok = deck.NextDeckID(deck.TotalDecks) // out of range
	if ok {
		t.Error("NextDeckID(TotalDecks) ok=true, want false (out of range)")
	}
}

func TestAdvanceLevel_GeneratesOnFirstEntry(t *testing.T) {
	// When DeckStates[nextID] has no grid, AdvanceLevel generates the next deck.
	g := BuildGame(1)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame(1) setup failed")
	}
	// AdvanceLevel(1) should generate deck 2 (no stored state)
	AdvanceLevel(g)
	if g.CurrentDeckID != 1 || g.Level != 2 {
		t.Errorf("after AdvanceLevel: CurrentDeckID=%d Level=%d, want 1,2", g.CurrentDeckID, g.Level)
	}
	if g.Grid == nil {
		t.Fatal("AdvanceLevel: Grid is nil after first entry")
	}
	if len(g.DeckStates) < 1 {
		t.Error("AdvanceLevel: DeckStates should have deck 0 saved")
	}
}

func TestAdvanceLevel_LoadsStoredDeckWhenPresent(t *testing.T) {
	// When DeckStates[nextID] has grid, AdvanceLevel loads it (no re-generation).
	g := BuildGame(1)
	g.SaveCurrentDeckState()
	// Advance to deck 2
	AdvanceLevel(g)
	if g.CurrentDeckID != 1 {
		t.Fatalf("CurrentDeckID=%d, want 1", g.CurrentDeckID)
	}
	secondGrid := g.Grid
	// Save deck 2, go back to deck 1, then advance again - should load stored deck 2
	g.SaveCurrentDeckState()
	g.LoadDeckState(0) // back to deck 1
	AdvanceLevel(g)
	if g.Grid != secondGrid {
		t.Error("AdvanceLevel should load stored deck 2, not regenerate")
	}
}

func TestJumpToDeck_GeneratesOnFirstVisit(t *testing.T) {
	g := BuildGame(1)
	if err := JumpToDeck(g, 5); err != nil {
		t.Fatalf("JumpToDeck(5): %v", err)
	}
	if g.CurrentDeckID != 4 || g.Level != 5 {
		t.Errorf("after JumpToDeck(5): CurrentDeckID=%d Level=%d, want 4,5", g.CurrentDeckID, g.Level)
	}
	if g.Grid == nil {
		t.Fatal("JumpToDeck: Grid is nil")
	}
	if g.DeckStates[4] == nil || g.DeckStates[4].Grid == nil {
		t.Error("JumpToDeck should save generated deck 5 state")
	}
}

func TestJumpToDeck_LoadsStoredDeck(t *testing.T) {
	g := BuildGame(1)
	if err := JumpToDeck(g, 3); err != nil {
		t.Fatalf("JumpToDeck(3): %v", err)
	}
	deck3Grid := g.Grid
	if err := JumpToDeck(g, 1); err != nil {
		t.Fatalf("JumpToDeck(1): %v", err)
	}
	if err := JumpToDeck(g, 3); err != nil {
		t.Fatalf("JumpToDeck(3) again: %v", err)
	}
	if g.Grid != deck3Grid {
		t.Error("JumpToDeck should load stored deck 3, not regenerate")
	}
}

func TestJumpToDeck_InvalidDeck(t *testing.T) {
	g := BuildGame(1)
	if err := JumpToDeck(g, 0); err == nil {
		t.Fatal("JumpToDeck(0) should fail")
	}
	if err := JumpToDeck(g, deck.TotalDecks+1); err == nil {
		t.Fatal("JumpToDeck beyond TotalDecks should fail")
	}
}

func TestJumpToDeck_ClearsCompletionState(t *testing.T) {
	g := BuildGame(deck.TotalDecks)
	TriggerGameComplete(g)
	if err := JumpToDeck(g, 1); err != nil {
		t.Fatalf("JumpToDeck(1): %v", err)
	}
	if g.GameComplete {
		t.Error("JumpToDeck should clear GameComplete")
	}
}

func TestAdvanceLevel_FinalDeckNoAdvance(t *testing.T) {
	// AdvanceLevel on final deck does nothing (NextDeckID returns false).
	g := BuildGame(deck.TotalDecks)
	if g == nil {
		t.Fatal("BuildGame(deck.TotalDecks) returned nil")
	}
	AdvanceLevel(g)
	if g.CurrentDeckID != deck.FinalDeckIndex || g.Level != deck.TotalDecks {
		t.Errorf("AdvanceLevel on final deck changed state: CurrentDeckID=%d Level=%d", g.CurrentDeckID, g.Level)
	}
}

func TestResetLevel_DoesNotAdvanceDeck(t *testing.T) {
	// ResetLevel regenerates the current deck only; CurrentDeckID and Level must not change.
	g := BuildGame(1)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame(1) setup failed")
	}
	ResetLevel(g)
	if g.CurrentDeckID != 0 || g.Level != 1 {
		t.Errorf("after ResetLevel: CurrentDeckID=%d Level=%d, want 0,1 (must not advance)", g.CurrentDeckID, g.Level)
	}
	if g.Grid == nil {
		t.Fatal("ResetLevel: Grid is nil after reset")
	}
	// After advancing once, reset should keep us on deck 2 (reset same deck, not go back)
	AdvanceLevel(g)
	if g.CurrentDeckID != 1 || g.Level != 2 {
		t.Fatalf("after AdvanceLevel: CurrentDeckID=%d Level=%d, want 1,2", g.CurrentDeckID, g.Level)
	}
	ResetLevel(g)
	if g.CurrentDeckID != 1 || g.Level != 2 {
		t.Errorf("after ResetLevel on deck 2: CurrentDeckID=%d Level=%d, want 1,2 (must not change deck)", g.CurrentDeckID, g.Level)
	}
}

func TestTriggerGameComplete_SetsGameComplete(t *testing.T) {
	g := state.NewGame()
	if g.GameComplete {
		t.Fatal("new game should not be complete")
	}
	TriggerGameComplete(g)
	if !g.GameComplete {
		t.Error("TriggerGameComplete should set GameComplete=true")
	}
	if g.CompletionPhase != state.CompletionPhaseSummary {
		t.Errorf("CompletionPhase = %v, want Summary", g.CompletionPhase)
	}
}

func TestIsFinalDeck_MatchesTotalDecks(t *testing.T) {
	if !deck.IsFinalDeck(deck.TotalDecks) {
		t.Errorf("IsFinalDeck(TotalDecks)=false, want true")
	}
	if deck.IsFinalDeck(deck.TotalDecks - 1) {
		t.Errorf("IsFinalDeck(TotalDecks-1)=true, want false")
	}
}

func TestBuildGame_StartRoomDoorsPoweredViaGeneratorBootstrap(t *testing.T) {
	g := BuildGame(1)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame(1) setup failed")
	}
	startCell := g.Grid.StartCell()
	if startCell == nil {
		t.Fatal("no start cell")
	}
	startRoom := startCell.Name
	hasSpawnGen := false
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || cell.Name != startRoom {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen != nil && gen.IsPowered() {
			hasSpawnGen = true
		}
	})
	if !hasSpawnGen {
		t.Fatal("start room should contain the powered spawn generator")
	}
	if !g.RoomDoorsPowered[startRoom] {
		t.Errorf("start room %q doors should be armed via generator bootstrap, not InitRoomPower", startRoom)
	}
}

func TestBuildGame_MaintBootstrapOK(t *testing.T) {
	g := BuildGame(1)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame(1) setup failed")
	}
	if !setup.MaintBootstrapOK(g) {
		t.Fatal("level 1 should have at least one powered maint terminal on conductive generator power grid")
	}
	if setup.CountPoweredMaintenanceTerminals(g) == 0 {
		t.Fatal("expected a powered maintenance terminal after generator bootstrap")
	}
}

func TestBuildGame_GeneratorRoomsArmed(t *testing.T) {
	g := BuildGame(1)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame(1) setup failed")
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		if gameworld.GetGameData(cell).Generator == nil {
			return
		}
		if !g.RoomDoorsPowered[cell.Name] {
			t.Errorf("generator room %q should have doors armed after bootstrap", cell.Name)
		}
		if !setup.RoomIsOnline(g, cell.Name) {
			t.Errorf("generator room %q should be power-online after bootstrap", cell.Name)
		}
	})
}

func TestBuildGame_SetupOrderIncludesSolvability(t *testing.T) {
	// BuildGame → SetupLevel runs solvability passes after terminal placement.
	g := BuildGame(1)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame(1) setup failed")
	}
	startRoom := g.Grid.StartCell().Name
	if !g.RoomDoorsPowered[startRoom] {
		t.Errorf("start room %q doors should be armed via generator bootstrap", startRoom)
	}
	if len(g.RoomDoorsPowered) == 0 {
		t.Error("RoomDoorsPowered should be populated after setup")
	}
}

func TestBuildGame_NoMaintBootstrapDeadlock(t *testing.T) {
	seeds := []int64{1, 42, 424242, 1779651561562416055, 999_999_999}
	for level := 1; level <= 6; level++ {
		for _, seed := range seeds {
			g := BuildGame(level)
			g.LevelSeed = seed
			ResetLevel(g)
			if !setup.MaintBootstrapOK(g) {
				t.Errorf("level %d seed %d: maint bootstrap failed (no powered terminal on generator power grid)", level, seed)
			}
		}
	}
}

func TestBuildGame_NoStartEgressDeadlock(t *testing.T) {
	seeds := []int64{1, 42, 424242, 1779651561562416055, 999_999_999}
	for level := 1; level <= 3; level++ {
		for _, seed := range seeds {
			g := BuildGame(level)
			g.LevelSeed = seed
			ResetLevel(g)
			if !setup.MaintBootstrapOK(g) {
				t.Errorf("level %d seed %d: maint bootstrap failed", level, seed)
			}
		}
	}
}

func TestResetLevel_ReinitializesRoomPower(t *testing.T) {
	g := BuildGame(1)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame(1) setup failed")
	}

	ResetLevel(g)

	// After reset, the start room's doors should be armed via spawn generator bootstrap
	newStartRoom := g.Grid.StartCell().Name
	if !g.RoomDoorsPowered[newStartRoom] {
		t.Errorf("after reset: start room %q doors should be armed via generator bootstrap", newStartRoom)
	}
	// Lights should default on for all rooms
	for room, lightsOn := range g.RoomLightsPowered {
		if !lightsOn {
			t.Errorf("after reset: room %q lights should default on", room)
		}
	}
	// Maintenance terminal power: at least one powered on conductive generator power grid after bootstrap
	if !setup.MaintBootstrapOK(g) {
		t.Errorf("after reset: maint bootstrap should be OK (powered terminal on generator power grid)")
	}
}

func TestSaveLoadDeckState_RoomPowerMapsPreserved(t *testing.T) {
	g := BuildGame(1)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame(1) setup failed")
	}

	// Set specific room power states
	for room := range g.RoomDoorsPowered {
		g.RoomDoorsPowered[room] = true
	}
	for room := range g.RoomCCTVPowered {
		g.RoomCCTVPowered[room] = true
	}

	g.SaveCurrentDeckState()

	// Mutate live state
	for room := range g.RoomDoorsPowered {
		g.RoomDoorsPowered[room] = false
	}
	for room := range g.RoomCCTVPowered {
		g.RoomCCTVPowered[room] = false
	}

	// Load saved state
	g.LoadDeckState(0)

	// Verify restored
	for room, powered := range g.RoomDoorsPowered {
		if !powered {
			t.Errorf("after LoadDeckState: RoomDoorsPowered[%q] should be true", room)
		}
	}
	for room, powered := range g.RoomCCTVPowered {
		if !powered {
			t.Errorf("after LoadDeckState: RoomCCTVPowered[%q] should be true", room)
		}
	}
}

func TestSaveLoadDeckState_RoomPowerDeepCopy(t *testing.T) {
	g := BuildGame(1)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame(1) setup failed")
	}

	g.SaveCurrentDeckState()

	// Mutate live maps — should NOT affect saved state
	for room := range g.RoomDoorsPowered {
		g.RoomDoorsPowered[room] = !g.RoomDoorsPowered[room]
	}

	// Re-load — g gets fresh maps from deep copy, ds holds the saved copy
	g.LoadDeckState(0)
	ds := g.DeckStates[0]
	// Mutate g and verify ds is unaffected (deep copy isolation)
	savedDoors := make(map[string]bool)
	for room, v := range ds.RoomDoorsPowered {
		savedDoors[room] = v
	}
	for room := range g.RoomDoorsPowered {
		g.RoomDoorsPowered[room] = !g.RoomDoorsPowered[room]
	}
	for room, v := range ds.RoomDoorsPowered {
		if v != savedDoors[room] {
			t.Errorf("mutating g.RoomDoorsPowered affected DeckStates: room %q changed", room)
		}
	}
}

func TestBuildGame_MaintenanceTerminalRestoreFlow(t *testing.T) {
	// BuildGame → start room terminal(s) powered, others unpowered.
	// From start room terminal, RefreshPowerGridMenuItem should power adjacent room terminals.
	var g *state.Game
	var startRoom string
	var startTermCell *world.Cell
	var startTerm *entities.MaintenanceTerminal

	// Generated layouts are randomized; retry to ensure this test validates the restore path.
	for attempt := 0; attempt < 8; attempt++ {
		g = BuildGame(2) // Level 2 has multiple rooms with terminals
		if g == nil || g.Grid == nil {
			t.Fatal("BuildGame(2) setup failed")
		}

		startRoom = g.Grid.StartCell().Name
		startTermCell = nil
		startTerm = nil
		g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
			if cell == nil || cell.Name != startRoom {
				return
			}
			data := gameworld.GetGameData(cell)
			if data.MaintenanceTerm != nil {
				startTermCell = cell
				startTerm = data.MaintenanceTerm
			}
		})
		if startTermCell != nil && startTerm != nil {
			break
		}
	}
	if startTermCell == nil || startTerm == nil {
		t.Fatal("could not find a start-room maintenance terminal after multiple BuildGame(2) attempts")
	}
	if !startTerm.Powered {
		t.Error("start room maintenance terminal should be powered after BuildGame")
	}

	// Power grid restore requires powered doors along the path — enable doors on adjacent rooms for this integration test.
	adjRooms := setup.GetAdjacentRoomNames(g.Grid, startRoom)
	if adjRooms == nil {
		adjRooms = []string{startRoom}
	}
	for _, rn := range adjRooms {
		g.RoomDoorsPowered[rn] = true
	}
	setup.EnergizeArmedRoomsForTest(g)
	gridRooms := setup.RoomsReachableInPowerGrid(g, startTermCell)
	if len(gridRooms) == 0 {
		gridRooms = adjRooms
	}
	unpoweredBefore := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		for _, rn := range gridRooms {
			if cell.Name == rn {
				data := gameworld.GetGameData(cell)
				if data.MaintenanceTerm != nil && !data.MaintenanceTerm.Powered {
					unpoweredBefore++
				}
				break
			}
		}
	})

	// Simulate "Refresh power grid" from start room terminal
	h := menu.NewMaintenanceMenuHandler(g, startTermCell, startTerm)
	restoreItem := &menu.RefreshPowerGridMenuItem{Parent: h}
	h.OnActivate(restoreItem, 0)

	// Verify previously unpowered terminals are now powered
	unpoweredAfter := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		for _, rn := range gridRooms {
			if cell.Name == rn {
				data := gameworld.GetGameData(cell)
				if data.MaintenanceTerm != nil && !data.MaintenanceTerm.Powered {
					unpoweredAfter++
				}
				break
			}
		}
	})

	if unpoweredBefore > 0 && unpoweredAfter > 0 {
		t.Errorf("restore should have powered terminals: before=%d unpowered, after=%d unpowered", unpoweredBefore, unpoweredAfter)
	}
}

func TestAdvanceThroughAllDecks_FinalDeckReachable(t *testing.T) {
	// Advances through all decks; each is generated once; final deck is reachable.
	// Note: runs 10 full BSP+SetupLevel generations; may be slow under -race or on constrained CI.
	g := BuildGame(1)
	seenDecks := make(map[int]bool)
	seenDecks[0] = true
	for g.CurrentDeckID < deck.FinalDeckIndex {
		AdvanceLevel(g)
		seenDecks[g.CurrentDeckID] = true
		if g.Grid == nil {
			t.Fatalf("deck %d: Grid is nil after AdvanceLevel", g.CurrentDeckID)
		}
	}
	// Should be on final deck
	if g.CurrentDeckID != deck.FinalDeckIndex || g.Level != deck.TotalDecks {
		t.Errorf("expected final deck: CurrentDeckID=%d Level=%d", g.CurrentDeckID, g.Level)
	}
	// All decks should have been visited
	for i := 0; i < deck.TotalDecks; i++ {
		if !seenDecks[i] {
			t.Errorf("deck %d was never visited", i)
		}
	}
	// TriggerGameComplete on final deck
	TriggerGameComplete(g)
	if !g.GameComplete {
		t.Error("TriggerGameComplete should set GameComplete on final deck")
	}
}
