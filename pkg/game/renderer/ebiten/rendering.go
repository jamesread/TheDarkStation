// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
package ebiten

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/leonelquinteros/gotext"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// Draw renders the game to the screen (Ebiten interface)
func (e *EbitenRenderer) Draw(screen *ebiten.Image) {
	// Fill background first
	screen.Fill(colorBackground)
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	defer e.drawBuildLabel(screen, screenWidth, screenHeight)

	if load := e.levelGenSnapshot(); load.active {
		if e.monoFontSource != nil && e.sansFontSource != nil {
			e.drawLevelGenLoading(screen, load)
		}
		e.drawConsole(screen)
		sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
		e.gameMutex.RLock()
		g := e.game
		e.gameMutex.RUnlock()
		e.drawDebugTopRight(screen, sw, sh, g)
		return
	}

	// Check if menu overlays are active - these should be drawn even without valid game state
	e.genericMenuMutex.RLock()
	genericMenuActive := e.genericMenuActive
	title := e.genericMenuTitle
	e.genericMenuMutex.RUnlock()

	// Get snapshot for consistent rendering
	e.snapshotMutex.RLock()
	snap := e.snapshot
	e.snapshotMutex.RUnlock()

	e.gameMutex.RLock()
	g := e.game
	e.gameMutex.RUnlock()

	// If menu is active but game state is invalid, draw only the menu and return
	if genericMenuActive && (!snap.valid || g == nil) {
		if e.monoFontSource == nil || e.sansFontSource == nil {
			// Can't draw menu without fonts
			return
		}

		// Draw floating tiles background for main menu (on top of background fill, before menu overlay)
		if genericMenuActive && title == "The Dark Station" {
			e.drawFloatingTilesBackground(screen)
		}

		// Draw menu overlays
		if genericMenuActive {
			e.drawGenericMenuOverlay(screen)
		}

		// Draw console overlay
		e.drawConsole(screen)

		e.drawConfirmDialog(screen)

		// Debug overlays (top right): FPS, player X/Y
		sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
		e.drawDebugTopRight(screen, sw, sh, g)
		return
	}

	if !snap.valid || e.monoFontSource == nil || e.sansFontSource == nil {
		// Can't draw without valid snapshot or fonts
		return
	}

	if g == nil {
		return
	}

	if g.GameComplete {
		e.drawGameCompleteScreen(screen, g, &snap, screenWidth, screenHeight, genericMenuActive)
		e.drawTextInputDialog(screen)
		e.drawConfirmDialog(screen)
		e.drawConsole(screen)
		e.drawDeveloperMessage(screen, screenWidth, screenHeight)
		e.drawTransientNotification(screen, screenWidth, screenHeight)
		e.drawDebugTopRight(screen, screenWidth, screenHeight, g)
		return
	}

	// Expand the viewport when the framebuffer is larger than the logical window (HiDPI).
	if screenWidth > 0 && screenHeight > 0 {
		if neededCols := viewportTilesForAxis(screenWidth, e.tileSize); neededCols > e.viewportCols {
			e.viewportCols = neededCols
		}
		if neededRows := viewportTilesForAxis(screenHeight, e.tileSize); neededRows > e.viewportRows {
			e.viewportRows = neededRows
		}
	}

	// Map draw area is the full window; tile grid is anchored on the player at screen center.
	mapAreaWidth := screenWidth
	mapAreaHeight := screenHeight

	e.drawGameplayMapLayer(screen, g, &snap, screenWidth, screenHeight, mapAreaWidth, mapAreaHeight, genericMenuActive)

	// Text input dialog (centered modal; e.g. load level seed)
	e.drawTextInputDialog(screen)
	e.drawConfirmDialog(screen)

	// Draw console overlay
	e.drawConsole(screen)

	// Developer message (bottom-left; map dump, etc.)
	e.drawDeveloperMessage(screen, screenWidth, screenHeight)

	e.drawTransientNotification(screen, screenWidth, screenHeight)

	// Debug overlays (top right): FPS, player X/Y
	e.drawDebugTopRight(screen, screenWidth, screenHeight, g)
}

func (e *EbitenRenderer) drawGameplayMapLayer(screen *ebiten.Image, g *state.Game, snap *renderSnapshot, screenWidth, screenHeight, mapAreaWidth, mapAreaHeight int, genericMenuActive bool) {
	uiFontSize := e.getUIFontSize()
	statusBarHeight := int(uiFontSize)*2 + 20
	const objectivesWindowMargin = 12

	e.drawHeaderFromSnapshot(screen, snap, screenWidth, 0)
	e.drawMap(screen, g, screenWidth, screenHeight, snap)
	if e.DrawMapAreaBorderEnabled() {
		e.drawMapAreaBorderOutline(screen, 0, 0, mapAreaWidth, mapAreaHeight)
	}
	statusX := objectivesWindowMargin + 10
	statusY := objectivesWindowMargin + 5
	e.drawStatusBarFromSnapshot(screen, snap, statusX, statusY, mapAreaWidth, statusBarHeight)
	if genericMenuActive {
		e.drawGenericMenuOverlay(screen)
	}
}

func (e *EbitenRenderer) drawGameCompleteScreen(screen *ebiten.Image, g *state.Game, snap *renderSnapshot, screenWidth, screenHeight int, genericMenuActive bool) {
	if g == nil || snap == nil {
		return
	}

	fadeActive, fade := creditsMapTransitionFade(g)
	showMap := g.CompletionPhase == state.CompletionPhaseSummary || fadeActive

	if showMap {
		if screenWidth > 0 && screenHeight > 0 {
			if neededCols := viewportTilesForAxis(screenWidth, e.tileSize); neededCols > e.viewportCols {
				e.viewportCols = neededCols
			}
			if neededRows := viewportTilesForAxis(screenHeight, e.tileSize); neededRows > e.viewportRows {
				e.viewportRows = neededRows
			}
		}
		e.drawGameplayMapLayer(screen, g, snap, screenWidth, screenHeight, screenWidth, screenHeight, genericMenuActive)
		if fadeActive {
			scrim := color.RGBA{15, 15, 26, uint8(220 * fade)}
			vector.DrawFilledRect(screen, 0, 0, float32(screenWidth), float32(screenHeight), scrim, false)
		}
	}

	if g.CompletionPhase == state.CompletionPhaseSummary && !fadeActive {
		e.drawCompletionSummaryScrim(screen, screenWidth, screenHeight)
		e.drawCompletionSummary(screen, g, screenWidth, screenHeight, 1)
		return
	}

	if g.CompletionPhase == state.CompletionPhaseCredits {
		creditsAlpha := 1.0
		if fadeActive {
			creditsAlpha = fade
		} else if g.CreditsLineStartMs == 0 {
			return
		}
		if creditsAlpha <= 0 {
			return
		}
		e.ensureFloatingTiles(screenWidth, screenHeight)
		e.drawFloatingTilesBackground(screen)
		if fadeActive {
			scrim := color.RGBA{15, 15, 26, uint8(185 * fade)}
			vector.DrawFilledRect(screen, 0, 0, float32(screenWidth), float32(screenHeight), scrim, false)
		} else {
			e.drawCompletionScrim(screen, screenWidth, screenHeight)
		}
		e.drawCompletionCredits(screen, g, screenWidth, screenHeight, creditsAlpha)
	}
}

func (e *EbitenRenderer) drawCompletionSummaryScrim(screen *ebiten.Image, w, h int) {
	scrim := color.RGBA{15, 15, 26, 140}
	vector.DrawFilledRect(screen, 0, 0, float32(w), float32(h), scrim, false)
}

// drawHeaderFromSnapshot draws the header (currently empty - deck number moved to objectives panel)
func (e *EbitenRenderer) drawHeaderFromSnapshot(screen *ebiten.Image, snap *renderSnapshot, screenWidth int, headerHeight int) {
	// Header is now empty - deck number has been moved to objectives panel
}

// drawDebugTopRight draws FPS and/or player grid position in the top-right corner.
func (e *EbitenRenderer) drawDebugTopRight(screen *ebiten.Image, screenWidth, screenHeight int, g *state.Game) {
	fontSize := e.getUIFontSize()
	margin := 12
	y := margin + int(fontSize)

	if e.ShowFPSCounterEnabled() {
		fpsText := fmt.Sprintf("%.0f FPS", ebiten.ActualFPS())
		x := screenWidth - int(e.getTextWidth(fpsText)) - margin
		e.drawColoredText(screen, fpsText, x, y, colorSubtle)
		y += int(fontSize)
	}

	if e.ShowPlayerPositionEnabled() {
		if posText := e.playerPositionDebugText(g); posText != "" {
			x := screenWidth - int(e.getTextWidth(posText)) - margin
			e.drawColoredText(screen, posText, x, y, colorSubtle)
		}
	}
}

