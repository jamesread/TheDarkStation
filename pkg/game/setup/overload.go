package setup

import (
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
)

// TriggerPowerOverloadForDev forces load above supply when needed, then runs overload
// resolution protecting protectedRoomName. For developer menu / testing only.
func TriggerPowerOverloadForDev(g *state.Game, protectedRoomName string) bool {
	if g == nil || protectedRoomName == "" || g.Grid == nil {
		return false
	}
	EnsureRoomPowerOnlineMap(g)
	g.UpdatePowerSupply()
	if CalculatePowerConsumption(g) <= g.PowerSupply {
		forceDevPowerOverload(g)
	}
	if ResolvePowerOverloadAfterToggle(g, protectedRoomName) {
		return true
	}
	grid := ArmedGridForRoom(g, protectedRoomName)
	if grid.Size() == 0 || ConsumptionOnArmedGrid(g, grid) <= ArmedGridSupply(g, grid) {
		return false
	}
	tripped := TripGeneratorsOnArmedGrid(g, grid)
	if tripped > 0 {
		g.UpdatePowerSupply()
		SchedulePowerPropagation(g, PowerNowMs())
		ApplyGridConductivePower(g)
		g.PowerConsumption = CalculatePowerConsumption(g)
	}
	return tripped > 0
}

func forceDevPowerOverload(g *state.Game) {
	if g.RoomDoorsPowered == nil {
		g.RoomDoorsPowered = make(map[string]bool)
	}
	if g.RoomCCTVPowered == nil {
		g.RoomCCTVPowered = make(map[string]bool)
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name == "" || cell.Name == "Corridor" {
			return
		}
		g.RoomDoorsPowered[cell.Name] = true
		g.RoomCCTVPowered[cell.Name] = true
	})
	EnergizeArmedRoomsForTest(g)
}

// ResolvePowerOverloadAfterToggle runs ShortOutIfOverload when load exceeds supply, then trips
// generators on the protected room's power grid and syncs maintenance terminals to generator-fed paths.
// Returns true if consumers were shed and/or generators tripped.
func ResolvePowerOverloadAfterToggle(g *state.Game, protectedRoomName string) bool {
	if g == nil {
		return false
	}
	g.UpdatePowerSupply()
	grid := ArmedGridForRoom(g, protectedRoomName)
	if grid.Size() == 0 {
		return false
	}
	if ConsumptionOnArmedGrid(g, grid) <= ArmedGridSupply(g, grid) {
		return false
	}
	shorted := ShortOutIfOverload(g, protectedRoomName)
	if ConsumptionOnArmedGrid(g, grid) <= ArmedGridSupply(g, grid) {
		if shorted {
			g.PowerConsumption = CalculatePowerConsumption(g)
			SchedulePowerPropagation(g, PowerNowMs())
			ApplyGridConductivePower(g)
		}
		return shorted
	}
	tripped := 0
	if shorted {
		tripped = TripGeneratorsOnArmedGrid(g, grid)
	}
	if tripped > 0 {
		g.UpdatePowerSupply()
	}
	if tripped > 0 || shorted {
		SchedulePowerPropagation(g, PowerNowMs())
		ApplyGridConductivePower(g)
		g.PowerConsumption = CalculatePowerConsumption(g)
	}
	return shorted || tripped > 0
}
