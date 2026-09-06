# Proposal: build-n4-module-campaign10

**Status**: proposal · **Phase**: propose (post-explore, pre-proposal handoff CONFIRMED)
**Source**: niagara-tools `v0.20.0` (tag `1fb63d6`, main `cb79676`, C9 archive `df8c7ec`) · **Target**: kit `v0.21.0` (MINOR)
**Client**: `angeles725/niagara-panccadia-leon` — main `ff1b659` (CompPan-rt 2.2.0, ColdRoomPan-rt 2.1.0, DashboardPan rt/ux 2.2.0)
**Inputs**: `openspec/changes/build-n4-module-campaign10/explore.md` (gate-passed; §1.1 items 1-5 are constraints) · research **B831** (`686b54ac5`) · apply-packages `2f710626d` · QA C10 RED branches · kit issue **#89** · `retros/2026-09-06-campaign9-close-process-meta-lessons.md` (25 lessons)
**Topic key**: `sdd/build-n4-module-campaign10/proposal`
**Delivery**: auto-chain, ONE wave (W1), 7 PRs · review budget 400 changed lines/PR

---

## 1. Intent

Campaign 9 shipped five new kit lints in one campaign. Campaign 10 is the bill for that speed: three of those rules are **coarse where the code is fine-grained**, and each one is now demonstrably wrong on the operator's real modules. C10 spends one WSL-only wave making them precise, and nothing else.

**First, the kit lies to its own users.** `lint-timers.sh` FAILs ColdRoomPan-rt at exit 1 on `anyNoHardware`, which is a method-LOCAL at `BDefrostController.java:718` inside `requestDefrostCycle()` (`:713`, no `Clock.schedule` anywhere in it); the schedules are at `:808`/`:810`/`:850` in other methods. Pass 1 (`lint-timers.sh:135-168`) scans FORWARD from the assignment and never asks field-vs-local. A gate that fails a clean tree teaches the team to ignore the gate. Kit issue **#89** is this exact defect. `[ev: B831 §S21]` `[ev: apply-package S21 2f710626d]` `[ev: kit issue #89]`

**Second, the kit is blind where it claims coverage.** `lint-ext-writable-shape.sh:82-91` exempts a slot when the CLASS carries any `@NiagaraAction` — including HIDDEN, unrelated ones. So `BCompressorControl.faultReset` (`BStatusBoolean`, `SUMMARY|OPERATOR`) is exempted by `powerOnExpired`/`tick`, and CompPan-rt reports 0 WARN while carrying exactly the unwritable-shape defect the rule exists to catch. `[ev: B831 §S22]` `[ev: apply-package S22 2f710626d]`

**Third, C9's own alarm work is flagged by C9's own lint.** PR9 wired CP-1 as Pattern B (`BCompressorControl implements BIAlarmSource` `:447` + `new AlarmSupport(` `:1882` + `newOffnormalAlarm` `:2093`), yet `lint-silent-protection.sh` still WARNs `CompressorControl.java:294` because its recogniser (`:222`, `:424`) knows only Pattern A (`BAlarmSourceExt`/`BAlarmRecord`) and never crosses the adapter→pure boundary. A false alarm on a real fix is worse than no rule. `[ev: B831 §S23]` `[ev: apply-package S23 2f710626d]`

**Fourth, three hygiene defects block the next campaign's REDs.** Structural REDs resolve sources from `cwd` (`run-pure-test.sh` never enters the module dir before the `java` call at `:62`); `lint-write-path.sh` has no advisory class at all — it only knows FAIL (uncovered, exit 1) and ERROR (exit 3), so a stale matrix row (a row whose slot no longer exists in source) is invisible; and the client worktree churns `build/tmp` + `*.class` into every diff. Each is cheap now and expensive once C10/C11 author more structural REDs on top. `[ev: explore-draft 45d550dac §S24-§S26]`

---

## 2. Scope

### 2.1 In scope — W1 lint-precision wave (COMMITTED, WSL-only, needs nothing from Cristian)

