---
stepsCompleted: ["step-01-validate-prerequisites", "step-01-complete", "step-02-design-epics", "step-03-create-stories", "step-04-final-validation"]
inputDocuments:
  - specs/gdd.md
  - docs/architecture.md
  - specs/power-system.md
  - specs/level-layout-and-solvability.md
---

# TheDarkCastle - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for TheDarkCastle, decomposing the requirements from the PRD, UX Design if it exists, and Architecture requirements into implementable stories.

## Requirements Inventory

### Functional Requirements

FR1: The station is finite and composed of connected decks stacked vertically and laterally; the player explores decks sequentially.
FR2: Each deck represents a functional layer (e.g. Habitation, Research, Logistics, Power Distribution, Emergency Systems, Core Infrastructure).
FR3: Core gameplay loop: enter a new deck → assess failing systems → restore limited power → route power to critical subsystems → accept instability elsewhere → progress deeper.
FR4: Power is the central mechanic: the player performs load balancing, priority assignment, and system reactivation; power is not "generated" but misrouted/degraded/insufficient.
FR5: Maintenance terminals are the primary narrative surface; they report (technical, dry language), do not speak; options and data change as decks progress.
FR6: Rooms have functional, cold, slightly outdated names; life support systems are always OFFLINE/FAILED/PERMANENTLY DISABLED (no restoration).
FR7: Items are tools left behind; naming is descriptive, impersonal, system-centric; some reference systems that no longer exist but still function.
FR8: Progressive decay: as decks progress, generator output decreases, power costs increase, automation overrides player choices, stability becomes impossible; the station "fights back" (doors relock, systems shut down, power reroutes).
FR9: Final deck: minimal rooms and systems, barely functional power; the lift has no destination; game ends with completion (job done, job no longer matters), not victory/failure/escape.
FR10: Procedural generation: each deck is generated when first entered; the station has a fixed number of decks; the end is real (no infinite descent).
FR11: Global power grid: supply from powered generators (100 W each when batteries inserted), consumption from doors (10 W per room when room doors powered), CCTV (10 W per terminal when room CCTV powered), solved puzzle terminals (3 W each); maintenance terminals and standard lighting do not consume power.
FR12: Room power: per-room state for door power and CCTV/hazard-control power; toggled only at maintenance terminals (own room and adjacent rooms); start room doors powered at init; gatekeeper rooms that would otherwise deadlock get doors powered (EnsureSolvabilityDoorPower).
FR13: Maintenance terminal power: each terminal has a Powered flag; only start room terminal(s) powered at init; "Restore power to nearby terminals" at a powered terminal powers terminals in adjacent rooms (including own room).
FR14: Overload handling: when turning ON a room's doors/CCTV would exceed supply, other rooms' doors/CCTV are auto-unpowered (short-out) in a deterministic order until within budget; the room just turned on is protected; passive overload shows one-time warning per cycle.
FR15: Lighting and exploration: lights do not consume power; visibility depends on GetAvailablePower() > 0 and visit state; when power ≤ 0, cells beyond a small radius of the player can darken (discovered/visited cleared).
FR16: Unpowered visual: any unpowered power-dependent cell (doors, maintenance terminal, CCTV, hazard control) uses the same red background to show unusable/off; powered generators may use distinct background (e.g. green).
FR17: Win conditions: player reaches exit cell, all generators powered (batteries inserted), all blocking hazards cleared; any gate blocking progress must be satisfiable without deadlock.
FR18: Movement gates: doors require room doors powered and (if locked) keycard; blocking hazards require fixed (control or item); generators block cell until powered for exit.
FR19: Door-power and control deadlock prevention: for every gatekeeper room R (every path from start to exit goes through R), either R's doors are initially powered or a maintenance terminal that can power R exists in a room adjacent to R and reachable from start without entering R; locked doors treated as impassable when computing reachability.
FR20: Keycard chain: keycards placed only in reachable set before placing the doors that lock that room; no circular keycard dependency.
FR21: Hazard chain: no hazard blocks the only path to its control or required item.
FR22: Generators and batteries: every generator reachable; enough batteries placed in reachable locations to power all generators.
FR23: Start room doors powered at init so the player can leave.
FR24: Room connectivity: every named room's walkable cells (excluding blocking entities) form a single connected component; placement of blocking entities must not disconnect room (all doorways mutually reachable within room).
FR25: Level setup order (power-related): InitRoomPower → placement (hazards, furniture, puzzles, maintenance terminals) → EnsureSolvabilityDoorPower → InitMaintenanceTerminalPower → player at start cell; reset/advance level re-runs setup so room and terminal power return to initial state.

