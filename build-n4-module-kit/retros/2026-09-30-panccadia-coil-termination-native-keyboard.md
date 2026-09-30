<!-- review-status: pending -->
# 2026-09-30 · DashboardPan · panccadia-coil-termination-native-keyboard

**Session**: PANCCADIA — Cliente/panccadia-leon feat/coil-termination-native-keyboard (base d7b07bd, HEAD c35ffaf): ColdRoomPan 2.4.2 (coil-only defrost cut-out) + DashboardPan 2.9.2 (termination-sensor choice removed, native OS keyboard); post-hoc gate
**Delta count**: 6

## What happened
Two changes on the panccadia-leon branch: defrost temperature termination now always evaluates `coilTemp` (slot `terminationSensor` and the facade `evap1..3TerminationSensor` removed), and the custom on-screen keypad/login keyboard of the HMI dashboard was replaced by native `inputmode="decimal"` inputs. The work was delegated to writers with strict TDD and built/packaged (Downloads PANCCADIA-modulos-2026-09-30, nothing installed). Process defects surfaced along the way: the `/build-n4-module` skill was NOT loaded before delegating, so writers ran toolbelt scripts from an outdated kit worktree (`niagara-tools-worktrees/c8-struct`, campaign 8) whose `build.sh` rejects `--no-drift-check`; a writer ran `pkill -f gradle` on a stuck foreground build (killed its own shell and possibly peer sessions' daemons); a writer swept an untracked manual (HTML + PDF) into a code-fix commit; the native RDD review of the whole branch failed with `lens_context_budget_exceeded`; and review R3-002 found a real defect (a poll-driven re-render overwrote a focused input). This retro was written post-hoc by the gate run.

## Evidence
- Skill not loaded, non-canonical kit used: writer invoked `run-pure-test.sh` / `build.sh` from `niagara-tools-worktrees/c8-struct`; that `build.sh` rejected `--no-drift-check` (present in the canonical kit) `[ev: odd/tasks/coil-termination-native-keyboard.md T4]`
- `pkill -f gradle` from a writer to stop a stuck foreground build: killed the writer's own shell; other sessions' Gradle daemons exposed (parent build of DashboardPan was running concurrently) `[ev: session 2026-09-30]`
- RDD review of the full branch refused with `lens_context_budget_exceeded` (1821 changed lines incl. a 787-line HTML manual + ~700 KB PDF); resolved by reviewing per task through worktrees (slices) `[ev: gentle-ai review 2026-09-30]`
- Untracked manual (HTML + PDF) committed inside code-fix commit `[ev: a5d9471]` (docs later touched again in `[ev: 9f48f0d]`, `[ev: 3bc54cd]`)
- R3-002: `.stp-val` became an editable native input; the periodic `prefillSetpoints` re-render only skipped dirty items, so a focused input was overwritten while typing; fixed by skipping the focused input `[ev: 74e84b7]`
- Chrome-83 touch behaviour of the login-before-focus gate (mousedown/pointerdown preventDefault + focusin blur) is proven only structurally (srcTest reads index.html; the harness cannot run JS) `[ev: b146f82]` `[ev: odd/tasks/coil-termination-native-keyboard.md T3]`
- Gate 2026-09-30 (HEAD c35ffaf): all rt lints exit 0 (WARN-only: lint-status-parity BEvaporatorUnit / BRoomPanel, ext-writable-shape x4 BRoomPanel setpoints); schema-risk LOSSY on both rt modules = the intended slot removals; every FAIL (lint-structure L10 gradle.properties, DashboardPan-wb L6/L9, rc-scan host-literal index.html:771, slot-coverage per-slot MISSING) also present on base d7b07bd, none introduced `[ev: gate run 2026-09-30]`
- Dev-mirror builds: ColdRoomPan verify-module 12/0, DashboardPan rt+ux 23/0 `[ev: c9473f7]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Canonical-kit guard: every toolbelt script (and `run-pure-test.sh`/`build.sh` first) resolves its own kit root and WARNs/FAILs when that root is under `niagara-tools-worktrees/` or is not the canonical checkout (or differs from `$KIT`); delegation prompts must pass `KIT=<canonical>` and the launcher must be loaded (`/build-n4-module`) BEFORE delegating any N4 implementation. | `toolbelt/lib` (new `kit-root-guard`) + `ORCHESTRATION.md` § delegation | `[ev: c8-struct worktree build.sh rejected --no-drift-check]` |
| Δ2 | Long builds run in the background (`run_in_background` + log + poll), never as a foreground call; `build.sh`/`fast-build.sh` should print a "run in background" banner and the exact PID to stop (`gradle --stop` scoped to the project), and the writer brief forbids broad `pkill -f gradle|java` (kills peer sessions' daemons and the agent's own shell). | `BUILD-LOOP.md` § build + `ORCHESTRATION.md` § writer brief | `[ev: session 2026-09-30 pkill -f gradle]` |
| Δ3 | Generated docs (HTML manual, PDF) are committed separately from code: a writer brief must say "add own paths only, never `git add -A`/`-a`" and the commit step checks `git diff --cached --stat` for non-code artifacts > N KB; keeps review slices under the native review lens budget (`lens_context_budget_exceeded` at 1821 lines incl. a 787-line HTML + 700 KB PDF). | `ORCHESTRATION.md` § commit discipline + `types/dashboard.md` § docs | `[ev: a5d9471]` |
| Δ4 | Any poll-driven re-render of an editable dashboard view must skip the element that has focus (`document.activeElement`) as well as dirty items; add an `rc-scan.sh` WARN for a periodic render function that assigns `.value` on inputs without an `activeElement` guard, and a srcTest pin in the dashboard type guide. | `types/dashboard.md` § polling/render + `toolbelt/rc-scan.sh` | `[ev: 74e84b7]` |
| Δ5 | HMI touch-gate proof: a structural srcTest cannot prove focus/keyboard behaviour on the panel; the verify step for any input-focus change must include a real Chromium 83 CDP touch run (as done for disabled buttons) or a checklist row "panel check pending" recorded in BUILD-STATE `open_issues` until a human confirms on the HMI. | `types/dashboard.md` § HMI verify + `BUILD-LOOP.md` § verify | `[ev: b146f82]` |
| Δ6 | Gate hygiene: `lint-lexicon-ascii.sh` takes a module-root (not a lexicon file) and `lint-write-path.sh` exits 3 (no matrix) for profile roots — document the expected arguments in the launcher step 5 lint list (`<module-root>` = profile dir for lexicon-ascii; matrix only at `Dashboard/docs`), and have `report-module.sh` be the single aggregate entry so post-hoc gates do not hand-roll arg shapes; `lint-timers.sh` takes >2 min on DashboardPan-rt sources (perf). | `SKILL.md` step 5 + `toolbelt/lint-timers.sh` | `[ev: gate run 2026-09-30]` |

## Lessons
- Load the build skill before delegating; a writer told nothing about the kit will find an old copy of it.
- Never broad-`pkill`; run long builds in the background and stop only the exact PID.
- Commit generated docs/PDF apart from code so native review slices stay inside the lens budget.
- A native (editable) input on a polled page needs a focus guard in every periodic re-render.
- Structural tests do not prove touch/keyboard behaviour on the panel; record the HMI check as owed.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-30-panccadia-coil-termination-native-keyboard.md | DashboardPan | 2026-09-30 | pending | 6 |`
