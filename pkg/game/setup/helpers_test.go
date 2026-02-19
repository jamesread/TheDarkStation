package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
)

func TestGetAdjacentRoomNames_NilGrid(t *testing.T) {
	result := GetAdjacentRoomNames(nil, "Any")
	if result != nil {
		t.Errorf("GetAdjacentRoomNames(nil, \"Any\") = %v, want nil", result)
	}
}

func TestGetAdjacentRoomNames_EmptyRoomName(t *testing.T) {
	g := world.NewGrid(2, 2)
	g.BuildAllCellConnections()
	result := GetAdjacentRoomNames(g, "")
	if result != nil {
		t.Errorf("GetAdjacentRoomNames(grid, \"\") = %v, want nil", result)
	}
}

func TestGetAdjacentRoomNames_RoomNameNotInGrid(t *testing.T) {
	g := world.NewGrid(2, 2)
	g.BuildAllCellConnections()
	// No cell has Name "MissingRoom"; all have "row:col".
	result := GetAdjacentRoomNames(g, "MissingRoom")
	if result != nil {
		t.Errorf("GetAdjacentRoomNames(grid, \"MissingRoom\") = %v, want nil", result)
	}
}

func TestGetAdjacentRoomNames_TwoRoomsSharingWall(t *testing.T) {
	// 2x2 grid: (0,0)=A, (0,1)=A, (1,0)=B, (1,1)=B. A and B share a vertical boundary.
	g := world.NewGrid(2, 2)
	g.MarkAsRoomWithName(0, 0, "A", "desc")
	g.MarkAsRoomWithName(0, 1, "A", "desc")
	g.MarkAsRoomWithName(1, 0, "B", "desc")
	g.MarkAsRoomWithName(1, 1, "B", "desc")
	g.BuildAllCellConnections()

	for _, roomName := range []string{"A", "B"} {
		t.Run(roomName, func(t *testing.T) {
			result := GetAdjacentRoomNames(g, roomName)
			if result == nil {
				t.Fatal("GetAdjacentRoomNames = nil, want non-nil")
			}
			if len(result) != 2 {
				t.Errorf("len(result) = %d, want 2 (A and B)", len(result))
			}
			want := []string{"A", "B"}
			for i, name := range result {
				if i >= len(want) || name != want[i] {
					t.Errorf("result = %v, want %v", result, want)
					break
				}
			}
		})
	}
}

func TestGetAdjacentRoomNames_CorridorMediatedAdjacency(t *testing.T) {
	// 1x3: (0,0)=A, (0,1)=Corridor, (0,2)=B. A borders Corridor, Corridor borders B.
	// So B is adjacent to A (corridor-mediated). Corridor name is still excluded from result.
	g := world.NewGrid(1, 3)
	g.MarkAsRoomWithName(0, 0, "A", "desc")
	g.MarkAsRoomWithName(0, 1, "Corridor", "desc")
	g.MarkAsRoomWithName(0, 2, "B", "desc")
	g.BuildAllCellConnections()

	result := GetAdjacentRoomNames(g, "A")
	if result == nil {
		t.Fatal("GetAdjacentRoomNames(grid, \"A\") = nil, want non-nil")
	}
	want := []string{"A", "B"}
	if len(result) != len(want) || result[0] != want[0] || result[1] != want[1] {
		t.Errorf("result = %v, want %v (B adjacent via corridor; Corridor excluded)", result, want)
	}
}

func TestGetAdjacentRoomNames_SingleCellRoom(t *testing.T) {
	// One cell only with name "Solo"; no neighbours (or only walls). Result is [Solo].
	g := world.NewGrid(1, 1)
	g.MarkAsRoomWithName(0, 0, "Solo", "desc")
	g.BuildAllCellConnections()

	result := GetAdjacentRoomNames(g, "Solo")
	if result == nil {
		t.Fatal("GetAdjacentRoomNames = nil, want non-nil")
	}
	if len(result) != 1 || result[0] != "Solo" {
		t.Errorf("result = %v, want [\"Solo\"]", result)
	}
}

func TestGetAdjacentRoomNames_IsolatedRoom(t *testing.T) {
	// Room "Iso" surrounded by corridor only (no other named room). Result is [Iso].
	g := world.NewGrid(3, 3)
	g.MarkAsRoomWithName(0, 0, "Corridor", "desc")
	g.MarkAsRoomWithName(0, 1, "Corridor", "desc")
	g.MarkAsRoomWithName(0, 2, "Corridor", "desc")
	g.MarkAsRoomWithName(1, 0, "Corridor", "desc")
	g.MarkAsRoomWithName(1, 1, "Iso", "desc")
	g.MarkAsRoomWithName(1, 2, "Corridor", "desc")
	g.MarkAsRoomWithName(2, 0, "Corridor", "desc")
	g.MarkAsRoomWithName(2, 1, "Corridor", "desc")
	g.MarkAsRoomWithName(2, 2, "Corridor", "desc")
	g.BuildAllCellConnections()

	result := GetAdjacentRoomNames(g, "Iso")
	if result == nil {
		t.Fatal("GetAdjacentRoomNames = nil, want non-nil")
	}
	if len(result) != 1 || result[0] != "Iso" {
		t.Errorf("result = %v, want [\"Iso\"] (only named room)", result)
	}
}
