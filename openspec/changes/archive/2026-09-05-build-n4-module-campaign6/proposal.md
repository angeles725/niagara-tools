# Proposal: build-n4-module-campaign6

**Status**: proposal · **Phase**: propose (post-explore)
**Source**: niagara-tools `v0.15.1` (main `48f3736`) · **Target**: `v0.16.0` (MINOR — kit doctrine growth + new toolbelt scripts)
**Inputs**: `openspec/changes/build-n4-module-campaign6/explore.md` · `/home/cristian/modulos_niagara_n4/niagara-tools-worktrees/campaign6-harvest.md` (uncommitted working doc) · QA RED `qa/c6-marker-index-drift` `cb0dd7d`
**Topic key**: `sdd/build-n4-module-campaign6/proposal`
**Delivery**: auto-chain, 6 chained PRs, review budget 800 lines/PR (CONTRIBUTING aims ~400 authored)

---

## 1. Intent

Five campaigns folded a 42-lesson corpus into the kit, but the closing mechanics were left half-done: **8 retros sit `pending` in `retros/INDEX.md`**, the sweep that guards retro continuity is **marker-blind** (it parses the INDEX row and never reads the in-file `<!-- review-status: -->`), and `run-pure-test.bats` **skips every case in CI**, so green CI does not mean `run-pure-test.sh` was tested. Meanwhile `build-verify.md` asserts a slotomatic gap the code closed for modes A/C.

Left alone, the kit ships stale guidance while its own promotion discipline is unenforced — the exact failure mode the discipline exists to prevent. This campaign closes the enforcement gaps mechanically, folds the remaining high-signal deltas, and leaves an explicit intake path for the research lane.

---

## 2. Scope

### 2.1 In scope

| Group | Items | Why |
|---|---|---|
| Enforcement | **B1** marker↔INDEX contract (option 2-lite), **B3** CI runs P1-P6, **B2** fold-audit | The three checks that make the sweep tell the truth |
| Doctrine | **A3-A12** meta cluster + **K9** + M1/M2 (METHODOLOGY), **A10** (CONTRIBUTING) | Un-folds 7 pending retros with `[ev:]` citations |
| Types | **A1**, **A13**, **A14/LC1-LC5** (logic), **A15** (dashboard), **A16**+**A17** (build-verify), **A18** (SOURCES) | Turns `types/logic.md` from "growing" to exemplar-backed; fixes the doc-vs-code drift |
| Tools | **B5** preflight, **D2** slot-coverage, **MM1** gate-coverage function, **B4** `--plano` (stretch) | Each automates a prose-only rule that was a real live failure |
| Close-out | **C1-C3** SKILL.md, **D4** archive slotomatic change + commit `openspec/` | Removes a superseded 40-task change and tracks the SDD store |

### 2.2 Judged OUT (with reason)

| Item | Verdict | Reason |
|---|---|---|
| **A2** (freeze-stat Δ3 — `BDouble.make` + import) | **OUT — already folded** | Grep proves it: `METHODOLOGY.md:9` carries the `BDouble.make` rule, `:11` carries `import javax.baja.sys.BDouble`. explore.md §2 #1 is wrong here; the harvest dedupe appendix is right. Flip the INDEX/marker only, fold nothing. |
| **D1** `scaffold-module.sh` | OUT | ~80L of new surface whose biting test needs a real gradle/Niagara layout; no pending retro demands it. Campaign 7. |
| **D3** `dashboard-screenshot.sh` | OUT | External Playwright dependency, no biting test possible in CI (explore §3d: "No"). |
| **E2/MM2** as a standalone script | OUT as a script, **IN as D2** | `set_coverage(declared, required)` is exactly what `slot-coverage.sh` computes; a second copy would be duplication, not a model. |
| **MM3-MM7** | OUT | MM3 needs two `module.xml` snapshots and a slot-diff parser (>60L, fuzzier); MM4-MM7 are metrics with no consumer that changes a decision today. |
| Enriching `types/wb-widgets.md` past SEED | OUT | No wb build happened; **C2** ships the SEED warning instead. |
| Any change to a module repo (`DashboardPan.verify_gate = unknown`) | OUT | Separate repos; no mechanical fix is possible from this kit. |

