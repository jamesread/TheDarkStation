package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func makePropagationTestGame() *state.Game {
	g := state.NewGame()
	grid := world.NewGrid(1, 5)
	g.Grid = grid
	g.CurrentDeckID = 0

	rooms := []struct {
		name     string
		row, col int
	}{
		{"GenRoom", 0, 0},
		{"MidRoom", 0, 2},
		{"FarRoom", 0, 4},
	}
	for _, r := range rooms {
		c := grid.GetCell(r.row, r.col)
		c.Room = true
		c.Name = r.name
		c.Discovered = true
	}
	for _, col := range []int{1, 3} {
		c := grid.GetCell(0, col)
		c.Room = true
		c.Name = "Corridor"
		c.Discovered = true
	}
	grid.BuildAllCellConnections()

	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteriesAndStart(1)
	genCell := grid.GetCell(0, 0)
	gameworld.GetGameData(genCell).Generator = gen
	g.AddGenerator(gen)

	g.RoomDoorsPowered = map[string]bool{
		"GenRoom": true, "MidRoom": true, "FarRoom": true,
		"Corridor": true,
	}
	g.RoomCCTVPowered = make(map[string]bool)
	return g
}

func TestSchedulePowerPropagation_instantActivation(t *testing.T) {
	g := makePropagationTestGame()
	now := int64(1000)
	SchedulePowerPropagation(g, now)
	if !RoomIsOnline(g, "GenRoom") {
		t.Fatal("GenRoom should be online immediately (generator depth 0)")
	}
	if !RoomIsOnline(g, "MidRoom") || !RoomIsOnline(g, "FarRoom") {
		t.Fatal("downstream armed rooms should be online immediately with zero delay")
	}
	if len(g.PowerPropPending) != 0 {
		t.Fatalf("pending len = %d, want 0 after instant propagation", len(g.PowerPropPending))
	}
}

func TestAdvancePowerPropagation_overloadOnActivation(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 14)
	g.Grid = grid
	g.CurrentDeckID = 0

	genCell := grid.GetCell(0, 0)
	genCell.Room, genCell.Name, genCell.Discovered = true, "GenRoom", true
	for col := 1; col < 13; col++ {
		c := grid.GetCell(0, col)
		c.Room, c.Discovered = true, true
		if col == 1 || col == 12 {
			c.Name = "Corridor"
		} else if col >= 2 && col <= 11 {
			c.Name = "MidRoom"
			gameworld.InitGameData(c)
			gameworld.GetGameData(c).Terminal = entities.NewCCTVTerminal("T")
		}
	}
	farCell := grid.GetCell(0, 13)
	farCell.Room, farCell.Name, farCell.Discovered = true, "FarRoom", true
	gameworld.InitGameData(farCell)
	gameworld.GetGameData(farCell).Door = &entities.Door{RoomName: "FarRoom", Locked: false}
	grid.BuildAllCellConnections()

	gen := entities.NewGenerator("G1", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(genCell).Generator = gen
	g.AddGenerator(gen)

	g.RoomDoorsPowered = map[string]bool{
		"GenRoom": true, "MidRoom": true, "FarRoom": true, "Corridor": true,
	}
	g.RoomCCTVPowered = map[string]bool{"MidRoom": true}
	g.RoomPowerOnline = map[string]bool{"MidRoom": true}
	g.UpdatePowerSupply()
	if ConsumptionIfRoomCameOnline(g, "FarRoom") <= ArmedGridSupplyForRoom(g, "FarRoom") {
		t.Fatalf("test setup: need overload when FarRoom comes online, got %d vs %d",
			ConsumptionIfRoomCameOnline(g, "FarRoom"), ArmedGridSupplyForRoom(g, "FarRoom"))
	}
	now := int64(5000)
	g.PowerPropPending = []state.PowerPropEntry{{RoomName: "FarRoom", ActivateAt: now}}
	if !AdvancePowerPropagation(g, now) {
		t.Fatal("expected overload short when FarRoom activates")
	}
	if g.PowerConsumption > g.PowerSupply {
		t.Errorf("consumption (%d) should be <= supply (%d) after short", g.PowerConsumption, g.PowerSupply)
	}
	if CalculatePowerConsumption(g) > g.PowerSupply && RoomIsOnline(g, "FarRoom") {
		t.Error("FarRoom should not stay online while still overloaded")
	}
}

func TestClearRoomPropagatedPower_onDisarm(t *testing.T) {
	g := makePropagationTestGame()
	g.RoomPowerOnline["MidRoom"] = true
	now := int64(0)
	SchedulePowerPropagation(g, now)

	ClearRoomPropagatedPower(g, "MidRoom")
	if RoomIsOnline(g, "MidRoom") {
		t.Error("MidRoom should be offline after clear")
	}
	for _, p := range g.PowerPropPending {
		if p.RoomName == "MidRoom" {
			t.Error("pending should not include MidRoom after disarm clear")
		}
	}
}

func TestRoomIsOnline_zeroLoadArmedRoomOnLiveGrid(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 3)
	grid.MarkAsRoomWithName(0, 0, "GenRoom", "")
	grid.MarkAsRoomWithName(0, 1, "Corridor", "ROOM_CORRIDOR")
	grid.MarkAsRoomWithName(0, 2, "EmptyRoom", "")
	grid.BuildAllCellConnections()
	g.Grid = grid
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil {
			gameworld.InitGameData(cell)
		}
	})

	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen
	g.AddGenerator(gen)
	g.RoomDoorsPowered = map[string]bool{"GenRoom": true, "EmptyRoom": true, "Corridor": true}
	g.RoomCCTVPowered = make(map[string]bool)

	PropagateRoomPowerOnlineFromGenerators(g)

	if !RoomIsOnline(g, "EmptyRoom") {
		t.Fatal("zero-load armed room on live grid should be online after propagation")
	}
	if !RoomConsideredPowered(g, "EmptyRoom") {
		t.Fatal("zero-load armed room should be considered powered")
	}
}
