<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-22 · kit · go-lint-service-types

**Session**: niagara-research quality round 8, issue #286, branch `r8/lint-service-types-go` from origin/main a7dda93
**Delta count**: 1

## What happened
Decision first: **Rule, not TreeRule.** The awk ran once per file (`awk -v FILE=$f … "$f"`), took the own class
from that file and resolved a returned field only through `a = …;` assignments of that same file; nothing
crossed files and rows were printed file by file, so a per-file `lint.Rule` reproduces the report order exactly.
`toolbelt/lint-service-types.sh` (awk `main.awk` over `lib/method-boundary.sh` `mb_strip`) became the rule package
`go/internal/lint/rules/servicetypes` on the shared engine (walker, parser, formatter and comment stripper are the
engine's and `javasrc.File.Stripped()`), and the script a thin `go_tool_bin` wrapper exec'ing
`n4lint lint-service-types`. Issue #286 asked to pin the literal blanker's text-block escape handling: the table
tests TB1-TB7 do, and each of TB1-TB5 flips under a naive blanker (no escape in text blocks, no escape at all, a
single `"` closing a text block, a look-behind `\` test, an escape carried across lines). The bash blanker
was already right on all seven; no bash bug was found there. The header keeps its description, `Usage:`,
`Exit:`, `[ev:]` tags and every `# Mutation:` line; the prose names the Go scanner and the #286 escape rule, so
`toolbelt/INDEX.md` does not change.

## Evidence
- RED: `go test ./internal/lint/rules/servicetypes/` build failed (`undefined: Spec`) before the package existed;
  GREEN: 60 table cases (ST1-ST15 one per bats file, TB1-TB7, Q1-Q12) pass `[ev: go/internal/lint/rules/servicetypes/servicetypes_test.go TestCheck]`
- Every table case was written to its own tree and run through the bash oracle: 60/60 identical rows, which caught
  two wrong `want`s (Q7/Q8: awk `match` reads only the first assignment of a line, now pinned as Q12) `[ev: r8 service-types oracle harness]`
- Text-block teeth: mutations M1 (no escape inside text blocks) fail TB1; M2 (no escapes) TB1+TB5; M3 (one `"` closes a text block) TB1+TB4; M4 (look-behind `\`) TB2; M5 (escape carried to the next line) TB3 `[ev: go/internal/lint/rules/servicetypes/scope.go blockBody]`
- Mutation ST1 in Go (`null`/`new Type[0]` classified unresolved) fails 24 Go cases and bats ST1, ST3, ST9-ST15; restored, `tests/lint-service-types.bats` 17/17 `[ev: tests/lint-service-types.bats]`
- Old-vs-new byte compare (stdout, stderr, exit): 60 table trees, the TB trees with and without `--strict`, a
  mixed 37-row tree, 17 argument/env shapes (no args, two roots, missing dir, a file root, `--`, `-h`, a bad flag,
  `--strict` alone, an empty root, no Java, an unterminated last line, CRLF, a mode-000 file, a mode-000 directory,
  relative and trailing-slash roots) and 11 real trees (9 `Cliente/*` roots with `--strict`, all of `Cliente`,
  `niagara-research/sources/decompiled`) plus the vendor tree `niagara-research/organized` (176,181 files, 10 rows):
  stdout/stderr/exit identical on all 93 runs (vendor: 10 WARN, exit 0, empty stderr in both), no deliberate difference `[ev: r8 service-types differential]`
- Timing on `niagara-research/organized` (4.8 GB, 176,181 `.java`): bash 2134 s (`nice 10`, load 14.4 → 5.1 across the run, one awk fork per file) vs the wrapper 34.7-39.8 s (3 runs, `nice 10`, load 3.5-4.9; ~55 x faster); an earlier wrapper pair at load 12-20 took 46-59 s. The Go time is the engine's walk/read/strip: `n4lint lint-null-context-write` on the same tree takes ~50 s at the same load, this rule adds ~5 s `[ev: r8 service-types timing]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **Run every table case of a ported lint through the bash oracle before trusting its `want` column**: write each case to its own tree, run the old script, compare rows. Why: two hand-written `want`s (Q7/Q8) encoded the intended rule, not what the awk printed. | `go/README.md` § Adding a lint rule (step 3) | `[ev: retro go-lint-service-types]` |

## Lessons
- A per-file awk that resolves only inside its own file is a `lint.Rule`; TreeRule is for reports ordered by check.
- Awk quirks a byte-identical port keeps (a one-line class reads from its first brace, a `;` before the
  declaration on its line skips it, one assignment read per line, no `class` keyword gives an empty own type) are pinned as named Q cases so a later deliberate fix is a visible test diff, not a silent drift.
- Awk regex `.` matches a newline and `match` positions are leftmost-longest; a Go port checks each regex whose
  match span is used (not just whether it matched) and uses byte columns consistently.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-22-go-lint-service-types.md | kit | 2026-10-22 | pending | 1 |`
