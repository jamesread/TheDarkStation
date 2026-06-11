package setup

import (
	"strings"
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/levelrand"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestSetupLevel_PlacesKeycards_level4Seed(t *testing.T) {
	const seed int64 = 0x18B512C7318DA329
	levelrand.Seed(seed)
	grid := generator.DefaultGenerator.Generate(4, deck.ThemeThermalReg)
	g := state.NewGame()
	g.Level = 4
	g.Grid = grid
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})
	SetupLevel(g)

	n := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			if item != nil && strings.Contains(item.Name, "Keycard") {
				n++
			}
		})
	})
	if n == 0 {
		t.Fatal("SetupLevel should place floor keycards for locked rooms")
	}
}
