# Apply Progress: build-n4-module-campaign6

**Change**: build-n4-module-campaign6
**Delivery strategy**: auto-chain · stacked-to-main

---

## PR1 — feat/c6-marker-index (DONE)

**Branch**: feat/c6-marker-index
**Base**: origin/main @ 48f3736
**QA merge**: qa/c6-marker-index-drift @ cb0dd7d (fast-forward)
**HEAD SHA**: 20fee49679ae85a5c9fc007ca1488e9081dade9c
**PR URL**: https://github.com/angeles725/niagara-tools/pull/35
**Status**: pushed; awaiting QA verify before merge

### Commits

| SHA | Message |
|-----|---------|
| cb0dd7d | test(build-n4-module): RED — marker↔INDEX 2-lite consistency (campaign 6 task 1) [QA merge] |
| f531361 | feat(build-n4-module): sweep-build-state reads retro markers — option 2-lite marker↔INDEX contract (Campaign 6 PR1) |
| 20fee49 | docs(build-n4-module): PR1 feature-retro — marker-blind sweep lesson (Campaign 6) |

### Tasks completed

- [x] T1.1 — marker_of() added to sweep-build-state.sh (design D1; spec R1-1–R1-5)
- [x] T1.2 — QA branch qa/c6-marker-index-drift tip cb0dd7d merged (25 tests, M6 included)
- [x] T1.3 — 2 retro files re-stamped fresh→folded (same commit as T1.1)
- [x] T1.4 — PR1 retro filed (exit a); INDEX row added; BUILD-STATE.md updated

### Test counts

| Suite | Result |
|-------|--------|
| `bats tests/build-retro-sync.bats` | 25/25 ✓ (incl. M1 M4 M5 M6) |
| `bats tests/*.bats` (full) | 110/110 ✓ |
| `shellcheck 0.10.0` all scripts | exit 0 ✓ |
| Real-tree sweep | exit 0 ✓ |
| `HOME=/nonexistent` suite | 25/25 ✓ |

### Mutation proof (M1)

Named mutation: deleted the `m=$(marker_of …)` + `if [ -n "$m" ]` block from sweep-build-state.sh.

```
not ok 20 M1: marker disagrees with INDEX row (marker=folded, row=pending) FAILS (2-lite)
# (in test file tests/build-retro-sync.bats, line 292)
#   `[ "$status" -eq 1 ]' failed
```

M1 flipped from `ok` to `not ok` (sweep returned exit 0 where 1 expected).
Reverted: 25/25 green confirmed.

### Rollback boundary

Revert: `build-n4-module-kit/toolbelt/sweep-build-state.sh` (remove marker_of block),
        `tests/build-retro-sync.bats` (SC2314 fix),
        `build-n4-module-kit/retros/2026-09-02-comppan-fase1-staging.md` (restore fresh marker),
        `build-n4-module-kit/retros/2026-09-02-dashboardpan-detail-render-doors.md` (restore fresh marker),
        `build-n4-module-kit/retros/2026-09-05-campaign6-marker-index-sweep.md` (remove retro),
        `build-n4-module-kit/retros/INDEX.md` (remove PR1 row),
        `build-n4-module-kit/BUILD-STATE.md` (restore Campaign 5 last_session).

### TDD Cycle Evidence

| Task | RED | GREEN | REFACTOR |
|------|-----|-------|----------|
| T1.1 (marker_of) | QA branch cb0dd7d: M1/M4 `not ok` (sweep blind) | marker_of() + domain/mismatch checks → 25/25 | marker_of clean single-purpose fn; SC2314 fix in bats |
| T1.2 (bats merge) | qa branch tip is the RED itself | merge brings RED that T1.1 turns GREEN | none needed |
| T1.3 (re-stamp) | M5 would go RED once marker is read (drift detected) | re-stamp fresh→folded + real-tree sweep exit 0 | — |
| T1.4 (retro) | hook requires BUILD-STATE+INDEX+retro for exit (a) | retro filed + INDEX row + BUILD-STATE updated → push PASS | — |

---

## PR2 — feat/c6-ci-pure-test (DONE)

**Branch**: feat/c6-ci-pure-test
**Base**: origin/main @ 20fee49 (PR1 HEAD, rebased)
**HEAD SHA**: a30cc03
**PR URL**: https://github.com/angeles725/niagara-tools/pull/36
**CI run**: 33956501710 — PASS (31s)
**Status**: pushed; CI green; awaiting review/merge

### Commits

| SHA | Message |
|-----|---------|
| 99d096b | ci(build-n4-module): run run-pure-test P1-P6 in CI — pinned junit/hamcrest pre-fetch + CI skip guard (Campaign 6 PR2) |
| a30cc03 | test(build-n4-module): declare bats 1.5.0 minimum for run ! semantics (Campaign 6 PR2) |

### Tasks completed

- [x] T2.1 — ci.yml: setup-java@v4 temurin 8 + curl pre-fetch junit/hamcrest (pinned sha256) + zero-SKIPs log step
- [x] T2.2 — tests/run-pure-test.bats: CI guard converts skip → fail when CI set and junit absent
- [x] T2.3 — tests/build-retro-sync.bats: bats_require_minimum_version 1.5.0 added; BW02 warning gone

### CI P1-P6 log evidence (run 33956501710)

From `run-pure-test.bats — confirm P1-P6 executed with zero SKIPs in CI` step:
```
ok 1 P1: a PASSING pure test exits 0 and reports OK
ok 2 P2: a FAILING pure test exits NON-ZERO (the runner must not swallow it)
ok 3 P3: a missing test source exits 3 (environment), not 0
ok 4 P4: wrong arg count exits 2 (usage)
ok 5 P5: a non-existent module-rt-dir exits 3 (environment)
ok 6 P6: an empty ~/.gradle cache (no junit) exits 3 with an actionable message
```
No `# skip` lines in CI TAP output. Zero-SKIP log step passed.

