# API Contracts — Main

**Generated:** 2026-02-01
**Part:** main

## Summary

This application is a single-player 2D game (Ebiten). It does **not** expose an HTTP/REST or RPC API. All interaction is through:

- **Input:** Keyboard and mouse (handled in `pkg/game/gameplay/input.go`, `pkg/game/renderer/ebiten/input.go`).
- **Internal flow:** Menu actions and game loop drive state changes; no external API contracts.

No API catalog or endpoint documentation applies. For integration with other systems, the only “interface” is the executable (e.g. `LEVEL` env, `-level` flag for dev testing).
