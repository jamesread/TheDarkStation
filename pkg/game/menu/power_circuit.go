package menu

import (
	"fmt"
	"strings"

	"darkstation/pkg/game/state"
)

// CircuitPreset is a room load profile for maintenance routing.
type CircuitPreset int

const (
	CircuitOff CircuitPreset = iota
	CircuitEssential
	CircuitFull
)

func (p CircuitPreset) String() string {
	switch p {
	case CircuitOff:
		return "OFF"
	case CircuitEssential:
		return "ESSENTIAL"
	case CircuitFull:
		return "FULL"
	default:
		return "?"
	}
}

// PrevPreset cycles FULL → ESSENTIAL → OFF → FULL.
func (p CircuitPreset) PrevPreset() CircuitPreset {
	switch p {
	case CircuitFull:
		return CircuitEssential
	case CircuitEssential:
		return CircuitOff
	default:
		return CircuitFull
	}
}

// NextPreset cycles OFF → ESSENTIAL → FULL → OFF.
func (p CircuitPreset) NextPreset() CircuitPreset {
	switch p {
	case CircuitOff:
		return CircuitEssential
	case CircuitEssential:
		return CircuitFull
	default:
		return CircuitOff
	}
}

// CurrentCircuitPreset derives the preset from room power maps.
func CurrentCircuitPreset(g *state.Game, roomName string) CircuitPreset {
	if g == nil {
		return CircuitOff
	}
	doors := g.RoomDoorsPowered[roomName]
	cctv := g.RoomCCTVPowered[roomName]
	if doors && cctv {
		return CircuitFull
	}
	if doors {
		return CircuitEssential
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

	g.RoomDoorsPowered[roomName] = doorsOn
	g.RoomCCTVPowered[roomName] = cctvOn

	if !doorsOn && !cctvOn {
		g.UpdatePowerSupply()
		g.PowerConsumption = g.CalculatePowerConsumption()
		return ""
	}

	help := ""
	if doorsOn {
		if g.ShortOutIfOverload(roomName) {
			help = "Power overload! Other systems shorted out."
		}
	}
	if cctvOn && g.RoomCCTVPowered[roomName] {
		if g.ShortOutIfOverload(roomName) {
			if help == "" {
				help = "Power overload! Other systems shorted out."
			}
		}
	}
	if help == "" && g.PowerConsumption > g.PowerSupply {
		help = "Power overload persists. Reduce load."
	}
	return help
}

// PreviewCircuitShed returns a human-readable preview of shedding if preset were applied.
func PreviewCircuitShed(g *state.Game, roomName string, preset CircuitPreset) string {
	if g == nil {
		return ""
	}
	doorsOn := preset == CircuitEssential || preset == CircuitFull
	cctvOn := preset == CircuitFull
	shed := g.PreviewShortOutIfOverload(roomName, doorsOn, cctvOn)
	if len(shed) == 0 {
		return "Preview: no shedding required"
	}
	var parts []string
	for _, e := range shed {
		parts = append(parts, fmt.Sprintf("%s %s", e.Room, e.Kind))
	}
	return "Will shed: " + strings.Join(parts, ", ")
}
