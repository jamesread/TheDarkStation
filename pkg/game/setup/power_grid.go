package setup

import (
	"fmt"
	"sort"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// GeneratorOutputWatts returns supply watts from a powered generator (fixed 100 W for now).
func GeneratorOutputWatts(g *state.Game, gen *entities.Generator) int {
	if g == nil || gen == nil || !gen.IsPowered() {
		return 0
	}
	return GeneratorOutputWattsNominal
}

func roomDoorsPoweredEffective(g *state.Game, override map[string]bool) map[string]bool {
	doors := make(map[string]bool)
	if g != nil && g.RoomDoorsPowered != nil {
		for k, v := range g.RoomDoorsPowered {
			doors[k] = v
		}
	}
	for k, v := range override {
		doors[k] = v
	}
	return doors
}

func roomDoorsArmed(g *state.Game, doorsPowered map[string]bool, roomName string) bool {
	if roomName == "" {
		return false
	}
	if doorsPowered != nil {
		return doorsPowered[roomName]
	}
	return g != nil && g.RoomDoorsPowered != nil && g.RoomDoorsPowered[roomName]
}

// CanTraverseCellForPowerGridArm reports whether armed power grid may pass through a cell (player-enabled circuits).
func CanTraverseCellForPowerGridArm(g *state.Game, cell *world.Cell) bool {
	return CanTraverseCellForPowerGridArmDoors(g, cell, nil)
}

// CanTraverseCellForPowerGridArmDoors is like CanTraverseCellForPowerGridArm but uses doorsPowered when non-nil.
func CanTraverseCellForPowerGridArmDoors(g *state.Game, cell *world.Cell, doorsPowered map[string]bool) bool {
	if g == nil || cell == nil || !cell.Room {
		return false
	}
	if gameworld.HasFurniture(cell) {
		return false
	}
	if gameworld.RelayBlocksGrid(cell) {
		return false
	}
	if gameworld.HasDoor(cell) {
		if gameworld.HasLockedDoor(cell) {
			return true
		}
		roomName := gameworld.GetGameData(cell).Door.RoomName
		if roomName == "" || !roomDoorsArmed(g, doorsPowered, roomName) {
			return false
		}
		return true
	}
	if cell.Name != "" && cell.Name != "Corridor" && !roomDoorsArmed(g, doorsPowered, cell.Name) {
		return false
	}
	return true
}

// CanTraverseCellForLocalGeneratorFeed reports whether power may spread locally from a
// powered generator without armed door circuits (same physical pocket only; no door cells).
func CanTraverseCellForLocalGeneratorFeed(g *state.Game, cell *world.Cell) bool {
	if g == nil || cell == nil || !cell.Room {
		return false
	}
	if gameworld.HasFurniture(cell) {
		return false
	}
	if gameworld.RelayBlocksGrid(cell) {
		return false
	}
	if gameworld.HasDoor(cell) {
		return false
	}
	return true
}

// CanTraverseCellForPowerGrid reports whether armed power grid may pass through a cell (routing UI / scheduling).
func CanTraverseCellForPowerGrid(g *state.Game, cell *world.Cell) bool {
	return CanTraverseCellForPowerGridArm(g, cell)
}

// roomArmedOrManualEgress reports whether a room's door circuit is armed or manually released for overlay traversal.
func roomArmedOrManualEgress(g *state.Game, roomName string) bool {
	if g == nil || roomName == "" {
		return false
	}
	if g.RoomDoorsPowered != nil && g.RoomDoorsPowered[roomName] {
		return true
	}
	return RoomManualEgressReleased(g, roomName)
}

// CanTraverseCellForPowerGridOverlay reports overlay grid traversal (armed circuits and manual egress releases).
// Does not affect live power propagation scheduling.
func CanTraverseCellForPowerGridOverlay(g *state.Game, cell *world.Cell) bool {
	if g == nil || cell == nil || !cell.Room {
		return false
	}
	if gameworld.HasFurniture(cell) {
		return false
	}
	if gameworld.RelayBlocksGrid(cell) {
		return false
	}
	if gameworld.HasDoor(cell) {
		if gameworld.HasLockedDoor(cell) {
			return true
		}
		roomName := gameworld.GetGameData(cell).Door.RoomName
		if !roomArmedOrManualEgress(g, roomName) {
			return false
		}
	}
	if cell.Name != "" && cell.Name != "Corridor" && !roomArmedOrManualEgress(g, cell.Name) {
		return false
	}
	return true
}

// CanTraverseCellForLivePowerGrid reports whether propagated power currently passes through a cell.
func CanTraverseCellForLivePowerGrid(g *state.Game, cell *world.Cell) bool {
	return CanTraverseCellForPowerGridArm(g, cell) || CanTraverseCellForLocalGeneratorFeed(g, cell)
}

// cellsReachableFromGeneratorSeed returns cells fed by a single powered generator: local pocket
// conduction first, then expansion through armed routing circuits.
func cellsReachableFromGeneratorSeed(g *state.Game, seed *world.Cell) *mapset.Set[*world.Cell] {
	empty := mapset.New[*world.Cell]()
	if g == nil || g.Grid == nil || seed == nil || !isConductivePowerSeed(g, seed) {
		return &empty
	}
	local := localCellsFromPoweredGeneratorSeed(g, seed)
	visited := mapset.New[*world.Cell]()
	var queue []*world.Cell
	local.Each(func(c *world.Cell) {
		if c == nil {
			return
		}
		visited.Put(c)
		queue = append(queue, c)
	})
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, n := range cur.GetNeighbors() {
			if n == nil || !n.Room || visited.Has(n) {
				continue
			}
			if !CanTraverseCellForPowerGridArm(g, n) {
				continue
			}
			visited.Put(n)
			queue = append(queue, n)
		}
	}
	return &visited
}

