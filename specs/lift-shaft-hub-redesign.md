# Lift Shaft Hub Redesign Specification

This document captures the **design decisions, findings, milestones, and implementation plan** for replacing the forward-only linear deck progression with a **centered lift-shaft hub**, **bidirectional travel**, **seed-procedural unlocks**, **deck theming**, and **thematic dependency chains**.

It supersedes the revisit policy described in `AGENTS.md` (forward-only lift) once implemented. The narrative GDD (`specs/gdd.md`) is not rewritten here; this spec focuses on **systems and layout**.

**Status:** Partial implementation (~65%). Build currently fails on missing `IsDeckAlwaysReachable` in `pkg/game/unlocks/plan.go`.

---

## 1. Background and motivation

### 1.1 Previous design

- Ten decks in a **linear forward-only** graph (deck 0 → 1 → … → 9).
- Each deck was a full **BSP map** with size/complexity scaling by deck number.
- Exit cell at the **furthest room** from start; stepping on it auto-advanced to the next deck.
- **No revisit** of earlier decks in UI (per-deck state was saved but not exposed).
- Final deck: lift had no destination → game completion.

### 1.2 Target design

- A **core lift shaft** (`Lift Shaft`): same centered rectangle on every deck, connected to the BSP layout by a corridor.
- **Lift menu** (USE on shaft): select any **unlocked** deck; backtracking encouraged.
- **Progressive unlocks**: decks 3–10 require seed-procedural objectives completed on accessible decks.
- **Bull-curve sizing** for decks 2–9 (deck 5 largest; decks 2 ≈ 9).
- **Deck themes** with room names and furniture tied to theme.
- **Thematic dependency chains**: long intuitive quest lines (e.g. reactor hub, life support downstream) without scattering puzzle mechanics across decks.

---

## 2. Confirmed design decisions

| # | Topic | Decision |
|---|--------|----------|
| D1 | Start access | Decks **1 (Airlock) + 2** unlocked at run start |
| D2 | Unlock graph | **Seed-procedural** — requirement types and source decks vary per run |
| D3 | Keycards | **Run-wide inventory**; persist across travel; **not consumed** on door use |
| D4 | Map item | **Run-wide** — once found, persists across deck travel |
| D5 | Backtracking | **Free return** to any unlocked deck via lift menu |
| D6 | Completion | **Deck 10**: USE lift in shaft when **`ExitLiftReady`** — not on arrival alone |
| D7 | Shaft | **Centered fixed rectangle** on every deck; exit cell at shaft center |
| D8 | Local lift gating | Unchanged: power + hazards + non-`SkipExitGate` repairs on **that deck** |
| D9 | Unlock types | Local shaft power + **routing repair coupler** + **security keycard** + **thematic flags** |
| D10 | Puzzle scope | **Self-contained per deck** — slime/hazard/repair chains stay on one deck; cross-deck links are **payoffs and gates** only |
| D11 | Deck themes | Seed-assigned from 23-theme pool; fixed decks **1 / 5 / 10** |
| D12 | Thematic chains | Reactor Control hub (deck 5); Life Support decks need **`ReactorOnline`**; early decks supply reactor authorization keycards |
| D13 | Map sizing | Bull curve decks 2–9; deck 5 apex; deck 1 airlock and deck 10 exit remain small |
| D14 | Discovery | **No quest log** — blocked lift rows, terminals, inventory teach the chain |
| D15 | Batteries | **Per-deck** (cleared on travel), unlike keycards and Map |

### 2.1 Assumptions (defaults unless revised)

- Lift menu lists **all decks 1–10**; locked entries **disabled with reason**.
- Inventory viewer via **gameplay pause menu** (F10 / Start); no dedicated hotkey initially.
- Deck 1 uses **Airlock** theme vocabulary; not a stripped tutorial-only room.
- Player may leave deck 10 and backtrack **until** USE on ready lift triggers completion.
- Life Support theme appears only on **decks 6–9** (after Reactor Control).

---

## 3. Architecture

### 3.1 Run-wide state

| Field | Purpose |
|-------|---------|
| `RunSeed` | Master seed for themes + unlock plan |
| `UnlockPlan` | Procedural + thematic requirements per target deck |
| `UnlockSatisfied` | Requirement IDs satisfied this run |
| `LiftRoutingPowered` | Per-deck lift routing (decks 1–2 true at start) |
| `RunInventory` | Keycards (and related run-wide items) |
| `ReactorOnline` | Set when deck 5 local chain complete; gates Life Support travel |
| `HasMap` | Run-wide once acquired (must survive deck travel) |
| `DeckThemes` | Theme per deck level (derived from `RunSeed`) |

