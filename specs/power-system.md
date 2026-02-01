# Power System Specification

This document describes the **current implementation** of the power system in The Dark Station. It covers supply, consumption, room power, maintenance terminals, and lighting/exploration.

---

## 1. Overview

The level has a **global power grid** supplied by **generators** and consumed by **room devices** (doors, CCTV, solved puzzles). A separate concept is **maintenance terminal power**: only some terminals can be used until the player restores power to others from a powered terminal. **Room power** (doors and CCTV) is toggled per room via maintenance terminals and does not depend on which terminal is “powered”; it is game state (`RoomDoorsPowered`, `RoomCCTVPowered`).

- **Supply**: Sum of power from all **powered generators** (batteries inserted).
- **Consumption**: Sum from **doors** (when room doors powered), **CCTV terminals** (when room CCTV powered), and **solved puzzle terminals**. Standard cells (lighting) do **not** consume power.
- **Maintenance terminals**: Each has a `Powered` flag. Only the start room’s terminal(s) start powered; others are restored via “Restore power to nearby terminals” at a powered terminal.
- **Room power**: Per-room booleans for door power and CCTV/hazard-control power, toggled at maintenance terminals (own room + adjacent rooms).

---

## 2. Power Supply

### 2.1 Generators

- **Entity**: `Generator` has `BatteriesRequired`, `BatteriesInserted`.
- **Powered when**: `BatteriesInserted >= BatteriesRequired`.
- **Output**: Each powered generator provides **100 W**.
- **Total supply**: `PowerSupply = (number of powered generators) × 100`.
- **Update**: `UpdatePowerSupply()` recomputes `g.PowerSupply` from all generators on the level.

### 2.2 Batteries

- Player carries **batteries** in inventory.
- Batteries are **inserted** into generators by interacting when adjacent; each generator has a fixed `BatteriesRequired`.
- Insertion is permanent for that level (no removal).

### 2.3 State

- **`PowerSupply`** (int): Total watts from all powered generators. Recalculated whenever generators or lighting/consumption is updated (e.g. after interaction or movement).
- **`GetAvailablePower()`**: `PowerSupply - PowerConsumption`.

---

## 3. Power Consumption

### 3.1 What Consumes Power

| Device / system           | Condition for consumption      | Watts |
|---------------------------|---------------------------------|-------|
| **Doors**                 | Room’s doors powered            | 10 per door (per room’s door set) |
| **CCTV terminals**       | Room’s CCTV powered             | 10 per terminal in that room |
| **Puzzle terminals**     | Puzzle solved                   | 3 per solved puzzle |
| **Maintenance terminals**| —                                | 0 (do not consume) |
| **Room lighting (cells)**| —                                | 0 (standard cells do not consume) |

- **Doors**: Consumption is per **room**: if `RoomDoorsPowered[roomName]` is true, every door cell that belongs to that room counts (e.g. 2 doors → 20 W for that room).
- **CCTV**: If `RoomCCTVPowered[roomName]` is true, each CCTV terminal in that room adds 10 W.
- **Puzzles**: Each cell with a puzzle terminal that `IsSolved()` adds 3 W.

### 3.2 What Does Not Consume Power

- **Standard cells / lighting**: Lit cells do **not** draw from the grid.
- **Maintenance terminals**: Purely control interfaces; no wattage.

### 3.3 Calculation and Overload

- **`g.CalculatePowerConsumption()`**: Method on `Game`; iterates the grid, sums the above. Result is stored in **`PowerConsumption`** (caller or lighting update does this).
- **Overload when toggling ON**: When the player turns **on** room doors or room CCTV at a maintenance terminal, if the new total consumption would exceed **PowerSupply**, the system **“shorts out”** instead of simply warning:
  - The requested room’s power is applied (doors or CCTV set to true).
  - **Short-out**: Other rooms’ doors and CCTV (never the room just turned on) are automatically turned **off** in a deterministic order until `PowerConsumption ≤ PowerSupply`. The room the player turned on is **protected** and stays on.
  - Order of unpowering: rooms (and within a room, doors then CCTV) in a fixed order (e.g. by room name) so behaviour is reproducible.
- **Passive overload**: If consumption already exceeds supply (e.g. after generators are damaged or supply drops), the game may warn once per cycle (`PowerOverloadWarned`). Lights still use `GetAvailablePower() > 0` for “lights on” logic.

### 3.4 Short-out API

- **`g.ShortOutIfOverload(protectedRoomName string) bool`**: Call **after** applying a room power toggle to ON (doors or CCTV for `protectedRoomName`). Recalculates supply and consumption; if `PowerConsumption > PowerSupply`, unpowers other rooms’ doors and CCTV (never `protectedRoomName`) in a deterministic order until `PowerConsumption ≤ PowerSupply`. Updates `g.PowerConsumption`. Returns **true** if any systems were unpowered. Used by the maintenance menu when the player toggles a room’s doors or CCTV on.