func (e *EbitenRenderer) playerPositionDebugText(g *state.Game) string {
	if g == nil || g.CurrentCell == nil {
		return ""
	}
	return fmt.Sprintf("X: %d Y: %d", g.CurrentCell.Col, g.CurrentCell.Row)
}

// syncPlayModeCamera keeps maintenance pan origin aligned with the on-screen play camera.
// During normal play drawMap uses playerMove.visualPosition directly; these fields are only
// read when the maintenance menu opens. Without syncing they stay at zero and the first
// maint pan eases from map origin.
func (e *EbitenRenderer) syncPlayModeCamera(g *state.Game) {
	if g == nil || g.CurrentCell == nil {
		return
	}
	nowMs := e.menuAnimClockMilli
	if nowMs == 0 {
		nowMs = time.Now().UnixMilli()
	}
	row, col := e.playerMove.visualPosition(g.Level, g.CurrentCell.Row, g.CurrentCell.Col, e.snapSeq, nowMs)
	e.cameraCenterRow = row
	e.cameraCenterCol = col
	e.cameraTargetRow = row
	e.cameraTargetCol = col
	e.cameraPlaySynced = true
}

// advanceMaintenanceCamera sets the map camera center. With the maintenance room list open,
// the camera eases toward the selected room center (~1s smootherstep). Normal play uses
// playerMoveTransition in drawMap (visual row/col + matching camera pan, ~140ms ease-out).
func (e *EbitenRenderer) advanceMaintenanceCamera() {
	const maintCameraPanMs = 1000

	e.gameMutex.RLock()
	g := e.game
	e.gameMutex.RUnlock()
	if g == nil || g.CurrentCell == nil || g.Grid == nil {
		return
	}

	targetRow := float64(g.CurrentCell.Row)
	targetCol := float64(g.CurrentCell.Col)
	if g.MaintenanceMenuRoom != "" {
		if r, c, ok := roomCenter(g.Grid, g.MaintenanceMenuRoom); ok {
			targetRow, targetCol = float64(r), float64(c)
		}
	}

	if g.MaintenanceMenuRoom != "" {
		if !e.cameraPlaySynced {
			e.syncPlayModeCamera(g)
		}
		const posEps = 1e-6
		if math.Abs(e.cameraTargetRow-targetRow) > posEps || math.Abs(e.cameraTargetCol-targetCol) > posEps {
			e.cameraFromRow = e.cameraCenterRow
			e.cameraFromCol = e.cameraCenterCol
			e.cameraTargetRow = targetRow
			e.cameraTargetCol = targetCol
			e.cameraTransitionStartNano = e.menuAnimTimeNano
			e.maintPanCameraTweenActive = true
			maintPanLogf(
				"tween TRIGGER from=(%.6f,%.6f) target=(%.6f,%.6f) room=%q transition_start_ns=%d dur_ms=%d",
				e.cameraFromRow, e.cameraFromCol,
				e.cameraTargetRow, e.cameraTargetCol,
				g.MaintenanceMenuRoom, e.cameraTransitionStartNano, maintCameraPanMs)
		}

		elapsedNano := e.menuAnimTimeNano - e.cameraTransitionStartNano
		if elapsedNano < 0 {
			elapsedNano = 0
		}
		elapsedMs := float64(elapsedNano) / 1e6
		progress := elapsedMs / float64(maintCameraPanMs)
		if progress >= 1.0 {
			e.cameraCenterRow = e.cameraTargetRow
			e.cameraCenterCol = e.cameraTargetCol
			progress = 1.0
			if e.maintPanCameraTweenActive {
				maintPanLogf(
					"tween COMPLETE cam=(%.6f,%.6f) tgt=(%.6f,%.6f) room=%q elapsed_ms=%.2f",
					e.cameraCenterRow, e.cameraCenterCol,
					e.cameraTargetRow, e.cameraTargetCol, g.MaintenanceMenuRoom, elapsedMs)
				e.maintPanCameraTweenActive = false
			}
		} else {
			t := smootherstep(progress)
			e.cameraCenterRow = e.cameraFromRow + (e.cameraTargetRow-e.cameraFromRow)*t
			e.cameraCenterCol = e.cameraFromCol + (e.cameraTargetCol-e.cameraFromCol)*t
		}
		maintPanLogfThrottled("Update maint pan progress=%.4f cam=%.4f,%.4f tgt=%.4f,%.4f room=%q vr=%d,%d",
			progress, e.cameraCenterRow, e.cameraCenterCol, e.cameraTargetRow, e.cameraTargetCol,
			g.MaintenanceMenuRoom, e.viewportRows, e.viewportCols)
		return
	}

	e.maintPanCameraTweenActive = false
	// Normal play: camera is synced to the render snapshot in RenderFrame only.
	// Do not follow live g.CurrentCell here — Update runs between RenderFrame ticks
	// and would desync the camera from the snapshot during slow frames (visible jitter).
}

// smootherstep maps [0,1] → [0,1] with zero first and second derivatives at the endpoints (Wikipedia / Perlin).
func smootherstep(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * t * (t*(t*6-15) + 10)
}

// maintCameraPanTweening is true while the maintenance map camera ease has not snapped to its room target.
// Used for fractional map blits and to trim expensive overlay draws during the pan — frame pacing, not lerping,
// tends to dominate perceived stutter: progress is sampled on missed frames from wall-clock elapsed.
func maintCameraPanTweening(g *state.Game, camRow, camCol, tgtRow, tgtCol float64) bool {
	if g == nil || g.MaintenanceMenuRoom == "" {
		return false
	}
	const posEps = 1e-5
	return math.Abs(camRow-tgtRow) > posEps || math.Abs(camCol-tgtCol) > posEps
}

// mapCameraStart returns the top-left grid cell for the current viewport.
// At rest the player sits on the center tile with equal rows/cols on each side.
func (e *EbitenRenderer) mapCameraStart(g *state.Game) (startRow, startCol int) {
	panningMaint := maintCameraPanTweening(g, e.cameraCenterRow, e.cameraCenterCol, e.cameraTargetRow, e.cameraTargetCol)
	if !panningMaint {
		centerRow := int(math.Round(e.cameraCenterRow))
		centerCol := int(math.Round(e.cameraCenterCol))
		return centerRow - e.viewportRows/2, centerCol - e.viewportCols/2
	}
	return mapCameraStartAt(e.cameraCenterRow, e.cameraCenterCol, e.viewportRows, e.viewportCols)
}

func (e *EbitenRenderer) playVisualCamera(snap *renderSnapshot) (camRow, camCol, visualRow, visualCol float64, startRow, startCol int) {
	nowMs := e.menuAnimClockMilli
	if nowMs == 0 {
		nowMs = time.Now().UnixMilli()
	}
	visualRow, visualCol = e.playerMove.visualPosition(snap.level, snap.playerRow, snap.playerCol, snap.seq, nowMs)
	camRow, camCol = visualRow, visualCol
	startRow, startCol = mapCameraStartAt(camRow, camCol, e.viewportRows, e.viewportCols)
	return camRow, camCol, visualRow, visualCol, startRow, startCol
}