// cellsReachableViaArmedRoutingFromSeed returns armed-routing reachability from one generator seed.
func cellsReachableViaArmedRoutingFromSeed(g *state.Game, seed *world.Cell) *mapset.Set[*world.Cell] {
	return cellsReachableViaArmedRoutingFromSeedDoors(g, seed, nil)
}

func cellsReachableViaArmedRoutingFromSeedDoors(g *state.Game, seed *world.Cell, doorsPowered map[string]bool) *mapset.Set[*world.Cell] {
	empty := mapset.New[*world.Cell]()
	if g == nil || seed == nil || !seed.Room {
		return &empty
	}
	globalSeen := mapset.New[*world.Cell]()
	return floodArmedRoutingComponentDoors(g, seed, doorsPowered, &globalSeen)
}

// armedGridComponentsFromSeeds returns disjoint armed-routing components that contain at least
// one seed. Uses one flood fill per component instead of one full-grid BFS per seed.
func armedGridComponentsFromSeeds(g *state.Game, seeds []*world.Cell, doorsPowered map[string]bool) []*mapset.Set[*world.Cell] {
	if g == nil || g.Grid == nil || len(seeds) == 0 {
		return nil
	}
	globalSeen := mapset.New[*world.Cell]()
	components := make([]*mapset.Set[*world.Cell], 0, len(seeds))
	for _, seed := range seeds {
		if seed == nil || globalSeen.Has(seed) {
			continue
		}
		component := floodArmedRoutingComponentDoors(g, seed, doorsPowered, &globalSeen)
		if component.Size() > 0 {
			components = append(components, component)
		}
	}
	return components
}

func floodArmedRoutingComponentDoors(g *state.Game, seed *world.Cell, doorsPowered map[string]bool, globalSeen *mapset.Set[*world.Cell]) *mapset.Set[*world.Cell] {
	empty := mapset.New[*world.Cell]()
	if g == nil || seed == nil || !seed.Room || globalSeen == nil {
		return &empty
	}
	component := mapset.New[*world.Cell]()
	queue := []*world.Cell{seed}
	component.Put(seed)
	globalSeen.Put(seed)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, n := range cur.GetNeighbors() {
			if n == nil || !n.Room || globalSeen.Has(n) {
				continue
			}
			if !CanTraverseCellForPowerGridArmDoors(g, n, doorsPowered) {
				continue
			}
			globalSeen.Put(n)
			component.Put(n)
			queue = append(queue, n)
		}
	}
	return &component
}

