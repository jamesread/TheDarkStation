package ebiten

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

var fovRayColor = color.RGBA{255, 220, 80, 140}

func fovSightBlocker(g *state.Game) world.SightBlocker {
	if g == nil {
		return nil
	}
	return func(cell *world.Cell) bool {
		if !gameworld.HasDoor(cell) {
			return false
		}
		roomName := gameworld.GetGameData(cell).Door.RoomName
		return !g.RoomDoorsPowered[roomName]
	}
}

func mapCellCenterScreen(mapX, mapY float64, row, col, startRow, startCol, tileSize int) (float32, float32) {
	vCol := col - startCol
	vRow := row - startRow
	half := float64(tileSize) / 2
	return float32(mapX + float64(vCol)*float64(tileSize) + half),
		float32(mapY + float64(vRow)*float64(tileSize) + half)
}

func (e *EbitenRenderer) drawFOVRays(screen *ebiten.Image, g *state.Game, mapX, mapY float64, startRow, startCol int) {
	if !e.DrawFOVRaysEnabled() || g == nil || g.Grid == nil || g.CurrentCell == nil {
		return
	}

	center := g.CurrentCell
	rays := world.CollectFOVRays(g.Grid, center, fovSightBlocker(g))
	if len(rays) == 0 {
		return
	}

	cx, cy := mapCellCenterScreen(mapX, mapY, center.Row, center.Col, startRow, startCol, e.tileSize)
	const lineWidth = 1.0
	for _, ray := range rays {
		ex, ey := mapCellCenterScreen(mapX, mapY, ray.EndRow, ray.EndCol, startRow, startCol, e.tileSize)
		vector.StrokeLine(screen, cx, cy, ex, ey, lineWidth, fovRayColor, false)
	}
}
