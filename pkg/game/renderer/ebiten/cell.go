// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
package ebiten

import (
	"image/color"
	"strings"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/features"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

var floorIconCache = make(map[string]string)

// cellKnowledge is what the player currently knows about a cell (information economy).
type cellKnowledge int

const (
	// knowledgeUnknown: never seen, no map — not drawn.
	knowledgeUnknown cellKnowledge = iota
	// knowledgeLayout: floor plan only (discovered while dark, or revealed by the Map
	// item) — architecture is drawn, contents are not.
	knowledgeLayout
	// knowledgeRemembered: seen illuminated before — entity identity is drawn dim,
	// live state (power, lock, charge) is not.
	knowledgeRemembered
	// knowledgeLive: currently illuminated (powered room lights or headlamp) — full detail.
	knowledgeLive
)

// cellKnowledgeTier classifies a room cell's knowledge tier for rendering.
func cellKnowledgeTier(g *state.Game, cell *world.Cell) cellKnowledge {
	if cell == nil || (!cell.Discovered && !g.HasMap) {
		return knowledgeUnknown
	}
	if cell.Discovered {
		data := gameworld.GetGameData(cell)
		if data.LightsOn {
			return knowledgeLive
		}
		if data.Lighted {
			return knowledgeRemembered
		}
	}
	return knowledgeLayout
}

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

	// Walls (non-room cells) have no internal state; show next to any known room,
	// or when the maintenance menu is focusing a room (full outline including unseen walls).
	if !cell.Room {
		if shouldRenderWallCell(cell, snap) {
			if opts, ok := shipWallRenderOptions(cell); ok {
				return opts
			}
			return CellRenderOptions{Icon: IconWall, Color: colorWall, HasBackground: true}
		}
		return CellRenderOptions{Icon: IconVoid, Color: colorBackground, HasBackground: false}
	}

	switch cellKnowledgeTier(g, cell) {
	case knowledgeLive:
		return e.liveCellRenderOptions(g, cell, snap)
	case knowledgeRemembered:
		return e.rememberedCellRenderOptions(g, cell, snap)
	case knowledgeLayout:
		return layoutCellRenderOptions(cell)
	default:
		return CellRenderOptions{Icon: IconVoid, Color: colorBackground, HasBackground: false}
	}
}

