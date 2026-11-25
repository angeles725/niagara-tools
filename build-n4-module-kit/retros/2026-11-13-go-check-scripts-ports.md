<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-11-13 · kit · go-check-scripts-ports

**Session**: "all tool logic in Go" pass: `check-client-source`, `check-installed-line` and `check-bats-skips` ported to Go.
**Delta count**: 3

## What happened
Three guard scripts that parse structured input (BUILD-STATE markdown, gradle version lines across git refs, bats
TAP) moved to `go/internal/checkclientsource`, `checkinstalledline` and `checkbatsskips` behind thin wrappers,
with one shared git runner (`go/internal/gitcmd`). The ports are byte-identical to the bash scripts on 94 of 95
differential runs; the one difference is a `grep:` line the old `check-bats-skips.sh` leaked on stderr. Keeping
parity kept three soft spots of the bash versions, found by the differential and by the native review.

## Evidence
- Differential (stdout, stderr, exit code): check-client-source 59/59, check-installed-line 17/17,
  check-bats-skips 18/19 (SK6 stderr only). `[ev: 92646399]`
- `check-installed-line` skips a group whose path git quotes (a non-ASCII directory): a 5.1 -> 5.0 downgrade in such
  a group is reported by neither version. `[ev: go/internal/checkinstalledline/git.go]`
- `check-client-source` reads an existing but unreadable BUILD-STATE as "not declared": WARN, exit 0.
  `[ev: go/internal/checkclientsource/command.go]`
- Allowlist lines compile with Go `regexp.CompilePOSIX`; a GNU-only construct (back-reference, `\w`, `\<`) is exit 3
  now, and a construct both dialects accept with different meanings is not detected.
  `[ev: go/internal/checkbatsskips/allowlist.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-15. `check-installed-line` reads git paths NUL-separated (`git grep -z`), so a group under a quoted path is compared instead of skipped; RED test with a non-ASCII group directory | `scripts/check-installed-line.sh` (Go `internal/checkinstalledline`) | `[ev: go/internal/checkinstalledline/git.go]` |
| Δ2 | APPLIED 2026-11-15. `check-client-source` treats a BUILD-STATE that exists and cannot be read as an env fault (exit 3), not as "not declared" | `scripts/check-client-source.sh` (Go `internal/checkclientsource`) · `BUILD-LOOP.md` § 0.b | `[ev: go/internal/checkclientsource/command.go]` |
| Δ3 | APPLIED 2026-11-15. `tests/bats-skip-allowlist.txt` header states the pattern dialect (POSIX ERE as Go `regexp.CompilePOSIX` reads it) and the check refuses a line using a class or escape outside it | `tests/bats-skip-allowlist.txt` · Go `internal/checkbatsskips` | `[ev: go/internal/checkbatsskips/allowlist.go]` |

## Lessons
- A byte-identical port also ports the fail-open paths; the differential lists them, the retro keeps them from
  being lost.
- Git prints a path it considers unusual in quotes; any tool that feeds git's path output back to git reads it
  NUL-separated.
- A second tool that shells out to git is the moment to share the runner, before a third copies it.

---
**Status**: FOLDED — INDEX row generated from the header: `| 2026-11-13-go-check-scripts-ports.md | kit | 2026-11-13 | folded | 3 |`
