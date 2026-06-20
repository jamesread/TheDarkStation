package levelgen

import (
	"fmt"
	"sort"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

type repairCandidate struct {
	cell *world.Cell
	dist int
}

// PlaceRepairObjectives adds deck-wide repair chains that gate the exit lift.
func PlaceRepairObjectives(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	if g == nil || g.Grid == nil || setup.PlayerEntryCell(g) == nil {
		return
	}
	if avoid == nil {
		empty := mapset.New[*world.Cell]()
		avoid = &empty
	}
	g.RepairObjectives = nil

	candidates := collectExitGatingRepairCandidates(g, avoid)
	if len(candidates) == 0 {
		return
	}

	exitGate := PickExitGateKind(g.Level)
	types := repairChainForLevel(g.Level, exitGate)
	fractions := []float64{0.65, 0.25, 0.45, 0.85}
	var placed []*entities.RepairObjective
	usedRooms := make(map[string]bool)

	for i, typ := range types {
		id := fmt.Sprintf("deck%d-repair%d", g.CurrentDeckID, i+1)
		var cell *world.Cell
		var repair *entities.RepairObjective
		if typ == entities.RepairWastePump {
			repair = entities.NewRepairObjective(id, typ, "", -1, -1)
			cell = placeWastePumpObjective(g, repair, candidates, avoid)
			if cell == nil {
				continue
			}
		} else {
			cell = pickValidatedRepairCandidate(g, candidates, fractions[i%len(fractions)], usedRooms, avoid)
			if cell == nil {
				break
			}
			repair = entities.NewRepairObjective(id, typ, cell.Name, cell.Row, cell.Col)
		}
		if repair == nil {
			break
		}
		repair.RequiresPower = entities.TypeRequiresPower(typ)
		if len(placed) > 0 {
			repair.PrereqIDs = []string{placed[len(placed)-1].ID}
		}
		if typ == entities.RepairWastePump {
			for _, dep := range placed {
				repair.PrereqIDs = appendIfMissing(repair.PrereqIDs, dep.ID)
			}
		}

		gameworld.GetGameData(cell).RepairDevice = repair
		avoid.Put(cell)
		usedRooms[cell.Name] = true
		placed = append(placed, repair)
		g.AddHint(fmt.Sprintf("Repair the %s in %s", renderer.StyledItem(repair.Name), renderer.StyledCell(cell.Name)))
	}

	g.RepairObjectives = placed
}

func repairChainForLevel(level int, exitGate ExitGateKind) []entities.RepairType {
	var chain []entities.RepairType
	if level >= 6 {
		chain = []entities.RepairType{
			entities.RepairPressureValve,
			entities.RepairSignalCalibrator,
			entities.RepairPowerCoupler,
		}
	} else if level >= 3 {
		chain = []entities.RepairType{
			entities.RepairPressureValve,
			entities.RepairPowerCoupler,
		}
	} else {
		chain = []entities.RepairType{entities.RepairPressureValve}
	}
	if exitGate == ExitGateSlime {
		chain = append(chain, entities.RepairWastePump)
	}
	return chain
}

func collectRepairCandidates(g *state.Game, avoid *mapset.Set[*world.Cell]) []repairCandidate {
	return collectExitGatingRepairCandidates(g, avoid)
}

// collectExitGatingRepairCandidates returns device cells for local lift-gating repairs,
// limited to rooms reachable or powerable from the player entry pocket at init.
func collectExitGatingRepairCandidates(g *state.Game, avoid *mapset.Set[*world.Cell]) []repairCandidate {
	candidates := collectRepairCandidatesUnfiltered(g, avoid)
	filtered := filterExitGatingRepairCandidates(g, candidates)
	if len(filtered) > 0 {
		return filtered
	}
	return fallbackExitGatingRepairCandidates(g, avoid)
}

func collectRepairCandidatesUnfiltered(g *state.Game, avoid *mapset.Set[*world.Cell]) []repairCandidate {
	dist := distancesFromStart(g.Grid, setup.PlayerEntryCell(g))
	var preferred, fallback []repairCandidate
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if !validRepairDeviceCell(g, cell, avoid) {
			return
		}
		c := repairCandidate{cell: cell, dist: dist[cell]}
		if cell.Name != "Corridor" {
			preferred = append(preferred, c)
		} else {
			fallback = append(fallback, c)
		}
	})
	sortRepairCandidates(preferred)
	sortRepairCandidates(fallback)
	return append(preferred, fallback...)
}

