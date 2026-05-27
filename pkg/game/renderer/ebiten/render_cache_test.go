package ebiten

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
)

func TestMapDrawCacheHit_reusesMatchingFrame(t *testing.T) {
	e := &EbitenRenderer{
		tileSize:     24,
		viewportRows: 11,
		viewportCols: 15,
	}
	e.storeMapDrawCache(3, 50, 100, 44, 92, 10.5, 20.25, 360, 264)

	if _, _, ok := e.mapDrawCacheHit(3, 50, 100, 44, 92, 360, 264); !ok {
		t.Fatal("expected cache hit for identical frame")
	}
	if _, _, ok := e.mapDrawCacheHit(4, 50, 100, 44, 92, 360, 264); ok {
		t.Fatal("expected cache miss when snap seq changes")
	}
	if _, _, ok := e.mapDrawCacheHit(3, 51, 100, 44, 92, 360, 264); ok {
		t.Fatal("expected cache miss when camera moves")
	}
}

func TestRefreshMapPowerSnapshot_reusesLiveCells(t *testing.T) {
	e := &EbitenRenderer{}
	snap := &renderSnapshot{}
	live := map[uint64]bool{123: true}
	e.mapPowerLiveCells = live
	e.mapPowerSnapCacheKey = mapPowerSnapCacheKey{powerSupply: 5}

	g := state.NewGame()
	g.PowerSupply = 5
	g.Grid = world.NewGrid(1, 1)

	e.refreshMapPowerSnapshot(g, snap)
	if snap.mapPower.livePowerCells == nil || !snap.mapPower.livePowerCells[123] {
		t.Fatal("expected cached live power cells to be reused")
	}
}
