package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func makeMeshTestGrid(t *testing.T) (*state.Game, *world.Grid, *world.Cell, *world.Cell) {
	t.Helper()
	g := state.NewGame()
	grid := world.NewGrid(1, 3)
	grid.MarkAsRoomWithName(0, 0, "RoomA", "")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "ROOM_CORRIDOR")
	grid.MarkAsRoomWithName(0, 2, "RoomB", "")
	grid.BuildAllCellConnections()
	g.Grid = grid
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})
	return g, grid, grid.GetCell(0, 0), grid.GetCell(0, 1)
}

func TestRoomsReachableInPowerMesh_blockedByOpenRelay(t *testing.T) {
	g, _, start, relayCell := makeMeshTestGrid(t)
	if start == nil || relayCell == nil {
		t.Fatal("test grid missing cells")
	}
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}
	g.RoomCCTVPowered = map[string]bool{}

	gameworld.GetGameData(relayCell).PowerRelay = entities.NewPowerRelayOpen()

	rooms := RoomsReachableInPowerMesh(g, start)
	for _, r := range rooms {
		if r == "RoomB" {
			t.Fatal("open relay should block reach to RoomB")
		}
	}

	gameworld.GetGameData(relayCell).PowerRelay.Closed = true
	rooms = RoomsReachableInPowerMesh(g, start)
	foundB := false
	for _, r := range rooms {
		if r == "RoomB" {
			foundB = true
		}
	}
	if !foundB {
		t.Fatalf("closed relay should allow RoomB, got %v", rooms)
	}
}

func TestRoomsReachableInPowerMesh_requiresPoweredDoors(t *testing.T) {
	g, _, start, _ := makeMeshTestGrid(t)
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": false}
	g.RoomCCTVPowered = map[string]bool{}

	rooms := RoomsReachableInPowerMesh(g, start)
	for _, r := range rooms {
		if r == "RoomB" {
			t.Fatal("unpowered RoomB doors should block mesh")
		}
	}
}

func TestRestoreTerminalsInRooms_mesh(t *testing.T) {
	g, grid, _, _ := makeMeshTestGrid(t)
	termB := entities.NewMaintenanceTerminal("T", "RoomB")
	termB.Powered = false
	gameworld.GetGameData(grid.GetCell(0, 2)).MaintenanceTerm = termB
	g.RoomDoorsPowered = map[string]bool{"RoomA": true, "RoomB": true}

	n, msg := RestoreTerminalsInRooms(g, map[string]bool{"RoomB": true})
	if n != 1 {
		t.Fatalf("restored %d, msg %q", n, msg)
	}
	if !termB.Powered {
		t.Fatal("terminal should be powered")
	}
}
