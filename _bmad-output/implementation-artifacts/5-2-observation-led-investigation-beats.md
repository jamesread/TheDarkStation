# Story 5.2: Observation-Led Investigation Beats

Status: done

<!-- Validation optional: run validate-create-story before dev-story if desired. -->

## Story

As a player,
I want some progression beats to begin with **what I notice in the world** before I rely on menus,
so that puzzles feel investigative rather than “open terminal → flip bit → door opens.”

## Acceptance Criteria

1. **Given** at least one puzzle-bearing deck configuration supported by deterministic generation rules (tiered when needed),
   **when** I encounter a gated interaction relevant to an observation-led beat,
   **then** there exists a **documented** beat path where **spatial / map / diegetic-world cues** (including Story 5.1 environmental layer where applicable—plaques, room naming, correlated labels) reasonably narrow hypotheses **before or alongside** opening a terminal or maintenance menu—and the existing **solve path** stays fair and reachable (**Epic 3** solvability, no unreachable codes or deadlocks introduced).

2. **Hypothesis → verify:** the beat still honours the current mechanics (e.g. codes in furniture → `FoundCodes`; puzzle terminals in `pkg/game/levelgen/puzzles.go` + `CheckAdjacentPuzzlesAtCell`). Any new prerequisite must remain **replayable**, **deterministic** where procedural, and tested on a pinned seed where feasible.

3. **Tiering:** early decks (**onboarding / low depth**) may omit or simplify observation-led arcs; richer beats activate only when **`CurrentDeckID`**, functional layer (`deck.FunctionalType`), or a **documented tier flag** meets the threshold so difficulty scales without surprising new players.

4. **Scope boundary (explicit):**
   - **In scope:** observation-first structuring of at least **one representative beat** tied to generation + UX (callouts, plaques, puzzle/furniture/copy alignment).
   - **Out of scope for 5.2:** multi-hop inference chains (**Story 5.3**) and layered maintenance-terminal instrumentation (**Story 5.4**). Do not build cross-room correlate graphs or new terminal “log strata” UI here beyond light copy/order tweaks if essential.

## Tasks / Subtasks

