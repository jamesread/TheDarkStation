package setup

import (
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// GeneratorOutputWattsNominal is the fixed output of each powered generator (watts).
const GeneratorOutputWattsNominal = 100

// PowerShedEntry describes one consumer that would be unpowered during short-out preview or apply.
type PowerShedEntry struct {
	Room string
	Kind string // "doors" or "cctv"
}

// CalculatePowerConsumption returns total deck power draw from online rooms.
func CalculatePowerConsumption(g *state.Game) int {
	if g == nil || g.RoomPowerOnline == nil {
		return 0
	}
	cctv := make(map[string]bool, len(g.RoomCCTVPowered))
	for k, v := range g.RoomCCTVPowered {
		cctv[k] = v
	}
	return calculateConsumptionFromMaps(g, g.RoomPowerOnline, cctv, nil)
}

// ConsumptionOnArmedGrid returns draw from online rooms on the given armed routing grid.
func ConsumptionOnArmedGrid(g *state.Game, grid *mapset.Set[*world.Cell]) int {
	if g == nil || grid == nil || g.RoomPowerOnline == nil {
		return 0
	}
	cctv := make(map[string]bool, len(g.RoomCCTVPowered))
	for k, v := range g.RoomCCTVPowered {
		cctv[k] = v
	}
	return calculateConsumptionFromMaps(g, g.RoomPowerOnline, cctv, grid)
}

// ConsumptionIfRoomCameOnline returns watts on the armed grid feeding roomName if it were online.
func ConsumptionIfRoomCameOnline(g *state.Game, roomName string) int {
	if g == nil || g.Grid == nil || roomName == "" {
		return 0
	}
	grid := ArmedGridForRoom(g, roomName)
	if grid.Size() == 0 {
		return 0
	}
	online := make(map[string]bool, len(g.RoomPowerOnline))
	for k, v := range g.RoomPowerOnline {
		online[k] = v
	}
	online[roomName] = true
	cctv := make(map[string]bool, len(g.RoomCCTVPowered))
	for k, v := range g.RoomCCTVPowered {
		cctv[k] = v
	}
	return calculateConsumptionFromMaps(g, online, cctv, grid)
}

// ArmedGridSupplyForRoom returns generator supply on the armed grid feeding roomName.
func ArmedGridSupplyForRoom(g *state.Game, roomName string) int {
	return ArmedGridSupply(g, ArmedGridForRoom(g, roomName))
}

// ArmedGridSupplyForRoomPreset returns supply on the armed grid if roomName doors were doorsOn.
func ArmedGridSupplyForRoomPreset(g *state.Game, roomName string, doorsOn bool) int {
	doors := roomDoorsPoweredEffective(g, map[string]bool{roomName: doorsOn})
	return ArmedGridSupply(g, ArmedGridForRoomDoors(g, roomName, doors))
}

// GridPowerSummary returns supply, consumption, and free watts for the armed grid at cell.
func GridPowerSummary(g *state.Game, cell *world.Cell) (supply, consumption, free int) {
	if g == nil {
		return 0, 0, 0
	}
	grid := ArmedGridForCell(g, cell)
	supply = ArmedGridSupply(g, grid)
	consumption = ConsumptionOnArmedGrid(g, grid)
	free = supply - consumption
	return supply, consumption, free
}

// RoomWouldOverloadGrid reports whether bringing roomName online exceeds its grid supply.
func RoomWouldOverloadGrid(g *state.Game, roomName string) bool {
	if g == nil || roomName == "" {
		return false
	}
	return ConsumptionIfRoomCameOnline(g, roomName) > ArmedGridSupplyForRoom(g, roomName)
}

// AnyArmedGridOverloaded reports whether any disjoint armed grid exceeds its local supply.
func AnyArmedGridOverloaded(g *state.Game) bool {
	if g == nil {
		return false
	}
	grids := armedGridComponentsForBalance(g, nil)
	if len(grids) == 0 {
		return CalculatePowerConsumption(g) > g.PowerSupply
	}
	for _, grid := range grids {
		if ConsumptionOnArmedGrid(g, grid) > ArmedGridSupply(g, grid) {
			return true
		}
	}
	return false
}

// ShortOutIfOverload sheds load on the protected room's armed grid until within local supply.
func ShortOutIfOverload(g *state.Game, protectedRoomName string) bool {
	if g == nil {
		return false
	}
	grid := ArmedGridForRoom(g, protectedRoomName)
	if grid.Size() == 0 {
		return false
	}
	supply := ArmedGridSupply(g, grid)
	consumption := ConsumptionOnArmedGrid(g, grid)
	if consumption <= supply {
		return false
	}

	type consumer struct{ room, kind string }
	var list []consumer
	for roomName, on := range g.RoomPowerOnline {
		if roomName == protectedRoomName || !on || !roomHasCellOnGrid(g, roomName, grid) {
			continue
		}
		list = append(list, consumer{roomName, "doors"})
	}
	for roomName, on := range g.RoomCCTVPowered {
		if roomName == protectedRoomName || !on {
			continue
		}
		if g.RoomPowerOnline != nil && g.RoomPowerOnline[roomName] && roomHasCellOnGrid(g, roomName, grid) {
			list = append(list, consumer{roomName, "cctv"})
		}
	}
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].room < list[i].room || (list[j].room == list[i].room && list[j].kind == "doors" && list[i].kind == "cctv") {
				list[i], list[j] = list[j], list[i]
			}
		}
	}

	shortOut := false
	for _, c := range list {
		consumption = ConsumptionOnArmedGrid(g, grid)
		if consumption <= supply {
			break
		}
		if c.kind == "doors" && g.RoomDoorsPowered[c.room] {
			g.RoomDoorsPowered[c.room] = false
			if g.RoomPowerOnline != nil {
				g.RoomPowerOnline[c.room] = false
			}
			if g.RoomCCTVPowered[c.room] {
				g.RoomCCTVPowered[c.room] = false
			}
			shortOut = true
		} else if c.kind == "cctv" && g.RoomCCTVPowered[c.room] {
			g.RoomCCTVPowered[c.room] = false
			shortOut = true
		}
	}
	if shortOut {
		g.PowerConsumption = CalculatePowerConsumption(g)
	}
	return shortOut
}

