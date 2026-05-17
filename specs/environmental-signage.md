# Environmental corridor signage (Story 5.1)

## Purpose

Diegetic plaques reinforce deck **functional identity** (`deck.FunctionalType`) without narrator voice or terminal menus.

## Taxonomy

- Gettext msgids are listed in `pkg/game/deck/environment.go` (`EnvironmentalPlaqueKeys`).
- Human-readable strings live in `po/default.pot` (English source).

## Placement rules (`setup.ApplyEnvironmentalSignage`)

1. Consider only cells that are BSP corridors: `Name == "Corridor"` and `Description == "ROOM_CORRIDOR"`.
2. **Junction:** corridor cell with **≥3** corridor neighbors (shared-wall corridor cells).
3. Candidate junctions are sorted by `(row, col)`, then shuffled with `rand.NewSource(LevelSeed)` for deterministic variety per deck.
4. Assign at most **`envMaxPlaquesPerDeck` (14)** plaques.
5. Msgid picked per cell: `(int(ft) + row*31 + col*17 + ordinal*13) % len(keys)` so functional layer biases motif rotation.

## Rendering

- Stored on `gameworld.GameCellData.EnvPlaqueMsgID`.
- Shown only when the cell is **visited or discovered** (`renderer/ebiten/snapshot.computeEnvPlaques`).
- Drawn small inside the tile bottom (`drawEnvironmentalPlaques`), colour `colorPlaque` — distinct from `LOCATION{}` room banners.

## QA regression

- Unit tests: `pkg/game/setup/environment_test.go` (junction vs straight corridor).
