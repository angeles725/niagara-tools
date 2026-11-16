package readtools

import (
	"fmt"
	"strings"
)

// The histories section: the points with no history extension below them, and the history
// extensions that are disabled, in fault or have never recorded.

// withoutExtension lists in check the expected points of the points table that hold no row of
// ext (an extension sits right below its point). An absence needs both tables read, and ext
// read whole: a truncated list of extensions proves no point lacks one.
func (r *commRun) withoutExtension(s *commSection, check string, ext *commTable, expect *expectFilter, fact, info string) error {
	points, err := r.table("points")
	if err != nil {
		return err
	}
	switch c := s.check(check); {
	case points.failed != "":
		c.why = points.failed
	case ext.failed != "":
		c.why = ext.failed
	case ext.cut:
		c.why = ext.cutWhy()
	default:
		if points.cut && s.truncated == 0 {
			s.truncated = max(len(points.rows), 1)
		}
		has := map[string]bool{}
		for _, row := range ext.rows {
			has[parentPath(cellText(row, "Slot Path"))] = true
		}
		if expect == nil && info != "" { // which points should have one is the caller's to say
			c.info = info
		}
		expected := 0
		for _, row := range points.rows {
			if !expect.match(row) {
				continue
			}
			if expected++; !has[cellText(row, "Slot Path")] {
				c.add(cellText(row, "Slot Path"), pointName(row), fact)
			}
		}
		c.extra = []any{"expected", expected}
	}
	return nil
}

// answered closes a check that reads one cell per row: blank is how many rows answered none.
// When no row answered, the check proves nothing and is not checked; else the blanks are named.
func answered(c *commCheck, what string, blank, rows int) {
	switch {
	case rows > 0 && blank == rows:
		c.why, c.count, c.items = fmt.Sprintf(commNoAnswer, what, rows), 0, nil
	case blank > 0:
		c.extra = append(c.extra, "not_answered", blank)
	}
}

// historiesSection is the histories section.
func (r *commRun) historiesSection() (*commSection, error) {
	ext, err := r.table("histories")
	if err != nil {
		return nil, err
	}
	s := r.section("histories", ext)
	if err := r.withoutExtension(s, "points_without_history", ext, r.expectHistories, factNoHistory, noteExpectHistory); err != nil {
		return nil, err
	}
	disabled, fault, never := s.check("disabled"), s.check("in_fault"), s.check("never_recorded")
	if ext.failed != "" {
		s.notChecked(ext.failed, "disabled", "in_fault", "never_recorded")
		return s, nil
	}
	noEnabled, noLast := 0, 0
	for _, row := range ext.rows {
		path, name, id := cellText(row, "Slot Path"), cellText(row, "Name"), cellText(row, "Id")
		switch enabledCell(cellText(row, enabledColumn)) {
		case false:
			disabled.add(path, name, fmt.Sprintf(factHistDisabled, id))
		case nil:
			noEnabled++
		}
		if flags := statusFlags(cellText(row, bitsColumn)); contains(flags, faultFlag) || contains(flags, unknownFlag) {
			fault.add(path, name, fmt.Sprintf(factHistStatus, strings.Join(flags, ", "), id))
		}
		switch last := strings.TrimSpace(cellText(row, "Last")); {
		case last == "":
			noLast++
		case strings.EqualFold(last, nullCell):
			never.add(path, name, fmt.Sprintf(factNeverRecorded, id))
		}
	}
	answered(disabled, "enabled", noEnabled, len(ext.rows))
	answered(never, "lastRecord.timestamp", noLast, len(ext.rows))
	s.extra = []any{"history_extensions", len(ext.rows)}
	return s, nil
}
