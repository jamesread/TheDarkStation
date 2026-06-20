package devtools

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// PerfMapScenarios lists deterministic rendering stress maps for bisecting FPS drops.
var PerfMapScenarios = []string{
	"checker",
	"entities",
	"entities_controls",
	"entities_doors",
	"entities_furniture",
	"entities_generators",
	"entities_hazards",
	"entities_items",
	"entities_relays",
	"entities_repairs",
	"entities_terminals",
	"labels",
	"mixed",
	"open",
	"walls",
}

// PerfMapLevel marks g.Level while a console perfmap scenario is active.
const PerfMapLevel = 997

// SwitchToPerfMap loads a deterministic performance test map. Scenarios isolate one
// rendering variable at a time while keeping the map fully revealed.
func SwitchToPerfMap(g *state.Game, scenario string) string {
	if g == nil {
		return ""
	}
	scenario = normalizePerfMapScenario(scenario)
	grid := newPerfGrid(72, 112)
	resetPerfGameState(g, grid)

	switch scenario {
	case "open":
		fillOpenPerfMap(grid, "Perf Open Floor")
	case "walls":
		fillWallGridPerfMap(grid)
	case "checker":
		fillCheckerPerfMap(grid)
	case "entities":
		fillOpenPerfMap(grid, "Perf Entity Floor")
		stampDenseEntities(g, grid)
	case "entities_controls", "entities_doors", "entities_furniture", "entities_generators",
		"entities_hazards", "entities_items", "entities_relays", "entities_repairs", "entities_terminals":
		fillOpenPerfMap(grid, "Perf Entity Floor")
		stampEntityFamily(g, grid, strings.TrimPrefix(scenario, "entities_"))
	case "labels":
		fillLabelPerfMap(grid)
	case "mixed":
		fillMixedPerfMap(g, grid)
	}

	finalizePerfMap(g, grid, scenario)
	logMessage(g, "Loaded performance test map: ITEM{%s}", scenario)
	logMessage(g, "Use console ITEM{perfmap list} for scenarios; compare FPS at the same zoom.")
	return scenario
}

func normalizePerfMapScenario(scenario string) string {
	scenario = strings.ToLower(strings.TrimSpace(scenario))
	if scenario == "" {
		return "open"
	}
	for _, name := range PerfMapScenarios {
		if scenario == name {
			return scenario
		}
	}
	return "open"
}

func PerfMapScenarioList() string {
	names := append([]string(nil), PerfMapScenarios...)
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func newPerfGrid(rows, cols int) *world.Grid {
	grid := world.NewGrid(rows, cols)
	grid.BuildAllCellConnections()
	return grid
}

func resetPerfGameState(g *state.Game, grid *world.Grid) {
	g.Grid = grid
	g.CurrentDeckID = deck.TotalDecks
	g.Level = PerfMapLevel
	g.PerfMapScenario = ""
	g.LevelSeed = 0
	g.OwnedItems = mapset.New[*world.Item]()
	g.Generators = make([]*entities.Generator, 0)
	g.RepairObjectives = make([]*entities.RepairObjective, 0)
	g.FoundCodes = make(map[string]bool)
	g.Hints = nil
	g.HasMap = true
	g.Batteries = 0
	g.PowerSupply = 0
	g.PowerConsumption = 0
	g.PowerOverloadWarned = false
	g.RoomDoorsPowered = make(map[string]bool)
	g.RoomCCTVPowered = make(map[string]bool)
	g.RoomLightsPowered = make(map[string]bool)
	g.RoomPowerOnline = make(map[string]bool)
	g.ManualEgressReleased = make(map[string]bool)
	g.PowerPropPending = nil
	g.RoomPowerOffPending = nil
	g.GeneratorShutdownAt = 0
	g.GeneratorShutdownRow = -1
	g.GeneratorShutdownCol = -1
	g.GeneratorShutdownRoomName = ""
	g.LongUse = nil
	g.HazardClear = nil
	g.HazardTour = nil
	g.ClearMessages()
}

func revealCell(cell *world.Cell, room bool, name string) {
	if cell == nil {
		return
	}
	cell.Room = room
	cell.Name = name
	cell.Description = ""
	cell.Discovered = true
	cell.Visited = true
	data := gameworld.InitGameData(cell)
	data.LightsOn = true
	data.Lighted = true
}

func fillOpenPerfMap(grid *world.Grid, name string) {
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		revealCell(cell, true, name)
	})
}