| R-id | Item | Repo | Why now |
|---|---|---|---|
| **R1** | **S21** `lint-timers.sh` companion-flag — two guards: (1) the flag must be a **class FIELD** (class-scope declaration pass, not a `type name = …;` inside a method body); (2) the `Clock.schedule*` must sit in the **SAME enclosing method body** as the `= true` assignment (brace-count back from the method signature, never forward from the assignment). Anchors `:135-212`, Pass 1 `:135-168`, FAIL row `:212` | kit | Closes issue #89; a FAIL on a clean tree is the highest-cost defect class. FP-only, no contract change. `[ev: apply-package S21 2f710626d]` |
| **R2** | **S22** `lint-ext-writable-shape.sh` per-slot writing action — replace class-level `has_action` (`:82-91`) with a per-slot body-follow: a complex OPERATOR slot `X` is exempt only when an `@NiagaraAction(name="x")` maps to its `doX()` body and THAT body writes `X` (`setX(`, `set<X>(`, `.set(<Xprop>,`), excluding `execute`/`changed`/generated setters. Reuses `lint-silent-protection.sh:124-165` `SURF_WRITE` slot→writer follow. **CONTRACT CHANGE — EW10 CompPan-rt 0 → 1** | kit | The only false-NEGATIVE in the wave: today the rule reports clean on a real defect. `[ev: apply-package S22 2f710626d]` `[ev: B831 §S22]` |
| **R3** | **S23** `lint-silent-protection.sh` Pattern B surface — extend the recogniser (`:222`, `:424`) with `implements … BIAlarmSource` AND (`newOffnormalAlarm` OR `new AlarmSupport(`), plus the adapter→pure follow via the `B<PureClass>` naming pair. Keep the private-field / effect-slot exemptions and the exactly-one-WARN-per-trip dedupe | kit | Un-flags C9's own shipped CP-1 fix. FP-only, removes a WARN. `[ev: apply-package S23 2f710626d]` |
| **R4** | **S24** cwd-independent structural REDs — the fix is `cd "$rt"` before the `java` call at `toolbelt/run-pure-test.sh:62`, so `src` resolves from the module root and never from the caller's `cwd`. Lands BEFORE any further C10 structural RED is authored | kit/test | Every structural RED authored after this inherits the fix; every one authored before it inherits the bug. RED already authored and mutation-shaped (S24-cwd FAIL→pass plus a same-dir regression pass). `[ev: explore-draft 45d550dac §S24]` `[ev: qa/c10-structural-cwd a792d7a]` |
| **R5** | **S25** `lint-write-path.sh` STALE advisory + `--strict` — a NEW advisory class `STALE` (a matrix row whose slot name matches no `@NiagaraProperty` / `--bog` slot in source), row grammar `write-path  STALE  <slot>  no source slot with that name`; default exit **0** (advisory), `--strict` promotes STALE to exit **1**; the existing uncovered **FAIL exit 1 is untouched** (never weakened to WARN-only), exit 3 unchanged | kit | Verified at `cb79676`: the lint is already a hard FAIL gate; the draft's "add --strict" premise was wrong. Weakening a shipped gate to WARN-only was REJECTED (a worker forgetting `--strict` would ship an uncovered OPERATOR slot). K20 disjoint exit codes preserved. `[ev: QA 2026-09-07 S25 question; lead decision]` |
| **R6** | **S26** client `.gitignore` for `build/tmp` + `*.class` — **WITHOUT** untracking the module jars under `build/` (tracked by repo convention) | client | Stops per-diff churn that hides real changes from review. Chore, no jar, no version bump. `[ev: explore-draft 45d550dac §S26]` |
| **R7** | **C10 close** — `tests/c10-close.bats` (`C10_CLOSE=1`), `VERSION` **0.21.0** + CHANGELOG, retro folds to pending = 0, `BUILD-STATE.md` envelope, `retros/INDEX.md` rows | kit | Hard close gate (BUILD-LOOP §7). Skeleton already on `qa/c10-close-checklist` `41bca42` (12/12 inert skip, BASE `1fb63d6`, `C10_CLOSE_COMMIT` param, VERSION/tag/SC-13 `TODO(freeze)`) |

### 2.2 Out of scope — explicit, with the gate that would open each

| Item | Class | Gate that must resolve first |
|---|---|---|
| **P1** viewer per-user re-auth + configurator role list | tunnel product | Tunnel PRs #1/#2/#3 merged by Cristian + Supabase operator re-check + role-table decision `[ev: explore §4 W2]` |
| **P2** HMI per-operator kiosk login (DashboardPan-ux) | client -ux product | Cristian answers: attribution (R14, already shipped) vs per-operator screens (NEW RBAC) + harness session green `[ev: 3d212e746]` |
| **P3** `airDefrost` module flag (rooms 1/2/4) | client -rt product | Cristian's defrost-trial green light `[ev: explore §4 W2]` |
| **P4** intercambiador Cuarto 3 control point | station + client | Cristian confirms it sits on a Niagara output (YES) — otherwise the panel control is demoted `[ev: explore §4 W2]` |
| **P5** `coolOnSensorFault` station-side link (all rooms) | station (Workbench) | Cristian approves the station-side link `[ev: explore §4 W2]` |
| **Deploy chain** (2.0.7 / 2.0.3 / 2.1.1, then the C9 jars) | prerequisite | Cristian executes the deploy runbook; the station is four repo versions behind. NOT C10 work `[ev: before-c10-checklist §B]` |
| **niagaraTest harness session** (`qa/c9-harness-runsheet.md`, `qa/c9-verify-runbooks` `18420d9`) | prerequisite | Runs on Cristian's Windows box; closes C9 gate 14/14 and unblocks P2/P3/P4 alarm REDs. NOT C10 work `[ev: before-c10-checklist §C]` |
| **Tunnel PR #1/#2/#3 merge** | prerequisite | Cristian merges; C10 authors no tunnel code `[ev: before-c10-checklist §A]` |
| **Any live station WRITE or jar deploy** | — | Only with Cristian's direct authorization, routed through him. No SDD slice performs one |

