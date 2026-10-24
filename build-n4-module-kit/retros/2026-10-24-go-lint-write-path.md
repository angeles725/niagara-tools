<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-lint-write-path

**Session**: niagara-research quality round 10 (Ayudante2, Go lint-engine owner), A2-17, branch `r10/write-path-go` from origin/main 9a2b660 (origin/main merged once, 170f5ddc)
**Delta count**: 3

## What happened
`toolbelt/lint-write-path.sh` (563 lines: bash, two awk programs, a python3 `--bog` helper) was ported onto the
Go lint engine as `go/internal/lint/rules/writepath`, byte for byte first, then its false results were fixed one
RED-first commit each. The `--bog` part reads the bog through `go/internal/xmlsrc` (R8-D12), not regexes.

- **Engine.** `Mode` `Options` (`<root>` with `--strict` and `Spec.ValueFlags` anywhere), `lint.OptionRule`
  (`CheckRootOptions`), `lint.RowError` (an env error printed as a stdout row, exit 3), `Finding.Status` (STALE,
  DRIFT rows that count as WARN), and `lint.Dirname/Basename/IsFile/IsDir` for RootRules.
- **Port.** 77 goldens recorded from the bash script: every lint-write-path.bats / write-path-drift.bats run the
  golden format can hold, the report-module write-path fixtures and 44 edge trees. Go tests hold the unreadable
  directory/file and symlinked-root cases. The bog walk replays the python helper's state machine over the
  xmlsrc element tree (a childless element read as self-closing, `</a>` popping a frame).
- **Declared at the port.** A malformed or unreadable `--bog` (the helper's `except: sys.exit(0)` and
  `2>/dev/null`) and an unreadable Java or matrix file are now exit 3 instead of a silent clean scan.
- **T1 matrix row.** `grep -q '^\|'` is "^ or empty" in GNU BRE, so any non-empty matrix stopped the walk-up.
- **T2 noeol.** The per-row `while read` dropped a last matrix line without a newline.
- **T3 code.** The slot scanner and the name harvest read raw lines across one `xargs` run: commented-out
  annotations and comment OPERATORs were write paths, a literal `(` swallowed the next annotations and files.

## Evidence
- Byte identity before the fixes (stdout, stderr, exit, `cmp`): 33 runs, 11 Cliente module roots × (plain,
  `--bog` the zipped PANCCADIA config.bog, `--strict --bog` its plain file.xml); 5 roots carry bog-linked FAIL
  rows `[ev: go/internal/lint/rules/writepath/testdata]`
- Timing per run: 0.12-13.3 s → 6-56 ms (host loaded) `[ev: retro go-lint-write-path]`
- Truth fixes: 0 real-tree rows changed (33/33 runs identical after T1-T3); goldens changed: T1 2 (+1 new), T2 1
  (1 → 2 STALE), T3 1 (-2 +4 FAIL) `[ev: tests/lint-write-path.bats]`
- Fork-per-line ratchet: the `lint-write-path.sh _mline 1` row is gone (declared improvement) `[ev: tests/fixtures/lint-fork-per-line/baseline.txt]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | The header says `auditLog.record(` satisfies the audit-call check, but only the identifier right before `(` counts (`record`), so that call WARNs; decide whether a receiver named *audit* counts, then fix the code or the example | `toolbelt/lint-write-path.sh` header | `[ev: go/internal/lint/rules/writepath/audit.go]` |
| Δ2 | `rules/structure` keeps private `dirname`/`isFile`/`isDir`; move it to `lint.Dirname`/`IsFile`/`IsDir` | `go/internal/lint/rules/structure/scan.go` | `[ev: go/internal/lint/paths.go]` |
| Δ3 | `lint-arbitrary-ord.sh` still names "lint-write-path.sh audit" as a sibling awk stripper; it is now `linescan.Code` | `toolbelt/lint-arbitrary-ord.sh` comment | `[ev: go/internal/lint/rules/writepath/audit.go]` |

## Lessons
- A grep pattern can lie by its dialect: `^\|` reads as a literal-pipe test but is an alternation in GNU BRE;
  a recorded edge tree (a heading-only matrix) caught what every bats fixture, each with a real table, could not.
- The golden format holds files only, so an empty directory (WP9b's src-less profile) needs a placeholder file
  in the recorded tree, and a tree whose basename is printed needs a stable name.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-lints-b`) — the Go port is noted on the `METHODOLOGY.md` § Kit maintenance lint-write-path line; Δ1, Δ2 and Δ3 not folded, needs code (an audit-receiver decision, moving structure's path helpers to `lint`, a `lint-arbitrary-ord.sh` comment outside this fold's surface), owed in `METHODOLOGY.md` § Kit maintenance (round-10 lint follow-ups) and `BUILD-STATE.md` open_issues. INDEX row generated from the header: `| 2026-10-24-go-lint-write-path.md | kit | 2026-10-24 | folded | 3 |`