// drawMap renders the game map
func (e *EbitenRenderer) drawMap(screen *ebiten.Image, g *state.Game, screenWidth, screenHeight int, snap *renderSnapshot) {
	if g.CurrentCell == nil || g.Grid == nil {
		return
	}

	// Camera: maintenance pan uses Update-driven tween; normal play follows smooth player motion.
	panningMaint := maintCameraPanTweening(g, e.cameraCenterRow, e.cameraCenterCol, e.cameraTargetRow, e.cameraTargetCol)

	var camRow, camCol, visualRow, visualCol float64
	var startRow, startCol int
	if snap.hazardTour != nil {
		nowMs := e.menuAnimClockMilli
		if nowMs == 0 {
			nowMs = time.Now().UnixMilli()
		}
		var ok bool
		camRow, camCol, ok = snap.hazardTour.CameraAt(nowMs)
		if !ok {
			camRow, camCol, visualRow, visualCol, startRow, startCol = e.playVisualCamera(snap)
		} else {
			startRow, startCol = mapCameraStartAt(camRow, camCol, e.viewportRows, e.viewportCols)
			visualRow, visualCol = float64(snap.playerRow), float64(snap.playerCol)
		}
	} else if snap.hazardClear != nil {
		nowMs := e.menuAnimClockMilli
		if nowMs == 0 {
			nowMs = time.Now().UnixMilli()
		}
		var ok bool
		camRow, camCol, ok = snap.hazardClear.CameraAt(nowMs)
		if !ok {
			camRow, camCol, visualRow, visualCol, startRow, startCol = e.playVisualCamera(snap)
		} else {
			startRow, startCol = mapCameraStartAt(camRow, camCol, e.viewportRows, e.viewportCols)
			visualRow, visualCol = float64(snap.playerRow), float64(snap.playerCol)
		}
	} else if g.MaintenanceMenuRoom != "" {
		camRow = e.cameraCenterRow
		camCol = e.cameraCenterCol
		startRow, startCol = e.mapCameraStart(g)
		visualRow, visualCol = float64(snap.playerRow), float64(snap.playerCol)
	} else {
		camRow, camCol, visualRow, visualCol, startRow, startCol = e.playVisualCamera(snap)
	}

	// Ensure map buffer exists and is correctly sized
	bufW := e.viewportCols * e.tileSize
	bufH := e.viewportRows * e.tileSize
	if e.mapBuffer == nil || e.mapBufferWidth != bufW || e.mapBufferHeight != bufH {
		if e.mapBuffer != nil {
			e.mapBuffer.Dispose()
		}
		e.mapBuffer = ebiten.NewImage(bufW, bufH)
		e.mapBufferWidth = bufW
		e.mapBufferHeight = bufH
		e.invalidateMapDrawCache()
	}

	var mapScrX, mapScrY float64
	if blitX, blitY, hit := e.mapDrawCacheHit(snap.seq, camRow, camCol, startRow, startCol, bufW, bufH); hit {
		mapScrX, mapScrY = blitX, blitY
	} else {
		// Draw tiles to offscreen buffer at integer coordinates - eliminates per-tile
		// sub-pixel jitter. The single blit with fractional offset is smooth.
		e.mapBuffer.Fill(colorBackground)
		pg := &snap.powerGrid
		for vRow := 0; vRow < e.viewportRows; vRow++ {
			for vCol := 0; vCol < e.viewportCols; vCol++ {
				e.drawTileToBuffer(e.mapBuffer, startRow, startCol, vRow, vCol, g, snap, pg)
			}
		}
		mapScrX, mapScrY = mapCameraScreenOrigin(screenWidth, screenHeight, camRow, camCol, startRow, startCol, e.tileSize)
		if g.MaintenanceMenuRoom != "" && !panningMaint {
			mapScrX = math.Round(mapScrX)
			mapScrY = math.Round(mapScrY)
		}
		e.storeMapDrawCache(snap.seq, camRow, camCol, startRow, startCol, mapScrX, mapScrY, bufW, bufH)
	}

	if maintPanDebugOn() && g.MaintenanceMenuRoom != "" {
		e.maintPanDrawCount++
		if e.maintPanDrawCount == 2 {
			maintPanLogf("second Draw() in same Update tick animClockMs=%d mapScr=%.0f,%.0f cam=%.4f,%.4f",
				e.menuAnimClockMilli, mapScrX, mapScrY, e.cameraCenterRow, e.cameraCenterCol)
		}
	}

	// Blit map buffer on an integer pixel grid (see mapScrX/Y above).
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(mapScrX, mapScrY)
	screen.DrawImage(e.mapBuffer, op)

	pg := &snap.powerGrid
	e.drawFOVRays(screen, g, snap, mapScrX, mapScrY, startRow, startCol)
	e.drawPowerGridOverlay(screen, g, pg, mapScrX, mapScrY, startRow, startCol)

	// Draw overlays using the same screen origin so labels/callouts match the quantized blit.
	mapXF := mapScrX
	mapYF := mapScrY
	e.drawRoomLabels(screen, snap, mapXF, mapYF, startRow, startCol)
	e.drawEnvironmentalPlaques(screen, snap, mapXF, mapYF, startRow, startCol)
	e.drawCallouts(screen, snap, mapXF, mapYF, startRow, startCol)
	e.drawLongUseProgress(screen, snap, mapXF, mapYF, startRow, startCol)
	e.drawPlayerWithDebounce(screen, g, snap, mapXF, mapYF, visualRow, visualCol, startRow, startCol)
	e.drawExitAnimation(screen, snap, mapXF, mapYF, startRow, startCol)
}

// drawTileToBuffer draws a single tile to the map buffer at integer coordinates.
// Used for jitter-free camera transitions - all tiles at exact pixel positions.
func (e *EbitenRenderer) drawTileToBuffer(buf *ebiten.Image, startRow, startCol, vRow, vCol int, g *state.Game, snap *renderSnapshot, pg *powerGridSnapshot) {
	if g == nil || g.Grid == nil {
		return
	}

	mapRow := startRow + vRow
	mapCol := startCol + vCol

	x := vCol * e.tileSize
	y := vRow * e.tileSize

	cell := g.Grid.GetCell(mapRow, mapCol)

	cellRenderOptions := e.getCellRenderOptions(g, cell, snap, false)

	if cell != nil && cell.Row == snap.playerRow && cell.Col == snap.playerCol {
		// Draw floor under the player (player drawn separately as overlay)
		underfootOptions := e.getCellRenderOptions(g, cell, snap, true)
		customBg := e.getTileCustomBg(g, cell, snap, &underfootOptions, pg)
		e.drawTileWithBg(buf, " ", x, y, colorBackground, underfootOptions.HasBackground, customBg)
		return
	}

	customBg := e.getTileCustomBg(g, cell, snap, &cellRenderOptions, pg)
	e.drawTileWithBg(buf, cellRenderOptions.Icon, x, y, cellRenderOptions.Color, cellRenderOptions.HasBackground, customBg)
}

// getTileCustomBg returns the background color for a cell (focus, hazard, floor, exit, etc.).
func (e *EbitenRenderer) getTileCustomBg(g *state.Game, cell *world.Cell, snap *renderSnapshot, opts *CellRenderOptions, pg *powerGridSnapshot) color.Color {
	var customBg color.Color
	isFocused := cell != nil && cell.Row == snap.focusedCellRow && cell.Col == snap.focusedCellCol
	isInteractable := false
	if cell != nil {
		for _, ic := range snap.interactableCells {
			if cell.Row == ic.row && cell.Col == ic.col {
				isInteractable = true
				break
			}
		}
	}
	needsClearing := false
	if cell != nil && (g.HasMap || cell.Discovered) {
		if gameworld.HasBlockingHazard(cell) || gameworld.HasLockedDoor(cell) {
			needsClearing = true
		}
	}
	if cell != nil {
		if (g.HasMap || cell.Discovered) && gameworld.HasBlockingHazard(cell) {
			customBg = colorHazardBackground
			if alpha := hazardClearVisualAlpha(snap, cell); alpha < 1 {
				customBg = e.applyAlpha(colorHazardBackground, alpha)
			}
		} else if (g.HasMap || cell.Discovered) && gameworld.HasDoor(cell) {
			roomName := gameworld.GetGameData(cell).Door.RoomName
			if !snapCellHasLivePower(snap, cell) {
				if snapRoomManualEgressReleased(snap, roomName) {
					customBg = focusPlateForForeground(colorDoorLocked)
				} else {
					customBg = colorHazardBackground
				}
			}
		} else if (g.HasMap || cell.Discovered) && gameworld.HasMaintenanceTerminal(cell) {
			data := gameworld.GetGameData(cell)
			if data.MaintenanceTerm != nil && !data.MaintenanceTerm.Powered {
				customBg = colorHazardBackground
			}
		} else if (g.HasMap || cell.Discovered) && gameworld.HasTerminal(cell) {
			if cell.Room && !snapRoomCCTVPowered(snap, cell.Name) {
				customBg = colorHazardBackground
			}
		} else if (g.HasMap || cell.Discovered) && gameworld.HasHazardControl(cell) {
			if cell.Room && !snapRoomCCTVPowered(snap, cell.Name) {
				customBg = colorHazardBackground
			}
		}
		if customBg == nil && opts != nil && opts.Icon == IconWall && snapMaintenanceMenuRoom(snap) != "" &&
			hasAdjacentRoomNamed(cell, snapMaintenanceMenuRoom(snap)) {
			customBg = colorWallHighlight
		}
		if customBg == nil && powerGridOverlayActiveFromSnap(pg) {
			if floorBg := powerGridRoomFloorBg(g, pg, cell, opts); floorBg != nil {
				customBg = floorBg
			} else if opts != nil && opts.Icon == IconWall {
				if wallBg := powerGridWallBg(g, pg, cell); wallBg != nil {
					customBg = wallBg
				}
			}
		}
		if customBg == nil {
			if gridBg := powerGridCellBg(g, pg, cell); gridBg != nil {
				customBg = gridBg
			}
		}
		if customBg == nil && opts != nil && opts.BackgroundColor != nil {
			customBg = opts.BackgroundColor
		}
		if customBg == nil && needsClearing {
			if opts != nil {
				customBg = focusPlateForForeground(opts.Color)
			} else {
				customBg = colorBlockedBackground
			}
		} else if customBg == nil && gameworld.HasPoweredGenerator(cell) {
			customBg = colorWallBgPowered
		} else if isFocused || isInteractable {
			if opts != nil {
				customBg = focusPlateForForeground(opts.Color)
			} else {
				customBg = colorFocusBackground
			}
		} else if cell != nil && cell.ExitCell && (g.HasMap || cell.Discovered) && setup.ExitLiftReady(g) {
			customBg = e.getPulsingExitBackgroundColor()
		}
	}
	return customBg
}

