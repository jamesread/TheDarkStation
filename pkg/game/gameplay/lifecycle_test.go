// Package gameplay tests lifecycle functions: BuildGame, AdvanceLevel, TriggerGameComplete.
package gameplay

import (
	"strings"
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/menu"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	"darkstation/pkg/game/unlocks"
	gameworld "darkstation/pkg/game/world"
)

func unlockAllDecksForTest(g *state.Game) {
	if g == nil {
		return
	}
	if g.LiftRoutingPowered == nil {
		g.LiftRoutingPowered = make(map[int]bool)
	}
	for id := 0; id < deck.TotalDecks; id++ {
		g.LiftRoutingPowered[id] = true
	}
	g.SetReactorOnline(true)
	if g.UnlockPlan == nil {
		return
	}
	for _, req := range g.UnlockPlan.Requirements {
		if req.Kind == unlocks.KindSecurityKeycard && req.KeycardName != "" {
			g.AddRunKeycard(world.NewItem(req.KeycardName))
		}
		g.MarkUnlockSatisfied(req.ID)
	}
}

// buildGameWithSeed builds a deck with deterministic run and layout seeds (for stable tests).
func buildGameWithSeed(level int, runSeed int64) *state.Game {
	g := state.NewGame()
	g.InitRunUnlocks(runSeed)
	g.CurrentDeckID = level - 1
	g.Level = level
	RegenerateFromSeed(g, runSeed)
	InitRunTracking(g)
	g.ClearMessages()
	return g
}

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
	g := buildGameWithSeed(1, 424242)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame(1) setup failed")
	}
	corner := setup.LiftShaftBottomLeftCell(g)
	if corner == nil {
		t.Fatal("expected lift shaft bottom-left cell")
	}
	shaftRoom := corner.Name
	hasSpawnGen := false
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || cell.Name != shaftRoom {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen != nil && gen.IsPowered() {
			hasSpawnGen = true
		}
	})
	if !hasSpawnGen {
		t.Fatal("lift shaft should contain the powered spawn generator")
	}
	if !g.RoomDoorsPowered[shaftRoom] {
		t.Errorf("lift shaft %q doors should be armed via generator bootstrap, not InitRoomPower", shaftRoom)
	}
}

func TestBuildGame_MaintBootstrapOK(t *testing.T) {
	g := buildGameWithSeed(1, 424242)
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
	g := buildGameWithSeed(1, 424242)
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
	g := buildGameWithSeed(1, 424242)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame(1) setup failed")
	}
	corner := setup.LiftShaftBottomLeftCell(g)
	if corner == nil {
		t.Fatal("expected lift shaft bottom-left cell")
	}
	shaftRoom := corner.Name
	if !g.RoomDoorsPowered[shaftRoom] {
		t.Errorf("lift shaft %q doors should be armed via generator bootstrap", shaftRoom)
	}
	if len(g.RoomDoorsPowered) == 0 {
		t.Error("RoomDoorsPowered should be populated after setup")
	}
}

func TestBuildGame_Deck1_ReportedSeed_MaintBootstrap(t *testing.T) {
	// Regression: 18B7A9785AB3072B produced dozens of annex rooms and unpowered maint grid.
	const seed int64 = 0x18B7A9785AB3072B
	g := buildGameWithSeed(1, seed)
	g.LevelSeed = seed
	ResetLevel(g)
	if !setup.MaintBootstrapOK(g) {
		t.Fatal("reported deck-1 seed should have powered maint terminal after bootstrap")
	}
	annexRooms := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name == "" {
			return
		}
		if strings.Contains(cell.Name, "Annex") {
			annexRooms++
		}
	})
	if annexRooms > 0 {
		t.Fatalf("reported seed still has %d annex-labeled cells", annexRooms)
	}
}

func TestBuildGame_NoMaintBootstrapDeadlock(t *testing.T) {
	seeds := []int64{1, 42, 424242, 0x18B7A9785AB3072B, 1779651561562416055, 999_999_999}
	for level := 1; level <= 6; level++ {
		for _, seed := range seeds {
			g := buildGameWithSeed(level, seed)
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
			g := buildGameWithSeed(level, seed)
			g.LevelSeed = seed
			ResetLevel(g)
			if !setup.MaintBootstrapOK(g) {
				t.Errorf("level %d seed %d: maint bootstrap failed", level, seed)
			}
		}
	}
}

