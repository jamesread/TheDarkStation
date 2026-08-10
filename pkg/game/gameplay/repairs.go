package gameplay

import (
	"fmt"
	"time"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/audio"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/levelgen"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func repairCanStart(g *state.Game, repair *entities.RepairObjective, cell *world.Cell) (bool, string) {
	if g == nil || repair == nil {
		return false, "Repair objective unavailable."
	}
	if repair.IsComplete() {
		return false, fmt.Sprintf("POWERED{%s repaired}", repair.Name)
	}
	if repair.IsDraining() {
		return false, repairDrainCallout(repair, time.Now().UnixMilli())
	}
	if !g.RepairPrereqsComplete(repair) {
		return false, repairPrereqCallout(g, repair)
	}
	if repair.NeedsLivePower() && (cell == nil || !setup.CellHasLivePower(g, cell)) {
		if dependent := inspectedRepairDependent(g, repair); dependent != nil {
			report := entities.RepairDependencyReport(dependent, repair)
			return false, fmt.Sprintf("UNPOWERED{%s}\nACTION{%s}\nSUBTLE{Restore this room's power, then return.}",
				repair.Name, report.Prediction)
		}
		return false, fmt.Sprintf("UNPOWERED{%s}\nSUBTLE{Restore this room's power, then return.}", repair.Name)
	}
	return true, ""
}

func repairPrereqCallout(g *state.Game, repair *entities.RepairObjective) string {
	upstream := g.FirstIncompleteRepairPrerequisite(repair)
	if upstream != nil {
		report := entities.RepairDependencyReport(repair, upstream)
		if repairDeviceSeen(g, upstream) {
			return fmt.Sprintf("UNPOWERED{%s}\nACTION{%s}\nSUBTLE{SOURCE: %s — %s}",
				report.Symptom, report.Diagnosis, upstream.Name, upstream.RoomName)
		}
		// The dependency has never been seen or inspected: report the failing
		// input without naming equipment the player cannot know about.
		return fmt.Sprintf("UNPOWERED{%s}\nACTION{%s}\nSUBTLE{SOURCE: unlocated — survey the deck}",
			report.Symptom, report.Diagnosis)
	}
	return fmt.Sprintf("UNPOWERED{%s locked}\nNeeds: ACTION{earlier repair}", repair.Name)
}

// repairDeviceSeen reports whether the player has seen this repair's device cell lit.
// Knowledge gates hint specificity: unseen systems are never named in callouts.
func repairDeviceSeen(g *state.Game, repair *entities.RepairObjective) bool {
	return g != nil && g.RepairDeviceKnown(repair)
}

func repairDrainCallout(repair *entities.RepairObjective, nowMs int64) string {
	remaining := repair.CompleteAtMs - nowMs
	if remaining < 0 {
		remaining = 0
	}
	secs := (remaining + 999) / 1000
	if secs < 1 {
		secs = 1
	}
	return fmt.Sprintf("UNPOWERED{%s draining}\nSUBTLE{Toxic slime clears in %ds.}", repair.Name, secs)
}

func repairBlockerCallout(g *state.Game, repair *entities.RepairObjective) string {
	if repair == nil {
		return "UNPOWERED{Blocked}"
	}
	if repair.IsDraining() {
		return repairDrainCallout(repair, time.Now().UnixMilli())
	}
	if repairDeviceSeen(g, repair) {
		return fmt.Sprintf("UNPOWERED{%s}\nNeeds: ACTION{%s}", repair.BlockerName, repair.Name)
	}
	return fmt.Sprintf("UNPOWERED{%s}\nSUBTLE{Whatever pumps this waste is still failed.}", repair.BlockerName)
}

func repairDeviceCallout(g *state.Game, repair *entities.RepairObjective, cell *world.Cell) string {
	if repair == nil {
		return "TITLE{Repair station}"
	}
	if ok, reason := repairCanStart(g, repair, cell); !ok {
		return reason
	}
	if repair.Type == entities.RepairSignalCalibrator {
		if repair.IsRoutingCoupler() {
			return fmt.Sprintf("TITLE{%s}\nSUBTLE{Press USE to open coupling alignment.}", repair.Name)
		}
		return fmt.Sprintf("TITLE{%s}\nSUBTLE{Press USE to align signal %s/%d.}", repair.Name, repair.CalibrationLabel(), entities.SignalCalibrationSteps)
	}
	if repair.Type == entities.RepairPowerCoupler {
		return fmt.Sprintf("TITLE{%s}\nSUBTLE{%s}", repair.Name, repair.CouplerCrankHint())
	}
	if dependent := inspectedRepairDependent(g, repair); dependent != nil {
		report := entities.RepairDependencyReport(dependent, repair)
		return fmt.Sprintf("TITLE{%s}\nACTION{%s}\nSUBTLE{Hold USE to repair.}",
			repair.Name, report.Prediction)
	}
	return fmt.Sprintf("TITLE{%s}\nSUBTLE{Hold USE to repair.}", repair.Name)
}

func inspectedRepairDependent(g *state.Game, upstream *entities.RepairObjective) *entities.RepairObjective {
	if g == nil || upstream == nil {
		return nil
	}
	for _, dependent := range g.RepairDependents(upstream.ID) {
		if dependent != nil && dependent.Inspected && !dependent.IsComplete() {
			return dependent
		}
	}
	return nil
}

// CheckAdjacentRepairAtCell handles tap interactions with repair devices.
func CheckAdjacentRepairAtCell(g *state.Game, cell *world.Cell) bool {
	if cell == nil || !gameworld.HasRepairDevice(cell) {
		return false
	}
	repair := gameworld.GetGameData(cell).RepairDevice
	if repair == nil {
		return false
	}
	repair.Inspect()
	if ok, reason := repairCanStart(g, repair, cell); !ok {
		renderer.AddCallout(cell.Row, cell.Col, reason, renderer.CalloutColorMaintenance, 0)
		audio.PlayUseDeny()
		return true
	}
	if repair.Type == entities.RepairSignalCalibrator {
		if repair.IsRoutingCoupler() {
			audio.PlayUseRepair()
			RunRoutingCouplerMenu(g, cell, repair)
			return true
		}
		advanceSignalCalibration(g, repair, cell)
		return true
	}
	if repair.Type == entities.RepairPowerCoupler {
		return TryPowerCouplerCrank(g, cell, repair)
	}
	renderer.AddCallout(cell.Row, cell.Col, repairDeviceCallout(g, repair, cell), renderer.CalloutColorMaintenance, 0)
	return true
}

func advanceSignalCalibration(g *state.Game, repair *entities.RepairObjective, cell *world.Cell) {
	if repair == nil || repair.IsComplete() {
		return
	}
	step := repair.CalibrationLabel()
	repair.CalibrationStep++
	if repair.CalibrationStep >= entities.SignalCalibrationSteps {
		completeRepair(g, repair, cell)
		return
	}
	audio.PlayUseRepair()
	renderer.AddCallout(cell.Row, cell.Col,
		fmt.Sprintf("TITLE{%s}\nAligned signal %s. Next: ACTION{%s}", repair.Name, step, repair.CalibrationLabel()),
		renderer.CalloutColorMaintenance, 0)
}

func OnRepairTimersAdvanced(g *state.Game) {
	if g == nil {
		return
	}
	for _, repair := range g.RepairObjectives {
		if repair != nil && repair.IsComplete() && !repair.Acknowledged {
			acknowledgeRepairConsequences(g, repair, false)
		}
	}
	levelgen.SpawnUnlockKeycardPayoffs(g)
	g.MaybeSetReactorOnlineFromDeck()
}

func completeRepair(g *state.Game, repair *entities.RepairObjective, cell *world.Cell) {
	if g == nil || repair == nil {
		return
	}
	wasLiftReady := setup.ExitLiftReady(g)
	audio.PlayUseRepair()
	now := time.Now().UnixMilli()
	repair.BeginTimedCompletion(now)
	if repair.IsComplete() {
		g.OnRoutingRepairComplete(repair.ID)
	}
	if repair.Type == entities.RepairConduitSplice && repair.IsComplete() {
		// The spliced segment conducts again: recompute grid power and lighting.
		setup.NotifyPowerGridChanged(g)
		UpdateLightingExploration(g)
	}
	levelgen.SpawnUnlockKeycardPayoffs(g)
	g.MaybeSetReactorOnlineFromDeck()
	if cell != nil {
		renderer.AddDevicePulse(cell.Row, cell.Col)
	}
	if repair.IsDraining() {
		renderer.AddCallout(cell.Row, cell.Col,
			fmt.Sprintf("TITLE{%s repaired}\nSUBTLE{Waste pumps draining toxic slime...}", repair.Name),
			renderer.CalloutColorSuccess, 0)
		logMessage(g, "%s repaired. Toxic slime is draining.", repair.Name)
		return
	}
	acknowledgeRepairConsequences(g, repair, wasLiftReady)
	renderer.AddCallout(cell.Row, cell.Col, fmt.Sprintf("POWERED{%s repaired}", repair.Name), renderer.CalloutColorSuccess, 0)
	logMessage(g, "%s repaired.", repair.Name)
}

func acknowledgeRepairConsequences(g *state.Game, upstream *entities.RepairObjective, wasLiftReady bool) {
	if g == nil || upstream == nil || !upstream.IsComplete() {
		return
	}
	upstream.Acknowledged = true
	for _, dependent := range g.RepairDependents(upstream.ID) {
		if dependent == nil || dependent.IsComplete() || !g.RepairPrereqsComplete(dependent) ||
			!g.RepairDeviceKnown(dependent) {
			continue
		}
		report := entities.RepairDependencyReport(dependent, upstream)
		renderer.AddDevicePulse(dependent.DeviceRow, dependent.DeviceCol)
		audio.PlaySignalLockReady()
		logMessage(g, "POWERED{"+report.Restored+"}")
	}
	if wasLiftReady || !setup.ExitLiftReady(g) {
		return
	}
	if exit := setup.ExitCell(g); exit != nil && exit.Discovered {
		renderer.AddDevicePulse(exit.Row, exit.Col)
	}
	audio.PlaySignalLockReady()
	logMessage(g, "POWERED{LIFT POWER TRANSFER AVAILABLE}")
}