Per-deck state (`DeckStates`) continues to store grid, room power, repairs, deck-local `OwnedItems`, etc.

### 3.2 Travel unlock (deck N ≥ 3)

All must pass:

1. `LiftRoutingPowered[N] == true` when a routing coupler payoff applies.
2. All **seed + thematic** requirements for deck N satisfied (keycards, routing repairs, **`ReactorOnline`** for Life Support–themed decks).
3. Example: deck 5 may require authorization keycards from early-deck puzzle payoffs.

### 3.3 Local lift use (per deck)

`ExitLiftReady` when:

- Shaft room has live power (or manual egress).
- All blocking hazards cleared.
- All repairs with `SkipExitGate == false` complete.

### 3.4 Generation seeds

- **Unlock plan + themes:** `RunSeed`
- **Per-deck layout:** `RunSeed + deckID * 9973` (deterministic revisits)

### 3.5 Lift interaction flow

```
USE on shaft (or adjacent exit cell)
  → if locked: blocked-lift messaging / hazard tour
  → if ready + deck 10: TriggerGameComplete
  → if ready + decks 1–9: Lift menu → TravelToDeck → spawn in shaft
```

Auto-advance on stepping exit cell is **removed**.

---

## 4. Puzzle vs unlock layering

### 4.1 Unchanged (per-deck)

Do **not** modify for cross-deck behavior:

- `PlaceHazards`, hazard-control solvability, multi-hop linkage
- Repair chains that gate **local** lift
- Power routing within a deck

### 4.2 Cross-deck payoffs (new)

| Payoff type | Placement | Effect |
|-------------|-----------|--------|
| **Security keycard** | After local exit-gating repair chain on source deck | `RunInventory` → unlock target deck or reactor auth |
| **Routing coupler** | `SkipExitGate` repair with prereqs on local chain | `LiftRoutingPowered[target] = true` |
| **ReactorOnline** | Deck 5 local completion | Gates Life Support deck travel |

Keycards must **not** be random floor spawns disconnected from local puzzles.

---

## 5. Thematic dependency chains

### 5.1 Hub model

**Deck 5 — Reactor Control** is the central anchor:

- Travel/startup requires **authorization keycards** from early decks (typically 2–3; **seed-selected** from decks 2–4).
- Completing deck 5 local work sets **`ReactorOnline`**.

**Life Support** (decks 6–9 only):

- Lift travel requires **`ReactorOnline`** plus any routing/keycard reqs.
- Maintenance terminals reference upstream reactor status when offline.

### 5.2 Example chain (seed varies details)

| Step | Player action | Result |
|------|---------------|--------|
| 1 | Complete deck 2 local puzzle | Reactor Authorization A → run inventory |
| 2 | Complete deck 3 local puzzle | Reactor Authorization B |
| 3 | Travel to deck 5 (needs A + B) | Reactor Control accessible |
| 4 | Complete deck 5 local puzzle | `ReactorOnline` |
| 5 | Routing payoff on another deck | Lift routing to Life Support deck |
| 6 | Travel to Life Support deck | Mid-game chain toward deck 10 |

### 5.3 Unlock plan generation (two-phase)

1. **`AssignThemes(runSeed)`** — deck theme per level.
2. **`BuildUnlockPlan(runSeed, themes)`** — procedural mix + fixed thematic anchors + seed shuffle of early-deck authorization sources.

---

## 6. Deck theming

### 6.1 Fixed decks (every run)

| Deck | Theme | Notes |
|------|-------|-------|
| 1 | **Airlock** | Starting level; small grid |
| 5 | **Reactor Control** | Bull-curve apex; chain hub |
| 10 | **Exit Deck** | Minimal; completion via ready lift |

### 6.2 Assignable theme pool (23 themes)

Decks **2–4** and **6–9** each receive one theme, **shuffled per `RunSeed`**, no duplicates on the same run.

1. Hydroponics  
2. Dormitories  
3. Life Support *(decks 6–9 only)*  
4. Cargo & Logistics  
5. Medical Bay  
6. Research Laboratories  
7. Communications Array  
8. Navigation / Astrogation  
9. Sanitation & Waste Processing  
10. Water Reclamation  
11. Cryogenic Storage  
12. Manufacturing Bay  
13. Observatory  
14. Security & Armory  
15. Data Archive  
16. Mess Hall / Galley  
17. Recreation Commons  
18. EVA Suit Maintenance  
19. Docking Ring  
20. Chemical Processing  
21. Particle Physics  
22. Thermal Regulation  
23. Atmospheric Processing  

