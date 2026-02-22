---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14]
inputDocuments:
  - specs/gdd.md
  - specs/power-system.md
  - specs/level-layout-and-solvability.md
  - _bmad-output/implementation-artifacts/1-1-grid-and-movement.md
  - _bmad-output/implementation-artifacts/1-2-rooms-and-corridors.md
  - _bmad-output/implementation-artifacts/1-3-start-room-and-exit.md
  - _bmad-output/implementation-artifacts/1-4-room-connectivity.md
  - _bmad-output/implementation-artifacts/tech-spec-game-mechanics-adjacent-rooms-connectivity.md
  - _bmad-output/implementation-artifacts/sprint-status.yaml
  - _bmad-output/planning-artifacts/epics.md
documentCounts:
  briefs: 0
  research: 0
  brainstorming: 0
  projectDocs: 0
  specs: 3
  implementationArtifacts: 6
  planningArtifacts: 1
workflowType: gdd
lastStep: 14
project_name: TheDarkStation
user_name: James
date: '2026-02-21'
game_type: horror
game_name: The Dark Station
---

# The Dark Station - Game Design Document

**Author:** James
**Game Type:** horror
**Target Platform(s):** Desktop (Windows, Linux)

---

## Executive Summary

### Game Name

The Dark Station

### Core Concept

*You are the last system still performing maintenance in a universe where maintenance no longer changes anything.*

The Dark Station is a narrative-driven exploration game set on a vast, long-abandoned space station. The player is an autonomous maintenance unit—never explicitly named as such during play (subtle hints only), until the final moment at the docking station, where the player is revealed as a robot—performing routine operations while inferring that the civilization that built the station is extinct and the universe is approaching thermal equilibrium. Power is the central mechanic and primary storytelling device: the player restores and routes limited power (generators, batteries, room power for doors and CCTV, maintenance terminals) rather than generating it; every technically correct action accelerates decay. The core loop is enter deck → assess failing systems → restore limited power → route to critical subsystems → accept instability elsewhere → progress deeper. What feels like progress is resource exhaustion.

The station is finite, with a fixed number of procedurally revealed decks representing functional layers (Habitation, Research, Power Distribution, Emergency Systems, Core Infrastructure). Maintenance terminals report in technical, dry language; rooms and items use functional, cold naming; life support is always OFFLINE or PERMANENTLY DISABLED. The tone aims for quiet dread, loneliness, and ritual without purpose. The game ends in *completion*—the job done, the job no longer mattering—not victory, failure, or escape. The lift has no destination on the final deck.

The design principle is non-negotiable: never explain the story directly; only show what systems no longer allow.

### Game Type

**Type:** Horror
**Framework:** This GDD uses the horror template with type-specific sections for atmosphere and tension, fear mechanics, resource scarcity, and puzzle integration.

### Target Audience

Adult, regular gamers who enjoy roguelikes and novel story ideas; familiar with minimal text-based presentation; 20–30 minute sessions; replayable via procedural generation. (See **Target Audience** section below for full detail.)

### Unique Selling Points (USPs)

No death/failure mechanics; “sudoku that fights back” (solvable with persistence); narrative as context for the puzzle; clear win state (solve the last room). See **Unique Selling Points** section below for full detail.

---

## Target Platform(s)

### Primary Platform

Desktop (Windows, Linux). The game already builds for Windows and Linux.

### Platform Considerations

- **Distribution:** Native desktop builds; no storefront specified.
- **Input:** Keyboard-first (movement N/S/E/W, arrows, vim-style; terminal interaction). Mouse optional for UI.
- **Performance:** Desktop allows stable frame rate and minimal thermal constraints; 20–30 minute sessions align well with desktop play.
- **Replayability:** Procedurally generated decks support repeated runs without additional platform-specific features.

### Control Scheme

Keyboard: movement (cardinal directions, arrows, vim-style keys), terminal and generator interaction. Input handling and walkability follow existing implementation (grid, blocking entities, door power, win conditions).

---

## Target Audience

### Demographics

**Adult gamers.** The storyline and mechanics (implicit narrative, completion-not-victory, thematic weight) are aimed at adults. No specific age bracket; the tone and themes are what define the audience.

### Gaming Experience

**Regular (core) gamers** — comfortable with roguelike structure, procedural runs, and systems-driven play. Not casual drop-in; not hardcore-only.

### Genre Familiarity

