# Epics vs GDD Review

**Reference:** `_bmad-output/gdd.md`
**Reviewed:** `_bmad-output/planning-artifacts/epics.md`
**Date:** 2026-02-21

---

## 1. Summary

The epics document is **largely aligned** with the GDD: FR/NFR coverage matches the GDD’s mechanics, win conditions, power system, and narrative intent. **Gaps and inconsistencies** are mainly:

- **Document title** still uses legacy name "TheDarkCastle".
- **NFR2 (player identity)** does not state the GDD exception (docking station may refer to player as robot).
- **Story 4.4** does not specify the docking station, shutdown, message content, or robot revelation.
- **Story 1.1** AC uses "my unit" in a way that could encourage in-game use of "unit" for the player, which the GDD forbids during play.

Recommended updates are listed in §4 and applied in §5 (where applicable).

---

## 2. Epic-by-epic alignment

### Epic 1: Deck Exploration and Structure

| GDD element | Epics | Status |
|-------------|--------|--------|
| Grid, rooms, corridors, functional layers | FR1, FR2, FR10, FR23, FR24; Stories 1.1–1.5 | ✓ Aligned |
| No revisit; forward-only; fixed decks | FR10; Story 1.5 AC | ✓ Aligned |
| Start room doors powered at init | FR23; Story 1.2, 1.3 | ✓ Aligned |
| Room connectivity (R8, single component) | FR24; Story 1.4 | ✓ Aligned |

**Issue:** Story 1.1 AC says "**my unit** moves to an adjacent walkable cell." The GDD says: during the game, never refer to the player explicitly as "unit," "robot," or "system" (subtle hints only). So in-game copy (e.g. "Unit moved") would conflict. Recommendation: treat "my unit" in the AC as a design-doc label only, and add a note that **in-game messaging must not name the player as unit/robot/system** except at the docking station (see Story 4.4).

---

### Epic 2: Power Grid and Room Control

| GDD element | Epics | Status |
|-------------|--------|--------|
| 100 W per generator, consumption (doors 10 W, CCTV 10 W, puzzles 3 W) | FR11; Stories 2.1, 2.4 | ✓ Aligned |
| Room power, terminals (own + adjacent), InitRoomPower, EnsureSolvabilityDoorPower | FR12, FR25; Story 2.2 | ✓ Aligned |
| Terminal Powered flag, restore nearby | FR13; Story 2.3 | ✓ Aligned |
| Short-out, protected room, deterministic order | FR14; Story 2.4 | ✓ Aligned |
| Lighting from GetAvailablePower(), no consumption | FR15; Story 2.5 | ✓ Aligned |
| Unpowered = red; powered generator = green | FR16; Story 2.6 | ✓ Aligned |

No conflicts with the GDD. Epic 2 is **fully aligned**.

---

### Epic 3: Gates, Keys, and Solvability

| GDD element | Epics | Status |
|-------------|--------|--------|
| Win: exit + all generators + all hazards cleared | FR17; Story 3.4 | ✓ Aligned |
| Doors need room power (+ keycard if locked) | FR18; Stories 3.1, 3.2 | ✓ Aligned |
| Gatekeeper / deadlock prevention (EnsureSolvabilityDoorPower) | FR19; Story 3.5 | ✓ Aligned |
| Keycard chain, no cycles | FR20; Story 3.2 | ✓ Aligned |
| Hazard chain, no deadlock | FR21; Story 3.3 | ✓ Aligned |
| Generators and batteries reachable | FR22; Story 3.4 | ✓ Aligned |
| No failure state (GDD) | Not an FR; no story contradicts | ✓ Aligned |

No conflicts. Epic 3 is **fully aligned**.

---

### Epic 4: Narrative, Tone, and Completion

