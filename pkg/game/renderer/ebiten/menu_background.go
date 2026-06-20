// Package ebiten provides background animation for the main menu.
package ebiten

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	floatingTileAreaPerTile     = 19661 // ~40 tiles at 1024×768
	floatingTileMinCount        = 20
	floatingTileMaxCount        = 120
	floatingTileMovementSpeed   = 1.3
	floatingTileMaxSpeed        = 1.0
	floatingTileCollisionRadius = 14.0
	floatingTileSpawnSeparation = floatingTileCollisionRadius * 2
	floatingTileRelaxIterations = 12
	floatingTileSpawnAttempts   = 48
)

var (
	floatingTileIcons = []string{
		IconGeneratorUnpowered, IconGeneratorPowered,
		IconDoorLocked, IconDoorUnlocked,
		IconTerminalUnused, IconTerminalUsed,
		IconBattery, IconKey,
		IconExitLocked, IconExitUnlocked,
		"◎", "◉", "▫", "▪", "◇", "◆",
	}
	floatingTileColors = []color.Color{
		color.RGBA{40, 40, 60, 255},
		color.RGBA{60, 40, 40, 255},
		color.RGBA{40, 60, 40, 255},
		color.RGBA{50, 50, 70, 255},
		color.RGBA{45, 45, 65, 255},
		color.RGBA{55, 45, 55, 255},
		color.RGBA{35, 50, 55, 255},
		color.RGBA{50, 40, 50, 255},
	}
)

// floatingTileTargetCount returns how many ambient tiles should cover a screen.
func floatingTileTargetCount(screenWidth, screenHeight int) int {
	if screenWidth <= 0 || screenHeight <= 0 {
		return floatingTileMinCount
	}
	n := (screenWidth * screenHeight) / floatingTileAreaPerTile
	if n < floatingTileMinCount {
		return floatingTileMinCount
	}
	if n > floatingTileMaxCount {
		return floatingTileMaxCount
	}
	return n
}

func newRandomFloatingTileAt(x, y float64) floatingTile {
	return floatingTile{
		x:             x,
		y:             y,
		vx:            (rand.Float64() - 0.5) * floatingTileMovementSpeed,
		vy:            (rand.Float64() - 0.5) * floatingTileMovementSpeed,
		icon:          floatingTileIcons[rand.Intn(len(floatingTileIcons))],
		color:         floatingTileColors[rand.Intn(len(floatingTileColors))],
		alpha:         2.5,
		rotation:      rand.Float64() * 2 * math.Pi,
		rotationSpeed: (rand.Float64() - 0.5) * 0.026,
	}
}

func clampFloatingTileSpeed(tile *floatingTile) {
	if tile.vx > floatingTileMaxSpeed {
		tile.vx = floatingTileMaxSpeed
	} else if tile.vx < -floatingTileMaxSpeed {
		tile.vx = -floatingTileMaxSpeed
	}
	if tile.vy > floatingTileMaxSpeed {
		tile.vy = floatingTileMaxSpeed
	} else if tile.vy < -floatingTileMaxSpeed {
		tile.vy = -floatingTileMaxSpeed
	}
}

// floatingTileScreenSize returns the logical screen size used for tile density and bounds.
func (e *EbitenRenderer) floatingTileScreenSize() (int, int) {
	if e.windowWidth > 0 && e.windowHeight > 0 {
		return e.windowWidth, e.windowHeight
	}
	w, h := ebiten.WindowSize()
	if w > 0 && h > 0 {
		return w, h
	}
	return 1024, 768
}

