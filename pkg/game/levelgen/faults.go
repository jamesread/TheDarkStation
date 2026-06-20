package levelgen

import (
	"fmt"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/levelrand"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// PlaceConduitFaults seeds burned power-conduit segments (grid faults) on straight
// corridor conduits. A faulted cell stops conducting power — everything beyond it on
// that branch goes dark — and is repaired in place with a hold-USE conduit splice.
// Faults gate the exit lift like other deck repairs.
//
// The splice sits in the floor channel and stays walkable: it interrupts power, not
// movement, so it never severs routing and needs no blocking-entity validation.
func PlaceConduitFaults(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	if g == nil || g.Grid == nil || setup.PlayerEntryCell(g) == nil {
		return
	}
	count := conduitFaultCount(g.Level)
	if count == 0 {
		return
	}
	if avoid == nil {
		empty := mapset.New[*world.Cell]()
		avoid = &empty
	}

	candidates := collectConduitFaultCandidates(g, avoid)
	if len(candidates) == 0 {
		return
	}
	rng := levelrand.NewDerived(g.LevelSeed, 0xc0d01f)
	rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	placed := 0
	for _, cell := range candidates {
		if placed >= count {
			break
		}
		// Re-validate against the current grid: earlier faults in this pass can
		// invalidate later candidates (jointly sealing a region).
		if !validConduitFaultCell(g, cell, avoid) {
			continue
		}
		id := fmt.Sprintf("deck%d-conduit%d", g.CurrentDeckID, placed+1)
		repair := entities.NewRepairObjective(id, entities.RepairConduitSplice, cell.Name, cell.Row, cell.Col)
		repair.SegmentLabel = conduitSegmentLabel(cell)
		repair.Name = fmt.Sprintf("Conduit Splice %s", repair.SegmentLabel)
		gameworld.GetGameData(cell).RepairDevice = repair
		avoid.Put(cell)
		g.RepairObjectives = append(g.RepairObjectives, repair)
		g.InvalidateLivePowerCache()
		placed++
	}
}

// conduitFaultCount scales faults with depth; deck 1 and the final deck stay clean.
func conduitFaultCount(level int) int {
	if level < 2 || deck.IsFinalDeck(level) {
		return 0
	}
	switch {
	case level >= 8:
		return 3
	case level >= 5:
		return 2
	default:
		return 1
	}
}

// conduitSegmentLabel derives a stable diegetic segment name from the cell position.
// The same label shows up in maintenance terminal bus traces and on-cell callouts.
func conduitSegmentLabel(cell *world.Cell) string {
	return fmt.Sprintf("SEG-%02X", (cell.Row*31+cell.Col*7)%256)
}

// collectConduitFaultCandidates returns corridor cells where a fault is meaningful:
// straight conduit runs (exactly two corridor-side neighbors) that currently carry
// live power, away from the lift shaft.
func collectConduitFaultCandidates(g *state.Game, avoid *mapset.Set[*world.Cell]) []*world.Cell {
	var out []*world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if !validConduitFaultCell(g, cell, avoid) {
			return
		}
		out = append(out, cell)
	})
	setup.SortCellsByPosition(out)
	return out
}

func validConduitFaultCell(g *state.Game, cell *world.Cell, avoid *mapset.Set[*world.Cell]) bool {
	if cell == nil || !cell.Room || cell.Name != "Corridor" || avoid.Has(cell) {
		return false
	}
	if cell == setup.PlayerEntryCell(g) || cell.ExitCell {
		return false
	}
	data := gameworld.GetGameData(cell)
	if data.Generator != nil || data.Door != nil || data.Terminal != nil || data.Puzzle != nil ||
		data.Furniture != nil || data.Hazard != nil || data.HazardControl != nil ||
		data.MaintenanceTerm != nil || data.PowerRelay != nil || data.RepairDevice != nil ||
		data.RepairBlocker != nil || cell.ItemsOnFloor.Size() > 0 || data.EnvPlaqueMsgID != "" {
		return false
	}
	// Straight run: exactly two walkable neighbors, so the fault severs one branch
	// without disabling a junction.
	if roomNeighborCount(cell) != 2 {
		return false
	}
	// Only fault segments that currently conduct: the outage must be observable.
	if !setup.CellHasLivePower(g, cell) {
		return false
	}
	// Don't fault next to a door or the shaft (keeps entries diagnosable on foot).
	for _, n := range cell.GetNeighbors() {
		if n == nil {
			continue
		}
		if gameworld.HasDoor(n) || n.ExitCell {
			return false
		}
	}
	return true
}

func roomNeighborCount(cell *world.Cell) int {
	count := 0
	for _, n := range cell.GetNeighbors() {
		if n != nil && n.Room {
			count++
		}
	}
	return count
}