func generatorCellsOnGrid(g *state.Game) []*world.Cell {
	if g == nil || g.Grid == nil {
		return nil
	}
	var seeds []*world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		if gameworld.GetGameData(cell).Generator != nil {
			seeds = append(seeds, cell)
		}
	})
	return seeds
}

// armedGridComponentsFromGenerators returns disjoint armed routing grids fed by powered generators.
func armedGridComponentsFromGenerators(g *state.Game) []*mapset.Set[*world.Cell] {
	if g == nil || g.Grid == nil {
		return nil
	}
	var seeds []*world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || !isConductivePowerSeed(g, cell) {
			return
		}
		seeds = append(seeds, cell)
	})
	if len(seeds) == 0 {
		return nil
	}
	return armedGridComponentsFromSeeds(g, seeds, nil)
}

func armedGridComponentsForBalance(g *state.Game, doorsPowered map[string]bool) []*mapset.Set[*world.Cell] {
	if doorsPowered != nil {
		return armedGridComponentsFromGeneratorLocations(g, doorsPowered)
	}
	if cached, ok := g.CachedArmedBalanceComponents(); ok {
		return cached
	}
	components := armedGridComponentsFromGeneratorLocations(g, nil)
	g.StoreArmedBalanceComponentsCache(components)
	return components
}

func armedGridComponentsFromGeneratorLocations(g *state.Game, doorsPowered map[string]bool) []*mapset.Set[*world.Cell] {
	if g == nil || g.Grid == nil {
		return nil
	}
	seeds := generatorCellsOnGrid(g)
	if len(seeds) == 0 {
		return nil
	}
	return armedGridComponentsFromSeeds(g, seeds, doorsPowered)
}

// ArmedGridForRoom returns the armed routing grid containing roomName, if any.
func ArmedGridForRoom(g *state.Game, roomName string) *mapset.Set[*world.Cell] {
	return ArmedGridForRoomDoors(g, roomName, nil)
}

// ArmedGridForRoomDoors returns the armed routing grid for roomName using doorsPowered when set.
func ArmedGridForRoomDoors(g *state.Game, roomName string, doorsPowered map[string]bool) *mapset.Set[*world.Cell] {
	empty := mapset.New[*world.Cell]()
	if g == nil || roomName == "" {
		return &empty
	}
	for _, grid := range armedGridComponentsForBalance(g, doorsPowered) {
		if roomHasCellOnGrid(g, roomName, grid) {
			return grid
		}
	}
	return &empty
}

// ArmedGridForCell returns the armed routing grid containing cell, if any.
func ArmedGridForCell(g *state.Game, cell *world.Cell) *mapset.Set[*world.Cell] {
	empty := mapset.New[*world.Cell]()
	if g == nil || cell == nil {
		return &empty
	}
	for _, grid := range armedGridComponentsFromGenerators(g) {
		if grid.Has(cell) {
			return grid
		}
	}
	return &empty
}

