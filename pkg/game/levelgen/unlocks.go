package levelgen

import (
	"fmt"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	"darkstation/pkg/game/unlocks"
	gameworld "darkstation/pkg/game/world"
)

// PlaceUnlockObjectives adds cross-deck routing repairs and security keycard payoffs for this deck.
func PlaceUnlockObjectives(g *state.Game, avoid *mapset.Set[*world.Cell]) {
	if g == nil || g.Grid == nil || g.UnlockPlan == nil {
		return
	}
	if avoid == nil {
		empty := mapset.New[*world.Cell]()
		avoid = &empty
	}
	finalRepair := finalExitGatingRepair(g)
	usedCouplerCells := make(map[*world.Cell]bool)

	for _, req := range g.UnlockPlan.ForSource(g.CurrentDeckID) {
		switch req.Kind {
		case unlocks.KindRoutingRepair:
			placeRoutingRepair(g, req, avoid, finalRepair, usedCouplerCells)
		case unlocks.KindSecurityKeycard:
			registerKeycardPayoff(g, req, finalRepair)
		}
	}
}

func finalExitGatingRepair(g *state.Game) *entities.RepairObjective {
	if g == nil {
		return nil
	}
	var last *entities.RepairObjective
	for _, repair := range g.RepairObjectives {
		if repair == nil || repair.SkipExitGate {
			continue
		}
		last = repair
	}
	return last
}

func placeRoutingRepair(g *state.Game, req unlocks.Requirement, avoid *mapset.Set[*world.Cell], finalRepair *entities.RepairObjective, usedCouplerCells map[*world.Cell]bool) {
	if g.RepairByID(req.RepairID) != nil {
		return
	}
	cell := pickRoutingCouplerCell(g, avoid, finalRepair, usedCouplerCells, false)
	if cell == nil {
		return
	}
	usedCouplerCells[cell] = true
	repair := entities.NewRepairObjective(req.RepairID, entities.RepairSignalCalibrator, cell.Name, cell.Row, cell.Col)
	repair.Name = unlocks.RoutingRepairName(req.TargetDeckID + 1)
	repair.SkipExitGate = true
	repair.TargetDeckID = req.TargetDeckID
	repair.RequiresPower = true
	if finalRepair != nil {
		repair.PrereqIDs = []string{finalRepair.ID}
	}
	gameworld.GetGameData(cell).RepairDevice = repair
	g.RepairObjectives = append(g.RepairObjectives, repair)
	avoid.Put(cell)
	g.AddHint(fmt.Sprintf("Lift routing payoff: %s in %s", renderer.StyledItem(repair.Name), renderer.StyledCell(cell.Name)))
}

func registerKeycardPayoff(g *state.Game, req unlocks.Requirement, finalRepair *entities.RepairObjective) {
	if g == nil {
		return
	}
	if g.HasRunKeycard(req.KeycardName) {
		return
	}
	g.AddHint(fmt.Sprintf("Completing deck systems may yield: KEYCARD{%s}", req.KeycardName))
	if finalRepair != nil && finalRepair.DeviceRow >= 0 && finalRepair.DeviceCol >= 0 {
		if cell := g.Grid.GetCell(finalRepair.DeviceRow, finalRepair.DeviceCol); cell != nil {
			gameworld.GetGameData(cell).PendingUnlockKeycard = req.KeycardName
		}
	}
}

// SpawnUnlockKeycardPayoffs drops registered keycards when local exit-gating repairs are complete.
func SpawnUnlockKeycardPayoffs(g *state.Game) {
	if g == nil || g.Grid == nil || g.IncompleteRepairCount() > 0 {
		return
	}
	setup.DropPendingUnlockKeycards(g)
}

// SyncKeycardPayoffRegistration rebinds pending unlock keycards to the current final
// exit-gating repair after later placement passes (e.g. conduit faults) or relocation.
func SyncKeycardPayoffRegistration(g *state.Game) {
	if g == nil || g.Grid == nil || g.UnlockPlan == nil {
		return
	}
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		gameworld.GetGameData(cell).PendingUnlockKeycard = ""
	})
	finalRepair := finalExitGatingRepair(g)
	for _, req := range g.UnlockPlan.ForSource(g.CurrentDeckID) {
		if req.Kind != unlocks.KindSecurityKeycard {
			continue
		}
		registerKeycardPayoff(g, req, finalRepair)
	}
}

func pickRoutingCouplerCell(g *state.Game, avoid *mapset.Set[*world.Cell], finalRepair *entities.RepairObjective, usedCouplerCells map[*world.Cell]bool, requirePowered bool) *world.Cell {
	_ = finalRepair
	return pickUnlockPlacementCell(g, avoid, usedCouplerCells, requirePowered)
}

