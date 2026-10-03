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
- **Count what ran:** the writer runs `bats tests/*.bats` serially unless GNU `parallel` is installed. Without it,
  `bats -j` runs 0 tests and prints no `not ok` line. A parallel run reports its `ok` count next to `bats --count`,
  and the two must be equal. `[ev: retro retro-fold-campaign-close Δ2]`
- **Two shell traps that do nothing silently:** quote every pattern-substitution replacement,
  `${v//pat/"$rep"}` — under bash 5.2 `patsub_replacement` an unquoted `&` in the value expands to the matched text —
  and pin a value that contains `&` (`tests/shell-hygiene.bats` SH1 scans the kit scripts for an unquoted `$` or `&`
  replacement). In bats, never write `! cmd` as an assertion line: it cannot fail the test (shellcheck SC2314, which
  CI runs at every severity, flags it); use `run …; [ "$status" -ne 0 ]` or `if cmd; then return 1; fi`, and prove
  the negative branch with a mutation. Details: `CONTRIBUTING.md` §2. `[ev: retro polish-2026-10-02-close Δ3]`
  `[ev: retro polish-2026-10-02-close Δ4]`

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

**A START with an unknown outcome is recovered through the lineage STATUS, never by a new START.** When the granted
START returns `candidate_context_unavailable`, the mutation outcome is unknown: the transaction may already be frozen.
Do not re-run START and do not report an approval. Run the lineage-bound STATUS for the same lineage and target and
continue only from the `next_transition` it returns. It may show the transaction reviewing and reoffer the lens
captures, which then complete and are acknowledged as usual. `[ev: retro polish-2026-10-02-close Δ5]`

**Start every review from the canonical preflight, never from the command `review assess` returns.** Run
`gentle-ai review status --cwd <repo> --contract gentle-ai.review-integration/v2 --agent <runtime> --next-transition`
(add `--base-ref <boundary> --committed-only` when the candidate is a commit range). The `next_transition.command`
that `gentle-ai review assess` prints carries no `--agent`, so a lineage started from it hands back capture tokens
without one and every capture fails `invalid_request`. If that already happened, re-run the canonical preflight: it
returns the same lineage's START, and its exact replay (`replayed`) binds the agent so the slots are reoffered with
`--agent`. `[ev: retro deferred-lints-2026-10-03-close Δ1]`

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

**Route advisories away from an approved candidate.** The informational findings of an APPROVED review never go
into that candidate: an edit after approval invalidates its receipt and reopens the review. Record them in the
feature doc and on a follow-up issue. Only a finding that shows a fail-open (a check that can report PASS on a
real defect) or a regression introduced by the current feature becomes the next work unit. Every other finding
is parked: list it under the feature doc's `## Parked advisories` with its reason. Without that limit, each fix
draws new advisories and the feature never closes. `[ev: retro retro-fold-campaign-close Δ4]`
**Write the anti-cascade policy into the feature doc before the first advisory arrives.** An advisory-driven
feature states in its `## Authorization & delivery` section, up front, that only a fail-open or a regression
introduced by the feature becomes a sub-task. Every other advisory goes to `## Parked advisories` and, at close, to
ONE consolidated follow-up issue that keeps each finding's text next to its location (a lens/severity/location list
costs a full re-read of the reviewed head to reconstruct). Set after the cascade has started, the limit arrives
too late: the polish feature had already opened six sub-tasks. `[ev: retro polish-2026-10-02-close Δ1]`
**A review's fail-open sub-task rides in the next task's PR when it touches the same file or helper.** One review
per PR: fold the fix-forward (with its RED and observed mutation) into the next work unit's candidate when that unit
edits the same script or shared helper, and give it its own PR only at the end of the chain, where no next unit
exists. Record in the feature doc which one it was ("folded into the D3 PR" / "own PR"). `[ev: retro deferred-lints-2026-10-03-close Δ4]`

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