| GDD element | Epics | Status |
|-------------|--------|--------|
| Terminals as narrative surface; technical, dry; implicit narrative | FR5, NFR1; Story 4.1 | ✓ Aligned |
| Room/item naming; life support OFFLINE | FR6, FR7; Story 4.2 | ✓ Aligned |
| Progressive decay; "fights back"; tone | FR8, NFR3; Story 4.3 | ✓ Aligned |
| Final deck: minimal, no lift, completion not victory/failure/escape | FR9, NFR4; Story 4.4 | ✓ Aligned |
| **Docking station in last room** | — | ✗ Missing in epics |
| **Shutdown + message (no comms, no lifeforms, last semi-sentient, conserve logs, futility/finality)** | — | ✗ Missing in epics |
| **Robot revelation only at docking station** | NFR2 says no explicit "you are a robot" | ⚠ NFR2 needs exception |

**Gap:** Story 4.4 only mentions "optional final system line" and does not describe:

- The **docking station** as an interaction in the last room.
- The **shutdown** of the player.
- The **message content** (no communication, no lifeforms, last semi-sentient, no way to replenish power, shut down to conserve logs; tone: futility and finality).
- The **identity revelation**: the only place the game may refer to the player as a robot.

NFR2 currently says identity is revealed indirectly with no explicit "you are a robot." The GDD adds: **exception at the docking station**—there it is acceptable and necessary to refer to the player as a robot. The epics should state this exception (e.g. in NFR2 or in Story 4.4).

---

## 3. Cross-cutting checks

| Topic | GDD | Epics | Status |
|-------|-----|--------|--------|
| No failure state | Explicit throughout | No FR for "no game over"; no story contradicts | ✓ OK |
| Revisit / forward-only | Out of Scope; Level Progression | Story 4.4 "no revisit" | ✓ Aligned |
| Player identity during play | Never "unit/robot/system"; subtle hints only | NFR2; Story 1.1 "my unit" | ⚠ See §2 and §4 |
| Player identity at end | Docking station may and should refer to robot | Not in Story 4.4 or NFR2 | ✗ Add |
| Late-game unit decay (optional) | Progression; Horror | Additional Requirements | ✓ Aligned |
| Project name | The Dark Station | TheDarkCastle in title/overview | ✗ Fix title |

---

## 4. Recommended changes to epics.md

1. **Title and overview**
   - Replace "TheDarkCastle" with "The Dark Station" (and "TheDarkStation" where appropriate) in the title and overview.

2. **NFR2 (player identity)**
   - Add an explicit exception for the final moment, e.g.:
     "**Exception:** In the final room, at the docking station, the game may—and should—refer to the player as a robot; that is the intended revelation (player finds out as the last action of the game on first run)."

3. **Story 4.4: Final Deck and Completion Ending**
   - Add acceptance criteria (or expand existing) to include:
     - In the last room, the player encounters a **docking station** (interaction/cell).
     - Interacting with it **shuts the player down** and triggers the completion ending.
     - The **message** conveys: no communication for a long time; no lifeforms detected; player is the last semi-sentient thing; no way to replenish power; unit shut down to conserve the station’s computer logs; tone of futility and finality.
     - This is the **only** place the game may refer to the player explicitly as a robot (identity revelation).
   - Keep or refine "optional final system line" as one possible element of that message (e.g. "NO FURTHER WORK REQUESTS DETECTED" / "ENERGY GRADIENT EQUALIZED") if desired.

4. **Story 1.1 (optional but recommended)**
   - Either rephrase "my unit moves" to "the player moves" / "my character moves" in the AC, or add a note: "In-game messaging must not label the player as 'unit', 'robot', or 'system' during play (see GDD: Player identity and in-game language; exception: docking station in Story 4.4)."

---

## 5. Conclusion

- **Epics 2 and 3:** No changes needed for GDD alignment.
- **Epic 1:** Clarify that Story 1.1’s "my unit" does not imply in-game use of "unit" (or rephrase + add note).
- **Epic 4 / Story 4.4 and NFR2:** Update to include the docking station, shutdown, message content, and robot-revelation exception so the epics match the GDD and can be implemented without ambiguity.

Applying the title fix, NFR2 exception, and Story 4.4 expansion in the epics file next.
