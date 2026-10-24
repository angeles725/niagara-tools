<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-23 · kit · go-lint-ext-writable-shape

**Session**: niagara-research quality round 8 unit E4a, branch `r8/e4-ext-writable-shape-go` from origin/main 544955d
**Delta count**: 1

## What happened
`toolbelt/lint-ext-writable-shape.sh` (295 lines: a private comment strip, `lib/method-boundary.sh` and one awk
fork per file) is now the rule package `go/internal/lint/rules/extwritable` and a thin wrapper that execs
`n4lint lint-ext-writable-shape`. Decision: **Rule, not TreeRule.** A property is exempt only through the
actions and do<Action> bodies of its own file, and the bash lint reported file by file in walk order. The bash
argument contract is the engine's `lint.First` mode with `BareUsage`, and its K20 row keeps the check column
`ext-writable-shape` through `EmptyError.Check`. The port went byte-identical first, keeping the private
strip; three false-result classes were then fixed, each with a RED test, and the header was rewritten to state
what the code checks.

## Evidence
- RED: 40 golden cases (every tests/ext-writable-shape.bats run on a tree plus golden-parser
  G-oneliner-extwritable, and edge trees: 18 argument shapes, type and flag spellings, multi-line and unbalanced
  annotations, action and do-body exemptions, comments and literals, CRLF, no final newline, an empty file,
  non-ASCII bytes, walk order and pruning), recorded from origin/main's bash through a recording wrapper; 21
  failed against a stubbed Check `[ev: go/internal/lint/rules/extwritable/extwritable_test.go]`
- Byte identity, old script (origin/main 544955d) vs the wrapper, stdout + stderr + exit: same on 7 argument
  shapes and on Honeywell (0 rows), panccadia-leon (5), Leon-Guanjuato (2), all of Cliente (7, ±`--strict`) and
  sources/decompiled (0) `[ev: toolbelt/lint-ext-writable-shape.sh]`
- Timing, one run each on a shared host (load 12-16): Cliente 2.1 s → 0.15 s, panccadia-leon 0.77 s → 0.03 s,
  Honeywell 0.37 s → 0.02 s, sources/decompiled 19.0 s → 0.36 s
- Declared differences (engine): an unreadable .java file exits 3 with "cannot scan" (bash printed awk's error
  and exited 0); a path holding a backslash prints as listed (awk -v read `a\nb` as a newline and split the row)
- Fix 1 (TestLiteralComments RED 3/3): a "//" or "/*" in a string is not a comment `[ev: retro
  mb-strip-string-literals Δ1]`; golden edge-comments gains 4 rows. Fix 2 (TestLiteralCode RED 4/4): a "(",
  an annotation marker or a setX( inside a literal is not code; golden edge-multiline gains 2 rows. Fix 3
  (TestWriteCallNames RED 2/6): resetX( or forgetX().setValue( no longer exempts x; golden edge-actions gains
  1 row. Real trees unchanged by all three (Cliente 7 rows, decompiled 0)
- Left as stated limits, not fixed (no real-tree row, column mapping across blanked literals needed): a second
  annotation on the same line and an annotation whose "(" is on a later line are not read. No suppression
  marker is documented, so none was removed

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **`awk -v FILE="$f"` is not a safe way to hand a path to awk: gawk reads its backslash escapes.** A path holding `\n`, `\t` or `\101` prints mangled (a row split in two, a wrong file name) and an unknown escape adds a `warning: escape sequence` line on stderr for every file. Four bash scripts still do it (lint-recovery-path.sh:58, lint-link-target-flags.sh:109, lint-wb-threading.sh:225-226, verify-module.sh:336); pass the path as `FILENAME` or through `ENVIRON`, or port the lint. Reviewer check: a new `awk -v <VAR>="$path"` in toolbelt/ is a finding. | `go/README.md` § Adding a lint rule | `[ev: go/internal/lint/rules/extwritable/extwritable_test.go]` |

## Lessons
- A port can stay under a per-commit line budget without dead code: first drop the awk commentary (a
  behavior-free commit proven by byte identity), then swap the wrapper and delete the body together, so no
  commit leaves a body below the exec (GWS4).
- Blanking literal contents changes column positions, so a rule that reads attribute values (name="...")
  must read markers and parentheses in the blanked lines but the values in the stripped ones, line by line.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-go-infra`) — Δ1 `BUILD-LOOP.md` § 5 (no path through `awk -v`); three of the four scripts are fixed, `verify-module.sh:336` is left (open issue updated).
