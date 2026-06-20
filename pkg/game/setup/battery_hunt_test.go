package setup

import (
	"testing"

	engineworld "darkstation/pkg/engine/world"
	"darkstation/pkg/game/gamemode"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/levelrand"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestSetupBatteryHuntLevel_PlacesGeneratorAndBatteries(t *testing.T) {
	levelrand.Seed(42)
	g := state.NewGame()
	g.SetMode(gamemode.FindTheBatteries)
	g.Level = 1
	g.Grid = generator.BSP.GenerateWithOptions(1, g.ThemeForDeck(0), generator.GenerateOptionsFromMode(g.Mode()))

	SetupBatteryHuntLevel(g)

	if len(g.Generators) != 1 {
		t.Fatalf("generators = %d, want 1", len(g.Generators))
	}
	if g.Generators[0].IsPowered() {
		t.Fatal("battery hunt generator should start unpowered")
	}
	required := g.Generators[0].BatteriesRequired
	if required < 5 || required > 8 {
		t.Fatalf("BatteriesRequired = %d, want 5–8", required)
	}

	batteries := 0
	g.Grid.ForEachCell(func(row, col int, cell *engineworld.Cell) {
		if cell == nil {
			return
		}
		cell.ItemsOnFloor.Each(func(item *engineworld.Item) {
			if item != nil && item.Name == "Battery" {
				batteries++
			}
		})
		if gameworld.GetGameData(cell).Door != nil {
			t.Fatalf("unexpected door at x:%d y:%d", cell.Col, cell.Row)
		}
		if gameworld.GetGameData(cell).Hazard != nil {
			t.Fatalf("unexpected hazard at x:%d y:%d", cell.Col, cell.Row)
		}
		if gameworld.GetGameData(cell).MaintenanceTerm != nil {
			t.Fatalf("unexpected maintenance terminal at x:%d y:%d", cell.Col, cell.Row)
		}
		if gameworld.GetGameData(cell).Puzzle != nil {
			t.Fatalf("unexpected puzzle at x:%d y:%d", cell.Col, cell.Row)
		}
	})
	if batteries != required {
		t.Fatalf("floor batteries = %d, want %d", batteries, required)
	}

	report := SimulatePlaythrough(g)
	if !report.Solvable {
		t.Fatalf("battery hunt layout not solvable: %v", report.Failures)
	}
}