// PreviewShortOutIfOverload simulates preset changes and returns consumers shed on that grid.
func PreviewShortOutIfOverload(g *state.Game, protectedRoomName string, doorsOn, cctvOn bool) []PowerShedEntry {
	if g == nil {
		return nil
	}
	doors := roomDoorsPoweredEffective(g, map[string]bool{protectedRoomName: doorsOn})
	grid := ArmedGridForRoomDoors(g, protectedRoomName, doors)
	if grid.Size() == 0 {
		return nil
	}

	armed := make(map[string]bool, len(g.RoomDoorsPowered))
	cctv := make(map[string]bool, len(g.RoomCCTVPowered))
	for k, v := range g.RoomDoorsPowered {
		armed[k] = v
	}
	for k, v := range g.RoomCCTVPowered {
		cctv[k] = v
	}
	armed[protectedRoomName] = doorsOn
	cctv[protectedRoomName] = cctvOn

	online := make(map[string]bool, len(g.RoomPowerOnline))
	for k, v := range g.RoomPowerOnline {
		online[k] = v
	}
	if doorsOn {
		online[protectedRoomName] = true
	} else {
		online[protectedRoomName] = false
	}

	supply := ArmedGridSupply(g, grid)
	consumption := calculateConsumptionFromMaps(g, online, cctv, grid)
	if consumption <= supply {
		return nil
	}

	type consumer struct{ room, kind string }
	var list []consumer
	for roomName, on := range online {
		if roomName == protectedRoomName || !on || !roomHasCellOnGrid(g, roomName, grid) {
			continue
		}
		list = append(list, consumer{roomName, "doors"})
	}
	for roomName, on := range cctv {
		if roomName == protectedRoomName || !on {
			continue
		}
		if online[roomName] && roomHasCellOnGrid(g, roomName, grid) {
			list = append(list, consumer{roomName, "cctv"})
		}
	}
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].room < list[i].room || (list[j].room == list[i].room && list[j].kind == "doors" && list[i].kind == "cctv") {
				list[i], list[j] = list[j], list[i]
			}
		}
	}

	var shed []PowerShedEntry
	for _, c := range list {
		if consumption <= supply {
			break
		}
		if c.kind == "doors" && armed[c.room] {
			armed[c.room] = false
			online[c.room] = false
			if cctv[c.room] {
				cctv[c.room] = false
			}
			shed = append(shed, PowerShedEntry{Room: c.room, Kind: "doors"})
		} else if c.kind == "cctv" && cctv[c.room] {
			cctv[c.room] = false
			shed = append(shed, PowerShedEntry{Room: c.room, Kind: "cctv"})
		}
		consumption = calculateConsumptionFromMaps(g, online, cctv, grid)
	}
	return shed
}

