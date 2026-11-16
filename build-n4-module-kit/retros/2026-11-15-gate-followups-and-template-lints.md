<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 17 -->
# 2026-11-15 · kit · gate-followups-and-template-lints

**Session**: phase B unit "kit gates and templates": six follow-ups an independent verifier left on four gate tools, and the template rows of a research retro.
**Delta count**: 17

## What happened
An independent verifier read the merged change "check scripts and vendor-floor fail closed" and left six
follow-ups (rows V1-V6 of the unit, Δ1-Δ6 here): untested sub-guards, constructs two regular-expression engines
read differently, and four inputs that read as "nothing to check" with exit 0. A read-only research pass over
template archives proposed three kit rows (K1-K3, Δ7-Δ9 here): a lint for the `.ntpl` files a module ships, a
station-template rule, and a card section on what a template upgrade keeps. All nine are applied in this unit.
The rest of the table is what the work found and did not fix.

## Evidence
- Differential, old tree against new, per touched tool: every difference is one a row names (the unit's report
  lists the inputs). `[ev: go/internal/checkbatsskips/dialect.go]`
- Each of the three bracket sub-guards of Δ1, neutralized by hand in a scratch copy, leaves only
  `TestMainRefusesEachBracketSubGuard` red. `[ev: go/internal/checkbatsskips/dialect_test.go]`
- GNU grep 3.11 reads `x{,2}` as the interval 0 to 2, refuses `a{}` ("Invalid content of \{\}") and refuses
  `[:alpha:]` ("character class syntax is [[:space:]], not [:space:]"); Go compiles all three as literals or a
  set. `[ev: go/internal/checkbatsskips/dialect.go]`
- git 2.56 answers `git show <ref>:<path>` with "ambiguous argument ... both revision and filename" when the
  working tree holds an entry of that name; with a closing `--` it prints the blob.
  `[ev: go/internal/checkinstalledline/git.go]`
- A report over an artifact whose `src/rc/vendor` is a dangling symlink printed no `vendor-floor` row and the
  verdict CLEAN before Δ4. `[ev: go/internal/reportmodule/testdata/truth-vf-dangling-vendor.txt]`
