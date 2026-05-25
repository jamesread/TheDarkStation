// Package devtools provides developer tools for testing and debugging.
package devtools

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

const mapDumpFilename = "map.txt"

// cellSymbol returns the single-character symbol for a cell (no player/exit overlay).
// If revealedOnly is true, non-revealed cells return '#'; otherwise they show their type.
func cellSymbol(g *state.Game, cell *world.Cell, revealedOnly bool) rune {
	if cell == nil {
		return '#'
	}
	if revealedOnly && !g.HasMap && !cell.Discovered {
		return '#'
	}
	if !cell.Room {
		return '#'
	}
	data := gameworld.GetGameData(cell)
	switch {
	case data.Door != nil:
		return 'D'
	case data.Generator != nil:
		return 'G'
	case data.Terminal != nil:
		return 'T'
	case data.Puzzle != nil:
		return 'P'
	case data.MaintenanceTerm != nil:
		return 'M'
	case data.Furniture != nil:
		return 'F'
	case data.Hazard != nil && data.Hazard.IsBlocking():
		return '!'
	case data.HazardControl != nil:
		return 'C'
	case cell.ItemsOnFloor.Size() > 0:
		return 'i'
	default:
		return '.'
	}
}

// writeMapGrid writes the grid to f with optional player/exit overlay.
func writeMapGrid(f *os.File, g *state.Game, rows, cols int, revealedOnly bool, playerRow, playerCol int) {
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			cell := g.Grid.GetCell(row, col)
			if row == playerRow && col == playerCol && g.CurrentCell != nil {
				fmt.Fprint(f, "@")
				continue
			}
			if cell != nil && cell.ExitCell {
				fmt.Fprint(f, "E")
				continue
			}
			if cell == nil {
				fmt.Fprint(f, "#")
				continue
			}
			fmt.Fprintf(f, "%c", cellSymbol(g, cell, revealedOnly))
		}
		fmt.Fprintln(f)
	}
}