---

## 3. Capabilities

> `openspec/specs/` does not exist in this repo. Per the campaign-6/7/8/9 convention, `sdd-spec` writes one change-local `spec.md`.

### New Capabilities

- None. C10 introduces no new script and no new module slot; every slice refines an existing contract. (Confirmed against `qa/c10-close-checklist` `41bca42`: tool-pins = the C9 set, no new tool files.)

### Modified Capabilities

- `module-timer-lint`: companion-flag verdict narrows — a FAIL requires a class **FIELD** flag AND a `Clock.schedule*` in the **same enclosing method body** as the assignment. A method-local flag, or a schedule in a different method, is clean. Row grammar, exit codes (0 / 1 any FAIL / 3 usage) unchanged. `[ev: B831 §S21]`
- `module-ext-writable-shape-lint`: exemption narrows from class-level "any `@NiagaraAction`" to **per-slot**: the action's `do<Action>()` body must write that slot. **Contract change** — EW10 re-pins CompPan-rt from 0 to **1 WARN** (subject `faultReset`). EW3 `setSetpoint` stays clean; EW6 stays WARN; the C9 EW-token / child-leaf note is unchanged. `[ev: B831 §S22]`
- `module-silent-protection-lint`: surface allowlist widens to Pattern B (`BIAlarmSource` + `AlarmSupport` / `newOffnormalAlarm`) and follows the `B<PureClass>` adapter→pure pair. A trip in a pure class is exempt when its adapter carries the surface. Pattern A, private-field/effect-slot exemptions and per-trip dedupe unchanged. `[ev: B831 §S23]`
- `module-write-path-matrix`: `lint-write-path.sh` gains the advisory `STALE` class (matrix row with no matching source/`--bog` slot) and `--strict` (STALE → exit 1). Uncovered FAIL exit 1 byte-identical to v0.20.0. `[ev: explore §4 S25]`
- `kit-test-harness`: structural REDs resolve their source tree from the test file's own location; running from any cwd yields the same verdict. `[ev: explore §4 S24]`

---

## 4. Approach — one wave, RED-first, one PR per slice

One kit PR per slice, each branching from kit `main` (`cb79676`). Every code PR **merges its QA RED branch commit first** and goes green against it (K13 — the RED is the contract; the tip is re-read at apply and workers NEVER edit a QA RED). Fixtures strip `//` and `/* */` before any identifier or WARN-string assertion and include a comment-only decoy (C9 lesson 21). Real-tree smokes run on the blessed worktree **`Leon-Guanjuato-worktrees/main-ff1b659`** (or a C10 worktree at that tip) — never the stale main checkout (C9 lesson 1) — and every smoke pin asserts **exact count + subject + absence**. Every rule change records an **OBSERVED** mutation flip (verbatim RED-then-GREEN); a "would flip" prose claim is not evidence. Process invariants: K11 no attribution trailers, K12 workers write only inside their own worktree, K14 metric names state what they measure, K19 every routed script named in `BUILD-LOOP.md` + `skill/SKILL.md`, K20 disjoint exit codes, D9b every scanner prunes dot-dirs, one `[ev:]` per paragraph/row. Each PR passes the **lead gate → QA verify → investigador1 second read** before merge.

