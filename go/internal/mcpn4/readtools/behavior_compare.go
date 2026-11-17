package readtools

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// pointState is one control point of a behaviour snapshot: its slot path, its out value as the
// station wrote it (BQL text, possibly in the station's locale) and its status.bits cell.
type pointState struct {
	Path, Value, Status string
}

// behaviorDiff is what changed between two point sets: each list sorted by path.
type behaviorDiff struct {
	statusChanged, valueChanged []pointChange
	missing, added              []string
	unchanged                   int
}

// pointChange is one point whose status or value differs.
type pointChange struct {
	path, before, after string
}

// compareStates compares before with after by path. A value counts as changed when the texts
// differ, unless both read as numbers within tolerance of each other (absolute difference).
func compareStates(before, after []pointState, tolerance float64) behaviorDiff {
	var d behaviorDiff
	now := map[string]pointState{}
	for _, p := range after {
		now[p.Path] = p
	}
	seen := map[string]bool{}
	for _, b := range before {
		seen[b.Path] = true
		a, ok := now[b.Path]
		if !ok {
			d.missing = append(d.missing, b.Path)
			continue
		}
		same := true
		if a.Status != b.Status {
			d.statusChanged = append(d.statusChanged, pointChange{b.Path, b.Status, a.Status})
			same = false
		}
		if !sameValue(b.Value, a.Value, tolerance) {
			d.valueChanged = append(d.valueChanged, pointChange{b.Path, b.Value, a.Value})
			same = false
		}
		if same {
			d.unchanged++
		}
	}
	for _, a := range after {
		if !seen[a.Path] {
			d.added = append(d.added, a.Path)
		}
	}
	sort.Strings(d.missing)
	sort.Strings(d.added)
	byPath := func(l []pointChange) { sort.Slice(l, func(i, j int) bool { return l[i].path < l[j].path }) }
	byPath(d.statusChanged)
	byPath(d.valueChanged)
	return d
}

func sameValue(a, b string, tolerance float64) bool {
	if a == b {
		return true
	}
	x, okA := numberText(a)
	y, okB := numberText(b)
	return okA && okB && math.Abs(x-y) <= tolerance
}

// numberText reads a BQL value cell as a number: "21.5", "21,5" (a locale decimal comma when the
// text holds no point), with an optional trailing unit after a space ("21.5 °C"). Anything else
// (true, an enum tag, "1,234.5") is not a number and compares as text.
func numberText(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, ' '); i > 0 {
		s = s[:i]
	}
	if !strings.Contains(s, ".") && strings.Count(s, ",") == 1 {
		s = strings.Replace(s, ",", ".", 1)
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}
