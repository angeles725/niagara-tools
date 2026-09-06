# Cross-cutting discipline — build-n4-module-campaign10

**Status**: spec · **Source**: v0.20.0 (kit `1fb63d6`; main `cb79676`) · **Target**: v0.21.0
**Topic key**: `sdd/build-n4-module-campaign10/spec`
**Applies to**: all PRs (PR1–PR7)

These rules apply to every code and chore PR in the campaign and are stated here once; capability specs reference them by code.

---

## Process constraints (chain-wide, from C9 carry-over)

| Rule | Requirement |
|------|-------------|
| K11 | No commit body in the entire PR1–PR7 range MAY carry an attribution trailer (Co-Authored-By, Generated-by, or equivalent). The close gate scans the full range and asserts 0 trailers. `[ev: retros/2026-09-06-campaign9-close-process-meta-lessons.md K11]` |
| K12 | Workers write only inside their own worktree. No parallel writer touches the same working tree simultaneously. `[ev: retros … K12]` |
| K13 | Every QA RED branch is cited by branch name and tip SHA. The tip is re-read at apply time before any code is written. Workers NEVER edit a QA RED branch. `[ev: retros … K13]` |
| K14 | Metric names in row output state what they measure (e.g. `companion-flag`, `lint-silent-protection`). No abbreviation that changes meaning. `[ev: retros … K14]` |
| K19 | Every toolbelt script referenced in BUILD-LOOP.md §5 and skill/SKILL.md MUST be updated in the same PR that modifies the script or its flag interface (e.g. new `--strict` flag row). `[ev: retros … K19]` |
| K20 | Exit codes MUST remain disjoint per script: 0 = no FAIL/WARN-only · 1 = any FAIL (or WARN under --strict) · 3 = usage/env/missing-matrix. No script MAY conflate these. `[ev: retros … K20]` |
| D9b | Every scanner that traverses the source tree MUST prune dot-directories (`-name '.*' -prune` or equivalent). A hidden directory is never traversed. `[ev: retros … D9b]` |

## Comment-strip rule (fixtures and pin matching)

Every fixture file used as a test input, and every regex or string match that asserts an identifier or WARN/FAIL token, MUST strip `//` single-line comments and `/* */` block comments BEFORE matching. A token that appears only in a comment MUST NOT satisfy any pin assertion. Every S21, S22, and S23 fixture file MUST include at least one comment-only decoy (a line or block that contains the relevant identifier or WARN string inside a comment) that must NOT satisfy the assertion. `[ev: retros/2026-09-06-campaign9-close-process-meta-lessons.md lesson 21]`

## Observed-flip requirement (all code PRs)

Every code PR (PR1–PR5) MUST record a verbatim OBSERVED mutation flip: a run of the lint over the fixture BEFORE the fix (shows FAIL or WARN) and AFTER the fix (shows PASS or absent), with the exact stdout captured. A "would flip" prose claim is NOT evidence and does not satisfy this requirement. `[ev: retros … lesson 24]`

## Real-tree smoke contract (all code PRs)

Every code PR that changes a lint's verdict MUST run the lint over all four client module roots (ColdRoomPan-rt, CompPan-rt, DashboardPan-rt, DashboardPan-ux) at the blessed worktree `Leon-Guanjuato-worktrees/main-ff1b659` (or a C10 worktree at that tip). Each smoke pin MUST assert: exact WARN/FAIL count, subject(s) by name, absence of unexpected subjects. A smoke that cannot run is a BLOCKER, never downgraded to advisory. The stale main checkout is never a smoke target. `[ev: retros … lesson 1]` `[ev: METHODOLOGY.md K22]`

## Always-conflict files (fragment-merge protocol)

The four files that always conflict across parallel kit PRs — `BUILD-LOOP.md` routing line, `skill/SKILL.md` toolbelt line, `retros/INDEX.md` row, `BUILD-STATE.md` envelope — MUST be merged by FRAGMENT: append the new row/line, keep both rows, dedupe by script name, never overwrite the existing entry. `[ev: retros … lesson 2]`

## `[ev:]` density

Every requirement row and every scenario in every capability spec MUST carry at least one `[ev:]` token citing a RED pin id, a B831 anchor, or a tool source line at kit main `cb79676`. `[ev: retros … lesson 12]`
