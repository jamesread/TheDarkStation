package deck

import "testing"

func TestAssignThemes_FixedDecks(t *testing.T) {
	themes := AssignThemes(12345)
	if themes[0] != ThemeAirlock {
		t.Fatalf("deck 1 theme = %q, want airlock", themes[0])
	}
	if themes[4] != ThemeReactorControl {
		t.Fatalf("deck 5 theme = %q, want reactor_control", themes[4])
	}
	if themes[FinalDeckIndex] != ThemeExitDeck {
		t.Fatalf("deck 10 theme = %q, want exit_deck", themes[FinalDeckIndex])
	}
}

func TestAssignThemes_LifeSupportOnlyLateDecks(t *testing.T) {
	for seed := int64(1); seed <= 50; seed++ {
		themes := AssignThemes(seed)
		for id, theme := range themes {
			if theme != ThemeLifeSupport {
				continue
			}
			if id <= 4 {
				t.Fatalf("seed %d: life support on deck %d (want decks 6–9 only)", seed, id+1)
			}
		}
	}
}

func TestRoomNamesForTheme_Airlock(t *testing.T) {
	bases, adjectives := RoomNamesForTheme(ThemeAirlock)
	if len(bases) == 0 || len(adjectives) == 0 {
		t.Fatal("expected non-empty airlock room names")
	}
}

func TestReactorAuthKeycardName(t *testing.T) {
	themes := AssignThemes(1)
	name := ReactorAuthKeycardName(1, themes)
	if name == "" {
		t.Fatal("empty keycard name")
	}
}
