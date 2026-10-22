<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-22 · kit · r8-t47-host-budget

**Session**: niagara-research quality round 8, task T47 (owed Go tool `host-budget` from T41, PR #323), branch `feat/r8-host-budget` from origin/main fd2db70
**Delta count**: 2

## What happened
T41 left a bash pids guard in `scripts/ci-local.sh` and an owed Go tool that gates every shared-host resource
of the 2026-10-22 incident (pids cgroups, the Go build cache, disk). T47 implemented it as
`go/internal/hostbudget` + `go/cmd/host-budget` (stdlib only, injectable statfs) and made ci-local call it
through `go/go-tool.sh` `go_tool_bin`; a FAIL row still exits 4, and CL10-CL12 pass unchanged. Two spec points
needed a reading: the spec lists the exact-half case (24064/48128) as PASS, but the same spec WARNs above 40%,
so the row is WARN with exit 0 (not FAIL); the spec says the tool reads `threads-max` without a rule, so the
system task count (loadavg total) uses the pids thresholds. No toolbelt wrapper was added: kit-links L5 needs
every `toolbelt/*.sh` named in BUILD-LOOP.md or skill/SKILL.md, which were outside this task's edit surface.

## Evidence
- `go test ./internal/hostbudget/` RED on a stub (19 of 20 top-level tests failed), GREEN after `[ev: go/internal/hostbudget/hostbudget_test.go]`
- mutations killed: `>` → `>=` in `classify`, `max` parsed as a number, apparent size dropped from the cache sum, `--strict` ignored `[ev: go/internal/hostbudget/command_test.go]`
- `tests/host-budget.bats` HB1-HB6 (incident FAIL exit 1, exact half WARN exit 0 / --strict exit 1, `max` skipped, 21 GB sparse cache WARN with the `go clean -cache` hint, disk FAIL, exit 3) `[ev: tests/host-budget.bats]`
- dropping the `check_host_budget` call in ci-local.sh fails CL10 only `[ev: tests/ci-local.bats]`
- a first draft printed `FAIL > 50%` inside PASS/WARN notes, so `[[ $output != *FAIL* ]]` failed in HB2; notes now read `fail above` `[ev: tests/host-budget.bats HB2]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Point the Shared host pids and Disk-at-round-close manual reads to `go/cmd/host-budget` (one run reports pids, threads, GOCACHE and `/`, `/mnt/c`), and add a host-budget wrapper in the toolbelt routed there | `BUILD-LOOP.md` § 7 Shared host; `skill/SKILL.md` step 7 | `[ev: tests/host-budget.bats]` |
| Δ2 | A row's free-text note must never contain a status word (PASS/WARN/FAIL), so callers can grep the status column | `CONTRIBUTING.md` § 2.2 Go module (`go/`) | `[ev: tests/host-budget.bats HB2]` |

## Lessons
- A status word inside a note column makes a grep for the status lie; keep notes in lower-case prose.
- An injectable statfs keeps disk thresholds testable without filling a filesystem; a sparse file does the same for a size walk.
- When a spec example and its own thresholds disagree, test the thresholds and record the reading.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-22-r8-t47-host-budget.md | kit | 2026-10-22 | pending | 2 |`