### NonFunctional Requirements

NFR1: Narrative must be implicit—never explain the story directly; only show what systems no longer allow.
NFR2: Player identity (UNIT / MAINTENANCE ENTITY / AUTONOMOUS SYSTEM) is revealed indirectly; no explicit "you are a robot" or references to food/rest/sleep.
NFR3: Tone and feel: quiet dread, loneliness, ritual without purpose; design succeeds when the player feels these.
NFR4: Ending philosophy: completion, not victory, failure, or escape; correct/incorrect/action converge to same outcome.
NFR5: Deterministic short-out order when overload occurs (e.g. by room name, then doors then CCTV) so behaviour is reproducible.
NFR6: One-time overload warning per cycle when consumption already exceeds supply (PowerOverloadWarned).
NFR7: Technical stack and structure: Go 1.24, Ebiten; single executable; engine primitives in pkg/engine/, game logic in pkg/game/; state in pkg/game/state/; deck-based forward-only progression; no HTTP API; tests via go test, *_test.go; Make build/test/codestyle.

### Additional Requirements

- Architecture: Single Go executable, Ebiten renderer; entry main.go → gettext, Ebiten, game loop (menu → game → quit). State in pkg/game/state/; deck/setup in pkg/game/deck/, pkg/game/setup/. Renderer abstraction in pkg/game/renderer/ with Ebiten implementation in renderer/ebiten/.
- Architecture: No database; all state in-memory (Game, DeckState, GameCellData, entities). Relationships by reference (pointers).
- Architecture: Development workflow: make or go run .; dev start deck LEVEL=N or -level N; make build, make test, make codestyle; i18n via po/default.pot and make mo.
- Power system: Supply/consumption calculated and stored (PowerSupply, PowerConsumption); GetAvailablePower(); ShortOutIfOverload(protectedRoomName) called after applying room power toggle ON; UpdatePowerSupply(), CalculatePowerConsumption(), UpdateLightingExploration in appropriate order.
- Power system: Messages/callouts for overload (passive and short-out), unpowered doors/CCTV/terminal, generator interaction (supply/consumption/available).
- Level layout: Invariants I1–I7 and rules R1–R8 (level generation and layout) must be followed; EnsureSolvabilityDoorPower runs after maintenance terminal placement; locked door cells impassable for reachability; placement checks to avoid room disconnection (R8).
- Level layout: Solvability validation recommended (e.g. devtools/CI): given dumped level, verify exit reachable and win conditions satisfiable; optionally verify no gatekeeper room with unpowered doors and no adjacent-reachable maintenance terminal.
- GDD: Late-game player decay (optional mechanical flavour): inputs occasionally require repetition, movement pauses, terminal options fail to register, actions complete with delay; system messages shift from STATION POWER UNSTABLE to UNIT RESPONSE DELAYED to UNIT POWER RESERVE CRITICAL.

### FR Coverage Map

FR1: Epic 1 - Station finite, connected decks, sequential exploration
FR2: Epic 1 - Each deck a functional layer
FR3: Epic 4 - Core gameplay loop (enter → assess → restore → route → progress)
FR4: Epic 2 - Power as central mechanic
FR5: Epic 4 - Maintenance terminals as narrative surface
FR6: Epic 4 - Room naming; life support offline
FR7: Epic 4 - Items as tools left behind; naming style
FR8: Epic 4 - Progressive decay across decks
FR9: Epic 4 - Final deck; lift no destination; completion ending
FR10: Epic 1 - Procedural generation; deck on first entry; fixed decks
FR11: Epic 2 - Global power grid (supply/consumption)
FR12: Epic 2 - Room power (doors/CCTV, terminals, solvability)
FR13: Epic 2 - Maintenance terminal power and restore nearby
FR14: Epic 2 - Overload and short-out behaviour
FR15: Epic 2 - Lighting and exploration from available power
FR16: Epic 2 - Unpowered visual (red background)
FR17: Epic 3 - Win conditions (exit, generators, hazards)
FR18: Epic 3 - Movement gates (doors, hazards, generators)
FR19: Epic 3 - Door-power and control deadlock prevention
FR20: Epic 3 - Keycard chain (no cycles)
FR21: Epic 3 - Hazard chain (no deadlock)
FR22: Epic 3 - Generators and batteries reachable
FR23: Epic 1 - Start room doors powered at init
FR24: Epic 1 - Room connectivity (single component, placement rules)
FR25: Epic 2 - Level setup order (power-related)

