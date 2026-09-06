# Orchestration — research-sdd · gentle SDD · BUILD-LOOP, and how they hand off

**Contents**: §1 Roles · §2 Model table · §3 Delegation triggers · §4 Escalation gate · §5 Adopt-list · §6 Keep-from-kit · §7 Pipeline · §8 Per-run retro/ticket loop

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

## 4. Escalation gate

Boundaries between concurrent lanes are enforced by the multi-session rule: **check the tree before editing a
shared file — a dirty working tree is a peer's live work, off-limits.** `[ev: retro dashboardpan-2d-to-3d-port · METHODOLOGY.md §Multi-session coordination]`

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

## 8. Per-run retro/ticket loop

Every run ENDS by writing its retro; the retro is a precondition for "done", not an at-STOP afterthought.

- `toolbelt/new-retro.sh <module|kit> <slug>` emits the retro stub (What happened / Evidence / Proposed kit deltas
  table / Lessons) and appends its `retros/INDEX.md` row (`pending`). `[ev: retro campaign8-retro-loop]`
- A defect in a KIT CHECK or DOCTRINE (a lint that misses/over-fires, a stale rule) additionally opens
  `toolbelt/kit-ticket.sh "<one line>"` (labels `kit`/`from-run`/`campaign-9`). `[ev: retro campaign8-retro-loop]`
- `toolbelt/sweep-build-state.sh --age` at orient (BUILD-LOOP §0.a) surfaces the accrued retro DEBT so it cannot
  be skipped across a continuous chain. `[ev: retro campaign8-retro-loop]`

WHY this is a hard loop and not a manual habit: §-close retros fire only at STOP / focus-close, but a continuous
lead-delegated chain (one unit → next task → next unit) NEVER reaches a STOP, so the trigger never arms — observed
live at ~8:1 (units landed : retros written) until the operator asked why. The debt counter makes the retro
un-skippable, same shape as the verify gate. `[ev: retro research-sdd-retro-automation §A]`