### 6.3 Theme content

Per theme define:

- Display name (terminals, hints)
- Room base names + adjectives (replaces `FunctionalType` / `level % 6` cycling)
- Furniture templates (replaces `FurnitureFallbackForFunctionalLayer`)

### 6.4 Primary integration files

| File | Change |
|------|--------|
| `pkg/game/deck/themes.go` *(new)* | Pool, `AssignThemes`, `ThemeForLevel` |
| `pkg/game/deck/deck.go` | Retire or map old 6-type cycle |
| `pkg/game/generator/bsp.go` | `RoomNamesForTheme` |
| `pkg/game/levelgen/furniture.go` | Theme-based furniture |
| `pkg/game/entities/deck_furniture.go` | Per-theme tables |
| `pkg/game/setup/environment.go` | Theme-driven signage |
| `pkg/game/menu/instrument_strata.go` | Theme-driven strata |

---

## 7. Map layout

### 7.1 Bull curve (decks 2–9)

- **Deck 5** largest playable area (apex of curve).
- **Decks 2 and 9** similar smaller size at ends of the 2–9 band.
- **Deck 1** and **deck 10** small (airlock / exit).

Implementation: `pkg/game/generator/dimensions.go` → `deckGridDimensions(level)`.

### 7.2 Lift shaft

- Fixed size: **7×5** cells, **centered** on grid (`pkg/game/generator/shaft.go`).
- Room name: **`Lift Shaft`**.
- Exit cell at shaft center; corridor connects shaft to BSP rooms.
- Start cell placed outside shaft when possible.

---

## 8. Findings from exploration

### 8.1 Existing systems that support the redesign

- **`DeckStates`** already saves/restores per-deck grid and power — enables revisit without new persistence layer.
- **`RepairObjective`** chain with `PrereqIDs` and `SkipExitGate` — routing couplers can hang off local chains without blocking local lift.
- **Menu system** (`pkg/game/menu`) — lift and inventory overlays fit existing `RunMenu` pattern.
- **Tiered input** — lift USE and pause-menu inventory align with current intent flow.

### 8.2 Gaps and risks

| Finding | Impact |
|---------|--------|
| **`IsDeckAlwaysReachable` missing** | Build failure in `pkg/game/unlocks/check.go` |
| **Early `PlaceUnlockObjectives`** uses random floor spawns | Does not match payoff-after-local-puzzle intent; must refactor |
| **`FunctionalType(level)` cycles 6 types** | Conflicts with seed theme assignment; must replace in BSP/furniture/environment |
| **Keycards still go to `OwnedItems` on pickup** | Run inventory not fully wired in `PickUpItemsOnFloor` |
| **`HasMap` cleared on deck travel** | Conflicts with D4; needs run-wide persistence |
| **Adjacent-only lift interact** | Misses USE while standing on exit cell (deck 10 completion) |
| **Pure random `unlocks.Generate`** | Must become theme-aware `BuildUnlockPlan` with reactor/life-support anchors |
| **AGENTS.md revisit policy** | Still describes forward-only lift; update after implementation |

### 8.3 Partial implementation inventory (~65%)

Files created or modified (need completion/wiring):

| Area | Files |
|------|-------|
| Unlock plan | `pkg/game/unlocks/plan.go`, `check.go` |
| Run state | `pkg/game/state/run_unlocks.go`, fields on `state.go` |
| Generator | `shaft.go`, `dimensions.go`, changes to `bsp.go` |
| Objectives | `pkg/game/levelgen/unlocks.go` *(needs payoff rework)* |
| Travel | `pkg/game/gameplay/travel.go` |
| Menus | `pkg/game/menu/lift.go`, `inventory.go` |
| Lifecycle | `lifecycle.go` — `InitRunUnlocks`, `PlaceUnlockObjectives`, `TravelToDeck` |
| Wiring | `interactions.go`, `movement.go`, `repairs.go`, `main.go` (auto-advance removed) |

**Not yet implemented:** `deck/themes.go`, `unlocks/thematic.go`, theme room/furniture tables, inventory pause-menu entry, `BuildUnlockPlan`, payoff placement hooks, tests, AGENTS.md update.

---

## 9. Milestones

### M1 — Compile and core unlock logic

