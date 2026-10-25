<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-25 · kit · r10-lint-vacuous-tests

**Session**: niagara-tools quality round 10, a new kit lint `lint-vacuous-tests` over a gradle root's `srcTest/`
Java, branch `r10/lint-vacuous-tests` from 63b1c44f (origin/main)
**Delta count**: 1

## What happened
A client root's test suite read green while some of its tests could not fail or never ran: seven
`assertTrue(true)` placeholders, a JUnit `assumeTrue(false)` placeholder that no run ever reported, and tests whose
only assertion was a helper call (those are real and must stay clean). The run counts (`run-niagara-test`) and the
ignored gate cannot see any of this: the methods run, or skip, as designed. The new lint reads the source, on the
shared `javasrc` scanner (no new parser), each rule RED first (the rule package and the wrapper did not exist):
1. A test is a method with a method-level `@Test` (TestNG or JUnit, any qualifier); `@BeforeMethod`,
   `@AfterMethod`, `@DataProvider` and a class-level `@Test` are not read.
2. `always-skipped`: the first statement is `assumeTrue(false)` / `assumeFalse(true)` (any qualifier, a message
   argument allowed) or `throw new SkipException(...)`. A later or a conditional assume is not one.
3. `trivially-true`: every assertion is `assertTrue(true)`, `assertFalse(false)`, or an `assertEquals` /
   `assertSame` of one literal with itself, a message allowed first or last.
4. `no-assertion`: no `assert*` / `verify*` / `expect*` / `fail(...)` call, no `new AssertionError`, no asserting
   helper; `expectedExceptions` or `expected =` on the `@Test` keeps a test clean.
5. Helper depth is ONE level: an unqualified (or `this.`/`super.`) call of a method of the test's class, an
   enclosing class, a `srcTest/` superclass of either, or a `srcTest/` class the file imports it from with
   `import static <pkg>.<Class>.*` or `.<method>`, whose own body holds a real assertion. A read-only run on the
   client root found the static-import case flagged first (a statically imported fixture `await(...)` whose body
   asserts); the message-first `assertTrue("msg", true)` form stays trivially true there.
6. Allowance by name only: `--allow <Class.method>` + `--allow-reason <why>` prints a PASS row with the reason; an
   allowance that names no flagged method FAILs (stale). A source ending inside a comment or a text block, or an
   unclosed class body, is exit 3.

## Evidence
- RED before the rule: `go test ./internal/lint/rules/vacuoustests/` failed to build (`undefined: Spec`) with the
  14 golden cases in place, and `bats tests/lint-vacuous-tests.bats` failed 6/6 (exit 127, no wrapper). GREEN after
  `go/internal/lint/rules/vacuoustests` and `toolbelt/lint-vacuous-tests.sh`
  `[ev: code go/internal/lint/rules/vacuoustests/judge.go]`.
- RED for the static-import case: golden `static-import` flagged `BWaitTest.viaStaticImport` and
  `BMemberTest.viaMember` (helper asserts) as no-assertion; GREEN once helpers resolve through the file's static
  imports of `srcTest/` classes. Its siblings stay FAIL: a non-asserting imported helper, an un-imported one, a
  two-level one, and a member import that does not name the called method.
- Mutation: dropping the always-skipped check (`skipFirst` returns "") turned `VT3` and the golden case
  `always-skipped` red.
- Fixtures are generic (ModuleA, `BAlphaTest`...); no client source was copied.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `lint-vacuous-tests.sh <gradle-root>` FAILs a `srcTest/` `@Test` method that is always skipped, trivially true or has no assertion; expected exceptions and a one-level asserting helper (incl. a static import of a `srcTest/` class) keep it clean; waived only by `--allow <Class.method> --allow-reason <why>`, a stale name FAIL | `types/moduleTest.md` § Known gotchas; `toolbelt/lint-vacuous-tests.sh` header | `[ev: code go/internal/lint/rules/vacuoustests/judge.go]` |

## Lessons
- A green test run counts methods, not proof: a test that cannot fail is found only by reading its source, and
  the helper-assertion case must be resolved, or the lint drowns real tests in false positives.

## Fold
- The lint is code in the branch that wrote this retro (`go/internal/lint/rules/vacuoustests`,
  `toolbelt/lint-vacuous-tests.sh`, `tests/lint-vacuous-tests.bats`); the generated lint index routes it.
- Δ1 is cited in `types/moduleTest.md` § Known gotchas (the vacuous-test bullet).
- Owed: `report-module.sh` does not run it yet (go/internal/reportmodule/completeness_test.go `notInReport`): a
  module with no `srcTest/` is the lint's exit-3 empty root, so a member needs a noSource SKIP row that changes
  every report golden; a follow-up.

---
**Status**: FOLDED 2026-10-25 (branch `r10/lint-vacuous-tests`; log in `ledger.d/2026-10-25-r10-lint-vacuous-tests.md`) — Δ1 in `types/moduleTest.md` § Known gotchas.
