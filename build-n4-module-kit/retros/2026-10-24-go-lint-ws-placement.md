<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-lint-ws-placement

**Session**: niagara-research quality round 10 (Ayudante2 lint-engine owner), branch `r10/ws-placement-go` from origin/main 6f58289
**Delta count**: 3

## What happened
`toolbelt/lint-ws-placement.sh` (bash + one gawk process per file over lib/method-boundary.sh) is now the thin
`n4lint lint-ws-placement` wrapper over `go/internal/lint/rules/wsplacement`, header, usage, row and `# Mutation:`
pins kept. No new engine field: `Each` mode, `ContinueOnWalkError` (awk_lint_run scans the partial list) and the
K20 row text through `Spec.EmptyError.Msg`; the method scope is `javasrc.Methods` and the code view `javasrc.Strip`
(the mb_parse / mb_strip ports), row trimming `linescan.Trimmed`. The awk's split_args, cast stripping, creation
shapes, value-type exclusion, per-method guard and local tracking are ported byte for byte.

Declared difference of the port (byte identity otherwise): an empty root among several stops the rule at its K20
row (the engine's Spec.EmptyError contract); awk_lint_run printed the row and went on to the later roots. The exit
code (3) is the same, and report-module passes one root.

Truth fixes after byte identity, each RED first (Go golden + bats pin): T1, add( calls, their argument structure
and local assignments are matched with literal contents masked (new `javasrc.MaskLiterals`, BlankLiterals with the
removed bytes blanked so indexes stay valid), so a quoted add, a quoted assignment or a text block is not code
(WSP-literal); T2, a `new B<Type>` whose type the file imports by name from javax.baja.ui or javax.baja.workbench
is a bajaui widget, a Workbench view child, never a wire-sheet node (WSP-widget); T3, `new BFoo[n]` is an array, not
a component (WSP-array). go/README.md also gained the retro go-lint-status-parity Δ2 note (a per-file rule may set
`Finding.Path` to any row subject).

## Evidence
- Goldens: 36 cases recorded from the bash lint (13 runs of `tests/lint-ws-placement.bats`, WSP-awkfail and WSP-finderr pinned by `golden.Unreadable`, 23 edge trees: argument shapes, K20, prune, add() shapes, value types, wrapping, locals, guard, literals, bytes) RED against a stub Check, then green `[ev: go/internal/lint/rules/wsplacement/wsplacement_test.go TestGolden]`
- Byte identity old vs new (stdout, stderr, exit), default and `--strict`: Cliente (10 rows, exit 0 / 1) and sources/decompiled (411 rows, exit 0 / 1) identical; 2.98 s bash vs 0.26 s Go on Cliente, 30.96 s vs 2.45 s on decompiled
- T1: Cliente 10 → 10, decompiled 411 → 411 `[ev: go/internal/lint/rules/wsplacement/testdata/truth-literals-1.txt]`; `TestMaskLiterals` pins the length-keeping mask `[ev: go/internal/javasrc/blank_test.go TestMaskLiterals]`
- T2: Cliente 10 → 4 (the six Apillm `buttons.add(null, new BButton(...))` rows go), decompiled 411 → 123 `[ev: go/internal/lint/rules/wsplacement/testdata/truth-widgets-1.txt]`
- T3: Cliente 4 → 4, decompiled 123 → 123 `[ev: go/internal/lint/rules/wsplacement/testdata/truth-array-1.txt]`
- After merging origin/main (#405, javasrc.Methods reads a signature wrapped before its brace): edge-guard-1 gains one row, BNested.java:20 `add("e", new BFolder()) in wrapped()`, the unplaced add of the wrapped method the bash mb_parse never saw; declared in the golden. Real trees after the merge: Cliente 4 → 4, decompiled 123 → 125 (lonworks-rt BDeviceEntryTable.java:58 and BRouterEntryTable.java:71, `add(entryName, new B…Entry())` in wrapped addEntry() methods) `[ev: go/internal/lint/rules/wsplacement/testdata/edge-guard-1.txt]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | The guard is a substring match on the Strip view: a log message quoting `wsAnnotation` or a name such as `hasWsGridX` silences a method (false negative); decide whether a guard must be a code token or the literal `"wsAnnotation"` | `build-n4-module-kit/toolbelt/lint-ws-placement.sh` | `[ev: go/internal/lint/rules/wsplacement/testdata/edge-guard-1.txt]` |
| Δ2 | Widget detection reads explicit imports only: a same-package or com.tridium widget (`new BTemplateManager(`, `new BLineChart(` in decompiled) and a wildcard `import javax.baja.ui.*` still WARN; a type-hierarchy or receiver check would close it | `build-n4-module-kit/toolbelt/lint-ws-placement.sh` | `[ev: go/internal/lint/rules/wsplacement/testdata/truth-widgets-1.txt]` |
| Δ3 | A rule that needs literal text and code positions together reads `javasrc.MaskLiterals` (same line lengths) instead of mapping BlankLiterals columns back; say so beside `linescan.Code` in § Adding a lint rule | `go/README.md` § Adding a lint rule | `[ev: go/internal/javasrc/blank_test.go TestMaskLiterals]` |

## Lessons
- The first Each-mode awk_lint_run port showed the engine's K20 stop differs from `_al_list` (which continues); it is a declared difference rather than a new Spec field because no other unported lint has that shape.
- A real-tree diff of the port's rows (6 of 10 Cliente rows were bajaui buttons) found the largest truth defect; the synthetic edge trees found the rest.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-lints-b`) — Δ1 and Δ2 not folded, needs code (a guard-token decision, a type-hierarchy widget check), owed in `METHODOLOGY.md` § Kit maintenance (round-10 lint follow-ups) and `BUILD-STATE.md` open_issues; Δ3 cross-cutting Go-port lesson, cited by the lints-a fold. INDEX row generated from the header: `| 2026-10-24-go-lint-ws-placement.md | kit | 2026-10-24 | folded | 3 |`
