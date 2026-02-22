# Story 2.4: Power Consumption and Overload

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a player,
I want power consumption to be calculated and overload to cause a short-out,
so that I must balance supply and demand.

## Acceptance Criteria

1. **Given** PowerSupply (powered generators × 100 W) and PowerConsumption (doors 10 W per room when powered, CCTV 10 W per terminal when room CCTV on, solved puzzles 3 W each)
   **When** I toggle a room's doors or CCTV ON and the new consumption would exceed PowerSupply
   **Then** ShortOutIfOverload(protectedRoomName) runs: other rooms' doors and CCTV are auto-unpowered in a deterministic order until PowerConsumption ≤ PowerSupply
   **And** the room I just turned on stays on (protected)
   **And** the player is told that other systems shorted out
   **And** when consumption already exceeds supply (passive), a one-time warning per cycle is shown (PowerOverloadWarned)

## Tasks / Subtasks

- [x] Task 1: Verify PowerConsumption calculation (AC: #1)
  - [x] 1.1 Expand CalculatePowerConsumption tests: doors (10 W per room when powered), CCTV (10 W per terminal when room CCTV on), puzzles (3 W per solved)
  - [x] 1.2 Add test: consumption updates when room power state changes

- [x] Task 2: Verify ShortOutIfOverload behaviour (AC: #1)
  - [x] 2.1 Verify ShortOutIfOverload protects the specified room (existing tests; add edge cases if needed)
  - [x] 2.2 Verify deterministic unpower order (rooms by name, then doors before CCTV)
  - [x] 2.3 Verify player receives "other systems shorted out" feedback when ShortOutIfOverload returns true (menu OnActivate helpText)

- [x] Task 3: Verify passive overload warning (AC: #1)
  - [x] 3.1 Add test: PowerOverloadWarned set once per cycle when PowerConsumption > PowerSupply (UpdateLightingExploration path)
  - [x] 3.2 Verify PowerOverloadWarned resets appropriately (e.g. AdvanceLevel, consumption drops)

- [x] Task 4: Integration and regression (AC: all)
  - [x] 4.1 Run full test suite; fix any regressions
  - [x] 4.2 Verify maintenance menu toggle flow: toggle ON → ShortOutIfOverload → feedback

## Dev Notes

### Existing Implementation Summary

**CalculatePowerConsumption** (`pkg/game/state/state.go`):
- Sums doors (10 W per room when RoomDoorsPowered), CCTV (10 W per terminal when RoomCCTVPowered), solved puzzles (3 W each).
- Tests: TestCalculatePowerConsumption_NoPoweredDevices.

**ShortOutIfOverload** (`pkg/game/state/state.go`):
- Sorts consumers by room name, then doors before CCTV; unpowers until consumption ≤ supply; protected room excluded.
- Tests: TestShortOutIfOverload_NoOverloadReturnsFalse, TestShortOutIfOverload_UnpowersOthersUntilWithinSupply, TestShortOutIfOverload_DeterministicOrder.
- Maintenance menu calls it when toggling doors/CCTV ON and shows "Power overload! Other systems shorted out." when it returns true.

**PowerOverloadWarned** (`pkg/game/gameplay/lighting.go`):
- UpdateLightingExploration: when PowerConsumption > PowerSupply and !PowerOverloadWarned, sets it true and logs warning; when within supply, resets false.
- Reset on AdvanceLevel (lifecycle.go).

### Architecture Requirements

- Go 1.24; `*_test.go` alongside source; table-driven tests.
- Packages: state, gameplay, menu. No new dependencies.

### Key Code Locations

| File | What |
|------|------|
| `pkg/game/state/state.go` | CalculatePowerConsumption, ShortOutIfOverload, PowerOverloadWarned |
| `pkg/game/state/state_test.go` | Existing overload and consumption tests |
| `pkg/game/gameplay/lighting.go` | UpdateLightingExploration, passive overload warning |
| `pkg/game/menu/maintenance.go` | RoomPowerToggleMenuItem OnActivate, ShortOutIfOverload call, helpText |

### Previous Story Learnings (from 2.3)

- Exclude renderer/ebiten from test runs if gotext build failure persists.
- Test menu item OnActivate directly; avoid RunMenu (Ebiten).

### References

- [Source: specs/power-system.md — §3 Power Consumption, §3.3–3.4]
- [Source: _bmad-output/planning-artifacts/epics.md — Story 2.4]
- [Source: _bmad-output/implementation-artifacts/2-2-room-power-doors-and-cctv.md]

## Dev Agent Record

### Agent Model Used

Claude (via Cursor)

### Debug Log References

### Completion Notes List

- Task 1: Added TestCalculatePowerConsumption_DoorsCCTVPuzzles and TestCalculatePowerConsumption_UpdatesWhenRoomPowerChanges in state_test.go.
- Task 2: Added helpText assertion to TestToggleDoorsON_ShortOutProtectsToggledRoom verifying "Power overload! Other systems shorted out." feedback.
- Task 3: Created lighting_test.go with TestUpdateLightingExploration_PassiveOverloadSetsPowerOverloadWarned and TestUpdateLightingExploration_ResetsPowerOverloadWarnedWhenWithinSupply.
- Task 4: Full test suite passes (excluding pre-existing renderer/ebiten build failure). Power consumption, ShortOutIfOverload, and passive overload behaviour all verified.
- Code review fixes (2026-02-22): corrected doors consumption to 10W per powered room (not per door), updated overload tests for deterministic short-out under corrected model, and added explicit help text for persistent overload when protected-room load alone exceeds supply.

### File List

- `pkg/game/state/state.go` (modified — door consumption corrected to per-room accounting)
- `pkg/game/state/state_test.go` (modified — added 2 CalculatePowerConsumption tests)
- `pkg/game/menu/maintenance.go` (modified — added persistent-overload warning help text)
- `pkg/game/menu/maintenance_test.go` (modified — added helpText assertion to ShortOut test)
- `pkg/game/gameplay/lighting_test.go` (new — passive overload warning tests)

### Change Log

- 2026-02-22: Story 2.4 implementation complete — Power consumption, ShortOutIfOverload, and passive overload (PowerOverloadWarned) all verified with tests. Implementation was largely present from story 2.2; added verification coverage.
- 2026-02-22: Code review fixes — aligned door consumption with AC (10W per powered room), strengthened deterministic short-out tests, and added persistent overload user feedback.