### 2.3 Research-lane intake (D5) — explicit, non-blocking

investigador1's `/research-sdd` lane on `niagara-research` (focuses `module-testing`, `module-servlet-security`) returns deltas **as new pending retros in `retros/`**. This campaign does **not** wait on them. Intake path: a **follow-up fold PR** after PR6, opened against `main`, that folds those retros and flips their INDEX rows. If the lane lands nothing before PR6 merges, campaign 6 still closes at 0 pending.

---

## 3. Capabilities

> `openspec/specs/` does not exist in this repo. Following the `build-n4-module-kit-v0.2` convention, `sdd-spec` writes one change-local `spec.md`.

### New Capabilities

- `kit-retro-sync-contract`: the marker↔INDEX agreement rule (option 2-lite: a present marker MUST equal its INDEX row; an absent marker is tolerated; an out-of-domain word FAILS) plus the fold-audit citation contract (`[ev: retro <slug>` must appear in a core kit file for every `folded` slug, else WARN).
- `kit-preflight-and-coverage`: contracts for `toolbelt/preflight.sh` (BUILD-LOOP §0.b automation, typed exits), `toolbelt/slot-coverage.sh` (lexicon ∩ declared types, WARN on empty lexicon), and the pure `coverage(covered, total)` function — `total == 0` returns an **N/A sentinel, never 100**.

### Modified Capabilities

- `build-n4-module-kit-doctrine`: retro `review-status` marker convention becomes machine-enforced; new kit-maintenance rules (gate-exit taxonomy, grep-before-fold, worktree-off-origin, env-coupling, pinned linters, WARN over silent removal, marker-promotes-not-prose, multi-session git-status guard, live-verify write/secret safety).
- `module-verify-gate`: **only if B4 lands** — adds `--plano <index.html>` (exactly one frame `aspect-ratio` declaration, four plano values agree).

---

## 4. Approach — six chained PRs, fixed merge order

Each PR is one destination-file group (explore §4, Option C lanes executed as Option B slices). PR1 targets `main`; each later PR branches after its predecessor merges. **`METHODOLOGY.md` is touched by exactly one PR (PR3).**

| # | Branch / work unit | Files | Est. authored | Test / mutation expectation | Kit retro? | VERSION |
|---|---|---|---|---|---|---|
| **PR1** | `feat/c6-marker-index` (B1) | `toolbelt/sweep-build-state.sh`, `tests/build-retro-sync.bats` (QA RED `cb0dd7d`), 2 retro markers re-stamped `fresh`→`folded` | ~60 | RED already staged: **M1** marker≠INDEX → FAIL, **M4** `fresh` shape, **M5** real tree green. Mutation: delete the marker read → M1 goes green ⇒ test bites | Yes — exit (a) | — |
| **PR2** | `feat/c6-ci-pure-test` (B3) | `.github/workflows/ci.yml`, `tests/run-pure-test.bats` | ~15 | CI pre-fetches junit so P1-P6 execute. Mutation: **P2** (failing pure test must exit non-zero) — break the exit code, CI must go red | No (no kit file touched) | — |
| **PR3** | `feat/c6-doctrine` (A3-A12, K9, M1, M2, A10) | `METHODOLOGY.md`, `CONTRIBUTING.md`, `retros/INDEX.md` + 7 markers | ~75 | No new test (doctrine). Guard: `sweep-build-state.sh` green with 7 rows flipped to `folded`, each carrying `[ev: retro <slug>]` | Yes — exit (a) | — |
| **PR4** | `feat/c6-types` (A1, A13, A14/LC1-5, A15, A16, A17, A18) | `types/logic.md`, `types/dashboard.md`, `build-verify.md`, `SOURCES.md`, `corpus-index.md`, freeze-stat retro + INDEX row | ~75 | No new test. Guard: `kit-links.bats` green; grep proves no superseded "-rt only" string survives | Yes — exit (a) | — |
| **PR5** | `feat/c6-tools` (B2, B5, D2, MM1, B4 stretch) | `toolbelt/sweep-fold-audit.sh`, `toolbelt/preflight.sh`, `toolbelt/slot-coverage.sh`, MM1 function, `tests/*.bats` ×4 | ~290 (~250 without B4) | Every script ships one biting case: fold-audit → `folded` without `[ev:]` = WARN; preflight → missing JDK 8 / locked jar = typed exit; slot-coverage → empty lexicon = WARN (CompPan T8, a real defect); MM1 → `coverage(0,0)` returning 100 flips the assertion; B4 → two `aspect-ratio` declarations = FAIL | Yes — exit (a) | — |
| **PR6** | `feat/c6-close` (C1-C3, D4) | launcher `SKILL.md`, `openspec/changes/niagara-tools-slotomatic-integration/` → `archive/` + supersession note, `openspec/` tracked, `VERSION`, `CHANGELOG.md` | ~60 authored (+ imported openspec files) | No new test. Guard: full suite green, sweep green, 0 pending | Yes — exit (a) | **0.15.1 → 0.16.0** |

