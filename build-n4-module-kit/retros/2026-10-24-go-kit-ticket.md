<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-kit-ticket

**Session**: niagara-research quality round 10, branch `feat/go-kit-ticket` from origin/main 1a086dd1
**Delta count**: 3

## What happened
`toolbelt/kit-ticket.sh` (182 lines of bash) is now the thin `go/go-tool.sh` wrapper over `go/cmd/kit-ticket`
(`go/internal/kitticket`). The port keeps every observable of the bash: the gh-absent SKIP path (exit 0), the gh
body built from the --from retro's "What happened" and "Proposed kit deltas" sections, the LOCAL (exit 4) / FAIL
(exit 3) rows that keep the ticket file, the unquoted `--repo` word splitting and globbing, and the messages the
bash shell, mkdir, cat, grep, mv and awk printed on a filesystem failure (the wrapper passes its own path as
`KIT_TICKET_SCRIPT` for the `<script>: line N:` prefix). gh is looked up on the run's PATH and inherits the
run's environment without the two wrapper variables. retro-loop.bats RL4 runs the script under a PATH without
go, find or readlink; it now links find and readlink (gh stays absent) and its `setup_file` builds the binary
through `go_tool_bin` with the full PATH, so the wrapper stays fail-closed on the staleness check.

## Evidence
- 55 goldens replayed by the shared `golden.Runner` (PR #516): 52 recorded from the bash at 5179bf23 by
  `golden/record/record.sh` and 3 truth-* cases that only rename the bash's `$$` temp files to the injected
  pid 4242 `[ev: go/internal/kitticket/golden_test.go TestGolden]`
- Gh-present cases run a fake gh from the case tree (`ghbin/gh`) that echoes its argv and cats the body file, so
  the body, the argv (incl. `--repo "a  b"` and `--repo "g*"`) and the exit classification are all pinned; no
  case PATH holds the host gh `[ev: go/internal/kitticket/testdata/setup.sh]`
- RED (stub Command): 55/55 cases fail; GREEN: 55/55 byte-identical plus multi-line title slugs and glob rules a
  golden cannot spell, each checked against the bash `[ev: go/internal/kitticket/kitticket_test.go TestSlugMultiLine]`
- retro-loop.bats (RL4 and the RL4b-RL4f gh-stub cases) and kit-links.bats 31/31 ok through the wrapper; without
  find on PATH the wrapper exits 3 naming the failed staleness check `[ev: tests/retro-loop.bats RL4]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A bats test that runs a Go-wrapped script under a restricted PATH builds the binary in `setup_file` with the full PATH and links `find`/`readlink`**: the wrapper's staleness check is fail-closed and must stay so | `BUILD-LOOP.md` § 5 What to port | `[ev: tests/retro-loop.bats RL4]` |
| Δ2 | **Pin an external command's call with a fake in the case tree that echoes its argv and the files it is handed**, never the host binary on PATH: the host gh sat in /usr/bin, the default case PATH | `go/internal/lint/rules/golden/record/README.md` § Case set | `[ev: go/internal/kitticket/testdata/setup.sh]` |
| Δ3 | **A recording that leaks the bash `$$` becomes a truth-* case with the pid replaced by the one the Go test injects** (`Host.Pid`), stated in its @kind, instead of being dropped | `go/internal/lint/rules/golden/record/README.md` § Case file | `[ev: go/internal/kitticket/testdata/truth-gh-tickets-is-file.txt]` |

## Follow-ups (not done here)
- GNU mkdir/mv quote a path holding a quote or a control character in shell-escape style; the port always prints
  `'path'`. No kit path holds one.
- The bash ran a non-executable fallback gh as a shell script on ENOEXEC; the port reports it as not runnable (126).

## Lessons
- Read the whole bats file before scoping a port: the gh-present kit-ticket cases (RL4b-RL4f) sit 140 lines below RL4.
- A restricted-PATH test is the first thing a Go wrapper breaks; check every `PATH=` run of the script before writing the shim.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-24-go-kit-ticket.md | kit | 2026-10-24 | pending | 3 |`
