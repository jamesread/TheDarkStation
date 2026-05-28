package ebiten

const calloutMargin = 8.0

type calloutRect struct {
	left, top, right, bottom float64
}

func (r calloutRect) overlaps(other calloutRect) bool {
	return r.left < other.right && r.right > other.left &&
		r.top < other.bottom && r.bottom > other.top
}

func playerIconRect(playerX, playerY float64, tileSize int) calloutRect {
	return calloutRect{
		left:   playerX + float64(tileSize/4),
		top:    playerY + float64(tileSize/4),
		right:  playerX + float64(tileSize*3/4),
		bottom: playerY + float64(tileSize*3/4),
	}
}

func calloutRectAt(x, y float64, boxWidth, boxHeight int) calloutRect {
	return calloutRect{x, y, x + float64(boxWidth), y + float64(boxHeight)}
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// calloutBasePosition picks the top-left corner for a tooltip beside anchor cell (row/col),
// preferring the side away from the player and avoiding overlap with the player icon.
func calloutBasePosition(
	cellX, cellY float64,
	tileSize, boxWidth, boxHeight int,
	playerRow, playerCol, anchorRow, anchorCol int,
	playerX, playerY float64,
	mapX, mapY float64,
	viewportCols, viewportRows int,
) (float64, float64) {
	mapRight := mapX + float64(viewportCols*tileSize)
	mapBottom := mapY + float64(viewportRows*tileSize)
	centerY := cellY + float64((tileSize-boxHeight)/2)
	centerX := cellX + float64((tileSize-boxWidth)/2)
	playerBox := playerIconRect(playerX, playerY, tileSize)

	dr := playerRow - anchorRow
	dc := playerCol - anchorCol

	horizontalAwayX := centerX
	switch {
	case dc > 0:
		horizontalAwayX = cellX - float64(boxWidth) - calloutMargin
	case dc < 0:
		horizontalAwayX = cellX + float64(tileSize) + calloutMargin
	}

	verticalAwayX := centerX
	switch {
	case dc > 0:
		verticalAwayX = cellX - float64(boxWidth) - calloutMargin
	case dc < 0:
		verticalAwayX = cellX + float64(tileSize) + calloutMargin
	case dr > 0:
		verticalAwayX = centerX
	case dr < 0:
		verticalAwayX = centerX
	}

	type pos struct{ x, y float64 }
	var candidates []pos

	left := pos{cellX - float64(boxWidth) - calloutMargin, centerY}
	right := pos{cellX + float64(tileSize) + calloutMargin, centerY}
	above := pos{verticalAwayX, cellY - float64(boxHeight) - calloutMargin}
	below := pos{verticalAwayX, cellY + float64(tileSize) + calloutMargin}

	switch {
	case dc > 0 && (absInt(dc) >= absInt(dr) || dr == 0):
		candidates = append(candidates, left, above, below, right)
	case dc < 0 && (absInt(dc) >= absInt(dr) || dr == 0):
		candidates = append(candidates, right, above, below, left)
	case dr > 0:
		candidates = append(candidates, above, left, right, below)
	case dr < 0:
		candidates = append(candidates, below, left, right, above)
	default:
		candidates = append(candidates, right, left, below, above)
	}

	clampX := func(x float64) float64 {
		if x < mapX {
			return mapX
		}
		if x+float64(boxWidth) > mapRight {
			return mapRight - float64(boxWidth)
		}
		return x
	}

	clampY := func(y float64) float64 {
		if y < mapY {
			return mapY
		}
		if y+float64(boxHeight) > mapBottom {
			return mapBottom - float64(boxHeight)
		}
		return y
	}

	fitsMap := func(p pos) bool {
		return p.x >= mapX && p.x+float64(boxWidth) <= mapRight &&
			p.y >= mapY && p.y+float64(boxHeight) <= mapBottom
	}

	for _, p := range candidates {
		p.x = clampX(p.x)
		p.y = clampY(p.y)
		if !fitsMap(p) {
			continue
		}
		if !calloutRectAt(p.x, p.y, boxWidth, boxHeight).overlaps(playerBox) {
			return p.x, p.y
		}
	}

	x := clampX(horizontalAwayX)
	return x, clampY(centerY)
}
