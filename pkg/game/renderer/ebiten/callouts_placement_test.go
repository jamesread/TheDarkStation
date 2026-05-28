package ebiten

import "testing"

func TestCalloutBasePosition_PlayerEastOfAnchor(t *testing.T) {
	const (
		tileSize   = 32
		boxWidth   = 120
		boxHeight  = 80
		cellX      = 100.0
		cellY      = 100.0
		mapX       = 0.0
		mapY       = 0.0
		viewport   = 20
		anchorRow  = 5
		anchorCol  = 5
		playerRow  = 5
		playerCol  = 6
	)
	playerX := cellX + float64(tileSize)
	playerY := cellY

	x, y := calloutBasePosition(
		cellX, cellY, tileSize, boxWidth, boxHeight,
		playerRow, playerCol, anchorRow, anchorCol,
		playerX, playerY,
		mapX, mapY, viewport, viewport,
	)

	box := calloutRectAt(x, y, boxWidth, boxHeight)
	player := playerIconRect(playerX, playerY, tileSize)
	if box.overlaps(player) {
		t.Fatalf("tooltip overlaps player at (%.0f, %.0f); box=%+v player=%+v", x, y, box, player)
	}
	// Preferred left placement is off-screen; clamped to map edge still clears the player.
	if x != mapX {
		t.Errorf("x = %v, want %v (clamped left of viewport)", x, mapX)
	}
	wantY := cellY + float64((tileSize-boxHeight)/2)
	if y != wantY {
		t.Errorf("y = %v, want %v (beside anchor, not over player)", y, wantY)
	}
}

func TestCalloutBasePosition_PlayerWestOfAnchor(t *testing.T) {
	const (
		tileSize  = 32
		boxWidth  = 120
		boxHeight = 80
		cellX     = 200.0
		cellY     = 100.0
		anchorRow = 3
		anchorCol = 8
	)
	playerX := cellX - float64(tileSize)
	playerY := cellY

	x, _ := calloutBasePosition(
		cellX, cellY, tileSize, boxWidth, boxHeight,
		anchorRow, anchorCol-1, anchorRow, anchorCol,
		playerX, playerY,
		0, 0, 30, 30,
	)

	box := calloutRectAt(x, cellY, boxWidth, boxHeight)
	player := playerIconRect(playerX, playerY, tileSize)
	if box.overlaps(player) {
		t.Fatal("tooltip overlaps player west of anchor")
	}
	wantX := cellX + float64(tileSize) + calloutMargin
	if x != wantX {
		t.Errorf("x = %v, want %v", x, wantX)
	}
}
