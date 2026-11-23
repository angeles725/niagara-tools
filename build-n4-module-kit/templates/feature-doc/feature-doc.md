# Feature: <one line> (<feature-name>)

Template from the build-n4-module kit (`templates/feature-doc/feature-doc.md`). Copy it to the module repository as
`odd/tasks/<feature-name>.md` at the start of the design step (`BUILD-LOOP.md` §1) and fill every section below in
order. Delete this paragraph. The section order is the gate: no source is written until `## Behavior decisions` holds
an answer for every unstated behavior.

Locator: `odd/tasks/<feature-name>.md` · base `<branch or SHA>` · tier `<P0|P1|P2|P3>` and the evidence for it
(`BUILD-LOOP.md` §0.c).

## Specs
- S1. The user's request, quoted. What done looks like, in field terms.
- S2. <one rule per line, each testable>

## Behavior decisions
Every behavior the code must decide that the user has not stated. Ask before the first source write
(`BUILD-LOOP.md` §1): 2-4 concrete options, the recommended one first and marked, each with its field consequence,
plus a free "your own option". A writer that meets a behavior missing here stops and returns the question
(`ORCHESTRATION.md` §3.c). Write "none" only when the change has no control, protection, automatic action or
operator-facing UI behavior.

| # | Question | Options shown | Answer | Who | Date |
|---|----------|---------------|--------|-----|------|
| B1 | | | | | |

## Design shard
States and guards, invalid-sensor behavior, min-on / min-off, config bounds, boot-critical expressions, and the
latch, clear condition and HMI rendering of an automatic protective action (`ORCHESTRATION.md` §3.b). Or
`Design shard: waived (<reason>)`.

## Tasks
| ID | S# | Scope | Route | Commit |
|----|----|-------|-------|--------|
| T1 | S1 | | inline / writer | |

## Consumer impact
Points an off-module consumer reads whose meaning changes (`METHODOLOGY.md` § Domain correctness). Or `none`.

| Point path | Old meaning | New meaning | Effective when |
|------------|-------------|-------------|----------------|

## Checks skipped
Every check the session skipped under time pressure, and why (`BUILD-LOOP.md` §5). Or `none`.

## Assumptions still open
Behaviors decided without an explicit user answer, repeated in the close message as "Assumptions I made — confirm or
change" (`BUILD-LOOP.md` §7). An empty register says `none`.

## Parked advisories
Review advisories that are not a fail-open or a regression of this feature (`ORCHESTRATION.md` §7). Or `none`.

## Log
Dated progress lines: the command run, its exit code and what it showed. Never the decisions table: those live above.
