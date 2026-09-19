<!-- review-status: folded -->
# 2026-09-19 · kit · wire-lints

**Session**: 2026-09-19 kit-improvement campaign (branch odd/issue-113-wire-lints) — wire the 9 new lints into report-module.sh (issue #113).
**Delta count**: 2

## What happened
The 9 new lints (PR #110) shipped standalone; issue #113 asked to surface them in the aggregated
report-module.sh punch-list. Wired all 9 following the existing per-artifact/module-root block idiom,
with report-module.bats coverage.

## Evidence
- report-module.bats 13/13 (RM7-RM11 new); full suite 576 ok / 0 not-ok; shellcheck 0.10.0 exit 0; smoke on CompPan clean. `[ev: tests/report-module.bats]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | per-artifact src lints wired (no-system-out FAIL; clock-zero-floor/null-context-write/bql-string-concat/arbitrary-ord WARN) + profile-gated se-display (FAIL, -se) / jasmine-ux (WARN, -ux) | `toolbelt/report-module.sh` §5.7-5.13 | `[ev: retro new-lints]` |
| Δ2 | module-root lints wired once per run: agent-on-shape (FAIL), uberjar-api-conflict (WARN) | `toolbelt/report-module.sh` §8-9 | `[ev: retro new-lints]` |

## Lessons
- FAIL lints must set the module's HAD_FAIL and drive the non-zero exit; WARN lints emit rows only.
- Profile-gated lints (se/ux) key off the artifact name suffix (`*-se`, `*-ux`); module-root lints run once after the artifact loop.

---
**Status**: FOLDED — 9 lints in report-module.sh + RM7-RM11 bats (2026-09-19), issue #113.
