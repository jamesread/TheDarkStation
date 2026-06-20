// Package setup — generation-time progression simulation (the deck acceptance gate).
//
// SimulatePlaythrough runs a fixed-point "greedy player" over the generated deck:
// starting at the lift entry with nothing, it repeatedly expands the reachable set,
// collects every newly obtainable grant (floor items, furniture contents, room door
// power from maintenance terminals, generator startups, repair completions, hazard
// clears) and applies them, until no further progress is possible. The deck is
// accepted only if the exit lift can become ready (all hazards cleared, all repairs
// completable, exit reachable) and every named room is enterable.
//
// Extensibility contract: a new puzzle/blocker mechanic participates by
//  1. blocking movement via CanEnterCellAtInit / gameplay.CanEnter semantics
//     (mirrored in simPassable), and
//  2. exposing its "requires -> grants" step as an action in simStep.
//
// Anything that follows this contract is automatically covered by the acceptance
// gate and by the deterministic regenerate-and-retry in level generation.
package setup

import (
	"fmt"
	"sort"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/state"
	"darkstation/pkg/game/unlocks"
	gameworld "darkstation/pkg/game/world"
)

// SimReport is the outcome of a simulated playthrough.
type SimReport struct {
	Solvable   bool
	Failures   []string // empty when Solvable
	Trace      []string // ordered actions the simulated player performed
	Iterations int
}

// simState is the simulated player's knowledge/inventory/world deltas.
type simState struct {
	g *state.Game

	items     map[string]int // item name -> count (includes batteries)
	batteries int

	doorsPowered map[string]bool // room -> door circuit armed (sim view)
	roomOnline   map[string]bool // room -> power online (sim view)

	pickedUp                map[*world.Cell]map[string]bool
	furnitureOpened         map[*entities.Furniture]bool
	generatorOn             map[*entities.Generator]bool
	repairDone              map[string]bool
	hazardCleared           map[*entities.Hazard]bool
	unlockKeycardsSpawned   bool
	trace                   []string
}

func newSimState(g *state.Game) *simState {
	s := &simState{
		g:               g,
		items:           map[string]int{},
		doorsPowered:    map[string]bool{},
		roomOnline:      map[string]bool{},
		pickedUp:        map[*world.Cell]map[string]bool{},
		furnitureOpened: map[*entities.Furniture]bool{},
		generatorOn:     map[*entities.Generator]bool{},
		repairDone:      map[string]bool{},
		hazardCleared:   map[*entities.Hazard]bool{},
	}
	for room, on := range g.RoomDoorsPowered {
		if on {
			s.doorsPowered[room] = true
			s.roomOnline[room] = true
		}
	}
	if g.Grid != nil {
		g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
			if cell == nil {
				return
			}
			gen := gameworld.GetGameData(cell).Generator
			if gen != nil && gen.IsPowered() {
				s.generatorOn[gen] = true
			}
		})
	}
	return s
}

func (s *simState) anyGeneratorOn() bool {
	return len(s.generatorOn) > 0
}

func (s *simState) hasItem(name string) bool {
	return s.items[name] > 0
}

func (s *simState) addTrace(format string, args ...interface{}) {
	s.trace = append(s.trace, fmt.Sprintf(format, args...))
}

// simPassable mirrors gameplay.CanEnter under the simulated state: which cells could
// the player eventually step on given current sim inventory/power/repair progress.
func (s *simState) simPassable(cell *world.Cell) bool {
	if cell == nil || !cell.Room {
		return false
	}
	data := gameworld.GetGameData(cell)
	if d := data.Door; d != nil {
		if d.Locked && !s.hasItem(d.KeycardName()) {
			return false
		}
		if !d.Locked && !d.KeycardGated && !s.doorsPowered[d.RoomName] {
			return false
		}
		// Locked door + keycard: passable (keycard overrides power).
		// Unlocked keycard-gated door: stays passable without power.
	}
	if data.Generator != nil || data.Furniture != nil || data.Terminal != nil ||
		data.Puzzle != nil || data.MaintenanceTerm != nil || data.HazardControl != nil ||
		gameworld.RepairDeviceBlocksMovement(cell) {
		return false
	}
	if h := data.Hazard; h != nil && h.IsBlocking() && !s.hazardCleared[h] {
		return false
	}
	if r := data.RepairBlocker; r != nil && r.BlockerName != "" && !s.repairDone[r.ID] && !r.IsComplete() {
		return false
	}
	return true
}

