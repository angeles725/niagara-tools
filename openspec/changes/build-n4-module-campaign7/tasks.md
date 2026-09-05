# Tasks: build-n4-module-campaign7

**Source**: v0.17.0 (c136e3b) → **Target**: v0.18.0 | **Chain**: stacked-to-main | 8 PRs + close

---

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1 360–1 530 total (excl. fixture binaries) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |
| PR4 declared | size:exception (~700–880 authored) |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | PR | Goal | Est. lines | Focused test | Rollback boundary |
|------|----|------|------------|-------------|-------------------|
| 1 | PR1 | Fold 9 retros → METHODOLOGY/BUILD-LOOP/CONTRIBUTING | ~40 | `sweep-fold-audit.sh --strict` exits 0 | Revert doc edits; flip 9 INDEX rows back to `pending` |
| 2 | PR2 | Tool routing in BUILD-LOOP + SKILL.md | ~20 | `grep -c 'preflight' BUILD-LOOP.md` >= 1 | Revert doc deltas; re-run `install-skill.sh` |
| 3 | PR3 | Dashboard B796 exemplar in types/dashboard.md | ~25 | `kit-links.bats` exits 0 | Revert types/dashboard.md |
| 4 | PR4 | scaffold-module.sh + fixtures/MinimalPan (size:exception) | ~700–880 | `bats tests/scaffold-module.bats` | `git revert` additive paths; fixture tree removable |
| 5 | PR5 | schema-risk.sh + B799 fixtures | ~200 | `bats tests/schema-risk.bats` | `git revert` additive paths |
| 6 | PR6 | verify-module.sh --plano | ~140 | `bats tests/plano-check.bats` | Revert --plano addition from verify-module.sh |
| 7 | PR7 | types/logic-authoring.md split + kit-links L4-L6 | ~70 | `bats tests/kit-links.bats` | Revert split; recombine logic.md |
| 8 | PR8 | report-module.sh (stretch) | ~50 | `bats tests/report-module.bats` | `git revert` additive paths |
| 9 | CLOSE | VERSION 0.18.0 + CHANGELOG | ~15 | SC8 assertions | Revert version bump |

---

## PR1 — docs/c7-retro-fold (~40 lines, doc-only)

**Branch**: `feat/c7-fold-retros` | **QA hook**: none (doc-only) | **Gate exit**: R1.1–R1.8

- [x] 1.1 `rg` grep-before-fold terms (`K11`, `K12`, `K13`, `K14`, `envelope-pairing`, `SDD-ledger`, `two-independent-reads`) in METHODOLOGY.md, BUILD-LOOP.md, CONTRIBUTING.md — confirm 0 hits (CD3, no double-fold).
- [x] 1.2 Add K11–K14 to `build-n4-module-kit/METHODOLOGY.md` §Kit-maintenance after K10 (R1.4).
- [x] 1.3 Add envelope-pairing rule (close-exit a/b/c pairs retro anchor + BUILD-STATE self-envelope in same push range) to `BUILD-LOOP.md` §7 (R1.5).
- [x] 1.4 Add SDD-ledger rule + two-independent-reads rule to `CONTRIBUTING.md` (R1.6).
- [x] 1.5 Flip 9 rows in `retros/INDEX.md` from `| pending |` to folded state with `[ev: retro <slug>]` citation (R1.1–R1.2). Evidence grep: `grep -c '| pending |' retros/INDEX.md` == 0 (CD5 predicate).
- [x] 1.6 Replace `pending` marker with `<!-- review-status: folded -->` in each of the 9 retro files `build-n4-module-kit/retros/2026-09-05-campaign6-*.md` (R1.3).
- [x] 1.7 Commit includes retro file + INDEX row + `BUILD-STATE.md` self-envelope (CD1).
- [x] 1.8 Gate: `sweep-fold-audit.sh --strict` exits 0 (R1.8); `sweep-build-state.sh` exits 0 (R1.7). Two-reads: investigador1 fidelity + QA verbatim spot-checks of each K-rule.

---

## PR2 — docs/c7-tool-integration (~20 lines, doc-only)

**Branch**: `feat/c7-tool-integration` | **QA hook**: none (doc-only) | **Gate exit**: R2.1–R2.7

