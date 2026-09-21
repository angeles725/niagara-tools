<!-- review-status: folded -->
# 2026-09-20 · kit · report-module orphan-lint wiring (automation audit fix, PR-A)

**Session**: automation audit (operator: module creation must be AUTOMATIC — no mechanical check may depend on the agent REMEMBERING to run it; the research-sdd flaw). Audit found 8 static-source lints existed but were NOT wired into `report-module.sh` (they lived only as BUILD-LOOP §5 checklist prose), plus `report-module.sh`/`preflight.sh` are invoked by no orchestrator. This PR closes the first half: wire the static-source orphans into the aggregator so ONE `report-module.sh` run catches them.

**Delta count**: 0 (implementation retro, not a proposal — no propose-never-apply deltas)

## What was done
Wired 9 lint invocations into `toolbelt/report-module.sh`, mirroring the existing §5.7 (per-artifact) and §8 (module-once) patterns, each with SKIP-if-no-src and ERROR-on-exit-3 handling:
- Per-artifact `*-rt`+src: §5.14 lint-subscribe-without-unsubscribe (WARN), §5.15 lint-recovery-path (FAIL), §5.16 lint-config-sanity (CS1/CS2 FAIL, CS3 WARN), §5.17 lint-status-parity (WARN).
- Per-artifact `*-ux`: §5.18 lint-servlet (FAIL; self-SKIPs when no BWebServlet), §5.19 rc-scan (FAIL; only when src/rc exists).
- Per-artifact `*-wb`+src: §5.20 lint-wb-threading (WARN).
- Module-once on $MODULE_ROOT: §10 lint-structure (FAIL), §11 lint-write-path (FAIL on uncovered; STALE/DRIFT → WARN; exit 3 = no matrix → SKIP).
Tests RM12–RM20 added to `tests/report-module.bats` (22/22). BUILD-LOOP §5 report-module line updated to note the newly-composed lints.

## Bug fixed en route
§11 initially mapped `lint-write-path` exit 3 (no `docs/write-path-matrix.md`) to ERROR/HAD_ENV, which made every fixture exit 3. A missing matrix is a legitimate SKIP, not an env fault — fixed. Also seeded two missing test fixtures (a `module.palette`, a clean `BHelper.java`) so unrelated lints didn't exit-3 on empty fixtures.

## Scope note
`lint-guard-pins` deliberately NOT wired here — it is a KIT self-check (does the kit's own lints carry guard-pins), not a per-MODULE check; it belongs in CI. Station-side checks (bog-audit, commissioning-verify --bog, triage-console, station-snapshot) stay manual-by-design (need a live station). The auto-chain half (build.sh → preflight + report-module) is PR-B.

## Self-verify
shellcheck report-module.sh exit 0; `bats tests/report-module.bats` 22/22; full `bats tests/*.bats` 0 not-ok; sweep-build-state exit 0. [ev: corpus build-n4-module/automation-audit]

---
**Status**: FOLDED (implementation complete — the wiring is in `report-module.sh`; this retro records it).
