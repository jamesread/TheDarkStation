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
	g.armedBalanceComponentsCache = nil
	g.armedBalanceCacheValid = false
}

// CachedArmedBalanceComponents returns cached armed-grid components for overload balance.
func (g *Game) CachedArmedBalanceComponents() ([]*mapset.Set[*world.Cell], bool) {
	if g == nil || !g.armedBalanceCacheValid || g.armedBalanceComponentsCache == nil {
		return nil, false
	}
	return g.armedBalanceComponentsCache, true
}

// StoreArmedBalanceComponentsCache stores armed-grid components until routing changes.
func (g *Game) StoreArmedBalanceComponentsCache(components []*mapset.Set[*world.Cell]) {
	if g == nil {
		return
	}
	g.armedBalanceComponentsCache = components
	g.armedBalanceCacheValid = components != nil
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