// focusPlateForForeground returns a dark, semi-opaque tile background aligned with the cell icon color.
// Warm families (amber maintenance, red alarm, bright yellow locks) use hue-consistent dark plates; other
// colors use a restrained cool-biased mix (see specs/map-tile-focus-and-contrast.md).
func focusPlateForForeground(fg color.Color) color.Color {
	if isAmberTerminalForeground(fg) {
		return warmFocusPlateForForeground(fg)
	}
	if isBrightYellowLockForeground(fg) {
		return yellowFamilyFocusPlate(fg)
	}
	if isRedDominantForeground(fg) {
		return redFamilyFocusPlate(fg)
	}
	if isPinkMagentaForeground(fg) {
		return magentaFocusPlateForForeground(fg)
	}
	r32, g32, b32, _ := fg.RGBA()
	r8 := uint8(r32 >> 8)
	g8 := uint8(g32 >> 8)
	b8 := uint8(b32 >> 8)

	// Default: dark cool base + scaled inverse of foreground (works for blues/greens/purples).
	const baseR, baseG, baseB = 18, 22, 38
	invR := 255 - r8
	invG := 255 - g8
	invB := 255 - b8
	const a = 220
	outR := uint8(min(255, int(baseR)+int(invR)*70/255))
	outG := uint8(min(255, int(baseG)+int(invG)*65/255))
	outB := uint8(min(255, int(baseB)+int(invB)*75/255))
	return color.RGBA{R: outR, G: outG, B: outB, A: a}
}

// isBrightYellowLockForeground matches locked-door yellow (+) without picking up orange batteries etc.
func isBrightYellowLockForeground(fg color.Color) bool {
	r32, g32, b32, _ := fg.RGBA()
	r8 := uint8(r32 >> 8)
	g8 := uint8(g32 >> 8)
	b8 := uint8(b32 >> 8)
	return r8 >= 235 && g8 >= 215 && b8 <= 50
}

// isRedDominantForeground matches hazard reds, unpowered generators, red exit tones, etc.
func isRedDominantForeground(fg color.Color) bool {
	r32, g32, b32, _ := fg.RGBA()
	r8 := uint8(r32 >> 8)
	g8 := uint8(g32 >> 8)
	b8 := uint8(b32 >> 8)
	if r8 < 130 {
		return false
	}
	return int(r8) >= int(g8)+20 && int(r8) >= int(b8)+20
}

// redFamilyFocusPlate is a dark plate in the red/alarm family (not complementary teal).
func redFamilyFocusPlate(fg color.Color) color.Color {
	r32, g32, b32, _ := fg.RGBA()
	r8 := uint8(r32 >> 8)
	g8 := uint8(g32 >> 8)
	b8 := uint8(b32 >> 8)
	const a = 220
	const br, bgBase, bb = 72, 22, 22
	outR := uint8(min(255, br+int(r8)*48/255))
	outG := uint8(min(255, bgBase+int(g8)*38/255))
	outB := uint8(min(255, bb+int(b8)*38/255))
	return color.RGBA{R: outR, G: outG, B: outB, A: a}
}

// yellowFamilyFocusPlate is a dark gold/brown behind bright yellow glyphs.
func yellowFamilyFocusPlate(fg color.Color) color.Color {
	r32, g32, b32, _ := fg.RGBA()
	r8 := uint8(r32 >> 8)
	g8 := uint8(g32 >> 8)
	b8 := uint8(b32 >> 8)
	const a = 220
	const br, bgBase, bb = 52, 44, 10
	outR := uint8(min(255, br+int(r8)*48/255))
	outG := uint8(min(255, bgBase+int(g8)*42/255))
	outB := uint8(min(255, bb+int(b8)*35/255))
	return color.RGBA{R: outR, G: outG, B: outB, A: a}
}

// isAmberTerminalForeground reports hues like maintenance/CCTV orange (not pure yellow door-locked, etc.).
func isAmberTerminalForeground(fg color.Color) bool {
	r32, g32, b32, _ := fg.RGBA()
	r8 := uint8(r32 >> 8)
	g8 := uint8(g32 >> 8)
	b8 := uint8(b32 >> 8)
	return r8 > 200 && g8 >= 100 && g8 < 235 && b8 < 100
}

// warmFocusPlateForForeground is a dark amber/brown plate aligned with maintenance terminal coloring.
func warmFocusPlateForForeground(fg color.Color) color.Color {
	r32, g32, b32, _ := fg.RGBA()
	r8 := uint8(r32 >> 8)
	g8 := uint8(g32 >> 8)
	b8 := uint8(b32 >> 8)
	const a = 220
	const br, bgBase, bb = 50, 32, 10
	outR := uint8(min(255, br+int(r8)*55/255))
	outG := uint8(min(255, bgBase+int(g8)*50/255))
	outB := uint8(min(255, bb+int(b8)*40/255))
	return color.RGBA{R: outR, G: outG, B: outB, A: a}
}

// isPinkMagentaForeground matches bright magenta/pink glyphs (unchecked furniture,
// hazard-control pink tones) where complementary math would wrongly shift toward green.
func isPinkMagentaForeground(fg color.Color) bool {
	r32, g32, b32, _ := fg.RGBA()
	r8 := uint8(r32 >> 8)
	g8 := uint8(g32 >> 8)
	b8 := uint8(b32 >> 8)
	if r8 < 180 || b8 < 180 {
		return false
	}
	minRB := r8
	if b8 < minRB {
		minRB = b8
	}
	return int(minRB)-int(g8) >= 35
}

// magentaFocusPlateForForeground is a dark magenta/plum plate in the pink glyph family (not teal/green complementary).
func magentaFocusPlateForForeground(fg color.Color) color.Color {
	r32, g32, b32, _ := fg.RGBA()
	r8 := uint8(r32 >> 8)
	g8 := uint8(g32 >> 8)
	b8 := uint8(b32 >> 8)
	const a = 220
	const br, bgBase, bb = 44, 18, 44
	outR := uint8(min(255, br+int(r8)*45/255))
	outG := uint8(min(255, bgBase+int(g8)*42/255))
	outB := uint8(min(255, bb+int(b8)*45/255))
	return color.RGBA{R: outR, G: outG, B: outB, A: a}
}

// drawTileWithBg draws a single tile with optional custom background color.
// When icon is " " or "", only the background is drawn (e.g. under the player).
func (e *EbitenRenderer) drawTileWithBg(screen *ebiten.Image, icon string, x, y int, col color.Color, hasBackground bool, bgColor color.Color) {
	e.drawTileWithBgF(screen, icon, float64(x), float64(y), col, hasBackground, bgColor)
}

// drawTileWithBgF is the float64 variant for sub-pixel positioning (smooth camera).
func (e *EbitenRenderer) drawTileWithBgF(screen *ebiten.Image, icon string, x, y float64, col color.Color, hasBackground bool, bgColor color.Color) {
	// Draw block background first if requested (so we can draw background-only under the player)
	if hasBackground {
		margin := float32(2)
		bgCol := colorWallBg
		if bgColor != nil {
			r, g, b, a := bgColor.RGBA()
			bgCol = color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
		}
		vector.DrawFilledRect(screen, float32(x)+margin, float32(y)+margin,
			float32(e.tileSize)-margin*2, float32(e.tileSize)-margin*2,
			bgCol, false)
	}

	// Skip drawing character for empty tiles (background-only draw)
	if icon == " " || icon == "" {
		return
	}

	// Convert color to RGBA
	r, g, b, a := col.RGBA()
	tileColor := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}

	// Skip if color is too dark (close to background)
	if tileColor.R < 30 && tileColor.G < 30 && tileColor.B < 30 {
		return
	}

	// Draw the colored character
	e.drawColoredCharF(screen, icon, x, y, tileColor)
}

