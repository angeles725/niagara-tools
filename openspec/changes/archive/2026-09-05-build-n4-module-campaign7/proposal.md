# Proposal: build-n4-module-campaign7

**Status**: proposal · **Phase**: propose (post-explore)
**Source**: niagara-tools `v0.17.0` (main `c136e3b`) · **Target**: `v0.18.0` (MINOR — new toolbelt scripts + flags, CONTRIBUTING §4-5)
**Inputs**: `openspec/changes/build-n4-module-campaign7/explore.md` (gate-passed) · research B794 / B795 / B796 (landed) · B797 / B798 (in progress, gate PR6 / PR8 only)
**Topic key**: `sdd/build-n4-module-campaign7/proposal`
**Delivery**: auto-chain, 8 chained PRs, review budget 800 lines/PR (CONTRIBUTING aims ~400 authored)

---

## 1. Intent

Campaign 6 doubled the toolbelt to 10 scripts, but the doctrine never learned they exist: `preflight.sh`, `slot-coverage.sh` and `lint-timers.sh` are cited **zero times** in `BUILD-LOOP.md` and the launcher `SKILL.md` (grep-verified). A builder following the loop literally runs the verify gate and never runs the three checks that catch what the gate cannot — the ColdRoomPan `BEvaporatorUnit` four-`Clock.Ticket` leak is invisible to the gate and caught only by `lint-timers.sh`. At the same time **9 campaign-6 retros sit `pending`** in `retros/INDEX.md`, all dated 2026-09-05, so `sweep-build-state.sh --age` escalates on **2026-10-05**.

Three research blocks now unblock work the kit has deferred for two campaigns: B794 proves a `scaffold-module.sh` round-trip (scaffold → build → gate ALL PASS → lint mutation RED → restore GREEN), B795 turns B754's survival matrix into an embeddable CSV classifier, and B796 supplies the `-ux` write-surface exemplar the kit has never had. This campaign closes the retro debt, wires the toolbelt into the doctrine, and lands the three tools the research made buildable.

---

## 2. Scope

### 2.1 In scope

| ID | Item | Issue | Why now |
|---|---|---|---|
| **A** | Fold the 9 campaign-6 retros: K11-K14 → METHODOLOGY §Kit-maintenance, BUILD-LOOP §7 envelope rule, CONTRIBUTING §SDD ledger, two-independent-reads rule, per-PR lessons (marker contract, CI pins, fold-audit citation, metric naming, jdk8) | #48 | Clears the debt before the `--age` deadline |
| **B** | Tool integration: BUILD-LOOP §0.b (preflight), §5 (lint-timers + slot-coverage pre-gate), §7 (`--age` + envelope pairing) + SKILL.md §References and step 5 | — | Removes 5 silent gaps between tools and doctrine |
| **D** | `types/dashboard.md` fold of B796 (DashboardPan-ux as the write-surface exemplar, 4/5 gates, gate 4 REQUIRED-but-absent) | — | The kit's thinnest surface gets a real citation |
| **C** | `toolbelt/scaffold-module.sh` from the B794 prototype + `build-n4-module-kit/fixtures/MinimalPan/` skeleton | #45 | Removes the 12-file boilerplate; round-trip already proven |
| **E** | `toolbelt/schema-risk.sh` — embeds B795 §795.4 CSV verbatim, two-snapshot slot diff, worst-cell verdict, unknown → OUTAGE | #46 | Pre-deploy guard for the ClassCastException boot-loop class |
| **G** | `verify-module.sh --plano` exact check | #47 | Gated on companero's B797 spec |
| **F** | `types/logic.md` split → control authoring (1-79) stays, framework-extension authoring (80-136) moves to `types/logic-authoring.md` | — | 136 lines = two documents for two audiences |
| **H** | `toolbelt/report-module.sh` punch-list mode (lint-timers + slot-coverage + `verify --src`, one aggregated report) | #49 | Stretch; gated on B798 baseline; first to cut |

### 2.2 Judged OUT (with reason)

| Item | Reason |
|---|---|
| **#50** station-required checks | No station in CI or WSL; would be a fake PASS |
| Dependency floor matrix beyond 4.14 | B784 residue; needs a built before/after pair, not doctrine |
| Fixes inside module repos (DashboardPan gate 4 / per-Ord lock) | Separate repos; #49 tracks it client-side, the kit only cites it |
| B795-G1 / B795-G2 / B793-G1 live-boot probes | Requires-execution; stays on the station backlog |