// liveCellRenderOptions renders a currently illuminated room cell at full detail.
func (e *EbitenRenderer) liveCellRenderOptions(g *state.Game, cell *world.Cell, snap *renderSnapshot) CellRenderOptions {
	data := gameworld.GetGameData(cell)

	// Hazard
	if gameworld.HasHazard(cell) {
		if data.Hazard.IsBlocking() {
			var iconColor color.Color = colorHazard
			if alpha := hazardClearVisualAlpha(snap, cell); alpha < 1 {
				iconColor = e.applyAlpha(colorHazard, alpha)
			}
			return CellRenderOptions{Icon: data.Hazard.GetIcon(), Color: iconColor, HasBackground: true}
		}
	}

	// Hazard Control
	if gameworld.HasHazardControl(cell) {
		if !data.HazardControl.Activated {
			return CellRenderOptions{Icon: entities.GetControlIcon(data.HazardControl.Type), Color: colorHazardCtrl, HasBackground: true}
		}
		return CellRenderOptions{Icon: entities.GetControlIcon(data.HazardControl.Type), Color: colorSubtle, HasBackground: false}
	}

	if gameworld.HasRepairBlocker(cell) {
		repair := data.RepairBlocker
		if repair != nil && repair.BlockerBlocksCell(cell.Row, cell.Col) {
			return CellRenderOptions{Icon: IconToxicSlime, Color: colorToxicSlime, HasBackground: true, BackgroundColor: colorToxicSlimeBg}
		}
		// Drained cells play a pop overlay; keep the floor tile clean underneath.
		return CellRenderOptions{Icon: "", Color: colorSubtle, HasBackground: false}
	}

	// Door
	if gameworld.HasDoor(cell) {
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
		// Passable doorway: plate darker than the surrounding walls so it reads as an opening.
		return CellRenderOptions{Icon: IconDoorUnlocked, Color: colorDoorUnlocked, HasBackground: true, BackgroundColor: colorDoorBg}
	}

	// Generator
	if gameworld.HasGenerator(cell) {
		if data.Generator.IsPowered() {
			return CellRenderOptions{Icon: IconGeneratorPowered, Color: colorGeneratorOn, HasBackground: true, BackgroundColor: colorGeneratorFocusBg}
		}
		return CellRenderOptions{Icon: IconGeneratorUnpowered, Color: colorGeneratorOff, HasBackground: true, BackgroundColor: colorGeneratorFocusBg}
	}

	// Maintenance Terminal
	if gameworld.HasMaintenanceTerminal(cell) {
		return CellRenderOptions{Icon: IconMaintenance, Color: colorMaintenance, HasBackground: true, BackgroundColor: colorMaintenanceBg}
	}

	if gameworld.HasRepairDevice(cell) {
		repair := data.RepairDevice
		icon := repairIcon(repair)
		if repair != nil && repair.IsComplete() {
			return CellRenderOptions{Icon: icon, Color: colorSubtle, HasBackground: false}
		}
		powered := !repair.NeedsLivePower() || snapCellHasLivePower(snap, cell)
		fg, bg := repairDeviceColors(repair, powered)
		return CellRenderOptions{Icon: icon, Color: fg, HasBackground: true, BackgroundColor: bg}
	}

	// CCTV Terminal - same orange as maintenance terminals
	if gameworld.HasTerminal(cell) {
		if data.Terminal.IsUsed() {
			return CellRenderOptions{Icon: IconTerminalUsed, Color: colorTerminalUsed, HasBackground: false}
		}
		return CellRenderOptions{Icon: IconTerminalUnused, Color: colorMaintenance, HasBackground: true, BackgroundColor: colorMaintenanceBg}
	}

	// Puzzle Terminal
	if gameworld.HasPuzzle(cell) {
		if data.Puzzle.IsSolved() {
			return CellRenderOptions{Icon: IconTerminalUsed, Color: colorTerminalUsed, HasBackground: false}
		}
		return CellRenderOptions{Icon: IconTerminalUnused, Color: colorTerminal, HasBackground: true}
	}

	// Furniture
	if gameworld.HasFurniture(cell) {
		if data.Furniture.IsChecked() {
			return CellRenderOptions{Icon: data.Furniture.Icon, Color: colorFurnitureCheck, HasBackground: false}
		}
		return CellRenderOptions{Icon: data.Furniture.Icon, Color: colorFurniture, HasBackground: true}
	}

	// Exit cell
	if cell.ExitCell {
		switch setup.ExitLiftState(g) {
		case state.ExitLiftLockedUnpowered:
			return CellRenderOptions{Icon: IconExitLocked, Color: colorExitLocked, HasBackground: true}
		case state.ExitLiftLockedIncomplete:
			return CellRenderOptions{Icon: IconExitLocked, Color: colorExitPending, HasBackground: true}
		default:
			pulseColor := e.getPulsingExitColor()
			return CellRenderOptions{Icon: IconExitUnlocked, Color: pulseColor, HasBackground: true}
		}
	}

	// Corridor power relay
	if gameworld.HasPowerRelay(cell) {
		relay := data.PowerRelay
		if relay != nil && relay.Closed {
			return CellRenderOptions{Icon: IconRelayClosed, Color: colorMaintenance, HasBackground: true, BackgroundColor: colorWallBgPowered}
		}
		return CellRenderOptions{Icon: IconRelayOpen, Color: colorHazard, HasBackground: true, BackgroundColor: colorHazardBackground}
	}

	// Items on floor
	if cell.ItemsOnFloor.Size() > 0 {
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

	// Deck 1 west overlay rooms use dedicated hull/connector styling.
	if opts, ok := overlayRoomFloorRenderOptions(cell, features.IsVisited(cell)); ok {
		return opts
	}

	// Visited rooms (when gameplay.visited cvar is enabled)
	if features.IsVisited(cell) {
		return CellRenderOptions{Icon: getFloorIcon(cell.Name, true), Color: colorFloorVisited, HasBackground: true, BackgroundColor: colorFloorVisitedBg}
	}

	return CellRenderOptions{Icon: getFloorIcon(cell.Name, false), Color: colorFloor, HasBackground: true, BackgroundColor: colorFloorBg}
}

// rememberedCellRenderOptions renders a cell the player has seen lit but which is dark
// now: entity identity (glyph) is kept, live state (colors, plates) is withheld.
func (e *EbitenRenderer) rememberedCellRenderOptions(g *state.Game, cell *world.Cell, snap *renderSnapshot) CellRenderOptions {
	live := e.liveCellRenderOptions(g, cell, snap)
	return CellRenderOptions{
		Icon:            live.Icon,
		Color:           colorRemembered,
		HasBackground:   true,
		BackgroundColor: colorRememberedBg,
	}
}

// layoutCellRenderOptions renders floor-plan knowledge only: room shape, door and
// lift positions. Equipment, items, and hazards are not drawn.
func layoutCellRenderOptions(cell *world.Cell) CellRenderOptions {
	switch {
	case gameworld.HasDoor(cell):
		return CellRenderOptions{Icon: IconDoorUnlocked, Color: colorLayout, HasBackground: true, BackgroundColor: colorLayoutBg}
	case cell.ExitCell:
		return CellRenderOptions{Icon: IconExitLocked, Color: colorLayout, HasBackground: true, BackgroundColor: colorLayoutBg}
	default:
		return CellRenderOptions{Icon: getFloorIcon(cell.Name, false), Color: colorLayout, HasBackground: true, BackgroundColor: colorLayoutBg}
	}
}

func hazardClearVisualAlpha(snap *renderSnapshot, cell *world.Cell) float64 {
	if snap == nil || snap.hazardClear == nil || cell == nil {
		return 1
	}
	hc := snap.hazardClear
	if cell.Row != hc.HazardRow || cell.Col != hc.HazardCol {
		return 1
	}
	if hc.Phase != state.HazardClearFlash && hc.Phase != state.HazardClearFade {
		return 1
	}
	if hc.VisualAlpha <= 0 {
		return 0
	}
	if hc.VisualAlpha >= 1 {
		return 1
	}
	return hc.VisualAlpha
}

func repairIcon(repair *entities.RepairObjective) string {
	if repair == nil {
		return IconRepairValve
	}
	switch repair.Type {
	case entities.RepairPressureValve:
		return IconRepairValve
	case entities.RepairSignalCalibrator:
		return IconRepairSignal
	case entities.RepairPowerCoupler:
		return IconRepairCoupler
	case entities.RepairWastePump:
		return IconRepairPump
	case entities.RepairConduitSplice:
		return IconRepairConduit
	default:
		return IconRepairValve
	}
}

func repairDeviceColors(repair *entities.RepairObjective, powered bool) (fg, bg color.Color) {
	if repair != nil && repair.Type == entities.RepairConduitSplice {
		return colorRepairConduit, colorRepairConduitBg
	}
	if !powered {
		return colorGeneratorOff, colorHazardBackground
	}
	return colorRepair, colorRepairBg
}

// shipWallRenderOptions styles bulkhead walls bordering the deck 1 Ship room.
func shipWallRenderOptions(cell *world.Cell) (CellRenderOptions, bool) {
	if cell == nil || !hasAdjacentRoomNamed(cell, generator.ShipRoomName) {
		return CellRenderOptions{}, false
	}
	return CellRenderOptions{
		Icon:            IconShipHullWall,
		Color:           colorShipWall,
		HasBackground:   true,
		BackgroundColor: colorShipWallBg,
	}, true
}

// overlayRoomFloorRenderOptions returns distinct floor styling for deck 1 Ship.
func overlayRoomFloorRenderOptions(cell *world.Cell, visited bool) (CellRenderOptions, bool) {
	if cell == nil || cell.Name != generator.ShipRoomName {
		return CellRenderOptions{}, false
	}
	if visited {
		return CellRenderOptions{
			Icon:            getFloorIcon(cell.Name, true),
			Color:           colorShipFloorVisited,
			HasBackground:   true,
			BackgroundColor: colorShipFloorVisitedBg,
		}, true
	}
	return CellRenderOptions{
		Icon:            getFloorIcon(cell.Name, false),
		Color:           colorShipFloor,
		HasBackground:   true,
		BackgroundColor: colorShipFloorBg,
	}, true
}

// getFloorIcon returns the appropriate floor icon for a room
func getFloorIcon(roomName string, visited bool) string {
	cacheKey := "u:" + roomName
	if visited {
		cacheKey = "v:" + roomName
	}
	if icon, ok := floorIconCache[cacheKey]; ok {
		return icon
	}
	for baseRoom, icons := range roomFloorIcons {
		if strings.Contains(roomName, baseRoom) {
			if visited {
				floorIconCache[cacheKey] = icons[0]
				return icons[0]
			}
			floorIconCache[cacheKey] = icons[1]
			return icons[1]
		}
	}
	if visited {
		floorIconCache[cacheKey] = IconVisited
		return IconVisited
	}
	floorIconCache[cacheKey] = IconUnvisited
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

// shouldRenderWallCell reports whether a non-room cell should draw as a wall tile.
func shouldRenderWallCell(cell *world.Cell, snap *renderSnapshot) bool {
	if cell == nil || cell.Room {
		return false
	}
	if cell.Discovered || hasAdjacentDiscoveredRoom(cell) {
		return true
	}
	if focusRoom := snapMaintenanceMenuRoom(snap); focusRoom != "" && hasAdjacentRoomNamed(cell, focusRoom) {
		return true
	}
	return false
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
