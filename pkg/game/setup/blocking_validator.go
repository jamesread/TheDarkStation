package setup

import (
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
)

// BlockingPlacementValidator amortizes the expensive invariants behind
// CanPlaceBlockingEntity when many candidate blockers are tested against the
// same level state.
type BlockingPlacementValidator struct {
	g                 *state.Game
	exitReachable     bool
	cutsExitPath      map[*world.Cell]bool
	baseInitReachable *mapset.Set[*world.Cell]
	baseInitRooms     map[string]bool
	keycards          []keycardLocation
}

func NewBlockingPlacementValidator(g *state.Game) *BlockingPlacementValidator {
	v := &BlockingPlacementValidator{g: g, exitReachable: true}
	if g == nil || g.Grid == nil {
		return v
	}
	v.exitReachable, v.cutsExitPath = completionExitCutCells(g)
	v.baseInitReachable = InitialReachableCells(g)
	v.baseInitRooms = reachableNamedRooms(v.baseInitReachable)
	v.keycards = keycardLocations(g)
	return v
}

func (v *BlockingPlacementValidator) CanPlace(candidate *world.Cell) bool {
	if v == nil || v.g == nil || candidate == nil {
		return false
	}
	if candidate.ItemsOnFloor.Size() > 0 {
		return false
	}
	if !v.exitReachable || v.cutsExitPath[candidate] {
		return false
	}
	if !v.blockingPlacementPreservesNavAccess(candidate) {
		return false
	}
	return v.initProgressPreserved(candidate)
}

func (v *BlockingPlacementValidator) blockingPlacementPreservesNavAccess(candidate *world.Cell) bool {
	extra := mapset.New[*world.Cell]()
	extra.Put(candidate)
	for _, n := range candidate.GetNeighbors() {
		if RequiresAdjacentNavSpace(n) && !EntityHasAdjacentNavSpace(v.g, n, &extra) {
			return false
		}
	}
	return CandidateBlockingCellHasAdjacentNavSpace(v.g, candidate, nil)
}

func (v *BlockingPlacementValidator) initProgressPreserved(candidate *world.Cell) bool {
	if candidate == nil {
		return true
	}
	with := InitialReachableCellsWithExtraBlock(v.g, candidate)
	if !keycardsStillAccessibleFromLocations(v.keycards, v.baseInitReachable, with) {
		return false
	}
	withRooms := reachableNamedRooms(with)
	for room := range v.baseInitRooms {
		if !withRooms[room] {
			return false
		}
	}
	return true
}

func keycardsStillAccessibleFromLocations(locations []keycardLocation, base, with *mapset.Set[*world.Cell]) bool {
	for _, loc := range locations {
		if loc.inFurniture {
			if !cellAccessibleFromReachable(base, loc.cell) {
				continue
			}
			if !cellAccessibleFromReachable(with, loc.cell) {
				return false
			}
			continue
		}
		if loc.cell == nil || base == nil || !base.Has(loc.cell) {
			continue
		}
		if with == nil || !with.Has(loc.cell) {
			return false
		}
	}
	return true
}

func completionExitCutCells(g *state.Game) (bool, map[*world.Cell]bool) {
	out := make(map[*world.Cell]bool)
	if g == nil || g.Grid == nil {
		return false, out
	}
	entry := PlayerEntryCell(g)
	exit := g.Grid.ExitCell()
	if entry == nil || exit == nil {
		return true, out
	}
	if !isPassableAtLevelCompletion(entry, nil) || !isPassableAtLevelCompletion(exit, nil) {
		return false, out
	}

	discovery := make(map[*world.Cell]int)
	low := make(map[*world.Cell]int)
	parent := make(map[*world.Cell]*world.Cell)
	subtreeHasExit := make(map[*world.Cell]bool)
	time := 0

	var visit func(*world.Cell)
	visit = func(cell *world.Cell) {
		time++
		discovery[cell] = time
		low[cell] = time
		subtreeHasExit[cell] = cell == exit

		for _, n := range cell.GetNeighbors() {
			if n == nil || !isPassableAtLevelCompletion(n, nil) {
				continue
			}
			if discovery[n] == 0 {
				parent[n] = cell
				visit(n)
				if subtreeHasExit[n] {
					subtreeHasExit[cell] = true
				}
				if low[n] < low[cell] {
					low[cell] = low[n]
				}
				if subtreeHasExit[n] && low[n] >= discovery[cell] {
					out[cell] = true
				}
				continue
			}
			if parent[cell] != n && discovery[n] < low[cell] {
				low[cell] = discovery[n]
			}
		}
	}

	visit(entry)
	if !subtreeHasExit[entry] {
		return false, out
	}
	out[entry] = true
	out[exit] = true
	return true, out
}
