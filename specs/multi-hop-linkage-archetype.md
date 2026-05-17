# Multi-hop linkage archetype (Story 5.3)

## Purpose

 Encode one **multi-hop inference** pattern: the player must correlate a **linkage token**
 from at least **two distinct reachable readings** (movement + furniture text) before a
 keyed security terminal will **admit** a known numeric fragment. Power, overload messaging,
 and `GetAvailablePower()` behaviour are unchanged (Epic 2).

## Tier

- **`MultiHopLinkageActive`**: **`level >= 5`** (1-based deck display) **and not** minimal final deck
  (`deck.IsFinalDeck`). Stricter than Story 5.2’s observation fingerprint tier (**`level >= 3`**).
- **Rollback**: if keyed puzzle, correlating furniture, or a distinct non-observation corridor
  junction plaque cannot be assigned, linkage fields are left unset for that deck (no partial
  gating deadlock).

## Keyed puzzle

The terminal that carries linkage matches **`deck.MultiHopKeyedSequenceSolution`** (currently **`2-4-6-8`**),
aligned with **`pkg/game/levelgen/puzzles.go`** numeric sequence order when two terminals exist.

## Linkage token

- Canonical string **`LINK-MHOP-A`** (`deck.MultiHopLinkageToken`).
- Persisted where **`DeckState.Grid`** already persists data:
  - **`PuzzleTerminal.LinkageToken`**
  - **`GameCellData.LinkageTag`** on the linkage junction cell (noted when **visited**)
  - **`GameCellData.EnvPlaqueMsgID`** → **`ENV_PLAQUE_LINK_MHOP_A`** (gettext)
  - **`Furniture.Description`** gains **`. Relay: LINK-MHOP-A`** on the correlating code surface (leading period terminates `CheckForPuzzleCode` parsing).

Runtime player memory: **`state.Game.LinkageTokensSeen`** (cleared on deck load/reset like **`FoundCodes`**).

## Hop graph (minimum)

1. **H1 — corridor hop:** visit the linkage junction (plaque + `LinkageTag` records the token).
2. **H2 — document hop:** read furniture that contains **`Code: 2-4-6-8`** (and relay line), recording
   **`FoundCodes`** and reinforcing the token via **`Relay:`** parse.

Order may be reversed; **both** token and code are required before admit.

## Generation ordering

1. **`levelgen.PlacePuzzles`**
2. **`setup.ApplyEnvironmentalSignage`**
3. **`setup.ApplyObservationLedPuzzleCues`** (may rewrite **one** junction to **`ENV_PLAQUE_OBS_*`**)
4. **`setup.ApplyMultiHopLinkage`** (chooses a **different** junction: skips **`ENV_PLAQUE_OBS_*`** prefixes)

This ordering avoids blind overwrites between Story 5.2 and 5.3 stamps.

## QA

- Pin **`LevelSeed`** for procedural repro, or use synthetic grid tests under **`pkg/game/setup`**.
- Assert tier off on **`level < 5`** and on final minimal deck (`level == deck.TotalDecks`).
