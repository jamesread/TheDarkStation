# Level Layout and Solvability Specification

This document defines the **logical layout** domain for The Dark Station: reachability, dependencies, and invariants that guarantee levels are **solvable** as long as the player works out the logical chain. It is the source of truth for level generation and layout rules, and for adding new puzzle or logic elements without introducing deadlocks.

---

## 1. Domain: Logical Layout

### 1.1 What “Logical Layout” Covers

- **Spatial structure**: Grid, rooms, corridors, which cells are walkable.
- **Gates**: Doors (physical cells that require conditions to pass).
- **Conditions**: Door power (per room), keycards (per locked door), hazards cleared, generators powered.
- **Control points**: Maintenance terminals (where the player toggles room door/CCTV power), hazard controls, puzzle terminals, etc.
- **Objectives**: Exit cell, generators to power, hazards to clear, items to collect.

Logical layout is **not** about visuals, FOV, or lighting—only about **whether a sequence of player actions can satisfy all win conditions**.

### 1.2 Win Conditions (Current)

A level is won when the player:

1. **Reaches the exit cell** (lift to next deck).
2. **Has all generators powered** (batteries inserted).
3. **Has all blocking hazards cleared** (fixed via control or item).

Any gate or dependency that blocks progress toward these must be satisfiable in some order without deadlock.

---

## 2. Dependency Model

### 2.1 Movement Gates

| Gate | Condition to pass | Controlled by |
|------|-------------------|----------------|
| **Door (physical cell)** | 1) Room’s doors must be **powered**. 2) If door is locked, player must have the **keycard** for that room. | Door power: maintenance terminal. Keycard: found in world (e.g. behind other doors / in furniture). |
| **Blocking hazard** | Hazard must be **fixed** (control activated or item used). | Hazard control in some room, or item in world. |
| **Generator** | Not a gate for movement; blocks the cell. Must be **powered** (batteries) for exit. | Batteries placed by player. |
| **Furniture / terminal / etc.** | Block movement (impassable). | N/A. |

### 2.2 Door Power

- **RoomDoorsPowered[room]** and **RoomCCTVPowered[room]** are per-room state.
- **Only maintenance terminals** change this state.
- A maintenance terminal can only control power for:
  - Its **own room**, and
  - **Rooms that are directly adjacent** (share a corridor boundary or cell boundary with that room).

So: to power room **R**’s doors, the player must stand at a maintenance terminal in a cell that is either **in R** or **in a room adjacent to R**.

### 2.3 Reachability

- **Start**: Player starts at **StartCell** (a specific room; its name is the “start room”).
- **Initial door power**: Only the **start room** has its doors powered at init. All other rooms start with doors (and CCTV) unpowered.
- **Reachable set**: The set of cells the player can step into, given current state (door power, keycards, hazards fixed), without stepping on blocking entities (generator, furniture, terminal, etc.).

A room **R** is **reachable** if some cell in R is reachable. A room is **reachable without passing R** if it is reachable using only doors/gates that do not require entering R first.

---

## 3. Problem Class: Control-Dependency Deadlock

### 3.1 Description

**Control-dependency deadlock** occurs when:

- To reach a **goal** (exit or required objective), the player **must** pass through a **gate** (e.g. a door into room R).
- The gate’s condition (e.g. “R’s doors powered”) can **only** be satisfied by using a **control point** (e.g. maintenance terminal).
- That control point is **only** reachable **after** passing the gate (e.g. the only maintenance terminal that can power R is **inside R**).

Then the player can never satisfy the condition (cannot enter R to power R’s doors) and the level is **unsolvable**.

### 3.2 Example (From Production Bug)

- Exit is in/near **Emergency Lab**; only path from start to exit goes through the **door into Emergency Lab**.
- That door requires **Emergency Lab’s doors to be powered**.
- The only maintenance terminal that can power Emergency Lab is **inside Emergency Lab**.
- So: cannot enter Emergency Lab without powering its doors, and cannot power its doors without entering Emergency Lab → **deadlock**.

### 3.3 General Form

For **any** gate that depends on **room power** (or any “control” that the player must use from a specific location):

- Let **R** = room whose state must be changed (e.g. “doors powered”) to pass the gate.
- Let **T** = set of cells that contain a control (maintenance terminal) that can change R’s state.
- **Deadlock** occurs if:
  - Every path from start to the exit (or to a required objective) goes through a door into R, and
  - R’s doors are not initially powered, and
  - Every cell in T is in R (or in a room that is only reachable by first entering R).

So to **avoid** this class: for every room R that is a **gatekeeper** (only path to exit/objective goes through R), either R’s doors are initially powered, or there must be a control for R’s doors in a room that is **reachable without entering R** (e.g. an adjacent room reachable from start without going through R).

---

## 4. Invariants for Solvability

