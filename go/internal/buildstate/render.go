package buildstate

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	openMarker  = "<!-- build-state.v1 -->"
	closeMarker = "<!-- /build-state.v1 -->"

	errNoKit   = "BUILD-STATE.md needs exactly one closed build-state.v1 envelope with `module: kit`"
	errFields  = "the kit envelope needs one `last_change:` line, then one `open_issues:` line, then one `last_session:` line"
	errLogTail = "not a session-log line; the generated session log runs from `last_session:` to the closing marker, so a hand-kept field belongs above `last_session:`"
)

// LastChange renders the kit `last_change:` line from the newest fragment: the latest date header,
// a tie going to the later filename. The note, when present, is the `# comment` at column 38, the
// alignment of every field comment in the envelope.
func LastChange(frags []Fragment) string {
	var best *Fragment
	for i := range frags {
		f := &frags[i]
		// Mutation: GBS-last -- taking the last fragment by name instead of the newest date
		if best == nil || f.Date > best.Date || (f.Date == best.Date && f.Name > best.Name) {
			best = f
		}
	}
	if best == nil {
		return "last_change: unknown"
	}
	line := "last_change: " + best.Date
	if best.Note != "" {
		line = fmt.Sprintf("%-37s# %s", line, best.Note)
	}
	return line
}

// byName returns frags sorted by filename: the 0000-archive fragment first, then by date (the
// filename prefix equals the date header).
func byName(frags []Fragment) []Fragment {
	sorted := append([]Fragment(nil), frags...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	return sorted
}

// Render returns state with the three generated parts of the kit envelope replaced: last_change, the
// open_issues items (newest fragment first, each fragment's items in file order) and the session log
// after last_session (oldest fragment first). Every other byte of state is kept.
func Render(state string, frags []Fragment) (string, error) {
	lines := strings.Split(state, "\n")
	open, end, err := kitEnvelope(lines)
	if err != nil {
		return "", err
	}
	lc, oi, ls := -1, -1, -1
	for i := open + 1; i < end; i++ {
		switch l := lines[i]; {
		case strings.HasPrefix(l, "last_change:") && lc < 0:
			lc = i
		case l == "open_issues:" && oi < 0:
			oi = i
		case strings.HasPrefix(l, "last_session:") && ls < 0:
			ls = i
		case strings.HasPrefix(l, "last_change:"), l == "open_issues:", strings.HasPrefix(l, "last_session:"):
			return "", errors.New(errFields)
		}
	}
	if lc < 0 || oi < 0 || ls < 0 || !(lc < oi && oi < ls) {
		return "", errors.New(errFields)
	}
	items := oi + 1
	for items < end && strings.HasPrefix(lines[items], "  - ") {
		items++
	}
	// Mutation: GBS-tail -- a hand-added field after last_session would be dropped silently on regeneration
	for i := ls + 1; i < end; i++ {
		if !logPattern.MatchString(lines[i]) {
			return "", fmt.Errorf("BUILD-STATE.md line %d (`%s`): %s", i+1, lines[i], errLogTail)
		}
	}

	sorted := byName(frags)
	var issues, log []string
	for i := len(sorted) - 1; i >= 0; i-- {
		for _, it := range sorted[i].Issues {
			issues = append(issues, "  - "+it)
		}
	}
	for _, f := range sorted {
		log = append(log, f.Logs...)
	}

	out := make([]string, 0, len(lines)+len(issues)+len(log))
	out = append(out, lines[:lc]...)
	out = append(out, LastChange(frags))
	out = append(out, lines[lc+1:oi+1]...)
	out = append(out, issues...)
	out = append(out, lines[items:ls+1]...)
	out = append(out, log...)
	out = append(out, lines[end:]...)
	return strings.Join(out, "\n"), nil
}

// kitEnvelope returns the line indexes of the opening and closing markers of the one kit envelope.
func kitEnvelope(lines []string) (int, int, error) {
	open, kitOpen, kitEnd := -1, -1, -1
	for i, l := range lines {
		switch {
		case l == openMarker:
			open = i
		case l == closeMarker && open >= 0:
			if isKit(lines[open+1 : i]) {
				if kitOpen >= 0 {
					return 0, 0, errors.New(errNoKit)
				}
				kitOpen, kitEnd = open, i
			}
			open = -1
		}
	}
	if kitOpen < 0 {
		return 0, 0, errors.New(errNoKit)
	}
	return kitOpen, kitEnd, nil
}

func isKit(body []string) bool {
	for _, l := range body {
		if v, ok := strings.CutPrefix(l, "module:"); ok {
			v, _, _ = strings.Cut(v, "#")
			return strings.TrimSpace(v) == "kit"
		}
	}
	return false
}
