<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · go-lint-lexicon-ascii

**Session**: niagara-research quality round 10, branch `r10/lexicon-ascii-go` from origin/main e6581fd
**Delta count**: 2

## What happened
`toolbelt/lint-lexicon-ascii.sh` (two `while read` loops, od/tr/grep forked per lexicon line) is now the per-file
rule `go/internal/lint/rules/lexiconascii` and a thin wrapper that execs `n4lint lint-lexicon-ascii`. Its row on the
fork-per-line ratchet (`tests/fixtures/lint-fork-per-line/baseline.txt`) is removed as a declared improvement. The
lint's LA5 contract (a module with no `*.lexicon` exits 0, pinned by tests/lint-empty-root.bats ER5) needed the one
engine opt-out the README reserved: `Spec.EmptyOK`, in its own commit with a test; `rules_test`'s empty-root sweep
pins an EmptyOK rule's own contract (exit 0, nothing, no `--all` row). `internal/props` is not used: the lint reads
raw bytes per line, never a key or value, and a non-ASCII byte in a comment is a defect too. The port went
byte-identical first, then one truth fix with RED first.

## Evidence
- Goldens recorded from the original bash through a recording wrapper on the bats suite plus an edge driver: 5 bats
  runs on trees the format can hold and 21 edge trees (argument shapes, many bytes per line, no final newline, CRLF,
  comments and `\u` escapes, blank and leading-space lines, walk order and pruning, a trailing backslash, a NUL
  byte, an empty file). `TestEmptyDir` (LA5) and `TestLocked` (LA-locked, text recorded from the bash) cover what the
  format cannot hold. RED: 16/26 cases failed against a stub that scanned no line `[ev: go/internal/lint/rules/lexiconascii/lexiconascii_test.go]`
- Byte identity old vs new (stdout, stderr, exit): Cliente and `Cliente/` (7 rows, exit 1), its panccadia-leon group,
  DashboardPan-rt, sources/decompiled (13 lexicons, 0 rows) and one decompiled module. Timing: Cliente 1358 → 27 ms,
  decompiled 5595 → 27 ms `[ev: toolbelt/lint-lexicon-ascii.sh]`
- Cliente's 7 rows are all true and in live source: panccadia-leon `DashboardPan-rt/module.lexicon` lines 58-66 hold
  raw UTF-8 Spanish (`Caído`, `en línea`, `última`, `Dirección`, `automática`), shown as mojibake on the station
  until written as `í`-style escapes. None comes from `_salvage/` or the stale Leon-Guanjuato checkout
- Declared port difference: an unreadable `*.lexicon` is exit 3 (`cannot scan <file>`); the bash loop printed the
  shell's error and went on with exit 0 or 1. bats LA-unreadable was RED on the bash
- Truth fix (RED: `truth-noeol` and bats LA-noeol): the bash `while read` skipped a last line with no final newline,
  so a non-ASCII byte there passed clean; every line is scanned now. edge-noeol gains its two rows as declared; real
  trees unchanged (Cliente 7 → 7, decompiled 0 → 0, though 7 of its 13 lexicons end without a newline, in ASCII)
  `[ev: go/internal/lint/rules/lexiconascii/lexiconascii.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **Every `while IFS= read -r` lint drops a last line with no final newline.** The bash idiom reads nothing for an unterminated last record; a byte-identical port must reproduce that and the truth fix removes it. 7 of the 13 Tridium lexicons in sources/decompiled end without a newline, so the gap is common in real files. Reviewer check for the remaining bash lints: `while IFS= read -r x || [ -n "$x" ]`, or port. | `toolbelt/lib` bash idioms; `go/README.md` § Adding a lint rule (step 6) | `[ev: go/internal/lint/rules/lexiconascii/testdata/truth-noeol.txt]` |
| Δ2 | **`javasrc.Lines` loses whether a file ends in a newline.** The identical port had to re-read the file's last byte; a `File` field (e.g. `NoFinalNewline`) would serve the next port of a `while read` lint without a second read. | `go/internal/javasrc` § File | `[ev: go/internal/lint/rules/lexiconascii/lexiconascii.go]` |

## Lessons
- An engine default with one legitimate exception gets one named, commented opt-out and a test that pins the
  exception's own contract, not a silent skip.
- A lint that only scans bytes does not need a format parser: reuse decides by what the lint reads, not by the file
  format.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-24-go-lint-lexicon-ascii.md | kit | 2026-10-24 | pending | 2 |`
