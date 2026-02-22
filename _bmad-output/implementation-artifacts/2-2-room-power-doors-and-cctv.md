# Story 2.2: Room Power (Doors and CCTV)

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a player,
I want to power doors and CCTV per room via maintenance terminals,
so that I can open doors and use CCTV/hazard controls in that room.

## Acceptance Criteria

1. **Given** `RoomDoorsPowered` and `RoomCCTVPowered` per room
   **When** I use a maintenance terminal (own room or adjacent room)
   **Then** I can toggle doors and CCTV for those rooms
   **And** toggling doors ON with consumption > supply triggers `ShortOutIfOverload(protectedRoomName)` which auto-unpowers other rooms' doors/CCTV in deterministic order until consumption <= supply (the toggled room stays protected)
   **And** toggling doors/CCTV OFF recalculates consumption immediately
   **And** the maintenance menu shows per-room power supply, consumption, available power, and device list

2. **Given** level setup
   **When** `InitRoomPower(g)` runs
   **Then** all rooms have `RoomDoorsPowered[R] = false` and `RoomCCTVPowered[R] = false`
   **And** start room doors are powered (`RoomDoorsPowered[startRoomName] = true`) so the player can leave (FR23)
   **And** lights default to on (`RoomLightsPowered[R] = true` for all rooms)

3. **Given** level setup after maintenance terminal placement
   **When** `EnsureSolvabilityDoorPower(g)` runs
   **Then** for every gatekeeper room R (every path from start to exit goes through R) where R's doors are unpowered and no adjacent room with a maintenance terminal is reachable from start without entering R, R's doors are initially powered
   **And** locked door cells are treated as impassable when computing reachability

4. **Given** the full setup pipeline
   **When** a level is built (`gameplay.SetupLevel`)
   **Then** the order is: `InitRoomPower` → placement (hazards, furniture, puzzles, maintenance terminals) → `EnsureSolvabilityDoorPower` → `InitMaintenanceTerminalPower` → player at start cell (FR25)

## Tasks / Subtasks

