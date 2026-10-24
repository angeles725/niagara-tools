<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-23 · kit · go-lint-silent-protection

**Session**: niagara-research quality round 8, branch `r8/silent-protection-go` from origin/main 5a7d346
**Delta count**: 1

## What happened
`toolbelt/lint-silent-protection.sh` (513 lines of bash and awk, two awk forks per file plus two xargs passes)
is now the rule package `go/internal/lint/rules/silentprotection` and a thin wrapper that execs
`n4lint lint-silent-protection`. Decision: **TreeRule, not Rule.** Rows are ordered by file, but every file's
check reads four indexes built from the whole tree first (surface slots, effect slots, fields written to a
surface slot, alarm classes), so no file can be reported before every file was read. The port went
byte-identical first, keeping the bash quirks (gawk leftmost-longest matching, C-locale bytes, `sort -u` slot
order that picks the reported effect slot, the xargs split of passes 0/1, the 2-blank-field dedupe key). Then
two false results were fixed, each in its own commit with a RED test. No gawk `for (k in a)` order was needed,
so `gawkOrder` stayed in rules/timers.

## Evidence
- RED: 40 golden cases (every tests/lint-silent-protection.bats run on a tree plus 14 edge trees, recorded from the
  bash lint through a recording wrapper); 27 failed against a stubbed CheckTree, the other 13 expect no row
  `[ev: go/internal/lint/rules/silentprotection/silentprotection_test.go]`
- Byte identity, old script (origin/main 5a7d346) vs the wrapper, stdout + stderr + exit: same on 11 argument
  shapes (no args, `-h`, a bad flag, `--strict` alone, a missing root, a file root, an empty root, two roots,
  a relative root, a trailing slash, `--strict --`), Honeywell, panccadia-leon and Leon-Guanjuato with and
  without `--strict`, all of Cliente with and without `--strict`, tests/, the 21 Cliente `src` roots one by
  one and the 15 client worktree checkouts `[ev: toolbelt/lint-silent-protection.sh]`
- Timing, one run each on a shared host (load 12-25): Cliente 23.6 s → 0.37 s, panccadia-leon 5.3 s → 0.15 s,
  Honeywell 2.0 s → 0.04 s, Leon-Guanjuato 1.1 s → 0.04 s, sources/decompiled (2,706 files) 231 s → 21 s
- Declared difference (port): comments are blanked by the shared javasrc Strip, which keeps string literals.
  On sources/decompiled the port prints 8 more rows, all in niagarad-ext `WebServer.java`, whose `"/*"` filter
  string on line 528 made the bash stripper blank the rest of the file; with that literal replaced, old and
  new agree byte for byte. The Pattern-B pre-pass reads the same stripped lines `[ev: retro mb-strip-string-literals Δ1]`
- Declared difference (port, architect-approved): an unreadable .java file exits 3 with "cannot scan" (bash ran
  awk with 2>/dev/null and skipped it, exit 0) `[ev: go/internal/lint/rules/silentprotection/silentprotection_test.go]`
- Declared difference (port, not replicated): awk `-v FILE=` expanded backslash escapes in a path; the port
  prints the path as listed
- Fix 1: passes 0/1 ran through xargs, so a slot declared under a path with a blank or quote was never indexed
  (TestSpacedPathIndexed: 2 false WARNs → 0). Fix 2: the dedupe key was the third 2-blank field of the row, so
  under a path holding two blanks only the first row of the whole tree printed (TestDedupeOnPathLine: 1 → 4 rows).
  Neither changes real-tree output (no Cliente .java path holds a blank or a quote)
- A profile showed 70% of the first port's time in per-line `get<Slot>()` string concatenation; building the
  accessor texts once halved panccadia-leon (0.40 s → 0.22 s)
- The header documents no suppression marker and Cliente holds none (two comments name the lint only)

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A port is one rule file on the shared engine: byte-identical first, then declared truth fixes.** Record the bash lint's goldens, port it with its quirks kept until the old and new scripts agree byte for byte on fixtures and real trees, and only then fix each false result in its own commit with a RED test; a difference the shared helpers force (the literal-aware stripper) is declared in the port commit and proven by neutralizing its trigger. Why: a port that fixes while it translates cannot show which output change came from which cause. Reviewer check: a port commit that changes real-tree output without a declared difference and a pinning test is a finding. | `go/README.md` § Adding a lint rule | `[ev: go/internal/lint/rules/silentprotection/silentprotection.go]` |

## Lessons
- A bash quirk that only fires on unusual paths (xargs splitting, a dedupe key on blank runs) is still part of
  the byte-identity contract; keep it in the port, then remove it with a RED test.
- An xargs file list is not path-safe; neither is a key cut from formatted output.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-go-infra`) — Δ1 `BUILD-LOOP.md` § 5 port rule.
