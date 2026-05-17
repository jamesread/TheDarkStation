# Story 5.3: Multi-Hop Inference & Cross-Room Linkage

Status: done

<!-- Validation optional: run validate-create-story before dev-story if desired. -->

## Story

As a player,
I want clues and consequences to occasionally span **multiple locations or readings** before I commit power or routing choices,
so that “aha” moments come from linking fragments—not single-room toggling.

## Acceptance Criteria

1. **Given** a deck tier where multi-hop arcs are enabled (documented threshold—stricter than Story 5.2’s **`level >= 3`** fingerprint tier so onboarding stays approachable),
   **when** generation places the **represented archetype** for this story,
   **then** solving or committing power **requires resolving at least two inferential hops** (e.g. identifier **A** in space **B** plus constraint **C** from a different readable surface in another reachable context before the payoff—puzzle admit, routing insight, or equivalent existing interaction).

2. **Cross-room / multi-reading linkage** is encoded in **data** (minimal fields + deterministic placement rules), not only prose hand-waving—QA can reproduce correlation on pinned **`LevelSeed`** or small synthetic grid fixtures.

3. **Epic 3 / solvability:** no deadlock; every clue surface needed for admissibility remains **reachable without circular equipment dependency** (`specs/level-layout-and-solvability.md`; `EnsureSolvabilityDoorPower` + keycard/hazard/generator chains unchanged).

4. **Epic 2 / power readability:** multi-hop UX must not hide overload, passive short-out messaging, or `GetAvailablePower()` feedback players already rely on—restrict beats to surfaces visible under normal exploration or document power prerequisites explicitly.

5. **Scope boundary (explicit):**
   - **In scope:** at least **one archetype** spanning **multiple rooms OR multiple distinct authored readings** (e.g. corridor stamp + CCTV/furniture/menu text + puzzle hint).
   - **Out of scope for 5.3:** Story **5.4** layered maintenance-terminal “instrument strata” UX (logs/alarms as first-class submenu system). Lightweight **read-only** copy additions to existing plaintext surfaces OK if tied to linkage tokens.

## Tasks / Subtasks

