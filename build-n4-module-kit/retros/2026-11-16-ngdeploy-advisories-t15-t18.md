<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-11-16 · kit · ngdeploy-advisories-t15-t18

**Session**: rows T15 and T18 of the 2026-11-15 coordination-stretch retro: the native-review advisories of the
#693 correction batch and the verifier advisories of the #698 run lock, triaged; the real ones fixed.
**Delta count**: 4

## What happened
T15 held six advisories. Two were real: the probe before the build dropped a held signal when it looked at a
jar it does not move, and it moved a `<jar>.ng-probe` of any type to the jar's name. The other four were declined
with a reason (positional tar options, child environment, the held-open heuristic, git ceiling directories).
T18 held three, all accepted: the two-step lock NOTICE printed per lock and before a refusal, a link error
other than an existing lock failed the run, and a run killed while it took its lock left its temporary file.

## Evidence
- RED before GREEN: the SIGHUP case of the probe ended with exit 0 and a build; a directory or symlink probe was
  renamed to the jar; two locks printed two NOTICEs and a refused lock printed one; EMLINK was exit 10; an old
  temporary lock file stayed. `[ev: go/internal/ngdeploy/probe_signal_test.go]`
  `[ev: go/internal/ngdeploy/probe_stale_type_test.go]` `[ev: go/internal/ngdeploy/lock_advisories_test.go]`
- The whole Go suite under the memcap shim and `tests/ng-deploy.bats` (63/63) pass on the branch.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. T15 probe-signal-dropped: the probe checks for a held SIGINT, SIGTERM or SIGHUP after every jar, moved or not | Go `internal/ngdeploy` `probe.go` · `usage.txt` | `[ev: go/internal/ngdeploy/probe.go]` |
| Δ2 | APPLIED 2026-11-16. T15 stale-probe-any-type: a `<jar>.ng-probe` that is not a regular file is exit 40 naming what it is, left in place | Go `internal/ngdeploy` `probe.go`, `messages_copy.go` · `usage.txt` | `[ev: go/internal/ngdeploy/probe_stale_type_test.go]` |
| Δ3 | APPLIED 2026-11-16. T18: the two-step lock NOTICE prints once per run, after the lock is held; any link error but an existing lock falls back to the two steps | Go `internal/ngdeploy` `lock.go`, `lock_make.go` · `usage.txt` | `[ev: go/internal/ngdeploy/lock_make.go]` |
| Δ4 | APPLIED 2026-11-16. T18: a `.ng-deploy.lock.*.tmp` older than a minute is swept with a NOTICE when the next run takes that lock | Go `internal/ngdeploy` `lock_sweep.go` · `usage.txt` | `[ev: go/internal/ngdeploy/lock_sweep.go]` |
