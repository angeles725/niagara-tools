<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-22 · kit · go-sweep-fold-audit

**Session**: niagara-research quality round 8 T17, branch `feat/r8-t17-foldaudit-go` from origin/main e3713cf
**Delta count**: 2

## What happened
`toolbelt/sweep-fold-audit.sh` (find + grep + sed + sort + awk per INDEX row) became `go/cmd/sweep-fold-audit`
(`go/internal/foldaudit`) and the script a thin `go/go-tool.sh` `go_tool_bin` wrapper, the gen-retro-index
pattern. The port is byte-identical by default, including the bash quirks the differential exposed: a deltas cell
with a leading zero is octal in bash arithmetic (`010` checks Δ1..Δ8 only), an `08`/`09` cell aborts the Δ loop
with a bash error on stderr and the row is never Δ-checked, an INDEX last row without a final newline is never read
(`while read`), and an unreadable INDEX prints a bash error and audits zero rows with exit 0, even under `--strict`.
All four pass bad input as clean; they are reported, and fixed in a separate follow-up commit.
A third commit, on the integrator's #226 triage, moves the deltas-cell parser into `go/internal/retrocells`
(`ParseDeltasCell`, decimal, error on an unreadable or >999 cell) for the sweep-build-state port to share, and
counts an unreadable cell on its own line instead of as a "not cited" Δ.

## Evidence
- Old `.sh` vs Go binary over 30 named argument shapes (real tree `--strict --deltas-since 2026-09-24`, no-flag,
  four other cutoffs, the bats fixtures F1-F11, usage/env errors, an edge kit with NUL bytes, invalid UTF-8, a
  binary corpus file, symlinked root with/without a trailing slash, CRLF, octal cells, an unterminated last row,
  a mode-000 INDEX) plus 300 seeded random kits: stdout, stderr and exit identical in every case
  `[ev: r8 T17 fold-audit differential]`
- Real tree, 5 runs, median: bash 5391 ms (`--strict --deltas-since`) / 3993 ms (no flag) → wrapper 27 ms / 18 ms
  `[ev: r8 T17 fold-audit timing]`
- Mutations F3, F4, F6, F9 fail both the Go tests and `tests/sweep-fold-audit.bats`
  `[ev: go/internal/foldaudit/command_test.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A byte-identical port keeps the old tool's fail-open quirks in its first commit and fixes them in a separate, droppable commit with a regression test each.** Why: the differential is the equivalence proof only while both sides behave the same; mixing a fix into the port hides which diff is which. | `CONTRIBUTING.md` § 2.2 Go module | `[ev: retro go-sweep-fold-audit]` |
| Δ2 | **A bash tool that does arithmetic on a numeric cell from a file must force base 10 (`$((10#$n))`).** Why: a leading zero makes bash read the cell as octal, and `08`/`09` abort the loop with a stderr message while the row passes unchecked. | `toolbelt/` authoring rules | `[ev: retro go-sweep-fold-audit]` |

Δ2 is OBSOLETE (quality round 8 W23 U0, main e66da0a): the bash tool it guarded is gone; `toolbelt/sweep-fold-audit.sh` execs the Go port, whose deltas cell is parsed by `go/internal/retrocells` `ParseDeltasCell` as decimal ("a leading zero is not octal", `toolbelt/sweep-fold-audit.sh` header). Δ1 stays pending: no kit doc states the quirks-first, fixes-in-a-separate-commit rule yet.

## Lessons
- `while IFS= read -r line; do ...; done < file` never runs the body for a last line without a newline.
- A seeded random differential (generated INDEX rows and corpus files) is cheap next to the named edge cases and
  covers argument and row shapes nobody listed.

---
**Status**: PENDING (fold review pending) — INDEX row generated from the header: `| 2026-10-22-go-sweep-fold-audit.md | kit | 2026-10-22 | pending | 2 |`