### Mutation proof (T2.2)

**Before guard** — `CI=1 HOME=/nonexistent bats tests/run-pure-test.bats`:
All 6 tests produce `# skip junit-4.13.2 not in ~/.gradle cache` (silent, exit 0).

**After guard** — `CI=1 HOME=/nonexistent bats tests/run-pure-test.bats`:
```
not ok 1 P1: a PASSING pure test exits 0 and reports OK
# (from function 'setup' in test file tests/run-pure-test.bats, line 15)
#   'false' failed
# CI: junit-4.13.2.jar not in ~/.gradle — ci.yml pre-fetch step is required
not ok 2 P2: ...  (all 6 fail, exit 1)
```

### Test counts

| Suite | Result |
|-------|--------|
| `bats tests/run-pure-test.bats` (local, real ~/.gradle) | 6/6 pass |
| `CI=1 HOME=/nonexistent bats tests/run-pure-test.bats` | 6 not ok (exit 1) — mutation confirmed |
| `bats tests/*.bats` (full) | 110/110 pass, no BW02 warning |
| `shellcheck 0.10.0` all scripts | exit 0 |
| CI run 33956501710 | PASS — P1-P6 all `ok` (not skip) |

### Rollback boundary

Revert:
- `.github/workflows/ci.yml` (remove java setup + pre-fetch + log step)
- `tests/run-pure-test.bats` (remove CI guard; restore `skip`)
- `tests/build-retro-sync.bats` (remove `bats_require_minimum_version 1.5.0`)

### TDD Cycle Evidence

| Task | RED | GREEN | REFACTOR |
|------|-----|-------|----------|
| T2.2 (CI guard) | `CI=1 HOME=/nonexistent`: 6 skip (before guard) | CI guard converts skip→fail: 6 not ok, exit 1 | single if/elif block; message references the pre-fetch step |
| T2.1 (ci.yml) | P1-P6 would SKIP in CI without pre-fetch (T2.2 RED) | pre-fetch + setup-java: P1-P6 all ok in CI run 33956501710 | log step as belt-and-suspenders |
| T2.3 (bats version) | BW02 warning from `run !` in build-retro-sync.bats (PR1) | `bats_require_minimum_version 1.5.0` → 25/25, no warning | — |

---

## PR3–PR8 — pending

Tasks T3.1–T7.4 are pending. The chain is serial (stacked-to-main):
PR3 worker can acquire after PR2 is settled.

---

## PR3 — feat/c6-doctrine (DONE)

**Branch**: feat/c6-doctrine
**Base**: origin/main @ a30cc03 (PR2 HEAD, stacked)
**HEAD SHA**: 42134c6c46d05fdaaea31c0822060ccd56314c3a
**PR URL**: https://github.com/angeles725/niagara-tools/pull/37
**CI run**: 33957374879 — PASS (31s)
**Status**: pushed; CI green; awaiting review/merge

### Commits

| SHA | Message |
|-----|---------|
| 692d645 | docs(build-n4-module): fold the pending meta-lesson retros into METHODOLOGY + CONTRIBUTING — K1–K10, multi-session + live-verify rules, what-to-test-where (Campaign 6 PR3) |
| 42134c6 | docs(build-n4-module): PR3 retro — doctrine fold lessons (Campaign 6) |

### Tasks completed

