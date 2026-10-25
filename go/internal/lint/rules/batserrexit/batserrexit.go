// Package batserrexit is lint-bats-errexit: a bats assertion that errexit cannot see, so it never fails its test.
//
// bats runs a test body under errexit, and bash ignores errexit for every command of an && or || list except
// the last one, and for a command negated with !. Mid-test, `[ a ] && [[ b ]]` therefore does nothing when
// [ a ] is false: the list returns non-zero, errexit does not fire and the test goes on; and `! cmd` never
// fails the test either. Only the status of the test's last command counts. Two checks, each a FAIL row:
//   - and-list: a mid-test && list whose elements are all assertions ([, [[, test, grep, diff, cmp, false, an
//     assert*/refute* helper, a negated pipeline). Fix: one assertion per line.
//   - negation: a mid-test `! cmd`. Fix: `run ! cmd` (bats 1.5+, it sets $status and $output) or
//     `! cmd || return 1`.
//
// Left alone, because their status decides or they are not assertions: the test's last command (and, when it
// is an if, a loop or a { } group, the last command of its branches), the last command of a standalone ( )
// subshell (the subshell's exit status fails the test), a list ending in `|| <cmd>` (`|| fail`, `|| skip`,
// and the deliberate `|| true`), a guard whose last element is an action (`[ -n "$p" ] && ln -s "$p" d`),
// if/while/until conditions, and everything run with errexit already ignored (a condition, the non-final
// element of a list). Only @test bodies are read; a helper function outside them is not, although the same
// rule holds inside it when a test calls it as a plain command. A file is never an error: a construct the
// parser cannot place is skipped (fewer rows, not a scan failure).
package batserrexit

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name n4lint accepts and every row prints.
const ID = "lint-bats-errexit"

// The check column of each row.
const (
	CheckAndList  = "bats-and-list"
	CheckNegation = "bats-negation"
)

// Spec registers the rule: [--strict] <dir>... (FAIL rows: exit 1), every *.bats under each root outside
// testdata/ and fixtures/ (inputs of other tests, not tests), and a root with no *.bats is the ERROR row
// (exit 3). Its roots are bats test directories, not Java source roots, so n4lint --all leaves it out
// (Spec.ModuleRoot).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.Each,
		ModuleRoot: true,
		Usage:      "usage: lint-bats-errexit.sh [--strict] <bats-dir>...",
		Select:     lint.Selection{Names: []string{"*.bats"}, Prune: []string{"testdata", "fixtures"}},
		EmptyError: lint.EmptyRow{Msg: "no *.bats under the root (wrong path: pass a bats test directory)"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

const (
	msgAndList  = "mid-test && list of assertions: errexit checks only its last command, so an earlier false assertion never fails the test -- put one assertion per line"
	msgNegation = "mid-test ! negation: errexit never fires on a negated command, so it never fails the test -- use `run ! cmd` or `! cmd || return 1`"
)

// Check reports every hidden assertion of every @test body, in source order.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	for _, body := range testBodies(strings.Join(f.Raw, "\n")) {
		walk(body, true, false, &out)
	}
	return out, nil
}

// testBodies parses src and returns the body of every `@test <name> {` block.
func testBodies(src string) [][]*list {
	p := &parser{toks: lex(src)}
	var bodies [][]*list
	for {
		p.skipNL()
		if p.cur() == nil {
			return bodies
		}
		if p.isWord("@test") {
			for p.i++; p.cur() != nil && !p.isWord("{"); p.i++ {
			}
			p.i++
			bodies = append(bodies, p.statements([]string{"}"}, nil))
			p.i++
			continue
		}
		if p.list(nil, nil) == nil {
			p.i++
		}
	}
}

// walk adds a row for each hidden assertion of body. tail: body's last list decides the status of the test;
// ignored: errexit is already off for all of body (a condition or a non-final list element).
func walk(body []*list, tail, ignored bool, out *[]lint.Finding) {
	for i, l := range body {
		last := tail && i == len(body)-1
		if !ignored && !last {
			if check, msg := classify(l); check != "" {
				*out = append(*out, lint.Finding{Check: check, Line: l.line, Msg: msg})
			}
		}
		alone := len(l.pipes) == 1 && len(l.pipes[0].cmds) == 1 && !l.pipes[0].bang
		for pi, pl := range l.pipes {
			off := ignored || pl.bang || pi < len(l.pipes)-1
			for _, c := range pl.cmds {
				for _, cond := range c.conds {
					walk(cond, false, true, out)
				}
				for _, b := range c.bodies {
					switch c.compound {
					case "subshell": // its exit status is a command status of its own: its last list decides it
						walk(b, alone && !off || last, off, out)
					case "func": // its return status is that of its last list, checked where it is called
						walk(b, true, false, out)
					default: // if, loops, case and { }: errexit does not fire on the compound's status
						walk(b, alone && last, off, out)
					}
				}
			}
		}
	}
}

// classify names the check a list fails, or "" when errexit sees it.
func classify(l *list) (string, string) {
	n := len(l.pipes)
	if n == 1 {
		if l.pipes[0].bang {
			return CheckNegation, msgNegation
		}
		return "", ""
	}
	for _, op := range l.ops {
		if op != "&&" {
			return "", "" // an || tail runs and fails when an assertion before it is false
		}
	}
	for _, pl := range l.pipes {
		if !assertion(pl) {
			return "", "" // a guard: `[ -f x ] && rm x` is a condition, not an assertion
		}
	}
	return CheckAndList, msgAndList
}

// assertWords are the commands whose only job in a test is to succeed or fail.
var assertWords = map[string]bool{"[": true, "[[": true, "test": true, "grep": true, "egrep": true, "fgrep": true,
	"diff": true, "cmp": true, "false": true}

// assertion reports whether a pipeline is an assertion: negated, or its last command is an assert word or an
// assert*/refute* helper (a pipeline's status is its last command's).
func assertion(pl *pipeline) bool {
	if pl.bang {
		return true
	}
	c := pl.cmds[len(pl.cmds)-1]
	if c.compound != "" || len(c.words) == 0 {
		return false
	}
	w := c.words[0]
	return assertWords[w] || strings.HasPrefix(w, "assert") || strings.HasPrefix(w, "refute")
}
