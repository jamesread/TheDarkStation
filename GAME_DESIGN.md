# The Dark Station — Game Design Document

**Purpose:** This document describes the current gameplay implementation so it can be externally reviewed for gameplay improvement suggestions. It focuses on mechanics and systems; presentation and design philosophy are summarized briefly at the end.

---

## 1. Core Loop and Win Condition

- **Goal:** Progress through decks (levels) by completing objectives and reaching the lift (exit).
- **Per deck:** The player must (1) power all generators with batteries, and (2) clear all environmental hazards. Once both are done, the lift becomes usable; entering it plays a short transition and advances to the next deck.
- **No fail state:** There is no death, health, or timer. The player can always keep moving and retrying; progress is gated only by keys, items, and puzzle solutions.

---

## 2. Movement and Space

- **Grid:** Each deck is a 2D grid of cells. Cells are either room (walkable) or wall. Movement is four-directional (orthogonal); the player occupies one cell at a time.
- **Discovery:** Cells are hidden until discovered. Discovery happens by (1) stepping on a cell, (2) using a CCTV terminal to reveal a room, or (3) using the “ping” action at a maintenance terminal to reveal terminals within a 15-cell radius. The map (if owned) shows discovered geometry.
- **Blocking entities:** The following block movement (cannot step on that cell): generators, furniture, CCTV terminals, puzzle terminals, maintenance terminals, hazard controls (shutoff valves, circuit breakers, etc.), locked doors (until unlocked), and impassable hazards (e.g. sparks, gas, vacuum). Locked doors and hazards are cleared by keys/items or controls; terminals and furniture are permanent obstacles that must be walked around.

---

## 3. Inventory and Keys

- **Inventory:** The player carries a set of items. No numeric limit is shown; items are used for keys and one-off consumables.
- **Keycards:** Locked doors are tied to a room name and require the matching keycard (e.g. “Med Bay Keycard”). Using the keycard on a locked door consumes it and unlocks all doors that use that keycard on the deck. Keycards are obtained from puzzles or (in setup) from furniture/floor.
- **Batteries:** Collected from furniture, floor pickups, or puzzle rewards. Used only to power generators (see below).
- **Other items:** Patch Kit (consumed to clear Vacuum hazard), Map (granted as puzzle reward; reveals discovered layout). Item names and keycard requirements are shown in short callouts (e.g. “Door Locked / Needs: ITEM{Med Bay Keycard}”).

---

## 4. Generators and Power

- **Role:** Each deck has one or more generators. All generators must be powered to satisfy the deck objective and enable the lift.
- **Mechanics:** A generator has a required battery count (e.g. 1–3). The player stands adjacent and uses the interact key; if they have enough batteries, they are consumed and inserted. When required count is reached, the generator is powered.
- **Feedback:** Adjacent interaction opens a callout with generator name, status (POWERED/UNPOWERED), batteries inserted/required, and deck power stats (supply, consumption, available). No separate “power grid” gameplay—power is a deck-level number used for flavour and objectives.
- **Placement:** One generator is in the starting room (pre-powered). Additional generators (levels 3+) are placed in other rooms. Placement avoids chokepoints so the generator does not block the only path.

---

## 5. Environmental Hazards

- **Role:** Hazards block cells (e.g. corridor entries into a room). Clearing all blocking hazards is required before the lift can be used.
- **Types (examples):** Coolant leak, electrical fault, gas leak, radiation leak (each with a themed control); vacuum (requires Patch Kit item).
- **Clearing:**
  - **Control-based:** The player finds the matching control (e.g. Shutoff Valve, Circuit Breaker) elsewhere on the deck and interacts with it. One activation clears all hazard cells that share that hazard instance.
  - **Item-based:** Vacuum is cleared by using a Patch Kit when attempting to enter the hazard cell (consumes item).
- **Hints:** Level hints mention hazard type and that a control or item is needed (e.g. “The Coolant Shutoff is in [Room]”). Hazard callouts show short blocked messages (e.g. “Supercooled coolant sprays across the passage. Find the Shutoff Valve.”).
- **Placement:** Hazards are placed at room entry points. Controls are placed in reachable, non-chokepoint cells so they don’t block the only path to a room.

---

## 6. Locked Doors

- **Role:** Doors lock access to one or more rooms until the player has the correct keycard.
- **Mechanics:** Stepping into a locked door cell is blocked. If the player has the matching keycard, interacting (e.g. moving into the cell) consumes the keycard and unlocks all doors for that keycard on the deck. A short callout confirms (e.g. “Used ITEM{Med Bay Keycard} to unlock the Med Bay Door!”).
- **Feedback when locked:** Callout: “Door Locked” / “Needs: ITEM{Keycard Name}”.

---

## 7. Furniture

- **Role:** Furniture fills rooms and can hide batteries, keycards, and puzzle codes. It blocks movement.
- **Interaction:** From an adjacent cell, the player interacts. First time: any contained item is given (battery or keycard), and the description is scanned for puzzle codes (e.g. “Code: 1-2-3-4”). Later interactions only show name and description (no long paragraphs; short flavour text).
- **Puzzle codes:** Codes in descriptions are parsed by pattern (e.g. “Code: …”, “Sequence: …”) and stored. They are used to solve puzzle terminals (see below).

---

## 8. CCTV Terminals

- **Role:** Reveal a specific room on the map without visiting it.
- **Mechanics:** Player is adjacent and interacts. The terminal is tied to a target room; all cells of that room are marked discovered (and optionally visited). One-shot (using it again only confirms the room is already explored).
- **Discovery:** CCTV terminals are only visible after the cell is discovered (e.g. via map, walking, or maintenance ping).

