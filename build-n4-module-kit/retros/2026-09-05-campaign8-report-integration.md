<!-- review-status: folded -->
# Retro: Campaign 8 — report-module.sh integration + v0.19.0 (Campaign 8 PR8)

**Date:** 2026-09-05
**Change:** build-n4-module-campaign8 PR8 — `feat/c8-report-integration`
**Scope:** `build-n4-module-kit/toolbelt/report-module.sh` (extended); `build-n4-module-kit/toolbelt/schema-risk.sh` (dot-dir fix); `VERSION`, `CHANGELOG.md`, `BUILD-LOOP.md`, `skill/SKILL.md`
**Lead:** Campaign 8 PR8 executor
**RED tip:** `d8322d4` (RM4–RM6b)

[ev: corpus B801] [ev: retro campaign8-report-integration]

---

## What shipped

| Member | Integration | SKIP condition |
|--------|-------------|----------------|
| `lint-delays.sh` | per artifact: `lint-delays.sh <artifact>/src`; FAIL rows aggregate to exit 1 | no `<artifact>/src` |
| `schema-risk.sh` | per artifact: `schema-risk.sh <artifact>/.deploy-baseline <artifact>`; exit 2 (OUTAGE) → FAIL row (D9a) | no `.deploy-baseline/` |
| `triage-console.sh` | once per run: `--console-dir <dir>`; rows pass-through to report output | `--console-dir` absent → `(run)  SKIP  triage-console  no --console-dir` |

**Exit map for schema-risk (D9a):** `0` → PASS (SAFE), `1` → WARN (LOSSY), `2` → **FAIL** (OUTAGE, never ERROR), `3/4` → ERROR + exit 3.

**D9b dot-dir fix in schema-risk.sh:** `parse_slots` changed from `find "$dir" -name "*.java"` to `find "$dir" -mindepth 1 -type d -name '.*' -prune -o -name "*.java" -print`. The `-mindepth 1` prevents the root from being pruned when it is itself a hidden directory (e.g. `.deploy-baseline/` passed as `before-dir`); subdirectory pruning removes `.deploy-baseline/` when it is nested inside the `after-dir` artifact.

**VERSION bumped to 0.19.0.** `CHANGELOG.md` `## [Unreleased]` renamed to `## [v0.19.0] - 2026-09-05`; `--console-dir` entry added; `### References` block added; fresh empty `## [Unreleased]` left above.

---

## TDD evidence

| Phase | Evidence |
|-------|----------|
| RED | Commit `d8322d4` on `feat/c8-report-integration`: RM4, RM5a, RM5b, RM6a, RM6b all failing because report-module.sh had no `--console-dir` flag and no lint-delays/schema-risk integration. |
| GREEN | Extended report-module.sh with lint-delays (step 5), schema-risk (step 6), triage-console (step 7). Fixed console-dir/console.txt fixture to use `com.angeles.BDefrost` (own-frame detectable). Fixed schema-risk.sh dot-dir pruning with `-mindepth 1`. All 8 RM tests green. |
| REFACTOR | Replaced `sr_out=$(...)` capture (unused, SC2034) with direct `>/dev/null 2>&1` redirect since schema-risk verdict is fully carried by its exit code; the individual change rows are intentionally suppressed in the report (only the aggregate verdict row is emitted). shellcheck exit 0. |

Full suite: **239/239** green (was 234 before PR8; +5 new pins RM4–RM6b, RM5a, RM5b).

---

## Named mutations

### (a) Drop lint-delays aggregation → RM4 flips (R8.5)

**Mutation:** removed the entire lint-delays invocation and row-emission block from the artifact loop.

**Observed output on `delays` fixture:**
```
DemoPan-rt  PASS  dup-keys  0
DemoPan-rt  WARN  slot-coverage  %
DemoPan-rt  SKIP  schema-risk  no .deploy-baseline
(run)  SKIP  triage-console  no --console-dir
report-module: 1 artifact · 1 PASS · 0 FAIL · 1 WARN · 2 SKIP  ->  CLEAN
mutant-exit=0
```
**Flip:** RM4 fails because `[ "$status" -eq 1 ]` receives 0; no FAIL + no "delay" in output.

### (b) Map schema-risk exit 2 to ERROR instead of FAIL → RM6b flips (D9a)

**Mutation:** changed `2) emit "$ANAME" FAIL schema-risk "verdict=OUTAGE"` to `2) emit "$ANAME" ERROR schema-risk "env fault (exit 2)"; HAD_ENV=1`.

**Observed output on `schema-outage` fixture:**
```
DemoPan-rt  ERROR  schema-risk  env fault (exit 2)
(run)  SKIP  triage-console  no --console-dir
report-module: 1 artifact · 0 PASS · 0 FAIL · 0 WARN · 1 SKIP  ->  CLEAN
mutant-exit=3
```
**Flip:** RM6b fails at `[ "$status" -eq 1 ]` (gets 3) AND `[[ "$output" != *"ERROR"* ]]` (ERROR appears).

### (c) Drop once-per-run triage call → RM5b flips