---

## 4. Room Power (Doors and CCTV)

### 4.1 State

- **`RoomDoorsPowered[roomName]`** (bool): When true, **door cells** that belong to that room are passable (subject to lock/keycard). When false, the player cannot pass through those doors.
- **`RoomCCTVPowered[roomName]`** (bool): When true, **CCTV terminals** and **hazard controls** in that room can be used. When false, interaction is blocked with a “no power” message.

Room power is **per room name** (e.g. "Depressurized Bridge", "Emergency Lab"). Corridor has no room power toggles.

### 4.2 Initialization

- **`InitRoomPower(g)`** (during level setup):
  - All rooms: `RoomDoorsPowered[R] = false`, `RoomCCTVPowered[R] = false`.
  - **Start room**: `RoomDoorsPowered[startRoomName] = true` so the player can leave.
  - CCTV remains false for all rooms unless changed later.

### 4.3 Solvability (Door Power)

- **`EnsureSolvabilityDoorPower(g)`** runs **after** maintenance terminals are placed.
- For any **gatekeeper room** R (every path from start to exit goes through R):
  - If R’s doors are unpowered **and** no room adjacent to R that has a maintenance terminal is reachable from start without entering R → **deadlock**.
  - In that case, R’s doors are **initially powered** so the level is solvable.
- So: the only rooms that may start with doors powered are the start room and (optionally) gatekeeper rooms that would otherwise be deadlocked.

### 4.4 Who Can Change Room Power

- Only **maintenance terminals** change `RoomDoorsPowered` and `RoomCCTVPowered`.
- A terminal can control:
  - Its **own room**,
  - **Adjacent rooms** (see §5.2).
- Adjacency: rooms that share a corridor boundary or a direct cell boundary with the terminal’s room (see `GetAdjacentRoomNames`).

### 4.5 Movement and Interaction

- **Doors**: `CanEnter` (or equivalent) checks `RoomDoorsPowered[roomName]` for the door’s room. If false, movement is blocked and a message indicates power must be restored via the maintenance terminal.
- **CCTV**: If `!RoomCCTVPowered[cell.Name]`, the CCTV terminal cannot be used; message to restore power via maintenance terminal.
- **Hazard controls**: Same as CCTV: require `RoomCCTVPowered[roomName]` for that room.

---

## 5. Maintenance Terminals

### 5.1 Role

- **Maintenance terminals** are the only way to:
  - Toggle **room door power** (per room),
  - Toggle **room CCTV power** (per room),
  - **Restore power to nearby maintenance terminals** (see §5.4).
- They also show power stats (supply, consumption, available) and a device list (doors, CCTV, puzzles, lighting, other terminals) with wattage where applicable.

### 5.2 Terminal Power (Powered Flag)

- Each **maintenance terminal** has a **`Powered`** (bool) field.
- **Use**: The player can **open the maintenance menu** only at a terminal that is **powered**. At an unpowered terminal, interaction shows a message like “Terminal has no power. Restore power from another maintenance terminal.” and the menu does not open.
- **Initial state** (after placement):
  - **`InitMaintenanceTerminalPower(g)`**: All maintenance terminals set `Powered = false`; then every terminal **in the start room** is set `Powered = true`.
  - So exactly the start room’s terminal(s) are usable at level start; all others must be restored (§5.4).

### 5.3 Restore Power to Nearby Terminals

- **Action**: “Restore power to nearby terminals” in the maintenance menu.
- **Available**: Only when using a **powered** terminal (menu is only open on powered terminals).
- **Effect**: For every room **adjacent** to the current terminal’s room (including the current room), every maintenance terminal in those rooms has **`Powered = true`**.
- **Adjacency**: Same as for room power control: `GetAdjacentRoomNames(grid, terminalRoomName)` — rooms that share a corridor boundary or a direct cell boundary with the terminal’s room, plus the terminal’s own room.
- **Feedback**: Message such as “Restored power to N terminal(s)” or “No unpowered terminals in nearby rooms”.

### 5.4 Menu Scope (Room Power Toggles)

- From one terminal the player can toggle **doors** and **CCTV** for:
  - The terminal’s **own room**,
  - All **adjacent rooms** (same definition as above).
- So the player does not need to visit every room to power its doors/CCTV; they can do it from any powered terminal that is in or adjacent to that room.

---

## 6. Lighting and Exploration

### 6.1 Lights and Power

- **Lights** (per-cell “lights on” state) do **not** consume power.
- **Available power** still controls whether lights are considered “on” for exploration:
  - If **`GetAvailablePower() > 0`** and the cell was **visited**, lights are turned on for that cell (`LightsOn = true`, `Lighted = true`, cell stays discovered/visited).
  - If **`GetAvailablePower() <= 0`**:
    - Lights off for that cell.
    - Cells **within a small radius of the player** (e.g. 3×3) remain visible regardless.
    - Other cells that are not “permanently lighted” can fade (discovered/visited cleared) so the map darkens when power is low.

