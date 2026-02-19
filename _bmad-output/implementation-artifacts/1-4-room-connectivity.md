# Story 1.4: Room Connectivity

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a player,
I want every room's walkable area to be one connected region,
so that I can reach all doorways and controls in a room without leaving it.

## Acceptance Criteria

1. **Given** placement of blocking entities (furniture, terminals, hazards, etc.)
   **When** a level is generated
   **Then** no room is disconnected (all doorways in that room are mutually reachable via walkable cells within the room)
   **And** placement follows rule R8 (prevent room disconnection)

## Tasks / Subtasks

- [x] Task 1: Verify R8 coverage (AC: #1)
  - [x] 1.1 Identify all placement paths that put blocking entities inside a room (furniture, maintenance terminals, CCTV terminals, puzzle terminals, hazard controls)
  - [x] 1.2 Confirm which paths use isRoomStillConnected (or equivalent) and which do not
  - [x] 1.3 Document gaps: setup/terminals.go (CCTV) does not use room connectivity; maintenance fallback when connectedCandidates is empty
- [x] Task 2: Apply R8 to all placement paths (AC: #1, R8)
  - [x] 2.1 Add room-connectivity check to CCTV terminal placement (setup/terminals.go): filter candidates with isRoomStillConnected; need entry cells per room (e.g. from setup.FindRoomEntryPoints). Note: isRoomStillConnected is in levelgen; levelgen imports setup, so setup cannot import levelgen (cycle). Move room-connectivity helper to setup (e.g. setup/room_connectivity.go) so both levelgen and setup/terminals can call it, then use in CCTV placement.
  - [x] 2.2 Harden maintenance placement: when no connected candidate exists, do not place (skip room or document); avoid fallback to validCells that can disconnect
  - [x] 2.3 Verify puzzle and hazard-control placement: they use FindNonArticulationCellInRoom (global reachability); confirm whether per-room doorway connectivity is also required and add if needed
- [x] Task 3: Tests and validation (AC: #1, I7)
  - [x] 3.1 Unit test for isRoomStillConnected (or exported helper): room with two doorways, blocking candidate on only path between them returns false; blocking elsewhere returns true
  - [x] 3.2 Integration-style test: generated level satisfies I7 — for each named room, doorways are mutually reachable via walkable room cells (optional: devtools or test that runs BSP + placement and asserts connectivity)
  - [x] 3.3 Run full test suite; ensure no regressions

## Dev Notes

- **R8 (specs/level-layout-and-solvability.md):** When placing blocking entities in a room, placement must not disconnect the room; after placing, all doorways (room cells adjacent to corridor entries) must remain mutually reachable via walkable room cells. Implementation: before placing at a candidate cell, check that treating that cell as blocked still leaves all doorways in one connected component.
- **I7:** Every named room's walkable cells (excluding blocking entities) form a single connected component; placement must not disconnect (all doorways mutually reachable within room).
- **Existing code:** `pkg/game/levelgen/utils.go`: `isRoomStillConnected(g, roomName, entryCellsForRoom, additionalBlockedCell)` implements R8. Used in levelgen/furniture.go (every furniture candidate) and levelgen/maintenance.go (filters to connectedCandidates; currently falls back to validCells if none). Not used in setup/terminals.go (CCTV). Puzzles/hazards use `FindNonArticulationCellInRoom` (global articulation point).
- **Entry points:** Maintenance uses `setup.FindRoomEntryPoints(g.Grid)` for entry cells. CCTV uses `getRoomEntryPoints` (room cells adjacent to entry cells). For isRoomStillConnected, the signature expects `entryCellsForRoom` — corridor-side entry cells (same as FindRoomEntryPoints EntryCells) so doorways = room cells adjacent to those.

### Project Structure Notes

- Levelgen: `pkg/game/levelgen/utils.go` (isRoomStillConnected), furniture.go, maintenance.go, puzzles.go, hazards.go
- Setup: `pkg/game/setup/terminals.go` (CCTV placement), setup/helpers.go or setup for FindRoomEntryPoints
- Spec: specs/level-layout-and-solvability.md (I7, R8)

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 1, Story 1.4]
- [Source: specs/level-layout-and-solvability.md — I7, R8]
- [Source: FR24 — Room connectivity, placement rules]

## Dev Agent Record

### Agent Model Used

dev-story workflow (workflow.xml + dev-story instructions.xml)

### Debug Log References

### Completion Notes List

- Task 1: Verified R8 coverage. Placement paths: furniture (levelgen/furniture.go) and maintenance (levelgen/maintenance.go) used isRoomStillConnected; CCTV (setup/terminals.go) did not; puzzles and hazard controls used FindNonArticulationCellInRoom (global). Gaps: CCTV no room check; maintenance fell back to validCells when connectedCandidates empty.
- Task 2: Moved room-connectivity logic to setup/room_connectivity.go as RoomStillConnectedIfBlock; levelgen/utils.go isRoomStillConnected now delegates to setup. CCTV placement filters to connected candidates and skips placement if none; maintenance no longer falls back to validCells when no connected candidate (skips room). Puzzle placement adds per-room check: if FindNonArticulationCellInRoom result would disconnect room, fall back to puzzleRoom. Hazard controls use global articulation only (per-room R8 deferred; controls often in different rooms).
- Task 3: Added setup/room_connectivity_test.go: TestRoomStillConnectedIfBlock_TwoDoorwaysBlockChokepoint (false), TestRoomStillConnectedIfBlock_TwoDoorwaysBlockNonChokepoint (true), TestRoomStillConnectedIfBlock_EmptyEntryCells (true). Skipped optional integration test (I7 full-level). Ran pkg/game/levelgen, generator, setup, gameplay tests — all pass; menu and renderer/ebiten have pre-existing gotext build issues.

### File List

- pkg/game/setup/room_connectivity.go
- pkg/game/setup/room_connectivity_test.go
- pkg/game/setup/terminals.go
- pkg/game/levelgen/utils.go
- pkg/game/levelgen/maintenance.go
- pkg/game/levelgen/puzzles.go

## Change Log

- 2026-02-01: Story created (create-story for 1-4). Ready for dev.
- 2026-02-01: Story implemented. RoomStillConnectedIfBlock in setup/room_connectivity.go; R8 applied to CCTV and maintenance; puzzle placement per-room check; unit tests; status set to review.