Players **know roguelikes** (runs, procedural generation, finite but replayable progress). They are **aware the game uses extremely basic text-based graphics** and **find that style appealing** — the minimal presentation is a draw, not a compromise.

### Session Length

**20–30 minutes** per session. The game is **replayable** because playthroughs are **dynamically generated**; each run can feel different while fitting within that session length.

### Player Motivations

- Roguelike structure and procedural runs
- Interesting, novel story ideas (implicit narrative, atmosphere over exposition)
- Minimalist, text-based aesthetic
- Short, focused sessions with meaningful variation between runs

---

## Goals and Context

### Project Goals

1. **Creative — logic puzzle for punishment-averse players:** A game that appeals to players who like **complex constraint-satisfaction** (resource allocation, prioritisation) but **no death, no failure state, no progress loss**. Challenge = working out the logic; persistence and effort always lead to a solution.

2. **Feel — “sudoku that fights back”:** The game feels like a **difficult constraint puzzle** where **systems impose limits** (power, decay, overload). The puzzle "fights back" by tightening constraints, not by killing the player or erasing progress. **Every run is solvable** with enough persistence; difficulty is logical, not punitive.

3. **Win state — reach the end, solve the last room:** The player “wins” by getting to the end of the station and solving the last room. Completion = **logic/puzzle resolution**; no survival or combat win condition.

4. **Narrative as context:** The heat-death-of-the-universe story **frames** the station, the mechanics, and the tone. It gives **meaning**, not extra puzzles or gates; the core loop stays **puzzle and logic**.

5. **Replayability:** The game stays interesting to replay **even when the story is known**. The **replay driver is the puzzle**: same rules, **new instance** (procedural deck). The mechanic stays rewarding; players replay for **"solve this new instance,"** not for the story again.

### Background and Rationale

The game targets players who want **deep logic challenges without punishment**. The heat-death premise frames the puzzles and the station as a dying system the player is maintaining; completion is solving the final room, with the narrative providing emotional and thematic weight. The “sudoku that fights back” metaphor captures **dynamic constraints** (limited power, decay, prioritisation) and the **guarantee that every run is solvable** with enough persistence.

---

## Unique Selling Points (USPs)

1. **No death or failure mechanics.** Low-stress constraint puzzle: no permadeath, no game over, no progress loss. Challenge = working out the logic; persistence and effort always lead to a solution. Appeals to **punishment-averse** players who like hard puzzles but not punishment.

2. **“Sudoku that fights back.”** Systems **impose constraints** (power limits, overload, decay, prioritisation); the puzzle "fights back" by tightening limits, not by failure states or randomness. **Every run is solvable**; difficulty is logical, not punitive.

3. **Narrative as context, not obstacle.** The heat-death-of-the-universe story gives the puzzles and the station meaning and tone. The player experiences the last moments of the universe through the lens of maintenance and logic; the narrative supports the puzzle experience instead of competing with it.

4. **Clear completion = solving the last room.** The player wins by reaching the end and solving the last room. A satisfying, logic-based win state that fits the target audience and the “no failure” design.

5. **Replay for the puzzle, not the story.** The game is built to be replayed many times; the primary reason to replay is **same rules, new instance** (procedural deck)—the puzzle remains rewarding. The story is important and powerful on a first run, but replay value comes from "solve this new instance."

### Competitive Positioning

The Dark Station sits at a specific intersection: **roguelike-style structure** (procedural decks, replayable runs) **without roguelike punishment**; **constraint-satisfaction / resource-allocation depth** with an **atmospheric, thematic wrapper**; and **minimal presentation** that keeps focus on the puzzle. It serves **punishment-averse** players who like complex logic and novel story ideas but not death/failure mechanics—a “difficult sudoku that fights back” (dynamic constraints, **guaranteed solvable**) with a heat-death narrative context.

---

## Core Gameplay

### Game Pillars

1. **Logic over punishment.** Every challenge is a constraint-satisfaction puzzle. No death, no failure state, no progress loss. Players can be stuck until they work out the logic or find the missing clue—difficulty is logical, not punitive.

2. **Constraint puzzle that fights back.** Systems impose limits (power, decay, overload); the puzzle "fights back" by tightening constraints. Every run is solvable with enough persistence. The station resists in a constrained, logical way, not through failure states or randomness.

3. **Narrative as context.** The heat-death premise and the player's identity (autonomous unit, last semi-sentient thing) frame the puzzles and the tone. Story gives meaning and finality; it does not gate progress or add puzzle types.

