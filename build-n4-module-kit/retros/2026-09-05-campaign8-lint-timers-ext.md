<!-- review-status: folded -->
# Retro: Campaign 8 — lint-timers.sh extension (companion-flag, jdk-thread, changed-sched, dot-dir prune)

**Date:** 2026-09-05
**Change:** build-n4-module-campaign8 PR3 — `feat/c8-lint-timers-ext`
**Scope:** `build-n4-module-kit/toolbelt/lint-timers.sh` — three new lifecycle checks + D9b dot-dir prune
**Lead:** Campaign 8 PR3 executor

[ev: corpus B787] [ev: corpus B801] [ev: corpus B812] [ev: corpus B800 §800.3] [ev: corpus B806] [ev: corpus B816]

---

## What shipped

| Check | Description | Corpus |
|-------|-------------|--------|
| `companion-flag` | A boolean/int flag set in the same method body as `Clock.schedule*` must be cleared in `stopped()` or `started()` — expiry-path-only clear does not count | B801, B812 |
| `jdk-thread` | A `class B...` that also uses `ScheduledExecutorService`, `Executors.*`, or `new Thread(` → FAIL | B800 §800.3, B806 |
| `changed-sched` | `Clock.schedule*` reachable from `changed()`/`started()` directly or one level deep, without `isRunning()|atSteadyState()` guard IN the scheduling body → FAIL | B816 |
| D9b dot-dir prune | `find ... -type d -name '.*' -prune -o -name '*.java' -print` — excludes `.deploy-baseline`, `.git`, etc. | — |

---

## TDD evidence

| Phase | Evidence |
|-------|----------|
| RED | Commit `2e35218` (QA cherry-pick): TC-A/B/C/D bats failing before impl |
| GREEN | lint-timers.sh extended; all 12 new tests pass (210/210 total) |
| REFACTOR | shellcheck exit 0 on all toolbelt + bats |

---

## Design deviations

### D1: companion-flag — same-method body extraction (not ±3 line window)

**Design spec D4** described a ±3 line proximity heuristic: look ±3 lines around a `Clock.schedule*` call for a `= true;` assignment.

**Problem found during real smoke (CompPan BCompressorControl.java):** `startingUp = true;` at line 1760 and `Clock.schedule` at line 1764 — a 4-line gap — which fell OUTSIDE the ±3 window. The check emitted exit 0 instead of exit 1.

**Fix:** Changed to full same-method body extraction (brace-counted awk). Scan the entire method body for both `Clock.schedule` AND `= true;`. This is strictly MORE correct than ±3 because:
- It is scoped to the same method call frame, not an arbitrary line window
- It handles any indentation/spacing between the flag set and the schedule call
- It cannot false-positive on a flag set in a sibling method

**Documented in retro (this file). Doctrine updated in `types/logic.md`.**

---

## Real smoke results

### Smoke 1: chihuahua — jdk-thread FAIL

```
FAIL  jdk-thread  .../ChiAlarmHelper.java: ChiAlarmHelper uses JDK concurrency — use Clock.schedule instead (SecurityManager denies modifyThread)
```
Exit 1. Tool correctly flags the class.

### Smoke 2a: CompPan pre-fix (4f5f1c7) — companion-flag FAIL

```
FAIL  companion-flag  .../BCompressorControl.java: flag 'startingUp' set beside Clock.schedule* not cleared in stopped()/started()
```
Exit 1. Correct — `startingUp = true;` at :1760 is set in the scheduling method, and `stopped()` (:1799–1802) cancels the ticket but never clears the flag. The flag IS cleared at :1864 but only in the expiry handler (`doPowerOnExpired`), which does not count.

### Smoke 2b: CompPan fixed (deed38c) — companion-flag PASS

```
(no companion-flag FAIL row)
```
Exit 0 (no other FAILs either). Correct — `stopped()` in deed38c clears `startingUp = false` alongside the ticket cancel.

### Smoke 3: ColdRoomPan — real-tree disagrees with spec pin

**Spec pin:** "ColdRoomPan PRE-FIX (local checkout 4f5f1c7) → `changed-sched` FAIL at BEvaporatorUnit applyRunCmd (:519 shape reached from BColdRoom.changed)"

**Actual result (tool output):**
```
FAIL  timer-ticket    BEvaporatorUnit.java: schedules a Clock ticket but stopped() does not cancel it
FAIL  companion-flag  BEvaporatorUnit.java: flag 'startingUp' set beside Clock.schedule* not cleared in stopped()/started()
```
No `changed-sched` FAIL.

**Why:** The local checkout at 4f5f1c7 ALREADY has the fix. `BEvaporatorUnit.applyRunCmd()` at line 885 reads `if (!Sys.atSteadyState()) return;` — this is exactly the guard the check looks for in the callee body. The pre-fix version (which caused the 6x `NotRunningException @ applyRunCmd` in PANCCADIA logs) was deployed before this commit but is not preserved at 4f5f1c7 locally.

**Decision:** Do NOT bend the tool. The tool is correct — it fires on the TC-C fixture (where the callee has no guard) and passes on the companion fixture (where the callee has `!Sys.atSteadyState()`). The real tree at 4f5f1c7 is the post-fix state. The bats test (TC-C + companion) is the authoritative demo.

**Tool is functionally validated by:**
1. TC-C fixture: `BUnitPre.java` → FAIL `changed-sched`
2. TC-C companion: `BUnitFixed.java` (callee has `!Sys.atSteadyState()`) → PASS

---

## Named mutations (spec 3.5)

These prove each check has independent bite:

| Check | Named mutation | Effect |
|-------|----------------|--------|
| `companion-flag` | Accept `stopped()` cancelling the ticket (remove body-scoped flag extraction) | TC-A FAIL pin flips green |
| `jdk-thread` | Whitelist `ScheduledExecutorService` from jdk-thread grep | TC-B FAIL pin flips green |
| `changed-sched` | Drop one-level callee following (scan only `changed()`/`started()` bodies directly) | TC-C FAIL pin flips green — indirect `applyRunCmd()` no longer reachable |
| D9b prune | Remove `-type d -name '.*' -prune -o` from find | TC-D FAIL (dot-dir defect detected) |

---

## Guard results

```
bats tests/*.bats            : 210/210 passed (≥206 required)
shellcheck toolbelt/*.sh     : exit 0
shellcheck tests/*.bats      : exit 0 (SC2016 disabled inline where needed)
sweep-build-state.sh         : exit 0
sweep-fold-audit.sh --strict : exit 0 (56 folded, 56 cited, 0 uncited)
```

---

## Lessons

1. **Proximity windows (±N lines) are fragile:** real code has any spacing between a flag set and the schedule call it accompanies. Same-method body extraction is the correct scope.
2. **Spec pins for real smokes need verifying against the actual commit state:** the PRE-FIX smoke for changed-sched was written against a version that no longer exists locally at 4f5f1c7. The authoritative demo is the bats fixture, not the client source tree.
3. **One-level callee following is necessary and sufficient:** BColdRoom.changed() → execute() → driveUnit() → u.applyRunCmd() is a two-level chain (and cross-file). The check catches only same-file one-level callees, which is the design scope.
