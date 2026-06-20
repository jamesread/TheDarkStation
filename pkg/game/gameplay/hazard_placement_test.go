package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/levelseed"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func hazardStats(g *state.Game) (blocking int, controls int) {
	if g == nil || g.Grid == nil {
		return 0, 0
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Hazard != nil && data.Hazard.IsBlocking() {
			blocking++
		}
		if data.HazardControl != nil {
			controls++
		}
	})
	return blocking, controls
}

func TestPlaceHazards_level5SeedsUsuallyPlaceHazards(t *testing.T) {
	withHazards := 0
	for seed := int64(1); seed <= 30; seed++ {
		g := state.NewGame()
		g.Level = 5
		RegenerateFromSeed(g, seed)
		h, _ := hazardStats(g)
		if h > 0 {
			withHazards++
		}
	}
	if withHazards < 25 {
		t.Fatalf("expected hazards on most level-5 seeds, got %d/30 with hazards", withHazards)
	}
}

func TestPlaceHazards_mapTxtSeed_hasSolvableControls(t *testing.T) {
	seed, err := levelseed.Parse("18B36098DAD166BA")
	if err != nil {
		t.Fatal(err)
	}
	g := state.NewGame()
	g.Level = 5
	RegenerateFromSeed(g, seed)

	blocking, controls := hazardStats(g)
	if blocking == 0 {
		t.Fatal("expected at least one hazard on map.txt seed after fix")
	}

	locked := lockedDoorCells(g)
	for _, hazardCell := range hazardCells(g) {
		// Match EnsureHazardControlsSolvable: vent/control must be reachable without entering this hazard.
		blocked := locked.clone()
		blocked.put(hazardCell)
		reach := reachableWithBlocked(g, blocked)

		data := gameworld.GetGameData(hazardCell)
		if data.Hazard == nil || data.Hazard.RequiresItem() {
			continue
		}
		if data.Hazard.Control == nil {
			t.Fatalf("hazard at (%d,%d) missing linked control", hazardCell.Row, hazardCell.Col)
		}
		controlCell := controlCellFor(g, data.Hazard.Control)
		if controlCell == nil {
			t.Fatal("linked hazard control not found on grid")
		}
		if !reach.has(controlCell) {
			t.Fatalf("control at (%d,%d) not reachable before clearing hazard at (%d,%d)",
				controlCell.Row, controlCell.Col, hazardCell.Row, hazardCell.Col)
		}
	}

	if controls == 0 && blocking > 0 {
		// Vacuum-only layouts use items instead of controls.
		hasItemHazard := false
		for _, hazardCell := range hazardCells(g) {
			if gameworld.GetGameData(hazardCell).Hazard.RequiresItem() {
				hasItemHazard = true
				break
			}
		}
		if !hasItemHazard {
			t.Fatal("expected hazard controls for non-item hazards")
		}
	}
}

func hazardCells(g *state.Game) []*world.Cell {
	var out []*world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && gameworld.HasBlockingHazard(cell) {
			out = append(out, cell)
		}
	})
	return out
}

func controlCellFor(g *state.Game, control *entities.HazardControl) *world.Cell {
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

type blockedCells map[*world.Cell]struct{}

func (s blockedCells) put(c *world.Cell) { s[c] = struct{}{} }
func (s blockedCells) clone() blockedCells {
	out := blockedCells{}
	for k := range s {
		out[k] = struct{}{}
	}
	return out
}

func lockedDoorCells(g *state.Game) blockedCells {
	out := blockedCells{}
	if g == nil || g.Grid == nil {
		return out
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell != nil && gameworld.HasLockedDoor(cell) {
			out.put(cell)
		}
	})
	return out
}

type reachSet map[*world.Cell]struct{}

func (r reachSet) has(c *world.Cell) bool { _, ok := r[c]; return ok }

func reachableWithBlocked(g *state.Game, blocked blockedCells) reachSet {
	out := reachSet{}
	start := setup.PlayerEntryCell(g)
	if start == nil {
		return out
	}
	queue := []*world.Cell{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == nil || !cur.Room {
			continue
		}
		if _, ok := out[cur]; ok {
			continue
		}
		if _, impassable := blocked[cur]; impassable {
			continue
		}
		out[cur] = struct{}{}
		for _, n := range cur.GetNeighbors() {
			if n != nil && n.Room {
				queue = append(queue, n)
			}
		}
	}
	return out
}
