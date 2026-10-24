<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · r10-generate-wiring-map-go

**Session**: niagara-research quality round 10, branch `r10/generate-wiring-map-go` from origin/main 15e792d1
**Delta count**: 3

## What happened
`toolbelt/generate-wiring-map.sh` (227 lines of bash and awk) was ported to Go as `go/internal/wiringmap` and
`go/cmd/generate-wiring-map`, byte for byte first. The script is now the `go_tool_bin` wrapper. The port reuses
`lint.Walk` for the fail-closed walk, `javasrc.NiagaraProperties` (later `PropertyScan`) for the annotation join
and `strerror` for the error text.

- **Goldens.** `testdata/record/record.sh` and `cases.sh` recorded 62 cases from the bash at 15e792d1: every
  bats run plus 52 edge trees (option order, lexicon discovery and line shapes, annotation shapes, walk pruning,
  symlinks, unreadable inputs). A case also holds the files the run writes, as `@expect <file>` sections.
  A review then hardened the recorder: it runs the bash under `env -i PATH=/usr/bin:/bin LC_ALL=C`, pins the
  script and `lib/scan-files.sh` hashes (the first labels said fa1201e6, the script's last commit, but the walk
  lib is the base's), refuses the Go wrapper and root, finds the written files itself, and the runner compares
  the whole tree after the run. The 58 recorded cases now include a nested locked directory and a locked walk
  with `--output` (no file written).
- **Byte identity.** 62/62 goldens. Bash and Go on 23 real facade trees (client source dirs, worktrees pruned):
  identical stdout, stderr and exit, 1354 rows.
- **Truth fixes.** Each one is its own commit with a `truth-*` case and a bats pin:
  - An annotation in a comment is no slot. A `(` in a javadoc no longer swallows the annotations below it.
    A comment inside an annotation no longer sets its flags. On one real facade module, a latched status slot
    whose comment said "never OPERATOR-writable" was listed in Table 1. It is now in Table 2, in the module and in
    its sibling checkout.
  - A `(` inside a string literal no longer keeps the annotation open.
  - An unreadable source or lexicon is exit 3 with no map. The bash printed a shorter map, or a map of
    placeholders, and exited 0.
  - An `--output` file that cannot be written is exit 3. The bash printed `wrote <file>` and exited 0.
- **Consumer bug.** `obixlinkaudit.DeclaredRows` read column 2 as the RT slot. Since the display-name column
  landed, that column is the Workbench name, so a lexicon-backed scaffold audited the display name as the source.
  It now finds the column by its `... RT slot` header cell.

## Evidence
- `go test ./internal/wiringmap/` (65 goldens: 58 recorded, 7 truth; `TestSlots`, `TestDisplayNames`, `TestOutputWriteError`,
  `TestMapReadBack`) and `TestDeclaredRowsDisplayColumn` green `[ev: go/internal/wiringmap/testdata]`
- bats GWM-comments, GWM-literal-paren, GWM-unreadable, GWM-output-unwritable `[ev: tests/generate-wiring-map.bats]`
- Real-tree parity (local, not committed): 23/23 identical before the truth fixes; after them, 2/23 trees differ,
  each by the one slot that moved from Table 1 to Table 2 `[ev: retro r10-generate-wiring-map-go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A port whose command writes files records them as `@expect <file>` sections and replays each case with its tree as the working directory | `go/README.md` § Adding a lint rule (step 3) | `[ev: go/internal/wiringmap/golden_test.go]` |
| Δ2 | A generator tests its output through its consumer's parser, and the consumer finds a column by its header cell, not by index | `go/README.md` § Adding a lint rule (after the shared-code rule) | `[ev: go/internal/wiringmap/wiringmap_test.go]` |
| Δ3 | An annotation harvest runs `PropertyScan` over `MaskLiterals(Strip(lines))` and reads the name and flags from the stripped text | `go/README.md` § Adding a lint rule (step 6) | `[ev: go/internal/wiringmap/slots.go]` |

## Lessons
- A raw-line scan that opens on a token and closes on balanced parentheses is only as good as its first line. A
  comment that names the token with an unbalanced `(` silently ate every slot after it.
- When a generator gains a column, every reader that reads columns by index shifts too. A round-trip test
  catches this; per-tool substring bats do not.
