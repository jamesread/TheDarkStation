# Story 1.1: Grid and Movement

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a player,
I want to move through a grid of cells (rooms and corridors),
so that I can explore the deck.

## Acceptance Criteria

1. **Given** a generated level with walkable and blocking cells  
   **When** I press movement keys (N/S/E/W or arrows or vim-style)  
   **Then** my unit moves to an adjacent walkable cell  
   **And** walls and blocking entities (furniture, terminals, hazards, generators) prevent movement

## Tasks / Subtasks

- [x] Task 1: Movement input handling (AC: #1)
  - [x] 1.1 Map N/S/E/W and arrow keys to movement intents
  - [x] 1.2 Map vim-style keys (H/J/K/L or equivalent) to movement when nav style supports it
  - [x] 1.3 Pass movement intents to gameplay layer; update CurrentCell when move is allowed
- [x] Task 2: Walkable vs blocking (AC: #1)
  - [x] 2.1 Treat cells with Room=true as walkable; non-room (e.g. walls) as blocking
  - [x] 2.2 Block movement into cells that contain blocking entities (generator, furniture, CCTV terminal, puzzle terminal, maintenance terminal, hazard control, blocking hazard) per CanEnter semantics
  - [x] 2.3 Do not move when target cell is nil or out of grid bounds
- [x] Task 3: Tests and validation (AC: #1)
  - [x] 3.1 Unit tests for movement: valid move updates CurrentCell; blocked move does not
  - [x] 3.2 Tests for CanEnter / MoveCell with walls, blocking entities
  - [x] 3.3 Run full test suite; ensure no regressions

## Dev Notes

- **Architecture:** Engine provides `pkg/engine/world`: Grid, Cell (Room bool, North/East/South/West). Game provides `pkg/game/gameplay`: ProcessIntent (input.go), MoveCell / CanEnter (movement.go). Input flows: Ebiten input → engine tiered input → ActionMove* → ProcessIntent → MoveCell(g, neighborCell). [Source: docs/architecture.md, docs/source-tree-analysis.md]
- **Existing code:** Movement is already implemented in `pkg/game/gameplay/input.go` (ProcessIntent, MoveCell) and `pkg/game/gameplay/movement.go` (CanEnter, MoveCell). Task is to verify behaviour matches AC, add or adjust tests, and fix any gaps. Do not duplicate logic; extend or refactor only as needed.
- **Walkability:** Cell.Room indicates walkable room/corridor. CanEnter(g, cell, logReason) checks door power, locked door, generator, furniture, terminals, hazard control, blocking hazard; returns false for blocking entities. [Source: pkg/game/gameplay/movement.go]
- **Testing:** Use `*_test.go` alongside source. Run `go test ./...` or `make test`. [Source: docs/architecture.md]

### Project Structure Notes

- Movement: `pkg/game/gameplay/input.go`, `pkg/game/gameplay/movement.go`
- Engine grid/cell: `pkg/engine/world/grid.go`, `pkg/engine/world/cell.go`
- Game cell data / entities: `pkg/game/world/cell.go`, `pkg/game/entities/`
- Align with existing patterns: state in `pkg/game/state`, no new packages unless justified

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 1, Story 1.1]
- [Source: docs/architecture.md — Source Tree, Data Architecture]
- [Source: docs/source-tree-analysis.md — Critical Directories]
- [Source: specs/level-layout-and-solvability.md — Movement gates, win conditions]

## Dev Agent Record

### Agent Model Used

(Set by dev-story agent)

### Debug Log References

### Completion Notes List

- Task 1–2: Verified existing implementation. Engine tiered input (`pkg/engine/input/tiered.go`) maps N/S/E/W, arrows, and vim (h/j/k/l) to ActionMove*; Ebiten (`pkg/game/renderer/ebiten/input.go`) sends codes; `ProcessIntent`/`MoveCell` in `input.go`/`movement.go` update `CurrentCell` when move allowed. Nil/out-of-bounds handled (neighbor nil → `CanEnter` false).
- Task 3: Added/expanded `pkg/game/gameplay/movement_test.go`: `TestCanEnter_NilCell`, `TestCanEnter_NonRoomCell`, `TestCanEnter_EmptyRoomCell`, `TestCanEnter_GeneratorBlocksMovement`, `TestProcessIntent_ValidMoveUpdatesCurrentCell`, `TestProcessIntent_BlockedMoveDoesNotUpdateCurrentCell`, `TestProcessIntent_AllFourDirections` (table-driven subtests). All gameplay tests pass. Full `go test ./...` still fails in other packages (pre-existing gotext lint); no regressions in gameplay/setup.
- Code review (2026-02-01): Addressed 2 HIGH + 2 MEDIUM findings. Nil guard for `g.CurrentCell` in `input.go` movement cases; `TestProcessIntent_NilCurrentCellNoPanic` and `TestCanEnter_FurnitureBlocksMovement` added; story File List updated (input.go, helpers.go); brief comments for test grid sizes. All fixes applied, tests pass.

### File List

- pkg/game/gameplay/movement_test.go
- pkg/game/gameplay/input.go
- pkg/game/setup/helpers.go

## Change Log

- 2026-02-01: Story completed. Verified Tasks 1–2 (movement input and walkability) against existing code; added unit tests in movement_test.go for CanEnter (nil, non-room, empty room, generator blocking) and ProcessIntent (valid move, blocked move, all four directions). Status set to review.
- 2026-02-01: Code review fixes applied. Nil CurrentCell guard in input.go; TestProcessIntent_NilCurrentCellNoPanic, TestCanEnter_FurnitureBlocksMovement; File List updated; test comments. Status set to done.

## Senior Developer Review (AI)

- **Review date:** 2026-02-01
- **Outcome:** Changes Requested → All addressed
- **Action items:** All resolved (nil guard, nil CurrentCell test, furniture blocking test, File List, comments)
