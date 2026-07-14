package setup

import (
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// BootstrapDeck1ShipSystems places the ship fusion reactor and emergency power conduits
// linking it to the lift shaft bootstrap generator.
func BootstrapDeck1ShipSystems(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	if g == nil || g.Grid == nil || g.Level != 1 {
		return
	}

	fusionCell := g.Grid.GetCell(generator.Deck1FusionReactorRow, generator.Deck1FusionReactorCol)
	if fusionCell == nil || fusionCell.Name != generator.ShipRoomName {
		return
	}
	if existing := gameworld.GetGameData(fusionCell).Generator; existing != nil && existing.Permanent {
		return
	}

	gen := entities.NewPermanentFusionReactor(generator.ShipFusionReactorName)
	gameworld.GetGameData(fusionCell).Generator = gen
	g.AddGenerator(gen)
	if avoid != nil {
		avoid.Put(fusionCell)
	}

	shaftGenCell := findLiftShaftBootstrapGeneratorCell(g)
	if shaftGenCell == nil {
		return
	}

	for _, cell := range emergencyConduitCellsBetween(g, fusionCell, shaftGenCell) {
		if cell == nil || gameworld.GetGameData(cell).Furniture != nil ||
			gameworld.HasDoor(cell) || gameworld.HasGenerator(cell) {
			continue
		}
		gameworld.GetGameData(cell).Furniture = entities.NewEmergencyPowerConduit()
		if avoid != nil {
			avoid.Put(cell)
		}
	}

	g.UpdatePowerSupply()
	NotifyPowerGridChanged(g)
}

func findLiftShaftBootstrapGeneratorCell(g *state.Game) *world.Cell {
	if g == nil || g.Grid == nil {
		return nil
	}
	var found *world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if found != nil || cell == nil || cell.Name != generator.ShaftRoomName {
			return
		}
		if gameworld.GetGameData(cell).Generator != nil {
			found = cell
		}
	})
	return found
}

func emergencyConduitCellsBetween(g *state.Game, from, to *world.Cell) []*world.Cell {
	if g == nil || from == nil || to == nil {
		return nil
	}
	parent := map[*world.Cell]*world.Cell{}
	queue := []*world.Cell{from}
	visited := map[*world.Cell]bool{from: true}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == to {
			break
		}
		for _, n := range cur.GetNeighbors() {
			if n == nil || !n.Room || visited[n] || !conduitPathPassable(g, n) {
				continue
			}
			visited[n] = true
			parent[n] = cur
			queue = append(queue, n)
		}
	}
	if !visited[to] {
		return nil
	}

	var path []*world.Cell
	for at := to; at != nil; at = parent[at] {
		path = append(path, at)
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	if len(path) <= 2 {
		return nil
	}
	return path[1 : len(path)-1]
}

func conduitPathPassable(g *state.Game, cell *world.Cell) bool {
	if cell == nil || !cell.Room || cell.ExitCell {
		return false
	}
	if gameworld.GetGameData(cell).Generator != nil {
		return true
	}
	if gameworld.FurnitureBlocksPowerGrid(cell) || gameworld.RepairDeviceBlocksPowerGrid(cell) {
		return false
	}
	if gameworld.RelayBlocksGrid(cell) {
		return false
	}
	if gameworld.HasDoor(cell) {
		return !gameworld.HasLockedDoor(cell)
	}
	return true
}
