<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-25 · kit · r10-rnt-ignored-names

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test run`, the ignored-methods gate named
from the source, branch `r10/rnt-ignored-names` from 04f031a1 (origin/main)
**Delta count**: 1

## What happened
The ignored gate of retro r10-rnt-ignored-methods compares a count: the TestNG in a Niagara `test-wb.jar` writes
the ignored methods into testng-results.xml as the root `ignored` count only. Disabling one new test and
re-enabling an old one keeps that count, so an exact `--allow-ignored-count` still passed. `run` now names the
ignored methods from the source under `--gradle-root`, each rule RED first on origin/main (no station boot):
1. Every `<type class>` of `<part>/moduleTest-include.xml` is looked up among the classes under `<part>/srcTest/`
   (the shared `javasrc` scanner, binary names from the test-type-in-prod lint). Each method-level
   `@Test(enabled = false)` (any spacing, other attributes, multi-line, qualified) a registered concrete class
   declares or inherits from a `srcTest/` superclass is one ignored method, `<SimpleName>.<method>`. An abstract
   registered type adds nothing itself; a nearer declaration of a method name decides.
2. The named count must equal the root `ignored`: a difference is FAIL "source and TestNG disagree" (a disable the
   scan cannot see: a superclass outside `srcTest/`, a group exclusion).
3. `--allow-ignored <Class.method>` (repeatable, with `--allow-ignored-reason`) waives exactly the named set: a
   disabled method not named (new) and a name no source disables (stale) are each FAIL.
4. A class-level `@Test(enabled = false)`, an `enabled` that is no `true`/`false` literal, a registered source
   that ends inside a comment or a text block, a registered type with no declaration and two registered types of
   one simple name are an env FAIL naming the file (exit 3), before the build: the scan never guesses.
5. `--allow-ignored-count` stays the fallback and skips the source; mixing it with `--allow-ignored` is exit 2.
   `verdict --testng` has no source and keeps the count pair only.

## Evidence
- RED on origin/main 04f031a1: `TestRunIgnoredNamedFromSource` — `two disabled in source, no allowance` (no row
  naming `BAlphaTest.a, BAlphaTest.b`), `allowed by name` (exit 2, want 0), `the swap` (exit 2, want 1),
  `source disables 1, TestNG ignored 3` (exit 2, want 1), `names and a count mixed` (no refusal row),
  `an unparseable registered source` (exit 0, want 3); the two count-fallback cases passed on main and guard it.
  GREEN after `go/internal/niagaratest/ignoredsrc.go` and the name judge in `ignored.go`
  `[ev: code go/internal/niagaratest/ignoredsrc.go]`.
- Mutation: replacing the new and stale name sets with empty ones (counts compared only) turned `the swap`
  (exit 0, want 1) and `two disabled in source, no allowance` red.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `run-niagara-test run` names the ignored test-methods from the registered types' source (`<SimpleName>.<method>`), FAILs when their count differs from the root `ignored`, and waives them only by name with `--allow-ignored <Class.method>` (new or stale name FAIL); class-level disables, non-literal `enabled` and unreadable registered sources are exit 3; the count pair is the fallback that skips the source | `types/moduleTest.md` § Known gotchas; `toolbelt/run-niagara-test.sh` header | `[ev: code go/internal/niagaratest/ignoredsrc.go]` |

## Lessons
- An allowance that is a count can be satisfied by a different set: when the evidence file cannot name what it
  counts, name it from the source and require both to agree.

## Fold
- The gate is code in the branch that wrote this retro (`go/internal/niagaratest/ignoredsrc.go`, `ignored.go`,
  `ignoredsrc_test.go`, `run_ignoredsrc_test.go`).
- Δ1 is cited in `types/moduleTest.md` § Known gotchas (the swapped-disable bullet). Nothing owed.

---
**Status**: FOLDED 2026-10-25 (branch `r10/rnt-ignored-names`; log in `ledger.d/2026-10-25-r10-rnt-ignored-names.md`) — Δ1 in `types/moduleTest.md` § Known gotchas.
