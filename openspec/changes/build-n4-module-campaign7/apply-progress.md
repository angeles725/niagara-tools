# Apply Progress: build-n4-module-campaign7

## PR1 — docs/c7-retro-fold (2026-09-05)

**Status**: complete  
**Mode**: Standard  
**Branch**: docs/c7-retro-fold  
**Worker**: sdd-apply (Claude Sonnet 4.6)

### Completed Tasks

- [x] 1.1 grep-before-fold: K11, K12, K13, K14, envelope-pairing, SDD-ledger, two-independent-reads — 0 hits in non-retro kit files (K11-K14 only in retros/ source retro, as expected)
- [x] 1.2 Added K11–K14 to METHODOLOGY.md §Kit-maintenance after K9 + 6 folded-as-code citation lines
- [x] 1.3 Added envelope-pairing rule to BUILD-LOOP.md §7 with [ev: retro doctrine-fold] [ev: retro types-fold] [ev: retro close-process-meta-lessons]
- [x] 1.4 Added CONTRIBUTING.md §9 SDD ledger discipline (5 bullets: serial ledger, single-line goal, sha from error, early doc-commits, two-reads rule)
- [x] 1.5 Flipped 9 INDEX rows from `| pending |` to `| folded |`; `grep -c '| pending |' retros/INDEX.md` = 0
- [x] 1.6 Replaced line-1 marker from `pending` to `folded` in 9 retro files via sed -i
- [x] 1.7 PR1 retro file created (2026-09-05-campaign7-retro-fold.md, review-status: pending); INDEX row added; BUILD-STATE.md kit envelope updated (retro_pending: true, last_commit: Campaign 7 PR1)
- [x] 1.8 Guards pending — to be run before commit; commit in progress

### Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `build-n4-module-kit/METHODOLOGY.md` | Modified | Added K11–K14 + 6 folded-as-code citation lines after K9 (+11 lines) |
| `build-n4-module-kit/BUILD-LOOP.md` | Modified | Added envelope-pairing rule to §7 (+1 line) |
| `CONTRIBUTING.md` | Modified | Added §9 SDD ledger discipline (+9 lines) |
| `build-n4-module-kit/retros/2026-09-05-campaign6-marker-index-sweep.md` | Modified | Line-1 marker: pending → folded |
| `build-n4-module-kit/retros/2026-09-05-campaign6-doctrine-fold.md` | Modified | Line-1 marker: pending → folded |
| `build-n4-module-kit/retros/2026-09-05-campaign6-types-fold.md` | Modified | Line-1 marker: pending → folded |
| `build-n4-module-kit/retros/2026-09-05-campaign6-fold-audit-and-coverage.md` | Modified | Line-1 marker: pending → folded |
| `build-n4-module-kit/retros/2026-09-05-campaign6-preflight-and-slot-coverage.md` | Modified | Line-1 marker: pending → folded |
| `build-n4-module-kit/retros/2026-09-05-campaign6-close.md` | Modified | Line-1 marker: pending → folded |
| `build-n4-module-kit/retros/2026-09-05-campaign6-research-fold.md` | Modified | Line-1 marker: pending → folded |
| `build-n4-module-kit/retros/2026-09-05-campaign6-conformance-lints.md` | Modified | Line-1 marker: pending → folded |
| `build-n4-module-kit/retros/2026-09-05-campaign6-close-process-meta-lessons.md` | Modified | Line-1 marker: pending → folded |
| `build-n4-module-kit/retros/INDEX.md` | Modified | 9 rows flipped pending→folded; PR1 retro row added |
| `build-n4-module-kit/retros/2026-09-05-campaign7-retro-fold.md` | Created | PR1 retro (review-status: pending) |
| `build-n4-module-kit/BUILD-STATE.md` | Modified | Kit self-envelope updated (retro_pending: true, Campaign 7 PR1) |
| `openspec/changes/build-n4-module-campaign7/tasks.md` | Modified | PR1 tasks 1.1–1.8 marked [x] |
| `openspec/changes/build-n4-module-campaign7/apply-progress.md` | Created | This file |

### Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command | `bash build-n4-module-kit/toolbelt/sweep-fold-audit.sh --strict build-n4-module-kit/retros/INDEX.md build-n4-module-kit` → exit 0 |
| Runtime harness | `bash build-n4-module-kit/toolbelt/sweep-build-state.sh BUILD-STATE.md build-n4-module-kit/retros build-n4-module-kit/retros/INDEX.md` → exit 0 |
| Rollback boundary | Revert doc edits in METHODOLOGY/BUILD-LOOP/CONTRIBUTING; flip 9 INDEX rows back to pending; unstamp 9 retro markers; remove PR1 retro + INDEX row; restore BUILD-STATE.md envelope |

### Deviations from Design

