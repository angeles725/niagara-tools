<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-lint-size

**Session**: niagara-research quality round 10 (Ayudante2 lint-engine owner), branch `r10/lint-size-go` from origin/main cb296a9
**Delta count**: 3

## What happened
`toolbelt/lint-size.sh` (bash + one gawk process per file over lib/method-boundary.sh) is now the thin
`n4lint lint-size` wrapper over `go/internal/lint/rules/size`, header, usage, limits, row and `# Mutation:` pins
kept. One engine feature landed first with its table test: `Spec.IntFlags` + `Spec.Configure`, the lint's own
`--max-class <n>` / `--max-method <n>` (its AL_FLAG_HOOK) taken among the leading flags, any order with
`--strict`, last one winning, a bad value printing `<lint>: <flag> needs a non-negative integer` plus the usage
line (exit 3). The rule reuses `javasrc.Strip` and `javasrc.Methods` (the mb_strip / mb_parse ports) and
`linescan.Code`. No declared difference in the port itself: every golden, Cliente and the decompiled tree are
byte-identical in all three argument shapes measured.

Declared change outside the rule: `tests/method-boundary.bats` MB-e2e ran lint-size to pin mb_strip's literal
state; lint-size no longer sources lib/method-boundary.sh, so MB-e2e now counts the same input's code lines through
mb_strip itself (805, and the MB-lit mutation, run, gives 3).

Truth fixes after byte identity, each one commit with the RED observed first (Go golden + bats pin): T1, a
slotomatic marker counts only inside a comment, so a string constant quoting it opens no region (LSZ10); T2,
`@NiagaraType`, `extends B<Upper>` and the class declaration are read in code with literal contents blanked
(LSZ11); T3, the row is named after the first class, interface or enum declaration, as the header promised
(LSZ12). From this branch on, every commit is green: the goldens landed with the port (RED quoted in its message).

## Evidence
- Engine: `TestCommandIntFlags` (defaults, given, with --strict, last wins, missing / negative / empty value, unknown flag first, after `--`, a rule list) `[ev: go/internal/lint/lint_test.go TestCommandIntFlags]`
- Goldens: 43 cases recorded from the bash lint (15 runs of `tests/lint-size.bats`, LSZ-awkfail and LSZ-finderr pinned by `golden.Unreadable`, 28 edge trees: argument and limit shapes, K20, prune, regions, names and the Baja test, methods, bytes); RED 19/43 and TestUnreadable against a stub Check `[ev: go/internal/lint/rules/size/size_test.go TestGolden]`
- Byte identity old vs new (stdout, stderr, exit): Cliente 25 rows (default, `--strict` exit 1) and 84 rows (`--max-class 300 --max-method 40`); sources/decompiled 240 and 1076 rows; all identical; 3.25 s bash vs 0.18 s Go on Cliente, 33.24 s vs 2.13 s on decompiled
- T1: Cliente 25 → 23 (two false `unclosed-region` WARNs on DashboardFord test code that quotes the marker: SourceScan.java:40, FordSlotsTest.java:38), decompiled 240 → 240 `[ev: go/internal/lint/rules/size/testdata/truth-marker-literal-1.txt]`
- T2: Cliente 23 → 23, decompiled 240 → 240 `[ev: go/internal/lint/rules/size/testdata/truth-literal-code-1.txt]`
- T3: Cliente 23 → 23, decompiled 240 → 240, no row text changed `[ev: go/internal/lint/rules/size/testdata/truth-type-name-1.txt]`
- After merging origin/main (#405, javasrc.Methods reads a signature wrapped before its brace): edge-methods-1 and edge-methods-2 gain the `wrapped()` row (M1.java:8, body from its brace line), declared in both goldens; real trees Cliente 23 → 23, decompiled 240 → 253 (13 long methods with wrapped signatures now measured, e.g. template-wb BulkDeployUtil.java:559 getTemplateConfigContents() 498 lines, nre-ext SyslogManager.java:192 writeSyslogConfig() 175 lines) `[ev: go/internal/lint/rules/size/testdata/edge-methods-1.txt]`
- MB-e2e mutation observed: dropping the quote state from mb_strip gives `got=3` `[ev: tests/method-boundary.bats MB-e2e]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | DONE by #405 (merged 2026-10-24). `javasrc.Methods` missed a method whose signature wraps before `{` (`void m(int a,\n int b) {`): a false negative for lint-size's method rule and every method-scoped rule; fixing it changes the shared scanner for all of them (and the lint-ws-placement edge-guard golden on r10/ws-placement-go holds that shape), so it needs its own unit with every rule's goldens re-run | `go/internal/javasrc/javasrc.go` Methods | `[ev: go/internal/lint/rules/size/testdata/edge-methods-1.txt]` |
| Δ2 | A comment holding both marker texts (`// BEGIN … END BAJA AUTO GENERATED CODE`) opens a region that never closes on that line (an unclosed-region WARN); slot-o-matic writes each marker on its own line, so decide whether one line may open and close | `build-n4-module-kit/toolbelt/lint-size.sh` | `[ev: go/internal/lint/rules/size/testdata/edge-regions-1.txt]` |
| Δ3 | Port order for a lint with its own options: land the engine option (Spec.IntFlags) as its own green commit first, then the goldens with the port in one commit | `go/README.md` § Adding a lint rule | `[ev: go/internal/lint/lint_test.go TestCommandIntFlags]` |

## Lessons
- A bats test that runs a migrated lint to pin a bash library loses its teeth when the lint stops sourcing it; move the input onto the library and run the mutation to prove it still fails.
- Running the port over the real trees after each truth fix shows which fixes change real rows (T1) and which only close synthetic shapes (T2, T3); both are reported.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-lints-b`) — Δ1 and Δ3 cross-cutting Go-port lessons, cited by the lints-a fold; Δ2 not folded, needs code (a decision on one-line BEGIN/END), owed in `METHODOLOGY.md` § Kit maintenance (round-10 lint follow-ups) and `BUILD-STATE.md` open_issues. INDEX row generated from the header: `| 2026-10-24-go-lint-size.md | kit | 2026-10-24 | folded | 3 |`