---

## 3. Capabilities

> `openspec/specs/` does not exist in this repo. Following the campaign-6 convention, `sdd-spec` writes one change-local `spec.md`.

### New Capabilities

- `kit-scaffold-generator`: `scaffold-module.sh` contract — CLI `<ModuleName> <out-dir> [--vendor|--target-version|--plugin-version]`, typed exits `0/2/3`, the six B793-corrected shapes, and the **home-independence rule** (skeleton resolved as `"${BASH_SOURCE[0]%/*}/../fixtures/MinimalPan"`, never `$HOME`).
- `module-schema-risk`: `schema-risk.sh` contract — two-snapshot slot diff over `module-include.xml` + slot declarations, table-driven classification from the embedded B795 CSV, `*_unknown` fallback when a subtype is unresolved, unrecognized kind → `UNKNOWN` → OUTAGE, module verdict = **worst cell** (OUTAGE > LOSSY > SAFE).
- `kit-module-report` *(stretch, H)*: `report-module.sh` aggregation contract — a FAIL from any member check must surface as a FAIL in the aggregate report.

### Modified Capabilities

- `build-n4-module-kit-doctrine`: every toolbelt script is cited at the loop step that runs it; `types/logic.md` splits into control vs framework-extension authoring with the SKILL.md decision table moving atomically; `types/dashboard.md` gains the B796 `-ux` exemplar; METHODOLOGY gains K11-K14.
- `module-verify-gate`: adds `--plano` (exact aspect-ratio agreement check) — **only if B797 lands in time**.

---

## 4. Approach — eight chained PRs, fixed merge order

Each PR is one destination-file group. PR1 targets `main`; each later PR branches only after its predecessor merges. Every kit-changing push pairs its retro + INDEX row + kit `BUILD-STATE.md` self-envelope **in the same push range** (campaign-6 close lesson 1).

| # | Branch / work unit | Files | Est. authored | Test / named mutation | QA RED |
|---|---|---|---|---|---|
| **PR1** | `feat/c7-fold-retros` (A) | `METHODOLOGY.md`, `BUILD-LOOP.md`, `CONTRIBUTING.md`, `retros/INDEX.md` + 9 markers | ~40 | None (doc-only). Guard: `sweep-build-state.sh` green, `sweep-fold-audit.sh --strict` green, every flipped row cites `[ev: retro <slug>]` | none — fidelity read |
| **PR2** | `feat/c7-tool-integration` (B) | `BUILD-LOOP.md`, `~/.claude/skills/build-n4-module/SKILL.md` | ~20 | None. Guard: grep asserts each of the 10 scripts is named at its step | none |
| **PR3** | `feat/c7-dashboard-fold` (D) | `types/dashboard.md` | ~25 | None. Guard: `kit-links.bats` green; fidelity read vs B796 §796.4 | none |
| **PR4** | `feat/c7-scaffold` (C) | `toolbelt/scaffold-module.sh`, `fixtures/MinimalPan/**`, `tests/scaffold-module.bats` | ~700-880 → **`size:exception`**, fixture as its own commit | TC1 missing arg → 2 · TC2 invalid name → 2 · TC3 emitted tree byte-equals fixture · TC-K8 identical under `HOME=/nonexistent` · TC4 round-trip build+gate (**SKIP in CI, run locally, never faked**). Mutation: drop the `stopped()`-cancel from the template → lint-timers FAIL | `qa/c7-scaffold` |
| **PR5** | `feat/c7-schema-risk` (E) | `toolbelt/schema-risk.sh`, `tests/schema-risk.bats`, fixtures | ~200 | 6 fixture classes: add → SAFE, remove → LOSSY, retype simple → OUTAGE, reorder → SAFE, rename → LOSSY, unknown kind → OUTAGE. Mutation: worst-cell → first-cell verdict flips the mixed fixture | `qa/c7-schema-risk` |
| **PR6** | `feat/c7-plano` (G) | `toolbelt/verify-module.sh`, `tests/verify-module.bats` | ~140 | PL1 two disagreeing `aspect-ratio` declarations → FAIL. Mutation: count-only check passes the disagreeing pair | `qa/c7-plano` |
| **PR7** | `feat/c7-logic-split` (F) | `types/logic.md`, `types/logic-authoring.md`, `SKILL.md`, `BUILD-LOOP.md` §2, `tests/kit-links.bats` | ~70 (moves) | None new; `kit-links.bats` extended to assert both targets resolve. Mutation: break one moved link → kit-links FAIL | none |
| **PR8** | `feat/c7-report` (H, stretch) | `toolbelt/report-module.sh`, `tests/report-module.bats` | ~50 | Report aggregates a FAIL from lint-timers. Mutation: drop aggregation → report reports PASS | `qa/c7-report` |