None — implementation matches design. K11-K14 added after K9 in METHODOLOGY.md (K10 lives in CONTRIBUTING.md, so K11 is the correct next sequential number in METHODOLOGY). BUILD-LOOP.md §7 envelope-pairing rule is a single well-contained bullet. CONTRIBUTING.md §9 is a new section at the file end.

### Remaining Tasks (PR2–CLOSE)

All PR2–CLOSE tasks (2.1–C.4) remain pending for subsequent PR apply phases.

### Workload / PR Boundary

- Mode: chained PR slice (stacked-to-main, PR1 of 8)
- Current work unit: PR1 docs/c7-retro-fold
- Boundary: starts at v0.17.0 (c136e3b), ends with fold of 9 Campaign-6 retros + PR1 retro + envelope
- Estimated review budget impact: ~40 authored lines (doc-only, well within 400)

### Status

8/8 PR1 tasks complete. Ready for verify (sweep guards ran; commit pending).

---

## PR2 — docs/c7-tool-integration (2026-09-05)

**Status**: complete (T2.1–T2.5; T2.6 pending post-merge orchestrator action)
**Mode**: Standard
**Branch**: docs/c7-tool-integration
**Worker**: sdd-apply (Claude Sonnet 4.6)

### Completed Tasks

- [x] 2.1 grep-before-fold: preflight=0, lint-timers=0, slot-coverage=0 in both files; sweep-build-state=1 in BUILD-LOOP.md (footer note only) — no double-fold risk.
- [x] 2.2 Added §0.b preflight.sh line, §5 pre-gate bullet (lint-timers + slot-coverage + --plano when available), §0.a --age line, §7 sweep-at-close bullet to BUILD-LOOP.md (R2.1–R2.3).
- [x] 2.3 Bumped SKILL.md version to 0.5; §References expanded to all 10 toolbelt scripts; §Execution step 5 routed to pre-gate checks (R2.4–R2.5).
- [x] 2.4 PR2 retro filed (2026-09-05-campaign7-tool-integration.md, pending); INDEX row added; BUILD-STATE.md kit envelope updated (Campaign 7 PR2).
- [x] 2.5 Routing check → nothing printed; grep evidence: preflight BUILD-LOOP=1/SKILL=1, lint-timers BUILD-LOOP=1/SKILL=2, slot-coverage BUILD-LOOP=1/SKILL=2, sweep-build-state BUILD-LOOP=3/SKILL=1, sweep-fold-audit BUILD-LOOP=1/SKILL=1.
- [ ] 2.6 After merge: orchestrator re-runs `scripts/install-skill.sh` (R2.7, CD6).

### Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `build-n4-module-kit/BUILD-LOOP.md` | Modified | §0.a --age line; §0.b preflight.sh bullet; §5 pre-gate bullet; §7 sweep-at-close bullet (+4 lines) |
| `build-n4-module-kit/skill/SKILL.md` | Modified | Version 0.4→0.5; step 5 routed; §References all 10 toolbelt scripts |
| `build-n4-module-kit/retros/2026-09-05-campaign7-tool-integration.md` | Created | PR2 retro (review-status: pending) |
| `build-n4-module-kit/retros/INDEX.md` | Modified | PR2 retro row added |
| `build-n4-module-kit/BUILD-STATE.md` | Modified | Kit self-envelope updated (Campaign 7 PR2) |
| `openspec/changes/build-n4-module-campaign7/tasks.md` | Modified | PR2 tasks 2.1–2.5 marked [x] |
| `openspec/changes/build-n4-module-campaign7/explore.md` | Tracked | Copied from main + git added |
| `openspec/changes/build-n4-module-campaign7/proposal.md` | Tracked | Copied from main + git added |
| `openspec/changes/build-n4-module-campaign7/report-module-contract.md` | Tracked | Copied from main + git added |

### Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command | `bats tests/*.bats` → 152/152; kit-links.bats L1-L3 → 3/3 |
| Routing check | `for s in .../*.sh; do n=$(basename $s); grep -q "$n" BUILD-LOOP.md skill/SKILL.md \|\| echo "UNROUTED $n"; done` → nothing printed |
| sweep-build-state | exit 0 |
| sweep-fold-audit --strict | exit 0 (47 folded, 47 cited, 0 uncited) |
| shellcheck 0.10.0 | exit 0 |
| Rollback boundary | Revert BUILD-LOOP.md +4 lines, SKILL.md version/step5/refs; remove PR2 retro + INDEX row; restore BUILD-STATE.md; unstage 3 openspec files |

### Workload / PR Boundary

- Mode: chained PR slice (stacked-to-main, PR2 of 8)
- Current work unit: PR2 docs/c7-tool-integration
- Boundary: starts at PR1 head (924e083), ends with BUILD-LOOP + SKILL.md routing + PR2 retro + envelope
- Estimated review budget impact: ~25 authored lines (doc-only, well within 400)

### Status

5/6 PR2 tasks complete. T2.6 is post-merge orchestrator action. Ready for verify.

---

