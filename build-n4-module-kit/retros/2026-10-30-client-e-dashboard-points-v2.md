<!-- review-status: pending -->
<!-- module: ClientEDashboard -->
<!-- deltas: 9 -->
# 2026-10-30 · ClientEDashboard · client-e-dashboard-points-v2

**Session**: client module ClientEDashboard (`client trees/client E/dashboard`, local repo, branch `feat/client-e-dashboard-points-v2`,
feature doc `odd/tasks/client-e-dashboard-points-v2.md`), T5 close: build 1.3.0 for Niagara 4.15.6.20 at f93ceae
**Delta count**: 9

## What happened
Points v2 aligned the AHU facade to the client's 53-point list (17 -> 53 slots, 5 removed), added a reset pulse to the
servlet, generated a 734-row multi-level inventory and ported a modular SPA with COB/EGL 3D viewers on the vendored
three r160. RDD review ran per work unit in a detached worktree because large or generated commits exceed the lens
context budget. T5 built on a new 4.15.6.20 mirror (`build.sh --plugin-version 7.6.22 --target-version 4.15`):
BUILD SUCCESSFUL, major 52, signed, verify-module 28 PASS / 0 FAIL. The close exposed two silent-pass channels in the
kit (schema-risk masks removals by name across types; report-module reports a clean servlet as "no BWebServlet")
and a lexicon gap the existing wiring map hid.

## Evidence
- RDD whole-branch and data-slice STARTs refused `lens_context_budget_exceeded` (T2 branch, T4a single commit, T4b
  `sets.js` slices 0.5-1.7 MB single-line geometry, fix commits c81c477/109eb12 carrying a regenerated preview
  `points.json`); the units passed one commit at a time in a detached worktree. `[ev: feature doc client-e-dashboard-points-v2.md Progress 2026-10-30]`
- T4a slice 4f1618c: native review stopped `corrupted_or_unverifiable_authority` after capture, the reoffered START
  refused `candidate_context_unavailable`; user chose report-and-continue, occurrence comment on gentle-ai#4704. `[ev: gentle-ai#4704]`
- schema-risk before (81c915d) / after (f93ceae) on source snapshots: rt LOSSY with 3 rows
  (`chwValve`, `filterDp`, `totalAirflow`) attributed to `ClientEDashboardService`; `runHours` and `energy` were removed
  from `BAhuPanel` but still exist on `BExhaustPanel`, so no row; ux SAFE. `[ev: commit f93ceae]`
- report-module `ClientEDashboard-ux SKIP lint-servlet no BWebServlet` while `BClientEDashboardServlet extends BWebServlet`
  (`BClientEDashboardServlet.java:73`); `lint-servlet.sh <ux>/src` exit 0 with no output. `go/internal/reportmodule/special.go:292-310`
  treats "no row and no INFO line" as SKIP. `[ev: commit 08cfed4]`
- `slot-coverage.sh per-slot` on rt: 86 DOTTED rows (`AhuPanel.<slot>` keys only), 0 bare keys, exit 0; type-set mode
  exit 0 with 4 WARN; ux type-set `pct=0.0 missing=ClientEDashboardServlet` exit 0. The committed wiring map listed the
  Spanish lexicon values as "Workbench display name". `[ev: commit f93ceae]`
- `verify-module.sh --target-version 4.15 --src ClientEDashboard --plano <index.html> <jars>` exit 2 "unknown flag
  --plano"; `--plano` is a leading mode (`go/internal/verifymodule/command.go:51-57`), not a gate flag. `[ev: commit f93ceae]`
- `rc-scan.sh --profile lan <dir>` exit 3 "unknown option: lan"; `rc-scan.sh <dir> --profile lan` exit 0 (0 FAIL, 153 WARN). `[ev: commit f93ceae]`
- No 4.15.6.20 mirror existed; `mirror-niagara-home.sh` built one (733 jars linked). build.sh printed "signed with a
  throwaway dev key ... do not deploy". `[ev: commit 08cfed4]`
- three r160 shim (T4b 0d17e40): r147 `outputEncoding`/`encoding` -> `outputColorSpace`/`colorSpace`;
  `useLegacyLights = true` to keep r147 intensities (prints a deprecation warning in r160);
  `BufferGeometryUtils.mergeBufferGeometries` renamed `mergeGeometries` and the addon had to be vendored separately. `[ev: commit 0d17e40]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | schema-risk keys slots per declaring type (`<Type>.<slot>` from the class, not the first `<type>` of module-include.xml): a slot removed from one type while another type keeps the name must report LOSSY; add a fixture with two types sharing a slot name | `build-verify.md` § `Schema-risk deploy workflow` (+ `go/internal/schemarisk`) | `[ev: commit f93ceae]` |
| Δ2 | report-module lint-servlet: decide SKIP from the source scan (a class extending BWebServlet), not from the absence of INFO rows; a clean servlet is PASS | `toolbelt/report-module.sh` (`go/internal/reportmodule/special.go` servlet) | `[ev: commit 08cfed4]` |
| Δ3 | slot-coverage per-slot: all-DOTTED with zero bare keys is a WARN (exit 1 under --strict) naming the Workbench consequence; a wiring-map "Workbench display name" column must come from bare keys or the friendly slot name | `BUILD-LOOP.md` § `5. Verify gate (before "done")` | `[ev: commit f93ceae]` |
| Δ4 | RDD on generated or large data: commit generated files (inventory tables, geometry, regenerated preview fixtures) alone, so code commits stay reviewable; a refused data-only slice takes the off path (assess + parent structural spot check) and is recorded | `ORCHESTRATION.md` § `3.d Shared working-tree git discipline` | `[ev: feature doc client-e-dashboard-points-v2.md Progress 2026-10-30]` |
| Δ5 | Review per work unit runs in a detached worktree with `--base-ref <parent>`; record the occurrence (provider defect gentle-ai#4704: `corrupted_or_unverifiable_authority` then `candidate_context_unavailable`) in the incident journal | `ORCHESTRATION.md` § `9. Incident journal` | `[ev: gentle-ai#4704]` |
| Δ6 | SKILL step 5 and References: `--plano` is a leading mode (`verify-module.sh --plano <index.html>`), not a flag of the jar gate | `skill/SKILL.md` § `Execution Steps` | `[ev: commit f93ceae]` |
| Δ7 | rc-scan accepts options before the artifact dir (or its usage error says the dir comes first) | `toolbelt/rc-scan.sh` | `[ev: commit f93ceae]` |
| Δ8 | three r160 port card: outputEncoding/encoding -> colorSpace, `useLegacyLights = true` keeps r147 light intensities (deprecation warning expected), `mergeBufferGeometries` -> `mergeGeometries` via a global shim, BufferGeometryUtils vendored as its own addon file | `types/frontend-standard.md` § `Vendor catalog` | `[ev: commit 0d17e40]` |
| Δ9 | build.sh on a mirror signs with a throwaway dev key: the feature doc's install step must go through ng-deploy (station signing), never copy build/libs jars | `BUILD-LOOP.md` § `5. Verify gate (before "done")` | `[ev: commit 08cfed4]` |

## Lessons
- A name-keyed diff across several types can hide a removal; a verdict that matches the expectation is not proof that every expected row appeared - count the rows.
- A SKIP derived from "no output" is a silent-pass channel; detect applicability from the input, not from the lint's report.
- Generated data and code belong in separate commits: review budgets, diffs and bisects all work on the code commit.
- A wiring map's display-name column must match what Workbench shows, which depends on bare lexicon keys.
