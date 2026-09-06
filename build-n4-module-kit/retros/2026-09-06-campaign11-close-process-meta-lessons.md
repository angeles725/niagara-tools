<!-- review-status: folded -->
# 2026-09-06 · kit · campaign11-close-process-meta-lessons

**Session**: Campaign 11 CLOSE — v0.22.0 shared parser (T1), DRIFT (T3), client-root (T2), guard-pins (T4)
**Delta count**: 6

## What happened

Campaign 11 produced four lint-precision PRs (T1-T4). The close process surfaced six process-level lessons about the LD5 bug-vs-rule class, zero `# Mutation:` lines, three awk invocation mechanisms, the `$KIT` writable seam vs `BASH_SOURCE`, the spec agent's inferred golden table, I2/I3 defensive shapes with reachable-but-absent corpus entries, the fragment-merge slip and the 0-markers-before-add rule, real C8-archive markers found in production plus the CLOSE-no-conflict-markers pin, the c8-close SC1-smoke vacuity, the amend-range retro anchor gap, and two more K12 worktree breaches.

[ev: retro campaign11-close-process-meta-lessons]

## Evidence

- I2/I3 DEFENSIVE not inert: B832-G3 (Case-B @-stop misses a param-annotation-continuation method; 0 w/ stop, 1 without) and B832-G4 (I3 reachable via a control keyword in a non-entered instance initializer; 0 w/ I3 → 1 drop) — both shapes are REACHABLE-BUT-ABSENT in the C11 client tree, so the guards earn their fixtures. `[ev: ad2121b69]`
- Fragment-merge slip (process): a merger raised on a single-line hunk and chained `git add -A && rebase --continue && push` without checking; conflict markers were pushed for minutes. `[ev: C11 PR2 apply]`
- Real conflict markers in C8 archive: the same sweep found leftover `<<<<<<<`/`>>>>>>>` in the C8 archive apply-progress.md (resolved `7053907`). `[ev: C11 CLOSE-no-conflict-markers 46d3eff]`
- c8-close SC1-smoke VACUOUS: `[ status 1 ] || [ status 0 ]` — always true; the LD5 bug-vs-rule class. Tightened `d837ae5` (pins the blessed-tree verdict). `[ev: C11 PR3 second read 654535f]`
- Amend-range retro anchor gap: a `--amend` re-writes the tip so the pre-push hook's range no longer contains the `Retro:` trailer commit; workaround = an empty `Retro: none` commit to satisfy envelope-pairing. `[ev: C11 apply]`
- Two K12 breaches: workers ticked `tasks.md` in the MAIN checkout instead of their worktree. `[ev: C11 apply K12]`

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | I2/I3 are DEFENSIVE with reachable-but-absent shapes (B832-G3/G4): reclassify from "inert" to "defensive, reachable under alternate tree" | METHODOLOGY.md — note under B832 | `[ev: retro campaign11-close-process-meta-lessons]` |
| 2 | Assert 0 conflict markers (`! git grep -nE '^(<<<<<<<\|=======\|>>>>>>>)'`) BEFORE `git add`; never chain add/continue/push after an unchecked script | METHODOLOGY.md / BUILD-LOOP.md | `[ev: retro campaign11-close-process-meta-lessons]` |
| 3 | CLOSE-no-conflict-markers bats pin: close skeleton must assert 0 tracked files with conflict markers | tests/c11-close.bats | `[ev: retro campaign11-close-process-meta-lessons]` |
| 4 | An `--amend` pre-push hook gap: use an empty `Retro: none` commit to satisfy the envelope-pairing rule when no new retro is due | BUILD-LOOP.md §7 note | `[ev: retro campaign11-close-process-meta-lessons]` |
| 5 | K12 enforcement note: workers write only in their worktree — a tasks.md tick in the main checkout is a K12 breach; add a cross-ref to the K12 example | METHODOLOGY.md §K12 | `[ev: retro campaign11-close-process-meta-lessons]` |
| 6 | LD5 bug-vs-rule: a real-tree smoke that asserts a formerly-present defect passes for the wrong reason once the defect is fixed; pin the clean state and carry the rule in synthetic fixtures | METHODOLOGY.md §K26 | `[ev: retro campaign11-close-process-meta-lessons]` |

## Lessons

- I2/I3 defensive guards with REACHABLE-BUT-ABSENT corpus shapes (B832-G3/G4) earn their fixture; reclassify from "inert" to "defensive" only when no shape in the target tree exercises the guard — not when no shape anywhere does. `[ev: retro campaign11-close-process-meta-lessons]`
- Assert 0 conflict markers in tracked files BEFORE `git add`; never chain `add && rebase --continue && push` after a script-resolved merge without a marker check — the slip is invisible to a clean `git status`. `[ev: retro campaign11-close-process-meta-lessons]`
- A real-tree smoke asserting a known-fixed defect passes for the wrong reason once the defect is fixed (LD5 bug-vs-rule class, same as c8-close SC1-smoke vacuity); pin the tree's current correct verdict and carry the rule in synthetic fixtures. `[ev: retro campaign11-close-process-meta-lessons]`
- A `--amend` re-writes the tip commit; the pre-push hook's range no longer includes the `Retro:` trailer; satisfy the envelope-pairing rule with an empty `Retro: none` commit rather than an amend. `[ev: retro campaign11-close-process-meta-lessons]`
- Workers write only in their worktree (K12); a tasks.md tick committed to the main checkout is a K12 breach that can silently overwrite a peer's in-progress changes. `[ev: retro campaign11-close-process-meta-lessons]`
- A RED must never pin another bats file's line number by integer literal (K13); compute or read the target line — a stale integer pin passes when the fixture moves. `[ev: retro campaign11-close-process-meta-lessons]`

---
**Status**: FOLDED — METHODOLOGY.md K24(7) sub-bullet + K25 + K26 + K20 note + doctrine fold note. `[ev: retro campaign11-close-process-meta-lessons]`
