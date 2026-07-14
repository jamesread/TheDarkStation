package ebiten

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
)

func TestMapDrawCacheHit_reusesMatchingFrame(t *testing.T) {
	e := &EbitenRenderer{
		tileSize:     24,
		viewportRows: 11,
		viewportCols: 15,
	}
	e.storeMapDrawCache(3, 44, 92, 360, 264)

	if !e.mapDrawCacheHit(3, 44, 92, 360, 264) {
		t.Fatal("expected cache hit for identical frame")
	}
	if e.mapDrawCacheHit(4, 44, 92, 360, 264) {
		t.Fatal("expected cache miss when snap seq changes")
	}
	if e.mapDrawCacheHit(3, 45, 92, 360, 264) {
		t.Fatal("expected cache miss when viewport tile origin changes")
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

func TestBuildObjectivesCacheKey_includesRepairProgress(t *testing.T) {
	e := &EbitenRenderer{}
	g := state.NewGame()
	repair := entities.NewRepairObjective("pump", entities.RepairWastePump, "Pump", 0, 0)
	g.RepairObjectives = []*entities.RepairObjective{repair}

	before := e.buildObjectivesCacheKey(g)
	repair.Complete()
	after := e.buildObjectivesCacheKey(g)

	if before == after {
		t.Fatal("repair progress should affect objective cache key")
	}
}