// drawPlayerWithDebounce draws the player icon with debounce animation if active
func (e *EbitenRenderer) drawPlayerWithDebounce(screen *ebiten.Image, g *state.Game, snap *renderSnapshot, mapX, mapY, visualRow, visualCol float64, startRow, startCol int) {
	e.debounceMutex.RLock()
	direction := e.debounceDirection
	startTime := e.debounceStartTime
	e.debounceMutex.RUnlock()

	playerVRow := visualRow - float64(startRow)
	playerVCol := visualCol - float64(startCol)

	// Skip if player not in viewport (small margin for mid-slide tiles)
	if playerVRow < -0.5 || playerVRow >= float64(e.viewportRows)+0.5 ||
		playerVCol < -0.5 || playerVCol >= float64(e.viewportCols)+0.5 {
		return
	}

	baseX := mapX + playerVCol*float64(e.tileSize)
	baseY := mapY + playerVRow*float64(e.tileSize)

	// Calculate debounce offset
	offsetX := 0
	offsetY := 0
	if direction != "" {
		now := time.Now().UnixMilli()
		elapsed := now - startTime
		const debounceDuration = 150 // milliseconds

		if elapsed < debounceDuration {
			// Calculate bounce offset using a sine wave for smooth animation
			progress := float64(elapsed) / debounceDuration
			bounceAmount := math.Sin(progress*math.Pi) * 8.0 // Max 8 pixels offset

			switch direction {
			case "north":
				offsetY = int(-bounceAmount)
			case "south":
				offsetY = int(bounceAmount)
			case "east":
				offsetX = int(bounceAmount)
			case "west":
				offsetX = int(-bounceAmount)
			}
		} else {
			// Animation complete, clear it
			e.debounceMutex.Lock()
			e.debounceDirection = ""
			e.debounceMutex.Unlock()
		}
	}

	// Draw player icon at offset position (rotate smoothly when facing changes)
	playerX := baseX + float64(offsetX)
	playerY := baseY + float64(offsetY)
	angle := e.playerFacingRot.drawAngle(snap.playerFacing)
	e.drawColoredCharRotatedF(screen, IconPlayerArrow, playerX, playerY, colorPlayer, angle)
}

// drawExitAnimation draws the exit transition animation with a meaningful message
func (e *EbitenRenderer) drawExitAnimation(screen *ebiten.Image, snap *renderSnapshot, mapX, mapY float64, startRow, startCol int) {
	if !snap.exitAnimating {
		return
	}

	now := time.Now().UnixMilli()
	elapsed := now - snap.exitAnimStartTime
	const exitAnimDuration = 2000 // 2 seconds for transition

	if elapsed >= exitAnimDuration {
		return // Animation complete
	}

	// Calculate fade progress (0.0 to 1.0)
	progress := float64(elapsed) / exitAnimDuration

	// Get screen dimensions - use actual screen bounds to ensure full coverage
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	if w == 0 || h == 0 {
		// Fallback to window size if screen bounds are invalid
		w, h = ebiten.WindowSize()
		if w == 0 || h == 0 {
			w, h = e.windowWidth, e.windowHeight
		}
	}

	// Phase 1: Fade to dark background (first 40% of animation)
	// Phase 2: Show message on dark background (middle 40%)
	// Phase 3: Fade message out (last 20%)
	var overlayAlpha float64
	var textAlpha float64
	var showText bool

	if progress < 0.4 {
		// Phase 1: Fade to dark background
		overlayAlpha = progress / 0.4
		textAlpha = 0
		showText = false
	} else if progress < 0.8 {
		// Phase 2: Show message on dark background
		overlayAlpha = 1.0
		textProgress := (progress - 0.4) / 0.4
		textAlpha = textProgress
		if textAlpha > 1.0 {
			textAlpha = 1.0
		}
		showText = true
	} else {
		// Phase 3: Fade message out, keep dark background
		overlayAlpha = 1.0
		fadeProgress := (progress - 0.8) / 0.2
		textAlpha = 1.0 - fadeProgress
		if textAlpha < 0 {
			textAlpha = 0
		}
		showText = textAlpha > 0
	}

	// Draw dark overlay matching the game's aesthetic
	overlayColor := color.RGBA{15, 15, 26, uint8(255 * overlayAlpha)} // Same as colorMapBackground
	vector.DrawFilledRect(screen, 0, 0, float32(w), float32(h), overlayColor, false)

	// Draw transition message
	if showText && textAlpha > 0 {
		// Format string with translation - translate the format string first
		messageFormat := gotext.Get("DECK_CLEARED")
		message := fmt.Sprintf(messageFormat, snap.level)
		subMessage := gotext.Get("PROCEEDING_TO_NEXT_LEVEL")

		// Get font size for UI (use larger size for transition screen)
		fontSize := e.getUIFontSize() * 1.5
		face := e.getSansFontFace()

		// Calculate text position (centered)
		messageWidth, _ := text.Measure(message, face, 0)
		subMessageWidth, _ := text.Measure(subMessage, face, 0)

		centerX := float64(w) / 2
		centerY := float64(h) / 2

		messageX := centerX - float64(messageWidth)/2
		messageY := centerY - fontSize/2

		subMessageX := centerX - float64(subMessageWidth)/2
		subMessageY := centerY + fontSize + 10

		// Draw main message with fade (using action color for emphasis)
		mainTextColor := color.RGBA{220, 170, 255, uint8(255 * textAlpha)} // colorAction
		op := &text.DrawOptions{}
		op.GeoM.Translate(messageX, messageY+fontSize)
		op.ColorScale.ScaleWithColor(mainTextColor)
		text.Draw(screen, message, face, op)

		// Draw sub message with fade (using text color)
		subTextColor := color.RGBA{240, 240, 255, uint8(255 * textAlpha)} // colorText
		op2 := &text.DrawOptions{}
		op2.GeoM.Translate(subMessageX, subMessageY+fontSize)
		op2.ColorScale.ScaleWithColor(subTextColor)
		text.Draw(screen, subMessage, face, op2)
	}
}

// drawCompletionScrim dims the animated tile field so foreground text stays readable.
func (e *EbitenRenderer) drawCompletionScrim(screen *ebiten.Image, w, h int) {
	scrim := color.RGBA{15, 15, 26, 185}
	vector.DrawFilledRect(screen, 0, 0, float32(w), float32(h), scrim, false)
}

func creditsMapTransitionFade(g *state.Game) (active bool, fade float64) {
	if g == nil || g.CreditsTransitionStartMs == 0 {
		return false, 1
	}
	elapsed := time.Now().UnixMilli() - g.CreditsTransitionStartMs
	fade = float64(elapsed) / float64(state.CreditsMapFadeMs)
	if fade >= 1 {
		return false, 1
	}
	if fade < 0 {
		fade = 0
	}
	return true, completionEaseOut(fade)
}

func completionEaseOut(t float64) float64 {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return 1 - math.Pow(1-t, 3)
}

func completionEaseIn(t float64) float64 {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return t * t * t
}

func creditsSlideOffset(screenHeight int, panelH float64, g *state.Game) float64 {
	if g == nil || screenHeight <= 0 {
		return 0
	}
	centerY := float64(screenHeight)/2 - panelH/2
	offBottom := float64(screenHeight) - centerY + panelH
	offTop := centerY + panelH

	now := time.Now().UnixMilli()
	if active, fade := creditsMapTransitionFade(g); active {
		return (1 - fade) * offBottom
	}

	if g.CreditsExitStartMs != 0 {
		elapsed := now - g.CreditsExitStartMs
		t := completionEaseIn(float64(elapsed) / float64(state.CreditsSlideExitMs))
		return -t * offTop
	}

	start := g.CreditsLineStartMs
	if start == 0 {
		return offBottom
	}
	elapsed := now - start
	t := completionEaseOut(float64(elapsed) / float64(state.CreditsSlideEnterMs))
	return (1 - t) * offBottom
}

func completionColorAlpha(col color.RGBA, alpha float64) color.RGBA {
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	return color.RGBA{
		R: col.R,
		G: col.G,
		B: col.B,
		A: uint8(float64(col.A) * alpha),
	}
}

func drawCenteredTextTop(screen *ebiten.Image, line string, face *text.GoTextFace, cx, topY float64, col color.Color) {
	if line == "" || face == nil {
		return
	}
	width, _ := text.Measure(line, face, 0)
	op := &text.DrawOptions{}
	op.GeoM.Translate(cx-width/2, topY)
	op.ColorScale.ScaleWithColor(col)
	text.Draw(screen, line, face, op)
}

