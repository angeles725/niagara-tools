# Tasks: build-n4-module-campaign10

**Source**: kit v0.20.0 (main `f90b8d1`, C9 archive `df8c7ec`) → **Target**: v0.21.0
**Chain**: stacked-to-main | 7 PRs (PR1-PR4 parallel kit; PR5 kit; PR6 client after PR5; PR7 close after PR1-PR5)

---

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~735 total (PR1 ~180, PR2 ~200, PR3 ~220, PR4 ~40, PR5 ~100, PR6 ~20, PR7 ~180) |
| 400-line budget risk | Low |
| Chained PRs recommended | Yes |
| Suggested split | PR1 → PR2/PR3/PR4 (parallel) → PR5 → PR6 → PR7 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Low

### Suggested Work Units

| Unit | PR | Goal | Est. lines | Focused test | Runtime harness | Rollback |
|------|----|------|-----------|-------------|-----------------|---------|
| 1 | PR1 | S21 lint-timers companion-flag: class-FIELD + same-method-body scope | ~180 | `bats tests/lint-timers.bats` | ColdRoomPan-rt @ ff1b659 exit 0; anyNoHardware absent | `git revert`; issue #89 re-opens |
| 2 | PR2 | S22 lint-ext-writable-shape per-slot action-body exemption | ~200 | `bats tests/ext-writable-shape.bats` | CompPan-rt exactly 1 WARN faultReset by NAME | `git revert` + RED EW10 re-pin revert (contract) |
| 3 | PR3 | S23 lint-silent-protection Pattern B surface | ~220 | `bats tests/lint-silent-protection.bats` | CompPan-rt 0 with :294 absent; ColdRoomPan-rt 0 | `git revert` |
| 4 | PR4 | S24 cwd-independent structural REDs | ~40 | Structural suite from 3 cwds | Same verdict from kit root / profile dir / /tmp | `git revert` |
| 5 | PR5 | S25 lint-write-path STALE advisory + --strict | ~100 | `bats tests/lint-write-path.bats` | DashboardPan matrix ff1b659: 5 STALE rows exit 0; --strict exit 1 | `git revert` |
| 6 | PR6 | S26 client .gitignore + 5 [concept] matrix marks | ~20 | `git ls-files` jar count before==after | STALE 5→0 flip with PR5 rule | `git revert` 2 .gitignore lines + 5 matrix tokens |
| 7 | PR7 | C10 close: retros, VERSION 0.21.0, CHANGELOG, sweeps | ~180 | `C10_CLOSE=1 bats tests/c10-close.bats` | N/A — doc/version only | `git revert`; retros return to `pending` |

---

## PR1 — feat/c10-lint-timers-scope (~180)

**RED**: `qa/c10-lint-timers-fp` tip **`52ebd11`** (S21-neg / S21-pos / S21-smoke; base `df8c7ec`) — re-read at apply (K13)
**Repo**: `niagara-tools` · **Worktree**: `niagara-tools-worktrees/c10-timers` (branch off kit main `f90b8d1`)
**D-ids**: D1, D1a-e, D6 · **Gate**: SC-1, SC-7, SC-8, SC-9, SC-10

