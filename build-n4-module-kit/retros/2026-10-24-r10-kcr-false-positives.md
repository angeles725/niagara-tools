<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-24 · kit · r10-kcr-false-positives

**Session**: niagara-research quality round 10, kit-cite-resolve false-positive fix, branch
`fix/kcr-false-positives` from origin/main 11af042f
**Delta count**: 1

## What happened
On origin/main, `kit-cite-resolve` reported 4 strict findings on `odd/tasks/kit-improvement-candidates-2026-10-15.md`,
and all 4 were false positives. A kit-wide run turned up four more classes of the same kind. The parser is
`go/internal/citeparse`, not the resolver package, so the work was re-scoped there. Each case went RED first in the
existing table tests and then GREEN:

1. `code x.jar!entry` (with or without a `/` after the `!`) was UNPARSED. It is now a `jar` pointer: the jar resolves
   like a javap jar (organized/, then the Gradle cache), and the entry inside it is noted as not checked.
2. `§ 11.1-11.3` read as `§11.1 §11.11`, because the section-number pattern took only `-11` as the range end and the
   short end borrowed the prefix. A range is now its two ends, and both are checked.
3. An unclosed `[ev:` inside a code span that closes on its own line wrapped over the next lines. It is now literal
   code. A tag in a code span that runs on to the next line still wraps, as before.
4. Because of case 3, the next line's `[deferred: retro <stem> Δk]` was swallowed. It stays SKIPPED_TEMPLATE.
5. A backticked path with spaces (`Slide Code/…`, `Lab11 NDriver/…`) emitted no row. It now does, and it resolves
   under the course tree too (base, ellipsis search and infix prefix). A lowercase label before a path
   (`code x-rt a/B.java`, `devguide a/b.txt`) or an ellipsis after a space is not read as one spaced path, so
   backticked prose stays unchecked as before.
6. Only the first number of `§ 3, § 5.2, § 7` was checked, and `§§ 3, 5` emitted nothing. Every number is now checked.
   A bare number after a comma counts only under `§§`, so `§ 4.2, 5 retries` stays one section.

## Evidence
- RED then GREEN: `go/internal/citeparse/tag_test.go`, `gaps_test.go`, `source_test.go` and
  `go/internal/citeresolve/paths_test.go` `[ev: code go/internal/citeresolve/paths_test.go]`
- Candidates card, before → after: 302 pointers, strict findings LIVE 4 → 0 (2 UNPARSED jar entries, 2
  MISSING_SECTION §11.11 now RESOLVED).
- Kit-wide run over 579 tracked kit cards, before → after: strict findings LIVE 34 → 26, HISTORY 169 → 167, and no
  row went from passing to failing. The first GREEN run did add 4 MISSING_FILE rows for course paths with spaces.
  That is why the course tree became a path base, and the label rule of case 5 came from the backticked
  `code x-rt …` prose it would have read.
- `covered()` treated a directory under the research root as already indexed whenever the repository root was its
  parent (the fixture layout). It now asks whether the deepest tree root holding the directory is a real tree
  `[ev: code go/internal/citeresolve/lookup.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | kit-cite-resolve reads `x.jar!entry` (jar resolved, entry unchecked), section ranges as two ends, `§`/`§§` section lists, spaced backticked paths (course tree included), and an unclosed code-span tag as literal | `go/internal/citeparse/source.go`; `go/internal/citeparse/inline.go` | `[ev: code go/internal/citeparse/source.go]` |

## Lessons
- A parser fix measured only on the card that exposed it can add false positives elsewhere. A kit-wide before/after
  diff of row keys caught 4 new MISSING_FILE rows and backticked label prose before commit.
- Allowing a new character in a pattern (here, a space) widens what counts as a pointer. Each wider shape needs
  a negative test for the prose it must still reject.

## Owed
- The spaced-path rule is lexical: a capitalised or numbered first word opens a path. A capitalised label
  before a path (`Foo a/b.java` in backticks) would still be read as one spaced path. No kit card has one today.

## Fold
- Δ1 is code already shipped in the PR that wrote this retro (`go/internal/citeparse`, `go/internal/citeresolve`); not re-implemented. Its behavior and both lessons (kit-wide before/after row-key diff; a negative test per wider shape) → `BUILD-LOOP.md` § 5 "Citation pre-filter" bullet.
- The owed capitalised-label case stays lexical; no kit card has one, so no issue is opened.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-tail2`; log in `ledger.d/2026-10-24-r10-fold-tail2.md`) — Δ1 shipped code, cited in the `BUILD-LOOP.md` § 5 Citation pre-filter bullet.
