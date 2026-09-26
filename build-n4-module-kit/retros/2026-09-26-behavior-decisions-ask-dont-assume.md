<!-- review-status: pending -->
# 2026-09-26 · kit · behavior-decisions-ask-dont-assume

**Session**: PANCCADIA León, 2026-09-26: CompPan 2.6.0/2.6.1, DashboardPan 2.8.0/2.8.1, ColdRoomPan 2.3.0 in progress. The user asked for a kit rule so that field behaviors are ASKED with options, never assumed.
**Delta count**: 6

## What happened
In one session, five field defects shared a single root cause: the code chose a
field behavior that nobody had stated, and nobody asked the user.
1. **T3 auto-lockout (CompPan 2.3.x -> 2.6.0).** It assumed "commanded ON + ~0 A for N
   minutes = failed compressor". Compressor 2 has a mechanical suction cut-in (it only
   starts at 35 psi), so it is legitimately held off while commanded. Live at 03:20-03:27,
   `condenser2Fault = true` on a healthy unit, which would have been auto-locked ~03:30
   if the option had stayed on.
2. **The auto-lockout wrote the HOA mode slot.** It assumed that writing
   `condenserNMode = OFF` was an acceptable way to show and clear the lock. That slot is
   a link target fed by the dashboard (`comp3Mode -> condenser3Mode`, which propagates
   only on change), so the dashboard showed AUTO, pressing AUTO did nothing, and a
   station restart re-armed a burned compressor.
3. **Room demand.** It assumed "room calls for compressor = any evaporator has a run
   order" (`cooling = OR(runCmd)`). During the 03:00 defrosts, valves were closed with
   runCmd still true: demand was 3 with 1 open valve, and suction fell 26.2 -> 23.7 psig.
4. **HMI target.** It assumed a modern browser. The panel runs Chrome 83, so `inset` and
   flex `gap` broke the login overlay on two separate days (HMI login, then the second
   login modal).
5. **Write lock UX.** It assumed "no session => disable Save buttons". A disabled button
   fires no click, so Configuración never offered the login.
The writer also made a design choice during this very retro's session without asking
(a new `compressorCall` slot + 4 relinks vs. changing `cooling` itself). That is the
same failure mode at the orchestration level.

The existing 2026-09-25 retro Δ2 (restate a control-mode change in field terms and
confirm it with the field technician) covers only ambiguous PHRASES in control-mode
changes. The kit has no rule that makes every unstated behavior a question to the user,
answered by choosing from options or giving their own.

