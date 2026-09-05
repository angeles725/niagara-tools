<!-- review-status: folded -->
<!-- kit-retro -->
# Campaign 7 PR2: tool integration — BUILD-LOOP + launcher routed to preflight, lint-timers, slot-coverage, fold-audit, --age

Date: 2026-09-05 · Module: kit · SDD: build-n4-module-campaign7 PR2 (v0.17.0 → v0.18.0 chain)

PR2 of 8 in the Campaign-7 chain. Doc-only routing: zero new bats tests (padding = defect).
Branch: `docs/c7-tool-integration`. Gate exits: R2.1–R2.7.

## What was routed

Four toolbelt scripts that shipped in Campaign 6 (preflight.sh, lint-timers.sh,
slot-coverage.sh, sweep-fold-audit.sh) were invisible to the documented workflow.
This PR names them in BUILD-LOOP.md and skill/SKILL.md, closing five integration gaps
identified in explore.md §1.

| Gap | File:line | What was added |
|-----|-----------|----------------|
| §0.b missing preflight | BUILD-LOOP.md §0.b | `toolbelt/preflight.sh <niagara_home> <gradle-root>` as THE environment preflight |
| §5 missing pre-gate | BUILD-LOOP.md §5 | `lint-timers.sh <src>` + `slot-coverage.sh [--strict] <module-include.xml> <module.lexicon>` before `verify-module.sh`; `--plano` when available |
| §7 missing --age | BUILD-LOOP.md §0.a + §7 | `sweep-build-state.sh --age --today <date>` at orient and close; `sweep-fold-audit.sh --strict` at close |
| skill/SKILL.md §References stale | skill/SKILL.md §References | All 10 toolbelt scripts listed |
| skill/SKILL.md step 5 no routing | skill/SKILL.md §Execution step 5 | Routes to pre-gate checks (lint-timers, slot-coverage) before the verify gate |

## Lesson (proposed kit delta)

**Four shipped tools were invisible to the documented workflow.** The routing check
`for s in build-n4-module-kit/toolbelt/*.sh; do n=$(basename $s); grep -q "$n" build-n4-module-kit/BUILD-LOOP.md build-n4-module-kit/skill/SKILL.md || echo "UNROUTED $n"; done`
is now the guard — running it in PR body before merge catches any future gap.
This pattern generalises: a new script is not "done" until it appears in both
BUILD-LOOP.md and skill/SKILL.md.

Retro: retros/2026-09-05-campaign7-tool-integration.md (1 delta, review-status: pending)