func gridFloatingTilePositions(count, screenW, screenH int) [][2]float64 {
	if count <= 0 || screenW <= 0 || screenH <= 0 {
		return nil
	}
	aspect := float64(screenW) / float64(screenH)
	cols := int(math.Ceil(math.Sqrt(float64(count) * aspect)))
	if cols < 1 {
		cols = 1
	}
	rows := (count + cols - 1) / cols
	margin := floatingTileCollisionRadius
	usableW := float64(screenW) - 2*margin
	usableH := float64(screenH) - 2*margin
	if usableW <= 0 || usableH <= 0 {
		return nil
	}
	cellW := usableW / float64(cols)
	cellH := usableH / float64(rows)
	out := make([][2]float64, 0, count)
	for r := 0; r < rows && len(out) < count; r++ {
		for c := 0; c < cols && len(out) < count; c++ {
			cx := margin + cellW*(float64(c)+0.5)
			cy := margin + cellH*(float64(r)+0.5)
			jx := (rand.Float64() - 0.5) * cellW * 0.35
			jy := (rand.Float64() - 0.5) * cellH * 0.35
			out = append(out, [2]float64{cx + jx, cy + jy})
		}
	}
	return out
}

func floatingTileTooClose(x, y float64, existing []floatingTile) bool {
	minDistSq := floatingTileSpawnSeparation * floatingTileSpawnSeparation
	for i := range existing {
		dx := x - existing[i].x
		dy := y - existing[i].y
		if dx*dx+dy*dy < minDistSq {
			return true
		}
	}
	return false
}

// randomFloatingTilePos picks a spawn point for a new tile. After a resize it
// biases toward strips that were not visible at the previous size.
func randomFloatingTilePos(oldW, oldH, screenW, screenH int) (x, y float64) {
	if oldW <= 0 || oldH <= 0 || (screenW <= oldW && screenH <= oldH) {
		return rand.Float64() * float64(screenW), rand.Float64() * float64(screenH)
	}

	rightArea := 0
	if screenW > oldW {
		rightArea = (screenW - oldW) * screenH
	}
	bottomArea := 0
	if screenH > oldH {
		bottomArea = screenW * (screenH - oldH)
	}
	// Overlap corner is counted in both strips; either strip is fine for spawn.
	total := rightArea + bottomArea
	if total <= 0 {
		return rand.Float64() * float64(screenW), rand.Float64() * float64(screenH)
	}

	pick := rand.Intn(total)
	if pick < rightArea {
		x = float64(oldW) + rand.Float64()*float64(screenW-oldW)
		y = rand.Float64() * float64(screenH)
		return x, y
	}
	x = rand.Float64() * float64(screenW)
	y = float64(oldH) + rand.Float64()*float64(screenH-oldH)
	return x, y
}

// pickFloatingTileSpawnPos finds a spawn point with spacing from existing tiles.
func pickFloatingTileSpawnPos(existing []floatingTile, oldW, oldH, screenW, screenH int) (x, y float64) {
	for attempt := 0; attempt < floatingTileSpawnAttempts; attempt++ {
		x, y = randomFloatingTilePos(oldW, oldH, screenW, screenH)
		if !floatingTileTooClose(x, y, existing) {
			return x, y
		}
	}
	return x, y
}

func relaxFloatingTiles(tiles []floatingTile, screenW, screenH int) {
	for iter := 0; iter < floatingTileRelaxIterations; iter++ {
		resolveFloatingTileCollisions(tiles)
		for i := range tiles {
			bounceFloatingTileOffWalls(&tiles[i], screenW, screenH)
			clampFloatingTileSpeed(&tiles[i])
		}
	}
}

// syncFloatingTilesToScreenUnlocked grows or shrinks the tile field to match the
// current window. Caller must hold floatingTilesMutex.
func (e *EbitenRenderer) syncFloatingTilesToScreenUnlocked(screenWidth, screenHeight int) {
	if screenWidth <= 0 || screenHeight <= 0 {
		return
	}
	if len(floatingTileIcons) == 0 || len(floatingTileColors) == 0 {
		return
	}

	oldW, oldH := e.floatingTilesScreenW, e.floatingTilesScreenH
	target := floatingTileTargetCount(screenWidth, screenHeight)
	if e.floatingTilesScreenW == screenWidth &&
		e.floatingTilesScreenH == screenHeight &&
		len(e.floatingTiles) == target {
		return
	}

	initialFill := len(e.floatingTiles) == 0
	grew := len(e.floatingTiles) < target

	if initialFill {
		for _, pos := range gridFloatingTilePositions(target, screenWidth, screenHeight) {
			e.floatingTiles = append(e.floatingTiles, newRandomFloatingTileAt(pos[0], pos[1]))
		}
	} else {
		for len(e.floatingTiles) < target {
			x, y := pickFloatingTileSpawnPos(e.floatingTiles, oldW, oldH, screenWidth, screenHeight)
			e.floatingTiles = append(e.floatingTiles, newRandomFloatingTileAt(x, y))
		}
	}
	if len(e.floatingTiles) > target {
		e.floatingTiles = e.floatingTiles[:target]
	}

	if initialFill || grew {
		relaxFloatingTiles(e.floatingTiles, screenWidth, screenHeight)
	}

	e.floatingTilesScreenW = screenWidth
	e.floatingTilesScreenH = screenHeight
}