- [x] **Beat specification** (AC: #1–2): Author a short **`specs/` or `_bmad-output/`** note naming the archetype (“observation-led puzzle choke”), prerequisites, cues (which env plaques / room identifiers / hints), verification step (terminal or existing interaction), and **example seed(s)** or test grid for QA.
- [x] **Generation / data** (AC: #2–3): Extend `pkg/game/levelgen/puzzles.go` (and/or correlated placement helpers) so at least **one tier** aligns puzzle/furniture/environment markers—e.g. solution or hint fingerprints echoed in **`GameCellData.EnvPlaqueMsgID`** taxonomy (`ENV_PLAQUE_*`) or deterministic furniture description prefixes—without breaking **`PlacePuzzles`** / **`EnsureSolvability*`** ordering.
- [x] **Player-facing feedback** (AC: #1): Where the player discovers a cue **before** the terminal, reinforce with **thin** callouts or log messages (reuse **`renderer.FormatText` / markup** patterns; Epic 4 dry tone)—avoid narrator dumps (NFR1).
- [x] **Tests** (AC: #2–3): Table-driven tests: given fixed RNG / small grid fixture, assert expected correlation (presence of cue field + puzzle solution relation, or “tier off” decks below threshold)—use patterns from **`pkg/game/setup/environment_test.go`**.
- [x] **Docs** (AC: #4): Comments at touch points cite this story ID and **`specs/level-layout-and-solvability.md`** invariants—no regressions on keycard/hazard/generator chains.

## Dev Notes

### Developer context — what exists today

- **Environmental layer (5.1):** `GameCellData.EnvPlaqueMsgID`, `setup.ApplyEnvironmentalSignage`, signage keys in **`pkg/game/deck/environment.go`**, rendering via **`computeEnvPlaques` / `drawEnvironmentalPlaques`** (`pkg/game/renderer/ebiten/`). Prefer **reuse** before inventing a parallel cue system.
- **Puzzle flow:** `PlacePuzzles` → terminals with `Solution` / `Hint`; player gains codes via **`CheckForPuzzleCode`** on **`Furniture.Description`** (`interactions.go`); **`Game.FoundCodes`** gates solve in **`CheckAdjacentPuzzlesAtCell`** (`interactions.go`).
- **Gated movement:** Doors, hazards, generators—unchanged semantics; observation beats sit **above** substrate, never violate **`CanEnter`** / reachability tooling (`pkg/game/setup/room_connectivity.go`, **`EnsureSolvabilityDoorPower`**).
- **AGENTS.md:** Lift forward-only policy unchanged; plaques already visibility-gated (`visited`/discovered rules from 5.1).

### Technical requirements / guardrails

- **Reuse** gettext for any new strings (`po/default.pot`, **`make mo`**); preserve cold technical tone (**Epic 4**).
- **Do not** add HTTP, external assets, or a second plaque pipeline—stay within grid + text renderer.
- **Correlation** hints must survive **overload / power readability** (**Epic 2**): if a cue is power-dependent rooms, clarify in copy or gates so blackout does not silently hide mandatory info (or restrict beats to cues visible without extra toggles unless documented).
- **Determinism:** any shuffle or slot choice must derive from **`g.LevelSeed`/RNG plumbing** consistent with **`ApplyEnvironmentalSignage`** (pinned seed tests).

### Architecture compliance

| Topic | Requirement | Source |
|-------|-------------|--------|
| Stack | Go 1.24, Ebiten v2 game loop | `docs/architecture.md` |
| Layout | Logic `pkg/game/`, renderer `pkg/game/renderer/` + `ebiten/` | same |
| State | In-memory only; **`GameCellData`** on grid | same |
| i18n | gotext keys | Architecture dev workflow |

### Library / framework requirements

- Stick to **`github.com/hajimehoshi/ebiten/v2`** and existing **`darkstation`** module deps; no new gameplay frameworks.

### File structure (likely touch surfaces)

| Area | Files |
|------|-------|
| Gen / placement | `pkg/game/levelgen/puzzles.go`, optionally `furniture.go`, `pkg/game/setup/lifecycle.go` call order |
| World data | `pkg/game/world/cell.go` (only if a **minimal new optional field** is unavoidable—prefer reusing plaques + puzzle metadata) |
| Gameplay UX | `pkg/game/gameplay/interactions.go`, `hints.go` (thin hints—not full 5.3 chains) |
| Renderer | Only if cue needs subtle map affordance; prefer plaques before new HUD |
| Entities | `pkg/game/entities/puzzle.go`, `furniture.go` (copy/hint consistency only if needed) |
| Spec | `specs/` or `_bmad-output/implementation-artifacts/` beat note |

### Testing requirements

- `go test ./...`, `make codestyle` mandatory before marking **done**.
- Add **focused** `_test.go` near levelgen/setup; avoid flaky unseeded RNG.
- Optional **`devtools`/mapdump** seed documentation for QA handoff.

### Previous story intelligence (5.1)

- **Environmental plaques** wired per **`deck.FunctionalType`** with gettext keys **`ENV_PLAQUE_*`**; visibility follows discovered/visited semantics.
- **Furniture fallback** via **`entities.FurnitureFallbackForFunctionalLayer`** bridged thematic BSP room names vs legacy **`RoomFurniture`** keys—reuse when tying examine text to cues.
- **Renderer:** semantic markup (`parseMarkup`), focus plates (`focusPlateForForeground`)—extend patterns, avoid one-off globals.
- **Change log:** see **`5-1-environmental-legibility-and-diegetic-detail.md`** “Change Log” and file list (`environment.go`, `setup/environment.go`, `rendering.go` plaques).

### Git intelligence summary

- Latest work on **`d944d2d`** (**environmental signage & renderer polish**): plaques, gettext, magenta focus plates, pickup/markup cohesion—implement 5.2 as **continuation** (correlated diegetic cues + puzzles), not a parallel UI experiment.
- **`5293885`**: Semantic focus plates—player already reads map semantics; cues should reinforce without clutter.

### Latest tech notes

- **Go 1.24.0**, **Ebiten v2.9.7** per `docs/architecture.md`; no mandated version bump for text-only/feature-flagged logic.

### Project context reference

- No **`project-context.md`** in repo root; rely on **`AGENTS.md`**, **`specs/environmental-signage.md`**, this story, **`epics.md`**.

### Story completion status

- **Status:** done
- **Note:** Observation-led corridor stamps + first-visit thin callouts implemented; puzzles and solvability chain unchanged (`FoundCodes`). Ready for **`code-review`**.

---

## Change Log

- **2026-05-17:** Observation tier (`level >= 3`), `deck.ObservationSeqPlaqueMsgID`, `setup.ApplyObservationLedPuzzleCues`, movement `maybeAnnounceObservationCueOnMove`, gettext `ENV_PLAQUE_OBS_SEQ_*`, spec `specs/observation-led-puzzle-beat.md`, tests in `deck` and `setup`.

### Questions saved for backlog / refinement

1. Consider **`deck.FunctionalType`** as secondary tier signal alongside **`level`** (Story 5.3 alignment).
2. Whether observation arcs should subtly reuse **cctv/maintenance pings**.

---

## Dev Agent Record

### Agent Model Used

Cursor Agent — BMAD dev-story workflow

### Debug Log References

- `go test ./...`, `make codestyle`, `make mo` — pass

### Completion Notes List

- Tiering: **`deck.ObservationLedPuzzleCuesActive`** uses **`level >= 3`** (1-based); **`minimalSystems`** (final deck) skips fingerprints.
- **First** qualifying row-major **`PuzzleSequence`** with **`ObservationSeqPlaqueMsgID`** drives the corridor stamp remap; **`FoundCodes`** / furniture **`Code:`** unchanged.
- **`ObservationCueVisited`** key **`deckID:row:col`**; **`ResetObservationCueAnnounced`** on **`SetupLevel`** and **`LoadDeckState`**.

### File List

- `pkg/game/deck/observation.go`
- `pkg/game/deck/observation_test.go`
- `pkg/game/setup/observation.go`
- `pkg/game/setup/observation_test.go`
- `pkg/game/gameplay/lifecycle.go`
- `pkg/game/gameplay/movement.go`
- `pkg/game/gameplay/observation_cue.go`
- `pkg/game/gameplay/interactions.go`
- `pkg/game/levelgen/puzzles.go`
- `pkg/game/state/state.go`
- `po/default.pot`
- `mo/en_GB.utf8/LC_MESSAGES/default.mo`
- `specs/observation-led-puzzle-beat.md`