| # | R-id | Branch / work unit | Repo | RED branch + tip | Est. changed | Lead gate (all must pass before merge) |
|---|---|---|---|---|---|---|
| **PR1** | R1 | `feat/c10-lint-timers-scope` | kit | `qa/c10-lint-timers-fp` **`52ebd11`** (base `df8c7ec`) `[CERT]` | ~180 | S21-neg (method-local + schedule in another method → clean) / S21-pos (class FIELD cleared off-lifecycle → FAIL **stays**, regression guard) / S21-smoke green; OBSERVED flip on S21-pos; real-tree smoke: ColdRoomPan-rt @ `ff1b659` exit **0** with `anyNoHardware` ABSENT from output, CompPan-rt + DashboardPan rt/ux unchanged; comment-decoy pin; `kit-links.bats`; `shellcheck` 0; 0 trailers; **issue #89 closed by the PR** |
| **PR2** | R2 | `feat/c10-ext-writable-per-slot` | kit | `qa/c10-ext-writable-per-slot` **`954ebd7`** (pins-only successor of `00e31ae`; adds EW-s22-neg2, the `doAckAlarm` shape) (base `df8c7ec`) `[CERT]` — re-read the tip at apply | ~200 | EW-s22-pos (action body writes the slot → clean) / EW-s22-neg (unrelated action → WARN) / EW1-EW9 unchanged / **EW10 = exactly 1 WARN on CompPan-rt, subject pinned by NAME `faultReset`** (never by line — the lint anchors the `@NiagaraProperty(` open at `BCompressorControl.java:381`; `:1612` is the generated `Property faultReset` the lint never cites); absence pin: no other CompPan-rt slot flips; DashboardPan-rt still 1 (`BRoomPanel.setpoint`), ColdRoomPan-rt 0, DashboardPan-ux 0; **B831-G1 pin**: a `do`-body writing a DIFFERENT slot does NOT exempt `faultReset`; OBSERVED flip; `kit-links.bats`; 0 trailers |
| **PR3** | R3 | `feat/c10-silent-protection-pattern-b` | kit | `qa/c10-silent-protection-surfaces` **`f981754`** (base `df8c7ec`) `[CERT]` | ~220 | S23-pos (real SP1 `Math.min(target, onCount-1)` trip + Pattern-B adapter → clean) / S23-neg (trip, no surface → WARN guard) / SP1-SP8 unchanged; SP-smoke CompPan-rt **1 → 0** with `CompressorControl.java:294` ABSENT; ColdRoomPan-rt 0 (CR-3 Pattern A still recognised — absence pin proves Pattern A did not regress), DashboardPan 0; OBSERVED flip on S23-neg; `kit-links.bats`; 0 trailers |
| **PR4** | R4 | `feat/c10-cwd-independent-reds` | kit | `qa/c10-structural-cwd` **`a792d7a`** (base `df8c7ec`) `[CERT]` — S24-cwd (`run-pure-test.sh` from a NON-profile cwd → FAIL today) + S24-cwd-regression (from the module-rt dir → pass) | ~40 | Fix is runner-side: enter `"$rt"` (subshell) before the `java` call at `toolbelt/run-pure-test.sh:62` (companero package `4d5e6092c`: the WiringTests read `Paths.get("src/…")` and `../../build.gradle.kts` relative to cwd); same verdict from the kit root, the profile dir and `/tmp`; OBSERVED flip = reverting the `cd` fails S24-cwd; no existing test's verdict changes; no client test file touched |
| **PR5** | R5 | `feat/c10-write-path-stale` | kit | **RED being authored by QA** on `df8c7ec` — `lint-write-path.bats` WP-stale-neg (extra matrix row → exit 0 with the STALE row printed) / WP-stale-strict (`--strict` → exit 1) / WP-stale-regression (matching matrix → exit 0, no STALE row) / existing uncovered FAIL exit 1 with and without `--strict` | ~100 | STALE pass over the covered-slot extraction (`:156`) and the per-profile scan (`:305`); real-tree: DashboardPan matrix at `ff1b659` (62 rows) reports the exact observed STALE count (expected **0**; if not, the number and subjects are the pin); uncovered FAIL exit 1 byte-identical; exit **3** preserved (K20 disjoint); `kit-links.bats` + SKILL/BUILD-LOOP row updated (K19) |
| **PR6** | R6 | `chore/c10-gitignore-build-caches` | client | No RED (chore) — carries a diff-shows-no-jar check | ~15 | `git status --porcelain` clean of `build/tmp` and `*.class` after a build; **`git ls-files` before == after** for every tracked `build/**/*.jar` (named pin: NO tracked jar becomes untracked); no `vendorVersion` bump; K12 — the client worktree is the only tree touched |
| **PR7** | R7 | `chore/c10-close` | kit | `qa/c10-close-checklist` **`41bca42`** (skeleton; QA freezes the `TODO(freeze)` pins at close) | ~180 | `tests/c10-close.bats` green under `C10_CLOSE=1` with `C10_CLOSE_COMMIT` set; BASE pin `1fb63d6`; `sweep-build-state.sh` + `sweep-fold-audit.sh --strict` green; `retros/INDEX.md` pending = **0**; `VERSION` `0.20.0` → **`0.21.0`** + CHANGELOG per CONTRIBUTING §4-5; tool-pins = the C9 set (no new tool file); `shellcheck` 0; **no attribution trailer in the whole PR range** |

