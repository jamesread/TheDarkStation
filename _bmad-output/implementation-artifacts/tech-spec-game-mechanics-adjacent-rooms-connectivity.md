---
title: 'Game Mechanics — Adjacent Rooms and Room Connectivity'
slug: 'game-mechanics-adjacent-rooms-connectivity'
created: '2026-02-01'
status: 'Completed'
stepsCompleted: [1, 2, 3, 4]
tech_stack: ['Go', 'pkg/engine/world', 'pkg/game/setup', 'pkg/game/state', 'pkg/game/generator', 'pkg/game/menu', 'pkg/game/world']
files_to_modify: ['pkg/game/setup/helpers.go', 'pkg/game/setup/helpers_test.go']
code_patterns: ['Grid Build/BuildAllCellConnections set neighbour links; BSP carve then connect; GetAdjacentRoomNames single-pass ForEachCell (post-Task 2)']
test_patterns: ['No existing _test.go; add unit tests for GetAdjacentRoomNames and key call sites']
---

# Tech-Spec: Game Mechanics — Adjacent Rooms and Room Connectivity

**Created:** 2026-02-01

## Overview

### Problem Statement

The game (The Dark Station) is mostly functional, but **calculation of adjacent rooms is quite broken**. This affects maintenance menus (e.g. "power terminals in adjacent rooms", "room power for adjacent rooms"), solvability checks (gatekeeper rooms and reachable adjacent rooms), and any feature that relies on "rooms directly adjacent to this room". A clear, implementation-ready spec is needed to define correct adjacency semantics and fix the implementation.

### Solution

Produce a technical specification that (1) defines what "adjacent room" means for this codebase (corridor-mediated vs direct room-to-room, grid topology), (2) documents the current contract and failure modes, (3) specifies implementation tasks and acceptance criteria so a developer can fix `GetAdjacentRoomNames` and any dependent behaviour without guessing.

### Scope

**In Scope:**
- Definition of "adjacent room" (semantics and edge cases: corridors, direct boundaries, naming).
- `GetAdjacentRoomNames` and any shared helpers used for room adjacency / connectivity.
- Call sites that depend on adjacency: maintenance menu (adjacent rooms power, restore nearby terminals), solvability (gatekeeper and reachable adjacent rooms).
- Data and API contract: grid, cell neighbour links, room names (including `"Corridor"`), and how they are built (e.g. BSP carving).

**Out of Scope:**
- Other game mechanics not tied to adjacency (power decay, terminal copy, item naming, etc.).
- Renaming the repo directory from TheDarkCastle (legacy); project name is The Dark Station / TheDarkStation.
- Converting or archiving `specs/plan-deck-generation-and-navigation.md` (handled separately).

## Context for Development

### First Principles: Adjacency Definition

**Invariant (source of truth):**
Room B is *adjacent* to room A iff there exists at least one cell C such that `C.Name == A` and C has a N/S/E/W neighbour N with `N.Name == B` and `B != A`.
"Adjacent rooms to A" = the set of all such B (plus A itself if the API is "this room + adjacent").

**Implications:**
- No separate "via corridor" vs "direct" in the *definition*; both are "room A cell next to room B cell".
- Correctness depends on: (1) neighbour links set correctly after generation, (2) room names set consistently (including `"Corridor"` where used).
- Implementation may use one or two passes; the spec requires only that the result satisfy the invariant above.

### Failure Modes and Prevention

| Component | How it can fail | Prevention / mitigation |
|-----------|------------------|---------------------------|
| **Neighbour links** | Nil or wrong pointer after carve. Adjacent room missed or wrong room included. | Ensure every room cell's N/S/E/W is set from grid after generation. Spec: neighbour links must match grid topology. Validate or fix in setup. |
| **Room naming** | Corridor cells not named `"Corridor"`; named rooms with empty Name. Corridor pass finds nothing; rooms conflated or missed. | Generator contract: corridor cells get Name `"Corridor"`; named rooms get non-empty Name. Document in spec; tests assert naming after generation. |
| **Single-pass implementation** | Depends on links and names; if wrong, result wrong. | Implementation must satisfy first-principles invariant; tests on known grids assert result set. |
| **API contract** | roomName not in grid; empty string; nil grid. Empty/wrong result or panic. | Spec: roomName non-empty and present; grid non-nil. Defensive return (e.g. empty slice) rather than panic. |
| **Call sites** | Assume sorted, unique, or "roomName included". | Spec: returns sorted, unique names including roomName. Tests for GetAdjacentRoomNames and key call sites lock behaviour. |

