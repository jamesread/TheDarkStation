# Story 5.1: Environmental Legibility & Diegetic Detail

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a player,
I want the station’s spaces and fixtures to imply history, function, and relationships between systems **outside** terminal menus alone,
so that exploration feels like understanding a place—not hopping between disconnected toggle screens.

## Acceptance Criteria

1. **Given** a generated deck (BSP/placement pipeline unchanged at **architectural** level—still `BSPGenerator.Generate`, carve/connect, then placement/setup),
   **when** I move through rooms and corridors before solving the deck,
   **then** recurring motifs, signage-like labels, fixtures, or placement semantics reinforce **what systems exist** and **how areas relate** (subsystem zones, redundant paths, evidence of prior overrides) using **existing grid/text/renderer vocabulary** (no new art pipeline).

2. **Implicit narrative only (NFR1):** environmental additions are technical residue—labels, stamps, stale notices—not narrator voice or lore exposition dumps.

3. **Naming coherence (Epic 4):** copy stays functional, cold, slightly outdated; life-support contexts remain irreparable/OFFLINE where referenced (do not imply restoration gameplay).

4. **Determinism / QA:** behaviour tied to deck functional layer (`deck.FunctionalType(level)`) and/or deterministic placement seed must be documented so tests can assert presence of representative environmental cues on a fixed level seed.

5. **Scope boundary:** this story delivers **environmental legibility** only—do not implement full observation→puzzle gates here (defer causal puzzle beats to Story **5.2**); shallow hooks that future stories can reference are acceptable.

## Tasks / Subtasks