**Merge order and conflict discipline.** PR1-PR5 are independent kit branches off `main`; each rebases onto `main` BEFORE the QA ping, and the lead verifies `git log -1` equals the blessed tip BEFORE settling the ledger (C8 lesson 10). The four always-conflict files — `BUILD-LOOP.md` routing line, `skill/SKILL.md` toolbelt line, `retros/INDEX.md` row, `BUILD-STATE.md` envelope — merge by **FRAGMENT**: append, keep both rows, dedupe by script name, **never overwrite**. PR6 is a different repo and never blocks a kit merge. PR7 opens only after PR1-PR5 are merged. Drafts land in a repo path (`sources/probes/`), never only in `/tmp`.

**Versions.** Kit `VERSION` + `CHANGELOG.md` → **`0.21.0`** in PR7 (MINOR: no CLI removal, one behaviour narrowing in S22 that is documented as a contract change). No client `vendorVersion` bump — S26 ships no jar. Gate exit: every kit PR is a kit-changing push → close-gate exit (a) NEW RETRO.

---

## 5. Alternatives considered

| Option | Verdict | Why |
|---|---|---|
| **Single mega-PR** for S21+S22+S23 (they share one defect family) | **REJECTED** | The three rules would cross-contaminate each other's contract: S22 is the only contract change in the wave, and folding it into a FP-only diff makes the EW10 0→1 re-pin invisible to review. Each also carries its own real-tree smoke on four module roots; one failing smoke would block three fixes. Separate PRs keep every OBSERVED flip attributable to one rule and each diff well under the 400-line budget. `[ev: apply-package §Sequencing 2f710626d]` |
| **Skip S22's contract change** — keep class-level `has_action`, fix only S21/S23 | **REJECTED** | S22 is the only **false NEGATIVE** in the wave: the rule reports CompPan-rt clean while `faultReset` carries exactly the defect it exists to catch. A lint that under-reports is worse than one that over-reports, because nobody looks again. QA already cut the re-pin INSIDE the C10 RED branch so no worker touches the C9 RED (K13), which removes the only real cost of the change. `[ev: B831 §S22]` `[ev: explore §5 risk 1]` |
| **Defer S24/S25/S26 to C11** | **REJECTED** | S24 is a prerequisite for every structural RED authored after it; deferring means C10's own later REDs inherit the cwd bug and get rewritten. S25 makes matrix→source drift visible before the matrix grows past its 62 rows. S26 is ~15 lines. `[ev: explore §4 W1]` |
| **Open W2 (P1-P5) in parallel with W1** | **REJECTED** | All five P-slices depend on gates only Cristian can resolve (tunnel merge, deploy chain, harness session, three station answers). Queuing them preserves W1's WSL-only autonomy and prevents mid-wave scope creep. `[ev: explore §5 risk 7]` |

---

## 6. Affected Areas

| Area | Impact | Description |
|---|---|---|
| `build-n4-module-kit/toolbelt/lint-timers.sh` | Modified | R1 — class-field pass + enclosing-method anchor (`:135-212`) |
| `build-n4-module-kit/toolbelt/lint-ext-writable-shape.sh` | Modified | R2 — per-slot `do<Action>` body-follow replaces class-level `has_action` (`:82-91`) |
| `build-n4-module-kit/toolbelt/lint-silent-protection.sh` | Modified | R3 — Pattern B surface tokens + adapter→pure follow (`:222`, `:424`) |
| `build-n4-module-kit/toolbelt/lint-write-path.sh` | Modified | R5 — STALE advisory class + `--strict`; uncovered FAIL unchanged |
| `build-n4-module-kit/toolbelt/run-pure-test.sh` | Modified | R4 — runner-side only, no test class touched — `cd "$rt"` before the `java` call (`:62`), so `src` resolves from the module root, never from the caller's `cwd` |
| `tests/{lint-timers,ext-writable-shape,lint-silent-protection,lint-write-path}.bats` + `tests/fixtures/**` | Modified/New | Per-slice RED suites (from the QA branches) + comment-decoy fixtures |
| `tests/c10-close.bats` | New | R7 close gate, `C10_CLOSE=1` |
| `build-n4-module-kit/{BUILD-LOOP.md, METHODOLOGY.md, skill/SKILL.md}` | Modified | K19 routing + the `--strict` flag row; fragment-merge only |
| `build-n4-module-kit/{retros/INDEX.md, BUILD-STATE.md, VERSION, CHANGELOG.md}` | Modified | One retro per kit-changing push; `0.20.0 → 0.21.0` |
| client `.gitignore` | Modified | R6 — `build/tmp` + `*.class`, jars under `build/` untouched |

---

