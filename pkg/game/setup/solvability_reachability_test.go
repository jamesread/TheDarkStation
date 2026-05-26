package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// makeStartPocketGrid: Start room pocket (cols 0-2) connected to Lab via door at (1,3).
// Start has powered maint; Lab doors start unpowered.
func makeStartPocketGrid(t *testing.T) (*state.Game, *world.Cell) {
	t.Helper()
	grid := world.NewGrid(3, 5)
	for c := 0; c <= 2; c++ {
		grid.MarkAsRoomWithName(0, c, "Start", "desc")
		grid.MarkAsRoomWithName(1, c, "Start", "desc")
	}
	grid.MarkAsRoomWithName(0, 3, "Lab", "desc")
	grid.MarkAsRoomWithName(1, 3, "Corridor", "desc")
	grid.MarkAsRoomWithName(0, 4, "Lab", "desc")
	grid.MarkAsRoomWithName(1, 4, "Lab", "desc")
	grid.SetStartCellAt(1, 1)
	grid.SetExitCellAt(0, 4)
	grid.BuildAllCellConnections()
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})
	doorCell := grid.GetCell(1, 3)
	gameworld.GetGameData(doorCell).Door = &entities.Door{RoomName: "Lab", Locked: false}
	maintCell := grid.GetCell(1, 0)
	gameworld.GetGameData(maintCell).MaintenanceTerm = entities.NewMaintenanceTerminal("MT-Start", "Start")
	labMaint := grid.GetCell(1, 4)
	gameworld.GetGameData(labMaint).MaintenanceTerm = entities.NewMaintenanceTerminal("MT-Lab", "Lab")

	g := state.NewGame()
	g.Grid = grid
	InitRoomPower(g)
	InitMaintenanceTerminalPower(g)

	gen := entities.NewGenerator("Spawn Generator", 1)
	gen.InsertBatteriesAndStart(1)
	genCell := grid.GetCell(0, 0)
	gameworld.GetGameData(genCell).Generator = gen
	g.AddGenerator(gen)
	EnsureGeneratorRoomBootstrap(g)

	return g, doorCell
}

func TestEnsureSolvabilityStartRoomEgress_DoesNotPrePower(t *testing.T) {
	g, _ := makeStartPocketGrid(t)
	if !CanPowerRoomDoorsFromReachable(g, InitialReachableCells(g), "Lab") {
		t.Fatal("precondition: Lab doors should be remotely controllable from Start maint")
	}
	if g.RoomDoorsPowered["Lab"] {
		t.Fatal("precondition: Lab doors should start unpowered")
	}
}

func TestEnsureSolvabilityStartRoomEgress_NoStartTerminalStaysUnpowered(t *testing.T) {
	g, _ := makeStartPocketGrid(t)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || cell.Name != "Start" {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.MaintenanceTerm != nil {
			data.MaintenanceTerm.Powered = false
			data.MaintenanceTerm = nil
		}
	})
	gameworld.GetGameData(g.Grid.GetCell(1, 4)).MaintenanceTerm = nil
	if CanPowerRoomDoorsFromReachable(g, InitialReachableCells(g), "Lab") {
		t.Fatal("precondition: without start terminal, Lab should not be remotely controllable")
	}
	if g.RoomDoorsPowered["Lab"] {
		t.Error("Lab doors should stay unpowered without start-room egress pre-power")
	}
}

func TestCanControlRoomPower_AdjacentRemote(t *testing.T) {
	g, _ := makeStartPocketGrid(t)
	if !CanControlRoomPower(g, "Start", "Lab") {
		t.Error("Start with powered terminal should remotely control adjacent Lab")
	}
	if CanControlRoomPower(g, "Start", "Missing") {
		t.Error("should not control non-adjacent room")
	}
}

func TestSelectableRoomsForTerminal_IncludesAdjacent(t *testing.T) {
	g, _ := makeStartPocketGrid(t)
	rooms := SelectableRoomsForTerminal(g, g.Grid, "Start")
	hasLab := false
	for _, r := range rooms {
		if r == "Lab" {
			hasLab = true
		}
	}
	if !hasLab {
		t.Fatalf("selectable rooms from Start should include adjacent Lab, got %v", rooms)
	}
}

func TestInitialEgressDoors_FindsBlockedDoor(t *testing.T) {
	g, doorCell := makeStartPocketGrid(t)
	doors := InitialEgressDoors(g)
	if len(doors) != 1 {
		t.Fatalf("expected 1 blocked egress door, got %d", len(doors))
	}
	if doors[0].Row != doorCell.Row || doors[0].Col != doorCell.Col {
		t.Errorf("egress door = (%d,%d), want (%d,%d)", doors[0].Row, doors[0].Col, doorCell.Row, doorCell.Col)
	}
}

func TestAnalyzeSolvability_NoEgressWarningsWhenControllable(t *testing.T) {
	g, _ := makeStartPocketGrid(t)
	report := AnalyzeSolvability(g)
	for _, w := range report.Warnings {
		if w != "exit not reachable at init (expected until doors powered/keycards found)" {
			t.Errorf("unexpected warning: %s", w)
		}
	}
}
