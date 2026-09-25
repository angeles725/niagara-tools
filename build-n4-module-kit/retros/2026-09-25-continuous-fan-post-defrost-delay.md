<!-- review-status: pending -->
# 2026-09-25 · ColdRoomPan · continuous-fan-post-defrost-delay

**Session**: `Cliente/panccadia-leon` branch `fix/continuous-fan-post-defrost-delay` (from 1d016bc) — bounded fix + version bump for a live PANCCADIA field report.
**Delta count**: 2

## What happened
Live PANCCADIA Cuarto 3 trend data (2026-09-25) showed the evaporator valve and fan
re-energizing in the SAME engine scan right after a defrost (+drip) cycle ended, on units
with `fanRunMode=continuous`. This is the exact class of bug the 2026-09-24
restart-sequencing fix (`ColdRoomControl.restartFanGate` / `restartReleaseAssertsFan`,
opened via `beginRestartSequencing()`) had already solved for a STATION RESTART — but
`exitDefrost()` (and `endDrip()`, which calls it) never opened that window before
re-applying outputs, so the same-scan continuous-fan asserts in `applyRunCmd()` /
`applyFanRunMode()` were never gated on a defrost exit. Fix: `exitDefrost()` now calls
`beginRestartSequencing()` before `applyRunCmd()`/`applyFanRunMode()`, reusing the
existing gate/window unchanged. Also gated `applyHoaOutputs()`'s continuous AUTO branch
(a third same-scan assert site the 2026-09-24 fix had not touched, since it isn't reached
during a boot restart) so an operator HOA toggle during the window can't bypass the hold.

## Evidence
- Live oBIX 2026-09-25: ColdRoom_3 EvaporatorUnit_1/2 `fanRunMode=continuous`,
  `startDelay=PT1M30S`, `dripTime=PT5M0S` — the field report that triggered this session.
- `BEvaporatorUnit.exitDefrost()` pre-fix: called `applyRunCmd()`/`applyFanRunMode()`
  directly with no `beginRestartSequencing()` call — the restart-only gate covered ONE of
  the (at least) two release points that need it. `[ev: commit 35f84ca]`
- New source-structural test `PostDefrostFanDelayTest` (2 tests): asserts `exitDefrost()`'s
  call order and `applyHoaOutputs()`'s gate usage by reading `BEvaporatorUnit.java` as text
  (same pattern as `DashboardReaderSlotsTest` — `BEvaporatorUnit` needs the NRE to load, so
  no BComponent-level behavioral test is possible in WSL JUnit). RED observed (2 failures)
  before the fix, GREEN (`OK (2 tests)`) after; two named mutations
  (`T1-exitDefrost-bite`, `T1-applyHoaOutputs-bite`) applied and reverted, each confirmed to
  bite. `[ev: commit 35f84ca]`
- `schema-risk.sh` (1d016bc snapshot vs. working tree) -> `verdict=SAFE`, no slot change.
- `verify-module.sh`/`report-module.sh` via `toolbelt/build.sh`: verify gate ALL PASS (11
  PASS/0 FAIL/3 SKIP/23 WARN, all WARNs pre-existing); `report-module.sh`'s one FAIL
  (`lint-structure` — absolute host path in `gradle.properties`) confirmed pre-existing via
  `git diff 1d016bc -- Paccadia/gradle.properties` (zero diff).
- `preflight.sh` plugin-pin FAILed on a healthy environment: `settings.gradle.kts`'s
  `niagaraPluginVersion` default is `7.6.17`, but this repo's configured `niagara_home`
  (`PowerB-4.15.3.28`) ships ONLY `7.6.22` in `etc/m2` — the check reads the settings.gradle
  DEFAULT literal, not an actual `--plugin-version` override, so it can never pass for a repo
  whose default is stale relative to its own `niagara_home`. Worked around with
  `--no-preflight --plugin-version 7.6.22`.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | `preflight.sh`'s plugin-pin check should accept an optional `--plugin-version` override (or read the same `-PniagaraPluginVersion`/`NIAGARA_PLUGIN_VERSION` source `build.sh` forwards) instead of only ever re-deriving the version from `settings.gradle.kts`'s literal default, so a repo whose default has drifted from its own configured `niagara_home` doesn't FAIL preflight on an otherwise-healthy environment. | `toolbelt/preflight.sh` §plugin-pin | `[ev: commit 35f84ca]` |
| 2 | BUILD-LOOP.md §0.a / METHODOLOGY.md could name the "restart-only gate" failure class explicitly: a same-scan-assert gate opened at one release point (station restart) is not automatically open at every OTHER release point that re-applies the same outputs (here, a defrost/drip exit) — audit every caller of the guarded methods, not just the one that motivated the original fix. | `types/logic.md` §Safety fail-modes | `[ev: commit 35f84ca]` |

## Lessons
- A same-scan-assert gate fixed for ONE release point (station restart) does not
  automatically cover every OTHER caller that re-applies the same outputs (defrost/drip
  exit here) — audit all call sites of the guarded methods when a timing gate is added, not
  just the one in the original bug report.
- Reusing an existing pure gate/window (`beginRestartSequencing`/`restartFanGate`/
  `restartReleaseAssertsFan`) for a second, unrelated release point needed zero new pure
  functions — only a call-site + call-order change plus source-structural tests pinning the
  order.
- `preflight.sh`'s plugin-pin check can FAIL on a healthy environment when a repo's
  `settings.gradle.kts` default plugin version has drifted from its own `niagara_home` (a
  Windows path already resolved via `/mnt/c/...`) — `--no-preflight --plugin-version <v>` is
  the correct workaround, not evidence of a broken build.
- `report-module.sh`/`lint-structure`'s FAIL on a tracked `gradle.properties` absolute host
  path is a REPO-level, pre-existing condition (confirmed via `git diff <base-commit>`) —
  always diff a lint FAIL against the base commit before treating it as caused by the
  current change.
- BUILD-STATE.md's tracked `ColdRoomPan` envelope was stale (still pointing at the retired
  `Cliente/Leon-Guanjuato` tree) before this session updated it — a module-repo rename/move
  needs its `module_root`/`module_repo` fields corrected in the same session that notices
  the drift, not left for a future orient to rediscover.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-25-continuous-fan-post-defrost-delay.md | ColdRoomPan | 2026-09-25 | pending | 2 |`