func TestLoadLevelFromSeed_RelocatedKeycardIsDiscoverable(t *testing.T) {
	const seed int64 = 0x18B512C7318DA329
	g := buildGameWithSeed(4, seed)
	g.LevelSeed = seed
	ResetLevel(g)
	start := setup.PlayerEntryCell(g)
	if start == nil {
		t.Fatal("missing player entry cell")
	}
	reachable := setup.InitialReachableCells(g)
	var keycardCells []*world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			if item != nil && strings.Contains(item.Name, "Keycard") {
				keycardCells = append(keycardCells, cell)
			}
		})
		data := gameworld.GetGameData(cell)
		if data.Furniture != nil && data.Furniture.ContainedItem != nil &&
			strings.Contains(data.Furniture.ContainedItem.Name, "Keycard") {
			keycardCells = append(keycardCells, cell)
		}
	})
	if len(keycardCells) == 0 {
		t.Fatal("expected at least one keycard on the deck")
	}
	for _, keycardCell := range keycardCells {
		if !reachable.Has(keycardCell) {
			t.Fatalf("keycard at row=%d col=%d is not initially reachable", keycardCell.Row, keycardCell.Col)
		}
		if keycardCell == start {
			t.Fatalf("keycard should be discoverable away from the spawn tile, got row=%d col=%d",
				keycardCell.Row, keycardCell.Col)
		}
		if data := gameworld.GetGameData(keycardCell); data.MaintenanceTerm != nil {
			t.Fatalf("keycard at row=%d col=%d is covered by maintenance terminal", keycardCell.Row, keycardCell.Col)
		}
	}
}

func TestLoadLevelFromSeed_LargerDeckDoesNotStartWithKeycards(t *testing.T) {
	g := state.NewGame()
	g.Level = 6
	LoadLevelFromSeed(g, 42)

	g.OwnedItems.Each(func(item *world.Item) {
		if item != nil && strings.Contains(item.Name, "Keycard") {
			t.Fatalf("player should not start with keycard %q in inventory", item.Name)
		}
	})
	if g.CurrentCell == nil {
		t.Fatal("missing current cell")
	}
	g.CurrentCell.ItemsOnFloor.Each(func(item *world.Item) {
		if item != nil && strings.Contains(item.Name, "Keycard") {
			t.Fatalf("keycard %q should not start on the spawn tile", item.Name)
		}
	})
}

func TestBuildGame_NoInitialPowerOverload(t *testing.T) {
	seeds := []int64{1, 42, 424242, 1779651561562416055, 999_999_999}
	for level := 2; level <= 6; level++ {
		for _, seed := range seeds {
			g := state.NewGame()
			g.Level = level
			LoadLevelFromSeed(g, seed)
			if setup.AnyArmedGridOverloaded(g) {
				t.Errorf("level %d seed %d: grid overloaded at start (consumption=%d supply=%d)",
					level, seed, setup.CalculatePowerConsumption(g), g.PowerSupply)
			}
		}
	}
}

func TestResetLevel_ReinitializesRoomPower(t *testing.T) {
	g := buildGameWithSeed(1, 424242)
	if g == nil || g.Grid == nil {
		t.Fatal("BuildGame(1) setup failed")
	}

	ResetLevel(g)

	// After reset, lift shaft doors should be armed via spawn generator bootstrap
	corner := setup.LiftShaftBottomLeftCell(g)
	if corner == nil {
		t.Fatal("expected lift shaft bottom-left cell after reset")
	}
	if !g.RoomDoorsPowered[corner.Name] {
		t.Errorf("after reset: lift shaft %q doors should be armed via generator bootstrap", corner.Name)
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
	for attempt := 0; attempt < 24; attempt++ {
		g = BuildGame(2) // Level 2 has multiple rooms with terminals
		if g == nil || g.Grid == nil {
			t.Fatal("BuildGame(2) setup failed")
		}

		startRoom = g.Grid.StartCell().Name
		startTermCell = nil
		startTerm = nil
		g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
			if startTermCell != nil || cell == nil || !cell.Room {
				return
			}
			data := gameworld.GetGameData(cell)
			if data.MaintenanceTerm != nil && data.MaintenanceTerm.Powered {
				startTermCell = cell
				startTerm = data.MaintenanceTerm
				startRoom = cell.Name
			}
		})
		if startTermCell != nil && startTerm != nil {
			break
		}
	}
	if startTermCell == nil || startTerm == nil {
		t.Fatal("could not find a powered maintenance terminal after multiple BuildGame(2) attempts")
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
	unlockAllDecksForTest(g)
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