// ArmedGridSupply returns total watts from powered generators on the armed grid.
func ArmedGridSupply(g *state.Game, grid *mapset.Set[*world.Cell]) int {
	if g == nil || grid == nil || grid.Size() == 0 {
		return 0
	}
	total := 0
	grid.Each(func(cell *world.Cell) {
		if cell == nil {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen != nil && gen.IsPowered() {
			total += GeneratorOutputWatts(g, gen)
		}
	})
	return total
}

// cellsReachableViaArmedRoutingFromGenerators returns cells fed through armed routing circuits
// from powered generators (excludes local same-room feed when the room circuit is off).
func cellsReachableViaArmedRoutingFromGenerators(g *state.Game) *mapset.Set[*world.Cell] {
	empty := mapset.New[*world.Cell]()
	if g == nil || g.Grid == nil {
		return &empty
	}
	union := mapset.New[*world.Cell]()
	for _, grid := range armedGridComponentsFromGenerators(g) {
		grid.Each(func(c *world.Cell) {
			if c != nil {
				union.Put(c)
			}
		})
	}
	return &union
}

// CellsReachableFromPoweredGenerators returns every cell with live power from any online generator.
func CellsReachableFromPoweredGenerators(g *state.Game) *mapset.Set[*world.Cell] {
	empty := mapset.New[*world.Cell]()
	if g == nil || g.Grid == nil {
		return &empty
	}
	if cached, ok := g.CachedLivePowerCells(); ok {
		return cached
	}
	union := computeCellsReachableFromPoweredGenerators(g)
	g.StoreLivePowerCellsCache(&union)
	return &union
}

func localCellsFromPoweredGeneratorSeed(g *state.Game, seed *world.Cell) *mapset.Set[*world.Cell] {
	empty := mapset.New[*world.Cell]()
	if g == nil || seed == nil || !seed.Room || !isConductivePowerSeed(g, seed) {
		return &empty
	}
	genRoom := seed.Name
	visited := mapset.New[*world.Cell]()
	queue := []*world.Cell{seed}
	visited.Put(seed)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, n := range cur.GetNeighbors() {
			if n == nil || !n.Room || visited.Has(n) {
				continue
			}
			if !CanTraverseCellForLocalGeneratorFeed(g, n) {
				continue
			}
			if genRoom != "" && genRoom != "Corridor" && n.Name != genRoom {
				continue
			}
			visited.Put(n)
			queue = append(queue, n)
		}
	}
	return &visited
}

func computeCellsReachableFromPoweredGenerators(g *state.Game) mapset.Set[*world.Cell] {
	union := mapset.New[*world.Cell]()
	if g == nil || g.Grid == nil {
		return union
	}
	var seeds []*world.Cell
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || !isConductivePowerSeed(g, cell) {
			return
		}
		seeds = append(seeds, cell)
	})
	if len(seeds) == 0 {
		return union
	}

	roomExpanded := make(map[string]bool)
	var armedFrontier []*world.Cell
	for _, seed := range seeds {
		genRoom := seed.Name
		if genRoom != "" && genRoom != "Corridor" {
			if roomExpanded[genRoom] {
				union.Put(seed)
				continue
			}
			roomExpanded[genRoom] = true
			localCellsFromPoweredGeneratorSeed(g, seed).Each(func(c *world.Cell) {
				if c == nil {
					return
				}
				union.Put(c)
				armedFrontier = append(armedFrontier, c)
			})
			continue
		}
		union.Put(seed)
		armedFrontier = append(armedFrontier, seed)
	}

	visited := mapset.New[*world.Cell]()
	for _, c := range armedFrontier {
		if c != nil {
			visited.Put(c)
		}
	}
	queue := append([]*world.Cell(nil), armedFrontier...)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, n := range cur.GetNeighbors() {
			if n == nil || !n.Room || visited.Has(n) {
				continue
			}
			if !CanTraverseCellForPowerGridArm(g, n) {
				continue
			}
			visited.Put(n)
			union.Put(n)
			queue = append(queue, n)
		}
	}
	return union
}

// CellHasLivePower reports whether propagated power has reached a specific grid cell.
func CellHasLivePower(g *state.Game, cell *world.Cell) bool {
	if g == nil || cell == nil || !cell.Room {
		return false
	}
	return CellsReachableFromPoweredGenerators(g).Has(cell)
}