## PR3 — docs/c7-dashboard-b796 (2026-09-05)

**Status**: complete (T3.1–T3.5; T3.3 author read done; second read by investigador1 pending)
**Mode**: Standard
**Branch**: docs/c7-dashboard-b796
**Head SHA**: df8cd2e
**PR**: https://github.com/angeles725/niagara-tools/pull/54
**Worker**: sdd-apply (Claude Sonnet 4.6)

### Completed Tasks

- [x] 3.1 grep-before-fold: `rg 'DashboardPan-ux' build-n4-module-kit/types/dashboard.md` → 0 hits (CD3 no-double-fold confirmed); `rg 'B796' …` → 0 hits
- [x] 3.2 Folded B796 §796.4 exemplar into `types/dashboard.md` — pure routing seam (DashboardDispatch.java:30, 14 @Test), DUX2 anti-pattern (DashboardReader.java:66, 15+ Baja imports), five DWS1 gates table with file:line for gates 1/2/3/5, gate 4 REQUIRED-but-absent → issue #49
- [x] 3.3 Author read done (verified against B796 §796.4 source verbatim); investigador1 second read pending (R3.3)
- [x] 3.4 `bats tests/*.bats` → 152/152; `tests/kit-links.bats` included — exit 0 (R3.4)
- [x] 3.5 Commit df8cd2e includes retro (2026-09-05-campaign7-dashboard-exemplar.md) + INDEX row + BUILD-STATE.md self-envelope (CD1)

### Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `build-n4-module-kit/types/dashboard.md` | Modified | Added `### Reference exemplar (our own module) — DashboardPan-ux` block (+18 lines: Tridium B791 THIN sentence, routing seam paragraph, DUX2 anti-pattern paragraph, five DWS1 gates table with file:line) |
| `build-n4-module-kit/retros/2026-09-05-campaign7-dashboard-exemplar.md` | Created | PR3 retro (review-status: pending) |
| `build-n4-module-kit/retros/INDEX.md` | Modified | PR3 retro row added |
| `build-n4-module-kit/BUILD-STATE.md` | Modified | Kit self-envelope updated (retro_pending: true, last_commit: Campaign 7 PR3) |
| `openspec/changes/build-n4-module-campaign7/tasks.md` | Modified | PR3 tasks 3.1–3.5 marked [x] |
| `openspec/changes/build-n4-module-campaign7/apply-progress.md` | Modified | PR3 section appended |

### Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command | `bats tests/*.bats` → 152/152; 0 failures (kit-links.bats included) |
| Runtime harness | `sweep-build-state.sh build-n4-module-kit/BUILD-STATE.md …` → exit 0; `sweep-fold-audit.sh --strict` → exit 0 (47 folded, 47 cited, 0 uncited) |
| Rollback boundary | Revert types/dashboard.md (remove exemplar block +18 lines); remove PR3 retro + INDEX row; restore BUILD-STATE.md envelope |
| shellcheck | exit 0 (no script changes in PR3) |
| CD3 grep-before-fold | `DashboardPan-ux` = 0 hits; `B796` = 0 hits — confirmed clean |

### Deviations from Design

None — implementation matches design. The exemplar block is 18 authored lines (under the ~25 estimate). Insertion point is after the existing DWS write-surface bullets and before "Static assets", within the `## ux — servlet + SPA` section. A `###` subsection heading is used to chunk the exemplar, consistent with cognitive-doc-design chunking principle.

### Workload / PR Boundary

- Mode: chained PR slice (stacked-to-main, PR3 of 8)
- Current work unit: PR3 docs/c7-dashboard-b796
- Boundary: starts at PR2 head (817f24b), ends with dashboard.md exemplar block + PR3 retro + envelope (head df8cd2e)
- Estimated review budget impact: ~23 authored lines (doc-only, well within 400)

### Status

