package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestDeck1_shipHasFusionReactorAndNoProceduralEntities(t *testing.T) {
	g := BuildGame(1)

	fusionCell := g.Grid.GetCell(generator.Deck1FusionReactorRow, generator.Deck1FusionReactorCol)
	if fusionCell == nil {
		t.Fatal("fusion reactor cell missing")
	}
	gen := gameworld.GetGameData(fusionCell).Generator
	if gen == nil || !gen.Permanent || !gen.IsPowered() {
		t.Fatalf("fusion reactor = %+v, want permanent powered generator", gen)
	}
	if gen.Name != generator.ShipFusionReactorName {
		t.Fatalf("fusion reactor name = %q, want %q", gen.Name, generator.ShipFusionReactorName)
	}

	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || cell.Name != generator.ShipRoomName {
			return
		}
		data := gameworld.GetGameData(cell)
		if cell == fusionCell {
			return
		}
		if data.Furniture != nil && data.Furniture.PowerConduit {
			return
		}
		if data.Generator != nil || data.Furniture != nil || data.RepairDevice != nil ||
			data.Hazard != nil || data.Puzzle != nil || data.MaintenanceTerm != nil ||
			cell.ItemsOnFloor.Size() > 0 {
			t.Fatalf("unexpected entity in Ship at x:%d y:%d", col, row)
		}
	})
}

func TestDeck1_emergencyConduitsLinkFusionToShaftGenerator(t *testing.T) {
	g := BuildGame(1)

	fusionCell := g.Grid.GetCell(generator.Deck1FusionReactorRow, generator.Deck1FusionReactorCol)
	shaftGenCell := findShaftGeneratorCell(g)
	if fusionCell == nil || shaftGenCell == nil {
		t.Fatal("fusion or shaft generator cell missing")
	}

	conduitCount := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		furn := gameworld.GetGameData(cell).Furniture
		if furn == nil || !furn.PowerConduit {
			return
		}
		conduitCount++
		if furn.Name != "Emergency power conduit" {
			t.Fatalf("conduit at x:%d y:%d name = %q", col, row, furn.Name)
		}
		if gameworld.FurnitureBlocksMovement(cell) || gameworld.FurnitureBlocksPowerGrid(cell) {
			t.Fatalf("conduit at x:%d y:%d should be walkable and conductive", col, row)
		}
	})
	if conduitCount == 0 {
		t.Fatal("expected emergency power conduits between ship reactor and lift shaft generator")
	}

	if !setup.RoomsOnConductiveGeneratorGrid(g)[generator.ShipRoomName] {
		t.Fatal("ship room should be on conductive generator grid via emergency feed")
	}
}

func findShaftGeneratorCell(g *state.Game) *world.Cell {
	var found *world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if found != nil || cell == nil || cell.Name != generator.ShaftRoomName {
			return
		}
		if gameworld.GetGameData(cell).Generator != nil {
			found = cell
		}
	})
	return found
}

func TestDeck1_freshRunSpawnsInShip(t *testing.T) {
	g := BuildGame(1)
	if g.CurrentCell == nil || g.CurrentCell.Name != generator.ShipRoomName {
		t.Fatalf("player room = %q, want Ship", g.CurrentCell.Name)
	}
	start := g.Grid.StartCell()
	if start == nil {
		t.Fatal("missing start cell")
	}
	if g.CurrentCell != start {
		t.Fatalf("player at x:%d y:%d, want start x:%d y:%d",
			g.CurrentCell.Col, g.CurrentCell.Row, start.Col, start.Row)
	}
	if g.PlayerFacing != state.FaceSouth {
		t.Fatalf("PlayerFacing = %v, want FaceSouth", g.PlayerFacing)
	}
}

func TestDeck1_shipAlwaysPowered(t *testing.T) {
	g := BuildGame(1)
	room := generator.ShipRoomName
	if !g.RoomDoorsPowered[room] || !g.RoomCCTVPowered[room] || !g.RoomLightsPowered[room] {
		t.Fatalf("%q should be fully armed at run start", room)
	}
	if !setup.RoomConsideredPowered(g, room) {
		t.Fatalf("%q should read as powered at run start", room)
	}
}

func TestDeck1_liftReturnSpawnsOnLiftExit(t *testing.T) {
	g := BuildGame(1)
	unlockAllDecksForTest(g)
	exit := g.Grid.ExitCell()
	if exit == nil {
		t.Fatal("deck 1 missing lift exit cell")
	}
	if err := TravelToDeck(g, 2); err != nil {
		t.Fatalf("TravelToDeck(2): %v", err)
	}
	if err := TravelToDeck(g, 1); err != nil {
		t.Fatalf("TravelToDeck(1): %v", err)
	}
	if g.CurrentCell != exit {
		t.Fatalf("lift return: player at (%d,%d) %q, want lift exit (%d,%d) %q",
			g.CurrentCell.Row, g.CurrentCell.Col, g.CurrentCell.Name,
			exit.Row, exit.Col, exit.Name)
	}
}

func TestDeck1_seed2SimulatesSolvable(t *testing.T) {
	g := BuildGame(1)
	LoadLevelFromSeed(g, 2)
	report := setup.SimulatePlaythrough(g)
	if !report.Solvable {
		t.Fatalf("unsolvable: %v", report.Failures)
	}
}

func TestPermanentFusionReactor_neverTrips(t *testing.T) {
	gen := entities.NewPermanentFusionReactor(generator.ShipFusionReactorName)
	gen.Trip()
	if !gen.IsPowered() {
		t.Fatal("permanent fusion reactor should stay powered after Trip")
	}
}