// RoomsReachableInPowerGridExcluding is like RoomsReachableInPowerGrid but does not traverse cells in excludeRoom.
func RoomsReachableInPowerGridExcluding(g *state.Game, startCell *world.Cell, excludeRoom string) []string {
	if excludeRoom == "" {
		return RoomsReachableInPowerGrid(g, startCell)
	}
	if g == nil || g.Grid == nil || startCell == nil {
		return nil
	}
	if !CanTraverseCellForPowerGrid(g, startCell) {
		return nil
	}

	visited := mapset.New[*world.Cell]()
	rooms := make(map[string]bool)
	queue := []*world.Cell{startCell}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == nil || visited.Has(cur) {
			continue
		}
		if cur.Name == excludeRoom {
			continue
		}
		if !CanTraverseCellForPowerGrid(g, cur) {
			continue
		}
		visited.Put(cur)
		if cur.Name != "" && cur.Name != "Corridor" {
			rooms[cur.Name] = true
		}
		for _, n := range cur.GetNeighbors() {
			if n == nil || !n.Room || visited.Has(n) || n.Name == excludeRoom {
				continue
			}
			if CanTraverseCellForPowerGrid(g, n) {
				queue = append(queue, n)
			}
		}
	}

	names := make([]string, 0, len(rooms))
	for name := range rooms {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// CellsReachableInPowerGrid returns cells reachable from startCell via powered doors and closed relays.
func CellsReachableInPowerGrid(g *state.Game, startCell *world.Cell) *mapset.Set[*world.Cell] {
	empty := mapset.New[*world.Cell]()
	if g == nil || g.Grid == nil || startCell == nil {
		return &empty
	}
	if !CanTraverseCellForPowerGrid(g, startCell) {
		return &empty
	}

	visited := mapset.New[*world.Cell]()
	queue := []*world.Cell{startCell}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == nil || visited.Has(cur) {
			continue
		}
		if !CanTraverseCellForPowerGrid(g, cur) {
			continue
		}
		visited.Put(cur)
		for _, n := range cur.GetNeighbors() {
			if n == nil || !n.Room || visited.Has(n) {
				continue
			}
			if CanTraverseCellForPowerGrid(g, n) {
				queue = append(queue, n)
			}
		}
	}
	return &visited
}

// CellsReachableInPowerGridOverlay returns overlay cells reachable from startCell, including manual egress doors.
func CellsReachableInPowerGridOverlay(g *state.Game, startCell *world.Cell) *mapset.Set[*world.Cell] {
	empty := mapset.New[*world.Cell]()
	if g == nil || g.Grid == nil || startCell == nil {
		return &empty
	}
	if !CanTraverseCellForPowerGridOverlay(g, startCell) {
		return &empty
	}

	visited := mapset.New[*world.Cell]()
	queue := []*world.Cell{startCell}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == nil || visited.Has(cur) {
			continue
		}
		if !CanTraverseCellForPowerGridOverlay(g, cur) {
			continue
		}
		visited.Put(cur)
		for _, n := range cur.GetNeighbors() {
			if n == nil || !n.Room || visited.Has(n) {
				continue
			}
			if CanTraverseCellForPowerGridOverlay(g, n) {
				queue = append(queue, n)
			}
		}
	}
	return &visited
}

// PowerGridComponentCount returns the number of disjoint conducting power grids on the grid
// (split when open relays or unpowered doors break propagation).
func PowerGridComponentCount(g *state.Game) int {
	if g == nil || g.Grid == nil {
		return 0
	}
	globalSeen := mapset.New[*world.Cell]()
	count := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || globalSeen.Has(cell) {
			return
		}
		if !CanTraverseCellForPowerGrid(g, cell) {
			return
		}
		floodFillPowerGridComponent(g, cell, &globalSeen)
		count++
	})
	return count
}

func floodFillPowerGridComponent(g *state.Game, start *world.Cell, globalSeen *mapset.Set[*world.Cell]) {
	queue := []*world.Cell{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == nil || globalSeen.Has(cur) {
			continue
		}
		if !CanTraverseCellForPowerGrid(g, cur) {
			continue
		}
		globalSeen.Put(cur)
		for _, n := range cur.GetNeighbors() {
			if n != nil && n.Room && !globalSeen.Has(n) && CanTraverseCellForPowerGrid(g, n) {
				queue = append(queue, n)
			}
		}
	}
}