These invariants must hold **after** level setup (doors, room power init, generators, batteries, hazards, maintenance terminals, etc.) so that the level is solvable.

### I1. Exit reachable under some action order

There exists a sequence of player actions (move, interact, use keycard, toggle power, fix hazard, etc.) such that the player can eventually **step onto the exit cell** and satisfy **all** win conditions (generators powered, hazards cleared).

- Implies: no **permanent** deadlock (e.g. control-dependency deadlock, or keycard placed behind an impassable gate).

### I2. Door-power chain (no control deadlock)

For every room **R** that is not the start room:

- If **every path from start to the exit goes through a door into R** (R is a *gatekeeper*), then **at least one** of:
  - **R’s doors are initially powered**, or
  - There exists a maintenance terminal in a room **Q ≠ R** such that:
    - **Q is adjacent to R** (so the terminal can control R), and
    - **Q is reachable from start without stepping through an unpowered door into R** and **without stepping through a locked door** (locked doors are impassable until the player has the keycard).

So: you never **require** entering R to power R’s doors when R is the only way to the exit.

**Important**: When deciding “is R a gatekeeper?” and “is Q reachable without entering R?”, **locked door cells must be treated as impassable**. Otherwise a path that only exists by stepping through a locked door (e.g. Overgrown Maintenance Bay) would wrongly make R appear non-gatekeeper, and R’s doors would not be powered—leaving the real only path (through R) deadlocked.

### I3. Keycard chain (no keycard deadlock)

For every **locked** door (into room R):

- The keycard for R is placed in a cell that is **reachable from start** without using that keycard (i.e. keycard is behind other doors/gates that are either unlocked, or have their keycards also reachable in some order). No circular keycard dependency.

(Current implementation already ensures this: keycards are placed in “reachableWithDoors” — the area reachable **before** placing the new locked doors.)

### I4. Hazard chain (no hazard deadlock)

For every **blocking hazard**:

- Either the hazard is fixed by an **item** that is reachable (without passing that hazard), or
- It is fixed by a **hazard control** in a room that is reachable without passing that hazard (e.g. control in an adjacent room or earlier in the path).

(Level gen should not place a hazard that blocks the only path to its control or to its required item.)

### I5. Generator and batteries

- Every generator can be **reached** (no generator behind an impassable deadlock).
- Enough **batteries** are placed in reachable locations so that all generators can be powered.

### I6. Start room doors powered

- The **start room** always has its **doors powered** at init, so the player can leave the start room.

### I7. Room connectivity (no internal disconnection)

- For every **named room** (not "Corridor"), the **walkable** cells within that room (excluding blocking entities such as furniture, terminals, hazards, etc.) must form a **single connected component**.
- So: from any **doorway** (room cell adjacent to a corridor entry) the player can walk to any other doorway and to any **control point** (e.g. maintenance terminal) in that room without leaving the room.
- If a room is disconnected by blocking entities (e.g. furniture or a terminal placed such that one pocket has a doorway and another has the only maintenance terminal), the level can become unsolvable (e.g. player cannot reach the terminal to power doors).

---

## 5. Rules for Level Generation and Layout

These rules implement the invariants above and must be followed by level generation and layout code (and when adding new systems).

### R1. Room power initialization

- **Start room**: `RoomDoorsPowered[start_room] = true`, `RoomCCTVPowered[start_room] = false` (or as designed).
- **All other rooms**: `RoomDoorsPowered[R] = false`, `RoomCCTVPowered[R] = false`.
- No other room may be given initial door power unless **R2** is still satisfied.

### R2. Gatekeeper rooms and door power (control deadlock prevention)

- **Gatekeeper room**: A room R is a gatekeeper if **every** path from start to the exit goes through at least one door **into** R.
- **Rule**: For every gatekeeper room R whose doors are **not** initially powered:
  - At least one maintenance terminal that can power R’s doors must lie in a room **Q** such that:
    - Q is **adjacent** to R (so the terminal can control R), and
    - Q is **reachable from start without entering R** (e.g. Q is start room, or reachable via other doors that are powered or unlockable first).

**Implementation options** (choose one or combine):

- **(A) Exit / critical path first**: After placing exit and doors, compute “critical path” rooms (rooms through which every start→exit path goes). For each such room R with unpowered doors, ensure a maintenance terminal that can power R exists in an adjacent, non-gatekeeper (or earlier-reachable) room; or place such a terminal there.
- **(B) Power from adjacent reachable room**: When placing maintenance terminals, prefer (or require) that for every room R that has doors and is not the start room, either R’s doors are initially powered, or some adjacent room Q that is reachable without entering R has a maintenance terminal (so the player can power R from Q).
- **(C) Avoid gatekeeper unpowered rooms**: When choosing which room contains the exit (or which rooms get unpowered doors), reject layouts where the only path to the exit goes through a room whose doors are unpowered and whose only power control is inside that room.

