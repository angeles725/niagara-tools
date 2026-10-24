<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-24 · kit · r10-citeparse-dotted-section-path

**Session**: niagara-research quality round 10, branch `fix/citeparse-dotted-section-path` from origin/main f9624ea2
**Delta count**: 1

## What happened
The pre-push live cite gate refused a doctrine push with the row
`types/frontend-standard.md 15 path <src + .md> §How to read this file MISSING_FILE`. The card line read
"see `BUILD-STATE.src.md` § How to read this file". The inline cross-reference grammar (`crossRef` in
`go/internal/citeparse/inline.go`) allowed dots in directory segments only, so the leftmost match of the
file-name part began after the last dot but one and the pointer named only the `src` segment plus `.md`. The line was reworded on
main to get the push through.

The last path segment now takes dot-separated parts before `.md`. Table cases in `TestCardSources` pin two,
three and hyphen-plus-two dots, the directory form, the tag forms, a dotted prose name with lines and no
directory (still not a citation) and a version string (still not a path).

## Evidence
- RED before the fix: `go test ./internal/citeparse -run TestCardSources` failed three cases, e.g.
  `got` the bare last-but-one segment, `want Target:BUILD-STATE.src.md`; every other new case passed unchanged.
- `kit-cite-resolve --strict` over the live doctrine set and the full TSV over every tracked kit `*.md` are
  byte-identical before and after the fix: no existing pointer changed reading.
- With the natural wording restored, the live strict run gains one row,
  `types/frontend-standard.md 15 path BUILD-STATE.src.md §How to read this file RESOLVED`, and stays at 0 findings.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | **A pointer grammar matches every character a real file name uses.** A file-name pattern that rejects a character the kit's names hold (a dot inside `BUILD-STATE.src.md`) does not fail: the leftmost-match regex silently reads a shorter, wrong name. Each such grammar pins a case per legal name shape, and a pointer that resolves only after a rewording is a parser bug, not a doctrine edit. | BUILD-LOOP.md §7 | `pointer-grammar-full-names` |

## Fold
- Δ1 → `BUILD-LOOP.md` § 5 Citation pre-filter: a pointer grammar matches every character a real file name uses,
  pins a case per legal name shape, and a pointer that resolves only after a rewording is a parser bug. The code
  fix and its `TestCardSources` pins landed with the retro.

## Lessons
- A workaround rewording hides a parser defect; restoring the natural wording is the regression proof.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-tail`; log in `ledger.d/2026-10-24-r10-fold-tail.md`) — Δ1 cited on the `BUILD-LOOP.md` § 5 Citation pre-filter line.
