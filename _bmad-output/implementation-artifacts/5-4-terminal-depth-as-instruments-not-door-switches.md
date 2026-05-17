# Story 5.4: Terminal Depth as Instruments, Not Door Switches

Status: done

<!-- Validation optional: run validate-create-story before dev-story if desired. -->

## Story

As a player,
I want maintenance terminals to expose **layered, correlatable information**—logs, alarms, cross-references—beyond immediate toggles,
so that terminals feel like instrumentation embedded in a world rather than generic remote controls.

## Acceptance Criteria

1. **Powered-terminal baseline:** **Given** a maintenance terminal obeys Story **2.3** rules (power gating unchanged—see Epic 2 maintenance-terminal behaviour),
   **when** I open its menu while powered,
   **then** I can reach **secondary read-only strata** (diagnostic surfaces) distinct from door/CCTV/light toggles and device summaries already shown—implemented as **read-only** menu sections and/or a bounded submenu, not new gameplay verbs.

2. **Instrument semantics:** Strata include plausible **subsystem framing**—e.g. timestamp-like markers (synthetic/deterministic strings), **subsystem / circuit identifiers**, **alarm or fault-code lines**—written in Epic **4** tone (**technical, dry, non-personified** narrator per NFRs in `_bmad-output/planning-artifacts/epics.md`).

3. **Cross-check with FR26 surfaces:** Content **correlates with environmental labels and observational puzzle substrate** where FR26 arcs apply—identifiers or tokens **consistent with** plaques (`ENV_*` / observation tiers), optional **`LinkageTag`** / puzzle-adjacent copy from Story **5.3**, without duplicating 5.3’s multi-hop **placement** responsibilities (5.4 is **terminal UX + correlatable copy**, not new linkage archetypes).

4. **Browseability:** **Optional navigational grouping** (section headers, blank separators, or a shallow “Diagnostics” drill-down via existing `RunMenu` / `RunMenuDynamic`) improves scanability **without** dense HUD chrome—reuse **`InfoMenuItem`**, **`SUBTLE{}`** markup in labels where appropriate (`pkg/game/menu/maintenance.go`, `pkg/game/renderer/ebiten/menu.go`).

5. **Non-functional guardrails:** Generation/rendering stays **bounded**—no full log spam per tick; work happens when menu builds/refreshes (`RunMaintenanceMenu` → `RunMenuDynamic`). **Desktop performance** aligned with NFR7 ethos; **no networking**.

## Tasks / Subtasks