So: lighting **visibility** depends on available power, but lighting does **not** add to consumption.

### 6.2 Update Order

- Each update (e.g. after movement or interaction):
  1. **Consumption** is recalculated and stored in `PowerConsumption`.
  2. **Supply** is recalculated via `UpdatePowerSupply()`.
  3. **Available power** is used to decide lights on/off and exploration (e.g. `UpdateLightingExploration`).

---

## 7. Visual and Feedback

### 7.1 Rendering (unpowered = red background)

Any **unpowered** power-dependent cell uses the same **red background** colour as unpowered doors (e.g. dark red / hazard style) to show it is unusable or off:

- **Doors**: Unpowered (`!RoomDoorsPowered[roomName]`) → red background.
- **Maintenance terminals**: Unpowered (`!MaintenanceTerminal.Powered`) → red background.
- **CCTV terminals**: Room CCTV off (`!RoomCCTVPowered[roomName]`) → red background.
- **Hazard controls**: Room CCTV off (`!RoomCCTVPowered[roomName]`) → red background.

Walls and corridor cells do **not** change background based on power (no “green” for powered rooms). **Generators**: Powered generators may use a distinct background (e.g. green) to show they are on.

### 7.2 Messages and Callouts

- **Overload (passive)**: One-time warning when consumption exceeds supply (e.g. “Power consumption exceeds supply”).
- **Overload (short-out)**: When turning on a system causes a short-out, the player is told that other systems shorted out (e.g. “Power overload! Other systems shorted out.”).
- **Doors**: “Door has no power. Restore power via the maintenance terminal.”
- **CCTV / hazard control**: “No power” / “Restore power via the maintenance terminal.”
- **Unpowered maintenance terminal**: “Terminal has no power. Restore power from another maintenance terminal.”
- **Generator**: Callout can show supply, consumption, and available power when interacting with a generator.

---

## 8. Lifecycle and Reset

### 8.1 Level Setup Order (relevant to power)

1. Grid and rooms exist; doors (and optionally locks) placed.
2. **`InitRoomPower(g)`**: All rooms unpowered; start room doors powered.
3. Hazards, furniture, puzzles, **maintenance terminals** placed.
4. **`EnsureSolvabilityDoorPower(g)`**: Gatekeeper rooms that would be deadlocked get doors powered.
5. **`InitMaintenanceTerminalPower(g)`**: All terminals unpowered; start room terminal(s) powered.
6. Player moved to start cell.

### 8.2 Reset Level

- **`ResetLevel`** (or equivalent) regenerates the level with the same seed and re-runs setup, including:
  - `InitRoomPower`,
  - `EnsureSolvabilityDoorPower`,
  - `InitMaintenanceTerminalPower`.
- So after reset, room power and maintenance terminal power are again in the initial state (start room doors powered, start room terminal(s) powered).

### 8.3 Advance Level

- When advancing to the next level, level-specific state (generators, room power, terminal power, etc.) is reset for the new level; the new level’s setup runs from scratch.

---

## 9. Summary Table

| Concept                 | Implementation summary |
|-------------------------|-------------------------|
| **Supply**              | 100 W per powered generator; `UpdatePowerSupply()` sums them. |
| **Consumption**         | Doors (10 W per room when doors on), CCTV (10 W per terminal when room CCTV on), solved puzzles (3 W each). No lighting/maintenance consumption. |
| **Room doors**          | `RoomDoorsPowered[room]`; toggled at maintenance terminals (own + adjacent rooms). Start room true; gatekeeper deadlocks fixed by `EnsureSolvabilityDoorPower`. |
| **Room CCTV**           | `RoomCCTVPowered[room]`; toggled at maintenance terminals (own + adjacent rooms). All start false. |
| **Terminal power**      | `MaintenanceTerminal.Powered`; only start room terminal(s) true initially; “Restore power to nearby terminals” powers terminals in adjacent rooms. |
| **Lighting**            | Visibility depends on `GetAvailablePower() > 0` and visit state; no wattage cost for cells. |
| **Unpowered visual**    | Any unpowered power-dependent cell (doors, maint terminal, CCTV, hazard control) uses red background. |
| **Overload**            | Passive: warning when `PowerConsumption > PowerSupply`. **Short-out**: when turning ON a room’s doors/CCTV would exceed supply, other rooms’ doors/CCTV are auto-unpowered until within budget; protected room stays on. |

This spec reflects the **current implementation** as of the last update; code in `pkg/game/state`, `pkg/game/gameplay/lighting.go`, `pkg/game/setup/roompower.go`, `pkg/game/menu/maintenance.go`, `pkg/game/renderer/ebiten/rendering.go`, and related files is the source of truth.
