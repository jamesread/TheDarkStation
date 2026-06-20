package setup

import (
	"testing"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestFindRoom_usesInitReachabilityNotTopology(t *testing.T) {
	grid := world.NewGrid(5, 9)
	for r := 0; r < 5; r++ {
		for c := 0; c < 9; c++ {
			name := "Far"
			if c >= 3 && c <= 5 && r >= 1 && r <= 3 {
				name = generator.ShaftRoomName
			}
			if c == 2 && r == 2 {
				name = "Corridor"
			}
			grid.MarkAsRoomWithName(r, c, name, "desc")
		}
	}
	grid.SetExitCellAt(2, 4)
	grid.BuildAllCellConnections()
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})
	doorCell := grid.GetCell(2, 2)
	gameworld.GetGameData(doorCell).Door = &entities.Door{RoomName: "Far", Locked: false}

	g := state.NewGame()
	g.Level = 6
	g.Grid = grid
	InitRoomPower(g)
	g.RoomDoorsPowered["Far"] = false

	avoid := mapset.New[*world.Cell]()
	room := findRoom(g, PlayerEntryCell(g), &avoid)
	if room == nil {
		t.Fatal("findRoom returned nil")
	}
	reach := InitialReachableCells(g)
	if !reach.Has(room) {
		t.Fatalf("findRoom picked x:%d y:%d outside init reachability", room.Col, room.Row)
	}
	if IsLiftShaftBoundsCell(g, room) && room != g.Grid.ExitCell() {
		t.Fatalf("findRoom picked lift shaft cell x:%d y:%d", room.Col, room.Row)
	}
}

func TestFindRoomInReachable_excludesLiftShaftOffEntry(t *testing.T) {
	const rows, cols = 11, 11
	top, left, bottom, right := generator.ShaftBoundsForLevel(rows, cols, 6)

	grid := world.NewGrid(rows, cols)
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		name := generator.ShaftRoomName
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
	reach := mapset.New[*world.Cell]()
	reach.Put(grid.GetCell((top+bottom)/2, left-1))
	reach.Put(grid.ExitCell())
	reach.Put(grid.GetCell((top+bottom)/2, right))
	g := state.NewGame()
	g.Level = 6
	g.Grid = grid
	avoid := mapset.New[*world.Cell]()
	cell := findRoomInReachable(g, &reach, &avoid)
	if cell == nil {
		t.Fatal("findRoomInReachable returned nil")
	}
	if IsLiftShaftBoundsCell(g, cell) && cell != grid.ExitCell() {
		t.Fatalf("picked lift shaft cell x:%d y:%d", cell.Col, cell.Row)
	}
	if cell.Name != "Lab" {
		t.Fatalf("picked %q, want Lab", cell.Name)
	}
}
