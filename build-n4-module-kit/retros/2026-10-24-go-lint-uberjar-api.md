<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · go-lint-uberjar-api

**Session**: niagara-research quality round 10 (Ayudante2, Go lint-engine owner), branch `r10/uberjar-go` from origin/main 11c976e8 (origin/main merged once, 8e5c3b97)
**Delta count**: 2

## What happened
`toolbelt/lint-uberjar-api-conflict.sh` (grep/sed/comm over each `*.gradle.kts`) was ported onto the Go lint engine as
`go/internal/lint/rules/uberjarapi`, byte for byte first, then two RED-first truth fixes on a new shared Kotlin
scanner, `go/internal/ktssrc` (`Tokens`: strings with `$x`/`${...}` templates, raw `"""` strings, nested `/* */`).

- **Port.** 15 goldens from the bash (3 bats runs, 12 edge trees) and `TestUnreadable`. Mode `Raw`,
  `ContinueOnWalkError`, `EmptyOK` (the bash `AL_EMPTY_OK`, ER5) and `ShortScanError`. Declared: an unreadable
  `*.gradle.kts`, which the bash `grep 2>/dev/null` skipped (exit 0), is `cannot scan`, exit 3.
- **T1 tokens.** Commented-out and string-held `api(...)` no longer make a conflict; calls split over lines or with
  blanks inside the parentheses are found; `xapi(` is not `api(`. A script that is not Kotlin (an unterminated
  string or comment) is a scan error, exit 3.
- **T2 vals.** `uberjar(jackson)` with `val jackson = "g:a:v"` is resolved like the literal.

## Evidence
- Byte identity before the fixes (stdout, stderr, exit, `cmp`): all of Cliente (39 `*.gradle.kts`, 0 rows) and all
  of modulos_niagara_n4 (0 rows) `[ev: go/internal/lint/rules/uberjarapi/testdata]`
- Timing: Cliente 0.12 s → 0.02-0.04 s; modulos_niagara_n4 0.43 s → 0.04-0.05 s (host loaded) `[ev: retro go-lint-uberjar-api]`
- `ktssrc.TestSweepRealFiles`: 60 real build scripts tokenized clean `[ev: go/internal/ktssrc/sweep_test.go]`
- Truth fixes: 0 real-tree rows before or after (no client script both bundles and declares one library);
  golden edge-shapes 11 rows → exit 3 (mismatched quotes), truth-shapes 7 rows `[ev: tests/lint-uberjar-api-conflict.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A coordinate built from templates in its group or artifact (`"$group:lib:1"`) or from a version catalog (`libs.jackson`) is not resolved; resolve catalog aliases from `gradle/libs.versions.toml` if a client adopts one | `go/internal/lint/rules/uberjarapi` | `[ev: go/internal/lint/rules/uberjarapi/uberjarapi.go]` |
| Δ2 | Other `*.gradle.kts` lints (lint-structure L15 signing block, split-package-check) match regexes on raw text; move them onto `ktssrc.Tokens` when they are next touched | `go/internal/lint/rules/structure`, `splitpackage` | `[ev: go/internal/ktssrc/ktssrc.go]` |

## Fold
- Δ1 → deferred (waits for a client that adopts templates or a version catalog): `BUILD-LOOP.md` § 5 owed-code line; open issue in `ledger.d/2026-10-24-r10-fold-late.md`.
- Δ2 → `BUILD-LOOP.md` § 5 One layered Go lint stack: `internal/ktssrc` `Tokens` is the one Kotlin-script reader; L15 and split-package-check move onto it when next touched.

## Lessons
- A recorded edge tree can hold a line the target language rejects (`uberjar("g:mix:1')`); once the rule reads
  real tokens it fails closed there, so the valid shapes need their own truth case.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-late`) — INDEX row: `| 2026-10-24-go-lint-uberjar-api.md | kit | 2026-10-24 | folded | 2 |`
