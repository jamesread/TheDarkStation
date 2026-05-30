package ebiten

import (
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

func cameraCoordMilli(v float64) int64 {
	return int64(v*1000 + 0.5)
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
		movementCount:       g.MovementCount,
		interactionsCount:   g.InteractionsCount,
		unpoweredGenerators: g.UnpoweredGeneratorCount(),
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

func (e *EbitenRenderer) mapDrawCacheHit(snapSeq uint64, camRow, camCol float64, startRow, startCol, bufW, bufH int) (blitX, blitY float64, ok bool) {
	c := &e.mapDrawCache
	if !c.valid || c.snapSeq != snapSeq {
		return 0, 0, false
	}
	if c.camRowMilli != cameraCoordMilli(camRow) || c.camColMilli != cameraCoordMilli(camCol) {
		return 0, 0, false
	}
	if c.startRow != startRow || c.startCol != startCol {
		return 0, 0, false
	}
	if c.bufW != bufW || c.bufH != bufH {
		return 0, 0, false
	}
	if c.tileSize != e.tileSize || c.viewRows != e.viewportRows || c.viewCols != e.viewportCols {
		return 0, 0, false
	}
	return c.blitX, c.blitY, true
}

func (e *EbitenRenderer) storeMapDrawCache(snapSeq uint64, camRow, camCol float64, startRow, startCol int, blitX, blitY float64, bufW, bufH int) {
	e.mapDrawCache = mapDrawCache{
		valid:       true,
		snapSeq:     snapSeq,
		camRowMilli: cameraCoordMilli(camRow),
		camColMilli: cameraCoordMilli(camCol),
		startRow:    startRow,
		startCol:    startCol,
		blitX:       blitX,
		blitY:       blitY,
		bufW:        bufW,
		bufH:        bufH,
		tileSize:    e.tileSize,
		viewRows:    e.viewportRows,
		viewCols:    e.viewportCols,
	}
}