// reachable returns the set of cells the simulated player can stand on.
func (s *simState) reachable() *mapset.Set[*world.Cell] {
	out := mapset.New[*world.Cell]()
	for _, entry := range entryDoorPowerSeeds(s.g) {
		s.bfsSimReachable(entry, &out)
	}
	return &out
}

func (s *simState) bfsSimReachable(entry *world.Cell, out *mapset.Set[*world.Cell]) {
	if s == nil || entry == nil || out == nil {
		return
	}
	queue := []*world.Cell{entry}
	out.Put(entry)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, n := range cur.GetNeighbors() {
			if n == nil || out.Has(n) || !s.simPassable(n) {
				continue
			}
			out.Put(n)
			queue = append(queue, n)
		}
	}
}

// adjacentReachable reports whether the player can stand next to (or on) cell.
func adjacentReachable(reach *mapset.Set[*world.Cell], cell *world.Cell) bool {
	if cell == nil || reach == nil {
		return false
	}
	if reach.Has(cell) {
		return true
	}
	for _, n := range cell.GetNeighbors() {
		if reach.Has(n) {
			return true
		}
	}
	return false
}

// simStep applies every action available from the current reachable set.
// Returns true when at least one action fired (progress was made).
func (s *simState) simStep(reach *mapset.Set[*world.Cell]) bool {
	progress := false
	grid := s.g.Grid

	// 1. Pick up floor items on standable cells, or adjacent to cells blocked by repair devices.
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || cell.ItemsOnFloor.Size() == 0 {
			return
		}
		canPick := reach.Has(cell)
		if !canPick && adjacentReachable(reach, cell) && gameworld.RepairDeviceBlocksMovement(cell) {
			canPick = true
		}
		if !canPick {
			return
		}
		cell.ItemsOnFloor.Each(func(item *world.Item) {
			if item == nil {
				return
			}
			if s.pickedUp[cell] == nil {
				s.pickedUp[cell] = map[string]bool{}
			}
			if s.pickedUp[cell][item.Name] {
				return
			}
			s.pickedUp[cell][item.Name] = true
			s.items[item.Name]++
			if item.Name == "Battery" {
				s.batteries++
			}
			s.addTrace("pick up %q at x:%d y:%d", item.Name, cell.Col, cell.Row)
			progress = true
		})
	})

	// 2. Open furniture next to a standable cell; take contained items.
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		furn := gameworld.GetGameData(cell).Furniture
		if furn == nil || s.furnitureOpened[furn] || !adjacentReachable(reach, cell) {
			return
		}
		s.furnitureOpened[furn] = true
		if furn.ContainedItem != nil {
			name := furn.ContainedItem.Name
			s.items[name]++
			if name == "Battery" {
				s.batteries++
			}
			s.addTrace("take %q from %q at x:%d y:%d", name, furn.Name, cell.Col, cell.Row)
			progress = true
		}
	})

	// 3. Use powered maintenance terminals to arm door power for their own and
	//    adjacent rooms (mirrors CanControlRoomPower).
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		term := gameworld.GetGameData(cell).MaintenanceTerm
		if term == nil || !adjacentReachable(reach, cell) {
			return
		}
		if !term.Powered && !s.roomOnline[term.RoomName] {
			return
		}
		targets := []string{term.RoomName}
		targets = append(targets, GetAdjacentRoomNames(grid, term.RoomName)...)
		for _, target := range targets {
			if target == "" || target == "Corridor" || s.doorsPowered[target] {
				continue
			}
			s.doorsPowered[target] = true
			if s.anyGeneratorOn() {
				s.roomOnline[target] = true
			}
			s.addTrace("arm door power for %q from terminal in %q", target, term.RoomName)
			progress = true
		}
	})

	// 4. Insert batteries into adjacent-reachable unpowered generators.
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen == nil || s.generatorOn[gen] || !adjacentReachable(reach, cell) {
			return
		}
		needed := gen.BatteriesNeeded()
		if s.batteries < needed {
			return
		}
		s.batteries -= needed
		s.generatorOn[gen] = true
		s.addTrace("start %q with %d batteries at x:%d y:%d", gen.Name, needed, cell.Col, cell.Row)
		progress = true
	})

	// 5. Complete repairs whose device is adjacent-reachable, prereqs done, and
	//    power requirement met.
	for _, rep := range sortedRepairs(s.g) {
		if rep == nil || s.repairDone[rep.ID] || rep.IsComplete() {
			if rep != nil && rep.IsComplete() {
				s.repairDone[rep.ID] = true
			}
			continue
		}
		if rep.DeviceRow < 0 || rep.DeviceCol < 0 {
			continue
		}
		cell := grid.GetCell(rep.DeviceRow, rep.DeviceCol)
		if cell == nil || !adjacentReachable(reach, cell) {
			continue
		}
		prereqsDone := true
		for _, id := range rep.PrereqIDs {
			if !s.repairDone[id] {
				prereqsDone = false
				break
			}
		}
		if !prereqsDone {
			continue
		}
		if rep.NeedsLivePower() && !s.repairRoomPowered(rep.RoomName) {
			continue
		}
		s.repairDone[rep.ID] = true
		s.addTrace("complete repair %q (%s) in %q", rep.Name, rep.ID, rep.RoomName)
		progress = true
	}

	if !s.unlockKeycardsSpawned && s.allExitGatingRepairsDone() {
		DropPendingUnlockKeycards(s.g)
		s.unlockKeycardsSpawned = true
		progress = true
	}

	// 6. Clear hazards: activate adjacent-reachable controls, or apply a carried
	//    fix item to an adjacent-reachable hazard.
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		data := gameworld.GetGameData(cell)
		if ctrl := data.HazardControl; ctrl != nil && ctrl.Hazard != nil &&
			ctrl.Hazard.IsBlocking() && !s.hazardCleared[ctrl.Hazard] &&
			adjacentReachable(reach, cell) {
			s.hazardCleared[ctrl.Hazard] = true
			s.addTrace("activate hazard control at x:%d y:%d", cell.Col, cell.Row)
			progress = true
		}
		if h := data.Hazard; h != nil && h.IsBlocking() && !s.hazardCleared[h] &&
			h.RequiresItem() && s.hasItem(h.RequiredItemName()) &&
			adjacentReachable(reach, cell) {
			s.items[h.RequiredItemName()]--
			s.hazardCleared[h] = true
			s.addTrace("fix hazard with %q at x:%d y:%d", h.RequiredItemName(), cell.Col, cell.Row)
			progress = true
		}
	})

	return progress
}

