<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-23 · kit · truth-ported-rules

**Session**: niagara-research quality round 8 (Ayudante2, Go lint-engine owner), branch `r8/truth-ported-rules` from `r8/e1-blank-literals` c853135
**Delta count**: 2

## What happened
The 2026-10-22 truth audit seeded false results in Go rules already on main (policy R8-D14: a port is not done
while it keeps a known lie). Each fix is one commit with a RED seed first and a declared difference in the
rule wrapper header.

- **T1 lint-timers.** `discarded-ticket` and the timer-ticket owner test scanned raw lines, so a
  `Clock.schedule(` in a `//` comment or a string FAILed a timerless class; changed-sched and
  atSteadyState-only-timer read raw lines too, and companion-flag/orphan-flag/release-gate-parity read
  comment-stripped lines with literal text kept. Every check now reads the blanked code of the shared javasrc
  helpers (`Strip` + `BlankLine`, already used by ticket.awk's port). discarded-ticket judges the call's own
  statement (an `=` of an earlier statement on the line no longer hides a discard; a multi-line `t =` is kept).
  Every row now carries its line (`<file>:<line>  <msg>`); jdk-thread names the permissions file by its module
  directory instead of a second absolute path, and reads the grant directly (the bash `read -r` split a path
  with a space and FAILed a granted module). report-module.sh's lint-timers relay parsed `<file>: <msg>` and
  was moved to the lined shape.
- **T2 lint-null-context-write.** The write regex matched one line, so `c.set(p,\n BDouble.make(2),\n null);` and
  `c.set(p, v, (Context) null)` passed, while a null in a nested call (`c.set(p, foo(x, null))`) or in another
  call on the line WARNed. The rule now reads the write's own argument list to its balanced `)` across lines
  and WARNs when the last of two or more arguments is `null` or a null cast to Context; the row line is the
  call's line (where `.set(` is), one row per line as before.

## Evidence
- T1 RED: golden seeds `t1-schedule-in-comment`, `t1-schedule-in-string`, `t1-row-shape` failed on c853135 (FAIL rows / no lines); `t1-sched-comment` run through the c853135 binary gave 4 false rows and a missed changed-sched `[ev: go/internal/lint/rules/timers/testdata]`
- T1 mutations observed: raw lines in ticket.go fail `TR1-code`; `Line` dropped fails `TR1-shape` `[ev: tests/lint-timers.bats]`
- T1 real trees (main 3e59585 vs new, Honeywell / panccadia-leon / Leon-Guanjuato): 16 rows changed, every one only in shape (line added, jdk-thread text); no verdict added or removed; chihuahua setDaemon stays at :310 `[ev: retro truth-ported-rules]`

- T2 RED: three `T2` table cases failed on c853135 (multi-line write, `(Context) null`, nested/other-call null) `[ev: go/internal/lint/rules/nullcontext/nullcontext_test.go]`
- T2 mutation observed: stopping the argument scan at the line end fails `NCW-stmt` `[ev: tests/lint-null-context-write.bats]`
- T2 real trees: no row changed on Honeywell (22), panccadia-leon (0) or Leon-Guanjuato (1) `[ev: retro truth-ported-rules]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Modularity: a truth fix uses the shared javasrc helpers (`Strip`, `BlankLiterals`/`BlankLine`, `Methods`); never a private scanner or a raw-line regex in a rule | `go/README.md` § Adding a lint rule; `CONTRIBUTING.md` § 2.2 | `[ev: go/internal/lint/rules/timers/ticket.go]` |
| Δ2 | A rule row names its line: `Line` 0 (the file-level `<file>: <msg>` row) only for a finding about a whole file, so relays can split `<file>:<line>  <msg>` on the two-space separator | `go/README.md` § Adding a lint rule | `[ev: tests/report-module.bats RM2]` |

## Lessons
- A byte-for-byte port keeps the bash lint's lies with its behaviour; the golden files make each truth fix a reviewable diff of rows, not a silent change.
- A consumer that re-parses rows (report-module.sh) breaks on a row-shape change without failing loudly; pin the relayed shape in its bats.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-23-truth-ported-rules.md | kit | 2026-10-23 | pending | 2 |`
