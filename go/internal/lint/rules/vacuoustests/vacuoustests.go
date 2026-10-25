// Package vacuoustests is lint-vacuous-tests: a srcTest/ test method that cannot fail or never runs. A green
// moduleTest or JUnit run proves nothing for such a method, and a client root carried all three shapes (seven
// assertTrue(true) placeholders, an assumeTrue(false) placeholder no run ever reported, retro
// r10-lint-vacuous-tests). A method is a test when a method-level @Test annotation (any qualifier: TestNG or
// JUnit) precedes it; @BeforeMethod, @AfterMethod, @DataProvider and every other method are not tests, and a
// class-level @Test is not read. Each test is one of:
//   - always-skipped (FAIL): its first statement is assumeTrue(false) or assumeFalse(true) (any qualifier, an
//     optional message argument) or throw new SkipException(...);
//   - clean: its @Test sets expectedExceptions or expected (it fails when nothing is thrown), or its body holds a
//     real assertion;
//   - trivially-true (FAIL): every assertion it holds is trivially true (assertTrue(true), assertFalse(false), an
//     assertEquals or assertSame of one literal with itself, a message argument allowed);
//   - no-assertion (FAIL): it holds no assertion at all.
//
// An assertion is a call of a name starting with assert, verify or expect, a fail(...) call, or a new
// AssertionError. A call of a helper also counts when it is unqualified (or this./super.) and a method of that
// name, declared in the test's class, an enclosing class, a srcTest/ superclass of either (by simple name), or a
// srcTest/ class the file imports it from statically (`import static <pkg>.<Class>.*` or `.<method>`), holds a real
// assertion in its own body. That is ONE level deep: a helper that only calls another helper does not count,
// nor does a call qualified by another object or class. Java is read as javasrc.Tokens (comments stripped, a
// literal one token), so an assertion in a comment or a string is not one. --allow <Class.method> (repeatable, the
// simple name of the declaring class) turns its FAIL into a PASS row carrying --allow-reason; an --allow that names
// no flagged method FAILs (stale). A source that ends inside a comment or a text block, or a class body the scanner
// cannot close, is exit 3: the scan never guesses. The toolbelt wrapper toolbelt/lint-vacuous-tests.sh holds the
// user-facing contract and the mutation pins.
package vacuoustests

import (
	"fmt"
	"slices"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name n4lint accepts; rows print their check (trivially-true, no-assertion, ...) instead.
const ID = "lint-vacuous-tests"

const usage = "usage: lint-vacuous-tests.sh [--allow <Class.method>]... [--allow-reason <why>] <gradle-root>"

const emptyMsg = "no *.java under a srcTest/ (wrong path: pass a gradle or module root)"

// Spec registers the rule: [--allow <Class.method>]... [--allow-reason <why>] <gradle-root> in any order (Mode
// Valued), FAIL rows, every *.java under a srcTest/ (build/ not entered). Its root is a gradle or module root, not
// a source root, so n4lint --all leaves it out (Spec.ModuleRoot).
func Spec() lint.Spec {
	return lint.Spec{
		Tree: &rule{}, Mode: lint.Valued, Option: "--allow-reason", Options: []string{"--allow"}, ModuleRoot: true,
		Usage:      usage,
		Select:     lint.Selection{Names: []string{"*.java"}, Paths: []string{"*/srcTest/*"}, Prune: []string{"build"}},
		EmptyError: lint.EmptyRow{Msg: emptyMsg},
	}
}

// rule holds the allowances Configure read; the registered rule, run in a rule list, allows nothing.
type rule struct {
	allow  []string // Class.method names, each once, in the order given
	reason string
}

func (*rule) ID() string              { return ID }
func (*rule) Severity() lint.Severity { return lint.Fail }

// Configure reads every --allow name and the --allow-reason (the last one given); an allowance needs a reason.
func (*rule) Configure(a lint.Args) (lint.Configured, error) {
	r := &rule{reason: a.Option}
	for _, n := range a.Values["--allow"] {
		class, method, ok := strings.Cut(n, ".")
		if !ok || class == "" || method == "" || strings.Contains(method, ".") {
			return nil, fmt.Errorf("%s: --allow %s: not a <Class.method> name\n", ID, n)
		}
		if !slices.Contains(r.allow, n) {
			r.allow = append(r.allow, n)
		}
	}
	if len(r.allow) > 0 && strings.TrimSpace(r.reason) == "" {
		return nil, fmt.Errorf("%s: --allow needs --allow-reason <why> (the reason is printed in the report)\n", ID)
	}
	return r, nil
}

// CheckTree judges every test method of the srcTest/ sources, in file and source order, then the stale
// allowances. An unreadable source is the error after the rows of the others; stale allowances are then not
// judged (the method they name may sit in that source).
func (r *rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	idx, bad := newIndex(files)
	flagged, err := idx.flagged()
	if err != nil {
		bad = append(bad, err.Error())
	}
	var out []lint.Finding
	used := map[string]bool{}
	for _, v := range flagged {
		if slices.Contains(r.allow, v.name) {
			used[v.name] = true
			out = append(out, lint.Finding{Check: "allowed", Severity: lint.Pass, Path: v.path, Line: v.line,
				Msg: v.name + ": " + v.check + ", allowed: " + r.reason})
			continue
		}
		out = append(out, lint.Finding{Check: v.check, Path: v.path, Line: v.line, Msg: v.name + ": " + v.why})
	}
	if len(bad) > 0 {
		return out, fmt.Errorf("%s", strings.Join(bad, "\n"+ID+": "))
	}
	for _, n := range r.allow {
		if !used[n] {
			out = append(out, lint.Finding{Check: "stale-allow", Msg: "--allow " + n + " names no flagged test method (remove it)"})
		}
	}
	return out, nil
}