// repairRoomPowered approximates RoomConsideredPowered under sim state.
func (s *simState) repairRoomPowered(roomName string) bool {
	if roomName == "" || roomName == "Corridor" || generator.IsPlacementExcludedRoom(roomName) {
		return s.anyGeneratorOn()
	}
	return s.roomOnline[roomName]
}

func sortedRepairs(g *state.Game) []*entities.RepairObjective {
	out := make([]*entities.RepairObjective, 0, len(g.RepairObjectives))
	for _, rep := range g.RepairObjectives {
		if rep != nil {
			out = append(out, rep)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// SimulatePlaythrough verifies the generated deck is completable by a simulated
// player starting at the lift entry with an empty inventory.
func SimulatePlaythrough(g *state.Game) SimReport {
	report := SimReport{}
	if g == nil || g.Grid == nil {
		report.Failures = append(report.Failures, "no grid")
		return report
	}
	entry := PlayerEntryCell(g)
	if entry == nil {
		report.Failures = append(report.Failures, "no player entry cell")
		return report
	}

	s := newSimState(g)
	const maxIterations = 256
	reach := s.reachable()
	for i := 0; i < maxIterations; i++ {
		report.Iterations = i + 1
		if !s.simStep(reach) {
			break
		}
		reach = s.reachable()
	}
	report.Trace = s.trace

	// --- Acceptance criteria ---
	if exit := g.Grid.ExitCell(); exit != nil && !reach.Has(exit) {
		report.Failures = append(report.Failures,
			fmt.Sprintf("exit lift at x:%d y:%d never reachable", exit.Col, exit.Row))
	}

	for _, rep := range sortedRepairs(g) {
		if !s.repairDone[rep.ID] && !rep.IsComplete() {
			report.Failures = append(report.Failures,
				fmt.Sprintf("repair %q (%s) in %q never completable", rep.Name, rep.ID, rep.RoomName))
		}
	}

	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil {
			return
		}
		if h := gameworld.GetGameData(cell).Hazard; h != nil && h.IsBlocking() && !s.hazardCleared[h] {
			report.Failures = append(report.Failures,
				fmt.Sprintf("hazard %q at x:%d y:%d never clearable", h.Name, cell.Col, cell.Row))
		}
	})

	for _, room := range simNamedRooms(g) {
		if !simRoomEntered(g, reach, room) {
			report.Failures = append(report.Failures,
				fmt.Sprintf("room %q never enterable", room))
		}
	}

	report.Failures = append(report.Failures, simUnlockPayoffFailures(g, s)...)
	report.Failures = append(report.Failures, simUnlockKeycardObtainabilityFailures(g, s)...)

	report.Solvable = len(report.Failures) == 0
	return report
}

func (s *simState) allExitGatingRepairsDone() bool {
	if s == nil || s.g == nil {
		return false
	}
	for _, repair := range s.g.RepairObjectives {
		if repair == nil || repair.SkipExitGate {
			continue
		}
		if !s.repairDone[repair.ID] && !repair.IsComplete() {
			return false
		}
	}
	return true
}

func simNamedRooms(g *state.Game) []string {
	seen := map[string]bool{}
	var out []string
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name == "" || cell.Name == "Corridor" || seen[cell.Name] {
			return
		}
		seen[cell.Name] = true
		out = append(out, cell.Name)
	})
	sort.Strings(out)
	return out
}

