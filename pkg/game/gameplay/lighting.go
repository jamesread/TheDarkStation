// Package gameplay provides core game logic for player movement and interactions.
package gameplay

import (
	"time"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// HeadlampRadius is the Chebyshev distance the maintenance unit's own lamp reaches
// in the direction the player is facing. Cells inside the lamp cone (with line of
// sight) are illuminated even on a dead grid.
const HeadlampRadius = 2

// HeadlampBackRadius is the residual glow behind the player: turning to look is a
// real action — the beam points where the player points.
const HeadlampBackRadius = 1

// UpdateLightingExploration recalculates power supply/consumption and applies
// power-driven lighting: a cell is illuminated when it sits on a live conduit from a
// powered generator (and, for named rooms, the room's lights circuit is enabled), or
// when it is within the player's headlamp radius.
func UpdateLightingExploration(g *state.Game) {
	if g.Grid == nil || g.CurrentCell == nil {
		return
	}
	nowMs := time.Now().UnixMilli()
	if len(g.Generators) > 0 {
		setup.AdvancePowerPropagation(g, nowMs)
	}
	setup.AdvanceRoomPowerOff(g, nowMs)
	setup.AdvanceGeneratorShutdown(g, nowMs)
	for _, roomName := range setup.AdvanceEgressSeal(g, nowMs) {
		logMessage(g, "ATMOS-SEAL: %s egress re-secured.", roomName)
	}

	totalConsumption := g.CalculatePowerConsumption()
	g.PowerConsumption = totalConsumption
	g.UpdatePowerSupply()
	setup.ApplyGridConductivePower(g)

	if setup.AnyArmedGridOverloaded(g) && !g.PowerOverloadWarned {
		logMessage(g, "WARNING: Power consumption exceeds supply on a power grid!")
		g.PowerOverloadWarned = true
	} else if !setup.AnyArmedGridOverloaded(g) {
		g.PowerOverloadWarned = false
	}

	applyPowerDrivenLighting(g)
}

// applyPowerDrivenLighting recomputes per-cell illumination.
//
// LightsOn is the live illumination state (recomputed every pass). Lighted is sticky:
// it records that the player has seen this cell illuminated at least once, which the
// renderer uses as the "remembered" knowledge tier.
func applyPowerDrivenLighting(g *state.Game) {
	if g.Grid == nil {
		return
	}
	live := setup.CellsReachableFromPoweredGenerators(g)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		data := gameworld.GetGameData(cell)
		data.GridLit = live.Has(cell) && roomLightsEnabled(g, cell)
		data.LightsOn = data.GridLit
		if data.LightsOn && cell.Discovered {
			data.Lighted = true
		}
	})
	applyHeadlamp(g)
}

// RefreshHeadlampCone re-aims the headlamp after a facing change without the full
// power recompute: grid-lit state is cached per cell (GridLit), so only the small
// box around the player needs touching. Cells the cone swung off fall back to
// their grid-powered state.
func RefreshHeadlampCone(g *state.Game) {
	center := g.CurrentCell
	if center == nil || g.Grid == nil {
		return
	}
	for dr := -HeadlampRadius; dr <= HeadlampRadius; dr++ {
		for dc := -HeadlampRadius; dc <= HeadlampRadius; dc++ {
			cell := g.Grid.GetCell(center.Row+dr, center.Col+dc)
			if cell == nil || !cell.Room {
				continue
			}
			data := gameworld.GetGameData(cell)
			data.LightsOn = data.GridLit
		}
	}
	applyHeadlamp(g)
}

// roomLightsEnabled reports whether the room's lights circuit allows illumination.
// Corridors and unnamed cells light directly from live conduits (no toggle).
func roomLightsEnabled(g *state.Game, cell *world.Cell) bool {
	if cell.Name == "" || cell.Name == "Corridor" {
		return true
	}
	if g.RoomLightsPowered == nil {
		return true
	}
	on, ok := g.RoomLightsPowered[cell.Name]
	return !ok || on
}

// applyHeadlamp illuminates cells near the player that are in line of sight.
// The beam is a forward cone: full radius in the facing half-plane, a short
// residual glow behind, so turning in place changes what the player can see.
func applyHeadlamp(g *state.Game) {
	center := g.CurrentCell
	if center == nil || g.Grid == nil {
		return
	}
	blocker := unpoweredDoorSightBlocker(g)
	faceRow, faceCol := g.PlayerFacing.Delta()
	for dr := -HeadlampRadius; dr <= HeadlampRadius; dr++ {
		for dc := -HeadlampRadius; dc <= HeadlampRadius; dc++ {
			if !headlampConeCovers(dr, dc, faceRow, faceCol) {
				continue
			}
			cell := g.Grid.GetCell(center.Row+dr, center.Col+dc)
			if cell == nil || !cell.Room {
				continue
			}
			if !headlampReaches(g.Grid, center, cell, blocker) {
				continue
			}
			data := gameworld.GetGameData(cell)
			data.LightsOn = true
			data.Lighted = true
			cell.Discovered = true
		}
	}
}

// headlampConeCovers reports whether the lamp reaches a cell at offset (dr, dc)
// while facing (faceRow, faceCol): forward half-plane gets HeadlampRadius, the
// rest gets the residual HeadlampBackRadius glow.
func headlampConeCovers(dr, dc, faceRow, faceCol int) bool {
	chebyshev := absInt(dr)
	if absInt(dc) > chebyshev {
		chebyshev = absInt(dc)
	}
	if chebyshev <= HeadlampBackRadius {
		return true
	}
	// Forward = non-negative projection onto the facing vector.
	return dr*faceRow+dc*faceCol >= 0 && chebyshev <= HeadlampRadius
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// headlampReaches reports whether a short sight ray from center reaches target.
func headlampReaches(grid *world.Grid, center, target *world.Cell, blocker world.SightBlocker) bool {
	if center == target {
		return true
	}
	endRow, endCol, ok := world.RayCastEndpoint(grid, center.Row, center.Col, target.Row, target.Col, blocker)
	if !ok {
		return false
	}
	if endRow == target.Row && endCol == target.Col {
		return true
	}
	// Bresenham rays are asymmetric; accept the reverse ray too so the lamp lights
	// diagonal corners consistently.
	endRow, endCol, ok = world.RayCastEndpoint(grid, target.Row, target.Col, center.Row, center.Col, blocker)
	return ok && endRow == center.Row && endCol == center.Col
}
