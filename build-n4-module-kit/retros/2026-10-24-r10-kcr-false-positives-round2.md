<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-24 · kit · r10-kcr-false-positives-round2

**Session**: niagara-research quality round 10, kit-cite-resolve false positives round 2, branch
`fix/kcr-fp-round2` stacked on `feat/kcr-anchor-drift` (bc0f0345)
**Delta count**: 1

## What happened
A read-only triage named six more false-positive classes (G3-G8). Each was checked against the kit before any
code changed. Four were real and went RED first in the existing table tests, then GREEN:

1. G3: an elided placeholder in an unparsed clause (`[deferred: retro … Δk]`, `…Δ1`, `retro YYYY-MM-DD-…`,
   `corpus B…`) was UNPARSED. It is now SKIPPED_TEMPLATE. An ellipsis inside a word (`a1b…c2d`) still elides nothing.
   This was the only LIVE strict finding on the branch.
2. G5: the tag clause `javap x.jar[!entry] [<ver>]` (and `code javap x.jar …`) was UNPARSED, or read `javap` as a
   module. It now resolves the jar the way the prose `javap of <jar>` form does. A class with no jar
   (`javap BWebServlet 4.14`) stays UNPARSED ("no jar named").
3. G6: a path relative to niagara-help (`source/…`, `guides-clean/…`, `devguide-clean/…`) read the unversioned
   4.14 copy first, so a 4.15 line range landed on other text. It now reads `versions/<ver>/` first, as the guide
   trees already did.
4. G7: `[ev: code Labs/<lab> NDriver/x.zip!gradle.properties:5,8]` split the spaced path at its first space and
   came back MISSING_FILE. A `<path>.zip!<entry>` after `code` or `course` is now an `archive` pointer: the zip is
   found under the card, kit, repo, research or course trees, and the entry is reported as not checked. No zip is
   opened.

Two classes did not match the kit, so they were stopped and reported instead of fixed:

- G4 (`§a / b / c` as a list): no card in the kit has such a list. The four ` / ` titles in the history are single
  titles of headings that are missing as a whole and as parts. Several real kit headings contain ` / `
  (`## Zero-demand / idle state`), so a split would only add risk.
- G8 (line-less backticked spaced path): no row is emitted by design. `source_test.go` pins "a backticked path
  without lines is not a citation". The spaced paths that carry `:lines` already resolve.

## Evidence
- RED then GREEN: `go/internal/citeparse/refine_test.go` (G3), `source_test.go` (G5, G7),
  `go/internal/citeresolve/paths_test.go` (G6) and `kinds_test.go` (G7) `[ev: code go/internal/citeresolve/kinds.go]`
- Kit-wide run over 585 cards, before → after: strict findings LIVE 1 → 0, HISTORY 167 → 160. By row key:
  UNPARSED → SKIPPED_TEMPLATE 6, UNPARSED → RESOLVED 2, HISTORICAL_DRIFT → RESOLVED 2, and no row went from
  passing to failing. G6 changed only excerpts: five RESOLVED rows now quote the 4.15 copy, and on one live card the
  excerpt of an anchored guide line is now the anchored text itself.
- No history doc was edited to make a row pass.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | kit-cite-resolve reads elided placeholders as templates, the `javap` tag clause, niagara-help paths version-first, and `x.zip!entry` as an archive pointer (zip resolved, entry unchecked) | `go/internal/citeparse/refine.go`; `go/internal/citeparse/source.go`; `go/internal/citeresolve/kinds.go` | `[ev: code go/internal/citeparse/refine.go]` |

## Lessons
- A triage class is a hypothesis until a kit card shows it. Two of six classes (G4, G8) had no real instance, or
  had a pinned test saying the behaviour was intended. Grepping the kit and the pinned tests first kept them out
  of the code.
- Putting the versioned niagara-help copy first changed no status, but it did change excerpts. A row-key diff that
  compares only status hides this, so the excerpt column was diffed too.

## Owed
- The `archive` kind checks only that the zip exists. Opening it to check the entry and its lines would need a zip
  reader in the resolver, and no kit card needs that today.

## Fold
- Δ1 is code already shipped in the PR that wrote this retro (`go/internal/citeparse`, `go/internal/citeresolve`); not re-implemented. Its shapes and both lessons (a triage class is a hypothesis until a card shows it; diff excerpts too) → `BUILD-LOOP.md` § 5 Citation pre-filter bullet.
- G4 and G8 were rejected in the retro itself with reasons; the owed zip-entry check has no kit card that needs it; no issue.

---
**Status**: FOLDED 2026-10-25 (branch `docs/r10-fold-tail3`; log in `ledger.d/2026-10-25-r10-fold-tail3.md`) — Δ1 shipped code, cited in the `BUILD-LOOP.md` § 5 Citation pre-filter bullet.
