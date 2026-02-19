# Technology Stack

**Generated:** 2026-02-01
**Part:** main

## Technology Table

| Category | Technology | Version | Justification |
|----------|------------|---------|---------------|
| Language | Go | 1.24.0 (toolchain 1.24.9) | Primary language; go.mod |
| Module | darkstation | — | Application module name |
| Graphics / Game | Ebiten | v2.9.7 | 2D game engine; main renderer |
| Localization | gotext (leonelquinteros) | v1.7.2 | i18n; .mo/.pot in po/, mo/ |
| Terminal / CLI | golang.org/x/term | v0.39.0 | Terminal handling |
| Color output | gookit/color | v1.5.2 | Colored terminal output |
| Data structures | zyedidia/generic | v1.1.0 | Generic containers/algorithms |
| Build / Lint | Make, golangci-lint | — | Makefile; codestyle target |
| CI | GitHub Actions | — | .github/workflows/build.yml, codestyle.yml |

## Architecture Pattern

- **Style:** Game application with a single Ebiten-based renderer and a clear separation between engine primitives and game logic.
- **Entry:** `main.go` → init gettext, Ebiten renderer, then game loop: main menu → build game (generate or load) → run until exit.
- **Layering:** `pkg/engine/` (world, grid, cell, terminal, input, FOV) is engine-level; `pkg/game/` holds state, renderer (Ebiten), menu, gameplay, generator, setup, entities, deck, levelgen, devtools.
- **State:** Central game state in `pkg/game/state/`; deck-based progression and level setup in `pkg/game/setup/` and `pkg/game/deck/`.
