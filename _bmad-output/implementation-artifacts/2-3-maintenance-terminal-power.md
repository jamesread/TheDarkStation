# Story 2.3: Maintenance Terminal Power

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a player,
I want only some maintenance terminals to be usable until I restore power from another terminal,
so that I must route power through the station.

## Acceptance Criteria

1. **Given** each maintenance terminal has a `Powered` flag
   **When** the level starts
   **Then** only start room terminal(s) are powered (`InitMaintenanceTerminalPower`)
   **And** I can open the maintenance menu only at a powered terminal
   **And** at an unpowered terminal I see a message to restore power from another maintenance terminal
   **And** "Restore power to nearby terminals" at a powered terminal powers all terminals in adjacent rooms (including own room)

## Tasks / Subtasks

- [x] Task 1: Verify InitMaintenanceTerminalPower and terminal power state (AC: #1)
  - [x] 1.1 Verify `InitMaintenanceTerminalPower` sets all terminals unpowered except start room (expand `pkg/game/setup/roompower_test.go`)
  - [x] 1.2 Add test: multiple terminals in start room → all powered; terminals in other rooms → unpowered
  - [x] 1.3 Add test: InitMaintenanceTerminalPower idempotent; deck state Save/Load preserves terminal Powered flag

- [x] Task 2: Verify unpowered terminal blocks menu and shows message (AC: #1)
  - [x] 2.1 Add test: interaction at unpowered maintenance terminal does NOT open menu; callout/message shown (in `pkg/game/gameplay/interactions_test.go`)
  - [x] 2.2 Add test: interaction at powered maintenance terminal opens menu
  - [x] 2.3 Verify message text matches: "Terminal has no power. Restore power from another maintenance terminal." (or equivalent)

- [x] Task 3: Implement "Restore power to nearby terminals" menu action (AC: #1)
  - [x] 3.1 Add `RestorePowerNearbyTerminalsMenuItem` to maintenance menu (in `pkg/game/menu/maintenance.go`)
  - [x] 3.2 On activate: use `setup.GetAdjacentRoomNames(g.Grid, terminalRoomName)`; for each room (including own), set `Powered = true` on all maintenance terminals in those rooms
  - [x] 3.3 Show feedback: "Restored power to N terminal(s)" or "No unpowered terminals in nearby rooms"
  - [x] 3.4 Add unit tests for the restore action (powers terminals in adjacent rooms; own room; no duplicates)

- [x] Task 4: Integration and regression (AC: all)
  - [x] 4.1 Add integration test: `BuildGame` with multi-room level → start room terminal(s) powered, others unpowered; player can restore from start to adjacent
  - [x] 4.2 Run full test suite; fix any regressions
  - [x] 4.3 Verify ResetLevel and AdvanceLevel re-initialize terminal power correctly

## Dev Notes

### Existing Implementation Summary

**InitMaintenanceTerminalPower** (`pkg/game/setup/roompower.go`):
- Sets all maintenance terminals `Powered = false`; then every terminal in the start room gets `Powered = true`.
- Already called in setup order after `EnsureSolvabilityDoorPower`.
- Tests exist: `TestInitMaintenanceTerminalPower_StartRoomPowered`, `TestInitMaintenanceTerminalPower_NilGridNoPanic`.

**Unpowered terminal blocking** (`pkg/game/gameplay/interactions.go:CheckAdjacentMaintenanceTerminalAtCell`):
- If `!maintenanceTerm.Powered`, logs "Terminal has no power. Restore power from another maintenance terminal." and returns true (interaction consumed) **without opening the menu**.
- Menu opens only when terminal is powered.

**Maintenance menu** (`pkg/game/menu/maintenance.go`):
- `MaintenanceTerminalPowerMenuItem` exists: toggles individual terminal power for **other** terminals in the **same room** only. This is a per-terminal toggle.
- **Gap**: The spec requires **"Restore power to nearby terminals"** — a single batch action that powers ALL terminals in `GetAdjacentRoomNames` (including own room). The current per-terminal toggle does not match: (a) it only shows other terminals in the same room, (b) it toggles on/off, whereas "Restore" should only power (set true). Implement the batch action as specified.

**Adjacency** (`pkg/game/setup/roompower.go` or similar):
- `setup.GetAdjacentRoomNames(grid, roomName)` returns rooms that share a corridor or cell boundary with the terminal's room, plus the own room. Same definition as for room power toggles.

### Architecture Requirements

- **Language:** Go 1.24; module `darkstation`.
- **Tests:** `*_test.go` alongside source; `go test ./...` or `make test`.
- **Test pattern:** Table-driven tests; helpers: `makeGridWithRooms`, `makeMinimalGrid`, `makeTestGame` from previous stories.
- **Packages:** Setup in `pkg/game/setup/`, gameplay in `pkg/game/gameplay/`, menu in `pkg/game/menu/`, state in `pkg/game/state/`.
- **No new dependencies**; standard `testing` package.
- **Renderer coupling:** Menu handlers use `renderer.AddCallout`, `renderer.FormatPowerWatts`. Test `OnActivate` directly; avoid `RunMenu`/`RunMenuDynamic` (Ebiten-dependent).

### Key Code Locations

| File | What | Relevance |
|------|------|-----------|
| `pkg/game/setup/roompower.go` | InitMaintenanceTerminalPower | Terminal power initialization |
| `pkg/game/setup/helpers.go` | GetAdjacentRoomNames | Adjacent room list for restore action |
| `pkg/game/gameplay/interactions.go` | CheckAdjacentMaintenanceTerminalAtCell | Unpowered terminal check, menu open |
| `pkg/game/menu/maintenance.go` | MaintenanceMenuHandler, GetMenuItems | Add RestorePowerNearbyTerminalsMenuItem |
| `pkg/game/entities/maintenance.go` | MaintenanceTerminal.Powered | Entity field |
| `pkg/game/gameplay/lifecycle.go` | SetupLevel, ResetLevel, AdvanceLevel | Setup order, deck state |

### Previous Story Learnings (from 2.2)

- Avoid `RunMenu`/`RunMenuDynamic` in tests; construct handlers manually and call `OnActivate` on menu items.
- `makeTestGame`, `makeGridWithRooms`, `makeMinimalGrid` are reusable.
- Table-driven tests work well. Exclude `renderer/ebiten` from test runs if gotext build failure persists.
- Deep-copy of power maps in `SaveCurrentDeckState`/`LoadDeckState`; terminal `Powered` must be preserved in deck state if terminals are stored there (check DeckState structure).

### Testing Strategy

- **Unit tests:** `setup/roompower_test.go` (InitMaintenanceTerminalPower expansion), `menu/maintenance_test.go` (RestorePowerNearbyTerminals OnActivate, adjacency).
- **Interaction tests:** `gameplay/interactions_test.go` (unpowered vs powered terminal behaviour).
- **Integration tests:** `gameplay/lifecycle_test.go` (BuildGame terminal power state, restore flow).
- **Edge cases:** Nil grid, no adjacent rooms, all already powered, multiple terminals per room.

### Project Structure Notes

- New menu item in `maintenance.go`; tests in `maintenance_test.go`.
- No conflicts with unified project structure.

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 2, Story 2.3]
- [Source: specs/power-system.md — §5 Maintenance Terminals, §5.2 Terminal Power, §5.3 Restore Power to Nearby Terminals]
- [Source: docs/architecture.md — Technology Stack, Testing Strategy]
- [Source: pkg/game/setup/roompower.go — InitMaintenanceTerminalPower]
- [Source: pkg/game/gameplay/interactions.go — CheckAdjacentMaintenanceTerminalAtCell]
- [Source: pkg/game/menu/maintenance.go — MaintenanceMenuHandler, GetMenuItems]
- [Source: _bmad-output/implementation-artifacts/2-2-room-power-doors-and-cctv.md — Previous story]

## Dev Agent Record

### Agent Model Used

Claude (via Cursor)

### Debug Log References

### Completion Notes List

- **Code review fixes (2026-02-22):** (1) Added renderer.AddCallout for unpowered maintenance terminal (consistent with CCTV/door feedback). (2) Added nil Grid guard in RestorePowerNearbyTerminalsMenuItem to prevent panic. (3) Documented TestSaveLoadDeckState placement. (4) Documented MaintenanceTerminalPowerMenuItem toggle as intentional UX.
- **Code review fixes (2026-02-22, follow-up):** (5) Added explicit powered-terminal path test to verify maintenance menu open call. (6) Removed skip-based gap in BuildGame restore-flow integration by retrying generation and failing when the scenario cannot be produced.
- Task 1: Added 3 tests to `roompower_test.go`: TestInitMaintenanceTerminalPower_MultipleTerminalsInStartRoom, TestInitMaintenanceTerminalPower_Idempotent, TestSaveLoadDeckState_MaintenanceTerminalPowerPreserved.
- Task 2: Added TestCheckAdjacentMaintenanceTerminalAtCell_UnpoweredBlocksMenu in `interactions_test.go` — verifies unpowered terminal returns true, adds message, does not open menu. Task 2.2 (powered opens menu) covered by existing BuildGame integration; RunMaintenanceMenu blocks so no standalone unit test.
- Task 3: Implemented RestorePowerNearbyTerminalsMenuItem in `maintenance.go`; powers all terminals in GetAdjacentRoomNames (own + adjacent rooms); feedback messages. Added 3 unit tests in `maintenance_test.go`.
- Task 4: Added TestBuildGame_MaintenanceTerminalRestoreFlow (integration) and extended TestResetLevel_ReinitializesRoomPower to verify maintenance terminal power reset. Full test suite passes (excluding pre-existing renderer/ebiten gotext build failure).

### File List

- `pkg/game/setup/roompower_test.go` (modified — added 3 InitMaintenanceTerminalPower tests)
- `pkg/game/gameplay/interactions.go` (modified — AddCallout for unpowered terminal; code review)
- `pkg/game/gameplay/interactions_test.go` (modified — added TestCheckAdjacentMaintenanceTerminalAtCell_UnpoweredBlocksMenu)
- `pkg/game/menu/maintenance.go` (modified — added RestorePowerNearbyTerminalsMenuItem, nil Grid guard, MaintenanceTerminalPowerMenuItem doc; code review)
- `pkg/game/menu/maintenance_test.go` (modified — added 3 RestorePowerNearbyTerminals tests)
- `pkg/game/gameplay/lifecycle_test.go` (modified — added TestBuildGame_MaintenanceTerminalRestoreFlow, extended TestResetLevel_ReinitializesRoomPower)

### Change Log

- 2026-02-22: Code review fixes — AddCallout for unpowered terminal, nil Grid guard in restore action, documentation.
- 2026-02-22: Story 2.3 implementation complete — InitMaintenanceTerminalPower tests expanded, unpowered terminal interaction test added, RestorePowerNearbyTerminalsMenuItem implemented with unit and integration tests. All acceptance criteria satisfied.
- 2026-02-22: Code review follow-up — added powered maintenance-terminal menu-open test and hardened integration test to avoid skip-only validation.