## Epic List

### Epic 1: Deck Exploration and Structure
Player can explore a single generated deck: move through grid, rooms, and corridors, and reach the exit. Foundation for all other deck gameplay.
**FRs covered:** FR1, FR2, FR10, FR23, FR24

### Epic 2: Power Grid and Room Control
Player can interact with the power system: generators and batteries, room power (doors/CCTV), maintenance terminals (powered state and restore nearby), overload/short-out, and lighting tied to available power.
**FRs covered:** FR4, FR11, FR12, FR13, FR14, FR15, FR16, FR25

### Epic 3: Gates, Keys, and Solvability
Player can overcome gates (doors, hazards, generators) in a solvable order; win conditions are reachable; no control/keycard/hazard/generator deadlocks.
**FRs covered:** FR17, FR18, FR19, FR20, FR21, FR22

### Epic 4: Narrative, Tone, and Completion
Player experiences the intended narrative and tone: terminals as narrative surface, room/item naming, progressive decay across decks, final deck with no lift destination, and completion ending (not victory/failure/escape).
**FRs covered:** FR3, FR5, FR6, FR7, FR8, FR9 (+ NFR1–NFR4)

---

## Epic 1: Deck Exploration and Structure

Player can explore a single generated deck: move through grid, rooms, and corridors, and reach the exit. Foundation for all other deck gameplay. **FRs covered:** FR1, FR2, FR10, FR23, FR24.

### Story 1.1: Grid and Movement

As a player,
I want to move through a grid of cells (rooms and corridors),
So that I can explore the deck.

**Acceptance Criteria:**

**Given** a generated level with walkable and blocking cells  
**When** I press movement keys (N/S/E/W or arrows or vim-style)  
**Then** my unit moves to an adjacent walkable cell  
**And** walls and blocking entities (furniture, terminals, hazards, generators) prevent movement

### Story 1.2: Rooms and Corridors

As a player,
I want the level to consist of named rooms connected by corridors,
So that the deck has clear spatial structure.

**Acceptance Criteria:**

**Given** level generation  
**When** the level is created  
**Then** each room has a name and a set of walkable cells  
**And** corridors connect rooms  
**And** each deck represents a functional layer (e.g. Habitation, Research, Power Distribution)

### Story 1.3: Start Room and Exit

As a player,
I want a designated start cell and an exit cell,
So that I can begin and complete the deck.

**Acceptance Criteria:**

**Given** level setup  
**When** the level is ready  
**Then** I start at the start cell in the start room  
**And** the start room's doors are powered at init so I can leave (FR23)  
**And** an exit cell exists and is reachable when win conditions are met

### Story 1.4: Room Connectivity

As a player,
I want every room's walkable area to be one connected region,
So that I can reach all doorways and controls in a room without leaving it.

**Acceptance Criteria:**

**Given** placement of blocking entities (furniture, terminals, hazards, etc.)  
**When** a level is generated  
**Then** no room is disconnected (all doorways in that room are mutually reachable via walkable cells within the room)  
**And** placement follows rule R8 (prevent room disconnection)

### Story 1.5: Procedural Deck Generation

As a player,
I want each deck to be generated when I first enter it and the station to have a fixed number of decks,
So that the station feels finite and the end is real.

**Acceptance Criteria:**

**Given** I advance to a new deck  
**When** I enter it for the first time  
**Then** that deck is generated (e.g. BSP + placement)  
**And** the station has a fixed number of decks (no infinite descent)  
**And** the final deck exists and is reachable

---

## Epic 2: Power Grid and Room Control

Player can interact with the power system: generators and batteries, room power (doors/CCTV), maintenance terminals (powered state and restore nearby), overload/short-out, and lighting tied to available power. **FRs covered:** FR4, FR11, FR12, FR13, FR14, FR15, FR16, FR25.

### Story 2.1: Generators and Batteries

As a player,
I want to find generators and insert batteries into them,
So that I can supply power to the level.

**Acceptance Criteria:**