## Evidence
- Comp2 mechanical cut-in 35 psi (user, engineers' rule). Live `condenser2=true, amps2=0.09, condenser2Fault=true, suctionPressure 23.7-26.0` at 03:20-03:27. `[ev: odd/tasks/coldroom-compressor-call-valve-open.md]`
- Mode-slot write + link mismatch: live `condenser3Mode=2` vs dashboard `comp3Mode=0`. `[ev: odd/tasks/comppan-autolock-keeps-mode.md]`
- Demand vs open valves: live 03:20 table (runCmd=true, valveOut=false, resistanceOut=true in 3 units; `demand=3`). `[ev: odd/tasks/coldroom-compressor-call-valve-open.md]`
- Chrome 83 on the HMI (user-agent screenshot) and `#__cl{inset:0}` fix. `[ev: f12f0fe]`
- Disabled Save buttons never open the login gate. `[ev: f12f0fe]`
- No ask-the-user rule in `skill/SKILL.md`, `BUILD-LOOP.md`, `METHODOLOGY.md` or `ORCHESTRATION.md` (grep for "ask the user|sequence of operation|behavior decision|assumption" -> 0 hits). `[ev: grep 2026-09-26]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **Hard gate "Behavior decisions" before the first source write of any rt control, protection, automatic action, or operator-facing UI change.** The parent lists every behavior the code must decide that the user has not stated explicitly. Each one becomes a question to the user with 2-4 concrete options: the recommended one first and marked, each with its field consequence in plain terms, and always a free "your own option" answer. No behavior on that list may be coded until the user has answered. The answers go into a `## Behavior decisions` table in the feature doc (question, options shown, answer, who, date). Extends 2026-09-25 Δ2 from ambiguous phrases to ALL unstated behaviors. | `BUILD-LOOP.md` § `1. Design` + `skill/SKILL.md` § `Hard Rules (non-negotiable — full detail in the kit)` | `[ev: retro 2026-09-25-panccadia-commissioning-lessons.md Δ2]` |
| Δ2 | **A standing question catalog, by domain, seeded from today's defects**, so the list in Δ1 is generated rather than remembered. **Equipment and protections:** what mechanical/field protections can stop each unit while it is commanded (LP/HP switch, overload, oil), and at what values? Which states are "legitimately off while commanded"? **Demand:** what exactly counts as a zone/room "asking" for the plant (temperature, open valve, fan, any unit) during defrost, drip, restart and HAND? **Automatic actions:** for anything the control does by itself (lockout, trip, mode change), how is it shown, how does the operator clear it, what happens on a station restart, and is it on or off by default? **Operator links:** which slots are link targets fed from the dashboard or other logic (the automatic action must not write them)? **HMI:** which browsers/panels and versions display the page? **Write UX:** what does the operator see and do when a write needs a login? **Rollback:** how is a bad version recovered (see roll-forward retro)? | `METHODOLOGY.md` § `Domain correctness` (new "Behavior question catalog") | `[ev: odd/tasks/comppan-autolock-keeps-mode.md]`; `[ev: odd/tasks/coldroom-compressor-call-valve-open.md]`; `[ev: f12f0fe]` |
| Δ3 | **Writers return decision gaps; they never pick a behavior.** The delegated-writer prompt template gets a fixed clause: "If the code must choose a behavior that is not in the feature doc's Behavior decisions table, STOP and return the question with options; do not choose." The parent relays the question to the user in the Δ1 format. | `ORCHESTRATION.md` § `3. Delegation triggers` (writer prompt template) | `[ev: this retro § What happened, last paragraph]` |
| Δ4 | **Assumption register at close.** Any behavior still decided without an explicit answer (because the user deferred it, or it was discovered late) is listed in the feature doc under `## Assumptions still open`, and repeated to the user in the close message as "Assumptions I made — confirm or change". An empty register is stated explicitly. | `BUILD-LOOP.md` § `7. Retro + close (HARD close gate — not optional)` | `[ev: odd/tasks/comppan-auto-lock-indicator.md]` |
| Δ5 | **Scenario-based live check for each behavior decision after deploy.** For each row of the Behavior decisions table, name the live scenario that exercises it (e.g. "03:00 defrost window: rooms with all valves closed must not call the plant") and read it via the read-only oBIX audit when that scenario occurs. Several of today's defects were only visible in a specific window (defrost, low suction), not at an arbitrary read. | `BUILD-LOOP.md` § `6.a Post-deploy verification (after hot module reload or station restart)` | `[ev: odd/tasks/coldroom-compressor-call-valve-open.md]` |
| Δ6 | **Interaction format for behavior questions.** Use the native multiple-choice question UI when it is available: at most 4 related questions per round, each with 2-4 options, the recommended one first and labeled "(Recomendado)", each option's field consequence in the description, and the free-text answer always allowed. Otherwise, use a numbered plain-text list with the same content. Write the question in the user's language and in field terms (valves, compressors, psi, rooms), not slot names alone. | `skill/SKILL.md` § `Execution Steps` | `[ev: user request 2026-09-26]` |

## Lessons
- Every field defect today was a behavior the code chose silently; tests proved the code did what was assumed, not what the plant needs.
- Mechanical protections and link topology are facts only the field knows; they must be asked, not inferred from code.
- A question with options and a recommended default costs a minute; an assumed behavior cost a night of live diagnosis.
- Writers must surface choices, not make them; the parent owns the question to the user.
- An automatic action needs four answers before it is coded: how it shows, how it clears, what a restart does, and what the default is.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-26-behavior-decisions-ask-dont-assume.md | kit | 2026-09-26 | pending | 6 |`
