# Story 1.3: Start Room and Exit

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a player,
I want a designated start cell and an exit cell,
so that I can begin and complete the deck.

## Acceptance Criteria

1. **Given** level setup  
   **When** the level is ready  
   **Then** I start at the start cell in the start room  
   **And** the start room's doors are powered at init so I can leave (FR23)  
   **And** an exit cell exists and is reachable when win conditions are met

## Tasks / Subtasks

- [x] Task 1: Start cell and start room (AC: #1)
  - [x] 1.1 Ensure generator sets start cell (Grid.SetStartCellAt / StartCell); player begins at start cell (CurrentCell = Grid.StartCell() after setup)
  - [x] 1.2 Start cell is in a room (StartCell.Room == true and StartCell.Name set)
  - [x] 1.3 Document or assert: start cell is walkable and valid
- [x] Task 2: Start room doors powered at init (AC: #1, FR23)
  - [x] 2.1 InitRoomPower sets RoomDoorsPowered[startRoomName] = true for the start cell's room
  - [x] 2.2 SetupLevel calls InitRoomPower after grid and room structure exist; order preserved (setup/setup.go, setup/roompower.go)
  - [x] 2.3 Player can leave start room (CanEnter adjacent cells when doors powered)
- [x] Task 3: Exit cell exists and reachable (AC: #1)
  - [x] 3.1 Generator sets exit cell (Grid.SetExitCellAt / ExitCell); cell marked ExitCell = true
  - [x] 3.2 Exit cell is reachable from start via walkable path (solvability / reachability)
  - [x] 3.3 CanEnter(exitCell) requires AllGeneratorsPowered and AllHazardsCleared (movement.go); exit exists and is reachable when win conditions met
- [x] Task 4: Tests and validation (AC: #1)
  - [x] 4.1 Unit or integration tests: start cell set and in room; start room doors powered at init; exit cell set and reachable
  - [x] 4.2 Run full test suite; ensure no regressions

## Dev Notes

- **Architecture:** Engine provides `pkg/engine/world`: Grid (StartCell, ExitCell, SetStartCellAt, SetExitCellAt), Cell (Room, Name, ExitCell). Game provides `pkg/game/generator`: BSP sets start/exit in Generate(); `pkg/game/setup`: SetupLevel, InitRoomPower (start room doors powered); `pkg/game/gameplay`: BuildGame/SetupLevel then MoveCell(g, g.Grid.StartCell()); state.ResetLevel sets g.CurrentCell = g.Grid.StartCell(). [Source: docs/architecture.md]
- **Existing code:** BSP in `pkg/game/generator/bsp.go`: SetStartCellAt(startRow, startCol), SetExitCellAt(exitCell), collectRooms/startRoom; fallback center. Setup in `pkg/game/setup/setup.go`: SetupLevel calls InitRoomPower(g); roompower.go: InitRoomPower sets RoomDoorsPowered[startCell.Name] = true. Lifecycle in `pkg/game/gameplay/lifecycle.go`: after SetupLevel, MoveCell(g, g.Grid.StartCell()). state.ResetLevel: g.CurrentCell = g.Grid.StartCell() after SetupLevel. [Source: bsp.go, setup.go, roompower.go, lifecycle.go, state.go]
- **Exit win condition:** CanEnter(g, r, logReason) in movement.go: if r.ExitCell then requires AllGeneratorsPowered() and AllHazardsCleared(); otherwise cannot enter. Exit is reachable when win conditions are met. [Source: pkg/game/gameplay/movement.go]
- **Testing:** Use `*_test.go` alongside source. Run `go test ./...` or `make test`. Setup/generator tests can assert StartCell/ExitCell set, InitRoomPower sets start room powered. [Source: docs/architecture.md]

### Project Structure Notes

- Generator: `pkg/game/generator/bsp.go` (start/exit placement)
- Setup: `pkg/game/setup/setup.go`, `pkg/game/setup/roompower.go`
- Gameplay: `pkg/game/gameplay/lifecycle.go` (BuildGame, MoveCell to start)
- State: `pkg/game/state/state.go` (ResetLevel, CurrentCell = StartCell)
- Movement: `pkg/game/gameplay/movement.go` (CanEnter exit cell win conditions)
- Align with existing patterns: no new packages unless justified

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 1, Story 1.3]
- [Source: docs/architecture.md — Source Tree, Data Architecture]
- [Source: specs/gdd.md — FR23 start room doors powered]
- [Source: specs/level-layout-and-solvability.md — Solvability, reachability]

## Dev Agent Record

### Agent Model Used

(Set by dev-story agent)

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed — comprehensive developer guide created (create-story workflow).
- Tasks 1–3: Verified existing implementation. BSP (bsp.go) sets StartCell/ExitCell in Generate(); SetupLevel calls InitRoomPower (roompower.go) so start room doors powered; lifecycle MoveCell(g, g.Grid.StartCell()) and state.ResetLevel sets CurrentCell = StartCell(); movement.go CanEnter(exitCell) requires AllGeneratorsPowered and AllHazardsCleared. No production code changes.
- Task 4: Added pkg/game/generator/bsp_test.go TestBSPGenerate_StartAndExitSet (start/exit non-nil, start in room with name, exit marked); pkg/game/setup/roompower_test.go TestInitRoomPower_StartRoomDoorsPowered. Exit reachable covered by existing TestBSPGenerate_AllRoomsReachable. All tests pass.
- Code review (2026-02-01): Addressed 1 MEDIUM + 3 LOW findings. InitRoomPower nil Grid guard (roompower.go); roompower_test.go package comment and TestInitRoomPower_NilGridNoPanic; TestBSPGenerate_StartAndExitSet exit.Room assertion. Status set to done.

### File List

- pkg/game/generator/bsp_test.go
- pkg/game/setup/roompower_test.go
- pkg/game/setup/roompower.go

## Change Log

- 2026-02-01: Story completed. Verified Tasks 1–3 (start cell, start room doors powered, exit cell reachable) against BSP/setup/lifecycle/state/movement; added bsp_test.go TestBSPGenerate_StartAndExitSet and roompower_test.go TestInitRoomPower_StartRoomDoorsPowered. Status set to review.
- 2026-02-01: Code review fixes applied. InitRoomPower nil Grid guard; roompower_test package comment and NilGridNoPanic test; bsp_test exit.Room assertion. Status set to done.

## Senior Developer Review (AI)

- **Review date:** 2026-02-01
- **Outcome:** Changes Requested → All addressed
- **Action items:** All resolved (nil Grid guard, package comment, nil-grid test, exit walkable assertion)