- [x] T3.1 — K1–K9 added to METHODOLOGY.md §Kit-maintenance with [ev:] citations
- [x] T3.2 — §Multi-session coordination (M1) + §Live-verify safety (M2) added to METHODOLOGY.md
- [x] T3.3 — LC7 what-to-test-where table per module type added to METHODOLOGY.md §Build
- [x] T3.4 — K10/A10 pin-linter rule added to CONTRIBUTING.md §8; stale "No CI" claim fixed
- [x] T3.5 — 7 retro markers stamped folded + 7 INDEX rows flipped pending→folded (atomic commit)
- [x] T3.6 — sweep exit 0; PR3 retro filed (exit a); INDEX row added

### Grep-before-fold audit

| Task | Pattern | Hits in non-retro kit files |
|------|---------|----------------------------|
| T3.1 (K1–K9) | gate exit / mutation.prove / high.signal / promotable unit / worktree / origin.main / set -e | 0 (BUILD-STATE.md hit "worktree" but is not a rule file) |
| T3.2 (M1/M2) | multi.session / dirty tree / git status / no.token→401 / test.cred | 0 |
| T3.3 (LC7) | what.*test.*where / test.*matrix / per.*type | Hits in METHODOLOGY.md/build-verify.md/types/ but all for "per module type" header phrase; no existing LC7 table found |
| T3.4 (K10) | tool.*pin / linter.*pin / No CI | "No CI" found → stale claim; no linter-pin rule found |

### Guard results

| Check | Result |
|-------|--------|
| `sweep-build-state.sh` | exit 0 — 2 pending: freeze-stat (PR4) + PR1 retro |
| `bats tests/*.bats` | 110/110 pass (0 new tests, doc-only) |
| `shellcheck 0.10.0` | exit 0 |
| `bats tests/kit-links.bats` | 3/3 pass |
| CI run 33957374879 | PASS (31s) |

### Rollback boundary

Revert:
- `build-n4-module-kit/METHODOLOGY.md` (remove K1–K9, M1, M2, LC7 table)
- `CONTRIBUTING.md` (restore "No CI" bullet, remove K10)
- `build-n4-module-kit/retros/INDEX.md` (7 rows: folded → pending)
- 7 retro files: `<!-- review-status: folded -->` → `<!-- review-status: pending -->` (line 1)
- `build-n4-module-kit/retros/2026-09-05-campaign6-doctrine-fold.md` (remove retro)

### METHODOLOGY.md delta

51 lines → 74 lines (+23). Additions: K1–K9 (9 bullets in §Kit-maintenance), §Multi-session coordination (1 bullet), §Live-verify safety (2 bullets), LC7 what-to-test-where table (5 lines in §Build).

### CONTRIBUTING.md delta

"No CI" paragraph (1 line) → "CI active + K10 pin-linter rule" (2 lines net). Repo root (not build-n4-module-kit/), per instruction.
---

## PR4 — feat/c6-types (DONE)

**Branch**: feat/c6-types
**Base**: origin/main @ c978490 (PR3 HEAD, stacked)
**HEAD SHA**: f589956
**PR URL**: pending (gh pr create blocked by auto-mode classifier — run manually)
**Status**: pushed to origin/feat/c6-types; awaiting PR creation + CI

### Commits