- [x] **Archetype spec** (AC: #1–2): Add **`specs/multi-hop-linkage-archetype.md`**—define hop graph (H1→H2→commit), linkage token convention, tier rule, QA seed notes, rollback if fewer than two corridors/rooms qualify.
- [x] **Correlation model** (AC: #2–3): Introduce minimal **shared token/tag** usable by plaques (`EnvPlaqueMsgID`/gettext interpolation pattern), **`PuzzleTerminal` hint/description**, **`Furniture.Description`**, and/or CCTV/maintenance plaintext—prefer `GameCellData` or entity fields already serialised in **`DeckState.Grid`** rather than orphaned globals.
- [x] **Generation** (AC: #1–3): Deterministic **`levelgen`/setup** pass (tier gated) assigns correlated strings across **distinct rooms** respecting **R8 connectivity** (`room_connectivity`); document ordering relative to **`PlacePuzzles`** / **`ApplyEnvironmentalSignage`** / **`ApplyObservationLedPuzzleCues`** (avoid stamp overwrite races—integrate consciously).
- [x] **Player feedback** (AC: #4): Optional thin log/callouts when second hop surface is first read (reuse markup; no narrator dumps—NFR1).
- [x] **Tests** (AC: #2): Table tests on synthetic grid + mapping tests for tier off/on; no flaky unseeded RNG.
- [x] **Comments** (AC: #3): Touch points cite **Story 5.3** + solvability spec; no silent bypass of **`CanEnter`** or generator/hazard proofs.

## Dev Notes

### Foundation from 5.1 / 5.2 (do not fight)

| Layer | Artifact | Notes |
|-------|----------|-------|
| 5.1 | `setup.ApplyEnvironmentalSignage`, **`ENV_PLAQUE_*`** | Junction plaques; deterministic shuffle from **`plaqueSeed`**. |
| 5.2 | `deck.ObservationLedPuzzleCuesActive`, `ApplyObservationLedPuzzleCues` | Retargets **one** plaque to **`ENV_PLAQUE_OBS_SEQ_*`** for earliest mapped sequence puzzle — **coordinate** multi-hop stamping so tiers do not blindly overwrite linkage plaques. |

### Puzzle / code substrate (reuse)

- **Furniture + `FoundCodes` + `CheckAdjacentPuzzlesAtCell`** (`pkg/game/gameplay/interactions.go`) stays solve authority until you explicitly extend puzzle schema.
- **CCTV** reveal / terminal strings—read-only augmentation only if linkage needs a “reading” surface (watch **power gating** per room).

### Architecture compliance

- Go 1.24, Ebiten v2, in-memory state, gettext for copy (`po/default.pot`, `make mo`) [Source: `docs/architecture.md`].
- No HTTP; keep logic in `pkg/game/{levelgen,setup,gameplay,entities,world}` and renderer only if map affordance is essential.

### Proposed implementation vectors (choose one for v1)

1. **Token bridge:** short **alphanumeric bus label** (e.g. `RELAY-φ7`) embedded in **two** gettext-backed strings (plaque + hint or furniture line) + optional third hop in puzzle `Hint` after first code fragment found.
2. **Split code:** two furniture fragments in different rooms that **`CheckForPuzzleCode`** merges into `FoundCodes` only when both seen (requires careful change to `state.Game` + extractors—higher risk; prefer token + single `Code:` if split is too invasive for one story).

### File structure (expected touch)

| Area | Files |
|------|-------|
| Tier / tokens | `pkg/game/deck/` (new small helper) or `pkg/game/levelgen/` |
| Placement | `pkg/game/levelgen/puzzles.go`, `furniture.go`, `pkg/game/setup/*.go` |
| State | `pkg/game/state/state.go` / `pkg/game/world/cell.go` only if minimal |
| Gameplay | `interactions.go`, optional `hints.go` |
| Specs | `specs/multi-hop-linkage-archetype.md` |

### Testing requirements

- `go test ./...`, `make codestyle` before **review**.
- Prefer **synthetic grid** tests like `pkg/game/setup/observation_test.go` patterns.

### Previous story intelligence (5.2)

- Tiering precedent: **`level >= 3`** for observation stamps; **5.3 should use a higher bar** (e.g. `level >= 5` or `CurrentDeckID >= 4`) unless product says otherwise—**document in spec**.
- **`ObservationCueVisited`** / one-shot callouts—mirror sparingly for second-hop surfaces if needed.
- Files delivered: see **`5-2-observation-led-investigation-beats.md`** File List (`observation.go`, `observation_cue.go`, `specs/observation-led-puzzle-beat.md`).

### Git intelligence

- Recent Epic 5 work clusters on **renderer polish + environmental/observation**—extend **generation + data** for 5.3 rather than new HUD systems.

### Latest tech notes

- Ebiten v2.9.7; no version bump required for text/RNG placement work [Source: `docs/architecture.md`].

### Project context reference

- **`AGENTS.md`** (lift forward-only); **`specs/observation-led-puzzle-beat.md`**; **`specs/environmental-signage.md`**; **`specs/level-layout-and-solvability.md`**.

### Story completion status

- **Status:** done
- **Note:** Implemented tier `level >= 5`, token `LINK-MHOP-A`, keyed puzzle `2-4-6-8`; tests + `make codestyle` pass.

---

## Questions / Clarifications (non-blocking before dev-story)

1. Confirm **tier floor** (`level >= 5` vs `CurrentDeckID >= 4`): pick one authoritative rule here or in **`specs/multi-hop-linkage-archetype.md`**. **Resolved:** `level >= 5` (see spec).
2. Is **third hop** desirable in v1, or strict **minimum two hops** only? **Resolved:** minimum two hops for v1 (corridor `LinkageTag` + furniture `Code:`/`Relay:`).

---

## Dev Agent Record

### Agent Model Used

Composer (Cursor agent)

### Debug Log References

- `go test ./...`
- `make mo` / `make codestyle`

### Implementation Plan

- **Deck tier:** `deck.MultiHopLinkageActive` — `level >= 5`, not final deck (stricter than 5.2’s `level >= 3`).
- **Keyed terminal:** `deck.MultiHopKeyedSequenceSolution` aligned with `levelgen.PlacePuzzles` second numeric solution.
- **Placement:** `setup.ApplyMultiHopLinkage` after `ApplyObservationLedPuzzleCues`; chooses nearest junction plaque **excluding** `ENV_PLAQUE_OBS_*` so 5.2 retarget is not overwritten.
- **State:** `LinkageTokensSeen`, `LinkageCueVisited`; cleared on setup load/reset paths like other per-deck UI state.
- **Gameplay:** visit records `LinkageTag`; furniture parses `Relay:`; puzzle solve requires `HasLinkageToken` when `LinkageToken` set.

### Completion Notes List

- Added archetype spec, gettext `ENV_PLAQUE_LINK_MHOP_A`, synthetic tests for tier, placement, and puzzle admit gating.

### File List

- `specs/multi-hop-linkage-archetype.md`
- `pkg/game/deck/linkage.go`
- `pkg/game/deck/linkage_test.go`
- `pkg/game/setup/linkage.go`
- `pkg/game/setup/linkage_test.go`
- `pkg/game/gameplay/linkage_token.go`
- `pkg/game/gameplay/linkage_cue.go`
- `pkg/game/gameplay/linkage_interaction_test.go`
- `pkg/game/gameplay/lifecycle.go`
- `pkg/game/gameplay/movement.go`
- `pkg/game/gameplay/interactions.go`
- `pkg/game/state/state.go`
- `pkg/game/world/cell.go`
- `pkg/game/entities/puzzle.go`
- `pkg/game/levelgen/puzzles.go`
- `po/default.pot`
- `mo/en_GB.utf8/LC_MESSAGES/default.mo`
- `_bmad-output/implementation-artifacts/5-3-multi-hop-inference-and-cross-room-linkage.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

### Change Log

- **2026-05-16:** Story 5.3 — multi-hop linkage tier, gated puzzle admit, gettext + tests (`dev-story`).
