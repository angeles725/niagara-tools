package readtools

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The shapes every section of n4_commissioning_report is built of: a check (a count, a bounded
// list of offenders and how many more; or why it did not run) and a section of named checks
// with its state and summary line.

// commCheck is one check. It ran unless why (not checked: a failed or impossible read) or
// skipped (the caller did not ask for it) is set; a check that did not run has no count.
type commCheck struct {
	name         string
	count, limit int
	items        []any
	extra        []any // further key, value pairs of a check that ran
	why, skipped string
}

// add counts one finding and lists it while the list is under the limit.
func (c *commCheck) add(path, name, fact string) {
	if c.count++; len(c.items) < c.limit {
		c.items = append(c.items, obj("path", path, "name", name, "fact", fact))
	}
}

// reply is the check as the reply carries it. hidden are the findings counted but left out of
// the list on purpose (not by the limit), so more counts the limit's cut alone.
func (c *commCheck) reply(hidden int) *pyjson.Object {
	switch {
	case c.why != "":
		return obj("not_checked", c.why)
	case c.skipped != "":
		return obj("skipped", c.skipped)
	}
	items := c.items
	if items == nil {
		items = []any{}
	}
	out := obj("count", c.count, "items", items, "more", c.count-len(items)-hidden)
	for i := 0; i < len(c.extra); i += 2 {
		out.Set(c.extra[i].(string), c.extra[i+1])
	}
	return out
}

// commSection is one section: its checks in reply order and what else it carries.
type commSection struct {
	name      string
	limit     int
	checks    []*commCheck
	hidden    map[string]int // per check: findings counted, not listed
	extra     []any          // further key, value pairs
	truncated int            // the rows a truncated query answered (0: whole)
}

// check is the section's check called name, made on first use.
func (s *commSection) check(name string) *commCheck {
	for _, c := range s.checks {
		if c.name == name {
			return c
		}
	}
	c := &commCheck{name: name, limit: s.limit}
	s.checks = append(s.checks, c)
	return c
}

// notChecked marks every named check as not run, for the reason why.
func (s *commSection) notChecked(why string, names ...string) {
	for _, name := range names {
		s.check(name).why = why
	}
}

// reply is the section as the reply carries it and its summary line. Its state is findings
// when a check found something, ok when every check ran clean, partial when the checks that
// ran are clean but some did not run, not_checked when none ran, skipped when none was asked.
func (s *commSection) reply() (*pyjson.Object, string) {
	findings, ran, notRun, why, skipped := 0, 0, 0, "", ""
	checks := pyjson.NewObject()
	for _, c := range s.checks {
		switch {
		case c.why != "":
			if notRun++; why == "" {
				why = c.why
			}
		case c.skipped != "":
			skipped = c.skipped
		default:
			ran++
			findings += c.count
		}
		checks.Set(c.name, c.reply(s.hidden[c.name]))
	}
	state, line := "ok", fmt.Sprintf(sumOK, s.name)
	switch {
	case ran == 0 && notRun > 0:
		state, line = "not_checked", fmt.Sprintf(sumNotChecked, s.name, why)
	case ran == 0:
		state, line = "skipped", fmt.Sprintf(sumSkipped, s.name, skipped)
	case findings > 0:
		state, line = "findings", fmt.Sprintf(sumFindings, s.name, findings)
		if notRun > 0 {
			line += fmt.Sprintf(sumAlsoNot, notRun)
		}
	case notRun > 0:
		state, line = "partial", fmt.Sprintf(sumPartial, s.name, notRun)
	}
	out := obj("state", state)
	if ran > 0 { // a section none of whose checks ran has no count: a failed read is not a zero
		out.Set("findings", findings)
	}
	if s.truncated > 0 {
		out.Set("truncated", true)
		line += fmt.Sprintf(commTruncShort, s.truncated)
	}
	out.Set("checks", checks)
	for i := 0; i < len(s.extra); i += 2 {
		out.Set(s.extra[i].(string), s.extra[i+1])
	}
	return out, line
}
