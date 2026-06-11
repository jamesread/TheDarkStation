# Agent Notes

This file contains important notes and patterns for AI agents working on this codebase.

## Map coordinates

When discussing level layout, hazards, doors, and entities with the player or in debug analysis, use **`x:… y:…`** (horizontal, then vertical):

- **x** = column index (0-based, increases east/right)
- **y** = row index (0-based, increases south/down)

Example: gas at **x:37 y:30** is grid cell `(row=30, col=37)`.

The map dump (`map.txt`, F8) uses the same **`x:… y:…`** format and includes an `llm_coordinate_note` explaining the mapping to `Cell.Row` / `Cell.Col` and `Grid.GetCell(row, col)`.

The bottom-right build stamp uses **`BuildLabel`** (friendly local date/time to the minute, e.g. `28 May 2026, 14:35`). Position with the same bottom-aligned formula as below.

## Text Positioning in Ebiten Renderer

### Bottom-Aligned Text Positioning

When positioning text at the bottom of the screen using `drawColoredText()`, use the following formula:

```go
y := screenHeight - margin - int(textHeight * 2)
```

**Important Notes:**
- `drawColoredText()` uses baseline positioning and internally adds `fontSize` to the Y coordinate
- The formula `textHeight * 2` accounts for:
  1. The baseline offset added by `drawColoredText()` (approximately `fontSize`)
  2. The text height below the baseline (descenders and bounding box)
- Always measure the actual text string using `text.Measure(text, face, 0)` to get accurate `textHeight`
- This formula ensures the bottom of the text (including descenders) aligns with `screenHeight - margin`

**Example:**
```go
_, textHeight := text.Measure(versionText, face, 0)
versionY := screenHeight - margin - int(textHeight * 2)
e.drawColoredText(screen, versionText, versionX, versionY, colorSubtle)
```

**See:** `pkg/game/renderer/ebiten/menu.go` - version text positioning in main menu

## Input device hints

The active primary device (`pkg/engine/input`: keyboard vs gamepad) drives on-screen control text via `HintMove()`, `HintInteractPrefix()`, menu helpers in `hints.go`, etc. The Ebiten renderer switches primary on new input from either device, shows a short top-center notification, and refreshes tutorial callouts. Intent polling prefers the primary device first (gamepad-first when controller is active).

## Lift travel and deck revisit

The lift shaft is a **centered hub** on every deck. **USE** on the shaft (or exit cell) opens a **lift menu** listing all decks 1–10. The player may **travel bidirectionally** to any **unlocked** deck; locked entries show a disabled reason (missing keycard, routing repair, reactor offline, etc.).

- **Start access:** decks 1 (Airlock) and 2 are unlocked at run start.
- **Unlock graph:** seed-procedural requirements (keycards, routing couplers, thematic flags) plus fixed chains (e.g. reactor authorization → deck 5, `ReactorOnline` gates Life Support decks 6–9).
- **Run-wide inventory:** keycards and the Map persist across deck travel; keycards are **not consumed** on doors. Batteries remain **per-deck**.
- **Local lift gating:** `ExitLiftReady` on the current deck still requires local power, hazard clearance, and non-`SkipExitGate` repairs.
- **Completion:** on deck 10, **USE** the lift when `ExitLiftReady` — stepping on the exit cell does **not** auto-advance or complete the run.
- Per-deck state is saved in `DeckStates` so revisiting a deck restores its layout and local progress.