func (e *EbitenRenderer) drawCompletionSummary(screen *ebiten.Image, g *state.Game, w, h int, contentAlpha float64) {
	titleSize := e.getUIFontSize() * 1.5
	bodySize := e.getUIFontSize()
	titleFace := e.getSansFontFace()
	bodyFace := e.getSansFontFace()

	line1 := gotext.Get("ENERGY_GRADIENT_EQUALIZED")
	line2 := gotext.Get("NO_FURTHER_WORK_REQUESTS_DETECTED")
	prompt := gotext.Get("PRESS_ANY_KEY_CONTINUE")

	stats := g.RunStatsSnapshot
	statLines := []string{
		fmt.Sprintf(gotext.Get("STAT_DECKS_CLEARED"), stats.DecksCompleted),
		fmt.Sprintf(gotext.Get("STAT_MOVEMENTS"), stats.Movements),
		fmt.Sprintf(gotext.Get("STAT_INTERACTIONS"), stats.Interactions),
		state.FormatRunDuration(stats.ElapsedSeconds),
	}

	mainColor := completionColorAlpha(color.RGBA{220, 170, 255, 255}, contentAlpha)
	subColor := completionColorAlpha(color.RGBA{200, 200, 220, 255}, contentAlpha)
	statColor := completionColorAlpha(color.RGBA{180, 190, 210, 255}, contentAlpha)
	promptColor := completionColorAlpha(color.RGBA{140, 150, 170, 255}, contentAlpha)
	borderColor := completionColorAlpha(color.RGBA{120, 100, 180, 200}, contentAlpha)
	panelBg := completionColorAlpha(color.RGBA{30, 30, 50, 220}, contentAlpha)

	_, line1H := text.Measure(line1, titleFace, 0)
	_, line2H := text.Measure(line2, titleFace, 0)
	_, promptH := text.Measure(prompt, bodyFace, 0)

	titleGap := titleSize * 0.35
	statGap := bodySize * 0.45
	sectionGap := bodySize * 0.55
	promptGap := bodySize * 0.65

	statHeights := make([]float64, len(statLines))
	for i, line := range statLines {
		_, statHeights[i] = text.Measure(line, bodyFace, 0)
	}

	contentHeight := line1H + titleGap + line2H + sectionGap
	for i, sh := range statHeights {
		contentHeight += sh
		if i < len(statHeights)-1 {
			contentHeight += statGap
		}
	}
	contentHeight += promptGap + promptH

	contentWidth := 0.0
	for _, line := range append([]string{line1, line2}, statLines...) {
		if width, _ := text.Measure(line, bodyFace, 0); float64(width) > contentWidth {
			contentWidth = float64(width)
		}
	}
	if w1, _ := text.Measure(line1, titleFace, 0); float64(w1) > contentWidth {
		contentWidth = float64(w1)
	}
	if w2, _ := text.Measure(line2, titleFace, 0); float64(w2) > contentWidth {
		contentWidth = float64(w2)
	}
	if promptWidth, _ := text.Measure(prompt, bodyFace, 0); float64(promptWidth) > contentWidth {
		contentWidth = float64(promptWidth)
	}

	const panelPadX = 48.0
	const panelPadY = 36.0
	panelW := float32(contentWidth + panelPadX*2)
	panelH := float32(contentHeight + panelPadY*2)
	panelX := float32(float64(w)/2 - float64(panelW)/2)
	panelY := float32(float64(h)/2 - float64(panelH)/2)
	drawRoundedRectWithShadow(screen, panelX, panelY, panelW, panelH, 14, 2, panelBg, borderColor, float32(contentAlpha))

	cx := float64(w) / 2
	topY := float64(panelY) + panelPadY

	drawCenteredTextTop(screen, line1, titleFace, cx, topY, mainColor)
	topY += line1H + titleGap
	drawCenteredTextTop(screen, line2, titleFace, cx, topY, subColor)
	topY += line2H + sectionGap
	for i, line := range statLines {
		drawCenteredTextTop(screen, line, bodyFace, cx, topY, statColor)
		topY += statHeights[i]
		if i < len(statLines)-1 {
			topY += statGap
		}
	}
	topY += promptGap
	drawCenteredTextTop(screen, prompt, bodyFace, cx, topY, promptColor)
}

func (e *EbitenRenderer) drawCompletionCredits(screen *ebiten.Image, g *state.Game, w, h int, contentAlpha float64) {
	lines := state.CompletionCreditLineIDs
	if len(lines) == 0 || g == nil {
		return
	}
	lineIndex := g.CreditsLineIndex
	if lineIndex < 0 {
		lineIndex = 0
	}
	if lineIndex >= len(lines) {
		lineIndex = len(lines) - 1
	}

	bodySize := e.getUIFontSize() * 0.95
	titleFace := e.getSansBoldTitleFontFace()
	bodyFace := e.getSansFontFace()

	creditText := completionCreditText(lineIndex)
	prompt := gotext.Get("PRESS_ANY_KEY_SKIP_CREDITS")
	progress := fmt.Sprintf("%d / %d", lineIndex+1, len(lines))

	mainColor := completionColorAlpha(color.RGBA{220, 170, 255, 255}, contentAlpha)
	subColor := completionColorAlpha(color.RGBA{140, 150, 170, 255}, contentAlpha)
	accentColor := completionColorAlpha(color.RGBA{160, 130, 220, 255}, contentAlpha)
	borderColor := completionColorAlpha(color.RGBA{120, 100, 180, 200}, contentAlpha)
	panelBg := completionColorAlpha(color.RGBA{30, 30, 50, 220}, contentAlpha)

	creditWidth, creditH := text.Measure(creditText, titleFace, 0)
	promptWidth, promptH := text.Measure(prompt, bodyFace, 0)
	progressWidth, progressH := text.Measure(progress, bodyFace, 0)
	contentWidth := math.Max(float64(creditWidth), math.Max(float64(promptWidth), float64(progressWidth)))

	const panelPadX = 56.0
	const panelPadY = 42.0
	topRuleGap := bodySize * 0.6
	titleBlockPad := bodySize * 0.5
	ruleAfterTitle := bodySize * 0.4
	promptGap := bodySize * 1.15
	progressGap := bodySize * 0.7

	contentHeight := topRuleGap + creditH + titleBlockPad + ruleAfterTitle + promptH + promptGap + progressH
	panelW := float32(contentWidth + panelPadX*2)
	panelH := float64(contentHeight + panelPadY*2)
	slideOffset := creditsSlideOffset(h, panelH, g)
	panelX := float32(float64(w)/2 - float64(panelW)/2)
	panelY := float32(float64(h)/2 - panelH/2 + slideOffset)
	drawRoundedRectWithShadow(screen, panelX, panelY, panelW, float32(panelH), 14, 2, panelBg, borderColor, float32(contentAlpha))

	cx := float64(w) / 2
	innerTop := float64(panelY) + panelPadY
	topRuleY := innerTop + topRuleGap*0.3
	creditY := innerTop + topRuleGap
	bottomRuleY := creditY + creditH + titleBlockPad
	promptY := bottomRuleY + ruleAfterTitle
	progressY := promptY + promptH + progressGap

	ruleW := float32(contentWidth * 0.55)
	vector.StrokeLine(screen, float32(cx)-ruleW/2, float32(topRuleY), float32(cx)+ruleW/2, float32(topRuleY), 1.5, accentColor, false)
	vector.StrokeLine(screen, float32(cx)-ruleW/2, float32(bottomRuleY), float32(cx)+ruleW/2, float32(bottomRuleY), 1.5, accentColor, false)

	drawCenteredTextTop(screen, creditText, titleFace, cx, creditY, mainColor)
	drawCenteredTextTop(screen, prompt, bodyFace, cx, promptY, subColor)
	drawCenteredTextTop(screen, progress, bodyFace, cx, progressY, subColor)

	// Progress dots beneath the panel (move with the slide).
	dotSpacing := float32(14)
	dotStartX := float32(cx) - float32(len(lines)-1)*dotSpacing/2
	dotY := panelY + float32(panelH) + 18
	for i := range lines {
		dotColor := accentColor
		if i == lineIndex {
			dotColor = mainColor
		} else if i < lineIndex {
			dotColor = completionColorAlpha(color.RGBA{180, 160, 230, 255}, contentAlpha)
		}
		vector.DrawFilledCircle(screen, dotStartX+float32(i)*dotSpacing, dotY, 3, dotColor, false)
	}
}

