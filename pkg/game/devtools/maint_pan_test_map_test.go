package devtools

import (
	"slices"
	"testing"

	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestSwitchToMaintPanTestMap_topologyAndMaintenance(t *testing.T) {
	g := state.NewGame()
	SwitchToMaintPanTestMap(g)
	if g.Grid == nil {
		t.Fatal("Grid is nil")
	}
	if g.Level != MaintPanTestLevel {
		t.Fatalf("Level = %d, want %d", g.Level, MaintPanTestLevel)
	}
	start := g.Grid.StartCell()
	if start == nil || start.Row != 11 || start.Col != 11 {
		t.Fatalf("unexpected start cell: %+v", start)
	}

	adj := setup.GetAdjacentRoomNames(g.Grid, RoomPanTestCenter)
	if adj == nil {
		t.Fatal("GetAdjacentRoomNames returned nil")
	}
	for _, want := range []string{RoomPanTestWest, RoomPanTestEast, RoomPanTestCenter} {
		if !slices.Contains(adj, want) {
			t.Errorf("adjacent rooms %v should contain %q", adj, want)
		}
	}

	term := g.Grid.GetCell(11, 12)
	if term == nil {
		t.Fatal("terminal cell missing")
	}
	mt := gameworld.GetGameData(term).MaintenanceTerm
	if mt == nil || !mt.Powered {
		t.Fatal("maintenance terminal should exist and start powered from InitMaintenanceTerminalPower")
	}
	if mt.RoomName != RoomPanTestCenter {
		t.Fatalf("terminal room %q, want %q", mt.RoomName, RoomPanTestCenter)
	}
	if g.CurrentCell.East != term {
		t.Fatal("player should stand immediately west of the terminal for interact-east workflow")
	}
	g.UpdatePowerSupply()
	if len(g.Generators) == 0 {
		t.Fatal("maint pan debug map should register synthetic generators for supply headroom")
	}
	for _, gen := range g.Generators {
		if !gen.IsPowered() {
			t.Fatalf("generator %q should be fully powered in debug maint map", gen.Name)
		}
	}
	if g.PowerSupply < 400 {
		t.Fatalf("PowerSupply=%d, expected bench generators (~500 W nominal) to dominate", g.PowerSupply)
	}
}