// initFloatingTiles initializes the floating tiles animation for the main menu background.
// This version assumes the mutex is already locked by the caller.
func (e *EbitenRenderer) initFloatingTilesUnlocked(screenWidth, screenHeight int) {
	e.floatingTiles = nil
	e.floatingTilesScreenW = 0
	e.floatingTilesScreenH = 0
	e.syncFloatingTilesToScreenUnlocked(screenWidth, screenHeight)
}

// initFloatingTiles initializes the floating tiles animation for the main menu background.
// This version locks the mutex itself (for use when called without a lock).
func (e *EbitenRenderer) initFloatingTiles(screenWidth, screenHeight int) {
	e.floatingTilesMutex.Lock()
	defer e.floatingTilesMutex.Unlock()
	e.initFloatingTilesUnlocked(screenWidth, screenHeight)
}

// ensureFloatingTiles creates or expands the ambient tile field when needed.
func (e *EbitenRenderer) ensureFloatingTiles(screenWidth, screenHeight int) {
	if screenWidth <= 0 || screenHeight <= 0 {
		screenWidth, screenHeight = e.floatingTileScreenSize()
	}
	e.floatingTilesMutex.Lock()
	defer e.floatingTilesMutex.Unlock()
	e.syncFloatingTilesToScreenUnlocked(screenWidth, screenHeight)
}

func bounceFloatingTileOffWalls(tile *floatingTile, screenWidth, screenHeight int) {
	r := floatingTileCollisionRadius
	maxX := float64(screenWidth) - r
	maxY := float64(screenHeight) - r
	if tile.x < r {
		tile.x = r
		if tile.vx < 0 {
			tile.vx = -tile.vx
		}
	} else if tile.x > maxX {
		tile.x = maxX
		if tile.vx > 0 {
			tile.vx = -tile.vx
		}
	}
	if tile.y < r {
		tile.y = r
		if tile.vy < 0 {
			tile.vy = -tile.vy
		}
	} else if tile.y > maxY {
		tile.y = maxY
		if tile.vy > 0 {
			tile.vy = -tile.vy
		}
	}
}

func resolveFloatingTileCollisions(tiles []floatingTile) {
	minDist := floatingTileCollisionRadius * 2
	for i := range tiles {
		for j := i + 1; j < len(tiles); j++ {
			a := &tiles[i]
			b := &tiles[j]
			dx := b.x - a.x
			dy := b.y - a.y
			distSq := dx*dx + dy*dy
			if distSq >= minDist*minDist || distSq < 1e-6 {
				continue
			}
			dist := math.Sqrt(distSq)
			nx := dx / dist
			ny := dy / dist
			overlap := (minDist - dist) * 0.5
			a.x -= nx * overlap
			a.y -= ny * overlap
			b.x += nx * overlap
			b.y += ny * overlap

			dvx := a.vx - b.vx
			dvy := a.vy - b.vy
			dot := dvx*nx + dvy*ny
			if dot >= 0 {
				continue
			}
			a.vx -= dot * nx
			a.vy -= dot * ny
			b.vx += dot * nx
			b.vy += dot * ny
			clampFloatingTileSpeed(a)
			clampFloatingTileSpeed(b)
		}
	}
}