---

## 9. Puzzle Terminals

- **Role:** Optional challenge that gives rewards (battery, keycard, map, etc.).
- **Mechanics:** Player interacts when adjacent. The puzzle has a solution string (e.g. “1-2-3-4” or “up-down-left-right”). The solution is found in the world (e.g. in furniture descriptions). When the player has “found” that code (by reading it in a description), the next interaction solves the puzzle and grants the reward. No manual code entry in the current implementation—finding the code in the world is the solve.
- **Rewards:** Battery, keycard hint, map (reveals layout), or similar. Map is a strong reward (e.g. later levels).
- **Placement:** Puzzles are placed in rooms; placement avoids articulation points so the terminal does not block the only path to a room.

---

## 10. Maintenance Terminals

- **Role:** Information and utility hub per room (power stats, device list, ping).
- **Mechanics:** Interacting opens a menu. The menu shows: power supply/consumption/available, room device count, room power consumption, a list of devices (lights, CCTV, puzzles, etc.) with power and status, and two actions: “Ping nearby terminals” and “Close.”
- **Ping:** “Ping nearby terminals” discovers all CCTV and puzzle terminal cells within a 15-cell (Euclidean) radius of the maintenance terminal. Results open in a sub-menu (e.g. “Discovered N terminal(s):” plus list). Ping does not consume a resource.
- **Discovery:** Maintenance terminals themselves are only visible when their cell is discovered (same as other special cells).

---

## 11. Objectives and Hints

- **Objectives:** Shown on screen (e.g. “Power up 1 generator(s)”, “Clear N environmental hazard(s)”, “Find the lift”). Updates as the player powers generators and clears hazards; when both are done, the objective is to find the lift.
- **Hints:** Per-level hints are added at setup (e.g. “A generator is in [Room]”, “The Coolant Shutoff is in [Room]”, “A puzzle terminal is in [Room]”). Shown in a hint list. No long text; short, actionable lines.
- **Callouts:** Contextual popups next to cells (e.g. “Press E/Enter to interact”, generator status, door locked message). First few moves show movement hint; first few interactions show interact hint. Callouts are brief and dismiss automatically or when moving.

---

## 12. Level Structure and Progression

- **Decks:** Levels are “decks.” Deck 1 is the first playable deck. Completing the lift transition advances to the next deck (level counter increases, new grid, new layout).
- **Generation:** Grids are generated (e.g. BSP-style). Then setup places: start, exit (lift), locked doors, generators, batteries, CCTV terminals, hazards and controls, furniture, puzzles, maintenance terminals. Placement uses avoid sets and chokepoint checks so critical paths are not blocked.
- **Persistence:** Per deck: inventory (until lift), batteries, found codes, generator list, power stats, hints. Between decks: level number; inventory and batteries reset (keycards and Map are lost unless re-found).

---

## 13. Menus and Input

- **Main menu:** Title screen with Generate (start new game), Debug (developer map), Bindings, Quit. Navigate with up/down, activate with Enter, close with Escape/Menu.
- **In-game menu:** Pause-style menu with Bindings and “Quit to Title” (returns to main menu, no save).
- **Bindings:** Configurable keys for move, interact, menu, etc.
- **Maintenance menu:** See §10. Column-style layout for stats; Ping and Close as selectable actions.
- **Input:** Movement (e.g. WASD/arrows), Interact (e.g. E/Enter), Menu (e.g. Escape). No timers or reflex-based challenges.

---

## 14. Message Log and Communication

- **Message log:** Short messages (e.g. “Welcome to the Abandoned Station!”, “You are on deck N”, “Power up N generator(s) with batteries.”). Messages are brief; no long narrative or lore dumps.
- **Markup:** In-game text uses simple markup for emphasis (e.g. ITEM{…}, ROOM{…}, ACTION{…}) so key terms are highlighted. All copy is kept short to avoid “reading walls.”

---

## 15. Presentation and Design Philosophy (Brief)

- **Visuals:** The game uses very simple graphics: character-based (symbol) tiles and menu-driven UI. There are no detailed sprites, cutscenes, or elaborate animations; the focus is on readability and low cognitive load.
- **Text:** There is a deliberate avoidance of long text. Descriptions, hints, objectives, and callouts are short. The belief is that long reading is not an enjoyable core mechanic here; information is conveyed in small chunks.
- **No risk mechanics:** There are no timers, no health, no death, and no survival pressure. The player cannot “lose” in the traditional sense; challenges are about finding keys, items, and solutions, not avoiding damage or time limits. The experience is exploratory and puzzle-like rather than punitive.

---

## 16. Summary for Reviewers

When suggesting gameplay improvements, consider:

- **Consistency with constraints:** Simple presentation, short text, no death/timers.
- **Clarity of goals:** Objectives and hints are meant to be clear and actionable.
- **Pacing and gating:** Progress is gated by keys, batteries, hazard controls, and puzzle codes. Suggestions could address balance (e.g. battery scarcity, hint usefulness) or new gating types.
- **Discovery and layout:** Discovery (walking, CCTV, ping) and layout (generation, placement) heavily affect how the player learns the deck. Improvements could target fairness (e.g. no softlocks, no mandatory blind search) or variety.
- **Interactions:** Most depth comes from generators, hazards/controls, doors/keycards, furniture (items/codes), and terminals (CCTV, puzzle, maintenance). New interaction types or refinements to existing ones (e.g. puzzle difficulty, ping radius, maintenance info) are in scope.

This document reflects the implementation as of the current codebase and is intended to support external review and iteration on gameplay design.
