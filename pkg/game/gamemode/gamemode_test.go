package gamemode

import "testing"

func TestGet_SinglePlayerPuzzle(t *testing.T) {
	m := Get(SinglePlayerPuzzle)
	if m.ID != SinglePlayerPuzzle {
		t.Fatalf("ID = %q, want %q", m.ID, SinglePlayerPuzzle)
	}
	if m.TotalDecks != 10 {
		t.Fatalf("TotalDecks = %d, want 10", m.TotalDecks)
	}
	if !m.UsesCrossDeckUnlocks {
		t.Fatal("UsesCrossDeckUnlocks = false, want true")
	}
	if !m.Items.PlaceUnlockObjectives {
		t.Fatal("PlaceUnlockObjectives = false, want true")
	}
}

func TestGet_UnknownFallsBackToDefault(t *testing.T) {
	m := Get(ID("UnknownMode"))
	if m.ID != SinglePlayerPuzzle {
		t.Fatalf("unknown ID fallback = %q, want %q", m.ID, SinglePlayerPuzzle)
	}
}

func TestSingleDeckSandbox(t *testing.T) {
	m := Get(SingleDeckSandbox)
	if m.TotalDecks != 1 {
		t.Fatalf("TotalDecks = %d, want 1", m.TotalDecks)
	}
	if m.UsesCrossDeckUnlocks {
		t.Fatal("UsesCrossDeckUnlocks = true, want false")
	}
	if m.Items.PlaceUnlockObjectives {
		t.Fatal("PlaceUnlockObjectives = true, want false")
	}
}

func TestAll_IncludesFindTheBatteries(t *testing.T) {
	modes := All()
	if len(modes) != 3 {
		t.Fatalf("All() = %d modes, want 3", len(modes))
	}
	found := false
	for _, m := range modes {
		if m.ID == FindTheBatteries {
			found = true
			if !m.LevelGen.BatteryHunt {
				t.Fatal("FindTheBatteries should enable BatteryHunt")
			}
		}
	}
	if !found {
		t.Fatal("FindTheBatteries missing from All()")
	}
}

func TestBatteryHuntRequiredRoll(t *testing.T) {
	lg := Get(FindTheBatteries).LevelGen
	for i := 0; i < 20; i++ {
		got := lg.BatteryHuntRequiredRoll(func(n int) int { return i % n })
		if got < 5 || got > 8 {
			t.Fatalf("roll = %d, want 5–8", got)
		}
	}
}

func TestExtraBatteryRoll(t *testing.T) {
	p := ItemPlacementPrefs{ExtraBatteryMin: 1, ExtraBatteryMax: 2}
	if got := p.ExtraBatteryRoll(func(n int) int { return 0 }); got != 1 {
		t.Fatalf("roll 0 = %d, want 1", got)
	}
	if got := p.ExtraBatteryRoll(func(n int) int { return 1 }); got != 2 {
		t.Fatalf("roll 1 = %d, want 2", got)
	}
}
