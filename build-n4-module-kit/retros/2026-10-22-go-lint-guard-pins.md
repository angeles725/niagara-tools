<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-22 · kit · go-lint-guard-pins

**Session**: niagara-research quality round 8, branch `r8/lint-guard-pins-go` from origin/main e3713cf
**Delta count**: 1

## What happened
`toolbelt/lint-guard-pins.sh` (a `find | sort` loop, `head -60 | while read` per lint, and one
`grep ... tests/*.bats` per declared mutation) became `go/cmd/lint-guard-pins` (`go/internal/guardpins`) and the
script a thin wrapper over `go/go-tool.sh` `go_tool_bin`. The script is itself a `lint-*.sh`, so its header
(description, `Usage:`, `Exits:`, `[ev:]` tags, the `# Mutation: T4-nomutation` line) is kept byte for byte and
`toolbelt/INDEX.md` does not change; the new note sits below the header and carries no `[ev:]` tag. Capturing the
old output first exposed two bash defects: an unreadable `toolbelt/` made `find` print an error while the lint
passed with zero rows even under `--strict` (fail-open), and a header whose last line is an unterminated
`# Mutation:` line was dropped by `while read` (a false `no # Mutation: line` WARN). The bats index is built once
instead of one grep per mutation, so a real-tree run drops from seconds to milliseconds.

## Evidence
- Old-vs-new over 34 edge shapes (the 7 bats scenarios, usage with 0/1/2 args and `--strict` misplaced, empty
  root, missing toolbelt/tests, nested `build-n4-module-kit/toolbelt`, `.`/`./`/trailing-slash/symlinked roots,
  the declaration grammar, the 60-line cap, CRLF, `@test` prefix rules, directories/broken and good symlinks/
  empty and mode-000 lints, unreadable `tests/` and `.bats`, sort order, hidden and `.bak` bats) plus 32 real-tree
  runs (7 root spellings of this repo and `--strict .` on 25 past revisions): stdout and exit identical for
  62/66, all 32 real-tree runs byte-identical. The 4 differences: unreadable `toolbelt/` exit 0 → 3
  (`lint-guard-pins: cannot list ...`); unterminated last `# Mutation:` line WARN → MATCH; and two stderr notes
  worded by Go instead of `head:` (same WARN rows) `[ev: r8 guard-pins byte-identity harness]`
- `go/internal/guardpins` table tests port every `tests/guard-pins.bats` scenario plus the edge shapes; RED
  (package missing, build failed) → GREEN `[ev: go/internal/guardpins/command_test.go TestMain]`
- Mutation T4-nomutation (drop the no-declaration WARN) fails 4 Go tests and bats T4-nomutation / T4-strict
  `[ev: go/internal/guardpins/command.go Main]`
- `--strict .` on the real repo (199 mutation lines; coordinator baseline 12.2 s, hot spots the per-header-line
  `printf | grep` + `sed` and one `grep tests/*.bats` per mutation): load ~13, bash ~9.3 s/call (5 calls), wrapper
  ~26 ms/call, binary ~11 ms/call (20 calls); re-run at load ~10: bash ~7.6 s/call (2), wrapper ~16 ms/call (20)
  `[ev: r8 guard-pins timing]`
- Deliberate contract fix (follow-up commit): the usage said `<kit-root>`, but `tests/` is read from the argument,
  so a kit root always exited 3 `no tests/ found` (T27 profiling). Usage and header now say `<repo-root>`
  (`toolbelt/INDEX.md` regenerated: that Usage cell only), and a kit root whose parent holds `tests/` is exit 3
  naming the repository root to pass `[ev: go/internal/guardpins/command_test.go TestMainKitRootGiven]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A lint that lists its inputs must fail (exit 3) when the listing itself fails; an unreadable scan root is never zero findings.** Why: bash `find` on a mode-000 `toolbelt/` printed an error and `lint-guard-pins.sh --strict` still exited 0. | `CONTRIBUTING.md` § 2.2 Go module | `[ev: retro go-lint-guard-pins]` |

Δ1 is applied in this unit for lint-guard-pins only; the retro stays pending until a fold review decides the rule.

## Lessons
- `find ... | sort` in a process substitution hides find's exit status: the loop sees an empty list, not an error.
- A usage line is a contract: `<kit-root>` that only works with the repository root is a defect, not a typo.
- A ported `lint-*.sh` is also an input of `gen-lint-index` and of itself; keep its header (and its `[ev:]` tags)
  unchanged or the generated lint index moves with the port.

---
**Status**: FOLDED 2026-10-23 (branch `docs/r8-u1-tool-doctrine`; log in `ledger.d/2026-10-23-r8-u1-tool-doctrine.md`) — Δ1 `CONTRIBUTING.md` § 2.2 "Fail closed on I/O".
