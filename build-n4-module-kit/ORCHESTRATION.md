# Orchestration — research-sdd · gentle SDD · BUILD-LOOP, and how they hand off

**Contents**: §1 Roles · §2 Model table · §3 Delegation triggers · §4 Escalation gate · §5 Adopt-list · §6 Keep-from-kit · §7 Pipeline · §8 Per-run retro/ticket loop · §9 Incident journal

## 1. Roles

This kit is one of THREE tools a run may use. They are not alternatives to pick between once; a mature change
flows THROUGH all three. This file says WHEN each applies, WHICH model runs each phase, and HOW they hand off.

| Tool | Use it to | Produces | Marker of "done" |
|---|---|---|---|
| **research-sdd** | answer an open technical question against the framework (three sources: corpus + niagara-help + decompiled code) | a numbered `[CERT]`/`[INFER]` block ending in a **Kit implication** | a self-verify table with every claim marked; a named gap if not closed `[ev: research-sdd METHODOLOGY §3/§8/§11]` |
| **gentle SDD** | turn a decided change into a durable, reviewable contract | `proposal → spec → design → tasks`, ledgered attempts | spec requirements have scenarios; tasks map to spec; verify passes `[ev: CLAUDE.md SDD workflow]` |
| **BUILD-LOOP** (this kit) | build / verify / deploy an actual N4 module | a signed Java-8 jar past the verify gate + a per-module `BUILD-STATE` envelope | `verify-module.sh` passes; the HARD close gate (§7) is satisfied `[ev: BUILD-LOOP.md §5/§7]` |

## 2. Model table

Gentle-SDD phase models are fixed by the CLAUDE.md **Model Assignments** table `[ev: CLAUDE.md Model Assignments]`:

| Phase | Model | Why |
|---|---|---|
| sdd-explore | sonnet | structural reads, not architectural |
| sdd-research | sonnet | collects source-backed evidence |
| sdd-propose | **opus** | architectural decisions |
| sdd-spec | sonnet | structured writing |
| sdd-design | **opus** | architecture decisions |
| sdd-tasks | sonnet | mechanical breakdown |
| sdd-apply | sonnet | implementation |
| sdd-verify | sonnet | validation against spec |
| sdd-archive | haiku | copy + close |
| jd-judge-a / jd-judge-b / jd-fix-agent | sonnet | adversarial review + surgical fixes |
| default (generic delegation) | sonnet | fallback |

**research-sdd investigation lanes** (the corpus-block authoring that feeds the pipeline) run on **opus** — that
is a heavier reasoning task than gentle-SDD's `sdd-research` PHASE (sonnet), which only collects external
evidence. Do not conflate the two: `sdd-research` = a sonnet sub-agent phase; a research-sdd lane = an opus
authoring session. `[ev: CLAUDE.md Model Assignments; team practice campaign-8]`

## 3. Delegation triggers

Rule of thumb: **research-sdd finds the WHY, gentle SDD fixes the WHAT/contract, BUILD-LOOP produces the
artifact.** A one-line mechanical edit skips straight to BUILD-LOOP; a novel framework behavior starts in
research-sdd; a multi-file change with real ambiguity earns a gentle-SDD proposal first.

### 3.a The blast-radius tier picks the topology
Classify the change with the tier table in `BUILD-LOOP.md` §0.c BEFORE the first write — file count alone never
picks the ceremony. P0 Cosmetic stays inline (no feature doc, no delegated writer); P1 Additive indicator gets one
writer with the recipe map; P2 Control and P3 Structural get the design-shard checklist (§3.b), full TDD and native
review. The tier, its evidence and its budget go in the feature doc's first line. `[ev: retro panccadia-commissioning-lessons Δ14]` `[ev: retro change-tier-time-budgets Δ1]`