**Gate-exit taxonomy**: PR1, PR3, PR4, PR5, PR6 are kit-changing pushes and use close-gate exit **(a) NEW RETRO** (C5 rule). PR2 touches only `ci.yml` and `tests/`, so the kit pre-push hook does not apply.

**Slicing note**: if PR5 exceeds 400 authored lines after one honest pass, split into **PR5a** (fold-audit + MM1) and **PR5b** (preflight + slot-coverage + B4). Do not shrink code to fit.

---

## 5. Affected Areas

| Area | Impact | Description |
|---|---|---|
| `build-n4-module-kit/toolbelt/sweep-build-state.sh` | Modified | Reads and validates each retro's in-file marker |
| `build-n4-module-kit/toolbelt/{sweep-fold-audit,preflight,slot-coverage}.sh` | New | Fold-citation audit, §0.b preflight, lexicon coverage |
| `build-n4-module-kit/toolbelt/verify-module.sh` | Modified (stretch) | `--plano` aspect-ratio check |
| `build-n4-module-kit/{METHODOLOGY,build-verify,SOURCES,corpus-index}.md`, `types/*.md` | Modified | Doctrine + type deltas with `[ev:]` citations |
| `build-n4-module-kit/retros/INDEX.md` + 8 retro files | Modified | 8 pending → folded, markers stamped |
| `tests/*.bats`, `.github/workflows/ci.yml` | Modified/New | Biting cases; P1-P6 actually execute |
| `openspec/` | New (tracked) | Store committed; slotomatic change archived |
| `~/.claude/skills/build-n4-module/SKILL.md` | Modified | **Outside git** — not revertible with `git revert` |

---

## 6. Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| R1 | **Double-fold**: a delta already lives in a core file (A2 was exactly this) | **High** | Apply A8 to ourselves: `rg` every kit file for the rule BEFORE writing. If found, flip the INDEX/marker only. `sdd-tasks` must carry a grep-before-fold column per row |
| R2 | Two writers touch `METHODOLOGY.md` | Med | PR3 owns it exclusively; other lanes submit deltas as retro proposals |
| R3 | Fold-audit false positives on slugs appearing in cross-links or INDEX rows | Med | Scope the grep to the literal `[ev: <slug>` pattern; test against one known-folded and one known-not-folded retro |
| R4 | Test padding inflates the suite and slows it | Med | Only B1-B5, D2 and MM1 get tests, each with a named mutation stated in §4. Doctrine deltas get zero tests. Suite must stay under ~15 s wall (baseline 8.6 s) |
| R5 | `preflight.sh` (~60L, 4 environment probes) is the largest new surface and is env-coupled | Med | Prove it bites under `HOME=/nonexistent` (A11/K8); fakebin probes, no network |
| R6 | Launcher `SKILL.md` is outside git | High | PR6 description carries the full before/after; engram records it; rollback is a manual re-edit |
| R7 | The research lane lands retros mid-campaign and re-opens the 0-pending target | Low | §2.3 — intake is a follow-up PR after PR6, never a blocker |
| R8 | Chain drift: a child PR diff shows its parent's commits | Med | Branch only after the parent merges; rebase until the child diff is clean |

---

## 7. Rollback Plan