// SumGeneratorOutputInGrid returns total watts from powered generators on cells in power grid.
func SumGeneratorOutputInGrid(g *state.Game, grid *mapset.Set[*world.Cell]) int {
	if g == nil || grid == nil || grid.Size() == 0 {
		return 0
	}
	total := 0
	grid.Each(func(c *world.Cell) {
		gen := gameworld.GetGameData(c).Generator
		if gen != nil && gen.IsPowered() {
			total += GeneratorOutputWatts(g, gen)
		}
	})
	return total
}

// GeneratorGridSupplyAtCell returns this generator's output, total supply on its power grid,
// and how many disjoint power grids exist on the deck (for split-grid tooltips).
func GeneratorGridSupplyAtCell(g *state.Game, cell *world.Cell) (individual, gridTotal, gridCount int) {
	if g == nil || cell == nil {
		return 0, 0, 0
	}
	gridCount = PowerGridComponentCount(g)
	gen := gameworld.GetGameData(cell).Generator
	if gen != nil {
		individual = GeneratorOutputWatts(g, gen)
	}
	grid := CellsReachableInPowerGrid(g, cell)
	if grid.Size() == 0 {
		return individual, individual, gridCount
	}
	return individual, SumGeneratorOutputInGrid(g, grid), gridCount
}

// CellsReachableInLivePowerGrid returns cells with propagated power reachable from startCell.
func CellsReachableInLivePowerGrid(g *state.Game, startCell *world.Cell) *mapset.Set[*world.Cell] {
	empty := mapset.New[*world.Cell]()
	if g == nil || g.Grid == nil || startCell == nil {
		return &empty
	}
	live := CellsReachableFromPoweredGenerators(g)
	if !live.Has(startCell) {
		return &empty
	}
	if isConductivePowerSeed(g, startCell) {
		return cellsReachableFromGeneratorSeed(g, startCell)
	}
	visited := mapset.New[*world.Cell]()
	queue := []*world.Cell{startCell}
	visited.Put(startCell)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, n := range cur.GetNeighbors() {
			if n == nil || !n.Room || visited.Has(n) || !live.Has(n) {
				continue
			}
			visited.Put(n)
			queue = append(queue, n)
		}
	}
	return &visited
}

// RoomsFedByPoweredGeneratorGrid returns room names with propagated power from powered generators.
func RoomsFedByPoweredGeneratorGrid(g *state.Game) map[string]bool {
	rooms := make(map[string]bool)
	if g == nil || g.Grid == nil {
		return rooms
	}
	cellsReachableViaArmedRoutingFromGenerators(g).Each(func(c *world.Cell) {
		if c == nil || c.Name == "" || c.Name == "Corridor" {
			return
		}
		if CanTraverseCellForPowerGridArm(g, c) {
			rooms[c.Name] = true
		}
	})
	return rooms
}

// RoomsOnConductiveGeneratorGrid returns room names whose maintenance terminals may receive
// conductive feed from a powered generator on the live power grid.
func RoomsOnConductiveGeneratorGrid(g *state.Game) map[string]bool {
	rooms := make(map[string]bool)
	if g == nil || g.Grid == nil {
		return rooms
	}
	CellsReachableFromPoweredGenerators(g).Each(func(c *world.Cell) {
		if c == nil || c.Name == "" || c.Name == "Corridor" {
			return
		}
		rooms[c.Name] = true
	})
	return rooms
}

// RoomHasLivePower reports whether any cell in roomName has propagated generator power.
func RoomHasLivePower(g *state.Game, roomName string) bool {
	if g == nil || g.Grid == nil || roomName == "" {
		return false
	}
	live := CellsReachableFromPoweredGenerators(g)
	found := false
	live.Each(func(cell *world.Cell) {
		if !found && cell != nil && cell.Name == roomName {
			found = true
		}
	})
	return found
}

