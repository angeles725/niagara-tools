<!-- review-status: pending -->
# 2026-09-26 · kit · change-tier-time-budgets

**Session**: PANCCADIA León, ODD feature `comppan-auto-lock-indicator` (panccadia-leon branch `feat/comppan-auto-lock-indicator`, doc commit `ae98e15`); user asked for time budgets so small additions ship faster without dropping test protection.
**Delta count**: 5

## What happened
The user asked for a small, additive change: a latched read-only flag per
compressor (`condenserNAutoLocked`) set by the existing T3 auto-lockout, mirrored
as a "Bloqueado: no arrancó" badge on the DashboardPan compressor card. The
change has no control-logic effect: it never changes a stage, a command, or a
mode. It still went through the full ceremony: feature doc + Engram mirror, a
delegated writer that must re-map the rt -> panel link-in -> ux reader -> rc
path from scratch, strict TDD, two group builds, and native RDD review. The
up-front estimate was 45-75 min end to end, and the user judged that too slow for
this class of change. The retro `2026-09-25-panccadia-commissioning-lessons.md`
Δ14 already proposes a blast-radius tier table (Small / Control / Structural) and
Δ12 a non-skippable check floor. This retro does not re-derive them. It adds the
missing pieces that turn the tiers into speed: a time budget per tier, a cached
recipe for the most frequent Structural sub-case, and review and build scope
proportional to the tier.

## Evidence
- Estimate given to the user before the run: writer 20-40 min + review 15-30 min + parent verify ~5 min. `[ev: odd/tasks/comppan-auto-lock-indicator.md]`
- The change is additive and alarm-only: T3 `fault[k]` requires `cmd[k]`, so it self-clears after the adapter writes MODE_OFF. The new flag only latches that edge for display (`CompressorControl.java:294,332`, `BCompressorControl.java:2687-2697`). `[ev: ae98e15]`
- The writer prompt had to ask it to "trace how `compNNoStart` gets to the HTML/JS". That path (rt slot -> `BCompressorPanel` link-in -> `DashboardReader` -> rc badge) was already mapped in earlier features but is not recorded anywhere reusable. `[ev: odd/tasks/comppan-fase2-amps-alarms.md]`
- A known trap (a READONLY link-in target cannot be linked, `LinkCheck.java:148`) lives only in session memory and in Δ1 of the 2026-09-25 retro, not in a recipe. `[ev: retro 2026-09-25-panccadia-commissioning-lessons.md Δ1]`
- Tier table exists as a proposal only, without budgets: `[ev: retro 2026-09-25-panccadia-commissioning-lessons.md Δ14, Δ12]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **Attach a wall-clock budget and a ceremony profile to each Δ14 tier, and classify BEFORE the first write.** Proposed priority scale: **P0 Cosmetic** (HMI copy, CSS, a default value, a comment; no slot/schema change) budget <=10 min: inline, no feature doc, no delegated writer, build the one touched group, no review beyond structural readback. **P1 Additive-indicator** (new READONLY/alarm-only slot + UI mirror; never read by staging/commands/modes) budget <=25 min: feature doc, one writer, focused pure test of the new latch only, touched groups built, single-lens review. **P2 Control** (changes a state machine, staging, rotation, protection, defrost) budget ~45-60 min: full Δ7 design checklist, full TDD, full RDD. **P3 Structural/Deploy-risk** (facet/unit, persistence, schema rename, boot path, link-target flags) no budget cap: P2 + Δ14 structural checks + boot smoke. The classification and its evidence (which slots are read by control logic) go in the feature doc's first line. When the run exceeds its tier budget, the overrun is recorded as retro input, not silently absorbed. | `BUILD-LOOP.md` § `0. Orient (before touching anything)` + `ORCHESTRATION.md` § `3. Delegation triggers` | `[ev: retro 2026-09-25-panccadia-commissioning-lessons.md Δ14]` |
| Δ2 | **Add a cached recipe "add a read-only indicator rt -> dashboard" with the exact file hops and known traps**, so a P1 writer does not re-map 4+ files each time: pure model field + pure test -> adapter slot (flags) -> panel link-in slot (same flags as a working sibling; never READONLY on a link target) -> ux reader field -> rc render + CSS class -> version bumps -> generated link table row (Δ9 of the 2026-09-25 retro). Per-client paths live in the client repo's feature docs; the kit holds the generic hop list plus a "find the sibling you are mirroring" step. | `skill/SKILL.md` (new recipe section) + `BUILD-LOOP.md` § `2. Build the layers` | `[ev: odd/tasks/comppan-auto-lock-indicator.md writer prompt]; [ev: retro 2026-09-25-panccadia-commissioning-lessons.md Δ1]` |
| Δ3 | **Scope the test run to the tier without dropping the floor.** In the loop, run only the touched module's pure tests: P0 runs build only, P1 runs the new latch test + that module's existing suite. Run full builds and the Δ12 non-skippable floor (build, `verify-module.sh`, `schema-risk.sh`) once at task close, not per edit. Speed comes from not re-running the unaffected suites mid-loop, never from skipping the floor. | `BUILD-LOOP.md` § `4.a Gradle task matrix` + § `5. Verify gate (before "done")` | `[ev: retro 2026-09-25-panccadia-commissioning-lessons.md Δ12]` |
| Δ4 | **Make review cost follow the tier by shaping the candidate, not by overriding native risk.** Native RDD owns lens selection; the kit cannot and must not pick lenses. What the kit can do: keep a P1 change a single small additive commit (no refactor or unrelated churn in the same commit), so native assessment has the chance to rate it passive/medium. Also batch the display-only DashboardPan commit with its rt commit into one reviewed slice instead of two review cycles. | `ORCHESTRATION.md` § `7. Pipeline` | `[ev: odd/tasks/comppan-auto-lock-indicator.md T3]` |
| Δ5 | **Parallelize independent group builds and give the writer the map instead of the mission.** The Compresores and Dashboard groups build independently, so run them concurrently (one Gradle invocation per group, both in background, then wait). The parent passes the recipe hops (Δ2) and the sibling slot name in the writer prompt, so the writer's first minutes go to the RED test, not to discovery. | `BUILD-LOOP.md` § `4. Build — the ONLY valid build` + `ORCHESTRATION.md` § `3. Delegation triggers` | `[ev: fast-build.sh per-group invocation]` |

## Lessons
- Speed comes from classifying before writing and from reusing maps, not from dropping tests: the floor (build, verify, schema-risk, one focused RED/GREEN) stays in every tier.
- An additive, alarm-only indicator is a different risk class from a control change; charging it the control-change ceremony is the main time sink.
- Any path mapped twice (rt slot -> dashboard) belongs in a recipe, not in a writer's rediscovery.
- A budget only helps if overruns are recorded; an overrun is retro input, not a failure.
- Native review risk is not the kit's to override; shape small, single-purpose candidates instead.

## Measured run (to fill at feature close)
- Writer wall-clock: _pending_
- Review lenses / wall-clock: _pending_
- Total vs P1 budget (25 min): _pending_

---
**Status**: PENDING — INDEX row appended: `| 2026-09-26-change-tier-time-budgets.md | kit | 2026-09-26 | pending | 5 |`
