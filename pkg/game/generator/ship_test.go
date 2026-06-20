package generator

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/levelrand"
)

func TestCarveDeck1ShipAndDock_fixedLayout(t *testing.T) {
	rows, cols := deckGridDimensions(1)
	grid := world.NewGrid(rows, cols)
	CarveDeck1ShipAndDock(grid)
	grid.BuildAllCellConnections()

	start := grid.StartCell()
	if start == nil || start.Name != ShipRoomName {
		t.Fatalf("start = %v, want Ship room", start)
	}
	if start.Row != deck1ShipStartRowCenter || start.Col != deck1ShipStartColCenter {
		t.Fatalf("start at x:%d y:%d, want x:%d y:%d", start.Col, start.Row, deck1ShipStartColCenter, deck1ShipStartRowCenter)
	}

	fusion := grid.GetCell(Deck1FusionReactorRow, Deck1FusionReactorCol)
	if fusion == nil || fusion.Name != ShipRoomName {
		t.Fatalf("fusion cell = %v, want Ship room at x:%d y:%d", fusion, Deck1FusionReactorCol, Deck1FusionReactorRow)
	}

	shipCells := 0
	wallCells := 0
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		if col == deck1ShipEastWallCol && row >= deck1ShipStartRow && row <= deck1ShipEndRow && row != deck1ShipDoorRow {
			if cell.Room {
				t.Fatalf("expected east bulkhead wall at x:%d y:%d", col, row)
			}
			wallCells++
			return
		}
		if !cell.Room {
			return
		}
		if cell.Name == ShipRoomName {
			shipCells++
		}
	})
	if wallCells != 4 {
		t.Fatalf("east bulkhead wall cells = %d, want 4", wallCells)
	}
	if shipCells != 15 {
		t.Fatalf("ship cells = %d, want 15", shipCells)
	}

	door := grid.GetCell(deck1ShipDoorRow, deck1ShipEastWallCol)
	if door == nil || !door.Room || door.Name != "Corridor" {
		t.Fatalf("ship east door at x:%d y:%d missing or wrong room (got %q)", deck1ShipEastWallCol, deck1ShipDoorRow, door.Name)
	}
}

func TestBSPGenerate_deck1WestOverlayNoBSPBleed(t *testing.T) {
	for _, seed := range []int64{1, 2, 42, 100, 999, 424242} {
		levelrand.Seed(seed)
		grid := BSP.Generate(1, deck.ThemeAirlock)
		grid.ForEachCell(func(row, col int, cell *world.Cell) {
			if cell == nil || !cell.Room {
				return
			}
			if row < deck1OverlayStartRow || row > deck1OverlayEndRow ||
				col < deck1OverlayStartCol || col > deck1WestOverlayRightCol {
				return
			}
			if cell.Name != ShipRoomName && cell.Name != "Corridor" {
				t.Fatalf("seed %d: unexpected room %q in deck 1 entry overlay at x:%d y:%d", seed, cell.Name, col, row)
			}
		})
	}
}

func TestBSPGenerate_deck1HasShipConnectedToShaft(t *testing.T) {
	for _, seed := range []int64{1, 2, 42, 999, 424242} {
		levelrand.Seed(seed)
		grid := BSP.Generate(1, deck.ThemeAirlock)
		if grid == nil {
			t.Fatal("nil grid")
		}
		if grid.StartCell() == nil || grid.StartCell().Name != ShipRoomName {
			t.Fatalf("seed %d: start not in Ship", seed)
		}
		if !reachableFrom(grid, grid.StartCell(), ShaftRoomName) {
			t.Fatalf("seed %d: Ship not connected to Lift Shaft", seed)
		}
	}
}

func reachableFrom(grid *world.Grid, from *world.Cell, roomName string) bool {
	if grid == nil || from == nil {
		return false
	}
	seen := map[*world.Cell]bool{}
	queue := []*world.Cell{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == nil || !cur.Room || seen[cur] {
			continue
		}
		seen[cur] = true
		if cur.Name == roomName {
			return true
		}
		for _, n := range cur.GetNeighbors() {
			if n != nil && n.Room && !seen[n] {
				queue = append(queue, n)
			}
		}
	}
	return false
}