- [x] 2.1 grep-before-fold: `rg 'preflight|lint-timers|slot-coverage|sweep-build-state' build-n4-module-kit/BUILD-LOOP.md build-n4-module-kit/skill/SKILL.md` — record hit counts before edit.
- [x] 2.2 Add §0.b preflight step (`toolbelt/preflight.sh`), §5 pre-gate listing (`lint-timers.sh` + `slot-coverage.sh` before `verify-module.sh`), §7 `--age` orient/close calls to `BUILD-LOOP.md` (R2.1–R2.3).
- [x] 2.3 Add all 10 toolbelt scripts to `build-n4-module-kit/skill/SKILL.md` §References; route step 5 to pre-gate checks (R2.4–R2.5).
- [x] 2.4 Commit includes retro + INDEX row + `BUILD-STATE.md` self-envelope (CD1).
- [x] 2.5 Gate: `grep -c 'preflight' BUILD-LOOP.md` >= 1 and same for `lint-timers`, `slot-coverage`, `sweep-build-state` in both files (R2.6). Paste grep evidence in PR body.
- [ ] 2.6 After merge: orchestrator re-runs `scripts/install-skill.sh` (R2.7, CD6).

---

## PR3 — docs/c7-dashboard-b796 (~25 lines, doc-only)

**Branch**: `feat/c7-dashboard-fold` | **QA hook**: `tests/kit-links.bats` | **Gate exit**: R3.1–R3.4

- [x] 3.1 grep-before-fold: `rg 'DashboardPan-ux' build-n4-module-kit/types/dashboard.md` — confirm 0 hits (CD3).
- [x] 3.2 Fold B796 §796.4 exemplar (4/5 gates documented) into `types/dashboard.md`; cite gate 4 as REQUIRED-but-absent with reference to issue #49 (R3.1–R3.2).
- [x] 3.3 Two-reads fidelity check vs B796 §796.4 source: author read done; second read by investigador1 pending (R3.3).
- [x] 3.4 Run `tests/kit-links.bats`; assert exit 0 (R3.4).
- [x] 3.5 Commit includes retro + INDEX row + `BUILD-STATE.md` self-envelope (CD1).

---

## PR4 — feat/c7-scaffold [size:exception] (~700–880 lines)

**Branch**: `feat/c7-scaffold` | **QA hook**: `qa/c7-scaffold` tip `54636ca` | **Gate exit**: R4.1–R4.9
**Two commits**: commit 1 = fixtures only; commit 2 = scaffold-module.sh; commit 3 = tests

- [x] 4.1 Re-read QA RED `qa/c7-scaffold` tip `54636ca` before writing (CD4).
- [x] 4.2 **Commit 1**: create `fixtures/MinimalPan/**` — pre-slotomatic tree, no AUTO region, no operator paths in `gradle.properties`, English lexicon in `module.lexicon`; mark binary files (gradlew*, gradle-wrapper.jar) as copy-only (D3). Confirm with companero that lexicon normalization bytes are aligned.
- [x] 4.3 **Commit 2**: write `toolbelt/scaffold-module.sh`: skeleton via `"${BASH_SOURCE[0]%/*}/../fixtures/MinimalPan"` (never `$HOME`, K8); emitted root = `<out-dir>/<ModuleName>` (D2); apply D3 substitution table (MinimalPan→ModuleName, angeles→vendor, 7.6.17→plugin-version, 4.14→target-version, etc.); `set -u`; typed exits 0/2/3; no VCS; `shellcheck` clean (R4.9).
- [x] 4.4 **Commit 3**: copy `tests/scaffold-module.bats` from `qa/c7-scaffold` tip (TC1/TC2/TC3/TC-K8 verbatim) + add TC5 (residue guard: ScaffoldPan --vendor Acme leaves zero `MinimalPan|angeles` occurrences) + TC6 (existing `<out>/<Mod>` → exit 3, destination untouched).
- [x] 4.5 Add CI step: fixture diff loop over `fixtures/MinimalPan` (scaffold → diff -r, assert exit 0); TC4 guarded by `[ -n "${NIAGARA_HOME:-}" ] || skip` (D9, R4.7).
- [x] 4.6 Record named mutations in PR body: TC2 (drop name-validation → exits 0); TC-K8 (reintroduce `$HOME` → exits 3); TC3 (drop MOD-rt.gradle.kts or swap module-include.xml → diff fails); TC5 (drop vendor substitution → residue found); TC4 (drop `stopped()` cancel → `lint-timers.sh` FAIL) (R4.6).
- [x] 4.7 TC4 run locally before bless: exact command + output pasted in PR body (R4.7).
- [x] 4.8 `shellcheck` exits 0 on `toolbelt/scaffold-module.sh` (R4.9).
- [x] 4.9 Commit includes retro + INDEX row + `BUILD-STATE.md` self-envelope (CD1).

---

## PR5 — feat/c7-schema-risk (~200 lines)

**Branch**: `feat/c7-schema-risk` | **QA hook**: `qa/c7-schema-risk` tip `6d27ff0` | **Gate exit**: R5.1–R5.9

