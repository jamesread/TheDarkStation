# Power Faults and Diagnosis

This document describes the **current implementation** of hidden power-grid faults and the diagnosis loop that lets the player find and fix them. It builds on [`power-system.md`](power-system.md) (grid conduction, room power) and the knowledge-tier rendering introduced with power-driven lighting.

---

## 1. Design intent

The station should feel like a real machine that degrades and is clawed back to life. The player never loses a run; instead, restoring power to a branch of the deck is an intellectual exercise:

1. **Observe**: a wing is dark; doors and devices there are dead.
2. **Diagnose**: a maintenance terminal's bus trace names the fault class and gives distance and bearing — never exact coordinates.
3. **Locate**: walk the conduit run with the headlamp; the fault is physically visible on the corridor floor when lit or in headlamp range.
4. **Fix**: close the relay, or hold-USE to splice the burned conduit. Power floods back; lights come on downstream.

## 2. Fault types

| Fault | Entity | Conduction | Movement | Fix |
|-------|--------|------------|----------|-----|
| **Tripped breaker** | `PowerRelay` (starts `Closed=false`) | Blocks while open | Blocks (housing) | USE to close |
| **Burned conduit** | `RepairObjective` of type `RepairConduitSplice` | Blocks until repair complete | **Walkable** (floor channel) | Hold-USE to splice |

- `gameworld.RepairDeviceBlocksPowerGrid(cell)`: a conduit splice conducts again once complete; every other repair housing blocks the grid while present.
- `gameworld.RepairDeviceBlocksMovement(cell)`: conduit splices stay walkable; all other repair devices are impassable.
- Completing a splice calls `setup.NotifyPowerGridChanged` + `UpdateLightingExploration`, so downstream rooms light up immediately — the payoff is visible.

## 3. Placement (`pkg/game/levelgen/faults.go`)

`PlaceConduitFaults(g, avoid)` runs in `setupLevel` after `ApplyPowerRelays`:

- **Count**: scales with depth (`conduitFaultCount`): decks 2–4 → 1, decks 5–7 → 2, deck 8+ → 3. Deck 1 and the final deck stay clean.
- **Candidates** (`validConduitFaultCell`): corridor cells on a **straight run** (exactly two walkable neighbors), currently conducting live power (the outage must be observable), free of other entities/items, not the entry or exit cell, and not adjacent to a door or the lift shaft.
- **Determinism**: candidates are position-sorted then shuffled with `levelrand.NewDerived(LevelSeed, salt)`; the same seed always yields the same faults.
- Because the splice is walkable it never severs routing, so no blocking-entity validation is needed. Solvability is still covered: splices are regular repair objectives the progression simulator must complete (adjacent-reachable, no prereqs, `RequiresPower=false` — a dead conduit must be splicable in the dark).
- Each fault gets a diegetic **segment label** (`SEG-xx`, derived from position) stored in `RepairObjective.SegmentLabel` and shown in its name ("Conduit Splice SEG-3F").

## 4. Diagnosis: bus trace (`pkg/game/setup/power_trace.go`)

`TraceBusFault(g, terminalCell, targetRoom)` BFS-walks the physical conduit network from the terminal toward the target room and reports the **first element that interrupts conduction**:

| `PowerFaultKind` | Meaning | Readout hint |
|------------------|---------|--------------|
| `PowerFaultNone` | Room is energized | "BUS OK" |
| `PowerFaultOpenRelay` | Open relay on the run | steps + bearing — "close relay" |
| `PowerFaultBurnedConduit` | Burned segment on the run | `SEG-xx` + steps + bearing — "splice conduit" |
| `PowerFaultNoSupply` | No generator online anywhere | "no generator online" |
| `PowerFaultUnarmed` | Bus intact, circuits not armed | "arm room circuits" |
| `PowerFaultNoRoute` | No conduit path exists | "bus not mapped" |

- The readout gives **distance (steps) and compass bearing** from the terminal, plus the segment label — never exact coordinates. The player still walks the run to find the scorched segment.
- Faulted elements are traversable for the trace itself (they are what it is looking for); hard non-conducting obstructions (furniture, other repair housings) are not.
- The trace line is shown in the maintenance terminal **Diagnostics** mode (`getDiagnosticsMenuItems` in `pkg/game/menu/maintenance_routing.go`) for the currently viewed room, formatted by `FormatBusTraceLine`.

