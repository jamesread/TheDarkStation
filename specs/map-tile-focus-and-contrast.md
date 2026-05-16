# Map tile focus, callout focus, and contrast plates

This spec defines how **highlight backgrounds** on map tiles behave when a cell is **in focus** (e.g. callout on that cell) or marked as **adjacent interactable** (see renderer snapshot). It complements palette definitions in code (`pkg/game/renderer/ebiten/constants.go`) and the high-level art direction in `specs/gdd.md`.

---

## 1. Foreground is the source of truth

Each map cell’s **icon foreground color** (`CellRenderOptions.Color`) encodes gameplay state: unpowered vs powered, locked vs open, hazard severity, maintenance/CCTV identity, etc.

**Rule:** Any **focus** or **emphasis plate** drawn *behind* that icon (semi-opaque tile background) must be **chromatically consistent** with that foreground color.

- Do **not** use a single global tint (e.g. fixed blue–grey “selection”) for all cell types.
- Do **not** derive the plate from **complementary / inverse hue** when the icon reads as **warning, blocked, or thermal** (red, orange, amber, bright yellow): that pushes backgrounds toward **cool cyan/blue/green** and reads as the wrong semantic (powered/calm) even though the glyph says **danger / locked / offline**.

Maintenance terminals already pair orange glyphs with a **dark warm** plate; the same **principle** applies to every glyph class.

---

## 2. Plate derivation (semantic families)

The implementation classifies the icon color into a small set of **families** and applies a **dark, semi-opaque plate** that sits in the **same hue family** as the glyph:

| Family | Typical icon colors | Plate intent |
|--------|---------------------|--------------|
| **Amber / maintenance** | CCTV, maintenance orange | Dark amber / brown (existing warm path) |
| **Red / alarm** | Hazard, unpowered generator, exit locked when shown in red tones | Dark desaturated **red** (similar mood to hazard tile backgrounds) |
| **Yellow / lock** | Locked door “+” when shown in bright yellow | Dark **warm gold / brown** (not cool blue under yellow) |
| **Other** (greens, blues, purples, greys) | Floors, walls, keycards, subtle UI | May use a **cool-biased** dark plate with restrained inverse hue, so greens/blues do not collapse to muddy red |

**“Needs clearing” / blocked semantics** (e.g. locked door until puzzle conditions, hazards that must be resolved): use the **same** foreground-based derivation as focus. The icon already communicates state; the background should **reinforce** that state, not contradict it with an unrelated palette.

---

## 3. What this is not

- **Room power shading** (e.g. generator-powered wall tint) remains a separate system for **corridors / walls**, not a substitute for per-cell focus plates.
- **Hazard / unpowered full-tile backgrounds** that are already set explicitly for specific game rules (e.g. unpowered door room circuit) stay as designed; this spec governs **focus and consistency** where the plate would otherwise be chosen from a generic or complementary formula.

---

## 4. Acceptance checks (visual)

- A **red** foreground (hazard, dead generator, etc.) with focus should show a **dark red / burgundy** plate, not a teal or slate blue.
- A **bright yellow** locked-door foreground with focus should show a **dark warm** plate, not a blue fringe from complementary math.
- **Orange** maintenance glyphs stay on **warm dark** plates (unchanged expectation).
- **Green** / **blue** interactables keep readable contrast without looking like “error red” backgrounds.

---

## 5. Implementation reference

- `pkg/game/renderer/ebiten/rendering.go`: `focusPlateForForeground`, `getTileCustomBg`.
- Prefer extending **family detection** and plate helpers here rather than ad hoc colors per entity, unless a glyph class truly needs a one-off (document that exception in a code comment).