### Codebase Patterns

- **Engine vs game**: `pkg/engine/world` provides `Cell` (with `North`/`South`/`East`/`West`), `Grid`, and `GetCell(row,col)`. Game code in `pkg/game` uses these; room adjacency is implemented in `pkg/game/setup/helpers.go` (`GetAdjacentRoomNames`).
- **Grid build**: `Grid.Build(rows, cols)` allocates all cells and sets `roomDir[name] = c` with `name = fmt.Sprintf("%v:%v", row, col)` (position-based keys only). `MarkAsRoomWithName` only sets `cell.Room`, `cell.Name`, `cell.Description`; it does **not** update `roomDir`. So after BSP carving, `roomDir` still has only `"0:0"`, `"0:1"`, …; `GetCellByName(roomName)` for a room name like `"Cargo Bay"` returns **nil**. No game code currently calls `GetCellByName(roomName)`; this is latent tech debt.
- **Neighbour links**: `BuildAllCellConnections()` runs after BSP carve and connect; it sets every cell's N/S/E/W via `GetCellRelative` (grid topology). So room cells have correct neighbour pointers to adjacent cells (room, corridor, or wall). Links are consistent with grid geometry.
- **Room naming (BSP)**: `carveRooms` gives all cells in a room the same `node.room.name` (e.g. `"Emergency Section"`). `createRooms` builds names as `adjective + " " + baseName` from thematic lists; **names are not guaranteed unique** (two rooms can share the same adjective+base). That can make two distinct rooms appear as one for adjacency and for "current room" identity.
- **Corridor carving**: `carveCorridorHorizontal` / `carveCorridorVertical` call `MarkAsRoomWithName(row, col, "Corridor", ...)` only when `!cell.Room`, so room cells are not overwritten. Corridor cells end up with `Room == true`, `Name == "Corridor"`.
- **Adjacency today**: Two passes in `GetAdjacentRoomNames`: (1) corridor-centric — cells where `cell.Room && cell.Name == "Corridor"`, collect neighbours' room names when a neighbour has `Name == roomName`; (2) direct room-to-room — cells where `cell.Name == roomName`, add N/S/E/W neighbour names (excluding Corridor and roomName). Result is sorted, unique, and includes `roomName`. Correctness depends on neighbour links and consistent naming; duplicate room names would merge distinct rooms.
- **Usage**: Maintenance menu `buildAdjacentRoomsPowerItems` and "restore nearby terminals" use `setup.GetAdjacentRoomNames(g.Grid, currentRoomName)`. Solvability `EnsureSolvabilityDoorPower` uses the same for gatekeeper checks (reachable adjacent rooms with terminals).

### Files to Reference

| File | Purpose |
| ---- | ------- |
| `pkg/game/setup/helpers.go` | `GetAdjacentRoomNames`, `getReachableCells`, `collectReachableRooms` |
| `pkg/game/setup/solvability.go` | `EnsureSolvabilityDoorPower`, `GetAdjacentRoomNames(roomName)` for gatekeeper adjacent-set |
| `pkg/game/menu/maintenance.go` | `buildAdjacentRoomsPowerItems`, RestoreNearbyTerminals (adjacentNames), AllRoomsPowerMenuHandler |
| `pkg/engine/world/cell.go` | `Cell` struct, `GetNeighbors()`, `SetNeighbor` |
| `pkg/engine/world/grid.go` | `Grid`, `Build`, `BuildAllCellConnections`, `GetCell`, `GetCellRelative`, `ForEachCell`, `MarkAsRoomWithName`; `roomDir` only has position keys after Build |
| `pkg/game/generator/bsp.go` | `carveRooms` (MarkAsRoomWithName per room), `connectRooms` → `carveCorridor*` (Corridor only if !cell.Room), `BuildAllCellConnections()` after carve |

### Technical Decisions

