<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-11-14 · kit · go-scaffold-module-port

**Session**: "all tool logic in Go" pass, wave 2: `toolbelt/scaffold-module.sh` becomes a launcher over `go/cmd/scaffold-module`.
**Delta count**: 4

## What happened
The scaffolder was ported to Go against a differential of 114 rows (stdout, stderr, exit code, `diff -r` of the
emitted tree and a type/mode listing): 106 identical, 8 declared. The declared rows are the places where the
bash tool reported success over a broken result, which the port refuses instead. Four gaps stay open (two raised by the native review of the port).

## Evidence
- The bash tool passed `--vendor`, `--target-version` and `--plugin-version` to sed as REPLACEMENT text. An
  unescaped `|`, a newline, a trailing `\` or `\1`-`\9` made sed fail about 45 times; the tool left a broken tree
  and exited 0. The port exits 2 with one line and writes nothing (4 rows), and refuses the GNU-only case
  escapes (2 rows). `[ev: 8ea57144]`
- The replacement semantics sed accepted are reproduced on purpose, so `--vendor 'A&B'` still yields
  `AAngelesB`: `&` expands to the replaced token. `go/internal/scaffoldmodule/sedrhs.go`. `[ev: 8ea57144]`
- A read-only out-dir made the bash tool print 49 errors and exit 0 with "emitted"; the port exits 3 (1 row).
  `[ev: 8ea57144]`
- Under `bats --jobs 4` the three `PF-jarlock-*` cases of `tests/preflight.bats` failed; the file alone passed
  40 of 40 twice. `[ev: 4159818f]`

- The lowercased `--vendor` value becomes a directory name of the emitted Java package
  (`go/internal/scaffoldmodule/subst.go`, fixturePathRules), so a value with `/` or `..` writes outside the
  module root; the bash tool did the same through sed on paths. `[ev: 8ea57144]`
- A write fault mid-copy leaves the partial tree in place (the next run then refuses with "already exists"), and
  no Go test injects that fault: only the differential row covers it. `[ev: 8ea57144]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-15. Option values are literal text: `--vendor 'A&B'` emits `A&B`, and a value with `|`, `\` or a newline is either emitted as typed or refused by name; the sed-replacement emulation (`sedrhs.go`) is deleted with its tests | `go/internal/scaffoldmodule` (subst, sedrhs) · `toolbelt/scaffold-module.sh` header | `[ev: 8ea57144]` |
| Δ2 | APPLIED 2026-11-15. The `PF-jarlock-*` cases are safe under a parallel bats run (own lock path and port per case), or the file declares itself serial; the parallel batch rule names the file meanwhile | `tests/preflight.bats` · `CONTRIBUTING.md` parallel batch rules | `[ev: 4159818f]` |
| Δ3 | APPLIED 2026-11-14 (b0723ce9). `--vendor` is validated before anything is written: its lowercased form must be one Java package segment (no `/`, `..`, whitespace or leading digit), else exit 2 naming the value | `go/internal/scaffoldmodule` (args, name) | `[ev: 8ea57144]` |
| Δ4 | APPLIED 2026-11-15. A failed emit removes the partial `<out-dir>/<ModuleName>` it created (and says so), with a Go test that injects the write fault | `go/internal/scaffoldmodule/command.go` · `copytree.go` | `[ev: 8ea57144]` |

## Lessons
- A script that pipes user text into sed's replacement side has a second, undocumented input language.
- "Exit 0 and a broken tree" is found only by comparing the emitted tree, not the exit code.
- A test suite is parallel-safe only if it was run in parallel at the baseline; record that baseline once.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-11-14-go-scaffold-module-port.md | kit | 2026-11-14 | pending | 4 |`
