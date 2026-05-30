package gameplay

import (
	"testing"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/levelgen"
	"darkstation/pkg/game/levelseed"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

const mapTxtSeed = "18B3FFAA8EC45C0C"

// Regression for map.txt seed 18B3FFAA8EC45C0C: gas at x:37 y:30 had vent in unreachable Emergency Core Junction.
func TestEnsureHazardControlsSolvable_mapTxtSeed(t *testing.T) {
	seed, err := levelseed.Parse(mapTxtSeed)
	if err != nil {
		t.Fatal(err)
	}
	g := state.NewGame()
	g.Level = 6
	LoadLevelFromSeed(g, seed)

	applyMapDumpDoorPower(g)
	player := g.Grid.GetCell(30, 40)

	var broken []*world.Cell
	for _, gas := range hazardCells(g) {
		data := gameworld.GetGameData(gas)
		if data.Hazard == nil || data.Hazard.RequiresItem() {
			continue
		}
		vent := ventControlForGas(g, gas)
		if vent == nil {
			t.Fatalf("missing vent for gas at x:%d y:%d", gas.Col, gas.Row)
		}
		if !playerCanUseVentWithDumpPower(g, player, vent, gas) {
			t.Logf("broken: gas x:%d y:%d vent x:%d y:%d (%s)", gas.Col, gas.Row, vent.Col, vent.Row, vent.Name)
			broken = append(broken, gas)
		}
	}
	if len(broken) == 0 {
		t.Skip("no player-side unreachable vents on fresh seed layout")
	}

	levelgen.EnsureHazardControlsSolvable(g)

	for _, gas := range broken {
		vent := ventControlForGas(g, gas)
		if vent == nil {
			t.Fatalf("missing vent after fix for gas x:%d y:%d", gas.Col, gas.Row)
		}
		t.Logf("after fix: gas x:%d y:%d vent x:%d y:%d (%s)", gas.Col, gas.Row, vent.Col, vent.Row, vent.Name)
		if !playerCanUseVentWithDumpPower(g, player, vent, gas) {
			t.Fatalf("player at x:40 y:30 still cannot reach vent for gas x:%d y:%d", gas.Col, gas.Row)
		}
	}
}

func TestEnsureHazardControlsSolvable_corridorGasTrap(t *testing.T) {
	seed, err := levelseed.Parse(mapTxtSeed)
	if err != nil {
		t.Fatal(err)
	}
	g := state.NewGame()
	g.Level = 6
	LoadLevelFromSeed(g, seed)

	gasCell := g.Grid.GetCell(30, 37)
	if gasCell == nil || !gasCell.Room {
		t.Fatal("missing corridor cell at x:37 y:30")
	}
	// Clear any existing entity and install the broken layout from map.txt.
	gameworld.GetGameData(gasCell).Hazard = nil
	gameworld.GetGameData(gasCell).HazardControl = nil
	hazard := entities.NewHazard(entities.HazardGas)
	gameworld.GetGameData(gasCell).Hazard = hazard

	oldVent := g.Grid.GetCell(23, 28)
	if oldVent != nil {
		gameworld.GetGameData(oldVent).HazardControl = nil
	}
	control := entities.NewHazardControl(entities.HazardGas, hazard)
	gameworld.GetGameData(g.Grid.GetCell(23, 28)).HazardControl = control

	applyMapDumpDoorPower(g)
	player := g.Grid.GetCell(30, 40)
	if playerCanUseVentWithDumpPower(g, player, g.Grid.GetCell(23, 28), gasCell) {
		t.Fatal("precondition: vent should be unreachable from player before fix")
	}

	levelgen.EnsureHazardControlsSolvable(g)
	vent := ventControlForGas(g, gasCell)
	if vent == nil {
		t.Fatal("missing vent after fix")
	}
	t.Logf("relocated vent to x:%d y:%d (%s)", vent.Col, vent.Row, vent.Name)
	if !playerCanUseVentWithDumpPower(g, player, vent, gasCell) {
		t.Fatalf("player still cannot reach vent at x:%d y:%d", vent.Col, vent.Row)
	}
}

func gasAt(g *state.Game, row, col int) *world.Cell {
	cell := g.Grid.GetCell(row, col)
	if cell == nil || !gameworld.HasBlockingHazard(cell) {
		return nil
	}
	return cell
}

func ventControlFor(g *state.Game, control *entities.HazardControl) *world.Cell {
	var found *world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if found != nil || cell == nil {
			return
		}
		if gameworld.GetGameData(cell).HazardControl == control {
			found = cell
		}
	})
	return found
}

