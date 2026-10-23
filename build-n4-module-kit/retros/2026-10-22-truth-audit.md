<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 5 -->
# 2026-10-22 · kit · truth-audit

**Session**: niagara-research quality round 8, kit truth audit of 2026-10-22 (40 lints x 16 client module roots) and the PR #346/#347 landings; fold unit U1, branch `docs/r8-u1-tool-doctrine` from origin/main 5ea892c
**Delta count**: 5

## What happened
The truth audit ran every kit lint over the client module roots and asked one question per tool: does it ever
report clean when it should not, or report a finding that is not there? Three classes came back. Ports kept known
false results on purpose (byte identity) with no follow-up owed; most lints pass an empty root; and the Java rules
still match raw lines, so a comment or a string literal is a finding and a call split over lines is missed. The
same day two process faults surfaced in the kit's own bookkeeping: retro citations in the dated form are silently
not counted by the fold audit, and a landing helper that piped `git push` through `grep | tail` hid a rejected
push, so PR #346 was opened from the stale remote head and merged unreviewed WIP into main.

## Evidence
- Audit summary rows (copied from the out-of-repo probe file `2026-10-22-truth-audit-summary.tsv`, 640 rows:
  lint, module root, exit, FAIL count, WARN count; the column reading is checked against a re-run of lint-delays
  on ColdRoomPan-rt, 4 FAIL rows, exit 1):
  - `lint-delays	Leon-Guanjuato/Paccadia/ColdRoomPan/ColdRoomPan-rt	1	4	0`
  - `lint-timers	Leon-Guanjuato/Paccadia/ColdRoomPan/ColdRoomPan-rt	1	2	1`
  - `lint-null-context-write	Honeywell/MX60/chihuahua/chihuahua/chihuahua-rt	0	0	19`
  - `lint-structure	Honeywell/MX60/chihuahua/chihuahua/chihuahua-rt	1	2	1`
  `[ev: 2026-10-22 truth-audit summary]`
- Δ1: `go/internal/lint/rules/timers/ticket.go:41` (discarded-ticket) and `:82` (`ownsTimer`) match on `s.Raw`,
  the raw line, as the bash greps did. A fixture whose only Clock text is the comment
  `// Clock.schedule(this, t, a, null);` and a string literal gives `FAIL discarded-ticket` and `FAIL timer-ticket`
  (exit 1) `[ev: go/internal/lint/rules/timers/ticket.go]`
- Δ2: the 40 audited lints run on one empty directory at origin/main 5ea892c: 35 exit 0 with no row, 5 exit 3
  (demand-scope, ext-writable-shape, link-target-flags, silent-protection, write-path). The Go engine already
  has the switch (`go/internal/lint/lint.go` `Spec.EmptyError`); most rules do not set it
  `[ev: go/internal/lint/lint.go]`
- Δ3: fixtures at 5ea892c. `lint-no-system-out` FAILs `/* System.out.println("old"); */` and
  `String help = "call System.out.println(x) to debug";`; `lint-delays` FAILs `literal-zero` on a
  `// Clock.schedule(this, BRelTime.make(0), …)` comment and on the same text in a string;
  `lint-null-context-write` WARNs `c.set(p, v, null);` but not the same call split over three lines.
  Rows without a line: `lint-timers` prints `<file>:` with no number (`BEvaporatorUnit.java: schedules a Clock
  ticket …` on ColdRoomPan-rt), and `lint-structure` puts the rule ID where `file:line` belongs
  (`toolbelt/lint-structure.sh:68` row format, `:93` L10, `:387` L14:
  `FAIL  lint-structure  <root>/settings.gradle.kts  L14: hardcoded gradlePluginVersion literal …`)
  `[ev: toolbelt/lint-structure.sh]`
- Δ4: `toolbelt/sweep-fold-audit.sh:13-15,22` (header) and `go/internal/foldaudit/foldaudit.go:24,178-179`: a
  citation's stem is matched exactly against the row stem, which drops the `YYYY-MM-DD-` prefix and `.md`, so
  `[ev: retro 2026-09-25-panccadia-commissioning-lessons.md Δ2]` (form still present in older retros, e.g.
  `retros/2026-09-26-behavior-decisions-ask-dont-assume.md:50`) credits nothing. Observed during the PR #347 fold
  `[ev: go/internal/foldaudit/foldaudit.go]`
- Δ5: PR #346 merged 2026-10-23T06:02Z with head `79ca5a2` (`wip(kit): follow a symlinked lint root (#329), in
  progress`), now an ancestor of origin/main; merge commit 171488d `[ev: 79ca5a2]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A port is byte-identical first; then each known false result is fixed in its own commit with a seeded RED test and declared in the PR.** A port is not done while it keeps a known lie. Review rule. | `CONTRIBUTING.md` § 2.2 Go module | `[ev: go/internal/lint/rules/timers/ticket.go]` |
| Δ2 | **A lint with no input files under its root is an ERROR (exit 3), never a silent pass.** Owed Go engine item G50. | `BUILD-LOOP.md` § 5 Kit tooling doctrine | `[ev: go/internal/lint/lint.go]` |
| Δ3 | **Every Java-matching rule matches on blanked source (comments and literal contents removed) and on the statement, not the raw line; a finding row carries `file:line` with a real line number, never a rule ID in that position.** Owed Go engine item G51. | `BUILD-LOOP.md` § 5 Kit tooling doctrine | `[ev: toolbelt/lint-structure.sh]` |
| Δ4 | **Retro citation tags use the stem without the `YYYY-MM-DD-` prefix and `.md`: `[ev: retro <stem> Δn]` / `[deferred: retro <stem> Δn]`.** The dated form is silently not counted. Owed check G52. | `CONTRIBUTING.md` § 6.1 | `[ev: go/internal/foldaudit/foldaudit.go]` |
| Δ5 | **A landing script checks the `git push` exit status (never through a pipe without pipefail), verifies `origin/<branch>` equals the reviewed local SHA before opening or merging the PR, and merges with `gh pr merge --match-head-commit <reviewed SHA>`.** Review evidence is bound to a SHA. | `CONTRIBUTING.md` § 8 | `[ev: 79ca5a2]` |

## Lessons
- A byte-identical port is the proof step, not the finish line: the known lies it keeps are owed work.
- "Exit 0, no rows" means clean only when the tool proves it read something.
- A raw-line regex over Java is wrong both ways: it flags comments and strings and it misses split calls.
- A citation format the auditor cannot count is a silent gap; the auditor should say so.
- Review evidence is bound to one SHA; a landing step that cannot prove the SHA lands unreviewed code.

---
**Status**: FOLDED 2026-10-23 (branch `docs/r8-u1-tool-doctrine`; log in `ledger.d/2026-10-23-r8-u1-tool-doctrine.md`) — Δ1 `CONTRIBUTING.md` § 2.2 "Port in two steps"; Δ2 `BUILD-LOOP.md` § 5 "Fail-closed input listing" (owed G50); Δ3 `BUILD-LOOP.md` § 5 "Literal-aware and scope-aware Java scanning" (owed G51); Δ4 `CONTRIBUTING.md` § 6.1 (owed G52); Δ5 `CONTRIBUTING.md` § 8 "Land only the reviewed SHA".
