// Package devtools provides developer tools for testing and debugging.
package devtools

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
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
		fmt.Fprintf(f, "  row: %d col: %d room_name: %q locked: %v keycard: %q\n", row, col, d.RoomName, d.Locked, d.KeycardName())
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

	// Maintenance Terminals
	fmt.Fprintln(f, "Maintenance Terminals:")
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.MaintenanceTerm == nil {
			return
		}
		m := data.MaintenanceTerm
		fmt.Fprintf(f, "  row: %d col: %d name: %q room_name: %q used: %v\n", row, col, m.Name, m.RoomName, m.Used)
	})
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

	// Room power (doors and CCTV per room)
	fmt.Fprintln(f, "Room power state:")
	var roomNames []string
	for rn := range g.RoomDoorsPowered {
		roomNames = append(roomNames, rn)
	}
	sort.Strings(roomNames)
	for _, rn := range roomNames {
		doorsOn := g.RoomDoorsPowered[rn]
		cctvOn := g.RoomCCTVPowered[rn]
		fmt.Fprintf(f, "  room: %q doors_powered: %v cctv_powered: %v\n", rn, doorsOn, cctvOn)
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
