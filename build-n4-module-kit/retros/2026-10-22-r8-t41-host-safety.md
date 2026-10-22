<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 7 -->
# 2026-10-22 · kit · r8-t41-host-safety

**Session**: niagara-research quality round 8, task T41 (host incident and landing lessons), branch `kit/r8-t41-host-safety` from origin/main 6d5d12c
**Delta count**: 7

## What happened
At about 07:59 on 2026-10-22 the WSL `init.scope` cgroup reached its `pids.max` of 48,129 and the kernel rejected
every fork in it; the four Claude Code sessions on the host (about 5.7 GB RSS each) aborted with SIGABRT at
08:00:07-08. Several writers were each running bats or `make check`, and a mutation test of a Go wrapper ran
through a symlinked toolbelt copy. The root trigger is not proven; the shared fork budget is. The same week the
Go build cache grew to 84 GB, C: reached 99% and WSL failed to start. Landing scripts hit three smaller faults:
`gh pr checks --watch` hung, a relative PR body path broke after `cd`, and helpers kept in `/tmp` vanished on a
WSL restart.

## Evidence
- `dmesg`: "cgroup: fork rejected by pids controller in /init.scope"; `/sys/fs/cgroup/init.scope/pids.max` = 48129 (systemd default TasksMax = 15% of `threads-max`) `[ev: dmesg 2026-10-22 07:59]`
- `du -sh ~/.cache/go-build` = 84 GB after 5 days; C: 99%; `Wsl/Service/E_UNEXPECTED`; 8 orphan `swap.vhdx` (22.9 GB) and 65 GB of crash dumps on the Windows side `[ev: host disk audit 2026-10-22]`
- `gh pr checks --watch` blocked 15 min with its single check already `pass` `[ev: landing log 2026-10-22]`
- `tests/ci-local.bats` CL10 RED before the guard (`[ "$status" -eq 4 ]` failed, exit 0), GREEN after; mutation `-gt` → `-ge` fails CL11, dropping the `check_host_pids` call fails CL10 `[ev: tests/ci-local.bats]`
- toolbelt real-path fixes already landed: #315 (`fa1201e`), `fix/r8-wrapper-real-path` (`b494c76`) `[ev: fa1201e]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Read `init.scope/pids.current` before a heavy suite and wait above 20,000; `ulimit -u 4000`; `ci-local.sh` refuses (exit 4) above 50% of any `pids.max` | `BUILD-LOOP.md` § 7 Shared host (lines 314-319); `skill/SKILL.md` step 7 (line 75); `scripts/ci-local.sh` `check_host_pids` (lines 114-129) | `[ev: tests/ci-local.bats]` |
| Δ2 | At most two writers per session run heavy suites at once; the integrator runs the full replay once at the end | `BUILD-LOOP.md` § 7 Shared host (lines 320-321); `skill/SKILL.md` line 75 | `[ev: dmesg 2026-10-22 07:59]` |
| Δ3 | `go clean -cache` and `df -h / /mnt/c` at round close; stop heavy work above 90% on C: | `BUILD-LOOP.md` § 7 Shared host (lines 322-327); `skill/SKILL.md` line 75 | `[ev: host disk audit 2026-10-22]` |
| Δ4 | Poll `gh pr checks <n>` in a bounded loop, never `--watch` | `BUILD-LOOP.md` § 7 Landing scripts (lines 328-333); `skill/SKILL.md` line 75 | `[ev: landing log 2026-10-22]` |
| Δ5 | Resolve path arguments to absolute paths before any `cd` in a landing script | `BUILD-LOOP.md` § 7 Landing scripts (lines 328-333); `skill/SKILL.md` line 75 | `[ev: landing log 2026-10-22]` |
| Δ6 | Keep helper scripts in the repository or the session scratchpad, never `/tmp`; toolbelt entry points resolve their real path (reference to #315, not a new rule) | `BUILD-LOOP.md` § 7 Landing scripts (lines 328-333); `skill/SKILL.md` line 75 | `[ev: fa1201e]` |
| Δ7 | Operator WSL disk hygiene: orphan `swap.vhdx`, crash dumps, `fstrim -av` + diskpart `compact vdisk` for `ext4.vhdx` | `BUILD-LOOP.md` § 7 Shared host (lines 322-327) | `[ev: host disk audit 2026-10-22]` |

## Lessons
- Every session on a WSL host shares one fork budget; a check before the suite is cheaper than four lost sessions.
- A cache with no size cap needs a scheduled clean at a fixed point in the loop.
- A blocking CI watcher is a hang risk; a bounded poll fails visibly.
- A path argument is only safe across `cd` when it is absolute.

---
**Status**: FOLDED (T41, same unit) — Δ1-Δ7 `BUILD-LOOP.md` § 7, Δ1-Δ6 `skill/SKILL.md` step 7, Δ1 guard in `scripts/ci-local.sh`. Owed: a Go host-budget tool (see `ledger.d/2026-10-22-r8-t41-host-safety.md`).