- Project name: **The Dark Station** (or **TheDarkStation** in code). "TheDarkCastle" is legacy (directory name only).
- GDD (`specs/gdd.md`) is the broad narrative/systems reference; this spec focuses only on adjacent-rooms mechanics.
- `specs/plan-deck-generation-and-navigation.md` is an implementation plan that has been implemented; it may be archived or converted to a reference spec separately.
- **roomDir**: Grid's `roomDir` is populated in `Build()` with position keys only; it is not updated when `MarkAsRoomWithName` changes `cell.Name`. Do not rely on `GetCellByName(roomName)` for room names; use `ForEachCell` + `cell.Name` (or fix roomDir in a separate change).
- **Room name uniqueness**: BSP room names (`adjective + base`) are not guaranteed unique. Fixing adjacency may require guaranteeing unique room names (e.g. suffix by BSP node or counter) so "adjacent to room A" is unambiguous; document choice in implementation plan.
- **Corridor in adjacent-rooms UI:** Exclude the name `Corridor` from the result of `GetAdjacentRoomNames` so the adjacent-rooms power menu shows only named rooms (not corridor cells). Document this in a comment in `GetAdjacentRoomNames`.

## Implementation Plan

### Tasks

- [x] **Task 1: Harden GetAdjacentRoomNames API (defensive)**
  - **File:** `pkg/game/setup/helpers.go`
  - **Action:** At the start of `GetAdjacentRoomNames`, if `grid` is nil, return an empty slice. If `roomName` is empty, return an empty slice. Do not panic. Return `nil` for these defensive cases (not `[]string{}`); document in a comment that callers must treat nil as empty (e.g. `len(result) == 0`). Preserve existing behaviour for all valid inputs.
  - **Notes:** Callers (maintenance menu, solvability) assume non-nil grid and non-empty room name in normal flow; defensive checks protect against misuse and future refactors.

- [x] **Task 2: Refactor GetAdjacentRoomNames to satisfy first-principles invariant**
  - **File:** `pkg/game/setup/helpers.go`
  - **Action:** Implement adjacency using the invariant: room B is adjacent to room A iff some cell C with `C.Name == A` has a N/S/E/W neighbour N with `N.Name == B` and `B != A`. Single pass: `grid.ForEachCell`; for each cell where `cell.Room && cell.Name == roomName`, iterate N/S/E/W neighbours; for each neighbour `n` with `n != nil && n.Room && n.Name != "" && n.Name != roomName`, add `n.Name` to the result set. Exclude the name `"Corridor"` from the result set (per Technical Decisions: adjacent-rooms UI shows named rooms only). Document this in a comment in `GetAdjacentRoomNames`. If no cell has `Name == roomName`, return empty slice (do not add roomName to the set). Otherwise add `roomName` to the set, sort, return. Remove or consolidate the existing two-pass (corridor-centric + direct) logic so the single pass suffices. Add a short comment referencing the first-principles invariant.
  - **Notes:** One pass is sufficient; corridor-mediated and direct boundaries are both "room A cell next to room B cell". If Corridor is excluded, document in a comment.

- [x] **Task 3: Add unit tests for GetAdjacentRoomNames**
  - **File:** `pkg/game/setup/helpers_test.go` (new)
  - **Action:** Add tests: (1) `GetAdjacentRoomNames(nil, "Any")` returns empty slice, no panic. (2) `GetAdjacentRoomNames(grid, "")` returns empty slice, no panic — use any grid (e.g. from `world.NewGrid(2,2)`); the empty string is the roomName under test. (3) Minimal grid: build a small grid, mark two rooms A and B sharing a wall (one cell of A has neighbour in B); call `GetAdjacentRoomNames(grid, "A")`; assert result contains A and B, sorted and unique. (4) Optional: grid with room A, corridor C, room B in a line; assert adjacency of A includes B and optionally C per product choice. Use `world.NewGrid`, `MarkAsRoomWithName`, and `BuildAllCellConnections()` to construct test grids.
  - **Notes:** No existing `*_test.go` in repo; create `helpers_test.go` in same package. Tests import the setup package and the same world package used by production (e.g. `darkstation/pkg/engine/world`) for `NewGrid`, `MarkAsRoomWithName`, `BuildAllCellConnections`. Tests lock API contract and defensive behaviour. AC5 and AC6 are verified by manual or integration test only; they are not covered by unit tests in helpers_test.go.

