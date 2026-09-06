<!-- review-status: folded -->
# 2026-09-06 · kit · campaign11-shared-method-boundary

**Session**: C11 PR1 — shared method-boundary awk fragment (`toolbelt/lib/method-boundary.sh`)
**Delta count**: 3

## What happened

[ev: campaign11 T1] The three lints (`lint-timers`, `lint-silent-protection`, `lint-ext-writable-shape`) each carried an independent copy of the same method-boundary parser. The two NET-depth copies (`lint-timers`, `lint-silent-protection`) silently missed one-liner methods (`void arm(){ ... }` — net brace change 0), producing false-negative companion-flag and silent-protection verdicts. The correct PEAK-depth copy in `lint-ext-writable-shape` was extracted into a shared awk function library (`$MB_AWK`) and the three different invocation mechanisms (inline awk, inline awk with adjacent-quote concatenation, and heredoc + second `-f`) dictated the fragment be function-only text — no BEGIN, END, or pattern rules.

## Evidence

- `[ev: QA d88af78]` C11-tl-oneliner and C11-sp-oneliner RED on dab0807 — NET parser misses one-liner methods
- `[ev: QA ed2088f]` G-oneliner-timers and G-oneliner-silent RED on dab0807 — same root cause
- `[ev: corpus B832 593019540]` investigador1 second read confirming PEAK vs NET distinction and three invocation mechanisms
- `[ev: design D1b]` function-only fragment is the unique form legal in both inline and `-f` contexts
- `[ev: design D1c]` `BASH_SOURCE`-only location, not `$KIT` (avoids writable-target seam)
- `[ev: design D1j]` 3×3 baselines byte-identical before/after — 0 one-liner schedules/trips/writes in 42 .java at ff1b659

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | Add fragment-merge rule to METHODOLOGY.md: when three lints share a parser, extract to `toolbelt/lib/`, never duplicate | `METHODOLOGY.md §K` | `[ev: campaign11-T1]` |
| 2 | Add K25: fragment rule — a function-only awk library in a shell variable can be injected into both inline awk (adjacent-quote) and multi-file awk (second `-f`); never store logic in only one consumer | `METHODOLOGY.md §K` | `[ev: design D1b]` |

## Lessons

- [ev: design D1a] Three copies of one parser were not three styles but three awk invocation mechanisms; the mechanism dictated a function-only fragment, not a `.awk` file.
- [ev: design D1g] PEAK depth fixes the one-liner false negative only because the close test runs in the SAME loop iteration as the open; NET depth never fires on a same-line open+close (net delta 0).
- [ev: design D1c] A lint that resolved its own parser through `$KIT` would load a different parser than the script the user invoked whenever `$KIT` is set (writable-target seam); `BASH_SOURCE` binds the fragment to the running script.
- [ev: design D1h I4] The accessor skip predicate is `(one-liner) AND mname ~ /^(get|set|is)[A-Z_]/`; a multi-line `isDirty()` that schedules must NOT be skipped — the one-liner form only.
- [ev: design D1j] The 3×3 real-tree baselines are the only pin for B832-G2 (`/* */` strip); a delta blocks the merge.
- [ev: second read ad2121b69] Only three fragment invariants have a biting fixture (depth guard → S21-misparse; PEAK vs NET → the four one-liner pins; accessor skip → G-accessor/C11-g1-setter). The Case-B `@`-stop and the keyword exclusion are redundant for lint output under PEAK (the depth guard rejects the depth-1 class body before Case B runs; nested keywords are dead under `!in_m`) — kept as documented defensive code, pinned only by the aggregate golden set + 3×3 baselines, never carried as OBSERVED mutations (K24(7)).

---
**Status**: FOLDED — METHODOLOGY.md K25 (shared method-boundary parser; PEAK depth; function-only fragment via BASH_SOURCE); BUILD-LOOP.md + skill/SKILL.md lint bullets updated by PR1. INDEX row: folded.
