<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-22 · kit · go-sweep-build-state

**Session**: niagara-research quality round 8, branch `r8/sweep-build-state-go`, rebased onto origin/main 6d5d12c
**Delta count**: 1

## What happened
`toolbelt/sweep-build-state.sh` (the content half of the pre-push retro gate and two CI steps: `grep -c` marker
counts, a `while read` envelope parser, per-row `echo | grep | awk` INDEX parsing, `sed | grep | awk` marker reads,
`date -d` ages, and two sub-shells into `gen-retro-index.sh --check` / `gen-build-state.sh --check`) became
`go/cmd/sweep-build-state` over a new parser package `go/internal/buildstatefields` (envelopes and `key: value`
fields, INDEX data rows, the review-status marker, the disposition skip), and the script a thin wrapper over
`go/go-tool.sh` `go_tool_bin`. The two freshness checks now call `go/internal/retroindex` and
`go/internal/buildstate` in process (read-only reuse, no change there), with the generators' own stderr and the
same `sweep:` messages. Capturing the old output first exposed `while read` dropping an unterminated last line
twice: a BUILD-STATE.md ending in an unterminated close marker passed with an envelope missing `retro_pending`
(silent pass; `grep -c` still counted the marker, so the markers looked balanced), and an unterminated last INDEX
row was skipped (in `--age` its debt was not counted; in the content check its retro then read "has no INDEX row").

## Evidence
- Oracle = origin/main's bash `sweep-build-state.sh` (unchanged since 96f8bc2) with main's generator wrappers. Old-vs-new
  over 81 edge shapes (usage and missing/unreadable inputs, every envelope field rule, CRLF, prose and stray markers,
  INDEX row parsing, the 5-line marker window, generated/stale/misplaced INDEX, fresh/stale/malformed/unreadable
  ledger.d, misnamed state, every `--age` flag shape, dispositions, impossible and future dates, `00` deltas) plus 71
  real-tree runs (3 argument spellings of the CI content check, `--age` at 4 todays x 2 `--max-age`, and both modes
  on 30 past revisions of BUILD-STATE.md + retros/ + ledger.d/): stdout, stderr and exit identical for 144/152, all
  71 real-tree runs byte-identical `[ev: r8 sweep-build-state byte-identity harness]`
- The 8 differences, all deliberate: unterminated close marker exit 0 → 1 (`missing required field:
  retro_pending`); unterminated last INDEX row now parsed (content: the row's own status error instead of "no
  INDEX row"; `--age`: counted); an unreadable BUILD-STATE.md exit 1 with `unbalanced ( open / close)` → exit 3
  `cannot read`; `--today garbage` and `--max-age x` exit 0 with a `date`/`[` error and a wrong report → exit 3
  naming the bad value; an unreadable retro's `sed:` note worded by Go (same exit); and the dispositions case below.
  One shape outside the harness: a sweep copied without its sibling `gen-*.sh` no longer exits 3 (`no generator`),
  the generators are in process `[ev: r8 sweep-build-state byte-identity harness]`
- The deltas cell is read by the shared `go/internal/retrocells.ParseDeltasCell` (T17, #314), not a local copy:
  decimal and fail closed, so `00` is zero (bash compared the string to `0` and aged the row) and an unreadable cell
  (`0 0`, `none`) still ages. Deliberate fix R3-002 (#226 triage): an allowed disposition value is a whole token
  (`tracked-by-issue #<n>` or `worklist-complete`, ending at whitespace, `-->` or the line end), so
  `worklist-completed` or `tracked-by-issue #9x` ages. Harness case `age-retrocells-and-token`: bash
  total_pending=2 → Go 3; both real dispositions are whole tokens, so no real-tree output changed
  `[ev: go/internal/buildstatefields/buildstatefields.go DispositionSkips]`
- `go/internal/buildstatefields` table tests port the S1-S9 / M1-M6 sweep scenarios of `tests/build-retro-sync.bats`,
  GRI-sweep, GBS-sweep and D1-D9 of `tests/retro-debt.bats`, plus the edge shapes; RED (package missing, build
  failed) → GREEN, and RED first for R3-002 and the retrocells cases; those bats files, `tests/gen-retro-index.bats`,
  `tests/gen-build-state.bats` and `tests/lint-fork-per-line.bats` unchanged 31/31, 9/9, 15/15, 7/7, 5/5 (the fork
  baseline lists `lint-*.sh` only, so it has no sweep row to delete) `[ev: go/internal/buildstatefields/command_test.go]`
- Mutation GBS-sweep (skip the in-process ledger check) fails Go TestMainLedger and bats GBS-sweep / H14; mutation
  BSF-pending (drop the retro_pending presence check) fails 4 Go tests but no bats test: bats S4 still exits 1 via
  the boolean check `[ev: go/internal/buildstatefields/buildstatefields.go CheckEnvelopes]`
- Real kit, load average ~9: content check bash ~4.4 s/call (3 calls), wrapper ~22 ms/call; `--age` bash ~3.0 s/call,
  wrapper ~14 ms/call (20 calls); earlier at load ~18-22: bash ~14.8 s vs wrapper ~40 ms `[ev: r8 sweep-build-state timing]`
- The wrapper resolves its own directory with `readlink -f` (`here="$(cd "$(dirname "$(readlink -f
  "${BASH_SOURCE[0]}")")" && pwd)"`), the canonical form being ratcheted across the toolbelt.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A gate that delegates to another kit tool calls its Go package in process, never its wrapper script: same messages, no sibling-script lookup, no second build or process.** Why: the bash sweep needed `gen-*.sh` beside it (exit 3 otherwise) and paid two sub-shells per push. | `CONTRIBUTING.md` § 2.2 Go module | `[ev: retro go-sweep-build-state]` |

Δ1 is applied in this unit for sweep-build-state only; the retro stays pending until a fold review decides the rule.

## Lessons
- `grep -c` and `while read` disagree on an unterminated last line: one counts it, the other never sees it, so a
  balance check over one and a parse over the other can pass a broken file.
- A bats test that asserts only the exit code can be satisfied by a different rule than the one it names (S4
  passes through the boolean check); pin the message when the rule matters.

---
**Status**: PENDING (Δ1 applied to sweep-build-state; fold review pending) — INDEX row generated from the header: `| 2026-10-22-go-sweep-build-state.md | kit | 2026-10-22 | pending | 1 |`