- [ ] **Task 4 (optional / follow-up): Ensure BSP room names unique**
  - **File:** `pkg/game/generator/bsp.go`
  - **Action:** If duplicate BSP room names are identified as a cause of wrong adjacency, ensure `createRooms` assigns unique names (e.g. append room index or node id). Document in BSP or in this spec.
  - **Notes:** Defer to a separate change if scope is limited to fixing `GetAdjacentRoomNames` and tests. Include in this implementation if product requires unambiguous "adjacent to room A" when multiple rooms could share a name.

### Acceptance Criteria

- [ ] **AC1:** Given a valid grid and a room name that appears on at least one cell, when `GetAdjacentRoomNames(grid, roomName)` is called, then the result includes `roomName` and every room name B such that some cell with `Name == roomName` has a N/S/E/W neighbour with `Name == B` (B ≠ roomName), with no duplicates and sorted lexicographically.

- [ ] **AC2:** Given a nil grid, when `GetAdjacentRoomNames(nil, "Some Room")` is called, then the function returns an empty slice and does not panic.

- [ ] **AC3:** Given a valid grid and an empty `roomName`, when `GetAdjacentRoomNames(grid, "")` is called, then the function returns an empty slice and does not panic.

- [ ] **AC4:** Given a grid where room A has at least one cell adjacent (N/S/E/W) to a cell of room B, when `GetAdjacentRoomNames(grid, "A")` is called, then the result includes "B" (and "A"). If A is also adjacent to a corridor, result includes "Corridor" iff the implementation does not exclude corridor names (see Task 2 for product choice).

- [ ] **AC5:** Given the player is at a maintenance terminal in room R, when the "adjacent rooms power" menu is opened, then the list of rooms shown matches the set returned by `GetAdjacentRoomNames(g.Grid, R)` (same room names; toggles apply to the correct rooms).

- [ ] **AC6:** Given a gatekeeper room R with no power and at least one adjacent room that has a maintenance terminal and is reachable without entering R, when solvability runs, then the gatekeeper check uses the same adjacency; R's doors are not incorrectly powered (no deadlock); the player can power R from the adjacent terminal.

- [ ] **AC7:** Given a valid grid and a non-empty `roomName` that does not appear on any cell, when `GetAdjacentRoomNames(grid, roomName)` is called, then the function returns an empty slice.

## Additional Context

### Dependencies

- Grid must have correct N/S/E/W links on cells after level generation (BSP or other). `BuildAllCellConnections()` is called after carve in BSP; no change required for this spec.
- Room names must be set consistently: corridors as `"Corridor"`, named rooms non-empty. No external libraries or services; dependency is on `pkg/engine/world` and existing game state.

### Testing Strategy

- **Unit tests:** Add `pkg/game/setup/helpers_test.go` with tests for `GetAdjacentRoomNames`: nil grid, empty roomName, minimal two-room grid (shared wall), and optionally corridor-between-rooms grid. Use `world.NewGrid`, `MarkAsRoomWithName`, `BuildAllCellConnections()` to build test grids.
- **Integration (optional):** Generate a level with BSP, call `GetAdjacentRoomNames` for the start cell's room name; assert result is non-empty and contains the start room name.
- **Manual:** Play to a maintenance terminal, open "adjacent rooms power" and "restore nearby terminals"; confirm the list matches expectations and toggles affect the correct rooms.

### Notes

- **High-risk / follow-up:** Duplicate BSP room names can make two distinct rooms appear as one for adjacency; consider ensuring unique room names in BSP in a separate change (Task 4 optional).
- **Known limitation:** `Grid.roomDir` is not updated when `MarkAsRoomWithName` is used; `GetCellByName(roomName)` returns nil for room names. Out of scope for this spec; document or fix in a future change.
- **Investigation (Step 2):** Neighbour links are set correctly by `BuildAllCellConnections()` after BSP carve. Likely failure modes: (1) non-unique room names merging distinct rooms, (2) roomDir not updated, (3) edge cases (empty roomName, nil grid) not defended. Tests must be added.
- **getReachableCells / collectReachableRooms:** These use cell N/S/E/W for BFS and do not call `GetAdjacentRoomNames`; both rely on the same neighbour links. No change required for this spec.

## Review Notes

- Adversarial code review completed (Quick Dev step-05).
- Findings: 10 total (all Low). 8 addressed via auto-fix, 2 skipped (F7 undecided, F10 no change required).
- Resolution approach: Auto-fix (Option 2).