**Mutation:** replaced the `triage-console.sh` invocation in the `--console-dir` branch with a no-op SKIP row.

**Observed output on `clean` fixture with `--console-dir console-dir`** (without the call, only artifact rows appear):
```
DemoPan-rt  PASS  dup-keys  0
DemoPan-rt  PASS  slot-coverage  100.0%
DemoPan-rt  SKIP  schema-risk  no .deploy-baseline
report-module: 1 artifact · 2 PASS · 0 FAIL · 0 WARN · 2 SKIP  ->  CLEAN
```
**Flip:** RM5b fails at `[[ "$output" == *"triage-console"* ]]` (string absent); "time <= 0" and "BDefrost" also absent.

---

## Design deviations

### D1: console.txt fixture used `com.x.BDefrost` — own-frame undetectable by default

The RED commit's `tests/fixtures/report-module/console-dir/console.txt` had `com.x.BDefrost.arm(BDefrost.java:8)`. With triage-console.sh's default `PKG=com.angeles`, the `com.x` frame did not match C1 (own-frame). The fixture was updated to `com.angeles.BDefrost.arm(BDefrost.java:8)` so triage-console detects the trace. RM5b test comment says "surfaces the own-frame trace" — the fixture was a placeholder.

**Impact:** fixture changed; RM5b still asserts the design contract (`triage-console` + `time <= 0`/`BDefrost`). No spec change.

### D2: schema-risk.sh parse_slots dot-dir fix required `-mindepth 1`

Initial pruning fix used `find "$dir" -type d -name '.*' -prune -o -name "*.java" -print`. This pruned the root when `dir = .deploy-baseline/` (the root itself is a hidden directory), causing the before-snapshot to return zero java files. Fixed with `-mindepth 1` so only subdirectories are eligible for pruning, not the root.

**Lesson:** `-type d -name '.*' -prune` on a hidden root dir prunes itself. When the root might be a hidden directory, always add `-mindepth 1` [ev: corpus B801].

---

## Real smoke

### Without `--console-dir` (pre-fix ColdRoomPan-rt at 4f5f1c7)

```
report-module.sh /home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato/Paccadia/ColdRoomPan
```

Key rows (abridged):
```
ColdRoomPan-rt  FAIL  timer-ticket  BEvaporatorUnit.java: schedules a Clock ticket but stopped() does not cancel it
ColdRoomPan-rt  FAIL  companion-flag  BEvaporatorUnit.java: flag 'startingUp' set beside Clock.schedule* not cleared in stopped()/started()
ColdRoomPan-rt  FAIL  lint-delays  BDefrostController.java:556  zero-floor  local d=Math.max(.,k) k=0 (<=0)
ColdRoomPan-rt  FAIL  lint-delays  BDefrostController.java:566  facet-min-zero  slot interval facet MIN=0 — schedule can receive 0
ColdRoomPan-rt  FAIL  lint-delays  BDefrostController.java:620  facet-min-zero  slot duration facet MIN=0 — schedule can receive 0
ColdRoomPan-rt  FAIL  lint-delays  BDefrostController.java:664  facet-min-zero  slot staggerDelay facet MIN=0 — schedule can receive 0
ColdRoomPan-rt  SKIP  schema-risk  no .deploy-baseline
(run)  SKIP  triage-console  no --console-dir
report-module: 1 artifact · 9 PASS · 6 FAIL · 12 WARN · 4 SKIP  ->  ISSUES
exit=1
```

**Result:** exits 1 (ISSUES). Aggregate FAIL row for lint-delays confirmed at :556/:566/:620/:664 (matches SC1/RM4 contract).

### With `--console-dir` (HoneywellMX605132026 real console dir)

```
report-module.sh ColdRoomPan --console-dir /mnt/c/Users/equipo/Niagara4.14/OptimizerSupervisor/stations/HoneywellMX605132026
```

Triage rows:
```
FAIL  triage-console  HoneywellMX605132026  1x 06:41:20 15-jun-26 CST -> 06:41:20 15-jun-26 CST  SEVERE NullPointerException @ [sys]
FAIL  triage-console  HoneywellMX605132026  1x 14:45:16 06-jun-26 CST -> 14:45:16 06-jun-26 CST  WARNING BChiDashboardService: cannot force-load ChiAlarmHelper (ack probe will run lazily on first endpoint hit): chihuahua-rt:com.angeles.chihuahua.ux.ChiAlarmHelper @ [chihuahua]
FAIL  triage-console  HoneywellMX605132026  9x 10:10:31 03-jun-26 CST -> 18:19:57 13-jun-26 CST  WARNING BChiDashboardService: failed to unschedule controlTick: access denied ("java.lang.RuntimePermission" "modifyThread") @ [chihuahua]
report-module: 1 artifact · 9 PASS · 9 FAIL · 12 WARN · 3 SKIP  ->  ISSUES
exit=1
```

**Result:** triage-console FAIL rows surface from the real station console — NullPointerException via [sys] + chihuahua access-denied warnings (pre-existing known issues).