func filterExitGatingRepairCandidates(g *state.Game, candidates []repairCandidate) []repairCandidate {
	if g == nil {
		return nil
	}
	var out []repairCandidate
	for _, candidate := range candidates {
		if candidate.cell == nil || !setup.ExitGatingRepairRoomAccessible(g, candidate.cell.Name) {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

func fallbackExitGatingRepairCandidates(g *state.Game, avoid *mapset.Set[*world.Cell]) []repairCandidate {
	if g == nil || g.Grid == nil {
		return nil
	}
	entry := setup.PlayerEntryCell(g)
	var out []repairCandidate
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !validRepairDeviceCell(g, cell, avoid) {
			return
		}
		if !setup.ExitGatingRepairRoomAccessible(g, cell.Name) {
			return
		}
		out = append(out, repairCandidate{cell: cell, dist: manhattan(cell, entry)})
	})
	sortRepairCandidates(out)
	return out
}

func validRepairDeviceCell(g *state.Game, cell *world.Cell, avoid *mapset.Set[*world.Cell]) bool {
	if cell == nil || !cell.Room || avoid.Has(cell) || cell == setup.PlayerEntryCell(g) {
		return false
	}
	if generator.IsPlacementExcludedRoom(cell.Name) || cell.ExitCell {
		return false
	}
	data := gameworld.GetGameData(cell)
	if data.Generator != nil || data.Door != nil || data.Terminal != nil || data.Puzzle != nil ||
		data.Furniture != nil || data.Hazard != nil || data.HazardControl != nil ||
		data.MaintenanceTerm != nil || data.PowerRelay != nil || data.RepairDevice != nil ||
		data.RepairBlocker != nil || cell.ItemsOnFloor.Size() > 0 {
		return false
	}
	return setup.CanPlaceBlockingEntity(g, cell)
}

// pickValidatedRepairCandidate picks repair device cells like pickRepairCandidate but
// re-validates each pick against the CURRENT grid state. The candidate list was
// collected (and validated) before any repair in this chain was placed; earlier
// placements in the same chain can invalidate later candidates (two individually
// legal devices may jointly seal off rooms), so each pick must pass
// CanPlaceBlockingEntity again at placement time.
func pickValidatedRepairCandidate(g *state.Game, candidates []repairCandidate, fraction float64, usedRooms map[string]bool, avoid *mapset.Set[*world.Cell]) *world.Cell {
	for {
		cell := pickRepairCandidate(candidates, fraction, usedRooms)
		if cell == nil {
			return nil
		}
		if validRepairDeviceCell(g, cell, avoid) {
			return cell
		}
		// pickRepairCandidate consumed this entry; try the next-best candidate.
	}
}

func pickRepairCandidate(candidates []repairCandidate, fraction float64, usedRooms map[string]bool) *world.Cell {
	if len(candidates) == 0 {
		return nil
	}
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	target := int(float64(len(candidates)-1) * fraction)
	for radius := 0; radius < len(candidates); radius++ {
		for _, idx := range []int{target - radius, target + radius} {
			if idx < 0 || idx >= len(candidates) {
				continue
			}
			cell := candidates[idx].cell
			if cell != nil && !usedRooms[cell.Name] {
				candidates[idx].cell = nil
				return cell
			}
		}
	}
	for i := range candidates {
		if candidates[i].cell != nil {
			cell := candidates[i].cell
			candidates[i].cell = nil
			return cell
		}
	}
	return nil
}

func placeWastePumpObjective(g *state.Game, repair *entities.RepairObjective, candidates []repairCandidate, avoid *mapset.Set[*world.Cell]) *world.Cell {
	if g == nil || repair == nil {
		return nil
	}
	exit := g.Grid.ExitCell()
	if exit == nil {
		return nil
	}
	for _, cells := range collectExitAdjacentBlockerGroups(g, exit, avoid) {
		pumpCell := pickRepairCandidateInRoom(g, candidates, cells[0].Name, cells, avoid)
		if pumpCell == nil {
			continue
		}
		if !setup.ExitGatingRepairRoomAccessible(g, pumpCell.Name) {
			continue
		}
		placeRepairBlockerCells(repair, cells, avoid)
		repair.RoomName = pumpCell.Name
		repair.DeviceRow = pumpCell.Row
		repair.DeviceCol = pumpCell.Col
		return pumpCell
	}
	return nil
}

func pickRepairCandidateInRoom(g *state.Game, candidates []repairCandidate, roomName string, exclude []*world.Cell, avoid *mapset.Set[*world.Cell]) *world.Cell {
	if roomName == "" {
		return nil
	}
	excluded := make(map[*world.Cell]bool, len(exclude))
	for _, cell := range exclude {
		if cell != nil {
			excluded[cell] = true
		}
	}
	for i := range candidates {
		cell := candidates[i].cell
		if cell == nil || cell.Name != roomName || excluded[cell] || !validRepairDeviceCellForAvoid(cell, avoid) {
			continue
		}
		// Re-validate against the current grid: earlier devices in this chain were
		// placed after the candidate list was collected.
		if !validRepairDeviceCell(g, cell, avoid) {
			continue
		}
		candidates[i].cell = nil
		return cell
	}
	return nil
}

func validRepairDeviceCellForAvoid(cell *world.Cell, avoid *mapset.Set[*world.Cell]) bool {
	return avoid == nil || !avoid.Has(cell)
}

func placeRepairBlockerCells(repair *entities.RepairObjective, cells []*world.Cell, avoid *mapset.Set[*world.Cell]) {
	sort.SliceStable(cells, func(i, j int) bool {
		if cells[i].Row != cells[j].Row {
			return cells[i].Row < cells[j].Row
		}
		return cells[i].Col < cells[j].Col
	})
	repair.BlockerName = "Toxic Slime"
	repair.BlockerCells = make([]entities.BlockerCell, len(cells))
	for i, cell := range cells {
		repair.BlockerCells[i] = entities.BlockerCell{Row: cell.Row, Col: cell.Col}
		gameworld.GetGameData(cell).RepairBlocker = repair
		avoid.Put(cell)
	}
	repair.BlockerRow = cells[0].Row
	repair.BlockerCol = cells[0].Col
}

func collectExitAdjacentBlockerGroups(g *state.Game, exit *world.Cell, avoid *mapset.Set[*world.Cell]) [][]*world.Cell {
	if g == nil || exit == nil {
		return nil
	}
	seen := make(map[*world.Cell]bool)
	byRoom := make(map[string][]*world.Cell)
	for _, cell := range exit.GetNeighbors() {
		if cell == nil || seen[cell] || !validRepairBlockerCell(g, cell, avoid) {
			continue
		}
		seen[cell] = true
		byRoom[cell.Name] = append(byRoom[cell.Name], cell)
	}
	if len(byRoom) == 0 {
		g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
			if cell == nil || seen[cell] || !validRepairBlockerCell(g, cell, avoid) {
				return
			}
			seen[cell] = true
			byRoom[cell.Name] = append(byRoom[cell.Name], cell)
		})
	}
	groups := make([][]*world.Cell, 0, len(byRoom))
	for _, cells := range byRoom {
		setup.SortCellsByPosition(cells)
		groups = append(groups, cells)
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if len(groups[i]) != len(groups[j]) {
			return len(groups[i]) > len(groups[j])
		}
		return manhattan(groups[i][0], exit) < manhattan(groups[j][0], exit)
	})
	return groups
}

