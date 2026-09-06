# Cross-cutting discipline — build-n4-module-campaign11

**Status**: spec · **Source**: v0.21.0 (kit `dab0807`) · **Target**: v0.22.0
**Topic key**: `sdd/build-n4-module-campaign11/spec`
**Applies to**: all PRs (PR1–PR5)

These rules apply to every code and chore PR in the campaign and are stated here once; capability specs reference them by code.

---

## Process constraints (chain-wide)

| Rule | Requirement |
|------|-------------|
| K11 | No commit body in the entire PR1–PR5 range MAY carry an attribution trailer (Co-Authored-By, Generated-by, or equivalent). The close gate scans the full range and asserts 0 trailers. `[ev: retros/2026-09-06-campaign10-close-process-meta-lessons.md K11]` |
| K12 | Workers write only inside their own worktree. No parallel writer touches the same working tree simultaneously. `[ev: retros … K12]` |
| K13 | Every QA RED branch is cited by branch name and tip SHA. The tip is re-read at apply time before any code is written. Workers NEVER edit a QA RED branch. `[ev: retros … K13]` `[ev: proposal §4]` |
| K19 | Every toolbelt script referenced in BUILD-LOOP.md §5 and skill/SKILL.md MUST be updated in the same PR that adds or modifies the script or its flag interface. New scripts `toolbelt/lib/method-boundary.sh` and `toolbelt/lint-guard-pins.sh` require routing rows in both docs in the same PR (PR1 and PR4 respectively). `[ev: retros … K19]` |
| K20 | Exit codes MUST remain disjoint per script: 0 = no FAIL/WARN-only · 1 = any FAIL (or WARN under --strict) · 3 = usage/env/missing-matrix. No script MAY conflate these. `[ev: retros … K20]` |
| K21 | Every load-bearing client tree cite names its worktree path AND its commit SHA. The live checkout `Cliente/Leon-Guanjuato` at `4f5f1c7` is NEVER a cite target or smoke target — it carries Cristian's uncommitted files. `[ev: memory client-reads-use-a109249-worktree]` |
| K22 | Every smoke pin MUST assert: exact WARN/FAIL count, subject(s) by name, absence of unexpected subjects. A smoke that cannot run is a BLOCKER, never downgraded to advisory. `[ev: METHODOLOGY.md K22]` |
| K24(7) | Every mutation named in a lint header's `# Mutation:` / `# OBSERVED mutation:` line MUST map to an existing bats fixture. A "would flip" prose claim is NOT evidence and does not satisfy this requirement. Each named mutation in a PR's lead gate MUST name the exact fixture it flips, confirmed by QA (K24(7)). `[ev: METHODOLOGY.md K24(7) @ dab0807]` |
| D9b | Every scanner that traverses the source tree MUST prune dot-directories (`-name '.*' -prune` or equivalent). A hidden directory is never traversed. `[ev: retros … D9b]` |

## Comment-strip rule (fixtures and pin matching)

Every fixture file used as a test input, and every regex or string match that asserts an identifier or WARN/FAIL token, MUST strip `//` single-line comments and `/* */` block comments BEFORE matching. A token that appears only in a comment MUST NOT satisfy any pin assertion. Every new fixture file MUST include at least one comment-only decoy (a line or block that contains the relevant identifier or WARN string inside a comment) that must NOT satisfy the assertion. `[ev: retros/2026-09-06-campaign9-close-process-meta-lessons.md lesson 21]`

## Observed-flip requirement (all code PRs)

Every code PR (PR1–PR4) MUST record a verbatim OBSERVED mutation flip: a run of the lint over the fixture BEFORE the fix (shows FAIL or WARN) and AFTER the fix (shows PASS or absent), with the exact stdout captured. A "would flip" prose claim is NOT evidence and does not satisfy this requirement. `[ev: retros … lesson 24]` `[ev: K24(7)]`

## Real-tree smoke contract (all code PRs that change a lint's verdict)

Every code PR that changes a lint's verdict MUST run that lint over the three rt client module roots (ColdRoomPan-rt, CompPan-rt, DashboardPan-rt) at the blessed worktree `Leon-Guanjuato-worktrees/main-ff1b659` (commit `ff1b659`). All smoke targets resolve through `tests/lib/client-root.bash` after PR3 merges. Each smoke pin MUST assert: exact WARN/FAIL count, subject(s) by name, absence of unexpected subjects. A smoke that cannot run is a BLOCKER, never downgraded to advisory. `[ev: METHODOLOGY.md K22]` `[ev: proposal SC-4]`

## Smoke-assertion class rule (NEW in C11 — from the LD5 lesson)

A real-tree smoke MUST assert the tree's **current CORRECT verdict**, not a known defect. Real trees legitimately carry rows the lints are designed to emit (lint-ext-writable-shape = 1 WARN `faultReset` on CompPan-rt @ ff1b659 is the correct S22 verdict; rc-scan RC8 = 1 FAIL for the DashboardPan-ux :701 host literal is the rule's correct steady-state detection) — those ARE the pinned rule on a real tree and need nothing else. What rots is a smoke whose FAIL/WARN is caused by a FIXABLE, transient defect on the target tree (LD5 asserted the ColdRoomPan defrost time<=0 bug): once the defect is fixed the smoke goes green for the wrong reason or red. Rule: the lint rule is pinned by synthetic fixtures; a real-tree smoke pins the current correct verdict; a smoke whose asserted FAIL/WARN is a fixable defect (not a steady-state detection) MUST cite the defect (issue or retro id) and the synthetic fixture that carries the rule, and is re-measured when the defect is fixed. `[ev: proposal §Intent ¶4; investigador1 spec read 2026-09-07; lead re-measure on main-ff1b659]`

## Always-conflict files (fragment-merge protocol)

The four files that always conflict across parallel kit PRs — `BUILD-LOOP.md` routing line, `skill/SKILL.md` toolbelt line, `retros/INDEX.md` row, `BUILD-STATE.md` envelope — MUST be merged by FRAGMENT: append the new row/line, keep both rows, dedupe by script name, never overwrite the existing entry. `[ev: retros … lesson 2]` `[ev: K24]`

## `[ev:]` density

Every requirement row and every scenario in every capability spec MUST carry at least one `[ev:]` token citing a RED pin id, a B832 anchor, or a source line at kit `dab0807`. `[ev: retros … lesson 12]`