// DumpRevealedMapToFile writes a full debug dump to map.txt: metadata, legend,
// revealed-only map, fully-revealed map, and detailed entity/hazard/item lists.
// Format is human- and LLM-readable (sections, key: value, consistent structure).
func DumpRevealedMapToFile(g *state.Game) (string, error) {
	if g.Grid == nil {
		return "", fmt.Errorf("no grid")
	}

	absPath, err := filepath.Abs(mapDumpFilename)
	if err != nil {
		return "", err
	}

	f, err := os.Create(absPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	rows := g.Grid.Rows()
	cols := g.Grid.Cols()
	playerRow, playerCol := -1, -1
	if g.CurrentCell != nil {
		playerRow, playerCol = g.CurrentCell.Row, g.CurrentCell.Col
	}
	startCell := g.Grid.StartCell()
	startRow, startCol := -1, -1
	if startCell != nil {
		startRow, startCol = startCell.Row, startCell.Col
	}

	// --- Metadata (seed, coordinates, power) ---
	fmt.Fprintln(f, "=== MAP DUMP DEBUG (level layout, routing, entities) ===")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "--- Metadata ---")
	fmt.Fprintf(f, "level: %d\n", g.Level)
	fmt.Fprintf(f, "level_seed: %d\n", g.LevelSeed)
	fmt.Fprintf(f, "grid_rows: %d\n", rows)
	fmt.Fprintf(f, "grid_cols: %d\n", cols)
	fmt.Fprintf(f, "coordinate_system: row,col (0-based, row=vertical, col=horizontal)\n")
	fmt.Fprintf(f, "player_row: %d\n", playerRow)
	fmt.Fprintf(f, "player_col: %d\n", playerCol)
	fmt.Fprintf(f, "player_cell: %d,%d\n", playerRow, playerCol)
	fmt.Fprintf(f, "start_cell: %d,%d\n", startRow, startCol)
	if startCell != nil && startCell.Name != "" {
		fmt.Fprintf(f, "start_room: %q\n", startCell.Name)
	}
	if g.CurrentCell != nil && g.CurrentCell.Name != "" {
		fmt.Fprintf(f, "player_room: %q\n", g.CurrentCell.Name)
	}
	fmt.Fprintf(f, "has_map: %v\n", g.HasMap)
	fmt.Fprintf(f, "power_supply: %d\n", g.PowerSupply)
	fmt.Fprintf(f, "power_consumption: %d\n", g.PowerConsumption)
	fmt.Fprintf(f, "power_available: %d\n", g.GetAvailablePower())
	fmt.Fprintf(f, "batteries_in_inventory: %d\n", g.Batteries)
	fmt.Fprintln(f, "")

	// --- Legend ---
	fmt.Fprintln(f, "--- Legend (cell symbols) ---")
	fmt.Fprintln(f, ". = walkable empty  # = unrevealed or wall  D = door  G = generator  T = CCTV terminal  P = puzzle terminal  M = maintenance terminal  F = furniture  ! = blocking hazard  C = hazard control  i = items on floor  @ = player  E = exit")
	fmt.Fprintln(f, "")

	// --- Map: Revealed only ---
	fmt.Fprintln(f, "--- Map (revealed cells only; unrevealed = #) ---")
	writeMapGrid(f, g, rows, cols, true, playerRow, playerCol)
	fmt.Fprintln(f, "")

	// --- Map: Fully revealed ---
	fmt.Fprintln(f, "--- Map (fully revealed; full layout) ---")
	writeMapGrid(f, g, rows, cols, false, playerRow, playerCol)
	fmt.Fprintln(f, "")

	// --- Solvability analysis ---
	fmt.Fprintln(f, "--- Solvability analysis (initial state) ---")
	report := setup.AnalyzeSolvability(g)
	fmt.Fprintf(f, "initial_reachable_cells: %d\n", report.InitialReachableCells)
	fmt.Fprintf(f, "initial_reachable_rooms: %q\n", report.InitialReachableRooms)
	fmt.Fprintf(f, "start_room_doors_powered: %v\n", report.StartRoomDoorsPowered)
	fmt.Fprintf(f, "start_maint_terminal_powered: %v\n", report.StartMaintPowered)
	fmt.Fprintf(f, "exit_reachable_at_init: %v\n", report.ExitReachableAtInit)
	if len(report.BlockedEgressDoors) == 0 {
		fmt.Fprintln(f, "blocked_egress_doors: (none — start pocket opens to all adjacent rooms)")
	} else {
		fmt.Fprintln(f, "blocked_egress_doors:")
		for _, door := range report.BlockedEgressDoors {
			controllable := setup.CanPowerRoomDoorsFromReachable(g, setup.InitialReachableCells(g), door.TargetRoom)
			fmt.Fprintf(f, "  row: %d col: %d from_room: %q target_room: %q remote_controllable: %v\n",
				door.Row, door.Col, door.FromRoom, door.TargetRoom, controllable)
		}
	}
	if len(report.Warnings) == 0 {
		fmt.Fprintln(f, "solvability_warnings: (none)")
	} else {
		fmt.Fprintln(f, "solvability_warnings:")
		for _, w := range report.Warnings {
			fmt.Fprintf(f, "  - %s\n", w)
		}
	}
	fmt.Fprintln(f, "")

	// --- Room adjacency (for maintenance control) ---
	fmt.Fprintln(f, "--- Room adjacency ---")
	var adjRoomNames []string
	adjSeen := make(map[string]bool)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name == "" || cell.Name == "Corridor" || adjSeen[cell.Name] {
			return
		}
		adjSeen[cell.Name] = true
		adjRoomNames = append(adjRoomNames, cell.Name)
	})
	sort.Strings(adjRoomNames)
	for _, rn := range adjRoomNames {
		adj := setup.GetAdjacentRoomNames(g.Grid, rn)
		filtered := make([]string, 0, len(adj))
		for _, a := range adj {
			if a != "Corridor" {
				filtered = append(filtered, a)
			}
		}
		sort.Strings(filtered)
		fmt.Fprintf(f, "  room: %q adjacent: %q doors_powered: %v\n", rn, filtered, g.RoomDoorsPowered[rn])
	}
	fmt.Fprintln(f, "")

	// --- Maintenance terminal selectable rooms ---
	fmt.Fprintln(f, "--- Maintenance terminal control scope ---")
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.MaintenanceTerm == nil {
			return
		}
		m := data.MaintenanceTerm
		selectable := setup.SelectableRoomsForTerminal(g, g.Grid, m.RoomName)
		fmt.Fprintf(f, "  row: %d col: %d room: %q powered: %v selectable_rooms: %q\n",
			row, col, m.RoomName, m.Powered, selectable)
	})
	fmt.Fprintln(f, "")

	// --- Player movement from current cell ---
	if g.CurrentCell != nil {
		fmt.Fprintln(f, "--- Player adjacent movement ---")
		for _, dir := range []struct {
			name string
			n    *world.Cell
		}{
			{"north", g.CurrentCell.North},
			{"south", g.CurrentCell.South},
			{"east", g.CurrentCell.East},
			{"west", g.CurrentCell.West},
		} {
			if dir.n == nil {
				fmt.Fprintf(f, "  %s: (no cell)\n", dir.name)
				continue
			}
			ok, reason := setup.CanEnterCellAtInit(g, dir.n)
			extra := ""
			if gameworld.HasDoor(dir.n) {
				d := gameworld.GetGameData(dir.n).Door
				extra = fmt.Sprintf(" door->%q locked=%v", d.RoomName, d.Locked)
			}
			if !ok {
				fmt.Fprintf(f, "  %s: row: %d col: %d room: %q blocked: %s%s\n",
					dir.name, dir.n.Row, dir.n.Col, dir.n.Name, reason, extra)
			} else {
				fmt.Fprintf(f, "  %s: row: %d col: %d room: %q passable%s\n",
					dir.name, dir.n.Row, dir.n.Col, dir.n.Name, extra)
			}
		}
		fmt.Fprintln(f, "")
	}

	// --- Entities: collect by type with coordinates and state ---
	fmt.Fprintln(f, "--- Entities (all with row,col and state) ---")

	// Doors
	fmt.Fprintln(f, "Doors:")
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Door == nil {
			return
		}
		d := data.Door
		blockReason := ""
		if ok, reason := setup.CanEnterCellAtInit(g, cell); !ok {
			blockReason = string(reason)
		}
		fmt.Fprintf(f, "  row: %d col: %d room_name: %q locked: %v keycard: %q doors_powered: %v init_passable: %v block_reason: %q\n",
			row, col, d.RoomName, d.Locked, d.KeycardName(), g.RoomDoorsPowered[d.RoomName], blockReason == "", blockReason)
	})
	fmt.Fprintln(f, "")

	// Generators
	fmt.Fprintln(f, "Generators:")
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Generator == nil {
			return
		}
		gen := data.Generator
		fmt.Fprintf(f, "  row: %d col: %d name: %q batteries_inserted: %d batteries_required: %d powered: %v\n", row, col, gen.Name, gen.BatteriesInserted, gen.BatteriesRequired, gen.IsPowered())
	})
	fmt.Fprintln(f, "")

	// CCTV Terminals
	fmt.Fprintln(f, "CCTV Terminals:")
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Terminal == nil {
			return
		}
		t := data.Terminal
		fmt.Fprintf(f, "  row: %d col: %d name: %q used: %v target_room: %q\n", row, col, t.Name, t.Used, t.TargetRoom)
	})
	fmt.Fprintln(f, "")

	// Puzzle Terminals
	fmt.Fprintln(f, "Puzzle Terminals:")
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Puzzle == nil {
			return
		}
		p := data.Puzzle
		rewardStr := "none"
		switch p.Reward {
		case entities.RewardKeycard:
			rewardStr = "keycard"
		case entities.RewardBattery:
			rewardStr = "battery"
		case entities.RewardRevealRoom:
			rewardStr = "reveal_room"
		case entities.RewardUnlockArea:
			rewardStr = "unlock_area"
		case entities.RewardMap:
			rewardStr = "map"
		}
		fmt.Fprintf(f, "  row: %d col: %d name: %q solved: %v solution: %q hint: %q reward: %s\n", row, col, p.Name, p.Solved, p.Solution, p.Hint, rewardStr)
	})
	fmt.Fprintln(f, "")

	// Maintenance Terminals (powered = only start room initially; accessible = reachable from start without locked doors)
	fmt.Fprintln(f, "Maintenance Terminals:")
	startRoomName := ""
	if startCell != nil && startCell.Name != "" {
		startRoomName = startCell.Name
	}
	poweredCount := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.MaintenanceTerm == nil {
			return
		}
		m := data.MaintenanceTerm
		if m.Powered {
			poweredCount++
		}
		inStartRoom := m.RoomName == startRoomName
		fmt.Fprintf(f, "  row: %d col: %d name: %q room_name: %q used: %v powered: %v (in_start_room: %v)\n", row, col, m.Name, m.RoomName, m.Used, m.Powered, inStartRoom)
	})
	fmt.Fprintf(f, "  (accessible powered terminals: %d - only start room terminals are powered at init)\n", poweredCount)
	fmt.Fprintln(f, "")

	// Furniture
	fmt.Fprintln(f, "Furniture:")
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Furniture == nil {
			return
		}
		furn := data.Furniture
		hasItem := furn.ContainedItem != nil
		itemName := ""
		if hasItem {
			itemName = furn.ContainedItem.Name
		}
		fmt.Fprintf(f, "  row: %d col: %d name: %q checked: %v has_contained_item: %v contained_item_name: %q description: %q\n", row, col, furn.Name, furn.Checked, hasItem, itemName, furn.Description)
	})
	fmt.Fprintln(f, "")

	// Hazards
	fmt.Fprintln(f, "Hazards:")
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Hazard == nil {
			return
		}
		h := data.Hazard
		info := entities.HazardTypes[h.Type]
		typeName := info.Name
		fmt.Fprintf(f, "  row: %d col: %d type: %s name: %q fixed: %v blocking: %v description: %q\n", row, col, typeName, h.Name, h.Fixed, h.IsBlocking(), h.Description)
	})
	fmt.Fprintln(f, "")

	// Hazard Controls
	fmt.Fprintln(f, "Hazard Controls:")
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.HazardControl == nil {
			return
		}
		c := data.HazardControl
		info := entities.HazardTypes[c.Type]
		hazardFixed := c.Hazard != nil && c.Hazard.Fixed
		fmt.Fprintf(f, "  row: %d col: %d name: %q type: %s activated: %v hazard_fixed: %v description: %q\n", row, col, c.Name, info.Name, c.Activated, hazardFixed, c.Description)
	})
	fmt.Fprintln(f, "")

	// Items on floor (per cell, list each item)
	fmt.Fprintln(f, "Items on floor:")
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || cell.ItemsOnFloor.Size() == 0 {
			return
		}
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			fmt.Fprintf(f, "  row: %d col: %d item_name: %q\n", row, col, item.Name)
		})
	})
	fmt.Fprintln(f, "")

	// Player inventory (owned items)
	fmt.Fprintln(f, "Player inventory (owned items):")
	if g.OwnedItems.Size() == 0 {
		fmt.Fprintln(f, "  (none)")
	} else {
		var names []string
		g.OwnedItems.Each(func(item *world.Item) {
			names = append(names, item.Name)
		})
		sort.Strings(names)
		for _, n := range names {
			fmt.Fprintf(f, "  item_name: %q\n", n)
		}
	}
	fmt.Fprintln(f, "")

	// Room power (doors, CCTV, lights per room)
	fmt.Fprintln(f, "Room power state:")
	var roomNames []string
	for rn := range g.RoomDoorsPowered {
		roomNames = append(roomNames, rn)
	}
	sort.Strings(roomNames)
	for _, rn := range roomNames {
		doorsOn := g.RoomDoorsPowered[rn]
		cctvOn := g.RoomCCTVPowered[rn]
		lightsOn := g.RoomLightsPowered[rn]
		if _, ok := g.RoomLightsPowered[rn]; !ok {
			lightsOn = true // default when not set
		}
		fmt.Fprintf(f, "  room: %q doors_powered: %v cctv_powered: %v lights_powered: %v\n", rn, doorsOn, cctvOn, lightsOn)
	}
	fmt.Fprintln(f, "")

	// Exit cell
	fmt.Fprintln(f, "Exit cell:")
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && cell.ExitCell {
			fmt.Fprintf(f, "  row: %d col: %d\n", row, col)
		}
	})
	fmt.Fprintln(f, "")

	fmt.Fprintln(f, "=== END MAP DUMP ===")

	if err := f.Sync(); err != nil {
		return absPath, err
	}
	return absPath, nil
}
