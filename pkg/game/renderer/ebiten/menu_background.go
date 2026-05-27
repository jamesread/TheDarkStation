// Package ebiten provides background animation for the main menu.
package ebiten

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// initFloatingTiles initializes the floating tiles animation for the main menu background.
// This version assumes the mutex is already locked by the caller.
func (e *EbitenRenderer) initFloatingTilesUnlocked(screenWidth, screenHeight int) {
	// Defensive checks
	if screenWidth <= 0 || screenHeight <= 0 {
		return
	}

	// Create 30-50 floating tiles
	numTiles := 30 + rand.Intn(21)
	e.floatingTiles = make([]floatingTile, numTiles)

	// Available icons for variety
	icons := []string{
		IconGeneratorUnpowered, IconGeneratorPowered,
		IconDoorLocked, IconDoorUnlocked,
		IconTerminalUnused, IconTerminalUsed,
		IconBattery, IconKey,
		IconExitLocked, IconExitUnlocked,
		"◎", "◉", "▫", "▪", "◇", "◆",
	}
	// Dark contrasting colors for background animation
	colors := []color.Color{
		color.RGBA{40, 40, 60, 255}, // Dark blue-gray
		color.RGBA{60, 40, 40, 255}, // Dark red-gray
		color.RGBA{40, 60, 40, 255}, // Dark green-gray
		color.RGBA{50, 50, 70, 255}, // Darker blue-gray
		color.RGBA{45, 45, 65, 255}, // Medium dark gray-blue
		color.RGBA{55, 45, 55, 255}, // Dark purple-gray
		color.RGBA{35, 50, 55, 255}, // Dark teal-gray
		color.RGBA{50, 40, 50, 255}, // Dark magenta-gray
	}

	// Ensure we have icons and colors
	if len(icons) == 0 || len(colors) == 0 {
		return
	}

	const tileMovementSpeed = 1.3

	for i := range e.floatingTiles {
		// Random starting position
		e.floatingTiles[i].x = rand.Float64() * float64(screenWidth)
		e.floatingTiles[i].y = rand.Float64() * float64(screenHeight)

		// Random velocity (slow drift) - increased by 30%
		e.floatingTiles[i].vx = (rand.Float64() - 0.5) * tileMovementSpeed
		e.floatingTiles[i].vy = (rand.Float64() - 0.5) * tileMovementSpeed

		// Random icon and color
		e.floatingTiles[i].icon = icons[rand.Intn(len(icons))]
		e.floatingTiles[i].color = colors[rand.Intn(len(colors))]

		// Random alpha (semi-transparent for depth)
		e.floatingTiles[i].alpha = 2.5

		// Random rotation
		e.floatingTiles[i].rotation = rand.Float64() * 2 * math.Pi
		e.floatingTiles[i].rotationSpeed = (rand.Float64() - 0.5) * 0.026 // Slow rotation (30% faster)
	}
}

// initFloatingTiles initializes the floating tiles animation for the main menu background.
// This version locks the mutex itself (for use when called without a lock).
func (e *EbitenRenderer) initFloatingTiles(screenWidth, screenHeight int) {
	e.floatingTilesMutex.Lock()
	defer e.floatingTilesMutex.Unlock()
	e.initFloatingTilesUnlocked(screenWidth, screenHeight)
}

// ensureFloatingTiles creates the ambient tile field when needed (main menu or completion screens).
func (e *EbitenRenderer) ensureFloatingTiles(screenWidth, screenHeight int) {
	if screenWidth <= 0 || screenHeight <= 0 {
		return
	}
	e.floatingTilesMutex.Lock()
	defer e.floatingTilesMutex.Unlock()
	if len(e.floatingTiles) == 0 {
		e.initFloatingTilesUnlocked(screenWidth, screenHeight)
	}
}

const floatingTileCollisionRadius = 14.0

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
		}
	}
}

// updateFloatingTiles updates the positions of floating tiles each frame.
func (e *EbitenRenderer) updateFloatingTiles(screenWidth, screenHeight int) {
	e.floatingTilesMutex.Lock()
	defer e.floatingTilesMutex.Unlock()

	for i := range e.floatingTiles {
		tile := &e.floatingTiles[i]

		// Update position
		tile.x += tile.vx
		tile.y += tile.vy

		bounceFloatingTileOffWalls(tile, screenWidth, screenHeight)

		// Update rotation
		tile.rotation += tile.rotationSpeed
		if tile.rotation > 2*math.Pi {
			tile.rotation -= 2 * math.Pi
		} else if tile.rotation < 0 {
			tile.rotation += 2 * math.Pi
		}

		// Add slight random drift changes for more organic movement (30% faster)
		if rand.Float64() < 0.01 { // 1% chance per frame
			tile.vx += (rand.Float64() - 0.5) * 0.13 // 30% faster
			tile.vy += (rand.Float64() - 0.5) * 0.13
			// Clamp velocity
			if tile.vx > 1.0 {
				tile.vx = 1.0
			} else if tile.vx < -1.0 {
				tile.vx = -1.0
			}
			if tile.vy > 1.0 {
				tile.vy = 1.0
			} else if tile.vy < -1.0 {
				tile.vy = -1.0
			}
		}
	}

	resolveFloatingTileCollisions(e.floatingTiles)
}

// floatingTilesAnimationActive reports whether the ambient tile field should advance this frame.
func (e *EbitenRenderer) floatingTilesAnimationActive() bool {
	e.genericMenuMutex.RLock()
	menuActive := e.genericMenuActive && e.genericMenuTitle == "The Dark Station"
	e.genericMenuMutex.RUnlock()
	if menuActive {
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
		// Apply alpha to color
		r, g, b, a := tile.color.RGBA()
		alpha := float64(tile.alpha)
		tileColor := color.RGBA{
			uint8(r >> 8),
			uint8(g >> 8),
			uint8(b >> 8),
			uint8(float64(a>>8) * alpha),
		}

		// Measure text to center rotation
		textWidth, textHeight := text.Measure(tile.icon, face, 0)
		if textWidth <= 0 || textHeight <= 0 {
			continue
		}

		// Draw tile with rotation - use simpler approach without rotation for now
		op := &text.DrawOptions{}
		// Position at tile location (centered)
		op.GeoM.Translate(tile.x-textWidth/2, tile.y-textHeight/2)

		// Apply color with alpha
		op.ColorScale.Reset()
		alpha32 := float32(alpha)
		op.ColorScale.Scale(alpha32, alpha32, alpha32, alpha32)
		op.ColorScale.ScaleWithColor(tileColor)

		text.Draw(screen, tile.icon, face, op)
	}
}
