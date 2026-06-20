package gameplay

import (
	"time"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

const (
	couplerCrankTargetProgress     = 1000
	couplerCrankPumpAmount         = 250
	couplerCrankDrainPerSec        = int64(500)
	couplerCrankAbandonDrainPerSec = int64(2000)
)

// AbandonCouplerCrank marks an in-progress crank to drain out after the player walks away.
func AbandonCouplerCrank(g *state.Game) {
	if !IsCouplerCrankActive(g) {
		return
	}
	g.LongUse.Abandoning = true
}

// IsHoldLongUseActive reports hold-to-use sessions that block normal gameplay until complete.
func IsHoldLongUseActive(g *state.Game) bool {
	return IsLongUseActive(g) && !IsCouplerCrankActive(g)
}

// IsCouplerCrankActive reports whether a power coupler crank session is in progress.
func IsCouplerCrankActive(g *state.Game) bool {
	return g != nil && g.LongUse != nil && LongUseKind(g.LongUse.Kind) == LongUseCouplerCrank
}

// TryPowerCouplerCrank starts or advances a rapid-tap coupler crank on the repair cell.
func TryPowerCouplerCrank(g *state.Game, cell *world.Cell, repair *entities.RepairObjective) bool {
	if g == nil || cell == nil || repair == nil || repair.IsComplete() {
		return false
	}
	if ok, _ := repairCanStart(g, repair, cell); !ok {
		return false
	}
	FaceTowardAdjacentCell(g, cell)
	if IsCouplerCrankActive(g) {
		if g.LongUse.Abandoning {
			return false
		}
		if g.LongUse.TargetRow != cell.Row || g.LongUse.TargetCol != cell.Col {
			return false
		}
		pumpCouplerCrank(g)
		return true
	}
	beginCouplerCrank(g, cell)
	pumpCouplerCrank(g)
	return true
}

func beginCouplerCrank(g *state.Game, cell *world.Cell) {
	if g == nil || cell == nil {
		return
	}
	now := time.Now().UnixMilli()
	g.LongUse = &state.LongUseSession{
		Kind:          string(LongUseCouplerCrank),
		TargetRow:     cell.Row,
		TargetCol:     cell.Col,
		DurationMs:    couplerCrankTargetProgress,
		StartedAtMs:   now,
		LastAdvanceMs: now,
	}
}

func pumpCouplerCrank(g *state.Game) {
	if !IsCouplerCrankActive(g) {
		return
	}
	g.LongUse.AccumulatedMs += couplerCrankPumpAmount
	if g.LongUse.AccumulatedMs >= g.LongUse.DurationMs {
		completeCouplerCrank(g)
	}
}

// AdvanceCouplerCrankIfActive drains crank progress over time. Call each frame from the renderer Update thread.
func AdvanceCouplerCrankIfActive(g *state.Game, nowMs int64) {
	if !IsCouplerCrankActive(g) {
		return
	}
	session := g.LongUse
	if !couplerCrankTargetStillValid(g) {
		CancelLongUse(g)
		return
	}
	if session.LastAdvanceMs == 0 {
		session.LastAdvanceMs = nowMs
		return
	}
	delta := nowMs - session.LastAdvanceMs
	session.LastAdvanceMs = nowMs
	if delta <= 0 {
		return
	}
	rate := couplerCrankDrainPerSec
	if session.Abandoning {
		rate = couplerCrankAbandonDrainPerSec
	}
	drain := delta * rate / 1000
	session.AccumulatedMs -= drain
	if session.AccumulatedMs <= 0 {
		session.AccumulatedMs = 0
		CancelLongUse(g)
	}
}

func couplerCrankTargetStillValid(g *state.Game) bool {
	if g == nil || g.LongUse == nil || g.Grid == nil {
		return false
	}
	cell := g.Grid.GetCell(g.LongUse.TargetRow, g.LongUse.TargetCol)
	if cell == nil || !gameworld.HasRepairDevice(cell) {
		return false
	}
	repair := gameworld.GetGameData(cell).RepairDevice
	if repair == nil || repair.Type != entities.RepairPowerCoupler || repair.IsComplete() {
		return false
	}
	ok, _ := repairCanStart(g, repair, cell)
	return ok
}

func completeCouplerCrank(g *state.Game) {
	if g == nil || g.LongUse == nil || g.Grid == nil {
		return
	}
	cell := g.Grid.GetCell(g.LongUse.TargetRow, g.LongUse.TargetCol)
	CancelLongUse(g)
	if cell == nil || !gameworld.HasRepairDevice(cell) {
		return
	}
	repair := gameworld.GetGameData(cell).RepairDevice
	if repair == nil {
		return
	}
	completeRepair(g, repair, cell)
}

// AdvanceInteractionProgress ticks coupler crank drain and hold-to-use sessions.
func AdvanceInteractionProgress(g *state.Game, held, released bool, nowMs int64) {
	AdvanceCouplerCrankIfActive(g, nowMs)
	AdvanceLongUseIfActive(g, held, released, nowMs)
}

// abandonCouplerCrankOnMove abandons an active coupler crank or cancels hold-to-use on movement.
func abandonCouplerCrankOnMove(g *state.Game) {
	if IsCouplerCrankActive(g) {
		AbandonCouplerCrank(g)
		return
	}
	CancelLongUse(g)
}