func completionCreditText(lineIndex int) string {
	switch lineIndex {
	case 0:
		return gotext.Get("CREDITS_GAME_TITLE")
	case 1:
		return gotext.Get("CREDITS_CREATED_BY")
	case 2:
		return gotext.Get("CREDITS_ENGINE")
	case 3:
		return gotext.Get("CREDITS_THANK_YOU")
	default:
		return gotext.Get("CREDITS_THANK_YOU")
	}
}

// drawRoomLabels renders persistent room name labels at the leftmost point of each room
func (e *EbitenRenderer) drawRoomLabels(screen *ebiten.Image, snap *renderSnapshot, mapX, mapY float64, startRow, startCol int) {
	if len(snap.roomLabels) == 0 {
		return
	}

	fontSize := e.getUIFontSize()

	for _, rl := range snap.roomLabels {
		// Check if the label position is visible in the viewport
		labelCol := rl.StartCol
		viewportStartCol := startCol
		viewportEndCol := startCol + e.viewportCols - 1

		// Skip if label is outside viewport
		if labelCol < viewportStartCol || labelCol > viewportEndCol {
			continue
		}

		// Convert to viewport coordinates
		vCol := labelCol - startCol
		vRow := (rl.Row - startRow) - 1

		// Skip if not in vertical range
		if vRow < 0 || vRow >= e.viewportRows {
			continue
		}

		// Compute pixel position (left edge of the cell where label should be)
		cellX := mapX + float64(vCol*e.tileSize)
		cellY := mapY + float64(vRow*e.tileSize)

		var labelMarkup string
		borderColor := colorHazard
		if rl.Powered {
			labelMarkup = fmt.Sprintf("POWERED{%s}", rl.RoomName)
			borderColor = colorGeneratorOn
		} else {
			labelMarkup = fmt.Sprintf("UNPOWERED{%s}", rl.RoomName)
		}
		textWidth := e.getMarkupWidth(labelMarkup)

		// Draw background box for readability
		paddingX := 6
		paddingY := 4
		boxW := int(textWidth) + paddingX*2
		boxH := int(fontSize) + paddingY*2

		// Position box starting at the leftmost point of the room cell
		// Raise it by half its height so it sits just above the wall
		boxX := cellX + 2 // Small offset from left edge of cell
		boxY := cellY - float64(boxH) - 4 - float64(boxH)/2

		bgColor := color.RGBA{15, 20, 40, 235}

		const labelCornerRadius = 4
		const labelBorderWidth = 1
		drawRoundedRectWithShadow(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), labelCornerRadius, labelBorderWidth, bgColor, borderColor, 1.0)

		// Position text: drawColoredText uses baseline positioning (adds fontSize to y)
		// Similar to callouts: subtract fontSize so baseline ends up inside the box
		textX := int(boxX) + paddingX
		textY := int(boxY) + paddingY - int(fontSize)

		segments := e.parseMarkup(labelMarkup)
		// Draw bold-ish by rendering twice with slight offset
		e.drawColoredTextSegments(screen, segments, textX, textY)
		e.drawColoredTextSegments(screen, segments, textX+1, textY)
	}
}

// drawEnvironmentalPlaques renders small diegetic corridor signage inside tiles (Story 5.1).
func (e *EbitenRenderer) drawEnvironmentalPlaques(screen *ebiten.Image, snap *renderSnapshot, mapX, mapY float64, startRow, startCol int) {
	if !e.EnvPlaquesEnabled() || len(snap.envPlaques) == 0 {
		return
	}

	const maxRunes = 28
	const baselinePadding = 5

	face := e.getMonoFontFace()

	for _, ep := range snap.envPlaques {
		labelRow := ep.Row
		labelCol := ep.Col
		if labelCol < startCol || labelCol > startCol+e.viewportCols-1 {
			continue
		}
		vCol := labelCol - startCol
		vRow := labelRow - startRow
		if vRow < 0 || vRow >= e.viewportRows {
			continue
		}

		cellX := mapX + float64(vCol*e.tileSize)
		cellY := mapY + float64(vRow*e.tileSize)

		txt := dynamicGet(ep.MsgID)
		rs := []rune(txt)
		if len(rs) > maxRunes {
			txt = string(rs[:maxRunes-1]) + "…"
		}

		w, _ := text.Measure(txt, face, 0)
		scale := 0.38
		maxW := float64(e.tileSize) - 6
		if w*scale > maxW && w > 0 {
			scale = maxW / w
			if scale < 0.26 {
				scale = 0.26
			}
		}

		px := cellX + 3
		baselineY := cellY + float64(e.tileSize) - baselinePadding

		op := &text.DrawOptions{}
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(px/scale, baselineY/scale)
		op.ColorScale.ScaleWithColor(colorPlaque)
		text.Draw(screen, txt, face, op)
	}
}

// drawDirectionLabels draws N/S/E/W labels around the map
func (e *EbitenRenderer) drawDirectionLabels(screen *ebiten.Image, g *state.Game, mapX, mapY, mapWidth, mapHeight int) {
	if g.CurrentCell == nil {
		return
	}

	fontSize := e.getUIFontSize()

	// North label (above map, centered)
	northText := e.getDirectionText(g, g.CurrentCell.North, "NORTH")
	northWidth := e.getTextWidth(northText)
	northX := mapX + mapWidth/2 - int(northWidth)/2
	e.drawColoredText(screen, northText, northX, mapY-int(fontSize)-15, colorText)

	// South label (below map, centered)
	southText := e.getDirectionText(g, g.CurrentCell.South, "SOUTH")
	southWidth := e.getTextWidth(southText)
	southX := mapX + mapWidth/2 - int(southWidth)/2
	e.drawColoredText(screen, southText, southX, mapY+mapHeight+10, colorText)

	// West label (left of map, vertically centered)
	westText := e.getDirectionText(g, g.CurrentCell.West, "WEST")
	westWidth := e.getTextWidth(westText)
	e.drawColoredText(screen, westText, mapX-int(westWidth)-20, mapY+mapHeight/2-int(fontSize)/2, colorText)

	// East label (right of map, vertically centered)
	eastText := e.getDirectionText(g, g.CurrentCell.East, "EAST")
	e.drawColoredText(screen, eastText, mapX+mapWidth+20, mapY+mapHeight/2-int(fontSize)/2, colorText)
}

// getDirectionText returns the text for a direction label
// direction should be a translation key (e.g., "NORTH") which will be translated in drawColoredText
func (e *EbitenRenderer) getDirectionText(g *state.Game, cell *world.Cell, direction string) string {
	if cell == nil || !cell.Room {
		return "WALL"
	}

	// Check if blocked
	if gameworld.HasLockedDoor(cell) {
		data := gameworld.GetGameData(cell)
		// Translate direction first, then format
		translatedDir := dynamicGet(direction)
		return fmt.Sprintf(gotext.Get("DIRECTION_NEED_KEYCARD"), translatedDir, data.Door.KeycardName())
	}

	if gameworld.HasBlockingHazard(cell) {
		// Translate direction first, then format
		translatedDir := dynamicGet(direction)
		return fmt.Sprintf(gotext.Get("DIRECTION_BLOCKED"), translatedDir)
	}

	return direction
}

