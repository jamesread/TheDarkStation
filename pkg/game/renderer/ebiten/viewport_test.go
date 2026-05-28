package ebiten

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
)

func TestSyncViewportForMap_fillsAvailableDrawArea(t *testing.T) {
	e := &EbitenRenderer{tileSize: 24}
	e.syncViewportForMap(1024, 768)
	if e.viewportCols != 45 || e.viewportRows != 33 {
		t.Fatalf("viewport = %d×%d, want 45×33", e.viewportCols, e.viewportRows)
	}
}

func TestSyncViewportForMap_coversPartialEdgeTiles(t *testing.T) {
	e := &EbitenRenderer{tileSize: 68}
	e.syncViewportForMap(918, 768)
	if e.viewportCols != 15 {
		t.Fatalf("viewportCols = %d, want 15", e.viewportCols)
	}
	if e.viewportCols*e.tileSize < 918 {
		t.Fatalf("viewport width %dpx does not cover 918px window", e.viewportCols*e.tileSize)
	}
}

func TestSyncViewportForMap_smallWindow(t *testing.T) {
	e := &EbitenRenderer{tileSize: 24}
	e.syncViewportForMap(200, 150)
	if e.viewportCols != 11 || e.viewportRows != 9 {
		t.Fatalf("viewport = %d×%d, want 11×9", e.viewportCols, e.viewportRows)
	}
}

func TestSyncViewportForMap_minimumOneTile(t *testing.T) {
	e := &EbitenRenderer{tileSize: 48}
	e.syncViewportForMap(30, 20)
	if e.viewportCols != 1 || e.viewportRows != 1 {
		t.Fatalf("viewport = %d×%d, want 1×1", e.viewportCols, e.viewportRows)
	}
}

func TestViewportTilesForAxis(t *testing.T) {
	tests := []struct {
		screen, tile, want int
	}{
		{1024, 68, 17},
		{1000, 68, 17},
		{918, 68, 15},
		{900, 68, 15},
		{68, 68, 1},
		{69, 68, 3},
	}
	for _, tc := range tests {
		got := viewportTilesForAxis(tc.screen, tc.tile)
		if got != tc.want {
			t.Errorf("viewportTilesForAxis(%d, %d) = %d, want %d", tc.screen, tc.tile, got, tc.want)
		}
	}
}

func TestMapCameraStart_symmetricAtRest(t *testing.T) {
	e := &EbitenRenderer{
		viewportCols:    17,
		viewportRows:    13,
		cameraCenterCol: 100,
		cameraCenterRow: 50,
	}
	g := &state.Game{}
	startRow, startCol := e.mapCameraStart(g)
	if startCol != 92 {
		t.Fatalf("startCol = %d, want 92 (8 cols each side)", startCol)
	}
	if startRow != 44 {
		t.Fatalf("startRow = %d, want 44 (6 rows each side)", startRow)
	}
	playerVCol := 100 - startCol
	playerVRow := 50 - startRow
	if playerVCol != e.viewportCols/2 {
		t.Fatalf("playerVCol = %d, want %d", playerVCol, e.viewportCols/2)
	}
	if playerVRow != e.viewportRows/2 {
		t.Fatalf("playerVRow = %d, want %d", playerVRow, e.viewportRows/2)
	}
}

func TestSyncPlayModeCamera_seedsMaintenancePanOrigin(t *testing.T) {
	grid := world.NewGrid(20, 40)
	g := &state.Game{Grid: grid, CurrentCell: grid.GetCell(12, 34)}
	e := &EbitenRenderer{snapSeq: 1, menuAnimClockMilli: 1_000_000}
	e.syncPlayModeCamera(g)
	if !e.cameraPlaySynced {
		t.Fatal("cameraPlaySynced = false, want true")
	}
	if e.cameraCenterRow != 12 || e.cameraCenterCol != 34 {
		t.Fatalf("camera center = (%.0f, %.0f), want (12, 34)", e.cameraCenterRow, e.cameraCenterCol)
	}
	if e.cameraTargetRow != 12 || e.cameraTargetCol != 34 {
		t.Fatalf("camera target = (%.0f, %.0f), want (12, 34)", e.cameraTargetRow, e.cameraTargetCol)
	}
}

func TestMapCameraScreenOrigin_playerCentered(t *testing.T) {
	mapX, mapY := mapCameraScreenOrigin(1024, 768, 50, 100, 44, 92, 68)
	playerScreenX := mapX + float64(8*68) + 34
	playerScreenY := mapY + float64(6*68) + 34
	if int(playerScreenX) != 512 || int(playerScreenY) != 384 {
		t.Fatalf("player screen = %.0f,%.0f want 512,384", playerScreenX, playerScreenY)
	}
}