// RoomPoweredOnPowerGrid reports whether a room is online on the power grid.
func RoomPoweredOnPowerGrid(g *state.Game, roomName string, fedRooms map[string]bool) bool {
	if g == nil || roomName == "" || roomName == "Corridor" {
		return false
	}
	_ = fedRooms
	return RoomConsideredPowered(g, roomName)
}

// RoomsReachableInPowerGrid returns sorted room names reachable from startCell via powered doors and closed relays.
func RoomsReachableInPowerGrid(g *state.Game, startCell *world.Cell) []string {
	visited := CellsReachableInPowerGrid(g, startCell)
	if visited.Size() == 0 {
		return nil
	}
	rooms := make(map[string]bool)
	visited.Each(func(c *world.Cell) {
		if c.Name != "" && c.Name != "Corridor" {
			rooms[c.Name] = true
		}
	})
	names := make([]string, 0, len(rooms))
	for name := range rooms {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RoomsReachableFromPoweredTerminals unions power grid reachability from every powered maintenance terminal.
func RoomsReachableFromPoweredTerminals(g *state.Game) []string {
	if g == nil || g.Grid == nil {
		return nil
	}
	roomSet := make(map[string]bool)
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.MaintenanceTerm == nil || !data.MaintenanceTerm.Powered {
			return
		}
		for _, name := range RoomsReachableInPowerGrid(g, cell) {
			roomSet[name] = true
		}
	})
	names := make([]string, 0, len(roomSet))
	for name := range roomSet {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SelectableRoomsForTerminal returns rooms the player may target from a maintenance terminal:
// union of power grid-reachable rooms and spec-adjacent rooms (§2.2: own room + directly adjacent).
func SelectableRoomsForTerminal(g *state.Game, grid *world.Grid, terminalRoom string) []string {
	if g == nil || grid == nil || terminalRoom == "" {
		return GetAdjacentRoomNames(grid, terminalRoom)
	}
	roomSet := make(map[string]bool)
	hasPoweredTerminal := false
	grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name != terminalRoom {
			return
		}
		data := gameworld.GetGameData(cell)
		if data.MaintenanceTerm == nil || !data.MaintenanceTerm.Powered {
			return
		}
		hasPoweredTerminal = true
		for _, name := range RoomsReachableInPowerGrid(g, cell) {
			roomSet[name] = true
		}
	})
	for _, name := range GetAdjacentRoomNames(grid, terminalRoom) {
		roomSet[name] = true
	}
	if !hasPoweredTerminal && len(roomSet) == 0 {
		return GetAdjacentRoomNames(grid, terminalRoom)
	}
	names := make([]string, 0, len(roomSet))
	for name := range roomSet {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// isConductivePowerSeed reports whether a cell feeds the power grid (powered generator only).
func isConductivePowerSeed(g *state.Game, cell *world.Cell) bool {
	if g == nil || cell == nil {
		return false
	}
	data := gameworld.GetGameData(cell)
	return data.Generator != nil && data.Generator.IsPowered()
}

// ConductiveGridFromSeed returns live power grid cells fed by a powered generator seed.
func ConductiveGridFromSeed(g *state.Game, seed *world.Cell) *mapset.Set[*world.Cell] {
	return cellsReachableFromGeneratorSeed(g, seed)
}

func powerMaintTerminalsInGrid(g *state.Game, grid *mapset.Set[*world.Cell]) int {
	if g == nil || grid == nil {
		return 0
	}
	restored := 0
	grid.Each(func(c *world.Cell) {
		if c == nil {
			return
		}
		mt := gameworld.GetGameData(c).MaintenanceTerm
		if mt == nil || mt.Powered || mt.Disabled {
			return
		}
		mt.Powered = true
		restored++
	})
	return restored
}

// gridIncludesRoom reports whether any cell in power grid belongs to roomName.
func gridIncludesRoom(grid *mapset.Set[*world.Cell], roomName string) bool {
	if grid == nil || roomName == "" {
		return false
	}
	found := false
	grid.Each(func(c *world.Cell) {
		if c != nil && c.Name == roomName {
			found = true
		}
	})
	return found
}

// TripGeneratorsOnArmedGrid trips every powered generator on the given armed routing grid.
func TripGeneratorsOnArmedGrid(g *state.Game, grid *mapset.Set[*world.Cell]) int {
	if g == nil || g.Grid == nil || grid == nil || grid.Size() == 0 {
		return 0
	}
	tripped := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !grid.Has(cell) {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen != nil && gen.IsPowered() {
			gen.Trip()
			tripped++
		}
	})
	return tripped
}

// TripGeneratorsFeedingRoom trips every powered generator whose power grid includes roomName.
func TripGeneratorsFeedingRoom(g *state.Game, roomName string) int {
	if g == nil || g.Grid == nil || roomName == "" {
		return 0
	}
	tripped := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room {
			return
		}
		gen := gameworld.GetGameData(cell).Generator
		if gen == nil || !gen.IsPowered() {
			return
		}
		grid := CellsReachableInPowerGrid(g, cell)
		if gridIncludesRoom(grid, roomName) {
			gen.Trip()
			tripped++
		}
	})
	return tripped
}

