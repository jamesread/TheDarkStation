package levelgen

import (
	"strings"
	"testing"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/levelrand"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestPlaceRepairObjectives_Deck3PresentsCouplerBeforeValve(t *testing.T) {
	seed := seedForExitGate(3, ExitGateNone)
	if seed < 0 {
		t.Fatal("could not find non-slime deck 3 seed")
	}
	levelrand.Seed(seed)
	g := state.NewGame()
	g.Level = 3
	g.CurrentDeckID = 2
	grid := world.NewGrid(7, 7)
	for row := 0; row < 7; row++ {
		for col := 0; col < 7; col++ {
			grid.MarkAsRoomWithName(row, col, string(rune('A'+row)), "")
			gameworld.InitGameData(grid.GetCell(row, col))
		}
	}
	grid.BuildAllCellConnections()
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(6, 6)
	g.Grid = grid

	avoid := mapset.New[*world.Cell]()
	avoid.Put(grid.StartCell())
	avoid.Put(grid.ExitCell())
	PlaceRepairObjectives(g, &avoid)

	var valve, coupler *entities.RepairObjective
	for _, repair := range g.RepairObjectives {
		switch repair.Type {
		case entities.RepairPressureValve:
			valve = repair
		case entities.RepairPowerCoupler:
			coupler = repair
		}
	}
	if valve == nil || coupler == nil {
		t.Fatalf("expected valve and coupler, got %+v", g.RepairObjectives)
	}
	dist := distancesFromStart(grid, setup.PlayerEntryCell(g))
	valveCell := grid.GetCell(valve.DeviceRow, valve.DeviceCol)
	couplerCell := grid.GetCell(coupler.DeviceRow, coupler.DeviceCol)
	if dist[couplerCell] >= dist[valveCell] {
		t.Fatalf("coupler distance %d should be before valve distance %d", dist[couplerCell], dist[valveCell])
	}
	if dist[valveCell]-dist[couplerCell] > causalReturnMaxStartDistanceDelta {
		t.Fatalf("return distance delta %d exceeds bound", dist[valveCell]-dist[couplerCell])
	}
	for _, hint := range g.Hints {
		if strings.Contains(hint, valve.Name) || strings.Contains(hint, valve.RoomName) ||
			strings.Contains(hint, coupler.Name) || strings.Contains(hint, coupler.RoomName) {
			t.Fatalf("generated hint spoils unseen repair identity: %q", hint)
		}
	}
}