- [x] 5.1 Re-read QA RED `qa/c7-schema-risk` tip `6d27ff0` before writing (CD4).
- [x] 5.2 Write `toolbelt/schema-risk.sh`: embed B795 §795.4 CSV verbatim as `CSV_TABLE=$(cat <<'CSV'…CSV)` (D5); awk lookup on `kind`; fail-safe UNKNOWN row; worst-cell verdict (OUTAGE > LOSSY > SAFE); output format `verdict=<V>` (D4a); row shape `<verdict>  <change_kind>  <Type>.<slot>: <detail>`; exits 0 SAFE/1 LOSSY/2 OUTAGE/3 usage/4 env; `set -u`; no VCS; `shellcheck` clean.
- [x] 5.3 Copy `tests/schema-risk.bats` from `qa/c7-schema-risk` tip (SR1–SR8 verbatim) + add SR9 (unreadable `module-include.xml` → exit 4) + SR-CSV (embedded heredoc byte-equals `tests/fixtures/schema-risk/b795-795.4.csv`).
- [x] 5.4 Copy B799 fixture pairs from `qa/c7-schema-risk` into `tests/fixtures/schema-risk/` (7 before/after/expected.txt pairs + `b795-795.4.csv` oracle).
- [x] 5.5 Add CI step: loop 7 fixture pairs; assert mapped exit (0/1/2) and `verdict=<V>` token matches `expected.txt` (D4a — `expected.txt` is oracle for token only, not byte-golden).
- [x] 5.6 Named mutations in PR body: worst-cell → first-cell (mixed reads SAFE not OUTAGE); drop UNKNOWN fail-safe; map usage exit to 2 (SR8 reads as OUTAGE); edit one CSV cell (SR-CSV fails).
- [x] 5.7 `shellcheck` exits 0 on `toolbelt/schema-risk.sh` (R5.8).
- [x] 5.8 Commit includes retro + INDEX row + `BUILD-STATE.md` self-envelope (CD1).

---

## PR6 — feat/c7-plano (~140 lines)

**Branch**: `feat/c7-plano` | **QA hook**: `qa/c7-plano` tip `c49504f` | **Gate exit**: R6.1–R6.6

- [x] 6.1 Re-read QA RED `qa/c7-plano` tip `c49504f` before writing (CD4).
- [x] 6.2 Extend `toolbelt/verify-module.sh` with `--plano <index.html|jar>` mode dispatched before the flag loop (D6a): extract Rc = `IMG_W/IMG_H` from `const` declarations, Ri = intrinsic PNG size via `base64 -d | od -An -tu1 -N24` (verify 8-byte PNG sig + IHDR; bytes 17-20 BE width, 21-24 BE height), Rv = first `viewBox` non-zero dimensions; integer cross-multiplication only; `auto` exempt; unparseable value → FAIL; `.jar` operand → `unzip -p rc/**/index.html` to temp file (R6.1–R6.4).
- [x] 6.3 Guard `base64`, `od`, `unzip` via `command -v || exit 3`.
- [x] 6.4 Copy `tests/plano-check.bats` from `qa/c7-plano` tip (PL1–PL4 verbatim) + add PL5 (`aspect-ratio: var(--x)` → FAIL, not exempt) + PL6 (`.jar` operand yields same verdict as HTML form).
- [x] 6.5 Create `tests/fixtures/plano/ok/index.html` (inline 2×3 base64 PNG fixture; Rc=Ri=Rv=2/3).
- [x] 6.6 Add CI step: `verify-module.sh --plano tests/fixtures/plano/ok/index.html` → exit 0.
- [x] 6.7 Named mutations: count-only check → PL2+PL4 lose FAIL; treat unparseable as exempt → PL5 passes (R6.5).
- [x] 6.8 `shellcheck` exits 0 on modified `verify-module.sh` (R6.6).
- [x] 6.9 Commit includes retro + INDEX row + `BUILD-STATE.md` self-envelope (CD1).

---

## PR7 — docs/c7-logic-split (~70 lines)

**Branch**: `feat/c7-logic-split` | **QA hook**: `tests/kit-links.bats` | **Gate exit**: R7.1–R7.8

