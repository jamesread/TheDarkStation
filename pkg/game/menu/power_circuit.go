package menu

import (
	"fmt"
	"strings"

	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
)

// CircuitPreset is a room load profile for maintenance routing.
// CircuitEssential is retained for future use but is not exposed in the maintenance menu.
type CircuitPreset int

const (
	CircuitOff CircuitPreset = iota
	CircuitEssential
	CircuitFull // menu label: ON
)

func (p CircuitPreset) String() string {
	switch p {
	case CircuitOff:
		return "OFF"
	case CircuitEssential:
		return "ESSENTIAL"
	case CircuitFull:
		return "ON"
	default:
		return "?"
	}
}

// CircuitPresetMarkup returns menu markup for the preset label (OFF=unpowered, ON=powered).
func CircuitPresetMarkup(p CircuitPreset) string {
	switch p {
	case CircuitOff:
		return "UNPOWERED{OFF}"
	case CircuitFull:
		return "POWERED{ON}"
	case CircuitEssential:
		return "POWERED{ON}" // legacy doors-only reads as ON in menu
	default:
		return p.String()
	}
}

// PrevPreset cycles ON → OFF → ON (essential skipped in menu).
func (p CircuitPreset) PrevPreset() CircuitPreset {
	if p == CircuitOff {
		return CircuitFull
	}
	return CircuitOff
}

// NextPreset cycles OFF → ON → OFF (essential skipped in menu).
func (p CircuitPreset) NextPreset() CircuitPreset {
	if p == CircuitOff {
		return CircuitFull
	}
	return CircuitOff
}

// CurrentCircuitPreset derives the menu preset from room power maps (OFF or ON only).
func CurrentCircuitPreset(g *state.Game, roomName string) CircuitPreset {
	if g == nil {
		return CircuitOff
	}
	if setup.RoomPowerOffScheduled(g, roomName) {
		return CircuitOff
	}
	if g.RoomDoorsPowered[roomName] {
		return CircuitFull
	}
	return CircuitOff
}

// ApplyCircuitPreset sets doors/CCTV for roomName per preset and runs short-out when loads turn on.
// Returns help text for overload feedback (may be empty).
func ApplyCircuitPreset(g *state.Game, roomName string, preset CircuitPreset) string {
	if g == nil || roomName == "" {
		return ""
	}
	doorsOn := preset == CircuitEssential || preset == CircuitFull
	cctvOn := preset == CircuitFull

	if !doorsOn {
		if !g.RoomDoorsPowered[roomName] && !g.RoomCCTVPowered[roomName] {
			setup.CancelRoomPowerOff(g, roomName)
			return ""
		}
		if setup.RoomPowerOffScheduled(g, roomName) {
			setup.ScheduleRoomPowerOff(g, roomName, setup.PowerNowMs())
			secs := int(setup.RoomPowerOffDelay.Seconds())
			return fmt.Sprintf("Power shutdown timer reset — %d seconds to leave the room", secs)
		}
		setup.ScheduleRoomPowerOff(g, roomName, setup.PowerNowMs())
		secs := int(setup.RoomPowerOffDelay.Seconds())
		return fmt.Sprintf("Power shutting down in %d seconds — leave the room now", secs)
	}

	setup.CancelRoomPowerOff(g, roomName)

	g.RoomDoorsPowered[roomName] = doorsOn
	g.RoomCCTVPowered[roomName] = cctvOn

	help := ""
	setup.NotifyPowerGridChanged(g)

	if cctvOn && setup.RoomIsOnline(g, roomName) {
		if setup.ResolvePowerOverloadAfterToggle(g, roomName) {
			help = "Power overload! Generators tripped — other systems shorted out."
		} else if setup.ConsumptionOnArmedGrid(g, setup.ArmedGridForRoom(g, roomName)) > setup.ArmedGridSupplyForRoom(g, roomName) {
			help = "Power overload persists. Reduce load."
		}
	}
	return help
}

// PreviewCircuitToggleImpact describes the watt change if targetPreset were applied to roomName.
func PreviewCircuitToggleImpact(g *state.Game, roomName string, targetPreset CircuitPreset) string {
	if g == nil || roomName == "" {
		return ""
	}
	doorsOn := targetPreset == CircuitEssential || targetPreset == CircuitFull
	cctvOn := targetPreset == CircuitFull

	before, afterApply, afterShed := setup.PreviewRoomPresetConsumption(g, roomName, doorsOn, cctvOn)
	delta := afterApply - before
	if delta == 0 {
		return "No change to station load"
	}
	if delta < 0 {
		return fmt.Sprintf("%dw usage", delta)
	}
	gridSupply := setup.ArmedGridSupplyForRoomPreset(g, roomName, doorsOn)
	if afterShed > gridSupply {
		over := afterApply - gridSupply
		if over < 0 {
			over = 0
		}
		return fmt.Sprintf("+%dw, %dw over supply - will trigger overload!", delta, over)
	}
	return fmt.Sprintf("+%dw usage", delta)
}

// PreviewCircuitShed returns a human-readable preview of shedding if preset were applied.
func PreviewCircuitShed(g *state.Game, roomName string, preset CircuitPreset) string {
	if g == nil {
		return ""
	}
	doorsOn := preset == CircuitEssential || preset == CircuitFull
	cctvOn := preset == CircuitFull
	shed := setup.PreviewShortOutIfOverload(g, roomName, doorsOn, cctvOn)
	if len(shed) == 0 {
		return "Preview: no shedding required"
	}
	var parts []string
	for _, e := range shed {
		parts = append(parts, fmt.Sprintf("%s %s", e.Room, e.Kind))
	}
	return "Will shed: " + strings.Join(parts, ", ")
}
