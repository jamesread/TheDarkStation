package ebiten

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

var (
	colorPowerGridCell         = color.RGBA{30, 70, 38, 230} // Corridor / live power grid conduit
	colorPowerGridCellOff      = color.RGBA{80, 30, 30, 230} // Corridor / armed-but-offline conduit
	colorPowerGridTerminal     = color.RGBA{72, 52, 22, 255} // Maintenance terminal seed (live)
	colorPowerGridTerminalOff  = color.RGBA{72, 32, 32, 255} // Maintenance terminal seed (offline armed)
	colorPowerGridGenerator    = color.RGBA{40, 80, 40, 255} // Generator seed (live)
	colorPowerGridGeneratorOff = color.RGBA{80, 35, 35, 255} // Generator seed (offline armed)
	colorPowerGridFloorOn      = color.RGBA{32, 72, 40, 235} // Named room floor, live power
)

// powerGridEdgeColor returns the stroke color for live power grid links.
func powerGridEdgeColor() color.Color {
	r, g, b, _ := colorGeneratorOn.RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 150}
}

// powerGridEdgeOffColor returns the stroke color for armed-but-offline grid links.
func powerGridEdgeOffColor() color.Color {
	r, g, b, _ := colorHazard.RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 150}
}

func powerGridOverlayActive(g *state.Game) bool {
	if g == nil {
		return false
	}
	if g.MaintenanceMenuRoom != "" {
		return true
	}
	return g.PowerGridOverlayActive
}

func powerGridOverlayActiveFromSnap(pg *powerGridSnapshot) bool {
	return pg != nil && pg.active
}

func cellCoordKey(row, col int) uint64 {
	return (uint64(uint32(row)) << 32) | uint64(uint32(col))
}

func cellSetToKeyMap(set *mapset.Set[*world.Cell]) map[uint64]bool {
	if set == nil || set.Size() == 0 {
		return nil
	}
	m := make(map[uint64]bool, set.Size())
	set.Each(func(c *world.Cell) {
		if c != nil {
			m[cellCoordKey(c.Row, c.Col)] = true
		}
	})
	return m
}

func snapshotHasCell(m map[uint64]bool, cell *world.Cell) bool {
	if cell == nil || m == nil {
		return false
	}
	return m[cellCoordKey(cell.Row, cell.Col)]
}