4. **Replay through variation.** Same rules, new instance. The replay driver is the puzzle—procedural decks and layout variation keep each run engaging even after the story is known.

**Pillar prioritization:** When pillars conflict, prioritize in this order: (1) Logic over punishment, (2) Constraint puzzle that fights back, (3) Narrative as context, (4) Replay through variation.

### Core Gameplay Loop

**Loop:** Enter deck → Assess failing systems (power, doors, hazards) → Restore limited power (generators, batteries) → Route power to critical subsystems via maintenance terminals → Accept instability elsewhere (prioritisation, short-outs) → Progress deeper (next deck) → Repeat until final deck.

**Loop diagram:**
```
[Enter deck] → [Assess: power, doors, CCTV, hazards] → [Restore power / insert batteries]
       ↑                                                                  ↓
[Lift to next deck] ← [Reach exit; all generators powered; hazards cleared] ← [Route power; toggle doors/CCTV; solve puzzles]
```

**Loop timing:** One deck can take several minutes to 20–30 minutes depending on layout and player pace; a full run (all decks) fits the target 20–30 minute session or spans multiple sessions.

**Loop variation:** Each deck is procedurally generated (rooms, corridors, terminal and generator placement, keycards, hazards). Same mechanics, new constraint puzzle each time.

### Win/Loss Conditions

#### Victory Conditions

- **Reach the end:** The player reaches the final deck and the last room.
- **Solve the last room:** All generators powered, all blocking hazards cleared, exit reached (per level win conditions).
- **Completion ending:** In the last room, the player encounters a **docking station** that shuts the player down. The message conveys:
  - No communication for a long time; no lifeforms detected.
  - The player is the last semi-sentient thing in the universe.
  - There is no way to replenish power.
  - The unit is being shut down to conserve the station's computer logs for as long as possible.
  - **Tone:** Futility and finality. The game ends in completion—the job done, the job no longer mattering.
  - **Identity revelation:** It is acceptable and necessary here—and only here—to refer to the player explicitly as a robot. The player finds out as the last action of the game (on their first run). See *Player identity and in-game language* below.

#### Failure Conditions

**There is no failure state.** No game over, no death, no progress wipe. Players can be **stuck** until they work out the logic or find the missing clue; persistence and effort always lead to a solution. Getting stuck is part of the puzzle, not a penalty.

#### Failure Recovery

Not applicable—there is no failure state. If the player is stuck, "recovery" is **working out the logic** or **finding the missing clue** (e.g. keycard, hazard control, power route). No reset, no lives; the same run remains solvable.

**Player identity and in-game language:** During the game, never refer to the player explicitly as "unit," "robot," "system," or equivalent—subtle hints are fine (e.g. system messages that imply nature without naming the player). The **only exception** is the final room at the docking station: there it is acceptable and necessary to refer to the player as a robot. That moment is the revelation; the player finds out exactly then, as the last action of the game on their first run.

---

## Game Mechanics

### Primary Mechanics

1. **Move.** Grid-based movement through rooms and corridors (N/S/E/W). Walkable cells vs blocking (walls, furniture, terminals, hazards, generators). Doors require room power (and keycard if locked); movement is blocked until the player satisfies conditions. **Pillars:** Logic over punishment (no death); constraint puzzle (routing, reachability). **When:** Constantly. **Skill:** Spatial reasoning, route planning. **Interacts with:** Interact (reach terminals/generators), Discover (explore deck).

2. **Interact.** Use cells when adjacent: generators (insert batteries), maintenance terminals (open menu: toggle room power, restore nearby terminals, view power stats), CCTV terminals (view room), hazard controls (fix hazards), puzzle terminals (solve), doors (attempt passage). **Pillars:** Constraint puzzle (power routing, prioritisation); narrative as context (terminals report). **When:** Situationally at key cells. **Skill:** Prioritisation, resource allocation. **Interacts with:** Manage power, Move, Discover.

3. **Manage power.** Insert batteries into generators (permanent per level); toggle room door power and room CCTV power at maintenance terminals (own room + adjacent rooms); "Restore power to nearby terminals" to power other terminals. Observe supply vs consumption; overload causes short-out (other rooms unpowered in deterministic order). **Pillars:** Constraint puzzle that fights back; logic over punishment (solvable). **When:** At maintenance terminals. **Skill:** Resource allocation, trade-offs. **Interacts with:** Interact, Discover (find batteries, assess layout).