## 7. Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| RK1 | **S22 contract-change drift** — a worker reads the C9 RED tip (`qa/c9-ext-writable-shape` `3726722`, EW10 = 0) and ships the wrong contract | High | The re-cut lives ONLY in `qa/c10-ext-writable-per-slot` (`954ebd7`); K13 — cite by branch, re-read the tip at apply, workers never edit a QA RED. EW10 pinned as exact count **1**, subject `faultReset` **by name**, plus an absence pin that no other CompPan-rt slot flips (module-find: 1 OPERATOR complex property in CompPan-rt) `[ev: QA 2026-09-07 max-WARN run on ff1b659]` |
| RK2 | **B831-G1 open** — `doAckAlarm` indirection: if the action→`do<Action>` mapping is wrong, `faultReset` is either wrongly exempted or a real action is wrongly WARNed | Med | The rule maps `@NiagaraAction(name="x")` → `doX()` and scans THAT body. EW-s22-pos already uses the shape (`bumpSetpoint` → `doBumpSetpoint`); `doAckAlarm` writes the alarm ack, not `faultReset`, so `faultReset` stays unexempted. An extra pin is requested from QA: a `do`-body writing a DIFFERENT slot must NOT exempt `faultReset` `[ev: B831-G1]` |
| RK3 | **Stale-checkout reads** — the campaign's recurring defect class (C9 lesson 1) | High | Every file cite names worktree + commit. Smoke tree = `Leon-Guanjuato-worktrees/main-ff1b659` or a C10 worktree at that tip; the stale main checkout is never a smoke target. A cite that cannot name its tip is not `[CERT]` `[ev: C9 lesson 1]` |
| RK4 | **Comment-satisfiable pins** (C9 lesson 21) — a `//`-commented token satisfies a regex pin and hides the defect | Med | Strip `//` and `/* */` before any identifier or WARN-string match; every S21/S22/S23 fixture carries a comment-only decoy that must NOT satisfy the pin `[ev: C9 lesson 21]` |
| RK5 | **S21 over-narrows** — requiring FIELD + same-method could silence a real `startingUp`/`powerOnTicket` defect | Med | S21-pos is a regression guard authored from the REAL CompPan shape (`:1760`/`:1764`) and must **stay FAIL**; the OBSERVED flip is recorded on S21-pos, not only on S21-neg `[ev: apply-package S21 2f710626d]` |
| RK6 | **S23 adapter follow is name-convention-based** (`B<PureClass>`) — a module that breaks the convention loses the exemption | Med | Documented as a stated limitation in the rule's doctrine row; the fallback is a WARN (advisory), never a FAIL, so a convention miss over-reports rather than hides. Pattern A recognition is pinned unchanged by the ColdRoomPan-rt 0 absence pin |
| RK7 | **S26 untracks the tracked jars** — a broad `build/` ignore silently removes tracked artifacts from the index | Med | Named pin: `git ls-files` for every tracked `build/**/*.jar` is **identical before and after**; the ignore targets `build/tmp` and `*.class` only. If the pin cannot be made to hold, S26 is dropped, not weakened |
| RK8 | **Always-conflict files across parallel kit PRs** (BUILD-LOOP routing, SKILL toolbelt line, `retros/INDEX.md`, `BUILD-STATE.md`) | High | Fragment-merge: append, keep both rows, dedupe by script name, never overwrite; rebase onto `main` before the QA ping; merge the wave before PR7 opens `[ev: C8 lesson 2]` |
| RK9 | **S25 RED not yet authored** (S24 RED landed at `a792d7a`) | Low | RED-first is a hard precondition: PR5 does not open until QA blesses the STALE pins. If not blessed by close, R5 rolls to C11 without holding the version bump — PR7 depends on PR1-PR4 only |
| RK10 | **P-slice scope creep if a station gate resolves mid-wave** | Med | P1-P5 queue for W2 after W1 closes. A resolved gate updates the W2 queue; it never interrupts the lint wave `[ev: explore §5 risk 7]` |
| RK11 | **`c10-close.bats` freezes the wrong pins** — VERSION/tag/SC-13 are `TODO(freeze)` on the skeleton | Low | QA freezes them at close against the actual merged tip; PR7's gate re-reads `qa/c10-close-checklist` at apply and verifies BASE `1fb63d6` and the C9 tool-pin set (no new tool file) |

---

## 8. Rollback Plan