// PreviewRoomPresetConsumption returns grid watt draw before, after apply, and after shedding.
func PreviewRoomPresetConsumption(g *state.Game, roomName string, doorsOn, cctvOn bool) (before, afterApply, afterShed int) {
	if g == nil {
		return 0, 0, 0
	}
	gridBefore := ArmedGridForRoom(g, roomName)
	if gridBefore.Size() == 0 {
		return 0, 0, 0
	}
	before = ConsumptionOnArmedGrid(g, gridBefore)

	doorsPreview := roomDoorsPoweredEffective(g, map[string]bool{roomName: doorsOn})
	grid := ArmedGridForRoomDoors(g, roomName, doorsPreview)
	if grid.Size() == 0 {
		return before, 0, 0
	}
	supply := ArmedGridSupply(g, grid)

	doors := make(map[string]bool, len(g.RoomDoorsPowered))
	cctv := make(map[string]bool, len(g.RoomCCTVPowered))
	online := make(map[string]bool)
	for k, v := range g.RoomDoorsPowered {
		doors[k] = v
	}
	for k, v := range g.RoomCCTVPowered {
		cctv[k] = v
	}
	if g.RoomPowerOnline != nil {
		for k, v := range g.RoomPowerOnline {
			online[k] = v
		}
	}
	doors[roomName] = doorsOn
	cctv[roomName] = cctvOn
	if doorsOn {
		online[roomName] = true
	} else {
		online[roomName] = false
	}

	afterApply = calculateConsumptionFromMaps(g, online, cctv, grid)
	afterShed = afterApply
	if afterShed <= supply {
		return before, afterApply, afterShed
	}

	type consumer struct{ room, kind string }
	var list []consumer
	for r, on := range online {
		if r == roomName || !on || !roomHasCellOnGrid(g, r, grid) {
			continue
		}
		list = append(list, consumer{r, "doors"})
	}
	for r, on := range cctv {
		if r == roomName || !on {
			continue
		}
		if online[r] && roomHasCellOnGrid(g, r, grid) {
			list = append(list, consumer{r, "cctv"})
		}
	}
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].room < list[i].room || (list[j].room == list[i].room && list[j].kind == "doors" && list[i].kind == "cctv") {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	for _, c := range list {
		if afterShed <= supply {
			break
		}
		if c.kind == "doors" && doors[c.room] {
			doors[c.room] = false
			online[c.room] = false
			if cctv[c.room] {
				cctv[c.room] = false
			}
		} else if c.kind == "cctv" && cctv[c.room] {
			cctv[c.room] = false
		}
		afterShed = calculateConsumptionFromMaps(g, online, cctv, grid)
	}
	return before, afterApply, afterShed
}

func calculateConsumptionFromMaps(g *state.Game, online, cctv map[string]bool, grid *mapset.Set[*world.Cell]) int {
	if g == nil || g.Grid == nil {
		return 0
	}
	rawConsumption := 0
	doorRoomCounted := make(map[string]bool)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Door != nil && online[data.Door.RoomName] && !doorRoomCounted[data.Door.RoomName] {
			if grid == nil || roomHasCellOnGrid(g, data.Door.RoomName, grid) {
				rawConsumption += 10
				doorRoomCounted[data.Door.RoomName] = true
			}
		}
		if grid != nil && !grid.Has(cell) {
			return
		}
		if data.Terminal != nil && cctv[cell.Name] && online[cell.Name] {
			rawConsumption += 10
		}
		if data.Puzzle != nil && data.Puzzle.IsSolved() {
			rawConsumption += 3
		}
	})
	params := deck.DecayParamsForDeck(g.CurrentDeckID)
	return int(float64(rawConsumption) * params.PowerCostMultiplier)
}

func roomHasCellOnGrid(g *state.Game, roomName string, grid *mapset.Set[*world.Cell]) bool {
	if g == nil || g.Grid == nil || roomName == "" || grid == nil {
		return false
	}
	found := false
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if found || cell == nil || cell.Name != roomName {
			return
		}
		if grid.Has(cell) {
			found = true
		}
	})
	return found
}