4. **Discover.** Explore the deck (visibility/lighting tied to available power); read terminal messages and stats; find keycards and batteries; assess which rooms/doors/hazards block progress. **Pillars:** Narrative as context (terminals, tone); replay through variation (new layout each run). **When:** Throughout the run. **Skill:** Information gathering, memory. **Interacts with:** Move, Interact.

5. **Solve.** Satisfy constraints to progress: route power so doors can be powered; prioritise which rooms get power when supply is limited; fix hazards via controls or items; unlock doors with keycards; power all generators and clear hazards to reach exit. **Pillars:** All four (logic, constraint puzzle, narrative context, replay). **When:** The core engagement. **Skill:** Logic, planning, persistence. **Interacts with:** All other mechanics.

### Mechanic Interactions

- **Move + Interact:** Player must move to cells to interact (generators, terminals). Door power and keycards gate movement.
- **Interact + Manage power:** Maintenance terminals are the only way to change room power and terminal power; power state determines what the player can do (doors open, CCTV usable).
- **Manage power + Solve:** Power is finite; toggling one room on can short-out others. Solving = finding a valid allocation and order.
- **Discover + Solve:** Keycards and batteries are in the world; discovering layout and items informs solve order. Lighting (tied to available power) affects what the player can see and revisit.

### Mechanic Progression

- **No unlocks or upgrades within a run.** The same verbs (move, interact, manage power, discover, solve) apply from first deck to last. Mastery is understanding the rules and applying them to each new procedural layout.
- **Cross-run:** Players learn terminal language, power maths, and solvability patterns; later runs feel faster and more intentional (replay through variation).

---

## Controls and Input

### Control Scheme (Desktop: Windows, Linux)

| Action | Input | Notes |
|--------|--------|------|
| Move N/S/E/W | Arrow keys, or N/S/E/W, or vim-style (h/j/k/l) | Cardinal grid movement |
| Interact / Confirm | Enter, Space, or context key | At generator, terminal, door |
| Open maintenance menu | Interact at maintenance terminal (when powered) | Menu: room power toggles, restore nearby terminals, stats |
| (Reserve) | — | Mouse optional for UI/menus if needed |

Movement and interaction are keyboard-first; no combat or twitch timing. Controls align with desktop and the existing implementation (grid, blocking, door power, win conditions).

### Input Feel

- **Movement:** Deliberate, one cell per input; no run or dash. Suits a logic-puzzle pace.
- **Interaction:** Confirm at cell; menus (e.g. maintenance) are step-by-step. No hand gymnastics; common actions (move, interact) are easy to reach.
- **Platform:** Keyboard-centric; mouse optional for menus or navigation if added later.

### Accessibility Controls

- Rebindable keys (movement, interact) recommended for accessibility.
- No time pressure on inputs; persistence and logic are the challenge.
- Text-based or minimal graphics support screen readers and low-motion preferences where UI is text-heavy.

---

## Horror Game Type: Atmosphere, Scarcity, and Puzzles

*(Inferred from specs and GDD; horror template applied.)*

### Atmosphere and Tension Building

**Psychological horror, no jump scares.** Tone: quiet dread, loneliness, ritual without purpose. Visual design: minimal, text-based; lighting tied to available power (darkness when power low). Environmental storytelling via maintenance terminals (technical, dry reports; life support always OFFLINE/FAILED). Pacing: tension from constraint and futility, not from threat; release when a deck is solved. No safe zones vs danger zones—the whole station is “post-danger”; unease comes from meaning, not from being chased.

### Fear Mechanics

**No traditional fear mechanics.** No pursuer, no sanity meter, no combat. “Fear” is existential: the player infers heat death, last semi-sentient unit, systems that no longer allow. Visibility/darkness: when `GetAvailablePower() <= 0`, cells beyond a small radius can darken (discovered/visited cleared). Vulnerability is logical (stuck until puzzle solved), not physical. Optional late-game flavour: unit decay (inputs repeat, movement pauses, terminal lag); messages shift from STATION POWER UNSTABLE to UNIT RESPONSE DELAYED to UNIT POWER RESERVE CRITICAL.

### Resource Scarcity

**Power and batteries.** Supply = powered generators (100 W each); consumption = doors (10 W per room when powered), CCTV (10 W per terminal when room powered), solved puzzles (3 W each). Batteries are finite per level; insertion is permanent. No ammo/health; scarcity is “which rooms get power” and “what order to do things.” Risk vs reward: exploring vs conserving power; overload causes short-out (other rooms unpowered). Lighting does not consume power but visibility depends on available power.