| Slice | Rollback |
|---|---|
| PR1 / PR3 / PR4 / PR5 | `git revert` — FP-only or additive-flag changes to existing scripts plus new test files. Nothing downstream depends on the narrowed verdict; reverting restores the v0.20.0 behaviour exactly (issue #89 re-opens for PR1) |
| PR2 | `git revert` — restores class-level `has_action` and returns EW10 to CompPan-rt = 0. Because this is a **contract** revert, it must also revert the RED's EW10 re-pin in the same commit, otherwise the suite is red against a green tree |
| PR6 (client) | `git revert` of the `.gitignore` hunk. No jar, no slot, no deploy: schema-neutral, `schema-risk.sh` has no jurisdiction |
| PR7 | `git revert` — doc/version only; retro files are never deleted (propose-never-apply); `retros/INDEX.md` rows return to `pending`, `VERSION` returns to `0.20.0` |

No station write, no operator data mutation, and no live jar deploy is performed by any C10 slice. Deploys stay with the operator under BUILD-LOOP §6.

---

## 9. Dependencies

- `bats-core`, `shellcheck 0.10.0` (pinned in `ci.yml`), the live pre-push hook — installed.
- Research landed: **B831** (`686b54ac5`) — the single source-backed block for all three rules, `[CERT]` at client `ff1b659`. Open gaps carried as gates, not blockers: **B831-G1** (`doAckAlarm` indirection → PR2 pin), **B831-G2** (build the three fixtures → the RED branches already carry them). No further `research-sdd` iteration before spec.
- QA RED branches on kit origin, base `df8c7ec`: `qa/c10-lint-timers-fp` **`52ebd11`**, `qa/c10-ext-writable-per-slot` **`954ebd7`** (pins-only successor of `00e31ae`; adds EW-s22-neg2, the `doAckAlarm` shape), `qa/c10-silent-protection-surfaces` **`f981754`**, `qa/c10-structural-cwd` **`a792d7a`**, `qa/c10-close-checklist` **`41bca42`** (skeleton). **Being authored**: the S25 STALE pins (`lint-write-path.bats`).
- Apply-packages (niagara-research): S21/S22/S23 `4d5e6092c` (S22 corrected to the NAME pin and the `doX` mapping), S24/S25/S26 `2026-09-07-c10-s24-s25-s26-apply-packages.md` (S25 section being re-cut to the STALE rule); investigador1 S22 reuse-vs-new note `685e7981e` (new action-aware pass over the section-D method-boundary parser `:250-306` + Pass-0 paren-join; scope filter excludes `execute()`/`changed()`/generated setters).
- Real tree for every smoke: `Leon-Guanjuato-worktrees/main-ff1b659` (client main `ff1b659`), four module roots — ColdRoomPan-rt, CompPan-rt, DashboardPan-rt, DashboardPan-ux.
- Kit issue **#89** is closed by PR1.
- Not dependencies of any C10 PR, but blocking W2: tunnel PRs #1/#2/#3 merge, the deploy chain, the niagaraTest harness session (`qa/c9-verify-runbooks` `18420d9`), and Cristian's four product answers.

---

## 10. Success Criteria

- [ ] **SC-1 (S21)** — `lint-timers.sh` on `ColdRoomPan-rt/src` @ `ff1b659` exits **0**, and `anyNoHardware` is **ABSENT** from the output; S21-pos (class FIELD flag, schedule in the same method, never cleared) **still FAILs** with an OBSERVED flip; CompPan-rt / DashboardPan rt/ux verdicts unchanged; **kit issue #89 closed**.
- [ ] **SC-2 (S22)** — `lint-ext-writable-shape.sh` on `CompPan-rt/src` @ `ff1b659` emits **exactly 1 WARN**, subject `faultReset` asserted **by name**, with an absence pin that no other CompPan-rt slot flips; ColdRoomPan-rt **0**, DashboardPan-ux **0**, DashboardPan-rt still **1** (`BRoomPanel.setpoint`); the B831-G1 pin holds (a `do`-body writing a different slot does not exempt `faultReset`); OBSERVED flip on EW-s22-neg.
- [ ] **SC-3 (S23)** — `lint-silent-protection.sh` on `CompPan-rt/src` @ `ff1b659` emits **0 WARN** and `CompressorControl.java:294` is **ABSENT**; `ColdRoomPan-rt` **0** (Pattern A / CR-3 still recognised — absence pin proves no Pattern A regression); DashboardPan **0**; S23-neg (trip with no surface) still WARNs with an OBSERVED flip.
- [ ] **SC-4 (S24)** — `qa/c10-structural-cwd` `a792d7a` goes green: S24-cwd (non-profile cwd) flips FAIL → pass and S24-cwd-regression (module-rt dir) stays pass; the structural suite yields the **same verdict from three cwds** (kit root, profile dir, `/tmp`); reverting the `cd "$rt"` at `run-pure-test.sh:62` fails S24-cwd (OBSERVED flip); no existing test's verdict changes and no client test file is touched.
- [ ] **SC-5 (S25)** — `lint-write-path.sh` prints one `STALE` row per matrix row with no matching source/`--bog` slot: default exit **0** with the rows printed, `--strict` exit **1**; on a fully matching matrix no STALE row and exit 0; the uncovered **FAIL exit 1 is unchanged with and without `--strict`**; exit **3** on usage / no-matrix preserved (K20 disjoint); real-tree DashboardPan matrix at `ff1b659` reports the exact observed STALE count (expected 0).
- [ ] **SC-6 (S26)** — after a client build, `git status --porcelain` shows no `build/tmp` or `*.class`; **`git ls-files` for every tracked `build/**/*.jar` is identical before and after** — no tracked jar is untracked; no `vendorVersion` changes.
- [ ] **SC-7 (method)** — every rule-changing PR records an **OBSERVED** mutation flip (verbatim RED-then-GREEN); no PR merges on a "would flip" claim; every smoke pin asserts exact count + subject + absence; a smoke that cannot run is a BLOCKER, never an advisory.
- [ ] **SC-8 (fixtures)** — every S21/S22/S23 fixture carries a comment-only decoy that does NOT satisfy the pin; comments are stripped before matching.
- [ ] **SC-9 (routing)** — `kit-links.bats` green with every touched script routed in both `BUILD-LOOP.md` and `skill/SKILL.md` (K19); the `--strict` flag row added; exit codes remain disjoint (K20); every scanner still prunes dot-dirs (D9b); metric names state what they measure (K14).
- [ ] **SC-10 (process)** — each PR passes lead gate → QA verify → investigador1 second read before merge; K12 holds (each worker wrote only in its own worktree); the four always-conflict files were fragment-merged (both rows kept, deduped by script name).
- [ ] **SC-11 (close)** — `tests/c10-close.bats` green under `C10_CLOSE=1` with BASE `1fb63d6`; `sweep-build-state.sh` and `sweep-fold-audit.sh --strict` green; `retros/INDEX.md` pending = **0**; `VERSION` = **`0.21.0`** with a CHANGELOG entry per CONTRIBUTING §4-5; `shellcheck` exit 0; **no commit body in the whole PR range carries an attribution trailer** (K11).
- [ ] **SC-12 (evidence)** — one `[ev:]` token per paragraph/row across every doc the campaign touches; every load-bearing client cite names its worktree and commit.

---

## 11. Open Gates for Cristian (do NOT block W1)

| Gate | Unblocks | Status |
|---|---|---|
| Merge tunnel PRs #1/#2/#3 (viewer must send `x-config-token`) | P1, P2 | Blessed, awaiting merge (tunnel main `872c64c`) |
| Execute the deploy chain (2.0.7 / 2.0.3 / 2.1.1, then the C9 jars) | every future C10/C11 client jar | Station is four repo versions behind |
| Run the niagaraTest harness session (`qa/c9-harness-runsheet.md`) | C9 gate 14/14; P2/P3/P4 alarm REDs | Runsheet owed to Cristian on `qa/c9-verify-runbooks` `18420d9` |
| P2 need: attribution (R14, already shipped) **or** per-operator screens (new RBAC)? | P2 | One question, two answers |
| Defrost trial green light (rooms 1/2/4) | P3 | — |
| Intercambiador Cuarto 3 on a Niagara output? YES / NO | P4 (YES) or demote the panel control (NO) | — |
| `coolOnSensorFault` station-side link approval | P5 | — |

---

## 12. Review Workload Note

`delivery_strategy` = **auto-chain**.
`Decision needed before apply: No`
`Chained PRs recommended: Yes` — one wave, 7 slices; PR1-PR5 open in parallel off kit `main` and merge with fragment-merge discipline, PR6 is a separate repo, PR7 opens only after PR1-PR5 merge.
`400-line budget risk: Low` — every PR is under 400 changed lines by construction (largest ~220, PR3). No generated code, no client slot block, no `size:exception` expected.

---

## 13. Next Phases

- `sdd-spec` — the three narrowed verdict domains and their row grammar: S21 (class-FIELD + same-enclosing-method-body companion-flag), S22 (per-slot `@NiagaraAction` → `do<Action>()` body-follow, the excluded methods, and the EW10 contract change), S23 (the Pattern B surface token set and the `B<PureClass>` adapter→pure follow); plus the STALE advisory + `--strict` exit contract for `lint-write-path.sh` (uncovered FAIL untouched) and the cwd-independence contract for `run-pure-test.sh`.
- `sdd-design` (parallel with `sdd-spec`) — the awk class-scope field-collector pass and the backward method-body anchor; the S22 **NEW action-aware pass** built in-script on the section-D method-boundary parser (`:250-306`) + the Pass-0 paren-join with an `execute()`/`changed()`/generated-setter scope filter — explicitly NOT a reuse of `lint-silent-protection.sh`, which never reads `@NiagaraAction` `[ev: investigador1 685e7981e]`; the adapter→pure resolution algorithm and its convention limits; the STALE-row detection and its `--strict` promotion beside the untouched uncovered FAIL; fixture layout with comment decoys; the `cd "$rt"` source resolver at `run-pure-test.sh:62`; the 7-PR chain and fragment-merge mechanics.
- Then `sdd-tasks` → `sdd-apply` (RED-first on every code slice) → `sdd-verify` → `sdd-archive`.
