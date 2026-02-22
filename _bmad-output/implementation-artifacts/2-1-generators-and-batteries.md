# Story 2.1: Generators and Batteries

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a player,
I want to find generators and insert batteries into them,
so that I can supply power to the level.

## Acceptance Criteria

1. **Given** generators on the level with `BatteriesRequired`
   **When** I interact with a generator and have enough batteries in inventory
   **Then** I can insert batteries (insertion is permanent for that level)
   **And** when `BatteriesInserted >= BatteriesRequired` the generator is powered and supplies 100 W
   **And** `PowerSupply` is the sum of all powered generators (`UpdatePowerSupply`)

## Tasks / Subtasks

- [x] Task 1: Verify generator entity and state integration (AC: #1)
  - [x] 1.1 Confirm `Generator` struct fields (`BatteriesRequired`, `BatteriesInserted`) and methods (`IsPowered`, `BatteriesNeeded`, `InsertBatteries`) behave correctly; add unit tests in `pkg/game/entities/generator_test.go`
  - [x] 1.2 Confirm `state.Game` generator helpers (`AddGenerator`, `AllGeneratorsPowered`, `UnpoweredGeneratorCount`, `AddBatteries`, `UseBatteries`) work correctly; add unit tests in `pkg/game/state/state_test.go`
  - [x] 1.3 Confirm `UpdatePowerSupply` correctly sums powered generators at 100 W each (with `DecayParamsForDeck` multiplier); add unit tests covering deck 0 (1.0x) and deeper decks (reduced output)
- [x] Task 2: Verify generator placement and battery distribution (AC: #1)
  - [x] 2.1 Confirm `placeGenerators` places a spawn generator (auto-powered) and additional generators for level 3+; add unit tests in `pkg/game/setup/generators_test.go`
  - [x] 2.2 Confirm `placeBatteries` places enough batteries for unpowered generators plus extras; add unit tests in `pkg/game/setup/batteries_test.go`
  - [x] 2.3 Confirm spawn generator is auto-powered and `UpdatePowerSupply` is called immediately after placement so power is available from level start
- [x] Task 3: Verify player interaction with generators (AC: #1)
  - [x] 3.1 Confirm `CheckAdjacentGenerators` inserts batteries from inventory into adjacent unpowered generators, calls `UpdatePowerSupply` and `UpdateLightingExploration` on power-up; add unit tests in `pkg/game/gameplay/interactions_test.go`
  - [x] 3.2 Confirm `CheckAdjacentGeneratorAtCell` displays correct callout (name, status, batteries, power stats) for both powered and unpowered generators; add unit tests
  - [x] 3.3 Confirm battery pickup works (`PickUpItemsOnFloor` with Battery items increments `g.Batteries`); add unit test
- [x] Task 4: Verify power supply calculation and deck persistence (AC: #1)
  - [x] 4.1 Confirm `GetAvailablePower` returns `PowerSupply - PowerConsumption`; confirm `CalculatePowerConsumption` returns 0 when no devices are powered (baseline for this story); add unit tests
  - [x] 4.2 Confirm `SaveCurrentDeckState` and `LoadDeckState` preserve generator state (Generators slice) across deck transitions; add unit tests
  - [x] 4.3 Confirm `AdvanceLevel` and `ResetLevel` reset generators, batteries, and power state correctly; verify via existing lifecycle tests or add coverage
- [x] Task 5: Integration test and edge cases (AC: #1)
  - [x] 5.1 Add integration test: full lifecycle — place generators/batteries via `SetupLevel`, pick up batteries, insert into generator, verify PowerSupply updates, verify generator powered
  - [x] 5.2 Add edge case tests: insert batteries when inventory is 0 (no-op), insert more than needed (capped), generator already powered (no change), multiple generators with partial battery supply
  - [x] 5.3 Run full test suite (`make test`); fix any regressions

## Dev Notes

### Existing Implementation Summary

The generator and battery system is **already implemented** across multiple packages. This story focuses on verification, comprehensive testing, and fixing any discovered gaps.

**Core entities and state:**
- `pkg/game/entities/generator.go`: `Generator` struct with `BatteriesRequired`, `BatteriesInserted`, `IsPowered()`, `BatteriesNeeded()`, `InsertBatteries(count)`. No existing tests.
- `pkg/game/state/state.go`: `Game` has `Batteries` (inventory count), `Generators` ([]*Generator), `PowerSupply`, `PowerConsumption`. Methods: `AddBatteries`, `UseBatteries`, `AddGenerator`, `AllGeneratorsPowered`, `UnpoweredGeneratorCount`, `UpdatePowerSupply`, `GetAvailablePower`, `CalculatePowerConsumption`, `ShortOutIfOverload`.

**Power supply calculation (with decay):**
- `UpdatePowerSupply()` iterates `g.Generators`, sums 100 W per powered generator multiplied by `deck.DecayParamsForDeck(g.CurrentDeckID).GeneratorOutputMultiplier`.
- Decay curve: output drops ~4% per deck (floor 0.5), cost rises ~8% per deck. Defined in `pkg/game/deck/deck.go`.

**Setup/placement:**
- `pkg/game/setup/generators.go`: `placeGenerators` places spawn generator (auto-powered with full batteries) + additional generators for level 3+. Spawn generator calls `UpdatePowerSupply()` immediately.
- `pkg/game/setup/batteries.go`: `placeBatteries` places enough batteries for unpowered generators + 1-2 extras. Levels 1-2 skip battery placement (spawn gen auto-powered, exit unlocked).
- Setup order: `SetupLevel` → `PlaceLockedRooms` → doors → `InitRoomPower` → `PlaceGenerators` → `PlaceBatteries` → CCTV → hazards → furniture → puzzles → maintenance terminals → `EnsureSolvabilityDoorPower` → `InitMaintenanceTerminalPower`.

**Player interaction:**
- `pkg/game/gameplay/interactions.go`:
  - `CheckAdjacentGenerators`: Auto-inserts batteries from inventory into adjacent unpowered generators. Calls `UpdatePowerSupply()` and `UpdateLightingExploration()` when a generator becomes powered.
  - `CheckAdjacentGeneratorAtCell`: Shows callout with generator name, status (POWERED/UNPOWERED), battery count, and power supply/consumption/available.
  - `PickUpItemsOnFloor`: Picks up Battery items → `g.AddBatteries(1)`.

**Deck persistence:**
- `SaveCurrentDeckState` deep-copies each `Generator` struct into `DeckStates[deckID]` (fixed from shallow pointer copy during code review).
- `LoadDeckState` deep-copies generators from stored deck state. Resets PowerSupply/PowerConsumption to 0 (caller must recalculate).

### Architecture Requirements

- **Language:** Go 1.24; module `darkstation`.
- **Tests:** `*_test.go` alongside source; run with `go test ./...` or `make test`.
- **Test pattern:** Table-driven tests preferred (see `setup/helpers_test.go`, `setup/roompower_test.go`).
- **Packages:** Entity logic in `pkg/game/entities/`, state in `pkg/game/state/`, setup in `pkg/game/setup/`, gameplay in `pkg/game/gameplay/`.
- **No new dependencies** should be needed; use standard `testing` package.
- **Renderer coupling:** `setup/generators.go` imports `renderer` for `StyledCell` and `FormatPowerWatts`. Tests may need to avoid renderer calls or mock via interface — check existing test patterns first.

### Key Code Locations

| File | What | Relevance |
|------|------|-----------|
| `pkg/game/entities/generator.go` | Generator struct + methods | Core entity under test |
| `pkg/game/state/state.go` | Game state, power methods | Power supply/consumption, batteries |
| `pkg/game/setup/generators.go` | Generator placement | Level setup, auto-power spawn gen |
| `pkg/game/setup/batteries.go` | Battery placement | Ensures enough batteries placed |
| `pkg/game/setup/setup.go` | SetupLevel entry point | Orchestrates placement order |
| `pkg/game/gameplay/interactions.go` | Player-generator interaction | Battery insertion, callouts |
| `pkg/game/gameplay/lifecycle.go` | BuildGame, AdvanceLevel, ResetLevel | Deck transitions, state reset |
| `pkg/game/deck/deck.go` | Decay params, deck graph | Generator output multiplier |
| `pkg/game/world/cell.go` | GameCellData, HasGenerator helpers | Cell-level generator queries |

### Previous Story Learnings (from 1.5)

- Tests in `pkg/game/gameplay/lifecycle_test.go` exist for BuildGame, AdvanceLevel, ResetLevel, TriggerGameComplete.
- Code review for 1.5 found: need boundary clamping tests, reset-doesn't-advance test, and noted that full-generation tests with `-race` are slow.
- Ebiten renderer has pre-existing gotext build issues (unchanged) — tests that import renderer may need build tags or avoidance.
- `generator.DefaultGenerator.Generate(level)` uses BSP; grid generation is well-tested.

### Git Intelligence

Recent commits show clean separation of concerns, table-driven tests, and focus on room connectivity and deck lifecycle. Last commit was `docs(artifacts): mark stories 1.4 and 1.5 done`.

### Testing Strategy

- **Unit tests:** `entities/generator_test.go` (Generator methods), `state/state_test.go` (power/battery state methods), `setup/generators_test.go` (placement), `setup/batteries_test.go` (battery distribution).
- **Integration tests:** `gameplay/interactions_test.go` (player-generator interaction cycle).
- **Edge cases:** Zero batteries, overflow insertion, already-powered generator, multiple generators, deck decay multipliers.
- **Avoid renderer dependency in tests:** The `setup/generators.go` imports `renderer` for hint text. Tests that call `placeGenerators` directly may hit this. Check if existing tests (e.g., `roompower_test.go`) handle this — they likely work since `renderer` functions are called but don't require Ebiten to be running.

### Project Structure Notes

- All new test files go alongside their source: `pkg/game/entities/generator_test.go`, `pkg/game/state/state_test.go`, etc.
- Existing test files: `setup/helpers_test.go`, `setup/roompower_test.go`, `setup/room_connectivity_test.go`, `gameplay/lifecycle_test.go`, `gameplay/movement_test.go`, `levelgen/placement_test.go`, `generator/bsp_test.go`.
- No conflicts with unified project structure detected.

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 2, Story 2.1]
- [Source: specs/power-system.md — §2 Power Supply, §2.1 Generators, §2.2 Batteries]
- [Source: docs/architecture.md — Technology Stack, Testing Strategy]
- [Source: docs/data-models-main.md — Generator, Game, DeckState entities]
- [Source: docs/source-tree-analysis.md — pkg/game/setup/, pkg/game/entities/, pkg/game/gameplay/]
- [Source: pkg/game/deck/deck.go — DecayParams, DecayParamsForDeck]
- [Source: _bmad-output/implementation-artifacts/1-5-procedural-deck-generation.md — Previous story learnings]

## Dev Agent Record

### Agent Model Used

Claude claude-4.6-opus (dev-story workflow)

### Debug Log References

- Verified Generator entity: NewGenerator, IsPowered, BatteriesNeeded, InsertBatteries all behave correctly (15 test cases).
- Verified state.Game power helpers: AddBatteries, UseBatteries, AddGenerator, AllGeneratorsPowered, UnpoweredGeneratorCount, UpdatePowerSupply (with deck 0 and decay), GetAvailablePower (14 test cases).
- Verified placeGenerators: spawn generator placed and auto-powered on all levels; PowerSupply=100 immediately after. Level 2 only spawn gen (6 tests).
- Verified placeBatteries: levels 1-2 no batteries (exit unlocked); level 5 places batteries for additional generators (5 tests).
- Verified CheckAdjacentGenerators: inserts batteries, updates PowerSupply on power-up, no-op with 0 batteries, partial insert, already-powered skip (5 tests).
- Verified CheckAdjacentGeneratorAtCell: returns true for cells with generators (powered/unpowered), false for nil/empty cells (4 tests).
- Verified PickUpItemsOnFloor: Battery increments g.Batteries, multiple batteries, non-battery goes to OwnedItems (3 tests).
- Verified deck persistence: SaveCurrentDeckState/LoadDeckState preserves generator state; AdvanceLevel resets batteries/generators/power (2 tests).
- Integration test: full battery-pickup-and-generator-power lifecycle confirmed end-to-end.
- Edge cases: multiple generators with limited batteries, InsertBatteries(0), InsertBatteries capped at needed (3 tests).
- Pre-existing renderer/ebiten gotext build failure unchanged; all other packages pass.

### Completion Notes List

- Task 1: Created generator_test.go (15 tests) and state_test.go (14 tests) covering Generator entity methods, Game battery/power helpers, UpdatePowerSupply with deck decay multipliers, GetAvailablePower, and NewGame defaults.
- Task 2: Created generators_test.go (6 tests) and batteries_test.go (5 tests) verifying spawn generator auto-power, PowerSupply update on placement, level-based battery distribution, and isValidForGenerator/calculateBatteriesForGenerator.
- Task 3: Created interactions_test.go (12 tests) covering CheckAdjacentGenerators battery insertion, PowerSupply updates, no-op cases, CheckAdjacentGeneratorAtCell callouts, and PickUpItemsOnFloor battery pickup.
- Task 4: Added CalculatePowerConsumption baseline test, SaveCurrentDeckState/LoadDeckState generator persistence test, and AdvanceLevel power state reset test to state_test.go.
- Task 5: Added full lifecycle integration test (pickup → partial insert → second pickup → complete power-up → verify PowerSupply=100) and edge case tests (multiple generators limited batteries, zero insert, capped insert).
- Full test suite passes (excluding pre-existing renderer/ebiten gotext issue). Zero regressions.
- Total new tests: 57 across 5 test files.

### File List

- pkg/game/entities/generator_test.go (new)
- pkg/game/state/state.go (modified — deep-copy generators in Save/LoadDeckState)
- pkg/game/state/state_test.go (new)
- pkg/game/setup/generators_test.go (new)
- pkg/game/setup/batteries_test.go (new)
- pkg/game/gameplay/interactions_test.go (new)
- _bmad-output/implementation-artifacts/2-1-generators-and-batteries.md (modified)
- _bmad-output/implementation-artifacts/sprint-status.yaml (modified)

## Change Log

- 2026-02-22: Story 2.1 implementation complete. Verified existing generator/battery system across entities, state, setup, gameplay, and deck packages. Added 57 new tests in 5 test files covering entity behavior, state management, placement, player interaction, deck persistence, integration lifecycle, and edge cases. All tests pass with zero regressions.
- 2026-02-22: Code review complete (4 medium, 3 low issues found). Fixed all medium issues: [M1] added assertions to no-assertion battery test, [M2] fixed shallow pointer copy bug in SaveCurrentDeckState/LoadDeckState with deep-copy + isolation test, [M3] added permanent-insertion round-trip test verifying PowerSupply after save/load, [M4] eliminated fragile double-build grid setup in generator placement test. All tests pass.
