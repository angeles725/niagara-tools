#!/usr/bin/env bash
# lint-vacuous-tests.sh — FAILs a srcTest/ @Test method that cannot fail or never runs.
#
# A green moduleTest or JUnit run proves nothing for a test that cannot fail: a client root carried seven
# assertTrue(true) placeholders, a JUnit assumeTrue(false) placeholder no run ever reported, and tests that assert
# only through a helper (which must stay clean). Scans every *.java under a srcTest/ of ONE gradle or module root
# (build/ not entered). A test is a method with a method-level @Test (TestNG or JUnit, any qualifier); @BeforeMethod,
# @AfterMethod, @DataProvider and a class-level @Test are not read. Checks (each a FAIL row):
#   always-skipped  its first statement is assumeTrue(false) / assumeFalse(true) (any qualifier, a message allowed)
#                   or throw new SkipException(...)
#   trivially-true  every assertion is assertTrue(true), assertFalse(false), or assertEquals/assertSame of one
#                   literal with itself (a message allowed)
#   no-assertion    no assert*/verify*/expect*/fail(...) call, no new AssertionError, no asserting helper, and no
#                   expectedExceptions / expected = on its @Test (either makes the test clean)
# Helper depth: ONE level. An unqualified (or this./super.) call of a method declared in the test's class, an
# enclosing class, a srcTest/ superclass of either, or a srcTest/ class the file imports it from with
# `import static <pkg>.<Class>.*` / `.<method>`, whose own body holds a real assertion, is an assertion; a
# helper that only calls another helper, or a call qualified by another object or class, is not.
# Allowance by name only: --allow <Class.method> (repeatable; the declaring class's simple name) prints a PASS row
# carrying --allow-reason <why> (required with --allow); an --allow that names no flagged method is a stale-allow
# FAIL. There is no count allowance. Java is read as tokens (go/internal/javasrc): a comment or a string is never a
# call. A source that ends inside a comment or a text block, or a class body that is not closed, is exit 3.
# [ev: retro r10-lint-vacuous-tests Δ1]
#
# Implementation: rule package go/internal/lint/rules/vacuoustests on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-vacuous-tests <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-vacuous-tests.sh [--allow <Class.method>]... [--allow-reason <why>] <gradle-root>
#   Row:  FAIL  <check>  <file>:<line>  <Class.method>: <why>   ·   PASS  allowed  <file>:<line>  <Class.method>: <check>, allowed: <why>
#   Row:  FAIL  stale-allow  --allow <Class.method> names no flagged test method (remove it)
#   Exit: 0  clean or allowed only · 1  any FAIL · 3  usage/env, an unparseable source, a root with no srcTest/ *.java
# Mutation: VT1 -- dropping the trivially-true check reads assertTrue(true) as a real assertion (exit 0)
# Mutation: VT2 -- not resolving helpers flags the check(...) test as no-assertion
# Mutation: VT3 -- dropping the always-skipped check reads assumeTrue(false) as a test that runs
# Mutation: VT4 -- skipping the stale check passes an --allow that names a test which is not flagged
# Mutation: VT5-static -- not resolving static imports flags the static-imported await(...) test as no-assertion
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-vacuous-tests: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-vacuous-tests "$@"
