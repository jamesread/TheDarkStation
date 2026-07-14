package setup

import (
	"fmt"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// PowerFaultKind classifies the first conduction blocker found on a bus trace.
type PowerFaultKind string

const (
	// PowerFaultNone: the target room is energized; nothing to trace.
	PowerFaultNone PowerFaultKind = ""
	// PowerFaultOpenRelay: an open corridor relay interrupts the bus.
	PowerFaultOpenRelay PowerFaultKind = "relay_open"
	// PowerFaultBurnedConduit: a burned conduit segment (conduit splice repair) interrupts the bus.
	PowerFaultBurnedConduit PowerFaultKind = "conduit_burned"
	// PowerFaultNoSupply: no generator is online anywhere on the deck.
	PowerFaultNoSupply PowerFaultKind = "no_supply"
	// PowerFaultUnarmed: the bus is physically intact but the room circuits are not armed.
	PowerFaultUnarmed PowerFaultKind = "unarmed"
	// PowerFaultNoRoute: no physical conduit path exists from the terminal to the room.
	PowerFaultNoRoute PowerFaultKind = "no_route"
)

// PowerTraceResult is one functional bus-trace readout from a maintenance terminal.
type PowerTraceResult struct {
	Kind    PowerFaultKind
	Row     int    // fault location (physical faults only)
	Col     int
	Label   string // diegetic segment label (conduit faults: SEG-xx)
	Steps   int    // conduit steps from the terminal to the fault
	Bearing string // compass direction from the terminal (N/NE/E/SE/S/SW/W/NW)
}

// TraceBusFault walks the physical conduit network from a terminal cell toward the
// target room and reports the first element that interrupts conduction. The result
// names the fault class and gives distance plus bearing — never exact coordinates —
// so the player still has to walk the run and find the segment.
func TraceBusFault(g *state.Game, from *world.Cell, targetRoom string) PowerTraceResult {
	if g == nil || g.Grid == nil || from == nil || targetRoom == "" {
		return PowerTraceResult{Kind: PowerFaultNoRoute}
	}
	if RoomConsideredPowered(g, targetRoom) {
		return PowerTraceResult{Kind: PowerFaultNone}
	}

	path := shortestConduitPath(g, from, targetRoom)
	if path == nil {
		return PowerTraceResult{Kind: PowerFaultNoRoute}
	}
	for i, cell := range path {
		if gameworld.RelayBlocksGrid(cell) {
			return physicalFault(PowerFaultOpenRelay, from, cell, i, "")
		}
		if gameworld.RepairDeviceBlocksPowerGrid(cell) {
			label := ""
			if dev := gameworld.GetGameData(cell).RepairDevice; dev != nil {
				if dev.Type == entities.RepairConduitSplice {
					label = dev.SegmentLabel
				}
			}
			return physicalFault(PowerFaultBurnedConduit, from, cell, i, label)
		}
	}
	if !anyGeneratorPowered(g) {
		return PowerTraceResult{Kind: PowerFaultNoSupply}
	}
	return PowerTraceResult{Kind: PowerFaultUnarmed}
}

func physicalFault(kind PowerFaultKind, from, cell *world.Cell, steps int, label string) PowerTraceResult {
	return PowerTraceResult{
		Kind:    kind,
		Row:     cell.Row,
		Col:     cell.Col,
		Label:   label,
		Steps:   steps,
		Bearing: compassBearing(from, cell),
	}
}

// shortestConduitPath BFS-walks room cells from start to the nearest cell of
// targetRoom. Faulted elements (open relays, burned conduits) are traversable —
// they are exactly what the trace is looking for — but hard obstructions that
// never conduct (furniture, non-conduit repair housings) are not.
func shortestConduitPath(g *state.Game, start *world.Cell, targetRoom string) []*world.Cell {
	parents := map[*world.Cell]*world.Cell{start: nil}
	queue := []*world.Cell{start}
	var goal *world.Cell
	for len(queue) > 0 && goal == nil {
		cur := queue[0]
		queue = queue[1:]
		for _, n := range cur.GetNeighbors() {
			if n == nil || !n.Room {
				continue
			}
			if _, seen := parents[n]; seen {
				continue
			}
			if !conduitTraceTraversable(n) {
				continue
			}
			parents[n] = cur
			if n.Name == targetRoom {
				goal = n
				break
			}
			queue = append(queue, n)
		}
	}
	if goal == nil {
		return nil
	}
	var rev []*world.Cell
	for c := goal; c != nil; c = parents[c] {
		rev = append(rev, c)
	}
	path := make([]*world.Cell, 0, len(rev))
	for i := len(rev) - 1; i >= 0; i-- {
		path = append(path, rev[i])
	}
	return path
}

func conduitTraceTraversable(cell *world.Cell) bool {
	if gameworld.FurnitureBlocksPowerGrid(cell) {
		return false
	}
	if dev := gameworld.GetGameData(cell).RepairDevice; dev != nil && dev.Type != entities.RepairConduitSplice {
		return false
	}
	return true
}

// compassBearing returns the rough direction from a to b in screen coordinates
// (north = decreasing row, east = increasing col).
func compassBearing(a, b *world.Cell) string {
	if a == nil || b == nil {
		return ""
	}
	dRow := b.Row - a.Row
	dCol := b.Col - a.Col
	ns := ""
	if dRow < 0 {
		ns = "N"
	} else if dRow > 0 {
		ns = "S"
	}
	ew := ""
	if dCol > 0 {
		ew = "E"
	} else if dCol < 0 {
		ew = "W"
	}
	// Suppress the minor axis when the major axis dominates strongly (>2:1).
	absR, absC := absIntTrace(dRow), absIntTrace(dCol)
	if absR > 2*absC {
		ew = ""
	}
	if absC > 2*absR {
		ns = ""
	}
	return ns + ew
}

func absIntTrace(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// FormatBusTraceLine renders a trace result as a tab-aligned diagnostics line.
func FormatBusTraceLine(res PowerTraceResult) string {
	switch res.Kind {
	case PowerFaultNone:
		return "TRACE\tPOWERED{BUS OK}\tsegment energized"
	case PowerFaultOpenRelay:
		return fmt.Sprintf("TRACE\tUNPOWERED{RELAY OPEN}\t%dm %s — close relay", res.Steps, res.Bearing)
	case PowerFaultBurnedConduit:
		label := res.Label
		if label == "" {
			label = "SEGMENT"
		}
		return fmt.Sprintf("TRACE\tUNPOWERED{%s BURNOUT}\t%dm %s — splice conduit", label, res.Steps, res.Bearing)
	case PowerFaultNoSupply:
		return "TRACE\tUNPOWERED{NO SUPPLY}\tno generator online"
	case PowerFaultUnarmed:
		return "TRACE\tUNPOWERED{BUS IDLE}\tarm room circuits"
	default:
		return "TRACE\tSUBTLE{NO ROUTE}\tbus not mapped"
	}
}
