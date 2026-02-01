// Package levelgen provides level generation utilities for placing entities.
package levelgen

import (
	"fmt"
	"math/rand"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// PlaceHazards places environmental hazards that block progress
func PlaceHazards(g *state.Game, avoid *mapset.Set[*world.Cell], lockedDoorCells *mapset.Set[*world.Cell]) {
	// Number of hazards scales with level: level 2 = 1, level 3 = 1-2, level 4+ = 2-3
	numHazards := 1
	if g.Level >= 4 {
		numHazards = 2 + rand.Intn(2)
	} else if g.Level >= 3 {
		numHazards = 1 + rand.Intn(2)
	}

	// Available hazard types (excluding Vacuum initially, add it at level 3+)
	hazardTypes := []entities.HazardType{
		entities.HazardCoolant,
		entities.HazardElectrical,
		entities.HazardGas,
	}
	if g.Level >= 3 {
		hazardTypes = append(hazardTypes, entities.HazardVacuum)
	}
	if g.Level >= 5 {
		hazardTypes = append(hazardTypes, entities.HazardRadiation)
	}

	// Find room entry points (like doors, hazards block entry to rooms)
	roomEntries := setup.FindRoomEntryPoints(g.Grid)

	// Build list of candidate rooms (rooms with 1-3 entry points that we can fully block)
	type roomCandidate struct {
		name    string
		entries *setup.RoomEntryPoints
	}
	var candidates []roomCandidate
	for roomName, entries := range roomEntries {
		// Only consider rooms with 1-3 entry points (manageable to block)
		if len(entries.EntryCells) >= 1 && len(entries.EntryCells) <= 3 {
			candidates = append(candidates, roomCandidate{name: roomName, entries: entries})
		}
	}

	// Shuffle candidates for variety
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	// Track which rooms have hazards
	roomsWithHazards := mapset.New[string]()
	hazardsPlaced := 0

	// Place hazards to fully block selected rooms (like doors)
	for _, candidate := range candidates {
		if hazardsPlaced >= numHazards {
			break
		}

		roomName := candidate.name
		entryCells := candidate.entries.EntryCells

		// Skip if already has hazards or doors
		if roomsWithHazards.Has(roomName) {
			continue
		}

		// Check if all entry cells are available and reachable
		currentlyReachable := GetReachableCells(g.Grid, g.Grid.StartCell(), lockedDoorCells)
		allValid := true
		for _, cell := range entryCells {
			if avoid.Has(cell) || lockedDoorCells.Has(cell) || !currentlyReachable.Has(cell) {
				allValid = false
				break
			}
		}
		if !allValid {
			continue
		}

		// Test if blocking ALL entry cells reduces reachability
		testBlocked := mapset.New[*world.Cell]()
		lockedDoorCells.Each(func(c *world.Cell) { testBlocked.Put(c) })
		for _, cell := range entryCells {
			testBlocked.Put(cell)
		}
		reachableWithHazard := GetReachableCells(g.Grid, g.Grid.StartCell(), &testBlocked)

		// Must actually block something
		if reachableWithHazard.Size() >= currentlyReachable.Size() {
			continue
		}

		// Choose a random hazard type
		hazardType := hazardTypes[rand.Intn(len(hazardTypes))]
		hazard := entities.NewHazard(hazardType)
		info := entities.HazardTypes[hazardType]

		// Place hazards on ALL entry cells (they share the same solution)
		for _, cell := range entryCells {
			gameworld.GetGameData(cell).Hazard = hazard
			avoid.Put(cell)
		}

		roomsWithHazards.Put(roomName)
		hazardsPlaced++

		// Place the solution (item or control) in the area reachable BEFORE these hazards
		if hazard.RequiresItem() {
			// Place the required item (e.g., Patch Kit) in a reachable area
			itemRoom := FindRoomInReachable(reachableWithHazard, avoid)
			if itemRoom == nil {
				itemRoom = FindRoomInReachable(currentlyReachable, avoid)
			}
			if itemRoom != nil {
				item := world.NewItem(info.ItemName)
				itemRoom.ItemsOnFloor.Put(item)
				avoid.Put(itemRoom)
				g.AddHint("A " + renderer.StyledItem(info.ItemName) + " is in " + renderer.StyledCell(itemRoom.Name))
			}
		} else {
			// Place the control panel in a reachable area on a cell that is NOT an articulation point,
			// so the shutoff valve doesn't block the only path to a room
			controlRoom := FindNonArticulationCellInReachable(g.Grid, g.Grid.StartCell(), lockedDoorCells, reachableWithHazard, avoid)
			if controlRoom == nil {
				controlRoom = FindNonArticulationCellInReachable(g.Grid, g.Grid.StartCell(), lockedDoorCells, currentlyReachable, avoid)
			}
			if controlRoom != nil {
				control := entities.NewHazardControl(hazardType, hazard)
				gameworld.GetGameData(controlRoom).HazardControl = control
				avoid.Put(controlRoom)
				g.AddHint("The " + renderer.StyledHazardCtrl(info.ControlName) + " is in " + renderer.StyledCell(controlRoom.Name))
			}
		}

		if len(entryCells) == 1 {
			g.AddHint(fmt.Sprintf("A %s blocks access to %s", info.Name, renderer.StyledCell(roomName)))
		} else {
			g.AddHint(fmt.Sprintf("%d %s hazards block access to %s", len(entryCells), info.Name, renderer.StyledCell(roomName)))
		}
	}
}
