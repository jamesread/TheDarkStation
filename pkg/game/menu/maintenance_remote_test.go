package menu

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestRoomPowerToggleMenuItem_RemoteAdjacentWithoutTargetMaint(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 4)
	grid.MarkAsRoomWithName(0, 0, "Start", "desc")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "desc")
	grid.MarkAsRoomWithName(0, 2, "Corridor", "desc")
	grid.MarkAsRoomWithName(0, 3, "Lab", "desc")
	grid.BuildAllCellConnections()
	g.Grid = grid
	for c := 0; c < 4; c++ {
		gameworld.InitGameData(grid.GetCell(0, c))
	}
	gameworld.GetGameData(grid.GetCell(0, 2)).Door = &entities.Door{RoomName: "Lab", Locked: false}
	termStart := entities.NewMaintenanceTerminal("MT-S", "Start")
	termStart.Powered = true
	gameworld.GetGameData(grid.GetCell(0, 0)).MaintenanceTerm = termStart
	termLab := entities.NewMaintenanceTerminal("MT-L", "Lab")
	termLab.Powered = false
	gameworld.GetGameData(grid.GetCell(0, 3)).MaintenanceTerm = termLab
	g.RoomDoorsPowered = map[string]bool{"Start": true, "Lab": false, "Corridor": false}

	toggle := &RoomPowerToggleMenuItem{G: g, RoomName: "Lab", ControllerRoom: "Start", PowerType: "doors"}
	if !toggle.IsSelectable() {
		t.Fatal("should toggle Lab doors remotely from Start terminal")
	}
}
