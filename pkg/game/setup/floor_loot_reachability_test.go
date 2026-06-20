package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestEnsureFloorLootReachability_movesLootOutOfLiftShaft(t *testing.T) {
	const rows, cols = 11, 11
	top, left, bottom, right := generator.ShaftBoundsForLevel(rows, cols, 6)

	grid := world.NewGrid(rows, cols)
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		name := "Corridor"
		if row >= top && row <= bottom && col >= left && col <= right {
			name = generator.ShaftRoomName
		}
		if col < left && row >= top && row <= bottom {
			name = "Lab"
		}
		grid.MarkAsRoomWithName(row, col, name, "desc")
	})
	grid.SetExitCellAt((top+bottom)/2, (left+right)/2)
	grid.BuildAllCellConnections()
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})

	g := state.NewGame()
	g.Level = 6
	g.Grid = grid
	InitRoomPower(g)

	shaftCell := grid.GetCell(top+1, right)
	shaftCell.ItemsOnFloor.Put(world.NewItem("Battery"))

	EnsureFloorLootReachability(g)

	if shaftCell.ItemsOnFloor.Size() != 0 {
		t.Fatal("battery should be relocated out of lift shaft")
	}
	found := false
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || IsLiftShaftBoundsCell(g, cell) {
			return
		}
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			if item != nil && item.Name == "Battery" {
				found = true
			}
		})
	})
	if !found {
		t.Fatal("battery should exist outside lift shaft bounds")
	}
}
