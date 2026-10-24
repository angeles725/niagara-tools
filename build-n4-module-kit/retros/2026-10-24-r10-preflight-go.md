<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-10-24 · kit · r10-preflight-go

**Session**: niagara-research quality round 10, branch `r10/preflight-go` from origin/main 5a4668f0
**Delta count**: 4

## What happened
`toolbelt/preflight.sh` (BUILD-LOOP §0.b, 387 lines of bash) was ported to `go/cmd/preflight`
(`go/internal/preflight`, one func per check, lsof / `java -version` / `df` behind an injectable `Host`).
126 golden runs recorded from the bash (every `tests/preflight.bats` run plus edge cases, under `env -i` with a
per-case PATH and mounts file) were all byte-identical in Go before the truth fixes turned 6 of them into hand-edited truth-* cases, and a live bash-vs-Go matrix over three niagara_homes and
the client trees agreed after one port fix. Two bash truth bugs were fixed after the byte-identical port. `toolbelt/preflight.sh` is now the
go_tool_bin wrapper: lint-structure L15 and the preflight port share one signing-block table
(`go/internal/signblock`), which `tests/lint-structure.bats` LS15-sync pins to `lib/signing-block.sh`. The review
of the golden commit added provenance (`testdata/record/`, an `@kind` per case), `@expect` side-effect checks, an
exact case count and commands run only in the case's declared environment. Follow-ups: `golden.RunMain` has no
`@env`/mode/dir/symlink/`@expect` sections, so `internal/preflight/golden_test.go` keeps its own loop (the
extension is over 40 lines); `build-verify.md` still cites `toolbelt/preflight.sh:117-170`, the bash body.

## Deltas
| Δ | Change | Where | Evidence |
|---|---|---|---|
| Δ1 | Go port of preflight.sh, byte-identical on all 126 recorded goldens before the truth fixes; jar-lock keeps one `lsof -Fn` run (a `/proc/<pid>/fd` scan misses mapped files and other users' processes), SKIP semantics unchanged; `toolbelt/preflight.sh` execs it. | `go/internal/preflight/`, `go/cmd/preflight/`, `toolbelt/preflight.sh` | `go test ./internal/preflight` TestGolden |
| Δ2 | Port pitfall: bash sorts the matches of `"<dir>"/*/` with their trailing slash, so `x-worktrees/` sorts before `x/`; a Go port that sorts bare names reorders rows. | `internal/preflight/jdk.go` globDirs | golden edge-glob-slash-order-drift, -jdk |
| Δ3 | Truth fix: plugin-pin without --plugin-version matched the settings version anywhere in `find <nh>/etc/m2 -path "*V*"`, niagara_home included: a home named `n4-7.6.17` with an empty etc/m2, a `7.6.170` dir or a `7.6.17.pom` file PASSed. Now an exact directory name, as the flag already was. | `internal/preflight/plugin.go` | goldens truth-plugin-*, bats PF-plugin-substring |
| Δ4 | Truth fix: version-drift took a tree's first defaultModuleVersion in `find` readdir order (ext4 hash order), so the same tree drifted on one host and not another; Go walks in bytewise order. | `internal/preflight/find.go` | goldens truth-drift-sorted-*, bats PF-drift-sorted |

## Fold
- Δ1 → `BUILD-LOOP.md` § 5 What to port (preflight.sh is no longer the glue example; it is `go/cmd/preflight`), and `skill/SKILL.md` Kit tooling; the LS15-sync model in Pin every shared helper names `go/internal/signblock`.
- Δ2 → `BUILD-LOOP.md` § 5 Record the shell's silent input drops as edge goldens: a `*/` glob sorts with its trailing slash.
- Δ3 → same bullet: `find -path "*V*"` matches anywhere in the path; a version is an exact directory name.
- Δ4 → same bullet: a first `find` hit is readdir order; a Go walk reads in bytewise order.
- Follow-up (1), `golden.RunMain` sections, stays the open issue in `ledger.d/2026-10-24-r10-preflight-go.md`; follow-up (2) is dropped there: no kit file cites `toolbelt/preflight.sh` line numbers any more.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-tail`; log in `ledger.d/2026-10-24-r10-fold-tail.md`) — Δ1-Δ4 cited in `BUILD-LOOP.md` § 5 and `skill/SKILL.md`.
