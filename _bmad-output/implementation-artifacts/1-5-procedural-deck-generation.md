# Story 1.5: Procedural Deck Generation

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a player,
I want each deck to be generated when I first enter it and the station to have a fixed number of decks,
so that the station feels finite and the end is real.

## Acceptance Criteria

1. **Given** I advance to a new deck
   **When** I enter it for the first time
   **Then** that deck is generated (e.g. BSP + placement)
   **And** the station has a fixed number of decks (no infinite descent)
   **And** the final deck exists and is reachable

## Tasks / Subtasks

- [x] Task 1: Verify and document “generate on first entry” (AC: #1)
  - [x] 1.1 Confirm AdvanceLevel generates next deck when DeckStates[nextID] has no grid; document flow in Dev Notes
  - [x] 1.2 Confirm BuildGame generates only the starting deck (no pre-generation of all decks)
  - [x] 1.3 If any path generates a deck without going through a single “first entry” path, refactor so generation is consistent
- [x] Task 2: Fixed deck count and final deck (AC: #1)
  - [x] 2.1 Confirm deck.TotalDecks and deck.Graph define fixed count; final deck has empty Connections (no next deck)
  - [x] 2.2 Confirm exit on final deck triggers TriggerGameComplete (lift has no destination); no advance option shown
  - [x] 2.3 Add or update tests: fixed deck count, final deck reachable, no generation beyond TotalDecks
- [x] Task 3: Integration and validation (AC: #1)
  - [x] 3.1 Run full test suite; ensure no regressions
  - [x] 3.2 Optional: devtools or test that advances through decks and asserts each is generated once and final deck completes

## Dev Notes

- **FR10 (GDD):** Procedural generation; each deck generated when first entered; fixed number of decks; end is real.
- **Current flow:** `gameplay.BuildGame(startLevel)` generates the starting deck only (GenerateGrid + SetupLevel). `gameplay.AdvanceLevel` saves current deck to DeckStates, then for next deck: if DeckStates[nextID] has Grid, loads it; else generates on first entry (GenerateGrid(g.Level) + SetupLevel(g)) and saves to DeckStates. So “generate when first enter” is already implemented for advancing; starting deck is generated at game start.
- **Deck model:** `pkg/game/deck`: TotalDecks (10), FinalDeckIndex, Graph (linear 0→1→…→FinalDeckIndex), NextDeckID(deckID), IsFinalDeck(level). Final deck has Graph[i].Connections == nil/empty.
- **State:** `state.Game` has CurrentDeckID (0-based), Level (1-based), DeckStates map[int]*DeckState. SaveCurrentDeckState / LoadDeckState for per-deck persistence (used for reset and for “revisit” data; UI is forward-only per GDD).
- **Generation:** `generator.DefaultGenerator` is BSP; `generator.Generate(level)` returns Grid; `gameplay.SetupLevel(g)` runs placement (hazards, furniture, puzzles, maintenance), EnsureSolvabilityDoorPower, InitMaintenanceTerminalPower, then moves player to start cell. See Story 1.4 for R8/I7 placement rules.
- **Final deck:** When player reaches exit on final deck, TriggerGameComplete(g) sets GameComplete and shows completion message; lift must not offer “advance” (no next deck). Check menu/renderer for lift/exit UI on final deck.

### Project Structure Notes

- **pkg/game/deck:** deck count, graph, FinalDeckIndex, NextDeckID, IsFinalDeck, FunctionalType (for naming/decay).
- **pkg/game/gameplay/lifecycle.go:** BuildGame, AdvanceLevel, ResetLevel, TriggerGameComplete; GenerateGrid, SetupLevel.
- **pkg/game/generator:** BSP, LineWalker; DefaultGenerator.Generate(level).
- **pkg/game/state/state.go:** DeckStates, CurrentDeckID, Level, SaveCurrentDeckState, LoadDeckState.
- **pkg/game/levelgen, pkg/game/setup:** Used by SetupLevel (see Story 1.4).

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 1, Story 1.5]
- [Source: docs/architecture.md — deck-based progression, state, no DB]
- [Source: GDD FR10 — procedural generation, fixed decks]
- [Source: AGENTS.md — Revisit policy: forward-only lift, no return; per-deck state stored but UI does not expose revisit]

## Dev Agent Record

### Agent Model Used

dev-story workflow (workflow.xml + dev-story instructions.xml)

### Debug Log References

- Verified AdvanceLevel (lifecycle.go:145-172): when DeckStates[nextID] is nil or has no Grid, generates via GenerateGrid+SetupLevel; otherwise LoadDeckState.
- Verified BuildGame (lifecycle.go:24-48): generates only startLevel deck; no iteration over decks.
- Verified main.go: IsFinalDeck → TriggerGameComplete; non-final deck → exit animation then AdvanceLevel. state.AdvanceLevel is legacy/unused (gameplay.AdvanceLevel used).
- deck.Graph[FinalDeckIndex].Connections is empty; NextDeckID returns false on final deck.

### Completion Notes List

- Task 1: Confirmed AdvanceLevel generates on first entry when DeckStates[nextID] has no grid. BuildGame generates only starting deck. No other paths generate decks (ResetLevel resets same deck; state.AdvanceLevel unused). Documented flow in Completion Notes.
- Task 2: Confirmed TotalDecks(10), Graph linear 0→…→9, final deck empty Connections. main.go: IsFinalDeck→TriggerGameComplete, non-final→AdvanceLevel after animation. Added lifecycle_test.go: fixed count, final deck reachable, AdvanceLevel load vs generate, AdvanceThroughAllDecks.
- Task 3: pkg/game/gameplay, generator, levelgen, setup tests pass. Ebiten renderer has pre-existing gotext build issues (unchanged).

### File List

- pkg/game/gameplay/lifecycle_test.go
- _bmad-output/implementation-artifacts/sprint-status.yaml

## Senior Developer Review (AI)

- **Review:** Code review (code-review workflow); MEDIUM/LOW issues addressed.
- **Git vs story:** lifecycle_test.go was untracked; staged for commit. Story File List matched implementation.
- **Fixes applied:** (1) MEDIUM: Added TestBuildGame_ClampsStartLevelToValidRange (BuildGame boundary clamping). (2) MEDIUM: Added TestResetLevel_DoesNotAdvanceDeck (ResetLevel does not advance deck). (3) MEDIUM: Comment on TestAdvanceThroughAllDecks noting 10 full generations / -race slowness. (4) lifecycle_test.go staged so it can be committed for CI.

## Change Log

- 2026-02-21: Story 1.5 implementation complete. Verified generate-on-first-entry flow; added lifecycle tests for BuildGame, AdvanceLevel, deck graph, final deck, TriggerGameComplete.
- 2026-02-21: Code review. Fixes: TestBuildGame_ClampsStartLevelToValidRange, TestResetLevel_DoesNotAdvanceDeck; comment on TestAdvanceThroughAllDecks; lifecycle_test.go staged for commit.
