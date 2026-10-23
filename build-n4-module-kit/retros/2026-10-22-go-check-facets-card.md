<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-22 · kit · go-check-facets-card

**Session**: niagara-research quality round 8, branch `r8/facets-card-go` from origin/main c30a19a
**Delta count**: 1

## What happened
The third Go port: `toolbelt/check-facets-card.sh` (awk + sed + grep + three `while read` loops) became
`go/cmd/check-facets-card` (`go/internal/facetscard`) and the script a thin wrapper over `go/go-tool.sh`
`go_tool_bin`, the gen-retro-index pattern. Capturing the old output before the swap exposed one silent pass in
the bash version: when the last line of a `.java` source is a key constant with no final newline, `sed` emits it
unterminated and `while read` drops it, so the key is never checked as MISSING nor compared for ORDER and a card
that lacks it still reported `N keys match`. The port reads that line like any other.

## Evidence
- Old-vs-new over 34 argument shapes (the 6 bats scenarios, usage with 0/1/3 args, missing/empty/directory/
  `/dev/null`/mode-000 paths, empty-string and `--help` paths, duplicates, CRLF card and keys, a repeated card
  heading, odd Java spacing, a non-`.java` source with Java lines): stdout, stderr and exit identical for 32/34;
  the 2 differences are the unterminated-`.java` cases above (`1 keys match` exit 0 → `MISSING MAX=max` exit 1)
  `[ev: r8 facets-card byte-identity harness]`
- `go/internal/facetscard` table tests port every bats input plus the edge shapes; RED (package missing) → GREEN
  `[ev: go/internal/facetscard/command_test.go TestMain]`
- Mutations FC-order (drop the positional compare) and FC-empty (drop the no-card-rows guard) fail both the Go
  tests and `tests/check-facets-card.bats` FC-order / FC-empty `[ev: go/internal/facetscard/facetscard.go Compare]`
- Live card, 50-call loop on a loaded host: bash ~78-162 ms/call, wrapper ~10-14 ms/call, binary ~2-3 ms/call
  `[ev: r8 facets-card timing]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A byte-identical port reproduces every documented behaviour but not a silent pass: capture old vs new on edge inputs first, and fix and record any case where the old tool reports clean on bad input.** Why: `while read` drops an unterminated last line, so the bash checker passed a card missing that key. | `CONTRIBUTING.md` § 2.2 Go module | `[ev: retro go-check-facets-card]` |

Δ1 is applied in this unit for check-facets-card only; the retro stays pending until a fold review decides the rule.

## Lessons
- `while IFS= read -r k` skips a final line without a newline; any bash loop over `sed`/`cut` output of a user file
  can drop its last record.
- Capture the old tool's output on edge inputs before replacing it; the diff is both the equivalence proof and the
  bug finder.

---
**Status**: FOLDED 2026-10-23 (branch `docs/r8-u1-tool-doctrine`; log in `ledger.d/2026-10-23-r8-u1-tool-doctrine.md`) — Δ1 `CONTRIBUTING.md` § 2.2 "Port in two steps".
