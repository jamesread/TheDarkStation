# Observation-led puzzle beat (Story 5.2)

## Archetype: **Structural stamp ↔ numeric sequence puzzle**

Goal: progression toward a gated **sequence** security terminal begins with something the player **reads on the deck**—a corridor junction “stamp”—before or alongside opening the puzzle UI. This stays within existing grid/markup tooling (Story 5.1 plaques) and the existing **furniture code → `FoundCodes` → puzzle** chain.

### Tier rule

Observation correlation runs only when **`deck.ObservationLedPuzzleCuesActive(level, minimalSystems)`** is true:

- `level >= 3` (1-based deck depth)
- not the deliberately minimal **final deck** systems pass (`minimalSystems == false`)

Decks **1–2** keep generic environmental plaques only; onboarding stays lighter.

### Data flow

1. `ApplyEnvironmentalSignage` assigns junction plaques (`ENV_PLAQUE_*`).
2. `ApplyObservationLedPuzzleCues` finds the **first** row-major **`PuzzleSequence`** whose solution maps via `deck.ObservationSeqPlaqueMsgID` (currently **`1-2-3-4`**, **`2-4-6-8`** — keep synced with `levelgen.PlacePuzzles`'s sequence entries).
3. Among cells that already bear a plaque, the **nearest Manhattan** corridor junction plaque is rewritten to the matching **`ENV_PLAQUE_OBS_SEQ_*`** msgid.

### Verification (hypothesis → confirm)

| Step | Mechanism |
|------|-----------|
| See stamp | Corridor tile micro-text ( gettext `dynamicGet`), visible when fog rules allow plaques. |
| First visit cue | Thin `TITLE{Structural stamp}` map callout (Story 5.2, one-shot via `ObservationCueVisited`). |
| Confirm | Furniture still carries **`Code:`** line extracted by existing `CheckForPuzzleCode`; terminal solve via `CheckAdjacentPuzzlesAtCell` unchanged. |

### Solvability

No change to **`EnsureSolvabilityDoorPower`**, keycard graphs, hazard reachability (`specs/level-layout-and-solvability.md`): only **cosmetic/display** cue on plaques already decorative.

### Regression / QA fixtures

- Pinned-setup tests: `pkg/game/setup/observation_test.go` (junction retarget vs early-level no-op).
- Mapping tests: `pkg/game/deck/observation_test.go`.

### Explicit non-goals (defer)

- Story **5.3** multi-hop inference graphs.
- Story **5.4** maintenance-terminal log strata UI.
