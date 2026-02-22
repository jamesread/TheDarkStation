# Story 2.5: Lighting and Exploration

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a player,
I want lighting to depend on available power and whether I have visited a cell,
so that darkness reflects power state.

## Acceptance Criteria

1. **Given** GetAvailablePower() = PowerSupply - PowerConsumption, visit state per cell, and room lights enabled for that room (`RoomLightsPowered`)
   **When** GetAvailablePower() > 0 and I have visited a cell
   **Then** lights are on for that cell (LightsOn, Lighted; cell stays discovered)
   **And** when GetAvailablePower() <= 0, cells beyond a small radius of the player can darken (discovered/visited cleared)
   **And** lighting does not consume power

## Tasks / Subtasks

- [x] Task 1: Verify lighting and exploration behaviour (AC: #1)
  - [x] 1.1 Add/expand tests: when GetAvailablePower() > 0 and cell visited → LightsOn, Lighted, discovered/visited preserved
  - [x] 1.2 Add test: when GetAvailablePower() <= 0, cells beyond small radius (e.g. 5×5) darken (discovered/visited cleared) unless Lighted
  - [x] 1.3 Add test: cells within radius of player stay visible even when power <= 0
  - [x] 1.4 Verify lighting does not consume power (assert no wattage from cells; existing spec confirms)

- [x] Task 2: Reconcile RoomLightsPowered with spec (AC: #1)
  - [x] 2.1 Verify RoomLightsPowered toggle (from story 2.2) correctly gates lights-on behaviour when availablePower > 0
  - [x] 2.2 Document or adjust if spec requires lights to depend only on GetAvailablePower + visit (no per-room lights toggle)

- [x] Task 3: Integration and regression (AC: all)
  - [x] 3.1 Run full test suite (excluding renderer/ebiten if gotext build fails); fix regressions
  - [x] 3.2 Verify UpdateLightingExploration is invoked in correct order (consumption → supply → lighting) per power-system spec §6.2

## Dev Notes

### Existing Implementation Summary

**UpdateLightingExploration** (`pkg/game/gameplay/lighting.go`):
- Recalculates PowerConsumption, UpdatePowerSupply(); uses GetAvailablePower().
- When availablePower > 0 and cell.Visited and RoomLightsPowered[room] true → Sets LightsOn, Lighted; keeps discovered/visited.
- When availablePower <= 0 or lights disabled → Lights off; cells within 5×5 radius of player stay discovered; far cells with !Lighted get discovered/visited cleared.
- PowerOverloadWarned set/reset (from story 2.4).

**State and cells:**
- `GameCellData.LightsOn`, `Lighted` in `pkg/game/world/cell.go`.
- `RoomLightsPowered map[string]bool` in state (per-room lights toggle; default on from InitRoomPower).
- `GetAvailablePower()` = PowerSupply - PowerConsumption.

**Invocation sites:** lifecycle.go (after build/advance), movement.go (after move), interactions.go (after generator interaction), main.go.

### Architecture Requirements

- Go 1.24; `*_test.go` alongside source; table-driven tests.
- Packages: state, gameplay, world. No new dependencies.
- Exclude renderer/ebiten from test runs if gotext build failure persists.

### Key Code Locations

| File | What |
|------|------|
| `pkg/game/gameplay/lighting.go` | UpdateLightingExploration |
| `pkg/game/gameplay/lighting_test.go` | Existing PowerOverloadWarned tests (story 2.4) |
| `pkg/game/world/cell.go` | LightsOn, Lighted, SetLightsOn, AreLightsOn, IsLighted |
| `pkg/game/state/state.go` | GetAvailablePower, PowerSupply, PowerConsumption, RoomLightsPowered |
| `pkg/game/setup/roompower.go` | InitRoomPower sets RoomLightsPowered default true |
| `pkg/game/renderer/ebiten/cell.go` | Rendering uses GetAvailablePower, cell.Discovered, cell.Visited |

### Previous Story Learnings (from 2.3, 2.4)

- Exclude renderer/ebiten from test runs if gotext build failure persists.
- Test gameplay logic directly; avoid RunMenu (Ebiten).
- PowerOverloadWarned and UpdateLightingExploration already tested in lighting_test.go.

### Spec vs implementation notes

- **Radius:** Spec says "small radius" (e.g. 3×3); current code uses 5×5 (rowDist<=2, colDist<=2). Either is acceptable; document and test consistently.
- **RoomLightsPowered:** Story 2.2 introduced per-room lights toggle; FR15 says "visibility depends on GetAvailablePower() > 0 and visit state". RoomLightsPowered acts as an additional gate (lights off if room lights disabled). Retain unless spec explicitly forbids.

### References

- [Source: specs/power-system.md — §6 Lighting and Exploration]
- [Source: _bmad-output/planning-artifacts/epics.md — Story 2.5, FR15]
- [Source: _bmad-output/implementation-artifacts/2-4-power-consumption-and-overload.md — PowerOverloadWarned, UpdateLightingExploration]

## Dev Agent Record

### Agent Model Used

Composer (via Cursor)

### Debug Log References

### Completion Notes List

- Task 1: Added 5 tests to lighting_test.go: WhenPowerAndVisited_SetsLightsOnAndLighted, WhenNoPower_FarCellsDarkenUnlessLighted, WhenNoPower_NearCellsStayVisible, WhenNoPower_LightedFarCellsStayDiscovered, LightingDoesNotConsumePower.
- Task 2: Added TestUpdateLightingExploration_WhenRoomLightsOff_LightsStayOffDespitePower. RoomLightsPowered retained; gates lights-on when false.
- Task 3: Full test suite passes (excluding pre-existing renderer/ebiten gotext build failure). UpdateLightingExploration order (consumption → supply → lighting) confirmed in lighting.go.
- Code review fixes (2026-02-22): clarified radius comments and added test `TestUpdateLightingExploration_RecalculatesPowerStateBeforeApplyingLighting` to verify lighting uses freshly recalculated supply/consumption state each update.

### File List

- `pkg/game/gameplay/lighting.go` (modified — clarified 5x5 neighborhood comments and behavior notes)
- `pkg/game/gameplay/lighting_test.go` (modified — added lighting/exploration and power-recalculation tests)

### Change Log

- 2026-02-22: Story 2.5 implementation complete — Lighting and exploration behaviour verified with tests; RoomLightsPowered toggle confirmed; lighting does not consume power.
- 2026-02-22: Code review fixes — aligned AC wording with RoomLightsPowered gating, clarified radius documentation, and added recalculation-order safety test coverage.
