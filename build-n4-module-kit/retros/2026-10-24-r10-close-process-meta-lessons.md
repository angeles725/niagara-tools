<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 6 -->
# 2026-10-24 · kit · r10-close-process-meta-lessons

**Session**: niagara-research quality round 10 close (`odd/tasks/quality-round10.md`), branch
`docs/r10-close-retro` from origin/main 77ea6008
**Delta count**: 6

## What happened
The user's standing rule for round 10 was: work modular and line-efficient, but always with quality, so every
part stays maintainable. This retro lists each doctrine line round 10 produced, names where it now lives, and
folds only the ones the kit did not yet state. Each lesson is the same idea seen from another side. One copy of a
concern (a helper, a resolver, a doctrine set) is cheaper to keep true than many copies. "Fewer lines" counts only
when tests, positive controls and evidence stay in place.

## Doctrine lines and where they live
| # | Doctrine line | Lives at | Evidence |
|---|---|---|---|
| L1 | Truth audits are a standing practice: card citations, client lint rows, own reports | `BUILD-LOOP.md` § 5 Citation pre-filter (`:318`); `.githooks/pre-push:18-21` live gate; `CONTRIBUTING.md` "Settle a testable review claim" (`:321`); the round habit is new here (Δ1), the report side is Δ2 | PRs #384-#406 (nine audit batches), #457 (48 live findings), #412 kit-cite-resolve, #425/#446 `--exact-stems`, issue #456 → #458, #460 pre-push gate; issue #459 → #464 |
| L2 | Generated ledgers are never hand-edited; R10-D1 (never committed) is NOT landed | `BUILD-LOOP.md:404` (kit ledger), `:437` (retro index), `:227` (`toolbelt/INDEX.md` via `gen-lint-index.sh`) | `git ls-files` on 77ea6008 still tracks `BUILD-STATE.md`, `retros/INDEX.md` and `toolbelt/INDEX.md`; R10-D1 is owed (no delta here) |
| L3 | A tool checkout is resolved fail-closed, never used as found | `BUILD-LOOP.md:6` (`scripts/kit-resolve.sh`); the general rule is new here (Δ3) | #422 (R10-D7), #439 (auto fast-forward, refuse ahead/diverged); sdd-investigacion #2337 (a0ef18ad, closes #2334) |
| L4 | A gate keyed on paths must not skip data | `BUILD-LOOP.md:301` | niagara-research PR #46; kit #404; retro 2026-10-24-r10-path-keyed-gates-skip-data |
| L5 | No fork per line or per item | `BUILD-LOOP.md:264` | #440 (b576b98b, one awk pass, #432); retro 2026-10-24-r10-fork-per-line-again |
| L6 | A fix branch starts from the installed line | `BUILD-LOOP.md:31` | #461 (2d637879); retro 2026-10-24-r10-installed-line-guard |
| L7 | One shared helper per concern | `METHODOLOGY.md:31`; sync pin `BUILD-LOOP.md:249` | `toolbelt/lib/vendor-version.sh` (#444, after #437), `toolbelt/lib/slotomatic-skip.sh` (#429), `scripts/lib/live-doctrine-set.sh` (#460); client: `ChiNumberParse.parseFinite` (honeywell-mx60-chihuahua PR #52, open); `JsonUtil.parseFiniteDouble` patch for Umbrella (not applied, tree has no git) |
| L8 | A lint false positive is fixed by modeling the framework, never by suppressing the row | new here (Δ4) | #445 (fa262633) models Baja's equal-value skip; retro 2026-10-24-go-lint-hot-write T5; the issue #459 shapes fixed in #464 |
| L9 | A guard that runs while the station loads needs proof against the live bog | new here (Δ5) | see Evidence |
| L10 | A table-driven pin gives each row a positive control | new here (Δ6) | #463 (9d469cb0, closes #299); retro 2026-10-23-r10-dedupe-pins-measured-cuts for measured cuts |

## Evidence
- L1, verify before reporting: the PANCCADIA catch-no-400 row was not a defect. At the installed 17c80d3,
  `BDashboardServlet.java:330` `parseFiniteDouble` answers 400 before `coerceValue`. The chihuahua jdk-thread row
  ("controlTick may never run") was wrong too: controlTick runs, and the real failure was the executor shutdown
  being denied (`modifyThread`). The fix is chihuahua PR #53, open. Source: niagara-research
  `odd/tasks/quality-round10-ayudante1.md` "Client lint sweep" and the chihuahua/ford section.
- L9: `BColdRoom.isChildLegal` (≤3 EvaporatorUnit, ≤1 DefrostController) runs while `config.bog` decodes. In
  baja vineflower, `ValueDocDecoder.java:473` calls `parent.asComponent().add(..., Context.decoding)` with no
  catch (the frozen `set` just above it has one). `ComponentSlotMap.checkAdd` throws `IllegalChildException` at
  `:1673`. If the live bog differs from what the guard expects, the station refuses those components at boot.
  The guard was reverted from the PANCCADIA installable set (`fix/r10-installable-set`, revert e9908d7) and
  parked on `fix/r10-coldroom-childlegal` 5f6b13b. Install it only after the live bog is verified.
- L8: 7 of the first 10 live persist-hot-write rows were Baja's equal-value skip (`ComplexSlotMap.java:1239`
  `setDouble`, `:860` `setBoolean`, `:637` `set`). #445 models that skip in `hotwrite.Hot`, pinned by RED
  `truth-equal-value-skip-1`. The rule did not get an allow-list.
