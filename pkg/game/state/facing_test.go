package state

import (
	"testing"

	"darkstation/pkg/engine/world"
)

func TestPlayerFacing_Icon(t *testing.T) {
	tests := []struct {
		facing PlayerFacing
		want   string
	}{
		{FaceNorth, "↑"},
		{FaceSouth, "↓"},
		{FaceEast, "→"},
		{FaceWest, "←"},
	}
	for _, tt := range tests {
		if got := tt.facing.Icon(); got != tt.want {
			t.Errorf("%v.Icon() = %q, want %q", tt.facing, got, tt.want)
		}
	}
}

func TestFacingToward(t *testing.T) {
	grid := world.NewGrid(3, 3)
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			grid.MarkAsRoom(r, c)
		}
	}
	grid.BuildAllCellConnections()
	center := grid.GetCell(1, 1)

	tests := []struct {
		target *world.Cell
		want   PlayerFacing
	}{
		{grid.GetCell(0, 1), FaceNorth},
		{grid.GetCell(2, 1), FaceSouth},
		{grid.GetCell(1, 2), FaceEast},
		{grid.GetCell(1, 0), FaceWest},
	}
	for _, tt := range tests {
		got, ok := FacingToward(center, tt.target)
		if !ok {
			t.Fatalf("FacingToward(%v) ok=false", tt.target)
		}
		if got != tt.want {
			t.Errorf("FacingToward neighbor (%d,%d) = %v, want %v", tt.target.Row, tt.target.Col, got, tt.want)
		}
	}

	if _, ok := FacingToward(center, grid.GetCell(0, 0)); ok {
		t.Error("FacingToward diagonal should be false")
	}
}

func TestAdjacentCellsClockwiseFromFacing(t *testing.T) {
	grid := world.NewGrid(3, 3)
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			grid.MarkAsRoom(r, c)
		}
	}
	grid.BuildAllCellConnections()
	center := grid.GetCell(1, 1)

	tests := []struct {
		facing PlayerFacing
		want   []*world.Cell
	}{
		{FaceNorth, []*world.Cell{center.North, center.East, center.South, center.West}},
		{FaceEast, []*world.Cell{center.East, center.South, center.West, center.North}},
		{FaceSouth, []*world.Cell{center.South, center.West, center.North, center.East}},
		{FaceWest, []*world.Cell{center.West, center.North, center.East, center.South}},
	}
	for _, tt := range tests {
		got := AdjacentCellsClockwiseFromFacing(center, tt.facing)
		if len(got) != len(tt.want) {
			t.Fatalf("facing %v: len=%d, want %d", tt.facing, len(got), len(tt.want))
		}
		for i := range tt.want {
			if got[i] != tt.want[i] {
				t.Errorf("facing %v index %d = (%d,%d), want (%d,%d)",
					tt.facing, i, got[i].Row, got[i].Col, tt.want[i].Row, tt.want[i].Col)
			}
		}
	}
}
