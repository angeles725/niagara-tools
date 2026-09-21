# ODD feature — Make build-n4-module fully automatic (no "must remember" step)

**Created**: 2026-09-20 · **Repo**: angeles725/niagara-tools · from the automation audit (memory build-n4-module/automation-audit)

## Problem (operator)
Module creation must be AUTOMATIC — no mechanical check should depend on the agent REMEMBERING to
run it (the research-sdd flaw). Audit found: `report-module.sh` (the aggregated punch-list) and
`preflight.sh` are invoked by NO orchestrator — only documented as steps in SKILL.md/BUILD-LOOP.md.
And 8 mechanical lints aren't even wired into `report-module.sh`.

## Tasks
- [ ] **PR-A** — wire the static-source orphan lints into `report-module.sh` (so ONE report-module run is complete) + RM bats:
      per-artifact: lint-subscribe-without-unsubscribe (rt WARN), lint-recovery-path (rt FAIL),
      lint-config-sanity (rt FAIL), lint-status-parity (rt WARN), lint-servlet (ux FAIL),
      lint-wb-threading (wb WARN), rc-scan (ux rc/ FAIL); module-once: lint-structure (FAIL),
      lint-write-path (FAIL). Mirror the existing §5.7 pattern (SKIP if no src, ERROR on exit 3).
- [ ] **PR-B** — auto-chain: `build.sh` runs `preflight.sh` at start + `report-module.sh` at end
      (with `--no-preflight`/`--no-report` bypasses), so one command does preflight→build→verify→report.
      Update SKILL.md/BUILD-LOOP.md to present the single automatic flow. Update build.bats.

## Out of scope / by-design-manual (keep manual)
- `bog-audit`, `commissioning-verify --bog`, `triage-console`, `station-snapshot` — need a live station.
- `lint-guard-pins` — a KIT self-check, belongs in CI, not the per-MODULE report (separate follow-up).
- UI preview gate — human visual approval.

## Progress log
- 2026-09-20: audit done (memory build-n4-module/automation-audit); starting PR-A.