### 3.b Design-shard checklist — a gate for a new state machine or a new facet/unit
Before writing a new control state machine, or a new facet/unit, the session produces a short design-shard
artifact in the feature doc — or an explicit waiver `Design shard: waived (<reason>)`, mirroring the
`Retro: none (trivial: <reason>)` pattern. A silent skip is not allowed. The checklist:
1. States, and the entry/exit guard of each (including a validity gate on every sensor read on entry).
2. Invalid-sensor behavior in every state.
3. min-on / min-off interplay — which units a bypass applies to (only the last one, never all).
4. Config bounds — every duration/limit slot has a safe floor; `<= 0` is rejected or means disabled, never "run".
5. Boot-critical expressions — a facet's `BUnit.getUnit(...)` id, a static initializer, anything that can fail at
   station load.
6. For an automatic protective action: the latch slot, its operator-owned clear condition and its HMI rendering
   (`METHODOLOGY.md` § Domain correctness).
`[ev: retro panccadia-commissioning-lessons Δ7]`

### 3.c Writer prompt template — fixed clauses
- **Give the writer the map, not the mission:** the parent passes the recipe hops (`skill/SKILL.md` § Recipe) and
  the name of the sibling slot being mirrored, so the writer's first minutes go to the RED test, not to discovery. `[ev: retro change-tier-time-budgets Δ5]`
- **Writers return decision gaps; they never pick a behavior:** "If the code must choose a behavior that is not in
  the feature doc's Behavior decisions table, STOP and return the question with options; do not choose." The parent
  relays it to the user in the `BUILD-LOOP.md` §1 format. `[ev: retro behavior-decisions-ask-dont-assume Δ3]`
- **One feature doc, many writers:** when concurrent writers in the same checkout each add their own section to one
  shared `odd/tasks/<feature-name>.md`, resolve by APPEND per the fragment-merge rule (`METHODOLOGY.md` § Kit
  maintenance) — keep both sections, never overwrite — or name one doc owner per checkout who applies every writer's
  section text serially. `[ev: retro comppan-fase2-amps-alarms Δ8]`

### 3.d Shared working-tree git discipline
Several sessions may commit in the SAME checkout. Every writer: run `git status --short` first; stage only its own
paths (never `git add -A` / `git add .`); never reset, rebase, stash or checkout over files it did not change; on a
failed push, inspect `HEAD..origin/<branch>` before retrying, and never force over a peer's commits. `[ev: retro kit-meta-hygiene-2026-10-01 Δ1]`

## 4. Escalation gate

Boundaries between concurrent lanes are enforced by the multi-session rule: **check the tree before editing a
shared file — a dirty working tree is a peer's live work, off-limits.** `[ev: retro dashboardpan-2d-to-3d-port · METHODOLOGY.md §Multi-session coordination]`

The concurrent-lane roles here (coordinator / researcher / QA) are the build-n4-module half of a shared
three-session template. The research-sdd half lives in `angeles725/sdd-investigacion` issue #867
(*METHODOLOGY: three-session coordinator/researcher/QA template for kit changes*); the two halves are
cross-referenced — the §7 pipeline below is this side's concrete instance.

**Price an estimate from the likely native plan, not the worst case, and state its assumption.** Before quoting
time to the user, run `gentle-ai review assess` on the planned diff shape (or on a comparable past commit) and quote
the tier budget (`BUILD-LOOP.md` §0.c) plus "N lenses expected" — a medium candidate commonly runs one lens, not four
plus a correction. `[ev: retro change-tier-time-budgets Δ6]`

## 5. Adopt-list

1. **Research block → spec requirement.** A `[CERT]` block's **Kit implication** names the target kit file/§ and
   an `[ev: corpus B<n>]` token; that becomes a gentle-SDD spec requirement (with a scenario). `[ev: corpus B801/B806/B815 Kit-implication sections]`
3. **RED → apply (GREEN).** `sdd-apply` (sonnet) implements to green; the automated half of the gate is
   `verify-module.sh` — a jar that has not passed it does not go to a station. `[ev: BUILD-LOOP.md §5]`