// drawStatusBarFromSnapshot draws deck/objectives plus inventory and generator lines using snapshot data.
// Caller supplies anchor x,y so panel/outlining aligns with layout (window top-left in gameplay).
func (e *EbitenRenderer) drawStatusBarFromSnapshot(screen *ebiten.Image, snap *renderSnapshot, x, y, width, height int) {
	// Check if there's anything to show
	hasObjectives := len(snap.objectives) > 0
	hasInventory := len(snap.ownedItems) > 0 || snap.batteries > 0
	hasGenerators := len(snap.generators) > 0

	// Always show at least the deck number
	hasDeckNumber := true

	// Don't draw anything if everything is empty (but we always have deck number)
	if !hasDeckNumber && !hasObjectives && !hasInventory && !hasGenerators {
		return
	}

	fontSize := e.getUIFontSize()
	regularFace := e.getSansFontFace()
	titleFace := e.getSansBoldTitleFontFace()
	_, bodyLineHeight := text.Measure("Ag", regularFace, 0)
	lineHeight := int(bodyLineHeight) + 4 // measured height + spacing

	// Calculate how many lines we need (always include deck number)
	linesNeeded := 0
	if hasDeckNumber {
		linesNeeded++ // Deck number is always first
	}
	if hasObjectives {
		linesNeeded += len(snap.objectives)
	}
	if hasInventory {
		linesNeeded++
	}
	if hasGenerators {
		linesNeeded++
	}

	// Gaps between sections are added to currentY when drawing but must be included in height
	const sectionGap = 2
	gaps := 0
	if hasDeckNumber && hasObjectives {
		gaps += sectionGap
	}
	if hasObjectives && (hasInventory || hasGenerators) {
		gaps += sectionGap
	}

	// Calculate the maximum width needed for all text lines
	maxTextWidth := 0.0
	// Deck number text - uses title face (larger), measure with that
	deckTextFormat := gotext.Get("DECK_NUMBER")
	deckText := fmt.Sprintf(deckTextFormat, snap.level)
	deckWidth := e.getTextWidthWithFace(deckText, titleFace)
	if deckWidth > maxTextWidth {
		maxTextWidth = deckWidth
	}
	_, firstLineMeasured := text.Measure(deckText, titleFace, 0)
	firstLineHeight := int(firstLineMeasured) + 4 // measured height + spacing
	if hasObjectives {
		for _, objective := range snap.objectives {
			// Translate objective if it's a translation key for width calculation
			translatedObjective := dynamicGet(objective)
			// Parse markup to get actual text width (not markup)
			segments := e.parseMarkup(translatedObjective)
			textWidth := 0.0
			for _, seg := range segments {
				textWidth += e.getTextWidth(seg.text)
			}
			if textWidth > maxTextWidth {
				maxTextWidth = textWidth
			}
		}
	}
	if hasInventory {
		// Build inventory text with markup for width calculation (same format as rendering)
		invLabel := gotext.Get("INVENTORY")
		invParts := []string{invLabel}
		for i, itemName := range snap.ownedItems {
			if i > 0 {
				invParts = append(invParts, ",")
			}
			invParts = append(invParts, fmt.Sprintf("ITEM{%s}", itemName))
		}
		if snap.batteries > 0 {
			if len(snap.ownedItems) > 0 {
				invParts = append(invParts, ",")
			}
			invParts = append(invParts, fmt.Sprintf("ACTION{Batteries x%d}", snap.batteries))
		}
		invText := strings.Join(invParts, " ")
		// Calculate width using parsed segments (actual text width, not markup)
		segments := e.parseMarkup(invText)
		textWidth := 0.0
		for _, seg := range segments {
			textWidth += e.getTextWidth(seg.text)
		}
		if textWidth > maxTextWidth {
			maxTextWidth = textWidth
		}
	}
	if hasGenerators {
		genLabel := gotext.Get("GENERATORS")
		genText := genLabel + " "
		genParts := []string{}
		for i, gen := range snap.generators {
			if gen.powered {
				genParts = append(genParts, fmt.Sprintf("#%d POWERED{ONLINE}", i+1))
			} else {
				genParts = append(genParts, fmt.Sprintf("#%d %d/%d", i+1, gen.batteriesInserted, gen.batteriesRequired))
			}
		}
		genText += strings.Join(genParts, ", ")
		w := e.getTextWidth(genText)
		if w > maxTextWidth {
			maxTextWidth = w
		}
	}

	// Adjust panel height based on actual content (measured heights + gaps)
	contentHeight := gaps
	if hasDeckNumber {
		contentHeight += firstLineHeight + (linesNeeded-1)*lineHeight
	} else {
		contentHeight += lineHeight * linesNeeded
	}
	panelHeight := contentHeight + 10
	if panelHeight < int(bodyLineHeight)+10 {
		panelHeight = int(bodyLineHeight) + 10
	}

	// Calculate panel width based on widest text, with padding
	panelWidth := int(maxTextWidth) + 20 // 10px padding on each side
	if panelWidth < 100 {
		panelWidth = 100 // Minimum width
	}

	// Draw panel background with rounded rect and drop shadow (more opaque for overlay on map)
	// Border matches title color (Deck X uses colorAction)
	bgX := float32(x - 10)
	bgY := float32(y - 5)
	bgW := float32(panelWidth)
	bgH := float32(panelHeight)
	borderColor := colorAction
	// More opaque background for overlay readability
	overlayBackground := color.RGBA{20, 20, 35, 250} // More opaque than colorPanelBackground

	const objectivesCornerRadius = 8
	const objectivesBorderWidth = 2
	drawRoundedRectWithShadow(screen, bgX, bgY, bgW, bgH, objectivesCornerRadius, objectivesBorderWidth, overlayBackground, borderColor, 1.0)

	// Calculate vertical center (contentHeight already includes gaps)
	deckFontSize := fontSize + 2
	firstLineY := y + (panelHeight-contentHeight)/2 - int(deckFontSize)

	currentY := firstLineY

	// Deck number (always first line, uses title font)
	if hasDeckNumber {
		deckTextFormat := gotext.Get("DECK_NUMBER")
		deckText := fmt.Sprintf(deckTextFormat, snap.level)
		e.drawColoredTextWithFace(screen, deckText, x, currentY, colorAction, e.getSansBoldTitleFontFace())
		currentY += firstLineHeight
		// Add a small gap between deck number and objectives
		if hasObjectives {
			currentY += 2
		}
	}

	// Objectives (displayed after deck number)
	if hasObjectives {
		for _, objective := range snap.objectives {
			// Translate objective if it's a translation key, then parse markup
			translatedObjective := dynamicGet(objective)
			// Parse markup to properly color ACTION{} segments
			segments := e.parseMarkup(translatedObjective)
			e.drawColoredTextSegments(screen, segments, x, currentY)
			currentY += lineHeight
		}
		// Add a small gap between objectives and inventory
		if hasInventory || hasGenerators {
			currentY += 2
		}
	}

	// Inventory line (only if not empty)
	if hasInventory {
		// Build inventory text with item colors using markup, commas in default color
		invLabel := gotext.Get("INVENTORY")
		invParts := []string{invLabel}
		for i, itemName := range snap.ownedItems {
			if i > 0 {
				invParts = append(invParts, ",") // Comma in default text color
			}
			invParts = append(invParts, fmt.Sprintf("ITEM{%s}", itemName))
		}
		if snap.batteries > 0 {
			if len(snap.ownedItems) > 0 {
				invParts = append(invParts, ",") // Comma in default text color
			}
			invParts = append(invParts, fmt.Sprintf("ACTION{Batteries x%d}", snap.batteries))
		}
		invText := strings.Join(invParts, " ")

		// Parse markup to apply item colors (commas will be in default color)
		segments := e.parseMarkup(invText)
		e.drawColoredTextSegments(screen, segments, x, currentY)
		currentY += lineHeight
	}

	// Generator status (if applicable)
	if hasGenerators {
		genLabel := gotext.Get("GENERATORS")
		genText := genLabel + ": "
		genParts := []string{}
		for i, gen := range snap.generators {
			if gen.powered {
				genParts = append(genParts, fmt.Sprintf("#%d POWERED{ONLINE}", i+1))
			} else {
				genParts = append(genParts, fmt.Sprintf("#%d %d/%d", i+1, gen.batteriesInserted, gen.batteriesRequired))
			}
		}
		genText += strings.Join(genParts, ", ")
		e.drawColoredTextSegments(screen, e.parseMarkup(genText), x, currentY)
	}
}

// drawLongUseProgress renders a hold-to-use progress bar over the target cell.
func (e *EbitenRenderer) drawLongUseProgress(screen *ebiten.Image, snap *renderSnapshot, mapScrX, mapScrY float64, startRow, startCol int) {
	if !snap.longUseActive {
		return
	}
	vRow := snap.longUseTargetRow - startRow
	vCol := snap.longUseTargetCol - startCol
	if vRow < 0 || vCol < 0 || vRow >= e.viewportRows || vCol >= e.viewportCols {
		return
	}
	x := int(mapScrX) + vCol*e.tileSize
	y := int(mapScrY) + vRow*e.tileSize
	margin := 3
	barW := e.tileSize - margin*2
	barH := e.tileSize / 6
	if barH < 4 {
		barH = 4
	}
	barY := y + e.tileSize - barH - margin

	bg := color.RGBA{30, 30, 40, 200}
	fill := color.RGBA{0, 220, 120, 255}
	border := color.RGBA{180, 180, 200, 255}

	vector.FillRect(screen, float32(x+margin), float32(barY), float32(barW), float32(barH), bg, false)
	fillW := int(float32(barW) * float32(snap.longUseProgress))
	if fillW > 0 {
		vector.FillRect(screen, float32(x+margin), float32(barY), float32(fillW), float32(barH), fill, false)
	}
	vector.StrokeRect(screen, float32(x+margin), float32(barY), float32(barW), float32(barH), 1, border, false)
}
