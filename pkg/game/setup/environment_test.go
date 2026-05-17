package setup

import (
	"testing"

	engineWorld "darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// Cross-shaped corridor gives junction cells with ≥3 corridor neighbors (Story 5.1 placement rule).
func TestApplyEnvironmentalSignage_junctionGetsPlaque(t *testing.T) {
	grid := engineWorld.NewGrid(14, 14)
	for row := 4; row <= 10; row++ {
		for col := 4; col <= 10; col++ {
			if row == 7 || col == 7 {
				grid.MarkAsRoomWithName(row, col, "Corridor", "ROOM_CORRIDOR")
			}
		}
	}
	grid.BuildAllCellConnections()

	g := state.NewGame()
	g.Grid = grid
	g.Level = 1
	g.LevelSeed = 91_827_361

	ApplyEnvironmentalSignage(g)

	found := 0
	grid.ForEachCell(func(row, col int, cell *engineWorld.Cell) {
		if gameworld.GetGameData(cell).EnvPlaqueMsgID != "" {
			found++
			t.Logf("plaque at %d,%d: %s", row, col, gameworld.GetGameData(cell).EnvPlaqueMsgID)
		}
	})
	if found == 0 {
		t.Fatal("expected at least one environmental plaque on corridor junctions")
	}
}

func TestApplyEnvironmentalSignage_straightCorridor_noPlaque(t *testing.T) {
	grid := engineWorld.NewGrid(12, 12)
	for col := 2; col <= 9; col++ {
		grid.MarkAsRoomWithName(6, col, "Corridor", "ROOM_CORRIDOR")
	}
	grid.BuildAllCellConnections()
	g := state.NewGame()
	g.Grid = grid
	g.Level = 1
	g.LevelSeed = 42

	ApplyEnvironmentalSignage(g)

	grid.ForEachCell(func(row, col int, cell *engineWorld.Cell) {
		if gameworld.GetGameData(cell).EnvPlaqueMsgID != "" {
			t.Fatalf("did not expect plaque on straight corridor-only layout at %d,%d", row, col)
		}
	})
}