**Current implementation**: A post-pass `EnsureSolvabilityDoorPower` runs **after** maintenance terminals are placed. It finds every gatekeeper room R with unpowered doors by computing “reachable from start without entering R” with **both** (1) door cells into R and (2) **all locked door cells** treated as impassable. It then checks whether any room adjacent to R that **has a maintenance terminal** is in that set. If not (e.g. the only adjacent reachable room is Corridor, which has no terminal), it sets `RoomDoorsPowered[R] = true` so the level is solvable.

### R3. Locked doors and keycards

- Keycard for room R is placed only in the **reachable set before** placing the doors that lock R (current behaviour).
- Do not create a **keycard cycle** (A’s keycard behind B’s door, B’s keycard behind A’s door) unless one of those doors is unlockable by other means first.

### R4. Hazards and controls

- Do not place a blocking hazard such that **every** path to its control (or to the item that fixes it) goes through that hazard.
- Prefer placing hazard controls in rooms that are **reachable before** the hazard (e.g. control in a room adjacent to the hazard room, reachable without passing the hazard).

### R5. Generators and batteries

- All generator cells must be in the **reachable set** (possibly after unlocking doors / powering doors / fixing hazards).
- Total batteries placed in reachable locations must be at least the sum of `BatteriesRequired` over all generators.

### R6. Maintenance terminal placement and adjacency

- Maintenance terminals only control **own room** and **adjacent rooms** (by current definition of “adjacent”).
- When adding new placement logic or new “control” types, preserve the rule: **for any room R that is gatekeeper and has unpowered doors, there must be a way to power R from a room reachable without entering R.**

### R7. Exit and start placement

- Exit cell must be **reachable** when all win conditions are satisfied (generators powered, hazards cleared, door power and keycards used as needed).
- Start cell must be in a room that has its doors powered at init (so the player can leave).

### R8. Prevent room disconnection

- When placing **blocking entities** (furniture, maintenance terminals, CCTV terminals, puzzle terminals, hazard controls, etc.) in a room, placement must **not disconnect** the room.
- After placing an entity at a cell, all **doorways** (room cells adjacent to corridor entries) in that room must still be **mutually reachable** via walkable room cells (i.e. all doorways must lie in a single connected component of walkable cells within the room).
- **Implementation**: Before placing an entity at a candidate cell, check that treating that cell as blocked (in addition to existing blocking entities in the room) still leaves all doorways in one connected component. If not, skip that candidate and try another.

---

## 6. Adding New Puzzle or Logic Elements

When adding new mechanics (e.g. new gate types, new controls, new objectives):

### 6.1 Identify dependencies

- What **state** does the new mechanic depend on (e.g. “room power”, “item”, “puzzle solved”)?
- **Where** can that state be changed (which cells / terminals / items)?
- Can that “control” be reached **before** the player needs to pass the gate that depends on it?

### 6.2 Check for deadlocks

- For any new **gate** (something that blocks movement or blocks win condition):
  - List the conditions required to pass it.
  - For each condition, list the **control points** (cells/entities that satisfy it).
  - Ensure at least one control point is **reachable without passing that gate** (or without passing another gate that itself depends on passing this gate).

### 6.3 Update invariants and rules

- Add an **invariant** (like I2–I5) that states “no deadlock for this new dependency”.
- Add **rules** (like R2–R6) that level generation must follow so the invariant holds.
- If the new mechanic is **optional** (e.g. bonus), it may be acceptable that it is sometimes unreachable; document that clearly.

### 6.4 Validation (recommended)

- Add a **solvability check** (e.g. in devtools or CI): given a dumped level (e.g. `map.txt`), compute reachability under “ideal” state (all keycards found, all doors powerable in valid order, etc.) and verify the exit is reachable and all win conditions can be met. Optionally, verify that **no** room is a gatekeeper with unpowered doors and no adjacent-reachable maintenance terminal.

---

## 7. Summary Checklist for Level Layout

Before considering a level layout complete, ensure:

- [ ] **I6**: Start room has doors powered at init.
- [ ] **I2**: Every gatekeeper room (only path to exit goes through it) either has doors powered at init or has a maintenance terminal that can power it in a room **adjacent** and **reachable without entering that room**.
- [ ] **I3**: Keycards are only behind doors that are unlockable in some order (no keycard cycles).
- [ ] **I4**: No hazard blocks the only path to its control or required item.
- [ ] **I5**: All generators and enough batteries are reachable.
- [ ] **I7**: Every named room remains a single connected component of walkable cells (no internal disconnection by blocking entities).
- [ ] **I1**: Exit is reachable after some valid sequence of actions (can be validated by reachability + dependency resolution).

This spec should be updated whenever new gates, controls, or objectives are added, so that the “control-dependency deadlock” class of bug cannot reoccur.
