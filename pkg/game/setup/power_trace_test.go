package setup

import (
	"strings"
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// makeTraceGrid builds: RoomA(0,0..1) — corridor(0,2..4) — RoomB(0,5..6).
func makeTraceGrid(t *testing.T) (*state.Game, *world.Grid) {
	t.Helper()
	grid := world.NewGrid(1, 7)
	grid.MarkAsRoomWithName(0, 0, "RoomA", "")
	grid.MarkAsRoomWithName(0, 1, "RoomA", "")
	grid.MarkAsRoomWithName(0, 2, "Corridor", "")
	grid.MarkAsRoomWithName(0, 3, "Corridor", "")
	grid.MarkAsRoomWithName(0, 4, "Corridor", "")
	grid.MarkAsRoomWithName(0, 5, "RoomB", "")
	grid.MarkAsRoomWithName(0, 6, "RoomB", "")
	grid.BuildAllCellConnections()
	for c := 0; c < 7; c++ {
		gameworld.InitGameData(grid.GetCell(0, c))
	}
	g := state.NewGame()
	g.Grid = grid
	return g, grid
}

func TestTraceBusFault_burnedConduit(t *testing.T) {
	g, grid := makeTraceGrid(t)
	splice := entities.NewRepairObjective("c1", entities.RepairConduitSplice, "Corridor", 0, 3)
	splice.SegmentLabel = "SEG-AA"
	gameworld.GetGameData(grid.GetCell(0, 3)).RepairDevice = splice

	res := TraceBusFault(g, grid.GetCell(0, 0), "RoomB")
	if res.Kind != PowerFaultBurnedConduit {
		t.Fatalf("kind = %q, want burned conduit", res.Kind)
	}
	if res.Label != "SEG-AA" {
		t.Fatalf("label = %q, want SEG-AA", res.Label)
	}
	if res.Bearing != "E" {
		t.Fatalf("bearing = %q, want E", res.Bearing)
	}
	if res.Steps != 3 {
		t.Fatalf("steps = %d, want 3", res.Steps)
	}
	line := FormatBusTraceLine(res)
	if !strings.Contains(line, "SEG-AA BURNOUT") {
		t.Fatalf("trace line %q should name the segment", line)
	}
}

func TestTraceBusFault_openRelay(t *testing.T) {
	g, grid := makeTraceGrid(t)
	gameworld.GetGameData(grid.GetCell(0, 3)).PowerRelay = entities.NewPowerRelayOpen()

	res := TraceBusFault(g, grid.GetCell(0, 0), "RoomB")
	if res.Kind != PowerFaultOpenRelay {
		t.Fatalf("kind = %q, want open relay", res.Kind)
	}
}

func TestTraceBusFault_noSupplyAndUnarmed(t *testing.T) {
	g, grid := makeTraceGrid(t)

	res := TraceBusFault(g, grid.GetCell(0, 0), "RoomB")
	if res.Kind != PowerFaultNoSupply {
		t.Fatalf("kind = %q, want no supply (no generators)", res.Kind)
	}

	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 1)).Generator = gen
	res = TraceBusFault(g, grid.GetCell(0, 0), "RoomB")
	if res.Kind != PowerFaultUnarmed {
		t.Fatalf("kind = %q, want unarmed (intact bus, circuits off)", res.Kind)
	}
}

func TestTraceBusFault_poweredRoomIsBusOK(t *testing.T) {
	g, grid := makeTraceGrid(t)
	g.RoomPowerOnline = map[string]bool{"RoomB": true}

	res := TraceBusFault(g, grid.GetCell(0, 0), "RoomB")
	if res.Kind != PowerFaultNone {
		t.Fatalf("kind = %q, want none for powered room", res.Kind)
	}
	if !strings.Contains(FormatBusTraceLine(res), "BUS OK") {
		t.Fatal("powered room should format as BUS OK")
	}
}

func TestConduitSplice_conductsOnlyWhenComplete(t *testing.T) {
	g, grid := makeTraceGrid(t)
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen

	splice := entities.NewRepairObjective("c1", entities.RepairConduitSplice, "Corridor", 0, 3)
	gameworld.GetGameData(grid.GetCell(0, 3)).RepairDevice = splice

	far := grid.GetCell(0, 4)
	if CellsReachableFromPoweredGenerators(g).Has(far) {
		t.Fatal("burned conduit should block conduction past the fault")
	}

	splice.Complete()
	g.InvalidateLivePowerCache()
	if !CellsReachableFromPoweredGenerators(g).Has(far) {
		t.Fatal("completed conduit splice should conduct again")
	}
}

func TestRepairDevice_nonConduitAlwaysBlocksGrid(t *testing.T) {
	g, grid := makeTraceGrid(t)
	gen := entities.NewGenerator("G", 1)
	gen.InsertBatteriesAndStart(1)
	gameworld.GetGameData(grid.GetCell(0, 0)).Generator = gen

	valve := entities.NewRepairObjective("v1", entities.RepairPressureValve, "Corridor", 0, 3)
	valve.Complete()
	gameworld.GetGameData(grid.GetCell(0, 3)).RepairDevice = valve

	if CellsReachableFromPoweredGenerators(g).Has(grid.GetCell(0, 4)) {
		t.Fatal("non-conduit repair housings should block conduction even when complete")
	}
}