// UnpowerTerminalsOffGeneratorGrid clears maintenance terminals not on a generator-fed power grid.
func UnpowerTerminalsOffGeneratorGrid(g *state.Game) int {
	if g == nil || g.Grid == nil {
		return 0
	}
	conductive := RoomsOnConductiveGeneratorGrid(g)
	cleared := 0
	g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
		if cell == nil || !cell.Room || cell.Name == "" || cell.Name == "Corridor" {
			return
		}
		if g.MaintenanceMenuTerminalRow >= 0 && cell.Row == g.MaintenanceMenuTerminalRow &&
			cell.Col == g.MaintenanceMenuTerminalCol {
			return
		}
		mt := gameworld.GetGameData(cell).MaintenanceTerm
		if mt == nil || !mt.Powered {
			return
		}
		if conductive[cell.Name] {
			return
		}
		mt.Powered = false
		cleared++
	})
	return cleared
}

// isGeneratorPowerSeed reports whether a cell is a powered generator (sole grid supply source).
func isGeneratorPowerSeed(g *state.Game, cell *world.Cell) bool {
	if g == nil || cell == nil {
		return false
	}
	data := gameworld.GetGameData(cell)
	return data.Generator != nil && data.Generator.IsPowered()
}

// ApplyGridConductivePower powers maintenance terminals on power grids fed only by powered generators.
func ApplyGridConductivePower(g *state.Game) int {
	if g == nil || g.Grid == nil {
		return 0
	}
	UnpowerTerminalsOffGeneratorGrid(g)
	live := CellsReachableFromPoweredGenerators(g)
	return powerMaintTerminalsInGrid(g, live)
}

// RestoreTerminalsInRooms powers unpowered maintenance terminals in the given rooms.
func RestoreTerminalsInRooms(g *state.Game, roomSet map[string]bool) (restored int, message string) {
	if g == nil || g.Grid == nil || len(roomSet) == 0 {
		return 0, "No target rooms"
	}
	g.Grid.ForEachCell(func(row, col int, c *world.Cell) {
		if c == nil || !c.Room || !roomSet[c.Name] {
			return
		}
		data := gameworld.GetGameData(c)
		if data.MaintenanceTerm == nil || data.MaintenanceTerm.Powered {
			return
		}
		data.MaintenanceTerm.Disabled = false
		data.MaintenanceTerm.Powered = true
		restored++
	})
	if restored > 0 {
		return restored, fmt.Sprintf("Restored power to %d terminal(s) via power grid", restored)
	}
	return 0, "No unpowered terminals on power grid"
}
