<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-22 · kit · go-gen-lint-index

**Session**: niagara-research quality round 8, branch `r8/gen-lint-index-go` from origin/main 6416fb0
**Delta count**: 1

## What happened
`toolbelt/gen-lint-index.sh` (an awk header parser per lint, a `grep -v | grep -qE` Auto probe and a tab-split
`read` per row) became `go/cmd/gen-lint-index` (`go/internal/lintindex`) and the script a thin wrapper over
`go/go-tool.sh` `go_tool_bin`, the gen-retro-index pattern. Capturing the old output first exposed two bash
defects: a description made only of `[ev:]` tags left an empty first field that the IFS-tab `read` swallowed, so
every later cell shifted one column left; and a failed `printf > INDEX.md` still printed `wrote` and exited 0. Three
#226 triage findings were folded in test-first: Usage/Exit lines are read only from the header block (the leading
run of `#` lines), so a `# Exit …` code comment below it never joins the Exit cell (R2/R3/R4-gli-exit); Auto=yes
needs the `$TOOLBELT/<lint>` reference in command position, so a message string, a printf argument or an
assignment no longer counts (R3-gli-auto-message-string-overclaim); and the wrap/exit mutation label now names the
continuation join it guards (R2-gli-wrap-exit-mutation-label). The bats GLI-awkfail pin faked `awk` on PATH, which
a Go generator never runs; it now makes a lint header unreadable (mode 000), the same env failure, and still asserts
exit 3 with the committed INDEX.md untouched (approved edit of `tests/gen-lint-index.bats`).

## Evidence
- Old-vs-new over 50 edge shapes (every bats scenario, usage/env, generate vs `--check` fresh/stale/missing/extra
  newlines/index-is-a-dir, Auto shapes, separators, wrong line-2 name, the 60-line and line-5 caps, CRLF, tabs, `[ev:]`
  dedupe, empty Usage/Exit values, continuation rules, no final newline, backslash/`&`, invalid UTF-8, sort order,
  directories and symlinks named lint-*.sh, unreadable lint, write failure, unreadable INDEX) plus 42 real-tree
  runs (`--check` with the default and an explicit root, and a full regenerate on 40 past toolbelt revisions): all
  42 real-tree runs byte-identical (the committed `toolbelt/INDEX.md` is unchanged) `[ev: r8 gen-lint-index byte-identity harness]`
- Deliberate differences, all on edge shapes: `[ev:]`-only description keeps its columns (bash shifted them); a
  failed write is exit 3 `cannot write` (bash: `wrote`, exit 0); an unreadable header names the lint and says "a
  lint header could not be read" (bash: awk's fatal line, "awk failed"; exit 3 both); a missing cat message on an
  unreadable INDEX under `--check` (exit 1 both); an assignment `y=$TOOLBELT/lint-e.sh` is Auto no; a Usage line
  after code is a contract violation (exit 1) `[ev: go/internal/lintindex/lintindex_test.go TestExtract]`
- RED (package missing, build failed) → GREEN; the three #226 fixes each RED first in TestExtract /
  TestInvokedByReport `[ev: go/internal/lintindex/lintindex_test.go TestInvokedByReport]`
- Mutations: GLI-stale (compare dropped) fails TestMainGenerate and bats GLI-stale; GLI-awkfail (unreadable treated
  as a contract violation) fails TestMainUnreadableHeader and bats GLI-awkfail `[ev: go/internal/lintindex/command.go Main]`
- Real kit `--check`, load ~7: bash ~470 ms/call, wrapper ~42 ms/call, binary ~32 ms/call (20 calls) `[ev: r8 gen-lint-index timing]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A pin that fakes an external binary on PATH tests the implementation, not the contract; when a tool is ported, restate it as the behaviour the binary stood for (here: an unreadable header is exit 3, nothing written).** Why: GLI-awkfail could never pass against the Go generator. | `CONTRIBUTING.md` § 2.2 Go module | `[ev: retro go-gen-lint-index]` |

Δ1 is applied in this unit for gen-lint-index only; the retro stays pending until a fold review decides the rule.

## Lessons
- An IFS-whitespace `read` collapses empty fields: a TAB-separated record with an empty first field shifts every
  later field.
- A bash redirect failure without `set -e` is only a stderr line; the next `printf 'wrote'` still runs.

---
**Status**: PENDING (Δ1 applied to gen-lint-index; fold review pending) — INDEX row generated from the header: `| 2026-10-22-go-gen-lint-index.md | kit | 2026-10-22 | pending | 1 |`
