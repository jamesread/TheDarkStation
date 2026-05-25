package world

import "testing"

func makeFOVGrid(t *testing.T, rows, cols int, rooms [][2]int) (*Grid, *Cell) {
	t.Helper()
	g := NewGrid(rows, cols)
	g.BuildAllCellConnections()
	for _, rc := range rooms {
		cell := g.GetCell(rc[0], rc[1])
		if cell != nil {
			cell.Room = true
			cell.Name = "Room"
		}
	}
	center := g.GetCell(rows/2, cols/2)
	if center != nil {
		center.Room = true
	}
	return g, center
}

func TestCalculateFOV_openRoomFullyVisible(t *testing.T) {
	var rooms [][2]int
	for r := 0; r < 15; r++ {
		for c := 0; c < 15; c++ {
			rooms = append(rooms, [2]int{r, c})
		}
	}
	grid, _ := makeFOVGrid(t, 15, 15, rooms)
	center := grid.GetCell(7, 7)
	if center == nil {
		t.Fatal("missing center cell")
	}
	center.Room = true

	visible := VisibleCellSet(grid, center, nil)
	if len(visible) != len(rooms) {
		t.Fatalf("expected all %d room cells visible, got %d", len(rooms), len(visible))
	}
}

func TestCalculateFOV_wallBlocksSight(t *testing.T) {
	var rooms [][2]int
	for c := 3; c <= 7; c++ {
		rooms = append(rooms, [2]int{5, c})
	}
	grid, _ := makeFOVGrid(t, 11, 11, rooms)
	center := grid.GetCell(5, 3)
	if center == nil {
		t.Fatal("missing center cell")
	}
	center.Room = true

	visible := VisibleCellSet(grid, center, nil)
	if !visible[grid.GetCell(5, 7)] {
		t.Fatal("cell at end of corridor should be visible")
	}
	if visible[grid.GetCell(5, 8)] {
		t.Fatal("cell beyond wall should not be visible")
	}
}

func TestCalculateFOV_openCorridorFullLength(t *testing.T) {
	var rooms [][2]int
	for c := 3; c <= 9; c++ {
		rooms = append(rooms, [2]int{5, c})
	}
	grid, _ := makeFOVGrid(t, 11, 13, rooms)
	center := grid.GetCell(5, 3)
	if center == nil {
		t.Fatal("missing center cell")
	}
	center.Room = true

	visible := VisibleCellSet(grid, center, nil)
	if !visible[grid.GetCell(5, 9)] {
		t.Fatal("should see full length of open corridor")
	}
	if visible[grid.GetCell(5, 10)] {
		t.Fatal("cell beyond corridor wall should not be visible")
	}
}

func TestCalculateFOV_sightBlockerStopsRay(t *testing.T) {
	var rooms [][2]int
	for c := 0; c <= 4; c++ {
		rooms = append(rooms, [2]int{2, c})
	}
	grid, _ := makeFOVGrid(t, 5, 7, rooms)
	block := func(cell *Cell) bool {
		return cell.Row == 2 && cell.Col == 2
	}
	visible := VisibleCellSet(grid, grid.GetCell(2, 0), block)
	if !visible[grid.GetCell(2, 2)] {
		t.Fatal("blocker cell should be visible")
	}
	if visible[grid.GetCell(2, 4)] {
		t.Fatal("cells beyond sight blocker should not be visible")
	}
}

func TestCollectFOVRays_matchesVisibleEndpoints(t *testing.T) {
	var rooms [][2]int
	for c := 0; c <= 4; c++ {
		rooms = append(rooms, [2]int{2, c})
	}
	grid, _ := makeFOVGrid(t, 5, 7, rooms)
	center := grid.GetCell(2, 0)
	if center == nil {
		t.Fatal("missing center")
	}
	center.Room = true

	block := func(cell *Cell) bool {
		return cell.Row == 2 && cell.Col == 2
	}
	rays := CollectFOVRays(grid, center, block)
	if len(rays) == 0 {
		t.Fatal("expected rays")
	}
	visible := VisibleCellSet(grid, center, block)
	endpoints := make(map[[2]int]struct{}, len(rays))
	for _, ray := range rays {
		endpoints[[2]int{ray.EndRow, ray.EndCol}] = struct{}{}
	}
	if _, ok := endpoints[[2]int{2, 2}]; !ok {
		t.Fatal("expected ray ending at blocker cell")
	}
	if _, ok := endpoints[[2]int{2, 4}]; ok {
		t.Fatal("should not have ray endpoint beyond blocker")
	}
	if len(endpoints) > len(visible) {
		t.Fatalf("endpoints %d exceed visible cells %d", len(endpoints), len(visible))
	}
}
func TestRevealFOV_doesNotMarkVisited(t *testing.T) {
	var rooms [][2]int
	for c := 0; c <= 4; c++ {
		rooms = append(rooms, [2]int{2, c})
	}
	grid, _ := makeFOVGrid(t, 5, 7, rooms)
	center := grid.GetCell(2, 0)
	if center == nil {
		t.Fatal("missing center")
	}
	center.Room = true
	center.Visited = true

	RevealFOV(grid, center, nil)

	for _, rc := range rooms {
		cell := grid.GetCell(rc[0], rc[1])
		if cell == nil {
			continue
		}
		if !cell.Discovered {
			t.Fatalf("cell (%d,%d) should be discovered", rc[0], rc[1])
		}
		if cell != center && cell.Visited {
			t.Fatalf("cell (%d,%d) should not be marked visited from FOV alone", rc[0], rc[1])
		}
	}
}

func TestCastRay_stopsAtWall(t *testing.T) {
	var rooms [][2]int
	for c := 0; c <= 4; c++ {
		rooms = append(rooms, [2]int{2, c})
	}
	grid, _ := makeFOVGrid(t, 5, 7, rooms)
	visible := make(map[*Cell]bool)
	castRay(grid, 2, 0, 2, 4, visible, nil)
	if visible[grid.GetCell(2, 4)] == false {
		t.Fatal("endpoint in room should be visible")
	}
	castRay(grid, 2, 0, 2, 6, visible, nil)
	if visible[grid.GetCell(2, 5)] {
		t.Fatal("non-room cell should not be marked visible")
	}
}