## 5. Physical inspection cues

- The splice device renders with icon `=` (`IconRepairConduit`) when the cell is in the **live** knowledge tier (lit or in headlamp range); in the **remembered** tier it keeps its identity in neutral dim colors.
- Walking onto a splice (it is walkable) pops its callout: name with segment label and "Hold USE to repair".
- Hold-USE works from an adjacent cell or standing on the splice (`findAdjacentLongUseTarget` includes the current cell).
- The segment label in the callout matches the label in the terminal bus trace, closing the loop between diagnosis and field repair.

## 6. Exit gating

Conduit splices are normal deck repairs: `ExitLiftReady` requires them complete (no `SkipExitGate`). They are exempt from the room-based exit-gating relocation logic (`ExitGatingRepairsAccessible` skips `RepairConduitSplice`) because they live on corridors and their reachability is validated by the progression simulator.

## 7. Debugging

`map.txt` (F8) includes:

- Glyph `R` for repair devices and `r` for power relays in the grid.
- A `Repairs:` section listing every objective with `type`, `status`, and `segment` label for conduit faults.
- A `Power relays:` section listing each relay position with its `closed` state.

## 8. Conservation policies (the station fights back)

Per the GDD: the station is "obedient to dead rules" — deterministic, legible, never random attrition, never a lost run.

### 8.1 Model (`pkg/game/entities/policy.go`)

`ConservationPolicy{ID, Code, Kind, TargetRoom, DelayMs, Overridden}`:

| Kind | Code | Rule |
|------|------|------|
| `PolicyShedFirst` | `HAB-PRI` | Under overload, the target room's loads shed **before** any other consumer |
| `PolicyEgressSeal` | `ATMOS-SEAL` | A manual door release on a room without power re-seals after 30s |

### 8.2 Placement (`pkg/game/levelgen/policies.go`)

`PlaceConservationPolicies(g)` runs in `setupLevel`: decks 4+ get a shed-first policy (deterministic room pick via `levelrand.NewDerived`), decks 6+ add egress-seal. Any policy-bearing deck also gets exactly one **Crew Override Authorization** item on an init-reachable floor cell.

Policies can never make a deck unsolvable: shed-first only reorders overload shedding, and egress-seal re-seals a release the player can simply pull again (or counter properly by powering the room).

### 8.3 Enforcement

- **Shed-first**: `sortShedQueue` in `pkg/game/setup/power_balance.go` ranks policy-targeted rooms first. The same ordering drives `ShortOutIfOverload`, `PreviewShortOutIfOverload`, and `PreviewRoomPresetConsumption`, so the maintenance menu preview always matches what the station will actually do.
- **Egress-seal**: manual releases record a timestamp (`ManualEgressReleasedAtMs`); `setup.AdvanceEgressSeal` (ticked from `UpdateLightingExploration`) re-seals due releases on rooms without live power and logs "ATMOS-SEAL: room egress re-secured."

### 8.4 Discoverability and override

- The Diagnostics panel of every maintenance terminal lists the deck's policies (`STATION POLICIES` section): code, ACTIVE/DEPRECATED status, and the full rule text. Learning the rule **is** the counter.
- `Deprecate station policies` menu action: consumes a carried Crew Override Authorization and permanently sets every deck policy `Overridden` (persisted in `DeckStates`). Nothing the player repaired or overrode ever reverts.

## 9. Tests

- `pkg/game/setup/power_trace_test.go`: trace classification (burned conduit with label/steps/bearing, open relay, no supply, unarmed, BUS OK) and conduction semantics (splice conducts only when complete; other housings always block).
- `pkg/game/gameplay/conduit_fault_test.go`: seed sweep — faults placed on most mid-depth seeds, always on corridors with labels, decks remain simulator-solvable, placement deterministic per seed, splicing extends live power coverage.
- `pkg/game/setup/policies_test.go`: shed queue policy bias, egress re-seal timing (skips powered rooms, inert without policy), override permanence.
- `pkg/game/gameplay/policy_placement_test.go`: policy presence by depth, exactly one override item, deterministic per seed.