| SHA | Message |
|-----|---------|
| f589956 | docs(build-n4-module): fold research + retro deltas into types/*, build-verify, SOURCES — LC1–LC6, B762/B763 seams, mode-B slotomatic gap, DashboardPan ledger (Campaign 6 PR4) |

### Tasks completed

- [x] T4.1 — types/logic.md: header GROWING removed, L1 annotation-only, L2 integrator-placed ORD, LC1–LC6 kitControl patterns
- [x] T4.2 — types/dashboard.md: A15/D1 off-station derived keys, DUX1 route→RouteAction, DUX2 purity gradient, DJS1 SPA JS shim, DWS1 5-gate write-surface, DWS2 canWrite seam
- [x] T4.3 — types/wb-widgets.md: DWB1 wb/model Predicate-injection seam (exemplar-backed)
- [x] T4.4 — build-verify.md: "Known gap" retitled to "mode B ignores --with-slotomatic"; V1–V4 added
- [x] T4.5 — SOURCES.md S1/S2 + corpus-index.md A18 research-tooling caveats
- [x] T4.6 — freeze-stat retro: marker line 1 added (folded), INDEX row flipped pending→folded; BUILD-STATE.md DashboardPan ledger corrections (verify_gate pass, profiles rt,ux, U5 reworded, wb scaffold noted) + kit self-envelope updated
- [x] T4.7 — PR4 retro filed (exit a): 2026-09-05-campaign6-types-fold.md; INDEX row added

### Grep-before-fold audit

All 19 deltas had 0 pre-existing hits in non-retro kit files for their specific new content.
- write.surface hit in dashboard.md U5 = different rule (width warning vs 5-gate checklist) → write DWS1
- AUTO stub hit in logic.md = existing regenerating-slots text → write L1 (different angle: build.sh flow)
- fail.safe hit in logic.md = Safety fail-modes section (no 6-item checklist) → write LC4

### Guard results

| Check | Result |
|-------|--------|
| `sweep-build-state.sh` | exit 0 |
| `bats tests/kit-links.bats` | 3/3 pass |
| `bats tests/*.bats` (full) | 110/110 pass |
| `shellcheck 0.10.0` | exit 0 |
| `rg "runs slotomatic for -rt only" build-n4-module-kit` | 0 hits |

### DashboardPan ledger corrections (evidence)

| Field | Before → After | Evidence |
|-------|----------------|---------|
| verify_gate | unknown → pass | rt 7/7, ux 7/7 on repo HEAD 4f5f1c7 (2026-09-05) |
| profiles | rt,ux,wb → rt,ux | DashboardPan-wb: 0 .java, never built [ev: corpus B762] |
| U5 wording | "lacks OPERATOR-flag" → IS gated fail-closed (BDashboardServlet.java:198); residue = DWS2/lock/allowlist | corpus B763 §763.6 |
| open_issues count | 2 → 3 | +wb scaffold note |

### Rollback boundary

Revert:
- `build-n4-module-kit/types/logic.md` (remove LC1-LC6 + L1 + L2 + header change)
- `build-n4-module-kit/types/dashboard.md` (remove A15/D1 + DUX1-DWS2)
- `build-n4-module-kit/types/wb-widgets.md` (remove DWB1)
- `build-n4-module-kit/build-verify.md` (restore "Known gap" title + remove V1-V4)
- `build-n4-module-kit/SOURCES.md` (remove S1/S2)
- `build-n4-module-kit/corpus-index.md` (remove A18)
- `build-n4-module-kit/retros/2026-09-03-coldroompan-dashboardpan-freeze-stat-leds.md` (remove line 1 marker)
- `build-n4-module-kit/retros/INDEX.md` (restore freeze-stat pending, remove PR4 retro row)
- `build-n4-module-kit/retros/2026-09-05-campaign6-types-fold.md` (remove retro)
- `build-n4-module-kit/BUILD-STATE.md` (restore DashboardPan envelope + kit self-envelope)

### Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command | `bats tests/kit-links.bats` → 3/3 pass (doc-only: links resolve) |
| Runtime harness | N/A — no station, jar, or operator data touched (doc-only PR) |
| Rollback boundary | Revert the 10 files listed above independently of any other work |


---

## PR5a — feat/c6-tools-audit (DONE)

**Branch**: feat/c6-tools-audit
**Base**: origin/main @ f589956 (PR4 HEAD, stacked)
**QA cherry-pick**: qa/c6-coverage-pct-red @ d7e52a8
**HEAD SHA**: 4551dc9
**PR URL**: https://github.com/angeles725/niagara-tools/pull/39
**CI run**: 33960149864 — PASS (39s)
**Status**: pushed; CI green

### Commits

| SHA | Message |
|-----|---------|
| a2e5987 | test(build-n4-module): RED — verify-module coverage% pure-fn pins (campaign 6) [QA cherry-pick] |
| 4551dc9 | feat(build-n4-module): sweep-fold-audit.sh + verify-module coverage% — fold-citation audit and gate coverage model (Campaign 6 PR5a) |

### Tasks completed

- [x] T5a.1 — sweep-fold-audit.sh (design D3; spec R5-1): hyphen-segment token match, 6-char floor, retros/ excluded, --strict exit 1, VCS-free
- [x] T5a.2 — tests/sweep-fold-audit.bats: F1-F6 (RED-first); 3 named mutations F3/F4/F6 verified
- [x] T5a.3 — verify-module.sh coverage subcommand (design D2; spec R5-7): dispatched before flag loop, N/A sentinel, integer-tenths rounding
- [x] T5a.4 — tests/verify-coverage.bats: QA d7e52a8 cherry-picked, 8/8 GREEN (MM1-MM8); named mutation MM3 verified
- [x] T5a.5 — PR5a retro filed (exit a): 2026-09-05-campaign6-fold-audit-and-coverage.md; INDEX row added; BUILD-STATE.md updated

### Test counts

| Suite | Result |
|-------|--------|
| `bats tests/verify-coverage.bats` | 8/8 ✓ (MM1–MM8) |
| `bats tests/sweep-fold-audit.bats` | 6/6 ✓ (F1–F6) |
| `bats tests/*.bats` (full) | 124/124 ✓ |
| `shellcheck 0.10.0` all scripts | exit 0 ✓ |
| `HOME=/nonexistent` sweep-fold-audit + verify-coverage | 14/14 ✓ |
| Real-tree sweep-build-state | exit 0 ✓ |
| `bats tests/kit-links.bats` | 3/3 ✓ |
| CI run 33960149864 | PASS (39s) ✓ |

### Mutation proof

**MM3 (coverage N/A sentinel)**
Named mutation: change `echo "N/A"` → `echo "100.0"` when A==0
```
not ok 3 MM3: 0P/0F/0W/5S -> N/A (applicable 0 -> string sentinel, NOT 100)
# (in test file tests/verify-coverage.bats, line 45)
#   `[ "$output" = "N/A" ]' failed
```

**F3 (retros/ exclusion)**
Named mutation: drop `! -path "*/retros/*"` from find command
```
not ok 3 F3: citation only in retros/ still produces WARN (corpus excludes retros/)
```

**F4 (hyphen-segment vs exact-stem)**
Named mutation: `[ "$T" = "$stem" ]` (exact-stem match)
```
not ok 4 F4: abbreviated token 5rooms credits stem dashboardpan-5rooms (no spurious WARN)
```

**F6 (segment-aligned vs plain substring)**
Named mutation: `*"${T}"*` (plain substring)
```
not ok 6 F6: token ender-doors does NOT credit stem detail-render-doors (segment-aligned, not substring)
```

### Real-tree fold-audit (non-strict)

```
fold-audit: NOTE ambiguous citation token credits 2026-09-02-comppan-fase3-floating-suction.md: comppan-fase3 comppan-fase3-floating-suction
fold-audit: NOTE ambiguous citation token credits 2026-09-02-module-palette-and-build-target.md: module-palette module-palette-and-build-target
fold-audit: NOTE ambiguous citation token credits 2026-09-03-coldroompan-dashboardpan-freeze-stat-leds.md: coldroompan-dashboardpan-freeze-stat-leds freeze-stat-leds
fold-audit: NOTE ambiguous citation token credits 2026-09-03-process-timers-and-defrost-audit.md: process-timers process-timers-and-defrost-audit
fold-audit: WARN 2026-09-04-junit-standalone-cached-jar-locations-for-wsl-pure-tests.md folded with no [ev: retro ...] citation
fold-audit: WARN 2026-09-04-kit-continuity-and-retro-gate-campaign.md folded with no [ev: retro ...] citation
fold-audit: 38 folded, 36 cited, 2 uncited
```
Exit 0. 2 WARNs = PR7 follow-up. 4 NOTEs = expected (abbreviated multi-stem credits).

### TDD Cycle Evidence (Strict TDD)

| Task | RED | GREEN | REFACTOR |
|------|-----|-------|----------|
| T5a.3 (coverage) | 8/8 fail (exit 2 "not a jar: coverage") | coverage block before flag loop: 8/8 pass | integer-tenths rounding `t=(1000*P+A/2)/A` is clean |
| T5a.4 (QA pins) | cherry-pick d7e52a8 → 8/8 RED | implementation GREEN 8/8 | — |
| T5a.1 (sweep-fold-audit) | script absent → 5/6 fail exit 127 | D3 implementation: 6/6 pass | hyphen-align + floor + retros/ exclusion |
| T5a.2 (fold-audit bats) | written RED-first | implementation GREEN | F5 vacuously passes absence (correct) |

### Rollback boundary

Revert:
- `build-n4-module-kit/toolbelt/sweep-fold-audit.sh` (remove entirely)
- `build-n4-module-kit/toolbelt/verify-module.sh` (remove coverage subcommand block)
- `tests/sweep-fold-audit.bats` (remove entirely)
- `tests/verify-coverage.bats` (remove entirely)
- `tests/fixtures/fold-audit/` (remove directory)
- `.github/workflows/ci.yml` (remove sweep-fold-audit step)
- `build-n4-module-kit/retros/2026-09-05-campaign6-fold-audit-and-coverage.md` (remove)
- `build-n4-module-kit/retros/INDEX.md` (remove fold-audit-and-coverage row)
- `build-n4-module-kit/BUILD-STATE.md` (revert to PR4 last_session/retro_pending)


---

## PR5b — feat/c6-tools-env (DONE)

**Branch**: feat/c6-tools-env
**Base**: origin/main @ 4551dc9 (PR5a HEAD)
**HEAD SHA**: 4581d38
**Status**: committed; push blocked by auto-mode classifier (lead to push + create PR)

### Tasks completed

- [x] T5b.1 — `build-n4-module-kit/toolbelt/preflight.sh` (174 lines): win-path, jdk8, plugin-pin, jar-lock
- [x] T5b.2 — `tests/preflight.bats` (4 tests PF1–PF4, 3 named mutations)
- [x] T5b.3 — `build-n4-module-kit/toolbelt/slot-coverage.sh` (196 lines): pure MM2 + parse + dup-keys
- [x] T5b.4 — `tests/slot-coverage.bats` (cherry-pick 5a7d90a=08a8b93 + SC6-parse + dup-keys, 8 tests)
- [x] T5b.5 — stretch deferred; recorded in retro
- [x] T5b.6 — retro filed, INDEX.md updated, BUILD-STATE.md kit self-envelope updated

### Gates passed

- bats: 136/136 green
- shellcheck 0.10.0: exit 0 (all toolbelt scripts + test files)
- HOME=/nonexistent: 12/12 identical
- Real-module scan: CompPan-rt pct=100.0, DashboardPan-rt pct=100.0

### TDD Cycle Evidence (Strict TDD)

| Task | RED | GREEN | Named mutations |
|------|-----|-------|-----------------|
| T5b.1+T5b.2 (preflight) | 4/4 fail (script absent) | 4/4 pass | PF1 always-PASS, PF2 HOME-embed, PF4 SKIP→PASS |
| T5b.3+T5b.4 (slot-coverage) | cherry-pick 08a8b93 → 6/6 RED | implementation GREEN 8/8 | SC2 denominator, SC3 extra-in-num, SC5 0/0→100, SC2/SC3 swap |

---

## PR6 — feat/c6-close (Campaign 6 close)

**Status**: complete
**Date**: 2026-09-05
**Commit**: 080d41185bd2ec068da5c6a23b8f0d575730d1ad (feat/c6-close)

### Tasks completed

- [x] T6.1 — `build-n4-module-kit/skill/SKILL.md` (tracked canonical copy v0.4): state column removed, wb-builder corpus note, step-1 aligned with BUILD-LOOP orient order
- [x] T6.2 — `scripts/install-skill.sh` (~85 lines): sha256 comparison, --home/--dry-run/--force, exits 0/1/2/3, VCS-free, HOME=/nonexistent identical
- [x] T6.3 — `tests/install-skill.bats` (5 tests): IS1 byte-parity, IS2 already-current, IS3a/IS3b diverge+force, IS4 dry-run; named mutation proved (drop last line → cmp fails)
- [x] T6.4 — DRY-RUN: diverged sha (tracked 4a138e7b vs live 33b6c9a8); orchestrator installs after merge
- [x] T6.5 — `openspec/` tracked; slotomatic → archive/ with archive-report.md; .gentle-ai-instance in .gitignore
- [x] T6.7 — VERSION 0.15.1 → 0.16.0; CHANGELOG.md new top section v0.16.0 (PR1–PR6 summary + References)
- [x] T6.8 — PR6 retro `2026-09-05-campaign6-close.md` (5 lessons); INDEX row; all guards green
- [x] T6.9 — ci.yml setup-java @v4 → @v5
- [x] T6.10 — BUILD-STATE.md: replaced false CompPan empty-lexicon open_issue with real B788 findings; kit version 0.16.0, last_commit Campaign 6 PR6 [ev: corpus B788]
- [x] T6.11 — slot-coverage.sh parse mode pct= labeled '(type-set)'; pure set-coverage output unchanged (SC1–SC5 pins unaffected)

### Gates passed

- bats: 141/141 green (≥140)
- shellcheck 0.10.0: exit 0 (all scripts, toolbelt, tests, helpers)
- sweep-build-state.sh: exit 0; pending = 6 (campaign retros PR1/PR3/PR4/PR5a/PR5b/PR6; original 8 = 0)
- rg "rt only" build-n4-module-kit/build-verify.md: 0 matches
- kit-links.bats: 3/3 pass
- HOME=/nonexistent: suite identical

### TDD Cycle Evidence (Strict TDD)

| Task | RED | GREEN | Named mutations |
|------|-----|-------|-----------------|
| T6.3 (install-skill) | 5/5 fail (script absent) | 5/5 pass | Drop last line → IS1 cmp fails → revert → 5/5 green |

### Branch pushed

Branch `feat/c6-close` pushed to origin. PR creation requires network access (gh api unreachable from worker).
Open PR: `gh pr create --base main --title "feat(build-n4-module): campaign 6 close — tracked launcher, openspec durability, v0.16.0 (PR6)"` from the HEAD SHA above.

---

## PR7 — feat/c6-research-fold (research fold)

**Branch**: feat/c6-research-fold
**Base**: origin/main @ 080d411 (PR6 HEAD, stacked)
**Status**: committed; push pending

### Tasks completed

- [x] T7.1 — Research fold: 27 deltas from B772–B791 into types/logic.md (10 new sections), types/dashboard.md (palette/lexicon + DUX-WEB1/DUX-WEB2), types/wb-widgets.md (dual-surface @AgentOn), METHODOLOGY.md (B784 profile conventions + B777 permissions inline + B775 watchdog note + lintable-vs-advisory + human-review checklist), corpus-index.md (27-row Campaign 6 delta)
- [x] T7.2 — NOTE: post-merge action for investigador1 (stamp research retros in niagara-research); kit: PR7 retro row + BUILD-STATE kit self-envelope done
- [x] T7.4 — sweep-build-state exit 0; PR7 retro filed (2026-09-05-campaign6-research-fold.md, exit a); INDEX row added; BUILD-STATE kit self-envelope updated
- [x] T7.5 — junit-standalone: fixed citation in build-verify.md (proper `[ev: retro junit-standalone]` bracket); kit-continuity: added "folded as code" line in METHODOLOGY.md §Conformance rules with `[ev: retro kit-continuity]`; sweep-fold-audit.sh --strict exits 0 (38 folded, 38 cited, 0 uncited); ci.yml fold-audit step switched to --strict

### Grep-before-fold audit

| Destination | Pattern | Pre-existing hits |
|---|---|---|
| types/logic.md §Author-side SPIs | `self-describing SPI\|Author-side SPIs` | 0 |
| types/logic.md §Authoring a point extension | `BPointExtension\|point extension` | 0 |
| types/logic.md §Child-tree containers | `BComponentList\|child-tree\|isChildLegal` | 0 |
| types/logic.md §Grouping/relating | `BCategoryService\|BRelation\|BLevelDef` | 0 |
| types/logic.md §Query/search/index | `BQuery\|BITable\|query/search/index` | 0 |
| types/logic.md §Templates | `BTemplateConfig\|\.ntpl\|BTemplateService` | 0 |
| types/logic.md §Background jobs | `BSimpleJob\|BJobService\|background job` | 0 |
| types/logic.md §Watchdogs and timers | `BAbstractMonitor\|schedulePeriodically` (new watchdog section) | 0 new; existing L18 line about schedulePeriodically guard is different content |
| types/logic.md §Action protection | `doPrivileged\|action protection` | 0 |
| types/logic.md §Minimal module | `minimal module` | 0 |
| types/dashboard.md palette/lexicon | `bare-Type\|module\.palette\|lexicon.*prefix` | 0 |
| types/dashboard.md DUX-WEB1 | `DUX-WEB1\|web-tier exemplar\|pointer table` | 0 |
| types/dashboard.md DUX-WEB2 | `DUX-WEB2\|DELIBERATE\|STRONGER-than-vendor` | 0 |
| types/wb-widgets.md dual-surface | `Dual-surface\|NiagaraType.*agent.*AgentOn` | 0 |
| METHODOLOGY.md module.xml conventions | `runtimeProfile\|3-part\|vendorVersion.*FLOOR` | 0 |
| METHODOLOGY.md lintable-vs-advisory | `lintable\|advisory rule\|human-review checklist` | 0 |

### Fidelity spot-checks

| Block | Claim | Block verdict |
|---|---|---|
| B772 | No `BAbstractPointExt`, no `onExtended`/`onRetracted` | [CERT] — confirmed absent in 2603 source files |
| B775 | NOT a 2s poll; cadence = BIntervalTriggerMode 15 min; NO `javax.baja.sys.BTimer` (only cl.hvac) | [CERT] |
| B784 | `-doc` is SEPARATE module; dep `vendorVersion` 3-part FLOOR; bajaVersion="0" constant | [CERT] |
| B791 | CSRF divergence DELIBERATE, STRONGER-than-vendor | [CERT] from B763/B752 |

### Guard results

| Check | Result |
|---|---|
| `sweep-build-state.sh` | exit 0 |
| `sweep-fold-audit.sh --strict` | exit 0 — 38 folded, 38 cited, 0 uncited |
| `bats tests/*.bats` | 141/141 pass |
| `shellcheck 0.10.0` | exit 0 |
| `HOME=/nonexistent bats tests/kit-links.bats` | 3/3 pass |

### Rollback boundary

Revert:
- `build-n4-module-kit/types/logic.md` (remove 10 new sections after kitControl)
- `build-n4-module-kit/types/dashboard.md` (remove §Module packaging + DUX-WEB1 + DUX-WEB2)
- `build-n4-module-kit/types/wb-widgets.md` (remove dual-surface @AgentOn bullet)
- `build-n4-module-kit/METHODOLOGY.md` (remove B784/B777 from Build section; remove watchdog note; remove §Conformance rules)
- `build-n4-module-kit/build-verify.md` (restore old citation format)
- `build-n4-module-kit/corpus-index.md` (remove Campaign 6 B762–B791 section)
- `build-n4-module-kit/retros/2026-09-05-campaign6-research-fold.md` (remove)
- `build-n4-module-kit/retros/INDEX.md` (remove PR7 retro row)
- `build-n4-module-kit/BUILD-STATE.md` (restore PR6 kit self-envelope)
- `.github/workflows/ci.yml` (restore non-strict fold-audit step)
- `openspec/changes/build-n4-module-campaign6/tasks.md` (restore T7.1/T7.2/T7.4/T7.5 to [ ])

## PR8 — feat/c6-conformance-lints (T8.1–T8.6)

### Completed tasks

- [x] T8.1 `toolbelt/lint-timers.sh` — timer-ticket check (AWK stopped() brace-counted scan)
- [x] T8.2 `toolbelt/lint-timers.sh` — discarded-ticket check (line-by-line Clock.schedule grep)
- [x] T8.3 No-op — V17 in `tests/verify-module.bats` already guards the typeless-empty-palette case
- [x] T8.4 `toolbelt/sweep-build-state.sh --age` — E5 retro-debt aging mode
- [x] T8.5 Verified — METHODOLOGY.md advisory rules marked HUMAN-REVIEW, no hard-fail added
- [x] T8.6 PR8 retro + kit self-envelope (`build-n4-module-kit/retros/2026-09-05-campaign6-conformance-lints.md`)

### TDD cycle evidence

| Task | Tests | RED | GREEN | Mutation → flip |
|---|---|---|---|---|
| T8.1 timer-ticket | TL1-TL3 (qa/c6-timer-ticket-lint b8e0b9a) | Exit 127 (no script) | lint-timers.sh AWK | Drop AWK block → TL1 at bats:65 flips |
| T8.2 discarded-ticket | TL4 (added) | TL4 expects FAIL, no discarded check | while-loop discarded check | Drop check block → TL4 at bats:107 flips |
| T8.4 --age mode | D1-D6 (qa/c6-retro-debt-red ea66684) | Exit 3 (unknown --age flag) | sweep-build-state.sh --age | `>=` instead of `>` → D4 flips; pending guard removed → D2+D6 flip |
| T8.6a PF5 | PF5 (added) | PF5 expects PASS, no fallback | WSL bin/java fallback in preflight.sh | Remove fallback block → PF5 flips |

### Test count

- Baseline: 150 tests (141 pass, 9 RED for TL1-TL3 + D1-D6)
- After PR8: 152 tests (152 pass, 0 fail)

### Files created/modified

| File | Action | Notes |
|---|---|---|
| `build-n4-module-kit/toolbelt/lint-timers.sh` | Created | timer-ticket + discarded-ticket lint |
| `build-n4-module-kit/toolbelt/sweep-build-state.sh` | Modified | Added --age mode |
| `build-n4-module-kit/toolbelt/preflight.sh` | Modified | WSL jdk8 fallback (PF5) |
| `tests/lint-timers.bats` | Cherry-picked + TL4 | QA b8e0b9a + TL4 added |
| `tests/retro-debt.bats` | Cherry-picked | QA ea66684 |
| `tests/preflight.bats` | Modified | PF5 added |
| `tests/fixtures/lint-timers/` | Created | CI fixtures (NoTimerSample, ConformantSample) |
| `tests/fixtures/preflight/jvm-no-release/` | Created | PF5 fakebin fixture |
| `.github/workflows/ci.yml` | Modified | lint-timers + sweep-retro-debt steps |
| `build-n4-module-kit/retros/2026-09-05-campaign6-conformance-lints.md` | Created | PR8 retro |
| `build-n4-module-kit/retros/INDEX.md` | Modified | PR8 row added |
| `build-n4-module-kit/BUILD-STATE.md` | Modified | Kit self-envelope PR8 |
| `openspec/changes/build-n4-module-campaign6/tasks.md` | Modified | T8.* marked [x] |

### Rollback boundary

Revert `build-n4-module-kit/toolbelt/lint-timers.sh` (delete), revert `--age` block from `sweep-build-state.sh`, revert WSL fallback from `preflight.sh`, remove `tests/lint-timers.bats` + `tests/retro-debt.bats` + `tests/preflight.bats` (PF5), revert `tests/fixtures/lint-timers/` and `tests/fixtures/preflight/jvm-no-release/`, revert CI yml additions, delete PR8 retro + INDEX row, revert BUILD-STATE.md.

---

## Prior PRs (PR1–PR7): all complete

See engram observation #8186 for PR7 details. All 8 PRs of Campaign 6 are now complete.