**Version**: `VERSION` + `CHANGELOG.md` bump to `0.18.0` in the last merged PR (MINOR — new scripts and flags per CONTRIBUTING §4-5).

**Gate exits**: PR1-PR7 are kit-changing pushes → close-gate exit **(a) NEW RETRO**. PR8 too if it lands. Every PR body carries `Closes #N`; conventional commits, English, and commit bodies are grepped for attribution trailers before publishing.

**Slicing note**: PR4 is the one honest `size:exception` — the fixture is a 12-file buildable module skeleton that cannot be split without shipping an unbuildable tree. Its fixture lands as its own commit so the reviewable script diff stays ~300 lines.

---

## 5. Affected Areas

| Area | Impact | Description |
|---|---|---|
| `build-n4-module-kit/toolbelt/{scaffold-module,schema-risk,report-module}.sh` | New | Generator, pre-deploy classifier, aggregate report |
| `build-n4-module-kit/toolbelt/verify-module.sh` | Modified | `--plano` exact check (B797-gated) |
| `build-n4-module-kit/fixtures/MinimalPan/**` | New | Bundled skeleton; removes the prototype's `$HOME` coupling |
| `build-n4-module-kit/{METHODOLOGY,BUILD-LOOP}.md`, `types/{logic,logic-authoring,dashboard}.md` | Modified/New | Fold + tool routing + split + B796 exemplar |
| `build-n4-module-kit/retros/INDEX.md` + 9 retro files | Modified | 9 pending → 0 |
| `build-n4-module-kit/BUILD-STATE.md` | Modified | Self-envelope paired with every kit-changing push |
| `tests/*.bats` | New/Modified | ~23 new cases, each with a named mutation |
| `CONTRIBUTING.md`, `VERSION`, `CHANGELOG.md` | Modified | SDD ledger rule; `0.17.0 → 0.18.0` |
| `~/.claude/skills/build-n4-module/SKILL.md` | Modified | **Outside git** — not revertible with `git revert` |

---

## 6. Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| R1 | **K8 violation carried over**: the prototype hardcodes `MINIMOD_ROOT` under the operator's home (`scaffold-module.sh:22`) | **High** | Bundle the skeleton under `fixtures/`, resolve relative to `BASH_SOURCE`, and prove it with TC-K8 under `HOME=/nonexistent`. This is a blocking acceptance test, not a review note |
| R2 | `--age` escalation on **2026-10-05** if the fold slips | Med | PR1 merges first and is doc-only; no PR4-scale dependency sits in front of it |
| R3 | `logic.md` split breaks references silently | Med | Atomic PR7: SKILL.md decision table + BUILD-LOOP §2 move in the same commit range; `kit-links.bats` extended before the move |
| R4 | The two-snapshot slot diff is **novel code** with no prior art in the toolbelt | Med | Classification is table-driven from the B795 CSV, never hand-coded; fixtures cover all six classes; unresolved subtype falls back to the `*_unknown` row, unknown kind → OUTAGE |
| R5 | PR4 exceeds the review budget (~700-880) | **High** (accepted) | Declared `size:exception` up front; fixture as its own commit; reviewer reviews the script diff, not the skeleton |
| R6 | CI cannot build a Niagara module (no Java 8 + `niagara_home`) | High | TC4 **SKIPs** in CI and is run locally by QA before bless, with the exact command output recorded in the PR body. A SKIP is never reported as a PASS |
| R7 | B797 / B798 do not land in time | Med | They gate **only** PR6 and PR8. If either stalls, that PR is deferred to campaign 8 and the chain closes at PR7 without blocking the version bump |
| R8 | Chain drift — a child PR diff shows its parent's commits | Med | Branch only after the parent merges; rebase until the child diff is clean |
| R9 | Double-fold: a retro delta already lives in a core kit file | Med | Grep-before-fold (A8/K-rule) on every row; if found, flip the INDEX row and marker only |

---

## 7. Rollback Plan

