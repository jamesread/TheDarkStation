package gameplay

import (
	"time"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// LongUseKind identifies a hold-to-use interaction type (extensible for future gameplay).
type LongUseKind string

const (
	LongUseGeneratorPowerUp  LongUseKind = "generator_power_up"
	LongUseDoorManualRelease LongUseKind = "door_manual_release"
	LongUseRepair            LongUseKind = "repair"
	LongUseCouplerCrank      LongUseKind = "coupler_crank"
)

// LongUseHoldDuration is how long the player must hold USE for hold-to-use interactions.
const LongUseHoldDuration = 3 * time.Second

// GeneratorPowerUpDuration is how long the player must hold USE to restart an unpowered generator.
const GeneratorPowerUpDuration = LongUseHoldDuration

// TryBeginLongUseOnAdjacent starts a hold interaction when an adjacent target requires it.
// Returns true if a session was started (caller should skip normal tap interact).
func TryBeginLongUseOnAdjacent(g *state.Game) bool {
	if g == nil || g.CurrentCell == nil {
		return false
	}
	cell, kind, ok := findAdjacentLongUseTarget(g)
	if !ok {
		return false
	}
	return beginLongUse(g, kind, cell)
}

func findAdjacentLongUseTarget(g *state.Game) (*world.Cell, LongUseKind, bool) {
	// Current cell first: walkable devices (conduit splices) can be underfoot.
	neighbors := append([]*world.Cell{g.CurrentCell},
		state.AdjacentCellsClockwiseFromFacing(g.CurrentCell, g.PlayerFacing)...)
	for _, cell := range neighbors {
		if cell == nil {
			continue
		}
		if kind, ok := longUseKindForCell(g, cell); ok {
			return cell, kind, true
		}
	}
	for _, cell := range neighbors {
		if cell == nil {
			continue
		}
		if doorNeedsManualRelease(g, cell) {
			return cell, LongUseDoorManualRelease, true
		}
	}
	return nil, "", false
}

func longUseKindForCell(g *state.Game, cell *world.Cell) (LongUseKind, bool) {
	if cell == nil {
		return "", false
	}
	if gameworld.HasRepairDevice(cell) {
		repair := gameworld.GetGameData(cell).RepairDevice
		if repair != nil && repair.NeedsLongUse() {
			if ok, _ := repairCanStart(g, repair, cell); ok {
				return LongUseRepair, true
			}
		}
	}
	if !gameworld.HasGenerator(cell) {
		return "", false
	}
	gen := gameworld.GetGameData(cell).Generator
	if generatorNeedsLongUsePowerUp(gen) {
		return LongUseGeneratorPowerUp, true
	}
	return "", false
}

// GeneratorNeedsLongUsePowerUp reports whether the generator must be powered up via hold-to-use.
func GeneratorNeedsLongUsePowerUp(gen *entities.Generator) bool {
	return generatorNeedsLongUsePowerUp(gen)
}

func generatorNeedsLongUsePowerUp(gen *entities.Generator) bool {
	return gen != nil && gen.NeedsStartupSequence()
}

func beginLongUse(g *state.Game, kind LongUseKind, cell *world.Cell) bool {
	if g == nil || cell == nil || kind == "" {
		return false
	}
	duration := longUseDuration(kind)
	if duration <= 0 {
		return false
	}
	FaceTowardAdjacentCell(g, cell)
	g.LongUse = &state.LongUseSession{
		Kind:        string(kind),
		TargetRow:   cell.Row,
		TargetCol:   cell.Col,
		DurationMs:  duration.Milliseconds(),
		StartedAtMs: time.Now().UnixMilli(),
	}
	return true
}

func longUseDuration(kind LongUseKind) time.Duration {
	switch kind {
	case LongUseGeneratorPowerUp, LongUseDoorManualRelease, LongUseRepair:
		return LongUseHoldDuration
	default:
		return 0
	}
}

// IsLongUseActive reports whether a hold-to-use session is in progress.
func IsLongUseActive(g *state.Game) bool {
	return g != nil && g.LongUse != nil
}

// LongUseProgress returns 0..1 progress for the active session (0 if inactive).
func LongUseProgress(g *state.Game, nowMs int64) float64 {
	if g == nil || g.LongUse == nil || g.LongUse.DurationMs <= 0 {
		return 0
	}
	_ = nowMs
	p := float64(g.LongUse.AccumulatedMs) / float64(g.LongUse.DurationMs)
	if p > 1 {
		return 1
	}
	if p < 0 {
		return 0
	}
	return p
}

// CancelLongUse aborts an in-progress hold interaction.
func CancelLongUse(g *state.Game) {
	if g != nil {
		g.LongUse = nil
	}
}

// TickLongUse advances the session; returns true when duration is reached.
func TickLongUse(g *state.Game, nowMs int64) bool {
	if g == nil || g.LongUse == nil {
		return false
	}
	if !longUseTargetStillValid(g) {
		CancelLongUse(g)
		return false
	}
	return nowMs-g.LongUse.StartedAtMs >= g.LongUse.DurationMs
}

func longUseTargetStillValid(g *state.Game) bool {
	if g == nil || g.LongUse == nil || g.Grid == nil {
		return false
	}
	cell := g.Grid.GetCell(g.LongUse.TargetRow, g.LongUse.TargetCol)
	if cell == nil {
		return false
	}
	switch LongUseKind(g.LongUse.Kind) {
	case LongUseGeneratorPowerUp:
		kind, ok := longUseKindForCell(g, cell)
		return ok && kind == LongUseGeneratorPowerUp
	case LongUseDoorManualRelease:
		return doorNeedsManualRelease(g, cell)
	case LongUseRepair:
		kind, ok := longUseKindForCell(g, cell)
		return ok && kind == LongUseRepair
	default:
		return false
	}
}

// CompleteLongUse applies the effect for a finished hold interaction.
func CompleteLongUse(g *state.Game) {
	if g == nil || g.LongUse == nil {
		return
	}
	cell := g.Grid.GetCell(g.LongUse.TargetRow, g.LongUse.TargetCol)
	kind := LongUseKind(g.LongUse.Kind)
	CancelLongUse(g)
	if cell == nil {
		return
	}
	switch kind {
	case LongUseGeneratorPowerUp:
		completeGeneratorPowerUp(g, cell)
	case LongUseDoorManualRelease:
		completeManualDoorRelease(g, cell)
	case LongUseRepair:
		completeRepairLongUse(g, cell)
	}
}

func completeGeneratorPowerUp(g *state.Game, cell *world.Cell) {
	if cell == nil || !gameworld.HasGenerator(cell) {
		return
	}
	gen := gameworld.GetGameData(cell).Generator
	if !generatorNeedsLongUsePowerUp(gen) {
		return
	}
	if !gen.Restart() {
		return
	}
	setup.NotifyPowerGridChanged(g)
	setup.BootstrapPoweredGenerators(g, cell)
	UpdateLightingExploration(g)
	renderer.AddDevicePulse(cell.Row, cell.Col)
	renderer.AddCallout(cell.Row, cell.Col,
		"POWERED{"+gen.Name+" - online}", renderer.CalloutColorGeneratorOn, 0)
	logMessage(g, "ITEM{%s} is now powered!", gen.Name)
	logMessage(g, "Power supply: %dw available", g.GetAvailablePower())
	ToggleGeneratorPowerGridOverlay(g, cell)
}

func completeRepairLongUse(g *state.Game, cell *world.Cell) {
	if cell == nil || !gameworld.HasRepairDevice(cell) {
		return
	}
	repair := gameworld.GetGameData(cell).RepairDevice
	if ok, _ := repairCanStart(g, repair, cell); !ok {
		return
	}
	completeRepair(g, repair, cell)
}

// AdvanceLongUseIfActive ticks an in-progress hold interaction. Call from the Ebiten Update
// thread each frame with live USE hold state so progress matches input and Draw.
// Cancel only on button-up (released); !held without release keeps the session paused.
func AdvanceLongUseIfActive(g *state.Game, held, released bool, nowMs int64) {
	if g == nil || g.LongUse == nil {
		return
	}
	if LongUseKind(g.LongUse.Kind) == LongUseCouplerCrank {
		return
	}
	session := g.LongUse
	if released {
		CancelLongUse(g)
		return
	}
	if !held {
		// Tap without hold: session may start after the press frame; cancel if USE never registers.
		if session.LastAdvanceMs == 0 && nowMs-session.StartedAtMs > 200 {
			CancelLongUse(g)
		}
		return
	}
	if !longUseTargetStillValid(g) {
		CancelLongUse(g)
		return
	}
	if session.LastAdvanceMs == 0 {
		session.LastAdvanceMs = nowMs
		return
	}
	delta := nowMs - session.LastAdvanceMs
	if delta > 0 {
		session.AccumulatedMs += delta
	}
	session.LastAdvanceMs = nowMs
	if session.AccumulatedMs >= session.DurationMs {
		CompleteLongUse(g)
	}
}

// WaitForLongUseComplete keeps rendering until the hold interaction finishes or is cancelled.
// AdvanceLongUseIfActive must run on the Ebiten Update thread while this waits.
func WaitForLongUseComplete(g *state.Game) {
	for IsLongUseActive(g) {
		renderer.RenderFrame(g)
		time.Sleep(16 * time.Millisecond)
	}
}
