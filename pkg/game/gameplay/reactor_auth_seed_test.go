package gameplay

import (
	"testing"

	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/levelgen"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	"darkstation/pkg/game/unlocks"
)

func TestReactorAuthKeycardPickupReachable_seed18B91193(t *testing.T) {
	const seed = int64(0x18B91193FEDFD91A)
	g := BuildGame(3)
	LoadLevelFromSeed(g, seed)

	keycard := reactorAuthKeycardForDeck(g, 2)
	if keycard == "" {
		t.Fatal("expected deck 3 to source a reactor authorization keycard")
	}

	report := setup.SimulatePlaythrough(g)
	if !report.Solvable {
		t.Fatalf("fresh generation unsolvable: %v", report.Failures)
	}

	for _, repair := range g.RepairObjectives {
		if repair == nil {
			continue
		}
		repair.Status = entities.RepairComplete
	}
	levelgen.SpawnUnlockKeycardPayoffs(g)

	midReport := setup.SimulatePlaythrough(g)
	if !midReport.Solvable {
		t.Fatalf("mid-play sim unsolvable after keycard spawn: %v", midReport.Failures)
	}
}

func reactorAuthKeycardForDeck(g *state.Game, sourceDeckID int) string {
	if g == nil || g.UnlockPlan == nil {
		return ""
	}
	for _, req := range g.UnlockPlan.ForSource(sourceDeckID) {
		if req.Kind == unlocks.KindSecurityKeycard {
			return req.KeycardName
		}
	}
	return ""
}
