<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · r10-report-module-go

**Session**: niagara-research quality round 10, branch `r10/report-module-go` from `r10/golden-runner` 2d1172d1
**Delta count**: 3

## What happened
`toolbelt/report-module.sh` (998 lines of bash and awk) was ported to Go as `go/internal/reportmodule` and
`go/cmd/report-module`, byte for byte first. The script is now the `go_tool_bin` wrapper; it exports its
unresolved directory as `REPORT_MODULE_TOOLBELT`, so the members still run through `$TOOLBELT/<member>.sh` and
the bats stub toolbelts keep working. The package split: `args.go` (the flag loop), `rows.go` (the ROW_AWK port
`parseRows`, gawk's 2+-whitespace split and the shell's `st|check|detail` record split), `members.go` (the
declarative per-artifact and per-module member tables), `special.go` and `eslint.go` (the members with their own
rows), `run.go` (the subprocess runner, bash's `command -v`, `[ -d/-f/-x ]`) and `report.go` (emit, summary,
exit precedence).

- **Goldens.** 116 cases recorded with the shared `golden/record/record.sh` from the bash at a98c3357 (sha256
  pinned), each under `env -i`: CLI and pre-checks, summary and exit precedence, wiring-map sources, every row
  mode, the verdicts, every special member, discovery edges and the 21 `tests/fixtures/report-module` module
  trees with the real members. A case holding `toolbelt/.stub-rest` stubs every other member, so orchestration
  cases cost a fraction of a second instead of 1.6 s of real lints. `TestGolden` replays them through
  `golden.Runner` with the whole post-run tree.
- **Byte identity.** 116/116 goldens. Bash and Go on the 22 `tests/fixtures/report-module` entries, each with and
  without `--lint-only`, plus `clean` with `--console-dir`: 45 runs, stdout, stderr and exit identical.
- **lintindex.** INDEX.md's Auto column grepped `report-module.sh` for `"$TOOLBELT/<lint>"`; it now reads the Go
  member table (`reportmodule.Scripts`), and the generated INDEX.md is byte-identical (49 lints Auto yes).
- **Completeness (#459).** `TestEveryLintIsAMemberOrListed`: every `rules.All()` rule and every
  `toolbelt/lint-*.sh` is a member or in `notInReport` with a reason (bacnet-status-flags, manual-labels,
  guard-pins, split-package-check). No lint was missing.
- **Bats.** 73 of 76 kept and green; RM45, RM-row-parser and RM-exit-capture grepped the bash source and were
  retired (their Go equivalents: no shell arrays, `TestParseRows`, the runner's return value).
- **Truth fixes**, each its own commit with a `truth-*` case and the RED in its body:
  - A lint that exits non-zero (not 3) with no FAIL row is a FAIL row. `lint_rows`, lint-write-path and
    lint-lexicon-ascii failed open (exit 127 read PASS "clean"); only lint-link-target-flags failed closed.
  - lint-timers WARN rows (job-cancel, jdk-thread, atSteadyState-only-timer) are relayed; the bash kept only
    PASS*/FAIL* lines.
  - A dot-directory under the module root is not an artifact (a `.git` gave floor SKIPs: INCOMPLETE).
  - verify-module checks the module jar, not a `-sources`/`-javadoc` sibling picked by directory order.
  - slot-coverage output with no `pct=` line is an ERROR, not `PASS dup-keys 0` + `WARN slot-coverage %`.
- **Kept for parity** (judged, not bugs or needing a product decision): `docs/` and `build/` still count as
  artifacts (which non-dot directories are profile artifacts is a product decision); a lexicon without a mixin
  (or the reverse) gets no slot-coverage row; ESLint exit 1 with only WARN rows stays a FAIL (fail-closed is
  right; only the "no parsable row" wording is loose); triage-console rows keep no `(run)` prefix and its
  non-row lines are printed uncounted (RM5b reads them); the line-0 site-mode detail was not reproduced with a
  shipped rule. The bash < 4.4 empty-array hazard (:730) is gone with the bash body.
- **Declared port differences** (no shipped member reaches them): artifacts sort in byte order whatever the
  locale; member output past 64 MiB is dropped and named; NUL bytes drop without bash's warning; bash's own
  "No such file or directory" line for a missing member is not in its output; a non-integer `pct` reads WARN
  without bash's stderr diagnostic.

## Evidence
- `go test ./internal/reportmodule/` (119 golden cases: 113 recorded, 6 truth; `TestParseRows`, `TestFirstJar*`,
  `TestEveryLintIsAMemberOrListed`) and `go test ./internal/lintindex/` (`TestScanAutoFromMemberTable`)
  green `[ev: go/internal/reportmodule/testdata]`
- `tests/report-module.bats` 73/73 (RM31, RM33 skip: no acorn, no eslint installed) `[ev: tests/report-module.bats]`
- Fixture parity (local, not committed): 45/45 runs identical before the truth fixes `[ev: retro r10-report-module-go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A port whose members stay subprocesses records orchestration cases against no-op member stubs (`toolbelt/.stub-rest`) and keeps the real members for the fixture cases only | `go/README.md` § Adding a lint rule (step 3) | `[ev: go/internal/reportmodule/testdata/setup.sh]` |
| Δ2 | Any member exit other than its success codes with no FAIL row is a FAIL (or ERROR) row, in every member shape, never PASS "clean" | `toolbelt/report-module.sh` header (fail-closed rule) | `[ev: go/internal/reportmodule/members.go]` |
| Δ3 | A table of the members an aggregator runs is pinned against every rule and lint script by a completeness test with an explicit not-run list | `go/README.md` § Adding a lint rule | `[ev: go/internal/reportmodule/completeness_test.go]` |

## Lessons
- A fail-closed rule enforced in one member shape is not a rule: the header said "never PASS" and four of five
  shapes still read a crashed lint as clean.
- A relay that filters by status prefix drops whatever status the member gains later; lint-timers grew WARN rows
  and the report never showed one.