- L10: #463 moves KP1-KP6 into one PINS table of pattern + banned sample. `check_pin` asserts that each pattern
  matches its sample before it asserts that the pattern matches nothing in the cards. Observed mutation: KP3
  sample replaced → KP3 red.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | **Truth-audit every round.** At a round close, run `kit-cite-resolve --strict` on the live set (`scripts/lib/live-doctrine-set.sh`) and read the excerpts. A false claim is corrected in place, never waived. | `METHODOLOGY.md` § Kit maintenance | `[ev: .githooks/pre-push live gate]` |
| 2 | **A lint row is a lead until it is read.** Before reporting a row as a defect, read the code at the installed commit and check what the framework does there. Report what you read, not what the row says. | `BUILD-LOOP.md` § 5, after the advisory-lint smoke bullet | `[ev: odd/tasks/quality-round10-ayudante1.md]` |
| 3 | **Every tool checkout is resolved fail-closed.** A launcher that runs tools from a checkout refuses one that is stale, dirty or diverged, fast-forwards a clean one that is behind, and prints the path and SHA it runs. | `METHODOLOGY.md` § Kit maintenance | `[ev: scripts/kit-resolve.sh]` |
| 4 | **Fix a lint false positive by modeling the framework.** Encode the framework behavior that makes the code correct, with a RED truth fixture. Never add an allow-list or a suppression. | `BUILD-LOOP.md` § 5 kit tooling doctrine | `[ev: go/internal/lint/rules/hotwrite/hotwrite.go]` |
| 5 | **A guard that runs at station load ships only after it is proven on the live bog.** `isChildLegal`/`isParentLegal` and a throwing `checkAdd` run while `config.bog` decodes, with no catch, so a mismatch refuses components at boot. Ship such a guard alone, after the live bog is checked. | `METHODOLOGY.md` § Schema / upgrade safety | `[ev: organized/baja/baja/vineflower/javax/baja/io/ValueDocDecoder.java:473]` |
| 6 | **A table-driven pin gives each row a positive control.** Each row carries a sample that its pattern must match, and the test checks that before it checks the absence. Otherwise a mis-escaped regex passes silently. | `BUILD-LOOP.md` § 5 kit tooling doctrine | `[ev: tests/kit-packaging-pins.bats]` |

## Not folded (owed elsewhere)
- R10-D1 (stop committing generated ledgers) is a tooling change owned by `odd/tasks/quality-round10.md`. Doctrine
  must not claim it before it lands. Until then, the "never hand-edit" rules above are the live doctrine.
- Retros 2026-10-24-r10-live-cite-prepush-gate and 2026-10-24-issue-459-truth stay pending with their own
  deltas. Δ2 here is the reporter side; the issue-459-truth Δ1 is the fixer side.

## Lessons
- Line efficiency comes from keeping one copy of each concern. Cutting tests, controls or evidence is not line
  efficiency.
- Every wrong report this round came from trusting a row, a stale checkout or an old base. The fix each time was
  to read the source at the right commit.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-close-retro`) — Δ1 and Δ3 `METHODOLOGY.md` § Kit maintenance;
Δ5 `METHODOLOGY.md` § Schema / upgrade safety; Δ2 `BUILD-LOOP.md` § 5 after the advisory-lint smoke bullet;
Δ4 and Δ6 `BUILD-LOOP.md` § 5 kit tooling doctrine.
