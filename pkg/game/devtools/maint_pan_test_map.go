package devtools

import (
	"fmt"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// MaintPanTestLevel identifies the static maintenance pan regression layout (debug only).
const MaintPanTestLevel = 998

// Room names for SwitchToMaintPanTestMap (stable for tests and room-picker UI).
const (
	RoomPanTestWest   = "PanTest West"
	RoomPanTestCenter = "PanTest Center"
	RoomPanTestEast   = "PanTest East"
)

// SwitchToMaintPanTestMap loads a small fixed layout for reproducing and eyeballing the
// maintenance room-picker map camera tween: three wide rooms separated by 1-cell corridors,
// with room centers far apart on the same row band.
//
// Open the developer menu (F9), then choose Developer test map; or use console maint_pan_test.
func SwitchToMaintPanTestMap(g *state.Game) {
	if g == nil {
		return
	}

	const rows, cols = 24, 44
	grid := world.NewGrid(rows, cols)
	grid.BuildAllCellConnections()

	// Default: walkable placeholder (same pattern as SwitchToDevMap).
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		cell.Room = true
		cell.Name = "PanTest Void"
		cell.Description = ""
		cell.Discovered = true
		cell.Visited = true
	})

	stampNamed := func(r0, r1, c0, c1 int, name string) {
		for r := r0; r <= r1; r++ {
			for c := c0; c <= c1; c++ {
				cell := grid.GetCell(r, c)
				if cell == nil {
					continue
				}
				cell.Room = true
				cell.Name = name
				cell.Description = ""
				cell.Discovered = true
				cell.Visited = true
			}
		}
	}

	stampCorridor := func(r0, r1, c0, c1 int) {
		for r := r0; r <= r1; r++ {
			for c := c0; c <= c1; c++ {
				cell := grid.GetCell(r, c)
				if cell == nil {
					continue
				}
				cell.Room = true
				cell.Name = "Corridor"
				cell.Description = "ROOM_CORRIDOR"
				cell.Discovered = true
				cell.Visited = true
			}
		}
	}

	// Horizontal chain: West | C | Center | C | East (room centers spaced ~14 cols apart).
	const r0, r1 = 10, 12
	stampNamed(r0, r1, 1, 6, RoomPanTestWest)
	stampCorridor(r0, r1, 7, 7)
	stampNamed(r0, r1, 8, 12, RoomPanTestCenter)
	stampCorridor(r0, r1, 13, 13)
	stampNamed(r0, r1, 14, 20, RoomPanTestEast)

	termRow, termCol := 11, 12
	if termCell := grid.GetCell(termRow, termCol); termCell != nil {
		data := gameworld.InitGameData(termCell)
		data.MaintenanceTerm = entities.NewMaintenanceTerminal("PanTest Maint", RoomPanTestCenter)
	}

	// Player west of the terminal (Interact east).
	start := grid.GetCell(11, 11)
	if start == nil {
		return
	}
	grid.SetStartCell(start)

	g.Grid = grid
	g.CurrentCell = start
	g.CurrentDeckID = deck.TotalDecks
	g.Level = MaintPanTestLevel
	g.HasMap = true
	g.MaintenanceMenuRoom = ""
	g.Batteries = 0
	g.FoundCodes = make(map[string]bool)
	g.PowerSupply = 0
	g.PowerConsumption = 0
	g.PowerOverloadWarned = false
	g.OwnedItems = mapset.New[*world.Item]()
	g.ResetObservationCueAnnounced()
	g.ResetLinkageTokensSeen()

	setup.InitRoomPower(g)
	setup.InitMaintenanceTerminalPower(g)

	// Synthetic bench generators (no map tiles): same purpose as SwitchToDevMap — comfortable supply for tests.
	g.Generators = nil
	for i := 1; i <= 5; i++ {
		gen := entities.NewGenerator(fmt.Sprintf("PanTest Bench Gen %d", i), 1)
		gen.BatteriesInserted = gen.BatteriesRequired
		g.AddGenerator(gen)
	}
	g.UpdatePowerSupply()
	g.PowerConsumption = g.CalculatePowerConsumption()

	g.ClearMessages()
	logMessage(g, "Static maintenance pan test map — console ITEM{maint_pan_test} again to reload.")
	logMessage(g, "Interact (E) east onto the terminal, then Viewing room → Select room and switch %s / %s.",
		RoomPanTestWest, RoomPanTestEast)
}