### Puzzle Integration

**Constraint-satisfaction and power routing.** Puzzles are environmental and systemic: route power, prioritise rooms, find keycards, fix hazards, power generators. Difficulty = logic and constraints; no locks/codes in the traditional sense. Narrative purpose: every correct action is maintenance in a dying universe; puzzles embody “systems that no longer allow.” Hint system: terminal reports and player inference; no explicit hints. Puzzle–tension balance: persistence always leads to solution; tension from futility and tone, not from failure state.

---

## Progression and Balance

### Player Progression

**No character or item upgrades within a run.** Progression is learning: rules of power, short-out behaviour, keycard and hazard placement patterns, terminal language. Cross-run: players get faster and more intentional; replay value is “solve this new instance” (procedural deck). Optional (from specs): late-game unit decay as flavour—inputs occasionally repeat, movement pauses, terminal options fail to register; system messages shift toward UNIT POWER RESERVE CRITICAL. That is narrative/atmosphere, not a progression unlock.

### Difficulty Curve

**Decks as difficulty progression.** Later decks (from specs): generator output can decrease, power costs increase, automation “fights back” (doors relock, systems shut down, power reroutes). The station obeys “dead rules”; difficulty is constraint tightness and logical complexity, not punishment. Final deck: minimal rooms and systems, barely functional power; the end is real. Curve is “increasing constraint and tone,” not “more damage or more enemies.”

### Economy and Resources

**Single resource economy: power.** Supply = sum of powered generators (100 W each). Consumption = doors (10 W per room when powered), CCTV (10 W per terminal when room powered), solved puzzle terminals (3 W each). Batteries: found in world, inserted into generators (permanent per level); no trading or currency. Keycards: one per locked room, found in world; no economy, only gating. “Economy” is allocation: which rooms get power, when to restore terminals, what to leave off when overload would occur.

---

## Level Design Framework

### Level Types

**Decks as functional layers.** Each deck represents a thematic layer (e.g. Habitation, Research, Logistics, Power Distribution, Emergency Systems, Core Infrastructure). Room names are functional and cold (e.g. Cryogenic Habitation Block, Central Power Exchange). Within a deck: **rooms** (named, walkable, with doors/CCTV/terminals/generators/hazards) and **corridors** (connecting rooms). Final deck: minimal rooms and systems, barely functional power (per GDD and specs). Level type is “single deck” per play space; “level” = one deck instance.

### Level Progression

**Sequential decks, forward-only, no revisit.** Player explores decks in order; lift advances to next deck only. Fixed number of decks; no infinite descent. Each deck is generated when first entered (procedural: BSP rooms, corridors, placement of terminals/generators/hazards/keycards). Revisit of previous decks is out of scope (per project policy). Final deck has no lift destination; reaching it and solving the last room triggers the completion ending (docking station shutdown).

---

## Art and Audio Direction

### Art Style

**Minimal, text-based.** Extremely basic text-based graphics; target audience finds this style appealing. Unpowered power-dependent cells (doors, maintenance terminal, CCTV, hazard control): same red background (e.g. dark red / hazard style). Powered generators: distinct background (e.g. green). Walls and corridors: no power-based colour change. Room and item names: functional, cold, slightly outdated (specs). No 3D or detailed sprites; focus on readability and tone. Life support systems in copy: always OFFLINE/FAILED/PERMANENTLY DISABLED.

### Audio and Music

**Minimal or TBD.** Specs do not define audio; tone is quiet dread and loneliness. Options: silence, subtle ambient (hum, distant systems), or minimal cues for terminal interaction and power events. No music required for current design; if added, should support unease and finality, not action. Accessibility: no critical information conveyed by audio alone.

---

## Technical Specifications

### Performance Requirements

**Desktop: 60fps target.** Single executable; no streaming or loading screens mid-run. Deck generation and power/lighting updates must stay within frame budget. No database; all state in-memory. Test suite: `go test ./...` or `make test`; no regressions. Build: `make build` or `go run .`; supported platforms Windows and Linux.

### Platform-Specific Details