func fillWallGridPerfMap(grid *world.Grid) {
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		wall := row == 0 || col == 0 || row == grid.Rows()-1 || col == grid.Cols()-1 ||
			(row%7 == 0 && col%5 != 2) || (col%11 == 0 && row%6 != 3)
		if wall {
			revealCell(cell, false, "Perf Wall")
			return
		}
		revealCell(cell, true, "Perf Wall Corridors")
	})
}

func fillCheckerPerfMap(grid *world.Grid) {
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		wall := row == 0 || col == 0 || row == grid.Rows()-1 || col == grid.Cols()-1 ||
			((row+col)%2 == 0 && row%5 != 0 && col%7 != 0)
		if wall {
			revealCell(cell, false, "Perf Checker Wall")
			return
		}
		revealCell(cell, true, "Perf Checker Floor")
	})
}

func fillLabelPerfMap(grid *world.Grid) {
	const roomH, roomW = 2, 4
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		roomName := fmt.Sprintf("Perf Room %02d-%02d", row/roomH, col/roomW)
		revealCell(cell, true, roomName)
	})
}

func fillMixedPerfMap(g *state.Game, grid *world.Grid) {
	fillWallGridPerfMap(grid)
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		if row%9 == 2 && col%13 == 2 {
			cell.Name = fmt.Sprintf("Perf Mixed %02d-%02d", row/9, col/13)
		}
	})
	stampDenseEntities(g, grid)
}

func stampDenseEntities(g *state.Game, grid *world.Grid) {
	hazardTypes := []entities.HazardType{
		entities.HazardVacuum,
		entities.HazardCoolant,
		entities.HazardElectrical,
		entities.HazardGas,
		entities.HazardRadiation,
	}
	idx := 0
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if !perfEntityCell(row, col, grid) || cell == nil || !cell.Room {
			return
		}
		data := gameworld.GetGameData(cell)
		switch idx % 12 {
		case 0:
			data.Door = entities.NewDoor(fmt.Sprintf("Perf Door %d", idx))
		case 1:
			data.Door = entities.NewUnlockedDoor(fmt.Sprintf("Perf Door %d", idx))
		case 2:
			data.Generator = newPerfPoweredGenerator(fmt.Sprintf("Perf Generator %d", idx))
		case 3:
			data.Terminal = entities.NewCCTVTerminal(fmt.Sprintf("Perf CCTV %d", idx))
		case 4:
			mt := entities.NewMaintenanceTerminal(fmt.Sprintf("Perf Maint %d", idx), cell.Name)
			data.MaintenanceTerm = mt
		case 5:
			data.Furniture = entities.NewFurniture("Perf Locker", "Dense entity stress furniture.", "L")
		case 6:
			data.Hazard = entities.NewHazard(hazardTypes[idx%len(hazardTypes)])
		case 7:
			hazard := entities.NewHazard(hazardTypes[idx%len(hazardTypes)])
			data.HazardControl = entities.NewHazardControl(hazard.Type, hazard)
		case 8:
			cell.ItemsOnFloor.Put(world.NewItem("Battery"))
		case 9:
			cell.ItemsOnFloor.Put(world.NewItem("Perf Keycard"))
		case 10:
			relay := entities.NewPowerRelay()
			if idx%2 == 0 {
				relay = entities.NewPowerRelayOpen()
			}
			data.PowerRelay = relay
		case 11:
			repair := entities.NewRepairObjective(fmt.Sprintf("perf-repair-%d", idx), entities.RepairPressureValve, cell.Name, row, col)
			data.RepairDevice = repair
		}
		idx++
	})
}

