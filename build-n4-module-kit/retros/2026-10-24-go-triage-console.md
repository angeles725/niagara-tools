<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 5 -->
# 2026-10-24 · kit · go-triage-console

**Session**: niagara-research quality round 10, branch `feat/go-triage-console` from origin/main 70af46b2
**Delta count**: 5

## What happened
`toolbelt/triage-console.sh` (bash around an embedded awk program, `grep`, `sort` and `cut`) is now the thin
`go/go-tool.sh` wrapper over `go/cmd/triage-console` (`go/internal/triageconsole`), a standalone `cli.Main` command.
The port keeps the bash quirks the goldens record: own rows in gawk `for (k in a)` order through the shared
`rules/awkorder`, awk `-v` escape processing of the package, lists and subject (with gawk's once-per-character
warnings), the `find -H` listing (dot paths, symlinked entries and non-`.txt` names left out, `dir/` and `dir//`
spellings), coreutils `basename`/`dirname` for the subject, the header offsets that assume one blank after the level,
the byte skipped after `]` and after an exception's colon, and `--` dropping every later operand. The goldens replay
through the shared `golden.Runner` (R10-D14, already on main), not a private loop.

The script's default own-tag and module-prefix lists name client modules. The repository is public, so the port
does not copy them into Go: the wrapper keeps the three default lines unchanged and passes them as
`TRIAGE_CONSOLE_DEFAULT_*`; `golden_test.go` reads them from the wrapper, so a drift between the two fails the test.
Run directly, the binary defaults to empty lists.

## Evidence
- 70 goldens recorded from the bash at 79ca5a2c by `golden/record/record.sh`: the 21 runs of triage-console.bats,
  triage-console-site.bats and report-module RM5b (reading the kit fixtures in place through `@REPO@`) and 49 edge
  cases (options, listing, subject, header/stack shapes, channel 3, escapes, a 52-key hash-table growth, site classes
  and their sort); RED (no `Main`) → GREEN, all byte-identical `[ev: go/internal/triageconsole/golden_test.go TestGolden]`
- Named mutations on the port each fail goldens: own rows in insertion order instead of gawk for-in order 17, no dot-path filter 3, a warning per
  escape occurrence 1, a reversed SITE tie key 4, `ids` dropped from the id token 1
- A bash-vs-Go differential over 900 random consoles drawn from the fixture and golden line pool, and 208 runs over
  104 local console directories (7 rows): no difference; those directories took 5.3 s in bash, 1.2 s in Go
- triage-console, triage-console-site, go-wrapper-symlink, kit-links, report-module, script-dir, shell-hygiene,
  pipefail-grep and station-snapshot bats through the wrapper: 134/134 ok

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | DONE (truth): an unreadable console reached awk, whose fatal open skipped the END block: no row at all, exit 0. The port refuses before any row, `triage-console: cannot read <path>: <strerror>`, exit 3 | `go/internal/triageconsole/triageconsole.go` Main | `[ev: go/internal/triageconsole/testdata/truth-unreadable-console.txt]` |
| Δ2 | DONE (truth): the bash filtered its rows with `grep` in the caller's locale; under a UTF-8 locale a row holding a Latin-1 byte (a Spanish console, B800 §800.5) made grep print `binary file matches` instead of the row, exit 0. The port prints the row in any locale | `go/internal/triageconsole/triageconsole.go` rows | `[ev: go/internal/triageconsole/testdata/truth-utf8-locale-non-utf8-row.txt]` |
| Δ3 | DONE (truth): an option without its value died on `set -u` (exit 1, a message naming the script line); a console named like `x=1.txt` was an awk variable assignment, never read (alone: awk read stdin); a NUL byte warned from the command substitution. Now usage exit 3, read as a file, no warning (the printed row still drops the NUL) | `go/internal/triageconsole/triageconsole.go` parse | `[ev: go/internal/triageconsole/testdata/truth-option-no-value.txt]` |
| Δ4 | A port of a tool whose defaults name clients keeps the default lines unchanged in its wrapper and passes them in the environment; the Go test reads them from the wrapper. The public repository gets no new client string and the defaults stay where they were | `BUILD-LOOP.md` § 5 What to port | `[ev: go/internal/triageconsole/golden_test.go shimDefaults]` |
| Δ5 | A golden that mirrors a bats run reads the bats fixture in place (`@REPO@/tests/fixtures/...` in args or a symlink section) instead of copying it into the case: no duplicate to drift, no client string copied | `BUILD-LOOP.md` § 5 golden recordings | `[ev: go/internal/triageconsole/testdata/bats-tr1.txt]` |

## Follow-ups (not done here)
- `is_level` compared `substr(s, 1, 8)` with the 9-byte `"INFORMACI"`, so an INFORMACIÓN line is no header and
  continues the open block (its frames can attach to the previous WARNING). Kept for identity
  (`edge-levels`); a truth fix would make it a level that closes the block.
- `--` drops every later operand (`edge-double-dash-drops-operands`, exit 3); kept for identity.
- The bash sorted a directory listing (`sort -z`) and the SITE tie order in the caller's locale; the port is byte
  order. They agree under C and C.UTF-8, the only locales on the recording host; a collating locale (en_US.UTF-8)
  could order them differently.

## Lessons
- Before porting a pipeline, run it under a UTF-8 locale on non-UTF-8 input: `grep`/`sort` stages that the awk's
  `LC_ALL=C` does not cover can fail open (Δ2).
- `fmt`'s `%c` prints a byte >= 0x80 as a UTF-8 rune; a port of a C `%c` message writes the raw byte.
- Parallel sessions can share one scratchpad directory: record into a per-task subdirectory, never a generic `out/`.

## Fold
- Δ1, Δ2 and Δ3 are truth fixes already shipped in the PR that wrote this retro (`go/internal/triageconsole`, truth-* goldens); not re-implemented, cited in the fold log. Δ2's shell lesson (a `grep` stage in the caller's locale drops a Latin-1 row) also joins the `BUILD-LOOP.md` § 5 silent-input-drops bullet.
- Δ4 → `BUILD-LOOP.md` § 5, the public-repository sentence of the "files outside the repository" bullet (client-named defaults stay in the wrapper).
- Δ5 → `BUILD-LOOP.md` § 5 golden-case bullet (a bats-mirroring golden reads the fixture in place).
- Follow-up `is_level` (INFORMACIÓN never a header) is a truth bug kept for identity: open issue in `ledger.d/2026-10-25-r10-fold-tail3.md`. `--` dropping operands and the locale sort order are kept quirks / recorded limits; no issue.

---
**Status**: FOLDED 2026-10-25 (branch `docs/r10-fold-tail3`; log in `ledger.d/2026-10-25-r10-fold-tail3.md`) — Δ1-Δ3 shipped truth fixes, cited in the fold log (Δ2 also in the § 5 silent-drops bullet); Δ4, Δ5 in `BUILD-LOOP.md` § 5.
