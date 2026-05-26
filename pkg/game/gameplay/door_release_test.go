package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func makeDoorReleaseTestGame(t *testing.T) (*state.Game, *world.Cell, *world.Cell) {
	t.Helper()
	g := state.NewGame()
	grid := world.NewGrid(1, 3)
	grid.MarkAsRoomWithName(0, 0, "RoomA", "")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "")
	grid.MarkAsRoomWithName(0, 2, "RoomB", "")
	grid.BuildAllCellConnections()
	g.Grid = grid
	g.CurrentCell = grid.GetCell(0, 0)

	doorCell := grid.GetCell(0, 1)
	gameworld.InitGameData(doorCell)
	gameworld.GetGameData(doorCell).Door = &entities.Door{RoomName: "RoomB", Locked: false}
	g.RoomDoorsPowered["RoomB"] = true
	return g, g.CurrentCell, doorCell
}

func TestDoorNeedsManualRelease_unpoweredDoor(t *testing.T) {
	g, _, doorCell := makeDoorReleaseTestGame(t)
	if !DoorNeedsManualRelease(g, doorCell) {
		t.Fatal("unpowered door should allow manual release")
	}
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(g.Grid.GetCell(0, 2)).Generator = gen
	setup.PropagateRoomPowerOnlineFromGenerators(g)
	if DoorNeedsManualRelease(g, doorCell) {
		t.Fatal("powered door should not need manual release")
	}
}

func TestTryBeginLongUseOnAdjacent_unpoweredDoor(t *testing.T) {
	g, _, doorCell := makeDoorReleaseTestGame(t)
	if !TryBeginLongUseOnAdjacent(g) {
		t.Fatal("expected long use to start for adjacent unpowered door")
	}
	if g.LongUse == nil || g.LongUse.Kind != string(LongUseDoorManualRelease) {
		t.Fatalf("LongUse = %+v, want door_manual_release", g.LongUse)
	}
	if g.LongUse.TargetRow != doorCell.Row || g.LongUse.TargetCol != doorCell.Col {
		t.Fatalf("target = (%d,%d), want door at (%d,%d)",
			g.LongUse.TargetRow, g.LongUse.TargetCol, doorCell.Row, doorCell.Col)
	}
}

func TestCompleteManualDoorRelease_allowsPassage(t *testing.T) {
	g, _, doorCell := makeDoorReleaseTestGame(t)
	TryBeginLongUseOnAdjacent(g)
	start := int64(1000)
	AdvanceLongUseIfActive(g, true, false, start)
	AdvanceLongUseIfActive(g, true, false, start+LongUseHoldDuration.Milliseconds())

	if DoorNeedsManualRelease(g, doorCell) {
		t.Fatal("door should not need release after completion")
	}
	ok, _ := CanEnter(g, doorCell, false)
	if !ok {
		t.Fatal("manual release should allow entering door cell")
	}
}

func TestCanEnter_ManualEgressReleasedBypassesUnpowered(t *testing.T) {
	g, _, doorCell := makeDoorReleaseTestGame(t)
	g.ManualEgressReleased["RoomB"] = true
	ok, _ := CanEnter(g, doorCell, false)
	if !ok {
		t.Fatal("manual egress release should bypass unpowered door check")
	}
}
