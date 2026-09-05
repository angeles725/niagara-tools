# Exploration — build-n4-module-campaign7

**Date**: 2026-09-05 | **Phase**: explore | **Status**: complete
**Engram**: `sdd/build-n4-module-campaign7/explore`
**Orchestrator gate notes** (investigador): integration-gap claims re-verified (`grep -c preflight|lint-timers|slot-coverage` = 0 in BUILD-LOOP.md and skill/SKILL.md); types/logic.md = 136 lines; prototype `scaffold-module.sh:22` hardcodes `MINIMOD_ROOT` under the operator's home (K8 violation confirmed); INDEX.md pending rows = 9 (all dated 2026-09-05 → `--age` escalation 2026-10-05).

---

## 1. Current-State Map — Post-Campaign 6

### Toolbelt inventory (10 scripts; 4 new since campaign 5)

| Script | Enforces | Bite confirmed | BUILD-LOOP citation |
|---|---|---|---|
| `build.sh` | Java 8 clean + slotomatic + verify gate | B1-B7 | §4 |
| `verify-module.sh` (+ `coverage`) | bytecode 52, signed, types resolve, typecount, facets, palette; coverage % | V1-V17 + MM1-MM8 | §5 |
| `run-pure-test.sh` | JUnit isolation for pure logic | P1-P6 (CI-executed since PR2) | §2/§5 |
| `sweep-build-state.sh` (+ `--age`) | envelope + INDEX ↔ retro markers (2-lite); retro-debt aging | M1-M6 + D1-D6 | §7 (no `--age` mention) |
| `sweep-fold-audit.sh` | fold-citation audit, `--strict` in CI | F1-F6 | not cited |
| `preflight.sh` | JDK 8 (WSL fallback) / win-path / plugin pin / jar lock | PF1-PF5 | **not cited (gap)** |
| `slot-coverage.sh` | type-set lexicon coverage %, dup bare keys | SC1-SC6 + parse + dup-keys | **not cited (gap)** |
| `lint-timers.sh` | `Clock.Ticket` owner without cancelling `stopped()`; discarded ticket | TL1-TL4 | **not cited (gap)** |
| `mirror-niagara-home.sh` | running-station build isolation | M1-M5 | §0.b |
| `stored-repack.sh` | STORED repack, never overwrites | S1-S3 | §5 |

### Integration gaps — highest-value doctrine items

1. **BUILD-LOOP §0.b** lists JDK 8 / niagara_home / station as manual checks but never names `toolbelt/preflight.sh`, whose own header declares "BUILD-LOOP §0.b environment preflight".
2. **BUILD-LOOP §5** cites only `verify-module.sh`; `lint-timers.sh` and `slot-coverage.sh` are missing pre-gate steps. Real consequence: ColdRoomPan `BEvaporatorUnit` leaks four `Clock.Ticket`s — caught only by lint-timers, invisible to the verify gate. [ev: retro campaign6-conformance-lints]
3. **BUILD-LOOP §7** does not mention `sweep-build-state.sh --age` at orient/close and does not state that every exit (a/b/c) pairs its anchor with the kit `BUILD-STATE.md` self-envelope in the same push range. [ev: retro campaign6-close lesson 1]
4. **skill/SKILL.md §References** lists build.sh, verify-module.sh, mirror-niagara-home.sh, stored-repack.sh; omits preflight.sh, slot-coverage.sh, lint-timers.sh, sweep-fold-audit.sh, run-pure-test.sh.
5. **skill/SKILL.md §Execution step 5** says "run the verify gate" with no routing to the pre-gate checks.

### Cognitive load assessment

| File | Lines | Assessment | Action |
|---|---|---|---|
| METHODOLOGY.md | 83 | 16-section checklist; manageable | add K11-K14 only |
| BUILD-LOOP.md | 65 | sequential; good length | minor additions for gaps 1-3 |
| types/logic.md | 136 | **two documents in one**: lines 1-79 control authoring proven from ColdRoomPan/CompPan builds (timers, interlocks, HOA, fail modes); lines 80-136 framework-extension authoring from PR7 (author-side SPIs, point extensions, containers, queries, templates, jobs, action protection, minimal module) — a different audience | split: `types/logic-authoring.md` for the framework-extension half, atomic with SKILL.md/BUILD-LOOP references |
| types/dashboard.md | ~70 | coherent; B796 now supplies the missing -ux write-surface exemplar (4/5 gates) | fold B796 in one doc PR |
| skill/SKILL.md | 58 | stale references; no pre-gate routing | update references + step 5 |

---

## 2. Backlog

