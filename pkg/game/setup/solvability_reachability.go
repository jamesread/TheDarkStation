// Package setup provides level setup functionality for The Dark Station.
package setup

import (
	"fmt"
	"sort"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// MovementBlockReason describes why a cell cannot be entered at level init (no keycards owned).
type MovementBlockReason string

const (
	MovementOK            MovementBlockReason = "ok"
	MovementWall          MovementBlockReason = "wall"
	MovementBlockedEntity MovementBlockReason = "blocked_entity"
	MovementUnpoweredDoor MovementBlockReason = "unpowered_door"
	MovementLockedDoor    MovementBlockReason = "locked_door"
)

// CanEnterCellAtInit reports whether the player could step onto cell at level start (no keycards).
func CanEnterCellAtInit(g *state.Game, cell *world.Cell) (bool, MovementBlockReason) {
	if g == nil || cell == nil || !cell.Room {
		return false, MovementWall
	}
	if gameworld.HasDoor(cell) {
		roomName := gameworld.GetGameData(cell).Door.RoomName
		if !g.RoomDoorsPowered[roomName] {
			return false, MovementUnpoweredDoor
		}
		if gameworld.HasLockedDoor(cell) {
			return false, MovementLockedDoor
		}
	}
	if gameworld.HasGenerator(cell) || gameworld.HasFurniture(cell) ||
		gameworld.HasTerminal(cell) || gameworld.HasPuzzle(cell) ||
		gameworld.HasMaintenanceTerminal(cell) || gameworld.HasHazardControl(cell) ||
		gameworld.HasBlockingHazard(cell) {
		return false, MovementBlockedEntity
	}
	return true, MovementOK
}

// InitialReachableCells returns cells reachable from start at level init (no keycards, current door power).
func InitialReachableCells(g *state.Game) *mapset.Set[*world.Cell] {
	empty := mapset.New[*world.Cell]()
	if g == nil || g.Grid == nil {
		return &empty
	}
	start := g.Grid.StartCell()
	if start == nil {
		return &empty
	}
	reachable := mapset.New[*world.Cell]()
	queue := []*world.Cell{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == nil || !cur.Room || reachable.Has(cur) {
			continue
		}
		ok, _ := CanEnterCellAtInit(g, cur)
		if !ok {
			continue
		}
		reachable.Put(cur)
		for _, n := range cur.GetNeighbors() {
			if n != nil && n.Room && !reachable.Has(n) {
				queue = append(queue, n)
			}
		}
	}
	return &reachable
}

// RoomMaintenanceTerminalPowered reports whether any maintenance terminal in roomName is powered.
func RoomMaintenanceTerminalPowered(g *state.Game, roomName string) bool {
	if g == nil || g.Grid == nil || roomName == "" {
		return false
	}
	powered := false
	g.Grid.ForEachCell(func(row, col int, c *world.Cell) {
		if c == nil || !c.Room || c.Name != roomName {
			return
		}
		data := gameworld.GetGameData(c)
		if data.MaintenanceTerm != nil && data.MaintenanceTerm.Powered {
			powered = true
		}
	})
	return powered
}

// CanControlRoomPower reports whether doors/CCTV/lights for targetRoom may be toggled from controllerRoom
// (local terminal powered, or remote from an adjacent room with a powered terminal per spec §2.2).
func CanControlRoomPower(g *state.Game, controllerRoom, targetRoom string) bool {
	if g == nil || targetRoom == "" {
		return false
	}
	if RoomMaintenanceTerminalPowered(g, targetRoom) {
		return true
	}
	if controllerRoom == "" || !RoomMaintenanceTerminalPowered(g, controllerRoom) {
		return false
	}
	for _, adj := range GetAdjacentRoomNames(g.Grid, controllerRoom) {
		if adj == targetRoom {
			return true
		}
	}
	return false
}

// CanPowerRoomDoorsFromReachable reports whether targetRoom door power can be toggled using terminals
// in the given reachable cell set (local or remote adjacent control).
func CanPowerRoomDoorsFromReachable(g *state.Game, reachable *mapset.Set[*world.Cell], targetRoom string) bool {
	if g == nil || reachable == nil || targetRoom == "" {
		return false
	}
	if RoomMaintenanceTerminalPowered(g, targetRoom) {
		reachableHasTarget := false
		reachable.Each(func(c *world.Cell) {
			if c.Name == targetRoom {
				reachableHasTarget = true
			}
		})
		if reachableHasTarget {
			return true
		}
	}
	controllerRooms := make(map[string]bool)
	reachable.Each(func(c *world.Cell) {
		if c.Name != "" && c.Name != "Corridor" {
			controllerRooms[c.Name] = true
		}
	})
	for q := range controllerRooms {
		if CanControlRoomPower(g, q, targetRoom) {
			return true
		}
	}
	return false
}

// EgressDoor describes a door leaving the initial reachable pocket that requires target room door power.
type EgressDoor struct {
	Row, Col     int
	FromRoom     string
	TargetRoom   string
	Locked       bool
	DoorsPowered bool
}

// InitialEgressDoors lists unlocked doors on the boundary of the initial reachable set whose target room doors are off.
func InitialEgressDoors(g *state.Game) []EgressDoor {
	reachable := InitialReachableCells(g)
	if reachable.Size() == 0 {
		return nil
	}
	var out []EgressDoor
	seen := make(map[string]bool)
	reachable.Each(func(c *world.Cell) {
		for _, n := range c.GetNeighbors() {
			if n == nil || !n.Room || reachable.Has(n) {
				continue
			}
			if !gameworld.HasDoor(n) {
				continue
			}
			d := gameworld.GetGameData(n).Door
			if d == nil || d.Locked {
				continue
			}
			key := fmt.Sprintf("%d,%d", n.Row, n.Col)
			if seen[key] {
				continue
			}
			seen[key] = true
			if g.RoomDoorsPowered[d.RoomName] {
				continue
			}
			out = append(out, EgressDoor{
				Row:          n.Row,
				Col:          n.Col,
				FromRoom:     c.Name,
				TargetRoom:   d.RoomName,
				Locked:       d.Locked,
				DoorsPowered: g.RoomDoorsPowered[d.RoomName],
			})
		}
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].Row != out[j].Row {
			return out[i].Row < out[j].Row
		}
		return out[i].Col < out[j].Col
	})
	return out
}

