<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-24 · kit · r10-stale-kit-resolution

**Session**: niagara-research quality round 10 (R10-D7), branch `feat/r10-kit-resolve-fail-closed` from
origin/main fd87e7c
**Delta count**: 1

## What happened
The `/build-n4-module` launcher resolved the kit as "`$BUILD_N4_KIT` if set, else the default checkout", and
checked only that `METHODOLOGY.md` existed there. The default checkout sat on an old feature branch, 821
commits behind origin/main, with uncommitted edits. A client build and verify ran on it: its
`lint-lexicon-ascii.sh` exited 0 on a real violation that origin/main's copy FAILs. Nothing said which kit
ran, so the stale gate passed silently.

## Evidence
- Launcher text before the fix: `skill/SKILL.md` "Resolve the kit (do this first)" steps 1-3
  (`$BUILD_N4_KIT`, default `/home/cristian/modulos_niagara_n4/niagara-tools/build-n4-module-kit`, else
  `fd -t f METHODOLOGY.md`): an existence check, no freshness or cleanliness check, no SHA printed
  `[ev: tests/kit-resolve.bats KR3]`
- The default checkout on 2026-10-24: a feature branch 821 commits behind origin/main, dirty; its
  `lint-lexicon-ascii.sh` passed a violation origin/main's FAILs (R10-D7 report)
- Constraint: `tests/kit-links.bats` L2 forbids version control in `toolbelt/*.sh` (scripts run in
  worktrees and on stations); `scripts/` already holds the git-dependent halves (`check-client-source.sh`,
  L12). So the resolver is `scripts/kit-resolve.sh`, shellchecked by the `scripts/*.sh` glob in `ci.yml`
- RED: `tests/kit-resolve.bats` KR1-KR13 all failed before the script existed (exit 127). Mutations that drop
  the dirty check, the behind check, the offline refusal or the kit-main-only refresh guard each flip at
  least one case (KR4/KR11, KR3/KR9/KR10, KR7, KR9/KR10)

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | **Resolve the kit fail-closed and say which one runs.** The launcher runs `scripts/kit-resolve.sh`: candidates `$BUILD_N4_KIT` → kit-main worktree (`$BUILD_N4_KIT_MAIN`) → default checkout; the first that holds `METHODOLOGY.md` must be at origin/main (fetched) and clean, or it is refused with exit 2 and the refresh command, never skipped for the next candidate. An unreachable origin is a named WARN and a refusal unless `--offline-ok`. `--refresh` checks out origin/main only in a clean kit-main worktree. Its first line `KIT=<path> SHA=<sha> (origin/main, clean)` is the session's first output. Reviewer check: a behind or dirty scratch kit exits 2. | `skill/SKILL.md` "Resolve the kit"; `BUILD-LOOP.md` § 0; `scripts/kit-resolve.sh` | `[ev: tests/kit-resolve.bats KR3]` |

## Lessons
- "The file exists" is not "the tool is current": a gate run from a stale tree is a stale gate.
- A resolver that falls through to the next candidate when the first is stale hides the problem; refuse and
  print the fix.
- Print the SHA of the tooling that ran, so a report can be traced to the code that produced it.

---
**Status**: FOLDED 2026-10-24 (branch `feat/r10-kit-resolve-fail-closed`) — Δ1 `skill/SKILL.md` "Resolve the
kit (do this first)" and `BUILD-LOOP.md` § 0 "Run a current kit", implemented by `scripts/kit-resolve.sh` +
`tests/kit-resolve.bats`. INDEX row generated from the header: `| 2026-10-24-r10-stale-kit-resolution.md | kit | 2026-10-24 | folded | 1 |`
