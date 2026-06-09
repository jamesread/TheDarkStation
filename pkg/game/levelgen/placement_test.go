package levelgen

import (
	"testing"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// TestPlaceMaintenanceTerminals_RespectsR8 verifies that maintenance terminal placement
// does not put a terminal on a cell that would disconnect the room (R8 / I7).
// Room R has doorways at (1,0),(1,1),(1,2) and interior row (2,0),(2,1),(2,2); (1,1) is the only path between (1,0) and (1,2).
// The terminal must not be on the chokepoint (1,1).
func TestPlaceMaintenanceTerminals_RespectsR8(t *testing.T) {
	g := state.NewGame()
	g.Level = 2 // level 2+ gets maintenance terminals
	grid := world.NewGrid(3, 3)
	grid.MarkAsRoomWithName(0, 0, "Corridor", "corridor")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "corridor")
	grid.MarkAsRoomWithName(0, 2, "Corridor", "corridor")
	grid.MarkAsRoomWithName(1, 0, "R", "room")
	grid.MarkAsRoomWithName(1, 1, "R", "room")
	grid.MarkAsRoomWithName(1, 2, "R", "room")
	grid.MarkAsRoomWithName(2, 0, "R", "room")
	grid.MarkAsRoomWithName(2, 1, "R", "room")
	grid.MarkAsRoomWithName(2, 2, "R", "room")
	grid.BuildAllCellConnections()
	grid.SetStartCellAt(1, 0)
	g.Grid = grid
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			if cell := grid.GetCell(r, c); cell != nil {
				gameworld.InitGameData(cell)
			}
		}
	}
	avoid := mapset.New[*world.Cell]()
	PlaceMaintenanceTerminals(g, &avoid)

	roomName := "R"
	var terminalCell *world.Cell
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.Room && cell.Name == roomName {
			if gameworld.GetGameData(cell).MaintenanceTerm != nil {
				terminalCell = cell
			}
		}
	})
	if terminalCell == nil {
		t.Fatal("expected a maintenance terminal in R (room has interior R8-compliant cells)")
	}
	chokepoint := grid.GetCell(1, 1)
	if chokepoint == nil {
		t.Fatal("chokepoint nil")
	}
	if terminalCell == chokepoint {
		t.Errorf("PlaceMaintenanceTerminals placed terminal on chokepoint (1,1), violating R8 (room would be disconnected)")
	}
	entryPoints := setup.FindRoomEntryPoints(grid)[roomName]
	if entryPoints == nil {
		return
	}
	connected := setup.RoomStillConnectedIfBlock(g, roomName, entryPoints.EntryCells, nil)
	if !connected {
		t.Errorf("after placement, room R doorways are not all mutually reachable (R8 violation)")
	}
}

// TestPlaceMaintenanceTerminals_SkipsRoomWhenNoR8CompliantCandidate verifies that when every
// valid cell in a room would disconnect it (chokepoint), the room is skipped and no terminal is placed.
// Room R: doorways (1,0),(1,2) only; (1,1) is chokepoint. Interior (2,0),(2,1),(2,2) have furniture,
// so the only valid cell is (1,1), which fails R8 → connectedCandidates empty → skip room.
func TestPlaceMaintenanceTerminals_SkipsRoomWhenNoR8CompliantCandidate(t *testing.T) {
	g := state.NewGame()
	g.Level = 2
	grid := world.NewGrid(3, 3)
	// Corridor only at (0,0) and (0,2) so (1,1) is not a doorway; doorways = (1,0),(1,2)
	grid.MarkAsRoomWithName(0, 0, "Corridor", "corridor")
	grid.MarkAsRoomWithName(0, 1, "Wall", "wall")
	grid.MarkAsRoomWithName(0, 2, "Corridor", "corridor")
	grid.MarkAsRoomWithName(1, 0, "R", "room")
	grid.MarkAsRoomWithName(1, 1, "R", "room")
	grid.MarkAsRoomWithName(1, 2, "R", "room")
	grid.MarkAsRoomWithName(2, 0, "R", "room")
	grid.MarkAsRoomWithName(2, 1, "R", "room")
	grid.MarkAsRoomWithName(2, 2, "R", "room")
	grid.BuildAllCellConnections()
	grid.SetStartCellAt(1, 0)
	g.Grid = grid
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			if cell := grid.GetCell(r, c); cell != nil {
				gameworld.InitGameData(cell)
			}
		}
	}
	// Block interior cells so the only valid cell for maintenance is (1,1) (the chokepoint)
	for _, pt := range [][2]int{{2, 0}, {2, 1}, {2, 2}} {
		cell := grid.GetCell(pt[0], pt[1])
		if cell != nil {
			gameworld.GetGameData(cell).Furniture = entities.NewFurniture("Desk", "Desk", "desk")
		}
	}
	avoid := mapset.New[*world.Cell]()
	PlaceMaintenanceTerminals(g, &avoid)

	roomName := "R"
	var terminalInR bool
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.Room && cell.Name == roomName {
			if gameworld.GetGameData(cell).MaintenanceTerm != nil {
				terminalInR = true
			}
		}
	})
	if terminalInR {
		t.Errorf("PlaceMaintenanceTerminals placed a terminal in R; expected room skipped when no R8-compliant candidate (only chokepoint (1,1) was valid)")
	}
}

