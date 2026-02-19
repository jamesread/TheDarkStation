# Story 1.2: Rooms and Corridors

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a player,
I want the level to consist of named rooms connected by corridors,
so that the deck has clear spatial structure.

## Acceptance Criteria

1. **Given** level generation  
   **When** the level is created  
   **Then** each room has a name and a set of walkable cells  
   **And** corridors connect rooms  
   **And** each deck represents a functional layer (e.g. Habitation, Research, Power Distribution)

## Tasks / Subtasks

- [x] Task 1: Room naming and walkable cells (AC: #1)
  - [x] 1.1 Ensure each room cell has Name and Description set; room cells are walkable (Room=true)
  - [x] 1.2 Verify room names are distinct per room and come from deck functional layer (deck.RoomNamesForType)
  - [x] 1.3 Document or assert invariant: every room has at least one cell with a non-empty Name
- [x] Task 2: Corridors connect rooms (AC: #1)
  - [x] 2.1 Ensure corridors are carved between rooms (e.g. L-shaped or straight segments)
  - [x] 2.2 Corridor cells are walkable (Room=true) and named "Corridor" (or equivalent)
  - [x] 2.3 Verify connectivity: from any room cell, other rooms are reachable via walkable cells (no isolated rooms)
- [x] Task 3: Deck functional layer (AC: #1)
  - [x] 3.1 Deck identity drives room naming: deck.FunctionalType(level) → RoomNamesForType(ft) for thematic names
  - [x] 3.2 Final deck uses minimal layout (fewer rooms) per GDD; other decks use full BSP/room count
- [x] Task 4: Tests and validation (AC: #1)
  - [x] 4.1 Unit or integration tests: generated grid has named rooms, corridor cells, and connectivity
  - [x] 4.2 Run full test suite; ensure no regressions

## Dev Notes

- **Architecture:** Engine provides `pkg/engine/world`: Grid, Cell (Name, Description, Room bool). Game provides `pkg/game/generator`: BSPGenerator (bsp.go) carves rooms with MarkAsRoomWithName and connects them with corridors; `pkg/game/deck`: FunctionalType(level), RoomNamesForType(Type) for thematic room names. [Source: docs/architecture.md]
- **Existing code:** BSP generator in `pkg/game/generator/bsp.go`: carveRooms (MarkAsRoomWithName with room name/description per bspRoom), connectRooms (carveCorridorHorizontal/Vertical with "Corridor"/"ROOM_CORRIDOR"), createRooms uses deck.RoomNamesForType(ft). Engine Grid: MarkAsRoom, MarkAsRoomWithName; Cell: Name, Description, Room. [Source: pkg/game/generator/bsp.go, pkg/engine/world/grid.go, pkg/engine/world/cell.go]
- **Room vs corridor:** Named rooms get thematic names (e.g. "Abandoned Cryogenic Habitation Block"); corridor cells get Name "Corridor", Description "ROOM_CORRIDOR". Do not overwrite room names when carving corridors (bsp checks !cell.Room before MarkAsRoomWithName for corridor). [Source: pkg/game/generator/bsp.go]
- **Testing:** Use `*_test.go` alongside source. Run `go test ./...` or `make test`. Generator tests can assert grid invariants (e.g. all Room cells have Name set, "Corridor" cells exist, BFS from start reaches all rooms). [Source: docs/architecture.md]
- **Named rooms scope:** Named rooms (Name/Description per cell) are guaranteed for the **default** generator (BSP). LineWalkerGenerator uses MarkAsRoom only and does not set names; it is out of scope for this story. The game uses DefaultGenerator = BSP. [Source: pkg/game/generator/generator.go, bsp.go, line_walker.go]

### Project Structure Notes

- Generator: `pkg/game/generator/bsp.go`, `pkg/game/generator/generator.go`, `pkg/game/generator/line_walker.go`
- Engine grid/cell: `pkg/engine/world/grid.go`, `pkg/engine/world/cell.go`
- Deck and naming: `pkg/game/deck/deck.go` (FunctionalType, RoomNamesForType)
- Align with existing patterns: state in `pkg/game/state`, no new packages unless justified

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 1, Story 1.2]
- [Source: docs/architecture.md — Source Tree, Data Architecture]
- [Source: specs/gdd.md — Deck structure, functional layers]
- [Source: specs/level-layout-and-solvability.md — Layout invariants]

## Dev Agent Record

### Agent Model Used

(Set by dev-story agent)

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed — comprehensive developer guide created (create-story workflow).
- Tasks 1–3: Verified existing BSP implementation in pkg/game/generator/bsp.go: carveRooms (MarkAsRoomWithName with room name/description), connectRooms (L-shaped corridors, "Corridor"/"ROOM_CORRIDOR"), createRooms uses deck.RoomNamesForType(ft); deck.FunctionalType(level) and final-deck minimal layout (minSize=14, smaller grid). No code changes required.
- Task 4: Added pkg/game/generator/bsp_test.go: TestBSPGenerate_HasNamedRooms, TestBSPGenerate_HasCorridors, TestBSPGenerate_AllRoomsReachable, TestBSPGenerate_DeckFunctionalLayer, TestBSPGenerate_FinalDeckMinimalLayout. All generator tests pass.
- Code review (2026-02-01): Addressed 2 MEDIUM + 4 LOW findings. Seeded rand in tests for determinism; strengthened DeckFunctionalLayer test (room name must contain adjective or base from RoomNamesForType); added midDeckLevelForTest constant and package comment; added Dev Note on named-rooms scope (BSP default, LineWalker out of scope).

### File List

- pkg/game/generator/bsp_test.go

## Change Log

- 2026-02-01: Story completed. Verified Tasks 1–3 (room naming, corridors, deck functional layer) against BSP generator; added bsp_test.go for named rooms, corridors, connectivity, deck naming, and final-deck minimal layout. Status set to review.
- 2026-02-01: Code review fixes applied. Test determinism (rand.Seed), stronger DeckFunctionalLayer assertion, midDeckLevelForTest constant, package comment, Dev Note on BSP vs LineWalker. Status set to done.

## Senior Developer Review (AI)

- **Review date:** 2026-02-01
- **Outcome:** Changes Requested → All addressed
- **Action items:** All resolved (test determinism, DeckFunctionalLayer assertion, constant/comment, package comment, Dev Note)
