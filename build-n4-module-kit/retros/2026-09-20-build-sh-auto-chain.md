<!-- review-status: folded -->
# 2026-09-20 · kit · build.sh auto-chain (automation audit fix, PR-B)

**Session**: automation audit, PR-B (PR-A wired the orphan lints into report-module.sh). Operator requirement: module creation must be AUTOMATIC — no mechanical step may depend on the agent REMEMBERING to run it (the research-sdd flaw). Audit found `report-module.sh` and `preflight.sh` were invoked by NO orchestrator: they lived only as SKILL.md/BUILD-LOOP.md steps the agent had to remember. This PR makes `build.sh` the single automatic entry.

**Delta count**: 0 (implementation retro)

## What was done
`toolbelt/build.sh` now runs, in ONE invocation: **preflight → gradle build → verify gate → report-module** (the aggregated hand-off punch-list, which since PR-A runs every static-source lint).
- `preflight.sh "$NIAGARA_HOME" "$GRADLE_ROOT"` runs after the env checks, before gradle; preflight FAIL (exit 1) or env (exit 3) → build.sh exit 10.
- `report-module.sh "$ROOT/$MOD"` runs after the verify gate passes; report FAILs → build.sh exit 50 ("not hand-off-ready"); report env → exit 10.
- Bypasses for the inner rebuild loop: `--no-preflight`, `--no-report`.
- Header/usage + exit-code table updated; `BUILD-LOOP.md` §4/§5 and `skill/SKILL.md` steps 4/5 rewritten to present the single-command flow (standalone tool references kept for kit-links L5). Extended the existing `tests/build-sh.bats` with 4 stub-driven tests (BS-preflight-fail/skip, BS-report-fail/skip) — the setup() now stubs preflight.sh + report-module.sh next to build.sh (mirroring the verify-module.sh stub, FAKE_PREFLIGHT_EXIT/FAKE_REPORT_EXIT).

## Why this closes the operator concern
Before: an agent following BUILD-LOOP had to remember to run preflight (start) and report-module (end) as separate steps — exactly the "mechanical function the user must activate/remember" flaw. After: one `build.sh` invocation does the whole mechanical chain; forgetting a step is no longer possible. The bypass flags exist only for the deliberate inner-loop case.

## Scope note
`lint-guard-pins` stays out of the per-module flow (a KIT self-check → CI). Station-side checks (bog-audit, commissioning-verify --bog, triage-console, station-snapshot) stay manual-by-design — they need a live station and cannot be part of the offline build chain.

## Self-verify
shellcheck build.sh exit 0; `bats tests/build.bats` 5/5; full `bats tests/*.bats` 0 not-ok; sweep-build-state exit 0. [ev: corpus build-n4-module/automation-audit]

---
**Status**: FOLDED (implementation complete — the chain is in `build.sh`; this retro records it). Automation audit CLOSED: report-module now complete (PR-A) and auto-invoked by build.sh alongside preflight (PR-B).
