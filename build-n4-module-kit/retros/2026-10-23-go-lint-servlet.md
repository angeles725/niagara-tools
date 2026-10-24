<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-23 · kit · go-lint-servlet

**Session**: niagara-research quality round 8, unit E3b, branch `r8/e3-lint-servlet-go` from origin/main d83f1bb
**Delta count**: 1

## What happened
Decision first: **Rule, not TreeRule.** The awk ran once per file and printed that file's rows (per handler,
then catch-no-400, input-400, cache-nofinger, csrf-xrw-only) before the next file, so a per-file `lint.Rule`
keeps the report order. `toolbelt/lint-servlet.sh` (292 lines, its own awk and its own `find -H` walk, not
`scan_files`) became the rule package `go/internal/lint/rules/servlet` and a thin `go_tool_bin` wrapper
exec'ing `n4lint lint-servlet`. Its contract needed four engine features, each one field or severity with a
doc and a table test: `Info` rows (never affect the exit), `Spec.StrictFails` (the awk printed WARN rows as
FAIL under `--strict`), `Spec.RelPaths` (rows name `${file#"$root/"}`) and `Mode` `Trailing` (`<src-dir>
[--strict]...`, `unknown option` before the directory check). Byte identity came first (commit 7334490); three
truth fixes followed, each seeded RED in `TestTruth`: T8 csrf-xrw-only anchored on a Javadoc line (truth
audit 2026-10-22), every check read comments and literal contents as code, and catch-no-400 never read the
usual `} catch (E e) {` on the try's closing line. The header now names catch-no-400, which the awk ran but
the header never listed.

## Evidence
- 45 golden cases (46 after re-recording from main at the merge) recorded from the bash lint through a recording wrapper (15 bats runs + 30 edge runs:
  every check, `--strict`, 11 argument shapes, trailing slashes, sort order, handler shapes, unbalanced
  braces, comments and literals, CRLF, no final newline, an empty file, UTF-8 and 0xff bytes)
  `[ev: go/internal/lint/rules/servlet/servlet_test.go TestGolden]`
- Byte identity at 7334490/1078ff6 (stdout, stderr, exit): the bats fixture dir plain/`--strict`/trailing
  slash, 4 argument shapes, 7 `Cliente/*` roots plain and absolute `--strict`, the two `*-worktrees` roots
  with `--strict`, `niagara-research/sources/decompiled` (2,706 files, 18 rows); all of `Cliente` differs only
  by the 58 rows under `*-worktrees` sub-directories, which the engine prunes `[ev: r8 e3 servlet differential]`
- Truth fixes, real-tree rows: T8 drops `BChiServlet.java:41` (chihuahua; the guard is in
  `ChiServletDispatch`) and nothing else; code-only reading changes no real row; the closing-line catch adds
  6 catch-no-400 rows in the decompiled vendor tree (`WebServer.java:921`, `CryptoServlet.java:235`,
  `DiagnosticServlet.java:65`, `FileServlet.java:702/728/748`, each a parse with a silent default) and none in
  `Cliente` `[ev: go/internal/lint/rules/servlet/servlet_test.go TestTruth]`
- Timing: all of `Cliente` 2.80 s → 0.10 s; `sources/decompiled` 15.7 s → 1.9 s (load 15-27)
  `[ev: r8 e3 servlet timing]`
- `tests/lint-servlet.bats` 15 → 20 (added LSV-empty, LSV-dotroot, LSV4b, LSV2c, LSV2d; no expectation
  changed); report-module 66/66, go-wrapper-symlink 4/4, lint-fork-per-line 5/5, kit-links 12/12
  `[ev: tests/lint-servlet.bats]`

## Declared differences from the bash lint
Measured against the bash lint on origin/main at the merge, which by then walked with
`lib/scan-files.sh` `scan_files_nonempty_or_exit` (dot and `*-worktrees` sub-directories pruned, the root
never pruned, a walk error or an empty root exit 3 with the K20 row): the port keeps all of that
(`Spec.EmptyError` carries the K20 text, `ContinueOnWalkError` unset), and the goldens were re-recorded from
that script (46 cases, identical apart from the truth fixes). Against the original private-`find` script
the port also differed on an empty root, a dot-named root and `*-worktrees`, as main itself now does.
- An unreadable source file prints the engine env line and exits 3 after scanning the other files; bash
  printed gawk's error and kept its 0/1 exit.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **Before porting a lint, record its goldens from the bash on the current origin/main, and re-record at the merge**: main moved lint-servlet from a private `find` (which also pruned a dot-named start point, so `.` passed clean) onto `scan_files_nonempty_or_exit` while the port was in flight, which changed its empty-root, walk-error and prune contract. | `go/README.md` § Adding a lint rule (step 5) | `[ev: retro go-lint-servlet]` |

## Lessons
- A lint whose bash rewrote its own severity column under `--strict`, printed relative paths or parsed
  options after the root needs engine fields, not a private formatter in the rule package.
- Header names (`X-Requested-With`, `Cache-Control`) are string literals, so a code-only lint keeps literal
  text for them and blanks literal contents only for code shapes (calls, braces, try/catch).
- `chihuahua` keeps its X-Requested-With guard in `ChiServletDispatch`, which declares no handler, so the
  lint does not read it at all: the T8 fix removes a false row but cannot report that guard.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-23-go-lint-servlet.md | kit | 2026-10-23 | pending | 1 |`
