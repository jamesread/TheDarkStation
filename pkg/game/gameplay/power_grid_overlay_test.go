package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestToggleGeneratorPowerGridOverlay(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 1)
	grid.MarkAsRoomWithName(0, 0, "Room", "")
	grid.BuildAllCellConnections()
	g.Grid = grid
	cell := grid.GetCell(0, 0)
	gameworld.InitGameData(cell)

	ToggleGeneratorPowerGridOverlay(g, cell)
	if !g.PowerGridOverlayActive || g.PowerGridOverlaySeedRow != 0 {
		t.Fatal("first toggle should enable overlay at generator")
	}

	ToggleGeneratorPowerGridOverlay(g, cell)
	if g.PowerGridOverlayActive {
		t.Fatal("second toggle on same cell should disable overlay")
	}
}

func TestClearGeneratorPowerGridOverlay_onMove(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 2)
	grid.MarkAsRoomWithName(0, 0, "A", "")
	grid.MarkAsRoomWithName(0, 1, "B", "")
	grid.BuildAllCellConnections()
	g.Grid = grid
	grid.ForEachCell(func(row, col int, c *world.Cell) {
		if c != nil {
			gameworld.InitGameData(c)
			c.Discovered = true
		}
	})
	g.CurrentCell = grid.GetCell(0, 0)
	g.RoomDoorsPowered = map[string]bool{"A": true, "B": true}
	g.PowerGridOverlayActive = true
	g.PowerGridOverlaySeedRow = 0
	g.PowerGridOverlaySeedCol = 0

	MoveCell(g, grid.GetCell(0, 1))
	if g.PowerGridOverlayActive {
		t.Fatal("overlay should clear when player moves away")
	}
}