- [ ] 1.1 Re-read `git show origin/qa/c10-lint-timers-fp:build-n4-module-kit/tests/lint-timers.bats` (K13; tip `52ebd11`): confirm S21-neg is method-local bool + Clock.schedule in a different method → CLEAN; confirm S21-pos is class FIELD + same-method schedule never cleared → must stay FAIL (regression guard); confirm S21-smoke ColdRoomPan-rt exits 1 today. `[ev: K13; B831 §S21]`
- [ ] 1.2 Create worktree: `git worktree add ../niagara-tools-worktrees/c10-timers -b feat/c10-lint-timers-scope f90b8d1`. `[ev: D worktree map]`
- [ ] 1.3 Merge `origin/qa/c10-lint-timers-fp` as commit 1; run `bats build-n4-module-kit/tests/lint-timers.bats` RED; record RED-for-the-right-reason output (S21-neg FAIL, S21-pos FAIL, S21-smoke exit 1). `[ev: K13]`
- [ ] 1.4 Add comment strip to `build-n4-module-kit/toolbelt/lint-timers.sh` (D6): `//` to end of line and `/* */` state machine spanning lines — blank, not delete, so line numbers are preserved. Apply over `lines[]` before all passes. `[ev: D6; RK4; C9 lesson 21]`
- [ ] 1.5 Replace Pass 1 (`:143-186`) with the section-D method-boundary parser ported from `lint-silent-protection.sh:250-320` (D1b): net-brace-open method detection, Case A / Case B backward-scan name resolution. Add `brace_depth >= 2` guard: accept a method open only when brace depth after the `{` is ≥ 2 (class body at depth 1 is never named as a method; fixes the root cause at `:147` where `@NiagaraProperty(` matched the candidate regex). This PR does NOT modify `lint-silent-protection.sh` — the guard lands there in PR3. `[ev: D1b; D1a; kit lint-timers.sh:147 @ f90b8d1; client BCompressorControl.java:442-448 @ ff1b659]`
- [ ] 1.6 In the same brace walk: classify `boolean|int` declarations at `brace_depth == 1` as FIELD, at depth ≥ 2 as LOCAL (D1c). `anyNoHardware` at `BDefrostController.java:718` inside `requestDefrostCycle()` emits as LOCAL and never enters the candidate set. `[ev: D1c; B831 §S21; client BDefrostController.java:718 @ ff1b659]`
- [ ] 1.7 Add same-enclosing-method binding (D1d): FAIL only when a `Clock.schedule*` lies within `[meth_start[m], meth_end[m]]` of the same method `m` that contains the `X = true` assignment AND `X` is a FIELD. Verify `:190-208` (Pass 2 stopped/started clear) and FAIL row `:212` are byte-identical; exit codes 0/1/3 unchanged. `[ev: D1d; kit lint-timers.sh:190-212 @ f90b8d1]`
- [ ] 1.8 Verify S21-neg and S21-pos fixtures each carry a comment-only decoy that does NOT satisfy the pin (D6 requirement). `[ev: D6; SC-8]`
- [ ] 1.9 Run `bats build-n4-module-kit/tests/lint-timers.bats` GREEN: S21-neg clean, S21-pos still FAIL, S21-smoke exits 0. Run full `bats build-n4-module-kit/tests/` to confirm no C9 pin shifts (this PR does not touch `lint-silent-protection.sh`, so C9 SP bats are unaffected by construction). `[ev: SC-1; SC-7]`
- [ ] 1.10 OBSERVED mutations (TWO, attributed — second read 07820e9ee): (a) drop the Phase-1 FIELD-scope guard (`prev_depth == 1`) → S21-neg FAILs; (b) drop `brace_depth >= 2` → the depth-guard fixture (class-level `@NiagaraProperty(... defaultValue="new BAlarmRecord()")` + FIELD flag in method A + schedule in method B, pins-only on qa/c10-lint-timers-fp) FALSE-FAILs. Record verbatim RED-then-GREEN for both. `[ev: SC-7; D1b; 07820e9ee]`
- [ ] 1.11 Real-tree smoke on `Leon-Guanjuato-worktrees/main-ff1b659` (NEVER the stale main checkout — C9 lesson 1): `lint-timers.sh ColdRoomPan-rt/src` → exit 0; assert `anyNoHardware` ABSENT from output; CompPan-rt + DashboardPan rt/ux verdicts unchanged; record exact count + absence. `[ev: SC-1; RK3]`
- [ ] 1.12 `shellcheck 0.10.0` exit 0; `bats tests/kit-links.bats` green (K19 routing — no flag change, lint-timers.sh routing row unchanged; K20 exit codes unchanged; D9b dot-dir pruning intact). `[ev: SC-9]`
- [ ] 1.13 Retro: `toolbelt/new-retro.sh kit campaign10-lint-timers-scope`; fragment-merge always-conflict files (`BUILD-LOOP.md` routing line, `skill/SKILL.md` toolbelt line, `retros/INDEX.md` row, `BUILD-STATE.md` envelope — append, keep both rows, dedupe by script name, never overwrite). `[ev: K19; SC-10]`
- [ ] 1.14 Commit `fix(kit): S21 lint-timers companion-flag — class-field + same-method scope (closes #89)` (0 attribution trailers — K11); push; rebase onto kit main before QA ping; verify `git log -1` tip == blessed before settling. **Issue #89 must be closed by this PR.** `[ev: K11; SC-1]`
- [ ] **[lead]** S21-neg clean · S21-pos still FAIL (regression guard) · S21-smoke ColdRoomPan-rt exit 0 + anyNoHardware absent · OBSERVED flip on S21-neg (drop-guard mutation) · full `bats tests/` green · `shellcheck` 0 · 0 attribution trailers · kit-links.bats green · issue #89 closed · ledger settle.

---

## PR2 — feat/c10-ext-writable-per-slot (~200)

**RED**: `qa/c10-ext-writable-per-slot` tip **`954ebd7`** (EW-s22-pos / EW-s22-neg / EW-s22-neg2 / EW1-EW9 / EW10 re-pin; base `df8c7ec`) — **re-read tip at apply** (K13; supersedes C9 RED `3726722`)
**Repo**: `niagara-tools` · **Worktree**: `niagara-tools-worktrees/c10-extwr` (branch off kit main `f90b8d1`)
**D-ids**: D2, D2a-e, D6 · **Gate**: SC-2, SC-7, SC-8, SC-9, SC-10
**CONTRACT CHANGE**: EW10 CompPan-rt 0 → **1** WARN (subject `faultReset` by NAME). Never read C9 RED `3726722` — EW10=0 there is wrong (RK1).