**Given** generators on the level with BatteriesRequired  
**When** I interact with a generator and have enough batteries in inventory  
**Then** I can insert batteries (insertion is permanent for that level)  
**And** when BatteriesInserted >= BatteriesRequired the generator is powered and supplies 100 W  
**And** PowerSupply is the sum of all powered generators (UpdatePowerSupply)

### Story 2.2: Room Power (Doors and CCTV)

As a player,
I want to power doors and CCTV per room via maintenance terminals,
So that I can open doors and use CCTV/hazard controls in that room.

**Acceptance Criteria:**

**Given** RoomDoorsPowered and RoomCCTVPowered per room  
**When** I use a maintenance terminal (own room or adjacent room)  
**Then** I can toggle doors and CCTV for those rooms  
**And** start room doors are powered at init (InitRoomPower)  
**And** gatekeeper rooms that would otherwise deadlock get doors powered (EnsureSolvabilityDoorPower)  
**And** level setup order is: InitRoomPower → placement → EnsureSolvabilityDoorPower → InitMaintenanceTerminalPower (FR25)

### Story 2.3: Maintenance Terminal Power

As a player,
I want only some maintenance terminals to be usable until I restore power from another terminal,
So that I must route power through the station.

**Acceptance Criteria:**

**Given** each maintenance terminal has a Powered flag  
**When** the level starts  
**Then** only start room terminal(s) are powered (InitMaintenanceTerminalPower)  
**And** I can open the maintenance menu only at a powered terminal  
**And** at an unpowered terminal I see a message to restore power from another maintenance terminal  
**And** "Restore power to nearby terminals" at a powered terminal powers all terminals in adjacent rooms (including own room)

### Story 2.4: Power Consumption and Overload

As a player,
I want power consumption to be calculated and overload to cause a short-out,
So that I must balance supply and demand.

**Acceptance Criteria:**

**Given** PowerSupply (powered generators × 100 W) and PowerConsumption (doors 10 W per room when powered, CCTV 10 W per terminal when room CCTV on, solved puzzles 3 W each)  
**When** I toggle a room's doors or CCTV ON and the new consumption would exceed PowerSupply  
**Then** ShortOutIfOverload(protectedRoomName) runs: other rooms' doors and CCTV are auto-unpowered in a deterministic order until PowerConsumption ≤ PowerSupply  
**And** the room I just turned on stays on (protected)  
**And** the player is told that other systems shorted out  
**And** when consumption already exceeds supply (passive), a one-time warning per cycle is shown (PowerOverloadWarned)

### Story 2.5: Lighting and Exploration

As a player,
I want lighting to depend on available power and whether I have visited a cell,
So that darkness reflects power state.

**Acceptance Criteria:**

**Given** GetAvailablePower() = PowerSupply - PowerConsumption and visit state per cell  
**When** GetAvailablePower() > 0 and I have visited a cell  
**Then** lights are on for that cell (LightsOn, Lighted; cell stays discovered)  
**And** when GetAvailablePower() <= 0, cells beyond a small radius of the player can darken (discovered/visited cleared)  
**And** lighting does not consume power

### Story 2.6: Unpowered Visual Feedback

As a player,
I want unpowered power-dependent cells to look distinct,
So that I know what needs power.

**Acceptance Criteria:**

**Given** doors, maintenance terminals, CCTV terminals, hazard controls  
**When** a door is unpowered (RoomDoorsPowered[room] false), or a maintenance terminal is unpowered (Powered false), or room CCTV is off (RoomCCTVPowered[room] false)  
**Then** that cell uses the same red background (e.g. dark red / hazard style)  
**And** powered generators may use a distinct background (e.g. green)  
**And** walls and corridor cells do not change background based on power

---

## Epic 3: Gates, Keys, and Solvability

Player can overcome gates (doors, hazards, generators) in a solvable order; win conditions are reachable; no control/keycard/hazard/generator deadlocks. **FRs covered:** FR17, FR18, FR19, FR20, FR21, FR22.

### Story 3.1: Doors and Room Power

As a player,
I want doors to require room power to pass through,
So that I must use maintenance terminals to open them.

**Acceptance Criteria:**

**Given** a door cell belonging to a room  
**When** RoomDoorsPowered[roomName] is false  
**Then** I cannot pass through and see a message to restore power via the maintenance terminal  
**And** when true I can pass (subject to lock/keycard if door is locked)

### Story 3.2: Locked Doors and Keycards

As a player,
I want to find keycards to unlock locked doors,
So that I can progress through the deck.