**Goal:** Clean build; theme-aware unlock plan skeleton.

- Add `IsDeckAlwaysReachable` (decks 0 and 1).
- Introduce `BuildUnlockPlan(runSeed, themes)` with reactor hub + life-support gates.
- Guarantee routable path for every deck 3–10.
- `go build ./...` passes.

**Exit criteria:** Build green; unit tests for plan determinism and routing-repair presence.

---

### M2 — Deck themes

**Goal:** Themed rooms and furniture per run.

- Add `pkg/game/deck/themes.go` with 23-theme pool and `AssignThemes`.
- Wire themes at run init (`InitRunUnlocks` / `BuildGame`).
- Migrate BSP naming and furniture to theme tables.
- Life Support only on decks 6–9; fixed themes on 1, 5, 10.

**Exit criteria:** `deck/themes_test.go` passes; generated rooms reflect assigned theme.

---

### M3 — Payoff placement and thematic chains

**Goal:** Cross-deck unlocks as local puzzle payoffs; reactor chain playable.

- Refactor `PlaceUnlockObjectives` after `PlaceRepairObjectives`.
- Routing couplers: prereqs on local exit-gating chain; `SkipExitGate`.
- Keycards: spawn after local chain or behind cleared slime.
- `ReactorOnline` flag + Life Support travel gate.
- Thematic keycard labels (e.g. `Reactor Authorization — Hydroponics Bay`).
- Terminal hints on Life Support when reactor offline.

**Exit criteria:** `levelgen/unlocks_test.go`, `unlocks/thematic_test.go` pass; no changes to hazard solvability graph.

---

### M4 — Run-wide inventory and backtracking

**Goal:** Keycards and Map persist; player can map chains via inventory.

- Keycard pickup → `RunInventory` (floor + furniture paths).
- `HasMap` survives deck travel.
- `HasKeycardNamed` used consistently; keycards not consumed on doors.
- Inventory entry in gameplay pause menu.
- Optional: status bar keycard summary.

**Exit criteria:** Keycard on deck 5 unlocks door on deck 2 after backtracking; Map persists across travel.

---

### M5 — Lift interaction and completion

**Goal:** Full lift hub UX.

- USE on current exit cell opens lift / completes run.
- Lift hints for routing menu vs deck 10 completion.
- Lift menu shows all decks with disabled + reason when locked.
- `TravelToDeck` spawns player in shaft.
- Deck 10 completion only when `ExitLiftReady` + USE.

**Exit criteria:** Manual playtest path decks 1 → 2 → backtrack; deck 10 completion path verified.

---

### M6 — Tests and documentation

**Goal:** Regression safety and agent guidance.

- New tests: unlock plan, thematic chains, shaft/dimensions, run unlocks.
- Update: `lifecycle_test.go`, `bsp_test.go`, deck-power/keycard tests.
- Update `AGENTS.md` revisit policy, lift hub, run inventory.

**Exit criteria:** `go test ./...` passes; AGENTS.md accurate.

---

### M7 — Cleanup (low priority)

- Bidirectional `deck.Graph` for docs/dev tools.
- Retire `FunctionalType` cycling where replaced by themes.
- Optional gettext for lift/inventory strings.

---

## 10. Key behavioral rules (summary)

**Travel unlock (deck N ≥ 3):** routing powered + all thematic/procedural reqs + `ReactorOnline` for Life Support decks.

**Local lift:** `ExitLiftReady` on current deck only.

**Puzzles:** Mechanics local; meaning cross-deck via payoffs and flags.

**Discovery:** Lift block reasons + terminals + inventory; no quest log.

---

## 11. Out of scope

- Full rewrite of `specs/gdd.md` narrative sections
- Dedicated inventory hotkey (unless requested later)
- Custom shaft art beyond existing exit/lift rendering
- gettext for all new UI strings (initial pass may use markup patterns)

---

## 12. Related documents

| Document | Relationship |
|----------|--------------|
| `specs/gdd.md` | Narrative GDD; deck progression text will diverge until GDD is updated |
| `specs/level-layout-and-solvability.md` | Per-deck solvability invariants unchanged |
| `AGENTS.md` | Revisit policy must be updated when M6 completes |
| `pkg/game/deck/deck.go` | Current forward-only graph; travel authority moves to unlock plan |

---

## 13. Revision history

| Date | Notes |
|------|-------|
| 2026-06-09 | Initial spec from design/plan session: lift hub, unlocks, themes, thematic chains, partial implementation findings |