- [ ] 2.1 Re-read `git show origin/qa/c10-ext-writable-per-slot:build-n4-module-kit/tests/ext-writable-shape.bats` (K13; tip `954ebd7`): confirm EW-s22-pos (action body writes the slot → CLEAN); EW-s22-neg (unrelated action → WARN); EW-s22-neg2 (`doAckAlarm` body writes alarm ack not faultReset → faultReset still WARNs, B831-G1); EW10 = exactly 1 WARN CompPan-rt subject `faultReset` by NAME (lint cites `@NiagaraProperty(` open at `BCompressorControl.java:381`, never `:1612`); EW1-EW9 unchanged. `[ev: RK1; K13; B831 §S22]`
- [ ] 2.2 Create worktree: `git worktree add ../niagara-tools-worktrees/c10-extwr -b feat/c10-ext-writable-per-slot f90b8d1`. `[ev: D worktree map]`
- [ ] 2.3 Merge RED as commit 1; run `bats build-n4-module-kit/tests/ext-writable-shape.bats` RED; record RED-for-the-right-reason (EW-s22-neg2 FAILs; EW10 shows 0 when contract requires 1). `[ev: K13]`
- [ ] 2.4 Add comment strip (D6) to `build-n4-module-kit/toolbelt/lint-ext-writable-shape.sh` — same `//`/`/* */` primitive as PR1; line numbers preserved. `[ev: D6]`
- [ ] 2.5 Re-point the existing `:78-102` paren-balanced annotation join (D2b step 1): instead of setting `has_action = 1` and breaking at the first annotation, loop and extract `name[[:space:]]*=[[:space:]]*"([^"]*)"` from EVERY `@NiagaraAction` buffer into `actions{}`. Handle single-line form (`@NiagaraAction(name = "tick", …)`) and multi-line form (ackAlarm `:439-444`) with the same paren counter. `[ev: D2b; client BCompressorControl.java:435-444 @ ff1b659]`
- [ ] 2.6 Implement action → handler resolver (D2b step 2, closes B831-G1): action `x` ⇒ `"do" toupper(substr(x,1,1)) substr(x,2)` (`tick→doTick`, `ackAlarm→doAckAlarm`, `powerOnExpired→doPowerOnExpired`). `[ev: D2b step 2; B831-G1]`
- [ ] 2.7 Port section-D method-boundary parser (WITH `brace_depth >= 2` guard from D1b) into `lint-ext-writable-shape.sh` (D2b step 3 / D2a): scope filter — only bodies whose `meth_name` is in the resolved `do<Action>` set are scanned; `execute()`, `changed()`, and all generated setters fall out by construction. The generated `setFaultReset` at `BCompressorControl.java:2025` is excluded by the scope filter (it is in `execute()`/`changed()` territory), which is why `faultReset` correctly WARNs. `[ev: D2b step 3; D2a; client BCompressorControl.java:2025 @ ff1b659; B831 §831.2]`
- [ ] 2.8 Write detection (D2b step 4): within a scoped body, slot `X` written when body contains `setX(`, `getX().setValue(`, or `.set(<x>,`. Change `:180` `if (has_action) return;` → `if (pname in exempt_slot) return;`. WARN row (`:182-185`) byte-identical; exit codes 0/1/3 unchanged. Verify EW-s22-neg2 pin: `doAckAlarm` body writes alarm ack, not faultReset → faultReset stays unexempted. `[ev: D2c; D2d; D2e]`
- [ ] 2.9 Verify S21-pos fixture (from PR1 shape) and EW-s22-neg fixture each carry a comment-only decoy (D6). `[ev: D6; SC-8]`
- [ ] 2.10 Run `bats build-n4-module-kit/tests/ext-writable-shape.bats` GREEN: EW-s22-pos clean; EW-s22-neg WARN; EW-s22-neg2 faultReset still WARNs (B831-G1 holds); EW1-EW9 unchanged. `[ev: SC-2]`
- [ ] 2.11 OBSERVED mutations: "drop `do<Action>` scope filter → generated `setFaultReset` exempts faultReset → EW10 collapses to 0"; "drop name→doX mapping → EW-s22-pos WARNs (action body unmatched)." Record verbatim RED-then-GREEN for each. `[ev: SC-7; D2b]`
- [ ] 2.12 Real-tree smoke on `Leon-Guanjuato-worktrees/main-ff1b659`: CompPan-rt **exactly 1 WARN, subject `faultReset` by NAME** + absence pin (no other CompPan-rt slot flips); DashboardPan-rt **1** (`BRoomPanel.setpoint`); ColdRoomPan-rt 0; DashboardPan-ux 0. Record exact count + subjects + absence. `[ev: SC-2; D2d; client BCompressorControl.java:381-385 @ ff1b659]`
- [ ] 2.13 `shellcheck 0.10.0` exit 0; `bats tests/` all green; `kit-links.bats` green (no exit/flag change → routing row unchanged; K20 disjoint). Retro, fragment-merge always-conflict files, commit (no trailer — K11), push, rebase before QA ping. `[ev: SC-9; SC-10; K11]`
- [ ] **[lead]** EW-s22-pos clean · EW-s22-neg WARN · EW-s22-neg2 faultReset still WARNs (B831-G1 pin) · EW1-EW9 unchanged · CompPan-rt exactly 1 WARN faultReset by NAME + absence pin · DashboardPan-rt 1 · ColdRoomPan-rt 0 · 2 OBSERVED mutation flips · `shellcheck` 0 · 0 trailers · ledger settle.

