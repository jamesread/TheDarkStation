package gameplay

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/levelseed"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
)

// Regression for map.txt seed 18B813F8ED655BF5 (deck 9): the blocking Coolant Shutoff
// control was placed on the single corridor cell in front of the west room's only door.
// The door was locked, so the articulation-point check (computed with locked doors
// impassable) did not flag the cell; once unlocked with its on-deck keycard, the room —
// hosting two exit-gating repairs — was permanently walled off, soft-locking the deck.
func TestHazardControlNotOnDoorOpenableChokepoint_mapTxtSeed(t *testing.T) {
	levelSeed, err := levelseed.Parse("18B813F8ED655BF5")
	if err != nil {
		t.Fatal(err)
	}
	g := state.NewGame()
	g.InitRunUnlocks(levelSeed - 8*9973)
	g.Level = 9
	RegenerateFromSeed(g, levelSeed)

	reach := completionReachable(g)
	for _, repair := range g.RepairObjectives {
		if repair == nil || repair.SkipExitGate || repair.DeviceRow < 0 {
			continue
		}
		cell := g.Grid.GetCell(repair.DeviceRow, repair.DeviceCol)
		if cell == nil {
			t.Errorf("repair %q device cell missing at x:%d y:%d", repair.Name, repair.DeviceCol, repair.DeviceRow)
			continue
		}
		standable := false
		for _, n := range cell.GetNeighbors() {
			if n != nil && reach[n] {
				standable = true
				break
			}
		}
		if !standable {
			t.Errorf("exit-gating repair %q at x:%d y:%d in %q has no completion-reachable stand cell"+
				" (permanently walled off from the lift entry)", repair.Name, cell.Col, cell.Row, cell.Name)
		}
	}
}

// completionReachable returns cells walkable from the lift entry assuming locked doors are
// eventually opened and hazards cleared; only permanent blockers stop movement.
func completionReachable(g *state.Game) map[*world.Cell]bool {
	out := make(map[*world.Cell]bool)
	entry := setup.PlayerEntryCell(g)
	if entry == nil {
		return out
	}
	queue := []*world.Cell{entry}
	out[entry] = true
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, n := range cur.GetNeighbors() {
			if n == nil || !n.Room || out[n] || setup.IsPermanentlyBlockingCell(n) {
				continue
			}
			out[n] = true
			queue = append(queue, n)
		}
	}
	return out
}