func ventControlForGas(g *state.Game, gas *world.Cell) *world.Cell {
	if gas == nil {
		return nil
	}
	h := gameworld.GetGameData(gas).Hazard
	if h == nil || h.Control == nil {
		return nil
	}
	return ventControlFor(g, h.Control)
}

func playerCanUseVentWithDumpPower(g *state.Game, player, vent, gas *world.Cell) bool {
	reach := bfsCanEnterGameplay(g, player)
	for _, n := range vent.GetNeighbors() {
		if reach.Has(n) {
			return true
		}
	}
	block := mapset.New[*world.Cell]()
	block.Put(gas)
	for _, n := range vent.GetNeighbors() {
		if pathExistsGameplay(g, player, n, &block) {
			return true
		}
	}
	return false
}

func applyMapDumpDoorPower(g *state.Game) {
	powered := []string{
		"Abandoned Infrastructure Node 28", "Damaged Central Monitoring",
		"Damaged Control Conduit", "Damaged Maintenance Conduit",
		"Depressurized Maintenance Conduit", "Derelict Core Access",
		"Derelict Infrastructure Node", "Emergency Command Node",
		"Emergency Infrastructure Node", "Isolated Command Node",
		"Isolated Core Access", "Overgrown Command Node 23",
		"Overgrown Infrastructure Node", "Overgrown Primary Hub",
		"Overgrown Primary Hub 26", "Sealed Primary Conduit", "Sealed Station Spine",
	}
	for room := range g.RoomDoorsPowered {
		g.RoomDoorsPowered[room] = false
		g.RoomCCTVPowered[room] = false
	}
	for _, room := range powered {
		g.RoomDoorsPowered[room] = true
		g.RoomCCTVPowered[room] = true
	}
	for _, gen := range g.Generators {
		switch gen.Name {
		case "Generator #1", "Generator #3", "Generator #4":
			gen.InsertBatteriesAndStart(gen.BatteriesRequired - gen.BatteriesInserted)
		}
	}
	setup.PropagateRoomPowerOnlineFromGenerators(g)
}

func bfsCanEnterGameplay(g *state.Game, start *world.Cell) *mapset.Set[*world.Cell] {
	reach := mapset.New[*world.Cell]()
	if start == nil {
		return &reach
	}
	queue := []*world.Cell{start}
	reach.Put(start)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, n := range cur.GetNeighbors() {
			if n == nil || reach.Has(n) {
				continue
			}
			ok, _ := CanEnter(g, n, false)
			if !ok {
				continue
			}
			reach.Put(n)
			queue = append(queue, n)
		}
	}
	return &reach
}

func pathExistsGameplay(g *state.Game, start, goal *world.Cell, block *mapset.Set[*world.Cell]) bool {
	if start == nil || goal == nil {
		return false
	}
	seen := mapset.New[*world.Cell]()
	queue := []*world.Cell{start}
	seen.Put(start)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == goal {
			return true
		}
		for _, n := range cur.GetNeighbors() {
			if n == nil || seen.Has(n) || block.Has(n) {
				continue
			}
			ok, _ := CanEnter(g, n, false)
			if !ok {
				continue
			}
			seen.Put(n)
			queue = append(queue, n)
		}
	}
	return false
}