| Slice | Rollback |
|---|---|
| PR8 / PR6 / PR5 | `git revert` — additive scripts and tests; `verify-module.sh` loses `--plano` only |
| PR4 | `git revert` — `toolbelt/scaffold-module.sh` and `fixtures/MinimalPan/` are new paths; nothing else references them |
| PR7 / PR3 / PR2 / PR1 | `git revert` — doc-only; retro files are never deleted (propose-never-apply), markers and INDEX rows revert to `pending`. Launcher `SKILL.md` (PR2, PR7) restored **manually** from the PR description |

No station, no deployed jar, and no operator data is touched by any slice. Rollback risk is limited to the developer workstation and CI.

---

## 8. Dependencies

- `bats-core`, `shellcheck 0.10.0` (pinned in `ci.yml`), live pre-push hook — already installed.
- Research: B794, B795, B796 landed in `niagara-research`. **B797** gates PR6; **B798** gates PR8.
- QA RED branches merged into their PR branch before it goes green: `qa/c7-scaffold`, `qa/c7-schema-risk`, `qa/c7-plano`, `qa/c7-report`.
- Fixed merge order PR1 → PR2 → … → PR8. Every PR body carries `Closes #N`.

---

## 9. Ownership

| Lane | Owns |
|---|---|
| investigador (orchestrator) | SDD chain, merges, live-doc updates |
| `sdd-apply` workers | One writer per PR, disjoint file sets, worktree-only writes |
| QA | RED branches + verification; local TC4 round-trip run before bless |
| investigador1 | Fidelity reads of PR1 / PR3 / PR7; any research residue |
| companero | B797 (`--plano` spec) + B798 (conformance baseline); second read of the scaffold and plano fixtures |

Every research fold gets **two independent reads**; QA REDs are cited by branch, and tips are re-read at apply time.

---

## 10. Success Criteria

- [ ] `retros/INDEX.md` pending **9 → 0**; only campaign-7 retros are pending at close; every flipped row cites `[ev: retro <slug>]` in a core kit file.
- [ ] `BUILD-LOOP.md` + launcher `SKILL.md` cite **all 10** toolbelt scripts at the step that runs them, asserted by a grep/`kit-links` check, not by review opinion.
- [ ] `scaffold-module.sh`: TC3 emitted tree **byte-equals** `fixtures/MinimalPan/` in CI; TC-K8 identical under `HOME=/nonexistent`; TC4 round-trip build + gate ALL PASS **locally**, SKIPped in CI with the local output pasted in the PR body.
- [ ] `schema-risk.sh` classifies all **6 fixture classes** per the B795 CSV and returns the worst-cell verdict; the worst-cell → first-cell mutation flips the mixed fixture.
- [ ] `kit-links.bats` green after the `logic.md` split; both `types/logic.md` and `types/logic-authoring.md` resolve from SKILL.md and BUILD-LOOP §2.
- [ ] bats suite **152 → ~175** passing, each new case with a named mutation recorded; `shellcheck` exit 0; `sweep-fold-audit.sh --strict` green.
- [ ] Every kit-changing push range contains its retro + INDEX row + `BUILD-STATE.md` self-envelope.
- [ ] `VERSION` = `0.18.0` with a `CHANGELOG.md` entry per CONTRIBUTING §4-5.
- [ ] Each PR merges in order with a clean child diff; no commit body carries an attribution trailer.

---

## 11. Review Workload Note

Session `review_budget_lines` = **800**; `delivery_strategy` = **auto-chain**.

`Decision needed before apply: No` (auto-chain resolves the slicing; PR4's `size:exception` is declared here)
`Chained PRs recommended: Yes` — 8 slices, fixed merge order
`400-line budget risk: Low` for PR1, PR2, PR3, PR6, PR7, PR8; **Medium** for PR5 (~200); **High and accepted** for PR4 (~700-880, `size:exception`)

---

## 12. Next Phases

- `sdd-spec` — formalize the scaffold emission contract and typed exits, the slot-diff `change_kind` domain and worst-cell predicate, the `--plano` exact-agreement rule, and the report aggregation contract. Can run in parallel with `sdd-design`.
- `sdd-design` — fixture layout for `MinimalPan`, the CI SKIP mechanics for TC4, per-file fold placement with `[ev:]` citations, the `logic.md` split boundary, and the eight-branch chain mechanics.
- Then `sdd-tasks` → `sdd-apply` (TDD strict for PR4-PR6, PR8) → `sdd-verify` → `sdd-archive`.
