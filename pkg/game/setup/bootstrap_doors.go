// Package setup — lift-pocket bootstrap door access (manual egress reachability).
package setup

import (
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// initPocketBootstrapDoors returns unlocked doors bordering the initial lift-pocket rooms.
// The player must stand adjacent to these to hold USE for manual egress release.
func initPocketBootstrapDoors(g *state.Game) []*world.Cell {
	if g == nil || g.Grid == nil {
		return nil
	}
	initRooms := reachableNamedRooms(InitialReachableCells(g))
	if len(initRooms) == 0 {
		return nil
	}
	var doors []*world.Cell
	seen := make(map[*world.Cell]bool)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !gameworld.HasDoor(cell) || seen[cell] {
			return
		}
		d := gameworld.GetGameData(cell).Door
		if d == nil || d.Locked {
			return
		}
		for _, n := range cell.GetNeighbors() {
			if n == nil || !n.Room || n.Name == "" || n.Name == "Corridor" {
				continue
			}
			if initRooms[n.Name] {
				doors = append(doors, cell)
				seen[cell] = true
				return
			}
		}
	})
	return doors
}

// doorHasStandNavAccess reports whether the player can reach a floor tile adjacent to doorCell
// from the lift entry using init movement rules (before manual door release on that door).
func doorHasStandNavAccess(g *state.Game, doorCell *world.Cell, extraBlocked *world.Cell) bool {
	if g == nil || doorCell == nil {
		return true
	}
	reach := InitialReachableCellsWithExtraBlock(g, extraBlocked)
	for _, n := range doorCell.GetNeighbors() {
		if n == nil || n == doorCell {
			continue
		}
		if extraBlocked != nil && n == extraBlocked {
			continue
		}
		if reach.Has(n) {
			return true
		}
	}
	return false
}

func bootstrapDoorNavPreserved(g *state.Game, extraBlocked *world.Cell) bool {
	for _, door := range initPocketBootstrapDoors(g) {
		if !doorHasStandNavAccess(g, door, extraBlocked) {
			return false
		}
	}
	return true
}

// EnsureBootstrapDoorNavAccess removes furniture that blocks manual egress on lift-pocket doors.
func EnsureBootstrapDoorNavAccess(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	for attempt := 0; attempt < 64; attempt++ {
		var blocked []*world.Cell
		for _, door := range initPocketBootstrapDoors(g) {
			if !doorHasStandNavAccess(g, door, nil) {
				blocked = append(blocked, door)
			}
		}
		if len(blocked) == 0 {
			return
		}
		var furnitureCells []*world.Cell
		g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
			if cell != nil && gameworld.GetGameData(cell).Furniture != nil {
				furnitureCells = append(furnitureCells, cell)
			}
		})
		if len(furnitureCells) == 0 {
			return
		}
		target := pickFurnitureBlockingBootstrapDoors(g, furnitureCells)
		if target == nil {
			if !bootstrapDoorsBlockedByFurniture(g, furnitureCells) {
				// Furniture is not the impediment; removing it cannot help.
				return
			}
			target = furnitureCells[0]
		}
		data := gameworld.GetGameData(target)
		if data.Furniture != nil && data.Furniture.ContainedItem != nil {
			target.ItemsOnFloor.Put(data.Furniture.ContainedItem)
		}
		data.Furniture = nil
	}
}

// bootstrapDoorsBlockedByFurniture reports whether removing all furniture restores
// bootstrap door stand access, i.e. furniture is at least part of the impediment.
func bootstrapDoorsBlockedByFurniture(g *state.Game, furniture []*world.Cell) bool {
	saved := make(map[*world.Cell]*entities.Furniture, len(furniture))
	for _, f := range furniture {
		data := gameworld.GetGameData(f)
		saved[f] = data.Furniture
		data.Furniture = nil
	}
	ok := bootstrapDoorNavPreserved(g, nil)
	for cell, f := range saved {
		gameworld.GetGameData(cell).Furniture = f
	}
	return ok
}

func pickFurnitureBlockingBootstrapDoors(g *state.Game, furniture []*world.Cell) *world.Cell {
	for _, f := range furniture {
		data := gameworld.GetGameData(f)
		saved := data.Furniture
		data.Furniture = nil
		allOK := true
		for _, door := range initPocketBootstrapDoors(g) {
			if !doorHasStandNavAccess(g, door, nil) {
				allOK = false
				break
			}
		}
		if allOK {
			return f
		}
		data.Furniture = saved
	}
	return nil
}
