<!-- review-status: pending -->
<!-- Campaign 7 PR7: types/logic.md split — framework-extension authoring moves to types/logic-authoring.md -->

# Retro — Campaign 7 PR7: types/logic-authoring.md split

**Date**: 2026-09-05  
**Scope**: types/logic.md split; kit-links L4/L5/L6; BUILD-LOOP + SKILL.md routing references  
**Branch**: docs/c7-logic-split  
**Head SHA**: (set at commit)

## Lessons (PROPOSED kit deltas — propose-never-apply)

### L1 — Audience boundary: line 91 is the right split; kitControl patterns (lines 80-90) belong with control authoring
The explore phase estimated the split at line 80, but the real audience boundary is `## Author-side SPIs` (line 91). Lines 80-90 (`## kitControl patterns`) are exemplar-backed control authoring, not framework SPIs — they are proven from ColdRoomPan-rt and belong in `types/logic.md` with the control engineer audience. Always anchor a split boundary to a heading that marks a clear audience change, not an approximate line count.

### L2 — Split invariant is a concrete, machine-verifiable proof: cat + sort | uniq diff = only the infrastructure lines
The split invariant check (`comm -23 <(cat logic.md logic-authoring.md | grep . | sort) <(grep . original | sort)`) proves that ZERO original lines were lost and ZERO were duplicated. The only new lines are the 3-line HTML comment header and the 1-line cross-reference. Run this check before committing any file split. Invariant: `wc -l logic.md ≤ 91` + `diff(original_sorted, cat_sorted_minus_infra) == empty`.

### L3 — The L5 routing guard catches scripts that landed in PR4/PR5 but were never added to routing docs
`scaffold-module.sh` and `schema-risk.sh` were added in PR4/PR5 but not documented in BUILD-LOOP.md or SKILL.md. L5 was naturally RED when written — the test catches exactly this gap. The fix: add both scripts to SKILL.md §References + step 5, and to BUILD-LOOP.md §1/§5. Tag each with `[ev: retro tool-integration]`. This is the expected workflow for a routing guard: new scripts start undocumented, the guard turns RED, PR7 is where the coverage is closed.

### L4 — shellcheck disable directives must go before the entire compound command, not inside it
A `# shellcheck disable=SC2016` inside a `while ... done` body (before `done`) triggers SC1123 ("ShellCheck directives are only valid in front of complete compound commands"). The fix: move the disable directive to the line immediately BEFORE `while IFS= read -r ref; do`. This is the canonical form for all compound statements (while, for, if, case).

### L5 — corpus-index.md §Campaign 6 must be retargeted from logic.md to logic-authoring.md
The `### → types/logic.md` section in corpus-index.md and its prose reference (`See types/logic.md §Author-side SPIs`) point at content that moved to logic-authoring.md. Both must be retargeted atomically with the split. Leaving them pointing at logic.md would make corpus-index.md a dangling guide (the section it describes no longer exists there). Always retarget corpus-index.md section pointers when moving content to a companion file.