- The twelve guards of the new lint, each neutralized by hand in a scratch copy, turn the test that names it
  red. `[ev: go/internal/lint/rules/ntplmanifest/ntplmanifest_test.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-15. V1: `check-bats-skips` pins the three bracket sub-guards no test covered, one smallest line each (`[[:]`, `[]a\]]`, `[[=alpha=]]`, all exit 3) | Go `internal/checkbatsskips` `dialect_test.go` | `[ev: go/internal/checkbatsskips/dialect_test.go]` |
| Δ2 | APPLIED 2026-11-15. V2: `check-bats-skips` refuses, as outside the dialect and by name, `{,2}`, `{}` and a class name in single brackets (`[:alpha:]`); an unescaped brace must open `{n}`, `{n,}` or `{n,m}`; the allowlist header states it and the committed list passes unchanged | `tests/bats-skip-allowlist.txt` header · Go `internal/checkbatsskips` `dialect.go` | `[ev: go/internal/checkbatsskips/dialect.go]` |
| Δ3 | APPLIED 2026-11-15. V3: `lint-vendor-floor` exits 3 (`env fault: cannot look for vendor/ under <rc>: <reason>`) on an rc dir it may not search; an rc dir with no `vendor/`, or with a file of that name, keeps its note and exit 0 | `toolbelt/lint-vendor-floor.sh` header · `BUILD-LOOP.md` § 5 · Go `internal/vendorfloor` `walk.go` | `[ev: go/internal/vendorfloor/walk.go]` |
| Δ4 | APPLIED 2026-11-15. V4: `report-module` reads `src/rc/vendor` without following it and lets `lint-vendor-floor` judge a link that leads nowhere: the report shows `ERROR  vendor-floor  env fault (exit 3)`; no recorded golden changed, one truth case added | Go `internal/reportmodule` `run.go`, `special.go`, `testdata/README.md` | `[ev: go/internal/reportmodule/testdata/truth-vf-dangling-vendor.txt]` |
| Δ5 | APPLIED 2026-11-15. V5: `check-client-source` reads a BUILD-STATE with CRLF line endings (the CR is not part of a value) and takes `--build-state ""` for a usage error (exit 2) unless `--declared` wins | Go `internal/checkclientsource` `envelope.go`, `args.go`, `command.go` | `[ev: go/internal/checkclientsource/envelope.go]` |
| Δ6 | APPLIED 2026-11-15. V6: `check-installed-line` closes `git show <ref>:<path>` with `--`, so a working-tree entry of that name never hides a group, and prints a path holding a control character, a double quote or a backslash quoted with C escapes, one row per group | Go `internal/checkinstalledline` `git.go`, `quote.go` | `[ev: go/internal/checkinstalledline/git.go]` |
| Δ7 | APPLIED 2026-11-15. K1: lint G92 `toolbelt/lint-ntpl-manifest-deps.sh <module-root>` FAILs a shipped `.ntpl` without `template.bog` or `template-manifest.xml`, a manifest that is not well-formed XML and a `<dependency>` no `*.gradle.kts` of the module declares; bounded reads, unsafe entry names refused; a module with no `.ntpl` is one PASS row | `toolbelt/lint-ntpl-manifest-deps.sh` · Go `internal/lint/rules/ntplmanifest` · `types/logic-authoring.md` § Shipping a template module | `[ev: go/internal/lint/rules/ntplmanifest/ntplmanifest.go]` |
| Δ8 | APPLIED 2026-11-15. K2: no verify step read station templates, so the rule is the lint's sub-check `ntpl-station`: a `.ntpl` under a `newStations/` directory whose manifest root lacks `isStation="true"` FAILs | `types/distribution.md` § 8 · Go `internal/lint/rules/ntplmanifest` | `[ev: go/internal/lint/rules/ntplmanifest/ntplmanifest.go]` |
| Δ9 | APPLIED 2026-11-15. K3: card section "What a template upgrade keeps and what it wipes" (deploy is Workbench client code; upgrade is remove-then-redeploy; the station needs the template service; the manifest signature is no code signature), stated as read from 4.14 code and not re-verified on 4.15 | `types/logic-authoring.md` § What a template upgrade keeps and what it wipes | `[ev: build-n4-module-kit/types/logic-authoring.md]` |
| Δ10 | `verify-module`'s phantom-dep check calls `lint.DeclaredDeps` instead of its private `declaredDeps` (the two hold the same rule today; the file was outside this unit's surface), and `go/README.md` lists the reader | Go `internal/verifymodule` `srcmore.go` · `go/README.md` | `[ev: go/internal/lint/gradledeps.go]` |
| Δ11 | `report-module` runs `lint-ntpl-manifest-deps` as a module member (FAIL rows relayed, its empty-root text as the SKIP reason); it is in `notInReport` now because joining re-records every real-member golden | Go `internal/reportmodule` `members.go`, `completeness_test.go`, `testdata/` | `[ev: go/internal/reportmodule/completeness_test.go]` |
| Δ12 | Decide how `ntpl-deps` treats a stock module: a station template lists every module its station uses, and the rule FAILs each one the module's build does not declare. Options: a `--niagara-home` whose `modules/` are taken as present, a declared allowlist file in the module, or WARN for a template under `newStations/` | `toolbelt/lint-ntpl-manifest-deps.sh` · `types/distribution.md` § 8 | `[ev: go/internal/lint/rules/ntplmanifest/declared.go]` |
| Δ13 | The other half of the first G92 spec is not built: a module the bog (`m="alias=module"`) or a Px entry uses with no `<dependency>` row, a template module `.jar` as input, the guide's `template.manifest.xml` spelling as a fallback name, and `.napl` application archives | `toolbelt/lint-ntpl-manifest-deps.sh` · `ledger.d/2026-10-23-r8-u10-platform-templates.md` | `[ev: build-n4-module-kit/ledger.d/2026-10-23-r8-u10-platform-templates.md]` |
| Δ14 | The header of `scripts/check-installed-line.sh` states the quoted row path and the `--` of Δ6 (the script was outside this unit's surface; the Go package comment and the bats file state them) | `scripts/check-installed-line.sh` header | `[ev: go/internal/checkinstalledline/quote.go]` |
| Δ15 | The `open_issue` of the 2026-10-23 fragment still lists G92 as owed whole; narrow it to the half of Δ13 | `ledger.d/2026-10-23-r8-u10-platform-templates.md` | `[ev: build-n4-module-kit/ledger.d/2026-10-23-r8-u10-platform-templates.md]` |
| Δ16 | `check-bats-skips` still accepts `^*a`: Go reads a repeated anchor, GNU grep 3.11 warns "* at start of expression" and reads a literal `*`. Refuse a `*` right after `^`, `(` or a bar as outside the dialect | Go `internal/checkbatsskips` `dialect.go` · `tests/bats-skip-allowlist.txt` header | `[ev: go/internal/checkbatsskips/dialect.go]` |
| Δ17 | Re-verify the section of Δ9 on a 4.15 install (the upgrade job's save and restore list, which module holds deploy, whether a station without the Workbench template module can upgrade) and drop its caveat or correct it | `types/logic-authoring.md` § What a template upgrade keeps and what it wipes | `[ev: build-n4-module-kit/types/logic-authoring.md]` |

## Lessons
- "Not found" and "could not look" are different answers: a presence test that follows a link, or that reads
  every stat error as absence, turns the second into a pass.
- A dialect statement is only as good as the list of constructs the two engines were asked about; ask both
  engines, with the real binary, before writing the list.
- A value that goes back to git as `<ref>:<path>` is closed with `--`; git otherwise asks the working tree what
  the argument means.
- A new lint meets four registries at once (the rule list, the root-rule contract test, the report's
  member-or-listed test, the guard-pin check); read them before choosing the rule's kind and its empty-root
  answer.
- A row that prints a path taken from a repository or an archive prints it quoted when it holds a control
  character: one row, one line.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-11-15-gate-followups-and-template-lints.md | kit | 2026-11-15 | pending | 17 |`