- [x] 7.1 grep-before-fold: `rg 'Author-side SPIs|logic-authoring' build-n4-module-kit/types/logic.md build-n4-module-kit/skill/SKILL.md build-n4-module-kit/BUILD-LOOP.md` — confirm 0 hits (CD3).
- [x] 7.2 Remove lines 91–136 (`## Author-side SPIs` through EOF) from `types/logic.md`; add cross-reference to `types/logic-authoring.md` at the split boundary (R7.1, D8).
- [x] 7.3 Create `types/logic-authoring.md` with removed lines 91–136; add cross-reference back to `types/logic.md` (R7.2, D8).
- [x] 7.4 Update `skill/SKILL.md` decision table to reference both files; update `BUILD-LOOP.md` §2 references — in the same commit range as file creation (R7.3–R7.4).
- [x] 7.5 Extend `tests/kit-links.bats` with L4 (every `toolbelt/*.sh` named in BUILD-LOOP.md), L5 (same for `skill/SKILL.md`; skip when launcher not installed, mirroring L3), L6 (both `types/logic.md` and `types/logic-authoring.md` exist and cite each other) (D10).
- [x] 7.6 Run `tests/kit-links.bats`; assert exit 0 (R7.6).
- [x] 7.7 Named mutation in PR body: break one moved link (e.g. `types/logic-authoring.md` → wrong path) → `kit-links.bats` FAIL (R7.7). Two-reads: investigador1 confirms line 91 boundary is correct and no content is lost.
- [x] 7.8 Commit includes retro + INDEX row + `BUILD-STATE.md` self-envelope (CD1).
- [ ] 7.9 After merge: orchestrator re-runs `scripts/install-skill.sh` (R7.8, CD6).

---

## PR8 — feat/c7-report-module [stretch] (~50 lines)

**Branch**: `feat/c7-report` | **QA hook**: `qa/c7-report-module` tip `412ee8e` | **Gate exit**: R8.1–R8.6

- [x] 8.1 Re-read QA RED `qa/c7-report-module` tip `412ee8e` before writing (CD4); confirm `<module-root> [--target-version x.y]` signature and exits 0/1/3 (D7a).
- [x] 8.2 Write `toolbelt/report-module.sh <module-root> [--target-version x.y]`: discover profile artifacts `<MOD>-{rt,ux,wb,…}`; per artifact invoke `verify-module.sh --src` (SKIP no jar), `slot-coverage.sh`, `lint-timers.sh <artifact>/src`, `--plano` when `<artifact>/src/rc/index.html` exists; aggregate rows + member exits (D7a severity map); exit 0 CLEAN / 1 FAIL / 3 env; `set -u`; `shellcheck` clean (R8.6).
- [x] 8.3 Copy `tests/report-module.bats` from `qa/c7-report-module` tip (RM1–RM3 verbatim) + add: slot-coverage WARN stays WARN (not FAIL) mutation; member-exiting-3 emits ERROR row + report exits 3.
- [x] 8.4 Create `tests/fixtures/report/` clean rt-only fixture + leak fixture (`BLeak.java` with timer-cancel omitted).
- [x] 8.5 Add CI step: `report-module.sh tests/fixtures/report/<mod>` → exit 0.
- [x] 8.6 Named mutations in PR body: drop lint-timers aggregation → RM2 exits 0; promote WARN → FAIL flip; swallow member env faults (R8.5).
- [x] 8.7 Local bless: run ColdRoomPan-rt B798 report (7 PASS · 1 FAIL · 1 WARN → ISSUES, exit 1); paste output in PR body. Not a CI pin.
- [x] 8.8 Commit includes retro + INDEX row + `BUILD-STATE.md` self-envelope (CD1).

---

## CLOSE — VERSION 0.18.0 + CHANGELOG (~15 lines)

**Gate exit**: SC8 | doc-only, no new bats

- [ ] C.1 Rename `## [Unreleased]` → `## [v0.18.0] - YYYY-MM-DD` in `CHANGELOG.md`; add `### References` block (SDD slug + engram IDs) (SC8).
- [ ] C.2 Set `VERSION` = `0.18.0`.
- [ ] C.3 Close campaign-7 retro: mark final state in `retros/INDEX.md`; `sweep-fold-audit.sh --strict` exits 0 (SC1).
- [ ] C.4 Final SC sweep: `grep -c '| pending |' retros/INDEX.md` == 0; bats total >= 175 passing; `shellcheck` exit 0 across all new scripts; SC9 (no attribution trailers in PR bodies).

---

## Deferred / Out of Scope

| Item | Reason |
|------|---------|
| Issue #49 — gate 4 REQUIRED-but-absent (DashboardPan-ux) | Client punch-list; PR3 cites it as known gap; upstream resolution not in this campaign |
| Issue #50 | Not in campaign-7 spec; deferred to campaign 8 |
| Dependency floor matrix | Deferred to campaign 8 |
| `retype_complex` / `remove_slot_complex` reachable (L2 escape hatch) | Documented limit; operator-supplied type list deferred to campaign 8 |
| Module-repo fixes (operator gradle.properties live paths) | External repo; out of niagara-tools scope |
| RP2/RP3 extras if PR8 cut | PR8 is stretch; if cut, report-module.sh ships campaign 8 |
| D10 PR2 kit-links relaxation | Resolved in PR7 (L4/L5 guards); PR2 uses PR-body grep — accepted tradeoff |
