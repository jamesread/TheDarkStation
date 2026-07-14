package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/levelseed"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// Regression for map.txt seed 18B7D94525F6E372 (deck 6): furniture blocked manual egress
// on an init-pocket bootstrap door. Asserts every unlocked door bordering the lift-entry
// pocket keeps an init-reachable stand cell for manual egress release.
func TestBootstrapDoorNavAccess_mapTxtSeed(t *testing.T) {
	levelSeed, err := levelseed.Parse("18B7D94525F6E372")
	if err != nil {
		t.Fatal(err)
	}
	g := state.NewGame()
	g.InitRunUnlocks(levelSeed - int64(5)*9973)
	g.Level = 6
	RegenerateFromSeed(g, levelSeed)

	reach := setup.InitialReachableCells(g)
	initRooms := map[string]bool{}
	reach.Each(func(c *world.Cell) {
		if c.Name != "" && c.Name != "Corridor" {
			initRooms[c.Name] = true
		}
	})

	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !gameworld.HasDoor(cell) {
			return
		}
		door := gameworld.GetGameData(cell).Door
		if door == nil || door.Locked {
			return
		}
		borders := false
		for _, n := range cell.GetNeighbors() {
			if n != nil && n.Room && n.Name != "" && n.Name != "Corridor" && initRooms[n.Name] {
				borders = true
				break
			}
		}
		if !borders {
			return
		}
		for _, n := range cell.GetNeighbors() {
			if n != nil && reach.Has(n) {
				return
			}
		}
		t.Errorf("bootstrap door x:%d y:%d (room %q) has no init-reachable stand cell", cell.Col, cell.Row, door.RoomName)
	})
}