// updateFloatingTiles updates the positions of floating tiles each frame.
func (e *EbitenRenderer) updateFloatingTiles(screenWidth, screenHeight int) {
	e.floatingTilesMutex.Lock()
	defer e.floatingTilesMutex.Unlock()

	if screenWidth > 0 && screenHeight > 0 &&
		(screenWidth != e.floatingTilesScreenW || screenHeight != e.floatingTilesScreenH) {
		e.syncFloatingTilesToScreenUnlocked(screenWidth, screenHeight)
	}

	for i := range e.floatingTiles {
		tile := &e.floatingTiles[i]

		tile.x += tile.vx
		tile.y += tile.vy

		bounceFloatingTileOffWalls(tile, screenWidth, screenHeight)

		tile.rotation += tile.rotationSpeed
		if tile.rotation > 2*math.Pi {
			tile.rotation -= 2 * math.Pi
		} else if tile.rotation < 0 {
			tile.rotation += 2 * math.Pi
		}

		if rand.Float64() < 0.01 {
			tile.vx += (rand.Float64() - 0.5) * 0.13
			tile.vy += (rand.Float64() - 0.5) * 0.13
			clampFloatingTileSpeed(tile)
		}
	}

	resolveFloatingTileCollisions(e.floatingTiles)
	for i := range e.floatingTiles {
		clampFloatingTileSpeed(&e.floatingTiles[i])
	}
}

// titleScreenFloatingTilesMenu reports menus that keep the title-screen ambient field.
func titleScreenFloatingTilesMenu(title string) bool {
	switch title {
	case "The Dark Station", "Settings":
		return true
	default:
		return false
	}
}

// onTitleScreen reports whether the renderer is in pre-game menu context (no live map).
func (e *EbitenRenderer) onTitleScreen() bool {
	e.snapshotMutex.RLock()
	snapValid := e.snapshot.valid
	e.snapshotMutex.RUnlock()
	e.gameMutex.RLock()
	gameNil := e.game == nil
	e.gameMutex.RUnlock()
	return !snapValid || gameNil
}

// titleScreenFloatingTilesActive reports whether the drifting tile field should run on a title-screen menu.
func (e *EbitenRenderer) titleScreenFloatingTilesActive() bool {
	e.genericMenuMutex.RLock()
	active := e.genericMenuActive && titleScreenFloatingTilesMenu(e.genericMenuTitle)
	e.genericMenuMutex.RUnlock()
	return active && e.onTitleScreen()
}

// floatingTilesAnimationActive reports whether the ambient tile field should advance this frame.
func (e *EbitenRenderer) floatingTilesAnimationActive() bool {
	if e.titleScreenFloatingTilesActive() {
		return true
	}
	e.gameMutex.RLock()
	g := e.game
	e.gameMutex.RUnlock()
	return g != nil && g.GameComplete
}

// drawFloatingTilesBackground draws the floating tiles animation behind the menu.
func (e *EbitenRenderer) drawFloatingTilesBackground(screen *ebiten.Image) {
	e.floatingTilesMutex.RLock()
	tiles := make([]floatingTile, len(e.floatingTiles))
	copy(tiles, e.floatingTiles)
	e.floatingTilesMutex.RUnlock()

	if len(tiles) == 0 {
		return
	}

	face := e.getMonoFontFace()
	if face == nil {
		return
	}

	for _, tile := range tiles {
		r, g, b, a := tile.color.RGBA()
		alpha := float64(tile.alpha)
		tileColor := color.RGBA{
			uint8(r >> 8),
			uint8(g >> 8),
			uint8(b >> 8),
			uint8(float64(a>>8) * alpha),
		}

		textWidth, textHeight := text.Measure(tile.icon, face, 0)
		if textWidth <= 0 || textHeight <= 0 {
			continue
		}

		op := &text.DrawOptions{}
		op.GeoM.Translate(tile.x-textWidth/2, tile.y-textHeight/2)

		op.ColorScale.Reset()
		alpha32 := float32(alpha)
		op.ColorScale.Scale(alpha32, alpha32, alpha32, alpha32)
		op.ColorScale.ScaleWithColor(tileColor)

		text.Draw(screen, tile.icon, face, op)
	}
}