- [x] **Design vocabulary** (AC: #1–3): Define a small catalogue per `deck.Type` (e.g. Habitation … CoreInfrastructure) of **subsystem codes**, corridor plaque strings, or repeating motifs—gettext keys in `po/default.pot`, cold/system tone.
- [x] **Data model** (AC: #1, #4): Add minimal structured attachment for “environmental markers” (preferred: thin field on `GameCellData` or parallel map keyed by cell coords—avoid bloating engine `Cell` unless justified). Markers must be serializable in-memory only (existing architecture).
- [x] **Placement** (AC: #1, #4): Deterministic pass after BSP carve/connect (likely `pkg/game/setup/` or `pkg/game/levelgen/`) assigns markers to corridor junctions, room thresholds, or sparse interior cells—bounded count per deck so readability stays high.
- [x] **Rendering** (AC: #1–3): Extend Ebiten snapshot/render path (`pkg/game/renderer/ebiten/`) to draw secondary labels distinct from `LOCATION{room}` banners—reuse markup/colors patterns (`parseMarkup`, subtle vs action) so signage reads as **diegetic**, not HUD objectives.
- [x] **Optional furniture alignment** (AC: #3): Audit `entities.RoomFurniture` keyed by legacy names vs BSP thematic room names (`deck.RoomNamesForType`); bridge or phase-key templates so examined furniture flavour matches functional deck identity without breaking placement.
- [x] **Tests** (AC: #4): Table-driven unit tests for marker assignment rules per functional type OR snapshot/helper asserting non-empty environmental layer on a pinned RNG seed / fixed small grid.
- [x] **Docs** (AC: #4): Short comment block or `_bmad-output`/spec note listing marker taxonomy keys and placement rules for QA/regression.

## Dev Notes

### Architecture compliance

- Single Go binary; game logic under `pkg/game/`; renderer abstraction under `pkg/game/renderer/` with Ebiten in `renderer/ebiten/` [Source: `docs/architecture.md`].
- No HTTP API; state in-memory (`Game`, grid, `GameCellData`) [Source: `docs/architecture.md#Data Architecture`].
- i18n via gotext (`gotext.Get`, keys in `po/default.pot`, `make mo`)—environment strings must go through gettext like existing terminal strings.

### Where generation & identity live today

- Deck functional layer cycles via `deck.FunctionalType(level)`; thematic room naming via `deck.RoomNamesForType(ft)` consumed by BSP naming (`pkg/game/generator/bsp.go`).
- Maintenance terminal flavour bands already vary by deck depth (`deck.TerminalFlavourKey`)—environmental signage should **feel consistent** with that layering without duplicating terminal menus verbatim.

### Renderer hotspots

- Room banners use `LOCATION{}` markup and `drawRoomLabels` (`pkg/game/renderer/ebiten/rendering.go`).
- Recent renderer work emphasized semantic clarity (`feat(renderer): semantic focus plates for map cells`)—extend patterns rather than inventing unrelated UI chrome.

### Risks / pitfalls

- **Furniture templates** (`entities.RoomFurniture`) still keyed partly by legacy room-type strings; BSP emits thematic compound names—mapping mismatch may mute Epic 4 examine flavour until bridged (called out in tasks).
- Keep marker density low—text-heavy maps undermine minimal aesthetic and readability (NFR7 desktop performance is trivial but UX clutter is not).

### Testing standards

- `go test ./...` / `make test`; place tests beside code (`*_test.go`).
- Prefer deterministic seeds over flaky RNG in placement tests.

### Project Structure Notes

- Touch surfaces likely: `pkg/game/world/cell.go` (`GameCellData`), `pkg/game/setup/` or `pkg/game/levelgen/`, `pkg/game/renderer/ebiten/{snapshot,rendering,text}.go`, `pkg/game/deck/deck.go` (taxonomy helpers only—avoid coupling renderer to deck package if layering forbids; pass strings from setup).

### References

- Story definition & Epic 5 preamble: `_bmad-output/planning-artifacts/epics.md` (§ Epic 5, Story 5.1).
- FR26 scope honesty & constraints: `_bmad-output/planning-artifacts/epics.md` (FR26).
- NFR1 implicit narrative: `_bmad-output/planning-artifacts/epics.md` (NFR1).
- Architecture overview: `docs/architecture.md`.
- BSP + thematic naming: `pkg/game/generator/bsp.go`, `pkg/game/deck/deck.go`.
- Room labels rendering: `pkg/game/renderer/ebiten/rendering.go` (`drawRoomLabels`).
- AGENTS.md revisit/lift policy remains unchanged—environment is per-deck legibility only.

## Git Intelligence Summary

Recent commits emphasize renderer UX polish suitable for layering diegetic text:

- `5726f92` — objectives panel anchoring (layout discipline).
- `5293885` — semantic focus plates for map cells (pattern for visually distinct semantic layers).

Prefer extending established renderer semantics rather than ad-hoc global overlays.

## Latest Tech Notes

- **Go 1.24**, **Ebiten v2.9.7** per `docs/architecture.md` / `technology-stack.md`—no upgrade required for text overlays; use existing font measurement helpers (`text.Measure`, `drawColoredText` baseline rules per `AGENTS.md`).

## Previous Story Intelligence

Epic 4 story markdown files are not present under `_bmad-output/implementation-artifacts/` for `4-x`; completion was tracked in `sprint-status.yaml` only. Treat **Story 4.4** acceptance (final deck, docking completion overlay strings such as `ENERGY_GRADIENT_EQUALIZED`) as validated in codebase (`drawCompletionOverlay` in `rendering.go`)—environmental copy must remain stylistically aligned with completion tone (technical, finality) without preempting finale beats.

## Project Context Reference

- Optional: `**/project-context.md` — not required if absent.

## Story Completion Status

- **Status:** done
- **Note:** Corridor junction plaques + functional-layer furniture fallback implemented; Epic 5 closed in sprint tracking after review.

## Questions / Clarifications (non-blocking)

1. Should subsystem plaques appear **only when adjacent** vs **persistent at corridor hubs** (performance/readability vs mystique)?
2. Exact gettext namespace convention (`ENV_*`, `PLAQUE_*`)—align with existing key naming in `po/default.pot`.

## Change Log

- **2026-05-16:** Implemented environmental plaques (`GameCellData.EnvPlaqueMsgID`), `deck.EnvironmentalPlaqueKeys`, `setup.ApplyEnvironmentalSignage` (junction rule, seeded shuffle), Ebiten rendering (`computeEnvPlaques`, `drawEnvironmentalPlaques`), gettext strings in `po/default.pot`, `specs/environmental-signage.md`, furniture fallback per functional layer (`entities.FurnitureFallbackForFunctionalLayer`). Tests: `pkg/game/setup/environment_test.go`.

## Dev Agent Record

### Agent Model Used

Composer (BMAD dev-story workflow)

### Debug Log References

- `go test ./...` — pass
- `make codestyle` — pass

### Completion Notes List

- Corridor signage uses gettext keys `ENV_PLAQUE_*` per `deck.Type`; Habitation includes explicit non-restorable life-support line (`ENV_PLAQUE_HAB_LIFE_BUS`).
- Visibility gated on visited/discovered cells so fog-of-war stays coherent.
- Furniture fallback ensures thematic rooms still receive cold examine copy when legacy `RoomFurniture` substring keys miss BSP compound names.

### File List

- `pkg/game/world/cell.go`
- `pkg/game/deck/environment.go`
- `pkg/game/setup/environment.go`
- `pkg/game/setup/environment_test.go`
- `pkg/game/gameplay/lifecycle.go`
- `pkg/game/entities/deck_furniture.go`
- `pkg/game/levelgen/furniture.go`
- `pkg/game/renderer/ebiten/constants.go`
- `pkg/game/renderer/ebiten/types.go`
- `pkg/game/renderer/ebiten/snapshot.go`
- `pkg/game/renderer/ebiten/rendering.go`
- `po/default.pot`
- `specs/environmental-signage.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/5-1-environmental-legibility-and-diegetic-detail.md`
