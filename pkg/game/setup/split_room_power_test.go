package setup

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// Two physically disconnected pockets share one room name; only the generator pocket may receive power.
func TestSplitRoomName_onlyGeneratorPocketPowered(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 7)
	g.Grid = grid

	roomName := "SplitRoom"
	for _, col := range []int{0, 1, 2} {
		c := grid.GetCell(0, col)
		c.Room, c.Name, c.Discovered = true, roomName, true
	}
	for _, col := range []int{4, 5, 6} {
		c := grid.GetCell(0, col)
		c.Room, c.Name, c.Discovered = true, roomName, true
	}
	block := grid.GetCell(0, 3)
	block.Room, block.Name, block.Discovered = true, "Corridor", true
	gameworld.InitGameData(block)
	gameworld.GetGameData(block).PowerRelay = entities.NewPowerRelayOpen()
	grid.BuildAllCellConnections()

	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	genCell := grid.GetCell(0, 0)
	gameworld.GetGameData(genCell).Generator = gen

	termNearGen := entities.NewMaintenanceTerminal("T-gen", roomName)
	termNearGen.Powered = false
	gameworld.GetGameData(grid.GetCell(0, 1)).MaintenanceTerm = termNearGen

	termFar := entities.NewMaintenanceTerminal("T-far", roomName)
	termFar.Powered = false
	gameworld.GetGameData(grid.GetCell(0, 5)).MaintenanceTerm = termFar

	g.RoomDoorsPowered = map[string]bool{roomName: true}

	PropagateRoomPowerOnlineFromGenerators(g)
	ApplyGridConductivePower(g)

	if !CellHasLivePower(g, grid.GetCell(0, 1)) {
		t.Fatal("generator pocket cell should have live power")
	}
	if CellHasLivePower(g, grid.GetCell(0, 5)) {
		t.Fatal("isolated pocket should not have live power despite shared room name")
	}
	if !termNearGen.Powered {
		t.Fatal("terminal in generator pocket should be powered")
	}
	if termFar.Powered {
		t.Fatal("terminal in isolated pocket should stay unpowered")
	}
}