func stampEntityFamily(g *state.Game, grid *world.Grid, family string) {
	idx := 0
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if !perfEntityCell(row, col, grid) || cell == nil || !cell.Room {
			return
		}
		data := gameworld.GetGameData(cell)
		switch family {
		case "controls":
			hazard := entities.NewHazard(entities.HazardGas)
			data.HazardControl = entities.NewHazardControl(hazard.Type, hazard)
		case "doors":
			if idx%2 == 0 {
				data.Door = entities.NewDoor(fmt.Sprintf("Perf Door %d", idx))
			} else {
				data.Door = entities.NewUnlockedDoor(fmt.Sprintf("Perf Door %d", idx))
			}
		case "furniture":
			data.Furniture = entities.NewFurniture("Perf Locker", "Dense furniture stress.", "L")
		case "generators":
			data.Generator = newPerfPoweredGenerator(fmt.Sprintf("Perf Generator %d", idx))
		case "hazards":
			data.Hazard = entities.NewHazard(entities.HazardGas)
		case "items":
			if idx%2 == 0 {
				cell.ItemsOnFloor.Put(world.NewItem("Battery"))
			} else {
				cell.ItemsOnFloor.Put(world.NewItem("Perf Keycard"))
			}
		case "relays":
			if idx%2 == 0 {
				data.PowerRelay = entities.NewPowerRelay()
			} else {
				data.PowerRelay = entities.NewPowerRelayOpen()
			}
		case "repairs":
			data.RepairDevice = entities.NewRepairObjective(fmt.Sprintf("perf-repair-%d", idx), entities.RepairPressureValve, cell.Name, row, col)
		case "terminals":
			if idx%2 == 0 {
				data.Terminal = entities.NewCCTVTerminal(fmt.Sprintf("Perf CCTV %d", idx))
			} else {
				mt := entities.NewMaintenanceTerminal(fmt.Sprintf("Perf Maint %d", idx), cell.Name)
				data.MaintenanceTerm = mt
			}
		}
		idx++
	})
}

func perfEntityCell(row, col int, grid *world.Grid) bool {
	if grid == nil || row < 2 || col < 2 || row > grid.Rows()-3 || col > grid.Cols()-3 {
		return false
	}
	diag := row - col
	if diag%3 != 0 {
		return false
	}
	linePos := (row + col) / 3
	return linePos%6 != 5
}

func finalizePerfMap(g *state.Game, grid *world.Grid, scenario string) {
	start := firstRoomCell(grid)
	if start != nil {
		grid.SetStartCell(start)
		g.CurrentCell = start
	}
	exit := lastRoomCell(grid)
	if exit != nil {
		exit.ExitCell = true
		exit.Locked = false
		grid.SetExitCell(exit)
	}
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name == "" {
			return
		}
		g.RoomDoorsPowered[cell.Name] = true
		g.RoomCCTVPowered[cell.Name] = true
		g.RoomLightsPowered[cell.Name] = true
		g.RoomPowerOnline[cell.Name] = true
	})
	g.RebuildGeneratorsFromGrid()
	setup.NotifyPowerGridChanged(g)
	g.PerfMapScenario = scenario
	g.AddHint("Performance map: " + scenario)
}

func newPerfPoweredGenerator(name string) *entities.Generator {
	gen := entities.NewGenerator(name, 1)
	gen.InsertBatteriesAndStart(1)
	return gen
}

func firstRoomCell(grid *world.Grid) *world.Cell {
	var out *world.Cell
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if out == nil && cell != nil && cell.Room {
			out = cell
		}
	})
	return out
}

func lastRoomCell(grid *world.Grid) *world.Cell {
	var out *world.Cell
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.Room {
			out = cell
		}
	})
	return out
}
