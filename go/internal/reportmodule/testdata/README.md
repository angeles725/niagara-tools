# report-module golden cases

Each `*.txt` file is one run of `report-module` in the `golden.Runner` format
(`go/internal/lint/rules/golden/record/README.md`). `@DIR@` is the case tree (the run's cwd) and `@REPO@` the
repository root. `golden_test.go` replays every case with only the case's environment through a copy of the
toolbelt whose entries are symlinks to the real one; a case's `toolbelt/<member> (mode 755)` sections are stubs
that replace a member. A case that holds `toolbelt/.stub-rest` gets a stub that prints nothing and exits 0 for
every other `*.sh` member, so an orchestration case runs no real lint. The run must leave the tree as it was.

## Recorded cases (every name not starting with `truth-`)

Recorded from the bash tool as it stood before the port: `build-n4-module-kit/toolbelt/report-module.sh` at
commit `a98c3357` (`record.env` pins its sha256). The case families:

| Prefix | Covers |
|---|---|
| `cli-` | usage and pre-check exits (2, 3), `--` handling, every flag forwarded, empty flag values |
| `sum-` | summary pluralization, verdict and exit precedence (env 3 > FAIL 1 > INCOMPLETE 4), `--lint-only`, `--first-deploy`, uncounted statuses |
| `wm-` | the wiring-map sources (flag, `<root>/docs`, `<root>/../docs`), a path holding `&`, the map named in the detail |
| `rows-` | the shared row parser: `site`, `ws`, `base`, `nosite`, `swap` + keep, and the STALE/DRIFT as-WARN relay |
| `verdict-`, `lint-` | the lint verdicts (`-`, `rows`, `fail`), exit 3, a non-zero exit with no row |
| `par-`, `adv-` | the PASS wording after relayed WARN/ADVISORY rows, the advisory members' ERROR reason |
| `gate-` | the artifact-suffix (`-rt`, `-ux`, `-wb`, `-se`) and `src/`, `src/rc/` gates |
| `sc-`, `lt-`, `la-`, `ltf-`, `plano-`, `vm-`, `srv-`, `esl-`, `vf-`, `schema-`, `wp-`, `mm-`, `tc-` | one member family each: slot-coverage, lint-timers, lint-lexicon-ascii, lint-link-target-flags, verify-module --plano and --src, lint-servlet, ESLint resolution and exits, lint-vendor-floor, schema-risk, lint-write-path, the module-root members, triage-console |
| `disc-` | artifact discovery: a symlinked, unreadable, relative, absolute or trailing-slash root, which entries count |
| `fx-` | the `tests/fixtures/report-module` trees with the real members |

To re-record them and prove the port byte-identical:

```sh
git show a98c3357:build-n4-module-kit/toolbelt/report-module.sh > /tmp/rm/report-module.sh
memcap go/internal/lint/rules/golden/record/record.sh /tmp/rm/report-module.sh \
  go/internal/reportmodule/testdata /tmp/rm/out
diff -r go/internal/reportmodule/testdata /tmp/rm/out   # only the truth-* cases and README.md differ
```

`setup.sh` runs the script as `report-module.sh` of the same toolbelt copy; the members it runs are this
checkout's wrappers. The `fx-` cases therefore pin the current lint rows too: a lint change that alters a row
on those trees is re-recorded with it.

### Re-recorded through the Go tool

The modularity campaign M5 (2026-10-27) added the `lint-module-map` member and the `(module)  INFO  modularity`
row. The bash tool at `a98c3357` has neither, so the 104 cases that print the module members were re-recorded
through the Go tool (each case's `want` replaced by the replayed output) and checked by diff: every change is the
new `lint-module-map` row(s) (`PASS ... clean`, plus the real lint's `WARN ... MM1` row in the `fx-` cases), the new
INFO row and the summary line's PASS/WARN counts (verdict and exit unchanged). Their `@kind` still names the bash
recording they started from.

Kit backlog B6 (2026-10-27) added the `lint-vacuous-tests` module member. The 104 cases that print the module
members were re-recorded the same way and checked by a script that compared each old and new `want`: the exit
and stderr are unchanged, the only added lines are one `(module)  PASS|SKIP  lint-vacuous-tests ...` row (80 PASS
`clean` under stubs, 24 SKIP `no *.java under a srcTest/` from the real lint), nothing was removed or reordered
but the summary line, whose PASS/SKIP counts moved by exactly the added rows and whose verdict is unchanged.
`truth-vacuous-tests-member` pins the FAIL relay and `truth-vacuous-tests-envfault` the exit-3 env fault (an exit 3
whose text is not the noSource text is an ERROR row and exit 3; the summary verdict text reads CLEAN there, as for
every member env fault).

## Truth cases (`truth-*`)

Hand-kept expectations that differ from the bash tool on purpose: each one is a declared fix (retro
r10-report-module-go). They were produced through the Go tool after the fix and checked by hand.
