// Package levelgen provides level generation utilities for placing entities.
package levelgen

import (
	"darkstation/pkg/game/levelrand"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
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
	for _, roomName := range SortedRoomMapKeys(roomCells) {
		if generator.IsPlacementExcludedRoom(roomName) {
			continue
		}
		cells := roomCells[roomName]
		templates := entities.GetAllFurnitureForRoom(roomName)
		if len(templates) == 0 {
			templates = entities.FurnitureFallbackForTheme(g.ThemeForCurrentDeck())
		}
		if len(templates) == 0 {
			continue
		}

		// Shuffle templates for variety
		levelrand.Shuffle(len(templates), func(i, j int) {
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
				data.Furniture == nil && data.Hazard == nil && data.HazardControl == nil &&
				data.Puzzle == nil && data.MaintenanceTerm == nil &&
				data.RepairDevice == nil && data.RepairBlocker == nil {
				validCells = append(validCells, cell)
			}
		}

		// Shuffle valid cells
		levelrand.Shuffle(len(validCells), func(i, j int) {
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
				if !setup.CanPlaceBlockingEntity(g, cell) {
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
			// Record in the shared avoid set so later placement passes (batteries,
			// keycards, repairs) never target an occupied cell.
			avoid.Put(chosen)
			placedFurniture = append(placedFurniture, furniture)
		}

		// Chance to hide items from the floor in furniture
		if len(placedFurniture) > 0 {
			hideItemsInFurniture(g, cells, placedFurniture, roomName)
		}
	}

	if !gridHasFurniture(g) {
		placeFallbackFurniture(g, avoid)
	}
}

func gridHasFurniture(g *state.Game) bool {
	if g == nil || g.Grid == nil {
		return false
	}
	found := false
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if found || cell == nil {
			return
		}
		if gameworld.GetGameData(cell).Furniture != nil {
			found = true
		}
	})
	return found
}

func placeFallbackFurniture(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	templates := entities.FurnitureFallbackForTheme(g.ThemeForCurrentDeck())
	if len(templates) == 0 {
		return
	}
	template := templates[0]
	var fallback *world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if fallback != nil || cell == nil || !cell.Room || cell.Name == "" || cell.Name == "Corridor" ||
			generator.IsPlacementExcludedRoom(cell.Name) || cell.ExitCell {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Generator != nil || data.Door != nil || data.Terminal != nil ||
			data.Furniture != nil || data.Hazard != nil || data.HazardControl != nil ||
			data.MaintenanceTerm != nil || data.Puzzle != nil ||
			data.RepairDevice != nil || data.RepairBlocker != nil {
			return
		}
		if avoid != nil && avoid.Has(cell) {
			return
		}
		if !setup.CanPlaceBlockingEntity(g, cell) {
			return
		}
		fallback = cell
	})
	if fallback == nil {
		return
	}
	gameworld.GetGameData(fallback).Furniture = entities.NewFurniture(template.Name, template.Description, template.Icon)
	if avoid != nil {
		avoid.Put(fallback)
	}
}

// hideItemsInFurniture moves items from floor cells into furniture with a chance
func hideItemsInFurniture(g *state.Game, roomCells []*world.Cell, furniture []*entities.Furniture, roomName string) {
	prefs := g.ItemPlacement()
	if !prefs.HideItemsInFurniture {
		return
	}
	chance := prefs.HideInFurnitureChancePct
	if chance <= 0 {
		return
	}
	// Find items on the floor in this room (keycards, patch kits - not batteries or maps)
	for _, cell := range roomCells {
		if cell.ItemsOnFloor.Size() == 0 {
			continue
		}

		var itemsToMove []*world.Item
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			// Only hide keycards and patch kits - items that are part of puzzles
			if ContainsSubstring(item.Name, "Keycard") || item.Name == "Patch Kit" {
				if levelrand.Intn(100) < chance {
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
