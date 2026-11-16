package readtools

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The alarms section: the points with no alarm extension (only where the caller says which
// should have one), the extensions naming a class the station does not have, the classes with
// no recipient linked, and each class's open and unacknowledged alarms.

// alarmsSection is the alarms section.
func (r *commRun) alarmsSection() (*commSection, error) {
	ext, err := r.table("alarms")
	if err != nil {
		return nil, err
	}
	classes, err := r.table("classes")
	if err != nil {
		return nil, err
	}
	s := r.section("alarms", ext, classes)
	if r.expectAlarms == nil {
		s.check("points_without_alarm").skipped = skipNoExpectAlarms
	} else if err := r.withoutExtension(s, "points_without_alarm", ext, r.expectAlarms, factNoAlarm, ""); err != nil {
		return nil, err
	}
	names := []string{"unknown_alarm_class", "classes_without_recipient", "open_alarms", "unacked_alarms"}
	for _, name := range names {
		s.check(name)
	}
	if classes.failed != "" {
		s.notChecked(classes.failed, names...)
		return s, nil
	}
	named := r.alarmClassNames(s, ext, classes)
	var kept []*pyjson.Object // the classes reported: all of them, or those the scope names
	for _, row := range classes.rows {
		if r.slot == "/" || named == nil || named[lastSegment(cellText(row, "Slot Path"))] || named[cellText(row, "Name")] {
			kept = append(kept, row)
		}
	}
	if r.slot != "/" {
		s.extra = append(s.extra, "note", noteClassScope)
	}
	s.extra = append(s.extra, "alarm_classes", len(kept))
	classCounts(s.check("open_alarms"), kept, "Open", "openAlarmCount", factOpen)
	classCounts(s.check("unacked_alarms"), kept, "Unacked", "unackedAlarmCount", factUnacked)
	return s, r.classRecipients(s.check("classes_without_recipient"), kept)
}

// alarmClassNames checks the class each alarm extension names against the station's classes
// (by slot name, escaped or not) and answers the names the extensions use; nil when the
// extensions could not be read.
func (r *commRun) alarmClassNames(s *commSection, ext, classes *commTable) map[string]bool {
	c := s.check("unknown_alarm_class")
	if ext.failed != "" {
		c.why = ext.failed
		return nil
	}
	if classes.cut {
		c.why = classes.cutWhy()
	}
	known, named, blank := map[string]bool{}, map[string]bool{}, 0
	for _, row := range classes.rows {
		known[lastSegment(cellText(row, "Slot Path"))], known[cellText(row, "Name")] = true, true
	}
	for _, row := range ext.rows {
		class := strings.TrimSpace(cellText(row, "Class"))
		if class == "" {
			blank++
			continue
		}
		if named[class] = true; c.why == "" && !known[class] && !known[bql.UnescapeSlot(class)] {
			c.add(cellText(row, "Slot Path"), cellText(row, "Name"), fmt.Sprintf(factUnknownClass, class))
		}
	}
	if c.why == "" {
		answered(c, "alarmClass", blank, len(ext.rows))
	}
	return named
}

// classCounts lists in c the classes whose count cell (column, the slot what) is above zero;
// alarms is their sum. A cell that is no whole number is not answered.
func classCounts(c *commCheck, classes []*pyjson.Object, column, what, fact string) {
	blank, total := 0, 0
	for _, row := range classes {
		n, err := strconv.Atoi(strings.TrimSpace(cellText(row, column)))
		if err != nil || n < 0 {
			blank++
			continue
		}
		if n > 0 {
			c.add(cellText(row, "Slot Path"), cellText(row, "Name"), fmt.Sprintf(fact, n))
			total += n
		}
	}
	c.extra = []any{"alarms", total}
	answered(c, what, blank, len(classes))
}