func validRepairBlockerCell(g *state.Game, cell *world.Cell, avoid *mapset.Set[*world.Cell]) bool {
	if cell == nil || !cell.Room || cell == setup.PlayerEntryCell(g) || avoid.Has(cell) {
		return false
	}
	if generator.IsPlacementExcludedRoom(cell.Name) || cell.ExitCell {
		return false
	}
	data := gameworld.GetGameData(cell)
	return data.Generator == nil && data.Door == nil && data.Terminal == nil &&
		data.Puzzle == nil && data.Furniture == nil && data.Hazard == nil &&
		data.HazardControl == nil && data.MaintenanceTerm == nil && data.PowerRelay == nil &&
		data.RepairDevice == nil && data.RepairBlocker == nil && cell.ItemsOnFloor.Size() == 0
}

func distancesFromStart(grid *world.Grid, start *world.Cell) map[*world.Cell]int {
	dist := make(map[*world.Cell]int)
	if grid == nil || start == nil {
		return dist
	}
	queue := []*world.Cell{start}
	dist[start] = 0
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, n := range cur.GetNeighbors() {
			if n == nil || !n.Room {
				continue
			}
			if _, ok := dist[n]; ok {
				continue
			}
			dist[n] = dist[cur] + 1
			queue = append(queue, n)
		}
	}
	return dist
}

func sortRepairCandidates(candidates []repairCandidate) {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].dist != candidates[j].dist {
			return candidates[i].dist < candidates[j].dist
		}
		if candidates[i].cell.Row != candidates[j].cell.Row {
			return candidates[i].cell.Row < candidates[j].cell.Row
		}
		return candidates[i].cell.Col < candidates[j].cell.Col
	})
}

func appendIfMissing(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func manhattan(a, b *world.Cell) int {
	if a == nil || b == nil {
		return 0
	}
	dr := a.Row - b.Row
	if dr < 0 {
		dr = -dr
	}
	dc := a.Col - b.Col
	if dc < 0 {
		dc = -dc
	}
	return dr + dc
}
