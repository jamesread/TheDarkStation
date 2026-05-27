// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
package ebiten

import (
	"image/color"
	"strings"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/features"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// getCellRenderOptions returns rendering options for a cell.
// When forUnderfoot is true, the cell is treated as if the player were not on it (used to draw floor under the player).
func (e *EbitenRenderer) getCellRenderOptions(g *state.Game, cell *world.Cell, snap *renderSnapshot, forUnderfoot bool) CellRenderOptions {
	if cell == nil {
		return CellRenderOptions{Icon: IconVoid, Color: colorBackground, HasBackground: false}
	}

	// Player position - use snapshot coordinates for consistency (unless we want underfoot options)
	if !forUnderfoot && cell.Row == snap.playerRow && cell.Col == snap.playerCol {
		return CellRenderOptions{Icon: snap.playerFacing.Icon(), Color: colorPlayer, HasBackground: false}
	}

	// Get game-specific data for this cell
	data := gameworld.GetGameData(cell)

	// Hazard (show if has map or discovered)
	if gameworld.HasHazard(cell) && (g.HasMap || cell.Discovered) {
		if data.Hazard.IsBlocking() {
			return CellRenderOptions{Icon: data.Hazard.GetIcon(), Color: colorHazard, HasBackground: true}
		}
	}

	// Hazard Control (show if has map or discovered)
	if gameworld.HasHazardControl(cell) && (g.HasMap || cell.Discovered) {
		if !data.HazardControl.Activated {
			return CellRenderOptions{Icon: entities.GetControlIcon(data.HazardControl.Type), Color: colorHazardCtrl, HasBackground: true}
		}
		return CellRenderOptions{Icon: entities.GetControlIcon(data.HazardControl.Type), Color: colorSubtle, HasBackground: false}
	}

	// Door (show if has map or discovered)
	if gameworld.HasDoor(cell) && (g.HasMap || cell.Discovered) {
		roomName := data.Door.RoomName
		if !snapCellHasLivePower(snap, cell) {
			if snapRoomManualEgressReleased(snap, roomName) {
				return CellRenderOptions{Icon: IconDoorUnlocked, Color: colorDoorLocked, HasBackground: true}
			}
			// Unpowered: use hazard color (matches UNPOWERED{} markup)
			return CellRenderOptions{Icon: IconDoorUnlocked, Color: colorHazard, HasBackground: true}
		}
		if data.Door.Locked {
			return CellRenderOptions{Icon: IconDoorLocked, Color: colorDoorLocked, HasBackground: true}
		}
		return CellRenderOptions{Icon: IconDoorUnlocked, Color: colorDoorUnlocked, HasBackground: true}
	}

	// Generator (show if has map or discovered)
	if gameworld.HasGenerator(cell) && (g.HasMap || cell.Discovered) {
		if data.Generator.IsPowered() {
			return CellRenderOptions{Icon: IconGeneratorPowered, Color: colorGeneratorOn, HasBackground: true}
		}
		return CellRenderOptions{Icon: IconGeneratorUnpowered, Color: colorGeneratorOff, HasBackground: true}
	}

	// Maintenance Terminal (show if has map or discovered) - same visibility as other cells
	if gameworld.HasMaintenanceTerminal(cell) && (g.HasMap || cell.Discovered) {
		return CellRenderOptions{Icon: IconMaintenance, Color: colorMaintenance, HasBackground: true, BackgroundColor: colorMaintenanceBg}
	}

	// CCTV Terminal (show if has map or discovered) - same orange as maintenance terminals
	if gameworld.HasTerminal(cell) && (g.HasMap || cell.Discovered) {
		if data.Terminal.IsUsed() {
			return CellRenderOptions{Icon: IconTerminalUsed, Color: colorTerminalUsed, HasBackground: false}
		}
		return CellRenderOptions{Icon: IconTerminalUnused, Color: colorMaintenance, HasBackground: true, BackgroundColor: colorMaintenanceBg}
	}

	// Puzzle Terminal (show if has map or discovered)
	if gameworld.HasPuzzle(cell) && (g.HasMap || cell.Discovered) {
		if data.Puzzle.IsSolved() {
			return CellRenderOptions{Icon: IconTerminalUsed, Color: colorTerminalUsed, HasBackground: false}
		}
		return CellRenderOptions{Icon: IconTerminalUnused, Color: colorTerminal, HasBackground: true}
	}

	// Furniture (show if has map or discovered)
	if gameworld.HasFurniture(cell) && (g.HasMap || cell.Discovered) {
		if data.Furniture.IsChecked() {
			return CellRenderOptions{Icon: data.Furniture.Icon, Color: colorFurnitureCheck, HasBackground: false}
		}
		return CellRenderOptions{Icon: data.Furniture.Icon, Color: colorFurniture, HasBackground: true}
	}

	// Exit cell (show if has map or discovered)
	if cell.ExitCell && (g.HasMap || cell.Discovered) {
		if cell.Locked && !g.AllGeneratorsPowered() {
			return CellRenderOptions{Icon: IconExitLocked, Color: colorExitLocked, HasBackground: true}
		}
		// Unlocked exit - apply continuous pulsing animation for icon
		pulseColor := e.getPulsingExitColor()
		// Background will be drawn with pulsing color separately
		return CellRenderOptions{Icon: IconExitUnlocked, Color: pulseColor, HasBackground: true}
	}

	// Corridor power relay (discovered corridors only)
	if gameworld.HasPowerRelay(cell) && (g.HasMap || cell.Discovered) {
		relay := gameworld.GetGameData(cell).PowerRelay
		if relay != nil && relay.Closed {
			return CellRenderOptions{Icon: IconRelayClosed, Color: colorMaintenance, HasBackground: true, BackgroundColor: colorWallBgPowered}
		}
		return CellRenderOptions{Icon: IconRelayOpen, Color: colorHazard, HasBackground: true, BackgroundColor: colorHazardBackground}
	}

	// Items on floor (show if has map or discovered)
	if cell.ItemsOnFloor.Size() > 0 && (g.HasMap || cell.Discovered) {
		if cellHasKeycard(cell) {
			return CellRenderOptions{Icon: IconKey, Color: colorKeycard, HasBackground: true}
		}
		if cellHasBattery(cell) {
			return CellRenderOptions{Icon: IconBattery, Color: colorBattery, HasBackground: true}
		}
		if cellHasMapItem(cell) {
			return CellRenderOptions{Icon: IconMap, Color: colorItem, HasBackground: true}
		}
		return CellRenderOptions{Icon: IconItem, Color: colorItem, HasBackground: true}
	}

	// Visited rooms (when gameplay.visited cvar is enabled)
	if features.IsVisited(cell) {
		return CellRenderOptions{Icon: getFloorIcon(cell.Name, true), Color: colorFloorVisited, HasBackground: true, BackgroundColor: colorFloorVisitedBg}
	}

	// Discovered but not visited
	if cell.Discovered {
		if cell.Room {
			return CellRenderOptions{Icon: getFloorIcon(cell.Name, false), Color: colorFloor, HasBackground: true, BackgroundColor: colorFloorBg}
		}
		return CellRenderOptions{Icon: IconWall, Color: colorWall, HasBackground: true} // Walls get background
	}

	// Has map - show rooms faintly
	if g.HasMap && cell.Room {
		return CellRenderOptions{Icon: getFloorIcon(cell.Name, false), Color: colorSubtle, HasBackground: true, BackgroundColor: colorFloorBg}
	}

	// Non-room cells adjacent to discovered/visited rooms render as walls
	if !cell.Room && hasAdjacentDiscoveredRoom(cell) {
		return CellRenderOptions{Icon: IconWall, Color: colorWall, HasBackground: true} // Walls get background
	}

	// Unknown/void
	return CellRenderOptions{Icon: IconVoid, Color: colorBackground, HasBackground: false}
}

// getFloorIcon returns the appropriate floor icon for a room
func getFloorIcon(roomName string, visited bool) string {
	for baseRoom, icons := range roomFloorIcons {
		if strings.Contains(roomName, baseRoom) {
			if visited {
				return icons[0]
			}
			return icons[1]
		}
	}
	if visited {
		return IconVisited
	}
	return IconUnvisited
}

// cellHasMapItem checks if a cell has the station map item on the floor.
func cellHasMapItem(c *world.Cell) bool {
	found := false
	c.ItemsOnFloor.Each(func(item *world.Item) {
		if item.Name == "Map" {
			found = true
		}
	})
	return found
}

// cellHasKeycard checks if a cell has a keycard item
func cellHasKeycard(c *world.Cell) bool {
	hasKeycard := false
	c.ItemsOnFloor.Each(func(item *world.Item) {
		if strings.Contains(strings.ToLower(item.Name), "keycard") {
			hasKeycard = true
		}
	})
	return hasKeycard
}

// cellHasBattery checks if a cell has a battery item
func cellHasBattery(c *world.Cell) bool {
	hasBattery := false
	c.ItemsOnFloor.Each(func(item *world.Item) {
		if strings.Contains(strings.ToLower(item.Name), "battery") {
			hasBattery = true
		}
	})
	return hasBattery
}

// hasAdjacentDiscoveredRoom checks if any adjacent cell is discovered
func hasAdjacentDiscoveredRoom(c *world.Cell) bool {
	neighbors := []*world.Cell{c.North, c.East, c.South, c.West}
	for _, n := range neighbors {
		if n != nil && n.Room && (n.Discovered || features.IsVisited(n)) {
			return true
		}
	}
	return false
}

// roomCenter returns the approximate center (row, col) of a room by averaging all
// room cells with the given name. Returns false if the room is not found or empty.
func roomCenter(grid *world.Grid, roomName string) (row, col int, ok bool) {
	if grid == nil || roomName == "" {
		return 0, 0, false
	}
	var sumRow, sumCol int
	var count int
	grid.ForEachCell(func(r, c int, cell *world.Cell) {
		if cell != nil && cell.Room && cell.Name == roomName {
			sumRow += r
			sumCol += c
			count++
		}
	})
	if count == 0 {
		return 0, 0, false
	}
	return sumRow / count, sumCol / count, true
}

// hasAdjacentRoomNamed checks if any adjacent cell belongs to the given room
func hasAdjacentRoomNamed(c *world.Cell, roomName string) bool {
	if c == nil || roomName == "" {
		return false
	}
	neighbors := []*world.Cell{c.North, c.East, c.South, c.West}
	for _, n := range neighbors {
		if n != nil && n.Room && n.Name == roomName {
			return true
		}
	}
	return false
}

// maintSelectableRoomWallBg tints walls for adjacent selectable rooms while the maintenance menu is open.
func maintSelectableRoomWallBg(snap *renderSnapshot, cell *world.Cell) color.Color {
	menuRoom := snapMaintenanceMenuRoom(snap)
	if snap == nil || cell == nil || menuRoom == "" || len(snap.mapPower.maintenanceSelectableRooms) == 0 {
		return nil
	}
	for _, room := range snap.mapPower.maintenanceSelectableRooms {
		if room == menuRoom {
			continue
		}
		if !hasAdjacentRoomNamed(cell, room) {
			continue
		}
		doors := snap.mapPower.roomDoorsPowered != nil && snap.mapPower.roomDoorsPowered[room]
		cctv := snap.mapPower.roomCCTVPowered != nil && snap.mapPower.roomCCTVPowered[room]
		switch {
		case doors && cctv:
			return color.RGBA{35, 90, 55, 255}
		case doors:
			return color.RGBA{40, 70, 48, 255}
		default:
			return color.RGBA{48, 44, 58, 255}
		}
	}
	return nil
}

// roomHasPower checks if the room adjacent to a wall cell has power
func (e *EbitenRenderer) roomHasPower(g *state.Game, wallCell *world.Cell) bool {
	if wallCell == nil || g == nil || g.Grid == nil {
		return false
	}

	// Check if there's available power (power supply > consumption)
	availablePower := g.GetAvailablePower()
	if availablePower <= 0 {
		return false
	}

	// If there's available power, check if any adjacent room exists
	// (if there's power, rooms should be considered powered)
	neighbors := []*world.Cell{wallCell.North, wallCell.East, wallCell.South, wallCell.West}
	for _, neighbor := range neighbors {
		if neighbor != nil && neighbor.Room {
			// Room exists and there's available power - room is powered
			return true
		}
	}
	return false
}