// EnsureSolvabilityStartRoomEgress powers doors for rooms that block leaving the start pocket when
// the player cannot toggle those doors from any initially reachable powered terminal (I6 extension).
func EnsureSolvabilityStartRoomEgress(g *state.Game) {
	if g == nil || g.Grid == nil {
		return
	}
	reachable := InitialReachableCells(g)
	for _, door := range InitialEgressDoors(g) {
		if CanPowerRoomDoorsFromReachable(g, reachable, door.TargetRoom) {
			continue
		}
		g.RoomDoorsPowered[door.TargetRoom] = true
	}
}

// EnsureSolvability applies all post-generation solvability fixes.
func EnsureSolvability(g *state.Game) {
	EnsureSolvabilityDoorPower(g)
	EnsureSolvabilityStartRoomEgress(g)
	EnsureExitReachability(g)
}

// SolvabilityReport holds analysis output for debug dumps and validation.
type SolvabilityReport struct {
	StartRoom             string
	PlayerRoom            string
	InitialReachableCells int
	InitialReachableRooms []string
	BlockedEgressDoors    []EgressDoor
	Warnings              []string
	ExitReachableAtInit   bool
	StartRoomDoorsPowered bool
	StartMaintPowered     bool
}

// AnalyzeSolvability computes reachability and solvability warnings for the current game state.
func AnalyzeSolvability(g *state.Game) SolvabilityReport {
	report := SolvabilityReport{}
	if g == nil || g.Grid == nil {
		report.Warnings = append(report.Warnings, "no grid")
		return report
	}
	start := g.Grid.StartCell()
	if start != nil {
		report.StartRoom = start.Name
		report.StartRoomDoorsPowered = g.RoomDoorsPowered[start.Name]
		report.StartMaintPowered = RoomMaintenanceTerminalPowered(g, start.Name)
	}
	if g.CurrentCell != nil {
		report.PlayerRoom = g.CurrentCell.Name
	}
	reachable := InitialReachableCells(g)
	report.InitialReachableCells = reachable.Size()
	roomSet := make(map[string]bool)
	reachable.Each(func(c *world.Cell) {
		if c.Name != "" && c.Name != "Corridor" {
			roomSet[c.Name] = true
		}
	})
	for name := range roomSet {
		report.InitialReachableRooms = append(report.InitialReachableRooms, name)
	}
	sort.Strings(report.InitialReachableRooms)

	report.BlockedEgressDoors = InitialEgressDoors(g)
	exit := g.Grid.ExitCell()
	if exit != nil {
		report.ExitReachableAtInit = reachable.Has(exit)
	}

	if !report.StartRoomDoorsPowered {
		report.Warnings = append(report.Warnings, "start room doors not powered at init (I6 violation)")
	}
	if !report.StartMaintPowered {
		report.Warnings = append(report.Warnings, "start room has no powered maintenance terminal")
	}
	for _, door := range report.BlockedEgressDoors {
		if !CanPowerRoomDoorsFromReachable(g, reachable, door.TargetRoom) {
			report.Warnings = append(report.Warnings,
				fmt.Sprintf("egress blocked: door (%d,%d) -> %q cannot be powered from initial reachable terminals",
					door.Row, door.Col, door.TargetRoom))
		}
	}
	if !report.ExitReachableAtInit {
		report.Warnings = append(report.Warnings, "exit not reachable at init (expected until doors powered/keycards found)")
	}
	if !ExitReachableWhenCompletable(g, nil) {
		report.Warnings = append(report.Warnings, "exit not reachable when level completable (R7 violation)")
	}
	return report
}
