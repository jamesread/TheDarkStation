package state

import (
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
)

// InvalidateLivePowerCache clears cached propagated-power reachability.
// Call when generator feed or power-grid routing changes.
func (g *Game) InvalidateLivePowerCache() {
	if g == nil {
		return
	}
	g.livePowerCacheValid = false
	g.livePowerCellsCache = nil
}

// CachedLivePowerCells returns the cached live-power cell set when valid.
func (g *Game) CachedLivePowerCells() (*mapset.Set[*world.Cell], bool) {
	if g == nil || !g.livePowerCacheValid || g.livePowerCellsCache == nil {
		return nil, false
	}
	return g.livePowerCellsCache, true
}

// StoreLivePowerCellsCache stores propagated-power reachability for reuse within a tick.
func (g *Game) StoreLivePowerCellsCache(cells *mapset.Set[*world.Cell]) {
	if g == nil {
		return
	}
	g.livePowerCellsCache = cells
	g.livePowerCacheValid = cells != nil
}
