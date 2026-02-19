# Architecture Patterns

**Generated:** 2026-02-01
**Part:** main

## Summary

Single executable, Go 1.24, Ebiten-based 2D game. Engine primitives live under `pkg/engine/`; all game-specific code lives under `pkg/game/` with a clear split between state, rendering, menu, gameplay, level generation, setup, and entities.

## Patterns

- **Game loop:** Main menu → build game (generate or load by deck) → run loop until quit.
- **Renderer abstraction:** `pkg/game/renderer/` interface with Ebiten implementation in `renderer/ebiten/`.
- **State-centric:** Core state in `pkg/game/state/`; deck and level setup drive progression.
- **Deck-based progression:** Forward-only deck/lift model (see AGENTS.md); per-deck state stored, no revisit in UI.
