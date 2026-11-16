package readtools

import (
	"fmt"
	"strings"
)

// The naming section, over the control points in scope: display names repeated among siblings,
// slot names holding escaped characters, and points still called by a palette default name.

// namingSection is the naming section.
func (r *commRun) namingSection() (*commSection, error) {
	points, err := r.table("points")
	if err != nil {
		return nil, err
	}
	s := r.section("naming", points)
	duplicates, escaped, palette := s.check("duplicate_display_names"), s.check("escaped_names"), s.check("palette_default_names")
	if points.failed != "" {
		s.notChecked(points.failed, duplicates.name, escaped.name, palette.name)
		return s, nil
	}
	shared, blank := map[string]int{}, 0 // siblings per parent and display name
	sibling := func(path, display string) string { return parentPath(path) + "\x00" + display }
	for _, row := range points.rows {
		if display := strings.TrimSpace(cellText(row, displayColumn)); display != "" {
			shared[sibling(cellText(row, "Slot Path"), display)]++
		} else {
			blank++
		}
	}
	for _, row := range points.rows {
		path, name := cellText(row, "Slot Path"), cellText(row, "Name")
		display := strings.TrimSpace(cellText(row, displayColumn))
		if n := shared[sibling(path, display)]; display != "" && n > 1 {
			duplicates.add(path, display, fmt.Sprintf(factDuplicate, display, n))
		}
		if slot := lastSegment(path); escapedNameRe.MatchString(slot) {
			escaped.add(path, pointName(row), fmt.Sprintf(factEscaped, slot, strings.Join(uniqueStrings(escapedNameRe.FindAllString(slot, -1)), ", ")))
		}
		if fact := paletteDefaultFact(name, cellText(row, "Type")); fact != "" {
			palette.add(path, pointName(row), fact)
		}
	}
	answered(duplicates, "displayName", blank, len(points.rows))
	s.extra = []any{"note", noteNaming}
	return s, nil
}

// paletteDefaultFact is why name is still a palette default name for a point of type kind, or "".
// A control palette point name is one with or without the number the station appends to a
// repeated name (NumericWritable, NumericWritable1). The simple name of the point's own type is
// one only with that number (Tstat1 of kitControl:Tstat): a palette drop names the first instance
// after its type, so a lone Tstat is as likely a meaningful name as a forgotten default.
func paletteDefaultFact(name, kind string) string {
	base := trailingDigitsRe.ReplaceAllString(name, "")
	if base == "" {
		return ""
	}
	if contains(paletteDefaultNames, base) {
		return fmt.Sprintf(factPaletteList, base)
	}
	if _, simple, _ := strings.Cut(kind, ":"); base == simple && base != name {
		return fmt.Sprintf(factPaletteType, name, kind)
	}
	return ""
}

// uniqueStrings is list without its repeats, in order.
func uniqueStrings(list []string) []string {
	seen, out := map[string]bool{}, []string{}
	for _, s := range list {
		if !seen[s] {
			seen[s], out = true, append(out, s)
		}
	}
	return out
}
