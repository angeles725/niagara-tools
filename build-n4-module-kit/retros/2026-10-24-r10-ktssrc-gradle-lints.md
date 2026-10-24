<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · r10-ktssrc-gradle-lints

**Session**: niagara-research quality round 10, branch `r10/ktssrc-gradle-lints` from origin/main 47a9ac9f
**Delta count**: 2

## What happened
lint-structure L15 read the root `build.gradle.kts` with the Java scanner (`javasrc.Strip` + `BlankLiterals`), which
does not know Kotlin: no nested `/* */`, no `${...}` templates, and a `\` escape inside a raw `"""` string. It now
reads the script with the shared Kotlin scanner `go/internal/ktssrc` (merged in PR #477 for lint-uberjar-api-conflict),
which gained token byte offsets and `Code`, a line-wise code view (comments dropped, literal contents removed) that
the shared signing-block ERE table (`toolbelt/lib/signing-block.sh`, pinned by LS15-sync) is matched on unchanged.
L14 matched its `val gradlePluginVersion = "` pattern on the raw `settings.gradle.kts` lines; it now uses the same code view.
split-package-check reads no `*.gradle.kts` content (its build markers are matched by file name only), so it had
nothing to move.

## Deltas
| Δ | Change | Where | Evidence |
|---|---|---|---|
| Δ1 | L15 on ktssrc: a block in a nested comment or in a template's inner string is not code, a raw `"""C:\"""` ends at its quotes, `id(` split over lines applies the plugin (and `xid(` is not `id(`), a script ktssrc cannot read is the visible L15 WARN. | `go/internal/lint/rules/structure/module.go` + `kotlin_test.go` TestSigningKotlin + `tests/lint-structure.bats` LS15-kotlin | RED: 5 TestSigningKotlin cases failed on the Java scanner; byte-identical on 336 roots (fixtures + all of modulos_niagara_n4, Cliente included) |
| Δ2 | L14 matches its line pattern on the settings script's ktssrc code lines, not the raw text: a literal pin in a `/* */` comment or a raw string is not a pin, one after a leading `/* */` comment is; a script ktssrc cannot read is `cannot scan`, exit 3. | `go/internal/lint/rules/structure/scan.go` `kotlinLines` + `kotlin_test.go` TestPluginPinKotlin + `tests/lint-structure.bats` LS14-block | RED: 4 TestPluginPinKotlin cases failed on the raw lines; byte-identical on the same 336 roots |

## Evidence
- Goldens: `n4lint lint-structure` from origin/main on 336 roots (every module root of modulos_niagara_n4 with
  `*-worktrees` pruned, Cliente's 18 included, plus the kit fixtures and templates): stdout, stderr and exit `cmp`
  identical after the port; 21 L14/L15 rows before and after.
- `ktssrc.TestSweepRealFiles` over modulos_niagara_n4: 60 real build scripts tokenized clean.

## Proposed kit deltas (propose-never-apply)
- None beyond the deltas above.