func simRoomEntered(g *state.Game, reach *mapset.Set[*world.Cell], room string) bool {
	entered := false
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if entered || cell == nil || cell.Name != room {
			return
		}
		// A room counts as entered when the player can stand on, or interact
		// adjacent to, at least one of its cells.
		if adjacentReachable(reach, cell) {
			entered = true
		}
	})
	return entered
}

// simUnlockPayoffFailures verifies this deck actually carries the unlock payoffs the
// run plan sources here: a missing routing coupler or keycard payoff would
// permanently lock a target deck for the whole run.
func simUnlockPayoffFailures(g *state.Game, s *simState) []string {
	if g.UnlockPlan == nil {
		return nil
	}
	var out []string
	for _, req := range g.UnlockPlan.ForSource(g.CurrentDeckID) {
		switch req.Kind {
		case unlocks.KindRoutingRepair:
			if g.RepairByID(req.RepairID) == nil {
				out = append(out, fmt.Sprintf(
					"unlock payoff missing: routing repair %q for deck %d not placed",
					req.RepairID, req.TargetDeckID+1))
			}
		case unlocks.KindSecurityKeycard:
			if simKeycardPayoffPresent(g, s, req.KeycardName) {
				continue
			}
			out = append(out, fmt.Sprintf(
				"unlock payoff missing: keycard %q for deck %d not registered on any repair",
				req.KeycardName, req.TargetDeckID+1))
		}
	}
	return out
}

func simKeycardPayoffPresent(g *state.Game, s *simState, name string) bool {
	if name == "" {
		return true
	}
	if g.HasRunKeycard(name) {
		return true
	}
	if s != nil && s.hasItem(name) {
		return true
	}
	if PendingUnlockKeycardRegistered(g, name) {
		return true
	}
	if UnlockKeycardOnFloor(g, name) {
		return true
	}
	return false
}

func simPendingKeycardRegistered(g *state.Game, name string) bool {
	return PendingUnlockKeycardRegistered(g, name)
}

func simUnlockKeycardObtainabilityFailures(g *state.Game, s *simState) []string {
	if g == nil || g.UnlockPlan == nil || s == nil || !s.allExitGatingRepairsDone() {
		return nil
	}
	var out []string
	for _, req := range g.UnlockPlan.ForSource(g.CurrentDeckID) {
		if req.Kind != unlocks.KindSecurityKeycard || req.KeycardName == "" {
			continue
		}
		if g.HasRunKeycard(req.KeycardName) || s.hasItem(req.KeycardName) {
			continue
		}
		if PendingUnlockKeycardRegistered(g, req.KeycardName) {
			out = append(out, fmt.Sprintf(
				"unlock keycard %q for deck %d registered but never obtainable after local repairs",
				req.KeycardName, req.TargetDeckID+1))
			continue
		}
		if UnlockKeycardOnFloor(g, req.KeycardName) {
			out = append(out, fmt.Sprintf(
				"unlock keycard %q for deck %d on floor but not pickup reachable",
				req.KeycardName, req.TargetDeckID+1))
		}
	}
	return out
}
