package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestExitLiftState_gridPower(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(2, 2)
	grid.MarkAsRoomWithName(0, 0, "Start", "")
	grid.MarkAsRoomWithName(0, 1, "Lift", "")
	grid.BuildAllCellConnections()
	grid.SetExitCellAt(0, 1)
	gameworld.InitGameData(grid.GetCell(0, 0))
	gameworld.InitGameData(grid.GetCell(0, 1))
	g.Grid = grid
	g.RoomDoorsPowered["Start"] = true
	g.RoomDoorsPowered["Lift"] = true

	if got := ExitLiftState(g); got != state.ExitLiftLockedUnpowered {
		t.Fatalf("no grid power: ExitLiftState = %v, want LockedUnpowered", got)
	}

	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.AddGenerator(gen)
	PropagateRoomPowerOnlineFromGenerators(g)

	if got := ExitLiftState(g); got != state.ExitLiftReady {
		t.Fatalf("exit on grid: ExitLiftState = %v, want Ready", got)
	}

	hazardCell := grid.GetCell(0, 0)
	gameworld.GetGameData(hazardCell).Hazard = entities.NewHazard(entities.HazardVacuum)
	if got := ExitLiftState(g); got != state.ExitLiftLockedIncomplete {
		t.Fatalf("blocking hazard: ExitLiftState = %v, want LockedIncomplete", got)
	}
}

func TestExitCellHasLivePower_manualEgress(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 1)
	grid.MarkAsRoomWithName(0, 0, "Lift", "")
	grid.SetExitCellAt(0, 0)
	gameworld.InitGameData(grid.GetCell(0, 0))
	g.Grid = grid
	g.RoomDoorsPowered["Lift"] = false

	if ExitCellHasLivePower(g) {
		t.Fatal("expected no live power without generators or manual egress")
	}

	g.ManualEgressReleased = map[string]bool{"Lift": true}
	if !ExitCellHasLivePower(g) {
		t.Fatal("manual egress should satisfy exit lift power like a door room")
	}
	if ExitLiftState(g) != state.ExitLiftReady {
		t.Fatal("powered via manual egress with no hazards should be ready")
	}
}

func TestExitLiftState_lockedByIncompleteRepairs(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 2)
	grid.MarkAsRoomWithName(0, 0, "Start", "")
	grid.MarkAsRoomWithName(0, 1, "Lift", "")
	grid.BuildAllCellConnections()
	grid.SetStartCellAt(0, 0)
	grid.SetExitCellAt(0, 1)
	g.Grid = grid
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.AddGenerator(gen)
	g.RoomDoorsPowered["Start"] = true
	g.RoomDoorsPowered["Lift"] = true
	PropagateRoomPowerOnlineFromGenerators(g)
	g.RepairObjectives = []*entities.RepairObjective{
		entities.NewRepairObjective("pump", entities.RepairWastePump, "Pump Room", 0, 0),
	}

	if got := ExitLiftState(g); got != state.ExitLiftLockedIncomplete {
		t.Fatalf("incomplete repair: ExitLiftState = %v, want LockedIncomplete", got)
	}
	g.RepairObjectives[0].Complete()
	if got := ExitLiftState(g); got != state.ExitLiftReady {
		t.Fatalf("complete repair: ExitLiftState = %v, want Ready", got)
	}
}