- [x] **Archetype spec** (AC: #1–3): Add **`specs/maintenance-terminal-instrument-strata.md`**—define strata types (log slice, alarm table, cross-ref lines), tier/deck rules, gettext msgid conventions, correlation rules with **`environment.go`** / **`observation.go`** / linkage tokens, QA fixtures or seeds.

- [x] **Data + synthesis** (AC: #2–3): Minimal deterministic helpers (likely under **`pkg/game/deck/`** or **`pkg/game/menu/`**) that compose diagnostic lines from existing **`Game`** / grid / deck ids—avoid orphaned globals; anything serialisable must ride **`DeckState`** / entities already saved if persistence matters.

- [x] **Menu integration** (AC: #1, #4): Extend **`MaintenanceMenuHandler.GetMenuItems()`** (`pkg/game/menu/maintenance.go`) with grouped read-only blocks **above or below** power toggles per spec; optional **`MenuHandler`** submenu for deep diagnostics—preserve **`MaintenanceRoomProvider`** highlighting and **`RunMenuDynamic`** refresh semantics.

- [x] **Copy / gettext** (AC: #2): Add/update **`po/default.pot`** and run **`make mo`**; keep strings short and procedural placeholders documented (fmt/msgctxt if needed).

- [x] **Renderer alignment** (AC: #4): Confirm **`SUBTLE{}`** / tab-column alignment still read correctly for multi-line diagnostics (`pkg/game/renderer/ebiten/menu.go`); adjust only if new patterns break layout.

- [x] **Regression tests** (AC: #1–5): Extend **`pkg/game/menu/maintenance_test.go`** (and/or synthetic grid tests) for strata visibility when powered/off, deterministic strings on pinned deck/level; **`go test ./...`**, **`make codestyle`**.

## Dev Notes

### Boundary vs Story 5.3

| Story | Responsibility |
|-------|----------------|
| **5.3** | Multi-hop **data placement** (`LinkageTag`, corridor/furniture surfaces, tier `level >= 5`). |
| **5.4** | Terminal **presentation layers** that **surface** correlated instrumentation text players can cross-check with room readings—not replacing linkage generation. |

Lightweight duplication of a **token substring** inside terminal strata is OK when spec’d as echo/reference, not a second linkage engine.

### Existing implementation anchors

| Concern | Location |
|---------|----------|
| Maintenance menu composition | `MaintenanceMenuHandler`, **`buildRoomDevices`**, **`GetMenuItems`** — `pkg/game/menu/maintenance.go` |
| Dynamic refresh loop | **`RunMaintenanceMenu`** → **`RunMenuDynamic`** — `pkg/game/gameplay/input.go`, `pkg/game/menu/menu.go` |
| Room highlight while menu open | **`MaintenanceRoomProvider`**, **`state.Game.MaintenanceMenuRoom`** — `pkg/game/menu/menu.go`, `pkg/game/state/state.go`, renderer `rendering.go` |
| Deck flavour line precedent | **`deck.TerminalFlavourText`** — cited in `GetMenuItems` comment |
| Epic 4 tone / procedural signage | `pkg/game/deck/environment.go`, `pkg/game/deck/observation.go`, **`specs/environmental-signage.md`**, **`specs/observation-led-puzzle-beat.md`** |

### Architecture compliance

- Go **1.24**, Ebiten **v2** — [Source: `docs/architecture.md`].
- gettext pipeline **`po/default.pot`**, **`make mo`** — [Source: `docs/architecture.md`].
- No HTTP / networked dependency.

### File structure (expected touch)

| Area | Files |
|------|-------|
| Spec | **`specs/maintenance-terminal-instrument-strata.md`** (new) |
| Deck / diagnostics helpers | `pkg/game/deck/*.go` or small `pkg/game/menu/diagnostics.go` (prefer cohesion with menu if thin) |
| Menu | `pkg/game/menu/maintenance.go`, possibly `pkg/game/menu/menu.go` |
| Tests | `pkg/game/menu/maintenance_test.go`, optional `pkg/game/setup/*_test.go` |
| i18n | `po/default.pot`, `locale/*/LC_MESSAGES/default.po` |

### Testing requirements

- **`go test ./...`** and **`make codestyle`** before moving story to **review**.
- Prefer deterministic assertions—mirror patterns in **`maintenance_test.go`** and Epic **5** setup tests.

### Previous story intelligence (5.3)

- **`specs/multi-hop-linkage-archetype.md`** defines linkage tokens and tier—terminal strata should **reference**, not redefine, those contracts.
- **`5-3-multi-hop-inference-and-cross-room-linkage.md`** — explicit **out of scope**: 5.4-class instrument strata; implement here without expanding hop graphs.

### Git intelligence

- Epic **5** activity concentrates on **`pkg/game/deck/`**, **`pkg/game/levelgen/`**, **`pkg/game/setup/`**, **`pkg/game/menu/`**, **`pkg/game/renderer/ebiten/`**—extend those seams rather than new packages unless justified in spec.

### Latest tech notes

- Ebiten **v2.9.7**—no version bump expected for text-only strata [Source: `docs/architecture.md`].

### Project context reference

- **`AGENTS.md`** — lift forward-only; unrelated to terminals but preserve global traversal assumptions in tests.
- **`specs/level-layout-and-solvability.md`** — strata must not imply unreachable clues or alter **`CanEnter`** / generator proofs.
- **`specs/environmental-signage.md`**, **`specs/observation-led-puzzle-beat.md`** — correlation vocabulary.

### Story completion status

- **Status:** done
- **Note:** Instrument strata wired into maintenance menu; deterministic trace + optional local correlates; tests and `make codestyle` pass.

---

## Change Log

- 2026-05-16: Story 5.4 implementation—maintenance diagnostic strata, gettext headers, spec, tests; sprint → review.

---

## Dev Agent Record

### Agent Model Used

Composer (Cursor agent)

### Debug Log References

- `go test ./...`
- `make mo`
- `make codestyle`

### Implementation Plan

1. Added `specs/maintenance-terminal-instrument-strata.md` and `pkg/game/menu/instrument_strata.go` for deterministic SUBSYS/LOG/FLT/CLK lines plus optional correlates from puzzle `LinkageToken` and adjacent Corridor `LinkageTag` / `EnvPlaqueMsgID`.
2. Inserted strata into `MaintenanceMenuHandler.GetMenuItems()` after flavour line; reused `SUBTLE{}` + tab-separated rows.
3. gettext msgids `MAINT_DIAG_TRACE_HEADER`, `MAINT_DIAG_CORRELATES_HEADER` in `po/default.pot`; rebuilt `mo/en_GB.utf8/LC_MESSAGES/default.mo`.
4. Renderer: verified existing `SUBTLE{}` / tab handling sufficient—no menu.go changes required.

### Completion Notes List

- Powered-terminal-only menu path unchanged (Story 2.3); strata computed only during `GetMenuItems` refresh (bounded work).
- Local correlates capped at six sorted lines per spec.
- Code review follow-up (2026-05-17): tiered correlate merge already landed; applied corridor junction helper (`setup.IsCorridorJunctionLayer`), single-pass grid scan, gettext tails for instrument rows, tab-column regression test, correlate fixture uses `ROOM_CORRIDOR`.

### File List

- `specs/maintenance-terminal-instrument-strata.md`
- `pkg/game/setup/environment.go`
- `pkg/game/menu/instrument_strata.go`
- `pkg/game/menu/instrument_strata_test.go`
- `pkg/game/menu/maintenance.go`
- `pkg/game/menu/maintenance_test.go`
- `po/default.pot`
- `mo/en_GB.utf8/LC_MESSAGES/default.mo`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