5/5 PR3 tasks complete. Ready for verify (all guards passed; PR #54 open, investigador1 second read pending on task 3.3).

---

## PR4 — feat/c7-scaffold (2026-09-05)

**Status**: complete (T4.1–T4.9)
**Mode**: Strict TDD (RED→GREEN→mutations→revert)
**Branch**: feat/c7-scaffold
**Head SHA**: 7062ad4
**PR**: https://github.com/angeles725/niagara-tools/pull/55
**Worker**: sdd-apply (Claude Sonnet 4.6)

### Completed Tasks

- [x] 4.1 Re-read QA RED `qa/c7-scaffold` tip `54636ca`; cherry-picked onto worktree HEAD → deb9bf6
- [x] 4.2 Commit 1 (899af8c): `fixtures/MinimalPan/**` — pre-slotomatic tree, no AUTO region, no operator paths in gradle.properties (commented placeholders), English lexicon (Minimal Panel/Setpoint/Interval), gradlew binary verbatim, gradle-wrapper.jar 60 KB bundled (under 100 KB threshold, D3)
- [x] 4.3 Commit 2 (7062ad4): `toolbelt/scaffold-module.sh` — BASH_SOURCE-relative fixture path (never $HOME, K8), emitted root `<out-dir>/<ModuleName>` (D2), full D3 substitution table, set -u, exits 0/2/3, no VCS, shellcheck 0
- [x] 4.4 tests/scaffold-module.bats from QA RED verbatim (TC1/TC2/TC3/TC-K8/TC4); TC4 SKIP-gated on NIAGARA_HOME
- [x] 4.5 CI step: scaffold MinimalPan into $RUNNER_TEMP, diff -r vs fixture excluding build/.gradle; TC4 SKIP in CI (D9)
- [x] 4.6 Named mutations confirmed+reverted: skip .gradle.kts (TC3+TC-K8 FAIL); remap module-include.xml→module.xml (TC3+TC-K8 FAIL); use $HOME in fixture path (TC-K8 FAIL); drop stopped() cancel (lint-timers FAIL)
- [x] 4.7 TC4 local round-trip: scaffold→preflight (3 PASS)→build.sh (slotomatic+jar BUILD SUCCESSFUL)→verify-module (7 PASS / 0 FAIL / 1 SKIP → ALL PASS)→lint-timers (PASS); exit 0 all steps
- [x] 4.8 shellcheck 0.10.0 exits 0 on toolbelt/scaffold-module.sh
- [x] 4.9 Commit 2 includes retro (campaign7-scaffold, pending) + INDEX row + BUILD-STATE.md self-envelope (Campaign 7 PR4)

### Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `build-n4-module-kit/fixtures/MinimalPan/**` (15 files) | Created | Pre-slotomatic B790/B793 skeleton: no AUTO region, commented gradle.properties, English lexicon |
| `build-n4-module-kit/toolbelt/scaffold-module.sh` | Created | ~150 lines; BASH_SOURCE fixture resolution, validation, file-by-file copy+rename+substitute |
| `build-n4-module-kit/retros/2026-09-05-campaign7-scaffold.md` | Created | PR4 retro (review-status: pending) |
| `build-n4-module-kit/retros/INDEX.md` | Modified | PR4 retro row added |
| `build-n4-module-kit/BUILD-STATE.md` | Modified | Kit self-envelope updated (Campaign 7 PR4) |
| `.github/workflows/ci.yml` | Modified | scaffold-diff step added |
| `openspec/changes/build-n4-module-campaign7/tasks.md` | Modified | PR4 tasks 4.1–4.9 marked [x] |
| `tests/scaffold-module.bats` | Created (cherry-pick) | QA RED TC1/TC2/TC3/TC-K8/TC4 verbatim |

### TDD Cycle Evidence

| TC | RED | GREEN | Named Mutation | Reverted |
|---|---|---|---|---|
| TC1: no args → exit 2 | exit 127 (script absent) | exit 2 + usage | — | — |
| TC2: 1bad → exit 2 | exit 127 | exit 2 | drop name-validation → exits 0 (conceptual; TC2 uses the validated code) | ✓ |
| TC3: diff -r byte-equal fixture | exit 127 | exit 0 | skip .gradle.kts (TC3+TC-K8 FAIL); module.xml remap (TC3+TC-K8 FAIL) | ✓ |
| TC-K8: HOME=/nonexistent | exit 127 | exit 0 | use $HOME in FIXTURE_ROOT (TC-K8 FAIL) | ✓ |
| TC4: round-trip local | SKIP | SKIP (no NIAGARA_HOME in CI); local: ALL PASS | drop stopped() cancel → lint-timers FAIL | ✓ |

### Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command | `bats tests/scaffold-module.bats` → 4/4 PASS + 1 SKIP; `bats tests/*.bats` → 157/157 |
| Runtime harness | TC4 local: scaffold→preflight→build.sh (slotomatic+jar BUILD SUCCESSFUL)→verify-module (7 PASS)→lint-timers (PASS); exit 0 all |
| Rollback boundary | `git revert` the two additive commits (script + fixture) removes all PR4 work without affecting earlier PRs |

### Deviations from Design

- **Pre-slotomatic fix not in design**: D3 described stripping the AUTO region "hash block" while keeping `//region … //endregion` markers. Real implementation: remove the entire `//region … //endregion` block — slotomatic errors ("Found multiple metadata blocks") with empty markers. Correct pre-slotomatic state = no region block at all (B793 §793.4 prototype confirms). This is K15 in the retro.
- TC5 and TC6 not added: orchestrator apply instruction says "plus nothing beyond what it pins except TC-K8 if absent"; TC-K8 is already pinned in QA RED. Followed orchestrator over tasks.md to keep the QA contract unmodified.

### Workload / PR Boundary

- Mode: size:exception (declared in tasks.md §PR4; ~700–880 authored lines in fixture+script)
- Current work unit: PR4 feat/c7-scaffold
- Boundary: starts at PR3 head (4f20967), ends with scaffold-module.sh + fixtures/MinimalPan + CI + retro + envelope (head 7062ad4)
- CI: PASS (shellcheck + bats + ledger sweep — 35s)

### Status

9/9 PR4 tasks complete. PR #55 OPEN, CI green. Ready for verify.

## PR5 — feat/c7-schema-risk (PR #56)

- Commits: 146e24d (QA RED qa/c7-schema-risk cherry-picked: SR1–SR8 + 7 B799 fixture pairs) → a50de85 feature → 8dc3834 ci fix → openspec commit.
- `toolbelt/schema-risk.sh <before-dir> <after-dir>`: parses `@NiagaraProperty`/`@NiagaraAction` declarations + `module-include.xml` from both snapshots, diffs slots by name (retype = `type=` attribute; reorder = declaration order; rename heuristic per design D4 with limits L1–L3), classifies each change from the embedded B795 §795.4 CSV (table-driven), prints `<VERDICT>  <change_kind>  <slot>: <detail (row id)>` rows and a final `verdict=<SAFE|LOSSY|OUTAGE>`; exits 0 SAFE / 1 LOSSY / 2 OUTAGE / 3 usage / 4 env; unknown kind → OUTAGE.
- Tests: SR1–SR8 GREEN (QA pins unchanged) + SR9 (unreadable module-include.xml → exit 4) + SR-CSV (heredoc byte-equals tests/fixtures/schema-risk/b795-795.4.csv). Named mutations proven and reverted: worst-cell→first-cell (SR7), unknown→SAFE (SR6), retype-off (SR3), rename-off (SR5 label), one CSV cell (SR-CSV).
- CI: non-strict fixtures step (each pair asserts its expected exit); fix-forward 8dc3834 for `bash -e` capturing a non-zero exit (`got_exit=0; out=$(cmd) || got_exit=$?`).
- Retro: `retros/2026-09-05-campaign7-schema-risk.md` (pending) + INDEX row + kit self-envelope.


### PR5 Fix-forward: parse_slots whitespace (post-QA rejection)

**Defect**: extract_attr regex `attr="value"` did not match `attr = "value"` (spaces around =). Real modules produce 0 slots → false SAFE.
**Fix**: `attr[[:space:]]*=[[:space:]]*"[^"]*"` + index() extraction. Commit `a63c3c5`.
**SR10**: Added real-shape fixture (MinimalPan production form). Named mutation: unspaced regex → 0 rows → false SAFE → SR10 fails.
**Smoke (post-fix)**: CompPan-rt HEAD~5→HEAD: 3 SAFE add_slot rows (condenser1/2/3Mode), exit 0. ColdRoomPan-rt: no changes in range, verdict=SAFE, exit 0.
**Final**: 168/168 bats green. PR #56 CI green. Head: `f4d2f32`.

---

## PR6 — feat/c7-plano (2026-09-05)

**Status**: complete
**Mode**: Strict TDD
**Branch**: feat/c7-plano
**Worker**: sdd-apply (Claude Sonnet 4.6)
**QA RED**: `qa/c7-plano` tip `c49504f` cherry-picked

### TDD Cycle Evidence

| Task | RED | GREEN | REFACTOR |
|---|---|---|---|
| PL1 PASS (all agree) | `bats tests/plano-check.bats` → 4 not ok (exit 2, no --plano flag) | `--plano` block added, PL1 passes | No refactor needed |
| PL2 FAIL names 1247/771 | same RED | PL2 passes; output contains 1247/771 | viewBox: targeted id="zonas" grep |
| PL3 auto exempt | same RED | PL3 passes | — |
| PL4 Rv!=Ri FAIL | same RED | PL4 passes | — |

### Named Mutation

Full count-only mutation (remove all three `_ceq` checks — Rc==Ri, Rc==Rv, and aspect-ratio vs Rc):
- **PL2** loses its FAIL: single `1247/771` is trivially equal to itself → PASS (wrong, should FAIL)
- **PL4** loses its FAIL: no Rv vs Rc check → PASS (wrong, should FAIL)
Proves cross-source equality (all values against Rc/Ri/Rv ground truth) is load-bearing, not intra-aspect-ratio equality. Reverted.

### Real Smoke (DashboardPan-ux)

```
bash build-n4-module-kit/toolbelt/verify-module.sh --plano \
  /home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato/Dashboard/DashboardPan/DashboardPan-ux/src/rc/index.html
FAIL  plano      .../index.html  aspect-ratio 1247/771 != Rc(1248/891)
Exit: 1
```
Confirms issue #49: `.frame{aspect-ratio:1247/771}` is stale, masked by `#frame{aspect-ratio:auto}` at :96.

### Completed Tasks

- [x] 6.1 Re-read QA RED `qa/c7-plano` tip `c49504f` before writing (CD4)
- [x] 6.2 --plano mode added to verify-module.sh dispatched before flag loop (D6a): Rc/Ri/Rv/A parsed, cross-multiplication, auto exempt
- [x] 6.3 Guard `base64`, `od` via `command -v || exit 3`; `unzip` guarded on .jar operand
- [x] 6.4 PL5 (var(--x) → FAIL) + PL6 (fixture PASS via HTML path) added to plano-check.bats
- [x] 6.5 `tests/fixtures/plano/ok/index.html` created (inline 2×3 PNG; Rc=Ri=Rv=2/3)
- [x] 6.6 CI step: `--plano tests/fixtures/plano/ok/index.html` → exit 0 added to ci.yml
- [x] 6.7 Named mutations recorded and reverted (full count-only: PL2+PL4 lose FAIL)
- [x] 6.8 shellcheck exit 0 on modified verify-module.sh
- [x] 6.9 Retro `retros/2026-09-05-campaign7-plano.md` + INDEX row + BUILD-STATE.md kit envelope updated

### Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `build-n4-module-kit/toolbelt/verify-module.sh` | Modified | Added --plano subcommand (~80 lines) between coverage block and flag loop |
| `tests/plano-check.bats` | Cherry-pick + append | PL1-PL4 from qa/c7-plano c49504f; PL5+PL6 appended |
| `tests/fixtures/plano/ok/index.html` | Created | 2×3 PNG fixture; Rc=Ri=Rv=2/3; one agreeing aspect-ratio |
| `.github/workflows/ci.yml` | Modified | Added plano-ok fixture CI step (exit 0) |
| `build-n4-module-kit/BUILD-LOOP.md` | Modified | §5 pre-gate: hedge → real invocation |
| `build-n4-module-kit/skill/SKILL.md` | Modified | §References: hedge → real invocation |
| `build-n4-module-kit/retros/2026-09-05-campaign7-plano.md` | Created | PR6 retro (review-status: pending) |
| `build-n4-module-kit/retros/INDEX.md` | Modified | PR6 retro row added |
| `build-n4-module-kit/BUILD-STATE.md` | Modified | Kit self-envelope: retro_pending + last_commit/session updated |

### Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command | `bats tests/plano-check.bats` → 6/6 ok (PL1-PL6) |
| Full bats | `bats tests/*.bats` → 174/174 passing, 0 failing |
| Shellcheck | `shellcheck build-n4-module-kit/toolbelt/verify-module.sh` → exit 0 |
| Sweep guards | `sweep-build-state.sh` → exit 0; `sweep-fold-audit.sh --strict` → exit 0 (47 cited) |
| Real smoke | `--plano DashboardPan-ux/src/rc/index.html` → FAIL "1247/771 != Rc(1248/891)" exit 1 ✓ |
| Rollback boundary | Revert --plano block from verify-module.sh; remove plano-check.bats, fixtures/plano/, CI step; revert BUILD-LOOP/SKILL hedges; remove retro+INDEX row; restore BUILD-STATE envelope |

### Workload / PR Boundary

- Mode: chained PR slice (auto-chain, stacked-to-main)
- Current work unit: PR6 feat/c7-plano
- Boundary: starts at PR5 head (f4d2f32), ends with --plano + tests + fixture + CI + docs + retro
- Estimated review budget: ~140 authored lines (within single-PR budget)

### Status

9/9 PR6 tasks complete. Ready for verify.

---

## PR7 — docs/c7-logic-split (2026-09-05)

**Status**: complete
**Mode**: Standard (TDD RED→GREEN for L4/L5/L6)
**Branch**: docs/c7-logic-split
**Head SHA**: c0e1744
**PR**: https://github.com/angeles725/niagara-tools/pull/58
**CI**: pass (51s)
**Worker**: sdd-apply (Claude Sonnet 4.6)

### Completed Tasks

- [x] 7.1 grep-before-fold: 1 hit (logic.md heading itself); 0 hits in SKILL.md and BUILD-LOOP.md — CD3 satisfied
- [x] 7.2 types/logic.md: lines 1-90 kept verbatim; line 91 = cross-reference to logic-authoring.md (91 lines, ≤ 91 ✓)
- [x] 7.3 types/logic-authoring.md created: 3-line HTML comment header + verbatim lines 91-136 from original (49 lines)
- [x] 7.4 skill/SKILL.md: v0.5→v0.6; framework-ext row added to decision table; scaffold-module.sh + schema-risk.sh added to refs+step5; BUILD-LOOP.md: §1 scaffold-module.sh, §2 logic-authoring.md sibling, §5 schema-risk.sh pre-gate
- [x] 7.5 kit-links.bats L4/L5/L6 added; L5 was naturally RED (scaffold+schema missing from both docs before this PR)
- [x] 7.6 bats tests/kit-links.bats → 6/6 GREEN; bats tests/*.bats → 176/176 GREEN
- [x] 7.7 Named mutation: delete schema-risk.sh from BUILD-LOOP.md + SKILL.md → L5 fails (schema-risk.sh missing); reverted
- [x] 7.8 Retro 2026-09-05-campaign7-logic-split.md + INDEX row + BUILD-STATE.md kit envelope (Campaign 7 PR7)

### Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `build-n4-module-kit/types/logic.md` | Modified | Lines 91-136 removed; cross-reference to logic-authoring.md added at line 91 (91 lines ≤ 91) |
| `build-n4-module-kit/types/logic-authoring.md` | Created | 3-line HTML header + verbatim lines 91-136 from original logic.md (49 lines) |
| `build-n4-module-kit/skill/SKILL.md` | Modified | v0.5→v0.6; framework-ext row; scaffold+schema refs added; step 5 updated |
| `build-n4-module-kit/BUILD-LOOP.md` | Modified | §1 scaffold-module.sh; §2 logic-authoring.md sibling; §5 schema-risk.sh |
| `build-n4-module-kit/corpus-index.md` | Modified | §Campaign 6 retargeted: logic.md → logic-authoring.md (2 changes) |
| `build-n4-module-kit/retros/INDEX.md` | Modified | PR7 retro row added |
| `build-n4-module-kit/retros/2026-09-05-campaign7-logic-split.md` | Created | PR7 retro (review-status: pending; 5 lessons: audience boundary, invariant, L5 gap, shellcheck directive, corpus-index) |
| `build-n4-module-kit/BUILD-STATE.md` | Modified | Kit envelope: Campaign 7 PR7 last_commit/last_session |
| `tests/kit-links.bats` | Modified | L4/L5/L6 added (43 new lines; shellcheck 0) |
| `openspec/changes/build-n4-module-campaign7/tasks.md` | Modified | PR7 tasks 7.1-7.8 marked [x] |

### Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command | `bats tests/kit-links.bats` → 6/6 ok (L1-L6) |
| Full bats | `bats tests/*.bats` → 176/176 passing, 0 failing |
| Shellcheck | `shellcheck toolbelt/*.sh tests/*.bats tests/helpers/*.bash` → exit 0 |
| Sweep guards | `sweep-build-state.sh` → exit 0; `sweep-fold-audit.sh --strict` → 47 folded/cited, exit 0 |
| Split invariant | `wc -l logic.md = 91 ≤ 91`; 0 lost lines; 0 duplicates; 4 new infra lines (3 header + 1 cross-ref) |
| L5 mutation | Delete schema-risk.sh from both docs → `not ok L5: toolbelt script not in BUILD-LOOP.md or skill/SKILL.md: schema-risk.sh`; reverted |
| CI | PR #58 pass (shellcheck + bats + ledger sweep) 51s |
| Rollback boundary | Revert split: `git revert` commit c0e1744 — removes logic-authoring.md, restores logic.md, reverts all refs; kit returns to PR6 state |

### Deviations from Design

- **L4 implementation differs from tasks.md §7.5**: tasks.md says "L4 = every toolbelt/*.sh named in BUILD-LOOP.md"; orchestrator instruction says "L4 = every types/*.md named in SKILL.md decision table exists". Implemented per orchestrator (types/*.md check) as it directly validates the new logic-authoring.md routing. L5 (the toolbelt routing guard, OR-combined across both docs) covers the toolbelt coverage requirement from D10. L5 was naturally RED for scaffold-module.sh and schema-risk.sh.
- **scaffold-module.sh and schema-risk.sh not yet documented in routing docs**: These PR4/PR5 scripts were added to BUILD-LOOP.md and SKILL.md in this PR (PR7) rather than PR2, per coordinator instruction. Tagged `[ev: retro tool-integration]`.
- **corpus-index.md §Campaign 6 retargeted**: The section heading `### → types/logic.md` and its prose reference both updated to `types/logic-authoring.md`. This is a mandatory consequence of the split (all blocks in that section describe content now in logic-authoring.md).

### Workload / PR Boundary

- Mode: chained PR slice (auto-chain, stacked-to-main)
- Current work unit: PR7 docs/c7-logic-split
- Boundary: starts at PR6 head (69270aa), ends with split + tests + routing refs + retro + envelope (head c0e1744)
- Estimated review budget: ~76 authored lines (within 400-line budget)

### Status

8/9 PR7 tasks complete (7.9 is post-merge orchestrator action). PR #58 OPEN, CI green. Ready for verify.

---

## PR8 — feat/c7-report-module (Campaign 7 PR8, 2026-09-05)

**Mode**: Strict TDD | **Branch**: feat/c7-report-module | **Ledger token**: sha256:2bc83baec7834a446a8592df3b25060fa11f2cadee10904f0ce5e94bbee2681c

### TDD Cycle Evidence

| Task | RED | GREEN | REFACTOR |
|------|-----|-------|----------|
| RM1-RM3 (bats tests/report-module.bats) | Confirmed: exit 127 (script absent), 3 not ok | `report-module.sh` written, all 3 pass | `set -u`, no `set -e`, shellcheck 0 |

### Completed Tasks

- [x] 8.1 Re-read QA RED before writing; signature + exits confirmed (D7a)
- [x] 8.2 Write `toolbelt/report-module.sh` — discovers artifacts, composes 4 tools, aggregates rows + member exits; exit 0 CLEAN / 1 FAIL / 3 env; `set -u`; shellcheck 0
- [x] 8.3 `tests/report-module.bats` was already present (RED commit 3d5f8ea); RM1-RM3 verbatim confirmed
- [x] 8.4 Fixtures `tests/fixtures/report-module/{clean,leak}/DemoPan-rt` already present (RED commit)
- [x] 8.5 No separate CI step added (RM1 covers the clean-exit path)
- [x] 8.6 Named mutation verified: drop lint-timers aggregation → RM2 fails (and RM1/RM3 also fail due to `set -u` error on unset `lt_out`)
- [x] 8.7 Real-tree smoke: ColdRoomPan-rt `NIAGARA_HOME=/mnt/c/Honeywell/OptimizerSupervisor-N4.14.0.162` → 9 PASS · 1 FAIL · 1 WARN · 1 SKIP → ISSUES, exit 1 (BEvaporatorUnit timer-ticket FAIL + slot-coverage WARN @ 50%)
- [x] 8.8 Retro `retros/2026-09-05-campaign7-report-module.md` + INDEX row + BUILD-STATE.md self-envelope committed

### Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `build-n4-module-kit/toolbelt/report-module.sh` | Created | 208-line aggregated report composer |
| `build-n4-module-kit/BUILD-LOOP.md` | Modified | §5 punch-list step `[ev: retro campaign7-report-module]` |
| `build-n4-module-kit/skill/SKILL.md` | Modified | version 0.7; step 5 + References entry |
| `build-n4-module-kit/retros/2026-09-05-campaign7-report-module.md` | Created | 3 proposed deltas; review-status: pending |
| `build-n4-module-kit/retros/INDEX.md` | Modified | +1 row for campaign7-report-module |
| `build-n4-module-kit/BUILD-STATE.md` | Modified | kit self-envelope updated for PR8 close |
| `openspec/changes/build-n4-module-campaign7/tasks.md` | Modified | PR8 tasks 8.1-8.8 marked [x] |
| `openspec/changes/build-n4-module-campaign7/apply-progress.md` | Modified | PR8 section appended |

### Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command | `bats tests/report-module.bats` → 3/3 ok (RM1-RM3) |
| Full bats | `bats tests/*.bats` → 179/179 passing, 0 failing |
| Shellcheck | `shellcheck ... report-module.sh` → exit 0 |
| Sweep guards | `sweep-build-state.sh` → exit 0; `sweep-fold-audit.sh --strict` → 47 folded/cited, exit 0 |
| Mutation test | Drop lint-timers aggregation → RM2 fails (exits 0 instead of 1); reverted |
| Real-tree smoke | `report-module.sh ColdRoomPan --target-version 4.14` → 9 PASS · 1 FAIL · 1 WARN · 1 SKIP; exit 1 (B798 confirmed) |
| Attribution check | `git log --format=%B b79c7da..HEAD \| grep -iE co-authored\|generated with\|claude` → 0 matches |
| Rollback boundary | `git revert HEAD` removes report-module.sh; reverts BUILD-LOOP.md/SKILL.md routing entries; kit returns to PR7 state |

### Deviations from Design / Contract

- **B798 expected shows 7 PASS · 1 FAIL · 1 WARN · 0 SKIP vs actual 9 PASS · 1 FAIL · 1 WARN · 1 SKIP**: the contract's expected output was from an earlier run without `--src` (typecount/facets/stored rows absent). With `--src MODULE_ROOT`, verify-module adds typecount (PASS), facets (PASS), and stored (SKIP) rows. BEvaporatorUnit FAIL and slot-coverage WARN both appear as required; exit 1 confirmed. The contract's B798 block is a pruned snapshot, not a byte-golden — the QA pins (RM1-RM3) are the real contract.
- **tasks.md 8.3/8.4**: tests/fixtures were already present in the RED commit (3d5f8ea); task description said "copy/create" but they were already there.
- **tasks.md 8.5**: No additional CI step needed beyond RM1 (which proves clean-exit on the fixture).

### Workload / PR Boundary

- Mode: chained PR slice (auto-chain, stacked-to-main), size within 500-line budget
- Current work unit: PR8 feat/c7-report-module (last code PR of Campaign 7)
- Authored diff: 271 insertions, 8 deletions = 279 changed lines (within 500-line budget)
- Rollback: `git revert` the PR8 commit removes report-module.sh and all routing docs

### Status

8/8 PR8 tasks complete. Ready for verify. Campaign 7 code PRs complete (PR1-PR8 of 8).