**Windows and Linux.** Builds produce native executables. Go 1.24+, Ebiten for rendering. Input: keyboard-first (movement, interact); mouse optional for menus. Dev: `make` or `go run .`; run specific deck with `LEVEL=N` or `-level N`. i18n: `po/default.pot`, `make mo`. Architecture: single binary; `pkg/engine/` (world, input), `pkg/game/` (state, gameplay, setup, deck, menu, renderer/ebiten). No HTTP API; no cloud save in current scope.

### Asset Requirements

**Minimal.** Text/cell-based presentation; terminal and menu UI (text or simple tiles). No 3D models, no large texture sets. Possible: small tile set or ASCII/Unicode glyphs for cells, doors, terminals, generator. Font(s) for terminal and HUD. If audio added: ambient and UI cues only. Localisation: string assets for terminal copy and messages (gettext/po).

---

## Development Epics

### Epic Structure

*(Summary from `_bmad-output/planning-artifacts/epics.md`. Full FR/NFR list and story breakdown are in that file.)*

- **Epic 1: Deck Exploration and Structure** — Grid, movement, rooms, corridors, start/exit, room connectivity, procedural deck generation. FR1, FR2, FR10, FR23, FR24. Stories: 1.1 Grid and Movement (done), 1.2 Rooms and Corridors (done), 1.3 Start Room and Exit (done), 1.4 Room Connectivity (review), 1.5 Procedural Deck Generation (backlog).

- **Epic 2: Power Grid and Room Control** — Generators, batteries, room power (doors/CCTV), maintenance terminal power and “restore nearby,” overload/short-out, lighting, unpowered visual feedback. FR4, FR11–FR16, FR25. Stories: 2.1–2.6 (backlog).

- **Epic 3: Gates, Keys, and Solvability** — Doors and room power, locked doors and keycards, blocking hazards and controls, generators and exit win condition, gatekeeper and door-power solvability. FR17–FR22. Stories: 3.1–3.5 (backlog).

- **Epic 4: Narrative, Tone, and Completion** — Maintenance terminals as narrative surface, room and item naming, progressive decay across decks, final deck and completion ending (docking station shutdown). FR3, FR5–FR9, NFR1–NFR4. Stories: 4.1–4.4 (backlog).

**Reference:** See `_bmad-output/planning-artifacts/epics.md` for full acceptance criteria and FR coverage.

---

## Success Metrics

### Technical Metrics

- **Performance:** 60fps on target desktop hardware (Windows, Linux).
- **Stability:** No crashes or freezes during normal play; clean exit and reset.
- **Build and test:** `make build`, `make test` pass; no regressions in gameplay/setup tests.
- **Code quality:** Existing codestyle and test patterns maintained.

### Gameplay Metrics

- **Session length:** Target 20–30 minutes per run (or per session); full run completable within that window or across sessions.
- **Completion:** Players can reach the final deck and trigger the completion ending (docking station).
- **Replay:** Runs are replayable via procedural decks; no failure state, so “runs completed” or “decks reached” are optional metrics if telemetry is added later.

---

## Out of Scope

- **Revisit previous decks.** The lift is forward-only; no returning to earlier decks (per project policy).
- **Permadeath or failure state.** No game over, no progress wipe, no death mechanic.
- **Multiplayer or co-op.** Single player only.
- **Explicit story exposition (except final moment).** Narrative is implicit during play; no cutscenes or dialogue that explain “you are a robot” or “the universe is ending.” The only exception: the docking station message in the final room may—and should—refer to the player as a robot; that is the intended revelation.
- **Combat, health, or survival mechanics.** Challenge is logic and constraint only.
- **Infinite descent or endless mode.** Fixed number of decks; the end is real.
- **HTTP API, cloud save, or online features.** Single executable, local state only (in current scope).

---

## Assumptions and Dependencies

- **Tech stack:** Go 1.24+, Ebiten; single executable; in-memory state; no database.
- **Architecture:** Engine in `pkg/engine/`, game in `pkg/game/`; state in `pkg/game/state/`; deck-based progression; renderer abstraction with Ebiten implementation.
- **Specs and invariants:** Power system and level layout follow `specs/power-system.md` and `specs/level-layout-and-solvability.md`; solvability invariants (I1–I7) and layout rules (R1–R8) are assumed by design and implementation.
- **Content and narrative:** Terminal copy and docking-station message are authored to convey futility and finality; tone and naming follow specs (functional, cold, no life-support restoration).
- **Platform:** Desktop Windows and Linux; keyboard primary input; 20–30 minute target session and replayability via procedural variation.
