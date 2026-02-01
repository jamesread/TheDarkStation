// Package levelgen provides level generation utilities for placing entities.
package levelgen

import (
	"math/rand"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// PlaceFurniture places thematically appropriate furniture in rooms
func PlaceFurniture(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	// Collect all unique rooms and their cells
	roomCells := make(map[string][]*world.Cell)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell.Room && cell.Name != "Corridor" && cell.Name != "" {
			roomCells[cell.Name] = append(roomCells[cell.Name], cell)
		}
	})

	// For each unique room, try to place 1-2 furniture pieces
	for roomName, cells := range roomCells {
		templates := entities.GetAllFurnitureForRoom(roomName)
		if len(templates) == 0 {
			continue
		}

		// Shuffle templates for variety
		rand.Shuffle(len(templates), func(i, j int) {
			templates[i], templates[j] = templates[j], templates[i]
		})

		// Place 1-2 furniture pieces per room (based on room size)
		numFurniture := 1
		if len(cells) > 6 {
			numFurniture = 2
		}
		if numFurniture > len(templates) {
			numFurniture = len(templates)
		}

		// Find valid cells (not already used for something else, and not blocking entrances/exits)
		// Get room entry points to avoid blocking them
		roomEntries := setup.FindRoomEntryPoints(g.Grid)
		entryPoints := mapset.New[*world.Cell]()
		var entryCells []*world.Cell
		if entryData, ok := roomEntries[roomName]; ok {
			entryCells = entryData.EntryCells
			for _, entryCell := range entryData.EntryCells {
				// Mark the room cells adjacent to entry points as blocked
				neighbors := []*world.Cell{entryCell.North, entryCell.East, entryCell.South, entryCell.West}
				for _, neighbor := range neighbors {
					if neighbor != nil && neighbor.Room && neighbor.Name == roomName {
						entryPoints.Put(neighbor)
					}
				}
			}
		}

		var validCells []*world.Cell
		for _, cell := range cells {
			data := gameworld.GetGameData(cell)
			// Don't place furniture on entry points, exit cells, or cells with other entities
			if !avoid.Has(cell) && !cell.ExitCell && !entryPoints.Has(cell) &&
				data.Generator == nil && data.Door == nil && data.Terminal == nil &&
				data.Furniture == nil && data.Hazard == nil && data.HazardControl == nil {
				validCells = append(validCells, cell)
			}
		}

		// Shuffle valid cells
		rand.Shuffle(len(validCells), func(i, j int) {
			validCells[i], validCells[j] = validCells[j], validCells[i]
		})

		// Track furniture placed in this room for item hiding (R8: only place where room stays connected)
		var placedFurniture []*entities.Furniture
		used := mapset.New[*world.Cell]()

		for i := 0; i < numFurniture; i++ {
			template := templates[i]
			var chosen *world.Cell
			for _, cell := range validCells {
				if used.Has(cell) {
					continue
				}
				if !isRoomStillConnected(g, roomName, entryCells, cell) {
					continue
				}
				chosen = cell
				break
			}
			if chosen == nil {
				continue
			}
			used.Put(chosen)
			furniture := entities.NewFurniture(template.Name, template.Description, template.Icon)
			gameworld.GetGameData(chosen).Furniture = furniture
			placedFurniture = append(placedFurniture, furniture)
		}

		// Chance to hide items from the floor in furniture (40% per item)
		if len(placedFurniture) > 0 {
			hideItemsInFurniture(g, cells, placedFurniture, roomName)
		}
	}
}

// hideItemsInFurniture moves items from floor cells into furniture with a chance
func hideItemsInFurniture(g *state.Game, roomCells []*world.Cell, furniture []*entities.Furniture, roomName string) {
	// Find items on the floor in this room (keycards, patch kits - not batteries or maps)
	for _, cell := range roomCells {
		if cell.ItemsOnFloor.Size() == 0 {
			continue
		}

		var itemsToMove []*world.Item
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			// Only hide keycards and patch kits - items that are part of puzzles
			if ContainsSubstring(item.Name, "Keycard") || item.Name == "Patch Kit" {
				// 50% chance to hide in furniture
				if rand.Intn(100) < 50 {
					itemsToMove = append(itemsToMove, item)
				}
			}
		})

		// Move items to furniture
		for _, item := range itemsToMove {
			// Find furniture without an item
			for _, f := range furniture {
				if f.ContainedItem == nil {
					cell.ItemsOnFloor.Remove(item)
					f.ContainedItem = item

					// Update hint to mention furniture instead of room
					updateHintForFurnitureItem(g, item, f, roomName)
					break
				}
			}
		}
	}
}

// updateHintForFurnitureItem updates the hint for an item to mention it's in furniture
func updateHintForFurnitureItem(g *state.Game, item *world.Item, furniture *entities.Furniture, roomName string) {
	// Find and update the existing hint for this item
	for i, hint := range g.Hints {
		if ContainsSubstring(hint, item.Name) && ContainsSubstring(hint, roomName) {
			// Replace the hint with one mentioning the furniture
			g.Hints[i] = "The " + renderer.StyledKeycard(item.Name) + " is hidden in the " +
				renderer.StyledFurniture(furniture.Name) + " in " + renderer.StyledCell(roomName)
			return
		}
	}
}
