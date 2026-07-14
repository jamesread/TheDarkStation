package ebiten

import (
	"time"

	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
)

func generatorPoweredMask(g *state.Game) uint64 {
	if g == nil {
		return 0
	}
	var mask uint64
	for i, gen := range g.Generators {
		if gen != nil && gen.IsPowered() && i < 64 {
			mask |= 1 << uint(i)
		}
	}
	return mask
}

func (e *EbitenRenderer) invalidateMapDrawCache() {
	e.mapDrawCache.valid = false
}

func (e *EbitenRenderer) buildMapPowerSnapCacheKey(g *state.Game) mapPowerSnapCacheKey {
	if g == nil {
		return mapPowerSnapCacheKey{}
	}
	return mapPowerSnapCacheKey{
		powerSupply:          g.PowerSupply,
		powerConsumption:     g.PowerConsumption,
		generatorPoweredMask: generatorPoweredMask(g),
		maintRoom:            g.MaintenanceMenuRoom,
		powerGridOverlay:     g.PowerGridOverlayActive,
	}
}

func (e *EbitenRenderer) refreshMapPowerSnapshot(g *state.Game, snap *renderSnapshot) {
	if snap == nil {
		return
	}
	mp := &snap.mapPower
	*mp = mapPowerSnapshot{}
	if g == nil || g.Grid == nil {
		return
	}

	mp.roomDoorsPowered = copyStringBoolMap(g.RoomDoorsPowered)
	mp.roomCCTVPowered = copyStringBoolMap(g.RoomCCTVPowered)
	mp.manualEgressReleased = copyStringBoolMap(g.ManualEgressReleased)
	mp.maintenanceMenuRoom = g.MaintenanceMenuRoom
	if len(g.MaintenanceSelectableRooms) > 0 {
		mp.maintenanceSelectableRooms = append([]string(nil), g.MaintenanceSelectableRooms...)
	}

	key := e.buildMapPowerSnapCacheKey(g)
	if key == e.mapPowerSnapCacheKey && e.mapPowerLiveCells != nil {
		mp.livePowerCells = e.mapPowerLiveCells
		return
	}

	mp.livePowerCells = cellSetToKeyMap(setup.CellsReachableFromPoweredGenerators(g))
	e.mapPowerSnapCacheKey = key
	e.mapPowerLiveCells = mp.livePowerCells
}

func (e *EbitenRenderer) buildRoomLabelsCacheKey(g *state.Game) roomLabelsCacheKey {
	if g == nil || g.CurrentCell == nil {
		return roomLabelsCacheKey{}
	}
	return roomLabelsCacheKey{
		level:                g.Level,
		playerRow:            g.CurrentCell.Row,
		playerCol:            g.CurrentCell.Col,
		maintRoom:            g.MaintenanceMenuRoom,
		maintMenuMode:        g.MaintenanceMenuMode != "",
		powerSupply:          g.PowerSupply,
		generatorPoweredMask: generatorPoweredMask(g),
	}
}

func (e *EbitenRenderer) refreshRoomLabels(g *state.Game) []roomLabel {
	key := e.buildRoomLabelsCacheKey(g)
	if key == e.roomLabelsCacheKey && e.roomLabelsCache != nil {
		return e.roomLabelsCache
	}
	labels := e.computeRoomLabels(g)
	e.roomLabelsCacheKey = key
	e.roomLabelsCache = labels
	return labels
}

func (e *EbitenRenderer) buildObjectivesCacheKey(g *state.Game) objectivesCacheKey {
	if g == nil {
		return objectivesCacheKey{}
	}
	return objectivesCacheKey{
		level:               g.Level,
		interactionsCount:   g.InteractionsCount,
		unpoweredGenerators: g.UnpoweredGeneratorCount(),
		repairSignature:     g.RepairProgressSignature(),
	}
}

func (e *EbitenRenderer) refreshObjectives(g *state.Game) []string {
	key := e.buildObjectivesCacheKey(g)
	if key == e.objectivesCacheKey && e.objectivesCache != nil {
		return e.objectivesCache
	}
	objectives := e.calculateObjectives(g)
	e.objectivesCacheKey = key
	e.objectivesCache = objectives
	return objectives
}

func (e *EbitenRenderer) buildEnvPlaquesCacheKey(g *state.Game) envPlaquesCacheKey {
	if g == nil {
		return envPlaquesCacheKey{}
	}
	return envPlaquesCacheKey{
		level:             g.Level,
		movementCount:     g.MovementCount,
		interactionsCount: g.InteractionsCount,
		envPlaquesEnabled: e.EnvPlaquesEnabled(),
	}
}

func (e *EbitenRenderer) refreshEnvPlaques(g *state.Game) []envPlaque {
	key := e.buildEnvPlaquesCacheKey(g)
	if key == e.envPlaquesCacheKey && e.envPlaquesCache != nil {
		return e.envPlaquesCache
	}
	plaques := e.computeEnvPlaques(g)
	e.envPlaquesCacheKey = key
	e.envPlaquesCache = plaques
	return plaques
}

// mapAnimBucketMs sets how often the tile buffer redraws while the snapshot is
// otherwise unchanged, so ambient animation (shimmer, flicker, pulses) plays
// during idle. 50ms = 20 redraws/s, well under the effect periods.
const mapAnimBucketMs = 50

func (e *EbitenRenderer) mapDrawCacheHit(snapSeq uint64, startRow, startCol, bufW, bufH int) bool {
	c := &e.mapDrawCache
	if !c.valid || c.snapSeq != snapSeq {
		return false
	}
	if c.startRow != startRow || c.startCol != startCol {
		return false
	}
	if c.bufW != bufW || c.bufH != bufH {
		return false
	}
	if c.tileSize != e.tileSize || c.viewRows != e.viewportRows || c.viewCols != e.viewportCols {
		return false
	}
	if c.animBucket != time.Now().UnixMilli()/mapAnimBucketMs {
		return false
	}
	return true
}

func (e *EbitenRenderer) storeMapDrawCache(snapSeq uint64, startRow, startCol int, bufW, bufH int) {
	e.mapDrawCache = mapDrawCache{
		valid:      true,
		snapSeq:    snapSeq,
		startRow:   startRow,
		startCol:   startCol,
		bufW:       bufW,
		bufH:       bufH,
		tileSize:   e.tileSize,
		viewRows:   e.viewportRows,
		viewCols:   e.viewportCols,
		animBucket: time.Now().UnixMilli() / mapAnimBucketMs,
	}
}
