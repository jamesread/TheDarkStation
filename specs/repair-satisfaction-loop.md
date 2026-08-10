# Repair Satisfaction Loop

## Intent

Repairs should make the player feel that they understood a failing machine,
predicted the effect of an intervention, and caused the station to respond.
Longer objective lists are not a substitute for that causal understanding.

The station never praises or addresses the player. It acknowledges agency
through changed telemetry, machinery, lighting, sound, routes, and persistent
world state.

## Core loop

1. **Observe** — encounter a physical symptom or inhibited device.
2. **Diagnose** — inspect local telemetry or a maintenance terminal.
3. **Predict** — learn what downstream state should change after the repair.
4. **Act** — perform an operation appropriate to the failed system.
5. **Confirm** — see and hear the predicted state transition.
6. **Exploit** — use the restored system to continue the repair chain.

The player should be able to explain both why an operation was needed and what
it changed. Dependencies therefore report evidence, not quest instructions.

## Knowledge rules

- Unseen equipment and rooms are never named.
- Inspecting a dependent device may identify the failed subsystem, but only a
  previously observed prerequisite may be identified by device and room.
- Maintenance diagnostics obey the same knowledge boundary as local readouts.
- Remote pulses and objective text must not reveal unknown map cells.
- Hints describe symptoms until the player has gathered enough evidence for a
  more specific diagnosis.

## Spatial and pacing rules

- Put the inhibited dependent on the likely route before its prerequisite so
  the player encounters a question before its answer.
- Keep the prerequisite farther from entry, but bound the return journey.
- Keep an individual chain to two to four meaningful deductions.
- Maintain at least one productive action: inspect, trace, test, repair, or
  revisit changed telemetry.
- Repairs remain safely completable when discovered out of sequence; early
  initiative is valid player agency, not an error.

## Systemic acknowledgement

A completed repair should layer several restrained responses:

- the repaired device changes state and pulses locally;
- newly actionable, already-known dependents change from inhibited to ready;
- dry telemetry records the causal transition;
- sound distinguishes operation from interlock release;
- affected lights, hazards, routes, and lift state update immediately;
- the objective display describes the resulting opportunity, not just a
  decrementing counter.

Do not move the camera to remote changes or narrate unseen consequences.

## First vertical slice

Deck 3 uses the existing Pressure Valve → Power Coupler dependency:

1. The nearer coupler reports `COUPLER DRIVE INHIBITED`.
2. Its local diagnosis reports `PRESSURE CONTROL INPUT OUT OF TOLERANCE`.
3. The deeper pressure valve predicts that bleeding auxiliary pressure will
   release the coupler interlock.
4. Completing the valve changes the known coupler to ready and reports
   `COUPLER INTERLOCK RELEASED`.
5. Completing the coupler makes lift transfer available when other local gates
   are already resolved.

Later repair types should adopt the same contract by providing type-pair
telemetry and a concrete downstream state transition. The progression
simulator continues to model legality through `PrereqIDs`; observation is a
player-understanding layer and never creates a new failure state.