func copyStringBoolMap(src map[string]bool) map[string]bool {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// capturePowerGridSnapshot records overlay routing on the game thread (RenderFrame).
func capturePowerGridSnapshot(g *state.Game, snap *renderSnapshot) {
	if snap == nil {
		return
	}
	pg := &snap.powerGrid
	*pg = powerGridSnapshot{}
	if g == nil || !powerGridOverlayActive(g) {
		return
	}
	pg.active = true
	pg.maintenanceMenuRoom = g.MaintenanceMenuRoom
	pg.maintTerminalRow = g.MaintenanceMenuTerminalRow
	pg.maintTerminalCol = g.MaintenanceMenuTerminalCol
	pg.overlaySeedRow = g.PowerGridOverlaySeedRow
	pg.overlaySeedCol = g.PowerGridOverlaySeedCol
	pg.overlayDevActive = g.PowerGridOverlayActive
	pg.liveCells = cellSetToKeyMap(powerGridLiveCellsForGame(g))
	pg.armedCells = cellSetToKeyMap(powerGridArmedCellsForGame(g))
	pg.fedRooms = copyStringBoolMap(setup.RoomsFedByPoweredGeneratorGrid(g))
}

// captureMapPowerSnapshot records live power and room circuit maps on the game thread (RenderFrame).
func captureMapPowerSnapshot(g *state.Game, snap *renderSnapshot) {
	if snap == nil {
		return
	}
	mp := &snap.mapPower
	*mp = mapPowerSnapshot{}
	if g == nil || g.Grid == nil {
		return
	}
	mp.livePowerCells = cellSetToKeyMap(setup.CellsReachableFromPoweredGenerators(g))
	mp.roomDoorsPowered = copyStringBoolMap(g.RoomDoorsPowered)
	mp.roomCCTVPowered = copyStringBoolMap(g.RoomCCTVPowered)
	mp.manualEgressReleased = copyStringBoolMap(g.ManualEgressReleased)
	mp.maintenanceMenuRoom = g.MaintenanceMenuRoom
	if len(g.MaintenanceSelectableRooms) > 0 {
		mp.maintenanceSelectableRooms = append([]string(nil), g.MaintenanceSelectableRooms...)
	}
}

func snapCellHasLivePower(snap *renderSnapshot, cell *world.Cell) bool {
	if snap == nil {
		return false
	}
	return snapshotHasCell(snap.mapPower.livePowerCells, cell)
}

func snapRoomManualEgressReleased(snap *renderSnapshot, roomName string) bool {
	if snap == nil || roomName == "" {
		return false
	}
	return snap.mapPower.manualEgressReleased != nil && snap.mapPower.manualEgressReleased[roomName]
}

func snapRoomCCTVPowered(snap *renderSnapshot, roomName string) bool {
	if snap == nil || roomName == "" {
		return false
	}
	return snap.mapPower.roomCCTVPowered != nil && snap.mapPower.roomCCTVPowered[roomName]
}

func snapMaintenanceMenuRoom(snap *renderSnapshot) string {
	if snap == nil {
		return ""
	}
	return snap.mapPower.maintenanceMenuRoom
}

func powerGridSeedCell(g *state.Game) *world.Cell {
	if g == nil || g.Grid == nil {
		return nil
	}
	if g.MaintenanceMenuRoom != "" && g.MaintenanceMenuTerminalRow >= 0 && g.MaintenanceMenuTerminalCol >= 0 {
		return g.Grid.GetCell(g.MaintenanceMenuTerminalRow, g.MaintenanceMenuTerminalCol)
	}
	if g.PowerGridOverlayActive && g.PowerGridOverlaySeedRow >= 0 && g.PowerGridOverlaySeedCol >= 0 {
		return g.Grid.GetCell(g.PowerGridOverlaySeedRow, g.PowerGridOverlaySeedCol)
	}
	return nil
}

// powerGridLiveCellsForGame returns cells on the live propagated grid from the overlay seed.
func powerGridLiveCellsForGame(g *state.Game) *mapset.Set[*world.Cell] {
	start := powerGridSeedCell(g)
	if start == nil {
		return nil
	}
	return setup.CellsReachableInLivePowerGrid(g, start)
}

// powerGridArmedCellsForGame returns cells reachable on the overlay grid (armed circuits and manual egress).
func powerGridArmedCellsForGame(g *state.Game) *mapset.Set[*world.Cell] {
	start := powerGridSeedCell(g)
	if start == nil {
		return nil
	}
	return setup.CellsReachableInPowerGridOverlay(g, start)
}

// powerGridCellsForGame returns the live propagated grid (overlay routing uses live power only).
func powerGridCellsForGame(g *state.Game) *mapset.Set[*world.Cell] {
	return powerGridLiveCellsForGame(g)
}

func cellVisibleForPowerGridOverlay(g *state.Game, cell *world.Cell) bool {
	return cell != nil && cell.Room && (g.HasMap || cell.Discovered)
}

// cellVisibleForPowerGridTint includes wall cells (non-room) when discovered or bordering a visible room.
func cellVisibleForPowerGridTint(g *state.Game, cell *world.Cell) bool {
	if g == nil || cell == nil {
		return false
	}
	if g.HasMap || cell.Discovered {
		return true
	}
	for _, n := range cell.GetNeighbors() {
		if n != nil && n.Room && (g.HasMap || n.Discovered) {
			return true
		}
	}
	return false
}

func cellBlocksPowerGridRoomTint(cell *world.Cell) bool {
	if cell == nil {
		return true
	}
	return gameworld.HasGenerator(cell) || gameworld.HasFurniture(cell) ||
		gameworld.HasMaintenanceTerminal(cell) || gameworld.HasTerminal(cell) ||
		gameworld.HasPuzzle(cell) || gameworld.HasDoor(cell) || gameworld.HasPowerRelay(cell) ||
		gameworld.HasBlockingHazard(cell)
}

func isPowerGridFloorCell(cell *world.Cell, opts *CellRenderOptions) bool {
	if cell == nil || !cell.Room || cell.Name == "" || cell.Name == "Corridor" {
		return false
	}
	if opts == nil || opts.Icon == IconWall || opts.Icon == IconVoid {
		return false
	}
	return !cellBlocksPowerGridRoomTint(cell)
}

// powerGridRoomFloorBg tints room floors green when power-grid enabled and fed by a powered generator, else red.
func powerGridRoomFloorBg(g *state.Game, pg *powerGridSnapshot, cell *world.Cell, opts *CellRenderOptions) color.Color {
	if g == nil || cell == nil || !powerGridOverlayActiveFromSnap(pg) || !cellVisibleForPowerGridOverlay(g, cell) {
		return nil
	}
	if !isPowerGridFloorCell(cell, opts) {
		return nil
	}
	if pg.fedRooms != nil && pg.fedRooms[cell.Name] {
		return colorPowerGridFloorOn
	}
	return colorHazardBackground
}

// powerGridWallBg tints walls by adjacent room power-grid state (green fed, red otherwise).
func powerGridWallBg(g *state.Game, pg *powerGridSnapshot, cell *world.Cell) color.Color {
	if g == nil || cell == nil || !powerGridOverlayActiveFromSnap(pg) || !cellVisibleForPowerGridTint(g, cell) {
		return nil
	}
	hasNamedAdjacent := false
	anyUnpowered := false
	anyPowered := false
	for _, n := range cell.GetNeighbors() {
		if n == nil || !n.Room || n.Name == "" || n.Name == "Corridor" {
			continue
		}
		hasNamedAdjacent = true
		if pg.fedRooms != nil && pg.fedRooms[n.Name] {
			anyPowered = true
		} else {
			anyUnpowered = true
		}
	}
	if !hasNamedAdjacent {
		return nil
	}
	if anyUnpowered {
		return colorHazardBackground
	}
	if anyPowered {
		return colorWallBgPowered
	}
	return nil
}

func isPowerGridSeedCellFromSnap(pg *powerGridSnapshot, cell *world.Cell) (isSeed bool, isGenerator bool) {
	if pg == nil || cell == nil || !pg.active {
		return false, false
	}
	if pg.maintenanceMenuRoom != "" &&
		cell.Row == pg.maintTerminalRow && cell.Col == pg.maintTerminalCol {
		return true, false
	}
	if pg.overlayDevActive &&
		cell.Row == pg.overlaySeedRow && cell.Col == pg.overlaySeedCol {
		return true, true
	}
	return false, false
}

// powerGridCellBg returns a tile tint for cells on the active power grid overlay.
// Live cells use green conduit tints; armed-but-offline cells use red (disconnected from routing).
func powerGridCellBg(g *state.Game, pg *powerGridSnapshot, cell *world.Cell) color.Color {
	if g == nil || cell == nil || !powerGridOverlayActiveFromSnap(pg) || !cellVisibleForPowerGridOverlay(g, cell) {
		return nil
	}
	onLive := snapshotHasCell(pg.liveCells, cell)
	onOfflineArmed := snapshotHasCell(pg.armedCells, cell) && !onLive
	if !onLive && !onOfflineArmed {
		return nil
	}
	if isSeed, isGen := isPowerGridSeedCellFromSnap(pg, cell); isSeed {
		if isGen {
			if onLive {
				return colorPowerGridGenerator
			}
			return colorPowerGridGeneratorOff
		}
		if onLive {
			return colorPowerGridTerminal
		}
		return colorPowerGridTerminalOff
	}
	conduit := colorPowerGridCell
	if !onLive {
		conduit = colorPowerGridCellOff
	}
	if cell.Name == "Corridor" {
		return conduit
	}
	if gameworld.HasDoor(cell) || gameworld.HasPowerRelay(cell) {
		return conduit
	}
	return nil
}

func (e *EbitenRenderer) drawPowerGridOverlay(screen *ebiten.Image, g *state.Game, pg *powerGridSnapshot, mapX, mapY float64, startRow, startCol int) {
	if !powerGridOverlayActiveFromSnap(pg) {
		return
	}
	liveCount := len(pg.liveCells)
	armedCount := len(pg.armedCells)
	if liveCount == 0 && armedCount == 0 {
		return
	}

	const lineWidth = 1.5
	if liveCount > 0 {
		e.drawPowerGridEdges(screen, g, pg.liveCells, pg.liveCells, mapX, mapY, startRow, startCol, lineWidth, powerGridEdgeColor())
	}
	if armedCount > 0 {
		e.drawPowerGridOfflineEdges(screen, g, pg.armedCells, pg.liveCells, mapX, mapY, startRow, startCol, lineWidth, powerGridEdgeOffColor())
		e.drawPowerGridBoundaryEdges(screen, g, pg.armedCells, pg.liveCells, mapX, mapY, startRow, startCol, lineWidth, powerGridEdgeOffColor())
	}
}

func (e *EbitenRenderer) drawPowerGridEdges(screen *ebiten.Image, g *state.Game, grid, requireBoth map[uint64]bool, mapX, mapY float64, startRow, startCol int, lineWidth float32, edgeColor color.Color) {
	seenEdge := make(map[uint64]bool)
	for key := range grid {
		row := int(uint32(key >> 32))
		col := int(uint32(key))
		cell := g.Grid.GetCell(row, col)
		if cell == nil || !cellVisibleForPowerGridOverlay(g, cell) {
			continue
		}
		cx, cy := mapCellCenterScreen(mapX, mapY, cell.Row, cell.Col, startRow, startCol, e.tileSize)
		for _, n := range cell.GetNeighbors() {
			if n == nil || !snapshotHasCell(requireBoth, n) || !cellVisibleForPowerGridOverlay(g, n) {
				continue
			}
			edgeKey := gridEdgeKey(cell, n)
			if seenEdge[edgeKey] {
				continue
			}
			seenEdge[edgeKey] = true
			nx, ny := mapCellCenterScreen(mapX, mapY, n.Row, n.Col, startRow, startCol, e.tileSize)
			vector.StrokeLine(screen, cx, cy, nx, ny, lineWidth, edgeColor, false)
		}
	}
}

func (e *EbitenRenderer) drawPowerGridOfflineEdges(screen *ebiten.Image, g *state.Game, armed, live map[uint64]bool, mapX, mapY float64, startRow, startCol int, lineWidth float32, edgeColor color.Color) {
	seenEdge := make(map[uint64]bool)
	for key := range armed {
		row := int(uint32(key >> 32))
		col := int(uint32(key))
		cell := g.Grid.GetCell(row, col)
		if snapshotHasCell(live, cell) || !cellVisibleForPowerGridOverlay(g, cell) {
			continue
		}
		cx, cy := mapCellCenterScreen(mapX, mapY, cell.Row, cell.Col, startRow, startCol, e.tileSize)
		for _, n := range cell.GetNeighbors() {
			if n == nil || snapshotHasCell(live, n) || !snapshotHasCell(armed, n) || !cellVisibleForPowerGridOverlay(g, n) {
				continue
			}
			edgeKey := gridEdgeKey(cell, n)
			if seenEdge[edgeKey] {
				continue
			}
			seenEdge[edgeKey] = true
			nx, ny := mapCellCenterScreen(mapX, mapY, n.Row, n.Col, startRow, startCol, e.tileSize)
			vector.StrokeLine(screen, cx, cy, nx, ny, lineWidth, edgeColor, false)
		}
	}
}

func (e *EbitenRenderer) drawPowerGridBoundaryEdges(screen *ebiten.Image, g *state.Game, overlay, live map[uint64]bool, mapX, mapY float64, startRow, startCol int, lineWidth float32, edgeColor color.Color) {
	seenEdge := make(map[uint64]bool)
	for key := range overlay {
		row := int(uint32(key >> 32))
		col := int(uint32(key))
		cell := g.Grid.GetCell(row, col)
		if snapshotHasCell(live, cell) || !cellVisibleForPowerGridOverlay(g, cell) {
			continue
		}
		cx, cy := mapCellCenterScreen(mapX, mapY, cell.Row, cell.Col, startRow, startCol, e.tileSize)
		for _, n := range cell.GetNeighbors() {
			if n == nil || !snapshotHasCell(live, n) || !cellVisibleForPowerGridOverlay(g, n) {
				continue
			}
			edgeKey := gridEdgeKey(cell, n)
			if seenEdge[edgeKey] {
				continue
			}
			seenEdge[edgeKey] = true
			nx, ny := mapCellCenterScreen(mapX, mapY, n.Row, n.Col, startRow, startCol, e.tileSize)
			vector.StrokeLine(screen, cx, cy, nx, ny, lineWidth, edgeColor, false)
		}
	}
}

func gridEdgeKey(a, b *world.Cell) uint64 {
	if a == nil || b == nil {
		return 0
	}
	ra, ca := uint32(a.Row), uint32(a.Col)
	rb, cb := uint32(b.Row), uint32(b.Col)
	if ra > rb || (ra == rb && ca > cb) {
		ra, ca, rb, cb = rb, cb, ra, ca
	}
	return (uint64(ra) << 48) | (uint64(ca) << 32) | (uint64(rb) << 16) | uint64(cb)
}