4. **Apply → retro.** The run writes its retro via `new-retro.sh` (the per-run precondition, below), capturing the
   proposed kit delta as `propose-never-apply`. `[ev: retro research-sdd-retro-automation] [ev: retro campaign8-retro-loop]`

## 6. Keep-from-kit

2. **Spec → RED.** QA authors a test that FAILS on the current tree and BITES only on the real defect
   (mutation-proven), pinned to a named branch, not a stale hash. `[ev: METHODOLOGY.md K2, K13]`
5. **Retro → fold.** A PROMOTION PR folds the proposed delta into the kit core under the §7 close gate exit (c),
   and the folded doc line carries `[ev: retro <slug>]` — which `toolbelt/sweep-fold-audit.sh` harvests to justify flipping
   the retro's INDEX row to `folded`. `[ev: BUILD-LOOP.md §7; toolbelt/sweep-fold-audit.sh]`

## 7. Pipeline

The campaign-8 pipeline, each arrow a real artifact:

```
research-sdd [CERT] block  →  gentle-SDD spec requirement  →  QA RED test  →  sdd-apply (→GREEN)  →  retro  →  fold
     (Kit implication)          (proposal/spec/design)         (a biting,        (implementation)     (new-retro   (promotion into
                                                                mutation-proven                        .sh stub)    the kit core)
                                                                failing test)
```

**Shape the review candidate to the tier; never override native risk.** Native review owns lens selection — the kit
cannot and must not pick lenses. What the kit controls is the candidate: keep a P1 change one small additive commit
(no refactor or unrelated churn in it), so native assessment can rate it on its real risk, and batch a display-only
dashboard commit with its rt commit into ONE reviewed slice instead of two review cycles. `[ev: retro change-tier-time-budgets Δ4]`

## 8. Per-run retro/ticket loop

Every run ENDS by writing its retro; the retro is a precondition for "done", not an at-STOP afterthought.

- `toolbelt/new-retro.sh <module|kit> <slug>` emits the retro stub (What happened / Evidence / Proposed kit deltas
  table / Lessons) and appends its `retros/INDEX.md` row (`pending`). `[ev: retro campaign8-retro-loop]`
- A defect in a KIT CHECK or DOCTRINE (a lint that misses/over-fires, a stale rule) additionally opens
  `toolbelt/kit-ticket.sh "<one line>"` (labels `kit`/`from-run`/`campaign-9`). `[ev: retro campaign8-retro-loop]`
- `toolbelt/sweep-build-state.sh --age` at orient (BUILD-LOOP §0.a) surfaces the accrued retro DEBT so it cannot
  be skipped across a continuous chain. `[ev: retro campaign8-retro-loop]`
- **The writer hand-back reports elapsed time per phase** — discovery/mapping, RED, GREEN, build, docs — so the tier
  budgets (`BUILD-LOOP.md` §0.c) are tuned from measured phases, not from a harness total. `[ev: retro comppan-auto-lock-indicator Δ6]`

WHY this is a hard loop and not a manual habit: §-close retros fire only at STOP / focus-close, but a continuous
lead-delegated chain (one unit → next task → next unit) NEVER reaches a STOP, so the trigger never arms — observed
live at ~8:1 (units landed : retros written) until the operator asked why. The debt counter makes the retro
un-skippable, same shape as the verify gate. `[ev: retro research-sdd-retro-automation §A]`

## 9. Incident journal

A live fault that outlives one session (a recurring console error, a field device misbehaving, an outage) gets ONE
journal file per incident in the client repository's journal folder — not in the kit, and not scattered across
feature docs. Template, kept current as the incident evolves:

| Section | Content |
|---|---|
| Symptom | what was observed, where, by whom |
| Timeline | timestamped events (first seen, changes made, restarts) |
| Hypotheses | each candidate cause |
| Evidence | per hypothesis: what supports or refutes it, with log/console/oBIX references |
| Ruled out | hypotheses closed, and the evidence that closed them |
| Next step | the single next action |
| Owner | who acts next (own module / site / platform vendor) |

`[ev: retro site-fault-triage-and-incident-journal Δ3]`
