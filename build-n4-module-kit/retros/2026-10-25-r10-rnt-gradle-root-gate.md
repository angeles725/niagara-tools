<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-25 · kit · r10-rnt-gradle-root-gate

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test run` (D5), a gate on the `--gradle-root`
git worktree and an opt-in `--restore-gradle-root`, branch `r10/rnt-gradle-root-gate` from 005b2565 (origin/main)
**Delta count**: 2

## What happened
A live run with niagara gradle plugin 7.3.40 (iC-Niagara 4.13) showed `:<part>:compileModuleTestJava` appending
every `@NiagaraType` test class to the PRODUCTION `<part>/module-include.xml` inside the gradle root, a tracked
source file; it never edits `moduleTest-include.xml`, and it depends on `:jar`. Every runner boot therefore
silently mutated the client's source tree. The runner already gated the install and the real user homes with
before/after snapshots; it had no gate on the gradle root. Each rule got a fake-runner RED first (the fake hands
`git` to a real, hermetic git on a fixture repository); no Windows boot was run:
1. Before the copy, `git -C <root> rev-parse --show-toplevel` then `git -C <root> status --porcelain=v1 -z
   --untracked-files=all`, every listed file sha256-hashed. A root that is no git worktree is `FAIL  run-niagara-test
   gradle root is not a git worktree: cannot verify it`, exit 3, before any copy (fail closed).
2. After cleanup the same status: any new, modified, deleted or rewritten path (a content change to a file already
   dirty, or a dirty file the build reverted, included) is `FAIL … gradle root changed by the build: <paths>` (10,
   then "and N more"), a failed build included; else `PASS … gradle root unchanged (<n> tracked changes before, same
   after)`.
3. `--restore-gradle-root` checks out (`git --literal-pathspecs -C <root> checkout -- <paths>`) only the paths clean
   before and now ` M`/` D`/` T`, re-reads the status and prints `gradle root restored: <paths>` (untracked files the
   build created are listed and kept), or a FAIL row when the re-read still shows them. The run stays FAIL. The fake
   runner asserts no `git clean`, `reset`, `restore`, `stash` or `checkout .` is ever issued.

## Evidence
- the REDs, then green: `TestRunGradleRootNotAWorktree`, `TestRunGradleRootUnchanged`, `TestRunGradleRootChangedIsFail`, `TestRunGradleRootChangedListsTenPaths`, `TestRunGradleRootRestore`, `TestRunGradleRootRestoreUntrackedOnly`, `TestRunGradleRootRestoreUnconfirmed` `[ev: go/internal/niagaratest/run_gradleroot_test.go]`
- the snapshot, the compare and the restore `[ev: go/internal/niagaratest/gradleroot.go]`
- guard-drop mutations: no per-file hash turned `TestRunGradleRootChangedIsFail` and `TestRunGradleRootRestore` red; restoring already-dirty files turned `TestRunGradleRootRestore` red; restoring without the flag turned `TestRunGradleRootChangedIsFail` red

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `run-niagara-test run` fails when the build changes the `--gradle-root` git worktree (status plus a hash per listed file, before the copy and after cleanup; no worktree is exit 3 before any copy); `--restore-gradle-root` checks out only the files clean before and modified by the build, the run stays FAIL | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/gradleroot.go]` |
| Δ2 | Plugin-version trap: with plugin 7.3.40 (4.13) `compileModuleTestJava` appends test types to the production `module-include.xml` and depends on `:jar`; check `git status` after any moduleTest compile | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/gradleroot.go]` |

## Lessons
- A sandbox that guards the install and the user homes still leaks through the one tree it builds from: gate every tree a build can write, the source tree included.
- `git status` codes alone miss a second write to a file that was already dirty; hash every listed file.