**Acceptance Criteria:**

**Given** locked doors for a room requiring a keycard  
**When** I have the keycard for that room  
**Then** I can pass through the door when room doors are powered  
**And** keycards are placed only in the reachable set before placing the doors that lock that room (no keycard cycles) (FR20)

### Story 3.3: Blocking Hazards and Controls

As a player,
I want to clear blocking hazards via hazard controls or items,
So that I can progress.

**Acceptance Criteria:**

**Given** blocking hazards that block movement or win condition  
**When** I use the hazard control (in a room with RoomCCTVPowered) or use the required item  
**Then** the hazard is fixed and I can pass  
**And** no hazard blocks the only path to its control or required item (FR21)

### Story 3.4: Generators and Exit Win Condition

As a player,
I want all generators powered (batteries inserted) and all blocking hazards cleared to be required to complete the deck,
So that I must restore power and clear hazards to win.

**Acceptance Criteria:**

**Given** win conditions (reach exit, all generators powered, all blocking hazards cleared)  
**When** I reach the exit cell  
**Then** I can complete the deck only if all generators are powered and all blocking hazards are cleared  
**And** every generator is reachable (FR22)  
**And** enough batteries are placed in reachable locations to power all generators

### Story 3.5: Gatekeeper and Door-Power Solvability

As a player,
I want no deadlock where I must enter a room to power its doors but cannot enter without power,
So that every level is solvable.

**Acceptance Criteria:**

**Given** level setup after maintenance terminal placement  
**When** EnsureSolvabilityDoorPower runs  
**Then** for every gatekeeper room R (every path from start to exit goes through R) either R's doors are initially powered or a maintenance terminal that can power R exists in a room Q adjacent to R and reachable from start without entering R  
**And** when computing reachability without entering R, locked door cells are treated as impassable  
**And** placement and invariants I1–I7, rules R1–R8 are followed

---

## Epic 4: Narrative, Tone, and Completion

Player experiences the intended narrative and tone: terminals as narrative surface, room/item naming, progressive decay across decks, final deck with no lift destination, and completion ending (not victory/failure/escape). **FRs covered:** FR3, FR5, FR6, FR7, FR8, FR9 (+ NFR1–NFR4).

### Story 4.1: Maintenance Terminals as Narrative Surface

As a player,
I want maintenance terminals to report in technical, dry language,
So that the tone is consistent and narrative is implicit.

**Acceptance Criteria:**

**Given** maintenance terminals  
**When** I interact with them  
**Then** they report (do not "speak"); language is technical, dry, occasionally obsolete  
**And** options and reports can change as decks progress  
**And** narrative is never explained directly; only what systems no longer allow is shown (NFR1)

### Story 4.2: Room and Item Naming

As a player,
I want rooms and items to use functional, cold naming,
So that the world feels consistent.

**Acceptance Criteria:**

**Given** rooms and items in the world  
**When** I encounter them  
**Then** room names are functional, cold, slightly outdated (e.g. Cryogenic Habitation Block, Central Power Exchange)  
**And** items are "tools left behind" with descriptive, impersonal, system-centric names (e.g. Priority Override Module)  
**And** life support systems are always OFFLINE/FAILED/PERMANENTLY DISABLED (no restoration)

### Story 4.3: Progressive Decay Across Decks

As a player,
I want later decks to feel more degraded,
So that the narrative of decay is conveyed.

**Acceptance Criteria:**

**Given** deck progression (decks become older, systems more rigid)  
**When** I advance to later decks  
**Then** generator output can decrease and power costs increase  
**And** automation can override or "fight back" (doors relock, systems shut down, power reroutes)  
**And** the station obeys "dead rules" (not malicious)  
**And** tone supports quiet dread, loneliness, ritual without purpose (NFR3)

### Story 4.4: Final Deck and Completion Ending

As a player,
I want the final deck to have no lift destination and the game to end with completion,
So that the ending matches the design philosophy.

**Acceptance Criteria:**

**Given** the final deck (fixed number of decks; end is real)  
**When** I reach the final deck  
**Then** it has minimal rooms and systems and barely functional power  
**And** the lift has no destination (no advance option; forward-only, no revisit)  
**And** the game ends with completion (job done, job no longer matters), not victory, failure, or escape (NFR4)  
**And** optional final system line (e.g. "NO FURTHER WORK REQUESTS DETECTED" or "ENERGY GRADIENT EQUALIZED")