func pickUnlockPlacementCell(g *state.Game, avoid *mapset.Set[*world.Cell], usedCouplerCells map[*world.Cell]bool, requirePowered bool) *world.Cell {
	candidates := collectRoutingCouplerCandidates(g, avoid, usedCouplerCells, requirePowered, false)
	if len(candidates) == 0 {
		return nil
	}
	entry := setup.PlayerEntryCell(g)
	dist := pathDistancesAtInit(g, entry)
	var best *world.Cell
	bestDist := -1
	for _, cell := range candidates {
		d := dist[cell]
		if d > bestDist {
			bestDist = d
			best = cell
		}
	}
	return best
}

func collectRoutingCouplerCandidates(g *state.Game, avoid *mapset.Set[*world.Cell], usedCouplerCells map[*world.Cell]bool, requirePowered, lightValidation bool) []*world.Cell {
	if g == nil || g.Grid == nil {
		return nil
	}
	entry := setup.PlayerEntryCell(g)
	reachable := setup.InitialReachableCells(g)
	var out []*world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if !routingCouplerCandidateCell(g, cell, avoid, usedCouplerCells, requirePowered, entry, reachable) {
			return
		}
		if lightValidation {
			if !setup.CandidateBlockingCellHasAdjacentNavSpace(g, cell, avoid) ||
				!setup.ProgressionNavPreservedByPlacement(g, cell) {
				return
			}
		} else if !setup.CanPlaceBlockingEntity(g, cell) {
			return
		}
		out = append(out, cell)
	})
	return out
}

func routingCouplerCandidateCell(g *state.Game, cell *world.Cell, avoid *mapset.Set[*world.Cell], usedCouplerCells map[*world.Cell]bool, requirePowered bool, entry *world.Cell, reachable *mapset.Set[*world.Cell]) bool {
	if cell == nil || !cell.Room || avoid.Has(cell) {
		return false
	}
	if cell == entry || cell.Name == "Corridor" || generator.IsPlacementExcludedRoom(cell.Name) {
		return false
	}
	if usedCouplerCells != nil && usedCouplerCells[cell] {
		return false
	}
	if reachable == nil {
		return false
	}
	// Accept rooms reachable at init, or whose doors can be armed from the lift-entry
	// pocket (same accessibility contract as exit-gating repairs); on small decks the
	// only spare room may sit behind a door the player powers from the shaft terminal.
	if !reachable.Has(cell) && !setup.CanPowerRoomDoorsFromReachable(g, reachable, cell.Name) {
		return false
	}
	if requirePowered && !setup.RoomConsideredPowered(g, cell.Name) {
		return false
	}
	data := gameworld.GetGameData(cell)
	if data.Generator != nil || data.Door != nil || data.Terminal != nil || data.Puzzle != nil ||
		data.Furniture != nil || data.Hazard != nil || data.HazardControl != nil ||
		data.MaintenanceTerm != nil || data.PowerRelay != nil || data.RepairDevice != nil ||
		data.RepairBlocker != nil || cell.ItemsOnFloor.Size() > 0 {
		return false
	}
	return true
}

func exitGatingRepairRooms(g *state.Game) map[string]bool {
	rooms := make(map[string]bool)
	for _, repair := range g.RepairObjectives {
		if repair == nil || repair.SkipExitGate || repair.RoomName == "" {
			continue
		}
		rooms[repair.RoomName] = true
	}
	return rooms
}

// EnsureRoutingCouplerNavAccess relocates unlock routing couplers that block progression interactables.
func EnsureRoutingCouplerNavAccess(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	armRoutingCouplerRoomPower(g)
	if setup.ProgressionNavPreserved(g, nil) && routingCouplersViable(g) {
		return
	}
	avoid := mapset.New[*world.Cell]()
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.Generator != nil || data.Door != nil || data.RepairDevice != nil ||
			data.HazardControl != nil || data.MaintenanceTerm != nil {
			avoid.Put(cell)
		}
	})
	for attempt := 0; attempt < 64; attempt++ {
		if setup.ProgressionNavPreserved(g, nil) && routingCouplersViable(g) {
			return
		}
		if !relocateOneBlockingRoutingCoupler(g, &avoid) {
			return
		}
	}
}

// armRoutingCouplerRoomPower arms door power for rooms hosting powered routing couplers when
// those rooms can be controlled from the entry pocket (e.g. via the lift shaft bootstrap
// terminal), so couplers are actionable without relocation.
func armRoutingCouplerRoomPower(g *state.Game) {
	if g.RoomDoorsPowered == nil {
		g.RoomDoorsPowered = make(map[string]bool)
	}
	changed := false
	for _, repair := range g.RepairObjectives {
		if repair == nil || !repair.SkipExitGate || repair.DeviceRow < 0 || repair.DeviceCol < 0 {
			continue
		}
		cell := g.Grid.GetCell(repair.DeviceRow, repair.DeviceCol)
		if cell == nil || cell.Name == "" || cell.Name == "Corridor" {
			continue
		}
		if g.RoomDoorsPowered[cell.Name] || setup.RoomConsideredPowered(g, cell.Name) {
			continue
		}
		reach := setup.InitialReachableCells(g)
		if !setup.CanPowerRoomDoorsFromReachable(g, reach, cell.Name) {
			continue
		}
		g.RoomDoorsPowered[cell.Name] = true
		changed = true
	}
	if changed {
		g.InvalidateLivePowerCache()
		setup.PropagateRoomPowerOnlineFromGenerators(g)
	}
}

