<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-23 · kit · r9-guardpins-header

**Session**: niagara-research quality round 9 (Ayudante3), branch `r9/guardpins-header` from origin/main 252e9a0
**Delta count**: 1

## What happened
`lint-guard-pins` read a lint's `# Mutation:` declarations from a fixed window of lines 1-60 (D4c, kept by the
Go port as `guardpins.HeaderLines`). `lint-config-sanity.sh` grew to 13 declarations; the last 8 sit on lines
61-68, so the guard never resolved them and printed nothing about them: a silent skip, not a WARN. The header is
now the leading comment block (every line up to the first that is neither blank nor a `#` comment), which is
how every lint is written: in all 43 `toolbelt/lint-*.sh` the last `# Mutation:` line precedes the first code
line. A declaration after the first code line is still not read, so a comment in a heredoc or a function body
cannot declare a pin.

## Evidence
- RED: two new `TestDeclarations` cases (a declaration past line 60 inside the comment block; one after the first
  code line) failed against the 60-line window `[ev: go/internal/guardpins/guardpins_test.go TestDeclarations]`
- GREEN after `Declarations` stops at the first non-header line; `go test ./internal/guardpins` ok
- Real tree, old vs new binary on this repo: identical except 8 new MATCH rows,
  `lint-config-sanity.sh:61-68` (LCS-floor-suffix, CS1-units, CS1-mixed, CS1-minutes, CS1-wrapped, CS1-bothzero,
  CS1-unreadable, LCS-finderr), every one resolving to a `tests/lint-config-sanity.bats` fixture; exit 0 both
  `[ev: r9 guard-pins old-vs-new diff]`
- Survey: lint-timers.sh has the longest comment block (to line 128), with its last declaration on line 44

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A header reader takes the leading comment block, never a fixed line count; a fixed window must report what it cuts off | `go/internal/guardpins/guardpins.go` § Declarations | `[ev: go/internal/guardpins/guardpins_test.go TestDeclarations]` |

## Fold
- Δ1 → `BUILD-LOOP.md` § 5 One layered Go lint stack (one header reader, `go/internal/lintheader` `Lines`, no line cap); guardpins uses it.

## Lessons
- A fixed read window is a silent cap: when the input outgrows it, the tool reports less and still passes.
- `gen-lint-index` (`go/internal/lintindex`) has the same 60-line window for `# Usage:`/`# Exit:`; moving it to
  the comment block can change `toolbelt/INDEX.md`, so it is a separate unit.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-meta`) — INDEX row: `| 2026-10-23-r9-guardpins-header.md | kit | 2026-10-23 | folded | 1 |`