| Slice | Rollback |
|---|---|
| PR6 | `git revert` the release commit (VERSION/CHANGELOG/archive move). Launcher `SKILL.md` restored **manually** from the PR description |
| PR5 | `git revert` — all new scripts and tests are additive; `verify-module.sh` loses `--plano` only |
| PR4 / PR3 | `git revert` — doc-only; retro files are never deleted (propose-never-apply), only their markers/INDEX rows revert to `pending` |
| PR2 | `git revert` — CI returns to skipping P1-P6 (the prior state) |
| PR1 | `git revert` — sweep returns to marker-blind; the 2 re-stamped markers revert to `fresh` |

No station, no deployed jar, and no operator data is touched by any slice. Rollback risk is limited to the developer workstation and CI.

---

## 8. Dependencies

- `bats-core` and `shellcheck` — already installed (CI `ci.yml` runs both plus `sweep-build-state`).
- QA RED branch `qa/c6-marker-index-drift` `cb0dd7d` must be merged into PR1's branch before PR1 goes green.
- Fixed merge order PR1 → PR2 → PR3 → PR4 → PR5 → PR6.
- Read-only: `niagara-research` corpus blocks B536-B557 (cited by A14/LC1-LC5), the uncommitted harvest table.
- Issue-first: every PR body carries `Closes #N`. Conventional commits, English, no AI trailers.

---

## 9. Ownership (explore §4 recommendation)

| Lane | Owns |
|---|---|
| investigador (orchestrator) | SDD chain; launches one `sdd-apply` writer per PR with disjoint file sets |
| QA | `tests/*.bats` RED commits and `.github/workflows/ci.yml` (PR1 RED, PR2) |
| investigador1 | `niagara-research` commits (research lane D5), the harvest and model-spec working documents |

---

## 10. Success Criteria

- [ ] `retros/INDEX.md` pending count **8 → 0**; every flipped row cites `[ev: retro <slug>]` in a core kit file.
- [ ] `sweep-build-state.sh` reads each in-file marker and FAILS when a present marker disagrees with its INDEX row (mutation-proven by M1).
- [ ] CI **executes** `run-pure-test.bats` P1-P6 — zero SKIPs in the CI log; P2 fails when the exit code is mutated.
- [ ] Every new script (`sweep-fold-audit.sh`, `preflight.sh`, `slot-coverage.sh`, MM1, `--plano` if landed) ships with at least one test whose named mutation flips it.
- [ ] Every doc delta was grep-audited against the whole kit before folding; A2 is recorded as already-folded and **not** re-folded.
- [ ] `build-verify.md` no longer asserts the blanket "-rt only" slotomatic gap; the mode-B wording is in place.
- [ ] Full bats suite green (≥104 pass, no regressions), `shellcheck` exit 0, suite wall time ≤ ~15 s.
- [ ] `openspec/` tracked in git; `niagara-tools-slotomatic-integration` under `openspec/changes/archive/` with a supersession note; zero tasks from it applied.
- [ ] `VERSION` = `0.16.0` with a `CHANGELOG.md` entry per CONTRIBUTING §4-5.
- [ ] Each PR merges in order with a clean child diff and a `Closes #N` reference.

---

## 11. Review Workload Note

Session `review_budget_lines` = **800**; `delivery_strategy` = **auto-chain**.

`Decision needed before apply: No` (auto-chain resolves the slicing)
`Chained PRs recommended: Yes` — 6 slices, fixed merge order
`400-line budget risk: Low` for PR1-PR4 and PR6; **Medium for PR5** (~290, split into PR5a/PR5b if it crosses 400)

---

## 12. Next Phases

- `sdd-spec` — formalize the marker↔INDEX predicate and its failure domain, the fold-audit grep contract, `preflight.sh`/`slot-coverage.sh` exits, and the `coverage(covered, total)` N/A sentinel. Can run in parallel with `sdd-design`.
- `sdd-design` — per-file fold placement and `[ev:]` citation format, the grep-before-fold audit table, bats fixture helpers, and the six-branch chain mechanics.
- Then `sdd-tasks` → `sdd-apply` (TDD strict for scripts) → `sdd-verify` → `sdd-archive`.