func routingCouplersViable(g *state.Game) bool {
	for _, repair := range g.RepairObjectives {
		if repair == nil || !repair.SkipExitGate || repair.DeviceRow < 0 || repair.DeviceCol < 0 {
			continue
		}
		cell := g.Grid.GetCell(repair.DeviceRow, repair.DeviceCol)
		if cell == nil {
			continue
		}
		if !setup.RoomConsideredPowered(g, cell.Name) {
			return false
		}
	}
	return true
}

func relocateOneBlockingRoutingCoupler(g *state.Game, avoid *mapset.Set[*world.Cell]) bool {
	usedCells := routingCouplerCells(g)
	for _, repair := range g.RepairObjectives {
		if repair == nil || !repair.SkipExitGate || repair.DeviceRow < 0 || repair.DeviceCol < 0 {
			continue
		}
		cur := g.Grid.GetCell(repair.DeviceRow, repair.DeviceCol)
		if cur == nil {
			continue
		}
		gameworld.GetGameData(cur).RepairDevice = repair
		withPresent := setup.ProgressionNavPreserved(g, nil)
		gameworld.GetGameData(cur).RepairDevice = nil
		withoutPresent := setup.ProgressionNavPreserved(g, nil)
		needsRelocate := !routingCouplerViable(g, repair) || (!withPresent && withoutPresent)
		if !needsRelocate {
			gameworld.GetGameData(cur).RepairDevice = repair
			continue
		}
		delete(usedCells, cur)
		if avoid != nil {
			avoid.Remove(cur)
		}
		replacement := pickUnlockRelocationCell(g, avoid, usedCells)
		if replacement == nil {
			if avoid != nil {
				avoid.Put(cur)
			}
			gameworld.GetGameData(cur).RepairDevice = repair
			continue
		}
		repair.DeviceRow = replacement.Row
		repair.DeviceCol = replacement.Col
		repair.RoomName = replacement.Name
		gameworld.GetGameData(replacement).RepairDevice = repair
		avoid.Put(replacement)
		usedCells[replacement] = true
		return true
	}
	return false
}

func pickUnlockRelocationCell(g *state.Game, avoid *mapset.Set[*world.Cell], usedCouplerCells map[*world.Cell]bool) *world.Cell {
	candidates := collectRoutingCouplerCandidates(g, avoid, usedCouplerCells, true, true)
	if len(candidates) == 0 {
		candidates = collectRoutingCouplerCandidates(g, avoid, usedCouplerCells, false, true)
	}
	if len(candidates) == 0 {
		return nil
	}
	entry := setup.PlayerEntryCell(g)
	dist := pathDistancesAtInit(g, entry)
	var best *world.Cell
	bestDist := -1
	for _, cell := range candidates {
		d := dist[cell]
		if d > bestDist {
			bestDist = d
			best = cell
		}
	}
	return best
}

func routingCouplerCells(g *state.Game) map[*world.Cell]bool {
	cells := make(map[*world.Cell]bool)
	for _, repair := range g.RepairObjectives {
		if repair == nil || !repair.SkipExitGate || repair.DeviceRow < 0 || repair.DeviceCol < 0 {
			continue
		}
		cell := g.Grid.GetCell(repair.DeviceRow, repair.DeviceCol)
		if cell != nil {
			cells[cell] = true
		}
	}
	return cells
}

func routingCouplerRooms(g *state.Game) map[string]bool {
	rooms := make(map[string]bool)
	for _, repair := range g.RepairObjectives {
		if repair == nil || !repair.SkipExitGate || repair.RoomName == "" {
			continue
		}
		rooms[repair.RoomName] = true
	}
	return rooms
}

func routingCouplerViable(g *state.Game, repair *entities.RepairObjective) bool {
	if repair == nil || repair.DeviceRow < 0 || repair.DeviceCol < 0 {
		return true
	}
	cell := g.Grid.GetCell(repair.DeviceRow, repair.DeviceCol)
	if cell == nil {
		return false
	}
	return setup.RoomConsideredPowered(g, cell.Name)
}

func pathDistancesAtInit(g *state.Game, from *world.Cell) map[*world.Cell]int {
	if g == nil || from == nil {
		return nil
	}
	visited := map[*world.Cell]int{from: 0}
	queue := []*world.Cell{from}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, n := range c.GetNeighbors() {
			if n == nil || !n.Room {
				continue
			}
			if _, ok := visited[n]; ok {
				continue
			}
			if n != from {
				ok, _ := setup.CanEnterCellAtInit(g, n)
				if !ok {
					continue
				}
			}
			visited[n] = visited[c] + 1
			queue = append(queue, n)
		}
	}
	return visited
}
