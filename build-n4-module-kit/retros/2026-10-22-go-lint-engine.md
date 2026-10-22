<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-22 · kit · go-lint-engine

**Session**: niagara-research quality round 8, work unit T8k-2a, branch `r8/t8k2a-n4lint` from origin/main 96f8bc2
**Delta count**: 2

## What happened
The kit's Java lints are bash + awk scripts that each run one awk per file on top of `lib/awk-lint.sh` and
`lib/scan-files.sh`. This unit added the modular Go lint engine of layout R8-D5 and ported the first lint onto it.
`go/internal/javasrc` gained `File` (path, raw lines, strip and method results computed once).
`go/internal/lint` is the engine: the `Rule` interface, one `Finding` type with one row formatter, `Parse` with the
four `awk_lint_parse` modes, `Walk` (the fail-closed walk of `scan-files.sh`) and `Run` (exit 0 / 1 under
`--strict` / 3 env). `go/internal/lint/rules/nullcontext` holds only the rule and its table tests, and
`go/cmd/n4lint` is a thin binary. `toolbelt/lint-null-context-write.sh` is now a wrapper that keeps its header and
execs `n4lint lint-null-context-write`. Issue #293 is closed: (1) the null-Context branch is bounded to the
get/resolve call's own parentheses (an intended behavior change), (2) `resolve(base)`/`resolve()` have tests,
(3) the awk parameter shadowing is moot because the awk is gone, and (4)/(5) `types/logic-authoring.md` now has the
`BFoxProxySession` release step and one recommended ORD form per case.

## Evidence
- RED: the engine, rule and cmd tests failed to build before the code existed; with the old unbounded
  `,[[:space:]]*null[[:space:]]*\)` match restored, the two I293-1 table cases fail; the new bats case
  NCW-bql-null-scope fails against the origin/main script `[ev: go/internal/lint/rules/nullcontext/nullcontext_test.go]`
- Byte identity, old script (git archive 96f8bc2) vs wrapper, stdout + stderr + exit: same on no args, `-h`,
  `--strict <tree>`, a file root, a missing root, an unreadable file, an unreadable sub-directory, CRLF, no final
  newline and a symlinked root; same rows on tests/fixtures (127 files), Cliente/Honeywell (43 rows),
  Cliente/panccadia-leon (0), Cliente/Leon-Guanjuato (1), and vendor smoke trees fox-rt (42) and baja (190 rows,
  901 files). The only difference is the #293(1) row `foo(BOrd.make("bql:x").get(base, cx), null);`
  `[ev: issue #293]`
- Timing, old vs new, one run each: Honeywell 4.50 s → 0.17 s, panccadia-leon 3.56 s → 0.13 s, Leon-Guanjuato
  0.87 s → 0.10 s, baja vendor tree 15.77 s → 0.22 s `[ev: go/internal/lint/run.go]`
- The unreadable-file message keeps gawk's own text (`awk: fatal: cannot open file ... for reading: ...`) and the
  walk message keeps find's (`find: '<dir>': Permission denied`), so stderr stays byte-identical under gawk
  `[ev: go/internal/lint/lint_test.go]`
- A stubbed toolbelt symlinks every script from another directory, so the wrapper resolves its own path with
  `readlink -f` before locating `go/` `[ev: tests/helpers/stub-toolbelt.bash]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A Go lint port reuses internal/lint and lists every intended difference.** A port adds one `internal/lint/rules/<name>` package and one `rules.All()` line, keeps the bash header in the wrapper, and proves byte identity (stdout, stderr, exit) against the old script on the bats fixtures and real trees, listing each changed row. Why: the engine already carries the walker, parser and formatter; a copy in a rule package would bring back the ten-copy skeleton problem of `lib/awk-lint.sh`. Reviewer check: a rule package that walks files, parses arguments or formats rows is a finding. | `go/README.md` § Adding a lint rule; `BUILD-LOOP.md` §5 | `[ev: go/internal/lint/run.go]` |
| Δ2 | **Wrappers resolve symlinks before locating go/.** A toolbelt wrapper that execs a Go binary computes `go/` from `readlink -f "${BASH_SOURCE[0]}"`, not from the invoked path. Why: `stub-toolbelt.bash` links the scripts into a temp dir, and a path-relative `../../go` points nowhere there. Reviewer check: a new Go wrapper without symlink resolution is a finding. | `toolbelt/gen-retro-index.sh` pattern; `go/README.md` | `[ev: tests/helpers/stub-toolbelt.bash]` |

## Lessons
- Port an intended fix as a separate RED table case, so the byte-identity diff has exactly one explained row.
- Error texts from find and awk are part of a lint's output contract; a port keeps them or lists them.
- A Go lint is 20-70x faster than one awk per file, so `--all` over one walk is worth the shared engine.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-22-go-lint-engine.md | kit | 2026-10-22 | pending | 2 |`
