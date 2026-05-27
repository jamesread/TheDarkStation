package ebiten

import (
	"math"
	"sync"
)

// playerMoveDurationMs lives in constants.go (shared with movement key repeat).

// playerMoveTransition smoothly interpolates the player marker between grid cells on the draw thread.
// Game logic still moves instantly; this is visual-only.
type playerMoveTransition struct {
	mu sync.Mutex

	initialized              bool
	row, col                 float64
	lastSnapSeq              uint64
	lastSnapLevel            int
	lastSnapRow, lastSnapCol int
	animStartMs              int64
	animFromRow, animFromCol float64
	animToRow, animToCol     float64
	animating                bool
}

func (t *playerMoveTransition) snapTo(level, row, col int, snapSeq uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.initialized = true
	t.row = float64(row)
	t.col = float64(col)
	t.lastSnapLevel = level
	t.lastSnapRow = row
	t.lastSnapCol = col
	t.lastSnapSeq = snapSeq
	t.animating = false
}

func chebyshevDist(r1, c1, r2, c2 int) int {
	dr := r1 - r2
	if dr < 0 {
		dr = -dr
	}
	dc := c1 - c2
	if dc < 0 {
		dc = -dc
	}
	if dr > dc {
		return dr
	}
	return dc
}

func (t *playerMoveTransition) visualPosition(level, snapRow, snapCol int, snapSeq uint64, nowMs int64) (float64, float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.initialized {
		t.initialized = true
		t.row = float64(snapRow)
		t.col = float64(snapCol)
		t.lastSnapLevel = level
		t.lastSnapRow = snapRow
		t.lastSnapCol = snapCol
		t.lastSnapSeq = snapSeq
		return t.row, t.col
	}

	teleport := level != t.lastSnapLevel ||
		(snapSeq != t.lastSnapSeq && chebyshevDist(snapRow, snapCol, t.lastSnapRow, t.lastSnapCol) > 1)

	if teleport {
		t.row = float64(snapRow)
		t.col = float64(snapCol)
		t.lastSnapLevel = level
		t.lastSnapRow = snapRow
		t.lastSnapCol = snapCol
		t.lastSnapSeq = snapSeq
		t.animating = false
		return t.row, t.col
	}

	if snapSeq != t.lastSnapSeq && (snapRow != t.lastSnapRow || snapCol != t.lastSnapCol) {
		t.animFromRow = t.row
		t.animFromCol = t.col
		t.animToRow = float64(snapRow)
		t.animToCol = float64(snapCol)
		t.animStartMs = nowMs
		t.animating = true
		t.lastSnapLevel = level
		t.lastSnapRow = snapRow
		t.lastSnapCol = snapCol
		t.lastSnapSeq = snapSeq
	} else if snapSeq != t.lastSnapSeq {
		t.lastSnapSeq = snapSeq
		t.lastSnapLevel = level
	}

	if t.animating {
		elapsed := nowMs - t.animStartMs
		if elapsed >= playerMoveDurationMs {
			t.row = t.animToRow
			t.col = t.animToCol
			t.animating = false
		} else {
			ease := easeOutCubic(float64(elapsed) / float64(playerMoveDurationMs))
			t.row = t.animFromRow + (t.animToRow-t.animFromRow)*ease
			t.col = t.animFromCol + (t.animToCol-t.animFromCol)*ease
		}
	} else {
		t.row = float64(snapRow)
		t.col = float64(snapCol)
	}

	return t.row, t.col
}

func mapCameraStartAt(centerRow, centerCol float64, viewportRows, viewportCols int) (startRow, startCol int) {
	topLeftRow := centerRow - float64(viewportRows)/2
	topLeftCol := centerCol - float64(viewportCols)/2
	return int(math.Floor(topLeftRow)), int(math.Floor(topLeftCol))
}
