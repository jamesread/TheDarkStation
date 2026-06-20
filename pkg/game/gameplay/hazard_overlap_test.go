package gameplay

import (
	"fmt"
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/levelgen"
	"darkstation/pkg/game/levelseed"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func hazardSolutionOverlaps(g *state.Game) []string {
	var out []string
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.HazardControl == nil {
			return
		}
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			if isHazardSolutionItemName(item.Name) {
				out = append(out, fmt.Sprintf("x:%d y:%d control=%q item=%q", col, row, data.HazardControl.Name, item.Name))
			}
		})
		if data.Furniture != nil && data.Furniture.ContainedItem != nil &&
			isHazardSolutionItemName(data.Furniture.ContainedItem.Name) {
			item := data.Furniture.ContainedItem
			out = append(out, fmt.Sprintf("x:%d y:%d control=%q item=%q (in furniture)", col, row, data.HazardControl.Name, item.Name))
		}
	})
	return out
}

func isHazardSolutionItemName(name string) bool {
	for _, info := range entities.HazardTypes {
		if info.RequiresItem && info.ItemName == name {
			return true
		}
	}
	return false
}

func findPatchKitCell(g *state.Game) *world.Cell {
	var found *world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if found != nil || cell == nil {
			return
		}
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			if found == nil && item != nil && item.Name == "Patch Kit" {
				found = cell
			}
		})
		if found != nil {
			return
		}
		f := gameworld.GetGameData(cell).Furniture
		if f != nil && f.ContainedItem != nil && f.ContainedItem.Name == "Patch Kit" {
			found = cell
		}
	})
	return found
}

// Regression for map.txt seed 18B42AD024167CA4: patch kit shared a cell with containment control.
func TestEnsureHazardSolutionsDisjoint_mapTxtSeed(t *testing.T) {
	seed, err := levelseed.Parse("18B42AD024167CA4")
	if err != nil {
		t.Fatal(err)
	}
	g := state.NewGame()
	g.InitRunUnlocks(seed)
	g.Level = 5
	RegenerateFromSeed(g, seed)

	if overlaps := hazardSolutionOverlaps(g); len(overlaps) > 0 {
		t.Fatalf("hazard solution item overlaps hazard control: %v", overlaps)
	}

	// The pinned seed produced a vacuum hazard under the original layout; with layout
	// changes the hazard mix may differ, so assert the patch-kit invariants only when an
	// item-requiring blocking hazard exists.
	var vacuum *world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if vacuum != nil || cell == nil {
			return
		}
		h := gameworld.GetGameData(cell).Hazard
		if h != nil && h.IsBlocking() && h.RequiresItem() {
			vacuum = cell
		}
	})
	if vacuum == nil {
		return
	}

	patch := findPatchKitCell(g)
	if patch == nil {
		t.Fatal("expected patch kit for vacuum hazard")
	}
	if gameworld.GetGameData(patch).HazardControl != nil {
		t.Fatalf("patch kit cell x:%d y:%d still has hazard control", patch.Col, patch.Row)
	}

	locked := lockedDoorCells(g)
	block := locked.clone()
	block.put(vacuum)
	reach := reachableWithBlocked(g, block)
	if !reach.has(patch) {
		t.Fatalf("patch kit at x:%d y:%d not reachable before vacuum x:%d y:%d", patch.Col, patch.Row, vacuum.Col, vacuum.Row)
	}
}

func TestEnsureHazardSolutionsDisjoint_relocatesConflict(t *testing.T) {
	g, hazardCell, hazard := hazardClearTestGame()
	hazard.Type = entities.HazardVacuum
	hazard.Name = "Vacuum"
	patch := world.NewItem("Patch Kit")
	gameworld.GetGameData(hazardCell).Hazard = hazard

	ctrlCell := g.Grid.GetCell(0, 1)
	ctrlCell.ItemsOnFloor.Put(patch)
	control := entities.NewHazardControl(entities.HazardRadiation, entities.NewHazard(entities.HazardRadiation))
	gameworld.GetGameData(ctrlCell).HazardControl = control

	levelgen.EnsureHazardSolutionsDisjoint(g)

	if ctrlCell.ItemsOnFloor.Size() > 0 {
		t.Fatal("patch kit should move off control cell")
	}
	if findPatchKitCell(g) == nil {
		t.Fatal("patch kit should still exist on grid")
	}
	if gameworld.GetGameData(findPatchKitCell(g)).HazardControl != nil {
		t.Fatal("patch kit should not share cell with hazard control")
	}
}