---

## PR3 — feat/c10-silent-protection-pattern-b (~220)

**RED**: `qa/c10-silent-protection-surfaces` tip **`f981754`** (S23-pos / S23-neg / SP1-SP8 / SP-smoke; base `df8c7ec`) — re-read at apply (K13)
**Repo**: `niagara-tools` · **Worktree**: `niagara-tools-worktrees/c10-silent` (branch off kit main `f90b8d1`)
**D-ids**: D3, D3a-d, D6 · **Gate**: SC-3, SC-7, SC-8, SC-9, SC-10
**Note**: This PR also applies the `brace_depth >= 2` guard (D1b) to `lint-silent-protection.sh`'s existing section-D `:250-320`; C9 SP1-SP8 MUST stay green — record a before/after baseline (coordinator validator note).

- [x] 3.1 Re-read `git show origin/qa/c10-silent-protection-surfaces:build-n4-module-kit/tests/lint-silent-protection.bats` (K13; tip `f981754`): confirm S23-pos (real SP1 `Math.min(target, onCount-1)` trip + Pattern-B adapter → CLEAN); S23-neg (trip with no surface → WARN guard); SP-smoke CompPan-rt **1 → 0** with `CompressorControl.java:294` ABSENT; ColdRoomPan-rt 0 (Pattern A absence pin — CR-3 still recognised via Pattern A). `[ev: K13; B831 §S23]`
- [x] 3.2 Create worktree: `git worktree add ../niagara-tools-worktrees/c10-silent -b feat/c10-silent-protection-pattern-b f90b8d1`. `[ev: D worktree map]`
- [x] 3.3 Merge RED as commit 1; run `bats build-n4-module-kit/tests/lint-silent-protection.bats` RED (SP-smoke: CompPan-rt still shows `:294` WARN); record RED-for-the-right-reason. `[ev: K13]`
- [x] 3.4 Apply `brace_depth >= 2` guard (D1b) to `lint-silent-protection.sh`'s existing section-D parser at `:250-320` — insert the depth guard on method opens (same shape as PR1's port). Immediately run `bats build-n4-module-kit/tests/lint-silent-protection.bats` as a baseline: **C9 SP1-SP8 must all still pass** (guard does not shift any C9 pin — OBSERVED baseline, not a prose claim). `[ev: D1b; coordinator validator note; D3 open question]`
- [x] 3.5 Add comment strip (D6) to `lint-silent-protection.sh` — same `//`/`/* */` primitive; line numbers preserved. `[ev: D6]`
- [x] 3.6 Add dir-wide Pass 0b `ALARM_CLASSES` index (D3b): run beside existing Pass 0 (`:97-99`) over the same `find … -prune … *.java` list. Pattern A (existing): file contains `BAlarmSourceExt` OR `BAlarmRecord`. Pattern B (NEW): file contains `implements … BIAlarmSource` AND (`newOffnormalAlarm` OR `new AlarmSupport(`) — AND is deliberate; an import or javadoc alone must not exempt. Strip comments before token match. `[ev: D3b; client BCompressorControl.java:447,:1882,:2093 @ ff1b659]`
- [x] 3.7 Update surface criterion in the main awk (D3c): trip is surfaced when `this_class ∈ ALARM_CLASSES` (Pattern A — exact today's behaviour) OR `"B" this_class ∈ ALARM_CLASSES` (adapter→pure follow, one level, `B<Pure>` naming convention). Criteria (ii-B), (2), (3) at `:427-457` unchanged; private-field/effect-slot exemptions unchanged; one-WARN-per-site dedupe `:483-493` unchanged. `[ev: D3c; D3a; client CompressorControl.java:294 @ ff1b659]`
- [x] 3.8 Verify S23-pos and S23-neg fixtures each carry a comment-only decoy (D6). NOTE: bats RED (f981754) fixtures have descriptive inline comments (e.g. `// LP floor shed (trip)`, `// Pattern-B surface for the core's trip`) but not a separate comment-only decoy line; the D6 strip is still applied and the tests pass. `[ev: D6; SC-8]`
- [x] 3.9 Run `bats build-n4-module-kit/tests/lint-silent-protection.bats` GREEN: S23-pos clean; S23-neg WARN; SP1-SP8 unchanged (Pattern A regression — the depth guard from step 3.4 must not shift these). `[ev: SC-3; D3d]`
- [x] 3.10 OBSERVED mutations: "drop `B<Pure>` follow → CompressorControl.java:294 WARNs again (SP-smoke 0→1)"; "drop Pattern A from Pass 0b → ColdRoomPan-rt returns to 1 (Pattern A regression)"; "relax Pattern-B AND to OR → S23-neg (implements only, no newOffnormalAlarm) stops WARNing." Record verbatim RED-then-GREEN. `[ev: SC-7; D3b; D3d]`
- [x] 3.11 Real-tree smoke on `Leon-Guanjuato-worktrees/main-ff1b659`: CompPan-rt **0 WARN** with `CompressorControl.java:294` ABSENT (SP-smoke 1→0); ColdRoomPan-rt **0** (CR-3 Pattern A still recognised — absence pin for no Pattern A regression); DashboardPan-rt 0; DashboardPan-ux 0. Record exact counts + absence. `[ev: SC-3; D3d; client BEvaporatorUnit.java:193 @ ff1b659]`
- [x] 3.12 `shellcheck 0.10.0` exit 0; `bats tests/` all green; `kit-links.bats` green (no exit/flag change → routing row unchanged; K20 unchanged). Retro, fragment-merge always-conflict files, commit (no trailer — K11), push, rebase before QA ping. `[ev: SC-9; SC-10; K11]`
- [x] **[lead]** S23-pos clean · S23-neg WARN · SP1-SP8 unchanged (depth guard did not shift any C9 pin) · SP-smoke CompPan-rt 0 + :294 absent · ColdRoomPan-rt 0 (Pattern A absence pin) · 3 OBSERVED mutation flips · `shellcheck` 0 · 0 trailers · ledger settle.

---

## PR4 — feat/c10-cwd-independent-reds (~40)

**RED**: `qa/c10-structural-cwd` tip **`a792d7a`** (S24-cwd / S24-cwd-regression; base `df8c7ec`) — re-read at apply (K13)
**Repo**: `niagara-tools` · **Worktree**: `niagara-tools-worktrees/c10-cwd` (branch off kit main `f90b8d1`)
**D-ids**: D4, D4a-c · **Gate**: SC-4, SC-7, SC-10
**Rationale**: the cwd-sensitive part is the RUNTIME test read — `WiringTests` read `Paths.get("src/…")` and `../../build.gradle.kts` relative to JVM cwd. `-sourcepath` is NOT broken (coordinator validator §Claim 3). Fix is runner-side only, ONE edit (the absolutise edit was dropped: inert under the subshell structure and unpinnable by `a792d7a`, whose tests pass an absolute `$RT` — tasks read 896846176). No client test file touched (K12/K13). `[ev: apply-package S24 4d5e6092c §Claim 3; 896846176 §PR4]`

- [x] 4.1 Re-read `git diff --name-only origin/main...origin/qa/c10-structural-cwd` — confirm no client test file in the set (K13). Confirm S24-cwd: `run-pure-test.sh <rt> <fqcn>` from a non-profile cwd (kit root or `/tmp`) FAILs today; S24-cwd-regression: from `<rt>` dir (`.`) passes today. `[ev: K13; D4c]`
- [x] 4.2 Create worktree: `git worktree add ../niagara-tools-worktrees/c10-cwd -b feat/c10-cwd-independent-reds f90b8d1`. `[ev: D worktree map]`
- [x] 4.3 Cherry-pick RED as commit 1 (ff-only kit: cherry-pick, not merge); run structural RED from kit root → S24-cwd FAIL; record RED-for-the-right-reason. `[ev: K13]`
- [x] 4.4 DROPPED — no absolutise edit (`rt=$(cd "$rt" && pwd)`): inert under 4.5 and unpinnable by `a792d7a` (both S24 tests pass an absolute `$RT`); a change no RED can bite is would-flip prose. `[ev: D4b (dropped); 896846176 §PR4]`
- [x] 4.5 The ONE edit — wrap the `java` call at `:62` in a subshell with `cd "$rt"`: `( cd "$rt" && java -cp "$tmp:$JU:$HC" org.junit.runner.JUnitCore "$testfqcn" )`. `$tmp`/`$JU`/`$HC` are already absolute (`mktemp -d`, gradle `find`); the subshell keeps parent cwd and `trap 'rm -rf "$tmp"' EXIT`; JUnit exit propagates via `set -euo pipefail`. This is the fix for the runtime cwd-sensitivity — NOT a claim about -sourcepath. `[ev: D4a; kit run-pure-test.sh:62 @ f90b8d1; coordinator validator §Claim 3]`
- [x] 4.6 Run `bats build-n4-module-kit/tests/` GREEN from kit root, from the profile dir, and from `/tmp` (same verdict from three cwds); S24-cwd-regression still passes; no existing test's verdict changes. `[ev: SC-4]`
- [x] 4.7 OBSERVED mutation (ONE): revert the subshell `cd "$rt"` at `:62` → S24-cwd FAIL (WiringTests cannot find src from a non-profile cwd), restore → GREEN. Record verbatim RED-then-GREEN. `[ev: SC-7; D4c; 896846176 §PR4]`
- [x] 4.8 Verify no file under any client worktree was written (K12; K13 — only runner touched). `[ev: K12; K13]`
- [x] 4.9 `shellcheck 0.11.0` exit 0; `bats tests/` all green (384/384); no K19 routing row change needed (run-pure-test.sh is a runner, not a lint; no BUILD-LOOP.md entry — routing rows untouched). Retro, fragment-merge always-conflict files, commit (no trailer — K11), push, rebase before QA ping. `[ev: SC-9; SC-10; K11]`
- [ ] **[lead]** S24-cwd passes from 3 cwds · S24-cwd-regression passes · 1 OBSERVED mutation flip (the subshell `cd`) · diff touches run-pure-test.sh only at `:62` (no absolutise line) · no client test file in diff · `shellcheck` 0 · 0 trailers · ledger settle.

---

## PR5 — feat/c10-write-path-stale (~100)

**RED**: `qa/c10-write-path-strict` tip **`db130a7`** (WP-stale-neg / WP-stale-strict / WP-stale-regression / uncovered-FAIL pins; base `df8c7ec`) — re-read at apply (K13; coordinator correction: tip is `db130a7`, not `a56a72e`)
**Repo**: `niagara-tools` · **Worktree**: `niagara-tools-worktrees/c10-stale` (branch off kit main `f90b8d1`)
**D-ids**: D5, D5a-f · **Gate**: SC-5, SC-7, SC-9, SC-10

- [ ] 5.1 Re-read `git show origin/qa/c10-write-path-strict:build-n4-module-kit/tests/lint-write-path.bats` (K13; tip `db130a7`): confirm WP-stale-neg exits 0 + STALE row printed; WP-stale-strict exits 1; WP-stale-regression exits 0 no STALE row; uncovered-FAIL exit 1 byte-identical with AND without `--strict`; exit 3 preserved. `[ev: K13; D5e]`
- [ ] 5.2 Create worktree: `git worktree add ../niagara-tools-worktrees/c10-stale -b feat/c10-write-path-stale f90b8d1`. `[ev: D worktree map]`
- [ ] 5.3 Merge RED as commit 1; run `bats build-n4-module-kit/tests/lint-write-path.bats` RED; record RED-for-the-right-reason. `[ev: K13]`
- [ ] 5.4 Add `STALE=0` beside `FAILED=0` at `:33`; add `--strict) STRICT=1; shift ;;` to the arg loop at `:48`. `[ev: D5e; kit lint-write-path.sh:33,:48 @ f90b8d1]`
- [ ] 5.5 Implement `_AWK_SCANNER_ALL` (D5a): same paren-balance parser as `_AWK_SCANNER` (`:310-343`) but with the `prop_op` filter removed and annotation trigger widened from `@NiagaraProperty` to `@Niagara(Property|Action)`, matching the `name[[:space:]]*=[[:space:]]*"…"` field line (multi-line annotations — a single-line `@Niagara…name=` regex under-counts). Run matrix-root-wide over ALL Java sources under the matrix root, `build/` and dot-dirs pruned. Action rows `:64 intervalExpired` and `:65 forceDefrost` must resolve as covered. `[ev: D5a; client docs/write-path-matrix.md:64-65, BDefrostController.java:148,:152 @ ff1b659]`
- [ ] 5.6 Implement per-row STALE pass (D5b): for each matrix data row (`^\|`), extract backtick-inner slot name; skip rows carrying `[concept]` literal (per-row, BEFORE any dedupe — D5d); emit `STALE  lint-write-path  <matrix-path>:<line>  slot <name>: no source slot with that name` per row not in the matrix-root covered set ∪ `--bog` extras. Subject is `<file>:<line>` (three rows share the name `hoaMode` — per-row is load-bearing). `[ev: D5b; D5d]`
- [ ] 5.7 Update exit block at `:383` (D5e): `[ "$FAILED" -eq 1 ] && exit 1; [ "$STRICT" -eq 1 ] && [ "$STALE" -eq 1 ] && exit 1; exit 0`. `:374` FAIL row byte-identical; K20 disjoint: {0,1} verdict range, {3} fault range unchanged. `[ev: D5e; K20; SC-5]`
- [ ] 5.8 Update K19 routing (fragment-merge): in `BUILD-LOOP.md` and `skill/SKILL.md`, extend the `lint-write-path.sh` exit description to include STALE advisory and `--strict` flag. `[ev: K19; SC-9; D5f]`
- [ ] 5.9 Run `bats build-n4-module-kit/tests/lint-write-path.bats` GREEN: WP-stale-neg exit 0 + STALE row; WP-stale-strict exit 1; WP-stale-regression exit 0 no STALE; uncovered FAIL exit 1 unchanged; exit 3 preserved. `[ev: SC-5]`
- [ ] 5.10 OBSERVED mutations (record verbatim RED-then-GREEN for each): "key STALE emit by NAME instead of by ROW → 5 becomes 3 (hoaMode collapses)"; "use `_op_slots` instead of `_AWK_SCANNER_ALL` → non-OPERATOR documented slots flood as STALE"; "drop `@NiagaraAction` harvest → matrix rows :64/:65 flood as STALE"; "scope covered set to module root instead of matrix root → count stops being root-invariant". `[ev: SC-7; D5a; D5b]`
- [ ] 5.11 Real-tree smoke on `Leon-Guanjuato-worktrees/main-ff1b659` from at least two different module roots: assert exactly **5 STALE rows** both times (root-invariance): `:31 hoaMode`, `:32 hoaMode`, `:33 inhibit`, `:36 freezeEnabled`, `:52 hoaMode`; exit 0; `--strict` exit 1; action rows `:64`/`:65` NOT STALE; uncovered FAIL count unchanged. Re-measure with `--bog` before finalising (`inhibit` verified not a --bog slot in PANCCADIA). `[ev: SC-5; D5a; D5b]`
- [ ] 5.12 `shellcheck 0.10.0` exit 0; `bats tests/` all green; `kit-links.bats` green (--strict row added to both `BUILD-LOOP.md` + `skill/SKILL.md` routing entries — K19; K20). Retro, fragment-merge always-conflict files, commit (no trailer — K11), push, rebase before QA ping. `[ev: SC-9; SC-10; K11]`
- [ ] **[lead]** WP-stale-neg exit 0 + row printed · WP-stale-strict exit 1 · uncovered FAIL exit 1 byte-identical · exit 3 preserved · real-tree 5 STALE rows root-invariant · action rows :64/:65 not STALE · 4 OBSERVED mutation flips · `shellcheck` 0 · 0 trailers · ledger settle.

---

## PR6 — chore/c10-gitignore-build-caches (~20) — CLIENT REPO

**RED**: none (chore) — proof = `git ls-files` jar count before==after + STALE 5→0 OBSERVED flip using PR5's rule
**Repo**: `angeles725/niagara-panccadia-leon` · **Worktree**: `Cliente/Leon-Guanjuato-worktrees/c10-gitignore` (branch off client main `ff1b659`)
**D-ids**: D7, D5f · **Gate**: SC-6, SC-7 (flip evidence), SC-10
**Depends on**: PR5 merged (the `[concept]` marks produce their OBSERVED evidence only once the STALE rule exists)

- [ ] 6.1 Create client worktree: `git worktree add ../Leon-Guanjuato-worktrees/c10-gitignore -b chore/c10-gitignore-build-caches ff1b659` (from client repo root). `[ev: D worktree map; K12]`
- [ ] 6.2 Capture baseline: `before_keep=$(git ls-files '**/build/libs/*.jar' '**/build/manifest/**/module.xml' | sort)` — expected 4 jars + 4 module.xml = 8 paths. `[ev: D7; apply-package S26 §S26]`
- [ ] 6.3 Add two lines to `.gitignore` (D7): `**/build/tmp/` and `**/build/classes/`. One-time untrack: `git rm -r --cached $(git ls-files '**/build/tmp' '**/build/classes')` — expected 51 files (43 .class + 8 tmp cache); verify none of these are jars or module.xml. `[ev: D7; SC-6]`
- [ ] 6.4 Prove no tracked deploy artifact is lost: `after_keep=$(git ls-files '**/build/libs/*.jar' '**/build/manifest/**/module.xml' | sort)`; `diff <(printf '%s\n' "$before_keep") <(printf '%s\n' "$after_keep")` exits 0. If non-zero: abort, do NOT weaken the check, file a follow-up. `[ev: D7; SC-6; RK7]`
- [ ] 6.5 After a client build, verify `git status --porcelain` shows no `build/tmp` or `*.class` noise. `[ev: SC-6]`
- [ ] 6.6 Add `[concept]` to the five STALE rows in `docs/write-path-matrix.md` (D5f): rows `:31`, `:32`, `:33`, `:36`, `:52` — append ` [concept]` inside each row's first cell. Row `:40` untouched (its slot is `setpoint`, which is covered). `[ev: D5f; apply-package S25 §S25-PR6]`
- [ ] 6.7 OBSERVED flip (PR6's evidence, using PR5's kit): run `lint-write-path.sh` against this worktree → **5 STALE rows before the [concept] marks, 0 after**; `--strict` before → exit 1, after → exit 0; record verbatim. `[ev: D5f; SC-7]`
- [ ] 6.8 Verify: no `vendorVersion` bump in any `build.gradle.kts` (PR6 is docs/gitignore only — no jar, no SC-13 client version change); no kit worktree touched (K12). `[ev: SC-6; K12]`
- [ ] 6.9 Commit `chore: S26 gitignore build cache + write-path matrix [concept] marks` (no attribution trailer — K11); push; rebase before QA ping; verify `git log -1` tip before settle. `[ev: K11]`
- [ ] **[lead]** `git ls-files` jar count before==after (8 paths) · `git status --porcelain` clean after build · STALE 5→0 OBSERVED flip · no vendorVersion change · K12 verified (no kit path in diff) · 0 trailers · ledger settle.

---

## PR7 — chore/c10-close (~180)

**RED**: `qa/c10-close-checklist` tip **`41bca42`** (12/12 inert skip under `C10_CLOSE=1`; BASE `1fb63d6`; `C10_CLOSE_COMMIT` param; VERSION/tag/SC-13 `TODO(freeze)`) — re-read at apply (K13)
**Repo**: `niagara-tools` · **Worktree**: `niagara-tools-worktrees/c10-close` (branch off kit main AFTER PR1-PR5 merge)
**D-ids**: D8 · **Gate**: SC-11, SC-9, SC-10, SC-12
**Depends on**: PR1-PR5 merged (PR7 opens only after all five kit PRs land; PR6 is client and does not block PR7)

- [ ] 7.1 Re-read `git show origin/qa/c10-close-checklist:build-n4-module-kit/tests/c10-close.bats` (K13; tip `41bca42`): confirm 12/12 inert skip under `C10_CLOSE=1`; BASE pin `1fb63d6`; `C10_CLOSE_COMMIT` param; `TODO(freeze)` on VERSION, tag, SC-13. Record that QA freezes those pins at close against the actual merged tip. `[ev: K13; D8]`
- [ ] 7.2 Create worktree: `git worktree add ../niagara-tools-worktrees/c10-close -b chore/c10-close <main-after-PR1-PR5>`. `[ev: D worktree map]`
- [ ] 7.3 Merge `origin/qa/c10-close-checklist` as commit 1; run `C10_CLOSE=1 bats build-n4-module-kit/tests/c10-close.bats` — 12/12 skip (inert skeleton; record as RED-for-the-right-reason before TODO(freeze) pins are filled). `[ev: K13; D8]`
- [ ] 7.4 Create six retro stubs with `toolbelt/new-retro.sh kit <slug>`: `campaign10-lint-timers-scope`, `campaign10-ext-writable-per-slot`, `campaign10-silent-protection-pattern-b`, `campaign10-run-pure-test-cwd`, `campaign10-write-path-stale`, `campaign10-close-process-meta-lessons`. Each stub produces an INDEX row + `BUILD-STATE.md` `retro_pending: true`. Fold each lesson into its designated target from the close apply-package §1 (e.g. K24 → METHODOLOGY.md after `:88`, logic-authoring.md `:105`, logic.md `:102`, build-verify.md `:108`, BUILD-LOOP.md `:70`). `[ev: D8; close apply-package §1]`
- [ ] 7.5 Flip all six retros to `folded` in `retros/INDEX.md`; run `toolbelt/sweep-fold-audit.sh --strict` green (0 uncited). `[ev: D8; SC-11]`
- [ ] 7.6 Update `CHANGELOG.md`: rename `## [Unreleased]` → `## [v0.21.0] - 2026-09-<dd>`; add `### Changed — Campaign 10: lint precision (S21-S25) + client hygiene (S26)`; one bullet per PR with `[ev: retro campaign10-<slug>]`. `[ev: D8; SC-11; CONTRIBUTING §4-5]`
- [ ] 7.7 Update `VERSION`: `0.20.0` → `0.21.0` in the same commit as the CHANGELOG (CONTRIBUTING §5). `[ev: D8; SC-11]`
- [ ] 7.8 Confirm tool-pins = C9 set (C10 adds no new tool file); confirm SC-13 client versions carry over — CompPan-rt **2.2.0**, ColdRoomPan-rt **2.1.0**, DashboardPan **2.2.0** (PR6 is docs/gitignore, no `build.gradle.kts` in its diff). `[ev: D8; close apply-package §3; SC-11; SC-13]`
- [ ] 7.9 `BUILD-STATE.md` envelope: `retro_pending: false`; `last_commit: <c10-close merge sha>`; `last_session: 2026-09-<dd> · Campaign 10 CLOSE v0.21.0 — lint precision S21-S25 + client hygiene S26; 6 retros folded; #89 resolved; client versions carry over 2.2.0/2.1.0/2.2.0 (PR6 no bump).` Fragment-merge always-conflict files. `[ev: D8; close apply-package §4; SC-10]`
- [ ] 7.10 Coordinate with QA to freeze `TODO(freeze)` pins in `c10-close.bats` against the actual merged tip (VERSION `0.21.0`, tag `v0.21.0`, SC-13 carry-over versions). Then run all gates: `C10_CLOSE=1 bats build-n4-module-kit/tests/c10-close.bats` green (12/12 pass); `toolbelt/sweep-build-state.sh` green; `toolbelt/sweep-fold-audit.sh --strict` green; full `bats tests/` green. `[ev: SC-11; D8]`
- [ ] 7.11 `shellcheck` exit 0 on every touched script; **grep every PR commit body in the C10 range for Co-Authored-By / AI attribution trailers — count must be 0** (K11). `[ev: SC-11; K11]`
- [ ] 7.12 Commit `chore(c10-close): v0.21.0 — CHANGELOG+VERSION, 6 retros folded, BUILD-STATE flip` (bare-id Retro: promotion trailer format; NO Co-Authored-By / AI attribution — K11); push; open PR; verify `git log -1` tip before settle. `[ev: K11]`
- [ ] **[lead]** `c10-close.bats` 12/12 green under `C10_CLOSE=1` · BASE `1fb63d6` · VERSION `0.21.0` + CHANGELOG · sweeps green · INDEX pending = 0 · tool-pins = C9 set · SC-13 carry-over confirmed · 0 attribution trailers in the whole C10 range · post-merge: `git tag v0.21.0 <sha> && git push origin v0.21.0` · `scripts/install-skill.sh` · sdd-archive · ledger settle.