func TestPlaceMaintenanceTerminals_DoesNotCoverFloorItems(t *testing.T) {
	g := state.NewGame()
	g.Level = 4
	grid := world.NewGrid(2, 2)
	grid.MarkAsRoomWithName(0, 0, "Start", "room")
	grid.MarkAsRoomWithName(0, 1, "Start", "room")
	grid.MarkAsRoomWithName(1, 0, "Start", "room")
	grid.MarkAsRoomWithName(1, 1, "Start", "room")
	grid.BuildAllCellConnections()
	grid.SetStartCellAt(0, 0)
	g.Grid = grid
	for r := 0; r < 2; r++ {
		for c := 0; c < 2; c++ {
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}

	itemCell := grid.GetCell(0, 1)
	itemCell.ItemsOnFloor.Put(world.NewItem("Start Keycard"))
	avoid := mapset.New[*world.Cell]()
	avoid.Put(grid.GetCell(0, 0))
	avoid.Put(grid.GetCell(1, 0))
	avoid.Put(grid.GetCell(1, 1))

	PlaceMaintenanceTerminals(g, &avoid)

	if gameworld.GetGameData(itemCell).MaintenanceTerm != nil {
		t.Fatal("maintenance terminal should not be placed on a floor item")
	}
}

func TestPlaceRepairObjectives_placesChainAndBlocker(t *testing.T) {
	g := state.NewGame()
	g.Level = 6
	g.CurrentDeckID = 5
	grid := world.NewGrid(5, 5)
	for r := 0; r < 5; r++ {
		for c := 0; c < 5; c++ {
			roomName := string(rune('A' + r))
			if r >= 3 && c >= 3 {
				roomName = "Exit Approach"
			}
			grid.MarkAsRoomWithName(r, c, roomName, "room")
		}
	}
	grid.BuildAllCellConnections()
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(4, 4)
	g.Grid = grid
	for r := 0; r < 5; r++ {
		for c := 0; c < 5; c++ {
			gameworld.InitGameData(grid.GetCell(r, c))
		}
	}

	avoid := mapset.New[*world.Cell]()
	avoid.Put(grid.StartCell())
	avoid.Put(grid.ExitCell())
	PlaceRepairObjectives(g, &avoid)

	if len(g.RepairObjectives) < 3 {
		t.Fatalf("len(RepairObjectives) = %d, want at least 3", len(g.RepairObjectives))
	}
	var pump *entities.RepairObjective
	for _, repair := range g.RepairObjectives {
		if repair.Type == entities.RepairWastePump {
			pump = repair
		}
		if repair.DeviceRow < 0 || repair.DeviceCol < 0 {
			t.Fatalf("repair %s missing device coordinates", repair.ID)
		}
		cell := grid.GetCell(repair.DeviceRow, repair.DeviceCol)
		if cell == nil || gameworld.GetGameData(cell).RepairDevice != repair {
			t.Fatalf("repair %s not placed on its device cell", repair.ID)
		}
	}
	if pump == nil {
		t.Fatal("expected a waste pump objective")
	}
	if len(pump.PrereqIDs) == 0 {
		t.Fatal("waste pump should depend on earlier repairs")
	}
	if pump.BlockerName == "" {
		t.Fatal("waste pump should link a toxic-slime blocker")
	}
	if len(pump.BlockerCells) < 2 {
		t.Fatalf("expected toxic slime on multiple cells around exit, got %d", len(pump.BlockerCells))
	}
	pumpCell := grid.GetCell(pump.DeviceRow, pump.DeviceCol)
	if pumpCell == nil {
		t.Fatal("waste pump missing device cell")
	}
	exit := grid.ExitCell()
	adjacent := 0
	for _, bc := range pump.BlockerCells {
		blocker := grid.GetCell(bc.Row, bc.Col)
		if blocker == nil || gameworld.GetGameData(blocker).RepairBlocker != pump {
			t.Fatalf("blocker at (%d,%d) should point at waste pump", bc.Row, bc.Col)
		}
		if blocker.Name != pumpCell.Name || pump.RoomName != pumpCell.Name {
			t.Fatalf("waste pump room %q must match slime room %q", pumpCell.Name, blocker.Name)
		}
		for _, n := range exit.GetNeighbors() {
			if n == blocker {
				adjacent++
				break
			}
		}
	}
	if adjacent < 2 {
		t.Fatalf("expected at least two slime cells adjacent to exit, got %d", adjacent)
	}
}
