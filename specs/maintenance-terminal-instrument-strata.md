# Maintenance terminal instrument strata (Story 5.4)

## Purpose

Maintenance terminals expose **read-only diagnostic layers** alongside existing power summaries and toggles. This is **presentation** of correlatable instrumentation, not new placement rules for clues (Stories 5.1–5.3 own env / observation / linkage placement).

## Strata types

| Stratum | Role | Source |
|--------|------|--------|
| **Subsystem trace** | Timestamp-like markers, BUS id, fault class, deck/level clock | Deterministic synthesis from `Level`, `CurrentDeckID`, `LevelSeed`, selected room name, functional layer |
| **Local correlates** | Puzzle `LinkageToken`, adjacent **Corridor** `LinkageTag`, `EnvPlaqueMsgID` | Grid scan bounded to selected room + corridor **neighbors** (4-neighbour) |

Cap **local correlate** lines at **6** total: include **puzzle linkage (`XCORE-`)**, then **junction stamps (`JNCT-`)**, then **`ENVREF-`** signage rows (sorted within each tier)—not pure global alphabetical sort, so linkage clues are not truncated first when many plaques exist.

## Tiering

- **Trace** lines: always emitted when the maintenance menu is open (menu requires a **powered** terminal per Story 2.3 interaction path).
- **Correlates**: only when matching cells exist; section is omitted when the list is empty.

## gettext

Fixed section headers use msgids:

- `MAINT_DIAG_TRACE_HEADER`
- `MAINT_DIAG_CORRELATES_HEADER`

Trace tails and correlate suffix columns use gettext msgids (`MAINT_DIAG_LOG_CLOCK_SUFFIX`, etc.) so embedded `.mo` stays authoritative; procedural ids (BUS hex, timestamps, `ENV_*` msgids) stay literal.

## QA

- Synthetic output is **deterministic** for fixed `Game` fields + room name.
- With multi-hop linkage active, a room containing the keyed puzzle should list `XCORE-` relay line; adjacent stamped corridor should surface `JNCT-` / `ENVREF-` when plaques/tags are present.

## References

- `specs/multi-hop-linkage-archetype.md` — linkage token semantics (do not redefine).
- `pkg/game/menu/instrument_strata.go` — implementation.
- `pkg/game/setup/linkage.go` — `LinkageTag` / `EnvPlaqueMsgID` on junctions.