| ID | Item | Issue | Value | Size | Biting test? | Blocked? |
|---|---|---|---|---|---|---|
| A | Fold the 9 campaign-6 retros: K11-K14 → METHODOLOGY §Kit-maintenance; BUILD-LOOP §7 envelope rule; CONTRIBUTING §SDD ledger; two-independent-reads rule; per-PR retro lessons (marker contract, CI pins, fold-audit citation rules, metric naming, jdk8) | #48 | HIGH — clears the debt before `--age` escalates on 2026-10-05 | ~30 net lines + 9 INDEX flips + markers | none (doc-only; grep-before-fold per rule) | no |
| B | Tool integration: BUILD-LOOP §0.b/§5/§7 + SKILL.md references + step 5 | — | HIGH — removes 5 silent gaps | ~15 bullets, 2 files | none | no |
| C | `scaffold-module.sh` (#45): port the B794 prototype into `toolbelt/`, bundle the MinimalPan skeleton as `build-n4-module-kit/fixtures/MinimalPan/` (fixes the K8 violation), bats | #45 | HIGH — removes the 12-file boilerplate; round-trip already proven | medium (script ~300? + fixture ~300 + bats ~80) | YES: TC1 missing arg → 2; TC2 invalid name → 2; TC3 emitted tree byte-equals the fixture; TC-K8 identical under `HOME=/nonexistent`; TC4 round-trip build+gate (skipped when no niagara_home, never faked) | no |
| D | types/dashboard.md fold of B796 (DashboardPan -ux as the write-surface exemplar; gate 4 absent → #49) | — | HIGH — the kit's thinnest surface | ~20 lines | none | no |
| E | `schema-risk.sh` (#46): embed B795 §795.4 CSV; two-snapshot slot diff (`module-include.xml` + slot declarations) → per-change classification; worst-cell verdict; unknown → OUTAGE | #46 | HIGH — pre-deploy guard for the ClassCastException boot-loop class | medium (~200) | YES: fixtures for add (SAFE), remove (LOSSY), retype simple (OUTAGE), reorder (SAFE), rename (LOSSY), unknown kind (OUTAGE); mutation: worst-cell → first-cell | no (B795 landed) |
| F | types/logic.md split (+ SKILL.md decision table + BUILD-LOOP §2 refs, kit-links.bats extended) | — | MED — cognitive load | ~70 (moves) | none; kit-links validates | atomic with SKILL.md |
| G | `verify-module.sh --plano` (#47) | #47 | MED | ~100 bash + ~40 bats | YES: PL1 two disagreeing aspect-ratio → FAIL; mutation count-only | pending companero's B797 spec |
| H | `report-module.sh` punch-list mode (#49 evidence; runs preflight? no — lint-timers + slot-coverage + verify --src and prints one report) | #49 | MED | ~40 bash + 3 bats | YES: report aggregates a FAIL from lint-timers (mutation: drop aggregation) | after B798 baseline |

---

## 3. Campaign Structure — chained PRs by destination

| PR | Destination | Type | QA RED | Est. lines | Order rationale |
|---|---|---|---|---|---|
| PR1 | fold the 9 campaign-6 retros (A) | doc-only | none; fidelity read | ~40 | first: clears debt before 2026-10-05; stabilizes METHODOLOGY |
| PR2 | tool integration (B) | doc-only | none | ~20 | after PR1 (METHODOLOGY stable) |
| PR3 | dashboard.md B796 fold (D) | doc-only | none; fidelity vs B796 | ~25 | after PR2 |
| PR4 | scaffold-module.sh (C) + fixtures/MinimalPan | code | `qa/c7-scaffold` | ~700-880 → `size:exception`, fixture as its own commit | after PR3 |
| PR5 | schema-risk.sh (E) | code | `qa/c7-schema-risk` | ~200 | after PR4 |
| PR6 | --plano (G) | code | `qa/c7-plano` | ~140 | after B797 |
| PR7 | logic.md split + SKILL.md (F) | doc restructure | none; kit-links | ~70 | late (SKILL.md stable) |
| PR8 | report-module.sh (H) | code | `qa/c7-report` | ~50 | last |

Research state: B794 (scaffold round-trip), B795 (MM3 CSV), B796 (-ux exemplar) landed; B797 (--plano spec) and B798 (conformance baseline) in progress by companero. Out of scope: #50 (station-required), dependency floor matrix beyond 4.14.

---

## 4. Risks and Recommendations

1. **scaffold K8 violation (CRITICAL to fix before commit):** `MINIMOD_ROOT` hardcoded under the operator's home (prototype :22). Bundle the skeleton as `build-n4-module-kit/fixtures/MinimalPan/`, resolve via `"${BASH_SOURCE[0]%/*}/../fixtures/MinimalPan"`, prove with `HOME=/nonexistent`.
2. **`--age` deadline 2026-10-05:** PR1 must merge first.
3. **logic.md split atomicity:** SKILL.md decision table + BUILD-LOOP §2 must move in the same PR; extend kit-links.bats.
4. **schema-risk slot diff is novel:** fixtures must cover all classes; classification is table-driven from the CSV, not hand-coded.
5. **PR4 budget:** declare `size:exception`; fixture commit first, then script, then bats.
6. **Round-trip test in CI:** TC4 needs Java 8 + niagara_home → SKIP in CI (not a fake PASS), run locally by QA before bless (like PR2's CI guard pattern in reverse: local-only proof recorded in the PR body).
