<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-23 · kit · r9-lintindex-header

**Session**: niagara-research quality round 9 (Ayudante3), branch `r9/lintindex-header` stacked on `r9/guardpins-header` (PR #338)
**Delta count**: 1

## What happened
`gen-lint-index` (`go/internal/lintindex`) read a lint's header from lines 1-60 only, the same fixed window that
made `lint-guard-pins` skip 8 pins (retro r9-guardpins-header). `lint-timers.sh` has a 128-line comment header, so
six `[ev:]` tags on its lines 61-128 never reached `toolbelt/INDEX.md`. The two readers also defined "header"
twice. Both now use one helper, `go/internal/lintheader` `Lines`: the leading run of lines starting with `#`
(the definition lintindex already used for Usage/Exit lines), with no line cap. A blank line ends the header; no
current lint has one inside its header, so guard-pins output is unchanged by that choice.

## Evidence
- RED: new `lintheader` tests (package missing) and flipped lintindex cases (Usage past line 60 is read; the first
  non-comment line ends the header, so an `[ev:]` tag or `# Exit` comment in code is not read)
  `[ev: go/internal/lintheader/lintheader_test.go TestLines]` `[ev: go/internal/lintindex/lintindex_test.go TestExtract]`
- GREEN: `go test ./...` ok; `bats tests/gen-lint-index.bats tests/guard-pins.bats` ok
- Mutant: `Lines` without the `#` stop fails 11 subtests across lintheader, lintindex and guardpins
- `toolbelt/INDEX.md`: only the `lint-timers.sh` row changes, gaining `[ev: corpus B801] [ev: corpus B812]
  [ev: retro n4-course-threading-jobs-doctrine Δ2] [ev: retro n4-course-handson-services-history-threading Δ3]
  [ev: retro n4-course-threading-jobs-doctrine Δ6] [ev: audit-2026-10-03 A8]`
- Contract-violation stderr now says `(the leading comment block)` instead of `(lines 1-60)`
- Survey before the change: no lint has a blank line inside its leading `#` run, and none has an `[ev:]` tag in a code
  line within lines 1-60, so dropping code-line tags changes no row

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Every reader of a lint header goes through `go/internal/lintheader` `Lines`; a second header definition is a defect | `go/internal/lintheader/lintheader.go` | `[ev: go/internal/lintheader/lintheader_test.go TestLines]` |

## Lessons
- Two tools reading "the header" with their own copies drift apart; one helper keeps the definition in one place.
- A fixed window failed the same way in two tools; when one is found, search for its siblings in the same change set.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-23-r9-lintindex-header.md | kit | 2026-10-23 | pending | 1 |`
