# Power Routing UX

Companion to [`power-system.md`](power-system.md). Describes how players **control** the grid via maintenance terminals after the routing-table UX refresh.

## Goals

- **Controls-first** terminal: power actions reachable without scrolling past diagnostic strata.
- **Room circuits** as presets (OFF / ESSENTIAL / FULL) instead of three separate door/CCTV toggles for daily play.
- **Preview** overload shedding before committing a preset.
- **Targeted restore** of terminal power on the grid (all reachable vs selected room only).
- **Map context** while the menu is open: highlight viewed room, show per-room load on adjacent walls.

Watt math, solvability (`EnsureSolvabilityDoorPower`), and generator/battery rules are unchanged.

## Maintenance menu modes

| Mode | Contents |
|------|----------|
| **Controls** (default) | Flavour line, global supply stats, circuit preset, restore actions, ping, mode switch, close |
| **Diagnostics** | Instrument strata (Story 5.4), device list, per-room consumption, **Advanced** door/light/CCTV toggles, other terminals in room |

Toggle: **Tab** or menu row `Diagnostics…` / `Back to controls`.

## Room targeting

- **A / D** (or Left / Right): cycle `selectedRoomName` among `selectableRooms` (terminal room + corridor-adjacent rooms). Updates `MaintenanceMenuRoom` for map pan/highlight.
- No nested room-picker sub-menu in Controls mode.

## Room circuit presets

| Preset | Doors | CCTV | Lights |
|--------|-------|------|--------|
| OFF | off | off | unchanged |
| ESSENTIAL | on | off | unchanged |
| FULL | on | on | unchanged |

- **Enter** on `Circuit preset` cycles OFF → ESSENTIAL → FULL → OFF and applies immediately (one `ShortOutIfOverload` pass when turning loads on).
- **1 / 2 / 3** (while maintenance menu open): apply OFF / ESSENTIAL / FULL to the **currently viewed** room without cycling the menu row.
- Help text while focused shows **preview** of rooms/systems that would shed (`Game.PreviewShortOutIfOverload`).

Granular toggles remain under **Diagnostics → Advanced** for tests and edge cases.

## Restore actions

| Action | Behaviour |
|--------|-----------|
| Restore all adjacent | Same as legacy “Restore power to nearby terminals” |
| Restore selected room | Powers unpowered maintenance terminals only in `selectedRoomName` |

## Ping

Ping discovers undiscovered CCTV/puzzle cells within radius; results appear as **inline help text** (menu stays open). No nested results screen.

## Map overlay (menu open)

When `MaintenanceMenuRoom` is set:

- Viewed room walls use existing maintenance highlight.
- **Adjacent** selectable rooms show a compact power hint on wall cells: door/CCTV state and approximate room load (W).
- Menu chrome may show `Supply | Used | Free` (see renderer).

## API

- `menu.CircuitPreset`, `menu.ApplyCircuitPreset`, `menu.PreviewCircuitShed`
- `state.Game.PreviewShortOutIfOverload(protectedRoom, doorsOn, cctvOn) []state.PowerShedEntry`

## Phase 3: Power grid (spatial propagation)

Terminal control power propagates along a **power grid**, not flat room adjacency.

### Power grid rules

- BFS from the active maintenance terminal cell (or seeds for solvability).
- A cell is traversable when:
  - It is not blocked by an **open** relay (`PowerRelay` present and `Closed == false`).
  - **Locked doors** block traversal.
  - **Unpowered doors** block traversal (door cell’s room has `RoomDoorsPowered == false`).
- Rooms entered on the power grid are those with any visited non-corridor cell.

### Maintenance menu (Phase 3 updates)

| Action | Behaviour |
|--------|-----------|
| Restore power grid | Powers unpowered terminals in all rooms reachable from this terminal via the power grid |
| Restore selected room | Only succeeds if the viewed room is on the power grid from this terminal |
| Room list (A/D) | `SelectableRoomsForTerminal` — power grid-reachable rooms, not raw adjacency |

### Corridor relays

- Placed on corridor **junctions** (≥3 corridor neighbours), deck **level ≥ 3**, not on final deck.
- **Interact** (E) toggles open/closed. Default mostly **closed**; some start **open** on deeper decks.
- Rendered as `╬` (closed) / `╳` (open) on the map.

### Solvability

`EnsureSolvabilityDoorPower` uses `RoomsReachableInPowerGridExcluding` instead of only geometric adjacency when checking whether a gatekeeper can be powered from a neighbouring terminal.

## Out of scope

- Changes to generator battery insertion.