- [x] Task 1: Verify InitRoomPower and room power defaults (AC: #2)
  - [x] 1.1 Add tests for `InitRoomPower`: all rooms unpowered by default, start room doors powered, lights default on; verify with multi-room grid in `pkg/game/setup/roompower_test.go`
  - [x] 1.2 Add test: `InitRoomPower` with grid containing rooms with different names — verify each room is initialized correctly
  - [x] 1.3 Add test: `InitRoomPower` idempotent — calling twice produces same result

- [x] Task 2: Verify EnsureSolvabilityDoorPower (AC: #3)
  - [x] 2.1 Add test: gatekeeper room with no adjacent terminal reachable from start → doors powered initially in `pkg/game/setup/solvability_test.go`
  - [x] 2.2 Add test: non-gatekeeper room (exit reachable without entering R) → doors NOT powered
  - [x] 2.3 Add test: gatekeeper room with adjacent terminal reachable from start → doors NOT powered (terminal can handle it)
  - [x] 2.4 Add test: locked door cells treated as impassable in reachability computation

- [x] Task 3: Verify maintenance menu toggle behavior (AC: #1)
  - [x] 3.1 Add tests for `RoomPowerToggleMenuItem` toggling doors ON/OFF — verify `RoomDoorsPowered` state changes in `pkg/game/menu/maintenance_test.go`
  - [x] 3.2 Add tests for CCTV toggle — verify `RoomCCTVPowered` state changes
  - [x] 3.3 Add test: toggling doors ON triggers `ShortOutIfOverload`; verify protected room stays on and other room is unpowered
  - [x] 3.4 Add test: toggling doors OFF recalculates consumption correctly
  - [x] 3.5 Add test: toggle only possible when room's maintenance terminal is powered (`IsSelectable` returns false when unpowered)

- [x] Task 4: Verify CanEnter door power check (AC: #1)
  - [x] 4.1 Add test: `CanEnter` returns false when door cell's room has `RoomDoorsPowered = false` in `pkg/game/gameplay/movement_test.go`
  - [x] 4.2 Add test: `CanEnter` returns true when door cell's room has `RoomDoorsPowered = true`
  - [x] 4.3 Add test: CCTV terminal interaction blocked when `RoomCCTVPowered = false`, allowed when true in `pkg/game/gameplay/interactions_test.go`
  - [x] 4.4 Add test: hazard control interaction blocked when `RoomCCTVPowered = false`

- [x] Task 5: Verify setup order matches FR25 (AC: #4)
  - [x] 5.1 Add integration test: `BuildGame` produces a game where start room doors are powered, start room maintenance terminal(s) are powered, and gatekeeper rooms are solvable in `pkg/game/gameplay/lifecycle_test.go`
  - [x] 5.2 Add test: `ResetLevel` re-initializes room power correctly (start room powered, others not)
  - [x] 5.3 Add test: room power maps preserved across `SaveCurrentDeckState`/`LoadDeckState`

- [x] Task 6: Run full test suite and fix regressions (AC: all)
  - [x] 6.1 Run `make test`; fix any regressions; exclude pre-existing renderer/ebiten gotext issue

## Dev Notes

### Existing Implementation Summary

The room power system is **already fully implemented** across multiple packages. This story focuses on comprehensive verification, testing, and gap-filling.

**Room power state (`pkg/game/state/state.go`):**
- `RoomDoorsPowered map[string]bool` — per-room door power; toggled at maintenance terminals.
- `RoomCCTVPowered map[string]bool` — per-room CCTV/hazard-control power; toggled at maintenance terminals.
- `RoomLightsPowered map[string]bool` — per-room lights (0w, default on).
- All three maps are deep-copied in `SaveCurrentDeckState`/`LoadDeckState` and reset in `AdvanceLevel`.

**Room power initialization (`pkg/game/setup/roompower.go`):**
- `InitRoomPower(g)`: Discovers all room names from grid; sets all to unpowered; powers start room doors.
- `InitMaintenanceTerminalPower(g)`: Sets all maintenance terminals `Powered = false`; powers start room terminal(s).
- Existing tests: `TestInitRoomPower_NilGridNoPanic`, `TestInitRoomPower_StartRoomDoorsPowered` (basic; need expansion).

**Solvability (`pkg/game/setup/solvability.go`):**
- `EnsureSolvabilityDoorPower(g)`: For every gatekeeper room where doors are unpowered and no adjacent terminal is reachable from start without entering R, powers R's doors.
- `getReachableCellsBlockingDoorsInto`: BFS treating door cells into target room and locked doors as impassable.
- No existing tests for solvability.

**Setup order (`pkg/game/gameplay/lifecycle.go:SetupLevel`):**
```
setup.SetupLevel(g)        // doors, InitRoomPower, generators, batteries, CCTV
levelgen.PlaceHazards      // level 2+
levelgen.PlaceFurniture
levelgen.PlacePuzzles       // level 2+
levelgen.PlaceMaintenanceTerminals
setup.EnsureSolvabilityDoorPower(g)
setup.InitMaintenanceTerminalPower(g)
MoveCell(g, g.Grid.StartCell())
```
This matches FR25: InitRoomPower → placement → EnsureSolvabilityDoorPower → InitMaintenanceTerminalPower.

**Movement checks (`pkg/game/gameplay/movement.go`):**
- `CanEnter`: Checks `RoomDoorsPowered[roomName]` for door cells; blocks movement and shows callout when unpowered.
- Existing tests: door-blocking is indirectly tested via movement tests (rooms are pre-powered); no explicit unpowered-door test.

**CCTV/hazard control checks (`pkg/game/gameplay/interactions.go`):**
- `CheckAdjacentTerminalsAtCell`: Checks `g.RoomCCTVPowered[cell.Name]`; shows "no power" message when false.
- `CheckAdjacentHazardControlsAtCell`: Checks `g.RoomCCTVPowered[cell.Name]`; shows "no power" message when false.
- No existing tests for these power checks.

**Maintenance menu (`pkg/game/menu/maintenance.go`):**
- `RoomPowerToggleMenuItem`: Toggles `RoomDoorsPowered`/`RoomCCTVPowered` on `OnActivate`. Calls `ShortOutIfOverload` when toggling ON.
- `IsSelectable()` returns `roomMaintenanceTerminalPowered(g, roomName)` — toggle only allowed when room's terminal is powered.
- Room selector allows viewing own + adjacent rooms.
- No existing tests for menu toggle logic.

**Short-out (`pkg/game/state/state.go`):**
- `ShortOutIfOverload(protectedRoomName)`: Sorts consumers deterministically (room name, then doors before CCTV), unpowers until consumption <= supply. Protected room stays on.
- No existing tests for ShortOutIfOverload.

### Architecture Requirements

- **Language:** Go 1.24; module `darkstation`.
- **Tests:** `*_test.go` alongside source; run with `go test ./...` or `make test`.
- **Test pattern:** Table-driven tests preferred. Existing test helpers: `makeGridWithRooms`, `makeMinimalGrid`, `makeTestGame`.
- **Packages:** Setup in `pkg/game/setup/`, gameplay in `pkg/game/gameplay/`, menu in `pkg/game/menu/`, state in `pkg/game/state/`.
- **No new dependencies** needed; use standard `testing` package.
- **Renderer coupling:** `movement.go` and `interactions.go` call `renderer.AddCallout` and `renderer.ApplyMarkup`. These work in tests without Ebiten running (they write to internal state). Menu tests may need to avoid `RunMenu`/`RunMenuDynamic` (requires Ebiten); test the data model (toggle state changes) not the UI rendering.

### Key Code Locations

| File | What | Relevance |
|------|------|-----------|
| `pkg/game/setup/roompower.go` | InitRoomPower, InitMaintenanceTerminalPower | Room power initialization |
| `pkg/game/setup/solvability.go` | EnsureSolvabilityDoorPower | Gatekeeper deadlock prevention |
| `pkg/game/state/state.go` | ShortOutIfOverload, CalculatePowerConsumption | Overload and consumption |
| `pkg/game/gameplay/movement.go` | CanEnter | Door power blocking check |
| `pkg/game/gameplay/interactions.go` | CheckAdjacentTerminalsAtCell, CheckAdjacentHazardControlsAtCell | CCTV/hazard power check |
| `pkg/game/menu/maintenance.go` | MaintenanceMenuHandler, RoomPowerToggleMenuItem | Toggle UI |
| `pkg/game/gameplay/lifecycle.go` | SetupLevel, ResetLevel | Setup order (FR25) |
| `pkg/game/entities/maintenance.go` | MaintenanceTerminal | Entity with Powered flag |
| `pkg/game/entities/door.go` | Door | Entity with RoomName, Locked |
| `pkg/game/world/cell.go` | GameCellData, HasDoor, HasTerminal | Cell entity queries |

### Previous Story Learnings (from 2.1)

- Tests that call `placeGenerators` directly work fine (renderer functions don't require Ebiten at runtime).
- `makeGridWithRooms(rows, cols, roomName)` helper in `setup/generators_test.go` and `makeTestGame(rows, cols)` in `gameplay/interactions_test.go` are reusable.
- Table-driven tests with subtests work well for parametric scenarios.
- ShortOutIfOverload deep-copy fix: `SaveCurrentDeckState`/`LoadDeckState` now deep-copy generators (verified in 2.1 review).
- Pre-existing renderer/ebiten gotext build failure is unchanged; exclude `renderer/ebiten` from test runs.
- For menu tests: avoid calling `RunMenu`/`RunMenuDynamic` as they require Ebiten's game loop. Test the `OnActivate` method directly on menu handlers by constructing handlers manually.

### Git Intelligence

Recent commits show table-driven tests, clean separation of concerns, and comprehensive story-driven development. Stories 1.1-1.5 (Epic 1) and 2.1 are all done. Code patterns are well-established.

### Testing Strategy

- **Unit tests:** `setup/roompower_test.go` (InitRoomPower, InitMaintenanceTerminalPower), `setup/solvability_test.go` (EnsureSolvabilityDoorPower), `state/state_test.go` (ShortOutIfOverload).
- **Movement/interaction tests:** `gameplay/movement_test.go` (CanEnter with unpowered doors), `gameplay/interactions_test.go` (CCTV/hazard power checks).
- **Menu toggle tests:** `menu/maintenance_test.go` (OnActivate toggles state, ShortOutIfOverload called, IsSelectable checks).
- **Integration tests:** `gameplay/lifecycle_test.go` (BuildGame setup order, ResetLevel power reset, deck state persistence).
- **Edge cases:** ShortOutIfOverload with multiple rooms, empty generators, toggle when already in desired state.
- **Avoid Ebiten dependency in menu tests:** Construct `MaintenanceMenuHandler` manually and call `OnActivate` on `RoomPowerToggleMenuItem` directly. Do NOT call `RunMenu`/`RunMenuDynamic`.

### Project Structure Notes

- All new test files go alongside their source.
- Existing test files to expand: `setup/roompower_test.go` (2 tests), `gameplay/movement_test.go` (8 tests), `gameplay/interactions_test.go` (17 tests).
- New test files needed: `setup/solvability_test.go`, `menu/maintenance_test.go`.
- No conflicts with unified project structure detected.

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 2, Story 2.2]
- [Source: specs/power-system.md — §4 Room Power, §5 Maintenance Terminals, §3 Power Consumption, §8 Lifecycle]
- [Source: docs/architecture.md — Technology Stack, Testing Strategy]
- [Source: pkg/game/setup/roompower.go — InitRoomPower, InitMaintenanceTerminalPower]
- [Source: pkg/game/setup/solvability.go — EnsureSolvabilityDoorPower]
- [Source: pkg/game/state/state.go — ShortOutIfOverload, CalculatePowerConsumption]
- [Source: pkg/game/gameplay/movement.go — CanEnter]
- [Source: pkg/game/gameplay/interactions.go — CheckAdjacentTerminalsAtCell, CheckAdjacentHazardControlsAtCell]
- [Source: pkg/game/menu/maintenance.go — MaintenanceMenuHandler, RoomPowerToggleMenuItem]
- [Source: _bmad-output/implementation-artifacts/2-1-generators-and-batteries.md — Previous story learnings]

## Dev Agent Record

### Agent Model Used

Claude claude-4.6-opus (via Cursor)

### Debug Log References

- HazardControl constructor API mismatch in interactions_test.go: fixed `NewHazardControl(name, type)` → `NewHazardControl(hazardType, hazard)` and `Active` → `Activated`.
- ResetLevel test: start room name changes on re-generation; fixed test to check post-reset start room.

### Completion Notes List

- Task 1: Added 7 tests to `roompower_test.go` covering InitRoomPower (multi-room defaults, different room names, idempotency) and InitMaintenanceTerminalPower (start room powered, nil grid safety).
- Task 2: Created `solvability_test.go` with 6 tests covering EnsureSolvabilityDoorPower (gatekeeper no terminal → powered, non-gatekeeper → not powered, gatekeeper with adjacent terminal → not powered, locked door impassable, nil grid safety, start room already powered).
- Task 3: Created `maintenance_test.go` with 6 tests covering RoomPowerToggleMenuItem (doors ON/OFF, CCTV ON/OFF, ShortOutIfOverload with protected room, consumption recalculation on toggle OFF, IsSelectable requires powered terminal, OnActivate rejects unpowered terminal).
- Task 4: Added 2 tests to `movement_test.go` (CanEnter with unpowered/powered doors) and 4 tests to `interactions_test.go` (CCTV terminal unpowered/powered, hazard control unpowered/powered).
- Task 5: Added 5 tests to `lifecycle_test.go` (BuildGame start room doors powered, start room maintenance terminal powered, ResetLevel reinitializes room power, SaveLoadDeckState room power maps preserved, SaveLoadDeckState room power deep copy isolation).
- Task 6: Full test suite passes (excluding pre-existing renderer/ebiten gotext issue). Zero regressions.
- Total new tests: 30 tests across 5 test files (2 new files, 3 expanded).
- **Code review fixes (2026-02-22):** Added TestBuildGame_SetupOrderIncludesSolvability (integration test for setup order); added 3 ShortOutIfOverload unit tests to state_test.go; fixed dead assertion in TestSaveLoadDeckState_RoomPowerDeepCopy; updated File List to include state.go and state_test.go. Note: state.go contains generator deep-copy fix (from 2.1 verification).

### File List

- `pkg/game/setup/roompower_test.go` (rewritten — 7 tests for InitRoomPower and InitMaintenanceTerminalPower)
- `pkg/game/setup/solvability_test.go` (new — 6 tests for EnsureSolvabilityDoorPower)
- `pkg/game/menu/maintenance_test.go` (new — 6 tests for RoomPowerToggleMenuItem and MaintenanceMenuHandler)
- `pkg/game/gameplay/movement_test.go` (modified — added 2 tests for CanEnter with unpowered/powered doors)
- `pkg/game/gameplay/interactions_test.go` (modified — added 4 tests for CCTV/hazard control power checks)
- `pkg/game/gameplay/lifecycle_test.go` (modified — added 6 integration tests: setup order, ResetLevel, deck state persistence, solvability setup)
- `pkg/game/state/state.go` (modified — generator deep copy in SaveCurrentDeckState/LoadDeckState; from 2.1 verification)
- `pkg/game/state/state_test.go` (modified — added ShortOutIfOverload unit tests)

### Change Log

- 2026-02-22: Story 2.2 implementation complete — 30 new tests across 5 files verifying room power system (InitRoomPower, EnsureSolvabilityDoorPower, maintenance menu toggles, CanEnter door power, CCTV/hazard power checks, setup order FR25, deck state persistence). No production code changes needed; existing implementation verified correct.
- 2026-02-22: Code review fixes — added ShortOutIfOverload unit tests (state_test.go), TestBuildGame_SetupOrderIncludesSolvability, fixed TestSaveLoadDeckState_RoomPowerDeepCopy dead assertion, updated File List (state.go, state_test.go).
