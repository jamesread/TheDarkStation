package generator

import (
	"fmt"

	"darkstation/pkg/engine/world"
)

const (
	// ShipRoomName is the fixed player vessel room on deck 1 only.
	ShipRoomName = "Ship"
	// ShipFusionReactorName is the permanent generator placed in the ship on deck 1.
	ShipFusionReactorName = "Ship's fusion reactor"
)

// deck1 north-entry layout: Ship west of the lift shaft (cols 4–6, rows 1–5), east bulkhead
// at col 7 with a single airlock door row.
const (
	deck1ShipStartRow = 1
	deck1ShipEndRow   = 5
	deck1ShipStartCol = 4
	deck1ShipEndCol   = 6

	deck1ShipEastWallCol = 7
	deck1ShipDoorRow     = 5

	// deck1WestOverlayRightCol is the inclusive right edge of the reserved west pocket.
	// The shaft begins at col 8 on the 22-column deck 1 grid.
	deck1WestOverlayRightCol = 7

	deck1OverlayStartRow = deck1ShipStartRow
	deck1OverlayEndRow   = deck1ShipEndRow
	deck1OverlayStartCol = deck1ShipStartCol

	deck1ShipStartRowCenter = 2
	deck1ShipStartColCenter = 5

	Deck1FusionReactorRow = deck1ShipStartRowCenter + 2
	Deck1FusionReactorCol = deck1ShipStartColCenter
)

// IsPlacementExcludedRoom reports rooms that must never receive procedural entities.
func IsPlacementExcludedRoom(name string) bool {
	switch name {
	case ShaftRoomName, ShipRoomName:
		return true
	default:
		return false
	}
}

// IsEmptyOverlayRoom reports deck 1 overlay rooms excluded from procedural placement.
func IsEmptyOverlayRoom(name string) bool {
	return name == ShipRoomName
}

// reserveDeck1WestOverlayLeaf force-splits the BSP root so cols 4–7 become a fixed west
// pocket for the north-entry Ship. This prevents random rooms and corridors from intruding
// into the overlay before CarveDeck1ShipAndDock refines the shape.
func reserveDeck1WestOverlayLeaf(root *bspNode, rows, cols int) *bspNode {
	if root == nil {
		return root
	}
	splitX := deck1WestOverlayRightCol + 1
	if splitX-root.x < minShaftStripExtent {
		return root
	}
	if root.x+root.width-splitX < minShaftStripExtent {
		return root
	}
	left, right := splitNodeVertical(root, splitX)
	left.room = &bspRoom{
		x:           deck1OverlayStartCol,
		y:           deck1OverlayStartRow,
		width:       deck1WestOverlayRightCol - deck1OverlayStartCol + 1,
		height:      deck1OverlayEndRow - deck1OverlayStartRow + 1,
		name:        ShipRoomName,
		description: "ROOM_WEST_OVERLAY",
	}
	return right
}

func markDeck1OverlayWall(grid *world.Grid, row, col int) {
	cell := grid.GetCell(row, col)
	if cell == nil {
		return
	}
	cell.Room = false
	cell.ExitCell = false
	cell.Name = fmt.Sprintf("%v:%v", row, col)
	cell.Description = world.GenerateCellDescription()
}

// CarveDeck1ShipAndDock overlays the fixed Ship room on deck 1 and sets StartCell.
func CarveDeck1ShipAndDock(grid *world.Grid) {
	if grid == nil {
		return
	}

	for row := deck1OverlayStartRow; row <= deck1OverlayEndRow; row++ {
		for col := deck1OverlayStartCol; col <= deck1WestOverlayRightCol; col++ {
			markDeck1OverlayWall(grid, row, col)
		}
	}

	for row := deck1ShipStartRow; row <= deck1ShipEndRow; row++ {
		for col := deck1ShipStartCol; col <= deck1ShipEndCol; col++ {
			grid.MarkAsRoomWithName(row, col, ShipRoomName, "ROOM_SHIP")
		}
	}

	for row := deck1ShipStartRow; row <= deck1ShipEndRow; row++ {
		if row == deck1ShipDoorRow {
			grid.MarkAsRoomWithName(row, deck1ShipEastWallCol, "Corridor", "ROOM_SHIP_AIRLOCK")
			continue
		}
		markDeck1OverlayWall(grid, row, deck1ShipEastWallCol)
	}

	grid.SetStartCellAt(deck1ShipStartRowCenter, deck1ShipStartColCenter)
}
